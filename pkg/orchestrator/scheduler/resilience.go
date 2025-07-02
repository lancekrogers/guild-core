// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package scheduler

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/lancekrogers/guild/pkg/gerror"
)

// CircuitBreaker implements the circuit breaker pattern for agent health
type CircuitBreaker struct {
	agentID          string
	failureThreshold int
	resetTimeout     time.Duration
	halfOpenRequests int

	failures         int
	lastFailureTime  time.Time
	state            CircuitState
	halfOpenAttempts int
	mu               sync.RWMutex
}

// CircuitState represents the state of a circuit breaker
type CircuitState int

const (
	CircuitClosed CircuitState = iota
	CircuitOpen
	CircuitHalfOpen
)

// NewCircuitBreaker creates a new circuit breaker for an agent
func NewCircuitBreaker(agentID string, failureThreshold int, resetTimeout time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		agentID:          agentID,
		failureThreshold: failureThreshold,
		resetTimeout:     resetTimeout,
		halfOpenRequests: 3, // Allow 3 requests in half-open state
		state:            CircuitClosed,
	}
}

// Call executes a function with circuit breaker protection
func (cb *CircuitBreaker) Call(ctx context.Context, fn func() error) error {
	if err := ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("orchestrator.scheduler").
			WithOperation("CircuitBreaker.Call")
	}

	cb.mu.Lock()
	state := cb.state

	// Check if we should transition from Open to HalfOpen
	if state == CircuitOpen && time.Since(cb.lastFailureTime) > cb.resetTimeout {
		cb.state = CircuitHalfOpen
		cb.halfOpenAttempts = 0
		state = CircuitHalfOpen
	}
	cb.mu.Unlock()

	// Check circuit state
	switch state {
	case CircuitOpen:
		return gerror.New(gerror.ErrCodeResourceExhausted, "circuit breaker is open", nil).
			WithComponent("orchestrator.scheduler").
			WithOperation("CircuitBreaker.Call").
			WithDetails("agent_id", cb.agentID).
			WithDetails("failures", cb.failures)

	case CircuitHalfOpen:
		cb.mu.Lock()
		if cb.halfOpenAttempts >= cb.halfOpenRequests {
			cb.mu.Unlock()
			return gerror.New(gerror.ErrCodeResourceExhausted, "circuit breaker half-open limit reached", nil).
				WithComponent("orchestrator.scheduler").
				WithOperation("CircuitBreaker.Call").
				WithDetails("agent_id", cb.agentID)
		}
		cb.halfOpenAttempts++
		cb.mu.Unlock()
	}

	// Execute the function
	err := fn()

	cb.mu.Lock()
	defer cb.mu.Unlock()

	if err != nil {
		cb.recordFailure()
		return err
	}

	cb.recordSuccess()
	return nil
}

func (cb *CircuitBreaker) recordFailure() {
	cb.failures++
	cb.lastFailureTime = time.Now()

	if cb.failures >= cb.failureThreshold {
		cb.state = CircuitOpen
	}
}

func (cb *CircuitBreaker) recordSuccess() {
	if cb.state == CircuitHalfOpen {
		// Success in half-open state, close the circuit
		cb.state = CircuitClosed
		cb.failures = 0
	} else if cb.state == CircuitClosed && cb.failures > 0 {
		// Decay failures on success
		cb.failures--
	}
}

