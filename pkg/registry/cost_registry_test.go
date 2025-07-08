// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package registry

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lancekrogers/guild/tools"
)

func TestCostBasedAgentSelection(t *testing.T) {
	registry := NewAgentRegistry().(*DefaultAgentRegistry)

	// Register test agents with different cost magnitudes
	testAgents := []GuildAgentConfig{
		{
			ID:            "tools-only",
			Name:          "Tools Only Agent",
			Type:          "worker",
			Provider:      "local",
			Model:         "",
			Capabilities:  []string{"file_operations", "git"},
			CostMagnitude: 0, // Free
		},
		{
			ID:            "cheap-claude",
			Name:          "Cheap Claude",
			Type:          "worker",
			Provider:      "anthropic",
			Model:         "claude-3-haiku",
			Capabilities:  []string{"coding", "documentation"},
			CostMagnitude: 1, // Cheap
		},
		{
			ID:            "mid-claude",
			Name:          "Mid Claude",
			Type:          "specialist",
			Provider:      "anthropic",
			Model:         "claude-3-sonnet",
			Capabilities:  []string{"architecture", "review"},
			CostMagnitude: 3, // Mid cost
		},
		{
			ID:            "expensive-claude",
			Name:          "Expensive Claude",
			Type:          "manager",
			Provider:      "anthropic",
			Model:         "claude-3-opus",
			Capabilities:  []string{"planning", "management"},
			CostMagnitude: 8, // Most expensive
		},
	}

	// Register all agents
	for _, agent := range testAgents {
		err := registry.RegisterGuildAgent(agent)
		require.NoError(t, err)
	}

	t.Run("GetAgentsByCost", func(t *testing.T) {
		// Test getting agents within cost budget
		cheapAgents := registry.GetAgentsByCost(1)
		assert.Len(t, cheapAgents, 2)                    // tools-only and cheap-claude
		assert.Equal(t, "tools-only", cheapAgents[0].ID) // Should be sorted by cost
		assert.Equal(t, "cheap-claude", cheapAgents[1].ID)

		midRangeAgents := registry.GetAgentsByCost(3)
		assert.Len(t, midRangeAgents, 3) // All except expensive-claude

		allAgents := registry.GetAgentsByCost(8)
		assert.Len(t, allAgents, 4) // All agents
	})

	t.Run("GetCheapestAgentByCapability", func(t *testing.T) {
		// Test finding cheapest agent for specific capabilities
		cheapestCoding, err := registry.GetCheapestAgentByCapability("coding")
		require.NoError(t, err)
		assert.Equal(t, "cheap-claude", cheapestCoding.ID)
		assert.Equal(t, 1, cheapestCoding.CostMagnitude)

		cheapestFileOps, err := registry.GetCheapestAgentByCapability("file_operations")
		require.NoError(t, err)
		assert.Equal(t, "tools-only", cheapestFileOps.ID)
		assert.Equal(t, 0, cheapestFileOps.CostMagnitude)

		// Test capability not found
		_, err = registry.GetCheapestAgentByCapability("nonexistent")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no agent found")
	})

	t.Run("GetAgentsByCapability", func(t *testing.T) {
		// Test getting all agents with capability, sorted by cost
		codingAgents := registry.GetAgentsByCapability("coding")
		assert.Len(t, codingAgents, 1)
		assert.Equal(t, "cheap-claude", codingAgents[0].ID)

		// Test multiple agents with same capability (if any)
		architectureAgents := registry.GetAgentsByCapability("architecture")
		assert.Len(t, architectureAgents, 1)
		assert.Equal(t, "mid-claude", architectureAgents[0].ID)
	})

	t.Run("EffectiveCostMagnitude", func(t *testing.T) {
		// Test auto-detection of cost magnitude
		autoDetectAgent := GuildAgentConfig{
			ID:            "auto-detect",
			Name:          "Auto Detect Agent",
			Type:          "worker",
			Provider:      "anthropic",
			Model:         "claude-3-haiku-20240307",
			Capabilities:  []string{"testing"},
			CostMagnitude: 0, // Will be auto-detected
		}

		err := registry.RegisterGuildAgent(autoDetectAgent)
		require.NoError(t, err)

		agents := registry.GetRegisteredAgents()
		var autoAgent *GuildAgentConfig
		for _, agent := range agents {
			if agent.ID == "auto-detect" {
				autoAgent = &agent
				break
			}
		}
		require.NotNil(t, autoAgent)

		// Should auto-detect claude-3-haiku as cost magnitude 1
		// Note: getEffectiveCostMagnitude method doesn't exist in the current implementation
		// The auto-detection happens during registration, so we verify the registered value
		assert.Equal(t, 1, autoAgent.CostMagnitude)
	})
}

