package mocks

import (
	"time"
	"github.com/guild-ventures/guild-core/pkg/commission"
)

// MockObjective creates a mock objective for testing
func MockObjective(id, title, description string) *objective.Objective {
	now := time.Now()
	return &objective.Objective{
		ID:          id,
		Title:       title,
		Description: description,
		Tasks:       []*objective.ObjectiveTask{},
		Status:      objective.ObjectiveStatusActive,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

// MockObjectiveWithTasks creates a mock objective with tasks
func MockObjectiveWithTasks(id, title, description string, tasks []*objective.ObjectiveTask) *objective.Objective {
	now := time.Now()
	return &objective.Objective{
		ID:          id,
		Title:       title,
		Description: description,
		Tasks:       tasks,
		Status:      objective.ObjectiveStatusActive,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

// MockTask creates a mock objective task
func MockTask(id, title, description string, sortOrder int) *objective.ObjectiveTask {
	now := time.Now()
	return &objective.ObjectiveTask{
		ID:          id,
		Title:       title,
		Description: description,
		SortOrder:   sortOrder,
		Status:      "todo",
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}