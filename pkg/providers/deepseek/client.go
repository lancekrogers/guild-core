package deepseek

import (
	"context"
	"os"

	"github.com/guild-ventures/guild-core/pkg/providers/base"
	"github.com/guild-ventures/guild-core/pkg/providers/interfaces"
)

// DeepSeek models
const (
	DeepSeekChat     = "deepseek-chat"     // V3, general purpose
	DeepSeekReasoner = "deepseek-reasoner" // R1, chain-of-thought reasoning
)

// Client implements the AIProvider interface for DeepSeek
type Client struct {
	*base.OpenAICompatibleProvider
}

// NewClient creates a new DeepSeek client
func NewClient(apiKey string) *Client {
	if apiKey == "" {
		apiKey = os.Getenv("DEEPSEEK_API_KEY")
	}

	// Model mappings for OpenAI compatibility
	modelMap := map[string]string{
		"gpt-4":       DeepSeekChat,
		"gpt-4-turbo": DeepSeekReasoner,
		"gpt-3.5-turbo": DeepSeekChat,
	}

	capabilities := interfaces.ProviderCapabilities{
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
				InputCost:     0.07,  // Cached price
				OutputCost:    1.10,  // Per million tokens
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
	}

	provider := base.NewOpenAICompatibleProvider(
		"deepseek",
		apiKey,
		"https://api.deepseek.com/v1",
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
		return DeepSeekChat // Good for coding tasks
	case "reasoning":
		return DeepSeekReasoner // Advanced reasoning
	case "cost-efficient":
		return DeepSeekChat // Very cost-efficient
	default:
		return DeepSeekChat // General purpose
	}
}

// Legacy LLMClient interface support
func (c *Client) Complete(ctx context.Context, prompt string) (string, error) {
	req := interfaces.ChatRequest{
		Model: DeepSeekChat, // Default model
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

// Note: DeepSeek offers special pricing features:
// - 50-75% off-peak discount (16:30-00:30 UTC)
// - Cache hit pricing for repeated content
// This implementation doesn't track these features automatically,
// but they apply when using the API during off-peak hours.