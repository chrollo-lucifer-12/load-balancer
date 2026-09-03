package config

import (
	"fmt"
	"os"

	"go.yaml.in/yaml/v2"
)

type Config struct {
	Server       ServerConfig      `yaml:"server"`
	RateLimiter  RateLimiterConfig `yaml:"rate_limiter"`
	Logger       LoggerConfig      `yaml:"logger"`
	VirtualHosts []VirtualHost     `yaml:"virtual_hosts"`
}

type LoggerConfig struct {
	Enabled bool   `yaml:"enabled"`
	Level   string `yaml:"level"`
	Format  string `yaml:"format"`
	Output  string `yaml:"output"`
}

type RateLimiterConfig struct {
	Enabled bool   `yaml:"enabled"`
	Name    string `yaml:"name"`
	Rate    int    `yaml:"rate"`
	Burst   int    `yaml:"burst"`
}

type VirtualHost struct {
	Host        string            `yaml:"host"`
	Routes      []RouteConfig     `yaml:"routes"`
	Static      StaticConfig      `yaml:"static"`
	HealthCheck HealthCheckConfig `yaml:"health_check"`
}

type StaticConfig struct {
	Enabled   bool   `yaml:"enabled"`
	Directory string `yaml:"directory"`
	Path      string `yaml:"path"`
}

type RouteConfig struct {
	PathPrefix  string `yaml:"path"`
	Method      string `yaml:"method,omitempty"`
	StripPrefix string `yaml:"strip,omitempty"`

	Backends       []BackendConfig `yaml:"backends,omitempty"`
	CircuitBreaker bool            `yaml:"circuit_breaker"`
	Strategy       string          `yaml:"strategy"`
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

func DefaultConfig() Config {
	return Config{
		Server: ServerConfig{
			Host: "0.0.0.0",
			Port: ":8080",
		},
		Logger: LoggerConfig{
			Enabled: true,
			Level:   "info",
			Format:  "text",
			Output:  "stdout",
		},
		RateLimiter: RateLimiterConfig{
			Enabled: true,
			Name:    "token_bucket",
			Rate:    100,
			Burst:   200,
		},
	}
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}

	cfg := DefaultConfig()

	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	return &cfg, nil
}
