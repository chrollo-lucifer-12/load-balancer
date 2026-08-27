package vhost

import (
	"net/http"
	"time"

	"github.com/lb/internal/backend"
	"github.com/lb/internal/config"
	"github.com/lb/internal/rw"
	"github.com/lb/internal/selector"
	"github.com/lb/pkg/circuitbreaker"
)

type VHost struct {
	backends []*backend.Backend

	sl selector.Selector

	healthCheckInterval time.Duration

	cb *circuitbreaker.CircuitBreaker
}

func NewVHost(vhostConfig config.VirtualHost) *VHost {

	sl := selector.NewSelector(selector.SelectorType(vhostConfig.Strategy))
	maxFailCount := vhostConfig.HealthCheck.MaxFailures
	healthCheckInterval := vhostConfig.HealthCheck.Interval

	vh := &VHost{
		sl:                  sl,
		cb:                  circuitbreaker.NewCircuitBreaker(),
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

	go vh.healthCheck()

	return vh
}

func (vh *VHost) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	if !vh.cb.CanPass() {
		http.Error(w, "Service unavailable", http.StatusServiceUnavailable)
		return
	}

	attempts := 3

	for i := 1; i <= attempts; i++ {
		backend := vh.choose(r.RemoteAddr)
		if backend == nil {
			http.Error(w, "No available backends", http.StatusServiceUnavailable)
			return
		}

		backend.IncrementActive()

		backend.ServeHTTP(w, r)

		rec := w.(*rw.ResponseWrapper)

		backend.DecrementActive()

		reqFailed := rec.Status >= 500

		if !reqFailed {
			vh.cb.OnSuccess()
		}

		if reqFailed && isIdempotent(r) {
			rec.Reset()
			continue
		}

		vh.cb.OnFailure()
		return
	}

	vh.cb.OnFailure()
	http.Error(w, "All backend retry attempts failed", http.StatusBadGateway)
}

func isIdempotent(r *http.Request) bool {
	return r.Method == "GET" || r.Method == "HEAD" || r.Method == "PUT" || r.Method == "DELETE"
}

func (vh *VHost) choose(key string) *backend.Backend {
	return vh.sl.Choose(vh.backends, key)
}
