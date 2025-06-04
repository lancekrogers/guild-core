package orchestrator

import (
	"context"
	"fmt"

	"github.com/guild-ventures/guild-core/pkg/agent/manager"
	"github.com/guild-ventures/guild-core/pkg/config"
	guildctx "github.com/guild-ventures/guild-core/pkg/context"
	"github.com/guild-ventures/guild-core/pkg/kanban"
	"github.com/guild-ventures/guild-core/pkg/objective"
	"github.com/guild-ventures/guild-core/pkg/registry"
)

// CommissionIntegrationService coordinates the complete pipeline from commission to kanban tasks
type CommissionIntegrationService struct {
	registry              registry.ComponentRegistry
	commissionRefiner     manager.CommissionRefiner
	commissionPlanner     CommissionTaskPlanner
	kanbanManager         KanbanManager
	objectiveManager      *objective.Manager
	eventBus             *EventBus
}

// NewCommissionIntegrationService creates a new integration service
func NewCommissionIntegrationService(registry registry.ComponentRegistry) (*CommissionIntegrationService, error) {
	return &CommissionIntegrationService{
		registry: registry,
		// Components will be injected via setters or initialized when needed
	}, nil
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
	refined, err := s.commissionRefiner.RefineCommission(ctx, commission)
	if err != nil {
		return nil, fmt.Errorf("failed to refine commission: %w", err)
	}

	// Step 2: Convert refined commission to kanban tasks
	tasks, err := s.commissionPlanner.PlanFromRefinedCommission(ctx, refined, guildConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to plan tasks from commission: %w", err)
	}

	// Step 3: Assign tasks to artisans
	if err := s.commissionPlanner.AssignTasksToArtisans(ctx, tasks, guildConfig); err != nil {
		return nil, fmt.Errorf("failed to assign tasks to artisans: %w", err)
	}

	// Step 4: Update objective with task information
	if s.objectiveManager != nil {
		if err := s.updateObjectiveWithTasks(ctx, commission, tasks); err != nil {
			// Log error but don't fail the entire process
			// TODO: Add proper logging
		}
	}

	// Step 5: Emit completion event
	s.emitCommissionProcessedEvent(commission, tasks)

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
	// For now, return the context as-is
	// TODO: Add proper commission context tracking when needed
	return ctx
}

// updateObjectiveWithTasks updates the objective with generated task information
func (s *CommissionIntegrationService) updateObjectiveWithTasks(
	ctx context.Context,
	commission manager.Commission,
	tasks []*kanban.Task,
) error {
	// TODO: Implement objective update with task references
	// This would link the kanban tasks back to the original objective
	return nil
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
	
	return manager.Commission{
		Title:       obj.Title,
		Description: obj.Description,
		Domain:      domain,
		Context:     make(map[string]interface{}), // Convert context to map
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
				"commission_title": commission.Title,
				"task_count":      len(tasks),
				"domain":          commission.Domain,
			},
		}
		(*s.eventBus).Publish(event)
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