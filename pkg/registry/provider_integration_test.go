// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package registry

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lancekrogers/guild-core/pkg/providers/mocks"
)

func TestProviderIntegration(t *testing.T) {
	// Create registry
	registry := NewComponentRegistry()

	// Create provider registry and add mock providers
	providerRegistry := registry.Providers()

	// Register mock providers directly
	mockOpenAI := mocks.NewMockClient()
	mockOpenAI.CompletionResponses = []string{"Mock OpenAI response: Hello, world!"}
	err := providerRegistry.RegisterProvider("openai", mockOpenAI)
	require.NoError(t, err)

	mockAnthropic := mocks.NewMockClient()
	mockAnthropic.CompletionResponses = []string{"Mock Anthropic response: Hello, world!"}
	err = providerRegistry.RegisterProvider("anthropic", mockAnthropic)
	require.NoError(t, err)

	mockOllama := mocks.NewMockClient()
	mockOllama.CompletionResponses = []string{"Mock Ollama response: Hello, world!"}
	err = providerRegistry.RegisterProvider("ollama", mockOllama)
	require.NoError(t, err)

	// Set default provider
	err = providerRegistry.SetDefaultProvider("openai")
	require.NoError(t, err)

	// Check that providers are registered
	providers := providerRegistry.ListProviders()
	assert.Contains(t, providers, "openai")
	assert.Contains(t, providers, "anthropic")
	assert.Contains(t, providers, "ollama")

	// Test getting specific providers
	openaiProvider, err := providerRegistry.GetProvider("openai")
	assert.NoError(t, err)
	assert.NotNil(t, openaiProvider)

	anthropicProvider, err := providerRegistry.GetProvider("anthropic")
	assert.NoError(t, err)
	assert.NotNil(t, anthropicProvider)

	ollamaProvider, err := providerRegistry.GetProvider("ollama")
	assert.NoError(t, err)
	assert.NotNil(t, ollamaProvider)

	// Test default provider
	defaultProvider, err := providerRegistry.GetDefaultProvider()
	assert.NoError(t, err)
	assert.NotNil(t, defaultProvider)

	// Test that the provider actually works
	ctx := context.Background()
	response, err := defaultProvider.Complete(ctx, "Hello, world!")
	assert.NoError(t, err)
	assert.Contains(t, response, "Hello, world!")

	// Test getting nonexistent provider
	_, err = providerRegistry.GetProvider("nonexistent")
	assert.Error(t, err)
}

func TestProviderConfigLoading(t *testing.T) {
	// Test configuration loading with different scenarios
	testCases := []struct {
		name         string
		providerName string
		mockResponse string
		expectError  bool
	}{
		{
			name:         "Mock OpenAI provider",
			providerName: "openai",
			mockResponse: "Mock OpenAI: test prompt",
			expectError:  false,
		},
		{
			name:         "Mock Anthropic provider",
			providerName: "anthropic",
			mockResponse: "Mock Anthropic: test prompt",
			expectError:  false,
		},
		{
			name:         "Mock Ollama provider",
			providerName: "ollama",
			mockResponse: "Mock Ollama: test prompt",
			expectError:  false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create registry
			registry := NewComponentRegistry()
			providerRegistry := registry.Providers()

			// Register mock provider for this test
			mockClient := mocks.NewMockClient()
			mockClient.CompletionResponses = []string{tc.mockResponse}

			err := providerRegistry.RegisterProvider(tc.providerName, mockClient)
			require.NoError(t, err)

			// Set as default provider
			err = providerRegistry.SetDefaultProvider(tc.providerName)
			require.NoError(t, err)

			// Verify provider was registered
			provider, err := providerRegistry.GetProvider(tc.providerName)
			assert.NoError(t, err)
			assert.NotNil(t, provider)

			// Test that it works
			ctx := context.Background()
			response, err := provider.Complete(ctx, "test prompt")
			if tc.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.mockResponse, response)
			}
		})
	}
}
