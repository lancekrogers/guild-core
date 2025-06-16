// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package executor

import (
	"context"
	"time"

	"github.com/guild-ventures/guild-core/pkg/kanban"
)

// TaskExecutor defines the interface for agent task execution
type TaskExecutor interface {
	// Execute runs the task execution loop for a given task
	Execute(ctx context.Context, task *kanban.Task) (*ExecutionResult, error)

	// GetProgress returns the current execution progress (0.0 to 1.0)
	GetProgress() float64

	// GetStatus returns the current execution status
	GetStatus() ExecutionStatus

	// Stop gracefully stops the execution
	Stop() error
}

// ExecutionStatus represents the current state of task execution
type ExecutionStatus string

const (
	StatusInitializing ExecutionStatus = "initializing"
	StatusRunning      ExecutionStatus = "running"
	StatusPaused       ExecutionStatus = "paused"
	StatusCompleted    ExecutionStatus = "completed"
	StatusFailed       ExecutionStatus = "failed"
	StatusStopped      ExecutionStatus = "stopped"
)

// ExecutionResult contains the outcome of a task execution
type ExecutionResult struct {
	TaskID    string                 `json:"task_id"`
	Status    ExecutionStatus        `json:"status"`
	StartTime time.Time              `json:"start_time"`
	EndTime   time.Time              `json:"end_time"`
	Duration  time.Duration          `json:"duration"`
	Output    string                 `json:"output"`
	Artifacts []Artifact             `json:"artifacts"`
	ToolUsage []ToolUsage            `json:"tool_usage"`
	Errors    []ExecutionError       `json:"errors,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// Artifact represents a file or resource created during execution
type Artifact struct {
	Name        string    `json:"name"`
	Type        string    `json:"type"`
	Path        string    `json:"path"`
	Size        int64     `json:"size"`
	CreatedAt   time.Time `json:"created_at"`
	Description string    `json:"description,omitempty"`
}

// ToolUsage tracks which tools were used during execution
type ToolUsage struct {
	ToolName    string                   `json:"tool_name"`
	Invocations int                      `json:"invocations"`
	TotalTime   time.Duration            `json:"total_time"`
	Results     []map[string]interface{} `json:"results,omitempty"`
}

// ExecutionError represents an error that occurred during execution
type ExecutionError struct {
	Phase     string    `json:"phase"`
	Error     string    `json:"error"`
	Timestamp time.Time `json:"timestamp"`
	Retryable bool      `json:"retryable"`
}

// ExecutionContext provides context for task execution
type ExecutionContext struct {
	WorkspaceDir string                 // Isolated workspace directory
	ProjectRoot  string                 // Project root directory
	Commission   string                 // Parent commission description
	AgentID      string                 // Executing agent ID
	AgentType    string                 // Agent type (manager, worker, etc)
	Capabilities []string               // Agent capabilities
	Tools        []string               // Available tools
	Metadata     map[string]interface{} // Additional context
}
