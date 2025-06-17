// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package setup

import (
	"context"
	"sort"
	"strings"

	"github.com/guild-ventures/guild-core/pkg/gerror"
)

// ModelConfig handles model selection and cost estimation
type ModelConfig struct {
	models map[string][]ModelInfo
}

// ModelInfo contains information about a specific model
type ModelInfo struct {
	Name               string
	Provider           string
	DisplayName        string
	Description        string
	CostPerInputToken  float64 // Cost per token for input
	CostPerOutputToken float64 // Cost per token for output
	ContextWindow      int     // Maximum context window size
	CostMagnitude      int     // Guild's Fibonacci cost scale (0,1,2,3,5,8)
	Recommended        bool    // Whether this model is recommended for general use
	Capabilities       []string
	UseCases           []string
	Limitations        []string
}

// NewModelConfig creates a new model configuration handler
func NewModelConfig(ctx context.Context) (*ModelConfig, error) {
	mc := &ModelConfig{
		models: make(map[string][]ModelInfo),
	}

	// Initialize model database
	mc.initializeModels()

	return mc, nil
}

// initializeModels sets up the model database with current pricing and capabilities
func (mc *ModelConfig) initializeModels() {
	// OpenAI Models (as of 2024)
	mc.models["openai"] = []ModelInfo{
		{
			Name:               "gpt-4-turbo-preview",
			Provider:           "openai",
			DisplayName:        "GPT-4 Turbo",
			Description:        "Most capable GPT-4 model with latest improvements",
			CostPerInputToken:  0.00001, // $0.01 per 1K tokens
			CostPerOutputToken: 0.00003, // $0.03 per 1K tokens
			ContextWindow:      128000,
			CostMagnitude:      5, // High cost
			Recommended:        true,
			Capabilities:       []string{"reasoning", "coding", "analysis", "writing"},
			UseCases:           []string{"complex reasoning", "code generation", "analysis"},
			Limitations:        []string{"high cost", "slower than GPT-3.5"},
		},
		{
			Name:               "gpt-4",
			Provider:           "openai",
			DisplayName:        "GPT-4",
			Description:        "Original GPT-4 model with excellent reasoning",
			CostPerInputToken:  0.00003, // $0.03 per 1K tokens
			CostPerOutputToken: 0.00006, // $0.06 per 1K tokens
			ContextWindow:      8192,
			CostMagnitude:      8, // Most expensive
			Recommended:        false,
			Capabilities:       []string{"reasoning", "coding", "analysis", "writing"},
			UseCases:           []string{"complex tasks", "high-quality output"},
			Limitations:        []string{"most expensive", "smaller context window"},
		},
		{
			Name:               "gpt-3.5-turbo",
			Provider:           "openai",
			DisplayName:        "GPT-3.5 Turbo",
			Description:        "Fast and efficient model for most tasks",
			CostPerInputToken:  0.0000005, // $0.0005 per 1K tokens
			CostPerOutputToken: 0.0000015, // $0.0015 per 1K tokens
			ContextWindow:      4096,
			CostMagnitude:      1, // Cheap
			Recommended:        true,
			Capabilities:       []string{"coding", "writing", "conversation"},
			UseCases:           []string{"general tasks", "quick responses", "high-volume"},
			Limitations:        []string{"less reasoning capability", "smaller context"},
		},
		{
			Name:               "gpt-3.5-turbo-16k",
			Provider:           "openai",
			DisplayName:        "GPT-3.5 Turbo 16K",
			Description:        "GPT-3.5 with larger context window",
			CostPerInputToken:  0.000001, // $0.001 per 1K tokens
			CostPerOutputToken: 0.000002, // $0.002 per 1K tokens
			ContextWindow:      16384,
			CostMagnitude:      2, // Low-mid cost
			Recommended:        false,
			Capabilities:       []string{"coding", "writing", "conversation"},
			UseCases:           []string{"longer documents", "extended conversations"},
			Limitations:        []string{"less reasoning than GPT-4"},
		},
	}

	// Anthropic Models
	mc.models["anthropic"] = []ModelInfo{
		{
			Name:               "claude-3-5-sonnet-20241022",
			Provider:           "anthropic",
			DisplayName:        "Claude 3.5 Sonnet",
			Description:        "Most advanced Claude model with excellent reasoning",
			CostPerInputToken:  0.000003, // $3 per 1M tokens
			CostPerOutputToken: 0.000015, // $15 per 1M tokens
			ContextWindow:      200000,
			CostMagnitude:      3, // Mid cost
			Recommended:        true,
			Capabilities:       []string{"reasoning", "analysis", "coding", "writing", "math"},
			UseCases:           []string{"complex analysis", "code review", "research"},
			Limitations:        []string{"no function calling", "no image generation"},
		},
		{
			Name:               "claude-3-opus-20240229",
			Provider:           "anthropic",
			DisplayName:        "Claude 3 Opus",
			Description:        "Most powerful Claude model for complex tasks",
			CostPerInputToken:  0.000015, // $15 per 1M tokens
			CostPerOutputToken: 0.000075, // $75 per 1M tokens
			ContextWindow:      200000,
			CostMagnitude:      8, // Most expensive
			Recommended:        false,
			Capabilities:       []string{"advanced reasoning", "research", "complex analysis"},
			UseCases:           []string{"research", "complex problem solving", "detailed analysis"},
			Limitations:        []string{"most expensive", "slower response"},
		},
		{
			Name:               "claude-3-sonnet-20240229",
			Provider:           "anthropic",
			DisplayName:        "Claude 3 Sonnet",
			Description:        "Balanced model for most use cases",
			CostPerInputToken:  0.000003, // $3 per 1M tokens
			CostPerOutputToken: 0.000015, // $15 per 1M tokens
			ContextWindow:      200000,
			CostMagnitude:      3, // Mid cost
			Recommended:        true,
			Capabilities:       []string{"reasoning", "coding", "analysis", "writing"},
			UseCases:           []string{"general tasks", "balanced performance"},
			Limitations:        []string{"less capable than Opus"},
		},
		{
			Name:               "claude-3-haiku-20240307",
			Provider:           "anthropic",
			DisplayName:        "Claude 3 Haiku",
			Description:        "Fast and efficient model for simple tasks",
			CostPerInputToken:  0.00000025, // $0.25 per 1M tokens
			CostPerOutputToken: 0.00000125, // $1.25 per 1M tokens
			ContextWindow:      200000,
			CostMagnitude:      1, // Cheap
			Recommended:        true,
			Capabilities:       []string{"quick responses", "simple coding", "writing"},
			UseCases:           []string{"high-volume tasks", "quick responses", "simple analysis"},
			Limitations:        []string{"limited reasoning", "simpler capabilities"},
		},
	}

	// Ollama Models (local, free)
	mc.models["ollama"] = []ModelInfo{
		{
			Name:               "llama2",
			Provider:           "ollama",
			DisplayName:        "Llama 2",
			Description:        "Meta's open-source language model",
			CostPerInputToken:  0, // Free local model
			CostPerOutputToken: 0, // Free local model
			ContextWindow:      4096,
			CostMagnitude:      0, // Free
			Recommended:        true,
			Capabilities:       []string{"general chat", "basic coding", "writing"},
			UseCases:           []string{"local processing", "privacy-sensitive tasks"},
			Limitations:        []string{"requires local compute", "less capable than cloud models"},
		},
		{
			Name:               "codellama",
			Provider:           "ollama",
			DisplayName:        "Code Llama",
			Description:        "Specialized version of Llama 2 for code generation",
			CostPerInputToken:  0, // Free local model
			CostPerOutputToken: 0, // Free local model
			ContextWindow:      4096,
			CostMagnitude:      0, // Free
			Recommended:        true,
			Capabilities:       []string{"code generation", "code completion", "debugging"},
			UseCases:           []string{"local development", "code assistance", "privacy"},
			Limitations:        []string{"requires local compute", "specialized for code only"},
		},
		{
			Name:               "mistral",
			Provider:           "ollama",
			DisplayName:        "Mistral",
			Description:        "Efficient open-source model from Mistral AI",
			CostPerInputToken:  0, // Free local model
			CostPerOutputToken: 0, // Free local model
			ContextWindow:      8192,
			CostMagnitude:      0, // Free
			Recommended:        true,
			Capabilities:       []string{"general tasks", "multilingual", "reasoning"},
			UseCases:           []string{"local processing", "multilingual tasks"},
			Limitations:        []string{"requires local compute", "may need fine-tuning"},
		},
		{
			Name:               "neural-chat",
			Provider:           "ollama",
			DisplayName:        "Neural Chat",
			Description:        "Intel's optimized chat model",
			CostPerInputToken:  0, // Free local model
			CostPerOutputToken: 0, // Free local model
			ContextWindow:      4096,
			CostMagnitude:      0, // Free
			Recommended:        false,
			Capabilities:       []string{"conversation", "basic tasks"},
			UseCases:           []string{"chatbot", "simple interactions"},
			Limitations:        []string{"limited capabilities", "requires local compute"},
		},
	}

	// Claude Code (special case)
	mc.models["claude_code"] = []ModelInfo{
		{
			Name:               "claude-3-5-sonnet-20241022",
			Provider:           "claude_code",
			DisplayName:        "Claude 3.5 Sonnet (Code)",
			Description:        "Claude 3.5 Sonnet optimized for coding in Claude Code environment",
			CostPerInputToken:  0, // Included in Claude Code subscription
			CostPerOutputToken: 0, // Included in Claude Code subscription
			ContextWindow:      200000,
			CostMagnitude:      0, // No additional cost
			Recommended:        true,
			Capabilities:       []string{"advanced coding", "debugging", "code analysis", "refactoring"},
			UseCases:           []string{"code development", "debugging", "code review"},
			Limitations:        []string{"requires Claude Code environment"},
		},
	}

	// DeepSeek Models
	mc.models["deepseek"] = []ModelInfo{
		{
			Name:               "deepseek-chat",
			Provider:           "deepseek",
			DisplayName:        "DeepSeek Chat",
			Description:        "General purpose model from DeepSeek",
			CostPerInputToken:  0.0000014, // $0.14 per 1M tokens
			CostPerOutputToken: 0.0000028, // $0.28 per 1M tokens
			ContextWindow:      32768,
			CostMagnitude:      1, // Cheap
			Recommended:        true,
			Capabilities:       []string{"general chat", "reasoning", "coding"},
			UseCases:           []string{"cost-effective general tasks"},
			Limitations:        []string{"may have quality tradeoffs"},
		},
		{
			Name:               "deepseek-coder",
			Provider:           "deepseek",
			DisplayName:        "DeepSeek Coder",
			Description:        "Specialized coding model from DeepSeek",
			CostPerInputToken:  0.0000014, // $0.14 per 1M tokens
			CostPerOutputToken: 0.0000028, // $0.28 per 1M tokens
			ContextWindow:      32768,
			CostMagnitude:      1, // Cheap
			Recommended:        true,
			Capabilities:       []string{"code generation", "debugging", "code explanation"},
			UseCases:           []string{"cost-effective coding tasks"},
			Limitations:        []string{"specialized for coding"},
		},
	}

	// DeepInfra Models (hosting various open-source models)
	mc.models["deepinfra"] = []ModelInfo{
		{
			Name:               "meta-llama/Llama-2-70b-chat-hf",
			Provider:           "deepinfra",
			DisplayName:        "Llama 2 70B Chat",
			Description:        "Large Llama 2 model hosted on DeepInfra",
			CostPerInputToken:  0.0000007, // $0.70 per 1M tokens
			CostPerOutputToken: 0.0000009, // $0.90 per 1M tokens
			ContextWindow:      4096,
			CostMagnitude:      1, // Cheap
			Recommended:        true,
			Capabilities:       []string{"general chat", "reasoning", "large context"},
			UseCases:           []string{"cost-effective high-quality responses"},
			Limitations:        []string{"may have availability issues"},
		},
		{
			Name:               "codellama/CodeLlama-34b-Instruct-hf",
			Provider:           "deepinfra",
			DisplayName:        "Code Llama 34B",
			Description:        "Large Code Llama model for complex coding tasks",
			CostPerInputToken:  0.0000006, // $0.60 per 1M tokens
			CostPerOutputToken: 0.0000006, // $0.60 per 1M tokens
			ContextWindow:      4096,
			CostMagnitude:      1, // Cheap
			Recommended:        true,
			Capabilities:       []string{"advanced coding", "debugging", "code analysis"},
			UseCases:           []string{"complex coding tasks", "code review"},
			Limitations:        []string{"specialized for code", "may have latency"},
		},
	}

	// Ora Models (aggregated access to multiple providers)
	mc.models["ora"] = []ModelInfo{
		{
			Name:               "gpt-4",
			Provider:           "ora",
			DisplayName:        "GPT-4 (via Ora)",
			Description:        "OpenAI GPT-4 accessed through Ora",
			CostPerInputToken:  0.00002, // Varies by Ora pricing
			CostPerOutputToken: 0.00004, // Varies by Ora pricing
			ContextWindow:      8192,
			CostMagnitude:      5, // High cost
			Recommended:        false,
			Capabilities:       []string{"advanced reasoning", "coding", "analysis"},
			UseCases:           []string{"when direct OpenAI access unavailable"},
			Limitations:        []string{"third-party access", "potential rate limits"},
		},
		{
			Name:               "claude-3-sonnet",
			Provider:           "ora",
			DisplayName:        "Claude 3 Sonnet (via Ora)",
			Description:        "Anthropic Claude accessed through Ora",
			CostPerInputToken:  0.000004, // Varies by Ora pricing
			CostPerOutputToken: 0.00002, // Varies by Ora pricing
			ContextWindow:      200000,
			CostMagnitude:      3, // Mid cost
			Recommended:        false,
			Capabilities:       []string{"reasoning", "analysis", "writing"},
			UseCases:           []string{"when direct Anthropic access unavailable"},
			Limitations:        []string{"third-party access", "potential limitations"},
		},
	}
}

