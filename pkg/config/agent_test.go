// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package config

import (
	"context"
	"testing"
)

func TestCraftEnhancedAgentConfig_Validate(t *testing.T) {
	ctx := context.Background()

	t.Run("valid configuration", func(t *testing.T) {
		config := &EnhancedAgentConfig{
			ID:            "test-agent",
			Name:          "Test Agent",
			Type:          "worker",
			Role:          "tester",
			Backstory:     "A helpful testing agent",
			Model:         "gpt-3.5-turbo",
			ContextWindow: 4000,
			Temperature:   0.3,
			CostMagnitude: 2,
			Capabilities:  []string{"testing", "validation"},
			Tools: ToolAccessConfig{
				AllowAll: false,
				Allowed:  []string{"file_tool", "grep_tool"},
				Blocked:  []string{},
			},
		}

		err := config.Validate(ctx)
		if err != nil {
			t.Fatalf("Expected valid config to pass validation, got error: %v", err)
		}
	})

	t.Run("missing required fields", func(t *testing.T) {
		tests := []struct {
			name   string
			config EnhancedAgentConfig
		}{
			{
				name: "missing ID",
				config: EnhancedAgentConfig{
					Name:         "Test Agent",
					Type:         "worker",
					Model:        "gpt-3.5-turbo",
					Capabilities: []string{"testing"},
					Tools:        ToolAccessConfig{},
				},
			},
			{
				name: "missing name",
				config: EnhancedAgentConfig{
					ID:           "test-agent",
					Type:         "worker",
					Model:        "gpt-3.5-turbo",
					Capabilities: []string{"testing"},
					Tools:        ToolAccessConfig{},
				},
			},
			{
				name: "missing type",
				config: EnhancedAgentConfig{
					ID:           "test-agent",
					Name:         "Test Agent",
					Model:        "gpt-3.5-turbo",
					Capabilities: []string{"testing"},
					Tools:        ToolAccessConfig{},
				},
			},
			{
				name: "missing model",
				config: EnhancedAgentConfig{
					ID:           "test-agent",
					Name:         "Test Agent",
					Type:         "worker",
					Capabilities: []string{"testing"},
					Tools:        ToolAccessConfig{},
				},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				err := tt.config.Validate(ctx)
				if err == nil {
					t.Fatalf("Expected validation to fail for %s", tt.name)
				}
			})
		}
	})

	t.Run("invalid agent type", func(t *testing.T) {
		config := &EnhancedAgentConfig{
			ID:           "test-agent",
			Name:         "Test Agent",
			Type:         "invalid-type",
			Model:        "gpt-3.5-turbo",
			Capabilities: []string{"testing"},
			Tools:        ToolAccessConfig{},
		}

		err := config.Validate(ctx)
		if err == nil {
			t.Fatal("Expected validation to fail for invalid agent type")
		}
	})

	t.Run("invalid cost magnitude", func(t *testing.T) {
		config := &EnhancedAgentConfig{
			ID:            "test-agent",
			Name:          "Test Agent",
			Type:          "worker",
			Model:         "gpt-3.5-turbo",
			CostMagnitude: 4, // Invalid: not in Fibonacci sequence
			Capabilities:  []string{"testing"},
			Tools:         ToolAccessConfig{},
		}

		err := config.Validate(ctx)
		if err == nil {
			t.Fatal("Expected validation to fail for invalid cost magnitude")
		}
	})

	t.Run("invalid temperature", func(t *testing.T) {
		config := &EnhancedAgentConfig{
			ID:           "test-agent",
			Name:         "Test Agent",
			Type:         "worker",
			Model:        "gpt-3.5-turbo",
			Temperature:  2.5, // Invalid: too high
			Capabilities: []string{"testing"},
			Tools:        ToolAccessConfig{},
		}

		err := config.Validate(ctx)
		if err == nil {
			t.Fatal("Expected validation to fail for invalid temperature")
		}
	})
}

