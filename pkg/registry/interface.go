// Package registry provides component registration and discovery for the Guild framework.
// It implements the registry pattern for dynamic component management.
package registry

import (
	"context"
	"time"

	"github.com/guild-ventures/guild-core/pkg/gerror"
	"github.com/guild-ventures/guild-core/pkg/memory"
	"github.com/guild-ventures/guild-core/pkg/memory/vector"
	"github.com/guild-ventures/guild-core/pkg/prompts/layered"
	"github.com/guild-ventures/guild-core/pkg/providers"
	"github.com/guild-ventures/guild-core/tools"
)

// ComponentRegistry is the main registry that manages all component types.
// It provides access to specialized registries for different component types.
type ComponentRegistry interface {
	// Agents returns the agent registry for managing agent types
	Agents() AgentRegistry

	// Tools returns the tool registry for managing tools
	Tools() ToolRegistry

	// Providers returns the provider registry for managing LLM providers
	Providers() ProviderRegistry

	// Memory returns the memory registry for managing memory components
	Memory() MemoryRegistry

	// Project returns the project registry for managing project contexts
	Project() ProjectRegistry

	// Prompts returns the prompt registry for managing prompt templates
	Prompts() *PromptRegistry

	// GetPromptManager returns the configured layered prompt manager
	GetPromptManager() (LayeredPromptManager, error)

	// Storage returns the storage registry for managing storage backends
	Storage() StorageRegistry

	// Orchestrator returns the orchestrator registry for managing orchestrator components
	Orchestrator() interface{}

	// Initialize sets up all registries with the provided configuration
	Initialize(ctx context.Context, config Config) error

	// Shutdown cleanly shuts down all registries and their components
	Shutdown(ctx context.Context) error

	// Cost-based selection methods (convenience methods for cost-aware orchestration)
	// GetAgentsByCost returns agents with cost magnitude <= maxCost, sorted by cost
	GetAgentsByCost(maxCost int) []AgentInfo

	// GetCheapestAgentByCapability returns the lowest-cost agent with the given capability
	GetCheapestAgentByCapability(capability string) (*AgentInfo, error)

	// GetToolsByCost returns tools with cost magnitude <= maxCost, sorted by cost
	GetToolsByCost(maxCost int) []ToolInfo

	// GetCheapestToolByCapability returns the lowest-cost tool with the given capability
	GetCheapestToolByCapability(capability string) (*ToolInfo, error)

	// GetAgentsByCapability returns all agents that have the specified capability
	GetAgentsByCapability(capability string) []AgentInfo
}

// AgentRegistry manages agent types and their factory functions.
type AgentRegistry interface {
	// RegisterAgentType registers a new agent type with its factory function
	RegisterAgentType(name string, factory AgentFactory) error

	// GetAgent creates an agent instance of the specified type
	GetAgent(agentType string) (Agent, error)

	// ListAgentTypes returns all registered agent types
	ListAgentTypes() []string

	// HasAgentType checks if an agent type is registered
	HasAgentType(agentType string) bool

	// Cost-based selection methods
	// GetAgentsByCost returns agents with cost magnitude <= maxCost, sorted by cost
	GetAgentsByCost(maxCost int) []AgentInfo

	// GetCheapestAgentByCapability returns the lowest-cost agent with the given capability
	GetCheapestAgentByCapability(capability string) (*AgentInfo, error)

	// GetAgentsByCapability returns all agents that have the specified capability
	GetAgentsByCapability(capability string) []AgentInfo

	// RegisterGuildAgent registers a configured agent from guild config
	RegisterGuildAgent(config GuildAgentConfig) error

	// GetRegisteredAgents returns all registered agent configurations
	GetRegisteredAgents() []GuildAgentConfig
}

