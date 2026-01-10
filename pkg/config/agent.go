// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package config

import (
	"context"
	"strings"

	"github.com/lancekrogers/guild-core/pkg/gerror"
	"github.com/lancekrogers/guild-core/pkg/observability"
)

// EnhancedAgentConfig represents the enhanced configuration for a single agent
// This extends the base AgentConfig with additional fields for comprehensive agent management
type EnhancedAgentConfig struct {
	// Core identification
	ID        string `yaml:"id"`
	Name      string `yaml:"name"`
	Type      string `yaml:"type"`
	Role      string `yaml:"role"`
	Specialty string `yaml:"specialty,omitempty"`
	Backstory string `yaml:"backstory"`

	// Model and provider configuration
	Model         string  `yaml:"model"`
	Provider      string  `yaml:"provider,omitempty"` // Auto-detected from model if not specified
	ContextWindow int     `yaml:"context_window"`
	Temperature   float64 `yaml:"temperature"`
	CostMagnitude int     `yaml:"cost_magnitude"`

	// Capabilities and tools
	Capabilities []string         `yaml:"capabilities"`
	Tools        ToolAccessConfig `yaml:"tools"`
	Languages    []string         `yaml:"languages,omitempty"`
	Frameworks   []string         `yaml:"frameworks,omitempty"`

	// Advanced configuration
	Prompts  map[string]string      `yaml:"prompts,omitempty"`
	Metadata map[string]interface{} `yaml:"metadata,omitempty"`

	// Reasoning configuration
	Reasoning ReasoningConfig `yaml:"reasoning,omitempty"`
}

// ToolAccessConfig defines tool access control for an agent
type ToolAccessConfig struct {
	AllowAll bool     `yaml:"allow_all"`
	Allowed  []string `yaml:"allowed,omitempty"`
	Blocked  []string `yaml:"blocked,omitempty"`
}

// ReasoningConfig defines reasoning behavior for an agent
type ReasoningConfig struct {
	Enabled                    bool    `yaml:"enabled" default:"true"`
	ShowThinking               bool    `yaml:"show_thinking" default:"true"`
	MinConfidenceDisplay       float64 `yaml:"min_confidence_display" default:"0.3"`
	DeepReasoningMinComplexity float64 `yaml:"deep_reasoning_min_complexity" default:"0.5"`
	IncludeInPrompt            bool    `yaml:"include_in_prompt" default:"true"`
}

// Validate validates the enhanced agent configuration
func (c *EnhancedAgentConfig) Validate(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("EnhancedAgentConfig").
			WithOperation("Validate")
	}

	logger := observability.GetLogger(ctx)
	ctx = observability.WithComponent(ctx, "EnhancedAgentConfig")
	ctx = observability.WithOperation(ctx, "Validate")

	logger.DebugContext(ctx, "Validating enhanced agent configuration", "agent_id", c.ID)

	// Required fields
	if c.ID == "" {
		return gerror.New(gerror.ErrCodeValidation, "agent ID is required", nil).
			WithComponent("EnhancedAgentConfig").
			WithOperation("Validate")
	}
	if c.Name == "" {
		return gerror.New(gerror.ErrCodeValidation, "agent name is required", nil).
			WithComponent("EnhancedAgentConfig").
			WithOperation("Validate").
			WithDetails("agent_id", c.ID)
	}
	if c.Type == "" {
		return gerror.New(gerror.ErrCodeValidation, "agent type is required", nil).
			WithComponent("EnhancedAgentConfig").
			WithOperation("Validate").
			WithDetails("agent_id", c.ID)
	}
	if c.Model == "" {
		return gerror.New(gerror.ErrCodeValidation, "agent model is required", nil).
			WithComponent("EnhancedAgentConfig").
			WithOperation("Validate").
			WithDetails("agent_id", c.ID)
	}

	// Validate type
	validTypes := map[string]bool{
		"manager":    true,
		"worker":     true,
		"specialist": true,
	}
	if !validTypes[c.Type] {
		return gerror.Newf(gerror.ErrCodeValidation, "invalid agent type: %s (must be manager, worker, or specialist)", c.Type).
			WithComponent("EnhancedAgentConfig").
			WithOperation("Validate").
			WithDetails("agent_id", c.ID)
	}

	// Validate capabilities
	if len(c.Capabilities) == 0 {
		return gerror.New(gerror.ErrCodeValidation, "agent must have at least one capability", nil).
			WithComponent("EnhancedAgentConfig").
			WithOperation("Validate").
			WithDetails("agent_id", c.ID)
	}

	// Validate cost magnitude (Fibonacci scale)
	if c.CostMagnitude != 0 {
		validCosts := map[int]bool{
			1: true, // Cheap API usage
			2: true, // Low-mid cost
			3: true, // Mid cost
			5: true, // High cost
			8: true, // Most expensive models
		}
		if !validCosts[c.CostMagnitude] {
			return gerror.Newf(gerror.ErrCodeValidation, "invalid cost_magnitude: %d (must be 0 for bash tools, or Fibonacci values: 1,2,3,5,8)", c.CostMagnitude).
				WithComponent("EnhancedAgentConfig").
				WithOperation("Validate").
				WithDetails("agent_id", c.ID)
		}
	}

	// Validate context window
	if c.ContextWindow < 0 {
		return gerror.New(gerror.ErrCodeValidation, "context_window must be non-negative", nil).
			WithComponent("EnhancedAgentConfig").
			WithOperation("Validate").
			WithDetails("agent_id", c.ID)
	}

	// Validate temperature
	if c.Temperature < 0.0 || c.Temperature > 2.0 {
		return gerror.New(gerror.ErrCodeValidation, "temperature must be between 0.0 and 2.0", nil).
			WithComponent("EnhancedAgentConfig").
			WithOperation("Validate").
			WithDetails("agent_id", c.ID)
	}

	// Validate tool access configuration
	if err := c.Tools.Validate(ctx); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeValidation, "tool access configuration validation failed").
			WithComponent("EnhancedAgentConfig").
			WithOperation("Validate").
			WithDetails("agent_id", c.ID)
	}

	logger.DebugContext(ctx, "Enhanced agent configuration validation completed", "agent_id", c.ID)
	return nil
}

