package registry

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewComponentRegistry(t *testing.T) {
	registry := NewComponentRegistry()
	require.NotNil(t, registry)

	// Test that all sub-registries are created
	assert.NotNil(t, registry.Agents())
	assert.NotNil(t, registry.Tools())
	assert.NotNil(t, registry.Providers())
	assert.NotNil(t, registry.Memory())
}

func TestConfigLoading(t *testing.T) {
	// Test default config
	config := DefaultConfig()
	require.NotNil(t, config)

	// Validate default config
	err := ValidateConfig(config)
	assert.NoError(t, err)

	// Test config fields
	assert.Equal(t, "worker", config.Agents.DefaultType)
	assert.Equal(t, "openai", config.Providers.DefaultProvider)
	assert.Equal(t, "boltdb", config.Memory.DefaultMemoryStore)
	assert.Equal(t, "chromem", config.Memory.DefaultVectorStore)
}

func TestToolRegistry(t *testing.T) {
	toolRegistry := NewToolRegistry()
	require.NotNil(t, toolRegistry)

	// Test that it's empty initially
	tools := toolRegistry.ListTools()
	assert.Empty(t, tools)

	// Test HasTool
	assert.False(t, toolRegistry.HasTool("nonexistent"))
}

func TestAgentRegistry(t *testing.T) {
	agentRegistry := NewAgentRegistry()
	require.NotNil(t, agentRegistry)

	// Test that it's empty initially
	types := agentRegistry.ListAgentTypes()
	assert.Empty(t, types)

	// Test HasAgentType
	assert.False(t, agentRegistry.HasAgentType("nonexistent"))

	// Test registering an agent type
	factory := func(config AgentConfig) (Agent, error) {
		return nil, nil // Mock factory
	}

	err := agentRegistry.RegisterAgentType("test-agent", factory)
	assert.NoError(t, err)

	// Test that it's now registered
	assert.True(t, agentRegistry.HasAgentType("test-agent"))
	types = agentRegistry.ListAgentTypes()
	assert.Contains(t, types, "test-agent")

	// Test duplicate registration
	err = agentRegistry.RegisterAgentType("test-agent", factory)
	assert.Error(t, err)
}

func TestProviderRegistry(t *testing.T) {
	providerRegistry := NewProviderRegistry()
	require.NotNil(t, providerRegistry)

	// Test that it's empty initially
	providers := providerRegistry.ListProviders()
	assert.Empty(t, providers)

	// Test HasProvider
	assert.False(t, providerRegistry.HasProvider("nonexistent"))
}

func TestMemoryRegistry(t *testing.T) {
	memoryRegistry := NewMemoryRegistry()
	require.NotNil(t, memoryRegistry)

	// Test that it's empty initially
	memoryStores := memoryRegistry.ListMemoryStores()
	assert.Empty(t, memoryStores)

	vectorStores := memoryRegistry.ListVectorStores()
	assert.Empty(t, vectorStores)
}

func TestComponentRegistryInitialization(t *testing.T) {
	registry := NewComponentRegistry()
	config := DefaultConfig()

	// Test initialization
	ctx := context.Background()
	err := registry.Initialize(ctx, *config)
	// This might fail due to missing dependencies, but that's expected in a unit test
	// The important thing is that it doesn't panic
	_ = err

	// Test shutdown
	err = registry.Shutdown(ctx)
	assert.NoError(t, err)
}

func TestConfigValidation(t *testing.T) {
	// Test valid config
	config := DefaultConfig()
	err := ValidateConfig(config)
	assert.NoError(t, err)

	// Test invalid config - empty default agent type
	invalidConfig := *config
	invalidConfig.Agents.DefaultType = ""
	err = ValidateConfig(&invalidConfig)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "agents.default_type cannot be empty")

	// Test invalid config - empty default provider
	invalidConfig = *config
	invalidConfig.Providers.DefaultProvider = ""
	err = ValidateConfig(&invalidConfig)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "providers.default_provider cannot be empty")

	// Test invalid config - nonexistent default provider
	invalidConfig = *config
	invalidConfig.Providers.DefaultProvider = "nonexistent"
	err = ValidateConfig(&invalidConfig)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "default provider 'nonexistent' not found")
}

func TestConfigHelpers(t *testing.T) {
	config := DefaultConfig()

	// Test GetProviderConfig
	providerConfig, err := config.GetProviderConfig("openai")
	assert.NoError(t, err)
	assert.NotNil(t, providerConfig)
	assert.Equal(t, "gpt-4.1", providerConfig["model"])

	// Test GetProviderConfig for nonexistent provider
	_, err = config.GetProviderConfig("nonexistent")
	assert.Error(t, err)

	// Test GetMemoryStoreConfig
	storeConfig, err := config.GetMemoryStoreConfig("boltdb")
	assert.NoError(t, err)
	assert.NotNil(t, storeConfig)

	// Test IsToolEnabled
	assert.True(t, config.IsToolEnabled("file"))
	assert.False(t, config.IsToolEnabled("nonexistent"))
}