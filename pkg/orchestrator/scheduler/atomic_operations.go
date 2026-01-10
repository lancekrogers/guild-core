// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package scheduler

import (
	"context"
	"database/sql"
	"time"

	"github.com/lancekrogers/guild-core/pkg/gerror"
	"github.com/lancekrogers/guild-core/pkg/kanban"
)

// AtomicScheduler provides atomic database operations for task scheduling
type AtomicScheduler struct {
	db *sql.DB
}

// NewAtomicScheduler creates a new atomic scheduler
func NewAtomicScheduler(db *sql.DB) (*AtomicScheduler, error) {
	if db == nil {
		return nil, gerror.New(gerror.ErrCodeInvalidInput, "database connection cannot be nil", nil).
			WithComponent("orchestrator.scheduler").
			WithOperation("NewAtomicScheduler")
	}

	return &AtomicScheduler{db: db}, nil
}

// withTransaction executes a function within a database transaction
func (as *AtomicScheduler) withTransaction(ctx context.Context, fn func(tx *sql.Tx) error) error {
	if err := ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("orchestrator.scheduler").
			WithOperation("withTransaction")
	}

	tx, err := as.db.BeginTx(ctx, &sql.TxOptions{
		Isolation: sql.LevelSerializable, // Strongest isolation for critical operations
	})
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to begin transaction").
			WithComponent("orchestrator.scheduler").
			WithOperation("withTransaction")
	}

	defer func() {
		if p := recover(); p != nil {
			// Rollback on panic
			_ = tx.Rollback()
			panic(p) // Re-panic after rollback
		}
	}()

	if err := fn(tx); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			// Log rollback error but return original error
			_ = gerror.Wrap(rbErr, gerror.ErrCodeInternal, "failed to rollback transaction").
				WithComponent("orchestrator.scheduler").
				WithOperation("withTransaction")
		}
		return err
	}

	if err := tx.Commit(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to commit transaction").
			WithComponent("orchestrator.scheduler").
			WithOperation("withTransaction")
	}

	return nil
}

// getTaskForUpdate retrieves a task with row-level lock
func (as *AtomicScheduler) getTaskForUpdate(tx *sql.Tx, taskID string) (*kanban.Task, error) {
	query := `
		SELECT id, title, description, status, priority, assigned_to, 
		       created_at, updated_at, parent_id, progress
		FROM tasks 
		WHERE id = $1 
		FOR UPDATE NOWAIT` // NOWAIT fails fast if row is locked

	row := tx.QueryRow(query, taskID)

	var task kanban.Task
	var assignedTo sql.NullString
	var parentID sql.NullString

	err := row.Scan(
		&task.ID,
		&task.Title,
		&task.Description,
		&task.Status,
		&task.Priority,
		&assignedTo,
		&task.CreatedAt,
		&task.UpdatedAt,
		&parentID,
		&task.Progress,
	)

	if err == sql.ErrNoRows {
		return nil, gerror.New(gerror.ErrCodeNotFound, "task not found", nil).
			WithComponent("orchestrator.scheduler").
			WithOperation("getTaskForUpdate").
			WithDetails("task_id", taskID)
	}

	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to get task for update").
			WithComponent("orchestrator.scheduler").
			WithOperation("getTaskForUpdate").
			WithDetails("task_id", taskID)
	}

	// Handle nullable fields
	if assignedTo.Valid {
		task.AssignedTo = assignedTo.String
	}
	if parentID.Valid {
		task.ParentID = parentID.String
	}

	return &task, nil
}

// AssignTaskAtomic atomically assigns a task to an agent
func (as *AtomicScheduler) AssignTaskAtomic(ctx context.Context, taskID, agentID string) error {
	if err := ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("orchestrator.scheduler").
			WithOperation("AssignTaskAtomic")
	}

	return as.withTransaction(ctx, func(tx *sql.Tx) error {
		// Get task with lock
		task, err := as.getTaskForUpdate(tx, taskID)
		if err != nil {
			return err
		}

		// Check if already assigned
		if task.AssignedTo != "" {
			return gerror.New(gerror.ErrCodeConflict, "task already assigned", nil).
				WithComponent("orchestrator.scheduler").
				WithOperation("AssignTaskAtomic").
				WithDetails("task_id", taskID).
				WithDetails("existing_assignee", task.AssignedTo)
		}

		// Update assignment
		updateQuery := `
			UPDATE tasks 
			SET assigned_to = $1, 
			    status = $2,
			    updated_at = $3
			WHERE id = $4 AND assigned_to IS NULL`

		result, err := tx.Exec(updateQuery, agentID, kanban.StatusInProgress, time.Now().UTC(), taskID)
		if err != nil {
			return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to update task assignment").
				WithComponent("orchestrator.scheduler").
				WithOperation("AssignTaskAtomic").
				WithDetails("task_id", taskID).
				WithDetails("agent_id", agentID)
		}

		rowsAffected, err := result.RowsAffected()
		if err != nil {
			return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to get rows affected").
				WithComponent("orchestrator.scheduler").
				WithOperation("AssignTaskAtomic")
		}

		if rowsAffected == 0 {
			return gerror.New(gerror.ErrCodeConflict, "task assignment race condition detected", nil).
				WithComponent("orchestrator.scheduler").
				WithOperation("AssignTaskAtomic").
				WithDetails("task_id", taskID)
		}

		// Add history entry
		historyQuery := `
			INSERT INTO task_history (task_id, timestamp, changed_by, to_status, to_assignee, comment)
			VALUES ($1, $2, $3, $4, $5, $6)`

		_, err = tx.Exec(historyQuery, taskID, time.Now().UTC(), "scheduler",
			kanban.StatusInProgress, agentID, "Task assigned by orchestrator")
		if err != nil {
			return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to add history entry").
				WithComponent("orchestrator.scheduler").
				WithOperation("AssignTaskAtomic").
				WithDetails("task_id", taskID)
		}

		return nil
	})
}

