package providers

import (
	"fmt"
	"os"
	
	"github.com/guild-ventures/guild-core/pkg/providers/anthropic"
	"github.com/guild-ventures/guild-core/pkg/providers/claudecode"
	"github.com/guild-ventures/guild-core/pkg/providers/google"
	"github.com/guild-ventures/guild-core/pkg/providers/ollama"
	"github.com/guild-ventures/guild-core/pkg/providers/openai"
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
	case ProviderGoogle:
		return google.NewClient(apiKey, model), nil
	case ProviderClaudeCode:
		// For Claude Code, apiKey is used as binary path, model is configuration type
		return claudecode.NewClient(apiKey, model), nil
	default:
		return nil, fmt.Errorf("unsupported provider type: %s", providerType)
	}
}

// CreateClientFromConfig creates a client from configuration map
func (f *Factory) CreateClientFromConfig(providerType ProviderType, config map[string]interface{}) (LLMClient, error) {
	// Extract model
	model, ok := config["model"].(string)
	if !ok {
		model = "" // Will use provider defaults
	}

	// Extract API key - try config first, then environment variable
	var apiKey string
	if key, exists := config["api_key"]; exists {
		if keyStr, ok := key.(string); ok {
			apiKey = keyStr
		}
	}
	
	// If no direct API key, try environment variable reference
	if apiKey == "" {
		if envVar, exists := config["api_key_env"]; exists {
			if envVarStr, ok := envVar.(string); ok {
				apiKey = os.Getenv(envVarStr)
			}
		}
	}

	// For Ollama, API key might be optional
	if providerType == ProviderOllama && apiKey == "" {
		apiKey = "" // Ollama might not need API key for local usage
	}

	return f.CreateClient(providerType, apiKey, model)
}

// RegisterProvidersWithRegistry registers all available providers with a ProviderRegistry
func (f *Factory) RegisterProvidersWithRegistry(registry ProviderRegistry, providersConfig map[string]interface{}) error {
	for providerName, providerConfigRaw := range providersConfig {
		// Convert provider name to ProviderType
		var providerType ProviderType
		switch providerName {
		case "openai":
			providerType = ProviderOpenAI
		case "anthropic":
			providerType = ProviderAnthropic
		case "ollama":
			providerType = ProviderOllama
		case "google":
			providerType = ProviderGoogle
		case "claudecode":
			providerType = ProviderClaudeCode
		default:
			return fmt.Errorf("unknown provider type: %s", providerName)
		}

		// Extract provider config
		providerConfig, ok := providerConfigRaw.(map[string]interface{})
		if !ok {
			return fmt.Errorf("invalid config for provider %s", providerName)
		}

		// Create client
		client, err := f.CreateClientFromConfig(providerType, providerConfig)
		if err != nil {
			return fmt.Errorf("failed to create client for provider %s: %w", providerName, err)
		}

		// Register with registry
		if err := registry.RegisterProvider(providerName, client); err != nil {
			return fmt.Errorf("failed to register provider %s: %w", providerName, err)
		}
	}

	return nil
}

// ProviderRegistry is the interface we expect (matches the one in registry package)
type ProviderRegistry interface {
	RegisterProvider(name string, provider LLMClient) error
	SetDefaultProvider(name string) error
}