// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package routing

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lancekrogers/guild/pkg/events"
	"github.com/lancekrogers/guild/pkg/gerror"
)

// mockEventBus implements a simple event bus for testing
type mockEventBus struct {
	published []events.CoreEvent
}

func (m *mockEventBus) Publish(ctx context.Context, event events.CoreEvent) error {
	m.published = append(m.published, event)
	return nil
}

func (m *mockEventBus) Subscribe(ctx context.Context, pattern string, handler events.EventHandler) (events.SubscriptionID, error) {
	return events.SubscriptionID("test"), nil
}

func (m *mockEventBus) SubscribeAll(ctx context.Context, handler events.EventHandler) (events.SubscriptionID, error) {
	return events.SubscriptionID("test"), nil
}

func (m *mockEventBus) PublishJSON(ctx context.Context, jsonEvent string) error {
	return gerror.New(gerror.ErrCodeNotImplemented, "not implemented", nil)
}

func (m *mockEventBus) Unsubscribe(ctx context.Context, subscriptionID events.SubscriptionID) error {
	return nil
}

func (m *mockEventBus) Close(ctx context.Context) error {
	return nil
}

func (m *mockEventBus) IsRunning() bool {
	return true
}

func (m *mockEventBus) GetSubscriptionCount() int {
	return 0
}

func TestRouter_AddRoute(t *testing.T) {
	bus := &mockEventBus{}
	router := NewRouter(bus)

	// Test adding valid route
	route := &Route{
		ID:      "test-route",
		Name:    "Test Route",
		Handler: func(ctx context.Context, event events.CoreEvent) error { return nil },
	}
	err := router.AddRoute(route)
	assert.NoError(t, err)

	// Test adding duplicate route
	err = router.AddRoute(route)
	assert.Error(t, err)

	// Test adding route without handler
	err = router.AddRoute(&Route{ID: "no-handler"})
	assert.Error(t, err)
}

func TestRouter_RouteWithRules(t *testing.T) {
	bus := &mockEventBus{}
	router := NewRouter(bus)

	var handledEvents []events.CoreEvent
	handler := func(ctx context.Context, event events.CoreEvent) error {
		handledEvents = append(handledEvents, event)
		return nil
	}

	// Add route with type rule
	route := &Route{
		ID:   "type-route",
		Name: "Type Route",
		Rules: []RoutingRule{
			&EventTypeRule{Types: []string{"test.event"}},
		},
		Handler: handler,
		Enabled: true,
	}
	err := router.AddRoute(route)
	require.NoError(t, err)

	ctx := context.Background()

	// Test matching event
	event1 := events.NewBaseEvent("evt1", "test.event", "test", nil)
	err = router.Route(ctx, event1)
	assert.NoError(t, err)
	assert.Len(t, handledEvents, 1)

	// Test non-matching event
	event2 := events.NewBaseEvent("evt2", "other.event", "test", nil)
	err = router.Route(ctx, event2)
	assert.NoError(t, err)
	assert.Len(t, handledEvents, 1) // Should not increase
	assert.Len(t, bus.published, 1) // Should be published to bus instead
}

func TestRouter_Transform(t *testing.T) {
	bus := &mockEventBus{}
	router := NewRouter(bus)

	var receivedEvent events.CoreEvent
	handler := func(ctx context.Context, event events.CoreEvent) error {
		receivedEvent = event
		return nil
	}

	// Add route with transform
	route := &Route{
		ID:   "transform-route",
		Name: "Transform Route",
		Transform: func(ctx context.Context, event events.CoreEvent) (events.CoreEvent, error) {
			// Add enrichment
			data := event.GetData()
			data["enriched"] = true
			return events.NewBaseEvent(
				event.GetID(),
				event.GetType(),
				event.GetSource(),
				data,
			), nil
		},
		Handler: handler,
		Enabled: true,
	}
	err := router.AddRoute(route)
	require.NoError(t, err)

	// Route event
	ctx := context.Background()
	event := events.NewBaseEvent("evt1", "test.event", "test", map[string]interface{}{
		"original": "data",
	})
	err = router.Route(ctx, event)
	assert.NoError(t, err)

	// Check transformation was applied
	assert.NotNil(t, receivedEvent)
	data := receivedEvent.GetData()
	assert.Equal(t, "data", data["original"])
	assert.Equal(t, true, data["enriched"])
}

func TestRouter_Priority(t *testing.T) {
	bus := &mockEventBus{}
	router := NewRouter(bus)

	var executionOrder []string

	// Add routes with different priorities
	routes := []*Route{
		{
			ID:       "low",
			Priority: 10,
			Handler: func(ctx context.Context, event events.CoreEvent) error {
				executionOrder = append(executionOrder, "low")
				return nil
			},
			Enabled: true,
		},
		{
			ID:       "high",
			Priority: 100,
			Handler: func(ctx context.Context, event events.CoreEvent) error {
				executionOrder = append(executionOrder, "high")
				return nil
			},
			Enabled: true,
		},
		{
			ID:       "medium",
			Priority: 50,
			Handler: func(ctx context.Context, event events.CoreEvent) error {
				executionOrder = append(executionOrder, "medium")
				return nil
			},
			Enabled: true,
		},
	}

	for _, route := range routes {
		err := router.AddRoute(route)
		require.NoError(t, err)
	}

	// Route event
	ctx := context.Background()
	event := events.NewBaseEvent("evt1", "test.event", "test", nil)
	err := router.Route(ctx, event)
	assert.NoError(t, err)

	// Check execution order
	assert.Equal(t, []string{"high", "medium", "low"}, executionOrder)
}

