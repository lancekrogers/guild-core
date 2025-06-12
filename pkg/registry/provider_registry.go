package registry

import (
	"sync"

	"github.com/guild-ventures/guild-core/pkg/gerror"
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
		return gerror.New(gerror.ErrCodeInvalidInput, "provider name cannot be empty", nil).
			WithComponent("registry").
			WithOperation("RegisterProvider")
	}
	if provider == nil {
		return gerror.New(gerror.ErrCodeInvalidInput, "provider cannot be nil", nil).
			WithComponent("registry").
			WithOperation("RegisterProvider")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.providers[name]; exists {
		return gerror.Newf(gerror.ErrCodeAlreadyExists, "provider '%s' already registered", name).
			WithComponent("registry").
			WithOperation("RegisterProvider").
			WithDetails("provider", name)
	}

	// Convert the registry Provider interface to the actual LLMClient interface
	llmClient, ok := provider.(providers.LLMClient)
	if !ok {
		return gerror.New(gerror.ErrCodeInvalidFormat, "provider does not implement the expected LLMClient interface", nil).
			WithComponent("registry").
			WithOperation("RegisterProvider")
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
		return nil, gerror.Newf(gerror.ErrCodeNotFound, "provider '%s' not found", name).
			WithComponent("registry").
			WithOperation("GetProvider").
			WithDetails("provider", name)
	}

	return provider, nil
}

// Get is an alias for GetProvider for backward compatibility
// This method is expected by journey integration tests
func (r *DefaultProviderRegistry) Get(name string) (Provider, error) {
	return r.GetProvider(name)
}

// GetDefaultProvider returns the configured default provider
func (r *DefaultProviderRegistry) GetDefaultProvider() (Provider, error) {
	r.mu.RLock()
	defaultName := r.defaultProvider
	r.mu.RUnlock()

	if defaultName == "" {
		return nil, gerror.New(gerror.ErrCodeMissingRequired, "no default provider set", nil).
			WithComponent("registry").
			WithOperation("GetDefaultProvider")
	}

	return r.GetProvider(defaultName)
}

// SetDefaultProvider sets the default provider
func (r *DefaultProviderRegistry) SetDefaultProvider(name string) error {
	if !r.HasProvider(name) {
		return gerror.Newf(gerror.ErrCodeNotFound, "provider '%s' not registered", name).
			WithComponent("registry").
			WithOperation("SetDefaultProvider").
			WithDetails("provider", name)
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
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create provider").
			WithComponent("registry").
			WithOperation("CreateAndRegisterProvider").
			WithDetails("provider", name).
			WithDetails("providerType", string(providerType))
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
		return nil, gerror.Newf(gerror.ErrCodeNotFound, "provider '%s' not found", name).
			WithComponent("registry").
			WithOperation("GetProvider").
			WithDetails("provider", name)
	}

	return client, nil
}

// GetDefaultLLMClient returns the default LLMClient
func (r *DefaultProviderRegistry) GetDefaultLLMClient() (providers.LLMClient, error) {
	r.mu.RLock()
	defaultName := r.defaultProvider
	r.mu.RUnlock()

	if defaultName == "" {
		return nil, gerror.New(gerror.ErrCodeMissingRequired, "no default provider set", nil).
			WithComponent("registry").
			WithOperation("GetDefaultProvider")
	}

	return r.GetLLMClient(defaultName)
}
