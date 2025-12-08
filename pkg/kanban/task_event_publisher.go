// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package kanban

import (
	"context"
	"time"

	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/guild-framework/guild-core/pkg/gerror"
	pb "github.com/guild-framework/guild-core/pkg/grpc/pb/guild/v1"
	"github.com/guild-framework/guild-core/pkg/observability"
	"github.com/guild-framework/guild-core/pkg/orchestrator/interfaces"
)

// TaskEventPublisher publishes task events to both local kanban EventManager and gRPC EventService
type TaskEventPublisher struct {
	eventManager    *EventManager
	grpcEventClient pb.EventServiceClient
	orchestratorBus EventBus // Interface from orchestrator
}

// EventBus interface for orchestrator integration
type EventBus interface {
	Publish(event interfaces.Event)
	Subscribe(eventType interfaces.EventType, handler interfaces.EventHandler)
	SubscribeAll(handler interfaces.EventHandler)
}

// NewTaskEventPublisher creates a new task event publisher
func NewTaskEventPublisher(
	eventManager *EventManager,
	grpcEventClient pb.EventServiceClient,
	orchestratorBus EventBus,
) *TaskEventPublisher {
	return &TaskEventPublisher{
		eventManager:    eventManager,
		grpcEventClient: grpcEventClient,
		orchestratorBus: orchestratorBus,
	}
}

// PublishTaskCreated publishes a task creation event
func (p *TaskEventPublisher) PublishTaskCreated(ctx context.Context, task *Task, boardID, createdBy string) error {
	if err := ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("TaskEventPublisher").
			WithOperation("PublishTaskCreated")
	}

	logger := observability.GetLogger(ctx).
		WithComponent("TaskEventPublisher").
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

	// Create metadata struct
	metadata, err := p.createMetadataStruct(task.Metadata)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create metadata struct").
			WithComponent("TaskEventPublisher").
			WithOperation("PublishTaskCreated")
	}

	// 1. Publish to local kanban EventManager
	if p.eventManager != nil {
		kanbanEvent := &BoardEvent{
			EventType:  EventTaskCreated,
			BoardID:    boardID,
			TaskID:     task.ID,
			Data:       p.taskToEventData(task, map[string]string{"created_by": createdBy}),
			OccurredAt: time.Now().UTC(),
		}

		if err := p.eventManager.PublishEvent(kanbanEvent); err != nil {
			logger.WithError(err).Warn("Failed to publish to kanban event manager")
		}
	}

	// 2. Publish to gRPC EventService
	if p.grpcEventClient != nil {
		taskEvent := &pb.TaskEvent{
			Id:        uuid.New().String(),
			Type:      "task.created",
			Timestamp: timestamppb.Now(),
			Source:    "kanban-service",
			Payload: &pb.TaskEvent_Created{
				Created: &pb.TaskCreated{
					TaskId:      task.ID,
					BoardId:     boardID,
					Title:       task.Title,
					Description: task.Description,
					Status:      string(task.Status),
					Assignee:    task.AssignedTo,
					CreatedBy:   createdBy,
					Metadata:    metadata,
				},
			},
		}

		// Convert TaskEvent to generic Event for gRPC
		eventData, err := p.taskEventToEventData(taskEvent)
		if err != nil {
			return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to convert task event").
				WithComponent("TaskEventPublisher").
				WithOperation("PublishTaskCreated")
		}

		event := &pb.Event{
			Id:        taskEvent.Id,
			Type:      taskEvent.Type,
			Timestamp: taskEvent.Timestamp,
			Source:    taskEvent.Source,
			Data:      eventData,
		}

		_, err = p.grpcEventClient.PublishEvent(ctx, &pb.PublishEventRequest{
			Event: event,
		})
		if err != nil {
			logger.WithError(err).Warn("Failed to publish to gRPC event service")
		}
	}

	// 3. Publish to orchestrator EventBus
	if p.orchestratorBus != nil {
		orchestratorEvent := interfaces.Event{
			ID:        uuid.New().String(),
			Type:      interfaces.EventTypeTaskCreated,
			Timestamp: time.Now().UTC(),
			Source:    "kanban-service",
			Data: map[string]interface{}{
				"task_id":     task.ID,
				"board_id":    boardID,
				"title":       task.Title,
				"description": task.Description,
				"status":      string(task.Status),
				"assignee":    task.AssignedTo,
				"created_by":  createdBy,
				"priority":    string(task.Priority),
				"created_at":  task.CreatedAt,
			},
		}

		p.orchestratorBus.Publish(orchestratorEvent)
	}

	return nil
}

