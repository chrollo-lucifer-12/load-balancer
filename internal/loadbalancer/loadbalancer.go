package loadbalancer

import (
	"context"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/lb/internal/backend"
	"github.com/lb/internal/config"
)

type Strategy int

const (
	RoundRobin Strategy = iota
	PowerOfTwo
)

type LoadBalancer struct {
	backends []*backend.Backend
	current  int

	mu sync.Mutex

	healthCheckInterval time.Duration

	maxFailCount int64

	strategy Strategy
}

func NewLoadBalancer(backendConfigs []config.BackendConfig, healthCheckInterval time.Duration, maxFailCount int64, strategy Strategy) *LoadBalancer {

	backends := make([]*backend.Backend, len(backendConfigs))

	for i, backendConfig := range backendConfigs {
		backend, err := backend.NewBackend(backendConfig.URL, backendConfig.Weight, maxFailCount)
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

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	r = r.WithContext(ctx)

	attempts := 3

	rec := NewResponseBuffer(w)

	for i := 1; i <= attempts; i++ {
		backend := lb.chooseBackend()
		if backend == nil {
			http.Error(w, "No available backends", http.StatusServiceUnavailable)
			return
		}

		backend.IncrementActive()

		log.Printf("forwarding request to: %s", backend.URL)

		backend.ReverseProxy.ServeHTTP(rec, r)

		backend.DecrementActive()

		if rec.status >= 500 && isIdempotent(r) {
			rec.Reset()
			continue
		}

		rec.Flush()
		return
	}

	http.Error(w, "All backend retry attempts failed", http.StatusBadGateway)
}

func isIdempotent(r *http.Request) bool {
	return r.Method == "GET" || r.Method == "HEAD" || r.Method == "PUT" || r.Method == "DELETE"
}
