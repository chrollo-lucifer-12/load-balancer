package loadbalancer

import (
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync"
	"time"

	"github.com/lb/internal/backend"
)

type LoadBalancer struct {
	backends            []*backend.Backend
	current             int
	mu                  sync.Mutex
	healthCheckInterval time.Duration
	maxFailCount        int
}

func NewLoadBalancer(backendURLs []string, healthCheckInterval time.Duration, maxFailCount int) *LoadBalancer {
	backends := make([]*backend.Backend, len(backendURLs))

	for i, rawURL := range backendURLs {
		url, err := url.Parse(rawURL)
		if err != nil {
			return nil
		}

		backends[i] = &backend.Backend{
			URL:          url,
			Alive:        true,
			ReverseProxy: httputil.NewSingleHostReverseProxy(url),
		}

		backends[i].ReverseProxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
			backend := backends[i]
			failCount := backend.IncreaseFailCount()

			if failCount >= maxFailCount {
				log.Printf("Backend %s is marked as down due to too many failures", backend.URL.Host)
				backend.SetAlive(false)
			}

			lb := r.Context().Value("loadbalancer").(*LoadBalancer)
			if newBackend := lb.NextBackend(); newBackend != nil {
				log.Printf("Retrying request on backend %s", newBackend.URL.Host)
				newBackend.ReverseProxy.ServeHTTP(w, r)
				return
			}

			http.Error(w, "Service Unavailable", http.StatusServiceUnavailable)
		}
	}

	lb := &LoadBalancer{
		backends:            backends,
		healthCheckInterval: healthCheckInterval,
		maxFailCount:        maxFailCount,
	}

	go lb.healthCheck()

	return lb
}

func (lb *LoadBalancer) NextBackend() *backend.Backend {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	initialIndex := lb.current

	for i := 0; i < len(lb.backends); i++ {
		idx := (initialIndex + i) % len(lb.backends)
		if lb.backends[idx].IsAlive() {
			lb.current = idx
			return lb.backends[idx]
		}
	}

	return nil
}

func isBackendAlive(u *url.URL) bool {
	timeout := 1 * time.Second
	conn, err := net.DialTimeout("tcp", u.Host, timeout)
	if err != nil {
		log.Printf("Health check failed for %s: %v", u.Host, err)
		return false
	}
	defer conn.Close()
	return true
}

func (lb *LoadBalancer) healthCheck() {
	ticker := time.NewTicker(lb.healthCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			log.Println("starting health check")

			for _, backend := range lb.backends {
				alive := isBackendAlive(backend.URL)
				backend.SetAlive(alive)
				status := "up"
				if !alive {
					status = "down"
				}
				log.Printf("backend %s status: %s", backend.URL.Host, status)
			}
			log.Println("health check completed")
		default:
			continue
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

	backend.ResetFailCount()
}
