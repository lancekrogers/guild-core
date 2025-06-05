package registry

import (
	"context"
	"fmt"
	"path/filepath"
	"sync"

	"github.com/guild-ventures/guild-core/pkg/config"
	"github.com/guild-ventures/guild-core/pkg/providers"
	"github.com/guild-ventures/guild-core/pkg/storage"
)

// DefaultComponentRegistry is the default implementation of ComponentRegistry
type DefaultComponentRegistry struct {
	agentRegistry        AgentRegistry
	toolRegistry         ToolRegistry
	providerRegistry     ProviderRegistry
	memoryRegistry       MemoryRegistry
	projectRegistry      ProjectRegistry
	promptRegistry       *PromptRegistry
	storageRegistry      StorageRegistry
	orchestratorRegistry interface{}
	config              Config
	initialized         bool
	mu                  sync.RWMutex
}

// NewComponentRegistry creates a new ComponentRegistry instance
func NewComponentRegistry() ComponentRegistry {
	return &DefaultComponentRegistry{
		agentRegistry:        NewAgentRegistry(),
		toolRegistry:         NewToolRegistry(),
		providerRegistry:     NewProviderRegistry(),
		memoryRegistry:       NewMemoryRegistry(),
		projectRegistry:      NewProjectRegistry(),
		promptRegistry:       NewPromptRegistry(),
		storageRegistry:      NewStorageRegistry(),
		orchestratorRegistry: nil, // Will be initialized when needed
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

// Project returns the project registry
func (r *DefaultComponentRegistry) Project() ProjectRegistry {
	return r.projectRegistry
}

// Prompts returns the prompt registry
func (r *DefaultComponentRegistry) Prompts() *PromptRegistry {
	return r.promptRegistry
}

// Storage returns the storage registry
func (r *DefaultComponentRegistry) Storage() StorageRegistry {
	return r.storageRegistry
}

// SetStorageRegistry sets the storage registry (used for testing)
func (r *DefaultComponentRegistry) SetStorageRegistry(storageReg StorageRegistry, memoryStore MemoryStore) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.storageRegistry = storageReg
	
	// Also set the memory store in the storage registry if it's the default type
	if defaultStorageReg, ok := r.storageRegistry.(*DefaultStorageRegistry); ok {
		defaultStorageReg.SetMemoryStore(memoryStore)
	}
}

// Orchestrator returns the orchestrator registry
func (r *DefaultComponentRegistry) Orchestrator() interface{} {
	return r.orchestratorRegistry
}

// Initialize sets up all registries with the provided configuration
func (r *DefaultComponentRegistry) Initialize(ctx context.Context, config Config) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.config = config

	// Initialize project context first as other components may depend on it
	ctx, err := r.initializeProject(ctx)
	if err != nil {
		// Project initialization is optional - just log and continue
		// This allows the system to work without a project
		_ = err // Suppress error, project is optional
	}

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

	if err := r.initializeStorage(ctx); err != nil {
		return fmt.Errorf("failed to initialize storage: %w", err)
	}

	if err := r.initializePrompts(ctx); err != nil {
		return fmt.Errorf("failed to initialize prompts: %w", err)
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
		
		// Load agents from guild configuration if available
		if err := r.loadGuildAgents(ctx, agentReg); err != nil {
			// Log warning but don't fail - guild config is optional
			_ = err // Suppress error for now
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
	
	// Register basic tools with cost information
	if toolReg, ok := r.toolRegistry.(*DefaultToolRegistry); ok {
		if err := toolReg.RegisterBasicTools(); err != nil {
			return fmt.Errorf("failed to register basic tools: %w", err)
		}
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

func (r *DefaultComponentRegistry) initializeStorage(ctx context.Context) error {
	// Get project context to load guild configuration
	projectCtx, err := r.projectRegistry.GetCurrentContext(ctx)
	if err != nil {
		// No project context available, use default SQLite backend
		return r.initializeSQLiteStorage(".guild/guild.db")
	}

	// Load guild configuration to determine storage backend
	guildConfig, err := config.LoadGuildConfig((*projectCtx).GetRootPath())
	if err != nil {
		// If guild config fails to load, default to SQLite
		return r.initializeSQLiteStorage(filepath.Join((*projectCtx).GetGuildPath(), "guild.db"))
	}

	// Initialize storage based on configuration
	if guildConfig.IsUsingSQLite() {
		dbPath := filepath.Join((*projectCtx).GetGuildPath(), guildConfig.GetEffectiveSQLitePath())
		// Make path relative to guild directory
		if !filepath.IsAbs(guildConfig.GetEffectiveSQLitePath()) {
			dbPath = filepath.Join((*projectCtx).GetGuildPath(), guildConfig.GetEffectiveSQLitePath())
		} else {
			dbPath = guildConfig.GetEffectiveSQLitePath()
		}
		return r.initializeSQLiteStorage(dbPath)
	} else {
		// BoltDB legacy support
		dbPath := filepath.Join((*projectCtx).GetGuildPath(), guildConfig.GetEffectiveBoltDBPath())
		return r.initializeBoltDBStorage(dbPath)
	}
}

func (r *DefaultComponentRegistry) initializeSQLiteStorage(dbPath string) error {
	ctx := context.Background()
	
	// Initialize SQLite storage using the storage package's initialization function
	storageReg, memoryStoreAdapter, err := storage.InitializeSQLiteStorageForRegistry(ctx, dbPath)
	if err != nil {
		return fmt.Errorf("failed to initialize SQLite storage: %w", err)
	}
	
	// Replace the placeholder storage registry with the real one
	r.storageRegistry = storageReg.(StorageRegistry)
	
	// Register the SQLite store adapter as the default memory store
	// This allows existing components that expect memory.Store to work with SQLite
	if defaultStorageReg, ok := r.storageRegistry.(*DefaultStorageRegistry); ok {
		defaultStorageReg.SetMemoryStore(memoryStoreAdapter.(MemoryStore))
	}
	
	return nil
}

func (r *DefaultComponentRegistry) initializeBoltDBStorage(dbPath string) error {
	// Initialize legacy BoltDB storage
	// This would use the existing memory package implementations
	
	_ = dbPath // Suppress unused variable warning
	
	// TODO: Initialize BoltDB storage for backward compatibility
	return fmt.Errorf("BoltDB storage initialization not yet implemented")
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

func (r *DefaultComponentRegistry) initializeProject(ctx context.Context) (context.Context, error) {
	// Try to add project context to the context
	return r.projectRegistry.WithProjectContext(ctx)
}

func (r *DefaultComponentRegistry) initializePrompts(ctx context.Context) error {
	// Register default prompt provider
	defaultProvider, err := NewDefaultPromptProvider()
	if err != nil {
		return fmt.Errorf("error creating default prompt provider: %w", err)
	}

	if err := r.promptRegistry.Register("default", defaultProvider); err != nil {
		return fmt.Errorf("error registering default prompt provider: %w", err)
	}

	// TODO: Add any other prompt providers from config

	return nil
}

// loadGuildAgents loads agents from guild configuration
func (r *DefaultComponentRegistry) loadGuildAgents(ctx context.Context, agentReg *DefaultAgentRegistry) error {
	// Try to get project context and load guild configuration
	projectCtx, err := r.projectRegistry.GetCurrentContext(ctx)
	if err != nil {
		return fmt.Errorf("failed to get project context: %w", err)
	}
	
	// Load guild configuration
	guildConfig, err := config.LoadGuildConfig((*projectCtx).GetRootPath())
	if err != nil {
		return fmt.Errorf("failed to load guild config: %w", err)
	}
	
	// Register each agent with the registry
	for _, agent := range guildConfig.Agents {
		guildAgent := GuildAgentConfig{
			ID:            agent.ID,
			Name:          agent.Name,
			Type:          agent.Type,
			Provider:      agent.Provider,
			Model:         agent.Model,
			Description:   agent.Description,
			Capabilities:  agent.Capabilities,
			Tools:         agent.Tools,
			MaxTokens:     agent.MaxTokens,
			Temperature:   agent.Temperature,
			CostMagnitude: agent.CostMagnitude,
			ContextWindow: agent.ContextWindow,
			ContextReset:  agent.ContextReset,
			Settings:      agent.Settings,
		}
		
		if err := agentReg.RegisterGuildAgent(guildAgent); err != nil {
			return fmt.Errorf("failed to register agent %s: %w", agent.ID, err)
		}
	}
	
	return nil
}

// GetAgentsByCost provides access to cost-based agent selection
func (r *DefaultComponentRegistry) GetAgentsByCost(maxCost int) []AgentInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	if agentReg, ok := r.agentRegistry.(*DefaultAgentRegistry); ok {
		return agentReg.GetAgentsByCost(maxCost)
	}
	return nil
}

// GetCheapestAgentByCapability provides access to cost-optimal agent selection
func (r *DefaultComponentRegistry) GetCheapestAgentByCapability(capability string) (*AgentInfo, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	if agentReg, ok := r.agentRegistry.(*DefaultAgentRegistry); ok {
		return agentReg.GetCheapestAgentByCapability(capability)
	}
	return nil, fmt.Errorf("agent registry not properly initialized")
}

// GetToolsByCost provides access to cost-based tool selection
func (r *DefaultComponentRegistry) GetToolsByCost(maxCost int) []ToolInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	if toolReg, ok := r.toolRegistry.(*DefaultToolRegistry); ok {
		return toolReg.GetToolsByCost(maxCost)
	}
	return nil
}

// GetCheapestToolByCapability provides access to cost-optimal tool selection
func (r *DefaultComponentRegistry) GetCheapestToolByCapability(capability string) (*ToolInfo, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	if toolReg, ok := r.toolRegistry.(*DefaultToolRegistry); ok {
		return toolReg.GetCheapestToolByCapability(capability)
	}
	return nil, fmt.Errorf("tool registry not properly initialized")
}

// GetAgentsByCapability provides access to capability-based agent selection
func (r *DefaultComponentRegistry) GetAgentsByCapability(capability string) []AgentInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	if agentReg, ok := r.agentRegistry.(*DefaultAgentRegistry); ok {
		return agentReg.GetAgentsByCapability(capability)
	}
	return nil
}