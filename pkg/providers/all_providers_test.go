// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package providers_test

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/guild-framework/guild-core/pkg/providers"
	"github.com/guild-framework/guild-core/pkg/providers/anthropic"
	"github.com/guild-framework/guild-core/pkg/providers/deepinfra"
	"github.com/guild-framework/guild-core/pkg/providers/deepseek"
	"github.com/guild-framework/guild-core/pkg/providers/interfaces"
	"github.com/guild-framework/guild-core/pkg/providers/mock"
	"github.com/guild-framework/guild-core/pkg/providers/ollama"
	"github.com/guild-framework/guild-core/pkg/providers/openai"
	"github.com/guild-framework/guild-core/pkg/providers/ora"
)

// TestAllProvidersImplementInterface ensures all providers implement AIProvider
func TestAllProvidersImplementInterface(t *testing.T) {
	// This will fail at compile time if any provider doesn't implement the interface
	var _ interfaces.AIProvider = (*openai.Client)(nil)
	var _ interfaces.AIProvider = (*anthropic.Client)(nil)
	var _ interfaces.AIProvider = (*deepseek.Client)(nil)
	var _ interfaces.AIProvider = (*deepinfra.Client)(nil)
	var _ interfaces.AIProvider = (*ollama.Client)(nil)
	var _ interfaces.AIProvider = (*ora.Client)(nil)
	var _ interfaces.AIProvider = (*mock.Provider)(nil)
}

// TestAllProvidersBasicFunctionality tests basic functionality across all providers
func TestAllProvidersBasicFunctionality(t *testing.T) {
	// Create mock provider
	mockBuilder, err := mock.NewBuilder()
	require.NoError(t, err)
	mockProvider := mockBuilder.
		WithDefaultResponse("Test response").
		Build()

	// Create all providers with mock/test configurations
	providers := map[string]interfaces.AIProvider{
		"openai":    openai.NewClient("test-key"),
		"anthropic": anthropic.NewClient("test-key"),
		"deepseek":  deepseek.NewClient("test-key"),
		"deepinfra": deepinfra.NewClient("test-key"),
		"ollama":    ollama.NewClient("http://localhost:11434"),
		"ora":       ora.NewClient("test-key"),
		"mock":      mockProvider,
	}

	for name, provider := range providers {
		t.Run(name, func(t *testing.T) {
			// Test capabilities
			caps := provider.GetCapabilities()
			if caps.MaxTokens <= 0 {
				t.Errorf("%s: Invalid max tokens", name)
			}
			if len(caps.Models) == 0 {
				t.Errorf("%s: No models available", name)
			}

			// Skip API tests for non-mock providers without proper setup
			if name == "mock" {
				// Test chat completion
				ctx := context.Background()
				req := interfaces.ChatRequest{
					Model: caps.Models[0].ID,
					Messages: []interfaces.ChatMessage{
						{Role: "user", Content: "Hello"},
					},
				}

				resp, err := provider.ChatCompletion(ctx, req)
				if err != nil {
					t.Errorf("%s: ChatCompletion failed: %v", name, err)
				} else if len(resp.Choices) == 0 {
					t.Errorf("%s: No choices in response", name)
				}
			}
		})
	}
}

// TestProviderCostComparison compares costs across providers
func TestProviderCostComparison(t *testing.T) {
	providers := map[string]interfaces.AIProvider{
		"openai":    openai.NewClient("test-key"),
		"anthropic": anthropic.NewClient("test-key"),
		"deepseek":  deepseek.NewClient("test-key"),
		"deepinfra": deepinfra.NewClient("test-key"),
		"ollama":    ollama.NewClient(""),
	}

	t.Log("Provider Cost Comparison (per million tokens):")
	t.Log("=========================================")

	for name, provider := range providers {
		caps := provider.GetCapabilities()
		if len(caps.Models) > 0 {
			model := caps.Models[0]
			t.Logf("%s - %s: Input=$%.2f, Output=$%.2f",
				name, model.Name, model.InputCost, model.OutputCost)
		}
	}

	// Find cheapest provider
	cheapest := ""
	cheapestCost := 999999.0

	for name, provider := range providers {
		caps := provider.GetCapabilities()
		if len(caps.Models) > 0 {
			model := caps.Models[0]
			avgCost := (model.InputCost + model.OutputCost) / 2
			if avgCost < cheapestCost && avgCost > 0 { // Exclude free (Ollama)
				cheapest = name
				cheapestCost = avgCost
			}
		}
	}

	t.Logf("\nCheapest provider: %s (avg $%.2f per million tokens)", cheapest, cheapestCost)
}

