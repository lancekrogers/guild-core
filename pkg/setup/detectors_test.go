// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package setup

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/guild-ventures/guild-core/pkg/gerror"
)

func TestNewDetectors(t *testing.T) {
	ctx := context.Background()
	projectPath := "/tmp"

	detectors, err := NewDetectors(ctx, projectPath)
	if err != nil {
		t.Fatalf("Failed to create detectors: %v", err)
	}

	if detectors == nil {
		t.Fatal("Detectors is nil")
	}
	if detectors.projectPath != projectPath {
		t.Error("Project path not set correctly")
	}
}

func TestDetectProviders(t *testing.T) {
	ctx := context.Background()
	detectors := &Detectors{projectPath: "/tmp"}

	result, err := detectors.Providers(ctx)
	if err != nil {
		t.Fatalf("Failed to detect providers: %v", err)
	}

	if result == nil {
		t.Fatal("Detection result is nil")
	}

	// Should always have some results (at least empty lists)
	if result.Available == nil {
		t.Error("Available providers list is nil")
	}
	if result.Missing == nil {
		t.Error("Missing providers list is nil")
	}

	// Should detect at least some providers (available or missing)
	totalProviders := len(result.Available) + len(result.Missing)
	expectedProviders := 7 // claude_code, ollama, openai, anthropic, deepseek, deepinfra, ora
	if totalProviders != expectedProviders {
		t.Errorf("Expected %d total providers, got %d", expectedProviders, totalProviders)
	}
}

func TestDetectClaudeCode(t *testing.T) {
	ctx := context.Background()
	detectors := &Detectors{projectPath: "/tmp"}

	// Save original values
	originalSession := os.Getenv("CLAUDE_CODE_SESSION")
	originalAnthropic := os.Getenv("ANTHROPIC_CLAUDE_CODE")
	defer func() {
		if originalSession == "" {
			os.Unsetenv("CLAUDE_CODE_SESSION")
		} else {
			os.Setenv("CLAUDE_CODE_SESSION", originalSession)
		}
		if originalAnthropic == "" {
			os.Unsetenv("ANTHROPIC_CLAUDE_CODE")
		} else {
			os.Setenv("ANTHROPIC_CLAUDE_CODE", originalAnthropic)
		}
	}()

	// Test without environment variables
	os.Unsetenv("CLAUDE_CODE_SESSION")
	os.Unsetenv("ANTHROPIC_CLAUDE_CODE")
	provider, err := detectors.detectClaudeCode(ctx)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Should return nil if no Claude Code environment detected
	if provider != nil {
		t.Error("Expected nil provider when no Claude Code environment detected")
	}

	// Test with Claude Code session
	os.Setenv("CLAUDE_CODE_SESSION", "test-session-123")
	provider, err = detectors.detectClaudeCode(ctx)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if provider == nil {
		t.Fatal("Expected provider when CLAUDE_CODE_SESSION is set")
	}
	if provider.Name != "claude_code" {
		t.Errorf("Expected provider name 'claude_code', got '%s'", provider.Name)
	}
	if provider.Type != "cloud" {
		t.Errorf("Expected provider type 'cloud', got '%s'", provider.Type)
	}
	if !provider.HasCredentials {
		t.Error("Expected provider to have credentials")
	}

	// Test with Anthropic Claude Code environment
	os.Unsetenv("CLAUDE_CODE_SESSION")
	os.Setenv("ANTHROPIC_CLAUDE_CODE", "true")
	provider, err = detectors.detectClaudeCode(ctx)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if provider == nil {
		t.Fatal("Expected provider when ANTHROPIC_CLAUDE_CODE is set")
	}
	if provider.Name != "claude_code" {
		t.Errorf("Expected provider name 'claude_code', got '%s'", provider.Name)
	}
}