// PublishTaskMoved publishes a task move event
func (p *TaskEventPublisher) PublishTaskMoved(ctx context.Context, task *Task, boardID, fromStatus, toStatus, movedBy, reason string) error {
	if err := ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("TaskEventPublisher").
			WithOperation("PublishTaskMoved")
	}

	logger := observability.GetLogger(ctx).
		WithComponent("TaskEventPublisher").
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

	// Create metadata struct
	metadata, err := p.createMetadataStruct(task.Metadata)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create metadata struct").
			WithComponent("TaskEventPublisher").
			WithOperation("PublishTaskMoved")
	}

	// 1. Publish to local kanban EventManager
	if p.eventManager != nil {
		kanbanEvent := &BoardEvent{
			EventType: EventTaskMoved,
			BoardID:   boardID,
			TaskID:    task.ID,
			Data: p.taskToEventData(task, map[string]string{
				"from_status": fromStatus,
				"to_status":   toStatus,
				"moved_by":    movedBy,
				"reason":      reason,
			}),
			OccurredAt: time.Now().UTC(),
		}

		if err := p.eventManager.PublishEvent(kanbanEvent); err != nil {
			logger.WithError(err).Warn("Failed to publish to kanban event manager")
		}
	}

	// 2. Publish to gRPC EventService
	if p.grpcEventClient != nil {
		taskEvent := &pb.TaskEvent{
			Id:        uuid.New().String(),
			Type:      "task.moved",
			Timestamp: timestamppb.Now(),
			Source:    "kanban-service",
			Payload: &pb.TaskEvent_Moved{
				Moved: &pb.TaskMoved{
					TaskId:     task.ID,
					BoardId:    boardID,
					FromStatus: fromStatus,
					ToStatus:   toStatus,
					MovedBy:    movedBy,
					Reason:     reason,
					Metadata:   metadata,
				},
			},
		}

		// Convert TaskEvent to generic Event for gRPC
		eventData, err := p.taskEventToEventData(taskEvent)
		if err != nil {
			return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to convert task event").
				WithComponent("TaskEventPublisher").
				WithOperation("PublishTaskMoved")
		}

		event := &pb.Event{
			Id:        taskEvent.Id,
			Type:      taskEvent.Type,
			Timestamp: taskEvent.Timestamp,
			Source:    taskEvent.Source,
			Data:      eventData,
		}

		_, err = p.grpcEventClient.PublishEvent(ctx, &pb.PublishEventRequest{
			Event: event,
		})
		if err != nil {
			logger.WithError(err).Warn("Failed to publish to gRPC event service")
		}
	}

	// 3. Publish to orchestrator EventBus
	if p.orchestratorBus != nil {
		orchestratorEvent := interfaces.Event{
			ID:        uuid.New().String(),
			Type:      interfaces.EventTypeTaskStarted, // Map to closest orchestrator event type
			Timestamp: time.Now().UTC(),
			Source:    "kanban-service",
			Data: map[string]interface{}{
				"task_id":     task.ID,
				"board_id":    boardID,
				"from_status": fromStatus,
				"to_status":   toStatus,
				"moved_by":    movedBy,
				"reason":      reason,
			},
		}

		p.orchestratorBus.Publish(orchestratorEvent)
	}

	return nil
}

