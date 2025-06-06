package agent

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

// CostType represents a category of costs
type CostType string

const (
	// CostTypeLLM represents costs associated with language model API calls
	CostTypeLLM CostType = "llm"
	
	// CostTypeEmbedding represents costs for creating and storing vector embeddings
	CostTypeEmbedding CostType = "embedding"
	
	// CostTypeTool represents costs for using specialized tools
	CostTypeTool CostType = "tool"
	
	// CostTypeStorage represents costs for data storage and retrieval
	CostTypeStorage CostType = "storage"
	
	// CostTypeCompute represents costs for computational resources
	CostTypeCompute CostType = "compute"
)

// CostUnit represents the unit of measurement for costs
type CostUnit string

const (
	// CostUnitUSD represents costs in US dollars
	CostUnitUSD CostUnit = "usd"
	
	// CostUnitTokens represents costs in tokens
	CostUnitTokens CostUnit = "tokens"
)

// Cost represents the cost of using an LLM
type Cost struct {
	// Cost in USD
	USD float64
	
	// Tokens used
	Tokens int
}

// CostRecord represents a single cost entry
type CostRecord struct {
	// Type of cost
	Type CostType
	
	// Amount of cost
	Amount float64
	
	// Unit of cost measurement
	Unit CostUnit
	
	// Description of the cost
	Description string
	
	// When the cost was incurred
	Timestamp time.Time
	
	// Additional information
	Metadata map[string]string
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

// CostManager manages costs for different types and enforces budgets
type CostManager struct {
	// Map of cost type to records
	costs map[CostType][]*CostRecord
	
	// Budgets by cost type
	budgets map[CostType]float64
	
	// Total costs by type
	totalCosts map[CostType]float64
	
	// Default unit for costs
	defaultUnit CostUnit
	
	// Model costs (model name -> cost per 1K tokens)
	modelCosts map[string]float64
	
	// Tool costs (tool name -> cost per use)
	toolCosts map[string]float64
	
	// Thread safety
	mu sync.RWMutex
}

// newCostManager creates a new cost manager with default settings
func newCostManager() *CostManager {
	return &CostManager{
		costs:       make(map[CostType][]*CostRecord),
		budgets:     make(map[CostType]float64),
		totalCosts:  make(map[CostType]float64),
		defaultUnit: CostUnitUSD,
		modelCosts:  make(map[string]float64),
		toolCosts:   make(map[string]float64),
	}
}

// SetBudget sets the budget for a specific cost type
func (cm *CostManager) SetBudget(costType CostType, amount float64) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.budgets[costType] = amount
}

// GetBudget returns the budget for a specific cost type
func (cm *CostManager) GetBudget(costType CostType) float64 {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.budgets[costType]
}

// RecordCost records a cost
func (cm *CostManager) RecordCost(costType CostType, amount float64, description string, metadata map[string]string) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	
	record := &CostRecord{
		Type:        costType,
		Amount:      amount,
		Unit:        cm.defaultUnit,
		Description: description,
		Timestamp:   time.Now().UTC(),
		Metadata:    metadata,
	}
	
	cm.costs[costType] = append(cm.costs[costType], record)
	cm.totalCosts[costType] += amount
}

// RecordLLMCost records the cost of an LLM API call
func (cm *CostManager) RecordLLMCost(model string, promptTokens, completionTokens int, metadata map[string]string) float64 {
	// Get cost rate for model ($ per 1K tokens)
	promptRate, completionRate := cm.getLLMCostRates(model)
	
	// Calculate costs
	promptCost := float64(promptTokens) * promptRate / 1000.0
	completionCost := float64(completionTokens) * completionRate / 1000.0
	totalCost := promptCost + completionCost
	
	// Record the cost
	description := model + " API call ("
	description += "prompt: " + fmt.Sprintf("%d", promptTokens) + " tokens, "
	description += "completion: " + fmt.Sprintf("%d", completionTokens) + " tokens)"
	
	cm.RecordCost(CostTypeLLM, totalCost, description, metadata)
	
	return totalCost
}

