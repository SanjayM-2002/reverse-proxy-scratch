package router

import (
	"encoding/json"
	"fmt"
	"os"
)

// Config is the router's topology. Which fields are used depends on Mode.
type Config struct {
	Listen string `json:"listen"`
	Mode   string `json:"mode"` // "replica" or "sharded"

	// mode = replica
	Primary  Target   `json:"primary"`
	Replicas []Target `json:"replicas"`

	// mode = sharded
	Hashing string   `json:"hashing"` // "modulo" or "consistent"
	Shards  []Target `json:"shards"`
}

func LoadConfig(path string) (*Config, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	if cfg.Listen == "" {
		cfg.Listen = ":7070"
	}
	return &cfg, nil
}

// NewRouter builds the Router described by the config.
func NewRouter(cfg *Config) (Router, error) {
	switch cfg.Mode {
	case "replica":
		if cfg.Primary.Addr == "" {
			return nil, fmt.Errorf("replica mode needs a primary")
		}
		return NewReplicaRouter(cfg.Primary, cfg.Replicas), nil

	case "sharded":
		if len(cfg.Shards) == 0 {
			return nil, fmt.Errorf("sharded mode needs at least one shard")
		}
		switch cfg.Hashing {
		case "modulo":
			return NewShardRouter(cfg.Shards, NewModuloPicker(len(cfg.Shards))), nil
		case "consistent", "":
			return NewShardRouter(cfg.Shards, NewConsistentHashPicker(cfg.Shards)), nil
		default:
			return nil, fmt.Errorf("unknown hashing %q (use modulo or consistent)", cfg.Hashing)
		}

	default:
		return nil, fmt.Errorf("unknown mode %q (use replica or sharded)", cfg.Mode)
	}
}
