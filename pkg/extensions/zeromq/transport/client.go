package zeromq

import (
	"context"
	"fmt"

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