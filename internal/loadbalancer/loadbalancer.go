package loadbalancer

import (
	"log"
	"net/http"

	"github.com/lb/internal/config"
	"github.com/lb/internal/middleware"
	"github.com/lb/internal/vhost"
)

type LoadBalancer struct {
	vhosts  map[string]*vhost.VHost
	handler http.Handler
}

func NewLoadBalancer(virtualHosts []config.VirtualHost, limiterType string) *LoadBalancer {

	//	limiter := rl.NewRateLimiter(rl.RateLimiterType(limiterType))

	lb := &LoadBalancer{
		vhosts: make(map[string]*vhost.VHost),
	}

	for _, vh := range virtualHosts {
		lb.vhosts[vh.Host] = vhost.NewVHost(vh)
	}

	lb.handler = middleware.Chain(
		middleware.Recover,
		middleware.Buffer,
		middleware.Metric,
	//	middleware.Logger,
	)(http.HandlerFunc(lb.serveHTTP))

	return lb
}

func (lb *LoadBalancer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	lb.handler.ServeHTTP(w, r)
}

func (lb *LoadBalancer) serveHTTP(w http.ResponseWriter, r *http.Request) {

	host := r.Host

	vhost := lb.vhosts[host]

	if vhost == nil {
		log.Printf("no vhost found for :%s", host)
		return
	}

	vhost.ServeHTTP(w, r)
}
