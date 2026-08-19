package main

import (
	"log"
	"net/http"
	"time"

	"github.com/lb/internal/loadbalancer"
)

func main() {
	backends := []string{
		"http://localhost:8081",
		"http://localhost:8082",
		"http://localhost:8083",
	}

	lb := loadbalancer.NewLoadBalancer(backends, 30*time.Second, 3, 1)

	log.Println("Load balancer listening on :8080")

	if err := http.ListenAndServe(":8080", http.HandlerFunc(lb.ServerHTTP)); err != nil {
		log.Fatal(err)
	}
}
