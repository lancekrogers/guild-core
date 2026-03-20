// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package orchestrator

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/lancekrogers/guild-core/pkg/agents/core"
	"github.com/lancekrogers/guild-core/pkg/agents/core/manager"
	"github.com/lancekrogers/guild-core/pkg/config"
	"github.com/lancekrogers/guild-core/pkg/events"
	"github.com/lancekrogers/guild-core/pkg/gerror"
	"github.com/lancekrogers/guild-core/pkg/kanban"
	"github.com/lancekrogers/guild-core/pkg/observability"
	"github.com/lancekrogers/guild-core/pkg/prompts"
	"github.com/lancekrogers/guild-core/pkg/prompts/layered"
	"github.com/lancekrogers/guild-core/pkg/providers"
	"github.com/lancekrogers/guild-core/pkg/providers/interfaces"
	"github.com/lancekrogers/guild-core/pkg/registry"
	"github.com/lancekrogers/guild-core/pkg/storage"
)

// CommissionIntegrationService coordinates the complete pipeline from commission to kanban tasks
type CommissionIntegrationService struct {
	registry             registry.ComponentRegistry
	commissionRefiner    manager.CommissionRefiner
	commissionPlanner    CommissionTaskPlanner
	kanbanManager        KanbanManager
	commissionRepository registry.CommissionRepository
	eventBus             EventBus
	guildMasterFactory   *manager.DefaultGuildMasterFactory
	taskBridge           *manager.TaskBridge
}

// newCommissionIntegrationService creates a new integration service with full wiring (private constructor)
func newCommissionIntegrationService(ctx context.Context, registry registry.ComponentRegistry) (*CommissionIntegrationService, error) {
	// Check context early
	if err := ctx.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("orchestrator").
			WithOperation("newCommissionIntegrationService")
	}

	service := &CommissionIntegrationService{
		registry: registry,
	}

	// Initialize components from registry
	if err := service.initializeFromRegistry(ctx); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeOrchestration, "failed to initialize from registry").
			WithComponent("orchestrator").
			WithOperation("NewCommissionIntegrationService")
	}

	return service, nil
}

// DefaultCommissionIntegrationServiceFactory creates a commission integration service factory for registry use
func DefaultCommissionIntegrationServiceFactory(ctx context.Context, registry registry.ComponentRegistry) (*CommissionIntegrationService, error) {
	return newCommissionIntegrationService(ctx, registry)
}

