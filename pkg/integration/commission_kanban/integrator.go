// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package commission_kanban

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/lancekrogers/guild/pkg/commission"
	"github.com/lancekrogers/guild/pkg/gerror"
	"github.com/lancekrogers/guild/pkg/kanban"
	"github.com/lancekrogers/guild/pkg/observability"
	"github.com/lancekrogers/guild/pkg/storage/db"
)

// Integrator provides integration between commission refinement and kanban tasks
type Integrator struct {
	queries     *db.Queries
	kanbanMgr   *kanban.Manager
}

// TaskCreationResult contains the result of task creation
type TaskCreationResult struct {
	TasksCreated     int       `json:"tasks_created"`
	BoardID          string    `json:"board_id"`
	CommissionID     string    `json:"commission_id"`
	TaskIDs          []string  `json:"task_ids"`
	CreatedAt        time.Time `json:"created_at"`
	Errors           []string  `json:"errors,omitempty"`
}

// NewIntegrator creates a new commission-kanban integrator
func NewIntegrator(queries *db.Queries, kanbanMgr *kanban.Manager) *Integrator {
	return &Integrator{
		queries:   queries,
		kanbanMgr: kanbanMgr,
	}
}

// CreateTasksFromRefinedCommission creates kanban tasks from a refined commission
func (i *Integrator) CreateTasksFromRefinedCommission(ctx context.Context, refined *commission.RefinedCommission) (*TaskCreationResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("commission_kanban.integrator").
			WithOperation("CreateTasksFromRefinedCommission")
	}

	logger := observability.GetLogger(ctx)
	ctx = observability.WithComponent(ctx, "commission_kanban.integrator")
	ctx = observability.WithOperation(ctx, "CreateTasksFromRefinedCommission")

	startTime := time.Now()
	logger.InfoContext(ctx, "Creating kanban tasks from refined commission",
		"commission_id", refined.Original.ID,
		"tasks_to_create", len(refined.Tasks))

	// Create or get board for this commission
	boardID, err := i.ensureBoardExists(ctx, refined.Original)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to ensure board exists").
			WithComponent("commission_kanban.integrator").
			WithOperation("CreateTasksFromRefinedCommission").
			WithDetails("commission_id", refined.Original.ID)
	}

	result := &TaskCreationResult{
		TasksCreated: 0,
		BoardID:      boardID,
		CommissionID: refined.Original.ID,
		TaskIDs:      make([]string, 0),
		CreatedAt:    time.Now(),
		Errors:       make([]string, 0),
	}

	// Create tasks in batch
	for _, refinedTask := range refined.Tasks {
		err := i.createTask(ctx, refinedTask, boardID)
		if err != nil {
			errMsg := fmt.Sprintf("Failed to create task %s: %v", refinedTask.ID, err)
			result.Errors = append(result.Errors, errMsg)
			logger.WarnContext(ctx, "Failed to create individual task",
				"task_id", refinedTask.ID,
				"task_title", refinedTask.Title,
				"error", err.Error())
		} else {
			result.TasksCreated++
			result.TaskIDs = append(result.TaskIDs, refinedTask.ID)
			logger.DebugContext(ctx, "Created kanban task",
				"task_id", refinedTask.ID,
				"task_title", refinedTask.Title)
		}
	}

	// Record task relationships and dependencies
	err = i.recordTaskDependencies(ctx, refined.Tasks)
	if err != nil {
		logger.WarnContext(ctx, "Failed to record some task dependencies",
			"error", err.Error())
		result.Errors = append(result.Errors, fmt.Sprintf("Dependencies recording issues: %v", err))
	}

	duration := time.Since(startTime)
	logger.InfoContext(ctx, "Completed kanban task creation",
		"commission_id", refined.Original.ID,
		"tasks_created", result.TasksCreated,
		"tasks_failed", len(result.Errors),
		"board_id", boardID,
		"duration_ms", duration.Milliseconds())

	return result, nil
}

