package mocks

import (
	"context"

	"github.com/blockhead-consulting/guild/pkg/agent"
	"github.com/blockhead-consulting/guild/pkg/kanban"
	"github.com/blockhead-consulting/guild/pkg/memory"
	"github.com/blockhead-consulting/guild/tools"
	"github.com/stretchr/testify/mock"
)

// MockAgent is a mock implementation of the agent.GuildArtisan interface.
type MockAgent struct {
	mock.Mock
}

// ID mocks the ID method.
func (m *MockAgent) ID() string {
	args := m.Called()
	return args.String(0)
}

// Name mocks the Name method.
func (m *MockAgent) Name() string {
	args := m.Called()
	return args.String(0)
}

// Type mocks the Type method.
func (m *MockAgent) Type() string {
	args := m.Called()
	return args.String(0)
}

// Status mocks the Status method.
func (m *MockAgent) Status() agent.AgentStatus {
	args := m.Called()
	return args.Get(0).(agent.AgentStatus)
}

// CommissionWork mocks the CommissionWork method.
func (m *MockAgent) CommissionWork(ctx context.Context, task *kanban.Task) error {
	args := m.Called(ctx, task)
	return args.Error(0)
}

// CraftSolution mocks the CraftSolution method.
func (m *MockAgent) CraftSolution(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

// Stop mocks the Stop method.
func (m *MockAgent) Stop(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

// CleanSlate mocks the CleanSlate method.
func (m *MockAgent) CleanSlate(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

// SaveState mocks the SaveState method.
func (m *MockAgent) SaveState(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

// GetAvailableTools mocks the GetAvailableTools method.
func (m *MockAgent) GetAvailableTools() []tools.Tool {
	args := m.Called()
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).([]tools.Tool)
}

// GetConfig mocks the GetConfig method.
func (m *MockAgent) GetConfig() *agent.AgentConfig {
	args := m.Called()
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(*agent.AgentConfig)
}

// GetState mocks the GetState method.
func (m *MockAgent) GetState() *agent.AgentState {
	args := m.Called()
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(*agent.AgentState)
}

// GetMemoryManager mocks the GetMemoryManager method.
func (m *MockAgent) GetMemoryManager() memory.ChainManager {
	args := m.Called()
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(memory.ChainManager)
}

// SetCostBudget mocks the SetCostBudget method.
func (m *MockAgent) SetCostBudget(costType agent.CostType, amount float64) {
	m.Called(costType, amount)
}

// GetCostReport mocks the GetCostReport method.
func (m *MockAgent) GetCostReport() map[string]interface{} {
	args := m.Called()
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(map[string]interface{})
}