package agent

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// Test GetTotalCost
func TestCostManager_GetTotalCost(t *testing.T) {
	tests := []struct {
		name       string
		setupCosts func(*CostManager)
		expected   float64
	}{
		{
			name: "empty costs",
			setupCosts: func(cm *CostManager) {
				// No costs added
			},
			expected: 0.0,
		},
		{
			name: "single cost type",
			setupCosts: func(cm *CostManager) {
				cm.costs[CostTypeLLM] = 10.5
			},
			expected: 10.5,
		},
		{
			name: "multiple cost types",
			setupCosts: func(cm *CostManager) {
				cm.costs[CostTypeLLM] = 10.5
				cm.costs[CostTypeTool] = 5.25
				cm.costs[CostTypeMemory] = 2.75
			},
			expected: 18.5,
		},
		{
			name: "all cost types",
			setupCosts: func(cm *CostManager) {
				cm.costs[CostTypeLLM] = 1.0
				cm.costs[CostTypeTool] = 2.0
				cm.costs[CostTypeMemory] = 3.0
				cm.costs[CostTypeTotal] = 6.0 // Should be included
			},
			expected: 12.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cm := &CostManager{
				costs:   make(map[CostType]float64),
				budgets: make(map[CostType]float64),
			}
			
			tt.setupCosts(cm)
			
			total := cm.GetTotalCost()
			assert.Equal(t, tt.expected, total)
		})
	}
}

// Test ExceedsBudget
func TestCostManager_ExceedsBudget(t *testing.T) {
	tests := []struct {
		name      string
		setupCM   func() *CostManager
		costType  CostType
		amount    float64
		expected  bool
	}{
		{
			name: "within budget",
			setupCM: func() *CostManager {
				cm := &CostManager{
					costs:   make(map[CostType]float64),
					budgets: make(map[CostType]float64),
				}
				cm.budgets[CostTypeLLM] = 100.0
				cm.costs[CostTypeLLM] = 50.0
				return cm
			},
			costType: CostTypeLLM,
			amount:   30.0,
			expected: false,
		},
		{
			name: "exactly at budget",
			setupCM: func() *CostManager {
				cm := &CostManager{
					costs:   make(map[CostType]float64),
					budgets: make(map[CostType]float64),
				}
				cm.budgets[CostTypeLLM] = 100.0
				cm.costs[CostTypeLLM] = 50.0
				return cm
			},
			costType: CostTypeLLM,
			amount:   50.0,
			expected: false,
		},
		{
			name: "exceeds budget",
			setupCM: func() *CostManager {
				cm := &CostManager{
					costs:   make(map[CostType]float64),
					budgets: make(map[CostType]float64),
				}
				cm.budgets[CostTypeLLM] = 100.0
				cm.costs[CostTypeLLM] = 90.0
				return cm
			},
			costType: CostTypeLLM,
			amount:   20.0,
			expected: true,
		},
		{
			name: "no budget set",
			setupCM: func() *CostManager {
				cm := &CostManager{
					costs:   make(map[CostType]float64),
					budgets: make(map[CostType]float64),
				}
				// No budget for this type
				return cm
			},
			costType: CostTypeLLM,
			amount:   10.0,
			expected: false, // No budget means no limit
		},
		{
			name: "zero budget",
			setupCM: func() *CostManager {
				cm := &CostManager{
					costs:   make(map[CostType]float64),
					budgets: make(map[CostType]float64),
				}
				cm.budgets[CostTypeTool] = 0.0
				return cm
			},
			costType: CostTypeTool,
			amount:   0.001,
			expected: true, // Any amount exceeds zero budget
		},
		{
			name: "negative budget (edge case)",
			setupCM: func() *CostManager {
				cm := &CostManager{
					costs:   make(map[CostType]float64),
					budgets: make(map[CostType]float64),
				}
				cm.budgets[CostTypeMemory] = -10.0
				return cm
			},
			costType: CostTypeMemory,
			amount:   1.0,
			expected: true, // Any positive amount exceeds negative budget
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cm := tt.setupCM()
			result := cm.ExceedsBudget(tt.costType, tt.amount)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Test concurrent access to GetTotalCost
func TestCostManager_GetTotalCost_Concurrent(t *testing.T) {
	cm := &CostManager{
		costs:   make(map[CostType]float64),
		budgets: make(map[CostType]float64),
	}
	
	// Set initial costs
	cm.costs[CostTypeLLM] = 100.0
	cm.costs[CostTypeTool] = 50.0
	cm.costs[CostTypeMemory] = 25.0
	
	// Run concurrent reads
	numGoroutines := 100
	results := make(chan float64, numGoroutines)
	
	for i := 0; i < numGoroutines; i++ {
		go func() {
			total := cm.GetTotalCost()
			results <- total
		}()
	}
	
	// Collect results
	for i := 0; i < numGoroutines; i++ {
		total := <-results
		assert.Equal(t, 175.0, total, "Total cost should be consistent")
	}
}

// Test ExceedsBudget with concurrent modifications
func TestCostManager_ExceedsBudget_Concurrent(t *testing.T) {
	cm := &CostManager{
		costs:   make(map[CostType]float64),
		budgets: make(map[CostType]float64),
	}
	
	cm.budgets[CostTypeLLM] = 1000.0
	cm.costs[CostTypeLLM] = 0.0
	
	// Run concurrent checks while modifying costs
	numGoroutines := 50
	results := make(chan bool, numGoroutines*2)
	
	// Half goroutines check budget
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			amount := float64(id) * 10.0
			exceeds := cm.ExceedsBudget(CostTypeLLM, amount)
			results <- exceeds
		}(i)
	}
	
	// Other half track costs
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			cm.mu.Lock()
			cm.costs[CostTypeLLM] += float64(id)
			cm.mu.Unlock()
			results <- true // Just to balance channel reads
		}(i)
	}
	
	// Collect all results
	for i := 0; i < numGoroutines*2; i++ {
		<-results
	}
	
	// Final state should be consistent
	finalTotal := cm.GetTotalCost()
	assert.Greater(t, finalTotal, 0.0)
}

// Test edge cases
func TestCostManager_EdgeCases(t *testing.T) {
	t.Run("GetTotalCost with nil maps", func(t *testing.T) {
		cm := &CostManager{
			// costs map is nil
		}
		
		// Should not panic
		assert.NotPanics(t, func() {
			total := cm.GetTotalCost()
			assert.Equal(t, 0.0, total)
		})
	})
	
	t.Run("ExceedsBudget with nil maps", func(t *testing.T) {
		cm := &CostManager{
			// budgets and costs maps are nil
		}
		
		// Should not panic
		assert.NotPanics(t, func() {
			exceeds := cm.ExceedsBudget(CostTypeLLM, 10.0)
			assert.False(t, exceeds) // No budget means no limit
		})
	})
	
	t.Run("very large costs", func(t *testing.T) {
		cm := &CostManager{
			costs:   make(map[CostType]float64),
			budgets: make(map[CostType]float64),
		}
		
		// Set very large values
		cm.costs[CostTypeLLM] = 1e15
		cm.costs[CostTypeTool] = 1e15
		cm.costs[CostTypeMemory] = 1e15
		
		total := cm.GetTotalCost()
		assert.Equal(t, 3e15, total)
		
		// Test budget with large values
		cm.budgets[CostTypeTotal] = 1e16
		exceeds := cm.ExceedsBudget(CostTypeTotal, 1e15)
		assert.False(t, exceeds)
	})
}