// ensureBoardExists creates or retrieves a board for the commission
func (i *Integrator) ensureBoardExists(ctx context.Context, comm *commission.Commission) (string, error) {
	// Try to find existing board for this commission
	board, err := i.queries.GetBoardByCommission(ctx, comm.ID)
	if err == nil {
		// Board exists, return it
		return board.ID, nil
	}
	
	// Check if it's a "not found" error vs a real database error
	// If it's not a not-found error, return the error
	if !isNotFoundError(err) {
		return "", gerror.Wrap(err, gerror.ErrCodeStorage, "failed to query existing board").
			WithComponent("commission_kanban.integrator").
			WithOperation("ensureBoardExists").
			WithDetails("commission_id", comm.ID)
	}

	// Create new board
	boardID := fmt.Sprintf("board-%s", comm.ID)
	boardName := fmt.Sprintf("%s - Task Board", comm.Title)
	description := fmt.Sprintf("Kanban board for commission: %s", comm.Description)

	err = i.queries.CreateBoard(ctx, db.CreateBoardParams{
		ID:           boardID,
		CommissionID: comm.ID,
		Name:         boardName,
		Description:  &description,
		Status:       "active",
	})

	if err != nil {
		return "", gerror.Wrap(err, gerror.ErrCodeStorage, "failed to create board").
			WithComponent("commission_kanban.integrator").
			WithOperation("ensureBoardExists").
			WithDetails("commission_id", comm.ID).
			WithDetails("board_id", boardID)
	}

	return boardID, nil
}

// createTask creates a single kanban task from a refined task
func (i *Integrator) createTask(ctx context.Context, refinedTask *commission.RefinedTask, boardID string) error {
	// Prepare metadata
	metadata := make(map[string]interface{})
	metadata["task_type"] = refinedTask.Type
	metadata["complexity"] = refinedTask.Complexity
	metadata["estimated_hours"] = refinedTask.EstimatedHours
	
	// Add dependencies if they exist
	if len(refinedTask.Dependencies) > 0 {
		metadata["dependencies"] = refinedTask.Dependencies
	}
	
	// Add original task metadata
	for k, v := range refinedTask.Metadata {
		metadata[k] = v
	}

	// Serialize metadata to JSON
	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeParsing, "failed to serialize task metadata").
			WithComponent("commission_kanban.integrator").
			WithOperation("createTask").
			WithDetails("task_id", refinedTask.ID)
	}

	// Convert complexity to story points
	storyPoints := int64(refinedTask.Complexity)

	// Determine initial column based on task type and status
	column := i.determineInitialColumn(refinedTask)

	// Create the task
	err = i.queries.CreateTask(ctx, db.CreateTaskParams{
		ID:           refinedTask.ID,
		CommissionID: refinedTask.CommissionID,
		BoardID:      &boardID,
		Title:        refinedTask.Title,
		Description:  &refinedTask.Description,
		Status:       refinedTask.Status,
		Column:       column,
		StoryPoints:  &storyPoints,
		Metadata:     string(metadataJSON),
	})

	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to create task in database").
			WithComponent("commission_kanban.integrator").
			WithOperation("createTask").
			WithDetails("task_id", refinedTask.ID).
			WithDetails("board_id", boardID)
	}

	// Assign task to agent if specified
	if refinedTask.AssignedAgent != "" {
		err = i.queries.AssignTaskToAgent(ctx, db.AssignTaskToAgentParams{
			AssignedAgentID: &refinedTask.AssignedAgent,
			ID:              refinedTask.ID,
		})
		
		if err != nil {
			// Log warning but don't fail the task creation
			logger := observability.GetLogger(ctx)
			logger.WarnContext(ctx, "Failed to assign task to agent",
				"task_id", refinedTask.ID,
				"agent_id", refinedTask.AssignedAgent,
				"error", err.Error())
		}
	}

	return nil
}

// determineInitialColumn determines the initial kanban column for a task
func (i *Integrator) determineInitialColumn(task *commission.RefinedTask) string {
	// Map task status to kanban columns
	statusToColumn := map[string]string{
		"todo":        "todo",
		"ready":       "todo",
		"in_progress": "in_progress", 
		"blocked":     "blocked",
		"review":      "review",
		"done":        "done",
		"completed":   "done",
	}

	column, exists := statusToColumn[task.Status]
	if !exists {
		// Default to todo for unknown statuses
		return "todo"
	}

	// Special handling for dependencies
	if len(task.Dependencies) > 0 {
		// Tasks with dependencies start in backlog until dependencies are resolved
		return "backlog"
	}

	// Setup tasks go to todo immediately
	if task.Type == "setup" || task.Metadata["phase"] == "setup" {
		return "todo"
	}

	return column
}

