package backend

import (
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