// GetModelsForProvider returns available models for a specific provider
func (mc *ModelConfig) GetModelsForProvider(ctx context.Context, providerName string) ([]ModelInfo, error) {
	models, exists := mc.models[providerName]
	if !exists {
		return nil, gerror.Newf(gerror.ErrCodeNotFound, "no models found for provider: %s", providerName).
			WithComponent("setup").
			WithOperation("GetModelsForProvider")
	}

	// Return a copy to prevent modification
	result := make([]ModelInfo, len(models))
	copy(result, models)

	// Sort by recommendation and cost
	sort.Slice(result, func(i, j int) bool {
		// Recommended models first
		if result[i].Recommended && !result[j].Recommended {
			return true
		}
		if !result[i].Recommended && result[j].Recommended {
			return false
		}
		// Then by cost magnitude (lower is better)
		return result[i].CostMagnitude < result[j].CostMagnitude
	})

	return result, nil
}

// GetRecommendedModels returns recommended models across all providers
func (mc *ModelConfig) GetRecommendedModels(ctx context.Context) ([]ModelInfo, error) {
	var recommended []ModelInfo

	for _, providerModels := range mc.models {
		for _, model := range providerModels {
			if model.Recommended {
				recommended = append(recommended, model)
			}
		}
	}

	// Sort by cost magnitude and provider preference
	sort.Slice(recommended, func(i, j int) bool {
		// Prefer local models (cost magnitude 0)
		if recommended[i].CostMagnitude == 0 && recommended[j].CostMagnitude != 0 {
			return true
		}
		if recommended[i].CostMagnitude != 0 && recommended[j].CostMagnitude == 0 {
			return false
		}
		// Then by cost magnitude
		return recommended[i].CostMagnitude < recommended[j].CostMagnitude
	})

	return recommended, nil
}

