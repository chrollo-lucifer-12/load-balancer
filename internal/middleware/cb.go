package middleware

import (
	"log"
	"net/http"

	"github.com/lb/internal/rw"
	"github.com/lb/pkg/circuitbreaker"
)

type CircuitBreakerMiddleware struct {
	cb   *circuitbreaker.CircuitBreaker
	next http.Handler
}

func NewCircuitBreakerMiddleware(cb *circuitbreaker.CircuitBreaker, next http.Handler) http.Handler {
	return &CircuitBreakerMiddleware{
		cb:   cb,
		next: next,
	}
}

func (m *CircuitBreakerMiddleware) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	log.Println("cb middleware")

	if !m.cb.CanPass() {
		http.Error(
			w,
			"Service unavailable",
			http.StatusServiceUnavailable,
		)
		return
	}

	log.Println("circuit breaker PASS")

	rec := w.(*rw.ResponseWrapper)

	m.next.ServeHTTP(rec, r)

	if rec.Status >= 500 {
		log.Println("failed")
		m.cb.OnFailure()
	} else {
		log.Println("success")
		m.cb.OnSuccess()
	}
}
