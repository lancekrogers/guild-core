// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package kanban

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/lancekrogers/guild-core/pkg/events"
	"github.com/lancekrogers/guild-core/pkg/gerror"
	"github.com/lancekrogers/guild-core/pkg/observability"
)

// UnifiedTaskEventPublisher publishes task events using only the unified event system
type UnifiedTaskEventPublisher struct {
	eventBus events.EventBus
}

// NewUnifiedTaskEventPublisher creates a new task event publisher using only the unified event bus
func NewUnifiedTaskEventPublisher(eventBus events.EventBus) *UnifiedTaskEventPublisher {
	return &UnifiedTaskEventPublisher{
		eventBus: eventBus,
	}
}

// PublishTaskCreated publishes a task creation event
func (p *UnifiedTaskEventPublisher) PublishTaskCreated(ctx context.Context, task *Task, boardID, createdBy string) error {
	if err := ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("UnifiedTaskEventPublisher").
			WithOperation("PublishTaskCreated")
	}

	logger := observability.GetLogger(ctx).
		WithComponent("UnifiedTaskEventPublisher").
		WithOperation("PublishTaskCreated")

	startTime := time.Now()
	defer func() {
		duration := time.Since(startTime)
		logger.Info("Task created event published",
			"task_id", task.ID,
			"board_id", boardID,
			"created_by", createdBy,
			"duration_ms", duration.Milliseconds(),
		)
	}()

	// Create unified event
	event := events.NewBaseEvent(
		uuid.New().String(),
		events.EventTypeTaskCreated,
		"kanban-service",
		map[string]interface{}{
			"task_id":     task.ID,
			"board_id":    boardID,
			"title":       task.Title,
			"description": task.Description,
			"status":      string(task.Status),
			"assignee":    task.AssignedTo,
			"created_by":  createdBy,
			"priority":    string(task.Priority),
			"created_at":  task.CreatedAt,
			"metadata":    task.Metadata,
		},
	)

	// Publish to unified event bus
	if err := p.eventBus.Publish(ctx, event); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to publish task created event").
			WithComponent("UnifiedTaskEventPublisher").
			WithOperation("PublishTaskCreated").
			WithDetails("task_id", task.ID)
	}

	return nil
}

// PublishTaskMoved publishes a task move event
func (p *UnifiedTaskEventPublisher) PublishTaskMoved(ctx context.Context, task *Task, boardID, fromStatus, toStatus, movedBy, reason string) error {
	if err := ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("UnifiedTaskEventPublisher").
			WithOperation("PublishTaskMoved")
	}

	logger := observability.GetLogger(ctx).
		WithComponent("UnifiedTaskEventPublisher").
		WithOperation("PublishTaskMoved")

	startTime := time.Now()
	defer func() {
		duration := time.Since(startTime)
		logger.Info("Task moved event published",
			"task_id", task.ID,
			"board_id", boardID,
			"from_status", fromStatus,
			"to_status", toStatus,
			"moved_by", movedBy,
			"duration_ms", duration.Milliseconds(),
		)
	}()

	// Create unified event
	event := events.NewBaseEvent(
		uuid.New().String(),
		events.EventTypeKanbanTaskMoved,
		"kanban-service",
		map[string]interface{}{
			"task_id":     task.ID,
			"board_id":    boardID,
			"title":       task.Title,
			"from_status": fromStatus,
			"to_status":   toStatus,
			"moved_by":    movedBy,
			"reason":      reason,
			"updated_at":  task.UpdatedAt,
			"metadata":    task.Metadata,
		},
	)

	// Publish to unified event bus
	if err := p.eventBus.Publish(ctx, event); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to publish task moved event").
			WithComponent("UnifiedTaskEventPublisher").
			WithOperation("PublishTaskMoved").
			WithDetails("task_id", task.ID)
	}

	return nil
}

