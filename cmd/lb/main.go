package main

import (
	_ "net/http/pprof"

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

	server.Start(
		cfg.Server.Port,
		lb,
		middleware.Recover,
		middleware.Logger,
		middleware.RateLimit(ratelimiter),
		middleware.Metric,
	)
}
