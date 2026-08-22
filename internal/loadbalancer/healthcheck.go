package loadbalancer

import (
	"log"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/lb/internal/backend"
)

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

					if !alive {
						b.MarkFailure(lb.maxFailCount)
					} else {
						b.MarkSuccess()
					}
				}(b)
			}

			wg.Wait()
			log.Println("health check completed")
		default:
			continue
		}
	}
}
