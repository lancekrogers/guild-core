package manager

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/guild-ventures/guild-core/pkg/kanban"
	"github.com/guild-ventures/guild-core/pkg/objective"
)

// TaskBridge converts parsed commission structures into kanban tasks
type TaskBridge struct {
	kanbanBoard      *kanban.Board
	objectiveManager *objective.Manager
	parser           *ResponseParserImpl
}

// NewTaskBridge creates a new task bridge
func NewTaskBridge(
	kanbanBoard *kanban.Board,
	objectiveManager *objective.Manager,
) *TaskBridge {
	return &TaskBridge{
		kanbanBoard:      kanbanBoard,
		objectiveManager: objectiveManager,
		parser:           NewResponseParser(),
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
	
	// Create commission objective if it doesn't exist
	commissionObj := &objective.Objective{
		ID:          refinedCommission.CommissionID,
		Title:       fmt.Sprintf("Commission %s", refinedCommission.CommissionID),
		Description: "Refined commission tasks",
		Status:      objective.StatusDraft,
		Metadata:    make(map[string]string),
	}
	
	// Copy metadata with type assertions
	if title, ok := refinedCommission.Metadata["original_title"].(string); ok {
		commissionObj.Metadata["original_title"] = title
	}
	
	if err := tb.objectiveManager.SaveObjective(ctx, commissionObj); err != nil {
		// If already exists, that's okay
		if !strings.Contains(err.Error(), "already exists") {
			return fmt.Errorf("failed to create commission objective: %w", err)
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
			return fmt.Errorf("failed to create kanban task %s: %w", taskInfo.ID, err)
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
			return fmt.Errorf("failed to update task properties: %w", err)
		}
		createdTasks++
	}
	
	// Update objective with task count
	commissionObj.Metadata["total_tasks"] = fmt.Sprintf("%d", createdTasks)
	commissionObj.Metadata["status"] = "tasks_created"
	
	if err := tb.objectiveManager.SaveObjective(ctx, commissionObj); err != nil {
		return fmt.Errorf("failed to update commission objective: %w", err)
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
		return nil, fmt.Errorf("failed to parse refined content: %w", err)
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
			return taskIDs, fmt.Errorf("failed to create task %s: %w", taskInfo.ID, err)
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
			return taskIDs, fmt.Errorf("failed to update task properties: %w", err)
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
		return nil, fmt.Errorf("failed to get tasks: %w", err)
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
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
		
		// Write file
		if err := os.WriteFile(filePath, []byte(file.Content), 0644); err != nil {
			return fmt.Errorf("failed to write file %s: %w", filePath, err)
		}
	}
	
	return nil
}