// UpdateTaskStatusAtomic atomically updates task status
func (as *AtomicScheduler) UpdateTaskStatusAtomic(ctx context.Context, taskID string, status kanban.TaskStatus, comment string) error {
	if err := ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("orchestrator.scheduler").
			WithOperation("UpdateTaskStatusAtomic")
	}

	return as.withTransaction(ctx, func(tx *sql.Tx) error {
		// Get current task state with lock
		task, err := as.getTaskForUpdate(tx, taskID)
		if err != nil {
			return err
		}

		// Validate status transition
		if !isValidStatusTransition(task.Status, status) {
			return gerror.New(gerror.ErrCodeValidation, "invalid status transition", nil).
				WithComponent("orchestrator.scheduler").
				WithOperation("UpdateTaskStatusAtomic").
				WithDetails("task_id", taskID).
				WithDetails("from_status", string(task.Status)).
				WithDetails("to_status", string(status))
		}

		// Update status
		updateQuery := `
			UPDATE tasks 
			SET status = $1, 
			    updated_at = $2
			WHERE id = $3`

		_, err = tx.Exec(updateQuery, status, time.Now().UTC(), taskID)
		if err != nil {
			return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to update task status").
				WithComponent("orchestrator.scheduler").
				WithOperation("UpdateTaskStatusAtomic").
				WithDetails("task_id", taskID).
				WithDetails("status", string(status))
		}

		// Add history entry
		historyQuery := `
			INSERT INTO task_history (task_id, timestamp, changed_by, from_status, to_status, comment)
			VALUES ($1, $2, $3, $4, $5, $6)`

		_, err = tx.Exec(historyQuery, taskID, time.Now().UTC(), "scheduler",
			task.Status, status, comment)
		if err != nil {
			return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to add history entry").
				WithComponent("orchestrator.scheduler").
				WithOperation("UpdateTaskStatusAtomic").
				WithDetails("task_id", taskID)
		}

		return nil
	})
}

// BatchAssignTasks atomically assigns multiple tasks
func (as *AtomicScheduler) BatchAssignTasks(ctx context.Context, assignments map[string]string) error {
	if err := ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("orchestrator.scheduler").
			WithOperation("BatchAssignTasks")
	}

	return as.withTransaction(ctx, func(tx *sql.Tx) error {
		for taskID, agentID := range assignments {
			// Get task with lock
			task, err := as.getTaskForUpdate(tx, taskID)
			if err != nil {
				return err
			}

			// Skip if already assigned
			if task.AssignedTo != "" {
				continue
			}

			// Update assignment
			updateQuery := `
				UPDATE tasks 
				SET assigned_to = $1, 
				    status = $2,
				    updated_at = $3
				WHERE id = $4 AND assigned_to IS NULL`

			_, err = tx.Exec(updateQuery, agentID, kanban.StatusInProgress, time.Now().UTC(), taskID)
			if err != nil {
				return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to update task assignment in batch").
					WithComponent("orchestrator.scheduler").
					WithOperation("BatchAssignTasks").
					WithDetails("task_id", taskID).
					WithDetails("agent_id", agentID)
			}
		}

		return nil
	})
}

// GetTaskForUpdate implements KanbanClient interface
func (as *AtomicScheduler) GetTaskForUpdate(ctx context.Context, taskID string) (*kanban.Task, error) {
	if err := ctx.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("orchestrator.scheduler").
			WithOperation("GetTaskForUpdate")
	}

	var task *kanban.Task
	err := as.withTransaction(ctx, func(tx *sql.Tx) error {
		var err error
		task, err = as.getTaskForUpdate(tx, taskID)
		return err
	})

	return task, err
}

// WithTransaction implements KanbanClient interface
func (as *AtomicScheduler) WithTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	return as.withTransaction(ctx, func(tx *sql.Tx) error {
		// Create a new context with the transaction
		txCtx := context.WithValue(ctx, "tx", tx)
		return fn(txCtx)
	})
}

// isValidStatusTransition checks if a status transition is allowed
func isValidStatusTransition(from, to kanban.TaskStatus) bool {
	validTransitions := map[kanban.TaskStatus][]kanban.TaskStatus{
		kanban.StatusBacklog:        {kanban.StatusTodo, kanban.StatusCancelled},
		kanban.StatusTodo:           {kanban.StatusInProgress, kanban.StatusBacklog, kanban.StatusCancelled},
		kanban.StatusInProgress:     {kanban.StatusBlocked, kanban.StatusReadyForReview, kanban.StatusTodo, kanban.StatusCancelled},
		kanban.StatusBlocked:        {kanban.StatusInProgress, kanban.StatusCancelled},
		kanban.StatusReadyForReview: {kanban.StatusDone, kanban.StatusInProgress},
		kanban.StatusDone:           {}, // Terminal state
		kanban.StatusCancelled:      {}, // Terminal state
	}

	allowed, exists := validTransitions[from]
	if !exists {
		return false
	}

	for _, status := range allowed {
		if status == to {
			return true
		}
	}

	return false
}
