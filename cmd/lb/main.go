package main

import (
	"net/http"
	_ "net/http/pprof"

	"github.com/lb/internal/config"
	"github.com/lb/internal/loadbalancer"
	"github.com/lb/internal/middleware"
	"github.com/lb/pkg/metrics"
	"github.com/lb/pkg/ratelimiter"

	"github.com/lb/internal/server"
)

func main() {

	go func() {
		http.ListenAndServe(":6060", nil)
	}()

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

	// if cfg.Logger.Enabled {
	// 	logger := middleware.NewLogger(cfg.Logger)
	// 	middlewares = append(middlewares, logger.Handler)
	// }

	if cfg.RateLimiter.Enabled {
		middlewares = append(middlewares, middleware.RateLimit(ratelimiter))
	}

	//	middlewares = append(middlewares, middleware.Metric)

	server.Start(
		cfg.Server.Port,
		lb,
		middlewares...,
	)
}
