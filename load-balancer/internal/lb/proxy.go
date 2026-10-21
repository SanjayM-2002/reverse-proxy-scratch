package lb

import (
	"bufio"
	"io"
	"log"
	"net"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// Session is one entry in the connection map:
// client --(conn A)--> LB --(conn B)--> backend.
type Session struct {
	ID         uint64
	ClientAddr string // client's ip:port on conn A
	LBInAddr   string // LB's listen addr on conn A
	LBOutAddr  string // LB's ephemeral ip:port on conn B
	Backend    *Backend
	Strategy   string
	Started    time.Time

	client, backend net.Conn
}

type Proxy struct {
	pool       *Pool
	strategies *Strategies

	nextID   atomic.Uint64
	mu       sync.Mutex
	sessions map[uint64]*Session // client conn <-> backend conn mapping
}

func NewProxy(pool *Pool, strategies *Strategies) *Proxy {
	return &Proxy{pool: pool, strategies: strategies, sessions: make(map[uint64]*Session)}
}

func (p *Proxy) ListenAndServe(addr string) error {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	log.Printf("[lb] listening on %s (default strategy: %s)", addr, p.strategies.Default.Name())
	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Printf("[lb] accept: %v", err)
			continue
		}
		go p.handle(conn)
	}
}

// Sessions returns a snapshot of currently open sessions.
func (p *Proxy) Sessions() []Session {
	p.mu.Lock()
	defer p.mu.Unlock()
	out := make([]Session, 0, len(p.sessions))
	for _, s := range p.sessions {
		out = append(out, *s)
	}
	return out
}

func (p *Proxy) handle(client net.Conn) {
	defer client.Close()

	// Wrap the client conn so we can peek the first bytes for ?strategy= without consuming them.
	// Whatever we peek stays in the bufio buffer and is forwarded to the backend as-is.
	br := bufio.NewReader(client)
	strategy := p.pickStrategy(client, br)

	backend, beConn, err := p.dial(strategy)
	if err != nil {
		log.Printf("[lb] %s: %v", client.RemoteAddr(), err)
		return
	}
	defer beConn.Close()
	backend.activeConns.Add(1)
	defer backend.activeConns.Add(-1)

	s := &Session{
		ID:         p.nextID.Add(1),
		ClientAddr: client.RemoteAddr().String(),
		LBInAddr:   client.LocalAddr().String(),
		LBOutAddr:  beConn.LocalAddr().String(),
		Backend:    backend,
		Strategy:   strategy.Name(),
		Started:    time.Now(),
		client:     client,
		backend:    beConn,
	}
	p.mu.Lock()
	p.sessions[s.ID] = s
	p.mu.Unlock()
	defer func() {
		p.mu.Lock()
		delete(p.sessions, s.ID)
		p.mu.Unlock()
	}()

	log.Printf("[lb] conn#%d OPEN  client %s -> lb %s | lb %s -> %s (%s) strategy=%s active=%d",
		s.ID, s.ClientAddr, s.LBInAddr, s.LBOutAddr, backend.Name, backend.Addr, s.Strategy, backend.ActiveConns())

	up, down := pipe(beConn, br, client)

	log.Printf("[lb] conn#%d CLOSE %s took=%s up=%dB down=%dB",
		s.ID, backend.Name, time.Since(s.Started).Round(time.Millisecond), up, down)
}

// dial picks a backend and connects, retrying on others if a dial fails.
func (p *Proxy) dial(strategy Strategy) (*Backend, net.Conn, error) {
	var lastErr error = ErrNoHealthyBackends
	for range len(p.pool.All()) {
		b, err := strategy.Next(p.pool)
		if err != nil {
			return nil, nil, err
		}
		conn, err := net.DialTimeout("tcp", b.Addr, 2*time.Second)
		if err == nil {
			return b, conn, nil
		}
		log.Printf("[lb] dial %s (%s) failed: %v — marking down", b.Name, b.Addr, err)
		p.pool.SetHealthy(b, false)
		lastErr = err
	}
	return nil, nil, lastErr
}

// pickStrategy peeks at whatever the client sent first. If it looks like an HTTP request line
// with ?strategy=<name>, use that; otherwise use the configured default. This is a demo
// convenience — a real L4 LB never looks at the payload.
func (p *Proxy) pickStrategy(client net.Conn, br *bufio.Reader) Strategy {
	client.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
	defer client.SetReadDeadline(time.Time{})

	if _, err := br.Peek(1); err != nil { // blocks until the first packet arrives (or deadline)
		return p.strategies.Default
	}
	buf, _ := br.Peek(br.Buffered())
	line, _, _ := strings.Cut(string(buf), "\r\n")
	parts := strings.Fields(line) // e.g. ["GET", "/hello?strategy=wrr", "HTTP/1.1"]
	if len(parts) < 2 {
		return p.strategies.Default
	}
	u, err := url.ParseRequestURI(parts[1])
	if err != nil {
		return p.strategies.Default
	}
	if st, ok := p.strategies.Lookup(u.Query().Get("strategy")); ok {
		return st
	}
	return p.strategies.Default
}

// pipe copies bytes both ways until both directions finish. When one side stops sending,
// we half-close the other side's write so it sees EOF but can still send its reply.
func pipe(backend net.Conn, clientR io.Reader, client net.Conn) (up, down int64) {
	upCh, downCh := make(chan int64, 1), make(chan int64, 1)
	go copyHalf(backend, clientR, upCh)  // client -> backend
	go copyHalf(client, backend, downCh) // backend -> client
	return <-upCh, <-downCh
}

// copyHalf copies one direction until src hits EOF, half-closes dst, and reports bytes copied.
func copyHalf(dst net.Conn, src io.Reader, copied chan<- int64) {
	n, _ := io.Copy(dst, src)
	closeWrite(dst)
	copied <- n
}

func closeWrite(c net.Conn) {
	if tc, ok := c.(*net.TCPConn); ok {
		tc.CloseWrite()
		return
	}
	c.Close()
}
