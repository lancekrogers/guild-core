// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package manager

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/guild-ventures/guild-core/pkg/kanban"
)

func TestResponseParser_ParseResponse(t *testing.T) {
	parser := NewResponseParser()

	tests := []struct {
		name          string
		response      *ArtisanResponse
		expectedFiles int
		expectedTasks int
		expectedError bool
	}{
		{
			name: "parse_multiple_file_structure",
			response: &ArtisanResponse{
				Content: `## File: commission_overview.md

# Commission Overview

This commission involves building a web application.

## Tasks

- BACKEND-001: Set up API server (priority: high, estimate: 4h)
- BACKEND-002: Implement authentication (priority: high, estimate: 6h, depends: BACKEND-001)
- FRONTEND-001: Create React app (priority: medium, estimate: 2h)

## File: tasks/backend_tasks.md

# Backend Implementation Tasks

- [ ] Configure database connections
- [ ] Set up middleware pipeline
- [ ] Implement error handling

## File: tasks/frontend_tasks.md

# Frontend Tasks

Task: Build user interface components
Task: Integrate with backend API`,
			},
			expectedFiles: 3,
			expectedTasks: 3, // Only counting tasks from first file
			expectedError: false,
		},
		{
			name: "parse_single_hierarchical_document",
			response: &ArtisanResponse{
				Content: `# Commission: E-commerce Platform

## Overview
Building a modern e-commerce platform with microservices.

## Implementation Tasks

### Backend Services

- AUTH-001: Create authentication service (priority: high)
- AUTH-002: Implement JWT tokens (priority: high, depends: AUTH-001)
- CATALOG-001: Build product catalog service (priority: medium)

### Frontend Requirements

- [ ] Design responsive UI
- [ ] Implement shopping cart
- [ ] Add payment integration

### Infrastructure Tasks

Task: Set up Kubernetes cluster
Task: Configure CI/CD pipeline`,
			},
			expectedFiles: 3, // main + auth_tasks.md + catalog_tasks.md (general tasks are skipped)
			expectedTasks: 8,
			expectedError: false,
		},
		{
			name: "parse_workshop_board_format",
			response: &ArtisanResponse{
				Content: `# Guild Commission: API Development

## Workshop Board Tasks

**Artisan Assignments:**

- API-001: Design REST endpoints (priority: high, estimate: 3h)
  - Assigned to: backend-artisan
  - Capabilities required: api-design, openapi

- API-002: Implement data models (priority: medium, estimate: 4h)
  - Dependencies: API-001
  - Capabilities: database, orm

- TEST-001: Write integration tests (priority: medium, estimate: 5h)
  - Dependencies: API-001, API-002`,
			},
			expectedFiles: 3, // main + api_tasks.md + test_tasks.md
			expectedTasks: 3,
			expectedError: false,
		},
		{
			name: "empty_response",
			response: &ArtisanResponse{
				Content: "",
			},
			expectedFiles: 0,
			expectedTasks: 0,
			expectedError: true,
		},
		{
			name:          "nil_response",
			response:      nil,
			expectedFiles: 0,
			expectedTasks: 0,
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			structure, err := parser.ParseResponse(context.Background(), tt.response)

			if tt.expectedError {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, structure)

			// Check file count
			assert.Equal(t, tt.expectedFiles, len(structure.Files),
				"Expected %d files, got %d", tt.expectedFiles, len(structure.Files))

			// Count total unique tasks (only from main file to avoid duplicates)
			totalTasks := 0
			foundMainFile := false
			for _, file := range structure.Files {
				// Only count tasks from the main file
				if tasks, ok := file.Metadata["tasks"].([]TaskInfo); ok {
					if file.Path == "commission_refined.md" || (!foundMainFile && len(tasks) > 0) {
						totalTasks = len(tasks)
						foundMainFile = true
						break
					}
				}
			}
			assert.Equal(t, tt.expectedTasks, totalTasks,
				"Expected %d total tasks, got %d", tt.expectedTasks, totalTasks)
		})
	}
}

