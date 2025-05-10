package interfaces

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

	// --- Embedding methods ---

	// CreateEmbedding creates an embedding for the given text
	CreateEmbedding(ctx context.Context, req *EmbeddingRequest) (*EmbeddingResponse, error)

	// CreateEmbeddings creates embeddings for multiple texts
	CreateEmbeddings(ctx context.Context, req *EmbeddingRequest) (*EmbeddingResponse, error)

	// GetEmbeddingDimension returns the dimension of embeddings from this provider
	GetEmbeddingDimension(model string) int
}

// EmbeddingRequest represents a request to create embeddings
type EmbeddingRequest struct {
	// Text is the text to embed (for single text requests)
	Text string `json:"text,omitempty"`

	// Texts is an array of texts to embed (for batch requests)
	Texts []string `json:"texts,omitempty"`

	// Model is the embedding model to use
	Model string `json:"model,omitempty"`

	// Dimensions is the desired embedding dimensions (if supported)
	Dimensions int `json:"dimensions,omitempty"`
}

// EmbeddingResponse represents a response from an embedding request
type EmbeddingResponse struct {
	// Embedding is the vector for a single text
	Embedding []float32 `json:"embedding,omitempty"`

	// Embeddings is an array of vectors for multiple texts
	Embeddings [][]float32 `json:"embeddings,omitempty"`

	// Dimensions is the size of each embedding vector
	Dimensions int `json:"dimensions"`

	// Model is the model used to create the embedding
	Model string `json:"model"`

	// TokensUsed is the total number of tokens used
	TokensUsed int `json:"tokens_used,omitempty"`
}

// ProviderType represents a type of LLM provider
type ProviderType string

const (
	// ProviderOpenAI represents the OpenAI provider
	ProviderOpenAI ProviderType = "openai"
	
	// ProviderAnthropic represents the Anthropic provider
	ProviderAnthropic ProviderType = "anthropic"
	
	// ProviderOllama represents the Ollama provider
	ProviderOllama ProviderType = "ollama"
)