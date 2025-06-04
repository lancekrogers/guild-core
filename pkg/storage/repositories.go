package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/guild-ventures/guild-core/pkg/storage/db"
)

// SQLiteTaskRepository implements TaskRepository using SQLite
// Following Guild's repository pattern with proper error handling
type SQLiteTaskRepository struct {
	database *Database
}

// NewSQLiteTaskRepository creates a new SQLite task repository
// Following Guild's constructor pattern
func NewSQLiteTaskRepository(database *Database) TaskRepository {
	return &SQLiteTaskRepository{
		database: database,
	}
}

// CreateTask creates a new task following Guild's context-first pattern
func (r *SQLiteTaskRepository) CreateTask(ctx context.Context, task *Task) error {
	// Convert metadata to JSON
	var metadataJSON []byte
	if task.Metadata != nil {
		var err error
		metadataJSON, err = json.Marshal(task.Metadata)
		if err != nil {
			return fmt.Errorf("failed to marshal task metadata: %w", err)
		}
	}

	// Create task using SQLC - handle type conversions
	storyPoints := int64(task.StoryPoints)
	err := r.database.Queries().CreateTask(ctx, db.CreateTaskParams{
		ID:           task.ID,
		CommissionID: task.CommissionID,
		Title:        task.Title,
		Description:  task.Description,
		Status:       task.Status,
		Column:       task.Column,
		StoryPoints:  &storyPoints,
		Metadata:     metadataJSON,
	})

	if err != nil {
		return fmt.Errorf("failed to create task: %w", err)
	}

	return nil
}

// GetTask retrieves a task by ID following Guild's error wrapping pattern
func (r *SQLiteTaskRepository) GetTask(ctx context.Context, id string) (*Task, error) {
	dbTask, err := r.database.Queries().GetTask(ctx, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("task not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get task: %w", err)
	}

	task, err := r.convertDBTaskToTask(dbTask)
	if err != nil {
		return nil, fmt.Errorf("failed to convert task: %w", err)
	}

	return task, nil
}

// UpdateTask updates an existing task
func (r *SQLiteTaskRepository) UpdateTask(ctx context.Context, task *Task) error {
	// Convert metadata to JSON
	var metadataJSON []byte
	if task.Metadata != nil {
		var err error
		metadataJSON, err = json.Marshal(task.Metadata)
		if err != nil {
			return fmt.Errorf("failed to marshal task metadata: %w", err)
		}
	}

	storyPoints := int64(task.StoryPoints)
	err := r.database.Queries().UpdateTask(ctx, db.UpdateTaskParams{
		Title:        task.Title,
		Description:  task.Description,
		Status:       task.Status,
		Column:       task.Column,
		StoryPoints:  &storyPoints,
		Metadata:     metadataJSON,
		ID:           task.ID,
	})

	if err != nil {
		return fmt.Errorf("failed to update task: %w", err)
	}

	return nil
}

// DeleteTask removes a task by ID
func (r *SQLiteTaskRepository) DeleteTask(ctx context.Context, id string) error {
	if err := r.database.Queries().DeleteTask(ctx, id); err != nil {
		return fmt.Errorf("failed to delete task: %w", err)
	}
	return nil
}

// ListTasks returns all tasks
func (r *SQLiteTaskRepository) ListTasks(ctx context.Context) ([]*Task, error) {
	dbTasks, err := r.database.Queries().ListTasks(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list tasks: %w", err)
	}

	tasks := make([]*Task, len(dbTasks))
	for i, dbTask := range dbTasks {
		task, err := r.convertDBTaskToTask(dbTask)
		if err != nil {
			return nil, fmt.Errorf("failed to convert task %d: %w", i, err)
		}
		tasks[i] = task
	}

	return tasks, nil
}

// ListTasksByStatus returns tasks filtered by status
func (r *SQLiteTaskRepository) ListTasksByStatus(ctx context.Context, status string) ([]*Task, error) {
	dbTasks, err := r.database.Queries().ListTasksByStatus(ctx, status)
	if err != nil {
		return nil, fmt.Errorf("failed to list tasks by status: %w", err)
	}

	tasks := make([]*Task, len(dbTasks))
	for i, dbTask := range dbTasks {
		task, err := r.convertDBTaskToTask(dbTask)
		if err != nil {
			return nil, fmt.Errorf("failed to convert task %d: %w", i, err)
		}
		tasks[i] = task
	}

	return tasks, nil
}