// initializeFromRegistry sets up all components from the registry
func (s *CommissionIntegrationService) initializeFromRegistry(ctx context.Context) error {
	// Check context early
	if err := ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("orchestrator").
			WithOperation("initializeFromRegistry")
	}

	// Set up logging
	logger := observability.GetLogger(ctx)
	ctx = observability.WithComponent(ctx, "orchestrator")
	ctx = observability.WithOperation(ctx, "initializeFromRegistry")

	logger.InfoContext(ctx, "Initializing commission integration service from registry")

	// Get providers from registry
	providerRegistry := s.registry.Providers()
	providers := make(map[string]interfaces.AIProvider)

	// Get all available providers
	for _, providerName := range []string{"anthropic", "openai", "ollama", "deepseek", "mock"} {
		provider, err := providerRegistry.GetProvider(providerName)
		if err == nil {
			// Provider is LLMClient, we need to wrap it as AIProvider
			if aiProvider, ok := provider.(interfaces.AIProvider); ok {
				providers[providerName] = aiProvider
			} else {
				// Create a wrapper
				providers[providerName] = &llmClientWrapper{client: provider}
			}
		}
	}

	if len(providers) == 0 {
		return gerror.New(gerror.ErrCodeOrchestration, "no AI providers available in registry", nil).
			WithComponent("orchestrator").
			WithOperation("initializeFromRegistry").
			WithDetails("providersChecked", []string{"anthropic", "openai", "ollama", "deepseek", "mock"})
	}

	// Get prompt manager from registry
	promptRegistryFromReg := s.registry.Prompts()
	var promptManager prompts.Manager

	// Try to get prompt manager from registry first
	if promptRegistryFromReg != nil {
		// Use the global prompt registry
		promptRegistry := prompts.GetRegistry()

		// PromptRegistry doesn't have RegisterPrompt method - using GetDefaultManager instead
		// TODO: Implement proper prompt registration via manager strategies

		// Create a standard prompt manager
		var err error
		promptManager, err = promptRegistry.GetDefaultManager(ctx)
		if err != nil {
			return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create prompt manager").
				WithComponent("orchestrator").
				WithOperation("InitializeCommissionPlanner")
		}
	} else {
		// Fallback to creating a new one
		promptRegistry := prompts.GetRegistry()
		var err error
		promptManager, err = promptRegistry.GetDefaultManager(ctx)
		if err != nil {
			return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create prompt manager").
				WithComponent("orchestrator").
				WithOperation("InitializeCommissionPlanner")
		}
	}

	// Create layered manager adapter if needed
	// Since promptManager is concrete type, we always need the adapter
	layeredManager := &layeredManagerAdapter{Manager: prompts.Manager(promptManager)}

	// Create a component registry for the guild master factory
	componentRegistry := manager.NewComponentRegistry()

	// Create Guild Master factory
	s.guildMasterFactory = manager.NewDefaultGuildMasterFactory(layeredManager, providers, componentRegistry)

	// Create commission refiner
	var err error
	s.commissionRefiner, err = s.guildMasterFactory.CreateCommissionRefinerWithDefaults()
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeOrchestration, "failed to create commission refiner").
			WithComponent("orchestrator").
			WithOperation("initializeFromRegistry")
	}

	// Create kanban manager using SQLite storage via registry
	// First create a custom adapter for kanban that implements their ComponentRegistry interface
	kanbanAdapter := &kanbanRegistryAdapter{registry: s.registry}
	kanbanMgr, err := kanban.NewManagerWithRegistry(ctx, kanbanAdapter)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeOrchestration, "failed to create kanban manager with SQLite storage").
			WithComponent("orchestrator").
			WithOperation("initializeFromRegistry")
	}

	// Create a board using the SQLite-enabled manager
	kanbanBoard, err := kanbanMgr.CreateBoard(ctx, "commission-board", "Board for commission tasks")
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeOrchestration, "failed to create kanban board").
			WithComponent("orchestrator").
			WithOperation("initializeFromRegistry").
			WithDetails("boardName", "commission-board")
	}
	s.kanbanManager = DefaultKanbanManagerFactory(kanbanBoard)

	// Get or create event bus from orchestrator registry
	orchestratorReg, ok := s.registry.Orchestrator().(OrchestratorRegistry)
	if ok && orchestratorReg != nil {
		// Try to get default event bus
		if eb, err := orchestratorReg.GetDefaultEventBus(); err == nil {
			s.eventBus = eb
		} else {
			// Create and register new event bus
			s.eventBus = DefaultEventBusFactory()
			_ = orchestratorReg.RegisterEventBus("default", s.eventBus)
		}
	} else {
		// Fallback: create event bus directly
		s.eventBus = DefaultEventBusFactory()
	}

	// Create intelligent parser for commission planner
	parserConfig := manager.IntelligentParserConfig{
		Mode:          manager.ParserModeAuto,
		ArtisanClient: s.commissionRefiner.(*manager.GuildMasterRefiner).GetArtisanClient(),
		PromptManager: layeredManager,
	}
	intelligentParser := manager.NewIntelligentParser(parserConfig)
	parserAdapter := manager.NewResponseParserAdapter(intelligentParser)

	// Create commission planner - use unified version if available
	// Check if the event bus has a UnifiedEventBus method (duck typing to avoid import cycle)
	if adapter, ok := s.eventBus.(interface{ UnifiedEventBus() interface{} }); ok && adapter != nil {
		// We have a unified event bus available via adapter
		unifiedBus := adapter.UnifiedEventBus()
		// Need to cast to events.EventBus
		if unifiedEventBus, ok := unifiedBus.(events.EventBus); ok {
			s.commissionPlanner = UnifiedCommissionTaskPlannerFactory(s.kanbanManager, parserAdapter, unifiedEventBus)
		} else {
			// Fallback to legacy
			s.commissionPlanner = DefaultCommissionTaskPlannerFactory(s.kanbanManager, parserAdapter, s.eventBus)
		}
	} else {
		// Use legacy commission planner
		s.commissionPlanner = DefaultCommissionTaskPlannerFactory(s.kanbanManager, parserAdapter, s.eventBus)
	}

	// Get commission repository from storage registry
	storageRegistry := s.registry.Storage()
	if storageRegistry == nil {
		return gerror.New(gerror.ErrCodeOrchestration, "storage registry not available for commission repository", nil).
			WithComponent("orchestrator").
			WithOperation("initializeFromRegistry")
	}

	s.commissionRepository = storageRegistry.GetCommissionRepository()
	if s.commissionRepository == nil {
		return gerror.New(gerror.ErrCodeOrchestration, "commission repository not available from storage registry", nil).
			WithComponent("orchestrator").
			WithOperation("initializeFromRegistry")
	}

	// Create adapter to convert between registry.CommissionRepository and core.CommissionRepository
	commissionAdapter := newCommissionRepositoryAdapter(s.commissionRepository)

	// Create task bridge with commission repository
	s.taskBridge = manager.NewTaskBridgeWithCommissions(kanbanBoard, commissionAdapter)

	return nil
}

// SetCommissionRefiner sets the commission refiner (injected via registry or factory)
func (s *CommissionIntegrationService) SetCommissionRefiner(refiner manager.CommissionRefiner) {
	s.commissionRefiner = refiner
}

// SetCommissionPlanner sets the commission planner (injected via registry or factory)
func (s *CommissionIntegrationService) SetCommissionPlanner(planner CommissionTaskPlanner) {
	s.commissionPlanner = planner
}

// SetEventBus sets the event bus (injected via registry or factory)
func (s *CommissionIntegrationService) SetEventBus(eventBus EventBus) {
	s.eventBus = eventBus
}

// SetKanbanManager sets the kanban manager (injected via registry or factory)
func (s *CommissionIntegrationService) SetKanbanManager(kanbanManager KanbanManager) {
	s.kanbanManager = kanbanManager
}

// SetCommissionRepository sets the commission repository (injected via registry)
func (s *CommissionIntegrationService) SetCommissionRepository(repo registry.CommissionRepository) {
	s.commissionRepository = repo
}

