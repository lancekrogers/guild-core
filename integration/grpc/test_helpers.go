package grpc

import (
	"sync"

	guildgrpc "github.com/guild-ventures/guild-core/pkg/grpc"
)

// mockEventBus implements a simple EventBus for testing
type mockEventBus struct {
	mu     sync.Mutex
	events []interface{}
}

func newMockEventBus() *mockEventBus {
	return &mockEventBus{
		events: make([]interface{}, 0),
	}
}

func (m *mockEventBus) Publish(event interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.events = append(m.events, event)
}

func (m *mockEventBus) Subscribe(eventType string, handler func(event interface{})) {
	// For testing, we don't need real subscription
}

// Ensure mockEventBus implements EventBus
var _ guildgrpc.EventBus = (*mockEventBus)(nil)