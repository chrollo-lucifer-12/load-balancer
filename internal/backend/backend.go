package backend

import (
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync/atomic"
	"time"
)

type BackendState int

const (
	Healthy BackendState = iota
	Unhealthy
	Draining
)

var transport = &http.Transport{

	DialContext: (&net.Dialer{
		Timeout:   5 * time.Second,
		KeepAlive: 30 * time.Second,
	}).DialContext,

	TLSHandshakeTimeout: 5 * time.Second,

	ResponseHeaderTimeout: 10 * time.Second,

	DisableKeepAlives: true,

	MaxIdleConns:        100,
	MaxIdleConnsPerHost: 20,
	MaxConnsPerHost:     100,
	IdleConnTimeout:     90 * time.Second,
}

type Backend struct {
	URL *url.URL

	State atomic.Int32
	Alive atomic.Bool

	ReverseProxy *httputil.ReverseProxy

	failCount    atomic.Int64
	successCount atomic.Int32

	active atomic.Int64

	weight int
}

func NewBackend(rawURL string, weight int, maxFailCount int) (*Backend, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, err
	}

	backend := &Backend{
		URL:    u,
		weight: weight,
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

		ModifyResponse: func(r *http.Response) error {
			switch {
			case r.StatusCode >= 500:
				backend.MarkFailure(maxFailCount)
			default:
				backend.MarkSuccess()
			}
			return nil
		},
	}

	backend.ReverseProxy = proxy

	return backend, nil
}

func (b *Backend) MarkFailure(maxFail int) {
	fails := b.failCount.Add(1)

	if fails >= int64(maxFail) {
		b.Alive.Store(false)
	}
}

func (b *Backend) MarkSuccess() {
	b.failCount.Store(0)
	b.Alive.Store(true)
}

func (b *Backend) IsAlive() bool {
	return b.Alive.Load()
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
