package loadbalancer

import (
	"net/http"

	"github.com/lb/internal/config"
	"github.com/lb/internal/middleware"
	"github.com/lb/internal/rl"
	"github.com/lb/internal/rw"
	"github.com/lb/internal/vhost"
)

type LoadBalancer struct {
	vhosts map[string]*vhost.VHost
	mux    http.Handler
}

func NewLoadBalancer(virtualHosts []config.VirtualHost, limiter rl.RateLimiter) *LoadBalancer {

	lb := &LoadBalancer{
		vhosts: make(map[string]*vhost.VHost),
	}

	for _, vh := range virtualHosts {
		lb.vhosts[vh.Host] = vhost.NewVHost(vh)
	}

	mux := http.NewServeMux()

	handler := middleware.Chain(middleware.Recover,
		middleware.Buffer,
		middleware.Metric,
		middleware.Logger,
		middleware.RateLimit(limiter))(http.HandlerFunc(lb.serveHTTP))

	mux.Handle("/", handler)

	lb.mux = mux

	return lb
}

func (lb *LoadBalancer) Run(w http.ResponseWriter, r *http.Request) {
	lb.mux.ServeHTTP(w, r)
}

func (lb *LoadBalancer) serveHTTP(w http.ResponseWriter, r *http.Request) {

	host := r.Host

	vhost := lb.vhosts[host]

	attempts := 3

	for i := 1; i <= attempts; i++ {
		backend := vhost.Choose(r.RemoteAddr)
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