// Validate validates the tool access configuration
func (t *ToolAccessConfig) Validate(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("ToolAccessConfig").
			WithOperation("Validate")
	}

	// If allow_all is true, blocked list can exist but allowed list should be empty
	if t.AllowAll && len(t.Allowed) > 0 {
		return gerror.New(gerror.ErrCodeValidation, "cannot specify both allow_all=true and allowed list", nil).
			WithComponent("ToolAccessConfig").
			WithOperation("Validate")
	}

	// Check for conflicting tool specifications
	if len(t.Allowed) > 0 && len(t.Blocked) > 0 {
		allowed := make(map[string]bool)
		for _, tool := range t.Allowed {
			allowed[tool] = true
		}

		for _, tool := range t.Blocked {
			if allowed[tool] {
				return gerror.Newf(gerror.ErrCodeValidation, "tool '%s' cannot be both allowed and blocked", tool).
					WithComponent("ToolAccessConfig").
					WithOperation("Validate").
					WithDetails("tool", tool)
			}
		}
	}

	return nil
}

// GetEffectiveProvider returns the provider, auto-detecting from model if not explicitly set
func (c *EnhancedAgentConfig) GetEffectiveProvider() string {
	if c.Provider != "" {
		return c.Provider
	}

	// Auto-detect provider from model name
	modelLower := strings.ToLower(c.Model)
	switch {
	case strings.Contains(modelLower, "gpt-") || strings.Contains(modelLower, "openai"):
		return "openai"
	case strings.Contains(modelLower, "claude"):
		return "anthropic"
	case strings.Contains(modelLower, "deepseek"):
		return "deepseek"
	case strings.Contains(modelLower, "llama") || strings.Contains(modelLower, "ollama"):
		return "ollama"
	default:
		return "openai" // Default fallback
	}
}

// GetEffectiveContextWindow returns the context window with auto-detection based on model
func (c *EnhancedAgentConfig) GetEffectiveContextWindow() int {
	if c.ContextWindow > 0 {
		return c.ContextWindow
	}

	// Auto-detect based on known models
	modelLower := strings.ToLower(c.Model)
	switch {
	case strings.Contains(modelLower, "gpt-4-turbo"):
		return 128000
	case strings.Contains(modelLower, "gpt-4"):
		return 32000
	case strings.Contains(modelLower, "gpt-3.5"):
		return 16000
	case strings.Contains(modelLower, "claude-3"):
		return 200000
	case strings.Contains(modelLower, "claude-2"):
		return 100000
	case strings.Contains(modelLower, "deepseek"):
		return 32000
	default:
		return 8000 // Conservative default
	}
}