// TestProviderCapabilities compares capabilities across providers
func TestProviderCapabilities(t *testing.T) {
	providers := map[string]interfaces.AIProvider{
		"openai":    openai.NewClient("test-key"),
		"anthropic": anthropic.NewClient("test-key"),
		"deepseek":  deepseek.NewClient("test-key"),
		"deepinfra": deepinfra.NewClient("test-key"),
		"ollama":    ollama.NewClient(""),
		"ora":       ora.NewClient("test-key"),
	}

	t.Log("Provider Capabilities Comparison:")
	t.Log("================================")

	for name, provider := range providers {
		caps := provider.GetCapabilities()
		t.Logf("%s:", name)
		t.Logf("  Max Tokens: %d", caps.MaxTokens)
		t.Logf("  Context Window: %d", caps.ContextWindow)
		t.Logf("  Supports Vision: %v", caps.SupportsVision)
		t.Logf("  Supports Tools: %v", caps.SupportsTools)
		t.Logf("  Supports Stream: %v", caps.SupportsStream)
		t.Logf("  Models: %d available", len(caps.Models))
	}
}

// TestFactoryV2Creation tests the factory can create all providers
func TestFactoryV2Creation(t *testing.T) {
	factory := providers.NewFactoryV2()

	testCases := []struct {
		provider   providers.ProviderType
		apiKey     string
		shouldFail bool
	}{
		{providers.ProviderOpenAI, "test-key", false},
		{providers.ProviderAnthropic, "test-key", false},
		{providers.ProviderDeepSeek, "test-key", false},
		{providers.ProviderDeepInfra, "test-key", false},
		{providers.ProviderOllama, "", false},
		{providers.ProviderOra, "test-key", false},
		{providers.ProviderGoogle, "test-key", true},     // Not updated yet
		{providers.ProviderClaudeCode, "test-key", true}, // Not compatible
	}

	for _, tc := range testCases {
		t.Run(string(tc.provider), func(t *testing.T) {
			provider, err := factory.CreateAIProvider(tc.provider, tc.apiKey)
			if tc.shouldFail {
				if err == nil {
					t.Errorf("Expected error for %s, got nil", tc.provider)
				}
			} else {
				if err != nil {
					t.Errorf("Failed to create %s: %v", tc.provider, err)
				}
				if provider == nil {
					t.Errorf("Got nil provider for %s", tc.provider)
				}
			}
		})
	}
}

// TestLiveProviders tests against real APIs if API keys are available
func TestLiveProviders(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping live API tests")
	}

	ctx := context.Background()

	// Test each provider if API key is available
	testCases := []struct {
		name     string
		envVar   string
		provider func(string) interfaces.AIProvider
		model    string
	}{
		{
			name:     "OpenAI",
			envVar:   "OPENAI_API_KEY",
			provider: func(key string) interfaces.AIProvider { return openai.NewClient(key) },
			model:    openai.GPT41Mini,
		},
		{
			name:     "Anthropic",
			envVar:   "ANTHROPIC_API_KEY",
			provider: func(key string) interfaces.AIProvider { return anthropic.NewClient(key) },
			model:    anthropic.Claude4Sonnet,
		},
		{
			name:     "DeepSeek",
			envVar:   "DEEPSEEK_API_KEY",
			provider: func(key string) interfaces.AIProvider { return deepseek.NewClient(key) },
			model:    deepseek.DeepSeekChat,
		},
	}

	for _, tc := range testCases {
		apiKey := os.Getenv(tc.envVar)
		if apiKey == "" {
			t.Logf("Skipping %s live test (%s not set)", tc.name, tc.envVar)
			continue
		}

		t.Run(tc.name+"_Live", func(t *testing.T) {
			provider := tc.provider(apiKey)

			req := interfaces.ChatRequest{
				Model: tc.model,
				Messages: []interfaces.ChatMessage{
					{Role: "user", Content: "Say 'test successful' and nothing else"},
				},
				MaxTokens:   20,
				Temperature: 0,
			}

			resp, err := provider.ChatCompletion(ctx, req)
			if err != nil {
				t.Fatalf("Live API call failed: %v", err)
			}

			t.Logf("%s response: %s", tc.name, resp.Choices[0].Message.Content)
			t.Logf("Tokens used: %d", resp.Usage.TotalTokens)
		})
	}
}
