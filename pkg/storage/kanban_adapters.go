package storage

import (
	"context"
	"fmt"
	"time"
	
	"github.com/guild-ventures/guild-core/pkg/gerror"
)

// KanbanTaskRepositoryAdapter adapts storage.TaskRepository to handle interface{} calls from kanban
type KanbanTaskRepositoryAdapter struct {
	repo TaskRepository
}

// NewKanbanTaskRepositoryAdapter creates a new adapter for kanban task operations
func NewKanbanTaskRepositoryAdapter(repo TaskRepository) *KanbanTaskRepositoryAdapter {
	return &KanbanTaskRepositoryAdapter{repo: repo}
}

// CreateTask handles interface{} task creation from kanban package
func (a *KanbanTaskRepositoryAdapter) CreateTask(ctx context.Context, task interface{}) error {
	// Convert interface{} to proper Task struct
	storageTask, err := a.convertToTask(task)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to convert task").WithComponent("KanbanTaskRepositoryAdapter").WithOperation("CreateTask")
	}
	
	return a.repo.CreateTask(ctx, storageTask)
}

// DeleteTask handles task deletion
func (a *KanbanTaskRepositoryAdapter) DeleteTask(ctx context.Context, id string) error {
	return a.repo.DeleteTask(ctx, id)
}

// ListTasksByBoard handles board-specific task listing
func (a *KanbanTaskRepositoryAdapter) ListTasksByBoard(ctx context.Context, boardID string) ([]interface{}, error) {
	tasks, err := a.repo.ListTasksByBoard(ctx, boardID)
	if err != nil {
		return nil, err
	}
	
	// Convert to []interface{}
	result := make([]interface{}, len(tasks))
	for i, task := range tasks {
		result[i] = task
	}
	
	return result, nil
}

// UpdateTask handles task updating
func (a *KanbanTaskRepositoryAdapter) UpdateTask(ctx context.Context, task interface{}) error {
	// Convert interface{} to proper Task struct
	storageTask, err := a.convertToTask(task)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to convert task").WithComponent("KanbanTaskRepositoryAdapter").WithOperation("UpdateTask")
	}
	
	return a.repo.UpdateTask(ctx, storageTask)
}

// RecordTaskEvent handles task event recording
func (a *KanbanTaskRepositoryAdapter) RecordTaskEvent(ctx context.Context, event interface{}) error {
	// Convert interface{} to proper TaskEvent struct
	storageEvent, err := a.convertToTaskEvent(event)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to convert task event").WithComponent("KanbanTaskRepositoryAdapter").WithOperation("RecordTaskEvent")
	}
	
	return a.repo.RecordTaskEvent(ctx, storageEvent)
}

// convertToTask converts interface{} to storage.Task
func (a *KanbanTaskRepositoryAdapter) convertToTask(task interface{}) (*Task, error) {
	switch t := task.(type) {
	case *Task:
		// Apply status mapping for existing Task objects too
		t.Status = a.mapKanbanStatusToStorageStatus(t.Status)
		return t, nil
	case map[string]interface{}:
		return a.convertMapToTask(t)
	default:
		return nil, gerror.New(gerror.ErrCodeInvalidInput, "unsupported task type", nil).WithComponent("KanbanTaskRepositoryAdapter").WithOperation("convertToTask").WithDetails("type", fmt.Sprintf("%T", task))
	}
}

