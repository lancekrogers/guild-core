// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package events

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/lancekrogers/guild/pkg/gerror"
)

// MemoryEventBus provides an in-memory implementation of EventBus
type MemoryEventBus struct {
	config        EventBusConfig
	subscriptions map[SubscriptionID]*FilteredSubscription
	handlers      map[string][]SubscriptionID
	allHandlers   []SubscriptionID
	
	// Metrics
	stats EventBusStats
	
	// State management
	running    atomic.Bool
	closed     atomic.Bool
	mu         sync.RWMutex
	nextSubID  atomic.Int64
	
	// Event processing
	eventChan chan eventDelivery
	stopChan  chan struct{}
	wg        sync.WaitGroup
}

// eventDelivery represents an event ready for delivery
type eventDelivery struct {
	event CoreEvent
	ctx   context.Context
}

// NewMemoryEventBus creates a new in-memory event bus
func NewMemoryEventBus(config EventBusConfig) *MemoryEventBus {
	bus := &MemoryEventBus{
		config:        config,
		subscriptions: make(map[SubscriptionID]*FilteredSubscription),
		handlers:      make(map[string][]SubscriptionID),
		eventChan:     make(chan eventDelivery, config.BufferSize),
		stopChan:      make(chan struct{}),
	}
	
	bus.start()
	return bus
}

// NewMemoryEventBusWithDefaults creates a new event bus with default configuration
func NewMemoryEventBusWithDefaults() *MemoryEventBus {
	return NewMemoryEventBus(DefaultEventBusConfig())
}

// start begins event processing
func (b *MemoryEventBus) start() {
	if !b.running.CompareAndSwap(false, true) {
		return // Already running
	}
	
	b.wg.Add(1)
	go b.processEvents()
}

// processEvents handles event delivery in a separate goroutine
func (b *MemoryEventBus) processEvents() {
	defer b.wg.Done()
	
	for {
		select {
		case delivery := <-b.eventChan:
			b.deliverEvent(delivery.ctx, delivery.event)
		case <-b.stopChan:
			return
		}
	}
}

// deliverEvent delivers an event to all matching subscribers
func (b *MemoryEventBus) deliverEvent(ctx context.Context, event CoreEvent) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	
	startTime := time.Now()
	eventType := event.GetType()
	deliveredCount := 0
	
	// Deliver to specific type handlers
	if handlers, exists := b.handlers[eventType]; exists {
		for _, subID := range handlers {
			if sub, exists := b.subscriptions[subID]; exists {
				if b.deliverToHandler(ctx, event, sub) {
					deliveredCount++
				}
			}
		}
	}
	
	// Deliver to "all events" handlers
	for _, subID := range b.allHandlers {
		if sub, exists := b.subscriptions[subID]; exists {
			if b.deliverToHandler(ctx, event, sub) {
				deliveredCount++
			}
		}
	}
	
	// Update metrics
	deliveryTime := float64(time.Since(startTime).Nanoseconds()) / 1e6 // Convert to milliseconds
	atomic.AddInt64(&b.stats.EventsDelivered, int64(deliveredCount))
	
	// Update average delivery time (simple moving average)
	if b.stats.EventsDelivered > 0 {
		b.stats.AverageDeliveryTime = (b.stats.AverageDeliveryTime*float64(b.stats.EventsDelivered-int64(deliveredCount)) + deliveryTime*float64(deliveredCount)) / float64(b.stats.EventsDelivered)
	} else {
		b.stats.AverageDeliveryTime = deliveryTime
	}
}

// deliverToHandler delivers an event to a specific handler
func (b *MemoryEventBus) deliverToHandler(ctx context.Context, event CoreEvent, sub *FilteredSubscription) bool {
	// Apply filter if present
	if sub.Filter != nil && !sub.Filter(event) {
		return false
	}
	
	// Deliver to handler with panic recovery
	defer func() {
		if r := recover(); r != nil {
			atomic.AddInt64(&b.stats.EventsDropped, 1)
			// Log the panic but don't stop processing
			if b.config.LogEvents {
				fmt.Printf("Event handler panic for subscription %s: %v\n", sub.ID, r)
			}
		}
	}()
	
	err := sub.Handler(ctx, event)
	if err != nil {
		atomic.AddInt64(&b.stats.EventsDropped, 1)
		if b.config.LogEvents {
			fmt.Printf("Event handler error for subscription %s: %v\n", sub.ID, err)
		}
		return false
	}
	
	return true
}