// ProcessCommissionToTasks handles the complete pipeline from commission to kanban tasks
func (s *CommissionIntegrationService) ProcessCommissionToTasks(
	ctx context.Context,
	commission manager.Commission,
	guildConfig *config.GuildConfig,
) (*CommissionProcessingResult, error) {
	// Check context early
	if err := ctx.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("orchestrator").
			WithOperation("ProcessCommissionToTasks")
	}

	// Set up logging
	logger := observability.GetLogger(ctx)
	ctx = observability.WithComponent(ctx, "orchestrator")
	ctx = observability.WithOperation(ctx, "ProcessCommissionToTasks")

	// Validate dependencies
	if err := s.validateDependencies(); err != nil {
		return nil, err
	}

	// Add commission context to the request context
	ctx = s.addCommissionContext(ctx, commission)

	// Step 1: Refine the commission using IntelligentParser
	logger.InfoContext(ctx, "Refining commission", "title", commission.Title, "id", commission.ID)
	refined, err := s.commissionRefiner.RefineCommission(ctx, commission)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeTaskFailed, "failed to refine commission").
			WithComponent("orchestrator").
			WithOperation("ProcessCommissionToTasks").
			WithDetails("commissionID", commission.ID)
	}
	logger.InfoContext(ctx, "Commission refined successfully", "files_found", len(refined.Structure.Files))

	// Step 2: Convert refined commission to kanban tasks
	logger.InfoContext(ctx, "Planning tasks from refined commission")
	tasks, err := s.commissionPlanner.PlanFromRefinedCommission(ctx, refined, guildConfig)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeTaskFailed, "failed to plan tasks from commission").
			WithComponent("orchestrator").
			WithOperation("ProcessCommissionToTasks").
			WithDetails("commissionID", commission.ID)
	}
	logger.InfoContext(ctx, "Created tasks from commission", "task_count", len(tasks))

	// Step 3: Use task bridge to create tasks in kanban
	if s.taskBridge != nil {
		logger.InfoContext(ctx, "Creating tasks in kanban system")
		if err := s.taskBridge.CreateTasksFromRefinedCommission(ctx, refined); err != nil {
			logger.WarnContext(ctx, "Failed to create tasks via bridge", "error", err)
			// Don't fail - tasks were already created by planner
		}
	}

	// Step 4: Assign tasks to artisans
	logger.InfoContext(ctx, "Assigning tasks to artisans")
	if err := s.commissionPlanner.AssignTasksToArtisans(ctx, tasks, guildConfig); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeTaskFailed, "failed to assign tasks to artisans").
			WithComponent("orchestrator").
			WithOperation("ProcessCommissionToTasks").
			WithDetails("commissionID", commission.ID)
	}

	// Step 5: Log commission completion with task information
	if s.commissionRepository != nil {
		logger.InfoContext(ctx, "Commission completed", "commission_id", commission.ID, "task_count", len(tasks))
		// TODO: Implement commission metadata updates if needed
	}

	// Step 6: Emit completion event
	s.emitCommissionProcessedEvent(commission, tasks)

	// Step 7: Write refined files if output directory is configured
	if outputDir, ok := ctx.Value("output_dir").(string); ok && outputDir != "" {
		logger.InfoContext(ctx, "Writing refined files", "output_dir", outputDir)
		if err := s.taskBridge.WriteRefinedFiles(refined, outputDir); err != nil {
			logger.WarnContext(ctx, "Failed to write refined files", "error", err)
		}
	}

	return &CommissionProcessingResult{
		Commission:        commission,
		RefinedCommission: refined,
		Tasks:             tasks,
		AssignedArtisans:  s.extractAssignedArtisans(tasks),
	}, nil
}

// ProcessCommissionToTasksByID loads a commission and processes it to kanban tasks
func (s *CommissionIntegrationService) ProcessCommissionToTasksByID(
	ctx context.Context,
	commissionID string,
	guildConfig *config.GuildConfig,
) (*CommissionProcessingResult, error) {
	if s.commissionRepository == nil {
		return nil, gerror.New(gerror.ErrCodeOrchestration, "commission repository not configured", nil).
			WithComponent("orchestrator").
			WithOperation("ProcessCommissionToTasksByID")
	}

	// Load commission from storage
	registryCommission, err := s.commissionRepository.GetCommission(ctx, commissionID)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeNotFound, "failed to load commission").
			WithComponent("orchestrator").
			WithOperation("ProcessCommissionToTasksByID").
			WithDetails("commissionID", commissionID)
	}

	// Convert registry commission to manager commission
	managerCommission := s.registryToManagerCommission(registryCommission)

	// Process commission to tasks
	return s.ProcessCommissionToTasks(ctx, managerCommission, guildConfig)
}

// GetTaskBridge returns the task bridge for external use
func (s *CommissionIntegrationService) GetTaskBridge() *manager.TaskBridge {
	return s.taskBridge
}

// GetGuildMasterFactory returns the guild master factory
func (s *CommissionIntegrationService) GetGuildMasterFactory() *manager.DefaultGuildMasterFactory {
	return s.guildMasterFactory
}

// layeredManagerAdapter adapts a regular Manager to LayeredManager interface
type layeredManagerAdapter struct {
	prompts.Manager
}

// BuildLayeredPrompt assembles a complete layered prompt for an artisan
func (a *layeredManagerAdapter) BuildLayeredPrompt(ctx context.Context, artisanID, sessionID string, turnCtx layered.TurnContext) (*layered.LayeredPrompt, error) {
	// Simple implementation using the basic Manager
	var layers []layered.SystemPrompt
	var compiled strings.Builder

	// Try to get a system prompt based on context
	if turnCtx.CommissionID != "" {
		// Use manager role for commission refinement
		systemPrompt, err := a.GetSystemPrompt(ctx, "manager", "default")
		if err == nil {
			layers = append(layers, layered.SystemPrompt{
				Layer:     layered.LayerRole,
				ArtisanID: artisanID,
				Content:   systemPrompt,
				Version:   1,
				Priority:  100,
				Updated:   time.Now(),
			})
			compiled.WriteString(systemPrompt)
			compiled.WriteString("\n\n")
		}
	}

	// Add context if available
	if turnCtx.Context != nil {
		contextStr, err := a.FormatContext(ctx, turnCtx.Context)
		if err == nil && contextStr != "" {
			layers = append(layers, layered.SystemPrompt{
				Layer:     layered.LayerSession,
				ArtisanID: artisanID,
				SessionID: sessionID,
				Content:   contextStr,
				Version:   1,
				Priority:  50,
				Updated:   time.Now(),
			})
			compiled.WriteString(contextStr)
			compiled.WriteString("\n\n")
		}
	}

	// Add turn instructions
	if len(turnCtx.Instructions) > 0 {
		instructions := strings.Join(turnCtx.Instructions, "\n")
		layers = append(layers, layered.SystemPrompt{
			Layer:     layered.LayerTurn,
			ArtisanID: artisanID,
			SessionID: sessionID,
			Content:   instructions,
			Version:   1,
			Priority:  10,
			Updated:   time.Now(),
		})
		compiled.WriteString(instructions)
	}

	return &layered.LayeredPrompt{
		Layers:      layers,
		Compiled:    compiled.String(),
		TokenCount:  len(compiled.String()) / 4, // Rough estimate
		Truncated:   false,
		CacheKey:    fmt.Sprintf("%s-%s-%d", artisanID, sessionID, time.Now().Unix()),
		ArtisanID:   artisanID,
		SessionID:   sessionID,
		AssembledAt: time.Now(),
	}, nil
}