// PublishTaskUpdated publishes a task update event
func (p *TaskEventPublisher) PublishTaskUpdated(ctx context.Context, task *Task, boardID, updatedBy string, changes map[string]string) error {
	if err := ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("TaskEventPublisher").
			WithOperation("PublishTaskUpdated")
	}

	logger := observability.GetLogger(ctx).
		WithComponent("TaskEventPublisher").
		WithOperation("PublishTaskUpdated")

	startTime := time.Now()
	defer func() {
		duration := time.Since(startTime)
		logger.Info("Task updated event published",
			"task_id", task.ID,
			"board_id", boardID,
			"updated_by", updatedBy,
			"changes_count", len(changes),
			"duration_ms", duration.Milliseconds(),
		)
	}()

	// Create metadata and changes structs
	metadata, err := p.createMetadataStruct(task.Metadata)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create metadata struct").
			WithComponent("TaskEventPublisher").
			WithOperation("PublishTaskUpdated")
	}

	changesStruct, err := p.createMetadataStruct(changes)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create changes struct").
			WithComponent("TaskEventPublisher").
			WithOperation("PublishTaskUpdated")
	}

	// 1. Publish to local kanban EventManager
	if p.eventManager != nil {
		eventData := p.taskToEventData(task, map[string]string{"updated_by": updatedBy})
		for k, v := range changes {
			eventData["change_"+k] = v
		}

		kanbanEvent := &BoardEvent{
			EventType:  EventTaskUpdated,
			BoardID:    boardID,
			TaskID:     task.ID,
			Data:       eventData,
			OccurredAt: time.Now().UTC(),
		}

		if err := p.eventManager.PublishEvent(kanbanEvent); err != nil {
			logger.WithError(err).Warn("Failed to publish to kanban event manager")
		}
	}

	// 2. Publish to gRPC EventService
	if p.grpcEventClient != nil {
		taskEvent := &pb.TaskEvent{
			Id:        uuid.New().String(),
			Type:      "task.updated",
			Timestamp: timestamppb.Now(),
			Source:    "kanban-service",
			Payload: &pb.TaskEvent_Updated{
				Updated: &pb.TaskUpdated{
					TaskId:      task.ID,
					BoardId:     boardID,
					Title:       task.Title,
					Description: task.Description,
					Assignee:    task.AssignedTo,
					UpdatedBy:   updatedBy,
					Changes:     changesStruct,
					Metadata:    metadata,
				},
			},
		}

		// Convert TaskEvent to generic Event for gRPC
		eventData, err := p.taskEventToEventData(taskEvent)
		if err != nil {
			return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to convert task event").
				WithComponent("TaskEventPublisher").
				WithOperation("PublishTaskUpdated")
		}

		event := &pb.Event{
			Id:        taskEvent.Id,
			Type:      taskEvent.Type,
			Timestamp: taskEvent.Timestamp,
			Source:    taskEvent.Source,
			Data:      eventData,
		}

		_, err = p.grpcEventClient.PublishEvent(ctx, &pb.PublishEventRequest{
			Event: event,
		})
		if err != nil {
			logger.WithError(err).Warn("Failed to publish to gRPC event service")
		}
	}

	return nil
}

