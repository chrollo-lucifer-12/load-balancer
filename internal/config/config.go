package config

import (
	"os"

	"go.yaml.in/yaml/v2"
)

type Config struct {
	Server       ServerConfig       `yaml:"server"`
	LoadBalancer LoadBalancerConfig `yaml:"load_balancer"`
	Backends     []BackendConfig    `yaml:"backends"`
}

type ServerConfig struct {
	Host string `yaml:"host"`
	Port string `yaml:"port"`
}

type LoadBalancerConfig struct {
	Strategy    string            `yaml:"strategy"`
	HealthCheck HealthCheckConfig `yaml:"health_check"`
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
