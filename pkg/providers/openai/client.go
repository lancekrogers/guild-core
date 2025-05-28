package openai

import (
	"context"
	"os"

	"github.com/guild-ventures/guild-core/pkg/providers/base"
	"github.com/guild-ventures/guild-core/pkg/providers/interfaces"
)

// Latest OpenAI models as of May 2025
const (
	// GPT-4.1 Series (1M context)
	GPT41     = "gpt-4.1"      // $10/$30 per million tokens
	GPT41Mini = "gpt-4.1-mini" // $1/$4 per million tokens
	GPT41Nano = "gpt-4.1-nano" // $0.25/$1 per million tokens

	// GPT-4o Series (Multimodal)
	GPT4o     = "gpt-4o"      // 128K context, multimodal
	GPT4oMini = "gpt-4o-mini" // 128K context, cost-efficient

	// O3 Series (Advanced Reasoning)
	O3     = "o3"      // 200K context, $200/$800 per million tokens
	O3Mini = "o3-mini" // 200K context, reasoning

	// Embedding Models
	TextEmbedding3Small = "text-embedding-3-small"
	TextEmbedding3Large = "text-embedding-3-large"
)

// Client implements the AIProvider interface for OpenAI
type Client struct {
	*base.OpenAICompatibleProvider
}

// NewClient creates a new OpenAI client
func NewClient(apiKey string) *Client {
	if apiKey == "" {
		apiKey = os.Getenv("OPENAI_API_KEY")
	}

	capabilities := interfaces.ProviderCapabilities{
		MaxTokens:      1000000,
		ContextWindow:  1000000,
		SupportsVision: true,
		SupportsTools:  true,
		SupportsStream: true,
		Models: []interfaces.ModelInfo{
			{
				ID:            GPT41,
				Name:          "GPT-4.1",
				ContextWindow: 1000000,
				MaxOutput:     32768,
				InputCost:     10.0,
				OutputCost:    30.0,
			},
			{
				ID:            GPT41Mini,
				Name:          "GPT-4.1 Mini",
				ContextWindow: 1000000,
				MaxOutput:     32768,
				InputCost:     1.0,
				OutputCost:    4.0,
			},
			{
				ID:            GPT41Nano,
				Name:          "GPT-4.1 Nano",
				ContextWindow: 1000000,
				MaxOutput:     32768,
				InputCost:     0.25,
				OutputCost:    1.0,
			},
			{
				ID:            GPT4o,
				Name:          "GPT-4o",
				ContextWindow: 128000,
				MaxOutput:     16384,
				InputCost:     2.5,
				OutputCost:    10.0,
			},
			{
				ID:            GPT4oMini,
				Name:          "GPT-4o Mini",
				ContextWindow: 128000,
				MaxOutput:     16384,
				InputCost:     0.15,
				OutputCost:    0.6,
			},
			{
				ID:            O3,
				Name:          "O3",
				ContextWindow: 200000,
				MaxOutput:     32768,
				InputCost:     200.0,
				OutputCost:    800.0,
			},
			{
				ID:            O3Mini,
				Name:          "O3 Mini",
				ContextWindow: 200000,
				MaxOutput:     32768,
				InputCost:     50.0,
				OutputCost:    200.0,
			},
		},
	}

	provider := base.NewOpenAICompatibleProvider(
		"openai",
		apiKey,
		"https://api.openai.com/v1",
		nil, // No model mapping needed for OpenAI
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
		return GPT41 // Best for coding as of 2025
	case "reasoning":
		return O3Mini // Advanced reasoning
	case "multimodal":
		return GPT4o // Best multimodal
	case "cost-efficient":
		return GPT41Nano // Most cost-efficient
	case "general":
		return GPT41Mini // Balanced price/performance
	default:
		return GPT41Mini // Default
	}
}

// Legacy LLMClient interface support
func (c *Client) Complete(ctx context.Context, prompt string) (string, error) {
	req := interfaces.ChatRequest{
		Model: GPT41Mini, // Default model
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