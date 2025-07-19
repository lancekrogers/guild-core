// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package reasoning

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/lancekrogers/guild/pkg/gerror"
)

// CircuitState represents the state of the circuit breaker
type CircuitState int32

const (
	// StateClosed allows requests through
	StateClosed CircuitState = iota
	// StateOpen blocks all requests
	StateOpen
	// StateHalfOpen allows limited requests for testing
	StateHalfOpen
)

// String returns the string representation of the circuit state
func (s CircuitState) String() string {
	switch s {
	case StateClosed:
		return "closed"
	case StateOpen:
		return "open"
	case StateHalfOpen:
		return "half-open"
	default:
		return "unknown"
	}
}

// CircuitBreakerConfig holds configuration for the circuit breaker
type CircuitBreakerConfig struct {
	// FailureThreshold is the number of failures before opening
	FailureThreshold int
	// SuccessThreshold is the number of successes in half-open before closing
	SuccessThreshold int
	// Timeout is how long to wait before trying half-open
	Timeout time.Duration
	// HalfOpenRequests is the max requests allowed in half-open state
	HalfOpenRequests int
	// OnStateChange is called when state changes
	OnStateChange func(from, to CircuitState)
}

// CircuitBreaker implements the circuit breaker pattern
type CircuitBreaker struct {
	// Configuration (immutable after creation)
	config CircuitBreakerConfig

	// State management (atomic for lock-free reads)
	state atomic.Int32

	// Mutable state (protected by mutex)
	mu                 sync.Mutex
	failures           int
	successes          int
	lastFailureTime    time.Time
	lastSuccessTime    time.Time
	halfOpenRequests   int
	halfOpenInProgress int
	generation         uint64
}

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(config CircuitBreakerConfig) *CircuitBreaker {
	// Set defaults
	if config.FailureThreshold <= 0 {
		config.FailureThreshold = 5
	}
	if config.SuccessThreshold <= 0 {
		config.SuccessThreshold = 2
	}
	if config.Timeout <= 0 {
		config.Timeout = 30 * time.Second
	}
	if config.HalfOpenRequests <= 0 {
		config.HalfOpenRequests = 3
	}

	cb := &CircuitBreaker{
		config: config,
	}
	cb.state.Store(int32(StateClosed))
	return cb
}

// Execute runs the given function with circuit breaker protection
func (cb *CircuitBreaker) Execute(ctx context.Context, fn func() error) error {
	// Check context first
	if err := ctx.Err(); err != nil {
		return gerror.Wrap(err, "context cancelled").
			WithCode(gerror.ErrCodeCanceled).
			WithComponent("circuit_breaker")
	}

	// Fast path - check if we can execute
	if !cb.canExecute() {
		return gerror.New("circuit breaker open").
			WithCode(gerror.ErrCodeResourceExhausted).
			WithComponent("circuit_breaker").
			WithField("state", cb.State().String()).
			WithField("last_failure", cb.lastFailureTime)
	}

	// For half-open state, track in-progress requests
	if cb.State() == StateHalfOpen {
		if !cb.acquireHalfOpenSlot() {
			return gerror.New("circuit breaker half-open limit reached").
				WithCode(gerror.ErrCodeResourceExhausted).
				WithComponent("circuit_breaker").
				WithField("limit", cb.config.HalfOpenRequests)
		}
		defer cb.releaseHalfOpenSlot()
	}

	// Execute the function
	start := time.Now()
	err := fn()
	duration := time.Since(start)

	// Record the result
	cb.recordResult(err, duration)

	return err
}

// State returns the current state
func (cb *CircuitBreaker) State() CircuitState {
	return CircuitState(cb.state.Load())
}

// canExecute checks if a request can be executed
func (cb *CircuitBreaker) canExecute() bool {
	state := cb.State()

	switch state {
	case StateClosed:
		return true
	case StateOpen:
		// Check if timeout has passed
		cb.mu.Lock()
		defer cb.mu.Unlock()

		if time.Since(cb.lastFailureTime) > cb.config.Timeout {
			// Transition to half-open
			cb.transitionTo(StateHalfOpen)
			return true
		}
		return false
	case StateHalfOpen:
		// Check if we have capacity
		cb.mu.Lock()
		defer cb.mu.Unlock()
		return cb.halfOpenInProgress < cb.config.HalfOpenRequests
	default:
		return false
	}
}