// GetModelsByCapability returns models that have a specific capability
func (mc *ModelConfig) GetModelsByCapability(ctx context.Context, capability string) ([]ModelInfo, error) {
	var matching []ModelInfo

	for _, providerModels := range mc.models {
		for _, model := range providerModels {
			for _, cap := range model.Capabilities {
				if strings.EqualFold(cap, capability) {
					matching = append(matching, model)
					break
				}
			}
		}
	}

	if len(matching) == 0 {
		return nil, gerror.Newf(gerror.ErrCodeNotFound, "no models found with capability: %s", capability).
			WithComponent("setup").
			WithOperation("GetModelsByCapability")
	}

	// Sort by cost and recommendation
	sort.Slice(matching, func(i, j int) bool {
		if matching[i].Recommended && !matching[j].Recommended {
			return true
		}
		if !matching[i].Recommended && matching[j].Recommended {
			return false
		}
		return matching[i].CostMagnitude < matching[j].CostMagnitude
	})

	return matching, nil
}

// EstimateMonthlyCost estimates monthly cost for a given usage pattern
func (mc *ModelConfig) EstimateMonthlyCost(ctx context.Context, model ModelInfo, inputTokensPerDay, outputTokensPerDay int) *CostEstimate {
	inputCostPerDay := float64(inputTokensPerDay) * model.CostPerInputToken
	outputCostPerDay := float64(outputTokensPerDay) * model.CostPerOutputToken
	totalCostPerDay := inputCostPerDay + outputCostPerDay

	return &CostEstimate{
		Model:           model.Name,
		Provider:        model.Provider,
		InputTokens:     inputTokensPerDay * 30,
		OutputTokens:    outputTokensPerDay * 30,
		InputCost:       inputCostPerDay * 30,
		OutputCost:      outputCostPerDay * 30,
		TotalCost:       totalCostPerDay * 30,
		CostMagnitude:   model.CostMagnitude,
		IsLocal:         model.CostMagnitude == 0,
		UsageLevel:      mc.categorizeUsage(inputTokensPerDay + outputTokensPerDay),
	}
}

