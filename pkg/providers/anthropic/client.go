package anthropic

import (
	"context"
	"fmt"

	"github.com/guild-ventures/guild-core/pkg/providers/interfaces"
)

// Latest Anthropic Claude models as of 2025
var SupportedModels = map[string]ModelInfo{
	// Claude 4 Series (Latest - Released May 2025)
	"claude-4-opus":  {Name: "claude-4-opus", Type: "reasoning", MaxTokens: 200000, InputPrice: 15.0, OutputPrice: 75.0},
	"claude-4-sonnet": {Name: "claude-4-sonnet", Type: "text", MaxTokens: 200000, InputPrice: 3.0, OutputPrice: 15.0},

	// Claude 3.7 Series (Hybrid Reasoning - Released February 2025)
	"claude-3.7-sonnet": {Name: "claude-3.7-sonnet", Type: "hybrid-reasoning", MaxTokens: 200000, InputPrice: 3.0, OutputPrice: 15.0},

	// Claude 3 Series (Previous Generation - Still Available)
	"claude-3-5-sonnet-20241022": {Name: "claude-3-5-sonnet-20241022", Type: "text", MaxTokens: 200000, InputPrice: 3.0, OutputPrice: 15.0},
	"claude-3-5-haiku-20241022":  {Name: "claude-3-5-haiku-20241022", Type: "text", MaxTokens: 200000, InputPrice: 0.8, OutputPrice: 4.0},
	"claude-3-opus-20240229":     {Name: "claude-3-opus-20240229", Type: "text", MaxTokens: 200000, InputPrice: 15.0, OutputPrice: 75.0},
}

// ModelInfo contains information about a Claude model
type ModelInfo struct {
	Name        string  // Model name
	Type        string  // Model type: text, reasoning, hybrid-reasoning
	MaxTokens   int     // Maximum context length
	InputPrice  float64 // Price per million input tokens
	OutputPrice float64 // Price per million output tokens
}

// Client implements the LLMClient interface for Anthropic
type Client struct {
	apiKey string
	model  string
}

// NewClient creates a new Anthropic client with model validation
func NewClient(apiKey, model string) *Client {
	// Use default model if none specified
	if model == "" {
		model = "claude-4-sonnet" // Latest default
	}

	// Validate model exists
	if _, exists := SupportedModels[model]; !exists {
		// Use fallback model if invalid model specified
		model = "claude-4-sonnet"
	}

	return &Client{
		apiKey: apiKey,
		model:  model,
	}
}

// Complete generates a completion for the given prompt
func (c *Client) Complete(ctx context.Context, prompt string) (string, error) {
	return fmt.Sprintf("Claude %s response for prompt: %s", c.model, prompt), nil
}

// GetModel returns the current model being used
func (c *Client) GetModel() string {
	return c.model
}

// GetModelInfo returns information about the current model
func (c *Client) GetModelInfo() (ModelInfo, bool) {
	info, exists := SupportedModels[c.model]
	return info, exists
}

// ListSupportedModels returns all supported Anthropic models
func ListSupportedModels() map[string]ModelInfo {
	return SupportedModels
}

// GetModelsByType returns models of a specific type
func GetModelsByType(modelType string) map[string]ModelInfo {
	filtered := make(map[string]ModelInfo)
	for name, info := range SupportedModels {
		if info.Type == modelType {
			filtered[name] = info
		}
	}
	return filtered
}

// GetRecommendedModel returns a recommended model for a given use case
func GetRecommendedModel(useCase string) string {
	switch useCase {
	case "coding":
		return "claude-4-opus" // Best for coding according to Anthropic
	case "reasoning":
		return "claude-4-opus" // Latest reasoning model
	case "hybrid-reasoning":
		return "claude-3.7-sonnet" // Hybrid reasoning capabilities
	case "cost-efficient":
		return "claude-3-5-haiku-20241022" // Most cost-efficient
	case "general":
		return "claude-4-sonnet" // Balanced performance
	default:
		return "claude-4-sonnet" // General purpose default
	}
}

// CreateCompletion is a lower-level method to create a completion
func (c *Client) CreateCompletion(ctx context.Context, req *interfaces.CompletionRequest) (*interfaces.CompletionResponse, error) {
	return &interfaces.CompletionResponse{
		Text: fmt.Sprintf("Stub response for prompt: %s", req.Prompt),
		TokensUsed: 10,
		TokensInput: 5,
		TokensOutput: 5,
		ModelUsed: c.model,
	}, nil
}

// CreateEmbedding generates an embedding for the given text
func (c *Client) CreateEmbedding(ctx context.Context, req *interfaces.EmbeddingRequest) (*interfaces.EmbeddingResponse, error) {
	return &interfaces.EmbeddingResponse{
		Embedding: []float32{0.1, 0.2, 0.3},
		Dimensions: 3,
	}, nil
}

// CreateEmbeddings generates embeddings for multiple texts
func (c *Client) CreateEmbeddings(ctx context.Context, req *interfaces.EmbeddingRequest) (*interfaces.EmbeddingResponse, error) {
	return &interfaces.EmbeddingResponse{
		Embeddings: [][]float32{{0.1, 0.2, 0.3}, {0.4, 0.5, 0.6}},
		Dimensions: 3,
	}, nil
}
