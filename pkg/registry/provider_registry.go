package registry

import (
	"fmt"
	"sync"

	"github.com/guild-ventures/guild-core/pkg/providers"
)

// DefaultProviderRegistry implements the ProviderRegistry interface
type DefaultProviderRegistry struct {
	providers       map[string]providers.LLMClient
	defaultProvider string
	factory         *providers.Factory
	mu              sync.RWMutex
}

// NewProviderRegistry creates a new provider registry
func NewProviderRegistry() ProviderRegistry {
	return &DefaultProviderRegistry{
		providers: make(map[string]providers.LLMClient),
		factory:   providers.NewFactory(),
	}
}

// RegisterProvider registers an LLM provider
func (r *DefaultProviderRegistry) RegisterProvider(name string, provider Provider) error {
	if name == "" {
		return fmt.Errorf("provider name cannot be empty")
	}
	if provider == nil {
		return fmt.Errorf("provider cannot be nil")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.providers[name]; exists {
		return fmt.Errorf("provider '%s' already registered", name)
	}

	// Convert the registry Provider interface to the actual LLMClient interface
	llmClient, ok := provider.(providers.LLMClient)
	if !ok {
		return fmt.Errorf("provider does not implement the expected LLMClient interface")
	}

	r.providers[name] = llmClient
	return nil
}

// GetProvider retrieves a provider by name
func (r *DefaultProviderRegistry) GetProvider(name string) (Provider, error) {
	r.mu.RLock()
	provider, exists := r.providers[name]
	r.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("provider '%s' not found", name)
	}

	return provider, nil
}

// GetDefaultProvider returns the configured default provider
func (r *DefaultProviderRegistry) GetDefaultProvider() (Provider, error) {
	r.mu.RLock()
	defaultName := r.defaultProvider
	r.mu.RUnlock()

	if defaultName == "" {
		return nil, fmt.Errorf("no default provider set")
	}

	return r.GetProvider(defaultName)
}

// SetDefaultProvider sets the default provider
func (r *DefaultProviderRegistry) SetDefaultProvider(name string) error {
	if !r.HasProvider(name) {
		return fmt.Errorf("provider '%s' not registered", name)
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	r.defaultProvider = name
	return nil
}

// ListProviders returns all registered provider names
func (r *DefaultProviderRegistry) ListProviders() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.providers))
	for name := range r.providers {
		names = append(names, name)
	}
	return names
}

// HasProvider checks if a provider is registered
func (r *DefaultProviderRegistry) HasProvider(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	_, exists := r.providers[name]
	return exists
}

// CreateAndRegisterProvider creates a provider using the factory and registers it
func (r *DefaultProviderRegistry) CreateAndRegisterProvider(name string, providerType providers.ProviderType, apiKey, model string) error {
	client, err := r.factory.CreateClient(providerType, apiKey, model)
	if err != nil {
		return fmt.Errorf("failed to create provider: %w", err)
	}

	return r.RegisterProvider(name, client)
}

// GetLLMClient returns the underlying LLMClient for a provider
// This is useful for components that need direct access to the LLMClient interface
func (r *DefaultProviderRegistry) GetLLMClient(name string) (providers.LLMClient, error) {
	r.mu.RLock()
	client, exists := r.providers[name]
	r.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("provider '%s' not found", name)
	}

	return client, nil
}

// GetDefaultLLMClient returns the default LLMClient
func (r *DefaultProviderRegistry) GetDefaultLLMClient() (providers.LLMClient, error) {
	r.mu.RLock()
	defaultName := r.defaultProvider
	r.mu.RUnlock()

	if defaultName == "" {
		return nil, fmt.Errorf("no default provider set")
	}

	return r.GetLLMClient(defaultName)
}