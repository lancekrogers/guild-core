package registry

import (
	"context"
	"fmt"
	"log"
	"os"
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
		Agents: AgentConfig{
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
			DefaultMemoryStore: "boltdb",
			DefaultVectorStore: "chromem",
			Stores: map[string]interface{}{
				"boltdb":  map[string]interface{}{"path": "./.guild/memory.db"},
				"chromem": map[string]interface{}{"persistence_path": "./.guild/vectors"},
			},
		},
	}

	// Initialize the registry
	ctx := context.Background()
	if err := registry.Initialize(ctx, *config); err != nil {
		return fmt.Errorf("failed to initialize registry: %w", err)
	}

	// Get provider registry
	providerRegistry := registry.Providers()

	// List all available providers
	providers := providerRegistry.ListProviders()
	log.Printf("Available providers: %v", providers)

	// Use the default provider
	defaultProvider, err := providerRegistry.GetDefaultProvider()
	if err != nil {
		return fmt.Errorf("failed to get default provider: %w", err)
	}

	log.Printf("Using default provider")
	
	// Generate a completion
	response, err := defaultProvider.Complete(ctx, "Explain what a registry pattern is in software architecture.")
	if err != nil {
		return fmt.Errorf("failed to generate completion: %w", err)
	}

	log.Printf("Default provider response: %s", response)

	// Try a different provider
	anthropicProvider, err := providerRegistry.GetProvider("anthropic")
	if err != nil {
		return fmt.Errorf("failed to get Anthropic provider: %w", err)
	}

	response, err = anthropicProvider.Complete(ctx, "What are the benefits of dependency injection?")
	if err != nil {
		return fmt.Errorf("failed to generate completion with Anthropic: %w", err)
	}

	log.Printf("Anthropic provider response: %s", response)

	// Switch default provider
	if err := providerRegistry.SetDefaultProvider("anthropic"); err != nil {
		return fmt.Errorf("failed to set default provider: %w", err)
	}

	log.Printf("Switched default provider to Anthropic")

	// Use the new default provider
	newDefaultProvider, err := providerRegistry.GetDefaultProvider()
	if err != nil {
		return fmt.Errorf("failed to get new default provider: %w", err)
	}

	response, err = newDefaultProvider.Complete(ctx, "Summarize the registry pattern implementation.")
	if err != nil {
		return fmt.Errorf("failed to generate completion with new default: %w", err)
	}

	log.Printf("New default provider response: %s", response)

	// Cleanup
	if err := registry.Shutdown(ctx); err != nil {
		return fmt.Errorf("failed to shutdown registry: %w", err)
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
  default_memory_store: "boltdb"
  default_vector_store: "chromem"
  stores:
    boltdb:
      path: "./.guild/memory.db"
    chromem:
      persistence_path: "./.guild/vectors"
      dimension: 1536
`

	// Load configuration from YAML
	config, err := LoadConfigFromBytes([]byte(yamlConfig))
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Validate configuration
	if err := ValidateConfig(config); err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}

	// Create and initialize registry
	registry := NewComponentRegistry()
	ctx := context.Background()
	
	if err := registry.Initialize(ctx, *config); err != nil {
		return fmt.Errorf("failed to initialize registry: %w", err)
	}

	// Use the providers
	providerRegistry := registry.Providers()
	
	// Get and use default provider
	provider, err := providerRegistry.GetDefaultProvider()
	if err != nil {
		return fmt.Errorf("failed to get default provider: %w", err)
	}

	response, err := provider.Complete(ctx, "Hello from YAML-configured provider!")
	if err != nil {
		return fmt.Errorf("failed to generate completion: %w", err)
	}

	log.Printf("YAML-configured provider response: %s", response)

	// Cleanup
	if err := registry.Shutdown(ctx); err != nil {
		return fmt.Errorf("failed to shutdown registry: %w", err)
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
		Agents: AgentConfig{
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
			DefaultMemoryStore: "boltdb",
			DefaultVectorStore: "chromem",
			Stores: map[string]interface{}{
				"boltdb":  map[string]interface{}{"path": "./.guild/memory.db"},
				"chromem": map[string]interface{}{"persistence_path": "./.guild/vectors"},
			},
		},
	}

	registry := NewComponentRegistry().(*DefaultComponentRegistry)
	ctx := context.Background()
	
	if err := registry.Initialize(ctx, *config); err != nil {
		return nil, fmt.Errorf("failed to initialize registry: %w", err)
	}

	return registry, nil
}