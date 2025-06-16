// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package manager

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/guild-ventures/guild-core/pkg/agent"
	"github.com/guild-ventures/guild-core/pkg/gerror"
	"github.com/guild-ventures/guild-core/pkg/kanban"
)

// TaskBridge converts parsed commission structures into kanban tasks
type TaskBridge struct {
	kanbanBoard          *kanban.Board
	commissionRepository agent.CommissionRepository
	parser               *ResponseParserImpl
}

// NewTaskBridge creates a new task bridge (deprecated - use NewTaskBridgeWithCommissions)
func NewTaskBridge(
	kanbanBoard *kanban.Board,
	commissionManager interface{}, // Deprecated parameter - ignored
) *TaskBridge {
	return &TaskBridge{
		kanbanBoard:          kanbanBoard,
		commissionRepository: nil, // Will need to be set separately
		parser:               NewResponseParser(),
	}
}

// NewTaskBridgeWithCommissions creates a new task bridge with commission repository
func NewTaskBridgeWithCommissions(
	kanbanBoard *kanban.Board,
	commissionRepository agent.CommissionRepository,
) *TaskBridge {
	return &TaskBridge{
		kanbanBoard:          kanbanBoard,
		commissionRepository: commissionRepository,
		parser:               NewResponseParser(),
	}
}

// CreateTasksFromRefinedCommission creates kanban tasks from a refined commission
func (tb *TaskBridge) CreateTasksFromRefinedCommission(ctx context.Context, refinedCommission *RefinedCommission) error {
	// Extract tasks from all files in the structure
	var allTasks []TaskInfo

	for _, file := range refinedCommission.Structure.Files {
		// Get tasks from metadata if available
		if tasks, ok := file.Metadata["tasks"].([]TaskInfo); ok {
			allTasks = append(allTasks, tasks...)
		} else {
			// Otherwise extract tasks from content
			tasks := tb.parser.extractTasks(file.Content)
			allTasks = append(allTasks, tasks...)
		}
	}

	// Create commission record if it doesn't exist
	commission := map[string]interface{}{
		"ID":          refinedCommission.CommissionID,
		"Title":       fmt.Sprintf("Commission %s", refinedCommission.CommissionID),
		"Description": "Refined commission tasks",
		"Status":      "draft",
		"CampaignID":  "default-campaign", // TODO: Get from context
	}

	// Copy metadata with type assertions
	if title, ok := refinedCommission.Metadata["original_title"].(string); ok {
		commission["Title"] = title
	}

	if tb.commissionRepository != nil {
		// Convert map to agent.Commission struct
		description := commission["Description"].(string)
		registryCommission := &agent.Commission{
			ID:          refinedCommission.CommissionID,
			CampaignID:  "default-campaign", // TODO: Get from context
			Title:       commission["Title"].(string),
			Description: &description,
			Status:      commission["Status"].(string),
		}

		if err := tb.commissionRepository.CreateCommission(ctx, registryCommission); err != nil {
			// If already exists, that's okay - ignore UNIQUE constraint errors
			if !strings.Contains(err.Error(), "UNIQUE constraint failed") &&
				!strings.Contains(err.Error(), "already exists") {
				return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to create commission record").
					WithComponent("manager").
					WithOperation("CreateTasksFromRefinedCommission").
					WithDetails("commission_id", refinedCommission.CommissionID)
			}
		}
	}

	// Convert and create kanban tasks
	createdTasks := 0
	for _, taskInfo := range allTasks {
		kanbanTask := taskInfo.ConvertToKanbanTask(refinedCommission.CommissionID)

		// Add commission metadata with type assertions
		if title, ok := refinedCommission.Metadata["original_title"].(string); ok {
			kanbanTask.Metadata["commission_title"] = title
		}
		if timestamp, ok := refinedCommission.Metadata["refinement_timestamp"].(string); ok {
			kanbanTask.Metadata["refinement_timestamp"] = timestamp
		}

		// Create the task in kanban (CreateTask expects title and description)
		createdTask, err := tb.kanbanBoard.CreateTask(ctx, kanbanTask.Title, kanbanTask.Description)
		if err != nil {
			return gerror.Wrapf(err, gerror.ErrCodeStorage, "failed to create kanban task %s", taskInfo.ID).
				WithComponent("manager").
				WithOperation("CreateTasksFromRefinedCommission").
				WithDetails("task_id", taskInfo.ID).
				WithDetails("commission_id", refinedCommission.CommissionID)
		}

		// Update task with additional properties
		createdTask.Priority = kanbanTask.Priority
		createdTask.Status = kanbanTask.Status
		createdTask.Dependencies = kanbanTask.Dependencies
		createdTask.Tags = kanbanTask.Tags
		for k, v := range kanbanTask.Metadata {
			createdTask.Metadata[k] = v
		}

		// Save the updated task
		if err := tb.kanbanBoard.UpdateTask(ctx, createdTask); err != nil {
			return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to update task properties").
				WithComponent("manager").
				WithOperation("CreateTasksFromRefinedCommission").
				WithDetails("task_id", createdTask.ID).
				WithDetails("commission_id", refinedCommission.CommissionID)
		}
		createdTasks++
	}

	// Update commission with task count (if repository is available)
	if tb.commissionRepository != nil {
		// TODO: Add metadata update capability to commission repository
		// For now, just log the task count
		fmt.Printf("Created %d tasks for commission %s\n", createdTasks, refinedCommission.CommissionID)
	}

	return nil
}

