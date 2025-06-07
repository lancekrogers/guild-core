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
