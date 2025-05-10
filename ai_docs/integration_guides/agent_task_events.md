# Agent Task Events

This document explains how to use the event system for agent task coordination.

## Overview

Guild uses an event-driven architecture for task coordination. Events are published when task states change, and agents and other components subscribe to these events to react accordingly.

## Event Types

Guild defines the following standard event types:

| Event Type       | Description                            | Emitted By       | Consumed By             |
| ---------------- | -------------------------------------- | ---------------- | ----------------------- |
| `task_created`   | A new task has been created            | Orchestrator     | Agents, UI              |
| `task_assigned`  | A task has been assigned to an agent   | Orchestrator     | Agents, UI              |
| `task_started`   | An agent has started working on a task | Agents           | Orchestrator, UI        |
| `task_updated`   | A task has been updated                | Agents           | Orchestrator, UI        |
| `task_blocked`   | A task is blocked waiting for input    | Agents           | Orchestrator, UI, Human |
| `task_resumed`   | A blocked task has been resumed        | Orchestrator, UI | Agents                  |
| `task_completed` | A task has been completed              | Agents           | Orchestrator, UI        |
| `task_failed`    | A task has failed                      | Agents           | Orchestrator, UI        |

## Event Structure

Events use a standardized JSON structure:

```json
{
  "type": "task_blocked",
  "task_id": "task-123",
  "agent_id": "agent-456",
  "timestamp": "2025-05-08T10:15:30Z",
  "data": {
    "reason": "Need clarification on API endpoint",
    "options": ["Option A", "Option B"]
  }
}
```

## Publishing Events

Events are published using Guild's channel-based Pub/Sub system:

```go
// pkg/orchestrator/eventbus.go
package orchestrator

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/blockhead-consulting/guild/pkg/comms"
	"github.com/blockhead-consulting/guild/pkg/comms/channel"
)

// Event represents a system event
type Event struct {
	Type      string                 `json:"type"`
	TaskID    string                 `json:"task_id"`
	AgentID   string                 `json:"agent_id"`
	Timestamp string                 `json:"timestamp"`
	Data      map[string]interface{} `json:"data"`
}

// EventBus manages event publishing and subscription
type EventBus struct {
	pubsub    comms.PubSub
	callbacks map[string][]func(Event)
}

// NewEventBus creates a new event bus
func NewEventBus() (*EventBus, error) {
	// Create a new channel-based transport
	transport := channel.NewTransport()

	// Create a PubSub with a buffer size of 100
	pubsub, err := transport.NewPubSub(context.Background(), map[string]interface{}{
		"buffer_size": 100,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create pubsub: %w", err)
	}

	bus := &EventBus{
		pubsub:    pubsub,
		callbacks: make(map[string][]func(Event)),
	}

	// Start listening for events
	go bus.listen()

	return bus, nil
}

// Publish sends an event to all subscribers
func (b *EventBus) Publish(ctx context.Context, event Event) error {
	// Set timestamp if not provided
	if event.Timestamp == "" {
		event.Timestamp = time.Now().Format(time.RFC3339)
	}

	// Marshal the event to JSON
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	// Publish to the topic corresponding to the event type
	return b.pubsub.Publish(ctx, "events."+event.Type, data)
}

// listen receives events and triggers callbacks
func (b *EventBus) listen() {
	ctx := context.Background()

	// Subscribe to all event types
	if err := b.pubsub.Subscribe(ctx, "events."); err != nil {
		// Log error but continue
		fmt.Printf("Error subscribing to events: %v\n", err)
	}

	// Process events
	for {
		// Wait for next message
		msg, err := b.pubsub.Receive(ctx)
		if err != nil {
			// Log error but continue
			fmt.Printf("Error receiving message: %v\n", err)
			continue
		}

		// Unmarshal the event
		var event Event
		if err := json.Unmarshal(msg.Payload, &event); err != nil {
			fmt.Printf("Error unmarshaling event: %v\n", err)
			continue
		}

		// Trigger callbacks for this event type
		if callbacks, ok := b.callbacks[event.Type]; ok {
			for _, callback := range callbacks {
				go callback(event)
			}
		}

		// Trigger callbacks for all events
		if callbacks, ok := b.callbacks["*"]; ok {
			for _, callback := range callbacks {
				go callback(event)
			}
		}
	}
}

// Subscribe registers a callback for events
func (b *EventBus) Subscribe(eventType string, callback func(Event)) string {
	if _, ok := b.callbacks[eventType]; !ok {
		b.callbacks[eventType] = []func(Event){}
	}

	b.callbacks[eventType] = append(b.callbacks[eventType], callback)
	return fmt.Sprintf("%s-%d", eventType, len(b.callbacks[eventType])-1)
}

// Unsubscribe removes a subscription
func (b *EventBus) Unsubscribe(subscriptionID string) error {
	// Implementation details...
	return nil
}

// Close closes the event bus
func (b *EventBus) Close() error {
	return b.pubsub.Close()
}
```

