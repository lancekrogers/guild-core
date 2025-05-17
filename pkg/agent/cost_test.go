package agent

import (
	"testing"
	"time"
)

func TestCostManager_BasicOperations(t *testing.T) {
	cm := NewCostManager()
	
	// Test setting budgets
	cm.SetBudget(CostTypeLLM, 10.0)
	cm.SetBudget(CostTypeTool, 5.0)
	
	// Test getting budgets
	llmBudget := cm.GetBudget(CostTypeLLM)
	if llmBudget != 10.0 {
		t.Errorf("Expected LLM budget of 10.0, got %f", llmBudget)
	}
	
	// Test recording costs
	metadata := map[string]string{
		"agent_id": "test-agent",
		"task_id":  "test-task",
	}
	
	cm.RecordCost(CostTypeLLM, 2.5, "Test LLM call", metadata)
	
	// Test getting total cost
	totalLLMCost := cm.GetTotalCost(CostTypeLLM)
	if totalLLMCost != 2.5 {
		t.Errorf("Expected total LLM cost of 2.5, got %f", totalLLMCost)
	}
	
	// Test CanAfford
	if !cm.CanAfford(CostTypeLLM, 7.0) {
		t.Error("Should be able to afford 7.0 with 2.5 spent and 10.0 budget")
	}
	
	if cm.CanAfford(CostTypeLLM, 8.0) {
		t.Error("Should not be able to afford 8.0 with 2.5 spent and 10.0 budget")
	}
}

func TestCostManager_LLMCostCalculation(t *testing.T) {
	cm := NewCostManager()
	
	tests := []struct {
		model            string
		promptTokens     int
		completionTokens int
		expectedCost     float64
		description      string
	}{
		{
			model:            "claude-3-opus",
			promptTokens:     1000,
			completionTokens: 500,
			expectedCost:     (1000 * 0.01500 / 1000.0) + (500 * 0.04500 / 1000.0),
			description:      "Claude 3 Opus cost calculation",
		},
		{
			model:            "gpt-4",
			promptTokens:     1000,
			completionTokens: 500,
			expectedCost:     (1000 * 0.03000 / 1000.0) + (500 * 0.06000 / 1000.0),
			description:      "GPT-4 cost calculation",
		},
		{
			model:            "gpt-3.5-turbo",
			promptTokens:     1000,
			completionTokens: 500,
			expectedCost:     (1000 * 0.00050 / 1000.0) + (500 * 0.00150 / 1000.0),
			description:      "GPT-3.5 Turbo cost calculation",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			cost := cm.RecordLLMCost(tt.model, tt.promptTokens, tt.completionTokens, nil)
			
			if cost != tt.expectedCost {
				t.Errorf("Expected cost %f, got %f", tt.expectedCost, cost)
			}
			
			// Test estimation
			estimatedCost := cm.EstimateLLMCost(tt.model, tt.promptTokens, tt.completionTokens)
			if estimatedCost != tt.expectedCost {
				t.Errorf("Expected estimated cost %f, got %f", tt.expectedCost, estimatedCost)
			}
		})
	}
}

func TestCostManager_ToolCost(t *testing.T) {
	cm := NewCostManager()
	
	// Test default tool cost
	cost := cm.RecordToolCost("unknown-tool", nil)
	if cost != 0.01 {
		t.Errorf("Expected default tool cost of 0.01, got %f", cost)
	}
	
	// Test custom tool cost
	cm.toolCosts["expensive-tool"] = 0.10
	cost = cm.RecordToolCost("expensive-tool", nil)
	if cost != 0.10 {
		t.Errorf("Expected expensive tool cost of 0.10, got %f", cost)
	}
	
	// Verify total tool cost
	totalToolCost := cm.GetTotalCost(CostTypeTool)
	if totalToolCost != 0.11 {
		t.Errorf("Expected total tool cost of 0.11, got %f", totalToolCost)
	}
}

func TestCostManager_BudgetEnforcement(t *testing.T) {
	cm := NewCostManager()
	
	// Set a small budget
	cm.SetBudget(CostTypeLLM, 1.0)
	
	// Record some costs
	cm.RecordCost(CostTypeLLM, 0.6, "First call", nil)
	
	// Should be able to afford 0.3
	if !cm.CanAfford(CostTypeLLM, 0.3) {
		t.Error("Should be able to afford 0.3")
	}
	
	// Should not be able to afford 0.5
	if cm.CanAfford(CostTypeLLM, 0.5) {
		t.Error("Should not be able to afford 0.5")
	}
	
	// Test with no budget (unlimited)
	if !cm.CanAfford(CostTypeStorage, 1000.0) {
		t.Error("Should be able to afford any amount with no budget set")
	}
}

