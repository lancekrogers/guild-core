package bridges

import (
	"context"

	"github.com/lancekrogers/guild/pkg/agents/core"
	"github.com/lancekrogers/guild/pkg/gerror"
	"github.com/lancekrogers/guild/pkg/registry"
)

// ServiceAdapter wraps the service registry to provide campaign and orchestration components
type ServiceAdapter struct {
	serviceRegistry ServiceRegistryInterface
}

// ServiceRegistryInterface provides access to services
type ServiceRegistryInterface interface {
	Get(serviceName string) (interface{}, error)
}

// ServiceRegistryAdapter adapts the actual service registry to our interface
type ServiceRegistryAdapter struct {
	registry interface{} // actual service registry
}

// NewServiceRegistryAdapter creates an adapter for the service registry
func NewServiceRegistryAdapter(registry interface{}) ServiceRegistryInterface {
	return &ServiceRegistryAdapter{registry: registry}
}

// Get retrieves a service by name (adapted from concrete registry)
func (a *ServiceRegistryAdapter) Get(serviceName string) (interface{}, error) {
	// For now, return nil as we don't have the actual service implementations yet
	// This would be replaced with actual service retrieval logic
	return nil, gerror.New(gerror.ErrCodeNotFound, "service not found", nil).
		WithComponent("ServiceRegistryAdapter").
		WithDetails("service_name", serviceName)
}

// NewServiceAdapter creates a new service adapter
func NewServiceAdapter(serviceRegistry ServiceRegistryInterface) *ServiceAdapter {
	return &ServiceAdapter{
		serviceRegistry: serviceRegistry,
	}
}

// GetCampaignManager returns the campaign manager from the service registry
func (s *ServiceAdapter) GetCampaignManager() (CampaignManager, error) {
	service, err := s.serviceRegistry.Get("campaign-service")
	if err != nil {
		return nil, err
	}

	// Adapter pattern to convert the service to our interface
	return &campaignManagerAdapter{service: service}, nil
}

// GetTaskDispatcher returns the task dispatcher from the service registry
func (s *ServiceAdapter) GetTaskDispatcher() (TaskDispatcher, error) {
	service, err := s.serviceRegistry.Get("orchestrator-service")
	if err != nil {
		return nil, err
	}

	// Adapter pattern to convert the service to our interface
	return &taskDispatcherAdapter{service: service}, nil
}

// GetAgentRegistry returns the agent registry from the service registry
func (s *ServiceAdapter) GetAgentRegistry() (AgentRegistry, error) {
	service, err := s.serviceRegistry.Get("agent-manager-service")
	if err != nil {
		return nil, err
	}

	// Adapter pattern to convert the service to our interface
	return &agentRegistryAdapter{service: service}, nil
}

// campaignManagerAdapter adapts a service to the CampaignManager interface
type campaignManagerAdapter struct {
	service interface{}
}

// Get retrieves a campaign by ID (implements CampaignManager interface)
func (c *campaignManagerAdapter) Get(ctx context.Context, id string) (*Campaign, error) {
	// For now, return a placeholder
	// In a real implementation, we would call the actual campaign service method
	return &Campaign{
		ID:          id,
		Name:        "Campaign " + id,
		Status:      "active",
		Commissions: []string{}, // Would be populated from actual service
	}, nil
}

// MarkReady marks a campaign as ready (implements CampaignManager interface)
func (c *campaignManagerAdapter) MarkReady(ctx context.Context, id string) error {
	// For now, this is a no-op
	// In a real implementation, we would call the actual campaign service method
	return nil
}

// taskDispatcherAdapter adapts a service to the TaskDispatcher interface
type taskDispatcherAdapter struct {
	service interface{}
}

// DispatchTasks dispatches tasks to available agents (implements TaskDispatcher interface)
func (t *taskDispatcherAdapter) DispatchTasks(ctx context.Context) error {
	// For now, this is a no-op
	// In a real implementation, we would call the actual orchestrator service method
	return nil
}

// RegisterAgent registers an agent with the dispatcher (implements TaskDispatcher interface)
func (t *taskDispatcherAdapter) RegisterAgent(agent core.Agent) error {
	// For now, this is a no-op
	// In a real implementation, we would call the actual orchestrator service method
	return nil
}

// agentRegistryAdapter adapts a service to the AgentRegistry interface
type agentRegistryAdapter struct {
	service interface{}
}

// GetRegisteredAgents returns registered agents (implements AgentRegistry interface)
func (a *agentRegistryAdapter) GetRegisteredAgents() []AgentConfig {
	// For now, return a placeholder
	// In a real implementation, we would call the actual agent manager service method
	return []AgentConfig{
		{
			ID:           "agent-1",
			Name:         "Default Agent",
			Type:         "worker",
			Provider:     "anthropic",
			Model:        "claude-3",
			Capabilities: []string{"reasoning", "task-execution"},
		},
	}
}

// WireOrchestratorCampaignBridge wires the orchestrator campaign bridge with actual services
func WireOrchestratorCampaignBridge(
	bridge *OrchestratorCampaignBridge,
	serviceRegistry ServiceRegistryInterface,
) error {
	adapter := NewServiceAdapter(serviceRegistry)

	// Get campaign manager
	campaignMgr, err := adapter.GetCampaignManager()
	if err != nil {
		// Log but don't fail - the bridge will emit events instead
	} else {
		bridge.campaignManager = campaignMgr
	}

	// Get task dispatcher
	taskDispatcher, err := adapter.GetTaskDispatcher()
	if err != nil {
		// Log but don't fail - the bridge will emit events instead
	} else {
		bridge.taskDispatcher = taskDispatcher
	}

	// Get agent registry
	agentRegistry, err := adapter.GetAgentRegistry()
	if err != nil {
		// Log but don't fail - the bridge will emit events instead
	} else {
		bridge.agentRegistry = agentRegistry
	}

	return nil
}

// WireAgentRegistrationBridge wires the agent registration bridge with actual services
func WireAgentRegistrationBridge(
	bridge *AgentRegistrationBridge,
	serviceRegistry ServiceRegistryInterface,
	componentRegistry interface{}, // registry.ComponentRegistry
) error {
	// Get agent registry from component registry
	// This is a simplified approach - in reality we'd need proper interface adapters

	// For now, create adapters that provide the required interfaces
	// These would be replaced with actual service integration

	// Create agent factory adapter
	if compReg, ok := componentRegistry.(interface{ Agents() interface{} }); ok {
		agentReg := compReg.Agents()
		if agentReg != nil {
			agentFactory := NewAgentFactoryAdapter(agentReg.(registry.AgentRegistry))
			bridge.AgentFactory = agentFactory
			bridge.AgentRegistry = agentReg.(registry.AgentRegistry)
		}
	}

	// Create task dispatcher adapter
	taskDispatcher := NewTaskDispatcherAdapter(nil) // Placeholder
	bridge.TaskDispatcher = taskDispatcher

	return nil
}