// ListTasksByCommission returns tasks for a specific commission
func (r *SQLiteTaskRepository) ListTasksByCommission(ctx context.Context, commissionID string) ([]*Task, error) {
	dbTasks, err := r.database.Queries().ListTasksByCommission(ctx, commissionID)
	if err != nil {
		return nil, fmt.Errorf("failed to list tasks by commission: %w", err)
	}

	tasks := make([]*Task, len(dbTasks))
	for i, dbTask := range dbTasks {
		task, err := r.convertDBTaskToTask(dbTask)
		if err != nil {
			return nil, fmt.Errorf("failed to convert task %d: %w", i, err)
		}
		tasks[i] = task
	}

	return tasks, nil
}

// ListTasksForKanban returns tasks with agent information for kanban display
func (r *SQLiteTaskRepository) ListTasksForKanban(ctx context.Context, commissionID string) ([]*Task, error) {
	dbTasks, err := r.database.Queries().ListTasksForKanban(ctx, commissionID)
	if err != nil {
		return nil, fmt.Errorf("failed to list tasks for kanban: %w", err)
	}

	tasks := make([]*Task, len(dbTasks))
	for i, dbTask := range dbTasks {
		task, err := r.convertDBKanbanTaskToTask(dbTask)
		if err != nil {
			return nil, fmt.Errorf("failed to convert kanban task %d: %w", i, err)
		}
		tasks[i] = task
	}

	return tasks, nil
}

// AssignTask assigns a task to an agent
func (r *SQLiteTaskRepository) AssignTask(ctx context.Context, taskID, agentID string) error {
	if err := r.database.Queries().AssignTaskToAgent(ctx, db.AssignTaskToAgentParams{
		AssignedAgentID: &agentID,
		ID:              taskID,
	}); err != nil {
		return fmt.Errorf("failed to assign task: %w", err)
	}
	return nil
}

// UpdateTaskStatus updates a task's status
func (r *SQLiteTaskRepository) UpdateTaskStatus(ctx context.Context, taskID, status string) error {
	if err := r.database.Queries().UpdateTaskStatus(ctx, db.UpdateTaskStatusParams{
		Status: status,
		ID:     taskID,
	}); err != nil {
		return fmt.Errorf("failed to update task status: %w", err)
	}
	return nil
}

// UpdateTaskColumn updates a task's column
func (r *SQLiteTaskRepository) UpdateTaskColumn(ctx context.Context, taskID, column string) error {
	if err := r.database.Queries().UpdateTaskColumn(ctx, db.UpdateTaskColumnParams{
		Column: column,
		ID:     taskID,
	}); err != nil {
		return fmt.Errorf("failed to update task column: %w", err)
	}
	return nil
}

// RecordTaskEvent records a task event for audit trail
func (r *SQLiteTaskRepository) RecordTaskEvent(ctx context.Context, event *TaskEvent) error {
	if err := r.database.Queries().RecordTaskEvent(ctx, db.RecordTaskEventParams{
		TaskID:    event.TaskID,
		AgentID:   event.AgentID,
		EventType: event.EventType,
		OldValue:  event.OldValue,
		NewValue:  event.NewValue,
		Reason:    event.Reason,
	}); err != nil {
		return fmt.Errorf("failed to record task event: %w", err)
	}
	return nil
}

// GetTaskHistory returns the history of events for a task
func (r *SQLiteTaskRepository) GetTaskHistory(ctx context.Context, taskID string) ([]*TaskEvent, error) {
	dbEvents, err := r.database.Queries().GetTaskHistory(ctx, taskID)
	if err != nil {
		return nil, fmt.Errorf("failed to get task history: %w", err)
	}

	events := make([]*TaskEvent, len(dbEvents))
	for i, dbEvent := range dbEvents {
		var createdAt time.Time
		if dbEvent.CreatedAt != nil {
			createdAt = *dbEvent.CreatedAt
		}
		
		events[i] = &TaskEvent{
			ID:        dbEvent.ID,
			TaskID:    dbEvent.TaskID,
			AgentID:   dbEvent.AgentID,
			EventType: dbEvent.EventType,
			OldValue:  dbEvent.OldValue,
			NewValue:  dbEvent.NewValue,
			Reason:    dbEvent.Reason,
			CreatedAt: createdAt,
		}
	}

	return events, nil
}

