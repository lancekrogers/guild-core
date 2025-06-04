// Package registry provides component registration and discovery for the Guild framework.
// It implements the registry pattern for dynamic component management.
package registry

import (
	"context"
	"fmt"

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