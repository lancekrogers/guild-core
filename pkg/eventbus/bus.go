// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package eventbus

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/lancekrogers/guild/pkg/events"
	"github.com/lancekrogers/guild/pkg/gerror"
)

// Priority levels for event handling
const (
	PriorityLow    = 0
	PriorityNormal = 50
	PriorityHigh   = 100
)

// Config holds configuration for the enhanced event bus
type Config struct {
	// BufferSize is the size of the event buffer
	BufferSize int

	// WorkerCount is the number of worker goroutines
	WorkerCount int

	// EnablePersistence enables event persistence
	EnablePersistence bool

	// EnableDeadLetter enables dead letter queue
	EnableDeadLetter bool

	// MaxRetries is the maximum number of retries for failed events
	MaxRetries int

	// RetryDelay is the delay between retries
	RetryDelay time.Duration

	// CircuitBreakerThreshold is the failure threshold for circuit breaker
	CircuitBreakerThreshold int

	// CircuitBreakerTimeout is the circuit breaker reset timeout
	CircuitBreakerTimeout time.Duration
}

// DefaultConfig returns default configuration
func DefaultConfig() *Config {
	return &Config{
		BufferSize:              10000,
		WorkerCount:             10,
		EnablePersistence:       true,
		EnableDeadLetter:        true,
		MaxRetries:              3,
		RetryDelay:              100 * time.Millisecond,
		CircuitBreakerThreshold: 5,
		CircuitBreakerTimeout:   30 * time.Second,
	}
}

// EnhancedBus is an enhanced event bus with reliability features
type EnhancedBus struct {
	config         *Config
	innerBus       EventBus
	store          EventStore
	deadLetter     *DeadLetterQueue
	circuitBreaker *CircuitBreaker
	metrics        *BusMetrics

	// Priority queues
	highPriority   chan *priorityEvent
	normalPriority chan *priorityEvent
	lowPriority    chan *priorityEvent

	// Worker management
	workers    sync.WaitGroup
	workerStop chan struct{}

	// State
	mu      sync.RWMutex
	running bool
	closed  bool
}

// priorityEvent wraps an event with priority information
type priorityEvent struct {
	event    Event
	priority int
	retries  int
}

// NewEnhancedBus creates a new enhanced event bus
func NewEnhancedBus(innerBus EventBus, store EventStore, config *Config) (*EnhancedBus, error) {
	if innerBus == nil {
		return nil, gerror.New(gerror.ErrCodeValidation, "inner event bus is required", nil)
	}

	if config == nil {
		config = DefaultConfig()
	}

	bus := &EnhancedBus{
		config:         config,
		innerBus:       innerBus,
		store:          store,
		highPriority:   make(chan *priorityEvent, config.BufferSize/3),
		normalPriority: make(chan *priorityEvent, config.BufferSize/3),
		lowPriority:    make(chan *priorityEvent, config.BufferSize/3),
		workerStop:     make(chan struct{}),
		metrics:        NewBusMetrics(),
	}

	if config.EnableDeadLetter {
		bus.deadLetter = NewDeadLetterQueue(config.BufferSize)
	}

	bus.circuitBreaker = NewCircuitBreaker(config.CircuitBreakerThreshold, config.CircuitBreakerTimeout)

	// Start workers
	bus.start()

	return bus, nil
}

// Publish publishes an event with normal priority
func (bus *EnhancedBus) Publish(ctx context.Context, event Event) error {
	return bus.PublishWithPriority(ctx, event, PriorityNormal)
}

