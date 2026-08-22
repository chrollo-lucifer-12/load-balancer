package cb

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

type unreliableService struct {
	healthy atomic.Bool
}

func newUnreliableService() (*httptest.Server, *unreliableService) {
	service := &unreliableService{}
	service.healthy.Store(true)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !service.healthy.Load() {
			http.Error(w, "service unavailable", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))

	return server, service
}

func (s *unreliableService) SetHealthy(healthy bool) {
	s.healthy.Store(healthy)
}

func BenchmarkCircuitBreakerCallClosed(b *testing.B) {
	cb := NewCircuitBreaker()

	fn := func() error {
		return nil
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		err := cb.Call(fn)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkCircuitBreakerCallFailure(b *testing.B) {
	cb := NewCircuitBreaker()

	cb.errorThreshold = 2.0

	fn := func() error {
		return errors.New("failure")
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = cb.Call(fn)
	}
}

func BenchmarkCircuitBreakerCanPassClosed(b *testing.B) {
	cb := NewCircuitBreaker()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		if !cb.CanPass() {
			b.Fatal("circuit unexpectedly rejected request")
		}
	}
}

func BenchmarkCircuitBreakerConcurrent(b *testing.B) {
	cb := NewCircuitBreaker()

	fn := func() error {
		return nil
	}

	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			err := cb.Call(fn)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

func BenchmarkCircuitBreakerCanPassConcurrent(b *testing.B) {
	cb := NewCircuitBreaker()

	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			cb.CanPass()
		}
	})
}

func TestCircuitBreakerStartsClosed(t *testing.T) {
	cb := NewCircuitBreaker()

	if !cb.CanPass() {
		t.Fatal("new circuit breaker should allow requests")
	}

	if cb.state != StateClosed {
		t.Fatalf("expected CLOSED, got %v", cb.state)
	}
}

func TestCircuitBreakerOpensAfterFailures(t *testing.T) {
	cb := NewCircuitBreaker()

	// Make the threshold easy to trigger.
	cb.errorThreshold = 0.5

	// First request: failure.
	err := cb.handleClosedState(func() error {
		return errors.New("service failed")
	})

	if err == nil {
		t.Fatal("expected request failure")
	}

	// Second request: failure.
	err = cb.handleClosedState(func() error {
		return errors.New("service failed")
	})

	if err == nil {
		t.Fatal("expected request failure")
	}

	if cb.state != StateOpen {
		t.Fatalf("expected OPEN, got %v", cb.state)
	}

	if cb.CanPass() {
		t.Fatal("OPEN circuit should reject requests")
	}
}

func TestCircuitBreakerOpenRejectsRequests(t *testing.T) {
	cb := NewCircuitBreaker()

	cb.state = StateOpen
	cb.lastFailedAt = time.Now()

	if cb.CanPass() {
		t.Fatal("OPEN circuit should reject requests during timeout")
	}
}

func TestCircuitBreakerTransitionsToHalfOpen(t *testing.T) {
	cb := NewCircuitBreaker()

	cb.state = StateOpen
	cb.lastFailedAt = time.Now().Add(-cb.timeout - time.Second)

	if !cb.CanPass() {
		t.Fatal("circuit should allow a probe after timeout")
	}

	if cb.state != StateHalfOpen {
		t.Fatalf(
			"expected HALF_OPEN, got %v",
			cb.state,
		)
	}
}

func TestCircuitBreakerHalfOpenFailureReopens(t *testing.T) {
	cb := NewCircuitBreaker()

	cb.state = StateHalfOpen

	err := cb.handleHalfOpenState(func() error {
		return errors.New("service still broken")
	})

	if err == nil {
		t.Fatal("expected failure")
	}

	if cb.state != StateOpen {
		t.Fatalf(
			"expected OPEN after half-open failure, got %v",
			cb.state,
		)
	}
}

func TestCircuitBreakerHalfOpenSuccessCloses(t *testing.T) {
	cb := NewCircuitBreaker()

	cb.state = StateHalfOpen
	cb.halfOpenMaxRequests = 3

	for i := 0; i < 3; i++ {
		err := cb.handleHalfOpenState(func() error {
			return nil
		})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

	}

	if cb.state != StateClosed {
		t.Fatalf(
			"expected CLOSED after successful probes, got %v",
			cb.state,
		)
	}
}

func TestCircuitBreakerRecoveryWithHTTPServer(t *testing.T) {
	server, service := newUnreliableService()
	defer server.Close()

	client := &http.Client{
		Timeout: 2 * time.Second,
	}

	cb := NewCircuitBreaker()

	// Make the CB easier/faster to test.
	cb.errorThreshold = 0.5
	cb.timeout = 100 * time.Millisecond
	cb.halfOpenMaxRequests = 2

	call := func() error {
		resp, err := client.Get(server.URL)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		if resp.StatusCode >= 500 {
			return errors.New("backend returned 500")
		}

		return nil
	}

	// Service becomes unhealthy.
	service.SetHealthy(false)

	// Two failures => 100% error rate => OPEN.
	for i := 0; i < 2; i++ {
		err := cb.Call(call)

		if err == nil {
			t.Fatal("expected request to fail")
		}
	}

	if cb.state != StateOpen {
		t.Fatalf("expected OPEN, got %v", cb.state)
	}

	// Circuit is still sleeping.
	if cb.CanPass() {
		t.Fatal("circuit should still be OPEN")
	}

	// Wait for OPEN -> HALF_OPEN.
	time.Sleep(150 * time.Millisecond)

	if !cb.CanPass() {
		t.Fatal("circuit should allow half-open probe")
	}

	if cb.state != StateHalfOpen {
		t.Fatalf("expected HALF_OPEN, got %v", cb.state)
	}

	// Backend recovered.
	service.SetHealthy(true)

	// Successful half-open requests.
	for i := 0; i < 2; i++ {
		err := cb.handleHalfOpenState(call)

		if err != nil {
			t.Fatalf("unexpected half-open error: %v", err)
		}
	}

	if cb.state != StateClosed {
		t.Fatalf(
			"expected CLOSED after recovery, got %v",
			cb.state,
		)
	}

	err := cb.Call(call)

	if err != nil {
		t.Fatalf("unexpected error after recovery: %v", err)
	}

}
