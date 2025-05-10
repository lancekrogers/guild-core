package agent

import (
	"context"
	"strings"
	"time"

	"github.com/blockhead-consulting/Guild/pkg/kanban"
	"github.com/blockhead-consulting/Guild/pkg/memory"
	"github.com/blockhead-consulting/Guild/pkg/objective"
	"github.com/blockhead-consulting/Guild/pkg/providers"
	"github.com/blockhead-consulting/Guild/tools"
)

// AgentStatus represents the status of an agent
type AgentStatus string

const (
	// StatusIdle indicates the agent is not currently working on a task
	StatusIdle AgentStatus = "idle"

	// StatusWorking indicates the agent is working on a task
	StatusWorking AgentStatus = "working"

	// StatusError indicates the agent encountered an error
	StatusError AgentStatus = "error"

	// StatusPaused indicates the agent is paused
	StatusPaused AgentStatus = "paused"
)

// AgentConfig represents the configuration for an agent
type AgentConfig struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Type        string            `json:"type"` // "manager", "worker", etc.
	Provider    providers.ProviderType `json:"provider"`
	Model       string            `json:"model"`
	MaxTokens   int               `json:"max_tokens"`
	Temperature float64           `json:"temperature"`
	MemoryPath  string            `json:"memory_path,omitempty"`
	Tools       []string          `json:"tools,omitempty"` // Tool IDs the agent can use
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// AgentState represents the current state of an agent
type AgentState struct {
	Status      AgentStatus `json:"status"`
	CurrentTask string      `json:"current_task,omitempty"` // Task ID
	LastError   string      `json:"last_error,omitempty"`
	StartedAt   time.Time   `json:"started_at,omitempty"`
	UpdatedAt   time.Time   `json:"updated_at"`
	Memory      []string    `json:"memory,omitempty"` // Memory chain IDs
}

// GuildArtisan defines the interface for autonomous agents in the Guild system
type GuildArtisan interface {
	// ID returns the agent's unique identifier
	ID() string

	// Name returns the agent's human-readable name
	Name() string

	// Type returns the agent's type
	Type() string

	// Status returns the agent's current status
	Status() AgentStatus

	// CommissionWork assigns a task to the artisan
	CommissionWork(ctx context.Context, task *kanban.Task) error

	// CraftSolution runs the artisan's execution cycle
	CraftSolution(ctx context.Context) error

	// Stop gracefully stops the agent's execution
	Stop(ctx context.Context) error

	// CleanSlate resets the artisan to its initial state
	CleanSlate(ctx context.Context) error

	// SaveState saves the artisan's current state
	SaveState(ctx context.Context) error

	// GetAvailableTools returns the list of tools available to the agent
	GetAvailableTools() []tools.Tool

	// GetConfig returns the agent's configuration
	GetConfig() *AgentConfig

	// GetState returns the agent's current state
	GetState() *AgentState

	// GetMemoryManager returns the agent's memory manager
	GetMemoryManager() memory.ChainManager
}

// GuildMember provides a common implementation for all guild artisans
type GuildMember struct {
	config        *AgentConfig
	state         *AgentState
	llmClient     providers.LLMClient
	memoryManager memory.ChainManager
	toolRegistry  *tools.ToolRegistry
	currentTask   *kanban.Task
	objectiveMgr  *objective.Manager
	costManager   *CostManager
}

// NewGuildMember creates a new guild member with the given configuration
func NewGuildMember(
	config *AgentConfig,
	llmClient providers.LLMClient,
	memoryManager memory.ChainManager,
	toolRegistry *tools.ToolRegistry,
	objectiveMgr *objective.Manager,
) *GuildMember {
	state := &AgentState{
		Status:    StatusIdle,
		UpdatedAt: time.Now().UTC(),
	}

	// Initialize cost manager
	costManager := NewCostManager(CostUnitUSD)

	// Set default model costs if model is specified
	if config.Model != "" {
		// Set cost based on model type - simplified estimate
		switch {
		case strings.Contains(config.Model, "gpt-4") || strings.Contains(config.Model, "claude-3-opus"):
			costManager.SetModelCost(config.Model, 0.00003) // $0.03 per 1000 tokens
		case strings.Contains(config.Model, "gpt-3.5") || strings.Contains(config.Model, "claude-3-sonnet"):
			costManager.SetModelCost(config.Model, 0.000002) // $0.002 per 1000 tokens
		default:
			costManager.SetModelCost(config.Model, 0.000001) // Default cost
		}
	}

	return &GuildMember{
		config:        config,
		state:         state,
		llmClient:     llmClient,
		memoryManager: memoryManager,
		toolRegistry:  toolRegistry,
		objectiveMgr:  objectiveMgr,
		costManager:   costManager,
	}
}

// ID returns the agent's unique identifier
func (a *GuildMember) ID() string {
	return a.config.ID
}

// Name returns the agent's human-readable name
func (a *GuildMember) Name() string {
	return a.config.Name
}

// Type returns the agent's type
func (a *GuildMember) Type() string {
	return a.config.Type
}

// Status returns the agent's current status
func (a *GuildMember) Status() AgentStatus {
	return a.state.Status
}

// CommissionWork assigns a task to the guild member
func (a *GuildMember) CommissionWork(ctx context.Context, task *kanban.Task) error {
	if a.state.Status == StatusWorking {
		return ErrAgentBusy
	}

	a.currentTask = task
	a.state.CurrentTask = task.ID
	a.state.Status = StatusWorking
	a.state.StartedAt = time.Now().UTC()
	a.state.UpdatedAt = time.Now().UTC()

	return a.SaveState(ctx)
}

// GetAvailableTools returns the list of tools available to the guild member
func (a *GuildMember) GetAvailableTools() []tools.Tool {
	var availableTools []tools.Tool

	// Get all tools from the registry
	allTools := a.toolRegistry.ListTools()

	// If no specific tools are configured, return all tools
	if len(a.config.Tools) == 0 {
		return allTools
	}

	// Otherwise, filter tools based on configuration
	for _, tool := range allTools {
		for _, allowedTool := range a.config.Tools {
			if tool.Name() == allowedTool {
				availableTools = append(availableTools, tool)
				break
			}
		}
	}

	return availableTools
}

// GetConfig returns the guild member's configuration
func (a *GuildMember) GetConfig() *AgentConfig {
	return a.config
}

// GetState returns the guild member's current state
func (a *GuildMember) GetState() *AgentState {
	return a.state
}

// GetMemoryManager returns the guild member's memory manager
func (a *GuildMember) GetMemoryManager() memory.ChainManager {
	return a.memoryManager
}

// InscribeState saves the guild member's current state
// This method should be implemented by concrete member types
func (a *GuildMember) SaveState(ctx context.Context) error {
	// Update timestamp
	a.state.UpdatedAt = time.Now().UTC()
	
	// The actual saving logic should be implemented by concrete agents
	return nil
}

// CleanSlate resets the guild member to its initial state
func (a *GuildMember) CleanSlate(ctx context.Context) error {
	a.state = &AgentState{
		Status:    StatusIdle,
		UpdatedAt: time.Now().UTC(),
	}
	a.currentTask = nil
	
	return a.SaveState(ctx)
}

// ErrAgentBusy is returned when trying to assign a task to a busy agent
var ErrAgentBusy = AgentError{Message: "agent is busy with another task"}

// AgentError represents an error from the agent system
type AgentError struct {
	Message string
}

// Error implements the error interface
func (e AgentError) Error() string {
	return e.Message
}