func TestDetectOpenAI(t *testing.T) {
	ctx := context.Background()
	detectors := &Detectors{projectPath: "/tmp"}

	// Save original value
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

	// Test without API key
	os.Unsetenv("OPENAI_API_KEY")
	os.Unsetenv("OPENAI_ORG_ID")
	provider, err := detectors.detectOpenAI(ctx)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if provider != nil {
		t.Error("Expected nil provider when no API key")
	}

	// Test with valid API key
	os.Setenv("OPENAI_API_KEY", "sk-test123456789")
	provider, err = detectors.detectOpenAI(ctx)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if provider == nil {
		t.Fatal("Expected provider when API key is set")
	}
	if provider.Name != "openai" {
		t.Errorf("Expected provider name 'openai', got '%s'", provider.Name)
	}
	if provider.Type != "cloud" {
		t.Errorf("Expected provider type 'cloud', got '%s'", provider.Type)
	}
	if !provider.HasCredentials {
		t.Error("Expected provider to have credentials")
	}
	if provider.Endpoint != "https://api.openai.com" {
		t.Errorf("Expected endpoint 'https://api.openai.com', got '%s'", provider.Endpoint)
	}

	// Test with valid API key and organization ID
	os.Setenv("OPENAI_ORG_ID", "org-12345")
	provider, err = detectors.detectOpenAI(ctx)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if provider == nil {
		t.Fatal("Expected provider when API key and org ID are set")
	}
	if !provider.HasCredentials {
		t.Error("Expected provider to have credentials")
	}
	if provider.Notes != "API key available with organization ID" {
		t.Errorf("Expected notes about organization ID, got '%s'", provider.Notes)
	}

	// Test with invalid API key format
	os.Unsetenv("OPENAI_ORG_ID")
	os.Setenv("OPENAI_API_KEY", "invalid-key")
	provider, err = detectors.detectOpenAI(ctx)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if provider == nil {
		t.Fatal("Expected provider even with invalid key format")
	}
	if provider.HasCredentials {
		t.Error("Expected provider to not have valid credentials with invalid key format")
	}
	if provider.Notes != "Invalid API key format" {
		t.Errorf("Expected notes about invalid format, got '%s'", provider.Notes)
	}
}

func TestDetectAnthropic(t *testing.T) {
	ctx := context.Background()
	detectors := &Detectors{projectPath: "/tmp"}

	// Save original value
	originalKey := os.Getenv("ANTHROPIC_API_KEY")
	defer func() {
		if originalKey == "" {
			os.Unsetenv("ANTHROPIC_API_KEY")
		} else {
			os.Setenv("ANTHROPIC_API_KEY", originalKey)
		}
	}()

	// Test without API key
	os.Unsetenv("ANTHROPIC_API_KEY")
	provider, err := detectors.detectAnthropic(ctx)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if provider != nil {
		t.Error("Expected nil provider when no API key")
	}

	// Test with valid API key
	os.Setenv("ANTHROPIC_API_KEY", "sk-ant-test123456789")
	provider, err = detectors.detectAnthropic(ctx)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if provider == nil {
		t.Fatal("Expected provider when API key is set")
	}
	if provider.Name != "anthropic" {
		t.Errorf("Expected provider name 'anthropic', got '%s'", provider.Name)
	}
	if provider.Type != "cloud" {
		t.Errorf("Expected provider type 'cloud', got '%s'", provider.Type)
	}
	if !provider.HasCredentials {
		t.Error("Expected provider to have credentials")
	}
	if provider.Endpoint != "https://api.anthropic.com" {
		t.Errorf("Expected endpoint 'https://api.anthropic.com', got '%s'", provider.Endpoint)
	}
	if provider.Notes != "API key available" {
		t.Errorf("Expected notes 'API key available', got '%s'", provider.Notes)
	}

	// Test with invalid API key format
	os.Setenv("ANTHROPIC_API_KEY", "invalid-key")
	provider, err = detectors.detectAnthropic(ctx)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if provider == nil {
		t.Fatal("Expected provider even with invalid key format")
	}
	if provider.HasCredentials {
		t.Error("Expected provider to not have valid credentials with invalid key format")
	}
	if provider.Notes != "Invalid API key format" {
		t.Errorf("Expected notes about invalid format, got '%s'", provider.Notes)
	}
}

