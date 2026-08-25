package vhost

import (
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

func (vh *VHost) checkBackends() {
	var wg sync.WaitGroup

	for _, b := range vh.backends {
		if b == nil || b.URL == nil {
			continue
		}

		wg.Add(1)

		go func(b *backend.Backend) {
			defer wg.Done()

			alive := isBackendAlive(b.URL)

			// log.Printf(
			// 	"health update for %s: alive=%t",
			// 	b.URL,
			// 	alive,
			// )

			b.UpdateActiveStatus(alive)
		}(b)
	}

	wg.Wait()
	// log.Println("health check completed")
}

func (vh *VHost) healthCheck() {
	vh.checkBackends()

	ticker := time.NewTicker(vh.healthCheckInterval)
	defer ticker.Stop()

	for range ticker.C {
		vh.checkBackends()
	}
}