// PublishTaskAssigned publishes a task assignment event
func (p *UnifiedTaskEventPublisher) PublishTaskAssigned(ctx context.Context, task *Task, boardID, assignedTo, assignedBy string) error {
	if err := ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("UnifiedTaskEventPublisher").
			WithOperation("PublishTaskAssigned")
	}

	logger := observability.GetLogger(ctx).
		WithComponent("UnifiedTaskEventPublisher").
		WithOperation("PublishTaskAssigned")

	startTime := time.Now()
	defer func() {
		duration := time.Since(startTime)
		logger.Info("Task assigned event published",
			"task_id", task.ID,
			"board_id", boardID,
			"assigned_to", assignedTo,
			"assigned_by", assignedBy,
			"duration_ms", duration.Milliseconds(),
		)
	}()

	// Create unified event
	event := events.NewBaseEvent(
		uuid.New().String(),
		"task.assigned", // Custom event type since it's not in the constants
		"kanban-service",
		map[string]interface{}{
			"task_id":           task.ID,
			"board_id":          boardID,
			"title":             task.Title,
			"assigned_to":       assignedTo,
			"assigned_by":       assignedBy,
			"previous_assignee": task.AssignedTo,
			"updated_at":        task.UpdatedAt,
			"metadata":          task.Metadata,
		},
	)

	// Publish to unified event bus
	if err := p.eventBus.Publish(ctx, event); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to publish task assigned event").
			WithComponent("UnifiedTaskEventPublisher").
			WithOperation("PublishTaskAssigned").
			WithDetails("task_id", task.ID)
	}

	return nil
}

// PublishTaskCompleted publishes a task completion event
func (p *UnifiedTaskEventPublisher) PublishTaskCompleted(ctx context.Context, task *Task, boardID, completedBy, reason string) error {
	if err := ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("UnifiedTaskEventPublisher").
			WithOperation("PublishTaskCompleted")
	}

	logger := observability.GetLogger(ctx).
		WithComponent("UnifiedTaskEventPublisher").
		WithOperation("PublishTaskCompleted")

	startTime := time.Now()
	defer func() {
		duration := time.Since(startTime)
		logger.Info("Task completed event published",
			"task_id", task.ID,
			"board_id", boardID,
			"completed_by", completedBy,
			"duration_ms", duration.Milliseconds(),
		)
	}()

	// Create unified event
	event := events.NewBaseEvent(
		uuid.New().String(),
		events.EventTypeTaskCompleted,
		"kanban-service",
		map[string]interface{}{
			"task_id":      task.ID,
			"board_id":     boardID,
			"title":        task.Title,
			"completed_by": completedBy,
			"completed_at": time.Now().UTC(),
			"reason":       reason,
			"metadata":     task.Metadata,
		},
	)

	// Publish to unified event bus
	if err := p.eventBus.Publish(ctx, event); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to publish task completed event").
			WithComponent("UnifiedTaskEventPublisher").
			WithOperation("PublishTaskCompleted").
			WithDetails("task_id", task.ID)
	}

	return nil
}

// PublishTaskDeleted publishes a task deletion event
func (p *UnifiedTaskEventPublisher) PublishTaskDeleted(ctx context.Context, taskID, boardID, deletedBy, reason string) error {
	if err := ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("UnifiedTaskEventPublisher").
			WithOperation("PublishTaskDeleted")
	}

	logger := observability.GetLogger(ctx).
		WithComponent("UnifiedTaskEventPublisher").
		WithOperation("PublishTaskDeleted")

	startTime := time.Now()
	defer func() {
		duration := time.Since(startTime)
		logger.Info("Task deleted event published",
			"task_id", taskID,
			"board_id", boardID,
			"deleted_by", deletedBy,
			"duration_ms", duration.Milliseconds(),
		)
	}()

	// Create unified event
	event := events.NewBaseEvent(
		uuid.New().String(),
		"task.deleted", // Custom event type since it's not in the constants
		"kanban-service",
		map[string]interface{}{
			"task_id":    taskID,
			"board_id":   boardID,
			"deleted_by": deletedBy,
			"deleted_at": time.Now().UTC(),
			"reason":     reason,
		},
	)

	// Publish to unified event bus
	if err := p.eventBus.Publish(ctx, event); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to publish task deleted event").
			WithComponent("UnifiedTaskEventPublisher").
			WithOperation("PublishTaskDeleted").
			WithDetails("task_id", taskID)
	}

	return nil
}

