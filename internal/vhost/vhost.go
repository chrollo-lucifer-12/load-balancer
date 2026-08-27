package vhost

import (
	"net/http"
	"time"

	"github.com/lb/internal/backend"
	"github.com/lb/internal/config"
	"github.com/lb/internal/middleware"
	"github.com/lb/internal/rw"
	"github.com/lb/internal/selector"
	"github.com/lb/pkg/circuitbreaker"
)

type VHost struct {
	handler http.Handler

	backends []*backend.Backend
	sl       selector.Selector

	healthCheckInterval time.Duration
}

func NewVHost(vhostConfig config.VirtualHost) *VHost {

	sl := selector.NewSelector(selector.SelectorType(vhostConfig.Strategy))
	maxFailCount := vhostConfig.HealthCheck.MaxFailures
	healthCheckInterval := vhostConfig.HealthCheck.Interval

	vh := &VHost{
		sl:                  sl,
		healthCheckInterval: time.Duration(healthCheckInterval) * time.Second,
	}

	backends := make([]*backend.Backend, len(vhostConfig.Backends))

	for i, backendConfig := range vhostConfig.Backends {
		backend, err := backend.NewBackend(backendConfig.URL, backendConfig.Weight, int64(maxFailCount))
		if err != nil {
			return nil
		}

		backends[i] = backend
	}

	vh.backends = backends

	var handler http.Handler = http.HandlerFunc(vh.serve)

	cb := circuitbreaker.NewCircuitBreaker()

	handler = middleware.NewCircuitBreakerMiddleware(cb, handler)

	vh.handler = handler

	go vh.healthCheck()

	return vh
}

func (vh *VHost) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	vh.handler.ServeHTTP(w, r)
}

func (vh *VHost) serve(w http.ResponseWriter, r *http.Request) {

	attempts := 3

	for i := 1; i <= attempts; i++ {
		backend := vh.choose(r.RemoteAddr)
		if backend == nil {
			http.Error(w, "No available backends", http.StatusServiceUnavailable)
			return
		}

		rec := w.(*rw.ResponseWrapper)

		backend.IncrementActive()

		backend.ServeHTTP(rec, r)

		backend.DecrementActive()

		failed := rec.Status >= 500

		if failed && isIdempotent(r) && i < attempts {
			rec.Reset()
			continue
		}

		return
	}

}

func isIdempotent(r *http.Request) bool {
	return r.Method == "GET" || r.Method == "HEAD" || r.Method == "PUT" || r.Method == "DELETE"
}

func (vh *VHost) choose(key string) *backend.Backend {
	return vh.sl.Choose(vh.backends, key)
}
