package proxy

import (
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"
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

func NewProxy(u *url.URL) *httputil.ReverseProxy {
	return &httputil.ReverseProxy{
		Transport: Transport,

		Rewrite: func(pr *httputil.ProxyRequest) {
			originalHost := pr.In.Host

			pr.SetURL(u)

			pr.Out.Host = originalHost

			pr.SetXForwarded()
		},

		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			w.WriteHeader(http.StatusBadGateway)
		},

		BufferPool: newBufferPool(),
	}
}