## Handling Events

Agents and other components handle events through callbacks:

```go
// Example: Agent subscribing to task events
func (a *Agent) subscribeToEvents(eventBus *orchestrator.EventBus) {
	// Subscribe to assigned tasks
	eventBus.Subscribe("task_assigned", func(event orchestrator.Event) {
		if event.AgentID != a.ID() {
			return
		}

		// Get the task details
		taskID, ok := event.TaskID.(string)
		if !ok {
			return
		}

		// Get the task from the board
		task, err := a.board.Get(taskID)
		if err != nil {
			return
		}

		// Execute the task
		go func() {
			ctx := context.Background()
			a.Execute(ctx, task)
		}()
	})

	// Subscribe to resumed tasks
	eventBus.Subscribe("task_resumed", func(event orchestrator.Event) {
		// Handle resumed tasks...
	})
}
```

## Human Interaction

Blocked tasks require human interaction:

```go
// Example: Agent marking a task as blocked
func (a *Agent) requestHumanInput(ctx context.Context, task Task, reason string, options []string) error {
	// Update task status
	task.Status = StatusBlocked
	if err := a.board.Update(task); err != nil {
		return err
	}

	// Publish blocked event
	event := orchestrator.Event{
		Type:    "task_blocked",
		TaskID:  task.ID,
		AgentID: a.ID(),
		Data: map[string]interface{}{
			"reason":  reason,
			"options": options,
		},
	}

	return a.eventBus.Publish(ctx, event)
}

// Example: CLI handling blocked tasks
func handleBlockedTask(event orchestrator.Event) {
	taskID := event.TaskID
	reason, _ := event.Data["reason"].(string)
	options, _ := event.Data["options"].([]string)

	fmt.Printf("Task %s is blocked: %s\n", taskID, reason)
	if len(options) > 0 {
		fmt.Println("Options:")
		for i, opt := range options {
			fmt.Printf("  %d. %s\n", i+1, opt)
		}
	}

	// Get user input
	fmt.Print("Enter your response: ")
	var response string
	fmt.Scanln(&response)

	// Resume the task
	event := orchestrator.Event{
		Type:    "task_resumed",
		TaskID:  taskID,
		AgentID: event.AgentID,
		Data: map[string]interface{}{
			"input": response,
		},
	}

	eventBus.Publish(context.Background(), event)
}
```

## Example: Task State Machine

This state machine shows how events drive task state transitions:

```
         ┌───────────┐
         │           │
         │   To Do   │
         │           │
         └─────┬─────┘
               │ task_assigned
               ▼
         ┌───────────┐
         │           │
         │In Progress│
         │           │
         └─┬───────┬─┘
           │       │
task_blocked│       │task_completed
           │       │
           ▼       ▼
    ┌──────────┐  ┌───────────┐
    │          │  │           │
    │ Blocked  │  │   Done    │
    │          │  │           │
    └────┬─────┘  └───────────┘
         │
         │task_resumed
         │
         └─────────────┐
                       ▼
                ┌───────────┐
                │           │
                │In Progress│
                │           │
                └───────────┘
```

## Best Practices

1. **Event Documentation**

   - Document all custom event types
   - Specify required data fields

2. **Error Handling**

   - Gracefully handle missing or malformed data
   - Implement retries for failed event processing

3. **Performance Considerations**
   - Process events asynchronously
   - Limit event payload size
   - Use efficient serialization

## Related Documentation

- [../patterns/go_concurrency.md](../patterns/go_concurrency.md)
- [../api_docs/zeromq.md](../api_docs/zeromq.md) (Historical - ZeroMQ has been deferred to future versions)
