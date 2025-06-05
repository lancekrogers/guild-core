package agent

import (
	"context"
	"fmt"

	"github.com/guild-ventures/guild-core/pkg/memory"
	"github.com/guild-ventures/guild-core/pkg/commission"
	"github.com/guild-ventures/guild-core/pkg/providers"
	"github.com/guild-ventures/guild-core/pkg/tools"
)

// Agent is the interface for all Guild agents
type Agent interface {
	// Execute runs a task
	Execute(ctx context.Context, request string) (string, error)
	
	// GetID returns the agent's ID
	GetID() string
	
	// GetName returns the agent's name
	GetName() string
}

// GuildArtisan is the primary agent interface
type GuildArtisan interface {
	Agent
	
	// GetToolRegistry returns the tool registry
	GetToolRegistry() *tools.ToolRegistry
	
	// GetObjectiveManager returns the objective manager
	GetObjectiveManager() *objective.Manager
	
	// GetLLMClient returns the LLM client
	GetLLMClient() providers.LLMClient
	
	// GetMemoryManager returns the memory manager
	GetMemoryManager() memory.ChainManager
}

// WorkerAgent is a standard worker agent
type WorkerAgent struct {
	ID             string
	Name           string
	LLMClient      providers.LLMClient
	MemoryManager  memory.ChainManager
	ToolRegistry   *tools.ToolRegistry
	ObjectiveManager *objective.Manager
	CostManager    *CostManager
	
	// Context metadata
	capabilities []string
	description  string
}

// NewWorkerAgent creates a new worker agent
func NewWorkerAgent(id, name string, llmClient providers.LLMClient, 
	memoryManager memory.ChainManager, 
	toolRegistry *tools.ToolRegistry, 
	objectiveManager *objective.Manager) *WorkerAgent {
	
	return &WorkerAgent{
		ID:              id,
		Name:            name,
		LLMClient:       llmClient,
		MemoryManager:   memoryManager,
		ToolRegistry:    toolRegistry,
		ObjectiveManager: objectiveManager,
		CostManager:     NewCostManager(),
	}
}

// Execute runs a task
func (a *WorkerAgent) Execute(ctx context.Context, request string) (string, error) {
	// If we have a cost-aware implementation, use it
	if a.LLMClient != nil && a.CostManager != nil {
		return a.CostAwareExecute(ctx, request)
	}
	
	// Otherwise, simple execution
	if a.LLMClient == nil {
		return "", fmt.Errorf("no LLM client configured")
	}
	
	// Call the LLM
	response, err := a.LLMClient.Complete(ctx, request)
	if err != nil {
		return "", fmt.Errorf("LLM completion failed: %w", err)
	}
	
	return response, nil
}

// GetID returns the agent's ID
func (a *WorkerAgent) GetID() string {
	return a.ID
}

// GetName returns the agent's name
func (a *WorkerAgent) GetName() string {
	return a.Name
}

// GetToolRegistry returns the tool registry
func (a *WorkerAgent) GetToolRegistry() *tools.ToolRegistry {
	return a.ToolRegistry
}

// GetObjectiveManager returns the objective manager
func (a *WorkerAgent) GetObjectiveManager() *objective.Manager {
	return a.ObjectiveManager
}

// GetLLMClient returns the LLM client
func (a *WorkerAgent) GetLLMClient() providers.LLMClient {
	return a.LLMClient
}

// GetMemoryManager returns the memory manager
func (a *WorkerAgent) GetMemoryManager() memory.ChainManager {
	return a.MemoryManager
}

// SetCapabilities sets the agent's capabilities
func (a *WorkerAgent) SetCapabilities(capabilities []string) {
	a.capabilities = capabilities
}

// GetCapabilities returns the agent's capabilities
func (a *WorkerAgent) GetCapabilities() []string {
	return a.capabilities
}

// SetDescription sets the agent's description
func (a *WorkerAgent) SetDescription(description string) {
	a.description = description
}

// GetDescription returns the agent's description
func (a *WorkerAgent) GetDescription() string {
	return a.description
}

// HasCapability checks if the agent has a specific capability
func (a *WorkerAgent) HasCapability(capability string) bool {
	for _, cap := range a.capabilities {
		if cap == capability {
			return true
		}
	}
	return false
}

// SetCostBudget sets the budget for a specific cost type
func (a *WorkerAgent) SetCostBudget(costType CostType, amount float64) {
	a.CostManager.SetBudget(costType, amount)
}

// GetCostReport returns a report of all costs incurred by the agent
func (a *WorkerAgent) GetCostReport() map[string]interface{} {
	return a.CostManager.GetCostReport()
}

// ManagerAgent is a coordinator agent
type ManagerAgent struct {
	WorkerAgent
}

// NewManagerAgent creates a new manager agent
func NewManagerAgent(id, name string, llmClient providers.LLMClient, 
	memoryManager memory.ChainManager, 
	toolRegistry *tools.ToolRegistry, 
	objectiveManager *objective.Manager) *ManagerAgent {
	
	worker := NewWorkerAgent(id, name, llmClient, memoryManager, toolRegistry, objectiveManager)
	
	return &ManagerAgent{
		WorkerAgent: *worker,
	}
}