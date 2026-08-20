package backend

import (
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync/atomic"
	"time"
)

var transport = &http.Transport{
	MaxIdleConns:        100,
	MaxIdleConnsPerHost: 20,
	MaxConnsPerHost:     100,
	IdleConnTimeout:     90 * time.Second,
}

type Backend struct {
	URL *url.URL

	Alive atomic.Bool

	ReverseProxy *httputil.ReverseProxy

	failCount atomic.Int64
	active    atomic.Int64
}

func NewBackend(rawURL string) (*Backend, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, err
	}

	backend := &Backend{
		URL: u,
	}

	backend.Alive.Store(true)

	proxy := &httputil.ReverseProxy{
		Transport: transport,

		Rewrite: func(pr *httputil.ProxyRequest) {
			originalHost := pr.In.Host

			pr.SetURL(u)

			pr.Out.Host = originalHost

			pr.SetXForwarded()
		},

		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			log.Printf("backend %s failed: %v", u, err)

			backend.IncreaseFailCount()

			http.Error(w, "Backend unavailable", http.StatusBadGateway)
		},
	}

	backend.ReverseProxy = proxy

	return backend, nil
}

func (b *Backend) SetAlive(alive bool) {
	b.Alive.Store(alive)
}

func (b *Backend) IsAlive() bool {
	return b.Alive.Load()
}

func (b *Backend) IncreaseFailCount() int64 {
	return b.failCount.Add(1)
}

func (b *Backend) ResetFailCount() {
	b.failCount.Store(0)
}

func (b *Backend) FailCount() int64 {
	return b.failCount.Load()
}

func (b *Backend) IncreaseActive() int64 {
	return b.active.Add(1)
}

func (b *Backend) DecreaseActive() int64 {
	return b.active.Add(-1)
}

func (b *Backend) Active() int64 {
	return b.active.Load()
}
