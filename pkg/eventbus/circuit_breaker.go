// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package eventbus

import (
	"sync"
	"sync/atomic"
	"time"
)

// CircuitState represents the state of a circuit breaker
type CircuitState int32

const (
	// StateClosed allows requests through
	StateClosed CircuitState = iota
	// StateOpen blocks all requests
	StateOpen
	// StateHalfOpen allows a limited number of requests through
	StateHalfOpen
)

// CircuitBreaker implements the circuit breaker pattern
type CircuitBreaker struct {
	// Configuration
	failureThreshold int32
	resetTimeout     time.Duration

	// State
	state        int32 // atomic
	failures     int32 // atomic
	successes    int32 // atomic
	lastFailTime int64 // atomic (unix nano)

	// Half-open state management
	halfOpenMu     sync.Mutex
	halfOpenLimit  int32
	halfOpenCount  int32
	halfOpenWindow time.Time
}

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(failureThreshold int, resetTimeout time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		failureThreshold: int32(failureThreshold),
		resetTimeout:     resetTimeout,
		state:            int32(StateClosed),
		halfOpenLimit:    3, // Allow 3 requests in half-open state
	}
}

// Allow checks if a request should be allowed through
func (cb *CircuitBreaker) Allow() bool {
	state := cb.getState()

	switch CircuitState(state) {
	case StateClosed:
		return true

	case StateOpen:
		// Check if we should transition to half-open
		lastFail := atomic.LoadInt64(&cb.lastFailTime)
		if time.Since(time.Unix(0, lastFail)) > cb.resetTimeout {
			cb.transitionToHalfOpen()
			return cb.allowHalfOpen()
		}
		return false

	case StateHalfOpen:
		return cb.allowHalfOpen()

	default:
		return false
	}
}

// RecordSuccess records a successful operation
func (cb *CircuitBreaker) RecordSuccess() {
	state := cb.getState()

	switch CircuitState(state) {
	case StateClosed:
		// Reset failure count on success
		atomic.StoreInt32(&cb.failures, 0)

	case StateHalfOpen:
		cb.halfOpenMu.Lock()
		defer cb.halfOpenMu.Unlock()

		atomic.AddInt32(&cb.successes, 1)

		// If we've had enough successes, close the circuit
		if atomic.LoadInt32(&cb.successes) >= cb.halfOpenLimit {
			cb.transitionToClosed()
		}
	}
}

// RecordFailure records a failed operation
func (cb *CircuitBreaker) RecordFailure() {
	state := cb.getState()

	// Update last failure time
	atomic.StoreInt64(&cb.lastFailTime, time.Now().UnixNano())

	switch CircuitState(state) {
	case StateClosed:
		failures := atomic.AddInt32(&cb.failures, 1)
		if failures >= cb.failureThreshold {
			cb.transitionToOpen()
		}

	case StateHalfOpen:
		// Any failure in half-open state opens the circuit
		cb.transitionToOpen()
	}
}

// GetState returns the current state of the circuit breaker
func (cb *CircuitBreaker) GetState() CircuitState {
	return CircuitState(cb.getState())
}

// GetStats returns circuit breaker statistics
func (cb *CircuitBreaker) GetStats() map[string]interface{} {
	stats := make(map[string]interface{})

	state := cb.getState()
	stats["state"] = cb.stateString(CircuitState(state))
	stats["failures"] = atomic.LoadInt32(&cb.failures)
	stats["successes"] = atomic.LoadInt32(&cb.successes)

	lastFail := atomic.LoadInt64(&cb.lastFailTime)
	if lastFail > 0 {
		stats["last_failure"] = time.Unix(0, lastFail)
		stats["time_since_failure"] = time.Since(time.Unix(0, lastFail))
	}

	if state == int32(StateHalfOpen) {
		cb.halfOpenMu.Lock()
		stats["half_open_count"] = cb.halfOpenCount
		stats["half_open_limit"] = cb.halfOpenLimit
		cb.halfOpenMu.Unlock()
	}

	return stats
}

// Reset resets the circuit breaker to closed state
func (cb *CircuitBreaker) Reset() {
	atomic.StoreInt32(&cb.state, int32(StateClosed))
	atomic.StoreInt32(&cb.failures, 0)
	atomic.StoreInt32(&cb.successes, 0)
	atomic.StoreInt64(&cb.lastFailTime, 0)

	cb.halfOpenMu.Lock()
	cb.halfOpenCount = 0
	cb.halfOpenMu.Unlock()
}

// getState returns the current state as int32
func (cb *CircuitBreaker) getState() int32 {
	return atomic.LoadInt32(&cb.state)
}

// transitionToOpen transitions to open state
func (cb *CircuitBreaker) transitionToOpen() {
	atomic.StoreInt32(&cb.state, int32(StateOpen))
	atomic.StoreInt32(&cb.failures, 0)
	atomic.StoreInt32(&cb.successes, 0)
}

// transitionToHalfOpen transitions to half-open state
func (cb *CircuitBreaker) transitionToHalfOpen() {
	cb.halfOpenMu.Lock()
	defer cb.halfOpenMu.Unlock()

	if atomic.CompareAndSwapInt32(&cb.state, int32(StateOpen), int32(StateHalfOpen)) {
		cb.halfOpenCount = 0
		cb.halfOpenWindow = time.Now()
		atomic.StoreInt32(&cb.successes, 0)
		atomic.StoreInt32(&cb.failures, 0)
	}
}

// transitionToClosed transitions to closed state
func (cb *CircuitBreaker) transitionToClosed() {
	atomic.StoreInt32(&cb.state, int32(StateClosed))
	atomic.StoreInt32(&cb.failures, 0)
	atomic.StoreInt32(&cb.successes, 0)
	cb.halfOpenCount = 0
}

// allowHalfOpen checks if a request should be allowed in half-open state
func (cb *CircuitBreaker) allowHalfOpen() bool {
	cb.halfOpenMu.Lock()
	defer cb.halfOpenMu.Unlock()

	// Check if we're still in the half-open window
	if time.Since(cb.halfOpenWindow) > cb.resetTimeout {
		// Window expired, transition back to open
		cb.transitionToOpen()
		return false
	}

	// Check if we've reached the limit
	if cb.halfOpenCount >= cb.halfOpenLimit {
		return false
	}

	cb.halfOpenCount++
	return true
}

// stateString returns a string representation of the state
func (cb *CircuitBreaker) stateString(state CircuitState) string {
	switch CircuitState(state) {
	case StateClosed:
		return "closed"
	case StateOpen:
		return "open"
	case StateHalfOpen:
		return "half_open"
	default:
		return "unknown"
	}
}
