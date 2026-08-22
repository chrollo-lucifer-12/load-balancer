package cb

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/lb/internal/metrics"
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

	metrics *metrics.RollingWindow

	halfOpenSuccessCount int64

	errorThreshold      float64
	halfOpenMaxRequests int64

	timeout time.Duration
}

func NewCircuitBreaker() *CircuitBreaker {
	return &CircuitBreaker{
		state:               StateClosed,
		metrics:             metrics.NewRollingWindow(10),
		halfOpenMaxRequests: 10,
		errorThreshold:      0.5,
		timeout:             10 * time.Second,
	}
}

func (cb *CircuitBreaker) Call(fn func() (any, error)) (any, error) {

	cb.mu.Lock()

	switch cb.state {

	case StateClosed:
		cb.mu.Unlock()
		return cb.handleClosedState(fn)

	case StateOpen:
		cb.mu.Unlock()
		return cb.handleOpenState()

	case StateHalfOpen:
		cb.mu.Unlock()
		return cb.handleHalfOpenState(fn)

	default:
		cb.mu.Unlock()
		return nil, errors.New("unknown circuit state")

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

func (cb *CircuitBreaker) handleClosedState(fn func() (any, error)) (any, error) {
	result, err := cb.runWithTimeout(fn)

	if err != nil {
		cb.metrics.Record(true)

		errRate, _ := cb.metrics.ErrorRate()
		if errRate >= cb.errorThreshold {
			cb.state = StateOpen
		}

		return nil, err
	}

	cb.resetCircuit()
	return result, nil
}

func (cb *CircuitBreaker) handleOpenState() (any, error) {
	if time.Since(cb.lastFailedAt) > cb.timeout {
		cb.state = StateHalfOpen
		cb.halfOpenSuccessCount = 0
		return nil, nil
	}

	return nil, errors.New("circuit open")
}

func (cb *CircuitBreaker) handleHalfOpenState(fn func() (any, error)) (any, error) {
	result, err := cb.runWithTimeout(fn)

	if err != nil {
		cb.state = StateOpen
		cb.lastFailedAt = time.Now()
		return nil, err
	}

	cb.halfOpenSuccessCount++

	if cb.halfOpenSuccessCount >= cb.halfOpenMaxRequests {
		cb.resetCircuit()
	}

	return result, nil
}

func (cb *CircuitBreaker) resetCircuit() {
	cb.state = StateClosed
	cb.halfOpenSuccessCount = 0
}

func (cb *CircuitBreaker) runWithTimeout(fn func() (any, error)) (any, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resultChan := make(chan struct {
		result any
		err    error
	}, 1)

	go func() {
		result, err := fn()
		resultChan <- struct {
			result any
			err    error
		}{result, err}
	}()

	select {
	case <-ctx.Done():
		return nil, errors.New("request timed out")
	case res := <-resultChan:
		return res.result, res.err
	}
}
