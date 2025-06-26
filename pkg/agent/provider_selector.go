// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package agent

import (
	"context"
	"strings"

	"github.com/guild-ventures/guild-core/pkg/config"
	"github.com/guild-ventures/guild-core/pkg/gerror"
	"github.com/guild-ventures/guild-core/pkg/observability"
	"github.com/guild-ventures/guild-core/pkg/providers"
)

// ProviderSelector handles model and provider selection logic for agents
type ProviderSelector struct {
	factory  *providers.Factory
	fallback []string // Fallback provider chain
}

// NewProviderSelector creates a new provider selector
func NewProviderSelector(factory *providers.Factory) *ProviderSelector {
	return &ProviderSelector{
		factory: factory,
		fallback: []string{
			"anthropic", // Primary fallback
			"openai",    // Secondary fallback
			"ollama",    // Local fallback
		},
	}
}

// ProviderSelection represents the result of provider selection
type ProviderSelection struct {
	Provider    string
	Model       string
	Client      providers.LLMClient
	CostProfile CostEstimate
	Fallbacks   []string
}

// CostEstimate represents cost information for a model
type CostEstimate struct {
	Magnitude       int     // Fibonacci scale: 0,1,2,3,5,8
	PromptCostPer1K float64 // Cost per 1K prompt tokens
	OutputCostPer1K float64 // Cost per 1K output tokens
	MaxTokens       int     // Maximum context window
}

// SelectProvider selects the best provider and creates a client for the agent configuration
func (ps *ProviderSelector) SelectProvider(ctx context.Context, config *config.EnhancedAgentConfig) (*ProviderSelection, error) {
	if err := ctx.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("ProviderSelector").
			WithOperation("SelectProvider")
	}

	logger := observability.GetLogger(ctx)
	ctx = observability.WithComponent(ctx, "ProviderSelector")
	ctx = observability.WithOperation(ctx, "SelectProvider")

	logger.InfoContext(ctx, "Selecting provider for agent", 
		"agent_id", config.ID, 
		"model", config.Model,
		"preferred_provider", config.GetEffectiveProvider())

	// Try the preferred provider first
	provider := config.GetEffectiveProvider()
	model := config.Model

	// Parse model string to extract provider and model name
	parsedProvider, parsedModel := ps.parseModelString(model)
	if parsedProvider != "" {
		provider = parsedProvider
		model = parsedModel
	}

	// Attempt to create client with preferred provider
	client, err := ps.createClient(ctx, provider, model)
	if err == nil {
		logger.InfoContext(ctx, "Successfully selected preferred provider", 
			"agent_id", config.ID, 
			"provider", provider, 
			"model", model)

		costProfile := ps.estimateCost(provider, model, config.GetEffectiveContextWindow())
		
		return &ProviderSelection{
			Provider:    provider,
			Model:       model,
			Client:      client,
			CostProfile: costProfile,
			Fallbacks:   ps.fallback,
		}, nil
	}

	logger.WarnContext(ctx, "Preferred provider failed, trying fallbacks", 
		"agent_id", config.ID, 
		"preferred_provider", provider, 
		"error", err)

	// Try fallback providers
	for _, fallbackProvider := range ps.fallback {
		// Skip if it's the same as the preferred provider
		if fallbackProvider == provider {
			continue
		}

		// Check context before trying fallback
		if err := ctx.Err(); err != nil {
			return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled during fallback").
				WithComponent("ProviderSelector").
				WithOperation("SelectProvider")
		}

		// Map original model to fallback provider's equivalent
		fallbackModel := ps.mapModelToProvider(model, fallbackProvider)
		
		client, err := ps.createClient(ctx, fallbackProvider, fallbackModel)
		if err == nil {
			logger.InfoContext(ctx, "Successfully selected fallback provider", 
				"agent_id", config.ID, 
				"provider", fallbackProvider, 
				"model", fallbackModel,
				"original_model", model)

			costProfile := ps.estimateCost(fallbackProvider, fallbackModel, config.GetEffectiveContextWindow())
			
			return &ProviderSelection{
				Provider:    fallbackProvider,
				Model:       fallbackModel,
				Client:      client,
				CostProfile: costProfile,
				Fallbacks:   ps.fallback,
			}, nil
		}

		logger.WarnContext(ctx, "Fallback provider failed", 
			"agent_id", config.ID, 
			"fallback_provider", fallbackProvider, 
			"error", err)
	}

	// All providers failed
	return nil, gerror.Newf(gerror.ErrCodeInternal, "no available providers for model '%s'", model).
		WithComponent("ProviderSelector").
		WithOperation("SelectProvider").
		WithDetails("agent_id", config.ID).
		WithDetails("model", model).
		WithDetails("preferred_provider", provider)
}