// GetPromptLayer retrieves a specific prompt layer
func (a *layeredManagerAdapter) GetPromptLayer(ctx context.Context, layer layered.PromptLayer, artisanID, sessionID string) (*layered.SystemPrompt, error) {
	// Simple implementation - return a basic prompt based on layer
	var content string
	var err error

	switch layer {
	case layered.LayerRole:
		content, err = a.GetSystemPrompt(ctx, "manager", "default")
	case layered.LayerDomain:
		content, err = a.GetSystemPrompt(ctx, "manager", "web-app")
	default:
		return nil, gerror.New(gerror.ErrCodeNotFound, "layer not found", nil).
			WithComponent("orchestrator").
			WithOperation("GetPromptLayer")
	}

	if err != nil {
		return nil, err
	}

	return &layered.SystemPrompt{
		Layer:     layer,
		ArtisanID: artisanID,
		SessionID: sessionID,
		Content:   content,
		Version:   1,
		Priority:  100,
		Updated:   time.Now(),
	}, nil
}

// SetPromptLayer sets or updates a specific prompt layer
func (a *layeredManagerAdapter) SetPromptLayer(ctx context.Context, prompt layered.SystemPrompt) error {
	// Not implemented in adapter
	return gerror.New(gerror.ErrCodeInternal, "SetPromptLayer not supported in adapter", nil).
		WithComponent("orchestrator").
		WithOperation("SetPromptLayer")
}

// DeletePromptLayer removes a specific prompt layer
func (a *layeredManagerAdapter) DeletePromptLayer(ctx context.Context, layer layered.PromptLayer, artisanID, sessionID string) error {
	// Not implemented in adapter
	return gerror.New(gerror.ErrCodeInternal, "DeletePromptLayer not supported in adapter", nil).
		WithComponent("orchestrator").
		WithOperation("DeletePromptLayer")
}

// ListPromptLayers returns all layers for an artisan/session
func (a *layeredManagerAdapter) ListPromptLayers(ctx context.Context, artisanID, sessionID string) ([]layered.SystemPrompt, error) {
	// Return empty list
	return []layered.SystemPrompt{}, nil
}

// InvalidateCache clears the layered prompt cache
func (a *layeredManagerAdapter) InvalidateCache(ctx context.Context, artisanID, sessionID string) error {
	// No cache in adapter
	return nil
}

// ClearLayer clears a specific prompt layer (simplified interface)
func (a *layeredManagerAdapter) ClearLayer(layer prompts.PromptLayer) error {
	// Not implemented in adapter
	return gerror.New(gerror.ErrCodeInternal, "ClearLayer not supported in adapter", nil).
		WithComponent("orchestrator").
		WithOperation("ClearLayer")
}

// SetLayer sets content for a specific layer
func (a *layeredManagerAdapter) SetLayer(layer prompts.PromptLayer, content string) error {
	// Not implemented in adapter
	return gerror.New(gerror.ErrCodeInternal, "SetLayer not supported in adapter", nil).
		WithComponent("orchestrator").
		WithOperation("SetLayer")
}

// GetCompiledPrompt compiles all layers into a final prompt
func (a *layeredManagerAdapter) GetCompiledPrompt(ctx context.Context, config prompts.LayerConfig) (string, error) {
	// Simple implementation - just return basic system prompt
	return "You are a Guild Master responsible for task coordination.", nil
}

// GetSystemPrompt retrieves a system prompt (adapter implementation)
func (a *layeredManagerAdapter) GetSystemPrompt(ctx context.Context, role, domain string) (string, error) {
	// Simple implementation for adapter
	return "You are a Guild Master responsible for coordinating task execution and managing artisan workflows.", nil
}

// FormatContext formats context information into a prompt string
func (a *layeredManagerAdapter) FormatContext(ctx context.Context, contextInfo layered.Context) (string, error) {
	// Simple implementation for adapter
	return "Context: Processing commission with task coordination requirements.", nil
}

// GetTemplate retrieves a named template
func (a *layeredManagerAdapter) GetTemplate(ctx context.Context, templateName string) (string, error) {
	// Simple implementation for adapter - templates not supported
	return "", gerror.New(gerror.ErrCodeNotFound, "templates not supported in adapter", nil).
		WithComponent("orchestrator").
		WithOperation("GetTemplate")
}

// ListRoles returns all available roles
func (a *layeredManagerAdapter) ListRoles(ctx context.Context) ([]string, error) {
	// Return default roles
	return []string{"manager", "worker", "specialist"}, nil
}

// ListDomains returns all available domains for a role
func (a *layeredManagerAdapter) ListDomains(ctx context.Context, role string) ([]string, error) {
	// Return default domains
	return []string{"default", "web-app", "microservice", "cli-tool"}, nil
}

// llmClientWrapper wraps registry.Provider (LLMClient) to implement providers.AIProvider
type llmClientWrapper struct {
	client registry.Provider // This is providers.LLMClient
}

