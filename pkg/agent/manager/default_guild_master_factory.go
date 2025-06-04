package manager

import (
	"context"
	"fmt"

	"github.com/guild-ventures/guild-core/pkg/prompts"
	"github.com/guild-ventures/guild-core/pkg/providers"
)

// DefaultGuildMasterFactory creates a factory for Guild Master components
type DefaultGuildMasterFactory struct {
	promptManager prompts.LayeredManager
	providers     map[string]providers.AIProvider
}

// NewDefaultGuildMasterFactory creates a new Guild Master factory
func NewDefaultGuildMasterFactory(
	promptManager prompts.LayeredManager,
	providers map[string]providers.AIProvider,
) *DefaultGuildMasterFactory {
	return &DefaultGuildMasterFactory{
		promptManager: promptManager,
		providers:     providers,
	}
}

// CreateCommissionRefiner creates a fully configured commission refiner
func (f *DefaultGuildMasterFactory) CreateCommissionRefiner(providerName, model string) (CommissionRefiner, error) {
	// Get the AI provider
	provider, exists := f.providers[providerName]
	if !exists {
		return nil, fmt.Errorf("provider %s not found", providerName)
	}

	// Create Artisan client
	artisanClient := NewGuildArtisanClient(provider, model)

	// Create intelligent response parser
	parserConfig := IntelligentParserConfig{
		Mode:          ParserModeAuto,
		ArtisanClient: artisanClient,
		PromptManager: f.promptManager,
	}
	
	// Create the parser with ResponseParserAdapter
	responseParser := NewResponseParserAdapter(NewIntelligentParser(parserConfig))

	// Create structure validator
	validator := NewDefaultValidator()

	// Create the Guild Master refiner
	refiner := NewGuildMasterRefiner(
		artisanClient,
		f.promptManager,
		responseParser,
		validator,
	)

	return refiner, nil
}

// CreateCommissionRefinerWithDefaults creates a refiner with sensible defaults
func (f *DefaultGuildMasterFactory) CreateCommissionRefinerWithDefaults() (CommissionRefiner, error) {
	// Default to Claude (Anthropic) if available
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

	return f.CreateCommissionRefiner(providerName, model)
}

// ResponseParserAdapter adapts IntelligentParser to ResponseParser interface
type ResponseParserAdapter struct {
	parser *IntelligentParser
}

// NewResponseParserAdapter creates a new adapter
func NewResponseParserAdapter(parser *IntelligentParser) *ResponseParserAdapter {
	return &ResponseParserAdapter{
		parser: parser,
	}
}

// ParseResponse implements the ResponseParser interface
func (a *ResponseParserAdapter) ParseResponse(response *ArtisanResponse) (*FileStructure, error) {
	// Use context.Background() for backward compatibility
	// The IntelligentParser will handle the context internally
	return a.parser.ParseResponse(context.Background(), response)
}