// convertMapToTask converts map[string]interface{} to storage.Task
func (a *KanbanTaskRepositoryAdapter) convertMapToTask(taskMap map[string]interface{}) (*Task, error) {
	task := &Task{}
	
	// Required fields
	if id, ok := taskMap["ID"].(string); ok {
		task.ID = id
	} else {
		return nil, gerror.New(gerror.ErrCodeMissingRequired, "task ID is required", nil).WithComponent("KanbanTaskRepositoryAdapter").WithOperation("convertMapToTask")
	}
	
	if title, ok := taskMap["Title"].(string); ok {
		task.Title = title
	} else {
		return nil, gerror.New(gerror.ErrCodeMissingRequired, "task Title is required", nil).WithComponent("KanbanTaskRepositoryAdapter").WithOperation("convertMapToTask")
	}
	
	if status, ok := taskMap["Status"].(string); ok {
		task.Status = a.mapKanbanStatusToStorageStatus(status)
	} else {
		task.Status = "todo" // Default status
	}
	
	// Required commission ID field
	if commissionID, ok := taskMap["CommissionID"].(string); ok && commissionID != "" {
		task.CommissionID = commissionID
	} else {
		return nil, gerror.New(gerror.ErrCodeMissingRequired, "task CommissionID is required", nil).WithComponent("KanbanTaskRepositoryAdapter").WithOperation("convertMapToTask").WithDetails("commissionID", taskMap["CommissionID"])
	}
	
	// Optional fields with proper type handling
	if boardID, ok := taskMap["BoardID"].(string); ok && boardID != "" {
		task.BoardID = &boardID
	}
	
	if description, ok := taskMap["Description"].(*string); ok {
		task.Description = description
	} else if desc, ok := taskMap["Description"].(string); ok {
		task.Description = &desc
	}
	
	if assignedAgent, ok := taskMap["AssignedAgentID"].(*string); ok {
		task.AssignedAgentID = assignedAgent
	} else if agent, ok := taskMap["AssignedAgentID"].(string); ok && agent != "" {
		task.AssignedAgentID = &agent
	}
	
	// Story points with type conversion
	if points, ok := taskMap["StoryPoints"].(int32); ok {
		task.StoryPoints = points
	} else if points, ok := taskMap["StoryPoints"].(int); ok {
		task.StoryPoints = int32(points)
	} else {
		task.StoryPoints = 1 // Default
	}
	
	// Metadata
	if metadata, ok := taskMap["Metadata"].(map[string]interface{}); ok {
		task.Metadata = metadata
	}
	
	// Timestamps - these should be handled by the database defaults
	// but include them if provided
	if createdAt, ok := taskMap["CreatedAt"].(time.Time); ok {
		task.CreatedAt = createdAt
	}
	
	if updatedAt, ok := taskMap["UpdatedAt"].(time.Time); ok {
		task.UpdatedAt = updatedAt
	}
	
	return task, nil
}

// mapKanbanStatusToStorageStatus maps kanban status values to database-compatible values
func (a *KanbanTaskRepositoryAdapter) mapKanbanStatusToStorageStatus(kanbanStatus string) string {
	switch kanbanStatus {
	case "backlog":
		return "todo" // Map backlog to todo
	case "ready_for_review":
		return "pending_review" // Map ready_for_review to pending_review
	case "cancelled":
		return "done" // Map cancelled to done (closest semantic match)
	case "todo", "in_progress", "blocked", "pending_review", "done":
		return kanbanStatus // These map directly
	default:
		return "todo" // Default fallback
	}
}

// convertToTaskEvent converts interface{} to storage.TaskEvent
func (a *KanbanTaskRepositoryAdapter) convertToTaskEvent(event interface{}) (*TaskEvent, error) {
	switch e := event.(type) {
	case *TaskEvent:
		return e, nil
	case TaskEvent:
		return &e, nil
	default:
		return nil, gerror.New(gerror.ErrCodeInvalidInput, "unsupported task event type", nil).WithComponent("KanbanTaskRepositoryAdapter").WithOperation("convertToTaskEvent").WithDetails("type", fmt.Sprintf("%T", event))
	}
}

// KanbanBoardRepositoryAdapter adapts storage.BoardRepository to handle interface{} calls from kanban
type KanbanBoardRepositoryAdapter struct {
	repo BoardRepository
}

// NewKanbanBoardRepositoryAdapter creates a new adapter for kanban board operations
func NewKanbanBoardRepositoryAdapter(repo BoardRepository) *KanbanBoardRepositoryAdapter {
	return &KanbanBoardRepositoryAdapter{repo: repo}
}

// CreateBoard handles interface{} board creation from kanban package
func (a *KanbanBoardRepositoryAdapter) CreateBoard(ctx context.Context, board interface{}) error {
	storageBoard, err := a.convertToBoard(board)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to convert board").WithComponent("KanbanBoardRepositoryAdapter").WithOperation("CreateBoard")
	}
	
	return a.repo.CreateBoard(ctx, storageBoard)
}

// GetBoard handles board retrieval
func (a *KanbanBoardRepositoryAdapter) GetBoard(ctx context.Context, id string) (interface{}, error) {
	return a.repo.GetBoard(ctx, id)
}

// UpdateBoard handles board updates
func (a *KanbanBoardRepositoryAdapter) UpdateBoard(ctx context.Context, board interface{}) error {
	storageBoard, err := a.convertToBoard(board)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to convert board").WithComponent("KanbanBoardRepositoryAdapter").WithOperation("UpdateBoard")
	}
	
	return a.repo.UpdateBoard(ctx, storageBoard)
}

// DeleteBoard handles board deletion
func (a *KanbanBoardRepositoryAdapter) DeleteBoard(ctx context.Context, id string) error {
	return a.repo.DeleteBoard(ctx, id)
}

