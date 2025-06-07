package interfaces

import (
	"context"
	"io"
)

// AIProvider defines the universal provider interface for all AI providers
type AIProvider interface {
	ChatCompletion(ctx context.Context, req ChatRequest) (*ChatResponse, error)
	StreamChatCompletion(ctx context.Context, req ChatRequest) (ChatStream, error)
	CreateEmbedding(ctx context.Context, req EmbeddingRequest) (*EmbeddingResponse, error)
	GetCapabilities() ProviderCapabilities
}

// ChatRequest represents a chat completion request
type ChatRequest struct {
	Model       string        `json:"model"`
	Messages    []ChatMessage `json:"messages"`
	MaxTokens   int           `json:"max_tokens,omitempty"`
	Temperature float64       `json:"temperature,omitempty"`
	TopP        float64       `json:"top_p,omitempty"`
	Stream      bool          `json:"stream,omitempty"`
	Stop        []string      `json:"stop,omitempty"`
	// Provider-specific options
	Options map[string]interface{} `json:"options,omitempty"`
}

// ChatMessage represents a message in a chat conversation
type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
	// For multimodal support
	Images []string `json:"images,omitempty"`
}

// ChatResponse represents a chat completion response
type ChatResponse struct {
	ID           string       `json:"id"`
	Model        string       `json:"model"`
	Choices      []ChatChoice `json:"choices"`
	Usage        UsageInfo    `json:"usage"`
	FinishReason string       `json:"finish_reason"`
}

// ChatChoice represents a single choice in a chat response
type ChatChoice struct {
	Index        int         `json:"index"`
	Message      ChatMessage `json:"message"`
	FinishReason string      `json:"finish_reason"`
}

// ChatStream represents a streaming chat response
type ChatStream interface {
	Next() (ChatStreamChunk, error)
	Close() error
}

// ChatStreamChunk represents a chunk of streaming data
type ChatStreamChunk struct {
	Delta        ChatMessage `json:"delta"`
	FinishReason string      `json:"finish_reason,omitempty"`
}

// EmbeddingRequest represents an embedding request
type EmbeddingRequest struct {
	Model string   `json:"model"`
	Input []string `json:"input"`
}

// EmbeddingResponse represents an embedding response
type EmbeddingResponse struct {
	Model      string      `json:"model"`
	Embeddings []Embedding `json:"data"`
	Usage      UsageInfo   `json:"usage"`
}

// Embedding represents a single embedding
type Embedding struct {
	Index     int       `json:"index"`
	Embedding []float64 `json:"embedding"`
}

// ProviderCapabilities describes what a provider supports
type ProviderCapabilities struct {
	MaxTokens          int
	ContextWindow      int
	SupportsVision     bool
	SupportsTools      bool
	SupportsStream     bool
	SupportsEmbeddings bool
	Models             []ModelInfo
}

// ModelInfo contains information about a specific model
type ModelInfo struct {
	ID            string
	Name          string
	ContextWindow int
	MaxOutput     int
	InputCost     float64 // Cost per million tokens
	OutputCost    float64 // Cost per million tokens
}

// StreamHandler is a function that handles streaming chunks
type StreamHandler func(chunk ChatStreamChunk) error

// ProviderError represents a provider-specific error
type ProviderError struct {
	Provider   string
	StatusCode int
	Type       string
	Message    string
	Retryable  bool
}

func (e *ProviderError) Error() string {
	return e.Message
}

// Check if error is retryable
func IsRetryable(err error) bool {
	if perr, ok := err.(*ProviderError); ok {
		return perr.Retryable
	}
	return false
}

// Common error types
const (
	ErrorTypeAuth       = "auth_error"
	ErrorTypeRateLimit  = "rate_limit"
	ErrorTypeServer     = "server_error"
	ErrorTypeValidation = "validation_error"
	ErrorTypeUnknown    = "unknown"
)

// Check for EOF in streaming
func IsStreamEnd(err error) bool {
	return err == io.EOF
}
