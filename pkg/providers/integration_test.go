package providers

import (
	"testing"
)

// TestClaudeCodeProviderConfiguration tests Claude Code provider configuration parsing
func TestClaudeCodeProviderConfiguration(t *testing.T) {
	factory := NewFactory()
	
	// Test configuration with all Claude Code options
	claudeConfigMap := map[string]interface{}{
		"model":    "coding-focused",
		"bin_path": "/usr/local/bin/claude-code",
	}

	// Test creating Claude Code client from config
	client, err := factory.CreateClientFromConfig(ProviderClaudeCode, claudeConfigMap)
	if err != nil {
		t.Fatalf("Failed to create Claude Code client: %v", err)
	}

	if client == nil {
		t.Fatalf("Expected Claude Code client but got nil")
	}

	// Test with minimal configuration
	minimalConfig := map[string]interface{}{
		"model": "debugging-focused",
	}

	client2, err := factory.CreateClientFromConfig(ProviderClaudeCode, minimalConfig)
	if err != nil {
		t.Fatalf("Failed to create Claude Code client with minimal config: %v", err)
	}

	if client2 == nil {
		t.Fatalf("Expected Claude Code client but got nil")
	}
}

// TestAllProvidersWithClaudeCode tests that all providers including Claude Code can be registered together
func TestAllProvidersWithClaudeCode(t *testing.T) {
	factory := NewFactory()

	providersConfig := map[string]interface{}{
		"openai": map[string]interface{}{
			"model":   "gpt-4.1",
			"api_key": "test-openai-key",
		},
		"anthropic": map[string]interface{}{
			"model":   "claude-4-sonnet",
			"api_key": "test-anthropic-key",
		},
		// Google provider temporarily removed
		"ollama": map[string]interface{}{
			"model": "llama3.1:8b",
			"url":   "http://localhost:11434",
		},
		"claudecode": map[string]interface{}{
			"model":    "review-focused",
			"bin_path": "claude-code",
		},
	}

	// Create a mock registry
	registry := NewMockProviderRegistry()

	// Register all providers
	err := factory.RegisterProvidersWithRegistry(registry, providersConfig)
	if err != nil {
		t.Fatalf("Failed to register providers: %v", err)
	}

	// Verify all providers were registered
	expectedProviders := []string{"openai", "anthropic", "ollama", "claudecode"}
	
	if len(registry.providers) != len(expectedProviders) {
		t.Errorf("Expected %d providers, got %d", len(expectedProviders), len(registry.providers))
	}

	for _, providerName := range expectedProviders {
		if _, exists := registry.providers[providerName]; !exists {
			t.Errorf("Provider %s not registered", providerName)
		}
	}
}