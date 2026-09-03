package vhost

import (
	"net/http"
	"strings"

	"github.com/lb/internal/backend"
	"github.com/lb/internal/config"
	"github.com/lb/internal/selector"
	"github.com/lb/pkg/circuitbreaker"
)

type Route struct {
	PathPrefix  string
	Method      string
	backends    []*backend.Backend
	StripPrefix string

	selector selector.Selector
	cb       *circuitbreaker.CircuitBreaker
}

func NewRoute(routeConfig config.RouteConfig, maxFailCount int64) *Route {
	sl := selector.NewSelector(selector.SelectorType(routeConfig.Strategy))

	r := &Route{
		PathPrefix:  routeConfig.PathPrefix,
		Method:      routeConfig.Method,
		StripPrefix: routeConfig.StripPrefix,
		selector:    sl,
	}

	if routeConfig.CircuitBreaker {
		r.cb = circuitbreaker.NewCircuitBreaker()
	}

	backends := make([]*backend.Backend, len(routeConfig.Backends))

	for i, backendConfig := range routeConfig.Backends {

		var onResponse func(int)

		if r.cb != nil {
			onResponse = func(status int) {
				if status >= 500 {
					r.cb.OnFailure()
				} else {
					r.cb.OnSuccess()
				}
			}
		}

		b, err := backend.NewBackend(
			backendConfig.URL,
			backendConfig.Weight,
			maxFailCount,
			onResponse,
		)

		if err != nil {
			return nil
		}

		backends[i] = b

	}

	r.backends = backends

	return r
}

func (rt *Route) ServeHTTP(
	w http.ResponseWriter,
	r *http.Request,
) {

	if rt.cb != nil && !rt.cb.CanPass() {
		http.Error(
			w,
			"Service unavailable",
			http.StatusServiceUnavailable,
		)
		return
	}

	rt.serve(w, r)
}

func (rt *Route) serve(
	w http.ResponseWriter,
	r *http.Request,
) {
	const attempts = 1

	for i := 0; i < attempts; i++ {
		b := rt.choose(w, r)

		if b == nil {
			http.Error(
				w,
				"No available backends",
				http.StatusServiceUnavailable,
			)
			return
		}

		if rt.StripPrefix != "" {
			r.URL.Path = strings.TrimPrefix(
				r.URL.Path,
				rt.StripPrefix,
			)

			if r.URL.Path == "" {
				r.URL.Path = "/"
			}
		}

		b.ServeHTTP(w, r)

	}
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

	return true
}
