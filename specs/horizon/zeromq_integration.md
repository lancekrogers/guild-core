## ZeroMQ Integration

@context
@lore_conventions

This command guides you through integrating ZeroMQ into the Guild project, primarily for the Kanban event system but with design considerations for other messaging needs.

### 1. Check Existing Implementation

```bash
# Check for existing ZeroMQ-related files
find . -type f -name "*.go" | xargs grep -l "zmq\|zeromq" | grep -v "_test.go"

# Check for existing event system implementations
find . -type f -name "*.go" | xargs grep -l "event\|publish\|subscribe" | grep -v "_test.go"
```

### 2. ZeroMQ Package Structure

Implement ZeroMQ support with this package structure:

```
pkg/
└── comms/
    ├── interface.go           # Communication interfaces
    └── transport/
        ├── interface.go       # Transport interfaces
        └── zeromq/
            ├── config.go      # ZeroMQ configuration
            ├── client.go      # Client implementation
            ├── pubsub.go      # Publisher/Subscriber implementation
            └── zeromq_test.go # Tests
```

### 3. Core Implementation Steps

#### A. Communication Interfaces

Create `pkg/comms/interface.go` with these interfaces:

```go
package comms

import (
	"context"
)

// Message represents a serializable message
type Message struct {
	Topic   string
	Headers map[string]string
	Payload []byte
}

// Publisher defines a message publishing interface
type Publisher interface {
	// Publish sends a message to a topic
	Publish(ctx context.Context, topic string, payload []byte) error

	// PublishMessage sends a structured message
	PublishMessage(ctx context.Context, msg *Message) error

	// Close shuts down the publisher
	Close() error
}

// Subscriber defines a message subscription interface
type Subscriber interface {
	// Subscribe registers interest in a topic pattern
	Subscribe(ctx context.Context, topicPattern string) error

	// Unsubscribe removes interest in a topic pattern
	Unsubscribe(ctx context.Context, topicPattern string) error

	// Receive waits for and returns the next message
	Receive(ctx context.Context) (*Message, error)

	// Close shuts down the subscriber
	Close() error
}

// MessageHandler is a callback for received messages
type MessageHandler func(ctx context.Context, msg *Message) error

// PubSub combines Publisher and Subscriber interfaces
type PubSub interface {
	Publisher
	Subscriber
}
```

#### B. Transport Interfaces

Create `pkg/comms/transport/interface.go`:

```go
package transport

import (
	"context"

	"github.com/blockhead-consulting/guild/pkg/comms"
)

// Transport defines a communication transport mechanism
type Transport interface {
	// NewPublisher creates a new publisher
	NewPublisher(ctx context.Context, config map[string]interface{}) (comms.Publisher, error)

	// NewSubscriber creates a new subscriber
	NewSubscriber(ctx context.Context, config map[string]interface{}) (comms.Subscriber, error)

	// NewPubSub creates a combined publisher/subscriber
	NewPubSub(ctx context.Context, config map[string]interface{}) (comms.PubSub, error)
}

// Factory creates transport implementations
type Factory interface {
	// GetTransport returns a named transport
	GetTransport(name string) (Transport, error)
}
```

#### C. ZeroMQ Configuration

Create `pkg/comms/transport/zeromq/config.go`:

```go
package zeromq

import (
	"errors"
	"fmt"
)

// Config holds ZeroMQ configuration
type Config struct {
	// PubEndpoint is the ZeroMQ publisher endpoint (e.g., "tcp://127.0.0.1:5556")
	PubEndpoint string

	// SubEndpoint is the ZeroMQ subscriber endpoint (e.g., "tcp://127.0.0.1:5557")
	SubEndpoint string

	// Identity is an optional identity for the socket
	Identity string

	// HighWaterMark limits queue size (0 = no limit)
	HighWaterMark int

	// Timeout in milliseconds (0 = no timeout)
	Timeout int
}

// Validate checks the configuration
func (c *Config) Validate() error {
	if c.PubEndpoint == "" && c.SubEndpoint == "" {
		return errors.New("at least one endpoint must be specified")
	}

	return nil
}

// FromMap creates a Config from a map
func FromMap(m map[string]interface{}) (*Config, error) {
	config := &Config{}

	if v, ok := m["pub_endpoint"].(string); ok {
		config.PubEndpoint = v
	}

	if v, ok := m["sub_endpoint"].(string); ok {
		config.SubEndpoint = v
	}

	if v, ok := m["identity"].(string); ok {
		config.Identity = v
	}

	if v, ok := m["high_water_mark"].(int); ok {
		config.HighWaterMark = v
	}

	if v, ok := m["timeout"].(int); ok {
		config.Timeout = v
	}

	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid ZeroMQ configuration: %w", err)
	}

	return config, nil
}
```

