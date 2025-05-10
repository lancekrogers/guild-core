package providers

import (
	"context"
)

// CompletionRequest represents a request to complete text using an LLM
type CompletionRequest struct {
	Prompt      string            `json:"prompt"`
	MaxTokens   int               `json:"max_tokens,omitempty"`
	Temperature float64           `json:"temperature,omitempty"`
	StopTokens  []string          `json:"stop_tokens,omitempty"`
	Options     map[string]string `json:"options,omitempty"`
}

// CompletionResponse represents a response from an LLM completion request
type CompletionResponse struct {
	Text         string `json:"text"`
	TokensUsed   int    `json:"tokens_used,omitempty"`
	TokensInput  int    `json:"tokens_input,omitempty"`
	TokensOutput int    `json:"tokens_output,omitempty"`
	FinishReason string `json:"finish_reason,omitempty"`
	ModelUsed    string `json:"model_used,omitempty"`
}

// LLMClient defines the interface for interacting with language models
type LLMClient interface {
	// Complete sends a completion request to the LLM and returns the response
	Complete(ctx context.Context, req *CompletionRequest) (*CompletionResponse, error)

	// GetName returns the name of the provider
	GetName() string

	// GetModelInfo returns information about the model being used
	GetModelInfo() map[string]string

	// GetModelList returns a list of available models from this provider
	GetModelList(ctx context.Context) ([]string, error)

	// GetMaxTokens returns the maximum tokens supported by the current model
	GetMaxTokens() int
}

// EmbeddingRequest represents a request to create embeddings
type EmbeddingRequest struct {
	Text       string `json:"text"`
	Model      string `json:"model,omitempty"`
	Dimensions int    `json:"dimensions,omitempty"`
}

// EmbeddingResponse represents a response from an embedding request
type EmbeddingResponse struct {
	Embedding  []float32 `json:"embedding"`
	Dimensions int       `json:"dimensions"`
	Model      string    `json:"model"`
}

// EmbeddingClient defines the interface for creating embeddings
type EmbeddingClient interface {
	// CreateEmbedding creates an embedding for the given text
	CreateEmbedding(ctx context.Context, req *EmbeddingRequest) (*EmbeddingResponse, error)

	// BatchCreateEmbeddings creates embeddings for multiple texts
	BatchCreateEmbeddings(ctx context.Context, texts []string) ([]*EmbeddingResponse, error)
}
