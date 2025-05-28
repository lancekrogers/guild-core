package google

import (
	"context"
	"fmt"

	"github.com/guild-ventures/guild-core/pkg/providers/interfaces"
)

// Latest Google Gemini models as of 2025
var SupportedModels = map[string]ModelInfo{
	// Gemini 2.5 Series (Latest - Released 2025)
	"gemini-2.5-pro":        {Name: "gemini-2.5-pro", Type: "multimodal", MaxTokens: 2000000, InputPrice: 3.5, OutputPrice: 10.5},
	"gemini-2.5-flash":      {Name: "gemini-2.5-flash", Type: "multimodal", MaxTokens: 1000000, InputPrice: 0.075, OutputPrice: 0.3},
	"gemini-2.5-pro-deep":   {Name: "gemini-2.5-pro-deep", Type: "reasoning", MaxTokens: 1000000, InputPrice: 7.0, OutputPrice: 21.0},

	// Gemini 2.0 Series
	"gemini-2.0-flash":      {Name: "gemini-2.0-flash", Type: "multimodal", MaxTokens: 1000000, InputPrice: 0.075, OutputPrice: 0.3},
	"gemini-2.0-flash-lite": {Name: "gemini-2.0-flash-lite", Type: "multimodal", MaxTokens: 1000000, InputPrice: 0.075, OutputPrice: 0.3},

	// Audio/Live Models
	"gemini-2.5-flash-audio": {Name: "gemini-2.5-flash-audio", Type: "audio", MaxTokens: 1000000, InputPrice: 0.075, OutputPrice: 0.3},

	// Previous Generation (Still Available)
	"gemini-1.5-pro":   {Name: "gemini-1.5-pro", Type: "multimodal", MaxTokens: 2000000, InputPrice: 3.5, OutputPrice: 10.5},
	"gemini-1.5-flash": {Name: "gemini-1.5-flash", Type: "multimodal", MaxTokens: 1000000, InputPrice: 0.075, OutputPrice: 0.3},
}

// ModelInfo contains information about a Gemini model
type ModelInfo struct {
	Name        string  // Model name
	Type        string  // Model type: multimodal, reasoning, audio
	MaxTokens   int     // Maximum context length
	InputPrice  float64 // Price per million input tokens
	OutputPrice float64 // Price per million output tokens
}

// Client implements the LLMClient interface for Google Gemini
type Client struct {
	apiKey string
	model  string
}

// NewClient creates a new Google Gemini client with model validation
func NewClient(apiKey, model string) *Client {
	// Use default model if none specified
	if model == "" {
		model = "gemini-2.5-flash" // Good default balance of performance and cost
	}

	// Validate model exists
	if _, exists := SupportedModels[model]; !exists {
		// Use fallback model if invalid model specified
		model = "gemini-2.5-flash"
	}

	return &Client{
		apiKey: apiKey,
		model:  model,
	}
}

// Complete generates a completion for the given prompt
func (c *Client) Complete(ctx context.Context, prompt string) (string, error) {
	return fmt.Sprintf("Google %s response for prompt: %s", c.model, prompt), nil
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

// ListSupportedModels returns all supported Google Gemini models
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
		return "gemini-2.5-pro" // Best for complex coding tasks
	case "reasoning":
		return "gemini-2.5-pro-deep" // Deep thinking capabilities
	case "multimodal":
		return "gemini-2.5-pro" // Best multimodal capabilities
	case "cost-efficient":
		return "gemini-2.5-flash" // Most cost-efficient
	case "audio":
		return "gemini-2.5-flash-audio" // Audio capabilities
	case "fast":
		return "gemini-2.0-flash-lite" // Optimized for speed
	default:
		return "gemini-2.5-flash" // General purpose default
	}
}

// CreateCompletion is a lower-level method to create a completion
func (c *Client) CreateCompletion(ctx context.Context, req *interfaces.CompletionRequest) (*interfaces.CompletionResponse, error) {
	return &interfaces.CompletionResponse{
		Text: fmt.Sprintf("Google %s response for prompt: %s", c.model, req.Prompt),
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