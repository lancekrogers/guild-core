package manager

import (
	"context"

	"github.com/guild-ventures/guild-core/pkg/gerror"
	"github.com/guild-ventures/guild-core/pkg/providers"
)

// GuildArtisanClient implements ArtisanClient interface using Guild's AIProvider
type GuildArtisanClient struct {
	provider providers.AIProvider
	model    string
}

// NewGuildArtisanClient creates a new Guild Artisan client with an AI provider
func NewGuildArtisanClient(provider providers.AIProvider, model string) *GuildArtisanClient {
	return &GuildArtisanClient{
		provider: provider,
		model:    model,
	}
}

// Complete implements the ArtisanClient interface for Guild Artisan interactions
func (gac *GuildArtisanClient) Complete(ctx context.Context, request ArtisanRequest) (*ArtisanResponse, error) {
	// Convert ArtisanRequest to ChatRequest
	chatRequest := providers.ChatRequest{
		Model:       gac.model,
		Temperature: float64(request.Temperature),
		MaxTokens:   request.MaxTokens,
		Messages: []providers.ChatMessage{
			{
				Role:    "system",
				Content: request.SystemPrompt,
			},
			{
				Role:    "user",
				Content: request.UserPrompt,
			},
		},
	}

	// Call the AI provider
	response, err := gac.provider.ChatCompletion(ctx, chatRequest)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeAgent, "Guild Artisan request failed").
			WithComponent("manager").
			WithOperation("Complete").
			WithDetails("model", gac.model).
			WithDetails("temperature", request.Temperature)
	}

	// Validate response has choices
	if len(response.Choices) == 0 {
		return nil, gerror.New(gerror.ErrCodeAgent, "Guild Artisan returned no response choices", nil).
			WithComponent("manager").
			WithOperation("Complete").
			WithDetails("model", gac.model).
			WithDetails("response_id", response.ID)
	}

	// Extract the content from the first choice
	content := response.Choices[0].Message.Content

	// Create Guild Artisan response with metadata
	artisanResponse := &ArtisanResponse{
		Content: content,
		Metadata: map[string]interface{}{
			"model":         response.Model,
			"response_id":   response.ID,
			"finish_reason": response.FinishReason,
			"usage": map[string]interface{}{
				"prompt_tokens":     response.Usage.PromptTokens,
				"completion_tokens": response.Usage.CompletionTokens,
				"total_tokens":      response.Usage.TotalTokens,
			},
			"artisan_type": "guild_master",
			"provider":     "unknown", // TODO: Add ProviderType to capabilities
		},
	}

	return artisanResponse, nil
}

// GetModel returns the model being used by this Artisan client
func (gac *GuildArtisanClient) GetModel() string {
	return gac.model
}

// GetCapabilities returns the capabilities of the underlying AI provider
func (gac *GuildArtisanClient) GetCapabilities() providers.ProviderCapabilities {
	return gac.provider.GetCapabilities()
}
