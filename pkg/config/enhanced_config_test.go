// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEnhancedAgentConfiguration(t *testing.T) {
	t.Run("ValidCostMagnitudes", func(t *testing.T) {
		validCosts := []int{0, 1, 2, 3, 5, 8}

		for _, cost := range validCosts {
			agent := &AgentConfig{
				ID:            "test-agent",
				Name:          "Test Agent",
				Type:          "worker",
				Provider:      "anthropic",
				Model:         "claude-3-haiku",
				Capabilities:  []string{"testing"},
				CostMagnitude: cost,
			}

			err := agent.Validate()
			assert.NoError(t, err, "Cost magnitude %d should be valid", cost)
		}
	})

	t.Run("InvalidCostMagnitudes", func(t *testing.T) {
		invalidCosts := []int{4, 6, 7, 9, 10}

		for _, cost := range invalidCosts {
			agent := &AgentConfig{
				ID:            "test-agent",
				Name:          "Test Agent",
				Type:          "worker",
				Provider:      "anthropic",
				Model:         "claude-3-haiku",
				Capabilities:  []string{"testing"},
				CostMagnitude: cost,
			}

			err := agent.Validate()
			assert.Error(t, err, "Cost magnitude %d should be invalid", cost)
			assert.Contains(t, err.Error(), "invalid cost_magnitude")
		}
	})

	t.Run("ContextResetValidation", func(t *testing.T) {
		tests := []struct {
			reset     string
			shouldErr bool
		}{
			{"", false},          // Empty is valid (uses default)
			{"truncate", false},  // Valid option
			{"summarize", false}, // Valid option
			{"invalid", true},    // Invalid option
			{"compress", true},   // Invalid option
		}

		for _, tt := range tests {
			agent := &AgentConfig{
				ID:           "test-agent",
				Name:         "Test Agent",
				Type:         "worker",
				Provider:     "anthropic",
				Model:        "claude-3-haiku",
				Capabilities: []string{"testing"},
				ContextReset: tt.reset,
			}

			err := agent.Validate()
			if tt.shouldErr {
				assert.Error(t, err, "Context reset '%s' should be invalid", tt.reset)
			} else {
				assert.NoError(t, err, "Context reset '%s' should be valid", tt.reset)
			}
		}
	})

	t.Run("ContextWindowValidation", func(t *testing.T) {
		tests := []struct {
			window    int
			shouldErr bool
		}{
			{0, false},      // Auto-detection
			{1000, false},   // Valid positive value
			{200000, false}, // Large valid value
			{-1, true},      // Invalid negative
			{-1000, true},   // Invalid large negative
		}

		for _, tt := range tests {
			agent := &AgentConfig{
				ID:            "test-agent",
				Name:          "Test Agent",
				Type:          "worker",
				Provider:      "anthropic",
				Model:         "claude-3-haiku",
				Capabilities:  []string{"testing"},
				ContextWindow: tt.window,
			}

			err := agent.Validate()
			if tt.shouldErr {
				assert.Error(t, err, "Context window %d should be invalid", tt.window)
			} else {
				assert.NoError(t, err, "Context window %d should be valid", tt.window)
			}
		}
	})
}

func TestEffectiveCostMagnitude(t *testing.T) {
	tests := []struct {
		name           string
		model          string
		configuredCost int
		expectedCost   int
	}{
		{"Explicit cost takes precedence", "gpt-4", 2, 2},
		{"GPT-4 auto-detection", "gpt-4-turbo", 0, 5},
		{"GPT-3.5 auto-detection", "gpt-3.5-turbo", 0, 2},
		{"Claude Opus auto-detection", "claude-3-opus-20240229", 0, 8},
		{"Claude Sonnet auto-detection", "claude-3-sonnet-20240229", 0, 3},
		{"Claude Haiku auto-detection", "claude-3-haiku-20240307", 0, 1},
		{"Ollama auto-detection", "ollama/llama2", 0, 0},
		{"Local model auto-detection", "local-model", 0, 0},
		{"Unknown model default", "unknown-model-v1", 0, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agent := &AgentConfig{
				Model:         tt.model,
				CostMagnitude: tt.configuredCost,
			}

			result := agent.GetEffectiveCostMagnitude()
			assert.Equal(t, tt.expectedCost, result)
		})
	}
}

