package transport

import (
	"context"

	"github.com/guild-ventures/guild-core/pkg/comms"
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
