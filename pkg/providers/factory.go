package providers

import (
	"fmt"
	"os"
	"sync"

	"github.com/blockhead-consulting/guild/pkg/providers/anthropic"
	"github.com/blockhead-consulting/guild/pkg/providers/ollama"
	"github.com/blockhead-consulting/guild/pkg/providers/openai"
)

// Using ProviderType from interfaces package

// ProviderConfig holds configuration for a provider
type ProviderConfig struct {
	Type      ProviderType `json:"type"`
	ApiKey    string       `json:"api_key,omitempty"`
	Model     string       `json:"model,omitempty"`
	ApiURL    string       `json:"api_url,omitempty"`
	MaxTokens int          `json:"max_tokens,omitempty"`
}

// Factory is responsible for creating and managing LLM clients
type Factory struct {
	clients   map[ProviderType]LLMClient
	configs   map[ProviderType]ProviderConfig
	defaultProvider ProviderType
	mu        sync.RWMutex
}

// NewFactory creates a new provider factory
func NewFactory() *Factory {
	return &Factory{
		clients: make(map[ProviderType]LLMClient),
		configs: make(map[ProviderType]ProviderConfig),
		defaultProvider: ProviderOpenAI, // Default to OpenAI
	}
}

// RegisterProvider registers a provider configuration
func (f *Factory) RegisterProvider(config ProviderConfig) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	
	f.configs[config.Type] = config
	return nil
}

// SetDefaultProvider sets the default provider type
func (f *Factory) SetDefaultProvider(providerType ProviderType) {
	f.mu.Lock()
	defer f.mu.Unlock()
	
	f.defaultProvider = providerType
}

// GetClient returns a client for the given provider type, initializing it if necessary
func (f *Factory) GetClient(providerType ProviderType) (LLMClient, error) {
	f.mu.RLock()
	client, exists := f.clients[providerType]
	f.mu.RUnlock()
	
	if exists {
		return client, nil
	}
	
	// Client doesn't exist, create it
	f.mu.Lock()
	defer f.mu.Unlock()
	
	// Check again in case another goroutine created it while we were waiting for the lock
	client, exists = f.clients[providerType]
	if exists {
		return client, nil
	}
	
	// Get config for this provider
	config, exists := f.configs[providerType]
	if !exists {
		// Try to create with environment variables
		var err error
		client, err = f.createClientFromEnv(providerType)
		if err != nil {
			return nil, fmt.Errorf("provider %s not registered and could not be created from environment: %w", providerType, err)
		}
		f.clients[providerType] = client
		return client, nil
	}
	
	// Create client from config
	var err error
	client, err = f.createClient(config)
	if err != nil {
		return nil, err
	}
	
	f.clients[providerType] = client
	return client, nil
}

// GetDefaultClient returns the default LLM client
func (f *Factory) GetDefaultClient() (LLMClient, error) {
	return f.GetClient(f.defaultProvider)
}

// createClient creates a client based on the provided configuration
func (f *Factory) createClient(config ProviderConfig) (LLMClient, error) {
	switch config.Type {
	case ProviderOpenAI:
		apiKey := config.ApiKey
		if apiKey == "" {
			apiKey = os.Getenv("OPENAI_API_KEY")
			if apiKey == "" {
				return nil, fmt.Errorf("OpenAI API key not provided and OPENAI_API_KEY environment variable not set")
			}
		}
		
		options := []openai.ClientOption{}
		if config.Model != "" {
			options = append(options, openai.WithModel(config.Model))
		}
		
		return openai.NewClient(apiKey, options...)
		
	case ProviderAnthropic:
		apiKey := config.ApiKey
		if apiKey == "" {
			apiKey = os.Getenv("ANTHROPIC_API_KEY")
			if apiKey == "" {
				return nil, fmt.Errorf("Anthropic API key not provided and ANTHROPIC_API_KEY environment variable not set")
			}
		}
		
		options := []anthropic.ClientOption{}
		if config.Model != "" {
			options = append(options, anthropic.WithModel(config.Model))
		}
		
		return anthropic.NewClient(apiKey, options...)
		
	case ProviderOllama:
		options := []ollama.ClientOption{}
		
		if config.Model != "" {
			options = append(options, ollama.WithModel(config.Model))
		}
		
		if config.ApiURL != "" {
			options = append(options, ollama.WithEndpoint(config.ApiURL))
		}
		
		return ollama.NewClient(options...)
		
	default:
		return nil, fmt.Errorf("unknown provider type: %s", config.Type)
	}
}

// createClientFromEnv attempts to create a client using environment variables
func (f *Factory) createClientFromEnv(providerType ProviderType) (LLMClient, error) {
	switch providerType {
	case ProviderOpenAI:
		apiKey := os.Getenv("OPENAI_API_KEY")
		if apiKey == "" {
			return nil, fmt.Errorf("OPENAI_API_KEY environment variable not set")
		}
		return openai.NewClient(apiKey)
		
	case ProviderAnthropic:
		apiKey := os.Getenv("ANTHROPIC_API_KEY")
		if apiKey == "" {
			return nil, fmt.Errorf("ANTHROPIC_API_KEY environment variable not set")
		}
		return anthropic.NewClient(apiKey)
		
	case ProviderOllama:
		// Ollama typically runs locally and doesn't need API key
		return ollama.NewClient()
		
	default:
		return nil, fmt.Errorf("unknown provider type: %s", providerType)
	}
}

// CloseAll closes all clients
func (f *Factory) CloseAll() {
	f.mu.Lock()
	defer f.mu.Unlock()
	
	// Clear the clients map
	f.clients = make(map[ProviderType]LLMClient)
}