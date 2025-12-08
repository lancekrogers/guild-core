// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package routing

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/guild-framework/guild-core/pkg/eventbus"
	"github.com/guild-framework/guild-core/pkg/events"
	"github.com/guild-framework/guild-core/pkg/gerror"
)

// RateLimitingMiddleware limits event processing rate
func RateLimitingMiddleware(ratePerSecond int, burst int) eventbus.Middleware {
	limiter := &rateLimiter{
		rate:       ratePerSecond,
		burst:      burst,
		tokens:     burst,
		lastRefill: time.Now(),
	}

	return func(next events.EventHandler) events.EventHandler {
		return func(ctx context.Context, event events.CoreEvent) error {
			if !limiter.Allow() {
				return gerror.New(gerror.ErrCodeResourceExhausted, "rate limit exceeded", nil).
					WithDetails("event_type", event.GetType())
			}
			return next(ctx, event)
		}
	}
}

// rateLimiter implements token bucket algorithm
type rateLimiter struct {
	rate       int
	burst      int
	tokens     int
	lastRefill time.Time
	mu         sync.Mutex
}

func (rl *rateLimiter) Allow() bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	// Refill tokens
	now := time.Now()
	elapsed := now.Sub(rl.lastRefill)
	tokensToAdd := int(elapsed.Seconds() * float64(rl.rate))

	if tokensToAdd > 0 {
		rl.tokens = min(rl.tokens+tokensToAdd, rl.burst)
		rl.lastRefill = now
	}

	// Check if we have tokens
	if rl.tokens > 0 {
		rl.tokens--
		return true
	}

	return false
}

// RetryMiddleware adds retry capability with exponential backoff
func RetryMiddleware(maxRetries int, initialDelay time.Duration) eventbus.Middleware {
	return func(next events.EventHandler) events.EventHandler {
		return func(ctx context.Context, event events.CoreEvent) error {
			var lastErr error
			delay := initialDelay

			for attempt := 0; attempt <= maxRetries; attempt++ {
				if attempt > 0 {
					select {
					case <-time.After(delay):
						delay *= 2 // Exponential backoff
					case <-ctx.Done():
						return gerror.Wrap(ctx.Err(), gerror.ErrCodeCancelled, "context cancelled during retry")
					}
				}

				lastErr = next(ctx, event)
				if lastErr == nil {
					return nil
				}

				// Don't retry on certain errors
				if gerror.GetCode(lastErr) == gerror.ErrCodeCancelled {
					return lastErr
				}
			}

			return gerror.Wrap(lastErr, gerror.ErrCodeInternal, "max retries exceeded").
				WithDetails("attempts", maxRetries+1)
		}
	}
}

// CachingMiddleware caches event processing results
func CachingMiddleware(ttl time.Duration) eventbus.Middleware {
	cache := &eventCache{
		entries: make(map[string]*cacheEntry),
		ttl:     ttl,
	}

	// Start cleanup goroutine
	go cache.cleanup()

	return func(next events.EventHandler) events.EventHandler {
		return func(ctx context.Context, event events.CoreEvent) error {
			key := fmt.Sprintf("%s:%s", event.GetType(), event.GetID())

			// Check cache
			if entry := cache.Get(key); entry != nil {
				if entry.Error != nil {
					return entry.Error
				}
				return nil // Already processed successfully
			}

			// Process event
			err := next(ctx, event)

			// Cache result
			cache.Set(key, err)

			return err
		}
	}
}

// eventCache stores processed event results
type eventCache struct {
	entries map[string]*cacheEntry
	ttl     time.Duration
	mu      sync.RWMutex
}

type cacheEntry struct {
	Error     error
	Timestamp time.Time
}

func (c *eventCache) Get(key string) *cacheEntry {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, exists := c.entries[key]
	if !exists {
		return nil
	}

	// Check if expired
	if time.Since(entry.Timestamp) > c.ttl {
		return nil
	}

	return entry
}

func (c *eventCache) Set(key string, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries[key] = &cacheEntry{
		Error:     err,
		Timestamp: time.Now(),
	}
}

func (c *eventCache) cleanup() {
	ticker := time.NewTicker(c.ttl)
	defer ticker.Stop()

	for range ticker.C {
		c.mu.Lock()
		now := time.Now()
		for key, entry := range c.entries {
			if now.Sub(entry.Timestamp) > c.ttl {
				delete(c.entries, key)
			}
		}
		c.mu.Unlock()
	}
}

