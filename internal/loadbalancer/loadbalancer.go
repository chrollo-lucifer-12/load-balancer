package loadbalancer

import (
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync"

	"github.com/lb/internal/backend"
)

type LoadBalancer struct {
	backends []*backend.Backend
	current  int
	mu       sync.Mutex
}

func NewLoadBalancer(backendURLs []string) *LoadBalancer {
	backends := make([]*backend.Backend, len(backendURLs))

	for i, rawURL := range backendURLs {
		url, err := url.Parse(rawURL)
		if err != nil {
			return nil
		}

		proxy := httputil.NewSingleHostReverseProxy(url)
		proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
			log.Printf("Error: %v", err)
			w.WriteHeader(http.StatusBadGateway)
		}

		backends[i] = &backend.Backend{
			URL:          url,
			Alive:        true,
			ReverseProxy: proxy,
		}
	}

	return &LoadBalancer{
		backends: backends,
	}
}

func (lb *LoadBalancer) NextBackend() *backend.Backend {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	initialIndex := lb.current

	for {
		lb.current = (lb.current + 1) % len(lb.backends)
		if lb.backends[lb.current].IsAlive() {
			return lb.backends[lb.current]
		}

		if lb.current == initialIndex {
			return nil
		}
	}
}

func (lb *LoadBalancer) ServerHTTP(w http.ResponseWriter, r *http.Request) {
	backend := lb.NextBackend()

	if backend == nil {
		http.Error(w, "no available backends", http.StatusServiceUnavailable)
		return
	}

	log.Printf("forwarding requests to: %s", backend.URL)
	backend.ReverseProxy.ServeHTTP(w, r)
}
