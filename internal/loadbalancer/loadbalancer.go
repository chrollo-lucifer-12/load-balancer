package loadbalancer

import (
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/lb/internal/backend"
)

type Strategy int

const (
	RoundRobin Strategy = iota
)

type LoadBalancer struct {
	backends []*backend.Backend
	current  int

	mu sync.Mutex

	healthCheckInterval time.Duration

	maxFailCount int

	strategy Strategy
}

func NewLoadBalancer(backendURLs []string, healthCheckInterval time.Duration, maxFailCount int, strategy Strategy) *LoadBalancer {
	backends := make([]*backend.Backend, len(backendURLs))

	for i, rawURL := range backendURLs {
		backend, err := backend.NewBackend(rawURL)
		if err != nil {
			return nil
		}

		backends[i] = backend
	}

	lb := &LoadBalancer{
		backends:            backends,
		healthCheckInterval: healthCheckInterval,
		maxFailCount:        maxFailCount,
		strategy:            strategy,
	}

	go lb.healthCheck()

	return lb
}

func (lb *LoadBalancer) ServerHTTP(w http.ResponseWriter, r *http.Request) {
	backend := lb.chooseBackend()
	if backend == nil {
		http.Error(w, "No available backends", http.StatusServiceUnavailable)
		return
	}

	backend.IncreaseActive()

	log.Printf("forwarding request to: %s", backend.URL)

	backend.ReverseProxy.ServeHTTP(w, r)
}
