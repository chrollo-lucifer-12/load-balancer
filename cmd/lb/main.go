package main

import (
	"time"

	"github.com/lb/internal/loadbalancer"
)

func main() {
	backends := []string{
		"http://localhost:8081",
		"http://localhost:8082",
		"http://localhost:8083",
	}

	lb := loadbalancer.NewLoadBalancer(backends, 30*time.Second, 3)
	// server := http.Server{
	// 	Addr:    ":8080",
	// 	Handler: lb,
	// }

	// log.Printf("Starting load balancer on :8080")
	// log.Fatal(server.ListenAndServe())
}
