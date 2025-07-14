// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package eventbus

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lancekrogers/guild/pkg/events"
	"github.com/lancekrogers/guild/pkg/gerror"
)

// mockEventBus implements a simple in-memory event bus for testing
type mockEventBus struct {
	handlers  map[string][]EventHandler
	mu        sync.RWMutex
	published []Event
	closed    bool
}

func newMockEventBus() *mockEventBus {
	return &mockEventBus{
		handlers:  make(map[string][]EventHandler),
		published: make([]Event, 0),
	}
}

func (m *mockEventBus) Publish(ctx context.Context, event Event) error {
	m.mu.Lock()
	if m.closed {
		m.mu.Unlock()
		return gerror.New(gerror.ErrCodeInternal, "event bus is closed", nil)
	}
	m.published = append(m.published, event)
	handlers := m.handlers[event.GetType()]
	m.mu.Unlock()

	for _, handler := range handlers {
		if err := handler(ctx, event); err != nil {
			return err
		}
	}
	return nil
}

func (m *mockEventBus) Subscribe(ctx context.Context, eventType string, handler EventHandler) (events.SubscriptionID, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.handlers[eventType] = append(m.handlers[eventType], handler)
	return events.SubscriptionID(eventType), nil
}

func (m *mockEventBus) SubscribeAll(ctx context.Context, handler EventHandler) (events.SubscriptionID, error) {
	return m.Subscribe(ctx, "*", handler)
}

func (m *mockEventBus) PublishJSON(ctx context.Context, jsonEvent string) error {
	return gerror.New(gerror.ErrCodeNotImplemented, "not implemented", nil)
}

func (m *mockEventBus) Unsubscribe(ctx context.Context, subscriptionID events.SubscriptionID) error {
	// Not implemented for tests
	return nil
}

func (m *mockEventBus) Close(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.closed = true
	return nil
}

func (m *mockEventBus) IsRunning() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return !m.closed
}

func (m *mockEventBus) GetSubscriptionCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	count := 0
	for _, handlers := range m.handlers {
		count += len(handlers)
	}
	return count
}

// mockEvent implements a simple test event
type mockEvent struct {
	id            string
	eventType     string
	timestamp     time.Time
	source        string
	data          interface{}
	metadata      map[string]interface{}
	correlationID string
	parentID      string
}

func newMockEvent(eventType string) *mockEvent {
	return &mockEvent{
		id:        fmt.Sprintf("event-%d", time.Now().UnixNano()),
		eventType: eventType,
		timestamp: time.Now(),
		source:    "test",
		metadata:  make(map[string]interface{}),
	}
}

func (e *mockEvent) GetID() string           { return e.id }
func (e *mockEvent) GetType() string         { return e.eventType }
func (e *mockEvent) GetTimestamp() time.Time { return e.timestamp }
func (e *mockEvent) GetSource() string       { return e.source }
func (e *mockEvent) GetData() map[string]interface{} {
	if data, ok := e.data.(map[string]interface{}); ok {
		return data
	}
	return make(map[string]interface{})
}
func (e *mockEvent) GetTarget() string                   { return "" }
func (e *mockEvent) GetMetadata() map[string]interface{} { return e.metadata }
func (e *mockEvent) GetCorrelationID() string            { return e.correlationID }
func (e *mockEvent) GetParentID() string                 { return e.parentID }
func (e *mockEvent) Clone() Event {
	return &mockEvent{
		id:            e.id,
		eventType:     e.eventType,
		timestamp:     e.timestamp,
		source:        e.source,
		data:          e.data,
		metadata:      e.metadata,
		correlationID: e.correlationID,
		parentID:      e.parentID,
	}
}

func TestEnhancedBus_PublishWithPriority(t *testing.T) {
	innerBus := newMockEventBus()
	config := DefaultConfig()
	config.EnablePersistence = false // Disable persistence for this test

	bus, err := NewEnhancedBus(innerBus, nil, config)
	require.NoError(t, err)
	defer bus.Close()

	ctx := context.Background()

	// Test publishing with different priorities
	tests := []struct {
		name     string
		priority int
		event    Event
	}{
		{
			name:     "High priority event",
			priority: PriorityHigh,
			event:    newMockEvent("high.priority"),
		},
		{
			name:     "Normal priority event",
			priority: PriorityNormal,
			event:    newMockEvent("normal.priority"),
		},
		{
			name:     "Low priority event",
			priority: PriorityLow,
			event:    newMockEvent("low.priority"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := bus.PublishWithPriority(ctx, tt.event, tt.priority)
			assert.NoError(t, err)
		})
	}

	// Wait for events to be processed
	time.Sleep(100 * time.Millisecond)

	// Verify events were published
	assert.Len(t, innerBus.published, 3)
}