func TestCraftToolAccessConfig_Validate(t *testing.T) {
	ctx := context.Background()

	t.Run("valid allow_all configuration", func(t *testing.T) {
		config := ToolAccessConfig{
			AllowAll: true,
			Blocked:  []string{"dangerous_tool"},
		}

		err := config.Validate(ctx)
		if err != nil {
			t.Fatalf("Expected valid config to pass validation, got error: %v", err)
		}
	})

	t.Run("valid allowed list configuration", func(t *testing.T) {
		config := ToolAccessConfig{
			AllowAll: false,
			Allowed:  []string{"file_tool", "grep_tool"},
			Blocked:  []string{"dangerous_tool"},
		}

		err := config.Validate(ctx)
		if err != nil {
			t.Fatalf("Expected valid config to pass validation, got error: %v", err)
		}
	})

	t.Run("invalid allow_all with allowed list", func(t *testing.T) {
		config := ToolAccessConfig{
			AllowAll: true,
			Allowed:  []string{"file_tool"}, // Conflicts with allow_all
		}

		err := config.Validate(ctx)
		if err == nil {
			t.Fatal("Expected validation to fail for allow_all with allowed list")
		}
	})

	t.Run("conflicting allowed and blocked tools", func(t *testing.T) {
		config := ToolAccessConfig{
			AllowAll: false,
			Allowed:  []string{"file_tool", "grep_tool"},
			Blocked:  []string{"file_tool"}, // Conflicts with allowed
		}

		err := config.Validate(ctx)
		if err == nil {
			t.Fatal("Expected validation to fail for conflicting allowed and blocked tools")
		}
	})
}

func TestCraftEnhancedAgentConfig_EffectiveMethods(t *testing.T) {
	t.Run("GetEffectiveProvider", func(t *testing.T) {
		tests := []struct {
			name     string
			config   EnhancedAgentConfig
			expected string
		}{
			{
				name: "explicit provider",
				config: EnhancedAgentConfig{
					Provider: "anthropic",
					Model:    "claude-3-sonnet",
				},
				expected: "anthropic",
			},
			{
				name: "auto-detect from model - GPT",
				config: EnhancedAgentConfig{
					Model: "gpt-4-turbo",
				},
				expected: "openai",
			},
			{
				name: "auto-detect from model - Claude",
				config: EnhancedAgentConfig{
					Model: "claude-3-sonnet",
				},
				expected: "anthropic",
			},
			{
				name: "auto-detect from model - DeepSeek",
				config: EnhancedAgentConfig{
					Model: "deepseek-chat",
				},
				expected: "deepseek",
			},
			{
				name: "auto-detect from model - Ollama",
				config: EnhancedAgentConfig{
					Model: "llama3:8b",
				},
				expected: "ollama",
			},
			{
				name: "default fallback",
				config: EnhancedAgentConfig{
					Model: "unknown-model",
				},
				expected: "openai",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result := tt.config.GetEffectiveProvider()
				if result != tt.expected {
					t.Errorf("Expected provider %s, got %s", tt.expected, result)
				}
			})
		}
	})

	t.Run("GetEffectiveContextWindow", func(t *testing.T) {
		tests := []struct {
			name     string
			config   EnhancedAgentConfig
			expected int
		}{
			{
				name: "explicit context window",
				config: EnhancedAgentConfig{
					ContextWindow: 32000,
				},
				expected: 32000,
			},
			{
				name: "auto-detect GPT-4 Turbo",
				config: EnhancedAgentConfig{
					Model: "gpt-4-turbo",
				},
				expected: 128000,
			},
			{
				name: "auto-detect Claude-3",
				config: EnhancedAgentConfig{
					Model: "claude-3-sonnet",
				},
				expected: 200000,
			},
			{
				name: "default fallback",
				config: EnhancedAgentConfig{
					Model: "unknown-model",
				},
				expected: 8000,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result := tt.config.GetEffectiveContextWindow()
				if result != tt.expected {
					t.Errorf("Expected context window %d, got %d", tt.expected, result)
				}
			})
		}
	})

	t.Run("GetEffectiveCostMagnitude", func(t *testing.T) {
		tests := []struct {
			name     string
			config   EnhancedAgentConfig
			expected int
		}{
			{
				name: "explicit cost magnitude",
				config: EnhancedAgentConfig{
					CostMagnitude: 5,
				},
				expected: 5,
			},
			{
				name: "auto-detect GPT-4",
				config: EnhancedAgentConfig{
					Model: "gpt-4",
				},
				expected: 5,
			},
			{
				name: "auto-detect Claude-3 Opus",
				config: EnhancedAgentConfig{
					Model: "claude-3-opus",
				},
				expected: 8,
			},
			{
				name: "auto-detect Claude-3 Haiku",
				config: EnhancedAgentConfig{
					Model: "claude-3-haiku",
				},
				expected: 1,
			},
			{
				name: "auto-detect local model",
				config: EnhancedAgentConfig{
					Model: "ollama/llama2",
				},
				expected: 0,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result := tt.config.GetEffectiveCostMagnitude()
				if result != tt.expected {
					t.Errorf("Expected cost magnitude %d, got %d", tt.expected, result)
				}
			})
		}
	})

	t.Run("GetEffectiveTemperature", func(t *testing.T) {
		tests := []struct {
			name     string
			config   EnhancedAgentConfig
			expected float64
		}{
			{
				name: "explicit temperature",
				config: EnhancedAgentConfig{
					Temperature: 0.8,
				},
				expected: 0.8,
			},
			{
				name: "manager default",
				config: EnhancedAgentConfig{
					Type: "manager",
				},
				expected: 0.3,
			},
			{
				name: "worker default",
				config: EnhancedAgentConfig{
					Type: "worker",
				},
				expected: 0.1,
			},
			{
				name: "specialist default",
				config: EnhancedAgentConfig{
					Type: "specialist",
				},
				expected: 0.2,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result := tt.config.GetEffectiveTemperature()
				if result != tt.expected {
					t.Errorf("Expected temperature %.1f, got %.1f", tt.expected, result)
				}
			})
		}
	})
}

