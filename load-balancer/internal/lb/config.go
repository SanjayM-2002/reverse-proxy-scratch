package lb

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// ServerConfig is one row of what would normally live in a DB table.
type ServerConfig struct {
	Name     string `json:"name"`
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Weight   int    `json:"weight"`    // used by weighted round robin; falls back to cpu_cores if 0
	CPUCores int    `json:"cpu_cores"` // informational / weight fallback
	RAMGB    int    `json:"ram_gb"`    // informational
}

type Config struct {
	Listen              string         `json:"listen"`
	DefaultStrategy     string         `json:"default_strategy"`
	HealthCheckInterval string         `json:"health_check_interval"`
	Servers             []ServerConfig `json:"servers"`
}

func (c *Config) HealthInterval() time.Duration {
	d, err := time.ParseDuration(c.HealthCheckInterval)
	if err != nil || d <= 0 {
		return 5 * time.Second
	}
	return d
}

// LoadConfig stands in for "read servers from the DB at boot".
func LoadConfig(path string) (*Config, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}
	var cfg Config
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	if len(cfg.Servers) == 0 {
		return nil, fmt.Errorf("config has no servers")
	}
	if cfg.Listen == "" {
		cfg.Listen = ":8080"
	}
	if cfg.DefaultStrategy == "" {
		cfg.DefaultStrategy = "round_robin"
	}
	return &cfg, nil
}