// ListBoards handles board listing
func (a *KanbanBoardRepositoryAdapter) ListBoards(ctx context.Context) ([]interface{}, error) {
	boards, err := a.repo.ListBoards(ctx)
	if err != nil {
		return nil, err
	}
	
	// Convert to []interface{}
	result := make([]interface{}, len(boards))
	for i, board := range boards {
		result[i] = board
	}
	
	return result, nil
}

// convertToBoard converts interface{} to storage.Board
func (a *KanbanBoardRepositoryAdapter) convertToBoard(board interface{}) (*Board, error) {
	switch b := board.(type) {
	case *Board:
		return b, nil
	case map[string]interface{}:
		return a.convertMapToBoard(b)
	default:
		return nil, gerror.New(gerror.ErrCodeInvalidInput, "unsupported board type", nil).WithComponent("KanbanBoardRepositoryAdapter").WithOperation("convertToBoard").WithDetails("type", fmt.Sprintf("%T", board))
	}
}

// convertMapToBoard converts map[string]interface{} to storage.Board
func (a *KanbanBoardRepositoryAdapter) convertMapToBoard(boardMap map[string]interface{}) (*Board, error) {
	board := &Board{}
	
	// Required fields
	if id, ok := boardMap["ID"].(string); ok {
		board.ID = id
	} else {
		return nil, gerror.New(gerror.ErrCodeMissingRequired, "board ID is required", nil).WithComponent("KanbanBoardRepositoryAdapter").WithOperation("convertMapToBoard")
	}
	
	if name, ok := boardMap["Name"].(string); ok {
		board.Name = name
	} else {
		return nil, gerror.New(gerror.ErrCodeMissingRequired, "board Name is required", nil).WithComponent("KanbanBoardRepositoryAdapter").WithOperation("convertMapToBoard")
	}
	
	if commissionID, ok := boardMap["CommissionID"].(string); ok {
		board.CommissionID = commissionID
	} else {
		return nil, gerror.New(gerror.ErrCodeMissingRequired, "board CommissionID is required", nil).WithComponent("KanbanBoardRepositoryAdapter").WithOperation("convertMapToBoard")
	}
	
	// Optional fields
	if description, ok := boardMap["Description"].(*string); ok {
		board.Description = description
	} else if desc, ok := boardMap["Description"].(string); ok {
		board.Description = &desc
	}
	
	if status, ok := boardMap["Status"].(string); ok {
		board.Status = status
	} else {
		board.Status = "active" // Default status
	}
	
	// Timestamps
	if createdAt, ok := boardMap["CreatedAt"].(time.Time); ok {
		board.CreatedAt = createdAt
	}
	
	if updatedAt, ok := boardMap["UpdatedAt"].(time.Time); ok {
		board.UpdatedAt = updatedAt
	}
	
	return board, nil
}

// KanbanCampaignRepositoryAdapter adapts storage.CampaignRepository to handle interface{} calls from kanban
type KanbanCampaignRepositoryAdapter struct {
	repo CampaignRepository
}

// NewKanbanCampaignRepositoryAdapter creates a new adapter for kanban campaign operations
func NewKanbanCampaignRepositoryAdapter(repo CampaignRepository) *KanbanCampaignRepositoryAdapter {
	return &KanbanCampaignRepositoryAdapter{repo: repo}
}

// CreateCampaign handles interface{} campaign creation from kanban package
func (a *KanbanCampaignRepositoryAdapter) CreateCampaign(ctx context.Context, campaign interface{}) error {
	storageCampaign, err := a.convertToCampaign(campaign)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to convert campaign").WithComponent("KanbanCampaignRepositoryAdapter").WithOperation("CreateCampaign")
	}
	
	return a.repo.CreateCampaign(ctx, storageCampaign)
}

// convertToCampaign converts interface{} to storage.Campaign
func (a *KanbanCampaignRepositoryAdapter) convertToCampaign(campaign interface{}) (*Campaign, error) {
	switch c := campaign.(type) {
	case *Campaign:
		return c, nil
	case map[string]interface{}:
		return a.convertMapToCampaign(c)
	default:
		return nil, gerror.New(gerror.ErrCodeInvalidInput, "unsupported campaign type", nil).WithComponent("KanbanCampaignRepositoryAdapter").WithOperation("convertToCampaign").WithDetails("type", fmt.Sprintf("%T", campaign))
	}
}

