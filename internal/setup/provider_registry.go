// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package setup

import (
	"context"
	"sync"

	"github.com/guild-framework/guild-core/pkg/config"
	"github.com/guild-framework/guild-core/pkg/gerror"
)

// ProviderDefinition defines a provider's metadata and configuration
type ProviderDefinition struct {
	Name        string
	DisplayName string
	Description string
	Type        string // "cloud" or "local"
	ConfigKey   string // Key in ProvidersConfig struct
	SetConfig   func(*config.ProvidersConfig, config.ProviderSettings)
	GetConfig   func(*config.ProvidersConfig) config.ProviderSettings
}

// ProviderRegistry manages all available providers dynamically
type ProviderRegistry struct {
	mu           sync.RWMutex
	providers    map[string]*ProviderDefinition
	modelService *ModelQueryService
}

// NewProviderRegistry creates a new provider registry with default providers
func NewProviderRegistry() *ProviderRegistry {
	r := &ProviderRegistry{
		providers:    make(map[string]*ProviderDefinition),
		modelService: NewModelQueryService(),
	}

	// Register default providers
	r.registerDefaultProviders()

	return r
}

// registerDefaultProviders registers the built-in providers
func (r *ProviderRegistry) registerDefaultProviders() {
	// OpenAI
	r.Register(&ProviderDefinition{
		Name:        "openai",
		DisplayName: "OpenAI",
		Description: "GPT-4, GPT-3.5 and other OpenAI models",
		Type:        "cloud",
		ConfigKey:   "openai",
		SetConfig: func(cfg *config.ProvidersConfig, settings config.ProviderSettings) {
			cfg.OpenAI = settings
		},
		GetConfig: func(cfg *config.ProvidersConfig) config.ProviderSettings {
			return cfg.OpenAI
		},
	})

	// Anthropic
	r.Register(&ProviderDefinition{
		Name:        "anthropic",
		DisplayName: "Anthropic",
		Description: "Claude 3 Opus, Sonnet, and Haiku models",
		Type:        "cloud",
		ConfigKey:   "anthropic",
		SetConfig: func(cfg *config.ProvidersConfig, settings config.ProviderSettings) {
			cfg.Anthropic = settings
		},
		GetConfig: func(cfg *config.ProvidersConfig) config.ProviderSettings {
			return cfg.Anthropic
		},
	})

	// Ollama
	r.Register(&ProviderDefinition{
		Name:        "ollama",
		DisplayName: "Ollama",
		Description: "Local models with complete privacy",
		Type:        "local",
		ConfigKey:   "ollama",
		SetConfig: func(cfg *config.ProvidersConfig, settings config.ProviderSettings) {
			cfg.Ollama = settings
		},
		GetConfig: func(cfg *config.ProvidersConfig) config.ProviderSettings {
			return cfg.Ollama
		},
	})

	// Claude Code
	r.Register(&ProviderDefinition{
		Name:        "claude_code",
		DisplayName: "Claude Code",
		Description: "Claude 3 models with advanced capabilities",
		Type:        "cloud",
		ConfigKey:   "claude_code",
		SetConfig: func(cfg *config.ProvidersConfig, settings config.ProviderSettings) {
			cfg.ClaudeCode = settings
		},
		GetConfig: func(cfg *config.ProvidersConfig) config.ProviderSettings {
			return cfg.ClaudeCode
		},
	})

	// DeepSeek
	r.Register(&ProviderDefinition{
		Name:        "deepseek",
		DisplayName: "DeepSeek",
		Description: "DeepSeek coding and reasoning models",
		Type:        "cloud",
		ConfigKey:   "deepseek",
		SetConfig: func(cfg *config.ProvidersConfig, settings config.ProviderSettings) {
			cfg.DeepSeek = settings
		},
		GetConfig: func(cfg *config.ProvidersConfig) config.ProviderSettings {
			return cfg.DeepSeek
		},
	})

	// Add more providers as needed...
}

// Register adds a new provider to the registry
func (r *ProviderRegistry) Register(provider *ProviderDefinition) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if provider.Name == "" {
		return gerror.New(gerror.ErrCodeValidation, "provider name is required", nil).
			WithComponent("ProviderRegistry").
			WithOperation("Register")
	}

	r.providers[provider.Name] = provider
	return nil
}

// Get retrieves a provider definition by name
func (r *ProviderRegistry) Get(name string) (*ProviderDefinition, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	provider, exists := r.providers[name]
	return provider, exists
}

// List returns all registered providers
func (r *ProviderRegistry) List() []*ProviderDefinition {
	r.mu.RLock()
	defer r.mu.RUnlock()

	providers := make([]*ProviderDefinition, 0, len(r.providers))
	for _, p := range r.providers {
		providers = append(providers, p)
	}
	return providers
}

// ApplyProviderSettings applies provider settings to a config using the registry
func (r *ProviderRegistry) ApplyProviderSettings(cfg *config.ProvidersConfig, providerName string, settings config.ProviderSettings) error {
	provider, exists := r.Get(providerName)
	if !exists {
		return gerror.Newf(gerror.ErrCodeNotFound, "provider %s not found in registry", providerName).
			WithComponent("ProviderRegistry").
			WithOperation("ApplyProviderSettings")
	}

	provider.SetConfig(cfg, settings)
	return nil
}

// GetProviderSettings retrieves provider settings from a config using the registry
func (r *ProviderRegistry) GetProviderSettings(cfg *config.ProvidersConfig, providerName string) (config.ProviderSettings, error) {
	provider, exists := r.Get(providerName)
	if !exists {
		return config.ProviderSettings{}, gerror.Newf(gerror.ErrCodeNotFound, "provider %s not found in registry", providerName).
			WithComponent("ProviderRegistry").
			WithOperation("GetProviderSettings")
	}

	return provider.GetConfig(cfg), nil
}

// GetConfiguredProviders returns a list of providers that have configuration
func (r *ProviderRegistry) GetConfiguredProviders(cfg *config.ProvidersConfig) []ProviderStatus {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var configured []ProviderStatus
	ctx := context.Background() // Could be passed in if needed

	for _, provider := range r.providers {
		settings := provider.GetConfig(cfg)
		if settings.BaseURL != "" {
			// Query model count dynamically
			modelCount := r.modelService.GetModelCount(ctx, provider.Name)

			configured = append(configured, ProviderStatus{
				Name:      provider.Name,
				Available: true,
				Models:    modelCount,
			})
		}
	}

	return configured
}
