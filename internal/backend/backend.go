package backend

import (
	"fmt"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync/atomic"
	"time"

	"github.com/lb/internal/window"
)

type BackendState int32

const (
	Healthy BackendState = iota
	Unhealthy
	Draining
)

var Transport = &http.Transport{

	DialContext: (&net.Dialer{
		Timeout:   5 * time.Second,
		KeepAlive: 30 * time.Second,
	}).DialContext,

	TLSHandshakeTimeout: 5 * time.Second,

	ResponseHeaderTimeout: 10 * time.Second,

	MaxIdleConns:        100,
	MaxIdleConnsPerHost: 20,
	MaxConnsPerHost:     100,
	IdleConnTimeout:     90 * time.Second,
}

type Backend struct {
	URL *url.URL

	Alive atomic.Bool

	ReverseProxy *httputil.ReverseProxy

	passive *window.RollingWindow

	active atomic.Int64
	weight int
}

func NewBackend(rawURL string, weight int, maxFailCount int64) (*Backend, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("new backend :%s %w", rawURL, err)
	}

	backend := &Backend{
		URL:     u,
		weight:  weight,
		passive: window.NewRollingWindow(30),
	}

	backend.Alive.Store(true)

	proxy := &httputil.ReverseProxy{

		Transport: Transport,

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
			failure := r.StatusCode >= 500

			if failure {
				backend.RecordFailure(maxFailCount)
			} else {
				backend.RecordSuccess()
			}

			return nil
		},

		BufferPool: newBufferPool(),
	}

	backend.ReverseProxy = proxy

	return backend, nil
}

func (b *Backend) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	b.ReverseProxy.ServeHTTP(w, r)
}

func (b *Backend) RecordSuccess() {
	b.passive.Record(false)
}

func (b *Backend) RecordFailure(maxFailCount int64) {
	b.passive.Record(true)

	if b.passive.FailureCount() >= maxFailCount {
		b.Alive.Store(false)
	}
}

func (b *Backend) UpdateActiveStatus(isUp bool) {
	b.Alive.Store(isUp)
}

func (b *Backend) IsAlive() bool {
	return b.Alive.Load()
}

func (b *Backend) CanPass() bool {
	return b.Alive.Load()
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
