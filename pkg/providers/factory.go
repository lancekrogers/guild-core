package providers

import (
	"fmt"
	
	"github.com/blockhead-consulting/guild/pkg/providers/anthropic"
	"github.com/blockhead-consulting/guild/pkg/providers/ollama"
	"github.com/blockhead-consulting/guild/pkg/providers/openai"
)

// Factory creates LLM clients
type Factory struct {
	// Configuration fields would go here
}

// NewFactory creates a new factory
func NewFactory() *Factory {
	return &Factory{}
}

// CreateClient creates a new LLM client based on the provider type
func (f *Factory) CreateClient(providerType ProviderType, apiKey string, model string) (LLMClient, error) {
	switch providerType {
	case ProviderOpenAI:
		return openai.NewClient(apiKey, model), nil
	case ProviderAnthropic:
		return anthropic.NewClient(apiKey, model), nil
	case ProviderOllama:
		return ollama.NewClient(apiKey, model), nil
	default:
		return nil, fmt.Errorf("unsupported provider type: %s", providerType)
	}
}