package agent

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// CostUnit represents the unit of cost measurement
type CostUnit string

const (
	// CostUnitUSD represents cost in US dollars
	CostUnitUSD CostUnit = "USD"
	
	// CostUnitTokens represents cost in tokens
	CostUnitTokens CostUnit = "tokens"
	
	// CostUnitComputeUnits represents cost in compute units
	CostUnitComputeUnits CostUnit = "compute_units"
)

// CostType represents the type of cost
type CostType string

const (
	// CostTypeLLM represents cost from LLM API calls
	CostTypeLLM CostType = "llm"
	
	// CostTypeEmbedding represents cost from embedding API calls
	CostTypeEmbedding CostType = "embedding"
	
	// CostTypeTool represents cost from tool usage
	CostTypeTool CostType = "tool"
	
	// CostTypeStorage represents cost from storage usage
	CostTypeStorage CostType = "storage"
	
	// CostTypeCompute represents cost from compute usage
	CostTypeCompute CostType = "compute"
)

// CostRecord represents a record of cost incurred
type CostRecord struct {
	// Type is the type of cost
	Type CostType
	
	// Amount is the amount of cost
	Amount float64
	
	// Unit is the unit of cost measurement
	Unit CostUnit
	
	// Description is a description of the cost
	Description string
	
	// Timestamp is when the cost was incurred
	Timestamp time.Time
	
	// Metadata contains additional information about the cost
	Metadata map[string]string
}

// CostManager manages cost tracking and optimization
type CostManager struct {
	// costs is a map of cost type to records
	costs map[CostType][]*CostRecord
	
	// budget is the maximum cost allowed
	budget map[CostType]float64
	
	// totalCost is the total cost incurred
	totalCost map[CostType]float64
	
	// defaultUnit is the default unit for costs
	defaultUnit CostUnit
	
	// modelCosts maps model names to per-token costs
	modelCosts map[string]float64
	
	// toolCosts maps tool names to per-use costs
	toolCosts map[string]float64
	
	// mu protects the cost manager
	mu sync.RWMutex
}

// NewCostManager creates a new cost manager
func NewCostManager(defaultUnit CostUnit) *CostManager {
	return &CostManager{
		costs:       make(map[CostType][]*CostRecord),
		budget:      make(map[CostType]float64),
		totalCost:   make(map[CostType]float64),
		defaultUnit: defaultUnit,
		modelCosts:  make(map[string]float64),
		toolCosts:   make(map[string]float64),
		mu:          sync.RWMutex{},
	}
}

// SetBudget sets the budget for a cost type
func (cm *CostManager) SetBudget(costType CostType, amount float64) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	
	cm.budget[costType] = amount
}

// GetBudget gets the budget for a cost type
func (cm *CostManager) GetBudget(costType CostType) float64 {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	
	return cm.budget[costType]
}

// SetModelCost sets the per-token cost for a model
func (cm *CostManager) SetModelCost(model string, cost float64) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	
	cm.modelCosts[model] = cost
}

// GetModelCost gets the per-token cost for a model
func (cm *CostManager) GetModelCost(model string) float64 {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	
	return cm.modelCosts[model]
}

// SetToolCost sets the per-use cost for a tool
func (cm *CostManager) SetToolCost(tool string, cost float64) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	
	cm.toolCosts[tool] = cost
}

// GetToolCost gets the per-use cost for a tool
func (cm *CostManager) GetToolCost(tool string) float64 {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	
	return cm.toolCosts[tool]
}

// RecordCost records a cost
func (cm *CostManager) RecordCost(record *CostRecord) {
	if record == nil {
		return
	}
	
	// Set timestamp if not set
	if record.Timestamp.IsZero() {
		record.Timestamp = time.Now().UTC()
	}
	
	// Set unit if not set
	if record.Unit == "" {
		record.Unit = cm.defaultUnit
	}
	
	cm.mu.Lock()
	defer cm.mu.Unlock()
	
	// Add cost record
	records := cm.costs[record.Type]
	records = append(records, record)
	cm.costs[record.Type] = records
	
	// Update total cost
	cm.totalCost[record.Type] += record.Amount
}

// RecordLLMCost records an LLM API call cost
func (cm *CostManager) RecordLLMCost(model string, promptTokens, completionTokens int, metadata map[string]string) {
	cm.mu.RLock()
	costPerToken, exists := cm.modelCosts[model]
	cm.mu.RUnlock()
	
	if !exists {
		// Default cost estimation if not specified
		costPerToken = 0.000002 // $0.000002 per token as a default
	}
	
	totalTokens := promptTokens + completionTokens
	cost := float64(totalTokens) * costPerToken
	
	description := fmt.Sprintf("LLM API call: %s (prompt: %d tokens, completion: %d tokens)", 
		model, promptTokens, completionTokens)
	
	cm.RecordCost(&CostRecord{
		Type:        CostTypeLLM,
		Amount:      cost,
		Unit:        CostUnitUSD,
		Description: description,
		Timestamp:   time.Now().UTC(),
		Metadata:    metadata,
	})
}