// ValidationMiddleware validates events against schemas
func ValidationMiddleware(validator EventValidator) eventbus.Middleware {
	return func(next events.EventHandler) events.EventHandler {
		return func(ctx context.Context, event events.CoreEvent) error {
			if err := validator.Validate(ctx, event); err != nil {
				return gerror.Wrap(err, gerror.ErrCodeValidation, "event validation failed")
			}
			return next(ctx, event)
		}
	}
}

// EventValidator validates events
type EventValidator interface {
	Validate(ctx context.Context, event events.CoreEvent) error
}

// TracingMiddleware adds distributed tracing
func TracingMiddleware(tracer EventTracer) eventbus.Middleware {
	return func(next events.EventHandler) events.EventHandler {
		return func(ctx context.Context, event events.CoreEvent) error {
			span := tracer.StartSpan(ctx, "event.handle", event)
			defer span.End()

			err := next(span.Context(), event)
			if err != nil {
				span.RecordError(err)
			}

			return err
		}
	}
}

// EventTracer provides tracing capabilities
type EventTracer interface {
	StartSpan(ctx context.Context, name string, event events.CoreEvent) Span
}

// Span represents a trace span
type Span interface {
	Context() context.Context
	End()
	RecordError(error)
}

// BulkheadMiddleware implements the bulkhead pattern
func BulkheadMiddleware(maxConcurrent int) eventbus.Middleware {
	semaphore := make(chan struct{}, maxConcurrent)

	return func(next events.EventHandler) events.EventHandler {
		return func(ctx context.Context, event events.CoreEvent) error {
			select {
			case semaphore <- struct{}{}:
				defer func() { <-semaphore }()
				return next(ctx, event)
			case <-ctx.Done():
				return gerror.Wrap(ctx.Err(), gerror.ErrCodeCancelled, "context cancelled waiting for bulkhead")
			default:
				return gerror.New(gerror.ErrCodeResourceExhausted, "bulkhead capacity exceeded", nil).
					WithDetails("max_concurrent", maxConcurrent)
			}
		}
	}
}

// TimeoutMiddleware adds timeout to event processing
func TimeoutMiddleware(timeout time.Duration) eventbus.Middleware {
	return func(next events.EventHandler) events.EventHandler {
		return func(ctx context.Context, event events.CoreEvent) error {
			timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
			defer cancel()

			done := make(chan error, 1)
			go func() {
				done <- next(timeoutCtx, event)
			}()

			select {
			case err := <-done:
				return err
			case <-timeoutCtx.Done():
				return gerror.Wrap(timeoutCtx.Err(), gerror.ErrCodeTimeout, "event processing timeout").
					WithDetails("timeout", timeout)
			}
		}
	}
}

// DedupingMiddleware prevents duplicate event processing
func DedupingMiddleware(window time.Duration) eventbus.Middleware {
	seen := &dedupCache{
		events: make(map[string]time.Time),
		window: window,
	}

	// Start cleanup
	go seen.cleanup()

	return func(next events.EventHandler) events.EventHandler {
		return func(ctx context.Context, event events.CoreEvent) error {
			if seen.IsDuplicate(event.GetID()) {
				return nil // Skip duplicate
			}

			seen.Add(event.GetID())
			return next(ctx, event)
		}
	}
}

// dedupCache tracks seen events
type dedupCache struct {
	events map[string]time.Time
	window time.Duration
	mu     sync.RWMutex
}

func (d *dedupCache) IsDuplicate(eventID string) bool {
	d.mu.RLock()
	defer d.mu.RUnlock()

	timestamp, exists := d.events[eventID]
	if !exists {
		return false
	}

	return time.Since(timestamp) < d.window
}

func (d *dedupCache) Add(eventID string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.events[eventID] = time.Now()
}

func (d *dedupCache) cleanup() {
	ticker := time.NewTicker(d.window)
	defer ticker.Stop()

	for range ticker.C {
		d.mu.Lock()
		now := time.Now()
		for id, timestamp := range d.events {
			if now.Sub(timestamp) > d.window {
				delete(d.events, id)
			}
		}
		d.mu.Unlock()
	}
}

// Helper functions

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
