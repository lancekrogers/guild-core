// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/lancekrogers/guild/internal/ui/chat/session"
	"github.com/lancekrogers/guild/pkg/gerror"
	"github.com/lancekrogers/guild/pkg/storage/db"
)

// SQLiteTaskRepository implements TaskRepository using SQLite
// Following Guild's repository pattern with proper error handling
type SQLiteTaskRepository struct {
	database *Database
}

// newSQLiteTaskRepository creates a new SQLite task repository (private constructor)
// Following Guild's constructor pattern
func newSQLiteTaskRepository(database *Database) TaskRepository {
	return &SQLiteTaskRepository{
		database: database,
	}
}

// DefaultTaskRepositoryFactory creates a task repository for registry use
func DefaultTaskRepositoryFactory(database *Database) TaskRepository {
	return newSQLiteTaskRepository(database)
}

// CreateTask creates a new task following Guild's context-first pattern
func (r *SQLiteTaskRepository) CreateTask(ctx context.Context, task *Task) error {
	// Convert metadata to JSON
	var metadataJSON []byte
	if task.Metadata != nil {
		var err error
		metadataJSON, err = json.Marshal(task.Metadata)
		if err != nil {
			return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to marshal task metadata").
				WithComponent("SQLiteTaskRepository")
		}
	}

	// Create task using SQLC - handle type conversions
	storyPoints := int64(task.StoryPoints)
	err := r.database.Queries().CreateTask(ctx, db.CreateTaskParams{
		ID:           task.ID,
		CommissionID: task.CommissionID, // Required commission ID
		BoardID:      task.BoardID,      // Use BoardID (nullable)
		Title:        task.Title,
		Description:  task.Description,
		Status:       task.Status,
		Column:       task.Column, // Kanban column
		StoryPoints:  &storyPoints,
		Metadata:     metadataJSON,
	})

	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to create task").
			WithComponent("SQLiteTaskRepository").
			WithOperation("CreateTask")
	}

	return nil
}

// GetTask retrieves a task by ID following Guild's error wrapping pattern
func (r *SQLiteTaskRepository) GetTask(ctx context.Context, id string) (*Task, error) {
	dbTask, err := r.database.Queries().GetTask(ctx, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, gerror.New(gerror.ErrCodeNotFound, "task not found", nil).
				WithComponent("SQLiteTaskRepository").
				WithOperation("GetTask").
				WithDetails("task_id", id)
		}
		return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to get task").
			WithComponent("SQLiteTaskRepository").
			WithOperation("GetTask")
	}

	task, err := r.convertDBTaskToTask(dbTask)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to convert task").
			WithComponent("SQLiteTaskRepository").
			WithOperation("GetTask")
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
			return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to marshal task metadata").
				WithComponent("SQLiteTaskRepository")
		}
	}

	storyPoints := int64(task.StoryPoints)
	err := r.database.Queries().UpdateTask(ctx, db.UpdateTaskParams{
		Title:       task.Title,
		Description: task.Description,
		Status:      task.Status,
		Column:      task.Column,
		StoryPoints: &storyPoints,
		Metadata:    metadataJSON,
		ID:          task.ID,
	})

	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to update task").
			WithComponent("SQLiteTaskRepository").
			WithOperation("UpdateTask")
	}

	return nil
}

// DeleteTask removes a task by ID
func (r *SQLiteTaskRepository) DeleteTask(ctx context.Context, id string) error {
	// Delete task events first to avoid foreign key constraint issues
	if err := r.database.Queries().DeleteTaskEvents(ctx, id); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to delete task events").
			WithComponent("SQLiteTaskRepository").
			WithOperation("DeleteTask")
	}

	// Then delete the task
	if err := r.database.Queries().DeleteTask(ctx, id); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to delete task").
			WithComponent("SQLiteTaskRepository").
			WithOperation("DeleteTask")
	}
	return nil
}

// ListTasks returns all tasks
func (r *SQLiteTaskRepository) ListTasks(ctx context.Context) ([]*Task, error) {
	dbTasks, err := r.database.Queries().ListTasks(ctx)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to list tasks").
			WithComponent("SQLiteTaskRepository").
			WithOperation("ListTasks")
	}

	tasks := make([]*Task, len(dbTasks))
	for i, dbTask := range dbTasks {
		task, err := r.convertDBTaskToTask(dbTask)
		if err != nil {
			return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to convert task").
				WithComponent("SQLiteTaskRepository").
				WithOperation("ListTasks").
				WithDetails("task_index", i)
		}
		tasks[i] = task
	}

	return tasks, nil
}

