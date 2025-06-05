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

// SQLiteStorageRegistry implements StorageRegistry for SQLite storage
type SQLiteStorageRegistry struct {
	registry storage.StorageRegistry
	memoryStore MemoryStore
	
	// Kanban adapters to bridge interface{} expectations
	taskAdapter       *storage.KanbanTaskRepositoryAdapter
	boardAdapter      *storage.KanbanBoardRepositoryAdapter
	campaignAdapter   *storage.KanbanCampaignRepositoryAdapter
	commissionAdapter *storage.KanbanCommissionRepositoryAdapter
}

// promptChainRepositoryAdapter adapts storage.PromptChainRepository to registry.PromptChainRepository
type promptChainRepositoryAdapter struct {
	repo storage.PromptChainRepository
}

func (a *promptChainRepositoryAdapter) CreateChain(ctx context.Context, chain *PromptChain) error {
	// Convert registry.PromptChain to storage.PromptChain
	storageChain := &storage.PromptChain{
		ID:        chain.ID,
		AgentID:   chain.AgentID,
		TaskID:    chain.TaskID,
		CreatedAt: chain.CreatedAt,
		UpdatedAt: chain.UpdatedAt,
	}
	return a.repo.CreateChain(ctx, storageChain)
}

func (a *promptChainRepositoryAdapter) GetChain(ctx context.Context, id string) (*PromptChain, error) {
	storageChain, err := a.repo.GetChain(ctx, id)
	if err != nil {
		return nil, err
	}
	
	// Convert storage.PromptChain to registry.PromptChain
	chain := &PromptChain{
		ID:        storageChain.ID,
		AgentID:   storageChain.AgentID,
		TaskID:    storageChain.TaskID,
		CreatedAt: storageChain.CreatedAt,
		UpdatedAt: storageChain.UpdatedAt,
		Messages:  make([]*PromptChainMessage, 0, len(storageChain.Messages)),
	}
	
	// Convert messages
	for _, msg := range storageChain.Messages {
		chain.Messages = append(chain.Messages, &PromptChainMessage{
			ID:         msg.ID,
			ChainID:    msg.ChainID,
			Role:       msg.Role,
			Content:    msg.Content,
			Name:       msg.Name,
			Timestamp:  msg.Timestamp,
			TokenUsage: msg.TokenUsage,
		})
	}
	
	return chain, nil
}

func (a *promptChainRepositoryAdapter) AddMessage(ctx context.Context, chainID string, message *PromptChainMessage) error {
	// Convert registry.PromptChainMessage to storage.PromptChainMessage
	storageMsg := &storage.PromptChainMessage{
		ID:         message.ID,
		ChainID:    message.ChainID,
		Role:       message.Role,
		Content:    message.Content,
		Name:       message.Name,
		Timestamp:  message.Timestamp,
		TokenUsage: message.TokenUsage,
	}
	return a.repo.AddMessage(ctx, chainID, storageMsg)
}

func (a *promptChainRepositoryAdapter) GetChainsByAgent(ctx context.Context, agentID string) ([]*PromptChain, error) {
	storageChains, err := a.repo.GetChainsByAgent(ctx, agentID)
	if err != nil {
		return nil, err
	}
	
	chains := make([]*PromptChain, 0, len(storageChains))
	for _, sc := range storageChains {
		chains = append(chains, &PromptChain{
			ID:        sc.ID,
			AgentID:   sc.AgentID,
			TaskID:    sc.TaskID,
			CreatedAt: sc.CreatedAt,
			UpdatedAt: sc.UpdatedAt,
		})
	}
	return chains, nil
}

func (a *promptChainRepositoryAdapter) GetChainsByTask(ctx context.Context, taskID string) ([]*PromptChain, error) {
	storageChains, err := a.repo.GetChainsByTask(ctx, taskID)
	if err != nil {
		return nil, err
	}
	
	chains := make([]*PromptChain, 0, len(storageChains))
	for _, sc := range storageChains {
		chains = append(chains, &PromptChain{
			ID:        sc.ID,
			AgentID:   sc.AgentID,
			TaskID:    sc.TaskID,
			CreatedAt: sc.CreatedAt,
			UpdatedAt: sc.UpdatedAt,
		})
	}
	return chains, nil
}

func (a *promptChainRepositoryAdapter) DeleteChain(ctx context.Context, id string) error {
	return a.repo.DeleteChain(ctx, id)
}

func (s *SQLiteStorageRegistry) RegisterTaskRepository(repo TaskRepository) error {
	// Not needed for SQLite - repositories are created internally
	return nil
}

func (s *SQLiteStorageRegistry) GetTaskRepository() TaskRepository {
	// Return nil for the interface - kanban uses the adapter via interface calls
	return nil
}

