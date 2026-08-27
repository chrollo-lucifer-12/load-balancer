package main

import (
	"github.com/lb/internal/config"
	"github.com/lb/internal/loadbalancer"
	"github.com/lb/internal/middleware"
	"github.com/lb/pkg/metrics"
	"github.com/lb/pkg/ratelimiter"

	"github.com/lb/internal/server"
	"github.com/lb/internal/static"
)

func main() {

	metrics.NewMetrics()

	cfg, err := config.Load("test.yml")

	if err != nil {
		panic(err)
	}

	fs := static.NewStaticServer(
		"./public",
		"/static/",
	)

	lb := loadbalancer.NewLoadBalancer(
		cfg.VirtualHosts,
	)

	ratelimiter := ratelimiter.NewRateLimiter(ratelimiter.RateLimiterType(cfg.RateLimiter.Name))

	server.Start(
		cfg.Server.Port,
		lb,
		fs,
		//	middleware.Logger,
		middleware.Recover,
		middleware.RateLimit(ratelimiter),
		middleware.Metric,
	)
}