// ListTasksByStatus returns tasks filtered by status
func (r *SQLiteTaskRepository) ListTasksByStatus(ctx context.Context, status string) ([]*Task, error) {
	dbTasks, err := r.database.Queries().ListTasksByStatus(ctx, status)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to list tasks by status").
			WithComponent("SQLiteTaskRepository").
			WithOperation("ListTasksByStatus").
			WithDetails("status", status)
	}

	tasks := make([]*Task, len(dbTasks))
	for i, dbTask := range dbTasks {
		task, err := r.convertDBTaskToTask(dbTask)
		if err != nil {
			return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to convert task").
				WithComponent("SQLiteTaskRepository").
				WithOperation("ListTasks").
				WithDetails("task_index", i)
		}
		tasks[i] = task
	}

	return tasks, nil
}

// ListTasksByCommission returns tasks for a specific commission (DEPRECATED - use ListTasksByBoard)
func (r *SQLiteTaskRepository) ListTasksByCommission(ctx context.Context, commissionID string) ([]*Task, error) {
	dbTasks, err := r.database.Queries().ListTasksByCommission(ctx, commissionID)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to list tasks by commission").
			WithComponent("SQLiteTaskRepository").
			WithOperation("ListTasksByCommission").
			WithDetails("commission_id", commissionID)
	}

	tasks := make([]*Task, len(dbTasks))
	for i, dbTask := range dbTasks {
		task, err := r.convertDBTaskToTask(dbTask)
		if err != nil {
			return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to convert task").
				WithComponent("SQLiteTaskRepository").
				WithOperation("ListTasks").
				WithDetails("task_index", i)
		}
		tasks[i] = task
	}

	return tasks, nil
}

// ListTasksByBoard returns tasks for a specific board
func (r *SQLiteTaskRepository) ListTasksByBoard(ctx context.Context, boardID string) ([]*Task, error) {
	dbTasks, err := r.database.Queries().ListTasksByBoard(ctx, &boardID) // Pass as pointer
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to list tasks by board").
			WithComponent("SQLiteTaskRepository").
			WithOperation("ListTasksByBoard").
			WithDetails("board_id", boardID)
	}

	tasks := make([]*Task, len(dbTasks))
	for i, dbTask := range dbTasks {
		task, err := r.convertDBTaskToTask(dbTask)
		if err != nil {
			return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to convert task").
				WithComponent("SQLiteTaskRepository").
				WithOperation("ListTasks").
				WithDetails("task_index", i)
		}
		tasks[i] = task
	}

	return tasks, nil
}

// ListTasksForKanban returns tasks with agent information for kanban display
func (r *SQLiteTaskRepository) ListTasksForKanban(ctx context.Context, boardID string) ([]*Task, error) {
	dbTasks, err := r.database.Queries().ListTasksForKanban(ctx, &boardID) // Pass as pointer
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to list tasks for kanban").
			WithComponent("SQLiteTaskRepository").
			WithOperation("ListTasksForKanban").
			WithDetails("board_id", boardID)
	}

	tasks := make([]*Task, len(dbTasks))
	for i, dbTask := range dbTasks {
		task, err := r.convertDBKanbanTaskToTask(dbTask)
		if err != nil {
			return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to convert kanban task").
				WithComponent("SQLiteTaskRepository").
				WithOperation("ListTasksForKanban").
				WithDetails("task_index", i)
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
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to assign task").
			WithComponent("SQLiteTaskRepository").
			WithOperation("AssignTask").
			WithDetails("task_id", taskID).
			WithDetails("agent_id", agentID)
	}
	return nil
}

// UpdateTaskStatus updates a task's status
func (r *SQLiteTaskRepository) UpdateTaskStatus(ctx context.Context, taskID, status string) error {
	if err := r.database.Queries().UpdateTaskStatus(ctx, db.UpdateTaskStatusParams{
		Status: status,
		ID:     taskID,
	}); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to update task status").
			WithComponent("SQLiteTaskRepository").
			WithOperation("UpdateTaskStatus").
			WithDetails("task_id", taskID).
			WithDetails("status", status)
	}
	return nil
}