// getLLMCostRates returns the cost rates for prompt and completion tokens
func (cm *CostManager) getLLMCostRates(model string) (float64, float64) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	
	// Check if we have specific rates for this model
	if rate, ok := cm.modelCosts[model]; ok {
		// For simplicity, use the same rate for both prompt and completion
		return rate, rate
	}
	
	// Default rates if not specified
	switch {
	case strings.Contains(model, "claude-3-opus"):
		return 0.01500, 0.04500  // $15.00 per 1M prompt, $45.00 per 1M completion
	case strings.Contains(model, "claude-3-sonnet"):
		return 0.00300, 0.01500  // $3.00 per 1M prompt, $15.00 per 1M completion
	case strings.Contains(model, "claude-3-haiku"):
		return 0.00025, 0.00125  // $0.25 per 1M prompt, $1.25 per 1M completion
	case strings.Contains(model, "gpt-4-turbo"):
		return 0.00100, 0.00300  // $1.00 per 1M prompt, $3.00 per 1M completion
	case strings.Contains(model, "gpt-4"):
		return 0.03000, 0.06000  // $30.00 per 1M prompt, $60.00 per 1M completion
	case strings.Contains(model, "gpt-3.5-turbo"):
		return 0.00050, 0.00150  // $0.50 per 1M prompt, $1.50 per 1M completion
	default:
		// Default to a reasonable rate for unknown models
		return 0.00100, 0.00200
	}
}

// RecordToolCost records the cost of using a tool
func (cm *CostManager) RecordToolCost(toolName string, metadata map[string]string) float64 {
	cm.mu.RLock()
	// Get cost rate for tool
	rate, ok := cm.toolCosts[toolName]
	if !ok {
		// Default rate for unknown tools
		rate = 0.01
	}
	cm.mu.RUnlock()
	
	// Record the cost
	description := "Tool usage: " + toolName
	cm.RecordCost(CostTypeTool, rate, description, metadata)
	
	return rate
}

// GetTotalCost returns the total cost for a specific type
func (cm *CostManager) GetTotalCost(costType CostType) float64 {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.totalCosts[costType]
}

// CanAfford checks if a cost can be afforded within the budget
func (cm *CostManager) CanAfford(costType CostType, amount float64) bool {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	
	budget, hasBudget := cm.budgets[costType]
	if !hasBudget || budget <= 0 {
		// No budget set or unlimited budget
		return true
	}
	
	current := cm.totalCosts[costType]
	return (current + amount) <= budget
}

// EstimateLLMCost estimates the cost of an LLM API call
func (cm *CostManager) EstimateLLMCost(model string, promptTokens, maxCompletionTokens int) float64 {
	// Get cost rates
	promptRate, completionRate := cm.getLLMCostRates(model)
	
	// Calculate costs
	promptCost := float64(promptTokens) * promptRate / 1000.0
	
	// Use max completion tokens for estimation (worst case)
	completionCost := float64(maxCompletionTokens) * completionRate / 1000.0
	
	return promptCost + completionCost
}

// GetCostReport returns a report of all costs
func (cm *CostManager) GetCostReport() map[string]interface{} {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	
	// Copy totals
	totalCosts := make(map[string]float64)
	for t, c := range cm.totalCosts {
		totalCosts[string(t)] = c
	}
	
	// Copy budgets
	budgets := make(map[string]float64)
	for t, b := range cm.budgets {
		budgets[string(t)] = b
	}
	
	// Count records
	recordCounts := make(map[string]int)
	for t, records := range cm.costs {
		recordCounts[string(t)] = len(records)
	}
	
	// Calculate totals
	grandTotal := 0.0
	for _, c := range cm.totalCosts {
		grandTotal += c
	}
	
	return map[string]interface{}{
		"total_costs":   totalCosts,
		"budgets":       budgets,
		"record_counts": recordCounts,
		"grand_total":   grandTotal,
	}
}

// TrackCost implements the CostManager interface
func (cm *CostManager) TrackCost(costType CostType, amount float64) error {
	cm.RecordCost(costType, amount, "", nil)
	return nil
}

// GetBudgetRemaining implements the CostManager interface
func (cm *CostManager) GetBudgetRemaining(costType CostType) float64 {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	
	budget := cm.budgets[costType]
	spent := cm.totalCosts[costType]
	return budget - spent
}

// GetTotalCost implements the CostManager interface
func (cm *CostManager) GetTotalCost() float64 {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	
	total := 0.0
	for _, cost := range cm.totalCosts {
		total += cost
	}
	return total
}

// Reset implements the CostManager interface
func (cm *CostManager) Reset() {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	
	cm.costs = make(map[CostType][]*CostRecord)
	cm.totalCosts = make(map[CostType]float64)
	// Keep budgets intact
}

// ExceedsBudget implements the CostManager interface
func (cm *CostManager) ExceedsBudget(costType CostType, amount float64) bool {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	
	budget := cm.budgets[costType]
	if budget == 0 {
		return false // No budget set means no limit
	}
	
	currentSpent := cm.totalCosts[costType]
	return currentSpent + amount > budget
}

// DefaultCostManagerFactory is a factory function for creating cost managers
// This should be registered with the agent registry
func DefaultCostManagerFactory() CostManager {
	return newCostManager()
}