// ToolRegistry manages tool registration and discovery.
type ToolRegistry interface {
	// RegisterTool registers a tool with the registry
	RegisterTool(name string, tool Tool) error

	// GetTool retrieves a registered tool by name
	GetTool(name string) (Tool, error)

	// ListTools returns all registered tool names
	ListTools() []string

	// GetToolsByCapability returns tools that have a specific capability
	GetToolsByCapability(capability string) []Tool

	// HasTool checks if a tool is registered
	HasTool(name string) bool

	// Cost-based tool selection methods
	// GetToolsByCost returns tools with cost magnitude <= maxCost, sorted by cost
	GetToolsByCost(maxCost int) []ToolInfo

	// GetCheapestToolByCapability returns the lowest-cost tool with the given capability
	GetCheapestToolByCapability(capability string) (*ToolInfo, error)

	// RegisterToolWithCost registers a tool with cost information
	RegisterToolWithCost(name string, tool Tool, costMagnitude int, capabilities []string) error

	// GetToolCost returns the cost for using a specific tool
	GetToolCost(toolName string) float64

	// SetToolCost sets the cost for using a specific tool
	SetToolCost(toolName string, cost float64)
}

// ProviderRegistry manages LLM provider registration and selection.
type ProviderRegistry interface {
	// RegisterProvider registers an LLM provider
	RegisterProvider(name string, provider Provider) error

	// GetProvider retrieves a provider by name
	GetProvider(name string) (Provider, error)

	// Get is an alias for GetProvider for backward compatibility
	Get(name string) (Provider, error)

	// GetDefaultProvider returns the configured default provider
	GetDefaultProvider() (Provider, error)

	// SetDefaultProvider sets the default provider
	SetDefaultProvider(name string) error

	// ListProviders returns all registered provider names
	ListProviders() []string

	// HasProvider checks if a provider is registered
	HasProvider(name string) bool
}

// MemoryRegistry manages memory system components.
type MemoryRegistry interface {
	// RegisterMemoryStore registers a memory store implementation
	RegisterMemoryStore(name string, store MemoryStore) error

	// GetMemoryStore retrieves a memory store by name
	GetMemoryStore(name string) (MemoryStore, error)

	// RegisterVectorStore registers a vector store implementation
	RegisterVectorStore(name string, store VectorStore) error

	// GetVectorStore retrieves a vector store by name
	GetVectorStore(name string) (VectorStore, error)

	// RegisterChainManager registers a chain manager implementation
	RegisterChainManager(name string, manager ChainManager) error

	// GetChainManager retrieves a chain manager by name
	GetChainManager(name string) (ChainManager, error)

	// GetDefaultMemoryStore returns the configured default memory store
	GetDefaultMemoryStore() (MemoryStore, error)

	// GetDefaultVectorStore returns the configured default vector store
	GetDefaultVectorStore() (VectorStore, error)

	// GetDefaultChainManager returns the configured default chain manager
	GetDefaultChainManager() (ChainManager, error)

	// ListMemoryStores returns all registered memory store names
	ListMemoryStores() []string

	// ListVectorStores returns all registered vector store names
	ListVectorStores() []string

	// ListChainManagers returns all registered chain manager names
	ListChainManagers() []string
}

// StorageRegistry manages storage backend implementations
type StorageRegistry interface {
	// RegisterTaskRepository registers a task repository implementation
	RegisterTaskRepository(repo TaskRepository) error

	// GetTaskRepository retrieves the registered task repository
	GetTaskRepository() TaskRepository

	// RegisterCampaignRepository registers a campaign repository implementation
	RegisterCampaignRepository(repo CampaignRepository) error

	// GetCampaignRepository retrieves the registered campaign repository
	GetCampaignRepository() CampaignRepository

	// RegisterCommissionRepository registers a commission repository implementation
	RegisterCommissionRepository(repo CommissionRepository) error

	// GetCommissionRepository retrieves the registered commission repository
	GetCommissionRepository() CommissionRepository

	// RegisterAgentRepository registers an agent repository implementation
	RegisterAgentRepository(repo AgentRepository) error

	// GetAgentRepository retrieves the registered agent repository
	GetAgentRepository() AgentRepository

	// RegisterPromptChainRepository registers a prompt chain repository implementation
	RegisterPromptChainRepository(repo PromptChainRepository) error

	// GetPromptChainRepository retrieves the registered prompt chain repository
	GetPromptChainRepository() PromptChainRepository

	// GetMemoryStore returns the configured memory store adapter
	GetMemoryStore() MemoryStore

	// Kanban-specific repository interfaces that handle interface{} parameters
	// These are used by the kanban package to work with SQLite storage
	GetBoardRepository() KanbanBoardRepository
	GetKanbanTaskRepository() KanbanTaskRepository
	GetKanbanCampaignRepository() KanbanCampaignRepository
	GetKanbanCommissionRepository() KanbanCommissionRepository
}

