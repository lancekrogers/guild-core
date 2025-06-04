// Package registry provides component registration and discovery for the Guild framework.
// It implements the registry pattern for dynamic component management.
package registry

import (
	"context"
	"fmt"
	"time"

	"github.com/guild-ventures/guild-core/pkg/memory"
	"github.com/guild-ventures/guild-core/pkg/memory/vector"
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
}

// ProviderRegistry manages LLM provider registration and selection.
type ProviderRegistry interface {
	// RegisterProvider registers an LLM provider
	RegisterProvider(name string, provider Provider) error
	
	// GetProvider retrieves a provider by name
	GetProvider(name string) (Provider, error)
	
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
	
	// GetDefaultMemoryStore returns the configured default memory store
	GetDefaultMemoryStore() (MemoryStore, error)
	
	// GetDefaultVectorStore returns the configured default vector store
	GetDefaultVectorStore() (VectorStore, error)
	
	// ListMemoryStores returns all registered memory store names
	ListMemoryStores() []string
	
	// ListVectorStores returns all registered vector store names
	ListVectorStores() []string
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
	
	// GetMemoryStore returns the configured memory store adapter
	GetMemoryStore() MemoryStore
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
// Agent represents an AI agent - using minimal interface for now
type Agent interface {
	Execute(ctx context.Context, request string) (string, error)
	GetID() string
	GetName() string
}

// Type aliases to use existing interfaces
type Tool = tools.Tool
type Provider = providers.LLMClient
type MemoryStore = memory.Store
type VectorStore = vector.VectorStore

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
	ID             string                 `json:"id"`
	Name           string                 `json:"name"`
	Type           string                 `json:"type"`
	Provider       *string                `json:"provider,omitempty"`
	Model          *string                `json:"model,omitempty"`
	Capabilities   map[string]interface{} `json:"capabilities,omitempty"`
	Tools          map[string]interface{} `json:"tools,omitempty"`
	CostMagnitude  int32                  `json:"cost_magnitude"`
	CreatedAt      time.Time              `json:"created_at"`
}

type TaskEvent struct {
	ID         int64     `json:"id"`
	TaskID     string    `json:"task_id"`
	AgentID    *string   `json:"agent_id,omitempty"`
	EventType  string    `json:"event_type"`
	OldValue   *string   `json:"old_value,omitempty"`
	NewValue   *string   `json:"new_value,omitempty"`
	Reason     *string   `json:"reason,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
}

type AgentWorkload struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	TaskCount   int64  `json:"task_count"`
	ActiveTasks int64  `json:"active_tasks"`
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
type AgentFactory func(config AgentConfig) (Agent, error)

// Configuration types
type Config struct {
	Agents    AgentConfig    `yaml:"agents"`
	Tools     ToolConfig     `yaml:"tools"`
	Providers ProviderConfig `yaml:"providers"`
	Memory    MemoryConfig   `yaml:"memory"`
}

type AgentConfig struct {
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

// AgentInfo contains agent metadata for cost-based selection
type AgentInfo struct {
	ID            string   `json:"id"`
	Name          string   `json:"name"`
	Type          string   `json:"type"`
	Provider      string   `json:"provider"`
	Model         string   `json:"model"`
	Capabilities  []string `json:"capabilities"`
	Tools         []string `json:"tools"`
	CostMagnitude int      `json:"cost_magnitude"`
	ContextWindow int      `json:"context_window"`
	ContextReset  string   `json:"context_reset"`
	Available     bool     `json:"available"`
}

// ToolInfo contains tool metadata for cost-based selection
type ToolInfo struct {
	Name          string   `json:"name"`
	Capabilities  []string `json:"capabilities"`
	CostMagnitude int      `json:"cost_magnitude"`
	Available     bool     `json:"available"`
	Tool          Tool     `json:"-"` // The actual tool instance
}

// GuildAgentConfig represents an agent from guild configuration
type GuildAgentConfig struct {
	ID            string            `json:"id"`
	Name          string            `json:"name"`
	Type          string            `json:"type"`
	Provider      string            `json:"provider"`
	Model         string            `json:"model"`
	Description   string            `json:"description,omitempty"`
	Capabilities  []string          `json:"capabilities"`
	Tools         []string          `json:"tools,omitempty"`
	MaxTokens     int               `json:"max_tokens,omitempty"`
	Temperature   float64           `json:"temperature,omitempty"`
	CostMagnitude int               `json:"cost_magnitude,omitempty"`
	ContextWindow int               `json:"context_window,omitempty"`
	ContextReset  string            `json:"context_reset,omitempty"`
	Settings      map[string]string `json:"settings,omitempty"`
}

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
	Messages []Message `json:"messages"`
	Model    string    `json:"model"`
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

// Common errors
var (
	ErrComponentNotFound     = fmt.Errorf("component not found")
	ErrComponentExists       = fmt.Errorf("component already exists")
	ErrInvalidConfiguration  = fmt.Errorf("invalid configuration")
	ErrRegistryNotInitialized = fmt.Errorf("registry not initialized")
)