// acquireHalfOpenSlot tries to acquire a slot for half-open execution
func (cb *CircuitBreaker) acquireHalfOpenSlot() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if cb.State() != StateHalfOpen {
		return false
	}

	if cb.halfOpenInProgress >= cb.config.HalfOpenRequests {
		return false
	}

	cb.halfOpenInProgress++
	return true
}

// releaseHalfOpenSlot releases a half-open execution slot
func (cb *CircuitBreaker) releaseHalfOpenSlot() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if cb.halfOpenInProgress > 0 {
		cb.halfOpenInProgress--
	}
}

// recordResult records the result of an execution
func (cb *CircuitBreaker) recordResult(err error, duration time.Duration) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	state := cb.State()

	if err != nil {
		cb.recordFailure(state)
	} else {
		cb.recordSuccess(state)
	}
}

// recordFailure handles a failed execution
func (cb *CircuitBreaker) recordFailure(state CircuitState) {
	cb.failures++
	cb.lastFailureTime = time.Now()

	switch state {
	case StateClosed:
		if cb.failures >= cb.config.FailureThreshold {
			cb.transitionTo(StateOpen)
		}
	case StateHalfOpen:
		// Any failure in half-open goes back to open
		cb.transitionTo(StateOpen)
	}
}

// recordSuccess handles a successful execution
func (cb *CircuitBreaker) recordSuccess(state CircuitState) {
	cb.lastSuccessTime = time.Now()

	switch state {
	case StateClosed:
		// Reset failure count on success
		cb.failures = 0
	case StateHalfOpen:
		cb.successes++
		if cb.successes >= cb.config.SuccessThreshold {
			cb.transitionTo(StateClosed)
		}
	}
}

// transitionTo changes the circuit breaker state
func (cb *CircuitBreaker) transitionTo(newState CircuitState) {
	oldState := cb.State()
	if oldState == newState {
		return
	}

	// Update state atomically
	cb.state.Store(int32(newState))

	// Reset counters based on new state
	switch newState {
	case StateClosed:
		cb.failures = 0
		cb.successes = 0
		cb.halfOpenInProgress = 0
	case StateOpen:
		cb.successes = 0
		cb.halfOpenInProgress = 0
	case StateHalfOpen:
		cb.failures = 0
		cb.successes = 0
		cb.halfOpenInProgress = 0
		cb.generation++
	}

	// Call state change callback if configured
	if cb.config.OnStateChange != nil {
		// Call in goroutine to avoid blocking
		go cb.config.OnStateChange(oldState, newState)
	}
}

// Reset manually resets the circuit breaker to closed state
func (cb *CircuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.transitionTo(StateClosed)
	cb.failures = 0
	cb.successes = 0
	cb.lastFailureTime = time.Time{}
	cb.lastSuccessTime = time.Time{}
}

// GetState returns the current state and statistics
func (cb *CircuitBreaker) GetState() (state CircuitState, failures int, lastFailure, lastSuccess time.Time) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	return cb.State(), cb.failures, cb.lastFailureTime, cb.lastSuccessTime
}

// RestoreState restores the circuit breaker state (used for persistence)
func (cb *CircuitBreaker) RestoreState(state CircuitState, failures int, lastFailure, lastSuccess time.Time) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.state.Store(int32(state))
	cb.failures = failures
	cb.lastFailureTime = lastFailure
	cb.lastSuccessTime = lastSuccess

	// Check if we should auto-transition from open to half-open
	if state == StateOpen && time.Since(lastFailure) > cb.config.Timeout {
		cb.transitionTo(StateHalfOpen)
	}
}

// Statistics returns current circuit breaker statistics
func (cb *CircuitBreaker) Statistics() map[string]interface{} {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	stats := map[string]interface{}{
		"state":                 cb.State().String(),
		"failures":              cb.failures,
		"successes":             cb.successes,
		"generation":            cb.generation,
		"half_open_in_progress": cb.halfOpenInProgress,
		"failure_threshold":     cb.config.FailureThreshold,
		"success_threshold":     cb.config.SuccessThreshold,
		"timeout_seconds":       cb.config.Timeout.Seconds(),
	}

	if !cb.lastFailureTime.IsZero() {
		stats["last_failure"] = cb.lastFailureTime
		stats["time_since_failure"] = time.Since(cb.lastFailureTime).Seconds()
	}

	if !cb.lastSuccessTime.IsZero() {
		stats["last_success"] = cb.lastSuccessTime
	}

	return stats
}