// ProjectRegistry manages project detection and context.
type ProjectRegistry interface {
	// GetProjectManager returns the project manager instance
	GetProjectManager() ProjectManager

	// SetProjectManager sets the project manager instance
	SetProjectManager(manager ProjectManager) error

	// GetCurrentContext returns the current project context
	GetCurrentContext(ctx context.Context) (*ProjectContext, error)

	// WithProjectContext returns a new context with project information
	WithProjectContext(ctx context.Context) (context.Context, error)
}

// Define minimal interfaces to avoid import cycles
// Agent interface is defined in agent_registry.go to avoid duplication

// Type aliases to use existing interfaces
type Tool = tools.Tool

type Provider = providers.LLMClient

type MemoryStore = memory.Store

type VectorStore = vector.VectorStore

type ChainManager = memory.ChainManager

type LayeredPromptManager = layered.LayeredManager

// Forward declarations for storage repositories to avoid import cycles
type TaskRepository interface {
	CreateTask(ctx context.Context, task *StorageTask) error
	GetTask(ctx context.Context, id string) (*StorageTask, error)
	UpdateTask(ctx context.Context, task *StorageTask) error
	DeleteTask(ctx context.Context, id string) error
	ListTasks(ctx context.Context) ([]*StorageTask, error)
	ListTasksByStatus(ctx context.Context, status string) ([]*StorageTask, error)
	ListTasksByCommission(ctx context.Context, commissionID string) ([]*StorageTask, error)
	ListTasksForKanban(ctx context.Context, commissionID string) ([]*StorageTask, error)
	AssignTask(ctx context.Context, taskID, agentID string) error
	UpdateTaskStatus(ctx context.Context, taskID, status string) error
	UpdateTaskColumn(ctx context.Context, taskID, column string) error
	RecordTaskEvent(ctx context.Context, event *TaskEvent) error
	GetTaskHistory(ctx context.Context, taskID string) ([]*TaskEvent, error)
	GetAgentWorkload(ctx context.Context) ([]*AgentWorkload, error)
}

type CampaignRepository interface {
	CreateCampaign(ctx context.Context, campaign *Campaign) error
	GetCampaign(ctx context.Context, id string) (*Campaign, error)
	UpdateCampaignStatus(ctx context.Context, id, status string) error
	DeleteCampaign(ctx context.Context, id string) error
	ListCampaigns(ctx context.Context) ([]*Campaign, error)
}

type CommissionRepository interface {
	CreateCommission(ctx context.Context, commission *Commission) error
	GetCommission(ctx context.Context, id string) (*Commission, error)
	UpdateCommissionStatus(ctx context.Context, id, status string) error
	DeleteCommission(ctx context.Context, id string) error
	ListCommissionsByCampaign(ctx context.Context, campaignID string) ([]*Commission, error)
}

type AgentRepository interface {
	CreateAgent(ctx context.Context, agent *StorageAgent) error
	GetAgent(ctx context.Context, id string) (*StorageAgent, error)
	UpdateAgent(ctx context.Context, agent *StorageAgent) error
	DeleteAgent(ctx context.Context, id string) error
	ListAgents(ctx context.Context) ([]*StorageAgent, error)
	ListAgentsByType(ctx context.Context, agentType string) ([]*StorageAgent, error)
}

type PromptChainRepository interface {
	CreateChain(ctx context.Context, chain *PromptChain) error
	GetChain(ctx context.Context, id string) (*PromptChain, error)
	AddMessage(ctx context.Context, chainID string, message *PromptChainMessage) error
	GetChainsByAgent(ctx context.Context, agentID string) ([]*PromptChain, error)
	GetChainsByTask(ctx context.Context, taskID string) ([]*PromptChain, error)
	DeleteChain(ctx context.Context, id string) error
}

