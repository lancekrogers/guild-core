package orchestrator

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/guild-ventures/guild-core/pkg/agent/manager"
	"github.com/guild-ventures/guild-core/pkg/config"
	"github.com/guild-ventures/guild-core/pkg/kanban"
	"github.com/guild-ventures/guild-core/pkg/objective"
	"github.com/guild-ventures/guild-core/pkg/prompts"
	"github.com/guild-ventures/guild-core/pkg/providers"
	"github.com/guild-ventures/guild-core/pkg/providers/interfaces"
	"github.com/guild-ventures/guild-core/pkg/registry"
)

// CommissionIntegrationService coordinates the complete pipeline from commission to kanban tasks
type CommissionIntegrationService struct {
	registry              registry.ComponentRegistry
	commissionRefiner     manager.CommissionRefiner
	commissionPlanner     CommissionTaskPlanner
	kanbanManager         KanbanManager
	objectiveManager      *objective.Manager
	eventBus              *EventBus
	guildMasterFactory    *manager.DefaultGuildMasterFactory
	taskBridge            *manager.TaskBridge
}

// NewCommissionIntegrationService creates a new integration service with full wiring
func NewCommissionIntegrationService(registry registry.ComponentRegistry) (*CommissionIntegrationService, error) {
	service := &CommissionIntegrationService{
		registry: registry,
	}

	// Initialize components from registry
	if err := service.initializeFromRegistry(); err != nil {
		return nil, fmt.Errorf("failed to initialize from registry: %w", err)
	}

	return service, nil
}

// initializeFromRegistry sets up all components from the registry
func (s *CommissionIntegrationService) initializeFromRegistry() error {
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
		return fmt.Errorf("no AI providers available in registry")
	}

	// Get prompt manager from registry
	promptRegistryFromReg := s.registry.Prompts()
	var promptManager prompts.Manager
	
	// Try to get prompt manager from registry first
	if promptRegistryFromReg != nil {
		// For now, create a temporary memory registry with some basic prompts
		// TODO: This should be properly integrated with the registry system
		promptRegistry := prompts.NewMemoryRegistry()
		
		// Register some basic prompts that may be needed
		basicGuildMasterPrompt := `You are the Guild Master, responsible for breaking down commissions into actionable tasks.

Format your response with XML task tags:

<task>
  <id>unique-task-id</id>
  <title>Clear Task Title</title>
  <description>Detailed description of what needs to be done</description>
  <priority>high|medium|low</priority>
  <estimate>estimated hours (e.g., 4h, 2d)</estimate>
  <category>backend|frontend|database|devops|testing|documentation</category>
  <dependencies>comma-separated task IDs that must be completed first</dependencies>
</task>`
		
		promptRegistry.RegisterPrompt("manager", "default", basicGuildMasterPrompt)
		promptRegistry.RegisterPrompt("manager", "web-development", basicGuildMasterPrompt)
		
		promptManager = prompts.NewDefaultManager(promptRegistry, nil)
	} else {
		// Fallback to creating a new one
		promptRegistry := prompts.NewMemoryRegistry()
		promptManager = prompts.NewDefaultManager(promptRegistry, nil)
	}

	// Create layered manager adapter if needed
	// Since promptManager is concrete type, we always need the adapter
	layeredManager := &layeredManagerAdapter{Manager: prompts.Manager(promptManager)}

	// Create Guild Master factory
	s.guildMasterFactory = manager.NewDefaultGuildMasterFactory(layeredManager, providers)

	// Create commission refiner
	var err error
	s.commissionRefiner, err = s.guildMasterFactory.CreateCommissionRefinerWithDefaults()
	if err != nil {
		return fmt.Errorf("failed to create commission refiner: %w", err)
	}

	// Get memory registry for kanban
	memoryRegistry := s.registry.Memory()
	memStore, err := memoryRegistry.GetDefaultMemoryStore()
	if err != nil {
		// Try to get any available store
		stores := memoryRegistry.ListMemoryStores()
		if len(stores) > 0 {
			memStore, err = memoryRegistry.GetMemoryStore(stores[0])
		}
		if err != nil || memStore == nil {
			return fmt.Errorf("failed to get memory store: %w", err)
		}
	}

	// Create kanban components
	kanbanBoard, err := kanban.NewBoard(memStore, "kanban", "default")
	if err != nil {
		return fmt.Errorf("failed to create kanban board: %w", err)
	}
	s.kanbanManager = NewDefaultKanbanManager(kanbanBoard)

	// Create event bus
	s.eventBus = NewEventBus()

	// Create intelligent parser for commission planner
	parserConfig := manager.IntelligentParserConfig{
		Mode:          manager.ParserModeAuto,
		ArtisanClient: s.commissionRefiner.(*manager.GuildMasterRefiner).GetArtisanClient(),
		PromptManager: layeredManager,
	}
	intelligentParser := manager.NewIntelligentParser(parserConfig)
	parserAdapter := manager.NewResponseParserAdapter(intelligentParser)

	// Create commission planner
	s.commissionPlanner = NewCommissionTaskPlanner(s.kanbanManager, parserAdapter, s.eventBus)

	// Create objective manager
	s.objectiveManager, err = objective.NewManager(memStore, "objectives")
	if err != nil {
		return fmt.Errorf("failed to create objective manager: %w", err)
	}

	// Create task bridge
	s.taskBridge = manager.NewTaskBridge(kanbanBoard, s.objectiveManager)

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
	s.eventBus = &eventBus
}

