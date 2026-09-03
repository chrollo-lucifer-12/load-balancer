package vhost

import (
	"net/http"
	"strings"

	"github.com/lb/internal/backend"
	"github.com/lb/internal/config"
	"github.com/lb/internal/middleware"
	"github.com/lb/internal/selector"
	"github.com/lb/pkg/circuitbreaker"
)

type Route struct {
	PathPrefix  string
	Method      string
	Headers     map[string]string
	backends    []*backend.Backend
	StripPrefix string

	selector selector.Selector
	handler  http.Handler
}

func NewRoute(routeConfig config.RuleConfig, maxFailCount int64) *Route {
	sl := selector.NewSelector(selector.SelectorType(routeConfig.Strategy))

	r := &Route{
		PathPrefix:  routeConfig.PathPrefix,
		Method:      routeConfig.Method,
		Headers:     routeConfig.Headers,
		StripPrefix: routeConfig.StripPrefix,
		selector:    sl,
	}

	backends := make([]*backend.Backend, len(routeConfig.Backends))

	for i, backendConfig := range routeConfig.Backends {
		backend, err := backend.NewBackend(backendConfig.URL, backendConfig.Weight, maxFailCount)
		if err != nil {
			return nil
		}

		backends[i] = backend
	}

	r.backends = backends

	var handler http.Handler = http.HandlerFunc(r.serve)

	if routeConfig.CircuitBreaker {
		cb := circuitbreaker.NewCircuitBreaker()

		handler = middleware.NewCircuitBreakerMiddleware(cb, handler)
	}

	r.handler = handler

	return r
}

func (rt *Route) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	rt.handler.ServeHTTP(w, r)
}

func (rt *Route) serve(w http.ResponseWriter, r *http.Request) {
	backend := rt.choose(w, r)

	if backend == nil {
		http.Error(
			w,
			"No available backends",
			http.StatusServiceUnavailable,
		)
		return
	}

	backend.IncrementActive()
	defer backend.DecrementActive()

	if rt.StripPrefix != "" {
		r.URL.Path = strings.TrimPrefix(
			r.URL.Path,
			rt.StripPrefix,
		)

		if r.URL.Path == "" {
			r.URL.Path = "/"
		}
	}

	backend.ServeHTTP(w, r)
}

func (rt *Route) choose(w http.ResponseWriter, r *http.Request) *backend.Backend {
	return rt.selector.Choose(rt.backends, w, r)
}

func (rt *Route) Matches(r *http.Request) bool {
	if rt.PathPrefix != "" &&
		!strings.HasPrefix(r.URL.Path, rt.PathPrefix) {
		return false
	}

	if rt.Method != "" && rt.Method != r.Method {
		return false
	}

	for key, value := range rt.Headers {
		if r.Header.Get(key) != value {
			return false
		}
	}

	return true
}
