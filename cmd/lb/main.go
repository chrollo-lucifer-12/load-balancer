package main

import (
	"github.com/lb/internal/config"
	"github.com/lb/internal/loadbalancer"
	"github.com/lb/internal/middleware"
	"github.com/lb/pkg/metrics"
	"github.com/lb/pkg/ratelimiter"

	"github.com/lb/internal/server"
)

func main() {

	metrics.NewMetrics()

	cfg, err := config.Load("lb.yml")

	if err != nil {
		panic(err)
	}

	lb := loadbalancer.NewLoadBalancer(
		cfg.VirtualHosts,
	)

	ratelimiter := ratelimiter.NewRateLimiter(cfg.RateLimiter)

	middlewares := []middleware.Middleware{
		middleware.Recover,
	}

	if cfg.Logger.Enabled {
		logger := middleware.NewLogger(cfg.Logger)
		middlewares = append(middlewares, logger.Handler)
	}

	if cfg.RateLimiter.Enabled {
		middlewares = append(middlewares, middleware.RateLimit(ratelimiter))
	}

	middlewares = append(middlewares, middleware.Metric)

	server.Start(
		cfg.Server.Port,
		lb,
		middlewares...,
	)
}
