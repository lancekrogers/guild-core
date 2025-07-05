// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package grpc

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/lancekrogers/guild/pkg/events"
	guildgrpc "github.com/lancekrogers/guild/pkg/grpc"
	"github.com/lancekrogers/guild/pkg/registry"
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

// newTestEventBus creates a proper EventBusAdapter for testing
func newTestEventBus() guildgrpc.EventBus {
	// Create a unified event bus
	unifiedBus := events.NewMemoryEventBus(events.DefaultEventBusConfig())
	// Wrap it in the adapter
	return guildgrpc.NewEventBusAdapter(unifiedBus)
}

// mockAgent implements a simple Agent for testing
type mockAgent struct {
	id           string
	name         string
	responses    map[string]string
	toolRegistry registry.ToolRegistry
}

func (m *mockAgent) GetID() string {
	return m.id
}

func (m *mockAgent) GetName() string {
	return m.name
}

func (m *mockAgent) GetType() string {
	return "mock"
}

func (m *mockAgent) GetCapabilities() []string {
	return []string{"chat", "test", "tools"}
}

func (m *mockAgent) Execute(ctx context.Context, input string) (string, error) {
	// Check if this is a tool command
	if strings.HasPrefix(input, "/tool ") {
		toolName := strings.TrimPrefix(input, "/tool ")

		// If we have a tool registry, actually execute the tool
		if m.toolRegistry != nil {
			tool, err := m.toolRegistry.GetTool(toolName)
			if err != nil {
				return fmt.Sprintf("Tool not found: %s", toolName), nil
			}

			// Execute the tool with empty input
			result, err := tool.Execute(ctx, "{}")
			if err != nil {
				return fmt.Sprintf("Tool execution failed: %v", err), nil
			}

			return result.Output, nil
		}
	}

	// Simple response based on input
	if response, exists := m.responses[input]; exists {
		return response, nil
	}
	return "Default mock response for: " + input, nil
}

func (m *mockAgent) GetCostProfile() interface{} {
	return nil
}
