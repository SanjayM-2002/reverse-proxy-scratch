package lb

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
)

var ErrNoHealthyBackends = errors.New("no healthy backends")

type Strategy interface {
	Name() string
	Next(p *Pool) (*Backend, error)
}

// RoundRobin cycles through healthy backends one by one.
type RoundRobin struct {
	counter atomic.Uint64
}

func (*RoundRobin) Name() string { return "round_robin" }

func (s *RoundRobin) Next(p *Pool) (*Backend, error) {
	healthy := p.Healthy()
	if len(healthy) == 0 {
		return nil, ErrNoHealthyBackends
	}
	n := s.counter.Add(1) - 1
	return healthy[n%uint64(len(healthy))], nil
}

// WeightedRoundRobin is nginx's "smooth" weighted round robin.
// With weights 5,3,1 it yields A A B A C A B A B (spread out) instead of A A A A A B B B C.
//
// Each pick: every backend's currentWeight += weight; pick the max; subtract the total from it.
type WeightedRoundRobin struct {
	mu sync.Mutex
}

func (*WeightedRoundRobin) Name() string { return "weighted_round_robin" }

func (s *WeightedRoundRobin) Next(p *Pool) (*Backend, error) {
	healthy := p.Healthy()
	if len(healthy) == 0 {
		return nil, ErrNoHealthyBackends
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	var best *Backend
	total := 0
	for _, b := range healthy {
		b.currentWeight += b.Weight
		total += b.Weight
		if best == nil || b.currentWeight > best.currentWeight {
			best = b
		}
	}
	best.currentWeight -= total
	return best, nil
}

// LeastConnections picks the healthy backend with the fewest open sessions.
// Ties go to the first in stable order.
type LeastConnections struct{}

func (LeastConnections) Name() string { return "least_connections" }

func (LeastConnections) Next(p *Pool) (*Backend, error) {
	healthy := p.Healthy()
	if len(healthy) == 0 {
		return nil, ErrNoHealthyBackends
	}
	best := healthy[0]
	for _, b := range healthy[1:] {
		if b.ActiveConns() < best.ActiveConns() {
			best = b
		}
	}
	return best, nil
}

// Strategies holds one shared instance of each, so their state (counters, weights)
// persists across connections regardless of which one a request asks for.
type Strategies struct {
	byName  map[string]Strategy
	Default Strategy
}

func NewStrategies(defaultName string) (*Strategies, error) {
	rr, wrr, lc := &RoundRobin{}, &WeightedRoundRobin{}, LeastConnections{}
	s := &Strategies{byName: map[string]Strategy{
		"round_robin": rr, "rr": rr,
		"weighted_round_robin": wrr, "wrr": wrr,
		"least_connections": lc, "lc": lc,
	}}
	d, ok := s.Lookup(defaultName)
	if !ok {
		return nil, fmt.Errorf("unknown strategy %q", defaultName)
	}
	s.Default = d
	return s, nil
}

func (s *Strategies) Lookup(name string) (Strategy, bool) {
	st, ok := s.byName[strings.ToLower(strings.TrimSpace(name))]
	return st, ok
}
