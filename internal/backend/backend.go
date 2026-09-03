package backend

import (
	"fmt"
	"net/http"
	"net/url"
	"sync/atomic"

	"github.com/lb/pkg/proxy"
)

type BackendState int32

const (
	Healthy BackendState = iota
	Unhealthy
	Draining
)

type Backend struct {
	URL     *url.URL
	Handler http.Handler

	alive atomic.Bool

	counter *Counter

	weight int
}

func NewBackend(rawURL string, weight int, maxFailCount int64) (*Backend, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("new backend :%s %w", rawURL, err)
	}

	backend := &Backend{
		URL:    u,
		weight: weight,
	}

	backend.alive.Store(true)
	backend.counter = NewCounter()

	proxy := proxy.NewProxy(u)

	backend.Handler = NewPassiveHealthHandlder(backend, proxy, maxFailCount)

	return backend, nil
}

func (b *Backend) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	b.Handler.ServeHTTP(w, r)
}

func (b *Backend) RecordSuccess() {
	b.counter.passive.Record(false)
}

func (b *Backend) RecordFailure(maxFailCount int64) {
	b.counter.passive.Record(true)

	if b.counter.passive.FailureCount() >= maxFailCount {
		b.alive.Store(false)
	}
}

func (b *Backend) UpdateActiveStatus(isUp bool) {
	b.alive.Store(isUp)
}

func (b *Backend) IsAlive() bool {
	return b.alive.Load()
}

func (b *Backend) CanPass() bool {
	return b.alive.Load()
}

func (b *Backend) IncrementActive() int64 {
	return b.counter.active.Add(1)
}

func (b *Backend) DecrementActive() int64 {
	return b.counter.active.Add(-1)
}

func (b *Backend) ActiveCount() int64 {
	return b.counter.active.Load()
}
