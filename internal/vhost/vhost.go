package vhost

import (
	"net/http"
	"strings"
	"time"

	"github.com/lb/internal/config"
	"github.com/lb/internal/middleware"
	"github.com/lb/internal/rw"
	"github.com/lb/internal/selector"
	"github.com/lb/pkg/circuitbreaker"
)

type VHost struct {
	handler http.Handler
	static  http.Handler

	routes []*Route
	sl     selector.Selector

	enabled             bool
	healthCheckInterval time.Duration
	timeout             time.Duration
}

func NewVHost(vhostConfig config.VirtualHost) *VHost {

	maxFailCount := vhostConfig.HealthCheck.MaxFailures
	healthCheckInterval := vhostConfig.HealthCheck.Interval

	vh := &VHost{
		enabled:             vhostConfig.HealthCheck.Enabled,
		healthCheckInterval: time.Duration(healthCheckInterval) * time.Second,
		timeout:             time.Duration(vhostConfig.HealthCheck.Timeout) * time.Second,
	}

	if vhostConfig.Static.Enabled {
		vh.static = http.StripPrefix(
			vhostConfig.Static.Path,
			http.FileServer(
				http.Dir(vhostConfig.Static.Directory),
			),
		)
	}

	routes := make([]*Route, len(vhostConfig.Rules))

	for i, routeConfig := range vhostConfig.Rules {

		route := NewRoute(routeConfig, int64(maxFailCount))

		routes[i] = route
	}

	var handler http.Handler = http.HandlerFunc(vh.serve)

	cb := circuitbreaker.NewCircuitBreaker()

	handler = middleware.NewCircuitBreakerMiddleware(cb, handler)

	vh.handler = handler

	if vh.enabled {
		go vh.healthCheck()
	}

	return vh
}

func (vh *VHost) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	rec := rw.NewResponseWrapper(w)
	vh.handler.ServeHTTP(rec, r)
}

func (vh *VHost) serve(w http.ResponseWriter, r *http.Request) {

	if vh.static != nil && strings.HasPrefix(r.URL.Path, "/static") {
		vh.static.ServeHTTP(w, r)
		return
	}

	route := vh.matchRoute(r)

	if route == nil {
		http.NotFound(w, r)
		return
	}

	attempts := 3

	for i := 1; i <= attempts; i++ {

		backend := route.choose(w, r)
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
			continue
		}

		return
	}

}

func (v *VHost) matchRoute(r *http.Request) *Route {
	for _, route := range v.routes {
		if route.Matches(r) {
			return route
		}
	}

	return nil
}

func isIdempotent(r *http.Request) bool {
	return r.Method == "GET" || r.Method == "HEAD" || r.Method == "PUT" || r.Method == "DELETE"
}