// PublishWithPriority publishes an event with specified priority
func (bus *EnhancedBus) PublishWithPriority(ctx context.Context, event Event, priority int) error {
	if err := ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled")
	}

	bus.mu.RLock()
	if bus.closed {
		bus.mu.RUnlock()
		return gerror.New(gerror.ErrCodeInternal, "event bus is closed", nil)
	}
	bus.mu.RUnlock()

	// Check circuit breaker
	if !bus.circuitBreaker.Allow() {
		bus.metrics.IncrementDropped("circuit_breaker_open")
		return gerror.New(gerror.ErrCodeResourceExhausted, "circuit breaker is open", nil)
	}

	// Persist event if enabled
	if bus.config.EnablePersistence && bus.store != nil {
		if err := bus.store.Store(ctx, event); err != nil {
			// Log but don't fail - persistence is best effort
			bus.metrics.IncrementError("persistence_failed")
		}
	}

	// Wrap event with priority
	pe := &priorityEvent{
		event:    event,
		priority: priority,
		retries:  0,
	}

	// Route to appropriate queue based on priority
	select {
	case <-ctx.Done():
		return gerror.Wrap(ctx.Err(), gerror.ErrCodeCancelled, "context cancelled while publishing")

	default:
		if priority >= PriorityHigh {
			select {
			case bus.highPriority <- pe:
				bus.metrics.IncrementPublished(event.GetType())
				return nil
			default:
				// Apply backpressure
				return bus.handleBackpressure(ctx, pe)
			}
		} else if priority >= PriorityNormal {
			select {
			case bus.normalPriority <- pe:
				bus.metrics.IncrementPublished(event.GetType())
				return nil
			default:
				return bus.handleBackpressure(ctx, pe)
			}
		} else {
			select {
			case bus.lowPriority <- pe:
				bus.metrics.IncrementPublished(event.GetType())
				return nil
			default:
				return bus.handleBackpressure(ctx, pe)
			}
		}
	}
}

// Subscribe subscribes to events
func (bus *EnhancedBus) Subscribe(pattern string, handler EventHandler) (Subscription, error) {
	// Wrap handler with reliability features
	wrappedHandler := func(ctx context.Context, event Event) error {
		// Add timeout to handler execution
		handlerCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()

		// Execute handler with panic recovery
		var err error
		func() {
			defer func() {
				if r := recover(); r != nil {
					err = gerror.New(gerror.ErrCodePanic, "handler panicked", nil).
						WithDetails("panic", fmt.Sprintf("%v", r)).
						WithDetails("event_type", event.GetType())
					bus.metrics.IncrementError("handler_panic")
				}
			}()
			err = handler(handlerCtx, event)
		}()

		if err != nil {
			bus.circuitBreaker.RecordFailure()
			bus.metrics.IncrementError("handler_failed")
			return err
		}

		bus.circuitBreaker.RecordSuccess()
		return nil
	}

	// Need to pass context for the Subscribe method
	ctx := context.Background()
	return bus.innerBus.Subscribe(ctx, pattern, wrappedHandler)
}

// Close closes the event bus
func (bus *EnhancedBus) Close() error {
	bus.mu.Lock()
	if bus.closed {
		bus.mu.Unlock()
		return nil
	}
	bus.closed = true
	bus.mu.Unlock()

	// Stop workers
	close(bus.workerStop)
	bus.workers.Wait()

	// Close channels
	close(bus.highPriority)
	close(bus.normalPriority)
	close(bus.lowPriority)

	// Close inner bus
	ctx := context.Background()
	if err := bus.innerBus.Close(ctx); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to close inner event bus")
	}

	// Close store
	if bus.store != nil {
		if err := bus.store.Close(); err != nil {
			return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to close event store")
		}
	}

	return nil
}

// GetMetrics returns bus metrics
func (bus *EnhancedBus) GetMetrics() events.EventBusMetrics {
	return bus.metrics
}

// start starts worker goroutines
func (bus *EnhancedBus) start() {
	bus.mu.Lock()
	defer bus.mu.Unlock()

	if bus.running {
		return
	}
	bus.running = true

	// Start workers
	for i := 0; i < bus.config.WorkerCount; i++ {
		bus.workers.Add(1)
		go bus.worker(i)
	}

	// Start dead letter processor if enabled
	if bus.config.EnableDeadLetter {
		bus.workers.Add(1)
		go bus.deadLetterProcessor()
	}
}

