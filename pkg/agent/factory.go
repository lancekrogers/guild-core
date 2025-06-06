package agent

import (
	"context"

	"github.com/guild-ventures/guild-core/internal/commission"
	"github.com/guild-ventures/guild-core/pkg/gerror"
	"github.com/guild-ventures/guild-core/pkg/memory"
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
	ToolRegistry     tools.Registry
	CommissionManager commission.CommissionManager
	CostManager      CostManager
}

// newFactory creates a new factory instance (private constructor)
func newFactory(
	llmClient providers.LLMClient,
	memoryManager memory.ChainManager,
	toolRegistry tools.Registry,
	commissionManager commission.CommissionManager,
	costManager CostManager,
) *DefaultFactory {
	return &DefaultFactory{
		LLMClient:        llmClient,
		MemoryManager:    memoryManager,
		ToolRegistry:     toolRegistry,
		CommissionManager: commissionManager,
		CostManager:      costManager,
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
		return nil, gerror.Newf(gerror.ErrCodeValidation, "unknown agent type: %s", agentType).
			WithComponent("agent").
			WithOperation("CreateAgent").
			WithDetails("agent_type", agentType)
	}
}

// CreateWorkerAgent creates a new worker agent
func (f *DefaultFactory) CreateWorkerAgent(ctx context.Context, id, name string) (Agent, error) {
	return newWorkerAgent(id, name, f.LLMClient, f.MemoryManager, f.ToolRegistry, f.CommissionManager, f.CostManager), nil
}

// CreateManagerAgent creates a new manager agent
func (f *DefaultFactory) CreateManagerAgent(ctx context.Context, id, name string) (Agent, error) {
	return newManagerAgent(id, name, f.LLMClient, f.MemoryManager, f.ToolRegistry, f.CommissionManager, f.CostManager), nil
}

// DefaultFactoryFactory creates a factory for registry use
func DefaultFactoryFactory(
	llmClient providers.LLMClient,
	memoryManager memory.ChainManager,
	toolRegistry tools.Registry,
	commissionManager commission.CommissionManager,
	costManager CostManager,
) Factory {
	return newFactory(llmClient, memoryManager, toolRegistry, commissionManager, costManager)
}