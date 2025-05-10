// Package channel provides a Go-native implementation of the comms interfaces
package channel

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/blockhead-consulting/guild/pkg/comms"
)

// Transport implements transport.Transport using Go channels
type Transport struct {
	// Shared topic registry for all publishers and subscribers
	registry *topicRegistry
}

// NewTransport creates a new channel-based transport
func NewTransport() *Transport {
	return &Transport{
		registry: newTopicRegistry(),
	}
}

// NewPublisher creates a new channel-based publisher
func (t *Transport) NewPublisher(ctx context.Context, config map[string]interface{}) (comms.Publisher, error) {
	bufferSize := 100 // Default buffer size
	if size, ok := config["buffer_size"].(int); ok && size > 0 {
		bufferSize = size
	}

	return newPublisher(t.registry, bufferSize), nil
}

// NewSubscriber creates a new channel-based subscriber
func (t *Transport) NewSubscriber(ctx context.Context, config map[string]interface{}) (comms.Subscriber, error) {
	bufferSize := 100 // Default buffer size
	if size, ok := config["buffer_size"].(int); ok && size > 0 {
		bufferSize = size
	}

	return newSubscriber(t.registry, bufferSize), nil
}

// NewPubSub creates a new channel-based combined publisher/subscriber
func (t *Transport) NewPubSub(ctx context.Context, config map[string]interface{}) (comms.PubSub, error) {
	bufferSize := 100 // Default buffer size
	if size, ok := config["buffer_size"].(int); ok && size > 0 {
		bufferSize = size
	}

	return newPubSub(t.registry, bufferSize), nil
}

// topicRegistry manages message distribution between publishers and subscribers
type topicRegistry struct {
	// Map from topic pattern to subscribers
	subscribers map[string][]*subscriber
	// Channel for registering new subscribers
	register chan subscriberRegistration
	// Channel for unregistering subscribers
	unregister chan subscriberUnregistration
	// Channel for publishing messages
	publish chan publishRequest
	mu      sync.RWMutex
	done    chan struct{}
}

type subscriberRegistration struct {
	sub          *subscriber
	topicPattern string
	resultCh     chan error
}

type subscriberUnregistration struct {
	sub          *subscriber
	topicPattern string
	resultCh     chan error
}

type publishRequest struct {
	topic   string
	payload []byte
}

func newTopicRegistry() *topicRegistry {
	tr := &topicRegistry{
		subscribers: make(map[string][]*subscriber),
		register:    make(chan subscriberRegistration),
		unregister:  make(chan subscriberUnregistration),
		publish:     make(chan publishRequest, 100),
		done:        make(chan struct{}),
	}

	go tr.run()
	return tr
}

func (tr *topicRegistry) run() {
	for {
		select {
		case <-tr.done:
			return

		case reg := <-tr.register:
			tr.mu.Lock()
			tr.subscribers[reg.topicPattern] = append(tr.subscribers[reg.topicPattern], reg.sub)
			tr.mu.Unlock()
			reg.resultCh <- nil

		case unreg := <-tr.unregister:
			tr.mu.Lock()
			subs := tr.subscribers[unreg.topicPattern]
			for i, sub := range subs {
				if sub == unreg.sub {
					tr.subscribers[unreg.topicPattern] = append(subs[:i], subs[i+1:]...)
					break
				}
			}
			tr.mu.Unlock()
			unreg.resultCh <- nil

		case pub := <-tr.publish:
			tr.mu.RLock()
			// Match exact topic and pattern subscriptions (simple implementation)
			for pattern, subs := range tr.subscribers {
				// Basic pattern matching - exact match or prefix match with wildcard
				if pattern == pub.topic || (pattern != "" && pattern[len(pattern)-1] == '#' && 
					len(pattern) > 1 && pub.topic[:len(pattern)-1] == pattern[:len(pattern)-1]) {
					for _, sub := range subs {
						// Don't block on slow subscribers, just send if possible
						select {
						case sub.msgCh <- comms.Message{
							Topic:   pub.topic,
							Payload: pub.payload,
						}:
						default:
							// Message dropped if subscriber is slow
						}
					}
				}
			}
			tr.mu.RUnlock()
		}
	}
}

func (tr *topicRegistry) close() {
	close(tr.done)
}

// Publisher implements comms.Publisher using Go channels
type Publisher struct {
	registry   *topicRegistry
	closed     bool
	closeMutex sync.Mutex
}

func newPublisher(registry *topicRegistry, bufferSize int) *Publisher {
	return &Publisher{
		registry: registry,
	}
}