// GetState returns the current circuit state
func (cb *CircuitBreaker) GetState() CircuitState {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

// Reset manually resets the circuit breaker
func (cb *CircuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.state = CircuitClosed
	cb.failures = 0
	cb.halfOpenAttempts = 0
}

// RetryPolicy defines retry behavior for failed operations
type RetryPolicy struct {
	MaxAttempts     int
	InitialDelay    time.Duration
	MaxDelay        time.Duration
	BackoffFactor   float64
	JitterFactor    float64
	RetryableErrors map[gerror.ErrorCode]bool
}

// DefaultRetryPolicy returns a sensible default retry policy
func DefaultRetryPolicy() *RetryPolicy {
	return &RetryPolicy{
		MaxAttempts:   3,
		InitialDelay:  100 * time.Millisecond,
		MaxDelay:      5 * time.Second,
		BackoffFactor: 2.0,
		JitterFactor:  0.1,
		RetryableErrors: map[gerror.ErrorCode]bool{
			gerror.ErrCodeTimeout:           true,
			gerror.ErrCodeResourceExhausted: true,
			gerror.ErrCodeInternal:          true,
		},
	}
}

// Retry executes a function with retry logic
func (rp *RetryPolicy) Retry(ctx context.Context, operation string, fn func() error) error {
	var lastErr error
	delay := rp.InitialDelay

	for attempt := 0; attempt < rp.MaxAttempts; attempt++ {
		// Check context before each attempt
		if err := ctx.Err(); err != nil {
			return gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled during retry").
				WithComponent("orchestrator.scheduler").
				WithOperation("RetryPolicy.Retry").
				WithDetails("operation", operation).
				WithDetails("attempt", attempt)
		}

		// Execute the function
		err := fn()
		if err == nil {
			return nil
		}

		lastErr = err

		// Check if error is retryable
		if gerr, ok := err.(*gerror.GuildError); ok {
			if !rp.RetryableErrors[gerr.Code] {
				return err
			}
		}

		// Don't sleep after the last attempt
		if attempt < rp.MaxAttempts-1 {
			// Add jitter to prevent thundering herd
			jitter := time.Duration(float64(delay) * rp.JitterFactor * (0.5 - float64(time.Now().UnixNano()%1000)/1000))
			sleepDuration := delay + jitter

			select {
			case <-time.After(sleepDuration):
			case <-ctx.Done():
				return gerror.Wrap(ctx.Err(), gerror.ErrCodeCancelled, "context cancelled during retry delay").
					WithComponent("orchestrator.scheduler").
					WithOperation("RetryPolicy.Retry").
					WithDetails("operation", operation).
					WithDetails("attempt", attempt)
			}

			// Exponential backoff
			delay = time.Duration(float64(delay) * rp.BackoffFactor)
			if delay > rp.MaxDelay {
				delay = rp.MaxDelay
			}
		}
	}

	return gerror.Wrap(lastErr, gerror.ErrCodeResourceExhausted, "retry attempts exhausted").
		WithComponent("orchestrator.scheduler").
		WithOperation("RetryPolicy.Retry").
		WithDetails("operation", operation).
		WithDetails("attempts", rp.MaxAttempts)
}

// HealthMonitor tracks agent health and performance
type HealthMonitor struct {
	agents      map[string]*AgentHealth
	window      time.Duration
	mu          sync.RWMutex
	stopChan    chan struct{}
	cleanupTick time.Duration
}

// AgentHealth tracks health metrics for a single agent
type AgentHealth struct {
	AgentID         string
	TotalRequests   int64
	SuccessfulTasks int64
	FailedTasks     int64
	TotalLatency    time.Duration
	LastSuccess     time.Time
	LastFailure     time.Time
	RecentErrors    []ErrorRecord
	CircuitBreaker  *CircuitBreaker
	mu              sync.RWMutex
}

// ErrorRecord tracks recent errors
type ErrorRecord struct {
	Timestamp time.Time
	Error     error
	TaskID    string
}

// NewHealthMonitor creates a new health monitor
func NewHealthMonitor(window time.Duration) *HealthMonitor {
	// Ensure minimum window duration
	if window < time.Second {
		window = 5 * time.Minute // Default window
	}

	cleanupInterval := window / 10
	if cleanupInterval < time.Second {
		cleanupInterval = time.Second // Minimum cleanup interval
	}

	hm := &HealthMonitor{
		agents:      make(map[string]*AgentHealth),
		window:      window,
		stopChan:    make(chan struct{}),
		cleanupTick: cleanupInterval,
	}

	go hm.cleanupLoop()
	return hm
}

// RecordSuccess records a successful task execution
func (hm *HealthMonitor) RecordSuccess(agentID string, latency time.Duration) {
	hm.mu.Lock()
	health, exists := hm.agents[agentID]
	if !exists {
		health = &AgentHealth{
			AgentID:        agentID,
			RecentErrors:   make([]ErrorRecord, 0),
			CircuitBreaker: NewCircuitBreaker(agentID, 5, 30*time.Second),
		}
		hm.agents[agentID] = health
	}
	hm.mu.Unlock()

	health.mu.Lock()
	defer health.mu.Unlock()

	health.TotalRequests++
	health.SuccessfulTasks++
	health.TotalLatency += latency
	health.LastSuccess = time.Now()
}

// RecordFailure records a failed task execution
func (hm *HealthMonitor) RecordFailure(agentID string, taskID string, err error) {
	hm.mu.Lock()
	health, exists := hm.agents[agentID]
	if !exists {
		health = &AgentHealth{
			AgentID:        agentID,
			RecentErrors:   make([]ErrorRecord, 0),
			CircuitBreaker: NewCircuitBreaker(agentID, 5, 30*time.Second),
		}
		hm.agents[agentID] = health
	}
	hm.mu.Unlock()

	health.mu.Lock()
	defer health.mu.Unlock()

	health.TotalRequests++
	health.FailedTasks++
	health.LastFailure = time.Now()

	// Add to recent errors
	health.RecentErrors = append(health.RecentErrors, ErrorRecord{
		Timestamp: time.Now(),
		Error:     err,
		TaskID:    taskID,
	})

	// Keep only last 100 errors
	if len(health.RecentErrors) > 100 {
		health.RecentErrors = health.RecentErrors[len(health.RecentErrors)-100:]
	}
}

// GetAgentHealth returns health metrics for an agent
func (hm *HealthMonitor) GetAgentHealth(agentID string) (*AgentHealth, bool) {
	hm.mu.RLock()
	defer hm.mu.RUnlock()

	health, exists := hm.agents[agentID]
	if !exists {
		return nil, false
	}

	// Return a copy to avoid race conditions
	health.mu.RLock()
	defer health.mu.RUnlock()

	copy := &AgentHealth{
		AgentID:         health.AgentID,
		TotalRequests:   health.TotalRequests,
		SuccessfulTasks: health.SuccessfulTasks,
		FailedTasks:     health.FailedTasks,
		TotalLatency:    health.TotalLatency,
		LastSuccess:     health.LastSuccess,
		LastFailure:     health.LastFailure,
		CircuitBreaker:  health.CircuitBreaker,
		RecentErrors:    make([]ErrorRecord, len(health.RecentErrors)),
	}

	copy.RecentErrors = append(copy.RecentErrors, health.RecentErrors...)
	return copy, true
}

// GetHealthReport generates a health report for all agents
func (hm *HealthMonitor) GetHealthReport() map[string]HealthMetrics {
	hm.mu.RLock()
	defer hm.mu.RUnlock()

	report := make(map[string]HealthMetrics)

	for agentID, health := range hm.agents {
		health.mu.RLock()

		var successRate float64
		if health.TotalRequests > 0 {
			successRate = float64(health.SuccessfulTasks) / float64(health.TotalRequests) * 100
		}

		var avgLatency time.Duration
		if health.SuccessfulTasks > 0 {
			avgLatency = health.TotalLatency / time.Duration(health.SuccessfulTasks)
		}

		metrics := HealthMetrics{
			AgentID:         agentID,
			SuccessRate:     successRate,
			TotalRequests:   health.TotalRequests,
			FailedRequests:  health.FailedTasks,
			AverageLatency:  avgLatency,
			RecentErrorRate: hm.calculateRecentErrorRate(health),
			CircuitState:    health.CircuitBreaker.GetState(),
			LastSuccess:     health.LastSuccess,
			LastFailure:     health.LastFailure,
		}

		health.mu.RUnlock()
		report[agentID] = metrics
	}

	return report
}

// HealthMetrics contains computed health metrics
type HealthMetrics struct {
	AgentID         string
	SuccessRate     float64
	TotalRequests   int64
	FailedRequests  int64
	AverageLatency  time.Duration
	RecentErrorRate float64
	CircuitState    CircuitState
	LastSuccess     time.Time
	LastFailure     time.Time
}

func (hm *HealthMonitor) calculateRecentErrorRate(health *AgentHealth) float64 {
	cutoff := time.Now().Add(-hm.window)
	recentErrors := 0

	for _, err := range health.RecentErrors {
		if err.Timestamp.After(cutoff) {
			recentErrors++
		}
	}

	// Estimate recent request rate based on total requests and time
	if health.LastSuccess.IsZero() && health.LastFailure.IsZero() {
		return 0
	}

	var lastActivity time.Time
	if health.LastSuccess.After(health.LastFailure) {
		lastActivity = health.LastSuccess
	} else {
		lastActivity = health.LastFailure
	}

	timeSinceStart := time.Since(lastActivity.Add(-hm.window))
	if timeSinceStart <= 0 {
		return 0
	}

	estimatedRecentRequests := float64(health.TotalRequests) * (float64(hm.window) / float64(timeSinceStart))
	if estimatedRecentRequests == 0 {
		return 0
	}

	return float64(recentErrors) / estimatedRecentRequests * 100
}

func (hm *HealthMonitor) cleanupLoop() {
	ticker := time.NewTicker(hm.cleanupTick)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			hm.cleanupOldErrors()
		case <-hm.stopChan:
			return
		}
	}
}