func TestDetectProjectContext(t *testing.T) {
	ctx := context.Background()
	
	// Create temporary directory with go.mod file
	tempDir, err := os.MkdirTemp("", "guild-setup-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	detectors := &Detectors{projectPath: tempDir}

	// Test with no project files
	context, err := detectors.DetectProjectContext(ctx)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if context == nil {
		t.Fatal("Project context is nil")
	}
	if context.Language != "unknown" {
		t.Errorf("Expected language 'unknown', got '%s'", context.Language)
	}

	// Create go.mod file
	goModPath := filepath.Join(tempDir, "go.mod")
	if err := os.WriteFile(goModPath, []byte("module test"), 0644); err != nil {
		t.Fatalf("Failed to create go.mod: %v", err)
	}

	// Test with go.mod
	context, err = detectors.DetectProjectContext(ctx)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if context.Language != "go" {
		t.Errorf("Expected language 'go', got '%s'", context.Language)
	}
	if len(context.Suggestions) == 0 {
		t.Error("Expected suggestions for Go project")
	}
}

// TestDetectProvidersContextCancellation tests context cancellation handling
func TestDetectProvidersContextCancellation(t *testing.T) {
	detectors := &Detectors{projectPath: "/tmp"}

	// Create a context that's already cancelled
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := detectors.Providers(ctx)
	if err == nil {
		t.Fatal("Expected error when context is cancelled")
	}

	// Should be a gerror with cancellation code
	if gerr, ok := err.(*gerror.GuildError); ok {
		if gerr.Code != gerror.ErrCodeCancelled {
			t.Errorf("Expected ErrCodeCancelled, got %s", gerr.Code)
		}
		if gerr.Component != "ProviderDetection" {
			t.Errorf("Expected component 'ProviderDetection', got '%s'", gerr.Component)
		}
	} else {
		t.Error("Expected gerror.GuildError")
	}
}

// TestDetectProvidersTimeout tests timeout scenarios
func TestDetectProvidersTimeout(t *testing.T) {
	detectors := &Detectors{projectPath: "/tmp"}

	// Create a context with very short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()

	// Give time for context to timeout
	time.Sleep(1 * time.Millisecond)

	_, err := detectors.Providers(ctx)
	if err == nil {
		t.Fatal("Expected error when context times out")
	}

	// Should be a gerror with cancellation/timeout code
	if gerr, ok := err.(*gerror.GuildError); ok {
		if gerr.Code != gerror.ErrCodeCancelled {
			t.Errorf("Expected ErrCodeCancelled, got %s", gerr.Code)
		}
	} else {
		t.Error("Expected gerror.GuildError")
	}
}

// TestNewDetectorsValidation tests input validation
func TestNewDetectorsValidation(t *testing.T) {
	ctx := context.Background()

	// Test with empty project path
	_, err := NewDetectors(ctx, "")
	if err == nil {
		t.Fatal("Expected error when project path is empty")
	}

	if gerr, ok := err.(*gerror.GuildError); ok {
		if gerr.Code != gerror.ErrCodeValidation {
			t.Errorf("Expected ErrCodeValidation, got %s", gerr.Code)
		}
	} else {
		t.Error("Expected gerror.GuildError")
	}

	// Test with cancelled context
	cancelledCtx, cancel := context.WithCancel(ctx)
	cancel()

	_, err = NewDetectors(cancelledCtx, "/tmp")
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

// TestDetectOllamaFunctionality tests Ollama detection scenarios
func TestDetectOllamaFunctionality(t *testing.T) {
	ctx := context.Background()
	detectors := &Detectors{projectPath: "/tmp"}

	// This test will pass regardless of whether Ollama is installed
	// It tests the detection logic functionality
	provider, err := detectors.detectOllama(ctx)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// If Ollama is detected, validate the provider structure
	if provider != nil {
		if provider.Name != "ollama" {
			t.Errorf("Expected provider name 'ollama', got '%s'", provider.Name)
		}
		if provider.Type != "local" {
			t.Errorf("Expected provider type 'local', got '%s'", provider.Type)
		}
		if !provider.IsLocal {
			t.Error("Expected provider to be marked as local")
		}
		if !provider.HasCredentials {
			t.Error("Expected Ollama to have credentials (no auth required)")
		}
		// Version should be set (either "installed" or actual version)
		if provider.Version == "" {
			t.Error("Expected version to be set")
		}
		// Should have helpful notes
		if provider.Notes == "" {
			t.Error("Expected notes to be provided")
		}
	}
}

// TestDetectAllProviderTypes tests detection of all provider types
func TestDetectAllProviderTypes(t *testing.T) {
	ctx := context.Background()
	detectors := &Detectors{projectPath: "/tmp"}

	// Save all original environment variables
	envVars := map[string]string{
		"OPENAI_API_KEY":        os.Getenv("OPENAI_API_KEY"),
		"ANTHROPIC_API_KEY":     os.Getenv("ANTHROPIC_API_KEY"),
		"DEEPSEEK_API_KEY":      os.Getenv("DEEPSEEK_API_KEY"),
		"DEEPINFRA_API_KEY":     os.Getenv("DEEPINFRA_API_KEY"),
		"ORA_API_KEY":           os.Getenv("ORA_API_KEY"),
		"CLAUDE_CODE_SESSION":   os.Getenv("CLAUDE_CODE_SESSION"),
		"ANTHROPIC_CLAUDE_CODE": os.Getenv("ANTHROPIC_CLAUDE_CODE"),
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

	result, err := detectors.Providers(ctx)
	if err != nil {
		t.Fatalf("Failed to detect providers: %v", err)
	}

	// Should detect multiple providers
	if len(result.Available) < 2 {
		t.Errorf("Expected at least 2 available providers, got %d", len(result.Available))
	}

	// Check that each provider has required fields
	for _, provider := range result.Available {
		if provider.Name == "" {
			t.Error("Provider name is empty")
		}
		if provider.Type == "" {
			t.Error("Provider type is empty")
		}
		if provider.Endpoint == "" {
			t.Error("Provider endpoint is empty")
		}
		// All providers in this test should have credentials
		if !provider.HasCredentials {
			t.Errorf("Provider %s should have credentials", provider.Name)
		}
	}
}