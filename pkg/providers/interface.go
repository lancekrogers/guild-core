package providers

import (
	"github.com/blockhead-consulting/guild/pkg/providers/interfaces"
)

// Re-export interfaces and types
type CompletionRequest = interfaces.CompletionRequest
type CompletionResponse = interfaces.CompletionResponse
type EmbeddingRequest = interfaces.EmbeddingRequest
type EmbeddingResponse = interfaces.EmbeddingResponse
type LLMClient = interfaces.LLMClient
type ProviderType = interfaces.ProviderType

// Re-export constants
const (
	ProviderOpenAI    = interfaces.ProviderOpenAI
	ProviderAnthropic = interfaces.ProviderAnthropic
	ProviderOllama    = interfaces.ProviderOllama
	ProviderMock      = ProviderType("mock")
)