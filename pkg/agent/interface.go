// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package agent

import (
	"context"

	"github.com/guild-ventures/guild-core/pkg/interfaces"
)

// CostManagerInterface defines the interface for tracking and managing agent operation costs
type CostManagerInterface interface {
	// TrackCost records a cost for a specific operation type
	TrackCost(costType CostType, amount float64) error

	// GetCostReport returns a detailed cost report
	GetCostReport() map[string]interface{}

	// SetBudget sets a budget limit for a specific cost type
	SetBudget(costType CostType, amount float64)

	// GetBudgetRemaining returns the remaining budget for a cost type
	GetBudgetRemaining(costType CostType) float64

	// GetTotalCost returns the total accumulated cost
	GetTotalCost() float64

	// Reset clears all cost tracking data
	Reset()

	// ExceedsBudget checks if a cost would exceed the budget
	ExceedsBudget(costType CostType, amount float64) bool

	// EstimateLLMCost estimates the cost of an LLM operation
	EstimateLLMCost(model string, estimatedTokens int) float64

	// CanAfford checks if a cost can be afforded within the budget
	CanAfford(costType CostType, amount float64) bool

	// RecordLLMCost records the actual cost of an LLM operation
	RecordLLMCost(model string, promptTokens, completionTokens int, metadata map[string]string) error
}

// AgentRegistry defines the interface for agent registration and discovery
// This interface is defined here to avoid circular dependencies with the registry package
type AgentRegistry interface {
	// GetAgentsByCost returns agents with cost magnitude <= maxCost, sorted by cost
	GetAgentsByCost(maxCost int) []AgentInfo

	// GetCheapestAgentByCapability returns the lowest-cost agent with the given capability
	GetCheapestAgentByCapability(capability string) (*AgentInfo, error)

	// GetAgentsByCapability returns all agents that have the specified capability
	GetAgentsByCapability(capability string) []AgentInfo

	// GetRegisteredAgents returns all registered agent configurations
	GetRegisteredAgents() []GuildAgentConfig
}

// GuildAgentConfig represents a configured agent from guild config
// Re-export shared agent configuration type from the interfaces package.
type GuildAgentConfig = interfaces.GuildAgentConfig

// AgentInfo holds agent information for registry operations
// Re-export shared agent information type from the interfaces package.
type AgentInfo = interfaces.AgentInfo

// CostProfile represents the cost characteristics of an agent
// Re-export cost profile type from the interfaces package.
type CostProfile = interfaces.CostProfile

// CommissionRepository defines the interface for commission storage operations
// This interface is defined here to avoid circular dependencies with the registry package
type CommissionRepository interface {
	CreateCommission(ctx context.Context, commission *Commission) error
	GetCommission(ctx context.Context, id string) (*Commission, error)
}

// Commission represents a commission in the system
// For now, using a simplified version to avoid importing registry
type Commission struct {
	ID          string
	CampaignID  string
	Title       string
	Description *string
	Status      string
}

// CostAwareClient extends the basic cost tracking with client-specific operations
type CostAwareClient interface {
	CostManagerInterface

	// EstimateCost estimates the cost of an operation before execution
	EstimateCost(ctx context.Context, operation string, params map[string]interface{}) (float64, error)

	// GetCostHistory returns historical cost data
	GetCostHistory() []CostEntry
}

// CostEntry represents a single cost tracking entry
type CostEntry struct {
	Timestamp int64
	CostType  CostType
	Amount    float64
	Operation string
	Details   map[string]interface{}
}

// TaskExecutor defines the interface for executing tasks with tools
type TaskExecutor interface {
	// ExecuteTask executes a task with optional tool usage
	ExecuteTask(ctx context.Context, task Task) (*TaskResult, error)

	// CanExecute checks if the executor can handle a task
	CanExecute(task Task) bool
}

// Task represents a unit of work for an agent
type Task struct {
	ID          string
	Type        string
	Description string
	Parameters  map[string]interface{}
	Tools       []string // Tool names that may be needed
}

// TaskResult represents the outcome of task execution
type TaskResult struct {
	Success   bool
	Output    string
	ToolsUsed []string
	Cost      float64
	Error     error
	Metadata  map[string]interface{}
}
