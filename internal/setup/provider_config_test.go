// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package setup

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/lancekrogers/guild-core/pkg/gerror"
)

func TestNewProviderConfig(t *testing.T) {
	ctx := context.Background()
	projectPath := "/tmp"

	config, err := NewProviderConfig(ctx, projectPath)
	if err != nil {
		t.Fatalf("Failed to create provider config: %v", err)
	}

	if config == nil {
		t.Fatal("Provider config is nil")
	}
	if config.projectPath != projectPath {
		t.Error("Project path not set correctly")
	}
	if config.factory == nil {
		t.Error("Factory not initialized")
	}
}

func TestNewProviderConfigValidation(t *testing.T) {
	ctx := context.Background()

	// Test with empty project path
	_, err := NewProviderConfig(ctx, "")
	if err == nil {
		t.Fatal("Expected error when project path is empty")
	}

	if gerr, ok := err.(*gerror.GuildError); ok {
		if gerr.Code != gerror.ErrCodeValidation {
			t.Errorf("Expected ErrCodeValidation, got %s", gerr.Code)
		}
		if gerr.Component != "ProviderConfiguration" {
			t.Errorf("Expected component 'ProviderConfiguration', got '%s'", gerr.Component)
		}
	} else {
		t.Error("Expected gerror.GuildError")
	}

	// Test with cancelled context
	cancelledCtx, cancel := context.WithCancel(ctx)
	cancel()

	_, err = NewProviderConfig(cancelledCtx, "/tmp")
	if err == nil {
		t.Fatal("Expected error when context is cancelled")
	}

	if gerr, ok := err.(*gerror.GuildError); ok {
		if gerr.Code != gerror.ErrCodeCancelled {
			t.Errorf("Expected ErrCodeCancelled, got %s", gerr.Code)
		}
	} else {
		t.Error("Expected gerror.GuildError")
	}
}

func TestValidateProviderContextCancellation(t *testing.T) {
	config := &ProviderConfig{projectPath: "/tmp"}
	provider := DetectedProvider{Name: "openai"}

	// Create a context that's already cancelled
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := config.ValidateProvider(ctx, provider)
	if err == nil {
		t.Fatal("Expected error when context is cancelled")
	}

	if gerr, ok := err.(*gerror.GuildError); ok {
		if gerr.Code != gerror.ErrCodeCancelled {
			t.Errorf("Expected ErrCodeCancelled, got %s", gerr.Code)
		}
		if gerr.Component != "ProviderConfiguration" {
			t.Errorf("Expected component 'ProviderConfiguration', got '%s'", gerr.Component)
		}
	} else {
		t.Error("Expected gerror.GuildError")
	}
}

func TestValidateProviderUnsupported(t *testing.T) {
	ctx := context.Background()
	config := &ProviderConfig{projectPath: "/tmp"}
	provider := DetectedProvider{Name: "unsupported_provider"}

	_, err := config.ValidateProvider(ctx, provider)
	if err == nil {
		t.Fatal("Expected error for unsupported provider")
	}

	if gerr, ok := err.(*gerror.GuildError); ok {
		if gerr.Code != gerror.ErrCodeValidation {
			t.Errorf("Expected ErrCodeValidation, got %s", gerr.Code)
		}
		if gerr.Component != "ProviderConfiguration" {
			t.Errorf("Expected component 'ProviderConfiguration', got '%s'", gerr.Component)
		}
	} else {
		t.Error("Expected gerror.GuildError")
	}
}

