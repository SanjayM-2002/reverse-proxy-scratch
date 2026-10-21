package main

import (
	"flag"
	"log"

	"github.com/SanjayM-2002/reverse-proxy-scratch/load-balancer/internal/lb"
)

func main() {
	configPath := flag.String("config", "load-balancer/config/servers.json", "path to servers config (stand-in for the DB)")
	strategy := flag.String("strategy", "", "default strategy override: round_robin|rr, weighted_round_robin|wrr, least_connections|lc")
	flag.Parse()

	cfg, err := lb.LoadConfig(*configPath)
	if err != nil {
		log.Fatal(err)
	}
	if *strategy != "" {
		cfg.DefaultStrategy = *strategy
	}

	pool := lb.NewPool(cfg.Servers)
	for _, b := range pool.All() {
		log.Printf("[boot] loaded %s addr=%s weight=%d cpu=%d ram=%dGB", b.Name, b.Addr, b.Weight, b.CPUCores, b.RAMGB)
	}
	pool.StartHealthChecks(cfg.HealthInterval())

	strategies, err := lb.NewStrategies(cfg.DefaultStrategy)
	if err != nil {
		log.Fatal(err)
	}

	log.Fatal(lb.NewProxy(pool, strategies).ListenAndServe(cfg.Listen))
}
