package loadbalancer

import (
	"net/http"

	"github.com/lb/internal/config"
	"github.com/lb/internal/vhost"
)

type LoadBalancer struct {
	vhosts map[string]*vhost.VHost
}

func NewLoadBalancer(virtualHosts []config.VirtualHost) *LoadBalancer {

	lb := &LoadBalancer{
		vhosts: make(map[string]*vhost.VHost),
	}

	for _, vh := range virtualHosts {
		lb.vhosts[vh.Host] = vhost.NewVHost(vh)
	}

	return lb
}

func (lb *LoadBalancer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	host := r.Host

	vhost := lb.vhosts[host]

	if vhost == nil {
		return
	}

	vhost.ServeHTTP(w, r)
}