// PublishTaskCompleted publishes a task completion event
func (p *TaskEventPublisher) PublishTaskCompleted(ctx context.Context, task *Task, boardID, completedBy, notes string) error {
	if err := ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("TaskEventPublisher").
			WithOperation("PublishTaskCompleted")
	}

	logger := observability.GetLogger(ctx).
		WithComponent("TaskEventPublisher").
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

	// Create metadata struct
	metadata, err := p.createMetadataStruct(task.Metadata)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create metadata struct").
			WithComponent("TaskEventPublisher").
			WithOperation("PublishTaskCompleted")
	}

	// 1. Publish to local kanban EventManager
	if p.eventManager != nil {
		kanbanEvent := &BoardEvent{
			EventType: EventTaskStatusChanged, // Use status changed for completion
			BoardID:   boardID,
			TaskID:    task.ID,
			Data: p.taskToEventData(task, map[string]string{
				"completed_by":     completedBy,
				"completion_notes": notes,
				"completed_at":     time.Now().UTC().Format(time.RFC3339),
			}),
			OccurredAt: time.Now().UTC(),
		}

		if err := p.eventManager.PublishEvent(kanbanEvent); err != nil {
			logger.WithError(err).Warn("Failed to publish to kanban event manager")
		}
	}

	// 2. Publish to gRPC EventService
	if p.grpcEventClient != nil {
		taskEvent := &pb.TaskEvent{
			Id:        uuid.New().String(),
			Type:      "task.completed",
			Timestamp: timestamppb.Now(),
			Source:    "kanban-service",
			Payload: &pb.TaskEvent_Completed{
				Completed: &pb.TaskCompleted{
					TaskId:          task.ID,
					BoardId:         boardID,
					CompletedBy:     completedBy,
					CompletedAt:     timestamppb.Now(),
					CompletionNotes: notes,
					Metadata:        metadata,
				},
			},
		}

		// Convert TaskEvent to generic Event for gRPC
		eventData, err := p.taskEventToEventData(taskEvent)
		if err != nil {
			return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to convert task event").
				WithComponent("TaskEventPublisher").
				WithOperation("PublishTaskCompleted")
		}

		event := &pb.Event{
			Id:        taskEvent.Id,
			Type:      taskEvent.Type,
			Timestamp: taskEvent.Timestamp,
			Source:    taskEvent.Source,
			Data:      eventData,
		}

		_, err = p.grpcEventClient.PublishEvent(ctx, &pb.PublishEventRequest{
			Event: event,
		})
		if err != nil {
			logger.WithError(err).Warn("Failed to publish to gRPC event service")
		}
	}

	// 3. Publish to orchestrator EventBus
	if p.orchestratorBus != nil {
		orchestratorEvent := interfaces.Event{
			ID:        uuid.New().String(),
			Type:      interfaces.EventTypeTaskCompleted,
			Timestamp: time.Now().UTC(),
			Source:    "kanban-service",
			Data: map[string]interface{}{
				"task_id":          task.ID,
				"board_id":         boardID,
				"completed_by":     completedBy,
				"completion_notes": notes,
				"completed_at":     time.Now().UTC(),
			},
		}

		p.orchestratorBus.Publish(orchestratorEvent)
	}

	return nil
}

