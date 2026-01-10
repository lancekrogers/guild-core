// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package scheduler

import (
	"context"
	"sync"
	"time"

	"github.com/lancekrogers/guild-core/pkg/kanban"
)

// AgentCapability represents what an agent can do
type AgentCapability string

const (
	CapabilityCode     AgentCapability = "code"
	CapabilityReview   AgentCapability = "review"
	CapabilityTest     AgentCapability = "test"
	CapabilityDocument AgentCapability = "document"
	CapabilityPlan     AgentCapability = "plan"
)

// AgentInfo tracks agent availability and capabilities
type AgentInfo struct {
	AgentID      string
	Capabilities []AgentCapability
	IsAvailable  bool
	CurrentTask  string
	LastAssigned time.Time
	TasksHandled int
	ErrorRate    float64 // Percentage of failed tasks
}

// ManagerAgentClient interfaces with the AI manager agent for task assignment decisions
type ManagerAgentClient interface {
	// RequestAssignment asks the manager agent to assign a task
	RequestAssignment(ctx context.Context, task *kanban.Task, availableAgents []*AgentInfo) (*TaskAssignment, error)

	// ReviewProgress asks the manager to review current progress
	ReviewProgress(ctx context.Context, progress map[string]*CommissionProgress) ([]string, error)
}

// TaskAssignment represents the manager's decision on task assignment
type TaskAssignment struct {
	TaskID      string
	AgentID     string
	Reasoning   string    // Why the manager chose this agent
	APIProvider string    // Which LLM provider to use
	Priority    int       // Adjusted priority based on manager's assessment
	Deadline    time.Time // When the task should be completed
}

// KanbanClient provides atomic operations on the kanban board
type KanbanClient interface {
	// GetTaskForUpdate retrieves a task with row-level lock
	GetTaskForUpdate(ctx context.Context, taskID string) (*kanban.Task, error)

	// AssignTaskAtomic atomically assigns a task to an agent
	AssignTaskAtomic(ctx context.Context, taskID, agentID string) error

	// UpdateTaskStatusAtomic atomically updates task status
	UpdateTaskStatusAtomic(ctx context.Context, taskID string, status kanban.TaskStatus) error

	// WithTransaction runs operations in a database transaction
	WithTransaction(ctx context.Context, fn func(ctx context.Context) error) error
}

// OrchestratorConfig configures the agent orchestrator
type OrchestratorConfig struct {
	MaxConcurrentTasks  int
	DefaultTaskTimeout  time.Duration
	ManagerAgentTimeout time.Duration
	EnableAutoRetry     bool
	MaxRetries          int
	RateLimitConfigs    map[string]RateLimitConfig
}

// RateLimitConfig defines rate limiting for an API provider
type RateLimitConfig struct {
	Provider    string
	MaxRequests int
	Window      time.Duration
}

// OrchestratorMetrics tracks orchestrator performance
type OrchestratorMetrics struct {
	TasksAssigned    int64
	TasksCompleted   int64
	TasksFailed      int64
	AssignmentTime   time.Duration
	AverageWaitTime  time.Duration
	AgentUtilization map[string]float64
	mu               sync.RWMutex
}
