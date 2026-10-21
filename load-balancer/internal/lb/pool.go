package lb

import (
	"fmt"
	"log"
	"net"
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

type Backend struct {
	Name     string
	Addr     string // "ip:port" — also the key in the pool's map
	Weight   int
	CPUCores int
	RAMGB    int

	healthy     atomic.Bool
	activeConns atomic.Int64 // open client<->backend sessions, used by least connections

	// smooth weighted round robin state, guarded by WeightedRoundRobin.mu
	currentWeight int
}

func (b *Backend) Healthy() bool      { return b.healthy.Load() }
func (b *Backend) ActiveConns() int64 { return b.activeConns.Load() }

// Pool is the in-memory hashmap of servers, keyed by "ip:port".
type Pool struct {
	mu      sync.RWMutex
	servers map[string]*Backend
	order   []string // stable key order so round robin is deterministic (map iteration is random)
}

func NewPool(cfgs []ServerConfig) *Pool {
	p := &Pool{servers: make(map[string]*Backend, len(cfgs))}
	for _, c := range cfgs {
		addr := net.JoinHostPort(c.Host, fmt.Sprint(c.Port))
		w := c.Weight
		if w <= 0 {
			w = max(c.CPUCores, 1)
		}
		b := &Backend{Name: c.Name, Addr: addr, Weight: w, CPUCores: c.CPUCores, RAMGB: c.RAMGB}
		b.healthy.Store(true) // optimistic until the first health check says otherwise
		p.servers[addr] = b
		p.order = append(p.order, addr)
	}
	sort.Strings(p.order)
	return p
}

func (p *Pool) Get(addr string) (*Backend, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	b, ok := p.servers[addr]
	return b, ok
}

// Healthy returns healthy backends in stable order.
func (p *Pool) Healthy() []*Backend {
	p.mu.RLock()
	defer p.mu.RUnlock()
	out := make([]*Backend, 0, len(p.order))
	for _, k := range p.order {
		if b := p.servers[k]; b.Healthy() {
			out = append(out, b)
		}
	}
	return out
}

func (p *Pool) All() []*Backend {
	p.mu.RLock()
	defer p.mu.RUnlock()
	out := make([]*Backend, 0, len(p.order))
	for _, k := range p.order {
		out = append(out, p.servers[k])
	}
	return out
}

func (p *Pool) SetHealthy(b *Backend, healthy bool) {
	if b.healthy.Swap(healthy) != healthy {
		state := "DOWN"
		if healthy {
			state = "UP"
		}
		log.Printf("[health] %s (%s) is %s", b.Name, b.Addr, state)
	}
}

// StartHealthChecks runs one check immediately, then keeps checking every interval in the background.
func (p *Pool) StartHealthChecks(interval time.Duration) {
	p.checkAll()
	go p.healthCheckLoop(interval)
}

func (p *Pool) healthCheckLoop(interval time.Duration) {
	t := time.NewTicker(interval)
	defer t.Stop()
	for range t.C {
		p.checkAll()
	}
}

// checkAll does a plain TCP dial against each backend.
func (p *Pool) checkAll() {
	for _, b := range p.All() {
		conn, err := net.DialTimeout("tcp", b.Addr, time.Second)
		if err == nil {
			conn.Close()
		}
		p.SetHealthy(b, err == nil)
	}
}
