// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package openai

import (
	"context"
	"testing"

	"github.com/lancekrogers/guild/pkg/providers/base"
	"github.com/lancekrogers/guild/pkg/providers/interfaces"
	providertesting "github.com/lancekrogers/guild/pkg/providers/testing"
)

func TestOpenAIProvider(t *testing.T) {
	// Create mock server
	mock := providertesting.NewMockHTTPServer()
	defer mock.Close()

	// Create client with mock server URL
	client := &Client{
		OpenAICompatibleProvider: base.NewOpenAICompatibleProvider(
			"openai",
			"test-api-key",
			mock.URL+"/v1",
			nil,
			interfaces.ProviderCapabilities{
				MaxTokens:      128000,
				ContextWindow:  128000,
				SupportsVision: true,
				SupportsTools:  true,
				SupportsStream: true,
				Models: []interfaces.ModelInfo{
					{
						ID:            GPT41Mini,
						Name:          "GPT-4.1 Mini",
						ContextWindow: 1000000,
						MaxOutput:     32768,
						InputCost:     1.0,
						OutputCost:    4.0,
					},
				},
			},
		),
	}

	// Run standard test suite
	suite := providertesting.NewProviderTestSuite(t, client, providertesting.TestConfig{
		ProviderName: "OpenAI",
		TestModel:    GPT41Mini,
		SkipLive:     true,
	})

	suite.RunBasicTests()
}

func TestOpenAIModelConstants(t *testing.T) {
	// Ensure model constants are defined
	models := []string{
		GPT41,
		GPT41Mini,
		GPT41Nano,
		GPT4o,
		GPT4oMini,
		O3,
		O3Mini,
	}

	for _, model := range models {
		if model == "" {
			t.Error("Model constant is empty")
		}
	}
}

func TestOpenAIRecommendations(t *testing.T) {
	testCases := []struct {
		useCase  string
		expected string
	}{
		{"coding", GPT41},
		{"reasoning", O3Mini},
		{"multimodal", GPT4o},
		{"cost-efficient", GPT41Nano},
		{"general", GPT41Mini},
		{"unknown", GPT41Mini},
	}

	for _, tc := range testCases {
		t.Run(tc.useCase, func(t *testing.T) {
			model := GetRecommendedModel(tc.useCase)
			if model != tc.expected {
				t.Errorf("Expected %s for %s, got %s", tc.expected, tc.useCase, model)
			}
		})
	}
}

func TestOpenAICapabilities(t *testing.T) {
	client := NewClient("test-key")
	caps := client.GetCapabilities()

	if caps.MaxTokens != 1000000 {
		t.Errorf("Expected max tokens 1000000, got %d", caps.MaxTokens)
	}

	if !caps.SupportsVision {
		t.Error("OpenAI should support vision")
	}

	if !caps.SupportsTools {
		t.Error("OpenAI should support tools")
	}

	if !caps.SupportsStream {
		t.Error("OpenAI should support streaming")
	}

	// Check that we have the expected models
	modelMap := make(map[string]bool)
	for _, model := range caps.Models {
		modelMap[model.ID] = true
	}

	expectedModels := []string{GPT41, GPT41Mini, GPT41Nano, GPT4o, GPT4oMini, O3, O3Mini}
	for _, expected := range expectedModels {
		if !modelMap[expected] {
			t.Errorf("Missing expected model: %s", expected)
		}
	}
}

func TestOpenAILegacyInterface(t *testing.T) {
	// Test that legacy Complete method works
	mock := providertesting.NewMockHTTPServer()
	defer mock.Close()

	client := &Client{
		OpenAICompatibleProvider: base.NewOpenAICompatibleProvider(
			"openai",
			"test-api-key",
			mock.URL+"/v1",
			nil,
			interfaces.ProviderCapabilities{
				Models: []interfaces.ModelInfo{
					{ID: GPT41Mini, Name: "GPT-4.1 Mini"},
				},
			},
		),
	}

	// Legacy Complete method should work
	result, err := client.Complete(context.Background(), "test prompt")
	if err != nil {
		t.Errorf("Complete failed: %v", err)
	}

	if result == "" {
		t.Error("Expected non-empty result")
	}
}