func (hm *HealthMonitor) cleanupOldErrors() {
	hm.mu.RLock()
	agents := make([]*AgentHealth, 0, len(hm.agents))
	for _, health := range hm.agents {
		agents = append(agents, health)
	}
	hm.mu.RUnlock()

	cutoff := time.Now().Add(-hm.window)

	for _, health := range agents {
		health.mu.Lock()

		// Remove old errors
		newErrors := make([]ErrorRecord, 0, len(health.RecentErrors))
		for _, err := range health.RecentErrors {
			if err.Timestamp.After(cutoff) {
				newErrors = append(newErrors, err)
			}
		}
		health.RecentErrors = newErrors

		health.mu.Unlock()
	}
}

// Stop stops the health monitor
func (hm *HealthMonitor) Stop() {
	close(hm.stopChan)
}

// GetCircuitBreaker returns the circuit breaker for an agent
func (hm *HealthMonitor) GetCircuitBreaker(agentID string) *CircuitBreaker {
	hm.mu.Lock()
	defer hm.mu.Unlock()

	health, exists := hm.agents[agentID]
	if !exists {
		health = &AgentHealth{
			AgentID:        agentID,
			RecentErrors:   make([]ErrorRecord, 0),
			CircuitBreaker: NewCircuitBreaker(agentID, 5, 30*time.Second),
		}
		hm.agents[agentID] = health
	}

	return health.CircuitBreaker
}

