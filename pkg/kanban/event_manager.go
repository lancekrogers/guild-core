package kanban

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/guild-ventures/guild-core/pkg/comms"
)

// EventManager handles kanban event publishing and subscription
type EventManager struct {
	pubsub      comms.PubSub
	handlers    map[EventType][]EventHandler
	mu          sync.RWMutex
	ctx         context.Context
	cancel      context.CancelFunc
	topicPrefix string
}

// EventHandler is a callback for handling events
type EventHandler func(event *BoardEvent) error

// NewEventManager creates a new event manager
func NewEventManager(ctx context.Context, pubsub comms.PubSub, topicPrefix string) *EventManager {
	ctx, cancel := context.WithCancel(ctx)

	em := &EventManager{
		pubsub:      pubsub,
		handlers:    make(map[EventType][]EventHandler),
		ctx:         ctx,
		cancel:      cancel,
		topicPrefix: topicPrefix,
	}

	// Start event receiver
	go em.receiveEvents()

	return em
}

// PublishEvent publishes an event
func (em *EventManager) PublishEvent(event *BoardEvent) error {
	data, err := MarshalEvent(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	topic := em.topicPrefix + string(event.EventType)
	return em.pubsub.Publish(em.ctx, topic, data)
}

// Subscribe adds a handler for a specific event type
func (em *EventManager) Subscribe(eventType EventType, handler EventHandler) error {
	em.mu.Lock()
	defer em.mu.Unlock()

	// Subscribe to the topic if this is the first handler
	if len(em.handlers[eventType]) == 0 {
		topic := em.topicPrefix + string(eventType)
		if err := em.pubsub.Subscribe(em.ctx, topic); err != nil {
			return fmt.Errorf("failed to subscribe to topic %s: %w", topic, err)
		}
	}

	em.handlers[eventType] = append(em.handlers[eventType], handler)
	return nil
}

// SubscribeAll adds a handler for all event types
func (em *EventManager) SubscribeAll(handler EventHandler) error {
	em.mu.Lock()
	defer em.mu.Unlock()

	// Subscribe to all kanban events
	if err := em.pubsub.Subscribe(em.ctx, em.topicPrefix); err != nil {
		return fmt.Errorf("failed to subscribe to all events: %w", err)
	}

	// Add handler to all event types
	for _, eventType := range []EventType{
		EventTaskCreated,
		EventTaskMoved,
		EventTaskUpdated,
		EventTaskDeleted,
		EventTaskBlocked,
		EventTaskUnblocked,
		EventTaskStatusChanged,
		EventTaskAssigned,
	} {
		em.handlers[eventType] = append(em.handlers[eventType], handler)
	}

	return nil
}

// Close shuts down the event manager
func (em *EventManager) Close() error {
	em.cancel()
	return nil
}

// receiveEvents processes incoming events
func (em *EventManager) receiveEvents() {
	for {
		select {
		case <-em.ctx.Done():
			return
		default:
			// Continue
		}

		// Receive next message
		msg, err := em.pubsub.Receive(em.ctx)
		if err != nil {
			// Check if context was canceled
			select {
			case <-em.ctx.Done():
				return
			default:
				// Just an error, continue
				continue
			}
		}

		// Unmarshal event
		event, err := UnmarshalEvent(msg.Payload)
		if err != nil {
			// Invalid event, skip
			continue
		}

		// Dispatch to handlers
		em.dispatchEvent(event)
	}
}

// dispatchEvent calls all registered handlers for an event
func (em *EventManager) dispatchEvent(event *BoardEvent) {
	em.mu.RLock()
	handlers := em.handlers[event.EventType]
	em.mu.RUnlock()

	// Call each handler
	for _, handler := range handlers {
		// Ignore errors from individual handlers
		_ = handler(event)
	}
}

// MarshalEvent marshals an event to JSON
func MarshalEvent(event *BoardEvent) ([]byte, error) {
	eventData, err := json.Marshal(event)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal event: %w", err)
	}
	return eventData, nil
}

// UnmarshalEvent unmarshals an event from JSON
func UnmarshalEvent(data []byte) (*BoardEvent, error) {
	var event BoardEvent
	if err := json.Unmarshal(data, &event); err != nil {
		return nil, fmt.Errorf("failed to unmarshal event: %w", err)
	}
	return &event, nil
}
