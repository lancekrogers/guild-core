package providers

import (
	"testing"
)

func TestFactory_CreateClient(t *testing.T) {
	factory := NewFactory()

	tests := []struct {
		name         string
		providerType ProviderType
		apiKey       string
		model        string
		expectError  bool
	}{
		{
			name:         "OpenAI client",
			providerType: ProviderOpenAI,
			apiKey:       "test-key",
			model:        "gpt-4.1",
			expectError:  false,
		},
		{
			name:         "Anthropic client",
			providerType: ProviderAnthropic,
			apiKey:       "test-key",
			model:        "claude-4-sonnet",
			expectError:  false,
		},
		{
			name:         "Google client",
			providerType: ProviderGoogle,
			apiKey:       "test-key",
			model:        "gemini-2.5-flash",
			expectError:  true, // Google provider temporarily removed
		},
		{
			name:         "Ollama client",
			providerType: ProviderOllama,
			apiKey:       "",
			model:        "llama3.1:8b",
			expectError:  false,
		},
		{
			name:         "Claude Code client",
			providerType: ProviderClaudeCode,
			apiKey:       "/usr/local/bin/claude-code",
			model:        "sonnet",
			expectError:  false,
		},
		{
			name:         "Unsupported provider",
			providerType: ProviderType("unsupported"),
			apiKey:       "test-key",
			model:        "test-model",
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := factory.CreateClient(tt.providerType, tt.apiKey, tt.model)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if client == nil {
				t.Errorf("Expected client but got nil")
			}
		})
	}
}

func TestFactory_CreateClientFromConfig(t *testing.T) {
	factory := NewFactory()

	tests := []struct {
		name         string
		providerType ProviderType
		config       map[string]interface{}
		expectError  bool
	}{
		{
			name:         "OpenAI with config",
			providerType: ProviderOpenAI,
			config: map[string]interface{}{
				"model":   "gpt-4.1",
				"api_key": "test-key",
			},
			expectError: false,
		},
		{
			name:         "Claude Code with config",
			providerType: ProviderClaudeCode,
			config: map[string]interface{}{
				"model":    "sonnet",
				"bin_path": "/usr/local/bin/claude-code",
			},
			expectError: false,
		},
		{
			name:         "Ollama with minimal config",
			providerType: ProviderOllama,
			config: map[string]interface{}{
				"model": "llama3.1:8b",
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := factory.CreateClientFromConfig(tt.providerType, tt.config)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if client == nil {
				t.Errorf("Expected client but got nil")
			}
		})
	}
}

// MockProviderRegistry implements ProviderRegistry for testing
type MockProviderRegistry struct {
	providers map[string]LLMClient
	defaultProvider string
}

func NewMockProviderRegistry() *MockProviderRegistry {
	return &MockProviderRegistry{
		providers: make(map[string]LLMClient),
	}
}

func (m *MockProviderRegistry) RegisterProvider(name string, provider LLMClient) error {
	m.providers[name] = provider
	return nil
}

func (m *MockProviderRegistry) SetDefaultProvider(name string) error {
	m.defaultProvider = name
	return nil
}

func TestFactory_RegisterProvidersWithRegistry(t *testing.T) {
	factory := NewFactory()
	registry := NewMockProviderRegistry()

	providersConfig := map[string]interface{}{
		"openai": map[string]interface{}{
			"model":   "gpt-4.1",
			"api_key": "test-key",
		},
		"claudecode": map[string]interface{}{
			"model":    "sonnet",
			"bin_path": "/usr/local/bin/claude-code",
		},
	}

	err := factory.RegisterProvidersWithRegistry(registry, providersConfig)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Check that providers were registered
	if len(registry.providers) != 2 {
		t.Errorf("Expected 2 providers, got %d", len(registry.providers))
	}

	if _, exists := registry.providers["openai"]; !exists {
		t.Errorf("OpenAI provider not registered")
	}

	if _, exists := registry.providers["claudecode"]; !exists {
		t.Errorf("Claude Code provider not registered")
	}
}

func TestFactory_RegisterProvidersWithRegistry_UnknownProvider(t *testing.T) {
	factory := NewFactory()
	registry := NewMockProviderRegistry()

	providersConfig := map[string]interface{}{
		"unknown": map[string]interface{}{
			"model":   "test-model",
			"api_key": "test-key",
		},
	}

	err := factory.RegisterProvidersWithRegistry(registry, providersConfig)
	if err == nil {
		t.Errorf("Expected error for unknown provider but got none")
	}
}