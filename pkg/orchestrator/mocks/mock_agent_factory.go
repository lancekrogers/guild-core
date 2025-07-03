// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package mocks

import (
	"sync"

	"github.com/lancekrogers/guild/pkg/agents/core"
)

// MockAgentFactory is a mock implementation of the core.Factory
type MockAgentFactory struct {
	agents      map[string]*MockAgent
	createError error
	mu          sync.Mutex
}

// NewMockAgentFactory creates a new mock agent factory
func NewMockAgentFactory() *MockAgentFactory {
	return &MockAgentFactory{
		agents: make(map[string]*MockAgent),
	}
}

// CreateAgent creates a mock agent
func (m *MockAgentFactory) CreateAgent(agentType, name string, options ...interface{}) (core.Agent, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.createError != nil {
		return nil, m.createError
	}

	// Create a unique ID
	id := name + "-" + agentType

	// Create a new mock agent
	mockAgent := NewMockAgent(id, name)

	// Store the agent
	m.agents[id] = mockAgent

	return mockAgent, nil
}

// RegisterAgent registers a mock agent
func (m *MockAgentFactory) RegisterAgent(mockAgent *MockAgent) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.agents[mockAgent.GetID()] = mockAgent
}

// GetAgent returns a registered mock agent
func (m *MockAgentFactory) GetAgent(id string) (*MockAgent, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	agent, exists := m.agents[id]
	return agent, exists
}

// SetCreateError sets the error to be returned by CreateAgent
func (m *MockAgentFactory) SetCreateError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.createError = err
}

// Reset resets the factory
func (m *MockAgentFactory) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.agents = make(map[string]*MockAgent)
	m.createError = nil
}
