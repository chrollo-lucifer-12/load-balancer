package middleware

import (
	"log"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/lb/internal/metrics"
	"github.com/lb/internal/rl"
	"github.com/lb/internal/rw"
)

type Middleware func(http.Handler) http.Handler

func Buffer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rec := rw.NewResponseWrapper(w)

		next.ServeHTTP(rec, r)

		rec.Flush()
	})
}

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

func Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		start := time.Now()

		next.ServeHTTP(w, r)

		if rw, ok := w.(*rw.ResponseWrapper); ok {
			log.Printf(
				"%s %s status=%d duration=%s",
				r.Method,
				r.URL.Path,
				rw.Status,
				time.Since(start),
			)
		}
	})
}

func RateLimit(limiter rl.RateLimiter) Middleware {
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

func Metric(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		metrics.MetricsRecord.RequestsTotal.Add(1)
		metrics.MetricsRecord.RequestsInFlight.Add(1)

		defer metrics.MetricsRecord.RequestsInFlight.Add(-1)

		next.ServeHTTP(w, r)

		if rw, ok := w.(*rw.ResponseWrapper); ok {
			if rw.Status >= 500 {
				metrics.MetricsRecord.RequestsFailed.Add(1)
			}
		}
	})
}

func Chain(middlewares ...Middleware) Middleware {
	return func(h http.Handler) http.Handler {
		for i := len(middlewares) - 1; i >= 0; i-- {
			h = middlewares[i](h)
		}
		return h
	}
}