func TestResponseParser_ExtractTasks(t *testing.T) {
	parser := NewResponseParser()

	tests := []struct {
		name          string
		content       string
		expectedTasks []TaskInfo
	}{
		{
			name: "category_number_format",
			content: `## Implementation Tasks

- BACKEND-001: Set up server (priority: high, estimate: 2h)
- BACKEND-002: Create database schema (priority: medium, depends: BACKEND-001)
- FRONTEND-001: Initialize React app`,
			expectedTasks: []TaskInfo{
				{ID: "BACKEND-001", Category: "BACKEND", Number: "001", Title: "Set up server", Priority: "high", Estimate: "2h"},
				{ID: "BACKEND-002", Category: "BACKEND", Number: "002", Title: "Create database schema", Priority: "medium", Dependencies: []string{"BACKEND-001"}},
				{ID: "FRONTEND-001", Category: "FRONTEND", Number: "001", Title: "Initialize React app"},
			},
		},
		{
			name: "checkbox_format",
			content: `## Tasks to Complete

- [ ] Configure environment variables
- [x] Set up Git repository
- [ ] Install dependencies`,
			expectedTasks: []TaskInfo{
				{Title: "Configure environment variables", Description: "Configure environment variables"},
				{Title: "Set up Git repository", Description: "Set up Git repository"},
				{Title: "Install dependencies", Description: "Install dependencies"},
			},
		},
		{
			name: "task_keyword_format",
			content: `## Development Work

Task: Implement user authentication system
Task: Create API documentation
Task: Set up monitoring and logging`,
			expectedTasks: []TaskInfo{
				{Title: "Implement user authentication system", Description: "Implement user authentication system"},
				{Title: "Create API documentation", Description: "Create API documentation"},
				{Title: "Set up monitoring and logging", Description: "Set up monitoring and logging"},
			},
		},
		{
			name: "mixed_formats",
			content: `# Commission Tasks

## Backend
- API-001: Create REST endpoints (priority: high)
- [ ] Set up middleware

## Frontend
Task: Build component library

## Testing
- TEST-001: Write unit tests`,
			expectedTasks: []TaskInfo{
				{ID: "API-001", Category: "API", Title: "Create REST endpoints", Priority: "high"},
				{Title: "Set up middleware", Description: "Set up middleware"},
				{Title: "Build component library", Description: "Build component library"},
				{ID: "TEST-001", Category: "TEST", Title: "Write unit tests"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tasks := parser.extractTasks(tt.content)

			// Compare task counts
			assert.Equal(t, len(tt.expectedTasks), len(tasks),
				"Expected %d tasks, got %d", len(tt.expectedTasks), len(tasks))

			// Check specific task properties
			for i, expectedTask := range tt.expectedTasks {
				if i >= len(tasks) {
					break
				}

				actualTask := tasks[i]

				// For tasks without explicit IDs, we generate them
				if expectedTask.ID != "" {
					assert.Equal(t, expectedTask.ID, actualTask.ID)
				}

				assert.Equal(t, expectedTask.Category, actualTask.Category)
				assert.Equal(t, expectedTask.Title, actualTask.Title)

				if expectedTask.Priority != "" {
					assert.Equal(t, expectedTask.Priority, actualTask.Priority)
				}

				if expectedTask.Estimate != "" {
					assert.Equal(t, expectedTask.Estimate, actualTask.Estimate)
				}

				if len(expectedTask.Dependencies) > 0 {
					assert.Equal(t, expectedTask.Dependencies, actualTask.Dependencies)
				}
			}
		})
	}
}

func TestResponseParser_ConvertToKanbanTask(t *testing.T) {
	tests := []struct {
		name         string
		taskInfo     TaskInfo
		commissionID string
		validate     func(t *testing.T, task *kanban.Task)
	}{
		{
			name: "high_priority_task",
			taskInfo: TaskInfo{
				ID:          "BACKEND-001",
				Category:    "BACKEND",
				Title:       "Set up API server",
				Description: "Initialize the backend API server with Express.js",
				Priority:    "high",
				Estimate:    "4h",
			},
			commissionID: "commission-123",
			validate: func(t *testing.T, task *kanban.Task) {
				assert.Equal(t, kanban.PriorityHigh, task.Priority)
				assert.Equal(t, kanban.StatusTodo, task.Status)
				assert.Equal(t, "commission-123", task.Metadata["commission_id"])
				assert.Equal(t, "BACKEND", task.Metadata["category"])
				assert.Contains(t, task.Tags, "backend")
			},
		},
		{
			name: "task_with_dependencies",
			taskInfo: TaskInfo{
				ID:           "FRONTEND-002",
				Category:     "FRONTEND",
				Title:        "Create user dashboard",
				Dependencies: []string{"BACKEND-001", "AUTH-001"},
				Priority:     "medium",
			},
			commissionID: "commission-456",
			validate: func(t *testing.T, task *kanban.Task) {
				assert.Equal(t, kanban.PriorityMedium, task.Priority)
				assert.Equal(t, []string{"BACKEND-001", "AUTH-001"}, task.Dependencies)
				assert.Contains(t, task.Tags, "frontend")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task := tt.taskInfo.ConvertToKanbanTask(tt.commissionID)

			assert.NotNil(t, task)
			assert.NotEmpty(t, task.ID)
			assert.Equal(t, tt.taskInfo.Title, task.Title)

			if tt.taskInfo.Description != "" {
				assert.Equal(t, tt.taskInfo.Description, task.Description)
			}

			tt.validate(t, task)
		})
	}
}

func TestResponseParser_GroupTasksByCategory(t *testing.T) {
	parser := NewResponseParser()

	tasks := []TaskInfo{
		{ID: "API-001", Category: "API", Title: "Task 1"},
		{ID: "API-002", Category: "API", Title: "Task 2"},
		{ID: "DB-001", Category: "DB", Title: "Task 3"},
		{ID: "TASK-001", Category: "", Title: "Task 4"},
		{ID: "UI-001", Category: "UI", Title: "Task 5"},
	}

	grouped := parser.groupTasksByCategory(tasks)

	assert.Equal(t, 4, len(grouped))
	assert.Equal(t, 2, len(grouped["API"]))
	assert.Equal(t, 1, len(grouped["DB"]))
	assert.Equal(t, 1, len(grouped["UI"]))
	assert.Equal(t, 1, len(grouped["general"]))
}

func TestResponseParser_FormatTasksAsMarkdown(t *testing.T) {
	parser := NewResponseParser()

	tasks := []TaskInfo{
		{
			ID:           "BACKEND-001",
			Title:        "Set up server",
			Description:  "Initialize Express.js server with middleware",
			Priority:     "high",
			Estimate:     "3h",
			Dependencies: []string{"ENV-001"},
		},
		{
			ID:       "FRONTEND-001",
			Title:    "Create React app",
			Priority: "medium",
		},
	}

	markdown := parser.formatTasksAsMarkdown(tasks)

	// Check that markdown contains expected content
	assert.Contains(t, markdown, "# Tasks")
	assert.Contains(t, markdown, "## BACKEND-001")
	assert.Contains(t, markdown, "**Title:** Set up server")
	assert.Contains(t, markdown, "**Description:** Initialize Express.js server")
	assert.Contains(t, markdown, "**Priority:** high")
	assert.Contains(t, markdown, "**Estimate:** 3h")
	assert.Contains(t, markdown, "**Dependencies:** ENV-001")
	assert.Contains(t, markdown, "## FRONTEND-001")
	assert.Contains(t, markdown, "---")
}

func TestResponseParser_LooksLikeTask(t *testing.T) {
	parser := NewResponseParser()

	tests := []struct {
		line     string
		expected bool
	}{
		{"Implement user authentication", true},
		{"Create database schema", true},
		{"Add unit tests for the API", true},
		{"Deploy to production server", true},
		{"Configure environment variables", true},
		{"# Header text", false},
		{"```javascript", false},
		{"This is just a regular sentence.", false},
		{"x", false},                      // Too short
		{strings.Repeat("a", 201), false}, // Too long
	}

	for _, tt := range tests {
		t.Run(tt.line, func(t *testing.T) {
			result := parser.looksLikeTask(tt.line)
			assert.Equal(t, tt.expected, result,
				"Expected looksLikeTask('%s') to be %v", tt.line, tt.expected)
		})
	}
}
