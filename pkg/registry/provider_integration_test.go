package registry

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProviderIntegration(t *testing.T) {
	// Create registry
	registry := NewComponentRegistry()

	// Create config with providers
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
		// Add minimal config for other components to avoid validation errors
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
				"boltdb": map[string]interface{}{"path": "./data/memory.db"},
				"chromem": map[string]interface{}{"persistence_path": "./data/vectors"},
			},
		},
	}

	// Set some dummy environment variables for testing
	os.Setenv("OPENAI_API_KEY", "test-openai-key")
	os.Setenv("ANTHROPIC_API_KEY", "test-anthropic-key")
	defer func() {
		os.Unsetenv("OPENAI_API_KEY")
		os.Unsetenv("ANTHROPIC_API_KEY")
	}()

	// Initialize registry
	ctx := context.Background()
	err := registry.Initialize(ctx, *config)
	require.NoError(t, err)

	// Test provider registry
	providerRegistry := registry.Providers()

	// Check that providers are registered
	providers := providerRegistry.ListProviders()
	assert.Contains(t, providers, "openai")
	assert.Contains(t, providers, "anthropic")
	assert.Contains(t, providers, "ollama")

	// Test getting specific providers
	openaiProvider, err := providerRegistry.GetProvider("openai")
	assert.NoError(t, err)
	assert.NotNil(t, openaiProvider)

	anthropicProvider, err := providerRegistry.GetProvider("anthropic")
	assert.NoError(t, err)
	assert.NotNil(t, anthropicProvider)

	ollamaProvider, err := providerRegistry.GetProvider("ollama")
	assert.NoError(t, err)
	assert.NotNil(t, ollamaProvider)

	// Test default provider
	defaultProvider, err := providerRegistry.GetDefaultProvider()
	assert.NoError(t, err)
	assert.NotNil(t, defaultProvider)

	// Test that the provider actually works
	response, err := defaultProvider.Complete(ctx, "Hello, world!")
	assert.NoError(t, err)
	assert.Contains(t, response, "Hello, world!")

	// Test getting nonexistent provider
	_, err = providerRegistry.GetProvider("nonexistent")
	assert.Error(t, err)

	// Cleanup
	err = registry.Shutdown(ctx)
	assert.NoError(t, err)
}

func TestProviderConfigLoading(t *testing.T) {
	// Test configuration loading with different scenarios
	testCases := []struct {
		name           string
		providerName   string
		config         map[string]interface{}
		expectError    bool
		expectedModel  string
		envVar         string
		envValue       string
	}{
		{
			name:         "OpenAI with environment variable",
			providerName: "openai",
			config: map[string]interface{}{
				"model":       "gpt-3.5-turbo",
				"api_key_env": "TEST_OPENAI_KEY",
			},
			expectError:   false,
			expectedModel: "gpt-3.5-turbo",
			envVar:        "TEST_OPENAI_KEY",
			envValue:      "test-key-123",
		},
		{
			name:         "Anthropic with direct API key",
			providerName: "anthropic",
			config: map[string]interface{}{
				"model":   "claude-3-haiku-20240307",
				"api_key": "direct-key-456",
			},
			expectError:   false,
			expectedModel: "claude-3-haiku-20240307",
		},
		{
			name:         "Ollama without API key",
			providerName: "ollama",
			config: map[string]interface{}{
				"model": "llama2",
				"url":   "http://localhost:11434",
			},
			expectError:   false,
			expectedModel: "llama2",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Set environment variable if needed
			if tc.envVar != "" && tc.envValue != "" {
				os.Setenv(tc.envVar, tc.envValue)
				defer os.Unsetenv(tc.envVar)
			}

			// Create provider registry (will be used by the main registry)
			_ = NewProviderRegistry()

			// Create config
			config := &Config{
				Providers: ProviderConfig{
					DefaultProvider: tc.providerName,
					Providers: map[string]interface{}{
						tc.providerName: tc.config,
					},
				},
				// Minimal config for other components
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
						"boltdb":  map[string]interface{}{"path": "./data/memory.db"},
						"chromem": map[string]interface{}{"persistence_path": "./data/vectors"},
					},
				},
			}

			// Create registry and initialize
			registry := NewComponentRegistry()
			ctx := context.Background()

			err := registry.Initialize(ctx, *config)
			if tc.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)

				// Verify provider was registered
				provider, err := registry.Providers().GetProvider(tc.providerName)
				assert.NoError(t, err)
				assert.NotNil(t, provider)

				// Test that it works
				response, err := provider.Complete(ctx, "test prompt")
				assert.NoError(t, err)
				assert.Contains(t, response, "test prompt")
			}
		})
	}
}