#### D. ZeroMQ Client Implementation

Create `pkg/comms/transport/zeromq/client.go`:

```go
package zeromq

import (
	"context"
	"fmt"
	"sync"

	zmq "github.com/pebbe/zmq4"
	"github.com/blockhead-consulting/guild/pkg/comms"
)

// Transport implements the transport.Transport interface for ZeroMQ
type Transport struct{}

// NewTransport creates a new ZeroMQ transport
func NewTransport() *Transport {
	return &Transport{}
}

// NewPublisher creates a new ZeroMQ publisher
func (t *Transport) NewPublisher(ctx context.Context, config map[string]interface{}) (comms.Publisher, error) {
	cfg, err := FromMap(config)
	if err != nil {
		return nil, err
	}

	if cfg.PubEndpoint == "" {
		return nil, fmt.Errorf("publisher endpoint must be specified")
	}

	return NewPublisher(ctx, cfg)
}

// NewSubscriber creates a new ZeroMQ subscriber
func (t *Transport) NewSubscriber(ctx context.Context, config map[string]interface{}) (comms.Subscriber, error) {
	cfg, err := FromMap(config)
	if err != nil {
		return nil, err
	}

	if cfg.SubEndpoint == "" {
		return nil, fmt.Errorf("subscriber endpoint must be specified")
	}

	return NewSubscriber(ctx, cfg)
}

// NewPubSub creates a new ZeroMQ publisher/subscriber
func (t *Transport) NewPubSub(ctx context.Context, config map[string]interface{}) (comms.PubSub, error) {
	cfg, err := FromMap(config)
	if err != nil {
		return nil, err
	}

	if cfg.PubEndpoint == "" || cfg.SubEndpoint == "" {
		return nil, fmt.Errorf("both publisher and subscriber endpoints must be specified")
	}

	return NewPubSub(ctx, cfg)
}
```

#### E. PubSub Implementation

Create `pkg/comms/transport/zeromq/pubsub.go`:

