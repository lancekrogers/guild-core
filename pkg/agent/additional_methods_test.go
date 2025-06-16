// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package agent

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// Test SetDescription and GetDescription
func TestWorkerAgent_Description(t *testing.T) {
	agent := &WorkerAgent{
		ID:   "test-agent",
		Name: "Test Agent",
	}

	// Test initial empty description
	assert.Empty(t, agent.GetDescription())

	// Test setting description
	description := "This is a test agent for demonstration purposes"
	agent.SetDescription(description)
	assert.Equal(t, description, agent.GetDescription())

	// Test setting empty description
	agent.SetDescription("")
	assert.Empty(t, agent.GetDescription())

	// Test setting very long description
	longDesc := "This is a very long description that tests the limits of what might be stored in an agent's description field. It contains multiple sentences and should handle edge cases properly."
	agent.SetDescription(longDesc)
	assert.Equal(t, longDesc, agent.GetDescription())
}

// Test HasCapability
func TestWorkerAgent_HasCapability(t *testing.T) {
	agent := &WorkerAgent{
		ID:           "test-agent",
		Name:         "Test Agent",
		capabilities: []string{"file-operations", "web-requests", "data-analysis"},
	}

	// Test existing capabilities
	assert.True(t, agent.HasCapability("file-operations"))
	assert.True(t, agent.HasCapability("web-requests"))
	assert.True(t, agent.HasCapability("data-analysis"))

	// Test non-existing capability
	assert.False(t, agent.HasCapability("non-existent"))
	assert.False(t, agent.HasCapability(""))

	// Test with empty capabilities
	emptyAgent := &WorkerAgent{
		ID:           "empty-agent",
		Name:         "Empty Agent",
		capabilities: []string{},
	}
	assert.False(t, emptyAgent.HasCapability("any-capability"))

	// Test with nil capabilities
	nilAgent := &WorkerAgent{
		ID:           "nil-agent",
		Name:         "Nil Agent",
		capabilities: nil,
	}
	assert.False(t, nilAgent.HasCapability("any-capability"))
}

// Test HasCapability edge cases
func TestWorkerAgent_HasCapability_EdgeCases(t *testing.T) {
	tests := []struct {
		name         string
		capabilities []string
		searchFor    string
		expected     bool
	}{
		{
			name:         "exact match",
			capabilities: []string{"exact-match"},
			searchFor:    "exact-match",
			expected:     true,
		},
		{
			name:         "case sensitive - different case",
			capabilities: []string{"Case-Sensitive"},
			searchFor:    "case-sensitive",
			expected:     false,
		},
		{
			name:         "partial match should not work",
			capabilities: []string{"file-operations"},
			searchFor:    "file",
			expected:     false,
		},
		{
			name:         "substring match should not work",
			capabilities: []string{"web-requests"},
			searchFor:    "web",
			expected:     false,
		},
		{
			name:         "empty string in capabilities",
			capabilities: []string{"valid", "", "also-valid"},
			searchFor:    "",
			expected:     true,
		},
		{
			name:         "duplicate capabilities",
			capabilities: []string{"dup", "unique", "dup"},
			searchFor:    "dup",
			expected:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agent := &WorkerAgent{
				ID:           "test-agent",
				Name:         "Test Agent",
				capabilities: tt.capabilities,
			}

			result := agent.HasCapability(tt.searchFor)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Test SetCostBudget and GetCostReport with real cost manager
func TestWorkerAgent_CostManagement(t *testing.T) {
	// Create a real cost manager
	costMgr := newCostManager()

	agent := &WorkerAgent{
		ID:          "test-agent",
		Name:        "Test Agent",
		CostManager: costMgr,
	}

	// Test setting cost budget
	agent.SetCostBudget(CostTypeLLM, 1000.0)
	assert.Equal(t, 1000.0, costMgr.GetBudget(CostTypeLLM))

	agent.SetCostBudget(CostTypeEmbedding, 300.0)
	assert.Equal(t, 300.0, costMgr.GetBudget(CostTypeEmbedding))

	// Test getting cost report (should not be nil)
	report := agent.GetCostReport()
	assert.NotNil(t, report)
	assert.IsType(t, map[string]any{}, report)
}

// Test cost management edge cases
func TestWorkerAgent_CostManagement_EdgeCases(t *testing.T) {
	costMgr := newCostManager()
	agent := &WorkerAgent{
		ID:          "test-agent",
		Name:        "Test Agent",
		CostManager: costMgr,
	}

	// Test setting zero budget
	agent.SetCostBudget(CostTypeLLM, 0.0)
	assert.Equal(t, 0.0, costMgr.GetBudget(CostTypeLLM))

	// Test setting negative budget (should still work)
	agent.SetCostBudget(CostTypeEmbedding, -100.0)
	assert.Equal(t, -100.0, costMgr.GetBudget(CostTypeEmbedding))

	// Test multiple cost types
	agent.SetCostBudget(CostTypeTool, 500.0)
	agent.SetCostBudget(CostTypeStorage, 200.0)
	agent.SetCostBudget(CostTypeCompute, 300.0)

	assert.Equal(t, 500.0, costMgr.GetBudget(CostTypeTool))
	assert.Equal(t, 200.0, costMgr.GetBudget(CostTypeStorage))
	assert.Equal(t, 300.0, costMgr.GetBudget(CostTypeCompute))
}

// Test comprehensive agent functionality
func TestWorkerAgent_ComprehensiveFunctionality(t *testing.T) {
	agent := &WorkerAgent{
		ID:           "comprehensive-agent",
		Name:         "Comprehensive Test Agent",
		description:  "Initial description",
		capabilities: []string{"test-capability"},
		CostManager:  newCostManager(),
	}

	// Test all getter methods work together
	assert.Equal(t, "comprehensive-agent", agent.GetID())
	assert.Equal(t, "Comprehensive Test Agent", agent.GetName())
	assert.Equal(t, "Initial description", agent.GetDescription())
	assert.True(t, agent.HasCapability("test-capability"))
	assert.False(t, agent.HasCapability("non-existent"))

	// Test setter methods work
	agent.SetDescription("Updated description")
	assert.Equal(t, "Updated description", agent.GetDescription())

	newCaps := []string{"new-cap1", "new-cap2"}
	agent.SetCapabilities(newCaps)
	assert.True(t, agent.HasCapability("new-cap1"))
	assert.True(t, agent.HasCapability("new-cap2"))
	assert.False(t, agent.HasCapability("test-capability")) // Should be replaced

	// Test cost management
	agent.SetCostBudget(CostTypeLLM, 500.0)
	report := agent.GetCostReport()
	assert.NotNil(t, report)
}