// ChatCompletion implements providers.AIProvider
func (w *llmClientWrapper) ChatCompletion(ctx context.Context, req providers.ChatRequest) (*providers.ChatResponse, error) {
	// Convert to the format expected by LLMClient
	// Registry Provider is registry.Provider (providers.LLMClient)
	// We need to convert the request to a simple prompt
	prompt := ""
	for _, msg := range req.Messages {
		if msg.Role == "system" {
			prompt += "System: " + msg.Content + "\n\n"
		} else if msg.Role == "user" {
			prompt += "User: " + msg.Content + "\n\n"
		} else if msg.Role == "assistant" {
			prompt += "Assistant: " + msg.Content + "\n\n"
		}
	}
	prompt += "Assistant: "

	completion, err := w.client.Complete(ctx, prompt)
	if err != nil {
		return nil, err
	}

	// Convert response
	return &providers.ChatResponse{
		ID:    "wrapped-response",
		Model: req.Model,
		Choices: []providers.ChatChoice{
			{
				Index: 0,
				Message: providers.ChatMessage{
					Role:    "assistant",
					Content: completion,
				},
				FinishReason: "stop",
			},
		},
		Usage: interfaces.UsageInfo{
			PromptTokens:     100, // Estimate
			CompletionTokens: 100, // Estimate
			TotalTokens:      200,
		},
	}, nil
}

// StreamChatCompletion implements providers.AIProvider
func (w *llmClientWrapper) StreamChatCompletion(ctx context.Context, req providers.ChatRequest) (providers.ChatStream, error) {
	return nil, gerror.New(gerror.ErrCodeInternal, "streaming not supported by wrapper", nil).
		WithComponent("orchestrator").
		WithOperation("StreamChatCompletion")
}

// CreateEmbedding implements providers.AIProvider
func (w *llmClientWrapper) CreateEmbedding(ctx context.Context, req providers.EmbeddingRequest) (*providers.EmbeddingResponse, error) {
	return nil, gerror.New(gerror.ErrCodeInternal, "embeddings not supported by wrapper", nil).
		WithComponent("orchestrator").
		WithOperation("CreateEmbedding")
}

// GetCapabilities implements providers.AIProvider
func (w *llmClientWrapper) GetCapabilities() providers.ProviderCapabilities {
	// Return default capabilities
	return providers.ProviderCapabilities{
		MaxTokens:          4096,
		ContextWindow:      4096,
		SupportsVision:     false,
		SupportsTools:      false,
		SupportsStream:     false,
		SupportsEmbeddings: false,
	}
}

// validateDependencies ensures all required components are configured
func (s *CommissionIntegrationService) validateDependencies() error {
	if s.commissionRefiner == nil {
		return gerror.New(gerror.ErrCodeOrchestration, "commission refiner not configured", nil).
			WithComponent("orchestrator").
			WithOperation("validateDependencies")
	}
	if s.commissionPlanner == nil {
		return gerror.New(gerror.ErrCodeOrchestration, "commission planner not configured", nil).
			WithComponent("orchestrator").
			WithOperation("validateDependencies")
	}
	if s.kanbanManager == nil {
		return gerror.New(gerror.ErrCodeOrchestration, "kanban manager not configured", nil).
			WithComponent("orchestrator").
			WithOperation("validateDependencies")
	}
	return nil
}

// addCommissionContext adds commission information to the request context
func (s *CommissionIntegrationService) addCommissionContext(ctx context.Context, commission manager.Commission) context.Context {
	ctx = context.WithValue(ctx, "commission_id", commission.ID)
	ctx = context.WithValue(ctx, "commission_title", commission.Title)
	ctx = context.WithValue(ctx, "commission_domain", commission.Domain)
	ctx = context.WithValue(ctx, "timestamp", fmt.Sprintf("%d", time.Now().Unix()))
	return ctx
}

// registryToManagerCommission converts a registry commission to a manager commission
func (s *CommissionIntegrationService) registryToManagerCommission(registryCommission *registry.Commission) manager.Commission {
	// Extract domain from commission if available
	domain := "general"
	if registryCommission.Domain != nil && *registryCommission.Domain != "" {
		domain = *registryCommission.Domain
	}

	// Get description
	description := ""
	if registryCommission.Description != nil {
		description = *registryCommission.Description
	}

	// Convert context
	context := make(map[string]interface{})
	if registryCommission.Context != nil {
		context = registryCommission.Context
	}

	return manager.Commission{
		ID:          registryCommission.ID,
		Title:       registryCommission.Title,
		Description: description,
		Domain:      domain,
		Context:     context,
	}
}

// extractAssignedArtisans extracts the list of artisans assigned to tasks
func (s *CommissionIntegrationService) extractAssignedArtisans(tasks []*kanban.Task) []string {
	artisanSet := make(map[string]bool)
	for _, task := range tasks {
		if task.AssignedTo != "" {
			artisanSet[task.AssignedTo] = true
		}
	}

	artisans := make([]string, 0, len(artisanSet))
	for artisan := range artisanSet {
		artisans = append(artisans, artisan)
	}

	return artisans
}

// emitCommissionProcessedEvent emits an event when commission processing is complete
func (s *CommissionIntegrationService) emitCommissionProcessedEvent(commission manager.Commission, tasks []*kanban.Task) {
	if s.eventBus != nil {
		event := Event{
			Type:   "commission_processed",
			Source: "commission_integration_service",
			Data: map[string]interface{}{
				"commission_id":    commission.ID,
				"commission_title": commission.Title,
				"task_count":       len(tasks),
				"domain":           commission.Domain,
			},
		}
		s.eventBus.Publish(event)
	}
}

// CommissionProcessingResult contains the results of commission processing
type CommissionProcessingResult struct {
	Commission        manager.Commission         `json:"commission"`
	RefinedCommission *manager.RefinedCommission `json:"refined_commission"`
	Tasks             []*kanban.Task             `json:"tasks"`
	AssignedArtisans  []string                   `json:"assigned_artisans"`
}

