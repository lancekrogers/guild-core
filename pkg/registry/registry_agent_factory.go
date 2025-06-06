package registry

import (
	"context"

	"github.com/guild-ventures/guild-core/pkg/agent"
	"github.com/guild-ventures/guild-core/pkg/gerror"
	"github.com/guild-ventures/guild-core/pkg/tools"
	"github.com/guild-ventures/guild-core/internal/commission"
)

// createAgentFactory creates an agent factory with all required dependencies
func (r *DefaultComponentRegistry) createAgentFactory(ctx context.Context) (AgentFactory, error) {
	// Get default provider for LLM client
	llmClient, err := r.providerRegistry.GetDefaultProvider()
	if err != nil {
		// If no provider is available, create a null factory that returns errors
		return &nullAgentFactory{}, nil
	}
	
	// Get default memory manager
	memoryManager, err := r.memoryRegistry.GetDefaultChainManager()
	if err != nil {
		// If no memory manager is available, create a null factory
		return &nullAgentFactory{}, nil
	}
	
	// Get tool registry - create empty one if none exists
	var toolRegistry tools.Registry
	if r.toolRegistry != nil {
		// Convert to tools.Registry interface
		toolRegistry = &toolRegistryAdapter{registry: r.toolRegistry}
	}
	
	// Get commission manager - this may be nil for now
	var commissionManager commission.CommissionManager
	// TODO: Get from internal/commission package when available
	
	// Get cost manager - create a default one
	costManager := agent.DefaultCostManagerFactory()
	
	// Create the agent factory using the existing DefaultFactoryFactory
	factory := agent.DefaultFactoryFactory(
		llmClient,
		memoryManager,
		toolRegistry,
		commissionManager,
		costManager,
	)
	
	return factory, nil
}

// nullAgentFactory is a placeholder factory that returns errors
type nullAgentFactory struct{}

func (f *nullAgentFactory) CreateAgent(ctx context.Context, id, name string, agentType string) (Agent, error) {
	return nil, gerror.New(gerror.ErrCodeInternal, "agent factory dependencies not available", nil).
		WithComponent("registry").
		WithOperation("CreateAgent")
}

func (f *nullAgentFactory) CreateWorkerAgent(ctx context.Context, id, name string) (Agent, error) {
	return nil, gerror.New(gerror.ErrCodeInternal, "agent factory dependencies not available", nil).
		WithComponent("registry").
		WithOperation("CreateWorkerAgent")
}

func (f *nullAgentFactory) CreateManagerAgent(ctx context.Context, id, name string) (Agent, error) {
	return nil, gerror.New(gerror.ErrCodeInternal, "agent factory dependencies not available", nil).
		WithComponent("registry").
		WithOperation("CreateManagerAgent")
}

// toolRegistryAdapter adapts ToolRegistry interface to tools.Registry
type toolRegistryAdapter struct {
	registry ToolRegistry
}

func (a *toolRegistryAdapter) RegisterTool(tool tools.Tool) error {
	return a.registry.RegisterTool(tool.GetName(), tool)
}

func (a *toolRegistryAdapter) GetTool(name string) (tools.Tool, bool) {
	tool, err := a.registry.GetTool(name)
	return tool, err == nil
}

func (a *toolRegistryAdapter) ListTools() []string {
	return a.registry.ListTools()
}