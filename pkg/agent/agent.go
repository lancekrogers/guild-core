package agent

import (
	"context"
	"fmt"
	"strings"

	"github.com/guild-ventures/guild-core/pkg/commission"
	"github.com/guild-ventures/guild-core/pkg/gerror"
	"github.com/guild-ventures/guild-core/pkg/memory"
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
	GetToolRegistry() tools.Registry

	// GetCommissionManager returns the commission manager
	GetCommissionManager() commission.CommissionManager

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
	ToolRegistry   tools.Registry
	CommissionManager commission.CommissionManager
	CostManager    CostManagerInterface

	// Context metadata
	capabilities []string
	description  string
}

// newWorkerAgent creates a new worker agent (private constructor)
func newWorkerAgent(id, name string, llmClient providers.LLMClient,
	memoryManager memory.ChainManager,
	toolRegistry tools.Registry,
	commissionManager commission.CommissionManager,
	costManager CostManagerInterface) *WorkerAgent {

	return &WorkerAgent{
		ID:                id,
		Name:              name,
		LLMClient:         llmClient,
		MemoryManager:     memoryManager,
		ToolRegistry:      toolRegistry,
		CommissionManager: commissionManager,
		CostManager:       costManager,
	}
}

// Execute runs a task with full tool support
func (a *WorkerAgent) Execute(ctx context.Context, request string) (string, error) {
	// If we have a cost-aware implementation, use it
	if a.LLMClient != nil && a.CostManager != nil {
		return a.CostAwareExecute(ctx, request)
	}

	// Otherwise, execute with tools if available
	if a.LLMClient == nil {
		return "", gerror.New(gerror.ErrCodeValidation, "no LLM client configured", nil).
			WithComponent("agent").
			WithOperation("Execute").
			WithDetails("agent_id", a.ID)
	}

	// If we have tools available, create a task executor for tool-enabled execution
	if a.ToolRegistry != nil {
		return a.executeWithTools(ctx, request)
	}

	// Fall back to simple LLM execution without tools
	response, err := a.LLMClient.Complete(ctx, request)
	if err != nil {
		return "", gerror.Wrap(err, gerror.ErrCodeProvider, "LLM completion failed").
			WithComponent("agent").
			WithOperation("Execute").
			WithDetails("agent_id", a.ID)
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
func (a *WorkerAgent) GetToolRegistry() tools.Registry {
	return a.ToolRegistry
}

// GetCommissionManager returns the commission manager
func (a *WorkerAgent) GetCommissionManager() commission.CommissionManager {
	return a.CommissionManager
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

// executeWithTools executes a task with tool awareness and basic tool execution
func (a *WorkerAgent) executeWithTools(ctx context.Context, request string) (string, error) {
	// Get available tools for context
	var toolContext string
	var availableTools []string

	if a.ToolRegistry != nil {
		availableTools = a.ToolRegistry.ListTools()
		if len(availableTools) > 0 {
			toolDescriptions := make([]string, 0, len(availableTools))
			for _, toolName := range availableTools {
				tool, err := a.ToolRegistry.GetTool(toolName)
				if err == nil {
					toolDescriptions = append(toolDescriptions, fmt.Sprintf("- %s: %s", toolName, tool.Description()))
				}
			}

			if len(toolDescriptions) > 0 {
				toolContext = "\n\nAvailable tools:\n" + strings.Join(toolDescriptions, "\n")
				toolContext += "\n\nYou can reference these tools in your response and I can execute them if needed."
			}
		}
	}

	// Create enhanced prompt with tool context
	enhancedRequest := request + toolContext

	// Execute with LLM
	response, err := a.LLMClient.Complete(ctx, enhancedRequest)
	if err != nil {
		return "", gerror.Wrap(err, gerror.ErrCodeProvider, "LLM completion failed with tool context").
			WithComponent("agent").
			WithOperation("executeWithTools").
			WithDetails("agent_id", a.ID)
	}

	// TODO: Parse the response for tool calls and execute them
	// For now, we're just providing tool awareness to the LLM
	// Future enhancement: Parse response for tool execution requests and execute them

	return response, nil
}

// ManagerAgent is a coordinator agent
type ManagerAgent struct {
	WorkerAgent
}

// newManagerAgent creates a new manager agent (private constructor)
func newManagerAgent(id, name string, llmClient providers.LLMClient,
	memoryManager memory.ChainManager,
	toolRegistry tools.Registry,
	commissionManager commission.CommissionManager,
	costManager CostManagerInterface) *ManagerAgent {

	worker := newWorkerAgent(id, name, llmClient, memoryManager, toolRegistry, commissionManager, costManager)

	return &ManagerAgent{
		WorkerAgent: *worker,
	}
}
