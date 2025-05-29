package providers

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/guild-ventures/guild-core/pkg/providers/anthropic"
	"github.com/guild-ventures/guild-core/pkg/providers/ollama"
	"github.com/guild-ventures/guild-core/pkg/providers/openai"
)

func TestOpenAIModels(t *testing.T) {
	// Test model validation and defaults
	client := openai.NewClient("test-key", "")
	assert.Equal(t, "gpt-4.1", client.GetModel()) // Should use default

	// Test with valid model
	client = openai.NewClient("test-key", "gpt-4o")
	assert.Equal(t, "gpt-4o", client.GetModel())

	// Test with invalid model (should fallback to default)
	client = openai.NewClient("test-key", "invalid-model")
	assert.Equal(t, "gpt-4.1", client.GetModel())

	// Test model info
	info, exists := client.GetModelInfo()
	assert.True(t, exists)
	assert.Equal(t, "gpt-4.1", info.Name)
	assert.Equal(t, "text", info.Type)
	assert.Equal(t, 1000000, info.MaxTokens)

	// Test completion
	ctx := context.Background()
	response, err := client.Complete(ctx, "test prompt")
	assert.NoError(t, err)
	assert.Contains(t, response, "gpt-4.1")
	assert.Contains(t, response, "test prompt")
}

func TestAnthropicModels(t *testing.T) {
	// Test model validation and defaults
	client := anthropic.NewClient("test-key", "")
	assert.Equal(t, "claude-4-sonnet", client.GetModel()) // Should use default

	// Test with valid model
	client = anthropic.NewClient("test-key", "claude-4-opus")
	assert.Equal(t, "claude-4-opus", client.GetModel())

	// Test with invalid model (should fallback to default)
	client = anthropic.NewClient("test-key", "invalid-model")
	assert.Equal(t, "claude-4-sonnet", client.GetModel())

	// Test model info
	info, exists := client.GetModelInfo()
	assert.True(t, exists)
	assert.Equal(t, "claude-4-sonnet", info.Name)
	assert.Equal(t, "text", info.Type)
	assert.Equal(t, 200000, info.MaxTokens)

	// Test completion
	ctx := context.Background()
	response, err := client.Complete(ctx, "test prompt")
	assert.NoError(t, err)
	assert.Contains(t, response, "claude-4-sonnet")
	assert.Contains(t, response, "test prompt")
}

// Google provider tests removed - pending post-MVP implementation

func TestOllamaModels(t *testing.T) {
	// Test model validation and defaults
	client := ollama.NewClient("", "")
	assert.Equal(t, "llama3.1:8b", client.GetModel()) // Should use default

	// Test with valid model
	client = ollama.NewClient("", "phi4:14b")
	assert.Equal(t, "phi4:14b", client.GetModel())

	// Test with invalid model (should fallback to default)
	client = ollama.NewClient("", "invalid-model")
	assert.Equal(t, "llama3.1:8b", client.GetModel())

	// Test model info
	info, exists := client.GetModelInfo()
	assert.True(t, exists)
	assert.Equal(t, "llama3.1:8b", info.Name)
	assert.Equal(t, "text", info.Type)
	assert.Equal(t, "8B", info.Size)
	assert.Equal(t, 8, info.MinRAM)

	// Test completion
	ctx := context.Background()
	response, err := client.Complete(ctx, "test prompt")
	assert.NoError(t, err)
	assert.Contains(t, response, "llama3.1:8b")
	assert.Contains(t, response, "test prompt")

	// Test URL functionality
	assert.Equal(t, "http://localhost:11434", client.GetBaseURL())

	// Test custom URL
	client = ollama.NewClient("http://custom:11434", "")
	assert.Equal(t, "http://custom:11434", client.GetBaseURL())
}