```go
package zeromq

import (
	"context"
	"fmt"
	"sync"
	"time"

	zmq "github.com/pebbe/zmq4"
	"github.com/blockhead-consulting/guild/pkg/comms"
)

// Publisher implements a ZeroMQ publisher
type Publisher struct {
	socket *zmq.Socket
	config *Config
	mu     sync.Mutex
	closed bool
}

// NewPublisher creates a new ZeroMQ publisher
func NewPublisher(ctx context.Context, config *Config) (*Publisher, error) {
	socket, err := zmq.NewSocket(zmq.PUB)
	if err != nil {
		return nil, fmt.Errorf("failed to create publisher socket: %w", err)
	}

	if config.Identity != "" {
		if err := socket.SetIdentity(config.Identity); err != nil {
			socket.Close()
			return nil, fmt.Errorf("failed to set socket identity: %w", err)
		}
	}

	if config.HighWaterMark > 0 {
		if err := socket.SetSndhwm(config.HighWaterMark); err != nil {
			socket.Close()
			return nil, fmt.Errorf("failed to set high water mark: %w", err)
		}
	}

	if err := socket.Bind(config.PubEndpoint); err != nil {
		socket.Close()
		return nil, fmt.Errorf("failed to bind publisher socket: %w", err)
	}

	// Allow time for connection to establish
	time.Sleep(100 * time.Millisecond)

	return &Publisher{
		socket: socket,
		config: config,
	}, nil
}

// Publish sends a message to a topic
func (p *Publisher) Publish(ctx context.Context, topic string, payload []byte) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return fmt.Errorf("publisher is closed")
	}

	// Check for context cancellation
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		// Continue
	}

	// ZeroMQ multipart message: [topic, payload]
	_, err := p.socket.SendMessage(topic, payload)
	if err != nil {
		return fmt.Errorf("failed to publish message: %w", err)
	}

	return nil
}

// PublishMessage sends a structured message
func (p *Publisher) PublishMessage(ctx context.Context, msg *comms.Message) error {
	return p.Publish(ctx, msg.Topic, msg.Payload)
}

// Close shuts down the publisher
func (p *Publisher) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return nil
	}

	p.closed = true
	return p.socket.Close()
}

// Subscriber implements a ZeroMQ subscriber
type Subscriber struct {
	socket   *zmq.Socket
	config   *Config
	mu       sync.Mutex
	closed   bool
	topics   map[string]struct{}
	recvChan chan *comms.Message
	errChan  chan error
	stopChan chan struct{}
	wg       sync.WaitGroup
}

// NewSubscriber creates a new ZeroMQ subscriber
func NewSubscriber(ctx context.Context, config *Config) (*Subscriber, error) {
	socket, err := zmq.NewSocket(zmq.SUB)
	if err != nil {
		return nil, fmt.Errorf("failed to create subscriber socket: %w", err)
	}

	if config.Identity != "" {
		if err := socket.SetIdentity(config.Identity); err != nil {
			socket.Close()
			return nil, fmt.Errorf("failed to set socket identity: %w", err)
		}
	}

	if config.HighWaterMark > 0 {
		if err := socket.SetRcvhwm(config.HighWaterMark); err != nil {
			socket.Close()
			return nil, fmt.Errorf("failed to set high water mark: %w", err)
		}
	}

	if config.Timeout > 0 {
		if err := socket.SetRcvtimeo(config.Timeout); err != nil {
			socket.Close()
			return nil, fmt.Errorf("failed to set receive timeout: %w", err)
		}
	}

	if err := socket.Connect(config.SubEndpoint); err != nil {
		socket.Close()
		return nil, fmt.Errorf("failed to connect subscriber socket: %w", err)
	}

	sub := &Subscriber{
		socket:   socket,
		config:   config,
		topics:   make(map[string]struct{}),
		recvChan: make(chan *comms.Message),
		errChan:  make(chan error),
		stopChan: make(chan struct{}),
	}

	// Start background receiver
	sub.wg.Add(1)
	go sub.receiveLoop(ctx)

	return sub, nil
}

// Subscribe registers interest in a topic pattern
func (s *Subscriber) Subscribe(ctx context.Context, topicPattern string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return fmt.Errorf("subscriber is closed")
	}

	// Check if already subscribed
	if _, exists := s.topics[topicPattern]; exists {
		return nil
	}

	// ZeroMQ topic subscription
	if err := s.socket.SetSubscribe(topicPattern); err != nil {
		return fmt.Errorf("failed to subscribe to topic: %w", err)
	}

	s.topics[topicPattern] = struct{}{}
	return nil
}

// Unsubscribe removes interest in a topic pattern
func (s *Subscriber) Unsubscribe(ctx context.Context, topicPattern string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return fmt.Errorf("subscriber is closed")
	}

	// Check if subscribed
	if _, exists := s.topics[topicPattern]; !exists {
		return nil
	}

	// ZeroMQ topic unsubscription
	if err := s.socket.SetUnsubscribe(topicPattern); err != nil {
		return fmt.Errorf("failed to unsubscribe from topic: %w", err)
	}

	delete(s.topics, topicPattern)
	return nil
}

// Receive waits for and returns the next message
func (s *Subscriber) Receive(ctx context.Context) (*comms.Message, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case msg := <-s.recvChan:
		return msg, nil
	case err := <-s.errChan:
		return nil, err
	}
}

// Close shuts down the subscriber
func (s *Subscriber) Close() error {
	s.mu.Lock()

	if s.closed {
		s.mu.Unlock()
		return nil
	}

	s.closed = true
	close(s.stopChan)
	s.mu.Unlock()

	// Wait for receiver goroutine to exit
	s.wg.Wait()

	return s.socket.Close()
}

// receiveLoop continuously receives messages in the background
func (s *Subscriber) receiveLoop(ctx context.Context) {
	defer s.wg.Done()

	poller := zmq.NewPoller()
	poller.Add(s.socket, zmq.POLLIN)

	for {
		select {
		case <-s.stopChan:
			return
		default:
			// Continue
		}

		// Poll with timeout
		sockets, err := poller.Poll(100 * time.Millisecond)
		if err != nil {
			select {
			case s.errChan <- fmt.Errorf("poll error: %w", err):
			case <-s.stopChan:
				return
			default:
				// Drop error if can't send
			}
			continue
		}

		if len(sockets) == 0 {
			continue
		}

		// Receive ZeroMQ multipart message
		parts, err := s.socket.RecvMessageBytes(0)
		if err != nil {
			select {
			case s.errChan <- fmt.Errorf("receive error: %w", err):
			case <-s.stopChan:
				return
			default:
				// Drop error if can't send
			}
			continue
		}

		if len(parts) < 2 {
			// Invalid message format
			continue
		}

		msg := &comms.Message{
			Topic:   string(parts[0]),
			Payload: parts[1],
		}

		select {
		case s.recvChan <- msg:
		case <-s.stopChan:
			return
		default:
			// Drop message if channel full
		}
	}
}

// PubSub implements both Publisher and Subscriber interfaces
type PubSub struct {
	pub *Publisher
	sub *Subscriber
}

// NewPubSub creates a new ZeroMQ PubSub
func NewPubSub(ctx context.Context, config *Config) (*PubSub, error) {
	pub, err := NewPublisher(ctx, config)
	if err != nil {
		return nil, err
	}

	sub, err := NewSubscriber(ctx, config)
	if err != nil {
		pub.Close()
		return nil, err
	}

	return &PubSub{
		pub: pub,
		sub: sub,
	}, nil
}

// Publish sends a message to a topic
func (ps *PubSub) Publish(ctx context.Context, topic string, payload []byte) error {
	return ps.pub.Publish(ctx, topic, payload)
}

// PublishMessage sends a structured message
func (ps *PubSub) PublishMessage(ctx context.Context, msg *comms.Message) error {
	return ps.pub.PublishMessage(ctx, msg)
}

// Subscribe registers interest in a topic pattern
func (ps *PubSub) Subscribe(ctx context.Context, topicPattern string) error {
	return ps.sub.Subscribe(ctx, topicPattern)
}

// Unsubscribe removes interest in a topic pattern
func (ps *PubSub) Unsubscribe(ctx context.Context, topicPattern string) error {
	return ps.sub.Unsubscribe(ctx, topicPattern)
}

// Receive waits for and returns the next message
func (ps *PubSub) Receive(ctx context.Context) (*comms.Message, error) {
	return ps.sub.Receive(ctx)
}

// Close shuts down the PubSub
func (ps *PubSub) Close() error {
	pubErr := ps.pub.Close()
	subErr := ps.sub.Close()

	if pubErr != nil {
		return pubErr
	}

	return subErr
}
```

