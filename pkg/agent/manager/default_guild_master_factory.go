// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package manager

import (
	"context"

	"github.com/guild-ventures/guild-core/pkg/gerror"
	"github.com/guild-ventures/guild-core/pkg/prompts/layered"
	"github.com/guild-ventures/guild-core/pkg/providers"
)

// DefaultGuildMasterFactory creates a factory for Guild Master components
type DefaultGuildMasterFactory struct {
	promptManager layered.LayeredManager
	providers     map[string]providers.AIProvider
	registry      ComponentRegistry
}

// NewDefaultGuildMasterFactory creates a new Guild Master factory
func NewDefaultGuildMasterFactory(
	promptManager layered.LayeredManager,
	providers map[string]providers.AIProvider,
	registry ComponentRegistry,
) *DefaultGuildMasterFactory {
	// Create a registry if not provided
	if registry == nil {
		registry = NewComponentRegistry()
	}

	return &DefaultGuildMasterFactory{
		promptManager: promptManager,
		providers:     providers,
		registry:      registry,
	}
}

// CreateCommissionRefiner creates a fully configured commission refiner
func (f *DefaultGuildMasterFactory) CreateCommissionRefiner(providerName, model string) (CommissionRefiner, error) {
	// Get the AI provider
	provider, exists := f.providers[providerName]
	if !exists {
		return nil, gerror.New(gerror.ErrCodeValidation, "provider not found", nil).
			WithComponent("manager").
			WithOperation("CreateCommissionRefiner").
			WithDetails("provider_name", providerName)
	}

	// Try to get components from registry first, create if not found

	// Get or create Artisan client
	clientKey := providerName + "-" + model
	artisanClient, err := f.registry.GetArtisanClient(clientKey)
	if err != nil {
		// Create and register if not found
		artisanClient = NewGuildArtisanClient(provider, model)
		if regErr := f.registry.RegisterArtisanClient(clientKey, artisanClient); regErr != nil {
			// Log registration error but continue
			_ = regErr
		}
	}

	// Get or create response parser
	parserKey := "intelligent-auto"
	responseParser, err := f.registry.GetParser(parserKey)
	if err != nil {
		// Create intelligent response parser
		parserConfig := IntelligentParserConfig{
			Mode:          ParserModeAuto,
			ArtisanClient: artisanClient,
			PromptManager: f.promptManager,
		}

		// Create the parser with ResponseParserAdapter
		responseParser = NewResponseParserAdapter(NewIntelligentParser(parserConfig))
		if regErr := f.registry.RegisterParser(parserKey, responseParser); regErr != nil {
			// Log registration error but continue
			_ = regErr
		}
	}

	// Get or create structure validator
	validatorKey := "default"
	validator, err := f.registry.GetValidator(validatorKey)
	if err != nil {
		validator = NewDefaultValidator()
		if regErr := f.registry.RegisterValidator(validatorKey, validator); regErr != nil {
			// Log registration error but continue
			_ = regErr
		}
	}

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
			return nil, gerror.New(gerror.ErrCodeInternal, "no AI providers available", nil).
				WithComponent("manager").
				WithOperation("CreateCommissionRefinerWithDefaults")
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

// ParseResponseWithContext implements the ResponseParser interface with context support
func (a *ResponseParserAdapter) ParseResponseWithContext(ctx context.Context, response *ArtisanResponse) (*FileStructure, error) {
	return a.parser.ParseResponse(ctx, response)
}