// TaskRecovery handles task recovery and dead letter queue
type TaskRecovery struct {
	deadLetterQueue []*DeadLetterTask
	maxRetries      int
	retryPolicy     *RetryPolicy
	mu              sync.RWMutex
}

// DeadLetterTask represents a task that failed all retry attempts
type DeadLetterTask struct {
	Task         *SchedulableTask
	FailureTime  time.Time
	FailureCount int
	LastError    error
	AgentID      string
}

// NewTaskRecovery creates a new task recovery handler
func NewTaskRecovery(maxRetries int) *TaskRecovery {
	return &TaskRecovery{
		deadLetterQueue: make([]*DeadLetterTask, 0),
		maxRetries:      maxRetries,
		retryPolicy:     DefaultRetryPolicy(),
	}
}

// HandleFailedTask processes a failed task
func (tr *TaskRecovery) HandleFailedTask(task *SchedulableTask, agentID string, err error, failureCount int) bool {
	// Check if task should be retried
	if failureCount >= tr.maxRetries {
		tr.mu.Lock()
		defer tr.mu.Unlock()

		// Add to dead letter queue
		tr.deadLetterQueue = append(tr.deadLetterQueue, &DeadLetterTask{
			Task:         task,
			FailureTime:  time.Now(),
			FailureCount: failureCount,
			LastError:    err,
			AgentID:      agentID,
		})

		return false // Don't retry
	}

	// Check if error is retryable
	if gerr, ok := err.(*gerror.GuildError); ok {
		return tr.retryPolicy.RetryableErrors[gerr.Code]
	}

	return true // Retry by default
}

// GetDeadLetterQueue returns tasks in the dead letter queue
func (tr *TaskRecovery) GetDeadLetterQueue() []*DeadLetterTask {
	tr.mu.RLock()
	defer tr.mu.RUnlock()

	// Return a copy
	result := make([]*DeadLetterTask, len(tr.deadLetterQueue))
	copy(result, tr.deadLetterQueue)
	return result
}

// ResubmitDeadLetterTask attempts to resubmit a task from the dead letter queue
func (tr *TaskRecovery) ResubmitDeadLetterTask(index int) (*SchedulableTask, error) {
	tr.mu.Lock()
	defer tr.mu.Unlock()

	if index < 0 || index >= len(tr.deadLetterQueue) {
		return nil, fmt.Errorf("invalid dead letter queue index: %d", index)
	}

	// Remove from dead letter queue
	dlt := tr.deadLetterQueue[index]
	tr.deadLetterQueue = append(tr.deadLetterQueue[:index], tr.deadLetterQueue[index+1:]...)

	// Reset failure count for resubmission
	return dlt.Task, nil
}

// ClearDeadLetterQueue removes all tasks from the dead letter queue
func (tr *TaskRecovery) ClearDeadLetterQueue() {
	tr.mu.Lock()
	defer tr.mu.Unlock()
	tr.deadLetterQueue = make([]*DeadLetterTask, 0)
}