func TestCraftEnhancedAgentConfig_HasCapability(t *testing.T) {
	config := &EnhancedAgentConfig{
		Capabilities: []string{"testing", "validation", "analysis"},
	}

	tests := []struct {
		capability string
		expected   bool
	}{
		{"testing", true},
		{"validation", true},
		{"analysis", true},
		{"coding", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.capability, func(t *testing.T) {
			result := config.HasCapability(tt.capability)
			if result != tt.expected {
				t.Errorf("Expected HasCapability(%s) = %v, got %v", tt.capability, tt.expected, result)
			}
		})
	}
}

func TestCraftEnhancedAgentConfig_IsToolOnlyAgent(t *testing.T) {
	tests := []struct {
		name          string
		costMagnitude int
		expected      bool
	}{
		{"tool-only agent", 0, true},
		{"cheap API agent", 1, false},
		{"expensive agent", 8, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &EnhancedAgentConfig{
				CostMagnitude: tt.costMagnitude,
			}

			result := config.IsToolOnlyAgent()
			if result != tt.expected {
				t.Errorf("Expected IsToolOnlyAgent() = %v, got %v", tt.expected, result)
			}
		})
	}
}

// Benchmark tests for performance
func BenchmarkCraftEnhancedAgentConfig_Validate(b *testing.B) {
	ctx := context.Background()
	config := &EnhancedAgentConfig{
		ID:            "test-agent",
		Name:          "Test Agent",
		Type:          "worker",
		Model:         "gpt-3.5-turbo",
		ContextWindow: 4000,
		Temperature:   0.3,
		CostMagnitude: 2,
		Capabilities:  []string{"testing", "validation"},
		Tools: ToolAccessConfig{
			AllowAll: false,
			Allowed:  []string{"file_tool", "grep_tool"},
			Blocked:  []string{},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = config.Validate(ctx)
	}
}

func BenchmarkCraftEnhancedAgentConfig_GetEffectiveProvider(b *testing.B) {
	config := &EnhancedAgentConfig{
		Model: "claude-3-sonnet",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = config.GetEffectiveProvider()
	}
}