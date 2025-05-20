package zeromq

import (
	"context"
	"fmt"
	"sync"
	"time"

	zmq "github.com/pebbe/zmq4"
	"github.com/guild-ventures/guild-core/pkg/comms"
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