// GetTasksByStatus returns tasks filtered by status
func (r *CommissionProcessingResult) GetTasksByStatus(status kanban.TaskStatus) []*kanban.Task {
	var filtered []*kanban.Task
	for _, task := range r.Tasks {
		if task.Status == status {
			filtered = append(filtered, task)
		}
	}
	return filtered
}

// GetTasksByArtisan returns tasks assigned to a specific artisan
func (r *CommissionProcessingResult) GetTasksByArtisan(artisanID string) []*kanban.Task {
	var filtered []*kanban.Task
	for _, task := range r.Tasks {
		if task.AssignedTo == artisanID {
			filtered = append(filtered, task)
		}
	}
	return filtered
}

// GetTaskCount returns the total number of tasks
func (r *CommissionProcessingResult) GetTaskCount() int {
	return len(r.Tasks)
}

// GetAssignedArtisanCount returns the number of unique assigned artisans
func (r *CommissionProcessingResult) GetAssignedArtisanCount() int {
	return len(r.AssignedArtisans)
}

// kanbanRegistryAdapter adapts registry.ComponentRegistry to kanban.ComponentRegistry
type kanbanRegistryAdapter struct {
	registry registry.ComponentRegistry
}

// Storage returns a kanban.StorageRegistry implementation
func (k *kanbanRegistryAdapter) Storage() kanban.StorageRegistry {
	return &kanbanStorageAdapter{storageRegistry: k.registry.Storage()}
}

// kanbanStorageAdapter adapts registry.StorageRegistry to kanban.StorageRegistry
type kanbanStorageAdapter struct {
	storageRegistry registry.StorageRegistry
}

// Implement kanban.StorageRegistry interface methods
func (k *kanbanStorageAdapter) GetKanbanCampaignRepository() kanban.CampaignRepository {
	// Get the underlying storage registry and create adapter
	if sqliteReg, ok := k.storageRegistry.(interface {
		GetStorageRegistry() storage.StorageRegistry
	}); ok {
		return &kanbanCampaignRepoAdapter{repo: sqliteReg.GetStorageRegistry().GetCampaignRepository()}
	}
	// For registry interfaces, we need to create adapters since the method signatures differ
	return &kanbanCampaignRepoAdapter{repo: nil} // This will use the interface{} approach
}

func (k *kanbanStorageAdapter) GetKanbanCommissionRepository() kanban.CommissionRepository {
	if sqliteReg, ok := k.storageRegistry.(interface {
		GetStorageRegistry() storage.StorageRegistry
	}); ok {
		return &kanbanCommissionRepoAdapter{repo: sqliteReg.GetStorageRegistry().GetCommissionRepository()}
	}
	return &kanbanCommissionRepoAdapter{repo: nil}
}

func (k *kanbanStorageAdapter) GetKanbanTaskRepository() kanban.TaskRepository {
	// First try to get the actual kanban task repository from the registry
	if kanbanTaskRepo := k.storageRegistry.GetKanbanTaskRepository(); kanbanTaskRepo != nil {
		// Convert registry.KanbanTaskRepository to kanban.TaskRepository
		if adapter, ok := kanbanTaskRepo.(kanban.TaskRepository); ok {
			return adapter
		}
	}

	// Fallback: try to get the underlying storage registry
	if sqliteReg, ok := k.storageRegistry.(interface {
		GetStorageRegistry() storage.StorageRegistry
	}); ok {
		return &kanbanTaskRepoAdapter{repo: sqliteReg.GetStorageRegistry().GetTaskRepository()}
	}

	// Last resort: return an adapter that will handle the conversion
	return &kanbanTaskRepoAdapter{repo: nil}
}

func (k *kanbanStorageAdapter) GetBoardRepository() kanban.BoardRepository {
	if sqliteReg, ok := k.storageRegistry.(interface {
		GetStorageRegistry() storage.StorageRegistry
	}); ok {
		return &kanbanBoardRepoAdapter{repo: sqliteReg.GetStorageRegistry().GetBoardRepository()}
	}
	return &kanbanBoardRepoAdapter{repo: nil}
}

func (k *kanbanStorageAdapter) GetMemoryStore() kanban.MemoryStore {
	// Return the memory store adapter from the storage registry
	if memStore := k.storageRegistry.GetMemoryStore(); memStore != nil {
		// Convert from registry.MemoryStore to kanban.MemoryStore
		return &kanbanMemoryStoreAdapter{memStore: memStore}
	}
	return nil
}

// kanbanMemoryStoreAdapter adapts registry.MemoryStore to kanban.MemoryStore
type kanbanMemoryStoreAdapter struct {
	memStore registry.MemoryStore
}

func (k *kanbanMemoryStoreAdapter) Get(ctx context.Context, bucket, key string) ([]byte, error) {
	return k.memStore.Get(ctx, bucket, key)
}

func (k *kanbanMemoryStoreAdapter) Put(ctx context.Context, bucket, key string, value []byte) error {
	return k.memStore.Put(ctx, bucket, key, value)
}

func (k *kanbanMemoryStoreAdapter) Delete(ctx context.Context, bucket, key string) error {
	return k.memStore.Delete(ctx, bucket, key)
}

func (k *kanbanMemoryStoreAdapter) List(ctx context.Context, bucket string) ([]string, error) {
	return k.memStore.List(ctx, bucket)
}

func (k *kanbanMemoryStoreAdapter) ListKeys(ctx context.Context, bucket, prefix string) ([]string, error) {
	// Default implementation - list all keys and filter by prefix
	allKeys, err := k.memStore.List(ctx, bucket)
	if err != nil {
		return nil, err
	}

	var filteredKeys []string
	for _, key := range allKeys {
		if strings.HasPrefix(key, prefix) {
			filteredKeys = append(filteredKeys, key)
		}
	}

	return filteredKeys, nil
}

