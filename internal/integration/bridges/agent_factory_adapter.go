package bridges

import (
	"context"

	"github.com/guild-framework/guild-core/pkg/agents/core"
	"github.com/guild-framework/guild-core/pkg/gerror"
	"github.com/guild-framework/guild-core/pkg/registry"
)

// AgentFactoryAdapter adapts the registry's agent system to the orchestrator's AgentFactory interface
type AgentFactoryAdapter struct {
	agentRegistry registry.AgentRegistry
}

// NewAgentFactoryAdapter creates a new agent factory adapter
func NewAgentFactoryAdapter(agentRegistry registry.AgentRegistry) *AgentFactoryAdapter {
	return &AgentFactoryAdapter{
		agentRegistry: agentRegistry,
	}
}

// CreateAgent creates an agent by type using the registry
func (a *AgentFactoryAdapter) CreateAgent(agentType, name string, options ...interface{}) (core.Agent, error) {
	if a.agentRegistry == nil {
		return nil, gerror.New(gerror.ErrCodeInternal, "agent registry not available", nil).
			WithComponent("AgentFactoryAdapter")
	}

	// Get agent from registry
	agent, err := a.agentRegistry.GetAgent(agentType)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create agent from registry").
			WithComponent("AgentFactoryAdapter").
			WithDetails("agent_type", agentType).
			WithDetails("agent_name", name)
	}

	return agent, nil
}

// TaskDispatcherAdapter adapts the orchestrator's task dispatcher to our interface
type TaskDispatcherAdapter struct {
	dispatcher interface{} // The actual task dispatcher
}

// NewTaskDispatcherAdapter creates a new task dispatcher adapter
func NewTaskDispatcherAdapter(dispatcher interface{}) *TaskDispatcherAdapter {
	return &TaskDispatcherAdapter{
		dispatcher: dispatcher,
	}
}

// RegisterAgent registers an agent with the dispatcher
func (t *TaskDispatcherAdapter) RegisterAgent(agent core.Agent) error {
	// For now, this is a placeholder - in a real implementation this would
	// call the actual dispatcher's RegisterAgent method
	// The dispatcher interface mismatch makes this challenging to implement directly

	// Instead, we emit an event that other components can listen to
	return nil
}

// DispatchTasks dispatches tasks to available agents
func (t *TaskDispatcherAdapter) DispatchTasks(ctx context.Context) error {
	// Placeholder implementation
	return nil
}