// parseModelString parses a model string to extract provider and model name
// Examples: "anthropic/claude-3-sonnet" -> "anthropic", "claude-3-sonnet"
//           "claude-3-sonnet" -> "", "claude-3-sonnet"
func (ps *ProviderSelector) parseModelString(model string) (string, string) {
	if strings.Contains(model, "/") {
		parts := strings.SplitN(model, "/", 2)
		if len(parts) == 2 {
			return parts[0], parts[1]
		}
	}
	return "", model
}

// mapModelToProvider maps a model name to an equivalent model for a different provider
func (ps *ProviderSelector) mapModelToProvider(originalModel, targetProvider string) string {
	originalModel = strings.ToLower(originalModel)
	
	switch targetProvider {
	case "anthropic":
		switch {
		case strings.Contains(originalModel, "gpt-4"):
			return "claude-3-5-sonnet-20241022" // High-performance model
		case strings.Contains(originalModel, "gpt-3.5"):
			return "claude-3-haiku-20240307" // Fast, cheap model
		default:
			return "claude-3-5-sonnet-20241022" // Default
		}
	case "openai":
		switch {
		case strings.Contains(originalModel, "claude-3-opus"):
			return "gpt-4-turbo-preview" // High-performance model
		case strings.Contains(originalModel, "claude-3-sonnet"):
			return "gpt-4" // Balanced model
		case strings.Contains(originalModel, "claude-3-haiku"):
			return "gpt-3.5-turbo" // Fast, cheap model
		default:
			return "gpt-4" // Default
		}
	case "ollama":
		// Map to local models
		switch {
		case strings.Contains(originalModel, "claude") || strings.Contains(originalModel, "gpt-4"):
			return "llama3.1:8b" // Good local model
		default:
			return "llama3.1:8b" // Default local model
		}
	case "deepseek":
		return "deepseek-chat" // DeepSeek's main model
	default:
		return originalModel // Return original if no mapping
	}
}

// createClient creates an LLM client for the specified provider and model
func (ps *ProviderSelector) createClient(ctx context.Context, provider, model string) (providers.LLMClient, error) {
	// Check if factory is valid (using interface{} comparison)
	if ps.factory == nil {
		return nil, gerror.New(gerror.ErrCodeInvalidInput, "provider factory is nil", nil).
			WithComponent("ProviderSelector").
			WithOperation("createClient")
	}

	logger := observability.GetLogger(ctx)
	logger.DebugContext(ctx, "Creating client", "provider", provider, "model", model)

	// Create provider-specific client using the correct factory method
	providerType := providers.ProviderType(provider)
	client, err := ps.factory.CreateClient(providerType, "", model)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create client").
			WithComponent("ProviderSelector").
			WithOperation("createClient").
			WithDetails("provider", provider).
			WithDetails("model", model)
	}

	return client, nil
}