func (k *kanbanMemoryStoreAdapter) Close() error {
	// The underlying store should handle cleanup
	if closer, ok := k.memStore.(interface{ Close() error }); ok {
		return closer.Close()
	}
	return nil
}

// Repository adapters that bridge the interface differences between storage and kanban packages

// kanbanCampaignRepoAdapter adapts storage.CampaignRepository to kanban.CampaignRepository
type kanbanCampaignRepoAdapter struct {
	repo storage.CampaignRepository
}

func (k *kanbanCampaignRepoAdapter) CreateCampaign(ctx context.Context, campaign interface{}) error {
	// Convert interface{} to storage campaign format
	if k.repo != nil {
		// Convert map to storage.Campaign if needed
		if campaignMap, ok := campaign.(map[string]interface{}); ok {
			storageCampaign := &storage.Campaign{
				ID:        campaignMap["ID"].(string),
				Name:      campaignMap["Name"].(string),
				Status:    campaignMap["Status"].(string),
				CreatedAt: campaignMap["CreatedAt"].(time.Time),
				UpdatedAt: campaignMap["UpdatedAt"].(time.Time),
			}
			return k.repo.CreateCampaign(ctx, storageCampaign)
		}
	}
	return gerror.New(gerror.ErrCodeStorage, "campaign repository not available", nil).
		WithComponent("orchestrator").
		WithOperation("CreateCampaign")
}

// kanbanCommissionRepoAdapter adapts storage.CommissionRepository to kanban.CommissionRepository
type kanbanCommissionRepoAdapter struct {
	repo storage.CommissionRepository
}

func (k *kanbanCommissionRepoAdapter) CreateCommission(ctx context.Context, commission interface{}) error {
	if k.repo != nil {
		if commissionMap, ok := commission.(map[string]interface{}); ok {
			storageCommission := &storage.Commission{
				ID:         commissionMap["ID"].(string),
				CampaignID: commissionMap["CampaignID"].(string),
				Title:      commissionMap["Title"].(string),
				Status:     commissionMap["Status"].(string),
				CreatedAt:  commissionMap["CreatedAt"].(time.Time),
			}
			if desc, exists := commissionMap["Description"]; exists && desc != nil {
				if descStr, ok := desc.(string); ok {
					storageCommission.Description = &descStr
				}
			}
			if domain, exists := commissionMap["Domain"]; exists && domain != nil {
				if domainStr, ok := domain.(string); ok {
					storageCommission.Domain = &domainStr
				}
			}
			if context, exists := commissionMap["Context"]; exists {
				if contextMap, ok := context.(map[string]interface{}); ok {
					storageCommission.Context = contextMap
				}
			}
			return k.repo.CreateCommission(ctx, storageCommission)
		}
	}
	return gerror.New(gerror.ErrCodeStorage, "commission repository not available", nil).
		WithComponent("orchestrator").
		WithOperation("CreateCommission")
}

func (k *kanbanCommissionRepoAdapter) GetCommission(ctx context.Context, id string) (interface{}, error) {
	if k.repo != nil {
		return k.repo.GetCommission(ctx, id)
	}
	return nil, gerror.New(gerror.ErrCodeStorage, "commission repository not available", nil).
		WithComponent("orchestrator").
		WithOperation("GetCommission").
		WithDetails("id", id)
}

// kanbanTaskRepoAdapter adapts storage.TaskRepository to kanban.TaskRepository
type kanbanTaskRepoAdapter struct {
	repo storage.TaskRepository
}

func (k *kanbanTaskRepoAdapter) CreateTask(ctx context.Context, task interface{}) error {
	if k.repo != nil {
		if taskMap, ok := task.(map[string]interface{}); ok {
			// Convert BoardID to pointer since it's nullable in the new schema
			var boardID *string
			if bid, exists := taskMap["BoardID"]; exists && bid != nil {
				if bidStr, ok := bid.(string); ok && bidStr != "" {
					boardID = &bidStr
				}
			}

			storageTask := &storage.Task{
				ID:          taskMap["ID"].(string),
				BoardID:     boardID, // Use nullable BoardID
				Title:       taskMap["Title"].(string),
				Status:      taskMap["Status"].(string),
				StoryPoints: taskMap["StoryPoints"].(int32),
				CreatedAt:   taskMap["CreatedAt"].(time.Time),
				UpdatedAt:   taskMap["UpdatedAt"].(time.Time),
			}
			if agentID, exists := taskMap["AssignedAgentID"]; exists && agentID != nil {
				if agentIDStr, ok := agentID.(*string); ok {
					storageTask.AssignedAgentID = agentIDStr
				}
			}
			if desc, exists := taskMap["Description"]; exists && desc != nil {
				if descStr, ok := desc.(*string); ok {
					storageTask.Description = descStr
				}
			}
			if metadata, exists := taskMap["Metadata"]; exists {
				if metadataMap, ok := metadata.(map[string]interface{}); ok {
					storageTask.Metadata = metadataMap
				}
			}

			// Try to update first (upsert logic for kanban compatibility)
			if err := k.repo.UpdateTask(ctx, storageTask); err != nil {
				// If update fails, try to create (task might not exist yet)
				return k.repo.CreateTask(ctx, storageTask)
			}
			return nil
		}
	}
	return gerror.New(gerror.ErrCodeStorage, "task repository not available", nil).
		WithComponent("orchestrator").
		WithOperation("CreateTask")
}

