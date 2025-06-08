package mocks

import (
	"time"

	"github.com/guild-ventures/guild-core/pkg/commission"
)

// MockObjective creates a mock objective for testing
func MockObjective(id, title, description string) *commission.Commission {
	now := time.Now()
	return &commission.Commission{
		ID:          id,
		Title:       title,
		Description: description,
		Tasks:       []*commission.CommissionTask{},
		Status:      commission.CommissionStatusActive,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

// MockObjectiveWithTasks creates a mock objective with tasks
func MockObjectiveWithTasks(id, title, description string, tasks []*commission.CommissionTask) *commission.Commission {
	now := time.Now()
	return &commission.Commission{
		ID:          id,
		Title:       title,
		Description: description,
		Tasks:       tasks,
		Status:      commission.CommissionStatusActive,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

// MockTask creates a mock objective task
func MockTask(id, title, description string, sortOrder int) *commission.CommissionTask {
	now := time.Now()
	return &commission.CommissionTask{
		ID:          id,
		Title:       title,
		Description: description,
		SortOrder:   sortOrder,
		Status:      "todo",
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}
