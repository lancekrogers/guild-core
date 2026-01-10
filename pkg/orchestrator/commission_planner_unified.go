// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package orchestrator

import (
	"context"

	"github.com/google/uuid"
	"github.com/lancekrogers/guild-core/pkg/agents/core/manager"
	"github.com/lancekrogers/guild-core/pkg/config"
	"github.com/lancekrogers/guild-core/pkg/events"
	"github.com/lancekrogers/guild-core/pkg/kanban"
	"github.com/lancekrogers/guild-core/pkg/orchestrator/interfaces"
)

// UnifiedCommissionTaskPlanner implements CommissionTaskPlanner using the unified event system
type UnifiedCommissionTaskPlanner struct {
	*defaultCommissionTaskPlanner
	unifiedEventBus events.EventBus
}

// NewUnifiedCommissionTaskPlanner creates a new commission task planner using unified events
func NewUnifiedCommissionTaskPlanner(
	kanbanManager KanbanManager,
	parser manager.ResponseParser,
	unifiedEventBus events.EventBus,
) CommissionTaskPlanner {
	// Create the base planner with nil legacy event bus
	basePlanner := newCommissionTaskPlanner(kanbanManager, parser, nil)

	return &UnifiedCommissionTaskPlanner{
		defaultCommissionTaskPlanner: basePlanner,
		unifiedEventBus:              unifiedEventBus,
	}
}

// UnifiedCommissionTaskPlannerFactory creates a unified commission task planner for registry use
func UnifiedCommissionTaskPlannerFactory(
	kanbanManager KanbanManager,
	parser manager.ResponseParser,
	unifiedEventBus events.EventBus,
) CommissionTaskPlanner {
	return NewUnifiedCommissionTaskPlanner(kanbanManager, parser, unifiedEventBus)
}

// PlanFromRefinedCommission converts refined commission tasks to kanban tasks
func (p *UnifiedCommissionTaskPlanner) PlanFromRefinedCommission(
	ctx context.Context,
	refined *manager.RefinedCommission,
	guildConfig *config.GuildConfig,
) ([]*kanban.Task, error) {
	// Extract tasks from the refined commission structure
	tasks := p.extractTasksFromStructure(refined.Structure)

	// Convert to kanban tasks
	kanbanTasks := make([]*kanban.Task, 0, len(tasks))
	for _, taskInfo := range tasks {
		kanbanTask, err := p.convertToKanbanTask(ctx, taskInfo, refined.CommissionID, guildConfig)
		if err != nil {
			return nil, err
		}
		kanbanTasks = append(kanbanTasks, kanbanTask)
	}

	// Emit events for task creation using unified event bus
	for _, task := range kanbanTasks {
		p.emitUnifiedTaskCreatedEvent(ctx, task, refined.CommissionID)
	}

	return kanbanTasks, nil
}

// AssignTasksToArtisans assigns tasks to appropriate guild members
func (p *UnifiedCommissionTaskPlanner) AssignTasksToArtisans(
	ctx context.Context,
	tasks []*kanban.Task,
	guild *config.GuildConfig,
) error {
	// Use the base implementation for the core logic
	err := p.defaultCommissionTaskPlanner.AssignTasksToArtisans(ctx, tasks, guild)
	if err != nil {
		return err
	}

	// Override event emission with unified events
	for _, task := range tasks {
		if task.AssignedTo != "" {
			p.emitUnifiedTaskAssignedEvent(ctx, task, task.AssignedTo)
		}
	}

	return nil
}

// emitUnifiedTaskCreatedEvent emits a task creation event to the unified event bus
func (p *UnifiedCommissionTaskPlanner) emitUnifiedTaskCreatedEvent(ctx context.Context, task *kanban.Task, commissionID string) {
	if p.unifiedEventBus != nil {
		event := events.NewBaseEvent(
			uuid.New().String(),
			string(interfaces.EventTypeTaskCreated),
			"commission_planner",
			map[string]interface{}{
				"task_id":       task.ID,
				"commission_id": commissionID,
				"title":         task.Title,
				"priority":      task.Priority,
			},
		)

		// Add additional metadata
		event.WithData("status", task.Status)
		event.WithData("estimated_hours", task.EstimatedHours)
		if len(task.Dependencies) > 0 {
			event.WithData("dependencies", task.Dependencies)
		}

		// Publish the event
		if err := p.unifiedEventBus.Publish(ctx, event); err != nil {
			// Log error but don't fail the operation
			// Since we don't have direct logger access, we silently continue
			_ = err
		}
	}
}

// emitUnifiedTaskAssignedEvent emits a task assignment event to the unified event bus
func (p *UnifiedCommissionTaskPlanner) emitUnifiedTaskAssignedEvent(ctx context.Context, task *kanban.Task, artisanID string) {
	if p.unifiedEventBus != nil {
		event := events.NewBaseEvent(
			uuid.New().String(),
			string(interfaces.EventTypeTaskAssigned),
			"commission_planner",
			map[string]interface{}{
				"task_id":    task.ID,
				"artisan_id": artisanID,
				"title":      task.Title,
			},
		)

		// Add additional context
		event.WithData("priority", task.Priority)
		event.WithData("status", task.Status)
		if commissionID, exists := task.Metadata["commission_id"]; exists {
			event.WithData("commission_id", commissionID)
		}

		// Publish the event
		if err := p.unifiedEventBus.Publish(ctx, event); err != nil {
			// Log error but don't fail the operation
			_ = err
		}
	}
}

// Override the legacy event methods to prevent dual publishing
func (p *UnifiedCommissionTaskPlanner) emitTaskCreatedEvent(task *kanban.Task, commissionID string) {
	// No-op: events are handled by unified methods
}

func (p *UnifiedCommissionTaskPlanner) emitTaskAssignedEvent(task *kanban.Task, artisanID string) {
	// No-op: events are handled by unified methods
}
