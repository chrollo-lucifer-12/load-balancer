package main

import (
	"log"
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
		log.Println("pprof listening on :6060")

		if err := http.ListenAndServe(":6060", nil); err != nil {
			log.Fatal(err)
		}
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

	server.Start(
		cfg.Server.Port,
		lb,
		middleware.Recover,
		middleware.Logger,
		middleware.RateLimit(ratelimiter),
		middleware.Metric,
	)
}
