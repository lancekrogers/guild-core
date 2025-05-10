package orchestrator

import (
	"encoding/json"
	"fmt"
	"sync"
)

// EventBus handles publishing and subscribing to events
type EventBus struct {
	subscribers map[string][]EventHandler
	mu          sync.RWMutex
}

// NewEventBus creates a new event bus
func NewEventBus() *EventBus {
	return &EventBus{
		subscribers: make(map[string][]EventHandler),
	}
}

// Subscribe registers a handler for a specific event type
func (b *EventBus) Subscribe(eventType string, handler EventHandler) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.subscribers[eventType] = append(b.subscribers[eventType], handler)
}

// SubscribeAll registers a handler for all event types
func (b *EventBus) SubscribeAll(handler EventHandler) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.subscribers["*"] = append(b.subscribers["*"], handler)
}

// Unsubscribe removes a handler for a specific event type
func (b *EventBus) Unsubscribe(eventType string, handler EventHandler) {
	b.mu.Lock()
	defer b.mu.Unlock()

	handlers, exists := b.subscribers[eventType]
	if !exists {
		return
	}

	var newHandlers []EventHandler
	for _, h := range handlers {
		// Compare function pointers
		if fmt.Sprintf("%p", h) != fmt.Sprintf("%p", handler) {
			newHandlers = append(newHandlers, h)
		}
	}

	b.subscribers[eventType] = newHandlers
}

// Publish sends an event to all subscribers
func (b *EventBus) Publish(event Event) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	// Handle specific event type subscribers
	if handlers, exists := b.subscribers[event.Type]; exists {
		for _, handler := range handlers {
			go handler(event)
		}
	}

	// Handle wildcard subscribers
	if handlers, exists := b.subscribers["*"]; exists {
		for _, handler := range handlers {
			go handler(event)
		}
	}
}

// PublishJSON publishes an event from a JSON string
func (b *EventBus) PublishJSON(jsonEvent string) error {
	var event Event
	if err := json.Unmarshal([]byte(jsonEvent), &event); err != nil {
		return fmt.Errorf("failed to unmarshal event JSON: %w", err)
	}

	b.Publish(event)
	return nil
}

// LoggingHandler returns an event handler that logs events
func LoggingHandler(prefix string) EventHandler {
	return func(event Event) {
		eventJSON, _ := json.MarshalIndent(event, "", "  ")
		fmt.Printf("%s Event: %s\n", prefix, string(eventJSON))
	}
}

// FilterHandler returns an event handler that filters events before passing them to another handler
func FilterHandler(filter func(event Event) bool, handler EventHandler) EventHandler {
	return func(event Event) {
		if filter(event) {
			handler(event)
		}
	}
}