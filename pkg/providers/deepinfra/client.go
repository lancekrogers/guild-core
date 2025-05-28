package deepinfra

import (
	"context"
	"os"

	"github.com/guild-ventures/guild-core/pkg/providers/base"
	"github.com/guild-ventures/guild-core/pkg/providers/interfaces"
)

// Popular DeepInfra models
const (
	// Llama Models
	Llama32_8B  = "meta-llama/Meta-Llama-3.2-8B-Instruct"
	Llama33_70B = "meta-llama/Meta-Llama-3.3-70B-Instruct"
	
	// Mistral Models
	Mistral7B = "mistralai/Mistral-7B-Instruct-v0.3"
	Mixtral8x7B = "mistralai/Mixtral-8x7B-Instruct-v0.1"
	
	// Other Models
	Qwen25_72B = "Qwen/Qwen2.5-72B-Instruct"
	Gemma2_9B = "google/gemma-2-9b-it"
)

// Client implements the AIProvider interface for DeepInfra
type Client struct {
	*base.OpenAICompatibleProvider
}

// NewClient creates a new DeepInfra client
func NewClient(apiKey string) *Client {
	if apiKey == "" {
		apiKey = os.Getenv("DEEPINFRA_TOKEN")
	}

	// Model mappings for OpenAI compatibility
	modelMap := map[string]string{
		"gpt-3.5-turbo": Llama32_8B,
		"gpt-4":         Llama33_70B,
		"gpt-4-turbo":   Mixtral8x7B,
	}

	capabilities := interfaces.ProviderCapabilities{
		MaxTokens:      131072, // Varies by model
		ContextWindow:  131072,
		SupportsVision: false,
		SupportsTools:  true,
		SupportsStream: true,
		Models: []interfaces.ModelInfo{
			{
				ID:            Llama32_8B,
				Name:          "Llama 3.2 8B",
				ContextWindow: 131072,
				MaxOutput:     8192,
				InputCost:     0.06,
				OutputCost:    0.06,
			},
			{
				ID:            Llama33_70B,
				Name:          "Llama 3.3 70B",
				ContextWindow: 131072,
				MaxOutput:     8192,
				InputCost:     0.59,
				OutputCost:    0.79,
			},
			{
				ID:            Mistral7B,
				Name:          "Mistral 7B v0.3",
				ContextWindow: 32768,
				MaxOutput:     8192,
				InputCost:     0.06,
				OutputCost:    0.06,
			},
			{
				ID:            Mixtral8x7B,
				Name:          "Mixtral 8x7B",
				ContextWindow: 32768,
				MaxOutput:     8192,
				InputCost:     0.27,
				OutputCost:    0.27,
			},
			{
				ID:            Qwen25_72B,
				Name:          "Qwen 2.5 72B",
				ContextWindow: 32768,
				MaxOutput:     8192,
				InputCost:     0.59,
				OutputCost:    0.79,
			},
			{
				ID:            Gemma2_9B,
				Name:          "Gemma 2 9B",
				ContextWindow: 8192,
				MaxOutput:     8192,
				InputCost:     0.09,
				OutputCost:    0.09,
			},
		},
	}

	provider := base.NewOpenAICompatibleProvider(
		"deepinfra",
		apiKey,
		"https://api.deepinfra.com/v1/openai",
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
		return Qwen25_72B // Good for coding
	case "reasoning":
		return Llama33_70B // Strong reasoning
	case "cost-efficient":
		return Llama32_8B // Most affordable
	case "fast":
		return Mistral7B // Smallest and fastest
	default:
		return Llama33_70B // General purpose
	}
}

// Legacy LLMClient interface support
func (c *Client) Complete(ctx context.Context, prompt string) (string, error) {
	req := interfaces.ChatRequest{
		Model: Llama33_70B, // Default model
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