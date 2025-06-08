package agent

import (
	"context"
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
