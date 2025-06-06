package orchestrator

import (
	"encoding/json"
	"fmt"
	"sync"

	"github.com/guild-ventures/guild-core/pkg/gerror"
	"github.com/guild-ventures/guild-core/pkg/orchestrator/interfaces"
)

// Re-export event types
type EventHandler = interfaces.EventHandler
type Event = interfaces.Event
type EventType = interfaces.EventType

// eventBus handles publishing and subscribing to events
type eventBus struct {
	subscribers map[string][]EventHandler
	mu          sync.RWMutex
}

// newEventBus creates a new event bus (private constructor)
func newEventBus() *eventBus {
	return &eventBus{
		subscribers: make(map[string][]EventHandler),
	}
}

// DefaultEventBusFactory creates a default event bus instance for registry
func DefaultEventBusFactory() EventBus {
	return newEventBus()
}

// Subscribe registers a handler for a specific event type
func (b *eventBus) Subscribe(eventType EventType, handler EventHandler) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.subscribers[string(eventType)] = append(b.subscribers[string(eventType)], handler)
}

// SubscribeAll registers a handler for all event types
func (b *eventBus) SubscribeAll(handler EventHandler) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.subscribers["*"] = append(b.subscribers["*"], handler)
}

// Unsubscribe removes a handler for a specific event type
func (b *eventBus) Unsubscribe(eventType EventType, handler EventHandler) {
	b.mu.Lock()
	defer b.mu.Unlock()

	typeStr := string(eventType)
	handlers, exists := b.subscribers[typeStr]
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

	b.subscribers[typeStr] = newHandlers
}

// Publish sends an event to all subscribers
func (b *eventBus) Publish(event Event) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	// Handle specific event type subscribers
	typeStr := string(event.Type)
	if handlers, exists := b.subscribers[typeStr]; exists {
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
func (b *eventBus) PublishJSON(jsonEvent string) error {
	var event Event
	if err := json.Unmarshal([]byte(jsonEvent), &event); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeValidation, "failed to unmarshal event JSON").
			WithComponent("orchestrator").
			WithOperation("PublishJSON")
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