// worker processes events from priority queues
func (bus *EnhancedBus) worker(id int) {
	defer bus.workers.Done()

	for {
		select {
		case <-bus.workerStop:
			return

		// Process high priority first
		case pe := <-bus.highPriority:
			bus.processEvent(pe)

		default:
			select {
			case <-bus.workerStop:
				return

			// Then normal priority
			case pe := <-bus.normalPriority:
				bus.processEvent(pe)

			default:
				select {
				case <-bus.workerStop:
					return

				// Finally low priority
				case pe := <-bus.lowPriority:
					bus.processEvent(pe)

				// No events, sleep briefly
				case <-time.After(10 * time.Millisecond):
					continue
				}
			}
		}
	}
}

// processEvent processes a single event
func (bus *EnhancedBus) processEvent(pe *priorityEvent) {
	ctx := context.Background()
	err := bus.innerBus.Publish(ctx, pe.event)

	if err != nil {
		bus.metrics.IncrementError("publish_failed")

		// Retry logic
		if pe.retries < bus.config.MaxRetries {
			pe.retries++

			// Exponential backoff
			delay := bus.config.RetryDelay * time.Duration(1<<pe.retries)
			time.Sleep(delay)

			// Re-queue based on priority
			if pe.priority >= PriorityHigh {
				select {
				case bus.highPriority <- pe:
				default:
					bus.sendToDeadLetter(pe)
				}
			} else if pe.priority >= PriorityNormal {
				select {
				case bus.normalPriority <- pe:
				default:
					bus.sendToDeadLetter(pe)
				}
			} else {
				select {
				case bus.lowPriority <- pe:
				default:
					bus.sendToDeadLetter(pe)
				}
			}
		} else {
			// Max retries exceeded, send to dead letter
			bus.sendToDeadLetter(pe)
		}
	} else {
		bus.metrics.IncrementDelivered(pe.event.GetType())
	}
}

// handleBackpressure handles backpressure scenarios
func (bus *EnhancedBus) handleBackpressure(ctx context.Context, pe *priorityEvent) error {
	bus.metrics.IncrementDropped("backpressure")

	// Try dead letter queue
	if bus.config.EnableDeadLetter {
		bus.sendToDeadLetter(pe)
		return gerror.New(gerror.ErrCodeResourceExhausted, "event queued to dead letter due to backpressure", nil)
	}

	return gerror.New(gerror.ErrCodeResourceExhausted, "event dropped due to backpressure", nil)
}

// sendToDeadLetter sends event to dead letter queue
func (bus *EnhancedBus) sendToDeadLetter(pe *priorityEvent) {
	if bus.deadLetter != nil {
		bus.deadLetter.Add(&DeadLetterEvent{
			Event:     pe.event,
			Error:     "max retries exceeded",
			Timestamp: time.Now(),
			Retries:   pe.retries,
		})
		bus.metrics.IncrementDropped("dead_letter")
	} else {
		bus.metrics.IncrementDropped("no_dead_letter")
	}
}

// deadLetterProcessor processes dead letter queue
func (bus *EnhancedBus) deadLetterProcessor() {
	defer bus.workers.Done()

	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-bus.workerStop:
			return

		case <-ticker.C:
			// Periodically retry dead letter events
			events := bus.deadLetter.GetOldest(10)
			for _, dle := range events {
				pe := &priorityEvent{
					event:    dle.Event,
					priority: PriorityLow, // Retry at low priority
					retries:  0,
				}

				select {
				case bus.lowPriority <- pe:
					bus.deadLetter.Remove(dle.Event.GetID())
				default:
					// Still congested, try again later
				}
			}
		}
	}
}

// ReplayEvents replays events from the store
func (bus *EnhancedBus) ReplayEvents(ctx context.Context, from time.Time) error {
	if bus.store == nil {
		return gerror.New(gerror.ErrCodeConfiguration, "event store not configured", nil)
	}

	return bus.store.Replay(ctx, from, func(event Event) error {
		return bus.Publish(ctx, event)
	})
}

// BusMetrics tracks event bus metrics
type BusMetrics struct {
	published map[string]*int64
	delivered map[string]*int64
	dropped   map[string]*int64
	errors    map[string]*int64
	mu        sync.RWMutex
}