// Publish publishes an event to all subscribers
func (b *MemoryEventBus) Publish(ctx context.Context, event CoreEvent) error {
	if b.closed.Load() {
		return ErrEventBusClosed
	}
	
	if event == nil {
		return gerror.New(gerror.ErrCodeValidation, "event cannot be nil", nil).
			WithComponent("MemoryEventBus").
			WithOperation("Publish")
	}
	
	// Validate event
	if baseEvent, ok := event.(*BaseEvent); ok {
		if err := baseEvent.Validate(); err != nil {
			return gerror.Wrap(err, gerror.ErrCodeValidation, "invalid event").
				WithComponent("MemoryEventBus").
				WithOperation("Publish")
		}
	}
	
	// Check event size if configured
	if b.config.MaxEventSize > 0 {
		if eventJSON, err := ToJSON(event); err == nil {
			if len(eventJSON) > b.config.MaxEventSize {
				return ErrEventTooLarge
			}
		}
	}
	
	atomic.AddInt64(&b.stats.EventsPublished, 1)
	
	// Log event if configured
	if b.config.LogEvents {
		fmt.Printf("Publishing event: %s (type: %s, source: %s)\n", 
			event.GetID(), event.GetType(), event.GetSource())
	}
	
	// Queue for delivery
	select {
	case b.eventChan <- eventDelivery{event: event, ctx: ctx}:
		return nil
	case <-ctx.Done():
		return gerror.Wrap(ctx.Err(), gerror.ErrCodeCancelled, "context cancelled while publishing").
			WithComponent("MemoryEventBus").
			WithOperation("Publish")
	default:
		atomic.AddInt64(&b.stats.EventsDropped, 1)
		return gerror.New(gerror.ErrCodeResourceExhausted, "event buffer full", nil).
			WithComponent("MemoryEventBus").
			WithOperation("Publish")
	}
}

// Subscribe subscribes to events of a specific type
func (b *MemoryEventBus) Subscribe(ctx context.Context, eventType string, handler EventHandler) (SubscriptionID, error) {
	if b.closed.Load() {
		return "", ErrEventBusClosed
	}
	
	if handler == nil {
		return "", ErrInvalidHandler
	}
	
	b.mu.Lock()
	defer b.mu.Unlock()
	
	// Check subscription limit
	if b.config.MaxSubscriptions > 0 && len(b.subscriptions) >= b.config.MaxSubscriptions {
		return "", ErrTooManySubscriptions
	}
	
	// Generate subscription ID
	subID := SubscriptionID(fmt.Sprintf("sub_%d", b.nextSubID.Add(1)))
	
	// Create subscription
	sub := &FilteredSubscription{
		ID:        subID,
		EventType: eventType,
		Handler:   handler,
	}
	
	b.subscriptions[subID] = sub
	
	// Add to appropriate handler list
	if eventType == "" {
		b.allHandlers = append(b.allHandlers, subID)
	} else {
		b.handlers[eventType] = append(b.handlers[eventType], subID)
	}
	
	b.stats.ActiveSubscriptions = len(b.subscriptions)
	
	return subID, nil
}

// SubscribeAll subscribes to all events
func (b *MemoryEventBus) SubscribeAll(ctx context.Context, handler EventHandler) (SubscriptionID, error) {
	return b.Subscribe(ctx, "", handler)
}

