package mocks

import (
	"context"
	"sync"

	"github.com/guild-ventures/guild-core/pkg/agent"
)

// MockAgent is a mock implementation of the agent.Agent interface
type MockAgent struct {
	id           string
	name         string
	mu           sync.Mutex
	executeFunc  func(ctx context.Context, request string) (string, error)
}

// NewMockAgent creates a new mock agent
func NewMockAgent(id, name string) *MockAgent {
	return &MockAgent{
		id:   id,
		name: name,
	}
}

// Execute runs a task
func (m *MockAgent) Execute(ctx context.Context, request string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if m.executeFunc != nil {
		return m.executeFunc(ctx, request)
	}
	
	return "mock response: " + request, nil
}

// GetID returns the agent's ID
func (m *MockAgent) GetID() string {
	return m.id
}

// GetName returns the agent's name
func (m *MockAgent) GetName() string {
	return m.name
}

// SetExecuteFunc sets the function to be called by Execute
func (m *MockAgent) SetExecuteFunc(f func(ctx context.Context, request string) (string, error)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.executeFunc = f
}

// Ensure MockAgent implements agent.Agent interface
var _ agent.Agent = (*MockAgent)(nil)