// NewBusMetrics creates new metrics tracker
func NewBusMetrics() *BusMetrics {
	return &BusMetrics{
		published: make(map[string]*int64),
		delivered: make(map[string]*int64),
		dropped:   make(map[string]*int64),
		errors:    make(map[string]*int64),
	}
}

// RecordEventPublished records that an event was published
func (m *BusMetrics) RecordEventPublished(eventType string) {
	m.IncrementPublished(eventType)
}

// IncrementPublished increments published counter
func (m *BusMetrics) IncrementPublished(eventType string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.published[eventType]; !ok {
		var zero int64
		m.published[eventType] = &zero
	}
	atomic.AddInt64(m.published[eventType], 1)
}

// RecordEventDelivered records that an event was delivered
func (m *BusMetrics) RecordEventDelivered(eventType string, deliveryTime float64) {
	m.IncrementDelivered(eventType)
	// TODO: Track delivery time metrics
}

// IncrementDelivered increments delivered counter
func (m *BusMetrics) IncrementDelivered(eventType string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.delivered[eventType]; !ok {
		var zero int64
		m.delivered[eventType] = &zero
	}
	atomic.AddInt64(m.delivered[eventType], 1)
}

// RecordEventDropped records that an event was dropped
func (m *BusMetrics) RecordEventDropped(eventType string, reason string) {
	m.IncrementDropped(reason)
}

// IncrementDropped increments dropped counter
func (m *BusMetrics) IncrementDropped(reason string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.dropped[reason]; !ok {
		var zero int64
		m.dropped[reason] = &zero
	}
	atomic.AddInt64(m.dropped[reason], 1)
}

// IncrementError increments error counter
func (m *BusMetrics) IncrementError(errorType string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.errors[errorType]; !ok {
		var zero int64
		m.errors[errorType] = &zero
	}
	atomic.AddInt64(m.errors[errorType], 1)
}

// EventsPublished returns published count
func (m *BusMetrics) EventsPublished() int64 {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var total int64
	for _, count := range m.published {
		total += atomic.LoadInt64(count)
	}
	return total
}

// EventsDelivered returns delivered count
func (m *BusMetrics) EventsDelivered() int64 {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var total int64
	for _, count := range m.delivered {
		total += atomic.LoadInt64(count)
	}
	return total
}

// EventsDropped returns dropped count
func (m *BusMetrics) EventsDropped() int64 {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var total int64
	for _, count := range m.dropped {
		total += atomic.LoadInt64(count)
	}
	return total
}

// RecordSubscription records a new subscription
func (m *BusMetrics) RecordSubscription(eventType string) {
	// TODO: Track subscription metrics
}

// RecordUnsubscription records an unsubscription
func (m *BusMetrics) RecordUnsubscription(eventType string) {
	// TODO: Track unsubscription metrics
}

// GetStats returns current statistics
func (m *BusMetrics) GetStats() events.EventBusStats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return events.EventBusStats{
		EventsPublished:     m.EventsPublished(),
		EventsDelivered:     m.EventsDelivered(),
		EventsDropped:       m.EventsDropped(),
		ActiveSubscriptions: 0, // TODO: Track subscriptions
		AverageDeliveryTime: 0, // TODO: Track delivery time
	}
}

// GetDetailedStats returns detailed statistics
func (m *BusMetrics) GetDetailedStats() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	stats := make(map[string]interface{})

	// Copy published stats
	published := make(map[string]int64)
	for k, v := range m.published {
		published[k] = atomic.LoadInt64(v)
	}
	stats["published"] = published

	// Copy delivered stats
	delivered := make(map[string]int64)
	for k, v := range m.delivered {
		delivered[k] = atomic.LoadInt64(v)
	}
	stats["delivered"] = delivered

	// Copy dropped stats
	dropped := make(map[string]int64)
	for k, v := range m.dropped {
		dropped[k] = atomic.LoadInt64(v)
	}
	stats["dropped"] = dropped

	// Copy error stats
	errors := make(map[string]int64)
	for k, v := range m.errors {
		errors[k] = atomic.LoadInt64(v)
	}
	stats["errors"] = errors

	return stats
}