// Storage model forward declarations
type StorageTask struct {
	ID              string                 `json:"id"`
	CommissionID    string                 `json:"commission_id"`
	AssignedAgentID *string                `json:"assigned_agent_id,omitempty"`
	Title           string                 `json:"title"`
	Description     *string                `json:"description,omitempty"`
	Status          string                 `json:"status"`
	Column          string                 `json:"column"`
	StoryPoints     int32                  `json:"story_points"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt       time.Time              `json:"created_at"`
	UpdatedAt       time.Time              `json:"updated_at"`
	AgentName       *string                `json:"agent_name,omitempty"`
	AgentType       *string                `json:"agent_type,omitempty"`
}

type Campaign struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Commission struct {
	ID          string                 `json:"id"`
	CampaignID  string                 `json:"campaign_id"`
	Title       string                 `json:"title"`
	Description *string                `json:"description,omitempty"`
	Domain      *string                `json:"domain,omitempty"`
	Context     map[string]interface{} `json:"context,omitempty"`
	Status      string                 `json:"status"`
	CreatedAt   time.Time              `json:"created_at"`
}

type StorageAgent struct {
	ID            string                 `json:"id"`
	Name          string                 `json:"name"`
	Type          string                 `json:"type"`
	Provider      *string                `json:"provider,omitempty"`
	Model         *string                `json:"model,omitempty"`
	Capabilities  map[string]interface{} `json:"capabilities,omitempty"`
	Tools         map[string]interface{} `json:"tools,omitempty"`
	CostMagnitude int32                  `json:"cost_magnitude"`
	CreatedAt     time.Time              `json:"created_at"`
}

type TaskEvent struct {
	ID        int64     `json:"id"`
	TaskID    string    `json:"task_id"`
	AgentID   *string   `json:"agent_id,omitempty"`
	EventType string    `json:"event_type"`
	OldValue  *string   `json:"old_value,omitempty"`
	NewValue  *string   `json:"new_value,omitempty"`
	Reason    *string   `json:"reason,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

type AgentWorkload struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	TaskCount   int64  `json:"task_count"`
	ActiveTasks int64  `json:"active_tasks"`
}

type PromptChain struct {
	ID        string                `json:"id"`
	AgentID   string                `json:"agent_id"`
	TaskID    *string               `json:"task_id,omitempty"`
	CreatedAt time.Time             `json:"created_at"`
	UpdatedAt time.Time             `json:"updated_at"`
	Messages  []*PromptChainMessage `json:"messages,omitempty"`
}

type PromptChainMessage struct {
	ID         int64     `json:"id"`
	ChainID    string    `json:"chain_id"`
	Role       string    `json:"role"`
	Content    string    `json:"content"`
	Name       *string   `json:"name,omitempty"`
	Timestamp  time.Time `json:"timestamp"`
	TokenUsage int32     `json:"token_usage"`
}

// Forward declarations for orchestrator interfaces to avoid import cycles
type CommissionTaskPlanner interface {
	PlanFromRefinedCommission(ctx context.Context, refined interface{}, guildConfig interface{}) ([]interface{}, error)
	AssignTasksToArtisans(ctx context.Context, tasks []interface{}, guild interface{}) error
}

type EventBus interface {
	Publish(event interface{})
	Subscribe(eventType string, handler interface{}) error
	Unsubscribe(eventType string, handler interface{}) error
}

// ProjectManager provides project management capabilities
type ProjectManager interface {
	// FindProjectRoot finds the project root from a starting path
	FindProjectRoot(startPath string) (string, error)
	// IsInitialized checks if a project exists at the given path
	IsInitialized(path string) bool
	// GetContext returns project context from current directory
	GetContext() (*ProjectContext, error)
	// GetContextFromPath returns project context from a specific path
	GetContextFromPath(path string) (*ProjectContext, error)
	// Initialize creates a new project structure
	Initialize(path string) error
}

// ProjectContext represents a project's context
type ProjectContext interface {
	GetRootPath() string
	GetGuildPath() string
	GetCorpusPath() string
	GetEmbeddingsPath() string
	GetConfigPath() string
	GetAgentsPath() string
	GetObjectivesPath() string
}

