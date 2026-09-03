package backend

import (
	"net/http"
	"net/http/httputil"
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

	proxy *httputil.ReverseProxy

	alive atomic.Bool

	counter *Counter

	weight int

	maxFailCount int64
}

func NewBackend(
	rawURL string,
	weight int,
	maxFailCount int64,
) (*Backend, error) {

	target, err := url.Parse(rawURL)
	if err != nil {
		return nil, err
	}

	b := &Backend{
		URL:          target,
		weight:       weight,
		maxFailCount: maxFailCount,
		counter:      NewCounter(),
	}

	b.alive.Store(true)

	b.proxy = proxy.NewProxy(
		target,
		func(status int) {
			b.RecordResponse(status)
		},
	)

	return b, nil
}

func (b *Backend) ServeHTTP(
	w http.ResponseWriter,
	r *http.Request,
) {
	b.IncrementActive()
	defer b.DecrementActive()

	b.proxy.ServeHTTP(w, r)
}

func (b *Backend) ServeHTTPWithCallback(
	w http.ResponseWriter,
	r *http.Request,
	onResponse func(status int),
) {
	b.IncrementActive()
	defer b.DecrementActive()

	p := *b.proxy

	p.ModifyResponse = func(resp *http.Response) error {
		status := resp.StatusCode

		b.RecordResponse(status)

		if onResponse != nil {
			onResponse(status)
		}

		return nil
	}

	p.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		b.RecordFailure(b.maxFailCount)

		if onResponse != nil {
			onResponse(http.StatusBadGateway)
		}

		http.Error(
			w,
			"Bad Gateway",
			http.StatusBadGateway,
		)
	}

	p.ServeHTTP(w, r)
}

func (b *Backend) RecordResponse(status int) {
	if status >= 500 {
		b.RecordFailure(b.maxFailCount)
		return
	}

	b.RecordSuccess()
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