// convertMapToCampaign converts map[string]interface{} to storage.Campaign
func (a *KanbanCampaignRepositoryAdapter) convertMapToCampaign(campaignMap map[string]interface{}) (*Campaign, error) {
	campaign := &Campaign{}
	
	// Required fields
	if id, ok := campaignMap["ID"].(string); ok {
		campaign.ID = id
	} else {
		return nil, gerror.New(gerror.ErrCodeMissingRequired, "campaign ID is required", nil).WithComponent("KanbanCampaignRepositoryAdapter").WithOperation("convertMapToCampaign")
	}
	
	if name, ok := campaignMap["Name"].(string); ok {
		campaign.Name = name
	} else {
		return nil, gerror.New(gerror.ErrCodeMissingRequired, "campaign Name is required", nil).WithComponent("KanbanCampaignRepositoryAdapter").WithOperation("convertMapToCampaign")
	}
	
	// Optional fields
	if status, ok := campaignMap["Status"].(string); ok {
		campaign.Status = status
	} else {
		campaign.Status = "active" // Default status
	}
	
	// Timestamps
	if createdAt, ok := campaignMap["CreatedAt"].(time.Time); ok {
		campaign.CreatedAt = createdAt
	}
	
	if updatedAt, ok := campaignMap["UpdatedAt"].(time.Time); ok {
		campaign.UpdatedAt = updatedAt
	}
	
	return campaign, nil
}

// KanbanCommissionRepositoryAdapter adapts storage.CommissionRepository to handle interface{} calls from kanban
type KanbanCommissionRepositoryAdapter struct {
	repo CommissionRepository
}

// NewKanbanCommissionRepositoryAdapter creates a new adapter for kanban commission operations
func NewKanbanCommissionRepositoryAdapter(repo CommissionRepository) *KanbanCommissionRepositoryAdapter {
	return &KanbanCommissionRepositoryAdapter{repo: repo}
}

// CreateCommission handles interface{} commission creation from kanban package
func (a *KanbanCommissionRepositoryAdapter) CreateCommission(ctx context.Context, commission interface{}) error {
	storageCommission, err := a.convertToCommission(commission)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to convert commission").WithComponent("KanbanCommissionRepositoryAdapter").WithOperation("CreateCommission")
	}
	
	return a.repo.CreateCommission(ctx, storageCommission)
}

// GetCommission handles commission retrieval
func (a *KanbanCommissionRepositoryAdapter) GetCommission(ctx context.Context, id string) (interface{}, error) {
	return a.repo.GetCommission(ctx, id)
}

// convertToCommission converts interface{} to storage.Commission
func (a *KanbanCommissionRepositoryAdapter) convertToCommission(commission interface{}) (*Commission, error) {
	switch c := commission.(type) {
	case *Commission:
		return c, nil
	case map[string]interface{}:
		return a.convertMapToCommission(c)
	default:
		return nil, gerror.New(gerror.ErrCodeInvalidInput, "unsupported commission type", nil).WithComponent("KanbanCommissionRepositoryAdapter").WithOperation("convertToCommission").WithDetails("type", fmt.Sprintf("%T", commission))
	}
}

// convertMapToCommission converts map[string]interface{} to storage.Commission
func (a *KanbanCommissionRepositoryAdapter) convertMapToCommission(commissionMap map[string]interface{}) (*Commission, error) {
	commission := &Commission{}
	
	// Required fields
	if id, ok := commissionMap["ID"].(string); ok {
		commission.ID = id
	} else {
		return nil, gerror.New(gerror.ErrCodeMissingRequired, "commission ID is required", nil).WithComponent("KanbanCommissionRepositoryAdapter").WithOperation("convertMapToCommission")
	}
	
	if campaignID, ok := commissionMap["CampaignID"].(string); ok {
		commission.CampaignID = campaignID
	} else {
		return nil, gerror.New(gerror.ErrCodeMissingRequired, "commission CampaignID is required", nil).WithComponent("KanbanCommissionRepositoryAdapter").WithOperation("convertMapToCommission")
	}
	
	if title, ok := commissionMap["Title"].(string); ok {
		commission.Title = title
	} else {
		return nil, gerror.New(gerror.ErrCodeMissingRequired, "commission Title is required", nil).WithComponent("KanbanCommissionRepositoryAdapter").WithOperation("convertMapToCommission")
	}
	
	// Optional fields
	if description, ok := commissionMap["Description"].(*string); ok {
		commission.Description = description
	} else if desc, ok := commissionMap["Description"].(string); ok {
		commission.Description = &desc
	}
	
	if domain, ok := commissionMap["Domain"].(*string); ok {
		commission.Domain = domain
	} else if dom, ok := commissionMap["Domain"].(string); ok {
		commission.Domain = &dom
	}
	
	if context, ok := commissionMap["Context"].(map[string]interface{}); ok {
		commission.Context = context
	}
	
	if status, ok := commissionMap["Status"].(string); ok {
		commission.Status = status
	} else {
		commission.Status = "active" // Default status
	}
	
	// Timestamps
	if createdAt, ok := commissionMap["CreatedAt"].(time.Time); ok {
		commission.CreatedAt = createdAt
	}
	
	return commission, nil
}