### 4. Kanban Event System Integration

#### A. Define Event Types

Update `pkg/kanban/events.go`:

```go
package kanban

import (
	"encoding/json"
	"fmt"
	"time"
)

// EventType represents the type of a kanban event
type EventType string

const (
	// Event types
	EventTaskCreated  EventType = "task_created"
	EventTaskMoved    EventType = "task_moved"
	EventTaskUpdated  EventType = "task_updated"
	EventTaskDeleted  EventType = "task_deleted"
	EventTaskBlocked  EventType = "task_blocked"
	EventTaskUnblocked EventType = "task_unblocked"
)

// Event represents a kanban board event
type Event struct {
	Type        EventType   `json:"type"`
	TaskID      string      `json:"task_id"`
	BoardID     string      `json:"board_id"`
	AgentID     string      `json:"agent_id,omitempty"`
	OldStatus   string      `json:"old_status,omitempty"`
	NewStatus   string      `json:"new_status,omitempty"`
	Changes     interface{} `json:"changes,omitempty"`
	Timestamp   time.Time   `json:"timestamp"`
}

// NewEvent creates a new event
func NewEvent(eventType EventType, taskID, boardID string) *Event {
	return &Event{
		Type:      eventType,
		TaskID:    taskID,
		BoardID:   boardID,
		Timestamp: time.Now(),
	}
}

// TaskMovedEvent creates an event for a task status change
func TaskMovedEvent(taskID, boardID, from, to string) *Event {
	event := NewEvent(EventTaskMoved, taskID, boardID)
	event.OldStatus = from
	event.NewStatus = to
	return event
}

// Marshal serializes an event to JSON
func (e *Event) Marshal() ([]byte, error) {
	return json.Marshal(e)
}

// Unmarshal deserializes an event from JSON
func Unmarshal(data []byte) (*Event, error) {
	var event Event
	err := json.Unmarshal(data, &event)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal event: %w", err)
	}
	return &event, nil
}
```

