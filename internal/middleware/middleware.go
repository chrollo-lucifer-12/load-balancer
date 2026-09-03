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
			if err := recover(); err != nil {
				log.Printf("panic: %v\n%s", err, debug.Stack())

				http.Error(w, "internal server error",
					http.StatusInternalServerError)
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

// func Metric(next http.Handler) http.Handler {
// 	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

// 		metrics.MetricsRecord.RequestsTotal.Add(1)
// 		metrics.MetricsRecord.RequestsInFlight.Add(1)

// 		defer metrics.MetricsRecord.RequestsInFlight.Add(-1)

// 		next.ServeHTTP(w, r)

// 		if rw, ok := w.(*rw.ResponseWrapper); ok {
// 			if rw.Status >= 500 {
// 				metrics.MetricsRecord.RequestsFailed.Add(1)
// 			}
// 		}
// 	})
// }

func Chain(middlewares ...Middleware) Middleware {
	return func(h http.Handler) http.Handler {
		for i := len(middlewares) - 1; i >= 0; i-- {
			h = middlewares[i](h)
		}
		return h
	}
}
