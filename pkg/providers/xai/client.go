// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package xai

import (
	"context"
	"os"

	"github.com/lancekrogers/guild-core/pkg/providers/base"
	"github.com/lancekrogers/guild-core/pkg/providers/interfaces"
)

// xAI Grok models
const (
	// Grok 4 models (latest flagship)
	Grok4       = "grok-4"       // Alias for latest Grok-4
	Grok4Latest = "grok-4-latest" // Alias for latest Grok-4
	Grok40709   = "grok-4-0709"   // Latest flagship reasoning model (256K context)
	Grok4Fast   = "grok-4-fast"   // High-speed model with 2M context

	// Grok 3 models
	Grok3Beta     = "grok-3-beta"      // Latest flagship model for enterprise tasks (131K context)
	Grok3FastBeta = "grok-3-fast-beta" // Fastest flagship model (131K context)
	Grok3MiniBeta = "grok-3-mini-beta" // Smaller model for basic tasks (32K context)
	Grok3MiniFastBeta = "grok-3-mini-fast-beta" // Faster mini model

	// Code-optimized models
	GrokCodeFast      = "grok-code-fast"       // Alias for grok-code-fast-1
	GrokCodeFast1     = "grok-code-fast-1"     // Speedy coding model (256K context)
	GrokCodeFast10825 = "grok-code-fast-1-0825" // Specific version (256K context)

	// Vision models
	Grok2Vision       = "grok-2-vision"        // Grok-2 Vision (32K context)
	Grok2VisionLatest = "grok-2-vision-latest" // Latest Grok-2 Vision
	Grok2Vision1212   = "grok-2-vision-1212"   // Grok-2 Vision version 1212

	// Legacy models
	GrokBeta       = "grok-beta"        // Legacy Grok Beta (131K context)
	GrokVisionBeta = "grok-vision-beta" // Legacy Grok Vision Beta (8K context)
)

// Client implements the AIProvider interface for xAI Grok
type Client struct {
	*base.OpenAICompatibleProvider
}

// NewClient creates a new xAI Grok client
func NewClient(apiKey string) *Client {
	if apiKey == "" {
		apiKey = os.Getenv("XAI_API_KEY")
		if apiKey == "" {
			apiKey = os.Getenv("GROK_API_KEY")
		}
	}

	// Model mappings for OpenAI compatibility
	modelMap := map[string]string{
		"gpt-4":         Grok4,
		"gpt-4-turbo":   Grok4Fast,
		"gpt-3.5-turbo": Grok3MiniBeta,
	}

	capabilities := interfaces.ProviderCapabilities{
		MaxTokens:      2000000, // Grok 4 Fast supports up to 2M tokens
		ContextWindow:  2000000,
		SupportsVision: true, // Vision supported via grok-2-vision models
		SupportsTools:  true, // Native tool use support
		SupportsStream: true,
		Models: []interfaces.ModelInfo{
			{
				ID:            Grok4,
				Name:          "Grok 4",
				ContextWindow: 256000,
				MaxOutput:     8192,
				InputCost:     3.00,  // Per million tokens
				OutputCost:    15.00, // Per million tokens
			},
			{
				ID:            Grok4Fast,
				Name:          "Grok 4 Fast",
				ContextWindow: 2000000, // 2M context window
				MaxOutput:     8192,
				InputCost:     0.20,  // Under 128K tokens
				OutputCost:    0.50,  // Under 128K tokens
			},
			{
				ID:            Grok3Beta,
				Name:          "Grok 3 Beta",
				ContextWindow: 131000,
				MaxOutput:     8192,
				InputCost:     3.00,
				OutputCost:    15.00,
			},
			{
				ID:            Grok3FastBeta,
				Name:          "Grok 3 Fast Beta",
				ContextWindow: 131000,
				MaxOutput:     8192,
				InputCost:     5.00,  // Premium for speed
				OutputCost:    25.00,
			},
			{
				ID:            Grok3MiniBeta,
				Name:          "Grok 3 Mini Beta",
				ContextWindow: 32000,
				MaxOutput:     8192,
				InputCost:     0.30,
				OutputCost:    0.50,
			},
			{
				ID:            Grok3MiniFastBeta,
				Name:          "Grok 3 Mini Fast Beta",
				ContextWindow: 32000,
				MaxOutput:     8192,
				InputCost:     0.60,
				OutputCost:    4.00,
			},
			{
				ID:            GrokCodeFast1,
				Name:          "Grok Code Fast",
				ContextWindow: 256000,
				MaxOutput:     8192,
				InputCost:     0.20, // Estimated based on fast pricing
				OutputCost:    0.50,
			},
			{
				ID:            Grok2Vision,
				Name:          "Grok 2 Vision",
				ContextWindow: 32000,
				MaxOutput:     8192,
				InputCost:     3.00, // Estimated
				OutputCost:    15.00,
			},
			{
				ID:            GrokBeta,
				Name:          "Grok Beta (Legacy)",
				ContextWindow: 131000,
				MaxOutput:     8192,
				InputCost:     3.00,
				OutputCost:    15.00,
			},
		},
	}

	provider := base.NewOpenAICompatibleProvider(
		"xai",
		apiKey,
		"https://api.x.ai/v1",
		modelMap,
		capabilities,
	)

	return &Client{
		OpenAICompatibleProvider: provider,
	}
}

// GetRecommendedModel returns a recommended model for a given use case
func GetRecommendedModel(useCase string) string {
	switch useCase {
	case "coding":
		return GrokCodeFast1 // Optimized for agentic coding
	case "reasoning":
		return Grok4 // Most intelligent model
	case "vision":
		return Grok2Vision // Image support
	case "cost-efficient":
		return Grok3MiniBeta // Most affordable
	case "fast":
		return Grok4Fast // Fastest with large context
	case "large-context":
		return Grok4Fast // 2M token context window
	default:
		return Grok4 // General purpose flagship
	}
}

// Legacy LLMClient interface support
func (c *Client) Complete(ctx context.Context, prompt string) (string, error) {
	req := interfaces.ChatRequest{
		Model: Grok4, // Default to flagship model
		Messages: []interfaces.ChatMessage{
			{Role: "user", Content: prompt},
		},
	}

	resp, err := c.ChatCompletion(ctx, req)
	if err != nil {
		return "", err
	}

	if len(resp.Choices) > 0 {
		return resp.Choices[0].Message.Content, nil
	}

	return "", nil
}

// Note: xAI Grok offers several unique features:
// - Real-time search integration (for applicable models)
// - Native tool use and agentic capabilities
// - Knowledge cutoff: November 2024
// - $25 of free API credits per month
// - OpenAI-compatible API for easy migration
