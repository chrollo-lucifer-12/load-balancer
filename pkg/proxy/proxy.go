package proxy

import (
	"context"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"runtime"
	"sync/atomic"
	"time"
)

var (
	transportPool []*http.Transport

	poolSize int

	roundRobinCounter uint64

	globalBufferPool = newBufferPool()
)

func init() {

	poolSize = runtime.NumCPU()
	transportPool = make([]*http.Transport, poolSize)

	for i := 0; i < poolSize; i++ {
		transportPool[i] = &http.Transport{

			DialContext: (&net.Dialer{
				Timeout:   5 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext,

			TLSHandshakeTimeout:   5 * time.Second,
			ResponseHeaderTimeout: 10 * time.Second,

			ReadBufferSize:  64 * 1024,
			WriteBufferSize: 64 * 1024,

			MaxIdleConns:        0,
			MaxIdleConnsPerHost: 2048,
			MaxConnsPerHost:     0,
			IdleConnTimeout:     90 * time.Second,

			ForceAttemptHTTP2: false,
		}
	}
}

func nextTransport() *http.Transport {
	idx := atomic.AddUint64(&roundRobinCounter, 1) % uint64(poolSize)
	return transportPool[idx]
}

func NewProxy(u *url.URL) *httputil.ReverseProxy {

	selectedTransport := nextTransport()

	return &httputil.ReverseProxy{

		Transport: selectedTransport,

		Rewrite: func(pr *httputil.ProxyRequest) {
			originalHost := pr.In.Host
			pr.SetURL(u)
			pr.Out.Host = originalHost
			pr.SetXForwarded()
		},

		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			if err == context.Canceled {
				w.WriteHeader(499)
				return
			}
			w.WriteHeader(http.StatusBadGateway)
		},

		BufferPool: globalBufferPool,
	}
}