func TestRouter_Middleware(t *testing.T) {
	bus := &mockEventBus{}
	router := NewRouter(bus)

	var middlewareExecuted bool
	middleware := func(next events.EventHandler) events.EventHandler {
		return func(ctx context.Context, event events.CoreEvent) error {
			middlewareExecuted = true
			return next(ctx, event)
		}
	}

	router.UseMiddleware(middleware)

	// Add route
	route := &Route{
		ID:      "test",
		Handler: func(ctx context.Context, event events.CoreEvent) error { return nil },
		Enabled: true,
	}
	err := router.AddRoute(route)
	require.NoError(t, err)

	// Route event
	ctx := context.Background()
	event := events.NewBaseEvent("evt1", "test.event", "test", nil)
	err = router.Route(ctx, event)
	assert.NoError(t, err)
	assert.True(t, middlewareExecuted)
}

func TestEventTypeRule(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name     string
		rule     *EventTypeRule
		event    events.CoreEvent
		expected bool
	}{
		{
			name:     "Exact match",
			rule:     &EventTypeRule{Types: []string{"test.event"}},
			event:    events.NewBaseEvent("id", "test.event", "src", nil),
			expected: true,
		},
		{
			name:     "No match",
			rule:     &EventTypeRule{Types: []string{"test.event"}},
			event:    events.NewBaseEvent("id", "other.event", "src", nil),
			expected: false,
		},
		{
			name:     "Pattern match",
			rule:     &EventTypeRule{Patterns: []string{"test.*"}},
			event:    events.NewBaseEvent("id", "test.event", "src", nil),
			expected: true,
		},
		{
			name:     "Multiple types",
			rule:     &EventTypeRule{Types: []string{"event1", "event2", "event3"}},
			event:    events.NewBaseEvent("id", "event2", "src", nil),
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matches, err := tt.rule.Matches(ctx, tt.event)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, matches)
		})
	}
}

func TestDataRule(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name     string
		rule     *DataRule
		data     map[string]interface{}
		expected bool
	}{
		{
			name:     "Equal match",
			rule:     &DataRule{FieldName: "status", Operator: "eq", Value: "active"},
			data:     map[string]interface{}{"status": "active"},
			expected: true,
		},
		{
			name:     "Not equal",
			rule:     &DataRule{FieldName: "status", Operator: "ne", Value: "active"},
			data:     map[string]interface{}{"status": "inactive"},
			expected: true,
		},
		{
			name:     "Greater than",
			rule:     &DataRule{FieldName: "count", Operator: "gt", Value: 10},
			data:     map[string]interface{}{"count": 15},
			expected: true,
		},
		{
			name:     "Contains",
			rule:     &DataRule{FieldName: "message", Operator: "contains", Value: "error"},
			data:     map[string]interface{}{"message": "an error occurred"},
			expected: true,
		},
		{
			name:     "Nested field",
			rule:     &DataRule{FieldName: "user.name", Operator: "eq", Value: "John"},
			data:     map[string]interface{}{"user": map[string]interface{}{"name": "John"}},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event := events.NewBaseEvent("id", "test", "src", tt.data)
			matches, err := tt.rule.Matches(ctx, event)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, matches)
		})
	}
}

func TestRateLimitingMiddleware(t *testing.T) {
	middleware := RateLimitingMiddleware(2, 2) // 2 per second, burst of 2

	var count int32
	handler := middleware(func(ctx context.Context, event events.CoreEvent) error {
		atomic.AddInt32(&count, 1)
		return nil
	})

	ctx := context.Background()
	event := events.NewBaseEvent("id", "test", "src", nil)

	// First two should succeed (burst)
	assert.NoError(t, handler(ctx, event))
	assert.NoError(t, handler(ctx, event))

	// Third should fail
	err := handler(ctx, event)
	assert.Error(t, err)
	assert.Equal(t, int32(2), atomic.LoadInt32(&count))

	// Wait for refill
	time.Sleep(550 * time.Millisecond)

	// Should succeed again
	assert.NoError(t, handler(ctx, event))
	assert.Equal(t, int32(3), atomic.LoadInt32(&count))
}

func TestRetryMiddleware(t *testing.T) {
	middleware := RetryMiddleware(2, 10*time.Millisecond)

	attempts := 0
	handler := middleware(func(ctx context.Context, event events.CoreEvent) error {
		attempts++
		if attempts < 3 {
			return gerror.New(gerror.ErrCodeInternal, "temporary error", nil)
		}
		return nil
	})

	ctx := context.Background()
	event := events.NewBaseEvent("id", "test", "src", nil)

	err := handler(ctx, event)
	assert.NoError(t, err)
	assert.Equal(t, 3, attempts)
}

func TestDedupingMiddleware(t *testing.T) {
	middleware := DedupingMiddleware(100 * time.Millisecond)

	var count int32
	handler := middleware(func(ctx context.Context, event events.CoreEvent) error {
		atomic.AddInt32(&count, 1)
		return nil
	})

	ctx := context.Background()

	// Same event ID
	event1 := events.NewBaseEvent("evt1", "test", "src", nil)
	event2 := events.NewBaseEvent("evt1", "test", "src", nil) // Duplicate
	event3 := events.NewBaseEvent("evt2", "test", "src", nil) // Different

	assert.NoError(t, handler(ctx, event1))
	assert.NoError(t, handler(ctx, event2)) // Should be skipped
	assert.NoError(t, handler(ctx, event3))

	assert.Equal(t, int32(2), atomic.LoadInt32(&count))

	// Wait for window to pass
	time.Sleep(150 * time.Millisecond)

	// Same ID should work again
	assert.NoError(t, handler(ctx, event1))
	assert.Equal(t, int32(3), atomic.LoadInt32(&count))
}
