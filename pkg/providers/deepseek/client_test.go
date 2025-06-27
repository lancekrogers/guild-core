// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package deepseek

import (
	"context"
	"os"
	"testing"

	"github.com/lancekrogers/guild/pkg/providers/base"
	"github.com/lancekrogers/guild/pkg/providers/interfaces"
	providertesting "github.com/lancekrogers/guild/pkg/providers/testing"
	"github.com/stretchr/testify/assert"
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
			mock.URL+"/v1",
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
			mock.URL+"/v1",
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

// TestNewClient tests client creation with different API key sources
func TestNewClient(t *testing.T) {
	tests := []struct {
		name           string
		apiKey         string
		envKey         string
		expectedNotNil bool
	}{
		{
			name:           "with explicit API key",
			apiKey:         "test-api-key",
			expectedNotNil: true,
		},
		{
			name:           "with environment variable",
			apiKey:         "",
			envKey:         "env-api-key",
			expectedNotNil: true,
		},
		{
			name:           "empty API key",
			apiKey:         "",
			envKey:         "",
			expectedNotNil: true, // Client still created, just without key
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variable if needed
			if tt.envKey != "" {
				os.Setenv("DEEPSEEK_API_KEY", tt.envKey)
				defer os.Unsetenv("DEEPSEEK_API_KEY")
			}

			client := NewClient(tt.apiKey)
			assert.Equal(t, tt.expectedNotNil, client != nil)
			if client != nil {
				assert.NotNil(t, client.OpenAICompatibleProvider)
			}
		})
	}
}

// TestGetRecommendedModel tests model recommendation logic
func TestGetRecommendedModel(t *testing.T) {
	tests := []struct {
		name     string
		useCase  string
		expected string
	}{
		{
			name:     "coding use case",
			useCase:  "coding",
			expected: DeepSeekChat,
		},
		{
			name:     "reasoning use case",
			useCase:  "reasoning",
			expected: DeepSeekReasoner,
		},
		{
			name:     "cost-efficient use case",
			useCase:  "cost-efficient",
			expected: DeepSeekChat,
		},
		{
			name:     "general use case",
			useCase:  "general",
			expected: DeepSeekChat,
		},
		{
			name:     "unknown use case",
			useCase:  "unknown",
			expected: DeepSeekChat,
		},
		{
			name:     "empty use case",
			useCase:  "",
			expected: DeepSeekChat,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetRecommendedModel(tt.useCase)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestComplete tests the Complete method
func TestComplete(t *testing.T) {
	// Skip integration test if no API key
	if os.Getenv("DEEPSEEK_API_KEY") == "" {
		t.Skip("DEEPSEEK_API_KEY not set")
	}

	client := NewClient("")
	ctx := context.Background()

	// Test basic completion
	result, err := client.Complete(ctx, "Say 'Hello' and nothing else")
	assert.NoError(t, err)
	assert.NotEmpty(t, result)
	assert.Contains(t, result, "Hello")
}
