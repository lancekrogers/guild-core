// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package commission_kanban

import (
	"testing"
	"time"

	"github.com/lancekrogers/guild/pkg/commission"
	"github.com/lancekrogers/guild/pkg/kanban"
)

func TestConvertRefinedCommissionToKanbanTasks(t *testing.T) {
	integrator := &Integrator{}

	// Create a refined commission with sample tasks
	refinedCommission := &commission.RefinedCommission{
		Original: &commission.Commission{
			ID:    "test-commission",
			Title: "Test Commission",
		},
		Tasks: []*commission.RefinedTask{
			{
				ID:             "task-1",
				CommissionID:   "test-commission",
				Title:          "High Complexity Task",
				Description:    "A complex task",
				Type:           "implementation",
				Status:         "todo",
				Complexity:     8,
				EstimatedHours: 16.0,
				AssignedAgent:  "agent-1",
				Dependencies:   []string{"task-2"},
				CreatedAt:      time.Now(),
				UpdatedAt:      time.Now(),
				Metadata:       map[string]string{"phase": "development"},
			},
			{
				ID:             "task-2",
				CommissionID:   "test-commission",
				Title:          "Medium Complexity Task",
				Description:    "A medium task",
				Type:           "design",
				Status:         "todo",
				Complexity:     4,
				EstimatedHours: 8.0,
				AssignedAgent:  "agent-2",
				Dependencies:   []string{},
				CreatedAt:      time.Now(),
				UpdatedAt:      time.Now(),
				Metadata:       map[string]string{"phase": "planning"},
			},
		},
	}

	// Convert to kanban tasks
	kanbanTasks := integrator.ConvertRefinedCommissionToKanbanTasks(refinedCommission)

	// Verify the conversion
	if len(kanbanTasks) != 2 {
		t.Errorf("Expected 2 kanban tasks, got %d", len(kanbanTasks))
	}

	// Check first task (high complexity)
	task1 := kanbanTasks[0]
	if task1.ID != "task-1" {
		t.Errorf("Expected task ID 'task-1', got '%s'", task1.ID)
	}
	if task1.Title != "High Complexity Task" {
		t.Errorf("Expected title 'High Complexity Task', got '%s'", task1.Title)
	}
	if task1.Priority != kanban.PriorityHigh {
		t.Errorf("Expected high priority for complexity 8, got %v", task1.Priority)
	}
	if task1.EstimatedHours != 16.0 {
		t.Errorf("Expected 16.0 hours, got %f", task1.EstimatedHours)
	}
	if len(task1.Dependencies) != 1 || task1.Dependencies[0] != "task-2" {
		t.Errorf("Expected dependency on task-2, got %v", task1.Dependencies)
	}

	// Check second task (medium complexity)
	task2 := kanbanTasks[1]
	if task2.ID != "task-2" {
		t.Errorf("Expected task ID 'task-2', got '%s'", task2.ID)
	}
	if task2.Priority != kanban.PriorityMedium {
		t.Errorf("Expected medium priority for complexity 4, got %v", task2.Priority)
	}
	if len(task2.Dependencies) != 0 {
		t.Errorf("Expected no dependencies, got %v", task2.Dependencies)
	}

	// Check metadata was transferred
	if task1.Metadata["commission_id"] != "test-commission" {
		t.Errorf("Expected commission_id in metadata, got %v", task1.Metadata)
	}
	if task1.Metadata["task_type"] != "implementation" {
		t.Errorf("Expected task_type 'implementation', got %s", task1.Metadata["task_type"])
	}
	if task1.Metadata["complexity"] != "8" {
		t.Errorf("Expected complexity '8', got %s", task1.Metadata["complexity"])
	}
	if task1.Metadata["phase"] != "development" {
		t.Errorf("Expected phase 'development', got %s", task1.Metadata["phase"])
	}
}

func TestDetermineInitialColumn(t *testing.T) {
	integrator := &Integrator{}

	tests := []struct {
		name           string
		task           *commission.RefinedTask
		expectedColumn string
	}{
		{
			name: "task with dependencies goes to backlog",
			task: &commission.RefinedTask{
				Status:       "todo",
				Dependencies: []string{"other-task"},
			},
			expectedColumn: "backlog",
		},
		{
			name: "setup task goes to todo",
			task: &commission.RefinedTask{
				Status: "todo",
				Type:   "setup",
			},
			expectedColumn: "todo",
		},
		{
			name: "in_progress task stays in_progress",
			task: &commission.RefinedTask{
				Status: "in_progress",
			},
			expectedColumn: "in_progress",
		},
		{
			name: "unknown status defaults to todo",
			task: &commission.RefinedTask{
				Status: "unknown_status",
			},
			expectedColumn: "todo",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			column := integrator.determineInitialColumn(tt.task)
			if column != tt.expectedColumn {
				t.Errorf("Expected column '%s', got '%s'", tt.expectedColumn, column)
			}
		})
	}
}