func TestValidateClaudeCodeProvider(t *testing.T) {
	ctx := context.Background()
	config := &ProviderConfig{projectPath: "/tmp"}

	// Save original values
	originalSession := os.Getenv("CLAUDE_CODE_SESSION")
	defer func() {
		if originalSession == "" {
			os.Unsetenv("CLAUDE_CODE_SESSION")
		} else {
			os.Setenv("CLAUDE_CODE_SESSION", originalSession)
		}
	}()

	provider := DetectedProvider{
		Name:           "claude_code",
		Type:           "cloud",
		HasCredentials: true,
	}

	// Test without session
	os.Unsetenv("CLAUDE_CODE_SESSION")
	validation, err := config.ValidateProvider(ctx, provider)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if validation == nil {
		t.Fatal("Validation result is nil")
	}
	if !validation.IsValid {
		t.Error("Expected Claude Code to be valid")
	}
	if validation.Settings["type"] != "claude_code" {
		t.Errorf("Expected type 'claude_code', got '%s'", validation.Settings["type"])
	}
	if len(validation.Models) == 0 {
		t.Error("Expected models to be available")
	}

	// Test with session
	os.Setenv("CLAUDE_CODE_SESSION", "test-session-123")
	validation, err = config.ValidateProvider(ctx, provider)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if validation.Settings["session_id"] != "test-session-123" {
		t.Errorf("Expected session_id 'test-session-123', got '%s'", validation.Settings["session_id"])
	}
}

func TestValidateOpenAIProvider(t *testing.T) {
	ctx := context.Background()
	config := &ProviderConfig{projectPath: "/tmp"}

	// Save original values
	originalKey := os.Getenv("OPENAI_API_KEY")
	originalOrg := os.Getenv("OPENAI_ORG_ID")
	defer func() {
		if originalKey == "" {
			os.Unsetenv("OPENAI_API_KEY")
		} else {
			os.Setenv("OPENAI_API_KEY", originalKey)
		}
		if originalOrg == "" {
			os.Unsetenv("OPENAI_ORG_ID")
		} else {
			os.Setenv("OPENAI_ORG_ID", originalOrg)
		}
	}()

	provider := DetectedProvider{
		Name:           "openai",
		Type:           "cloud",
		HasCredentials: true,
	}

	// Test without API key
	os.Unsetenv("OPENAI_API_KEY")
	os.Unsetenv("OPENAI_ORG_ID")
	validation, err := config.ValidateProvider(ctx, provider)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if validation == nil {
		t.Fatal("Validation result is nil")
	}
	if validation.IsValid {
		t.Error("Expected OpenAI to be invalid without API key")
	}
	if validation.Error != "OPENAI_API_KEY environment variable not set" {
		t.Errorf("Expected error about missing API key, got '%s'", validation.Error)
	}

	// Test with invalid API key format
	os.Setenv("OPENAI_API_KEY", "invalid-key")
	validation, err = config.ValidateProvider(ctx, provider)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if validation.IsValid {
		t.Error("Expected OpenAI to be invalid with invalid key format")
	}
	if validation.Error != "Invalid OpenAI API key format" {
		t.Errorf("Expected error about invalid format, got '%s'", validation.Error)
	}

	// Test with valid API key
	os.Setenv("OPENAI_API_KEY", "sk-test123456789")
	validation, err = config.ValidateProvider(ctx, provider)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if !validation.IsValid {
		t.Error("Expected OpenAI to be valid with proper API key")
	}
	if validation.Settings["type"] != "openai" {
		t.Errorf("Expected type 'openai', got '%s'", validation.Settings["type"])
	}
	if validation.Settings["base_url"] != "https://api.openai.com/v1" {
		t.Errorf("Expected base_url 'https://api.openai.com/v1', got '%s'", validation.Settings["base_url"])
	}
	if len(validation.Models) == 0 {
		t.Error("Expected models to be available")
	}

	// Test with organization ID
	os.Setenv("OPENAI_ORG_ID", "org-12345")
	validation, err = config.ValidateProvider(ctx, provider)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if validation.Settings["organization"] != "org-12345" {
		t.Errorf("Expected organization 'org-12345', got '%s'", validation.Settings["organization"])
	}
}

