package orchestrator

import (
	"context"
	"fmt"
	"strings"

	"github.com/guild-ventures/guild-core/pkg/agent/manager"
	"github.com/guild-ventures/guild-core/pkg/config"
	"github.com/guild-ventures/guild-core/pkg/kanban"
	"github.com/guild-ventures/guild-core/pkg/orchestrator/interfaces"
)

// CommissionTaskPlanner creates kanban tasks from refined commissions
type CommissionTaskPlanner interface {
	// PlanFromRefinedCommission converts refined commission tasks to kanban tasks
	PlanFromRefinedCommission(ctx context.Context, refined *manager.RefinedCommission, guildConfig *config.GuildConfig) ([]*kanban.Task, error)
	
	// AssignTasksToArtisans assigns tasks to appropriate guild members based on capabilities
	AssignTasksToArtisans(ctx context.Context, tasks []*kanban.Task, guild *config.GuildConfig) error
}

// defaultCommissionTaskPlanner implements CommissionTaskPlanner using IntelligentParser
type defaultCommissionTaskPlanner struct {
	kanbanManager  KanbanManager
	parser         manager.ResponseParser // IntelligentParser interface
	eventBus       EventBus
}

// newCommissionTaskPlanner creates a new commission task planner (private constructor)
func newCommissionTaskPlanner(
	kanbanManager KanbanManager,
	parser manager.ResponseParser,
	eventBus EventBus,
) *defaultCommissionTaskPlanner {
	return &defaultCommissionTaskPlanner{
		kanbanManager: kanbanManager,
		parser:        parser,
		eventBus:      eventBus,
	}
}

// DefaultCommissionTaskPlannerFactory creates a commission task planner for registry use
func DefaultCommissionTaskPlannerFactory(
	kanbanManager KanbanManager,
	parser manager.ResponseParser,
	eventBus EventBus,
) CommissionTaskPlanner {
	return newCommissionTaskPlanner(kanbanManager, parser, eventBus)
}

// PlanFromRefinedCommission converts refined commission tasks to kanban tasks
func (p *defaultCommissionTaskPlanner) PlanFromRefinedCommission(
	ctx context.Context,
	refined *manager.RefinedCommission,
	guildConfig *config.GuildConfig,
) ([]*kanban.Task, error) {
	// Extract tasks from the refined commission structure
	// The parser (via adapter) already handles intelligent extraction internally
	tasks := p.extractTasksFromStructure(refined.Structure)

	// Convert to kanban tasks
	kanbanTasks := make([]*kanban.Task, 0, len(tasks))
	for _, taskInfo := range tasks {
		kanbanTask, err := p.convertToKanbanTask(ctx, taskInfo, refined.CommissionID, guildConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to convert task %s: %w", taskInfo.ID, err)
		}
		kanbanTasks = append(kanbanTasks, kanbanTask)
	}

	// Emit events for task creation
	for _, task := range kanbanTasks {
		p.emitTaskCreatedEvent(task, refined.CommissionID)
	}

	return kanbanTasks, nil
}

// AssignTasksToArtisans assigns tasks to appropriate guild members
func (p *defaultCommissionTaskPlanner) AssignTasksToArtisans(
	ctx context.Context,
	tasks []*kanban.Task,
	guild *config.GuildConfig,
) error {
	for _, task := range tasks {
		// Find best artisan based on task requirements and current workload
		artisan, err := p.findBestArtisan(ctx, task, guild)
		if err != nil {
			return fmt.Errorf("failed to find artisan for task %s: %w", task.ID, err)
		}

		// Get the existing task to preserve all metadata
		existingTask, err := p.kanbanManager.GetTask(ctx, task.ID)
		if err != nil {
			return fmt.Errorf("failed to get existing task %s: %w", task.ID, err)
		}

		// Update the assignment
		existingTask.AssignedTo = artisan.ID
		existingTask.Status = kanban.StatusTodo
		
		// Preserve metadata from original task (in case existing task has incomplete metadata)
		for k, v := range task.Metadata {
			existingTask.Metadata[k] = v
		}

		// Assign task by updating the existing task
		err = p.kanbanManager.UpdateTask(ctx, existingTask)
		if err != nil {
			return fmt.Errorf("failed to assign task %s to %s: %w", task.ID, artisan.ID, err)
		}

		// Update the original task reference for return to caller
		task.AssignedTo = artisan.ID
		task.Status = kanban.StatusTodo
		// Also ensure metadata is preserved in the original task reference
		for k, v := range existingTask.Metadata {
			task.Metadata[k] = v
		}

		// Emit assignment event
		p.emitTaskAssignedEvent(task, artisan.ID)
	}

	return nil
}

// extractTasksFromStructure extracts tasks from file structure metadata
func (p *defaultCommissionTaskPlanner) extractTasksFromStructure(structure *manager.FileStructure) []manager.TaskInfo {
	var tasks []manager.TaskInfo

	for _, file := range structure.Files {
		if fileTasks, exists := file.Metadata["tasks"]; exists {
			if taskList, ok := fileTasks.([]manager.TaskInfo); ok {
				tasks = append(tasks, taskList...)
			}
		}
	}

	return tasks
}