func (k *kanbanTaskRepoAdapter) UpdateTask(ctx context.Context, task interface{}) error {
	if k.repo != nil {
		if taskMap, ok := task.(map[string]interface{}); ok {
			// Convert BoardID to pointer since it's nullable in the new schema
			var boardID *string
			if bid, exists := taskMap["BoardID"]; exists && bid != nil {
				if bidStr, ok := bid.(string); ok && bidStr != "" {
					boardID = &bidStr
				}
			}

			storageTask := &storage.Task{
				ID:          taskMap["ID"].(string),
				BoardID:     boardID, // Use nullable BoardID
				Title:       taskMap["Title"].(string),
				Status:      taskMap["Status"].(string),
				StoryPoints: taskMap["StoryPoints"].(int32),
				CreatedAt:   taskMap["CreatedAt"].(time.Time),
				UpdatedAt:   taskMap["UpdatedAt"].(time.Time),
			}
			if agentID, exists := taskMap["AssignedAgentID"]; exists && agentID != nil {
				if agentIDStr, ok := agentID.(*string); ok {
					storageTask.AssignedAgentID = agentIDStr
				}
			}
			if desc, exists := taskMap["Description"]; exists && desc != nil {
				if descStr, ok := desc.(*string); ok {
					storageTask.Description = descStr
				}
			}
			if metadata, exists := taskMap["Metadata"]; exists {
				if metadataMap, ok := metadata.(map[string]interface{}); ok {
					storageTask.Metadata = metadataMap
				}
			}
			return k.repo.UpdateTask(ctx, storageTask)
		}
	}
	return gerror.New(gerror.ErrCodeStorage, "task repository not available", nil).
		WithComponent("orchestrator").
		WithOperation("UpdateTask")
}

func (k *kanbanTaskRepoAdapter) DeleteTask(ctx context.Context, id string) error {
	if k.repo != nil {
		return k.repo.DeleteTask(ctx, id)
	}
	return gerror.New(gerror.ErrCodeStorage, "task repository not available", nil).
		WithComponent("orchestrator").
		WithOperation("DeleteTask").
		WithDetails("id", id)
}

func (k *kanbanTaskRepoAdapter) ListTasksByBoard(ctx context.Context, boardID string) ([]interface{}, error) {
	if k.repo != nil {
		tasks, err := k.repo.ListTasksByBoard(ctx, boardID)
		if err != nil {
			return nil, err
		}
		// Convert to []interface{}
		result := make([]interface{}, len(tasks))
		for i, task := range tasks {
			result[i] = task
		}
		return result, nil
	}
	return nil, gerror.New(gerror.ErrCodeStorage, "task repository not available", nil).
		WithComponent("orchestrator").
		WithOperation("ListTasksByBoard").
		WithDetails("boardID", boardID)
}

func (k *kanbanTaskRepoAdapter) RecordTaskEvent(ctx context.Context, event interface{}) error {
	if k.repo != nil {
		if eventMap, ok := event.(map[string]interface{}); ok {
			storageEvent := &storage.TaskEvent{
				TaskID:    eventMap["TaskID"].(string),
				EventType: eventMap["EventType"].(string),
				CreatedAt: eventMap["CreatedAt"].(time.Time),
			}
			if agentID, exists := eventMap["AgentID"]; exists && agentID != nil {
				if agentIDStr, ok := agentID.(string); ok {
					storageEvent.AgentID = &agentIDStr
				}
			}
			if oldValue, exists := eventMap["OldValue"]; exists && oldValue != nil {
				if oldValueStr, ok := oldValue.(*string); ok {
					storageEvent.OldValue = oldValueStr
				}
			}
			if newValue, exists := eventMap["NewValue"]; exists && newValue != nil {
				if newValueStr, ok := newValue.(*string); ok {
					storageEvent.NewValue = newValueStr
				}
			}
			if reason, exists := eventMap["Reason"]; exists && reason != nil {
				if reasonStr, ok := reason.(*string); ok {
					storageEvent.Reason = reasonStr
				}
			}
			return k.repo.RecordTaskEvent(ctx, storageEvent)
		}
	}
	return gerror.New(gerror.ErrCodeStorage, "task repository not available", nil).
		WithComponent("orchestrator").
		WithOperation("RecordTaskEvent")
}

// commissionRepositoryAdapter adapts registry.CommissionRepository to core.CommissionRepository
type commissionRepositoryAdapter struct {
	repo registry.CommissionRepository
}

// newCommissionRepositoryAdapter creates a new commission repository adapter
func newCommissionRepositoryAdapter(repo registry.CommissionRepository) core.CommissionRepository {
	return &commissionRepositoryAdapter{repo: repo}
}

// CreateCommission implements core.CommissionRepository
func (c *commissionRepositoryAdapter) CreateCommission(ctx context.Context, commission *core.Commission) error {
	if c.repo == nil {
		return gerror.New(gerror.ErrCodeStorage, "commission repository not available", nil).
			WithComponent("orchestrator").
			WithOperation("CreateCommission")
	}

	// Convert core.Commission to registry.Commission
	registryCommission := &registry.Commission{
		ID:         commission.ID,
		CampaignID: commission.CampaignID,
		Title:      commission.Title,
		Status:     commission.Status,
	}

	// Handle nullable Description field
	if commission.Description != nil && *commission.Description != "" {
		registryCommission.Description = commission.Description
	}

	return c.repo.CreateCommission(ctx, registryCommission)
}

// GetCommission implements core.CommissionRepository
func (c *commissionRepositoryAdapter) GetCommission(ctx context.Context, id string) (*core.Commission, error) {
	if c.repo == nil {
		return nil, gerror.New(gerror.ErrCodeStorage, "commission repository not available", nil).
			WithComponent("orchestrator").
			WithOperation("GetCommission").
			WithDetails("id", id)
	}

	registryCommission, err := c.repo.GetCommission(ctx, id)
	if err != nil {
		return nil, err
	}

	// Convert registry.Commission to core.Commission
	agentCommission := &core.Commission{
		ID:         registryCommission.ID,
		CampaignID: registryCommission.CampaignID,
		Title:      registryCommission.Title,
		Status:     registryCommission.Status,
	}

	// Handle nullable Description field
	if registryCommission.Description != nil {
		agentCommission.Description = registryCommission.Description
	}

	return agentCommission, nil
}
