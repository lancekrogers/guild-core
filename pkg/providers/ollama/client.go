package ollama

import (
	"context"
	"fmt"

	"github.com/guild-ventures/guild-core/pkg/providers/interfaces"
)

// Latest Ollama supported models as of 2025
var SupportedModels = map[string]ModelInfo{
	// Latest Flagship Models (2025)
	"llama3.3:70b":    {Name: "llama3.3:70b", Type: "text", Size: "70B", MinRAM: 32, Category: "large"},
	"qwen3:72b":       {Name: "qwen3:72b", Type: "text", Size: "72B", MinRAM: 32, Category: "large"},
	"deepseek-r1:70b": {Name: "deepseek-r1:70b", Type: "reasoning", Size: "70B", MinRAM: 32, Category: "reasoning"},
	"phi4:14b":        {Name: "phi4:14b", Type: "text", Size: "14B", MinRAM: 16, Category: "medium"},

	// Vision/Multimodal Models
	"llama3.2-vision:11b": {Name: "llama3.2-vision:11b", Type: "multimodal", Size: "11B", MinRAM: 16, Category: "vision"},
	"qwen2-vl:7b":         {Name: "qwen2-vl:7b", Type: "multimodal", Size: "7B", MinRAM: 8, Category: "vision"},

	// Medium Models (Good Performance, Reasonable Requirements)
	"llama3.1:8b":    {Name: "llama3.1:8b", Type: "text", Size: "8B", MinRAM: 8, Category: "medium"},
	"mistral-large": {Name: "mistral-large", Type: "text", Size: "123B", MinRAM: 64, Category: "large"},
	"gemma2:27b":    {Name: "gemma2:27b", Type: "text", Size: "27B", MinRAM: 16, Category: "medium"},
	"gemma2:9b":     {Name: "gemma2:9b", Type: "text", Size: "9B", MinRAM: 8, Category: "medium"},

	// Specialized Models
	"qwen2-math:7b":   {Name: "qwen2-math:7b", Type: "math", Size: "7B", MinRAM: 8, Category: "specialized"},
	"codegemma:7b":    {Name: "codegemma:7b", Type: "code", Size: "7B", MinRAM: 8, Category: "specialized"},
	"phi3-mini:3.8b":  {Name: "phi3-mini:3.8b", Type: "text", Size: "3.8B", MinRAM: 4, Category: "small"},

	// Small/Efficient Models
	"gemma2:2b":      {Name: "gemma2:2b", Type: "text", Size: "2B", MinRAM: 2, Category: "small"},
	"phi4-mini:3.8b": {Name: "phi4-mini:3.8b", Type: "text", Size: "3.8B", MinRAM: 4, Category: "small"},
}

// ModelInfo contains information about an Ollama model
type ModelInfo struct {
	Name     string // Model name as used in Ollama
	Type     string // Model type: text, multimodal, reasoning, code, math
	Size     string // Model size (e.g., "7B", "70B")
	MinRAM   int    // Minimum RAM in GB
	Category string // small, medium, large, specialized, vision, reasoning
}

// Client implements the LLMClient interface for Ollama
type Client struct {
	baseURL string // Ollama server URL
	model   string
}

// NewClient creates a new Ollama client with model validation
func NewClient(apiKey, model string) *Client {
	// Ollama doesn't typically use API keys, but we keep the parameter for interface compatibility
	// apiKey can be used as baseURL if provided, otherwise default to localhost
	baseURL := "http://localhost:11434"
	if apiKey != "" {
		baseURL = apiKey // Use as URL if provided
	}

	// Use default model if none specified
	if model == "" {
		model = "llama3.1:8b" // Good default balance
	}

	// Validate model exists
	if _, exists := SupportedModels[model]; !exists {
		// Use fallback model if invalid model specified
		model = "llama3.1:8b"
	}

	return &Client{
		baseURL: baseURL,
		model:   model,
	}
}

// Complete generates a completion for the given prompt
func (c *Client) Complete(ctx context.Context, prompt string) (string, error) {
	return fmt.Sprintf("Ollama %s response for prompt: %s", c.model, prompt), nil
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

// GetBaseURL returns the Ollama server URL
func (c *Client) GetBaseURL() string {
	return c.baseURL
}

// ListSupportedModels returns all supported Ollama models
func ListSupportedModels() map[string]ModelInfo {
	return SupportedModels
}

// GetModelsByCategory returns models of a specific category
func GetModelsByCategory(category string) map[string]ModelInfo {
	filtered := make(map[string]ModelInfo)
	for name, info := range SupportedModels {
		if info.Category == category {
			filtered[name] = info
		}
	}
	return filtered
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

// GetModelsByRAM returns models that can run with the given RAM amount
func GetModelsByRAM(availableRAM int) map[string]ModelInfo {
	filtered := make(map[string]ModelInfo)
	for name, info := range SupportedModels {
		if info.MinRAM <= availableRAM {
			filtered[name] = info
		}
	}
	return filtered
}

// GetRecommendedModel returns a recommended model for a given use case and available RAM
func GetRecommendedModel(useCase string, availableRAM int) string {
	// Filter by RAM first
	available := GetModelsByRAM(availableRAM)
	if len(available) == 0 {
		return "gemma2:2b" // Smallest model as fallback
	}

	switch useCase {
	case "coding":
		if _, exists := available["codegemma:7b"]; exists {
			return "codegemma:7b"
		}
		if _, exists := available["llama3.1:8b"]; exists {
			return "llama3.1:8b"
		}
	case "math":
		if _, exists := available["qwen2-math:7b"]; exists {
			return "qwen2-math:7b"
		}
	case "reasoning":
		if _, exists := available["deepseek-r1:70b"]; exists && availableRAM >= 32 {
			return "deepseek-r1:70b"
		}
		if _, exists := available["llama3.3:70b"]; exists && availableRAM >= 32 {
			return "llama3.3:70b"
		}
	case "multimodal", "vision":
		if _, exists := available["llama3.2-vision:11b"]; exists {
			return "llama3.2-vision:11b"
		}
		if _, exists := available["qwen2-vl:7b"]; exists {
			return "qwen2-vl:7b"
		}
	case "small", "fast":
		if _, exists := available["phi4-mini:3.8b"]; exists {
			return "phi4-mini:3.8b"
		}
		if _, exists := available["gemma2:2b"]; exists {
			return "gemma2:2b"
		}
	}

	// Default recommendations based on available RAM
	if availableRAM >= 32 {
		if _, exists := available["llama3.3:70b"]; exists {
			return "llama3.3:70b"
		}
	}
	if availableRAM >= 16 {
		if _, exists := available["phi4:14b"]; exists {
			return "phi4:14b"
		}
	}
	if availableRAM >= 8 {
		if _, exists := available["llama3.1:8b"]; exists {
			return "llama3.1:8b"
		}
	}

	return "gemma2:2b" // Fallback for low memory
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