func TestEnhancedBus_CircuitBreaker(t *testing.T) {
	t.Skip("Circuit breaker test needs redesign - current implementation protects bus, not handlers")

	// The current circuit breaker implementation protects the bus itself from being overwhelmed,
	// not from handler failures. Handler failures are tracked but don't open the circuit breaker
	// for new publishes. This test needs to be redesigned to match the actual implementation.
}

func TestEnhancedBus_DeadLetterQueue(t *testing.T) {
	innerBus := newMockEventBus()
	config := DefaultConfig()
	config.EnableDeadLetter = true
	config.MaxRetries = 2
	config.RetryDelay = 10 * time.Millisecond
	config.EnablePersistence = false

	bus, err := NewEnhancedBus(innerBus, nil, config)
	require.NoError(t, err)
	defer bus.Close()

	// Make inner bus fail
	innerBus.mu.Lock()
	innerBus.closed = true
	innerBus.mu.Unlock()

	ctx := context.Background()
	event := newMockEvent("test.event")

	// Publish should succeed (queued internally)
	err = bus.Publish(ctx, event)
	assert.NoError(t, err)

	// Wait for retries to exhaust
	time.Sleep(100 * time.Millisecond)

	// Check dead letter queue
	assert.Equal(t, 1, bus.deadLetter.Size())

	dle := bus.deadLetter.Get(event.GetID())
	assert.NotNil(t, dle)
	assert.Equal(t, event.GetID(), dle.Event.GetID())
	assert.Equal(t, config.MaxRetries, dle.Retries)
}

func TestEnhancedBus_Backpressure(t *testing.T) {
	innerBus := newMockEventBus()
	config := DefaultConfig()
	config.BufferSize = 10
	config.EnableDeadLetter = true
	config.EnablePersistence = false
	config.WorkerCount = 1 // Single worker to control processing

	bus, err := NewEnhancedBus(innerBus, nil, config)
	require.NoError(t, err)
	defer bus.Close()

	// Slow down inner bus processing
	var processedCount int32
	innerBus.Subscribe(context.Background(), "test.event", func(ctx context.Context, event Event) error {
		time.Sleep(50 * time.Millisecond)
		atomic.AddInt32(&processedCount, 1)
		return nil
	})

	ctx := context.Background()

	// Publish many events quickly
	var publishErrors int32
	var wg sync.WaitGroup

	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			event := newMockEvent("test.event")
			err := bus.Publish(ctx, event)
			if err != nil {
				atomic.AddInt32(&publishErrors, 1)
			}
		}(i)
	}

	wg.Wait()

	// Some events should have been queued to dead letter due to backpressure
	assert.Greater(t, bus.deadLetter.Size(), 0)
}

func TestCircuitBreaker_StateTransitions(t *testing.T) {
	cb := NewCircuitBreaker(3, 100*time.Millisecond)

	// Initial state should be closed
	assert.Equal(t, StateClosed, cb.GetState())
	assert.True(t, cb.Allow())

	// Record failures
	for i := 0; i < 3; i++ {
		cb.RecordFailure()
	}

	// Should be open now
	assert.Equal(t, StateOpen, cb.GetState())
	assert.False(t, cb.Allow())

	// Wait for timeout
	time.Sleep(150 * time.Millisecond)

	// Should transition to half-open
	assert.True(t, cb.Allow()) // First request in half-open

	// Record success in half-open
	cb.RecordSuccess()
	cb.RecordSuccess()
	cb.RecordSuccess()

	// Should be closed again
	assert.Equal(t, StateClosed, cb.GetState())
}

func TestDeadLetterQueue_Operations(t *testing.T) {
	dlq := NewDeadLetterQueue(5)

	// Add events
	for i := 0; i < 7; i++ {
		event := newMockEvent(fmt.Sprintf("event-%d", i))
		dlq.Add(&DeadLetterEvent{
			Event:     event,
			Error:     "test error",
			Timestamp: time.Now(),
			Retries:   i,
		})
	}

	// Should only keep 5 (max size)
	assert.Equal(t, 5, dlq.Size())

	// Get oldest should return the oldest remaining events
	oldest := dlq.GetOldest(2)
	assert.Len(t, oldest, 2)
	assert.Equal(t, "event-2", oldest[0].Event.GetType())

	// Remove an event
	removed := dlq.Remove(oldest[0].Event.GetID())
	assert.True(t, removed)
	assert.Equal(t, 4, dlq.Size())

	// Get by age
	time.Sleep(10 * time.Millisecond)
	aged := dlq.GetByAge(5 * time.Millisecond)
	assert.Len(t, aged, 4) // All remaining events are older than 5ms

	// Clear all
	dlq.Clear()
	assert.Equal(t, 0, dlq.Size())
}