// SetReasoningDefaults sets default values for reasoning configuration
func (c *EnhancedAgentConfig) SetReasoningDefaults() {
	// Set defaults if reasoning config is empty
	if c.Reasoning.Enabled == false && c.Reasoning.ShowThinking == false &&
		c.Reasoning.MinConfidenceDisplay == 0 && c.Reasoning.DeepReasoningMinComplexity == 0 {
		c.Reasoning = ReasoningConfig{
			Enabled:                    true,
			ShowThinking:               true,
			MinConfidenceDisplay:       0.3,
			DeepReasoningMinComplexity: 0.5,
			IncludeInPrompt:            true,
		}
	}
}

// GetEffectiveCostMagnitude returns the cost magnitude with smart defaults based on model
func (c *EnhancedAgentConfig) GetEffectiveCostMagnitude() int {
	if c.CostMagnitude != 0 {
		return c.CostMagnitude
	}

	// Auto-assign based on model characteristics
	modelLower := strings.ToLower(c.Model)
	switch {
	case strings.Contains(modelLower, "gpt-4"):
		return 5 // High cost
	case strings.Contains(modelLower, "gpt-3.5"):
		return 2 // Low-mid cost
	case strings.Contains(modelLower, "claude-3-opus"):
		return 8 // Most expensive
	case strings.Contains(modelLower, "claude-3-sonnet"):
		return 3 // Mid cost
	case strings.Contains(modelLower, "claude-3-haiku"):
		return 1 // Cheap
	case strings.Contains(modelLower, "deepseek"):
		return 1 // Generally cheap
	case strings.Contains(modelLower, "ollama") || strings.Contains(modelLower, "local"):
		return 0 // Free local models
	default:
		return 1 // Default to cheap for unknown models
	}
}

// GetEffectiveTemperature returns the temperature with defaults based on agent type
func (c *EnhancedAgentConfig) GetEffectiveTemperature() float64 {
	if c.Temperature > 0.0 {
		return c.Temperature
	}

	// Default based on agent type
	switch c.Type {
	case "manager":
		return 0.3 // Managers need consistency for decision making
	case "worker":
		return 0.1 // Workers need deterministic output
	case "specialist":
		return 0.2 // Specialists need focused responses
	default:
		return 0.1 // Conservative default
	}
}

// IsToolOnlyAgent returns true if this agent only uses tools (no LLM calls)
func (c *EnhancedAgentConfig) IsToolOnlyAgent() bool {
	// Check if CostMagnitude is explicitly set to 0
	return c.CostMagnitude == 0
}

// HasCapability checks if an agent has a specific capability
func (c *EnhancedAgentConfig) HasCapability(capability string) bool {
	for _, cap := range c.Capabilities {
		if cap == capability {
			return true
		}
	}
	return false
}

// ToBaseAgentConfig converts the enhanced config to the base AgentConfig for compatibility
func (c *EnhancedAgentConfig) ToBaseAgentConfig() AgentConfig {
	tools := make([]string, 0)
	if c.Tools.AllowAll {
		tools = append(tools, "*") // Indicate all tools allowed
	} else {
		tools = append(tools, c.Tools.Allowed...)
	}

	return AgentConfig{
		ID:            c.ID,
		Name:          c.Name,
		Type:          c.Type,
		Provider:      c.GetEffectiveProvider(),
		Model:         c.Model,
		Description:   c.Backstory,
		Capabilities:  c.Capabilities,
		Tools:         tools,
		MaxTokens:     c.GetEffectiveContextWindow(),
		Temperature:   c.GetEffectiveTemperature(),
		CostMagnitude: c.GetEffectiveCostMagnitude(),
		ContextWindow: c.GetEffectiveContextWindow(),
		Settings:      make(map[string]string),
	}
}

// FromBaseAgentConfig creates an enhanced config from a base AgentConfig
func FromBaseAgentConfig(base AgentConfig) *EnhancedAgentConfig {
	tools := ToolAccessConfig{
		AllowAll: false,
		Allowed:  base.Tools,
		Blocked:  make([]string, 0),
	}

	// Check if all tools are allowed
	for _, tool := range base.Tools {
		if tool == "*" {
			tools.AllowAll = true
			tools.Allowed = make([]string, 0)
			break
		}
	}

	return &EnhancedAgentConfig{
		ID:            base.ID,
		Name:          base.Name,
		Type:          base.Type,
		Role:          base.Type, // Use type as default role
		Specialty:     "",
		Backstory:     base.Description,
		Model:         base.Model,
		Provider:      base.Provider,
		ContextWindow: base.MaxTokens,
		Temperature:   base.Temperature,
		CostMagnitude: base.CostMagnitude,
		Capabilities:  base.Capabilities,
		Tools:         tools,
		Languages:     make([]string, 0),
		Frameworks:    make([]string, 0),
		Prompts:       make(map[string]string),
		Metadata:      make(map[string]interface{}),
	}
}