// PublishTaskBlocked publishes a task blocked event
func (p *TaskEventPublisher) PublishTaskBlocked(ctx context.Context, task *Task, boardID, blockedBy, reason string, blockerIDs []string) error {
	if err := ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("TaskEventPublisher").
			WithOperation("PublishTaskBlocked")
	}

	logger := observability.GetLogger(ctx).
		WithComponent("TaskEventPublisher").
		WithOperation("PublishTaskBlocked")

	startTime := time.Now()
	defer func() {
		duration := time.Since(startTime)
		logger.Info("Task blocked event published",
			"task_id", task.ID,
			"board_id", boardID,
			"blocked_by", blockedBy,
			"blocker_count", len(blockerIDs),
			"duration_ms", duration.Milliseconds(),
		)
	}()

	// Create metadata struct
	metadata, err := p.createMetadataStruct(task.Metadata)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create metadata struct").
			WithComponent("TaskEventPublisher").
			WithOperation("PublishTaskBlocked")
	}

	// 1. Publish to local kanban EventManager
	if p.eventManager != nil {
		eventData := p.taskToEventData(task, map[string]string{
			"blocked_by":     blockedBy,
			"blocker_reason": reason,
		})

		kanbanEvent := &BoardEvent{
			EventType:  EventTaskBlocked,
			BoardID:    boardID,
			TaskID:     task.ID,
			Data:       eventData,
			OccurredAt: time.Now().UTC(),
		}

		if err := p.eventManager.PublishEvent(kanbanEvent); err != nil {
			logger.WithError(err).Warn("Failed to publish to kanban event manager")
		}
	}

	// 2. Publish to gRPC EventService
	if p.grpcEventClient != nil {
		taskEvent := &pb.TaskEvent{
			Id:        uuid.New().String(),
			Type:      "task.blocked",
			Timestamp: timestamppb.Now(),
			Source:    "kanban-service",
			Payload: &pb.TaskEvent_Blocked{
				Blocked: &pb.TaskBlocked{
					TaskId:        task.ID,
					BoardId:       boardID,
					BlockedBy:     blockedBy,
					BlockerReason: reason,
					BlockerIds:    blockerIDs,
					Metadata:      metadata,
				},
			},
		}

		// Convert TaskEvent to generic Event for gRPC
		eventData, err := p.taskEventToEventData(taskEvent)
		if err != nil {
			return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to convert task event").
				WithComponent("TaskEventPublisher").
				WithOperation("PublishTaskBlocked")
		}

		event := &pb.Event{
			Id:        taskEvent.Id,
			Type:      taskEvent.Type,
			Timestamp: taskEvent.Timestamp,
			Source:    taskEvent.Source,
			Data:      eventData,
		}

		_, err = p.grpcEventClient.PublishEvent(ctx, &pb.PublishEventRequest{
			Event: event,
		})
		if err != nil {
			logger.WithError(err).Warn("Failed to publish to gRPC event service")
		}
	}

	return nil
}

// PublishTaskUnblocked publishes a task unblocked event
func (p *TaskEventPublisher) PublishTaskUnblocked(ctx context.Context, task *Task, boardID, unblockedBy, reason, resolvedBlockerID string) error {
	if err := ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("TaskEventPublisher").
			WithOperation("PublishTaskUnblocked")
	}

	logger := observability.GetLogger(ctx).
		WithComponent("TaskEventPublisher").
		WithOperation("PublishTaskUnblocked")

	startTime := time.Now()
	defer func() {
		duration := time.Since(startTime)
		logger.Info("Task unblocked event published",
			"task_id", task.ID,
			"board_id", boardID,
			"unblocked_by", unblockedBy,
			"resolved_blocker_id", resolvedBlockerID,
			"duration_ms", duration.Milliseconds(),
		)
	}()

	// Create metadata struct
	metadata, err := p.createMetadataStruct(task.Metadata)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create metadata struct").
			WithComponent("TaskEventPublisher").
			WithOperation("PublishTaskUnblocked")
	}

	// 1. Publish to local kanban EventManager
	if p.eventManager != nil {
		kanbanEvent := &BoardEvent{
			EventType: EventTaskUnblocked,
			BoardID:   boardID,
			TaskID:    task.ID,
			Data: p.taskToEventData(task, map[string]string{
				"unblocked_by":        unblockedBy,
				"unblock_reason":      reason,
				"resolved_blocker_id": resolvedBlockerID,
			}),
			OccurredAt: time.Now().UTC(),
		}

		if err := p.eventManager.PublishEvent(kanbanEvent); err != nil {
			logger.WithError(err).Warn("Failed to publish to kanban event manager")
		}
	}

	// 2. Publish to gRPC EventService
	if p.grpcEventClient != nil {
		taskEvent := &pb.TaskEvent{
			Id:        uuid.New().String(),
			Type:      "task.unblocked",
			Timestamp: timestamppb.Now(),
			Source:    "kanban-service",
			Payload: &pb.TaskEvent_Unblocked{
				Unblocked: &pb.TaskUnblocked{
					TaskId:            task.ID,
					BoardId:           boardID,
					UnblockedBy:       unblockedBy,
					UnblockReason:     reason,
					ResolvedBlockerId: resolvedBlockerID,
					Metadata:          metadata,
				},
			},
		}

		// Convert TaskEvent to generic Event for gRPC
		eventData, err := p.taskEventToEventData(taskEvent)
		if err != nil {
			return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to convert task event").
				WithComponent("TaskEventPublisher").
				WithOperation("PublishTaskUnblocked")
		}

		event := &pb.Event{
			Id:        taskEvent.Id,
			Type:      taskEvent.Type,
			Timestamp: taskEvent.Timestamp,
			Source:    taskEvent.Source,
			Data:      eventData,
		}

		_, err = p.grpcEventClient.PublishEvent(ctx, &pb.PublishEventRequest{
			Event: event,
		})
		if err != nil {
			logger.WithError(err).Warn("Failed to publish to gRPC event service")
		}
	}

	return nil
}

