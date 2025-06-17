// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package setup

import (
	"context"
	"os"
	"path/filepath"
	"testing"
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

	result, err := detectors.DetectProviders(ctx)
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
}

func TestDetectClaudeCode(t *testing.T) {
	ctx := context.Background()
	detectors := &Detectors{projectPath: "/tmp"}

	// Test without environment variables
	provider, err := detectors.detectClaudeCode(ctx)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Should return nil if no Claude Code environment detected
	if provider != nil && os.Getenv("CLAUDE_CODE_SESSION") == "" && os.Getenv("ANTHROPIC_CLAUDE_CODE") == "" {
		t.Error("Expected nil provider when no Claude Code environment detected")
	}
}

func TestDetectOpenAI(t *testing.T) {
	ctx := context.Background()
	detectors := &Detectors{projectPath: "/tmp"}

	// Save original value
	originalKey := os.Getenv("OPENAI_API_KEY")
	defer func() {
		if originalKey == "" {
			os.Unsetenv("OPENAI_API_KEY")
		} else {
			os.Setenv("OPENAI_API_KEY", originalKey)
		}
	}()

	// Test without API key
	os.Unsetenv("OPENAI_API_KEY")
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
		t.Error("Expected provider when API key is set")
	}
	if provider != nil {
		if provider.Name != "openai" {
			t.Errorf("Expected provider name 'openai', got '%s'", provider.Name)
		}
		if !provider.HasCredentials {
			t.Error("Expected provider to have credentials")
		}
	}

	// Test with invalid API key format
	os.Setenv("OPENAI_API_KEY", "invalid-key")
	provider, err = detectors.detectOpenAI(ctx)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if provider == nil {
		t.Error("Expected provider even with invalid key format")
	}
	if provider != nil && provider.HasCredentials {
		t.Error("Expected provider to not have valid credentials with invalid key format")
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
		t.Error("Expected provider when API key is set")
	}
	if provider != nil {
		if provider.Name != "anthropic" {
			t.Errorf("Expected provider name 'anthropic', got '%s'", provider.Name)
		}
		if !provider.HasCredentials {
			t.Error("Expected provider to have credentials")
		}
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