func TestCostBasedToolSelection(t *testing.T) {
	registry := NewToolRegistry().(*DefaultToolRegistry)

	// Test RegisterToolWithCost with mock tools
	mockTool1 := &MockTool{name: "shell", category: "execution"}
	mockTool2 := &MockTool{name: "http_client", category: "network"}
	mockTool3 := &MockTool{name: "expensive_ai", category: "ai"}

	t.Run("RegisterToolWithCost", func(t *testing.T) {
		// Register tools with different cost magnitudes
		err := registry.RegisterToolWithCost("shell", mockTool1, 0, []string{"execution", "file_operations"})
		require.NoError(t, err)

		err = registry.RegisterToolWithCost("http_client", mockTool2, 1, []string{"network", "api"})
		require.NoError(t, err)

		err = registry.RegisterToolWithCost("expensive_ai", mockTool3, 5, []string{"ai", "analysis"})
		require.NoError(t, err)

		// Test invalid cost magnitude
		err = registry.RegisterToolWithCost("invalid_tool", mockTool1, 4, []string{"invalid"})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid cost_magnitude")
	})

	t.Run("GetToolsByCost", func(t *testing.T) {
		// Get tools within budget
		cheapTools := registry.GetToolsByCost(0)
		assert.Len(t, cheapTools, 1)
		assert.Equal(t, "shell", cheapTools[0].Name)

		midRangeTools := registry.GetToolsByCost(1)
		assert.Len(t, midRangeTools, 2) // shell and http_client

		allTools := registry.GetToolsByCost(5)
		assert.Len(t, allTools, 3) // All tools
	})

	t.Run("GetCheapestToolByCapability", func(t *testing.T) {
		// Find cheapest tool for specific capability
		cheapestExecution, err := registry.GetCheapestToolByCapability("execution")
		require.NoError(t, err)
		assert.Equal(t, "shell", cheapestExecution.Name)
		assert.Equal(t, 0, cheapestExecution.CostMagnitude)

		cheapestNetwork, err := registry.GetCheapestToolByCapability("network")
		require.NoError(t, err)
		assert.Equal(t, "http_client", cheapestNetwork.Name)
		assert.Equal(t, 1, cheapestNetwork.CostMagnitude)

		// Test capability not found
		_, err = registry.GetCheapestToolByCapability("nonexistent")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no tool found")
	})

	t.Run("ToolAvailability", func(t *testing.T) {
		// Test setting tool availability
		err := registry.SetToolAvailability("shell", false)
		require.NoError(t, err)

		// Should not find unavailable tools
		cheapestExecution, err := registry.GetCheapestToolByCapability("execution")
		assert.Error(t, err) // No available tools with this capability

		// Re-enable tool
		err = registry.SetToolAvailability("shell", true)
		require.NoError(t, err)

		cheapestExecution, err = registry.GetCheapestToolByCapability("execution")
		require.NoError(t, err)
		assert.Equal(t, "shell", cheapestExecution.Name)
	})
}

func TestRegistryIntegration(t *testing.T) {
	registry := NewComponentRegistry().(*DefaultComponentRegistry)

	// Test that the main registry provides access to cost-based methods
	t.Run("ComponentRegistryMethods", func(t *testing.T) {
		// These should not panic and return empty results (no agents registered yet)
		agents := registry.GetAgentsByCost(5)
		assert.Empty(t, agents)

		tools := registry.GetToolsByCost(5)
		assert.Empty(t, tools)

		_, err := registry.GetCheapestAgentByCapability("coding")
		assert.Error(t, err)

		_, err = registry.GetCheapestToolByCapability("execution")
		assert.Error(t, err)
	})
}

// MockTool for testing
type MockTool struct {
	name     string
	category string
}

func (m *MockTool) Name() string { return m.name }

func (m *MockTool) Description() string { return "Mock tool" }

func (m *MockTool) Category() string { return m.category }

func (m *MockTool) Schema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"input": map[string]interface{}{
				"type":        "string",
				"description": "Test input",
			},
		},
	}
}

func (m *MockTool) Execute(ctx context.Context, input string) (*tools.ToolResult, error) {
	return &tools.ToolResult{
		Output:  "mock result",
		Success: true,
	}, nil
}

func (m *MockTool) Examples() []string { return []string{"example input"} }

func (m *MockTool) RequiresAuth() bool { return false }

func (m *MockTool) HealthCheck() error { return nil }
