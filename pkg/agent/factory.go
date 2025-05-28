package agent

import (
	"context"
	"fmt"

	"github.com/guild-ventures/guild-core/pkg/memory"
	"github.com/guild-ventures/guild-core/pkg/objective"
	"github.com/guild-ventures/guild-core/pkg/providers"
	"github.com/guild-ventures/guild-core/pkg/tools"
)

// Factory creates agent instances
type Factory interface {
	// CreateAgent creates a new agent with the given parameters
	CreateAgent(ctx context.Context, id, name string, agentType string) (Agent, error)
	
	// CreateWorkerAgent creates a new worker agent
	CreateWorkerAgent(ctx context.Context, id, name string) (Agent, error)
	
	// CreateManagerAgent creates a new manager agent
	CreateManagerAgent(ctx context.Context, id, name string) (Agent, error)
}

// DefaultFactory is the default implementation of Factory
type DefaultFactory struct {
	LLMClient        providers.LLMClient
	MemoryManager    memory.ChainManager
	ToolRegistry     *tools.ToolRegistry
	ObjectiveManager *objective.Manager
}

// NewFactory creates a new factory instance
func NewFactory(
	llmClient providers.LLMClient,
	memoryManager memory.ChainManager,
	toolRegistry *tools.ToolRegistry,
	objectiveManager *objective.Manager,
) *DefaultFactory {
	return &DefaultFactory{
		LLMClient:        llmClient,
		MemoryManager:    memoryManager,
		ToolRegistry:     toolRegistry,
		ObjectiveManager: objectiveManager,
	}
}

// CreateAgent creates a new agent with the given parameters
func (f *DefaultFactory) CreateAgent(ctx context.Context, id, name string, agentType string) (Agent, error) {
	switch agentType {
	case "worker":
		return f.CreateWorkerAgent(ctx, id, name)
	case "manager":
		return f.CreateManagerAgent(ctx, id, name)
	default:
		return nil, fmt.Errorf("unknown agent type: %s", agentType)
	}
}

// CreateWorkerAgent creates a new worker agent
func (f *DefaultFactory) CreateWorkerAgent(ctx context.Context, id, name string) (Agent, error) {
	return NewWorkerAgent(id, name, f.LLMClient, f.MemoryManager, f.ToolRegistry, f.ObjectiveManager), nil
}

// CreateManagerAgent creates a new manager agent
func (f *DefaultFactory) CreateManagerAgent(ctx context.Context, id, name string) (Agent, error) {
	return NewManagerAgent(id, name, f.LLMClient, f.MemoryManager, f.ToolRegistry, f.ObjectiveManager), nil
}