// PublishTaskUpdated publishes a generic task update event
func (p *UnifiedTaskEventPublisher) PublishTaskUpdated(ctx context.Context, task *Task, boardID, updatedBy string, changes map[string]interface{}) error {
	if err := ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("UnifiedTaskEventPublisher").
			WithOperation("PublishTaskUpdated")
	}

	logger := observability.GetLogger(ctx).
		WithComponent("UnifiedTaskEventPublisher").
		WithOperation("PublishTaskUpdated")

	startTime := time.Now()
	defer func() {
		duration := time.Since(startTime)
		logger.Info("Task updated event published",
			"task_id", task.ID,
			"board_id", boardID,
			"updated_by", updatedBy,
			"duration_ms", duration.Milliseconds(),
		)
	}()

	// Create unified event
	event := events.NewBaseEvent(
		uuid.New().String(),
		events.EventTypeTaskUpdated,
		"kanban-service",
		map[string]interface{}{
			"task_id":    task.ID,
			"board_id":   boardID,
			"title":      task.Title,
			"updated_by": updatedBy,
			"updated_at": task.UpdatedAt,
			"changes":    changes,
			"metadata":   task.Metadata,
		},
	)

	// Publish to unified event bus
	if err := p.eventBus.Publish(ctx, event); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to publish task updated event").
			WithComponent("UnifiedTaskEventPublisher").
			WithOperation("PublishTaskUpdated").
			WithDetails("task_id", task.ID)
	}

	return nil
}

// PublishTaskBlocked publishes a task blocked event
func (p *UnifiedTaskEventPublisher) PublishTaskBlocked(ctx context.Context, task *Task, boardID, blockedBy, reason string) error {
	if err := ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("UnifiedTaskEventPublisher").
			WithOperation("PublishTaskBlocked")
	}

	logger := observability.GetLogger(ctx).
		WithComponent("UnifiedTaskEventPublisher").
		WithOperation("PublishTaskBlocked")

	startTime := time.Now()
	defer func() {
		duration := time.Since(startTime)
		logger.Info("Task blocked event published",
			"task_id", task.ID,
			"board_id", boardID,
			"blocked_by", blockedBy,
			"duration_ms", duration.Milliseconds(),
		)
	}()

	// Create unified event
	event := events.NewBaseEvent(
		uuid.New().String(),
		"task.blocked", // Custom event type
		"kanban-service",
		map[string]interface{}{
			"task_id":    task.ID,
			"board_id":   boardID,
			"title":      task.Title,
			"blocked_by": blockedBy,
			"reason":     reason,
			"blocked_at": time.Now().UTC(),
			"metadata":   task.Metadata,
		},
	)

	// Publish to unified event bus
	if err := p.eventBus.Publish(ctx, event); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to publish task blocked event").
			WithComponent("UnifiedTaskEventPublisher").
			WithOperation("PublishTaskBlocked").
			WithDetails("task_id", task.ID)
	}

	return nil
}

// PublishTaskUnblocked publishes a task unblocked event
func (p *UnifiedTaskEventPublisher) PublishTaskUnblocked(ctx context.Context, task *Task, boardID, unblockedBy, reason string) error {
	if err := ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("UnifiedTaskEventPublisher").
			WithOperation("PublishTaskUnblocked")
	}

	logger := observability.GetLogger(ctx).
		WithComponent("UnifiedTaskEventPublisher").
		WithOperation("PublishTaskUnblocked")

	startTime := time.Now()
	defer func() {
		duration := time.Since(startTime)
		logger.Info("Task unblocked event published",
			"task_id", task.ID,
			"board_id", boardID,
			"unblocked_by", unblockedBy,
			"duration_ms", duration.Milliseconds(),
		)
	}()

	// Create unified event
	event := events.NewBaseEvent(
		uuid.New().String(),
		"task.unblocked", // Custom event type
		"kanban-service",
		map[string]interface{}{
			"task_id":      task.ID,
			"board_id":     boardID,
			"title":        task.Title,
			"unblocked_by": unblockedBy,
			"reason":       reason,
			"unblocked_at": time.Now().UTC(),
			"metadata":     task.Metadata,
		},
	)

	// Publish to unified event bus
	if err := p.eventBus.Publish(ctx, event); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to publish task unblocked event").
			WithComponent("UnifiedTaskEventPublisher").
			WithOperation("PublishTaskUnblocked").
			WithDetails("task_id", task.ID)
	}

	return nil
}
