package proxy

import (
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"
)

var (
	globalBufferPool = newBufferPool()
)

var transport = &http.Transport{
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

type ResponseObserver func(status int)

func NewProxy(u *url.URL, onResponse ResponseObserver) *httputil.ReverseProxy {

	return &httputil.ReverseProxy{

		Transport: transport,

		Rewrite: func(pr *httputil.ProxyRequest) {
			originalHost := pr.In.Host
			pr.SetURL(u)
			pr.Out.Host = originalHost
			pr.SetXForwarded()
		},

		ModifyResponse: func(r *http.Response) error {
			if onResponse != nil {
				onResponse(r.StatusCode)
			}

			return nil
		},

		BufferPool: globalBufferPool,
	}
}
