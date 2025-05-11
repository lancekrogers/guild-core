package agent

import (
	"context"

	"github.com/blockhead-consulting/guild/pkg/memory"
	"github.com/blockhead-consulting/guild/pkg/objective"
	"github.com/blockhead-consulting/guild/pkg/providers/interfaces"
	"github.com/blockhead-consulting/guild/pkg/tools"
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
	GetLLMClient() interfaces.LLMClient
	
	// GetMemoryManager returns the memory manager
	GetMemoryManager() memory.ChainManager
}

// WorkerAgent is a standard worker agent
type WorkerAgent struct {
	ID             string
	Name           string
	LLMClient      interfaces.LLMClient
	MemoryManager  memory.ChainManager
	ToolRegistry   *tools.ToolRegistry
	ObjectiveManager *objective.Manager
}

// NewWorkerAgent creates a new worker agent
func NewWorkerAgent(id, name string, llmClient interfaces.LLMClient, 
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
	}
}

// Execute runs a task
func (a *WorkerAgent) Execute(ctx context.Context, request string) (string, error) {
	// Stub implementation
	return "Executed: " + request, nil
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
func (a *WorkerAgent) GetLLMClient() interfaces.LLMClient {
	return a.LLMClient
}

// GetMemoryManager returns the memory manager
func (a *WorkerAgent) GetMemoryManager() memory.ChainManager {
	return a.MemoryManager
}

// ManagerAgent is a coordinator agent
type ManagerAgent struct {
	WorkerAgent
}

// NewManagerAgent creates a new manager agent
func NewManagerAgent(id, name string, llmClient interfaces.LLMClient, 
	memoryManager memory.ChainManager, 
	toolRegistry *tools.ToolRegistry, 
	objectiveManager *objective.Manager) *ManagerAgent {
	
	worker := NewWorkerAgent(id, name, llmClient, memoryManager, toolRegistry, objectiveManager)
	
	return &ManagerAgent{
		WorkerAgent: *worker,
	}
}