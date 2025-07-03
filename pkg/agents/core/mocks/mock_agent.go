// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package mocks

import (
	"context"

	"github.com/lancekrogers/guild/pkg/commission"
	"github.com/lancekrogers/guild/pkg/memory"
	"github.com/lancekrogers/guild/pkg/providers"
	"github.com/lancekrogers/guild/pkg/registry"
	"github.com/lancekrogers/guild/pkg/tools"
)

// MockAgent implements the core.Agent interface for testing
type MockAgent struct {
	ID           string
	Name         string
	ExecuteFunc  func(ctx context.Context, request string) (string, error)
	ExecuteCalls []string
}

// NewMockAgent creates a new mock agent
func NewMockAgent(id, name string) *MockAgent {
	return &MockAgent{
		ID:   id,
		Name: name,
		ExecuteFunc: func(ctx context.Context, request string) (string, error) {
			return "mock response", nil
		},
		ExecuteCalls: make([]string, 0),
	}
}

// Execute implements the Agent interface
func (m *MockAgent) Execute(ctx context.Context, request string) (string, error) {
	m.ExecuteCalls = append(m.ExecuteCalls, request)
	return m.ExecuteFunc(ctx, request)
}

// GetID implements the Agent interface
func (m *MockAgent) GetID() string {
	return m.ID
}

// GetName implements the Agent interface
func (m *MockAgent) GetName() string {
	return m.Name
}

// MockGuildArtisan implements the core.GuildArtisan interface for testing
type MockGuildArtisan struct {
	MockAgent
	ToolRegistry      *tools.ToolRegistry
	CommissionManager *commission.Manager
	LLMClient         providers.LLMClient
	MemoryManager     memory.ChainManager
}

// NewMockGuildArtisan creates a new mock guild artisan
func NewMockGuildArtisan(id, name string) *MockGuildArtisan {
	return &MockGuildArtisan{
		MockAgent:         *NewMockAgent(id, name),
		ToolRegistry:      registry.NewToolRegistry().(*registry.DefaultToolRegistry).GetUnderlyingRegistry(),
		CommissionManager: nil, // Will be set by test if needed
		LLMClient:         NewMockLLMClient(),
		MemoryManager:     nil, // Will be set by test if needed
	}
}

// GetToolRegistry implements the GuildArtisan interface
func (m *MockGuildArtisan) GetToolRegistry() *tools.ToolRegistry {
	return m.ToolRegistry
}

// GetCommissionManager implements the GuildArtisan interface
func (m *MockGuildArtisan) GetCommissionManager() *commission.Manager {
	return m.CommissionManager
}

// GetLLMClient implements the GuildArtisan interface
func (m *MockGuildArtisan) GetLLMClient() providers.LLMClient {
	return m.LLMClient
}

// GetMemoryManager implements the GuildArtisan interface
func (m *MockGuildArtisan) GetMemoryManager() memory.ChainManager {
	return m.MemoryManager
}