// categorizeUsage categorizes daily token usage
func (mc *ModelConfig) categorizeUsage(tokensPerDay int) string {
	switch {
	case tokensPerDay < 1000:
		return "light"
	case tokensPerDay < 10000:
		return "moderate"
	case tokensPerDay < 100000:
		return "heavy"
	default:
		return "enterprise"
	}
}

// GetCostComparison compares costs across different models for a usage pattern
func (mc *ModelConfig) GetCostComparison(ctx context.Context, capability string, inputTokensPerDay, outputTokensPerDay int) (*CostComparison, error) {
	models, err := mc.GetModelsByCapability(ctx, capability)
	if err != nil {
		return nil, err
	}

	var estimates []CostEstimate
	for _, model := range models {
		estimate := mc.EstimateMonthlyCost(ctx, model, inputTokensPerDay, outputTokensPerDay)
		estimates = append(estimates, *estimate)
	}

	// Sort by total cost
	sort.Slice(estimates, func(i, j int) bool {
		return estimates[i].TotalCost < estimates[j].TotalCost
	})

	return &CostComparison{
		Capability: capability,
		Usage:      estimates[0].UsageLevel,
		Estimates:  estimates,
		Cheapest:   estimates[0].Model,
		MostExpensive: estimates[len(estimates)-1].Model,
		Savings:    estimates[len(estimates)-1].TotalCost - estimates[0].TotalCost,
	}, nil
}

