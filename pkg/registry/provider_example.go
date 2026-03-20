// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package registry

import (
	"context"
	"log"
	"os"

	"github.com/lancekrogers/guild-core/pkg/gerror"
)

// ProviderExample demonstrates how to use the integrated provider system
func ProviderExample() error {
	// Set up environment variables (in real usage, these would be set externally)
	os.Setenv("OPENAI_API_KEY", "your-openai-api-key")
	os.Setenv("ANTHROPIC_API_KEY", "your-anthropic-api-key")

	// Create registry with provider configuration
	registry := NewComponentRegistry()

	// Load configuration (you could also load from YAML file)
	config := &Config{
		Providers: ProviderConfig{
			DefaultProvider: "openai",
			Providers: map[string]interface{}{
				"openai": map[string]interface{}{
					"model":       "gpt-4",
					"api_key_env": "OPENAI_API_KEY",
				},
				"anthropic": map[string]interface{}{
					"model":       "claude-3-sonnet-20240229",
					"api_key_env": "ANTHROPIC_API_KEY",
				},
				"ollama": map[string]interface{}{
					"model": "llama2",
					"url":   "http://localhost:11434",
				},
			},
		},
		// Minimal config for other components (required for validation)
		Agents: AgentConfigYaml{
			DefaultType: "worker",
			Types: map[string]interface{}{
				"worker": map[string]interface{}{"enabled": true},
			},
		},
		Tools: ToolConfig{
			EnabledTools: []string{"file", "shell"},
			Settings:     map[string]interface{}{"timeout": "30s"},
		},
		Memory: MemoryConfig{
			DefaultMemoryStore: "sqlite",
			DefaultVectorStore: "chromem",
			Stores: map[string]interface{}{
				"sqlite":  map[string]interface{}{"path": "./.guild/memory.db"},
				"chromem": map[string]interface{}{"persistence_path": "./.guild/vectors"},
			},
		},
	}

	// Initialize the registry
	ctx := context.Background()
	if err := registry.Initialize(ctx, *config); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "registry").WithComponent("provider_example").WithOperation("failed to initialize registry")
	}

	// Get provider registry
	providerRegistry := registry.Providers()

	// List all available providers
	providers := providerRegistry.ListProviders()
	log.Printf("Available providers: %v", providers)

	// Use the default provider
	defaultProvider, err := providerRegistry.GetDefaultProvider()
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "registry").WithComponent("provider_example").WithOperation("failed to get default provider")
	}

	log.Printf("Using default provider")

	// Generate a completion
	response, err := defaultProvider.Complete(ctx, "Explain what a registry pattern is in software architecture.")
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "registry").WithComponent("provider_example").WithOperation("failed to generate completion")
	}

	log.Printf("Default provider response: %s", response)

	// Try a different provider
	anthropicProvider, err := providerRegistry.GetProvider("anthropic")
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "registry").WithComponent("provider_example").WithOperation("failed to get Anthropic provider")
	}

	response, err = anthropicProvider.Complete(ctx, "What are the benefits of dependency injection?")
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "registry").WithComponent("provider_example").WithOperation("failed to generate completion with Anthropic")
	}

	log.Printf("Anthropic provider response: %s", response)

	// Switch default provider
	if err := providerRegistry.SetDefaultProvider("anthropic"); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "registry").WithComponent("provider_example").WithOperation("failed to set default provider")
	}

	log.Printf("Switched default provider to Anthropic")

	// Use the new default provider
	newDefaultProvider, err := providerRegistry.GetDefaultProvider()
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "registry").WithComponent("provider_example").WithOperation("failed to get new default provider")
	}

	response, err = newDefaultProvider.Complete(ctx, "Summarize the registry pattern implementation.")
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "registry").WithComponent("provider_example").WithOperation("failed to generate completion with new default")
	}

	log.Printf("New default provider response: %s", response)

	// Cleanup
	if err := registry.Shutdown(ctx); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "registry").WithComponent("provider_example").WithOperation("failed to shutdown registry")
	}

	return nil
}

// ProviderExampleFromYAML demonstrates loading provider configuration from YAML
func ProviderExampleFromYAML() error {
	yamlConfig := `
providers:
  default_provider: "openai"
  providers:
    openai:
      model: "gpt-4"
      api_key_env: "OPENAI_API_KEY"
    anthropic:
      model: "claude-3-sonnet-20240229"
      api_key_env: "ANTHROPIC_API_KEY"
    ollama:
      model: "llama2"
      url: "http://localhost:11434"

agents:
  default_type: "worker"
  types:
    worker:
      enabled: true

tools:
  enabled_tools: ["file", "shell"]
  settings:
    timeout: "30s"

memory:
  default_memory_store: "sqlite"
  default_vector_store: "chromem"
  stores:
    sqlite:
      path: "./.guild/memory.db"
    chromem:
      persistence_path: "./.guild/vectors"
      dimension: 1536
`

	// Load configuration from YAML
	config, err := LoadConfigFromBytes([]byte(yamlConfig))
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "registry").WithComponent("provider_example_yaml").WithOperation("failed to load config")
	}

	// Validate configuration
	if err := ValidateConfig(config); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInvalidInput, "registry").WithComponent("provider_example_yaml").WithOperation("invalid config")
	}

	// Create and initialize registry
	registry := NewComponentRegistry()
	ctx := context.Background()

	if err := registry.Initialize(ctx, *config); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "registry").WithComponent("provider_example").WithOperation("failed to initialize registry")
	}

	// Use the providers
	providerRegistry := registry.Providers()

	// Get and use default provider
	provider, err := providerRegistry.GetDefaultProvider()
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "registry").WithComponent("provider_example").WithOperation("failed to get default provider")
	}

	response, err := provider.Complete(ctx, "Hello from YAML-configured provider!")
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "registry").WithComponent("provider_example").WithOperation("failed to generate completion")
	}

	log.Printf("YAML-configured provider response: %s", response)

	// Cleanup
	if err := registry.Shutdown(ctx); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "registry").WithComponent("provider_example").WithOperation("failed to shutdown registry")
	}

	return nil
}

// CreateProviderOnlyRegistry shows how to create a minimal registry with just providers
func CreateProviderOnlyRegistry() (*DefaultComponentRegistry, error) {
	config := &Config{
		Providers: ProviderConfig{
			DefaultProvider: "openai",
			Providers: map[string]interface{}{
				"openai": map[string]interface{}{
					"model":       "gpt-3.5-turbo",
					"api_key_env": "OPENAI_API_KEY",
				},
			},
		},
		// Minimal required config for other components
		Agents: AgentConfigYaml{
			DefaultType: "worker",
			Types: map[string]interface{}{
				"worker": map[string]interface{}{"enabled": true},
			},
		},
		Tools: ToolConfig{
			EnabledTools: []string{},
			Settings:     map[string]interface{}{},
		},
		Memory: MemoryConfig{
			DefaultMemoryStore: "sqlite",
			DefaultVectorStore: "chromem",
			Stores: map[string]interface{}{
				"sqlite":  map[string]interface{}{"path": "./.guild/memory.db"},
				"chromem": map[string]interface{}{"persistence_path": "./.guild/vectors"},
			},
		},
	}

	registry := NewComponentRegistry().(*DefaultComponentRegistry)
	ctx := context.Background()

	if err := registry.Initialize(ctx, *config); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "registry").WithComponent("create_provider_only_registry").WithOperation("failed to initialize registry")
	}

	return registry, nil
}