// recordTaskDependencies records task dependencies in metadata and events
func (i *Integrator) recordTaskDependencies(ctx context.Context, tasks []*commission.RefinedTask) error {
	logger := observability.GetLogger(ctx)

	// Create a map of task IDs for validation
	taskMap := make(map[string]*commission.RefinedTask)
	for _, task := range tasks {
		taskMap[task.ID] = task
	}

	for _, task := range tasks {
		if len(task.Dependencies) == 0 {
			continue
		}

		// Validate dependencies exist
		validDependencies := make([]string, 0)
		for _, depID := range task.Dependencies {
			if _, exists := taskMap[depID]; exists {
				validDependencies = append(validDependencies, depID)
			} else {
				logger.WarnContext(ctx, "Task dependency not found in current batch",
					"task_id", task.ID,
					"dependency_id", depID)
			}
		}

		// Record dependency creation event
		if len(validDependencies) > 0 {
			dependencyJSON, _ := json.Marshal(validDependencies)
			
			err := i.queries.RecordTaskEvent(ctx, db.RecordTaskEventParams{
				TaskID:    task.ID,
				AgentID:   nil, // System event
				EventType: "dependencies_set",
				OldValue:  nil,
				NewValue:  stringPtr(string(dependencyJSON)),
				Reason:    stringPtr("Initial task dependencies from commission refinement"),
			})

			if err != nil {
				logger.WarnContext(ctx, "Failed to record dependency event",
					"task_id", task.ID,
					"error", err.Error())
			}
		}
	}

	return nil
}

// GetTasksForCommission retrieves all kanban tasks for a commission
func (i *Integrator) GetTasksForCommission(ctx context.Context, commissionID string) ([]*kanban.Task, error) {
	if err := ctx.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("commission_kanban.integrator").
			WithOperation("GetTasksForCommission")
	}

	logger := observability.GetLogger(ctx)
	logger.DebugContext(ctx, "Retrieving tasks for commission", "commission_id", commissionID)

	// Get tasks from database
	dbTasks, err := i.queries.ListTasksByCommission(ctx, commissionID)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to retrieve tasks from database").
			WithComponent("commission_kanban.integrator").
			WithOperation("GetTasksForCommission").
			WithDetails("commission_id", commissionID)
	}

	// Convert to kanban tasks
	kanbanTasks := make([]*kanban.Task, len(dbTasks))
	for idx, dbTask := range dbTasks {
		kanbanTask, err := i.convertDBTaskToKanban(dbTask)
		if err != nil {
			return nil, gerror.Wrap(err, gerror.ErrCodeParsing, "failed to convert database task to kanban task").
				WithComponent("commission_kanban.integrator").
				WithOperation("GetTasksForCommission").
				WithDetails("task_id", dbTask.ID)
		}
		kanbanTasks[idx] = kanbanTask
	}

	logger.InfoContext(ctx, "Retrieved tasks for commission",
		"commission_id", commissionID,
		"task_count", len(kanbanTasks))

	return kanbanTasks, nil
}