func TestModelRecommendations(t *testing.T) {
	// Test OpenAI recommendations
	assert.Equal(t, "gpt-4.1", openai.GetRecommendedModel("coding"))
	assert.Equal(t, "o3-mini", openai.GetRecommendedModel("reasoning"))
	assert.Equal(t, "gpt-4o", openai.GetRecommendedModel("multimodal"))
	assert.Equal(t, "gpt-4.1-nano", openai.GetRecommendedModel("cost-efficient"))

	// Test Anthropic recommendations
	assert.Equal(t, "claude-4-opus", anthropic.GetRecommendedModel("coding"))
	assert.Equal(t, "claude-4-opus", anthropic.GetRecommendedModel("reasoning"))
	assert.Equal(t, "claude-3.7-sonnet", anthropic.GetRecommendedModel("hybrid-reasoning"))
	assert.Equal(t, "claude-3-5-haiku-20241022", anthropic.GetRecommendedModel("cost-efficient"))

	// Google recommendations tests removed - pending post-MVP implementation

	// Test Ollama recommendations with RAM constraints
	assert.Equal(t, "codegemma:7b", ollama.GetRecommendedModel("coding", 8))
	assert.Equal(t, "qwen2-math:7b", ollama.GetRecommendedModel("math", 8))
	assert.Equal(t, "deepseek-r1:70b", ollama.GetRecommendedModel("reasoning", 64))
	assert.Equal(t, "phi4-mini:3.8b", ollama.GetRecommendedModel("fast", 4)) // Updated expected value
	
	// Test with low RAM
	assert.Equal(t, "gemma2:2b", ollama.GetRecommendedModel("anything", 2))
}

func TestModelFiltering(t *testing.T) {
	// Test OpenAI model filtering by type
	textModels := openai.GetModelsByType("text")
	assert.Greater(t, len(textModels), 0)
	for _, model := range textModels {
		assert.Equal(t, "text", model.Type)
	}

	multimodalModels := openai.GetModelsByType("multimodal")
	assert.Greater(t, len(multimodalModels), 0)
	for _, model := range multimodalModels {
		assert.Equal(t, "multimodal", model.Type)
	}

	// Test Ollama model filtering by category
	smallModels := ollama.GetModelsByCategory("small")
	assert.Greater(t, len(smallModels), 0)
	for _, model := range smallModels {
		assert.Equal(t, "small", model.Category)
	}

	// Test Ollama model filtering by RAM
	lowRAMModels := ollama.GetModelsByRAM(4)
	assert.Greater(t, len(lowRAMModels), 0)
	for _, model := range lowRAMModels {
		assert.LessOrEqual(t, model.MinRAM, 4)
	}
}

func TestFactoryWithNewModels(t *testing.T) {
	factory := NewFactory()

	// Test OpenAI with latest model
	client, err := factory.CreateClient(ProviderOpenAI, "test-key", "gpt-4.1")
	require.NoError(t, err)
	assert.NotNil(t, client)

	// Test Anthropic with latest model
	client, err = factory.CreateClient(ProviderAnthropic, "test-key", "claude-4-sonnet")
	require.NoError(t, err)
	assert.NotNil(t, client)

	// Google provider tests removed - pending post-MVP implementation

	// Test Ollama with latest model
	client, err = factory.CreateClient(ProviderOllama, "", "llama3.3:70b")
	require.NoError(t, err)
	assert.NotNil(t, client)
}

func TestCurrentModelCoverage(t *testing.T) {
	// Verify we have current models for each provider
	
	// OpenAI - should have GPT-4.1 series
	openaiModels := openai.ListSupportedModels()
	assert.Contains(t, openaiModels, "gpt-4.1")
	assert.Contains(t, openaiModels, "gpt-4.1-mini")
	assert.Contains(t, openaiModels, "gpt-4o")
	assert.Contains(t, openaiModels, "o1")
	assert.Contains(t, openaiModels, "o3-mini")

	// Anthropic - should have Claude 4 series
	anthropicModels := anthropic.ListSupportedModels()
	assert.Contains(t, anthropicModels, "claude-4-opus")
	assert.Contains(t, anthropicModels, "claude-4-sonnet")
	assert.Contains(t, anthropicModels, "claude-3.7-sonnet")

	// Google model tests removed - pending post-MVP implementation

	// Ollama - should have latest models
	ollamaModels := ollama.ListSupportedModels()
	assert.Contains(t, ollamaModels, "llama3.3:70b")
	assert.Contains(t, ollamaModels, "qwen3:72b")
	assert.Contains(t, ollamaModels, "phi4:14b")
	assert.Contains(t, ollamaModels, "deepseek-r1:70b")
}