// SetKanbanManager sets the kanban manager (injected via registry or factory)
func (s *CommissionIntegrationService) SetKanbanManager(kanbanManager KanbanManager) {
	s.kanbanManager = kanbanManager
}

// SetObjectiveManager sets the objective manager (injected via registry or factory)
func (s *CommissionIntegrationService) SetObjectiveManager(objectiveManager *objective.Manager) {
	s.objectiveManager = objectiveManager
}

// ProcessCommissionToTasks handles the complete pipeline from commission to kanban tasks
func (s *CommissionIntegrationService) ProcessCommissionToTasks(
	ctx context.Context,
	commission manager.Commission,
	guildConfig *config.GuildConfig,
) (*CommissionProcessingResult, error) {
	// Validate dependencies
	if err := s.validateDependencies(); err != nil {
		return nil, err
	}

	// Add commission context to the request context
	ctx = s.addCommissionContext(ctx, commission)

	// Step 1: Refine the commission using IntelligentParser
	log.Printf("Refining commission: %s", commission.Title)
	refined, err := s.commissionRefiner.RefineCommission(ctx, commission)
	if err != nil {
		return nil, fmt.Errorf("failed to refine commission: %w", err)
	}
	log.Printf("Commission refined successfully, found %d files", len(refined.Structure.Files))

	// Step 2: Convert refined commission to kanban tasks
	log.Printf("Planning tasks from refined commission")
	tasks, err := s.commissionPlanner.PlanFromRefinedCommission(ctx, refined, guildConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to plan tasks from commission: %w", err)
	}
	log.Printf("Created %d tasks from commission", len(tasks))

	// Step 3: Use task bridge to create tasks in kanban
	if s.taskBridge != nil {
		log.Printf("Creating tasks in kanban system")
		if err := s.taskBridge.CreateTasksFromRefinedCommission(ctx, refined); err != nil {
			log.Printf("Warning: failed to create tasks via bridge: %v", err)
			// Don't fail - tasks were already created by planner
		}
	}

	// Step 4: Assign tasks to artisans
	log.Printf("Assigning tasks to artisans")
	if err := s.commissionPlanner.AssignTasksToArtisans(ctx, tasks, guildConfig); err != nil {
		return nil, fmt.Errorf("failed to assign tasks to artisans: %w", err)
	}

	// Step 5: Update objective with task information
	if s.objectiveManager != nil {
		if err := s.updateObjectiveWithTasks(ctx, commission, tasks); err != nil {
			log.Printf("Warning: failed to update objective: %v", err)
			// Don't fail the entire process
		}
	}

	// Step 6: Emit completion event
	s.emitCommissionProcessedEvent(commission, tasks)

	// Step 7: Write refined files if output directory is configured
	if outputDir, ok := ctx.Value("output_dir").(string); ok && outputDir != "" {
		log.Printf("Writing refined files to: %s", outputDir)
		if err := s.taskBridge.WriteRefinedFiles(refined, outputDir); err != nil {
			log.Printf("Warning: failed to write refined files: %v", err)
		}
	}

	return &CommissionProcessingResult{
		Commission:       commission,
		RefinedCommission: refined,
		Tasks:           tasks,
		AssignedArtisans: s.extractAssignedArtisans(tasks),
	}, nil
}

