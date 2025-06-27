// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package providers

import (
	"context"
	"os"
	"testing"

	"github.com/lancekrogers/guild/pkg/providers/interfaces"
	"github.com/stretchr/testify/assert"
)

func TestFactoryV2Creation(t *testing.T) {
	factory := NewFactoryV2()
	assert.NotNil(t, factory)
}

func TestCreateAIProviderFromConfig_Basic(t *testing.T) {
	factory := NewFactoryV2()

	// Test OpenAI with direct API key
	config := map[string]interface{}{
		"api_key": "test-openai-key",
	}
	provider, err := factory.CreateAIProviderFromConfig(ProviderOpenAI, config)
	assert.NoError(t, err)
	assert.NotNil(t, provider)

	// Test with environment variable
	os.Setenv("TEST_API_KEY", "env-key")
	defer os.Unsetenv("TEST_API_KEY")

	config2 := map[string]interface{}{
		"api_key_env": "TEST_API_KEY",
	}
	provider2, err := factory.CreateAIProviderFromConfig(ProviderAnthropic, config2)
	assert.NoError(t, err)
	assert.NotNil(t, provider2)

	// Test Ollama with base URL
	config3 := map[string]interface{}{
		"base_url": "http://localhost:11434",
	}
	provider3, err := factory.CreateAIProviderFromConfig(ProviderOllama, config3)
	assert.NoError(t, err)
	assert.NotNil(t, provider3)

	// Test invalid provider
	_, err = factory.CreateAIProviderFromConfig("invalid", config)
	assert.Error(t, err)
}

func TestLLMClientAdapter_Basic(t *testing.T) {
	factory := NewFactoryV2()

	// Create a simple mock provider
	mockProvider := &SimpleMockProvider{
		models: []interfaces.ModelInfo{
			{ID: "test-model", Name: "Test Model"},
		},
	}

	adapter := factory.CreateLLMClientAdapter(mockProvider)
	assert.NotNil(t, adapter)

	ctx := context.Background()
	result, err := adapter.Complete(ctx, "test prompt")
	assert.NoError(t, err)
	assert.Equal(t, "mock response", result)
}

func TestExtractSystemPrompt(t *testing.T) {
	messages := []interfaces.ChatMessage{
		{Role: "system", Content: "You are helpful"},
		{Role: "user", Content: "Hello"},
	}

	prompt, remaining := extractSystemPromptFromMessages(messages)
	assert.Equal(t, "You are helpful", prompt)
	assert.Len(t, remaining, 1)
	assert.Equal(t, "user", remaining[0].Role)
}

func TestBuildPrompt(t *testing.T) {
	messages := []interfaces.ChatMessage{
		{Role: "user", Content: "Hello"},
		{Role: "assistant", Content: "Hi there"},
	}

	prompt := buildPromptFromMessages(messages)
	assert.Equal(t, "User: Hello\n\nAssistant: Hi there", prompt)

	// Test empty
	prompt2 := buildPromptFromMessages([]interfaces.ChatMessage{})
	assert.Equal(t, "", prompt2)
}

func TestGetProviderInfo(t *testing.T) {
	info := GetProviderInfo()
	assert.NotNil(t, info)

	// Check all providers are listed
	assert.Contains(t, info, ProviderOpenAI)
	assert.Contains(t, info, ProviderAnthropic)
	assert.Contains(t, info, ProviderDeepSeek)
	assert.Contains(t, info, ProviderDeepInfra)
	assert.Contains(t, info, ProviderOllama)
	assert.Contains(t, info, ProviderOra)
	assert.Contains(t, info, ProviderGoogle)
	assert.Contains(t, info, ProviderClaudeCode)

	// Check descriptions exist
	assert.NotEmpty(t, info[ProviderOpenAI])
	assert.NotEmpty(t, info[ProviderAnthropic])
}

// SimpleMockProvider for testing
type SimpleMockProvider struct {
	models []interfaces.ModelInfo
}

func (m *SimpleMockProvider) ChatCompletion(ctx context.Context, req interfaces.ChatRequest) (*interfaces.ChatResponse, error) {
	return &interfaces.ChatResponse{
		Choices: []interfaces.ChatChoice{
			{
				Message: interfaces.ChatMessage{
					Content: "mock response",
				},
			},
		},
	}, nil
}

func (m *SimpleMockProvider) StreamChatCompletion(ctx context.Context, req interfaces.ChatRequest) (interfaces.ChatStream, error) {
	return nil, nil
}

func (m *SimpleMockProvider) CreateEmbedding(ctx context.Context, req interfaces.EmbeddingRequest) (*interfaces.EmbeddingResponse, error) {
	return nil, nil
}

func (m *SimpleMockProvider) GetCapabilities() interfaces.ProviderCapabilities {
	return interfaces.ProviderCapabilities{
		Models: m.models,
	}
}