// RecordToolCost records a tool usage cost
func (cm *CostManager) RecordToolCost(toolName string, metadata map[string]string) {
	cm.mu.RLock()
	costPerUse, exists := cm.toolCosts[toolName]
	cm.mu.RUnlock()
	
	if !exists {
		// Default cost for tools if not specified
		costPerUse = 0.0 // Free by default
	}
	
	description := fmt.Sprintf("Tool usage: %s", toolName)
	
	cm.RecordCost(&CostRecord{
		Type:        CostTypeTool,
		Amount:      costPerUse,
		Unit:        CostUnitUSD,
		Description: description,
		Timestamp:   time.Now().UTC(),
		Metadata:    metadata,
	})
}

// GetTotalCost gets the total cost for a cost type
func (cm *CostManager) GetTotalCost(costType CostType) float64 {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	
	return cm.totalCost[costType]
}

// GetAllCosts gets the total cost for all cost types
func (cm *CostManager) GetAllCosts() map[CostType]float64 {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	
	// Make a copy to avoid concurrent map access
	costs := make(map[CostType]float64, len(cm.totalCost))
	for costType, amount := range cm.totalCost {
		costs[costType] = amount
	}
	
	return costs
}

// GetCostRecords gets all cost records for a cost type
func (cm *CostManager) GetCostRecords(costType CostType) []*CostRecord {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	
	records := cm.costs[costType]
	
	// Make a copy to avoid concurrent slice access
	result := make([]*CostRecord, len(records))
	copy(result, records)
	
	return result
}

// IsWithinBudget checks if a cost type is within budget
func (cm *CostManager) IsWithinBudget(costType CostType) bool {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	
	budget, hasBudget := cm.budget[costType]
	if !hasBudget {
		// No budget set, assume it's within budget
		return true
	}
	
	return cm.totalCost[costType] < budget
}

// CanAfford checks if a cost can be afforded
func (cm *CostManager) CanAfford(costType CostType, amount float64) bool {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	
	budget, hasBudget := cm.budget[costType]
	if !hasBudget {
		// No budget set, assume it can be afforded
		return true
	}
	
	return cm.totalCost[costType] + amount < budget
}

// EstimateLLMCost estimates the cost of an LLM API call
func (cm *CostManager) EstimateLLMCost(model string, promptTokens, maxCompletionTokens int) float64 {
	cm.mu.RLock()
	costPerToken, exists := cm.modelCosts[model]
	cm.mu.RUnlock()
	
	if !exists {
		// Default cost estimation if not specified
		costPerToken = 0.000002 // $0.000002 per token as a default
	}
	
	totalTokens := promptTokens + maxCompletionTokens
	return float64(totalTokens) * costPerToken
}

// SelectCostEfficientModel selects the most cost-efficient model for a task
func (cm *CostManager) SelectCostEfficientModel(models []string, promptTokens, requiredCompletionTokens int) (string, error) {
	if len(models) == 0 {
		return "", fmt.Errorf("no models provided")
	}
	
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	
	// Default to the first model
	bestModel := models[0]
	bestCost := float64(-1)
	
	for _, model := range models {
		costPerToken, exists := cm.modelCosts[model]
		if !exists {
			// Skip models with unknown cost
			continue
		}
		
		totalTokens := promptTokens + requiredCompletionTokens
		cost := float64(totalTokens) * costPerToken
		
		if bestCost < 0 || cost < bestCost {
			bestModel = model
			bestCost = cost
		}
	}
	
	return bestModel, nil
}

// GetCostReport generates a report of costs
func (cm *CostManager) GetCostReport() map[string]interface{} {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	
	report := make(map[string]interface{})
	
	// Add total costs
	totalCosts := make(map[string]float64)
	for costType, amount := range cm.totalCost {
		totalCosts[string(costType)] = amount
	}
	report["total_costs"] = totalCosts
	
	// Add budgets
	budgets := make(map[string]float64)
	for costType, amount := range cm.budget {
		budgets[string(costType)] = amount
	}
	report["budgets"] = budgets
	
	// Add model costs
	report["model_costs"] = cm.modelCosts
	
	// Add tool costs
	report["tool_costs"] = cm.toolCosts
	
	// Add cost records summary
	recordsSummary := make(map[string]int)
	for costType, records := range cm.costs {
		recordsSummary[string(costType)] = len(records)
	}
	report["records_count"] = recordsSummary
	
	return report
}