// PublishTaskDeleted publishes a task deletion event
func (p *TaskEventPublisher) PublishTaskDeleted(ctx context.Context, taskID, boardID, deletedBy, reason string) error {
	if err := ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("TaskEventPublisher").
			WithOperation("PublishTaskDeleted")
	}

	logger := observability.GetLogger(ctx).
		WithComponent("TaskEventPublisher").
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

	// 1. Publish to local kanban EventManager
	if p.eventManager != nil {
		kanbanEvent := &BoardEvent{
			EventType: EventTaskDeleted,
			BoardID:   boardID,
			TaskID:    taskID,
			Data: map[string]string{
				"deleted_by":      deletedBy,
				"deletion_reason": reason,
				"deleted_at":      time.Now().UTC().Format(time.RFC3339),
			},
			OccurredAt: time.Now().UTC(),
		}

		if err := p.eventManager.PublishEvent(kanbanEvent); err != nil {
			logger.WithError(err).Warn("Failed to publish to kanban event manager")
		}
	}

	// 2. Publish to gRPC EventService
	if p.grpcEventClient != nil {
		taskEvent := &pb.TaskEvent{
			Id:        uuid.New().String(),
			Type:      "task.deleted",
			Timestamp: timestamppb.Now(),
			Source:    "kanban-service",
			Payload: &pb.TaskEvent_Deleted{
				Deleted: &pb.TaskDeleted{
					TaskId:         taskID,
					BoardId:        boardID,
					DeletedBy:      deletedBy,
					DeletedAt:      timestamppb.Now(),
					DeletionReason: reason,
					Metadata:       &structpb.Struct{}, // Empty metadata since task is deleted
				},
			},
		}

		// Convert TaskEvent to generic Event for gRPC
		eventData, err := p.taskEventToEventData(taskEvent)
		if err != nil {
			return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to convert task event").
				WithComponent("TaskEventPublisher").
				WithOperation("PublishTaskDeleted")
		}

		event := &pb.Event{
			Id:        taskEvent.Id,
			Type:      taskEvent.Type,
			Timestamp: taskEvent.Timestamp,
			Source:    taskEvent.Source,
			Data:      eventData,
		}

		_, err = p.grpcEventClient.PublishEvent(ctx, &pb.PublishEventRequest{
			Event: event,
		})
		if err != nil {
			logger.WithError(err).Warn("Failed to publish to gRPC event service")
		}
	}

	return nil
}

// Helper methods

// createMetadataStruct converts map[string]string to structpb.Struct
func (p *TaskEventPublisher) createMetadataStruct(metadata map[string]string) (*structpb.Struct, error) {
	if metadata == nil {
		return &structpb.Struct{}, nil
	}

	// Convert map[string]string to map[string]interface{}
	interfaceMap := make(map[string]interface{})
	for k, v := range metadata {
		interfaceMap[k] = v
	}

	return structpb.NewStruct(interfaceMap)
}