// Factory function types for component creation

// Configuration types
type Config struct {
	Agents    AgentConfigYaml `yaml:"agents"`
	Tools     ToolConfig      `yaml:"tools"`
	Providers ProviderConfig  `yaml:"providers"`
	Memory    MemoryConfig    `yaml:"memory"`
}

type AgentConfigYaml struct {
	DefaultType string                 `yaml:"default_type"`
	Types       map[string]interface{} `yaml:"types"`
}

type ToolConfig struct {
	EnabledTools []string               `yaml:"enabled_tools"`
	Settings     map[string]interface{} `yaml:"settings"`
}

type ProviderConfig struct {
	DefaultProvider string                 `yaml:"default_provider"`
	Providers       map[string]interface{} `yaml:"providers"`
}

type MemoryConfig struct {
	DefaultMemoryStore string                 `yaml:"default_memory_store"`
	DefaultVectorStore string                 `yaml:"default_vector_store"`
	Stores             map[string]interface{} `yaml:"stores"`
}

// AgentInfo is defined in agent_registry.go to avoid duplication

// ToolInfo contains tool metadata for cost-based selection
type ToolInfo struct {
	Name          string   `json:"name"`
	Capabilities  []string `json:"capabilities"`
	CostMagnitude int      `json:"cost_magnitude"`
	Available     bool     `json:"available"`
	Tool          Tool     `json:"-"` // The actual tool instance
}

// GuildAgentConfig is defined in agent_registry.go to avoid duplication

// Data types used by components
type Task struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"`
	Description string                 `json:"description"`
	Input       map[string]interface{} `json:"input"`
}

type Result struct {
	Success bool                   `json:"success"`
	Output  map[string]interface{} `json:"output"`
	Error   string                 `json:"error,omitempty"`
}

type ToolInput struct {
	Parameters map[string]interface{} `json:"parameters"`
}

type ToolOutput struct {
	Result interface{} `json:"result"`
	Error  string      `json:"error,omitempty"`
}

type CompletionRequest struct {
	Messages []Message              `json:"messages"`
	Model    string                 `json:"model"`
	Options  map[string]interface{} `json:"options,omitempty"`
}

type CompletionResponse struct {
	Content string                 `json:"content"`
	Usage   map[string]interface{} `json:"usage,omitempty"`
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type VectorMatch struct {
	ID       string                 `json:"id"`
	Score    float32                `json:"score"`
	Metadata map[string]interface{} `json:"metadata"`
}

// Kanban-specific repository interfaces that handle interface{} parameters
// These bridge the gap between kanban's interface{} expectations and storage's typed interfaces

type KanbanTaskRepository interface {
	CreateTask(ctx context.Context, task interface{}) error
	UpdateTask(ctx context.Context, task interface{}) error
	DeleteTask(ctx context.Context, id string) error
	ListTasksByBoard(ctx context.Context, boardID string) ([]interface{}, error)
	RecordTaskEvent(ctx context.Context, event interface{}) error
}

type KanbanBoardRepository interface {
	CreateBoard(ctx context.Context, board interface{}) error
	GetBoard(ctx context.Context, id string) (interface{}, error)
	UpdateBoard(ctx context.Context, board interface{}) error
	DeleteBoard(ctx context.Context, id string) error
	ListBoards(ctx context.Context) ([]interface{}, error)
}

type KanbanCampaignRepository interface {
	CreateCampaign(ctx context.Context, campaign interface{}) error
}

type KanbanCommissionRepository interface {
	CreateCommission(ctx context.Context, commission interface{}) error
	GetCommission(ctx context.Context, id string) (interface{}, error)
}

// Common errors
var (
	ErrComponentNotFound      = gerror.New(gerror.ErrCodeNotFound, "component not found", nil)
	ErrComponentExists        = gerror.New(gerror.ErrCodeAlreadyExists, "component already exists", nil)
	ErrInvalidConfiguration   = gerror.New(gerror.ErrCodeInvalidFormat, "invalid configuration", nil)
	ErrRegistryNotInitialized = gerror.New(gerror.ErrCodeInternal, "registry not initialized", nil)
)