func TestValidateAnthropicProvider(t *testing.T) {
	ctx := context.Background()
	config := &ProviderConfig{projectPath: "/tmp"}

	// Save original value
	originalKey := os.Getenv("ANTHROPIC_API_KEY")
	defer func() {
		if originalKey == "" {
			os.Unsetenv("ANTHROPIC_API_KEY")
		} else {
			os.Setenv("ANTHROPIC_API_KEY", originalKey)
		}
	}()

	provider := DetectedProvider{
		Name:           "anthropic",
		Type:           "cloud",
		HasCredentials: true,
	}

	// Test without API key
	os.Unsetenv("ANTHROPIC_API_KEY")
	validation, err := config.ValidateProvider(ctx, provider)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if validation.IsValid {
		t.Error("Expected Anthropic to be invalid without API key")
	}

	// Test with invalid API key format
	os.Setenv("ANTHROPIC_API_KEY", "invalid-key")
	validation, err = config.ValidateProvider(ctx, provider)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if validation.IsValid {
		t.Error("Expected Anthropic to be invalid with invalid key format")
	}

	// Test with valid API key
	os.Setenv("ANTHROPIC_API_KEY", "sk-ant-test123456789")
	validation, err = config.ValidateProvider(ctx, provider)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if !validation.IsValid {
		t.Error("Expected Anthropic to be valid with proper API key")
	}
	if validation.Settings["type"] != "anthropic" {
		t.Errorf("Expected type 'anthropic', got '%s'", validation.Settings["type"])
	}
	if validation.Settings["base_url"] != "https://api.anthropic.com" {
		t.Errorf("Expected base_url 'https://api.anthropic.com', got '%s'", validation.Settings["base_url"])
	}
}

func TestValidateOllamaProvider(t *testing.T) {
	ctx := context.Background()
	config := &ProviderConfig{projectPath: "/tmp"}

	provider := DetectedProvider{
		Name:           "ollama",
		Type:           "local",
		HasCredentials: true,
		IsLocal:        true,
	}

	// Test with Ollama not running (empty endpoint)
	provider.Endpoint = ""
	validation, err := config.ValidateProvider(ctx, provider)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if validation.IsValid {
		t.Error("Expected Ollama to be invalid when not running")
	}
	if validation.Error != "Ollama service is not running" {
		t.Errorf("Expected error about service not running, got '%s'", validation.Error)
	}
	if validation.Warning == "" {
		t.Error("Expected warning about starting Ollama")
	}

	// Test with Ollama running (with endpoint)
	provider.Endpoint = "http://localhost:11434"
	validation, err = config.ValidateProvider(ctx, provider)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Note: This might fail if Ollama is not actually running, but that's expected
	// The test validates the logic, not the actual connection
	if validation.Settings["type"] != "ollama" {
		t.Errorf("Expected type 'ollama', got '%s'", validation.Settings["type"])
	}
	if validation.Settings["base_url"] != provider.Endpoint {
		t.Errorf("Expected base_url '%s', got '%s'", provider.Endpoint, validation.Settings["base_url"])
	}
}

func TestGetProviderRecommendations(t *testing.T) {
	ctx := context.Background()
	config := &ProviderConfig{projectPath: "/tmp"}

	// Test with no providers
	recommendations, err := config.ProviderRecommendations(ctx, []DetectedProvider{})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if recommendations == nil {
		t.Fatal("Recommendations is nil")
	}
	if recommendations.Primary != "" {
		t.Error("Expected no primary provider with empty list")
	}
	if len(recommendations.Suggestions) == 0 {
		t.Error("Expected suggestions when no providers available")
	}

	// Test with multiple cloud providers
	providers := []DetectedProvider{
		{
			Name:           "anthropic",
			Type:           "cloud",
			HasCredentials: true,
			IsLocal:        false,
		},
		{
			Name:           "openai",
			Type:           "cloud",
			HasCredentials: true,
			IsLocal:        false,
		},
		{
			Name:           "ollama",
			Type:           "local",
			HasCredentials: true,
			IsLocal:        true,
		},
	}

	recommendations, err = config.ProviderRecommendations(ctx, providers)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Should prefer Anthropic as primary
	if recommendations.Primary != "anthropic" {
		t.Errorf("Expected 'anthropic' as primary, got '%s'", recommendations.Primary)
	}
	// Should have OpenAI as secondary
	if recommendations.Secondary != "openai" {
		t.Errorf("Expected 'openai' as secondary, got '%s'", recommendations.Secondary)
	}
	// Should have Ollama as local
	if recommendations.Local != "ollama" {
		t.Errorf("Expected 'ollama' as local, got '%s'", recommendations.Local)
	}
	// Should have reasoning
	if len(recommendations.Reasoning) == 0 {
		t.Error("Expected reasoning for recommendations")
	}
}

