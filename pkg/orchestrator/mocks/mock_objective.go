package mocks

import (
	"github.com/blockhead-consulting/guild/pkg/objective"
)

// MockObjective creates a mock objective for testing
func MockObjective(id, title, description string) *objective.Objective {
	return &objective.Objective{
		ID:          id,
		Title:       title,
		Description: description,
		Tasks:       []objective.Task{},
		Status:      "pending",
	}
}

// MockObjectiveWithTasks creates a mock objective with tasks
func MockObjectiveWithTasks(id, title, description string, tasks []objective.Task) *objective.Objective {
	return &objective.Objective{
		ID:          id,
		Title:       title,
		Description: description,
		Tasks:       tasks,
		Status:      "pending",
	}
}

// MockTask creates a mock objective task
func MockTask(id, title, description string, priority int) objective.Task {
	return objective.Task{
		ID:          id,
		Title:       title,
		Description: description,
		Priority:    priority,
		Status:      "pending",
	}
}