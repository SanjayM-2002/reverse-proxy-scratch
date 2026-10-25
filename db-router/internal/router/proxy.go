package router

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"strings"
	"time"
)

// Proxy accepts client connections and routes every command line on its own.
//
// Difference from the load balancer: the LB picked one backend per TCP connection.
// Here one client connection can hit many databases — the target is chosen per command.
type Proxy struct {
	router Router
}

func NewProxy(router Router) *Proxy {
	return &Proxy{router: router}
}

func (p *Proxy) ListenAndServe(addr string) error {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	log.Printf("[router] listening on %s", addr)

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Printf("[router] accept: %v", err)
			continue
		}
		go p.handleClient(conn)
	}
}

// handleClient reads one command per line until the client disconnects.
func (p *Proxy) handleClient(client net.Conn) {
	defer client.Close()
	log.Printf("[router] client %s connected", client.RemoteAddr())

	lines := bufio.NewScanner(client)
	for lines.Scan() {
		line := strings.TrimSpace(lines.Text())
		if line == "" {
			continue
		}
		fmt.Fprintln(client, p.handleCommand(line))
	}

	log.Printf("[router] client %s disconnected", client.RemoteAddr())
}

// handleCommand is the core flow: parse -> pick target -> forward -> reply.
func (p *Proxy) handleCommand(line string) string {
	cmd, err := ParseCommand(line)
	if err != nil {
		return "ERR " + err.Error()
	}

	target := p.router.Route(cmd)
	log.Printf("[router] %s %s -> %s (%s)", cmd.Op, cmd.Key, target.Name, target.Addr)

	reply, err := forward(target, line)
	if err != nil {
		return fmt.Sprintf("ERR %s unreachable: %v", target.Name, err)
	}
	return reply
}

// forward sends one command to the target and returns its one-line reply.
// It opens a fresh connection per command — slow, but the simplest thing to read.
func forward(target Target, line string) (string, error) {
	conn, err := net.DialTimeout("tcp", target.Addr, 2*time.Second)
	if err != nil {
		return "", err
	}
	defer conn.Close()
	conn.SetDeadline(time.Now().Add(2 * time.Second))

	if _, err := fmt.Fprintln(conn, line); err != nil {
		return "", err
	}

	reply, err := bufio.NewReader(conn).ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(reply), nil
}
