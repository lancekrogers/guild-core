package mocks

import (
	"context"

	"github.com/blockhead-consulting/guild/pkg/memory"
	"github.com/stretchr/testify/mock"
)

// MockMemoryManager is a mock implementation of the memory.ChainManager interface.
type MockMemoryManager struct {
	mock.Mock
}

// CreateChain mocks the CreateChain method.
func (m *MockMemoryManager) CreateChain(ctx context.Context, agentID, taskID string) (string, error) {
	args := m.Called(ctx, agentID, taskID)
	return args.String(0), args.Error(1)
}

// GetChain mocks the GetChain method.
func (m *MockMemoryManager) GetChain(ctx context.Context, chainID string) (*memory.PromptChain, error) {
	args := m.Called(ctx, chainID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*memory.PromptChain), args.Error(1)
}

// AddMessage mocks the AddMessage method.
func (m *MockMemoryManager) AddMessage(ctx context.Context, chainID string, message memory.Message) error {
	args := m.Called(ctx, chainID, message)
	return args.Error(0)
}

// GetChainsByAgent mocks the GetChainsByAgent method.
func (m *MockMemoryManager) GetChainsByAgent(ctx context.Context, agentID string) ([]*memory.PromptChain, error) {
	args := m.Called(ctx, agentID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*memory.PromptChain), args.Error(1)
}

// GetChainsByTask mocks the GetChainsByTask method.
func (m *MockMemoryManager) GetChainsByTask(ctx context.Context, taskID string) ([]*memory.PromptChain, error) {
	args := m.Called(ctx, taskID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*memory.PromptChain), args.Error(1)
}

// BuildContext mocks the BuildContext method.
func (m *MockMemoryManager) BuildContext(ctx context.Context, agentID, taskID string, maxTokens int) ([]memory.Message, error) {
	args := m.Called(ctx, agentID, taskID, maxTokens)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]memory.Message), args.Error(1)
}

// DeleteChain mocks the DeleteChain method.
func (m *MockMemoryManager) DeleteChain(ctx context.Context, chainID string) error {
	args := m.Called(ctx, chainID)
	return args.Error(0)
}