// estimateCost estimates the cost profile for a provider and model
func (ps *ProviderSelector) estimateCost(provider, model string, contextWindow int) CostEstimate {
	modelLower := strings.ToLower(model)
	
	// Default cost estimate
	estimate := CostEstimate{
		Magnitude:       1,
		PromptCostPer1K: 0.001,
		OutputCostPer1K: 0.002,
		MaxTokens:       contextWindow,
	}

	switch provider {
	case "anthropic":
		switch {
		case strings.Contains(modelLower, "claude-3-opus"):
			estimate.Magnitude = 8
			estimate.PromptCostPer1K = 0.015
			estimate.OutputCostPer1K = 0.075
		case strings.Contains(modelLower, "claude-3-5-sonnet"):
			estimate.Magnitude = 3
			estimate.PromptCostPer1K = 0.003
			estimate.OutputCostPer1K = 0.015
		case strings.Contains(modelLower, "claude-3-haiku"):
			estimate.Magnitude = 1
			estimate.PromptCostPer1K = 0.00025
			estimate.OutputCostPer1K = 0.00125
		}
	case "openai":
		switch {
		case strings.Contains(modelLower, "gpt-4-turbo"):
			estimate.Magnitude = 5
			estimate.PromptCostPer1K = 0.01
			estimate.OutputCostPer1K = 0.03
		case strings.Contains(modelLower, "gpt-4"):
			estimate.Magnitude = 5
			estimate.PromptCostPer1K = 0.03
			estimate.OutputCostPer1K = 0.06
		case strings.Contains(modelLower, "gpt-3.5"):
			estimate.Magnitude = 2
			estimate.PromptCostPer1K = 0.0005
			estimate.OutputCostPer1K = 0.0015
		}
	case "deepseek":
		estimate.Magnitude = 1
		estimate.PromptCostPer1K = 0.00014
		estimate.OutputCostPer1K = 0.00028
	case "ollama":
		estimate.Magnitude = 0 // Local models are free
		estimate.PromptCostPer1K = 0.0
		estimate.OutputCostPer1K = 0.0
	}

	return estimate
}

// GetAvailableProviders returns a list of available providers
func (ps *ProviderSelector) GetAvailableProviders(ctx context.Context) ([]string, error) {
	if err := ctx.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("ProviderSelector").
			WithOperation("GetAvailableProviders")
	}

	if ps.factory == nil {
		return nil, gerror.New(gerror.ErrCodeInvalidInput, "provider factory is nil", nil).
			WithComponent("ProviderSelector").
			WithOperation("GetAvailableProviders")
	}

	// Note: This assumes the factory has a method to list providers
	// If not available, return the standard list
	return []string{
		"anthropic",
		"openai", 
		"ollama",
		"deepseek",
		"deepinfra",
		"ora",
		"claudecode",
	}, nil
}

// ValidateModelProvider validates that a model and provider combination is valid
func (ps *ProviderSelector) ValidateModelProvider(ctx context.Context, provider, model string) error {
	if err := ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("ProviderSelector").
			WithOperation("ValidateModelProvider")
	}

	logger := observability.GetLogger(ctx)
	ctx = observability.WithComponent(ctx, "ProviderSelector")
	ctx = observability.WithOperation(ctx, "ValidateModelProvider")

	logger.DebugContext(ctx, "Validating model-provider combination", "provider", provider, "model", model)

	// Try to create a client to validate the combination
	_, err := ps.createClient(ctx, provider, model)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeValidation, "invalid model-provider combination").
			WithComponent("ProviderSelector").
			WithOperation("ValidateModelProvider").
			WithDetails("provider", provider).
			WithDetails("model", model)
	}

	logger.DebugContext(ctx, "Model-provider combination validated successfully", "provider", provider, "model", model)
	return nil
}

// SetFallbackChain sets a custom fallback provider chain
func (ps *ProviderSelector) SetFallbackChain(fallbacks []string) {
	ps.fallback = make([]string, len(fallbacks))
	copy(ps.fallback, fallbacks)
}

// GetFallbackChain returns the current fallback provider chain
func (ps *ProviderSelector) GetFallbackChain() []string {
	fallbacks := make([]string, len(ps.fallback))
	copy(fallbacks, ps.fallback)
	return fallbacks
}