// UpdateTaskColumn updates a task's column
func (r *SQLiteTaskRepository) UpdateTaskColumn(ctx context.Context, taskID, column string) error {
	if err := r.database.Queries().UpdateTaskColumn(ctx, db.UpdateTaskColumnParams{
		Column: column,
		ID:     taskID,
	}); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to update task column").
			WithComponent("SQLiteTaskRepository").
			WithOperation("UpdateTaskColumn").
			WithDetails("task_id", taskID).
			WithDetails("column", column)
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
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to record task event").
			WithComponent("SQLiteTaskRepository").
			WithOperation("RecordTaskEvent").
			WithDetails("task_id", event.TaskID).
			WithDetails("event_type", event.EventType)
	}
	return nil
}

// GetTaskHistory returns the history of events for a task
func (r *SQLiteTaskRepository) GetTaskHistory(ctx context.Context, taskID string) ([]*TaskEvent, error) {
	dbEvents, err := r.database.Queries().GetTaskHistory(ctx, taskID)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to get task history").
			WithComponent("SQLiteTaskRepository").
			WithOperation("GetTaskHistory").
			WithDetails("task_id", taskID)
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
		return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to get agent workload").
			WithComponent("SQLiteTaskRepository").
			WithOperation("GetAgentWorkload")
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
		BoardID:         dbTask.BoardID,      // Nullable BoardID from new schema
		CommissionID:    dbTask.CommissionID, // Keep for backward compatibility
		AssignedAgentID: dbTask.AssignedAgentID,
		Title:           dbTask.Title,
		Description:     dbTask.Description,
		Status:          dbTask.Status,
		Column:          dbTask.Column, // Kanban column field
		StoryPoints:     storyPoints,
		CreatedAt:       createdAt,
		UpdatedAt:       updatedAt,
	}

	// Parse metadata JSON - handle interface{} type
	if dbTask.Metadata != nil {
		if metadataBytes, ok := dbTask.Metadata.([]byte); ok {
			if err := json.Unmarshal(metadataBytes, &task.Metadata); err != nil {
				return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to unmarshal task metadata").
					WithComponent("SQLiteTaskRepository").
					WithOperation("convertDBTaskToTask")
			}
		}
	}

	return task, nil
}

// SQLiteCampaignRepository implements CampaignRepository using SQLite
type SQLiteCampaignRepository struct {
	database *Database
}

// newSQLiteCampaignRepository creates a new SQLite campaign repository (private constructor)
func newSQLiteCampaignRepository(database *Database) CampaignRepository {
	return &SQLiteCampaignRepository{
		database: database,
	}
}

// DefaultCampaignRepositoryFactory creates a campaign repository for registry use
func DefaultCampaignRepositoryFactory(database *Database) CampaignRepository {
	return newSQLiteCampaignRepository(database)
}

// CreateCampaign creates a new campaign
func (r *SQLiteCampaignRepository) CreateCampaign(ctx context.Context, campaign *Campaign) error {
	if err := r.database.Queries().CreateCampaign(ctx, db.CreateCampaignParams{
		ID:     campaign.ID,
		Name:   campaign.Name,
		Status: campaign.Status,
	}); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to create campaign").
			WithComponent("SQLiteCampaignRepository").
			WithOperation("CreateCampaign")
	}
	return nil
}

// GetCampaign retrieves a campaign by ID
func (r *SQLiteCampaignRepository) GetCampaign(ctx context.Context, id string) (*Campaign, error) {
	dbCampaign, err := r.database.Queries().GetCampaign(ctx, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, gerror.New(gerror.ErrCodeNotFound, "campaign not found", nil).
				WithComponent("SQLiteCampaignRepository").
				WithOperation("GetCampaign").
				WithDetails("campaign_id", id)
		}
		return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to get campaign").
			WithComponent("SQLiteCampaignRepository").
			WithOperation("GetCampaign")
	}

	campaign := &Campaign{
		ID:        dbCampaign.ID,
		Name:      dbCampaign.Name,
		Status:    dbCampaign.Status,
		CreatedAt: *dbCampaign.CreatedAt,
		UpdatedAt: *dbCampaign.UpdatedAt,
	}

	return campaign, nil
}