// CostEstimate contains cost estimation for a model
type CostEstimate struct {
	Model         string
	Provider      string
	InputTokens   int
	OutputTokens  int
	InputCost     float64
	OutputCost    float64
	TotalCost     float64
	CostMagnitude int
	IsLocal       bool
	UsageLevel    string
}

// CostComparison compares costs across multiple models
type CostComparison struct {
	Capability    string
	Usage         string
	Estimates     []CostEstimate
	Cheapest      string
	MostExpensive string
	Savings       float64
}

// GetModelRecommendations provides model recommendations based on use case
func (mc *ModelConfig) GetModelRecommendations(ctx context.Context, useCase string, budget string) (*ModelRecommendations, error) {
	var recommendations []ModelRecommendation

	// Define budget constraints
	var maxCostMagnitude int
	switch strings.ToLower(budget) {
	case "free", "local":
		maxCostMagnitude = 0
	case "low", "cheap":
		maxCostMagnitude = 1
	case "medium", "moderate":
		maxCostMagnitude = 3
	case "high", "premium":
		maxCostMagnitude = 5
	case "enterprise", "unlimited":
		maxCostMagnitude = 8
	default:
		maxCostMagnitude = 3 // Default to moderate
	}

	// Find suitable models based on use case and budget
	for _, providerModels := range mc.models {
		for _, model := range providerModels {
			if model.CostMagnitude <= maxCostMagnitude {
				score := mc.scoreModelForUseCase(model, useCase)
				if score > 0 {
					recommendations = append(recommendations, ModelRecommendation{
						Model:     model,
						Score:     score,
						Reasoning: mc.getRecommendationReasoning(model, useCase, budget),
					})
				}
			}
		}
	}

	// Sort by score (higher is better)
	sort.Slice(recommendations, func(i, j int) bool {
		return recommendations[i].Score > recommendations[j].Score
	})

	// Limit to top 5 recommendations
	if len(recommendations) > 5 {
		recommendations = recommendations[:5]
	}

	return &ModelRecommendations{
		UseCase:         useCase,
		Budget:          budget,
		Recommendations: recommendations,
	}, nil
}

