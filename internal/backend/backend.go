package backend

import (
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync"
)

type Backend struct {
	URL          *url.URL
	Alive        bool
	ReverseProxy *httputil.ReverseProxy
	mu           sync.RWMutex
	failCount    int
}

func NewBackend(rawURL string) (*Backend, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, err
	}

	backend := &Backend{
		URL:       u,
		Alive:     true,
		failCount: 0,
	}

	proxy := httputil.NewSingleHostReverseProxy(u)

	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		log.Printf("backend %s failed: %v", u, err)

		backend.mu.Lock()
		backend.failCount++
		backend.mu.Unlock()

		http.Error(w, "Backend unavailable", http.StatusBadGateway)
	}

	backend.ReverseProxy = proxy

	return backend, nil
}

func (b *Backend) SetAlive(alive bool) {
	b.mu.Lock()
	b.Alive = alive
	if alive {
		b.failCount = 0
	}
	b.mu.Unlock()
}

func (b *Backend) IsAlive() bool {
	b.mu.RLock()
	alive := b.Alive
	b.mu.RUnlock()
	return alive
}

func (b *Backend) IncreaseFailCount() int {
	b.mu.Lock()
	b.failCount++
	count := b.failCount
	b.mu.Unlock()
	return count
}

func (b *Backend) ResetFailCount() {
	b.mu.Lock()
	b.failCount = 0
	b.mu.Unlock()
}
