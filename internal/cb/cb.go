package cb

import (
	"sync"
	"time"
)

type CBState int

const (
	StateClosed CBState = iota
	StateOpen
	StateHalfOpen
)

type CircuitBreaker struct {
	mu sync.Mutex

	cbState CBState

	metrics *RollingWindow

	errorThreshold float64
	minRequests    int64

	cbSleepWindow  time.Duration
	stateChangedAt time.Time

	halfOpenSuccess int64
}

func NewCircuitBreaker() *CircuitBreaker {
	return &CircuitBreaker{
		cbState:        StateClosed,
		metrics:        NewRollingWindow(10),
		minRequests:    10,
		errorThreshold: 0.5,
		cbSleepWindow:  5 * time.Second,
	}
}

func (cb *CircuitBreaker) CanPass() bool {
	switch cb.cbState {
	case StateClosed:
		return true
	case StateHalfOpen:
		return true
	case StateOpen:
		if time.Since(cb.stateChangedAt) >= cb.cbSleepWindow {
			cb.cbState = StateHalfOpen
			cb.stateChangedAt = time.Now()
			cb.halfOpenSuccess = 0
			return true
		}

		return false
	}
	return false
}

func (cb *CircuitBreaker) Record(isFailure bool) {
	cb.metrics.Record(isFailure)

	if cb.cbState == StateClosed {
		errRate, total := cb.metrics.ErrorRate()
		if total >= cb.minRequests && errRate >= cb.errorThreshold {
			cb.cbState = StateOpen
			cb.stateChangedAt = time.Now()
		}
	} else if cb.cbState == StateHalfOpen {
		if isFailure {
			cb.cbState = StateOpen
			cb.stateChangedAt = time.Now()
		} else {
			cb.halfOpenSuccess++

			if cb.halfOpenSuccess >= 5 {
				cb.cbState = StateClosed
				cb.stateChangedAt = time.Now()
			}
		}
	}
}
