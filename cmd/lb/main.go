package main

import (
	"log"
	"net/http"
	_ "net/http/pprof"

	"github.com/lb/internal/config"
	"github.com/lb/internal/loadbalancer"
	"github.com/lb/internal/metrics"
	"github.com/lb/internal/server"
	"github.com/lb/internal/static"
)

func main() {

	go func() {
		log.Println("pprof listening on :6060")
		log.Println(http.ListenAndServe("127.0.0.1:6060", nil))
	}()

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
		cfg.RateLimiter.Name,
	)

	server.Start(
		cfg.Server.Port,
		lb,
		fs,
	)
}