// UpdateCampaignStatus updates a campaign's status
func (r *SQLiteCampaignRepository) UpdateCampaignStatus(ctx context.Context, id, status string) error {
	if err := r.database.Queries().UpdateCampaignStatus(ctx, db.UpdateCampaignStatusParams{
		Status: status,
		ID:     id,
	}); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to update campaign status").
			WithComponent("SQLiteCampaignRepository").
			WithOperation("UpdateCampaignStatus").
			WithDetails("campaign_id", id).
			WithDetails("status", status)
	}
	return nil
}

// DeleteCampaign removes a campaign by ID
func (r *SQLiteCampaignRepository) DeleteCampaign(ctx context.Context, id string) error {
	if err := r.database.Queries().DeleteCampaign(ctx, id); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to delete campaign").
			WithComponent("SQLiteCampaignRepository").
			WithOperation("DeleteCampaign").
			WithDetails("campaign_id", id)
	}
	return nil
}

// ListCampaigns returns all campaigns
func (r *SQLiteCampaignRepository) ListCampaigns(ctx context.Context) ([]*Campaign, error) {
	dbCampaigns, err := r.database.Queries().ListCampaigns(ctx)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to list campaigns").
			WithComponent("SQLiteCampaignRepository").
			WithOperation("ListCampaigns")
	}

	campaigns := make([]*Campaign, len(dbCampaigns))
	for i, dbCampaign := range dbCampaigns {
		campaigns[i] = &Campaign{
			ID:        dbCampaign.ID,
			Name:      dbCampaign.Name,
			Status:    dbCampaign.Status,
			CreatedAt: *dbCampaign.CreatedAt,
			UpdatedAt: *dbCampaign.UpdatedAt,
		}
	}

	return campaigns, nil
}

// SQLiteCommissionRepository implements CommissionRepository using SQLite
type SQLiteCommissionRepository struct {
	database *Database
}

// newSQLiteCommissionRepository creates a new SQLite commission repository (private constructor)
func newSQLiteCommissionRepository(database *Database) CommissionRepository {
	return &SQLiteCommissionRepository{
		database: database,
	}
}

// DefaultCommissionRepositoryFactory creates a commission repository for registry use
func DefaultCommissionRepositoryFactory(database *Database) CommissionRepository {
	return newSQLiteCommissionRepository(database)
}

// CreateCommission creates a new commission
func (r *SQLiteCommissionRepository) CreateCommission(ctx context.Context, commission *Commission) error {
	// Convert context to JSON
	var contextJSON []byte
	if commission.Context != nil {
		var err error
		contextJSON, err = json.Marshal(commission.Context)
		if err != nil {
			return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to marshal commission context").
				WithComponent("SQLiteCommissionRepository").
				WithOperation("CreateCommission")
		}
	}

	if err := r.database.Queries().CreateCommission(ctx, db.CreateCommissionParams{
		ID:          commission.ID,
		CampaignID:  commission.CampaignID,
		Title:       commission.Title,
		Description: commission.Description,
		Domain:      commission.Domain,
		Context:     contextJSON,
		Status:      commission.Status,
	}); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to create commission").
			WithComponent("SQLiteCommissionRepository").
			WithOperation("CreateCommission")
	}
	return nil
}

// GetCommission retrieves a commission by ID
func (r *SQLiteCommissionRepository) GetCommission(ctx context.Context, id string) (*Commission, error) {
	dbCommission, err := r.database.Queries().GetCommission(ctx, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, gerror.New(gerror.ErrCodeNotFound, "commission not found", nil).
				WithComponent("SQLiteCommissionRepository").
				WithOperation("GetCommission").
				WithDetails("commission_id", id)
		}
		return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to get commission").
			WithComponent("SQLiteCommissionRepository").
			WithOperation("GetCommission")
	}

	commission := &Commission{
		ID:          dbCommission.ID,
		CampaignID:  dbCommission.CampaignID,
		Title:       dbCommission.Title,
		Description: dbCommission.Description,
		Domain:      dbCommission.Domain,
		Status:      dbCommission.Status,
		CreatedAt:   *dbCommission.CreatedAt,
	}

	// Parse context JSON
	if dbCommission.Context != nil {
		if contextBytes, ok := dbCommission.Context.([]byte); ok {
			if err := json.Unmarshal(contextBytes, &commission.Context); err != nil {
				return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to unmarshal commission context").
					WithComponent("SQLiteCommissionRepository").
					WithOperation("GetCommission")
			}
		}
	}

	return commission, nil
}

