package main

import (
	"log"
	"net/http"
	"time"

	"github.com/lb/internal/config"
	"github.com/lb/internal/loadbalancer"
)

func main() {
	cfg, err := config.Load("test.yml")
	if err != nil {
		panic(err)
	}

	maxFailCount := cfg.LoadBalancer.HealthCheck.MaxFailures
	interval := cfg.LoadBalancer.HealthCheck.Interval

	lb := loadbalancer.NewLoadBalancer(cfg.Backends, time.Duration(interval)*time.Second, int64(maxFailCount), 1)

	log.Println("Load balancer listening on ", cfg.Server.Port)

	if err := http.ListenAndServe(cfg.Server.Port, http.HandlerFunc(lb.ServerHTTP)); err != nil {
		log.Fatal(err)
	}
}
