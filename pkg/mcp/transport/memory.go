// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package transport

import (
	"context"
	"sync"

	"github.com/lancekrogers/guild/pkg/gerror"
)

// MemoryTransport implements an in-memory transport for testing
type MemoryTransport struct {
	config    *TransportConfig
	mu        sync.RWMutex
	connected bool
	channels  map[string]chan []byte
	buffer    chan []byte
}

// NewMemoryTransport creates a new memory transport
func NewMemoryTransport(config *TransportConfig) *MemoryTransport {
	bufferSize := 1000
	if config.Config != nil {
		if size, ok := config.Config["buffer_size"].(int); ok {
			bufferSize = size
		}
	}

	return &MemoryTransport{
		config:   config,
		channels: make(map[string]chan []byte),
		buffer:   make(chan []byte, bufferSize),
	}
}

// Connect establishes the memory transport connection
func (t *MemoryTransport) Connect(ctx context.Context) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.connected {
		return gerror.New(gerror.ErrCodeInternal, "mcp_memory_transport", nil).WithComponent("connect").WithOperation("already connected")
	}

	t.connected = true
	return nil
}

// Disconnect closes the memory transport connection
func (t *MemoryTransport) Disconnect(ctx context.Context) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if !t.connected {
		return gerror.New(gerror.ErrCodeInternal, "mcp_memory_transport", nil).WithComponent("disconnect").WithOperation("not connected")
	}

	// Close all channels
	for _, ch := range t.channels {
		close(ch)
	}
	close(t.buffer)

	// Reset state
	t.channels = make(map[string]chan []byte)
	t.buffer = make(chan []byte, cap(t.buffer))
	t.connected = false

	return nil
}

// Send sends a message via memory transport
func (t *MemoryTransport) Send(ctx context.Context, topic string, data []byte) error {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if !t.connected {
		return gerror.New(gerror.ErrCodeInternal, "mcp_memory_transport", nil).WithComponent("disconnect").WithOperation("not connected")
	}

	// Copy data to avoid race conditions
	dataCopy := make([]byte, len(data))
	copy(dataCopy, data)

	select {
	case t.buffer <- dataCopy:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	default:
		return gerror.New(gerror.ErrCodeInternal, "mcp_memory_transport", nil).WithComponent("send").WithOperation("buffer full")
	}
}

// Receive receives messages from memory transport
func (t *MemoryTransport) Receive(ctx context.Context) <-chan []byte {
	return t.buffer
}

// Request sends a request and waits for response (simplified for memory transport)
func (t *MemoryTransport) Request(ctx context.Context, topic string, data []byte) ([]byte, error) {
	// For memory transport, we'll just echo the request as response
	// In a real implementation, this would involve proper request-response handling
	return data, nil
}

// Subscribe subscribes to a topic (no-op for memory transport)
func (t *MemoryTransport) Subscribe(ctx context.Context, topic string) (<-chan []byte, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if !t.connected {
		return nil, gerror.New(gerror.ErrCodeInternal, "mcp_memory_transport", nil).WithComponent("receive").WithOperation("not connected")
	}

	ch := make(chan []byte, 100)
	t.channels[topic] = ch
	return ch, nil
}

// Unsubscribe unsubscribes from a topic
func (t *MemoryTransport) Unsubscribe(ctx context.Context, topic string) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if ch, exists := t.channels[topic]; exists {
		close(ch)
		delete(t.channels, topic)
	}

	return nil
}

// Publish publishes to a topic
func (t *MemoryTransport) Publish(ctx context.Context, topic string, data []byte) error {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if !t.connected {
		return gerror.New(gerror.ErrCodeInternal, "mcp_memory_transport", nil).WithComponent("disconnect").WithOperation("not connected")
	}

	if ch, exists := t.channels[topic]; exists {
		dataCopy := make([]byte, len(data))
		copy(dataCopy, data)

		select {
		case ch <- dataCopy:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		default:
			return gerror.New(gerror.ErrCodeInternal, "mcp_memory_transport", nil).WithComponent("subscribe").WithOperation("channel full")
		}
	}

	return nil
}