// ProcessObjectiveToTasks loads an objective and processes it to kanban tasks
func (s *CommissionIntegrationService) ProcessObjectiveToTasks(
	ctx context.Context,
	objectiveID string,
	guildConfig *config.GuildConfig,
) (*CommissionProcessingResult, error) {
	if s.objectiveManager == nil {
		return nil, fmt.Errorf("objective manager not configured")
	}

	// Load objective from storage
	obj, err := s.objectiveManager.GetObjective(ctx, objectiveID)
	if err != nil {
		return nil, fmt.Errorf("failed to load objective %s: %w", objectiveID, err)
	}

	// Convert objective to commission
	commission := s.objectiveToCommission(obj)

	// Process commission to tasks
	return s.ProcessCommissionToTasks(ctx, commission, guildConfig)
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
func (a *layeredManagerAdapter) BuildLayeredPrompt(ctx context.Context, artisanID, sessionID string, turnCtx prompts.TurnContext) (*prompts.LayeredPrompt, error) {
	// Simple implementation using the basic Manager
	var layers []prompts.SystemPrompt
	var compiled strings.Builder
	
	// Try to get a system prompt based on context
	if turnCtx.CommissionID != "" {
		// Use manager role for commission refinement
		systemPrompt, err := a.GetSystemPrompt(ctx, "manager", "default")
		if err == nil {
			layers = append(layers, prompts.SystemPrompt{
				Layer:     prompts.LayerRole,
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
			layers = append(layers, prompts.SystemPrompt{
				Layer:     prompts.LayerSession,
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
		layers = append(layers, prompts.SystemPrompt{
			Layer:     prompts.LayerTurn,
			ArtisanID: artisanID,
			SessionID: sessionID,
			Content:   instructions,
			Version:   1,
			Priority:  10,
			Updated:   time.Now(),
		})
		compiled.WriteString(instructions)
	}
	
	return &prompts.LayeredPrompt{
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
func (a *layeredManagerAdapter) GetPromptLayer(ctx context.Context, layer prompts.PromptLayer, artisanID, sessionID string) (*prompts.SystemPrompt, error) {
	// Simple implementation - return a basic prompt based on layer
	var content string
	var err error
	
	switch layer {
	case prompts.LayerRole:
		content, err = a.GetSystemPrompt(ctx, "manager", "default")
	case prompts.LayerDomain:
		content, err = a.GetSystemPrompt(ctx, "manager", "web-app")
	default:
		return nil, prompts.ErrLayerNotFound
	}
	
	if err != nil {
		return nil, err
	}
	
	return &prompts.SystemPrompt{
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
func (a *layeredManagerAdapter) SetPromptLayer(ctx context.Context, prompt prompts.SystemPrompt) error {
	// Not implemented in adapter
	return fmt.Errorf("SetPromptLayer not supported in adapter")
}

// DeletePromptLayer removes a specific prompt layer
func (a *layeredManagerAdapter) DeletePromptLayer(ctx context.Context, layer prompts.PromptLayer, artisanID, sessionID string) error {
	// Not implemented in adapter
	return fmt.Errorf("DeletePromptLayer not supported in adapter")
}

// ListPromptLayers returns all layers for an artisan/session
func (a *layeredManagerAdapter) ListPromptLayers(ctx context.Context, artisanID, sessionID string) ([]prompts.SystemPrompt, error) {
	// Return empty list
	return []prompts.SystemPrompt{}, nil
}

// InvalidateCache clears the layered prompt cache
func (a *layeredManagerAdapter) InvalidateCache(ctx context.Context, artisanID, sessionID string) error {
	// No cache in adapter
	return nil
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
	return nil, fmt.Errorf("streaming not supported by wrapper")
}

// CreateEmbedding implements providers.AIProvider
func (w *llmClientWrapper) CreateEmbedding(ctx context.Context, req providers.EmbeddingRequest) (*providers.EmbeddingResponse, error) {
	return nil, fmt.Errorf("embeddings not supported by wrapper")
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
		return fmt.Errorf("commission refiner not configured")
	}
	if s.commissionPlanner == nil {
		return fmt.Errorf("commission planner not configured")
	}
	if s.kanbanManager == nil {
		return fmt.Errorf("kanban manager not configured")
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

// updateObjectiveWithTasks updates the objective with generated task information
func (s *CommissionIntegrationService) updateObjectiveWithTasks(
	ctx context.Context,
	commission manager.Commission,
	tasks []*kanban.Task,
) error {
	if commission.ID == "" {
		return nil // No objective to update
	}

	// Get or create objective
	obj, err := s.objectiveManager.GetObjective(ctx, commission.ID)
	if err != nil {
		// Create new objective
		obj = &objective.Objective{
			ID:          commission.ID,
			Title:       commission.Title,
			Description: commission.Description,
			Status:      objective.StatusDraft,
			Metadata:    make(map[string]string),
		}
	}

	// Update metadata
	obj.Metadata["total_tasks"] = fmt.Sprintf("%d", len(tasks))
	obj.Metadata["tasks_created_at"] = time.Now().UTC().Format(time.RFC3339)
	obj.Metadata["domain"] = commission.Domain

	// Add task IDs
	taskIDs := make([]string, 0, len(tasks))
	for _, task := range tasks {
		taskIDs = append(taskIDs, task.ID)
	}
	obj.Metadata["task_ids"] = strings.Join(taskIDs, ",")

	// Save objective
	return s.objectiveManager.SaveObjective(ctx, obj)
}

// objectiveToCommission converts an objective to a commission for processing
func (s *CommissionIntegrationService) objectiveToCommission(obj *objective.Objective) manager.Commission {
	// Extract domain from objective metadata if available
	domain := "general"
	if obj.Metadata != nil {
		if d, ok := obj.Metadata["domain"]; ok && d != "" {
			domain = d
		}
	}
	
	// Convert metadata to context map
	context := make(map[string]interface{})
	for k, v := range obj.Metadata {
		context[k] = v
	}
	
	return manager.Commission{
		ID:          obj.ID,
		Title:       obj.Title,
		Description: obj.Description,
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
				"task_count":      len(tasks),
				"domain":          commission.Domain,
			},
		}
		s.eventBus.Publish(event)
	}
}

// CommissionProcessingResult contains the results of commission processing
type CommissionProcessingResult struct {
	Commission        manager.Commission        `json:"commission"`
	RefinedCommission *manager.RefinedCommission `json:"refined_commission"`
	Tasks            []*kanban.Task            `json:"tasks"`
	AssignedArtisans []string                  `json:"assigned_artisans"`
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