package mocks

import (
	"sync"

	"github.com/guild-ventures/guild-core/pkg/agent"
)

// MockCostManager implements the agent.CostManagerInterface for testing
type MockCostManager struct {
	mu        sync.RWMutex
	costs     map[agent.CostType]float64
	budgets   map[agent.CostType]float64
	history   []agent.CostEntry
	error     error
	totalCost float64
}

// NewMockCostManager creates a new mock cost manager
func NewMockCostManager() *MockCostManager {
	return &MockCostManager{
		costs:   make(map[agent.CostType]float64),
		budgets: make(map[agent.CostType]float64),
		history: make([]agent.CostEntry, 0),
	}
}

// WithError configures the mock to return an error
func (m *MockCostManager) WithError(err error) *MockCostManager {
	m.error = err
	return m
}

// TrackCost implements agent.CostManagerInterface.TrackCost
func (m *MockCostManager) TrackCost(costType agent.CostType, amount float64) error {
	if m.error != nil {
		return m.error
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.costs[costType] += amount
	m.totalCost += amount

	return nil
}

// GetCostReport implements agent.CostManagerInterface.GetCostReport
func (m *MockCostManager) GetCostReport() map[string]interface{} {
	if m.error != nil {
		return map[string]interface{}{"error": m.error.Error()}
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	report := make(map[string]interface{})
	report["total_cost"] = m.totalCost
	report["costs_by_type"] = m.costs
	report["budgets"] = m.budgets

	return report
}

// SetBudget implements agent.CostManagerInterface.SetBudget
func (m *MockCostManager) SetBudget(costType agent.CostType, amount float64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.budgets[costType] = amount
}

// GetBudgetRemaining implements agent.CostManagerInterface.GetBudgetRemaining
func (m *MockCostManager) GetBudgetRemaining(costType agent.CostType) float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()

	budget, exists := m.budgets[costType]
	if !exists {
		return 0
	}

	spent := m.costs[costType]
	return budget - spent
}

// GetTotalCost implements agent.CostManagerInterface.GetTotalCost
func (m *MockCostManager) GetTotalCost() float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.totalCost
}

// Reset implements agent.CostManagerInterface.Reset
func (m *MockCostManager) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.costs = make(map[agent.CostType]float64)
	m.budgets = make(map[agent.CostType]float64)
	m.history = make([]agent.CostEntry, 0)
	m.totalCost = 0
}

// ExceedsBudget implements agent.CostManagerInterface.ExceedsBudget
func (m *MockCostManager) ExceedsBudget(costType agent.CostType, amount float64) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	budget, exists := m.budgets[costType]
	if !exists {
		return false // No budget set
	}

	spent := m.costs[costType]
	return (spent + amount) > budget
}

// EstimateLLMCost implements agent.CostManagerInterface.EstimateLLMCost
func (m *MockCostManager) EstimateLLMCost(model string, estimatedTokens int) float64 {
	// Return a simple estimate for testing
	return float64(estimatedTokens) * 0.001 // $0.001 per token
}

// CanAfford implements agent.CostManagerInterface.CanAfford
func (m *MockCostManager) CanAfford(costType agent.CostType, amount float64) bool {
	return !m.ExceedsBudget(costType, amount)
}

// RecordLLMCost implements agent.CostManagerInterface.RecordLLMCost
func (m *MockCostManager) RecordLLMCost(model string, promptTokens, completionTokens int, metadata map[string]string) error {
	if m.error != nil {
		return m.error
	}

	// Simple cost calculation for testing
	cost := (float64(promptTokens) * 0.001) + (float64(completionTokens) * 0.002)
	return m.TrackCost(agent.CostTypeLLM, cost)
}
