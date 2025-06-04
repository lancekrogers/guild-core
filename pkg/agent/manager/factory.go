package manager

import (
	"fmt"

	"github.com/guild-ventures/guild-core/pkg/prompts"
	"github.com/guild-ventures/guild-core/pkg/providers"
)

// GuildMasterFactory creates configured Guild Master refinement components
type GuildMasterFactory struct {
	promptManager prompts.LayeredManager
	providers     map[string]providers.AIProvider
}

// NewGuildMasterFactory creates a new factory for Guild Master components
func NewGuildMasterFactory(
	promptManager prompts.LayeredManager,
	providers map[string]providers.AIProvider,
) *GuildMasterFactory {
	return &GuildMasterFactory{
		promptManager: promptManager,
		providers:     providers,
	}
}

// CreateGuildMasterRefiner creates a fully configured Guild Master refiner
func (f *GuildMasterFactory) CreateGuildMasterRefiner(providerName, model string) (*GuildMasterRefiner, error) {
	// Get the AI provider
	provider, exists := f.providers[providerName]
	if !exists {
		return nil, fmt.Errorf("provider %s not found", providerName)
	}

	// Create Artisan client
	artisanClient := NewGuildArtisanClient(provider, model)

	// Create response parser
	responseParser := NewResponseParser()

	// Create structure validator
	validator := NewDefaultValidator()

	// Create the Guild Master refiner
	refiner := NewGuildMasterRefiner(
		artisanClient,
		f.promptManager, // Uses LayeredManager which implements Manager interface
		responseParser,
		validator,
	)

	return refiner, nil
}

// CreateGuildMasterRefinerWithDefaults creates a Guild Master refiner with sensible defaults
func (f *GuildMasterFactory) CreateGuildMasterRefinerWithDefaults() (*GuildMasterRefiner, error) {
	// Default to Claude (Anthropic) if available, otherwise first available provider
	providerName := "anthropic"
	model := "claude-3-5-sonnet-20241022"

	// Check if Anthropic is available
	if _, exists := f.providers[providerName]; !exists {
		// Fall back to first available provider
		for name := range f.providers {
			providerName = name
			break
		}
		if providerName == "" {
			return nil, fmt.Errorf("no AI providers available")
		}
		
		// Set default model based on provider
		switch providerName {
		case "openai":
			model = "gpt-4"
		case "ollama":
			model = "llama3.1"
		default:
			model = "" // Let provider use its default
		}
	}

	return f.CreateGuildMasterRefiner(providerName, model)
}

// GetAvailableProviders returns the names of available AI providers
func (f *GuildMasterFactory) GetAvailableProviders() []string {
	var providers []string
	for name := range f.providers {
		providers = append(providers, name)
	}
	return providers
}

// GetProviderCapabilities returns the capabilities of a specific provider
func (f *GuildMasterFactory) GetProviderCapabilities(providerName string) (providers.ProviderCapabilities, error) {
	provider, exists := f.providers[providerName]
	if !exists {
		return providers.ProviderCapabilities{}, fmt.Errorf("provider %s not found", providerName)
	}
	return provider.GetCapabilities(), nil
}