func TestCostManager_Report(t *testing.T) {
	cm := NewCostManager()
	
	// Set budgets
	cm.SetBudget(CostTypeLLM, 10.0)
	cm.SetBudget(CostTypeTool, 5.0)
	
	// Record various costs
	cm.RecordLLMCost("gpt-4", 1000, 500, nil)
	cm.RecordToolCost("shell", nil)
	cm.RecordCost(CostTypeStorage, 0.05, "Storage usage", nil)
	
	// Get report
	report := cm.GetCostReport()
	
	// Verify report structure
	if report["grand_total"] == nil {
		t.Error("Report should include grand_total")
	}
	
	totalCosts, ok := report["total_costs"].(map[string]float64)
	if !ok {
		t.Error("Report should include total_costs as map[string]float64")
	}
	
	// Verify individual costs
	if totalCosts[string(CostTypeLLM)] == 0 {
		t.Error("LLM costs should be non-zero")
	}
	
	if totalCosts[string(CostTypeTool)] != 0.01 {
		t.Errorf("Tool costs should be 0.01, got %f", totalCosts[string(CostTypeTool)])
	}
	
	// Verify budgets in report
	budgets, ok := report["budgets"].(map[string]float64)
	if !ok {
		t.Error("Report should include budgets as map[string]float64")
	}
	
	if budgets[string(CostTypeLLM)] != 10.0 {
		t.Errorf("LLM budget should be 10.0, got %f", budgets[string(CostTypeLLM)])
	}
}

func TestCostManager_ConcurrentAccess(t *testing.T) {
	cm := NewCostManager()
	cm.SetBudget(CostTypeLLM, 100.0)
	
	// Test concurrent access
	done := make(chan bool, 10)
	
	for i := 0; i < 10; i++ {
		go func(id int) {
			for j := 0; j < 100; j++ {
				cm.RecordCost(CostTypeLLM, 0.01, "Concurrent test", map[string]string{
					"goroutine": string(rune(id)),
				})
			}
			done <- true
		}(i)
	}
	
	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
	
	// Verify total
	total := cm.GetTotalCost(CostTypeLLM)
	expected := 10.0 // 10 goroutines * 100 calls * 0.01
	
	// Use a small delta for floating point comparison
	delta := 0.0001
	if diff := expected - total; diff < -delta || diff > delta {
		t.Errorf("Expected total of %f, got %f", expected, total)
	}
}

func TestCostManager_RecordTimestamps(t *testing.T) {
	cm := NewCostManager()
	
	beforeRecord := time.Now()
	cm.RecordCost(CostTypeLLM, 1.0, "Test", nil)
	afterRecord := time.Now()
	
	// Get the record
	if len(cm.costs[CostTypeLLM]) != 1 {
		t.Fatal("Should have exactly one cost record")
	}
	
	record := cm.costs[CostTypeLLM][0]
	
	// Verify timestamp is within expected range
	if record.Timestamp.Before(beforeRecord) || record.Timestamp.After(afterRecord) {
		t.Error("Timestamp should be between test boundaries")
	}
}

// Benchmark tests
func BenchmarkCostManager_RecordCost(b *testing.B) {
	cm := NewCostManager()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cm.RecordCost(CostTypeLLM, 0.01, "Benchmark", nil)
	}
}

func BenchmarkCostManager_CanAfford(b *testing.B) {
	cm := NewCostManager()
	cm.SetBudget(CostTypeLLM, 100.0)
	cm.RecordCost(CostTypeLLM, 50.0, "Initial", nil)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cm.CanAfford(CostTypeLLM, 10.0)
	}
}

func BenchmarkCostManager_GetReport(b *testing.B) {
	cm := NewCostManager()
	
	// Add some data
	for i := 0; i < 100; i++ {
		cm.RecordCost(CostTypeLLM, 0.01, "Test", nil)
		cm.RecordCost(CostTypeTool, 0.001, "Test", nil)
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cm.GetCostReport()
	}
}