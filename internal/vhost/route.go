package vhost

import (
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/lb/internal/backend"
	"github.com/lb/internal/config"
	"github.com/lb/internal/middleware"
	"github.com/lb/internal/rw"
	"github.com/lb/internal/selector"
	"github.com/lb/pkg/circuitbreaker"
)

type Route struct {
	PathPrefix  string
	Method      string
	backends    []*backend.Backend
	StripPrefix string

	selector selector.Selector
	handler  http.Handler
}

func NewRoute(routeConfig config.RouteConfig, maxFailCount int64) *Route {
	sl := selector.NewSelector(selector.SelectorType(routeConfig.Strategy))

	r := &Route{
		PathPrefix:  routeConfig.PathPrefix,
		Method:      routeConfig.Method,
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
	const attempts = 3

	for i := 1; i <= attempts; i++ {
		backend := rt.choose(w, r)

		if backend == nil {
			http.Error(
				w,
				"No available backends",
				http.StatusServiceUnavailable,
			)
			return
		}

		attemptRec := httptest.NewRecorder()
		wrapped := rw.NewResponseWrapper(attemptRec)

		if rt.StripPrefix != "" {
			r2 := r.Clone(r.Context())

			r2.URL.Path = strings.TrimPrefix(
				r2.URL.Path,
				rt.StripPrefix,
			)

			if r2.URL.Path == "" {
				r2.URL.Path = "/"
			}

			r = r2
		}

		backend.IncrementActive()
		backend.ServeHTTP(wrapped, r)
		backend.DecrementActive()

		failed := wrapped.Status >= 500

		if !failed || !isIdempotent(r) || i == attempts {
			for k, vv := range attemptRec.Header() {
				w.Header()[k] = vv
			}
			w.WriteHeader(wrapped.Status)
			w.Write(attemptRec.Body.Bytes())
			return
		}

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