// CreateTasksFromRefinedContent creates tasks directly from refined content string
func (tb *TaskBridge) CreateTasksFromRefinedContent(ctx context.Context, commissionID string, refinedContent string) ([]string, error) {
	// Parse the refined content
	response := &ArtisanResponse{
		Content: refinedContent,
		Metadata: map[string]interface{}{
			"commission_id": commissionID,
		},
	}

	structure, err := tb.parser.ParseResponse(response)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to parse refined content").
			WithComponent("manager").
			WithOperation("CreateTasksFromRefinedContent").
			WithDetails("commission_id", commissionID)
	}

	// Note: refined variable removed as it was declared but not used

	// Extract all tasks
	var allTasks []TaskInfo
	var taskIDs []string

	for _, file := range structure.Files {
		if tasks, ok := file.Metadata["tasks"].([]TaskInfo); ok {
			allTasks = append(allTasks, tasks...)
		}
	}

	// Create kanban tasks
	for _, taskInfo := range allTasks {
		kanbanTask := taskInfo.ConvertToKanbanTask(commissionID)

		// Create the task
		createdTask, err := tb.kanbanBoard.CreateTask(ctx, kanbanTask.Title, kanbanTask.Description)
		if err != nil {
			return taskIDs, gerror.Wrapf(err, gerror.ErrCodeStorage, "failed to create task %s", taskInfo.ID).
				WithComponent("manager").
				WithOperation("CreateTasksFromRefinedContent").
				WithDetails("task_id", taskInfo.ID).
				WithDetails("commission_id", commissionID)
		}

		// Update task properties
		createdTask.Priority = kanbanTask.Priority
		createdTask.Status = kanbanTask.Status
		createdTask.Dependencies = kanbanTask.Dependencies
		createdTask.Tags = kanbanTask.Tags
		for k, v := range kanbanTask.Metadata {
			createdTask.Metadata[k] = v
		}

		// Save the updated task
		if err := tb.kanbanBoard.UpdateTask(ctx, createdTask); err != nil {
			return taskIDs, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to update task properties").
				WithComponent("manager").
				WithOperation("CreateTasksFromRefinedContent").
				WithDetails("task_id", createdTask.ID).
				WithDetails("commission_id", commissionID)
		}

		taskIDs = append(taskIDs, createdTask.ID)
	}

	return taskIDs, nil
}

// GetTasksForCommission retrieves all tasks for a commission
func (tb *TaskBridge) GetTasksForCommission(ctx context.Context, commissionID string) ([]*kanban.Task, error) {
	// Get all tasks from kanban
	allTasks, err := tb.kanbanBoard.GetAllTasks(ctx)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to get tasks").
			WithComponent("manager").
			WithOperation("GetTasksForCommission").
			WithDetails("commission_id", commissionID)
	}

	// Filter by commission ID
	var commissionTasks []*kanban.Task
	for _, task := range allTasks {
		if task.Metadata["commission_id"] == commissionID {
			commissionTasks = append(commissionTasks, task)
		}
	}

	return commissionTasks, nil
}

// WriteRefinedFiles writes the refined commission files to the filesystem
func (tb *TaskBridge) WriteRefinedFiles(refined *RefinedCommission, outputDir string) error {
	for _, file := range refined.Structure.Files {
		filePath := filepath.Join(outputDir, file.Path)

		// Create directory if needed
		dir := filepath.Dir(filePath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return gerror.Wrapf(err, gerror.ErrCodeInternal, "failed to create directory %s", dir).
				WithComponent("manager").
				WithOperation("WriteRefinedFiles").
				WithDetails("directory", dir)
		}

		// Write file
		if err := os.WriteFile(filePath, []byte(file.Content), 0644); err != nil {
			return gerror.Wrapf(err, gerror.ErrCodeInternal, "failed to write file %s", filePath).
				WithComponent("manager").
				WithOperation("WriteRefinedFiles").
				WithDetails("file_path", filePath)
		}
	}

	return nil
}