func (s *SQLiteStorageRegistry) RegisterCampaignRepository(repo CampaignRepository) error {
	// Not needed for SQLite - repositories are created internally
	return nil
}

func (s *SQLiteStorageRegistry) GetCampaignRepository() CampaignRepository {
	// For now, return nil - the kanban package uses the storage directly via type assertions
	return nil
}

func (s *SQLiteStorageRegistry) RegisterCommissionRepository(repo CommissionRepository) error {
	// Not needed for SQLite - repositories are created internally
	return nil
}

func (s *SQLiteStorageRegistry) GetCommissionRepository() CommissionRepository {
	// For now, return nil - the kanban package uses the storage directly via type assertions
	return nil
}

func (s *SQLiteStorageRegistry) RegisterAgentRepository(repo AgentRepository) error {
	// Not needed for SQLite - repositories are created internally
	return nil
}

func (s *SQLiteStorageRegistry) GetAgentRepository() AgentRepository {
	// For now, return nil - components should use type assertions to get the actual storage repos
	return nil
}

func (s *SQLiteStorageRegistry) RegisterPromptChainRepository(repo PromptChainRepository) error {
	// Not needed for SQLite - repositories are created internally
	return nil
}

func (s *SQLiteStorageRegistry) GetPromptChainRepository() PromptChainRepository {
	// Get the prompt chain repository from the underlying storage registry
	if s.registry != nil {
		storageRepo := s.registry.GetPromptChainRepository()
		if storageRepo != nil {
			// Wrap the storage repository with an adapter
			return &promptChainRepositoryAdapter{repo: storageRepo}
		}
	}
	return nil
}

func (s *SQLiteStorageRegistry) GetMemoryStore() MemoryStore {
	return s.memoryStore
}

func (s *SQLiteStorageRegistry) SetMemoryStore(store MemoryStore) {
	s.memoryStore = store
}

// GetStorageRegistry returns the underlying storage.StorageRegistry for components that need it
func (s *SQLiteStorageRegistry) GetStorageRegistry() storage.StorageRegistry {
	return s.registry
}

// Kanban repository adapter methods
func (s *SQLiteStorageRegistry) GetBoardRepository() KanbanBoardRepository {
	return s.boardAdapter
}

func (s *SQLiteStorageRegistry) GetKanbanTaskRepository() KanbanTaskRepository {
	return s.taskAdapter
}

func (s *SQLiteStorageRegistry) GetKanbanCampaignRepository() KanbanCampaignRepository {
	return s.campaignAdapter
}

func (s *SQLiteStorageRegistry) GetKanbanCommissionRepository() KanbanCommissionRepository {
	return s.commissionAdapter
}

// SetRegistry sets the underlying storage registry (for testing)
func (s *SQLiteStorageRegistry) SetRegistry(registry storage.StorageRegistry) {
	s.registry = registry
}

// SetAdapters sets the kanban adapters (for testing)
func (s *SQLiteStorageRegistry) SetAdapters(taskAdapter *storage.KanbanTaskRepositoryAdapter, boardAdapter *storage.KanbanBoardRepositoryAdapter, campaignAdapter *storage.KanbanCampaignRepositoryAdapter, commissionAdapter *storage.KanbanCommissionRepositoryAdapter) {
	s.taskAdapter = taskAdapter
	s.boardAdapter = boardAdapter
	s.campaignAdapter = campaignAdapter
	s.commissionAdapter = commissionAdapter
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
	// Cast to storage.StorageRegistry first, then wrap with our own interface
	if sqliteReg, ok := storageReg.(storage.StorageRegistry); ok {
		// Create kanban adapters
		taskAdapter := storage.NewKanbanTaskRepositoryAdapter(sqliteReg.GetTaskRepository())
		boardAdapter := storage.NewKanbanBoardRepositoryAdapter(sqliteReg.GetBoardRepository())
		campaignAdapter := storage.NewKanbanCampaignRepositoryAdapter(sqliteReg.GetCampaignRepository())
		commissionAdapter := storage.NewKanbanCommissionRepositoryAdapter(sqliteReg.GetCommissionRepository())
		
		r.storageRegistry = &SQLiteStorageRegistry{
			registry: sqliteReg,
			memoryStore: memoryStoreAdapter.(MemoryStore),
			taskAdapter: taskAdapter,
			boardAdapter: boardAdapter,
			campaignAdapter: campaignAdapter,
			commissionAdapter: commissionAdapter,
		}
	} else {
		return fmt.Errorf("unexpected storage registry type")
	}
	
	// SQLite storage registry already has the memory store set in the struct above
	
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