// SubscribeWithFilter subscribes to events with a custom filter
func (b *MemoryEventBus) SubscribeWithFilter(ctx context.Context, eventType string, handler EventHandler, filter EventFilter) (SubscriptionID, error) {
	subID, err := b.Subscribe(ctx, eventType, handler)
	if err != nil {
		return "", err
	}
	
	// Add filter to subscription
	b.mu.Lock()
	if sub, exists := b.subscriptions[subID]; exists {
		sub.Filter = filter
	}
	b.mu.Unlock()
	
	return subID, nil
}

// Unsubscribe removes a subscription
func (b *MemoryEventBus) Unsubscribe(ctx context.Context, subscriptionID SubscriptionID) error {
	if b.closed.Load() {
		return ErrEventBusClosed
	}
	
	b.mu.Lock()
	defer b.mu.Unlock()
	
	sub, exists := b.subscriptions[subscriptionID]
	if !exists {
		return ErrSubscriptionNotFound
	}
	
	// Remove from subscriptions map
	delete(b.subscriptions, subscriptionID)
	
	// Remove from appropriate handler list
	if sub.EventType == "" {
		b.allHandlers = b.removeSubscriptionID(b.allHandlers, subscriptionID)
	} else {
		if handlers, exists := b.handlers[sub.EventType]; exists {
			b.handlers[sub.EventType] = b.removeSubscriptionID(handlers, subscriptionID)
			if len(b.handlers[sub.EventType]) == 0 {
				delete(b.handlers, sub.EventType)
			}
		}
	}
	
	b.stats.ActiveSubscriptions = len(b.subscriptions)
	
	return nil
}

// removeSubscriptionID removes a subscription ID from a slice
func (b *MemoryEventBus) removeSubscriptionID(slice []SubscriptionID, target SubscriptionID) []SubscriptionID {
	for i, id := range slice {
		if id == target {
			return append(slice[:i], slice[i+1:]...)
		}
	}
	return slice
}

// PublishJSON publishes an event from a JSON string
func (b *MemoryEventBus) PublishJSON(ctx context.Context, jsonEvent string) error {
	event, err := FromJSON(jsonEvent)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeValidation, "invalid JSON event").
			WithComponent("MemoryEventBus").
			WithOperation("PublishJSON")
	}
	
	return b.Publish(ctx, event)
}

// Close shuts down the event bus
func (b *MemoryEventBus) Close(ctx context.Context) error {
	if !b.closed.CompareAndSwap(false, true) {
		return nil // Already closed
	}
	
	if b.running.Load() {
		close(b.stopChan)
		b.running.Store(false)
	}
	
	// Wait for event processing to complete
	done := make(chan struct{})
	go func() {
		b.wg.Wait()
		close(done)
	}()
	
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return gerror.Wrap(ctx.Err(), gerror.ErrCodeTimeout, "timeout waiting for event bus to close").
			WithComponent("MemoryEventBus").
			WithOperation("Close")
	}
}

// IsRunning returns true if the event bus is running
func (b *MemoryEventBus) IsRunning() bool {
	return b.running.Load() && !b.closed.Load()
}

// GetSubscriptionCount returns the current number of active subscriptions
func (b *MemoryEventBus) GetSubscriptionCount() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.subscriptions)
}

// GetStats returns current event bus statistics
func (b *MemoryEventBus) GetStats() EventBusStats {
	return EventBusStats{
		EventsPublished:     atomic.LoadInt64(&b.stats.EventsPublished),
		EventsDelivered:     atomic.LoadInt64(&b.stats.EventsDelivered),
		EventsDropped:       atomic.LoadInt64(&b.stats.EventsDropped),
		ActiveSubscriptions: b.GetSubscriptionCount(),
		AverageDeliveryTime: b.stats.AverageDeliveryTime,
	}
}

// GetSubscriptions returns a copy of current subscriptions (for debugging)
func (b *MemoryEventBus) GetSubscriptions() map[SubscriptionID]string {
	b.mu.RLock()
	defer b.mu.RUnlock()
	
	result := make(map[SubscriptionID]string)
	for id, sub := range b.subscriptions {
		eventType := sub.EventType
		if eventType == "" {
			eventType = "*" // All events
		}
		result[id] = eventType
	}
	return result
}