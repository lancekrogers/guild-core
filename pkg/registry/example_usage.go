package registry

import (
	"context"
	"fmt"
	"log"
)

// ExampleUsage demonstrates how to use the ComponentRegistry
func ExampleUsage() error {
	// Create a new component registry
	registry := NewComponentRegistry()

	// Load default configuration
	config := DefaultConfig()

	// Initialize the registry with configuration
	ctx := context.Background()
	if err := registry.Initialize(ctx, *config); err != nil {
		return fmt.Errorf("failed to initialize registry: %w", err)
	}

	// Use the agent registry
	agentRegistry := registry.Agents()
	
	// Register a simple mock agent type
	err := agentRegistry.RegisterAgentType("example", func(config AgentConfig) (Agent, error) {
		return &MockAgent{
			id:   "example-1",
			name: "Example Agent",
		}, nil
	})
	if err != nil {
		return fmt.Errorf("failed to register agent type: %w", err)
	}

	// Create an agent
	agent, err := agentRegistry.GetAgent("example")
	if err != nil {
		return fmt.Errorf("failed to get agent: %w", err)
	}

	log.Printf("Created agent: %s (%s)", agent.GetName(), agent.GetID())

	// Use the tool registry
	toolRegistry := registry.Tools()
	tools := toolRegistry.ListTools()
	log.Printf("Available tools: %v", tools)

	// Use the provider registry
	providerRegistry := registry.Providers()
	providers := providerRegistry.ListProviders()
	log.Printf("Available providers: %v", providers)

	// Test a provider if any are available
	if len(providers) > 0 {
		provider, err := providerRegistry.GetDefaultProvider()
		if err == nil {
			response, err := provider.Complete(ctx, "Hello from the registry system!")
			if err == nil {
				log.Printf("Provider response: %s", response)
			}
		}
	}

	// Shutdown the registry
	if err := registry.Shutdown(ctx); err != nil {
		return fmt.Errorf("failed to shutdown registry: %w", err)
	}

	return nil
}

// MockAgent is a simple mock implementation of the Agent interface
type MockAgent struct {
	id   string
	name string
}

// Execute implements the Agent interface
func (m *MockAgent) Execute(ctx context.Context, request string) (string, error) {
	return fmt.Sprintf("Mock agent %s processed request: %s", m.name, request), nil
}

// GetID implements the Agent interface
func (m *MockAgent) GetID() string {
	return m.id
}

// GetName implements the Agent interface
func (m *MockAgent) GetName() string {
	return m.name
}

// CreateRegistryFromYAML shows how to create a registry from YAML configuration
func CreateRegistryFromYAML(yamlConfig string) (*DefaultComponentRegistry, error) {
	// Parse configuration
	config, err := LoadConfigFromBytes([]byte(yamlConfig))
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	// Validate configuration
	if err := ValidateConfig(config); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	// Create and initialize registry
	registry := NewComponentRegistry().(*DefaultComponentRegistry)
	ctx := context.Background()
	if err := registry.Initialize(ctx, *config); err != nil {
		return nil, fmt.Errorf("failed to initialize registry: %w", err)
	}

	return registry, nil
}

// Example YAML configuration
const ExampleYAMLConfig = `
agents:
  default_type: "worker"
  types:
    worker:
      enabled: true
    manager:
      enabled: true

tools:
  enabled_tools:
    - "file"
    - "shell" 
    - "http"
  settings:
    timeout: "30s"

providers:
  default_provider: "openai"
  providers:
    openai:
      model: "gpt-4"
      api_key_env: "OPENAI_API_KEY"
    anthropic:
      model: "claude-3-sonnet-20240229"
      api_key_env: "ANTHROPIC_API_KEY"

memory:
  default_memory_store: "boltdb"
  default_vector_store: "chromem"
  stores:
    boltdb:
      path: "./data/memory.db"
    chromem:
      persistence_path: "./data/vectors"
      dimension: 1536
`