// convertDBTaskToKanban converts a database task to a kanban task
func (i *Integrator) convertDBTaskToKanban(dbTask db.Task) (*kanban.Task, error) {
	task := kanban.NewTask(dbTask.Title, "")
	task.ID = dbTask.ID
	
	// Set description
	if dbTask.Description != nil {
		task.Description = *dbTask.Description
	}

	// Convert status to kanban status
	task.Status = i.convertStatusToKanban(dbTask.Status)

	// Set assigned agent
	if dbTask.AssignedAgentID != nil {
		task.AssignedTo = *dbTask.AssignedAgentID
	}

	// Set story points as estimated hours
	if dbTask.StoryPoints != nil {
		task.EstimatedHours = float64(*dbTask.StoryPoints)
	}

	// Set timestamps
	if dbTask.CreatedAt != nil {
		task.CreatedAt = *dbTask.CreatedAt
	}
	if dbTask.UpdatedAt != nil {
		task.UpdatedAt = *dbTask.UpdatedAt
	}

	// Parse metadata
	if dbTask.Metadata != nil {
		if metadataStr, ok := dbTask.Metadata.(string); ok {
			var metadata map[string]interface{}
			if err := json.Unmarshal([]byte(metadataStr), &metadata); err == nil {
				// Convert metadata to string map for kanban task
				task.Metadata = make(map[string]string)
				for k, v := range metadata {
					if str, ok := v.(string); ok {
						task.Metadata[k] = str
					} else {
						task.Metadata[k] = fmt.Sprintf("%v", v)
					}
				}

				// Extract dependencies if present
				if deps, ok := metadata["dependencies"]; ok {
					if depSlice, ok := deps.([]interface{}); ok {
						dependencies := make([]string, len(depSlice))
						for i, dep := range depSlice {
							if depStr, ok := dep.(string); ok {
								dependencies[i] = depStr
							}
						}
						task.Dependencies = dependencies
					}
				}

				// Set priority based on complexity
				if complexity, ok := metadata["complexity"]; ok {
					if complexityFloat, ok := complexity.(float64); ok {
						switch {
						case complexityFloat >= 6:
							task.Priority = kanban.PriorityHigh
						case complexityFloat >= 3:
							task.Priority = kanban.PriorityMedium
						default:
							task.Priority = kanban.PriorityLow
						}
					}
				}
			}
		}
	}

	return task, nil
}

// convertStatusToKanban converts database status to kanban status
func (i *Integrator) convertStatusToKanban(dbStatus string) kanban.TaskStatus {
	statusMap := map[string]kanban.TaskStatus{
		"todo":        kanban.StatusTodo,
		"in_progress": kanban.StatusInProgress,
		"blocked":     kanban.StatusBlocked,
		"review":      kanban.StatusReadyForReview,
		"done":        kanban.StatusDone,
		"completed":   kanban.StatusDone,
		"cancelled":   kanban.StatusCancelled,
	}

	if status, exists := statusMap[dbStatus]; exists {
		return status
	}

	return kanban.StatusTodo // Default fallback
}

// UpdateTaskFromKanban updates a database task from kanban task changes
func (i *Integrator) UpdateTaskFromKanban(ctx context.Context, kanbanTask *kanban.Task) error {
	if err := ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("commission_kanban.integrator").
			WithOperation("UpdateTaskFromKanban")
	}

	logger := observability.GetLogger(ctx)
	logger.DebugContext(ctx, "Updating database task from kanban changes",
		"task_id", kanbanTask.ID)

	// Prepare metadata
	metadata := make(map[string]interface{})
	for k, v := range kanbanTask.Metadata {
		metadata[k] = v
	}
	
	// Add kanban-specific fields
	metadata["dependencies"] = kanbanTask.Dependencies
	metadata["estimated_hours"] = kanbanTask.EstimatedHours

	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeParsing, "failed to serialize task metadata").
			WithComponent("commission_kanban.integrator").
			WithOperation("UpdateTaskFromKanban").
			WithDetails("task_id", kanbanTask.ID)
	}

	// Convert priority to story points (approximation)
	var storyPoints *int64
	switch kanbanTask.Priority {
	case kanban.PriorityHigh:
		sp := int64(8)
		storyPoints = &sp
	case kanban.PriorityMedium:
		sp := int64(5)
		storyPoints = &sp
	case kanban.PriorityLow:
		sp := int64(3)
		storyPoints = &sp
	}

	// Determine column from status
	column := i.convertKanbanStatusToColumn(kanbanTask.Status)

	// Update the task
	err = i.queries.UpdateTask(ctx, db.UpdateTaskParams{
		Title:       kanbanTask.Title,
		Description: &kanbanTask.Description,
		Status:      string(kanbanTask.Status),
		Column:      column,
		StoryPoints: storyPoints,
		Metadata:    string(metadataJSON),
		ID:          kanbanTask.ID,
	})

	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to update task in database").
			WithComponent("commission_kanban.integrator").
			WithOperation("UpdateTaskFromKanban").
			WithDetails("task_id", kanbanTask.ID)
	}

	// Update agent assignment if changed
	if kanbanTask.AssignedTo != "" {
		err = i.queries.AssignTaskToAgent(ctx, db.AssignTaskToAgentParams{
			AssignedAgentID: &kanbanTask.AssignedTo,
			ID:              kanbanTask.ID,
		})
		
		if err != nil {
			logger.WarnContext(ctx, "Failed to update task assignment",
				"task_id", kanbanTask.ID,
				"agent_id", kanbanTask.AssignedTo,
				"error", err.Error())
		}
	}

	return nil
}

