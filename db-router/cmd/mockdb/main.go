// A fake database: stores nothing. It logs each command and replies with its own name,
// so you can see which database the router picked.
package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"net"
	"strings"
)

func main() {
	name := flag.String("name", "primary-9001", "database name shown in logs and replies")
	port := flag.Int("port", 9001, "port to listen on")
	flag.Parse()

	logger := log.New(log.Writer(), "["+*name+"] ", log.LstdFlags)

	addr := fmt.Sprintf("127.0.0.1:%d", *port)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		logger.Fatal(err)
	}
	logger.Printf("listening on %s", addr)

	for {
		conn, err := ln.Accept()
		if err != nil {
			continue
		}
		go handle(conn, *name, logger)
	}
}

func handle(conn net.Conn, name string, logger *log.Logger) {
	defer conn.Close()

	lines := bufio.NewScanner(conn)
	for lines.Scan() {
		cmd := strings.TrimSpace(lines.Text())
		logger.Printf("%s", cmd)
		fmt.Fprintf(conn, "%s -> handled by %s\n", cmd, name)
	}
}
