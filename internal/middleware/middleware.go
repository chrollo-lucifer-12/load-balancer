package middleware

import (
	"log"
	"net/http"
	"runtime/debug"

	"github.com/lb/pkg/ratelimiter"
)

type Middleware func(http.Handler) http.Handler

func Recover(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if v := recover(); v != nil {
				if v == http.ErrAbortHandler {
					return
				}

				log.Printf("panic: %v\n%s", v, debug.Stack())
			}
		}()

		next.ServeHTTP(w, r)
	})
}

func RateLimit(limiter ratelimiter.RateLimiter) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			keyIP := r.RemoteAddr

			if limiter.Allow(keyIP) {
				next.ServeHTTP(w, r)
			} else {
				http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
			}
		})
	}
}

func Chain(middlewares ...Middleware) Middleware {
	return func(h http.Handler) http.Handler {
		for i := len(middlewares) - 1; i >= 0; i-- {
			h = middlewares[i](h)
		}
		return h
	}
}
