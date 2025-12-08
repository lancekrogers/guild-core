// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package reasoning

import (
	"context"
	"math"
	"math/rand"
	"time"

	"github.com/guild-framework/guild-core/pkg/gerror"
)

// RetryConfig configures retry behavior
type RetryConfig struct {
	// MaxAttempts is the maximum number of retry attempts
	MaxAttempts int
	// InitialDelay is the initial delay between retries
	InitialDelay time.Duration
	// MaxDelay is the maximum delay between retries
	MaxDelay time.Duration
	// Multiplier is the exponential backoff multiplier
	Multiplier float64
	// JitterFactor adds randomness to prevent thundering herd (0.0 to 1.0)
	JitterFactor float64
	// RetryableErrors defines which error codes should be retried
	RetryableErrors []gerror.ErrorCode
	// OnRetry is called before each retry attempt
	OnRetry func(attempt int, err error, delay time.Duration)
}

// DefaultRetryConfig returns default retry configuration
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxAttempts:  3,
		InitialDelay: 100 * time.Millisecond,
		MaxDelay:     10 * time.Second,
		Multiplier:   2.0,
		JitterFactor: 0.1,
		RetryableErrors: []gerror.ErrorCode{
			gerror.ErrCodeTimeout,
			gerror.ErrCodeResourceExhausted,
			gerror.ErrCodeInternal,
			gerror.ErrCodeConnection,
		},
	}
}

// Retryer implements retry logic with exponential backoff
type Retryer struct {
	config RetryConfig
	rng    *rand.Rand
}

// NewRetryer creates a new retryer
func NewRetryer(config RetryConfig) *Retryer {
	// Validate config
	if config.MaxAttempts <= 0 {
		config.MaxAttempts = 1
	}
	if config.InitialDelay <= 0 {
		config.InitialDelay = 100 * time.Millisecond
	}
	if config.MaxDelay <= 0 {
		config.MaxDelay = 10 * time.Second
	}
	if config.Multiplier <= 1.0 {
		config.Multiplier = 2.0
	}
	if config.JitterFactor < 0 {
		config.JitterFactor = 0
	} else if config.JitterFactor > 1.0 {
		config.JitterFactor = 1.0
	}

	return &Retryer{
		config: config,
		rng:    rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// Execute runs the function with retry logic
func (r *Retryer) Execute(ctx context.Context, fn func() error) error {
	var lastErr error

	for attempt := 0; attempt < r.config.MaxAttempts; attempt++ {
		// Check context before attempt
		if err := ctx.Err(); err != nil {
			return gerror.Wrap(err, gerror.ErrCodeCanceled, "context cancelled during retry").
				WithComponent("retryer").
				WithDetails("attempt", attempt)
		}

		// Execute the function
		err := fn()
		if err == nil {
			return nil // Success
		}

		lastErr = err

		// Check if error is retryable
		if !r.isRetryable(err) {
			return err
		}

		// Check if we have more attempts
		if attempt >= r.config.MaxAttempts-1 {
			break // No more retries
		}

		// Calculate delay with exponential backoff
		delay := r.calculateDelay(attempt)

		// Add jitter to prevent thundering herd
		delay = r.addJitter(delay)

		// Call retry callback if configured
		if r.config.OnRetry != nil {
			r.config.OnRetry(attempt+1, err, delay)
		}

		// Wait before retry
		select {
		case <-time.After(delay):
			// Continue to next attempt
		case <-ctx.Done():
			return gerror.Wrap(ctx.Err(), gerror.ErrCodeCanceled, "context cancelled during retry delay").
				WithComponent("retryer").
				WithDetails("attempt", attempt).
				WithDetails("delay_ms", delay.Milliseconds())
		}
	}

	// All retries exhausted
	return gerror.Wrap(lastErr, gerror.ErrCodeResourceExhausted, "all retry attempts exhausted").
		WithComponent("retryer").
		WithDetails("attempts", r.config.MaxAttempts)
}

// ExecuteWithResult runs the function with retry logic and returns result
func (r *Retryer) ExecuteWithResult(ctx context.Context, fn func() (interface{}, error)) (interface{}, error) {
	var result interface{}

	err := r.Execute(ctx, func() error {
		var err error
		result, err = fn()
		return err
	})

	if err != nil {
		return nil, err
	}

	return result, nil
}

// isRetryable checks if an error should be retried
func (r *Retryer) isRetryable(err error) bool {
	// Check if it's a gerror with a code
	var gerr *gerror.GuildError
	if !gerror.As(err, &gerr) {
		// Not a gerror, don't retry by default
		return false
	}

	// Check if error code is in retryable list
	for _, code := range r.config.RetryableErrors {
		if gerr.Code == code {
			return true
		}
	}

	return false
}

// calculateDelay calculates the delay for the given attempt
func (r *Retryer) calculateDelay(attempt int) time.Duration {
	// Exponential backoff: delay = initialDelay * (multiplier ^ attempt)
	delay := float64(r.config.InitialDelay) * math.Pow(r.config.Multiplier, float64(attempt))

	// Cap at max delay
	if delay > float64(r.config.MaxDelay) {
		delay = float64(r.config.MaxDelay)
	}

	return time.Duration(delay)
}

// addJitter adds randomness to the delay
func (r *Retryer) addJitter(delay time.Duration) time.Duration {
	if r.config.JitterFactor <= 0 {
		return delay
	}

	// Calculate jitter range
	jitterRange := float64(delay) * r.config.JitterFactor

	// Add random jitter (positive or negative)
	jitter := (r.rng.Float64() - 0.5) * 2 * jitterRange

	finalDelay := float64(delay) + jitter

	// Ensure delay is positive
	if finalDelay < 0 {
		finalDelay = 0
	}

	return time.Duration(finalDelay)
}

// RetryStats tracks retry statistics
type RetryStats struct {
	TotalAttempts   int
	SuccessfulFirst int
	RequiredRetries int
	TotalFailures   int
	RetryDelayTotal time.Duration
}

// StatsRetryer wraps a retryer with statistics collection
type StatsRetryer struct {
	*Retryer
	stats RetryStats
}

// NewStatsRetryer creates a retryer that tracks statistics
func NewStatsRetryer(config RetryConfig) *StatsRetryer {
	return &StatsRetryer{
		Retryer: NewRetryer(config),
	}
}

// Execute runs the function and tracks statistics
func (sr *StatsRetryer) Execute(ctx context.Context, fn func() error) error {
	attempts := 0

	// Wrap the retry callback to track stats
	originalCallback := sr.config.OnRetry
	sr.config.OnRetry = func(attempt int, err error, delay time.Duration) {
		attempts = attempt
		sr.stats.RetryDelayTotal += delay
		if originalCallback != nil {
			originalCallback(attempt, err, delay)
		}
	}

	// Execute with retry
	err := sr.Retryer.Execute(ctx, fn)

	// Update statistics
	sr.stats.TotalAttempts++
	if err == nil {
		if attempts == 0 {
			sr.stats.SuccessfulFirst++
		} else {
			sr.stats.RequiredRetries++
		}
	} else {
		sr.stats.TotalFailures++
	}

	// Restore original callback
	sr.config.OnRetry = originalCallback

	return err
}

// GetStats returns current retry statistics
func (sr *StatsRetryer) GetStats() RetryStats {
	return sr.stats
}

// ResetStats resets the statistics
func (sr *StatsRetryer) ResetStats() {
	sr.stats = RetryStats{}
}
