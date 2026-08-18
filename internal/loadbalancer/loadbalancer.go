package loadbalancer

import (
	"context"
	"log"
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

		b := &backend.Backend{
			URL:          url,
			Alive:        true,
			ReverseProxy: httputil.NewSingleHostReverseProxy(url),
		}

		b.ReverseProxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {

			failCount := b.IncreaseFailCount()

			if failCount >= maxFailCount {
				log.Printf("Backend %s is marked as down due to too many failures", b.URL.Host)
				b.SetAlive(false)
			}

			lb := r.Context().Value("loadbalancer").(*LoadBalancer)

			if newBackend := lb.NextBackend(); newBackend != nil {
				log.Printf("Retrying request on backend %s", newBackend.URL.Host)
				newBackend.ReverseProxy.ServeHTTP(w, r)
				return
			}

			http.Error(w, "Service Unavailable", http.StatusServiceUnavailable)
		}

		backends[i] = b
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
	client := http.Client{
		Timeout: time.Second,
	}

	healthURL := *u
	healthURL.Path = "/health"

	resp, err := client.Get(healthURL.String())
	if err != nil {
		return false
	}

	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK
}

func (lb *LoadBalancer) healthCheck() {
	ticker := time.NewTicker(lb.healthCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			log.Println("starting health check")

			var wg sync.WaitGroup

			for _, b := range lb.backends {
				wg.Add(1)

				go func(b *backend.Backend) {
					defer wg.Done()

					alive := isBackendAlive(b.URL)
					b.SetAlive(alive)
					status := "up"
					if !alive {
						status = "down"
					}

					log.Printf("backend %s status: %s", b.URL.Host, status)
				}(b)
			}

			wg.Wait()
			log.Println("health check completed")
		default:
			continue
		}
	}
}

func (lb *LoadBalancer) ServerHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := context.WithValue(
		r.Context(),
		"loadbalancer",
		lb,
	)

	r = r.WithContext(ctx)

	b := lb.NextBackend()

	if b == nil {
		http.Error(w, "no available backends", http.StatusServiceUnavailable)
		return
	}

	log.Printf("forwarding request to: %s", b.URL)

	b.ReverseProxy.ServeHTTP(w, r)

	b.ResetFailCount()
}
