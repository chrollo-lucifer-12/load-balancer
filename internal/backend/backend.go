package backend

import (
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync/atomic"
	"time"
)

type BackendState int32

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

func NewBackend(rawURL string, weight int, maxFailCount int64) (*Backend, error) {
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

		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			backend.RecordFailure(maxFailCount)
			w.WriteHeader(http.StatusBadGateway)
		},

		ModifyResponse: func(r *http.Response) error {
			switch {
			case r.StatusCode >= 500:
				backend.RecordFailure(maxFailCount)
			default:
				backend.RecordSuccess()
			}
			return nil
		},
	}

	backend.ReverseProxy = proxy

	return backend, nil
}

func (b *Backend) RecordSuccess() {
	b.successCount.Add(1)

	if b.failCount.Load() > 0 {
		b.failCount.Add(-1)
	}

	if b.failCount.Load() == 0 && !b.Alive.Load() {
		b.Alive.Store(true)
		b.State.Store(int32(Healthy))
	}
}

func (b *Backend) RecordFailure(maxFailCount int64) {
	currentFailures := b.failCount.Load()

	b.successCount.Store(0)

	if currentFailures >= maxFailCount {
		b.Alive.Store(false)
		b.State.Store(int32(Unhealthy))
	}
}

func (b *Backend) UpdateActiveStatus(isUp bool) {
	b.Alive.Store(isUp)

	if isUp {
		b.State.Store(int32(Healthy))
		b.failCount.Store(0)
	} else {
		b.State.Store(int32(Unhealthy))
	}
}

func (b *Backend) IsAlive() bool {
	return b.Alive.Load()
}

func (b *Backend) IncrementFailCount() {
	b.failCount.And(1)
}

func (b *Backend) IncrementActive() int64 {
	return b.active.Add(1)
}

func (b *Backend) DecrementActive() int64 {
	return b.active.Add(-1)
}

func (b *Backend) ActiveCount() int64 {
	return b.active.Load()
}
