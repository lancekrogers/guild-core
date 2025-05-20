package providers

import (
	"context"
	
	"github.com/guild-ventures/guild-core/pkg/providers/interfaces"
)

// LLMClient defines the interface for LLM clients
type LLMClient interface {
	Complete(ctx context.Context, prompt string) (string, error)
}

// ProviderType is an alias for interfaces.ProviderType
type ProviderType = interfaces.ProviderType

const (
	ProviderOpenAI    = interfaces.ProviderOpenAI
	ProviderAnthropic = interfaces.ProviderAnthropic
	ProviderOllama    = interfaces.ProviderOllama
)