// GetAgentWorkload returns workload statistics for all agents
func (r *SQLiteTaskRepository) GetAgentWorkload(ctx context.Context) ([]*AgentWorkload, error) {
	dbWorkloads, err := r.database.Queries().GetAgentWorkload(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get agent workload: %w", err)
	}

	workloads := make([]*AgentWorkload, len(dbWorkloads))
	for i, dbWorkload := range dbWorkloads {
		var activeTasks int64
		if dbWorkload.ActiveTasks != nil {
			activeTasks = int64(*dbWorkload.ActiveTasks)
		}
		
		workloads[i] = &AgentWorkload{
			ID:          dbWorkload.ID,
			Name:        dbWorkload.Name,
			TaskCount:   dbWorkload.TaskCount,
			ActiveTasks: activeTasks,
		}
	}

	return workloads, nil
}

// Helper methods for converting between DB and domain models
func (r *SQLiteTaskRepository) convertDBTaskToTask(dbTask db.Task) (*Task, error) {
	// Handle nullable fields and type conversions
	var storyPoints int32
	if dbTask.StoryPoints != nil {
		storyPoints = int32(*dbTask.StoryPoints)
	}
	
	var createdAt, updatedAt time.Time
	if dbTask.CreatedAt != nil {
		createdAt = *dbTask.CreatedAt
	}
	if dbTask.UpdatedAt != nil {
		updatedAt = *dbTask.UpdatedAt
	}

	task := &Task{
		ID:              dbTask.ID,
		CommissionID:    dbTask.CommissionID,
		AssignedAgentID: dbTask.AssignedAgentID,
		Title:           dbTask.Title,
		Description:     dbTask.Description,
		Status:          dbTask.Status,
		Column:          dbTask.Column,
		StoryPoints:     storyPoints,
		CreatedAt:       createdAt,
		UpdatedAt:       updatedAt,
	}

	// Parse metadata JSON - handle interface{} type
	if dbTask.Metadata != nil {
		if metadataBytes, ok := dbTask.Metadata.([]byte); ok {
			if err := json.Unmarshal(metadataBytes, &task.Metadata); err != nil {
				return nil, fmt.Errorf("failed to unmarshal task metadata: %w", err)
			}
		}
	}

	return task, nil
}

func (r *SQLiteTaskRepository) convertDBKanbanTaskToTask(dbTask db.ListTasksForKanbanRow) (*Task, error) {
	// Handle nullable fields and type conversions
	var storyPoints int32
	if dbTask.StoryPoints != nil {
		storyPoints = int32(*dbTask.StoryPoints)
	}
	
	var createdAt, updatedAt time.Time
	if dbTask.CreatedAt != nil {
		createdAt = *dbTask.CreatedAt
	}
	if dbTask.UpdatedAt != nil {
		updatedAt = *dbTask.UpdatedAt
	}

	task := &Task{
		ID:              dbTask.ID,
		CommissionID:    dbTask.CommissionID,
		AssignedAgentID: dbTask.AssignedAgentID,
		Title:           dbTask.Title,
		Description:     dbTask.Description,
		Status:          dbTask.Status,
		Column:          dbTask.Column,
		StoryPoints:     storyPoints,
		CreatedAt:       createdAt,
		UpdatedAt:       updatedAt,
		AgentName:       dbTask.AgentName,
		AgentType:       dbTask.AgentType,
	}

	// Parse metadata JSON - handle interface{} type
	if dbTask.Metadata != nil {
		if metadataBytes, ok := dbTask.Metadata.([]byte); ok {
			if err := json.Unmarshal(metadataBytes, &task.Metadata); err != nil {
				return nil, fmt.Errorf("failed to unmarshal task metadata: %w", err)
			}
		}
	}

	return task, nil
}