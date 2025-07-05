// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package kanban

import (
	"context"
)

// TaskEventPublisherInterface defines the interface for publishing task events
type TaskEventPublisherInterface interface {
	PublishTaskCreated(ctx context.Context, task *Task, boardID, createdBy string) error
	PublishTaskMoved(ctx context.Context, task *Task, boardID, fromStatus, toStatus, movedBy, reason string) error
	PublishTaskAssigned(ctx context.Context, task *Task, boardID, assignedTo, assignedBy string) error
	PublishTaskCompleted(ctx context.Context, task *Task, boardID, completedBy, reason string) error
	PublishTaskDeleted(ctx context.Context, taskID, boardID, deletedBy, reason string) error
	PublishTaskUpdated(ctx context.Context, task *Task, boardID, updatedBy string, changes map[string]interface{}) error
	PublishTaskBlocked(ctx context.Context, task *Task, boardID, blockedBy, reason string) error
	PublishTaskUnblocked(ctx context.Context, task *Task, boardID, unblockedBy, reason string) error
}