#### B. Create EventManager

Create `pkg/kanban/event_manager.go`:

```go
package kanban

import (
	"context"
	"fmt"
	"sync"

	"github.com/blockhead-consulting/guild/pkg/comms"
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
type EventHandler func(event *Event) error

// NewEventManager creates a new event manager
func NewEventManager(pubsub comms.PubSub, topicPrefix string) *EventManager {
	ctx, cancel := context.WithCancel(context.Background())

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
func (em *EventManager) PublishEvent(event *Event) error {
	data, err := event.Marshal()
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	topic := em.topicPrefix + string(event.Type)
	return em.pubsub.Publish(em.ctx, topic, data)
}

// Subscribe adds a handler for a specific event type
func (em *EventManager) Subscribe(eventType EventType, handler EventHandler) error {
	em.mu.Lock()
	defer em.mu.Unlock()

	// Subscribe to ZeroMQ topic if this is the first handler
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
		event, err := Unmarshal(msg.Payload)
		if err != nil {
			// Invalid event, skip
			continue
		}

		// Dispatch to handlers
		em.dispatchEvent(event)
	}
}

// dispatchEvent calls all registered handlers for an event
func (em *EventManager) dispatchEvent(event *Event) {
	em.mu.RLock()
	handlers := em.handlers[event.Type]
	em.mu.RUnlock()

	// Call each handler
	for _, handler := range handlers {
		// Ignore errors from individual handlers
		_ = handler(event)
	}
}
```

#### C. Update Board Implementation

Update `pkg/kanban/board.go` to use the event manager:

```go
// Add to Board struct
type Board struct {
	// ... existing fields
	eventManager *EventManager
}

// Update MoveTask to publish events
func (b *Board) MoveTask(taskID, newStatus string) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	task, ok := b.tasks[taskID]
	if !ok {
		return fmt.Errorf("task not found: %s", taskID)
	}

	oldStatus := task.Status
	task.Status = newStatus
	task.UpdatedAt = time.Now()

	// Save to store
	if err := b.store.SaveTask(task); err != nil {
		return fmt.Errorf("failed to save task: %w", err)
	}

	// Publish event
	event := TaskMovedEvent(taskID, b.id, oldStatus, newStatus)
	if b.eventManager != nil {
		if err := b.eventManager.PublishEvent(event); err != nil {
			// Log but don't fail the operation
			fmt.Printf("Failed to publish task moved event: %v\n", err)
		}
	}

	return nil
}

// Similar updates for other methods (AddTask, UpdateTask, etc.)
```

### 5. Testing ZeroMQ Integration

Create `pkg/comms/transport/zeromq/zeromq_test.go`:

```go
package zeromq

import (
	"context"
	"testing"
	"time"

	"github.com/blockhead-consulting/guild/pkg/comms"
)

func TestPubSubIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Create publisher config
	pubConfig := &Config{
		PubEndpoint: "tcp://127.0.0.1:5556",
		Identity:    "test-publisher",
	}

	// Create subscriber config
	subConfig := &Config{
		SubEndpoint: "tcp://127.0.0.1:5556",
		Identity:    "test-subscriber",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Create publisher
	pub, err := NewPublisher(ctx, pubConfig)
	if err != nil {
		t.Fatalf("Failed to create publisher: %v", err)
	}
	defer pub.Close()

	// Create subscriber
	sub, err := NewSubscriber(ctx, subConfig)
	if err != nil {
		t.Fatalf("Failed to create subscriber: %v", err)
	}
	defer sub.Close()

	// Subscribe to test topic
	if err := sub.Subscribe(ctx, "test-topic"); err != nil {
		t.Fatalf("Failed to subscribe: %v", err)
	}

	// Give time for subscription to establish
	time.Sleep(500 * time.Millisecond)

	// Create test message
	testPayload := []byte("Hello, ZeroMQ!")

	// Publish message
	if err := pub.Publish(ctx, "test-topic", testPayload); err != nil {
		t.Fatalf("Failed to publish message: %v", err)
	}

	// Create context with timeout for receive
	receiveCtx, receiveCancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer receiveCancel()

	// Receive message
	msg, err := sub.Receive(receiveCtx)
	if err != nil {
		t.Fatalf("Failed to receive message: %v", err)
	}

	// Verify message
	if msg.Topic != "test-topic" {
		t.Errorf("Expected topic %q, got %q", "test-topic", msg.Topic)
	}

	if string(msg.Payload) != string(testPayload) {
		t.Errorf("Expected payload %q, got %q", string(testPayload), string(msg.Payload))
	}
}

func TestEventManager(t *testing.T) {
	// Test the kanban EventManager with ZeroMQ
	// Similar to above but using the kanban event system
}
```