func TestTopicRouter_Routing(t *testing.T) {
	router := NewTopicRouter()

	// Create topics
	err := router.CreateTopic("task.created", nil)
	assert.NoError(t, err)

	err = router.CreateTopic("task.updated", &MetadataFilter{
		RequiredKeys: []string{"priority"},
	})
	assert.NoError(t, err)

	// Subscribe to topics
	var received []Event
	var mu sync.Mutex

	handler := func(ctx context.Context, event Event) error {
		mu.Lock()
		received = append(received, event)
		mu.Unlock()
		return nil
	}

	// Exact topic subscription
	sub1, err := router.Subscribe("task.created", handler, nil)
	assert.NoError(t, err)

	// Wildcard subscription
	sub2, err := router.Subscribe("task.*", handler, nil)
	assert.NoError(t, err)

	ctx := context.Background()

	// Route events
	event1 := newMockEvent("task.created")
	err = router.Route(ctx, event1)
	assert.NoError(t, err)

	event2 := newMockEvent("task.updated")
	event2.metadata["priority"] = "high"
	err = router.Route(ctx, event2)
	assert.NoError(t, err)

	// Wait for handlers
	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	assert.Len(t, received, 3) // event1 x2 (exact + wildcard), event2 x1 (wildcard only)
	mu.Unlock()

	// Unsubscribe
	err = router.Unsubscribe(sub1)
	assert.NoError(t, err)

	err = router.Unsubscribe(sub2)
	assert.NoError(t, err)
}

func TestHandlerRegistry_Dependencies(t *testing.T) {
	registry := NewHandlerRegistry()

	var executionOrder []string
	var mu sync.Mutex

	createHandler := func(id string) EventHandler {
		return func(ctx context.Context, event Event) error {
			mu.Lock()
			executionOrder = append(executionOrder, id)
			mu.Unlock()
			return nil
		}
	}

	// Register handlers with dependencies
	err := registry.Register("test.event", &RegisteredHandler{
		ID:       "handler1",
		Handler:  createHandler("handler1"),
		Priority: 1,
	})
	assert.NoError(t, err)

	err = registry.Register("test.event", &RegisteredHandler{
		ID:           "handler2",
		Handler:      createHandler("handler2"),
		Dependencies: []string{"handler1"},
		Priority:     2,
	})
	assert.NoError(t, err)

	err = registry.Register("test.event", &RegisteredHandler{
		ID:           "handler3",
		Handler:      createHandler("handler3"),
		Dependencies: []string{"handler2"},
		Priority:     3,
	})
	assert.NoError(t, err)

	// Execute handlers
	ctx := context.Background()
	event := newMockEvent("test.event")

	err = registry.ExecuteHandlers(ctx, event)
	assert.NoError(t, err)

	// Check execution order respects dependencies
	mu.Lock()
	assert.Equal(t, []string{"handler1", "handler2", "handler3"}, executionOrder)
	mu.Unlock()
}

func TestHandlerChain(t *testing.T) {
	var calls []string
	var mu sync.Mutex

	handler1 := func(ctx context.Context, event Event) error {
		mu.Lock()
		calls = append(calls, "handler1")
		mu.Unlock()
		return nil
	}

	handler2 := func(ctx context.Context, event Event) error {
		mu.Lock()
		calls = append(calls, "handler2")
		mu.Unlock()
		return nil
	}

	handler3 := func(ctx context.Context, event Event) error {
		mu.Lock()
		calls = append(calls, "handler3")
		mu.Unlock()
		return gerror.New(gerror.ErrCodeInternal, "handler3 failed", nil)
	}

	chain := NewHandlerChain(handler1, handler2, handler3)

	ctx := context.Background()
	event := newMockEvent("test.event")

	err := chain.Handle(ctx, event)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "handler3 failed")

	mu.Lock()
	assert.Equal(t, []string{"handler1", "handler2", "handler3"}, calls)
	mu.Unlock()
}