// UpdateCommissionStatus updates a commission's status
func (r *SQLiteCommissionRepository) UpdateCommissionStatus(ctx context.Context, id, status string) error {
	if err := r.database.Queries().UpdateCommissionStatus(ctx, db.UpdateCommissionStatusParams{
		Status: status,
		ID:     id,
	}); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to update commission status").
			WithComponent("SQLiteCommissionRepository").
			WithOperation("UpdateCommissionStatus").
			WithDetails("commission_id", id).
			WithDetails("status", status)
	}
	return nil
}

// DeleteCommission removes a commission by ID
func (r *SQLiteCommissionRepository) DeleteCommission(ctx context.Context, id string) error {
	if err := r.database.Queries().DeleteCommission(ctx, id); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to delete commission").
			WithComponent("SQLiteCommissionRepository").
			WithOperation("DeleteCommission").
			WithDetails("commission_id", id)
	}
	return nil
}

// ListCommissionsByCampaign returns commissions for a specific campaign
func (r *SQLiteCommissionRepository) ListCommissionsByCampaign(ctx context.Context, campaignID string) ([]*Commission, error) {
	dbCommissions, err := r.database.Queries().ListCommissionsByCampaign(ctx, campaignID)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to list commissions by campaign").
			WithComponent("SQLiteCommissionRepository").
			WithOperation("ListCommissionsByCampaign").
			WithDetails("campaign_id", campaignID)
	}

	commissions := make([]*Commission, len(dbCommissions))
	for i, dbCommission := range dbCommissions {
		commissions[i] = &Commission{
			ID:          dbCommission.ID,
			CampaignID:  dbCommission.CampaignID,
			Title:       dbCommission.Title,
			Description: dbCommission.Description,
			Domain:      dbCommission.Domain,
			Status:      dbCommission.Status,
			CreatedAt:   *dbCommission.CreatedAt,
		}

		// Parse context JSON
		if dbCommission.Context != nil {
			if contextBytes, ok := dbCommission.Context.([]byte); ok {
				if err := json.Unmarshal(contextBytes, &commissions[i].Context); err != nil {
					return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to unmarshal commission context").
						WithComponent("SQLiteCommissionRepository").
						WithOperation("ListCommissionsByCampaign").
						WithDetails("commission_index", i)
				}
			}
		}
	}

	return commissions, nil
}

// SQLiteAgentRepository implements AgentRepository using SQLite
type SQLiteAgentRepository struct {
	database *Database
}

// newSQLiteAgentRepository creates a new SQLite agent repository (private constructor)
func newSQLiteAgentRepository(database *Database) AgentRepository {
	return &SQLiteAgentRepository{
		database: database,
	}
}

// DefaultAgentRepositoryFactory creates a agent repository for registry use
func DefaultAgentRepositoryFactory(database *Database) AgentRepository {
	return newSQLiteAgentRepository(database)
}

// CreateAgent creates a new agent
func (r *SQLiteAgentRepository) CreateAgent(ctx context.Context, agent *Agent) error {
	// Convert capabilities and tools to JSON
	var capabilitiesJSON, toolsJSON []byte
	var err error

	if agent.Capabilities != nil {
		capabilitiesJSON, err = json.Marshal(agent.Capabilities)
		if err != nil {
			return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to marshal agent capabilities").
				WithComponent("SQLiteAgentRepository").
				WithOperation("CreateAgent")
		}
	}

	if agent.Tools != nil {
		toolsJSON, err = json.Marshal(agent.Tools)
		if err != nil {
			return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to marshal agent tools").
				WithComponent("SQLiteAgentRepository").
				WithOperation("CreateAgent")
		}
	}

	costMagnitude := int64(agent.CostMagnitude)
	if err := r.database.Queries().CreateAgent(ctx, db.CreateAgentParams{
		ID:            agent.ID,
		Name:          agent.Name,
		Type:          agent.Type,
		Provider:      agent.Provider,
		Model:         agent.Model,
		Capabilities:  capabilitiesJSON,
		Tools:         toolsJSON,
		CostMagnitude: &costMagnitude,
	}); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to create agent").
			WithComponent("SQLiteAgentRepository").
			WithOperation("CreateAgent")
	}
	return nil
}

