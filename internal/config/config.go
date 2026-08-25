package config

import (
	"os"

	"go.yaml.in/yaml/v2"
)

type Config struct {
	Server       ServerConfig      `yaml:"server"`
	RateLimiter  RateLimiterConfig `yaml:"rate_limiter"`
	VirtualHosts []VirtualHost     `yaml:"virtual_hosts"`
}

type RateLimiterConfig struct {
	Name string `yaml:"name"`
}

type VirtualHost struct {
	Host        string            `yaml:"host"`
	Backends    []BackendConfig   `yaml:"backends"`
	Strategy    string            `yaml:"strategy"`
	HealthCheck HealthCheckConfig `yaml:"health_check"`
}

type ServerConfig struct {
	Host string `yaml:"host"`
	Port string `yaml:"port"`
}

type HealthCheckConfig struct {
	Enabled     bool `yaml:"enabled"`
	Interval    int  `yaml:"interval"`
	Timeout     int  `yaml:"timeout"`
	MaxFailures int  `yaml:"max_failures"`
}

type BackendConfig struct {
	Name   string `yaml:"name"`
	URL    string `yaml:"url"`
	Weight int    `yaml:"weight"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg Config

	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}
