package agent

// Cost represents the cost of using an LLM
type Cost struct {
	// Cost in USD
	USD float64
	
	// Tokens used
	Tokens int
}

// CostTracker is an interface for tracking LLM costs
type CostTracker interface {
	// TrackCost tracks the cost of a request
	TrackCost(cost Cost)
	
	// GetTotalCost returns the total cost
	GetTotalCost() Cost
	
	// GetCostBudget returns the cost budget
	GetCostBudget() float64
	
	// SetCostBudget sets the cost budget
	SetCostBudget(budget float64)
	
	// HasExceededBudget returns true if the budget has been exceeded
	HasExceededBudget() bool
}