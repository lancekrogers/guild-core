# ZeroMQ Support - Deferred

> ⚠️ **NOTICE**: ZeroMQ integration has been deferred to a future version of Guild. This document is maintained for reference only.

ZeroMQ support has been moved to the "horizon features" category. See `/specs/horizon/zeromq_integration.md` for details.

## Current Implementation

Guild currently uses a Go-native channel-based messaging system for inter-component communication. This approach:

1. Requires no external dependencies
2. Uses standard Go concurrency patterns
3. Is simpler to set up and use
4. Works perfectly for single-process deployments

The channel-based implementation is located in `pkg/comms/channel/` and implements the same interfaces previously intended for ZeroMQ.

## Future Plans

ZeroMQ integration will be revisited when:

1. Cross-language agent support is implemented
2. Distributed deployment across multiple machines is needed
3. The core functionality of Guild is stable and well-tested

The existing ZeroMQ code has been preserved in `pkg/extensions/zeromq/` for future reference.

## Original Documentation (For Reference Only)

The following is the original documentation and is kept for reference purposes only.

### Overview

ZeroMQ (also known as ØMQ) is a high-performance asynchronous messaging library that can enable efficient communication between agents, the CLI, and other system components.

### Installation

### Dependencies

```bash
# Debian/Ubuntu
sudo apt-get install libzmq3-dev

# macOS
brew install zeromq

# Windows
# Download from zeromq.org and add to PATH
```

### Go Library

```bash
go get github.com/zeromq/goczmq
```

## Pattern: Publisher/Subscriber

Guild uses the Pub/Sub pattern for event broadcasting:

```go
// pkg/comms/transport/zeromq/client.go
package zeromq

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/zeromq/goczmq"
)

// Publisher broadcasts events to subscribers
type Publisher struct {
	socket *goczmq.Sock
	mutex  sync.Mutex
}

// NewPublisher creates a new ZeroMQ publisher
func NewPublisher(address string) (*Publisher, error) {
	socket, err := goczmq.NewPub(address)
	if err != nil {
		return nil, fmt.Errorf("failed to create publisher: %w", err)
	}

	return &Publisher{
		socket: socket,
	}, nil
}

// Publish sends an event to all subscribers
func (p *Publisher) Publish(ctx context.Context, topic string, data interface{}) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal data: %w", err)
	}

	err = p.socket.SendFrame([]byte(topic), goczmq.FlagMore)
	if err != nil {
		return fmt.Errorf("failed to send topic: %w", err)
	}

	err = p.socket.SendFrame(jsonData, 0)
	if err != nil {
		return fmt.Errorf("failed to send data: %w", err)
	}

	return nil
}

// Close closes the publisher socket
func (p *Publisher) Close() error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	p.socket.Destroy()
	return nil
}

// Subscriber receives events from publishers
type Subscriber struct {
	socket *goczmq.Sock
	mutex  sync.Mutex
}

// NewSubscriber creates a new ZeroMQ subscriber
func NewSubscriber(address string, topics []string) (*Subscriber, error) {
	socket, err := goczmq.NewSub(address, topics)
	if err != nil {
		return nil, fmt.Errorf("failed to create subscriber: %w", err)
	}

	return &Subscriber{
		socket: socket,
	}, nil
}

// Receive waits for an event from a publisher
func (s *Subscriber) Receive(ctx context.Context) (string, []byte, error) {
	// Create a channel for the message
	messageCh := make(chan [][]byte, 1)
	errorCh := make(chan error, 1)

	// Start a goroutine to receive the message
	go func() {
		frames, err := s.socket.RecvMessage()
		if err != nil {
			errorCh <- err
			return
		}
		messageCh <- frames
	}()

	// Wait for a message or context cancellation
	select {
	case frames := <-messageCh:
		if len(frames) < 2 {
			return "", nil, fmt.Errorf("invalid message format")
		}
		return string(frames[0]), frames[1], nil
	case err := <-errorCh:
		return "", nil, err
	case <-ctx.Done():
		return "", nil, ctx.Err()
	}
}

// Close closes the subscriber socket
func (s *Subscriber) Close() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.socket.Destroy()
	return nil
}
```

## Usage Examples

### Publishing Events

```go
// Create a publisher
publisher, err := zeromq.NewPublisher("tcp://*:5555")
if err != nil {
	log.Fatalf("Failed to create publisher: %v", err)
}
defer publisher.Close()

// Publish an event
event := Event{
	Type:      "task_created",
	TaskID:    "task-123",
	AgentID:   "agent-456",
	Timestamp: time.Now().Format(time.RFC3339),
	Data: map[string]interface{}{
		"title":       "Create API endpoint",
		"description": "Implement the /users endpoint",
	},
}

err = publisher.Publish(context.Background(), "events", event)
if err != nil {
	log.Printf("Failed to publish event: %v", err)
}
```

### Subscribing to Events

```go
// Create a subscriber
subscriber, err := zeromq.NewSubscriber("tcp://localhost:5555", []string{"events"})
if err != nil {
	log.Fatalf("Failed to create subscriber: %v", err)
}
defer subscriber.Close()

// Receive events
for {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	topic, data, err := subscriber.Receive(ctx)
	cancel()

	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			continue
		}
		log.Printf("Failed to receive message: %v", err)
		continue
	}

	var event Event
	if err := json.Unmarshal(data, &event); err != nil {
		log.Printf("Failed to parse event: %v", err)
		continue
	}

	log.Printf("Received event: %s - Task: %s", event.Type, event.TaskID)
}
```

## Best Practices

1. **Socket Lifecycle Management**

   - Always close sockets when done
   - Use defer statements to ensure cleanup

2. **Context Cancellation**

   - Pass context to receive operations
   - Implement timeouts for operations

3. **Error Handling**
   - Implement retries for transient errors
   - Log connection issues

## Common Patterns

1. **Event Bus**

   - Central publish-subscribe system
   - Topic-based routing

2. **Request-Reply**

   - Synchronous communication
   - Used for direct agent communication

3. **Push-Pull**
   - Load-balanced task distribution
   - Multiple workers processing tasks

## Related Documentation

- [ZeroMQ Guide](https://zguide.zeromq.org/)
- [../integration_guides/agent_task_events.md](../integration_guides/agent_task_events.md)
