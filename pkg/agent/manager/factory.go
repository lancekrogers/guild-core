package manager

import (
	"context"

	"github.com/guild-ventures/guild-core/pkg/gerror"
	"github.com/guild-ventures/guild-core/pkg/prompts"
	"github.com/guild-ventures/guild-core/pkg/providers"
)

// GuildMasterFactory creates configured Guild Master refinement components
type GuildMasterFactory struct {
	promptManager prompts.LayeredManager
	providers     map[string]providers.AIProvider
	parserMode    ParserMode
}

// NewGuildMasterFactory creates a new factory for Guild Master components
func NewGuildMasterFactory(
	promptManager prompts.LayeredManager,
	providers map[string]providers.AIProvider,
) *GuildMasterFactory {
	return &GuildMasterFactory{
		promptManager: promptManager,
		providers:     providers,
		parserMode:    ParserModeAuto, // Default to auto mode
	}
}

// SetParserMode configures how the factory creates parsers
func (f *GuildMasterFactory) SetParserMode(mode ParserMode) {
	f.parserMode = mode
}

// CreateGuildMasterRefiner creates a fully configured Guild Master refiner
func (f *GuildMasterFactory) CreateGuildMasterRefiner(providerName, model string) (*GuildMasterRefiner, error) {
	// Get the AI provider
	provider, exists := f.providers[providerName]
	if !exists {
		return nil, gerror.New(gerror.ErrCodeValidation, "provider not found").
			WithComponent("manager").
			WithOperation("CreateGuildMasterRefiner").
			WithDetails("provider_name", providerName)
	}

	// Create Artisan client
	artisanClient := NewGuildArtisanClient(provider, model)

	// Create intelligent response parser based on configuration
	parserConfig := IntelligentParserConfig{
		Mode:          f.parserMode,
		ArtisanClient: artisanClient,
		PromptManager: f.promptManager,
	}
	
	// Create an adapter to make IntelligentParser implement ResponseParser
	responseParser := &intelligentParserAdapter{
		parser: NewIntelligentParser(parserConfig),
	}

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
			return nil, gerror.New(gerror.ErrCodeInternal, "no AI providers available").
				WithComponent("manager").
				WithOperation("CreateGuildMasterRefinerWithDefaults")
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
		return providers.ProviderCapabilities{}, gerror.New(gerror.ErrCodeValidation, "provider not found").
			WithComponent("manager").
			WithOperation("GetProviderCapabilities").
			WithDetails("provider_name", providerName)
	}
	return provider.GetCapabilities(), nil
}

// intelligentParserAdapter adapts IntelligentParser to ResponseParser interface
type intelligentParserAdapter struct {
	parser *IntelligentParser
}

// ParseResponse implements ResponseParser interface
func (a *intelligentParserAdapter) ParseResponse(response *ArtisanResponse) (*FileStructure, error) {
	// Use context.Background() for backward compatibility
	return a.parser.ParseResponse(context.Background(), response)
}

// ParseResponseWithContext implements ResponseParser interface with context support
func (a *intelligentParserAdapter) ParseResponseWithContext(ctx context.Context, response *ArtisanResponse) (*FileStructure, error) {
	return a.parser.ParseResponse(ctx, response)
}