// GetAgent retrieves an agent by ID
func (r *SQLiteAgentRepository) GetAgent(ctx context.Context, id string) (*Agent, error) {
	dbAgent, err := r.database.Queries().GetAgent(ctx, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, gerror.New(gerror.ErrCodeNotFound, "agent not found", nil).
				WithComponent("SQLiteAgentRepository").
				WithOperation("GetAgent").
				WithDetails("agent_id", id)
		}
		return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to get agent").
			WithComponent("SQLiteAgentRepository").
			WithOperation("GetAgent")
	}

	// Handle cost magnitude conversion
	var costMagnitude int32
	if dbAgent.CostMagnitude != nil {
		costMagnitude = int32(*dbAgent.CostMagnitude)
	}

	agent := &Agent{
		ID:            dbAgent.ID,
		Name:          dbAgent.Name,
		Type:          dbAgent.Type,
		Provider:      dbAgent.Provider,
		Model:         dbAgent.Model,
		CostMagnitude: costMagnitude,
		CreatedAt:     *dbAgent.CreatedAt,
	}

	// Parse capabilities JSON
	if dbAgent.Capabilities != nil {
		if capabilitiesBytes, ok := dbAgent.Capabilities.([]byte); ok {
			if err := json.Unmarshal(capabilitiesBytes, &agent.Capabilities); err != nil {
				return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to unmarshal agent capabilities").
					WithComponent("SQLiteAgentRepository").
					WithOperation("GetAgent")
			}
		}
	}

	// Parse tools JSON
	if dbAgent.Tools != nil {
		if toolsBytes, ok := dbAgent.Tools.([]byte); ok {
			if err := json.Unmarshal(toolsBytes, &agent.Tools); err != nil {
				return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to unmarshal agent tools").
					WithComponent("SQLiteAgentRepository").
					WithOperation("GetAgent")
			}
		}
	}

	return agent, nil
}

// UpdateAgent updates an existing agent
func (r *SQLiteAgentRepository) UpdateAgent(ctx context.Context, agent *Agent) error {
	// Convert capabilities and tools to JSON
	var capabilitiesJSON, toolsJSON []byte
	var err error

	if agent.Capabilities != nil {
		capabilitiesJSON, err = json.Marshal(agent.Capabilities)
		if err != nil {
			return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to marshal agent capabilities").
				WithComponent("SQLiteAgentRepository").
				WithOperation("CreateAgent")
		}
	}

	if agent.Tools != nil {
		toolsJSON, err = json.Marshal(agent.Tools)
		if err != nil {
			return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to marshal agent tools").
				WithComponent("SQLiteAgentRepository").
				WithOperation("CreateAgent")
		}
	}

	costMagnitude := int64(agent.CostMagnitude)
	if err := r.database.Queries().UpdateAgent(ctx, db.UpdateAgentParams{
		Name:          agent.Name,
		Type:          agent.Type,
		Provider:      agent.Provider,
		Model:         agent.Model,
		Capabilities:  capabilitiesJSON,
		Tools:         toolsJSON,
		CostMagnitude: &costMagnitude,
		ID:            agent.ID,
	}); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to update agent").
			WithComponent("SQLiteAgentRepository").
			WithOperation("UpdateAgent")
	}
	return nil
}

// DeleteAgent removes an agent by ID
func (r *SQLiteAgentRepository) DeleteAgent(ctx context.Context, id string) error {
	if err := r.database.Queries().DeleteAgent(ctx, id); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to delete agent").
			WithComponent("SQLiteAgentRepository").
			WithOperation("DeleteAgent").
			WithDetails("agent_id", id)
	}
	return nil
}