### 6. Integration with Kanban Board Manager

Update `pkg/kanban/manager.go`:

```go
// Add to BoardManager struct
type BoardManager struct {
	// ... existing fields
	eventManager *EventManager
}

// Initialize ZeroMQ in NewBoardManager
func NewBoardManager(store Store) (*BoardManager, error) {
	// Create ZeroMQ transport
	transport := zeromq.NewTransport()

	// Create PubSub
	pubSubConfig := map[string]interface{}{
		"pub_endpoint": "tcp://127.0.0.1:5556",
		"sub_endpoint": "tcp://127.0.0.1:5556",
		"identity":     "kanban-manager",
	}

	pubsub, err := transport.NewPubSub(context.Background(), pubSubConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create ZeroMQ pubsub: %w", err)
	}

	// Create event manager
	eventManager := NewEventManager(pubsub, "kanban.")

	// Subscribe to important events
	eventManager.SubscribeAll(func(event *Event) error {
		// Log events
		fmt.Printf("Kanban event: %s on board %s, task %s\n",
			event.Type, event.BoardID, event.TaskID)
		return nil
	})

	return &BoardManager{
		store:        store,
		boards:       make(map[string]*Board),
		eventManager: eventManager,
	}, nil
}

// Update CreateBoard to pass event manager to boards
func (m *BoardManager) CreateBoard(id string) (*Board, error) {
	// ... existing code

	board := &Board{
		id:           id,
		tasks:        make(map[string]*Task),
		store:        m.store,
		eventManager: m.eventManager,
	}

	// ... rest of method
}
```

### 7. CLI Integration

Add ZeroMQ-specific flags to the CLI:

```go
// cmd/guild/main.go
var (
	zmqPubEndpoint string
	zmqSubEndpoint string
)

func init() {
	rootCmd.PersistentFlags().StringVar(&zmqPubEndpoint, "zmq-pub", "tcp://127.0.0.1:5556", "ZeroMQ publisher endpoint")
	rootCmd.PersistentFlags().StringVar(&zmqSubEndpoint, "zmq-sub", "tcp://127.0.0.1:5556", "ZeroMQ subscriber endpoint")
}
```

### 8. Configuration Integration

Update your configuration to include ZeroMQ settings:

```go
// pkg/config/config.go
type ZeroMQConfig struct {
	PubEndpoint string `yaml:"pub_endpoint"`
	SubEndpoint string `yaml:"sub_endpoint"`
	Identity    string `yaml:"identity,omitempty"`
}

type Config struct {
	// ... existing fields
	ZeroMQ ZeroMQConfig `yaml:"zeromq"`
}
```

### Implementation Considerations

1. **Dependencies**:

   - Add the Go ZeroMQ library: `go get github.com/pebbe/zmq4`
   - This requires the ZeroMQ C library to be installed on the system

2. **Error Handling**:

   - Implement robust error handling with context support
   - Handle ZeroMQ-specific errors appropriately

3. **Graceful Shutdown**:

   - Ensure all ZeroMQ resources are properly closed
   - Implement graceful shutdown for event loops

4. **Performance**:

   - Use buffered channels for high-throughput scenarios
   - Consider separate goroutines for sending/receiving

5. **Security**:
   - For production, consider adding authentication and encryption
   - ZeroMQ supports various security mechanisms

### Expected Integration Result

With this implementation, your Kanban system will:

1. Publish events when tasks are created, moved, updated, or deleted
2. Allow components to subscribe to specific event types
3. Handle event distribution across different parts of the system
4. Provide a foundation for other messaging needs in the application

The ZeroMQ implementation is abstracted behind generic interfaces, allowing you to switch transport mechanisms in the future if needed.
