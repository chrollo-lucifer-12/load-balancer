package cb

import (
	"sync"
	"time"

	"github.com/lb/internal/window"
)

type CBState int

const (
	StateClosed CBState = iota
	StateOpen
	StateHalfOpen
)

type CircuitBreaker struct {
	mu sync.Mutex

	state        CBState
	lastFailedAt time.Time

	metrics *window.RollingWindow

	halfOpenSuccessCount int64

	errorThreshold      float64
	halfOpenMaxRequests int64

	timeout time.Duration
}

func NewCircuitBreaker() *CircuitBreaker {
	return &CircuitBreaker{
		state:               StateClosed,
		metrics:             window.NewRollingWindow(10),
		halfOpenMaxRequests: 10,
		errorThreshold:      0.5,
		timeout:             10 * time.Second,
	}
}

func (cb *CircuitBreaker) CanPass() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case StateClosed:
		return true

	case StateOpen:
		if time.Since(cb.lastFailedAt) >= cb.timeout {
			cb.state = StateHalfOpen
			cb.halfOpenSuccessCount = 0
			return true
		}
		return false

	case StateHalfOpen:
		return true
	}

	return false
}

func (cb *CircuitBreaker) OnSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if cb.state != StateHalfOpen {
		return
	}

	cb.handleHalfOpenState()
}

func (cb *CircuitBreaker) OnFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.metrics.Record(true)

	if cb.state == StateHalfOpen {
		cb.state = StateOpen
		cb.lastFailedAt = time.Now()
		cb.halfOpenSuccessCount = 0
		return
	}

	if cb.state == StateClosed {
		cb.handleClosedState()
	}
}

func (cb *CircuitBreaker) handleClosedState() {

	errRate, _ := cb.metrics.ErrorRate()
	if errRate >= cb.errorThreshold {
		cb.state = StateOpen
		cb.lastFailedAt = time.Now()
	}

	cb.resetCircuit()

}

func (cb *CircuitBreaker) handleHalfOpenState() error {

	cb.halfOpenSuccessCount++

	if cb.halfOpenSuccessCount >= cb.halfOpenMaxRequests {
		cb.resetCircuit()
	}

	return nil
}

func (cb *CircuitBreaker) resetCircuit() {
	cb.state = StateClosed
	cb.halfOpenSuccessCount = 0
}
