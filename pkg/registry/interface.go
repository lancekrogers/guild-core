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
	
	// Initialize sets up all registries with the provided configuration
	Initialize(ctx context.Context, config Config) error
	
	// Shutdown cleanly shuts down all registries and their components
	Shutdown(ctx context.Context) error
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