// convertKanbanStatusToColumn converts kanban status to database column
func (i *Integrator) convertKanbanStatusToColumn(status kanban.TaskStatus) string {
	statusToColumn := map[kanban.TaskStatus]string{
		kanban.StatusBacklog:         "backlog",
		kanban.StatusTodo:            "todo",
		kanban.StatusInProgress:      "in_progress",
		kanban.StatusBlocked:         "blocked",
		kanban.StatusReadyForReview:  "review",
		kanban.StatusDone:            "done",
		kanban.StatusCancelled:       "cancelled",
	}

	if column, exists := statusToColumn[status]; exists {
		return column
	}

	return "todo" // Default fallback
}

// DeleteTasksForCommission deletes all tasks associated with a commission
func (i *Integrator) DeleteTasksForCommission(ctx context.Context, commissionID string) error {
	if err := ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("commission_kanban.integrator").
			WithOperation("DeleteTasksForCommission")
	}

	logger := observability.GetLogger(ctx)
	logger.InfoContext(ctx, "Deleting all tasks for commission", "commission_id", commissionID)

	// Get all tasks for the commission
	tasks, err := i.queries.ListTasksByCommission(ctx, commissionID)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to list tasks for deletion").
			WithComponent("commission_kanban.integrator").
			WithOperation("DeleteTasksForCommission").
			WithDetails("commission_id", commissionID)
	}

	// Delete each task and its events
	for _, task := range tasks {
		// Delete task events first
		err = i.queries.DeleteTaskEvents(ctx, task.ID)
		if err != nil {
			logger.WarnContext(ctx, "Failed to delete task events",
				"task_id", task.ID,
				"error", err.Error())
		}

		// Delete the task
		err = i.queries.DeleteTask(ctx, task.ID)
		if err != nil {
			logger.WarnContext(ctx, "Failed to delete task",
				"task_id", task.ID,
				"error", err.Error())
		}
	}

	logger.InfoContext(ctx, "Completed task deletion for commission",
		"commission_id", commissionID,
		"tasks_deleted", len(tasks))

	return nil
}

// Utility functions

func stringPtr(s string) *string {
	return &s
}

// ConvertRefinedCommissionToKanbanTasks converts refined tasks to kanban tasks for database storage
func (i *Integrator) ConvertRefinedCommissionToKanbanTasks(refinedCommission *commission.RefinedCommission) []*kanban.Task {
	tasks := make([]*kanban.Task, len(refinedCommission.Tasks))
	
	for idx, refinedTask := range refinedCommission.Tasks {
		task := kanban.NewTask(refinedTask.Title, refinedTask.Description)
		task.ID = refinedTask.ID
		task.Status = kanban.StatusTodo // Default to todo
		task.AssignedTo = refinedTask.AssignedAgent
		task.EstimatedHours = refinedTask.EstimatedHours
		task.Dependencies = refinedTask.Dependencies
		task.CreatedAt = refinedTask.CreatedAt
		task.UpdatedAt = refinedTask.UpdatedAt
		
		// Set priority based on complexity
		switch {
		case refinedTask.Complexity >= 6:
			task.Priority = kanban.PriorityHigh
		case refinedTask.Complexity >= 3:
			task.Priority = kanban.PriorityMedium
		default:
			task.Priority = kanban.PriorityLow
		}
		
		// Add metadata
		task.Metadata = make(map[string]string)
		task.Metadata["commission_id"] = refinedTask.CommissionID
		task.Metadata["task_type"] = refinedTask.Type
		task.Metadata["complexity"] = fmt.Sprintf("%d", refinedTask.Complexity)
		for k, v := range refinedTask.Metadata {
			task.Metadata[k] = v
		}
		
		tasks[idx] = task
	}
	
	return tasks
}

// isNotFoundError checks if an error is a "not found" error
func isNotFoundError(err error) bool {
	return err == sql.ErrNoRows
}