func TestGetProviderRecommendationsContextCancellation(t *testing.T) {
	config := &ProviderConfig{projectPath: "/tmp"}

	// Create a context that's already cancelled
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := config.ProviderRecommendations(ctx, []DetectedProvider{})
	if err == nil {
		t.Fatal("Expected error when context is cancelled")
	}

	if gerr, ok := err.(*gerror.GuildError); ok {
		if gerr.Code != gerror.ErrCodeCancelled {
			t.Errorf("Expected ErrCodeCancelled, got %s", gerr.Code)
		}
	} else {
		t.Error("Expected gerror.GuildError")
	}
}

func TestProviderConfigTimeout(t *testing.T) {
	config := &ProviderConfig{projectPath: "/tmp"}

	// Create a context with very short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()

	// Give time for context to timeout
	time.Sleep(1 * time.Millisecond)

	provider := DetectedProvider{Name: "openai"}
	_, err := config.ValidateProvider(ctx, provider)
	if err == nil {
		t.Fatal("Expected error when context times out")
	}

	if gerr, ok := err.(*gerror.GuildError); ok {
		if gerr.Code != gerror.ErrCodeCancelled {
			t.Errorf("Expected ErrCodeCancelled, got %s", gerr.Code)
		}
	} else {
		t.Error("Expected gerror.GuildError")
	}
}

// TestValidateAllProviderTypes tests validation of all supported provider types
func TestValidateAllProviderTypes(t *testing.T) {
	ctx := context.Background()
	config := &ProviderConfig{projectPath: "/tmp"}

	// Save all original environment variables
	envVars := map[string]string{
		"OPENAI_API_KEY":      os.Getenv("OPENAI_API_KEY"),
		"ANTHROPIC_API_KEY":   os.Getenv("ANTHROPIC_API_KEY"),
		"DEEPSEEK_API_KEY":    os.Getenv("DEEPSEEK_API_KEY"),
		"DEEPINFRA_API_KEY":   os.Getenv("DEEPINFRA_API_KEY"),
		"ORA_API_KEY":         os.Getenv("ORA_API_KEY"),
		"CLAUDE_CODE_SESSION": os.Getenv("CLAUDE_CODE_SESSION"),
	}

	// Restore environment after test
	defer func() {
		for key, value := range envVars {
			if value == "" {
				os.Unsetenv(key)
			} else {
				os.Setenv(key, value)
			}
		}
	}()

	// Set up valid API keys for testing
	os.Setenv("OPENAI_API_KEY", "sk-test123")
	os.Setenv("ANTHROPIC_API_KEY", "sk-ant-test123")
	os.Setenv("DEEPSEEK_API_KEY", "test123")
	os.Setenv("DEEPINFRA_API_KEY", "test123")
	os.Setenv("ORA_API_KEY", "test123")
	os.Setenv("CLAUDE_CODE_SESSION", "test-session")

	// Test all provider types for basic validation functionality
	providerTypes := []string{
		"claude_code",
		"ollama",
		"openai",
		"anthropic",
		"deepseek",
		"deepinfra",
		"ora",
	}

	for _, providerType := range providerTypes {
		provider := DetectedProvider{
			Name:           providerType,
			Type:           "cloud",
			HasCredentials: true,
		}

		if providerType == "ollama" {
			provider.Type = "local"
			provider.IsLocal = true
			provider.Endpoint = "http://localhost:11434"
		}

		validation, err := config.ValidateProvider(ctx, provider)
		if err != nil {
			t.Fatalf("Unexpected error for %s: %v", providerType, err)
		}

		if validation == nil {
			t.Fatalf("Validation result is nil for %s", providerType)
		}

		// All validations should return settings
		if validation.Settings == nil {
			t.Errorf("Settings is nil for %s", providerType)
		}
		if validation.Settings["type"] != providerType {
			t.Errorf("Expected type '%s', got '%s'", providerType, validation.Settings["type"])
		}
	}
}
