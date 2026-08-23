package loadbalancer

import (
	"net/http"
	"time"

	"github.com/lb/internal/backend"
	"github.com/lb/internal/config"
	"github.com/lb/internal/middleware"
	"github.com/lb/internal/rl"
	"github.com/lb/internal/rw"
	"github.com/lb/internal/selector"
)

type LoadBalancer struct {
	backends []*backend.Backend
	current  int

	healthCheckInterval time.Duration

	maxFailCount int64

	sl selector.Selector

	mux http.Handler
}

func NewLoadBalancer(backendConfigs []config.BackendConfig, healthCheckInterval time.Duration, maxFailCount int64, limiter rl.RateLimiter, sl selector.Selector) *LoadBalancer {

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
		sl:                  sl,
	}

	mux := http.NewServeMux()

	handler := middleware.Chain(middleware.Recover,
		middleware.Buffer,
		middleware.Metric,
		middleware.Logger,
		middleware.RateLimit(limiter))(http.HandlerFunc(lb.serveHTTP))

	mux.Handle("/", handler)

	lb.mux = mux

	go lb.healthCheck()

	return lb
}

func (lb *LoadBalancer) Run(w http.ResponseWriter, r *http.Request) {
	lb.mux.ServeHTTP(w, r)
}

func (lb *LoadBalancer) serveHTTP(w http.ResponseWriter, r *http.Request) {

	attempts := 3

	for i := 1; i <= attempts; i++ {
		backend := lb.sl.Choose(lb.backends, r.RemoteAddr)
		if backend == nil {
			http.Error(w, "No available backends", http.StatusServiceUnavailable)
			return
		}

		backend.IncrementActive()

		backend.ServeHTTP(w, r)

		rec := w.(*rw.ResponseWrapper)

		backend.DecrementActive()

		if rec.Status >= 500 && isIdempotent(r) {
			rec.Reset()
			continue
		}

		return
	}

	http.Error(w, "All backend retry attempts failed", http.StatusBadGateway)
}

func isIdempotent(r *http.Request) bool {
	return r.Method == "GET" || r.Method == "HEAD" || r.Method == "PUT" || r.Method == "DELETE"
}