// taskToEventData converts task to event data for local kanban events
func (p *TaskEventPublisher) taskToEventData(task *Task, extra map[string]string) map[string]string {
	data := map[string]string{
		"task_id":     task.ID,
		"title":       task.Title,
		"description": task.Description,
		"status":      string(task.Status),
		"priority":    string(task.Priority),
		"assignee":    task.AssignedTo,
		"created_at":  task.CreatedAt.Format(time.RFC3339),
		"updated_at":  task.UpdatedAt.Format(time.RFC3339),
		"progress":    string(rune(task.Progress)),
	}

	// Add extra fields
	for k, v := range extra {
		data[k] = v
	}

	// Add task metadata
	for k, v := range task.Metadata {
		data["meta_"+k] = v
	}

	return data
}

// taskEventToEventData converts TaskEvent to generic Event data
func (p *TaskEventPublisher) taskEventToEventData(taskEvent *pb.TaskEvent) (*structpb.Struct, error) {
	// Convert the TaskEvent to a map for generic Event
	eventMap := map[string]interface{}{
		"task_event_id":   taskEvent.Id,
		"task_event_type": taskEvent.Type,
		"source":          taskEvent.Source,
		"timestamp":       taskEvent.Timestamp.AsTime().Format(time.RFC3339),
	}

	// Add payload-specific data based on event type
	switch payload := taskEvent.Payload.(type) {
	case *pb.TaskEvent_Created:
		eventMap["task_id"] = payload.Created.TaskId
		eventMap["board_id"] = payload.Created.BoardId
		eventMap["title"] = payload.Created.Title
		eventMap["status"] = payload.Created.Status
		eventMap["assignee"] = payload.Created.Assignee
		eventMap["created_by"] = payload.Created.CreatedBy
	case *pb.TaskEvent_Moved:
		eventMap["task_id"] = payload.Moved.TaskId
		eventMap["board_id"] = payload.Moved.BoardId
		eventMap["from_status"] = payload.Moved.FromStatus
		eventMap["to_status"] = payload.Moved.ToStatus
		eventMap["moved_by"] = payload.Moved.MovedBy
		eventMap["reason"] = payload.Moved.Reason
	case *pb.TaskEvent_Updated:
		eventMap["task_id"] = payload.Updated.TaskId
		eventMap["board_id"] = payload.Updated.BoardId
		eventMap["title"] = payload.Updated.Title
		eventMap["assignee"] = payload.Updated.Assignee
		eventMap["updated_by"] = payload.Updated.UpdatedBy
	case *pb.TaskEvent_Completed:
		eventMap["task_id"] = payload.Completed.TaskId
		eventMap["board_id"] = payload.Completed.BoardId
		eventMap["completed_by"] = payload.Completed.CompletedBy
		eventMap["completion_notes"] = payload.Completed.CompletionNotes
	case *pb.TaskEvent_Blocked:
		eventMap["task_id"] = payload.Blocked.TaskId
		eventMap["board_id"] = payload.Blocked.BoardId
		eventMap["blocked_by"] = payload.Blocked.BlockedBy
		eventMap["blocker_reason"] = payload.Blocked.BlockerReason
	case *pb.TaskEvent_Unblocked:
		eventMap["task_id"] = payload.Unblocked.TaskId
		eventMap["board_id"] = payload.Unblocked.BoardId
		eventMap["unblocked_by"] = payload.Unblocked.UnblockedBy
		eventMap["unblock_reason"] = payload.Unblocked.UnblockReason
	case *pb.TaskEvent_Deleted:
		eventMap["task_id"] = payload.Deleted.TaskId
		eventMap["board_id"] = payload.Deleted.BoardId
		eventMap["deleted_by"] = payload.Deleted.DeletedBy
		eventMap["deletion_reason"] = payload.Deleted.DeletionReason
	}

	return structpb.NewStruct(eventMap)
}
