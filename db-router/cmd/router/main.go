package main

import (
	"flag"
	"log"

	"github.com/SanjayM-2002/reverse-proxy-scratch/db-router/internal/router"
)

func main() {
	configPath := flag.String("config", "db-router/config/replica.json", "topology config: replica.json or sharded.json")
	flag.Parse()

	cfg, err := router.LoadConfig(*configPath)
	if err != nil {
		log.Fatal(err)
	}

	r, err := router.NewRouter(cfg)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("[router] mode=%s config=%s", cfg.Mode, *configPath)

	log.Fatal(router.NewProxy(r).ListenAndServe(cfg.Listen))
}
