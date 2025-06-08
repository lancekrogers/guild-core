package providers

import (
	"context"

	"github.com/guild-ventures/guild-core/pkg/providers/interfaces"
)

// LLMClient defines the interface for LLM clients (legacy - will be deprecated)
type LLMClient interface {
	Complete(ctx context.Context, prompt string) (string, error)
}

// AIProvider is an alias for the new universal provider interface
type AIProvider = interfaces.AIProvider

// Re-export all interfaces types for convenience
type (
	ChatRequest          = interfaces.ChatRequest
	ChatResponse         = interfaces.ChatResponse
	ChatMessage          = interfaces.ChatMessage
	ChatChoice           = interfaces.ChatChoice
	ChatStream           = interfaces.ChatStream
	ChatStreamChunk      = interfaces.ChatStreamChunk
	EmbeddingRequest     = interfaces.EmbeddingRequest
	EmbeddingResponse    = interfaces.EmbeddingResponse
	Embedding            = interfaces.Embedding
	ProviderCapabilities = interfaces.ProviderCapabilities
	ModelInfo            = interfaces.ModelInfo
	ProviderError        = interfaces.ProviderError
	StreamHandler        = interfaces.StreamHandler
)

// ProviderType is an alias for interfaces.ProviderType
type ProviderType = interfaces.ProviderType

const (
	ProviderOpenAI     = interfaces.ProviderOpenAI
	ProviderAnthropic  = interfaces.ProviderAnthropic
	ProviderOllama     = interfaces.ProviderOllama
	ProviderGoogle     = interfaces.ProviderGoogle
	ProviderClaudeCode = interfaces.ProviderClaudeCode
	ProviderDeepSeek   = interfaces.ProviderDeepSeek
	ProviderDeepInfra  = interfaces.ProviderDeepInfra
	ProviderOra        = interfaces.ProviderOra
)
