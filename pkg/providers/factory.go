// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package providers

import (
    "os"

    "github.com/guild-framework/guild-core/pkg/gerror"
)

// Factory creates LLM clients
type Factory struct {
	v2Factory *FactoryV2
}

// NewFactory creates a new factory
func NewFactory() *Factory {
	return &Factory{
		v2Factory: NewFactoryV2(),
	}
}

// CreateClient creates a new LLM client based on the provider type.
//
// Parameters:
//   - providerType: The type of provider (OpenAI, Anthropic, ClaudeCode, etc.)
//   - apiKey: API key for most providers. For ClaudeCode, this is the path to the claude binary
//     (defaults to "claude" if empty, assumes claude is in PATH)
//   - model: The model to use (e.g., "gpt-4", "claude-3-opus", etc.)
//
// Note: ClaudeCode provider doesn't require an API key as it uses the Claude Code CLI
// which handles its own authentication through the /login command. The apiKey parameter
// is repurposed as the binary path due to interface constraints.
func (f *Factory) CreateClient(providerType ProviderType, apiKey string, model string) (LLMClient, error) {
    // Use the V2 factory to create AIProvider,
    // then wrap it with the LLMClient adapter
	aiProvider, err := f.v2Factory.CreateAIProvider(providerType, apiKey)
	if err != nil {
		return nil, err
	}

	return f.v2Factory.CreateLLMClientAdapter(aiProvider), nil
}

// CreateClientFromConfig creates a client from configuration map.
//
// Config keys:
//   - "model": The model to use (optional, uses provider defaults if not specified)
//   - "api_key": API key for most providers. For ClaudeCode, this is the path to claude binary
//   - "api_key_env": Environment variable containing the API key (used if api_key not provided)
//   - "binary_path": Alternative key for ClaudeCode provider to specify claude binary path
//
// For ClaudeCode provider, the config can use either "api_key" or "binary_path" to specify
// the path to the claude CLI binary. If neither is provided, defaults to "claude" in PATH.
func (f *Factory) CreateClientFromConfig(providerType ProviderType, config map[string]interface{}) (LLMClient, error) {
	// Extract model
	model, ok := config["model"].(string)
	if !ok {
		model = "" // Will use provider defaults
	}

	// Extract API key - try config first, then environment variable
	var apiKey string

	// For ClaudeCode, also check for binary_path key for clarity
	if providerType == ProviderClaudeCode {
		if path, exists := config["binary_path"]; exists {
			if pathStr, ok := path.(string); ok {
				apiKey = pathStr
			}
		}
	}

	// Standard api_key extraction (also used as binary path for ClaudeCode)
	if apiKey == "" {
		if key, exists := config["api_key"]; exists {
			if keyStr, ok := key.(string); ok {
				apiKey = keyStr
			}
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
		providerType := ConvertToProviderType(providerName)
		if providerType == ProviderGoogle {
			// Special case for Google which isn't in our constants yet
			// Keep the error for now
			return gerror.Newf(gerror.ErrCodeProvider, "unknown provider type: %s", providerName).
				WithComponent("providers").
				WithOperation("RegisterProvidersWithRegistry").
				WithDetails("provider_name", providerName)
		}

		// Validate provider
		if !IsValidProvider(providerName) && providerName != "google" {
			return gerror.Newf(gerror.ErrCodeProvider, "unknown provider type: %s", providerName).
				WithComponent("providers").
				WithOperation("RegisterProvidersWithRegistry").
				WithDetails("provider_name", providerName)
		}

		// Extract provider config
		providerConfig, ok := providerConfigRaw.(map[string]interface{})
		if !ok {
			return gerror.Newf(gerror.ErrCodeInvalidInput, "invalid config for provider %s", providerName).
				WithComponent("providers").
				WithOperation("RegisterProvidersWithRegistry").
				WithDetails("provider_name", providerName)
		}

		// Create client
		client, err := f.CreateClientFromConfig(providerType, providerConfig)
		if err != nil {
			return gerror.Wrapf(err, gerror.ErrCodeProvider, "failed to create client for provider %s", providerName).
				WithComponent("providers").
				WithOperation("RegisterProvidersWithRegistry").
				WithDetails("provider_name", providerName)
		}

		// Register with registry
		if err := registry.RegisterProvider(providerName, client); err != nil {
			return gerror.Wrapf(err, gerror.ErrCodeProvider, "failed to register provider %s", providerName).
				WithComponent("providers").
				WithOperation("RegisterProvidersWithRegistry").
				WithDetails("provider_name", providerName)
		}
	}

	return nil
}

// ProviderRegistry is the interface we expect (matches the one in registry package)
type ProviderRegistry interface {
	RegisterProvider(name string, provider LLMClient) error
	SetDefaultProvider(name string) error
}
