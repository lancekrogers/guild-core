package registry

import (
	"context"
	"testing"
)

func TestPromptRegistry(t *testing.T) {
	registry := NewPromptRegistry()

	// Test registering a provider
	provider := &mockPromptProvider{
		prompts: map[string]*Prompt{
			"test-prompt": {
				ID:       "test-prompt",
				Version:  "1.0.0",
				Category: "test",
				Content:  "Test prompt content",
			},
		},
	}

	err := registry.Register("test-provider", provider)
	if err != nil {
		t.Fatalf("Failed to register provider: %v", err)
	}

	// Test duplicate registration
	err = registry.Register("test-provider", provider)
	if err == nil {
		t.Error("Expected error when registering duplicate provider")
	}

	// Test getting a registered provider
	retrieved, err := registry.Get("test-provider")
	if err != nil {
		t.Fatalf("Failed to get provider: %v", err)
	}
	if retrieved != provider {
		t.Error("Retrieved provider doesn't match registered provider")
	}

	// Test getting non-existent provider
	_, err = registry.Get("non-existent")
	if err == nil {
		t.Error("Expected error when getting non-existent provider")
	}

	// Test listing providers
	providers := registry.List()
	if len(providers) != 1 {
		t.Errorf("Expected 1 provider, got %d", len(providers))
	}
	if providers[0] != "test-provider" {
		t.Errorf("Expected provider name 'test-provider', got %s", providers[0])
	}
}

func TestDefaultPromptProvider(t *testing.T) {
	// This test requires the actual prompt files to be available
	// It will test the integration with the internal prompt manager
	
	provider, err := NewDefaultPromptProvider()
	if err != nil {
		t.Fatalf("Failed to create default prompt provider: %v", err)
	}

	// Test listing prompts
	prompts, err := provider.ListPrompts()
	if err != nil {
		t.Fatalf("Failed to list prompts: %v", err)
	}

	if len(prompts) == 0 {
		t.Error("Expected at least one prompt")
	}

	// Verify prompt metadata
	for _, meta := range prompts {
		if meta.ID == "" {
			t.Error("Prompt metadata missing ID")
		}
		if meta.Version == "" {
			t.Error("Prompt metadata missing version")
		}
		if meta.Category == "" {
			t.Error("Prompt metadata missing category")
		}
	}

	// Test validating prompt data
	validData := map[string]interface{}{
		"Description": "Test description",
	}
	err = provider.ValidatePrompt("objective.creation", validData)
	if err != nil {
		t.Errorf("Expected valid data to pass validation: %v", err)
	}

	invalidData := map[string]interface{}{}
	err = provider.ValidatePrompt("objective.creation", invalidData)
	if err == nil {
		t.Error("Expected invalid data to fail validation")
	}

	// Test rendering prompt
	result, err := provider.RenderPrompt("objective.creation", validData)
	if err != nil {
		t.Fatalf("Failed to render prompt: %v", err)
	}
	if result == "" {
		t.Error("Expected non-empty rendered prompt")
	}
}

func TestComponentRegistry_PromptIntegration(t *testing.T) {
	// Create a new component registry
	registry := NewComponentRegistry()

	// Initialize with empty config
	err := registry.Initialize(context.Background(), Config{})
	if err != nil {
		t.Fatalf("Failed to initialize registry: %v", err)
	}

	// Get the prompt registry
	promptRegistry := registry.Prompts()
	if promptRegistry == nil {
		t.Fatal("Expected non-nil prompt registry")
	}

	// Verify default provider is registered
	providers := promptRegistry.List()
	found := false
	for _, name := range providers {
		if name == "default" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected 'default' prompt provider to be registered")
	}

	// Test getting the default provider
	provider, err := promptRegistry.Get("default")
	if err != nil {
		t.Fatalf("Failed to get default provider: %v", err)
	}
	if provider == nil {
		t.Fatal("Expected non-nil provider")
	}

	// Test using the provider through the registry
	prompts, err := provider.ListPrompts()
	if err != nil {
		t.Fatalf("Failed to list prompts: %v", err)
	}
	if len(prompts) == 0 {
		t.Error("Expected at least one prompt from default provider")
	}
}

// Mock prompt provider for testing
type mockPromptProvider struct {
	prompts map[string]*Prompt
}

func (m *mockPromptProvider) GetPrompt(id string, version string) (*Prompt, error) {
	prompt, exists := m.prompts[id]
	if !exists {
		return nil, ErrComponentNotFound
	}
	return prompt, nil
}

func (m *mockPromptProvider) ListPrompts() ([]PromptMetadata, error) {
	var metadata []PromptMetadata
	for _, prompt := range m.prompts {
		if prompt.Metadata != nil {
			metadata = append(metadata, *prompt.Metadata)
		}
	}
	return metadata, nil
}

func (m *mockPromptProvider) ValidatePrompt(id string, data interface{}) error {
	// Simple validation - just check if prompt exists
	_, exists := m.prompts[id]
	if !exists {
		return ErrComponentNotFound
	}
	return nil
}

func (m *mockPromptProvider) RenderPrompt(id string, data interface{}) (string, error) {
	prompt, exists := m.prompts[id]
	if !exists {
		return "", ErrComponentNotFound
	}
	// Simple rendering - just return content
	return prompt.Content, nil
}