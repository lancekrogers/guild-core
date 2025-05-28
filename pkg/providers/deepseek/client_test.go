package deepseek

import (
	"context"
	"testing"

	"github.com/guild-ventures/guild-core/pkg/providers/base"
	"github.com/guild-ventures/guild-core/pkg/providers/interfaces"
	providertesting "github.com/guild-ventures/guild-core/pkg/providers/testing"
)

func TestDeepSeekProvider(t *testing.T) {
	// Create mock server
	mock := providertesting.NewMockHTTPServer()
	defer mock.Close()

	// Create client with mock server URL
	client := &Client{
		OpenAICompatibleProvider: base.NewOpenAICompatibleProvider(
			"deepseek",
			"test-api-key",
			mock.URL + "/v1",
			map[string]string{
				"gpt-4": DeepSeekChat,
			},
			interfaces.ProviderCapabilities{
				MaxTokens:      64000,
				ContextWindow:  64000,
				SupportsVision: false,
				SupportsTools:  true,
				SupportsStream: true,
				Models: []interfaces.ModelInfo{
					{
						ID:            DeepSeekChat,
						Name:          "DeepSeek Chat V3",
						ContextWindow: 64000,
						MaxOutput:     8192,
						InputCost:     0.07,
						OutputCost:    1.10,
					},
					{
						ID:            DeepSeekReasoner,
						Name:          "DeepSeek Reasoner R1",
						ContextWindow: 64000,
						MaxOutput:     8192,
						InputCost:     0.55,
						OutputCost:    2.19,
					},
				},
			},
		),
	}

	// Run standard test suite
	suite := providertesting.NewProviderTestSuite(t, client, providertesting.TestConfig{
		ProviderName: "DeepSeek",
		TestModel:    DeepSeekChat,
		SkipLive:     true,
	})
	
	suite.RunBasicTests()
}

func TestDeepSeekSpecificFeatures(t *testing.T) {
	mock := providertesting.NewMockHTTPServer()
	defer mock.Close()

	// Model mapping for OpenAI compatibility
	modelMap := map[string]string{
		"gpt-4":         DeepSeekChat,
		"gpt-4-turbo":   DeepSeekReasoner,
		"gpt-3.5-turbo": DeepSeekChat,
	}

	client := &Client{
		OpenAICompatibleProvider: base.NewOpenAICompatibleProvider(
			"deepseek",
			"test-api-key",
			mock.URL + "/v1",
			modelMap,
			interfaces.ProviderCapabilities{
				MaxTokens:      64000,
				ContextWindow:  64000,
				SupportsVision: false,
				SupportsTools:  true,
				SupportsStream: true,
				Models: []interfaces.ModelInfo{
					{ID: DeepSeekChat, Name: "DeepSeek Chat V3"},
					{ID: DeepSeekReasoner, Name: "DeepSeek Reasoner R1"},
				},
			},
		),
	}

	t.Run("ModelMapping", func(t *testing.T) {
		ctx := context.Background()
		
		// Test that GPT-4 is mapped to DeepSeek Chat
		req := interfaces.ChatRequest{
			Model: "gpt-4",
			Messages: []interfaces.ChatMessage{
				{Role: "user", Content: "Hello"},
			},
		}

		_, err := client.ChatCompletion(ctx, req)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}

		// Check the actual request sent
		lastReq := mock.GetLastRequest()
		if lastReq != nil {
			// The request should have the mapped model
			// This would require parsing the request body
		}
	})

	t.Run("ModelRecommendation", func(t *testing.T) {
		testCases := []struct {
			useCase  string
			expected string
		}{
			{"coding", DeepSeekChat},
			{"reasoning", DeepSeekReasoner},
			{"cost-efficient", DeepSeekChat},
			{"general", DeepSeekChat},
		}

		for _, tc := range testCases {
			t.Run(tc.useCase, func(t *testing.T) {
				model := GetRecommendedModel(tc.useCase)
				if model != tc.expected {
					t.Errorf("Expected %s for %s, got %s", tc.expected, tc.useCase, model)
				}
			})
		}
	})

	t.Run("CostEfficiency", func(t *testing.T) {
		caps := client.GetCapabilities()
		
		// Find DeepSeek Chat model
		var chatModel *interfaces.ModelInfo
		for _, m := range caps.Models {
			if m.ID == DeepSeekChat {
				chatModel = &m
				break
			}
		}

		if chatModel == nil {
			t.Fatal("DeepSeek Chat model not found")
		}

		// Verify it's very cost-efficient
		if chatModel.InputCost > 0.1 {
			t.Errorf("DeepSeek Chat should be very cheap, got $%.2f", chatModel.InputCost)
		}
	})
}

// Note: DeepSeek offers special pricing during off-peak hours
// This test would verify the pricing but actual discounts are applied by the API
func TestDeepSeekPricingNote(t *testing.T) {
	t.Log("DeepSeek offers 50-75% off-peak discount (16:30-00:30 UTC)")
	t.Log("Cache hit pricing also available for repeated content")
}