// scoreModelForUseCase scores a model's suitability for a use case
func (mc *ModelConfig) scoreModelForUseCase(model ModelInfo, useCase string) int {
	score := 0
	useCaseLower := strings.ToLower(useCase)

	// Check if model's use cases match
	for _, modelUseCase := range model.UseCases {
		if strings.Contains(strings.ToLower(modelUseCase), useCaseLower) {
			score += 10
		}
	}

	// Check capabilities
	for _, capability := range model.Capabilities {
		if strings.Contains(useCaseLower, strings.ToLower(capability)) {
			score += 5
		}
	}

	// Bonus for recommended models
	if model.Recommended {
		score += 3
	}

	// Bonus for free models
	if model.CostMagnitude == 0 {
		score += 2
	}

	return score
}

// getRecommendationReasoning provides reasoning for a model recommendation
func (mc *ModelConfig) getRecommendationReasoning(model ModelInfo, useCase, budget string) string {
	reasons := []string{}

	if model.Recommended {
		reasons = append(reasons, "recommended model")
	}

	if model.CostMagnitude == 0 {
		reasons = append(reasons, "free local model")
	} else if model.CostMagnitude <= 1 {
		reasons = append(reasons, "cost-effective")
	}

	if model.ContextWindow > 100000 {
		reasons = append(reasons, "large context window")
	}

	// Add capability-specific reasons
	for _, capability := range model.Capabilities {
		if strings.Contains(strings.ToLower(useCase), strings.ToLower(capability)) {
			reasons = append(reasons, "excellent "+capability+" capabilities")
		}
	}

	if len(reasons) == 0 {
		reasons = append(reasons, "suitable for general use")
	}

	return strings.Join(reasons, ", ")
}

// ModelRecommendation contains a model recommendation
type ModelRecommendation struct {
	Model     ModelInfo
	Score     int
	Reasoning string
}

// ModelRecommendations contains recommendations for a specific use case
type ModelRecommendations struct {
	UseCase         string
	Budget          string
	Recommendations []ModelRecommendation
}