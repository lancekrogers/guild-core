package orchestrator

import (
	"context"
	"fmt"
	"strings"

	"github.com/guild-ventures/guild-core/pkg/agent/manager"
	"github.com/guild-ventures/guild-core/pkg/config"
	"github.com/guild-ventures/guild-core/pkg/kanban"
	"github.com/guild-ventures/guild-core/pkg/objective"
)

// CommissionTaskPlanner creates kanban tasks from refined commissions
type CommissionTaskPlanner interface {
	// PlanFromRefinedCommission converts refined commission tasks to kanban tasks
	PlanFromRefinedCommission(ctx context.Context, refined *manager.RefinedCommission, guildConfig *config.GuildConfig) ([]*kanban.Task, error)
	
	// AssignTasksToArtisans assigns tasks to appropriate guild members based on capabilities
	AssignTasksToArtisans(ctx context.Context, tasks []*kanban.Task, guild *config.GuildConfig) error
}

// DefaultCommissionTaskPlanner implements CommissionTaskPlanner using IntelligentParser
type DefaultCommissionTaskPlanner struct {
	kanbanManager  KanbanManager
	parser         manager.ResponseParser // IntelligentParser interface
	eventBus       EventBus
}

// NewCommissionTaskPlanner creates a new commission task planner
func NewCommissionTaskPlanner(
	kanbanManager KanbanManager,
	parser manager.ResponseParser,
	eventBus EventBus,
) *DefaultCommissionTaskPlanner {
	return &DefaultCommissionTaskPlanner{
		kanbanManager: kanbanManager,
		parser:        parser,
		eventBus:      eventBus,
	}
}

// PlanFromRefinedCommission converts refined commission tasks to kanban tasks
func (p *DefaultCommissionTaskPlanner) PlanFromRefinedCommission(
	ctx context.Context,
	refined *manager.RefinedCommission,
	guildConfig *config.GuildConfig,
) ([]*kanban.Task, error) {
	// Extract tasks using IntelligentParser if available
	var tasks []manager.TaskInfo
	
	// Check if parser supports direct task extraction
	if intelligentParser, ok := p.parser.(*manager.IntelligentParser); ok {
		// Use LLM-based extraction for enhanced task understanding
		extractedTasks, err := intelligentParser.ExtractTasksDirectly(ctx, refined)
		if err == nil {
			tasks = extractedTasks
		} else {
			// Fall back to parsing from file structure
			tasks = p.extractTasksFromStructure(refined.Structure)
		}
	} else {
		// Extract from file structure metadata
		tasks = p.extractTasksFromStructure(refined.Structure)
	}

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
func (p *DefaultCommissionTaskPlanner) AssignTasksToArtisans(
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

		// Assign task
		err = p.kanbanManager.UpdateTask(ctx, &kanban.Task{
			ID:          task.ID,
			Title:       task.Title,
			Description: task.Description,
			AssignedTo:  artisan.ID,
			Status:      kanban.StatusTodo,
			Priority:    task.Priority,
			Metadata:    task.Metadata,
		})
		if err != nil {
			return fmt.Errorf("failed to assign task %s to %s: %w", task.ID, artisan.ID, err)
		}

		// Emit assignment event
		p.emitTaskAssignedEvent(task, artisan.ID)
	}

	return nil
}

// extractTasksFromStructure extracts tasks from file structure metadata
func (p *DefaultCommissionTaskPlanner) extractTasksFromStructure(structure *manager.FileStructure) []manager.TaskInfo {
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
func (p *DefaultCommissionTaskPlanner) convertToKanbanTask(
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

	// Create kanban task
	task := &kanban.Task{
		ID:          taskInfo.ID,
		Title:       taskInfo.Title,
		Description: taskInfo.Description,
		Status:      kanban.StatusTodo,
		Priority:    priority,
		EstimatedHours: parseEstimate(taskInfo.Estimate),
		Dependencies: taskInfo.Dependencies,
		Metadata: map[string]interface{}{
			"commission_id":           commissionID,
			"source_section":          taskInfo.Section,
			"required_capabilities":   extractCapabilities(taskInfo),
			"original_category":       taskInfo.Category,
			"extraction_source":       "intelligent_parser",
		},
	}

	// Create the task in kanban system
	createdTask, err := p.kanbanManager.CreateTask(ctx, task.Title, task.Description)
	if err != nil {
		return nil, err
	}

	// Update with additional metadata
	createdTask.Priority = task.Priority
	createdTask.EstimatedHours = task.EstimatedHours
	createdTask.Dependencies = task.Dependencies
	createdTask.Metadata = task.Metadata

	return createdTask, nil
}

// findBestArtisan finds the best artisan for a task based on capabilities and workload
func (p *DefaultCommissionTaskPlanner) findBestArtisan(
	ctx context.Context,
	task *kanban.Task,
	guild *config.GuildConfig,
) (*config.AgentConfig, error) {
	// Extract required capabilities from task metadata
	requiredCaps := []string{}
	if caps, exists := task.Metadata["required_capabilities"]; exists {
		if capList, ok := caps.([]string); ok {
			requiredCaps = capList
		}
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
func (p *DefaultCommissionTaskPlanner) emitTaskCreatedEvent(task *kanban.Task, commissionID string) {
	if p.eventBus != nil {
		event := Event{
			Type:   EventTypeTaskCreated,
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
func (p *DefaultCommissionTaskPlanner) emitTaskAssignedEvent(task *kanban.Task, artisanID string) {
	if p.eventBus != nil {
		event := Event{
			Type:   EventTypeTaskAssigned,
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
		if strings.Contains(strings.ToLower(agent.Role), strings.ToLower(req)) {
			return true
		}
	}
	return len(required) == 0 // Default to true if no specific requirements
}