// ListAgents returns all agents
func (r *SQLiteAgentRepository) ListAgents(ctx context.Context) ([]*Agent, error) {
	dbAgents, err := r.database.Queries().ListAgents(ctx)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to list agents").
			WithComponent("SQLiteAgentRepository").
			WithOperation("ListAgents")
	}

	agents := make([]*Agent, len(dbAgents))
	for i, dbAgent := range dbAgents {
		// Handle cost magnitude conversion
		var costMagnitude int32
		if dbAgent.CostMagnitude != nil {
			costMagnitude = int32(*dbAgent.CostMagnitude)
		}

		agents[i] = &Agent{
			ID:            dbAgent.ID,
			Name:          dbAgent.Name,
			Type:          dbAgent.Type,
			Provider:      dbAgent.Provider,
			Model:         dbAgent.Model,
			CostMagnitude: costMagnitude,
			CreatedAt:     *dbAgent.CreatedAt,
		}

		// Parse capabilities JSON
		if dbAgent.Capabilities != nil {
			if capabilitiesBytes, ok := dbAgent.Capabilities.([]byte); ok {
				if err := json.Unmarshal(capabilitiesBytes, &agents[i].Capabilities); err != nil {
					return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to unmarshal agent capabilities").
						WithComponent("SQLiteAgentRepository").
						WithOperation("ListAgents").
						WithDetails("agent_index", i)
				}
			}
		}

		// Parse tools JSON
		if dbAgent.Tools != nil {
			if toolsBytes, ok := dbAgent.Tools.([]byte); ok {
				if err := json.Unmarshal(toolsBytes, &agents[i].Tools); err != nil {
					return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to unmarshal agent tools").
						WithComponent("SQLiteAgentRepository").
						WithOperation("ListAgents").
						WithDetails("agent_index", i)
				}
			}
		}
	}

	return agents, nil
}

// ListAgentsByType returns agents filtered by type
func (r *SQLiteAgentRepository) ListAgentsByType(ctx context.Context, agentType string) ([]*Agent, error) {
	dbAgents, err := r.database.Queries().ListAgentsByType(ctx, agentType)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to list agents by type").
			WithComponent("SQLiteAgentRepository").
			WithOperation("ListAgentsByType").
			WithDetails("agent_type", agentType)
	}

	agents := make([]*Agent, len(dbAgents))
	for i, dbAgent := range dbAgents {
		// Handle cost magnitude conversion
		var costMagnitude int32
		if dbAgent.CostMagnitude != nil {
			costMagnitude = int32(*dbAgent.CostMagnitude)
		}

		agents[i] = &Agent{
			ID:            dbAgent.ID,
			Name:          dbAgent.Name,
			Type:          dbAgent.Type,
			Provider:      dbAgent.Provider,
			Model:         dbAgent.Model,
			CostMagnitude: costMagnitude,
			CreatedAt:     *dbAgent.CreatedAt,
		}

		// Parse capabilities JSON
		if dbAgent.Capabilities != nil {
			if capabilitiesBytes, ok := dbAgent.Capabilities.([]byte); ok {
				if err := json.Unmarshal(capabilitiesBytes, &agents[i].Capabilities); err != nil {
					return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to unmarshal agent capabilities").
						WithComponent("SQLiteAgentRepository").
						WithOperation("ListAgents").
						WithDetails("agent_index", i)
				}
			}
		}

		// Parse tools JSON
		if dbAgent.Tools != nil {
			if toolsBytes, ok := dbAgent.Tools.([]byte); ok {
				if err := json.Unmarshal(toolsBytes, &agents[i].Tools); err != nil {
					return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to unmarshal agent tools").
						WithComponent("SQLiteAgentRepository").
						WithOperation("ListAgents").
						WithDetails("agent_index", i)
				}
			}
		}
	}

	return agents, nil
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
		BoardID:         dbTask.BoardID,      // Nullable BoardID from new schema
		CommissionID:    dbTask.CommissionID, // Keep for backward compatibility
		AssignedAgentID: dbTask.AssignedAgentID,
		Title:           dbTask.Title,
		Description:     dbTask.Description,
		Status:          dbTask.Status,
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
				return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to unmarshal task metadata").
					WithComponent("SQLiteTaskRepository").
					WithOperation("convertDBTaskToTask")
			}
		}
	}

	return task, nil
}

// newSQLiteSessionStore creates a session store using the existing session package
func newSQLiteSessionStore(database *Database) session.SessionStore {
	return session.NewSQLiteStore(database.DB())
}

// DefaultSessionRepositoryFactory creates a session repository for registry use
func DefaultSessionRepositoryFactory(database *Database) SessionRepository {
	// Create the session store from the existing session package
	sessionStore := newSQLiteSessionStore(database)

	// Create the repository adapter that bridges to our new interface
	return NewSQLiteSessionRepository(sessionStore)
}
