package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server    ServerConfig    `yaml:"server"`
	Upstream  UpstreamConfig  `yaml:"upstream"`
	Database  DatabaseConfig  `yaml:"database"`
	Auth      AuthConfig      `yaml:"auth"`
	Endpoints EndpointConfig  `yaml:"endpoints"`
}

type ServerConfig struct {
	Listen string `yaml:"listen"`
}

type UpstreamConfig struct {
	URL     string        `yaml:"url"`
	Timeout time.Duration `yaml:"timeout"`
}

type DatabaseConfig struct {
	URL string `yaml:"url"`
}

type AuthConfig struct {
	TokenHashCost int `yaml:"token_hash_cost"`
}

type EndpointConfig struct {
	Read  []string `yaml:"read"`
	Write []string `yaml:"write"`
}

func Load(path string) (*Config, error) {
	cfg := &Config{
		Server: ServerConfig{
			Listen: ":8080",
		},
		Upstream: UpstreamConfig{
			Timeout: 30 * time.Second,
		},
		Auth: AuthConfig{
			TokenHashCost: 10,
		},
		Endpoints: EndpointConfig{
			Read:  []string{"GET /**"},
			Write: []string{"POST /**", "PUT /**", "DELETE /**"},
		},
	}

	if path != "" {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("reading config file: %w", err)
		}

		if err := yaml.Unmarshal(data, cfg); err != nil {
			return nil, fmt.Errorf("parsing config file: %w", err)
		}
	}

	// Override with environment variables if set
	if v := os.Getenv("LISTEN_ADDR"); v != "" {
		cfg.Server.Listen = v
	}
	if v := os.Getenv("UPSTREAM_URL"); v != "" {
		cfg.Upstream.URL = v
	}
	if v := os.Getenv("DATABASE_URL"); v != "" {
		cfg.Database.URL = v
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

func (c *Config) validate() error {
	if c.Upstream.URL == "" {
		return fmt.Errorf("upstream URL is required")
	}
	if c.Database.URL == "" {
		return fmt.Errorf("database URL is required")
	}
	return nil
}
