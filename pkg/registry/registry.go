package registry

import (
	"context"
	"fmt"
	"sync"

	"github.com/guild-ventures/guild-core/pkg/providers"
)

// DefaultComponentRegistry is the default implementation of ComponentRegistry
type DefaultComponentRegistry struct {
	agentRegistry    AgentRegistry
	toolRegistry     ToolRegistry
	providerRegistry ProviderRegistry
	memoryRegistry   MemoryRegistry
	config          Config
	initialized     bool
	mu              sync.RWMutex
}

// NewComponentRegistry creates a new ComponentRegistry instance
func NewComponentRegistry() ComponentRegistry {
	return &DefaultComponentRegistry{
		agentRegistry:    NewAgentRegistry(),
		toolRegistry:     NewToolRegistry(),
		providerRegistry: NewProviderRegistry(),
		memoryRegistry:   NewMemoryRegistry(),
	}
}

// Agents returns the agent registry
func (r *DefaultComponentRegistry) Agents() AgentRegistry {
	return r.agentRegistry
}

// Tools returns the tool registry
func (r *DefaultComponentRegistry) Tools() ToolRegistry {
	return r.toolRegistry
}

// Providers returns the provider registry
func (r *DefaultComponentRegistry) Providers() ProviderRegistry {
	return r.providerRegistry
}

// Memory returns the memory registry
func (r *DefaultComponentRegistry) Memory() MemoryRegistry {
	return r.memoryRegistry
}

// Initialize sets up all registries with the provided configuration
func (r *DefaultComponentRegistry) Initialize(ctx context.Context, config Config) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.config = config

	// Initialize each registry
	if err := r.initializeAgents(ctx); err != nil {
		return fmt.Errorf("failed to initialize agents: %w", err)
	}

	if err := r.initializeTools(ctx); err != nil {
		return fmt.Errorf("failed to initialize tools: %w", err)
	}

	if err := r.initializeProviders(ctx); err != nil {
		return fmt.Errorf("failed to initialize providers: %w", err)
	}

	if err := r.initializeMemory(ctx); err != nil {
		return fmt.Errorf("failed to initialize memory: %w", err)
	}

	r.initialized = true
	return nil
}

// Shutdown cleanly shuts down all registries and their components
func (r *DefaultComponentRegistry) Shutdown(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	var errors []error

	// Shutdown each registry
	if err := r.shutdownMemory(ctx); err != nil {
		errors = append(errors, fmt.Errorf("memory shutdown error: %w", err))
	}

	if err := r.shutdownProviders(ctx); err != nil {
		errors = append(errors, fmt.Errorf("providers shutdown error: %w", err))
	}

	if err := r.shutdownTools(ctx); err != nil {
		errors = append(errors, fmt.Errorf("tools shutdown error: %w", err))
	}

	if err := r.shutdownAgents(ctx); err != nil {
		errors = append(errors, fmt.Errorf("agents shutdown error: %w", err))
	}

	r.initialized = false

	if len(errors) > 0 {
		return fmt.Errorf("shutdown errors: %v", errors)
	}

	return nil
}

func (r *DefaultComponentRegistry) initializeAgents(ctx context.Context) error {
	// Register default agent types
	if agentReg, ok := r.agentRegistry.(*DefaultAgentRegistry); ok {
		// Register worker agent factory
		agentReg.RegisterAgentType("worker", func(config AgentConfig) (Agent, error) {
			// This would create a worker agent - implementation depends on your Agent interface
			// For now, return nil to avoid compilation errors
			return nil, fmt.Errorf("agent creation not yet implemented")
		})

		// Register manager agent factory  
		agentReg.RegisterAgentType("manager", func(config AgentConfig) (Agent, error) {
			// This would create a manager agent
			return nil, fmt.Errorf("agent creation not yet implemented")
		})

		// Set default if configured
		if r.config.Agents.DefaultType != "" {
			agentReg.SetDefaultAgentType(r.config.Agents.DefaultType)
		}
	}

	return nil
}

func (r *DefaultComponentRegistry) initializeTools(ctx context.Context) error {
	// Initialize enabled tools based on configuration
	for _, toolName := range r.config.Tools.EnabledTools {
		// Here you would create and register the actual tool instances
		// This is where you'd integrate with your existing tool implementations
		_ = toolName // Suppress unused variable warning
	}
	return nil
}

func (r *DefaultComponentRegistry) initializeProviders(ctx context.Context) error {
	// Create provider factory
	factory := providers.NewFactory()

	// Register all configured providers
	err := factory.RegisterProvidersWithRegistry(r.providerRegistry, r.config.Providers.Providers)
	if err != nil {
		return fmt.Errorf("failed to register providers: %w", err)
	}

	// Set default provider if configured
	if r.config.Providers.DefaultProvider != "" {
		err := r.providerRegistry.SetDefaultProvider(r.config.Providers.DefaultProvider)
		if err != nil {
			return fmt.Errorf("failed to set default provider: %w", err)
		}
	}

	return nil
}

func (r *DefaultComponentRegistry) initializeMemory(ctx context.Context) error {
	// Initialize memory stores based on configuration
	for storeName, storeConfig := range r.config.Memory.Stores {
		// Here you would create and register memory store instances
		_ = storeName
		_ = storeConfig
	}
	return nil
}

func (r *DefaultComponentRegistry) shutdownAgents(ctx context.Context) error {
	// Shutdown agents if needed
	return nil
}

func (r *DefaultComponentRegistry) shutdownTools(ctx context.Context) error {
	// Shutdown tools if needed
	return nil
}

func (r *DefaultComponentRegistry) shutdownProviders(ctx context.Context) error {
	// Shutdown providers if needed
	return nil
}

func (r *DefaultComponentRegistry) shutdownMemory(ctx context.Context) error {
	// Shutdown memory stores if needed
	return nil
}