// convertToKanbanTask converts TaskInfo to kanban.Task
func (p *defaultCommissionTaskPlanner) convertToKanbanTask(
	ctx context.Context,
	taskInfo manager.TaskInfo,
	commissionID string,
	guildConfig *config.GuildConfig,
) (*kanban.Task, error) {
	// Determine priority level
	priority := kanban.PriorityMedium
	switch taskInfo.Priority {
	case "high":
		priority = kanban.PriorityHigh
	case "low":
		priority = kanban.PriorityLow
	}

	// Create the task in kanban system first (this generates the actual UUID)
	createdTask, err := p.kanbanManager.CreateTask(ctx, taskInfo.Title, taskInfo.Description)
	if err != nil {
		return nil, err
	}

	// Update the created task with additional metadata and properties
	createdTask.Priority = priority
	createdTask.EstimatedHours = parseEstimate(taskInfo.Estimate)
	createdTask.Dependencies = taskInfo.Dependencies
	
	// Initialize metadata map if nil
	if createdTask.Metadata == nil {
		createdTask.Metadata = make(map[string]string)
	}
	
	// Add commission and task metadata
	createdTask.Metadata["commission_id"] = commissionID
	createdTask.Metadata["source_section"] = taskInfo.Section
	createdTask.Metadata["required_capabilities"] = strings.Join(extractCapabilities(taskInfo), ",")
	createdTask.Metadata["category"] = taskInfo.Category
	createdTask.Metadata["original_category"] = taskInfo.Category
	createdTask.Metadata["extraction_source"] = "intelligent_parser"
	createdTask.Metadata["original_task_id"] = taskInfo.ID // Store the original parsed ID for reference

	return createdTask, nil
}

// findBestArtisan finds the best artisan for a task based on capabilities and workload
func (p *defaultCommissionTaskPlanner) findBestArtisan(
	ctx context.Context,
	task *kanban.Task,
	guild *config.GuildConfig,
) (*config.AgentConfig, error) {
	// Extract required capabilities from task metadata
	requiredCaps := []string{}
	if caps, exists := task.Metadata["required_capabilities"]; exists && caps != "" {
		// Capabilities are stored as comma-separated string
		requiredCaps = strings.Split(caps, ",")
	}

	// Find artisans with matching capabilities
	var candidates []*config.AgentConfig
	for _, agent := range guild.Agents {
		if hasRequiredCapabilities(&agent, requiredCaps) {
			candidates = append(candidates, &agent)
		}
	}

	if len(candidates) == 0 {
		// Fall back to any available artisan
		if len(guild.Agents) > 0 {
			return &guild.Agents[0], nil
		}
		return nil, fmt.Errorf("no available artisans in guild")
	}

	// For now, select first matching candidate
	// TODO: Implement workload-based selection
	return candidates[0], nil
}

// emitTaskCreatedEvent emits a task creation event
func (p *defaultCommissionTaskPlanner) emitTaskCreatedEvent(task *kanban.Task, commissionID string) {
	if p.eventBus != nil {
		event := Event{
			Type:   interfaces.EventTypeTaskCreated,
			Source: "commission_planner",
			Data: map[string]interface{}{
				"task_id":       task.ID,
				"commission_id": commissionID,
				"title":         task.Title,
				"priority":      task.Priority,
			},
		}
		p.eventBus.Publish(event)
	}
}

// emitTaskAssignedEvent emits a task assignment event
func (p *defaultCommissionTaskPlanner) emitTaskAssignedEvent(task *kanban.Task, artisanID string) {
	if p.eventBus != nil {
		event := Event{
			Type:   interfaces.EventTypeTaskAssigned,
			Source: "commission_planner",
			Data: map[string]interface{}{
				"task_id":    task.ID,
				"artisan_id": artisanID,
				"title":      task.Title,
			},
		}
		p.eventBus.Publish(event)
	}
}

// Helper functions

func parseEstimate(estimate string) float64 {
	// Parse estimates like "4h", "2d", "1w" to hours
	// TODO: Implement proper parsing
	return 0
}

func extractCapabilities(taskInfo manager.TaskInfo) []string {
	// Extract capabilities from task category and description
	// TODO: Implement intelligent capability extraction
	return []string{taskInfo.Category}
}

func hasRequiredCapabilities(agent *config.AgentConfig, required []string) bool {
	// Check if agent has required capabilities
	// For now, just check if agent type/role matches any of the required capabilities
	for _, req := range required {
		if strings.Contains(strings.ToLower(agent.Type), strings.ToLower(req)) {
			return true
		}
		// Check capabilities as well
		for _, cap := range agent.Capabilities {
			if strings.Contains(strings.ToLower(cap), strings.ToLower(req)) {
				return true
			}
		}
	}
	return len(required) == 0 // Default to true if no specific requirements
}