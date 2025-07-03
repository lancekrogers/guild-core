package managers

import "github.com/lancekrogers/guild/pkg/agents/core"

// MinimalCostManager provides cost tracking for the chat system without budget enforcement
// This supports the cost magnitude system used by the manager agent for resource selection
type MinimalCostManager struct{}

func (mcm *MinimalCostManager) TrackCost(costType core.CostType, amount float64) error {
	return nil // Placeholder implementation
}

func (mcm *MinimalCostManager) GetCostReport() map[string]interface{} {
	return map[string]interface{}{} // Placeholder implementation
}

func (mcm *MinimalCostManager) GetTotalCost() float64 {
	return 0.0 // Placeholder implementation
}

func (mcm *MinimalCostManager) Reset() {
	// Placeholder implementation
}

func (mcm *MinimalCostManager) EstimateLLMCost(model string, estimatedTokens int) float64 {
	return 0.0 // Placeholder implementation
}

func (mcm *MinimalCostManager) RecordLLMCost(model string, promptTokens, completionTokens int, metadata map[string]string) error {
	return nil // Placeholder implementation
}

func (mcm *MinimalCostManager) SetBudget(costType core.CostType, amount float64) {
	// Placeholder implementation
}

func (mcm *MinimalCostManager) GetBudgetRemaining(costType core.CostType) float64 {
	return 0.0 // Placeholder implementation
}

func (mcm *MinimalCostManager) ExceedsBudget(costType core.CostType, amount float64) bool {
	return false // Placeholder implementation
}

func (mcm *MinimalCostManager) CanAfford(costType core.CostType, amount float64) bool {
	return true // Placeholder implementation - always return true for testing
}