// Publish sends a message to a topic
func (p *Publisher) Publish(ctx context.Context, topic string, payload []byte) error {
	p.closeMutex.Lock()
	if p.closed {
		p.closeMutex.Unlock()
		return errors.New("publisher is closed")
	}
	p.closeMutex.Unlock()

	// Check for context cancellation
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		// Continue
	}

	select {
	case p.registry.publish <- publishRequest{topic: topic, payload: payload}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// PublishMessage sends a structured message
func (p *Publisher) PublishMessage(ctx context.Context, msg *comms.Message) error {
	if msg == nil {
		return errors.New("message cannot be nil")
	}
	return p.Publish(ctx, msg.Topic, msg.Payload)
}

// Close shuts down the publisher
func (p *Publisher) Close() error {
	p.closeMutex.Lock()
	defer p.closeMutex.Unlock()

	if p.closed {
		return nil
	}

	p.closed = true
	return nil
}

// Subscriber implements comms.Subscriber using Go channels
type subscriber struct {
	registry     *topicRegistry
	msgCh        chan comms.Message
	subscriptions map[string]struct{}
	closed       bool
	closeMutex   sync.Mutex
}

func newSubscriber(registry *topicRegistry, bufferSize int) *subscriber {
	return &subscriber{
		registry:      registry,
		msgCh:         make(chan comms.Message, bufferSize),
		subscriptions: make(map[string]struct{}),
	}
}

// Subscribe registers interest in a topic pattern
func (s *subscriber) Subscribe(ctx context.Context, topicPattern string) error {
	s.closeMutex.Lock()
	if s.closed {
		s.closeMutex.Unlock()
		return errors.New("subscriber is closed")
	}

	// Check if already subscribed
	if _, exists := s.subscriptions[topicPattern]; exists {
		s.closeMutex.Unlock()
		return nil
	}
	s.closeMutex.Unlock()

	// Register with the topic registry
	resultCh := make(chan error, 1)
	select {
	case s.registry.register <- subscriberRegistration{
		sub:          s,
		topicPattern: topicPattern,
		resultCh:     resultCh,
	}:
		// Wait for registration to complete
		err := <-resultCh
		if err != nil {
			return fmt.Errorf("failed to subscribe: %w", err)
		}

		s.closeMutex.Lock()
		s.subscriptions[topicPattern] = struct{}{}
		s.closeMutex.Unlock()
		return nil

	case <-ctx.Done():
		return ctx.Err()
	}
}

// Unsubscribe removes interest in a topic pattern
func (s *subscriber) Unsubscribe(ctx context.Context, topicPattern string) error {
	s.closeMutex.Lock()
	if s.closed {
		s.closeMutex.Unlock()
		return errors.New("subscriber is closed")
	}

	// Check if subscribed
	if _, exists := s.subscriptions[topicPattern]; !exists {
		s.closeMutex.Unlock()
		return nil
	}
	s.closeMutex.Unlock()

	// Unregister from the topic registry
	resultCh := make(chan error, 1)
	select {
	case s.registry.unregister <- subscriberUnregistration{
		sub:          s,
		topicPattern: topicPattern,
		resultCh:     resultCh,
	}:
		// Wait for unregistration to complete
		err := <-resultCh
		if err != nil {
			return fmt.Errorf("failed to unsubscribe: %w", err)
		}

		s.closeMutex.Lock()
		delete(s.subscriptions, topicPattern)
		s.closeMutex.Unlock()
		return nil

	case <-ctx.Done():
		return ctx.Err()
	}
}

// Receive waits for and returns the next message
func (s *subscriber) Receive(ctx context.Context) (*comms.Message, error) {
	s.closeMutex.Lock()
	if s.closed {
		s.closeMutex.Unlock()
		return nil, errors.New("subscriber is closed")
	}
	s.closeMutex.Unlock()

	select {
	case msg := <-s.msgCh:
		return &msg, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// Close shuts down the subscriber
func (s *subscriber) Close() error {
	s.closeMutex.Lock()
	defer s.closeMutex.Unlock()

	if s.closed {
		return nil
	}

	s.closed = true
	// Note: We don't close the message channel here because the registry might still be sending to it
	// The garbage collector will clean it up when no more references exist
	return nil
}

// PubSub implements both Publisher and Subscriber interfaces
type PubSub struct {
	pub *Publisher
	sub *subscriber
}

func newPubSub(registry *topicRegistry, bufferSize int) *PubSub {
	return &PubSub{
		pub: newPublisher(registry, bufferSize),
		sub: newSubscriber(registry, bufferSize),
	}
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