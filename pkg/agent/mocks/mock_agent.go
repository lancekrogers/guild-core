package mocks

import (
	"context"
	
	"github.com/guild-ventures/guild-core/pkg/memory"
	"github.com/guild-ventures/guild-core/pkg/commission"
	"github.com/guild-ventures/guild-core/pkg/providers"
	"github.com/guild-ventures/guild-core/pkg/registry"
	"github.com/guild-ventures/guild-core/pkg/tools"
)

// MockAgent implements the agent.Agent interface for testing
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

// MockGuildArtisan implements the agent.GuildArtisan interface for testing
type MockGuildArtisan struct {
	MockAgent
	ToolRegistry     *tools.ToolRegistry
	CommissionManager *commission.Manager
	LLMClient        providers.LLMClient
	MemoryManager    memory.ChainManager
}

// NewMockGuildArtisan creates a new mock guild artisan
func NewMockGuildArtisan(id, name string) *MockGuildArtisan {
	return &MockGuildArtisan{
		MockAgent:        *NewMockAgent(id, name),
		ToolRegistry:     registry.NewToolRegistry().(*registry.DefaultToolRegistry).GetUnderlyingRegistry(),
		CommissionManager: nil, // Will be set by test if needed
		LLMClient:        NewMockLLMClient(),
		MemoryManager:    nil, // Will be set by test if needed
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