func TestEffectiveContextWindow(t *testing.T) {
	tests := []struct {
		name             string
		model            string
		configuredWindow int
		expectedWindow   int
	}{
		{"Explicit window takes precedence", "gpt-4", 50000, 50000},
		{"GPT-4-turbo auto-detection", "gpt-4-turbo", 0, 128000},
		{"GPT-4 auto-detection", "gpt-4", 0, 32000},
		{"GPT-3.5 auto-detection", "gpt-3.5-turbo", 0, 16000},
		{"Claude-3 auto-detection", "claude-3-sonnet", 0, 200000},
		{"Claude-2 auto-detection", "claude-2", 0, 100000},
		{"Unknown model default", "unknown-model", 0, 8000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agent := &AgentConfig{
				Model:         tt.model,
				ContextWindow: tt.configuredWindow,
			}

			result := agent.GetEffectiveContextWindow()
			assert.Equal(t, tt.expectedWindow, result)
		})
	}
}

func TestEffectiveContextReset(t *testing.T) {
	tests := []struct {
		name            string
		agentType       string
		configuredReset string
		expectedReset   string
	}{
		{"Explicit reset takes precedence", "worker", "summarize", "summarize"},
		{"Manager default", "manager", "", "summarize"},
		{"Worker default", "worker", "", "truncate"},
		{"Specialist default", "specialist", "", "truncate"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agent := &AgentConfig{
				Type:         tt.agentType,
				ContextReset: tt.configuredReset,
			}

			result := agent.GetEffectiveContextReset()
			assert.Equal(t, tt.expectedReset, result)
		})
	}
}

func TestIsToolOnlyAgent(t *testing.T) {
	tests := []struct {
		name          string
		costMagnitude int
		model         string
		tools         []string
		expected      bool
	}{
		{"Cost magnitude 0", 0, "claude-3-haiku", []string{"shell"}, true},
		{"No model with tools", 1, "", []string{"shell", "git"}, false},
		{"Regular agent", 1, "claude-3-haiku", []string{"shell"}, false},
		{"No tools, has model", 0, "claude-3-haiku", []string{}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agent := &AgentConfig{
				CostMagnitude: tt.costMagnitude,
				Model:         tt.model,
				Tools:         tt.tools,
			}

			result := agent.IsToolOnlyAgent()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestEnhancedConfigurationIntegration(t *testing.T) {
	t.Run("CompleteEnhancedAgent", func(t *testing.T) {
		agent := &AgentConfig{
			ID:            "enhanced-agent",
			Name:          "Enhanced Test Agent",
			Type:          "worker",
			Provider:      "anthropic",
			Model:         "claude-3-sonnet-20240229",
			Description:   "Test agent with enhanced configuration",
			Capabilities:  []string{"backend", "testing"},
			Tools:         []string{"shell", "file_system"},
			CostMagnitude: 3,
			ContextWindow: 100000,
			ContextReset:  "summarize",
			Temperature:   0.2,
		}

		// Should validate successfully
		err := agent.Validate()
		require.NoError(t, err)

		// Test effective values
		assert.Equal(t, 3, agent.GetEffectiveCostMagnitude())
		assert.Equal(t, 100000, agent.GetEffectiveContextWindow())
		assert.Equal(t, "summarize", agent.GetEffectiveContextReset())
		assert.False(t, agent.IsToolOnlyAgent())
	})

	t.Run("ToolOnlyAgent", func(t *testing.T) {
		agent := &AgentConfig{
			ID:            "tools-only",
			Name:          "Tools Only Agent",
			Type:          "worker",
			Provider:      "local",
			Model:         "", // No model for tool-only
			Description:   "Agent that only uses tools",
			Capabilities:  []string{"file_operations", "git"},
			Tools:         []string{"shell", "git", "file_system"},
			CostMagnitude: 0, // Zero cost
		}

		// Should validate successfully
		err := agent.Validate()
		require.NoError(t, err)

		// Test effective values
		assert.Equal(t, 0, agent.GetEffectiveCostMagnitude())
		assert.Equal(t, "truncate", agent.GetEffectiveContextReset()) // Worker default
		assert.True(t, agent.IsToolOnlyAgent())
	})
}
