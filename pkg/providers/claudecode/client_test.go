// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package claudecode

import (
	"context"
	"testing"
	"time"

	"github.com/lancekrogers/guild-core/pkg/providers/interfaces"
	"github.com/lancekrogers/claude-code-go/pkg/claude"
)

func TestNewClient(t *testing.T) {
	tests := []struct {
		name    string
		binPath string
		model   string
	}{
		{
			name:    "Default client",
			binPath: "claude-code",
			model:   "sonnet",
		},
		{
			name:    "Full path client",
			binPath: "/usr/local/bin/claude-code",
			model:   "opus",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewClient(tt.binPath, tt.model)
			if client == nil {
				t.Errorf("Expected client but got nil")
			}

			if client.binPath != tt.binPath {
				t.Errorf("Expected binPath %s, got %s", tt.binPath, client.binPath)
			}
		})
	}
}

func TestClient_GetBinPath(t *testing.T) {
	tests := []struct {
		name     string
		binPath  string
		expected string
	}{
		{
			name:     "Default path",
			binPath:  "claude-code",
			expected: "claude-code",
		},
		{
			name:     "Full path",
			binPath:  "/usr/local/bin/claude-code",
			expected: "/usr/local/bin/claude-code",
		},
		{
			name:     "Empty path defaults to claude",
			binPath:  "",
			expected: "claude",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewClient(tt.binPath, "coding-focused")
			result := client.GetBinPath()
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestClient_SetSystemPrompt(t *testing.T) {
	client := NewClient("claude-code", "")

	testPrompt := "You are a helpful assistant"
	client.SetSystemPrompt(testPrompt)

	opts := client.GetDefaultOptions()
	if opts.SystemPrompt != testPrompt {
		t.Errorf("Expected system prompt '%s', got '%s'", testPrompt, opts.SystemPrompt)
	}
}

func TestClient_ModelConfiguration(t *testing.T) {
	tests := []struct {
		name           string
		model          string
		expectedPrompt string
		expectedModel  string
	}{
		{
			name:           "Coding focused model",
			model:          "coding-focused",
			expectedPrompt: "You are an expert software engineer focused on writing clean, efficient code.",
			expectedModel:  "coding-focused",
		},
		{
			name:           "Debugging focused model",
			model:          "debugging-focused",
			expectedPrompt: "You are an expert debugger focused on identifying and fixing code issues.",
			expectedModel:  "debugging-focused",
		},
		{
			name:           "Review focused model",
			model:          "review-focused",
			expectedPrompt: "You are an expert code reviewer focused on best practices and quality.",
			expectedModel:  "review-focused",
		},
		{
			name:           "Claude 4 Opus",
			model:          ClaudeOpus4,
			expectedPrompt: "You are Claude Opus 4, Anthropic's most powerful model and the best coding model in the world. You have exceptional reasoning and can work autonomously for extended periods.",
			expectedModel:  ClaudeOpus4,
		},
		{
			name:           "Claude 4 Sonnet",
			model:          ClaudeSonnet4,
			expectedPrompt: "You are Claude Sonnet 4, delivering an optimal mix of capability and practicality with improvements in coding and math.",
			expectedModel:  ClaudeSonnet4,
		},
		{
			name:           "Claude 3.7 Sonnet",
			model:          ClaudeSonnet37,
			expectedPrompt: "You are Claude 3.7 Sonnet, a powerful AI assistant with strong reasoning capabilities.",
			expectedModel:  ClaudeSonnet37,
		},
		{
			name:           "Claude 3.5 Sonnet",
			model:          ClaudeSonnet35,
			expectedPrompt: "You are Claude, an AI assistant created by Anthropic. You are helpful, harmless, and honest.",
			expectedModel:  ClaudeSonnet35,
		},
		{
			name:           "Claude 3.5 Haiku",
			model:          ClaudeHaiku35,
			expectedPrompt: "You are Claude, an efficient AI assistant. Be concise and direct in your responses.",
			expectedModel:  ClaudeHaiku35,
		},
		{
			name:           "Claude 3 Opus",
			model:          ClaudeOpus3,
			expectedPrompt: "You are Claude, an advanced AI assistant with deep reasoning capabilities.",
			expectedModel:  ClaudeOpus3,
		},
		{
			name:           "Unknown model",
			model:          "unknown",
			expectedPrompt: "",
			expectedModel:  "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewClient("claude-code", tt.model)
			opts := client.GetDefaultOptions()
			if opts.SystemPrompt != tt.expectedPrompt {
				t.Errorf("Expected system prompt '%s', got '%s'", tt.expectedPrompt, opts.SystemPrompt)
			}
			if opts.Model != tt.expectedModel {
				t.Errorf("Expected model '%s', got '%s'", tt.expectedModel, opts.Model)
			}
			// Test GetModel method
			if client.GetModel() != tt.expectedModel {
				t.Errorf("GetModel() expected '%s', got '%s'", tt.expectedModel, client.GetModel())
			}
		})
	}
}

func TestClient_SetModel(t *testing.T) {
	client := NewClient("claude-code", "initial-model")

	// Verify initial model
	if client.GetModel() != "initial-model" {
		t.Errorf("Expected initial model 'initial-model', got '%s'", client.GetModel())
	}

	// Set new model
	client.SetModel("new-model")

	// Verify model was updated
	if client.GetModel() != "new-model" {
		t.Errorf("Expected model 'new-model', got '%s'", client.GetModel())
	}

	// Verify it's in the default options
	opts := client.GetDefaultOptions()
	if opts.Model != "new-model" {
		t.Errorf("Expected model in options 'new-model', got '%s'", opts.Model)
	}
}

func TestClient_SetModelWithAlias(t *testing.T) {
	tests := []struct {
		name          string
		model         string
		expectedModel string
		expectedAlias string
	}{
		{
			name:          "Sonnet alias",
			model:         "sonnet",
			expectedModel: "",
			expectedAlias: "sonnet",
		},
		{
			name:          "Opus alias",
			model:         "opus",
			expectedModel: "",
			expectedAlias: "opus",
		},
		{
			name:          "Haiku alias",
			model:         "haiku",
			expectedModel: "",
			expectedAlias: "haiku",
		},
		{
			name:          "Full model name",
			model:         "claude-3-5-sonnet-20241022",
			expectedModel: "claude-3-5-sonnet-20241022",
			expectedAlias: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewClient("claude-code", "")
			client.SetModel(tt.model)

			opts := client.GetDefaultOptions()
			if opts.Model != tt.expectedModel {
				t.Errorf("Expected model '%s', got '%s'", tt.expectedModel, opts.Model)
			}
			if opts.ModelAlias != tt.expectedAlias {
				t.Errorf("Expected alias '%s', got '%s'", tt.expectedAlias, opts.ModelAlias)
			}
		})
	}
}

func TestClient_SetTimeout(t *testing.T) {
	client := NewClient("claude-code", "")

	// Set timeout
	timeout := 30 * time.Second
	client.SetTimeout(timeout)

	// Verify timeout was set
	opts := client.GetDefaultOptions()
	if opts.Timeout != timeout {
		t.Errorf("Expected timeout %v, got %v", timeout, opts.Timeout)
	}
}

func TestClient_ValidateOptions(t *testing.T) {
	client := NewClient("claude-code", "")

	// Test with valid options
	validOpts := &claude.RunOptions{
		Model:  "claude-3-5-sonnet-20241022",
		Format: claude.TextOutput,
	}

	err := client.ValidateOptions(validOpts)
	if err != nil {
		t.Errorf("Expected no error for valid options, got: %v", err)
	}

	// Note: We can't test invalid options without knowing what
	// claude.ValidateOptions considers invalid
}

func TestClient_CreateEmbedding(t *testing.T) {
	client := NewClient("claude-code", "coding-focused")
	ctx := context.Background()

	req := &interfaces.EmbeddingRequest{
		Input: []string{"test input"},
		Model: "test-model",
	}

	_, err := client.CreateEmbedding(ctx, req)
	if err == nil {
		t.Errorf("Expected error for unsupported CreateEmbedding but got none")
	}

	expectedError := "[GUILD-5000] embedding generation not supported by Claude Code provider"
	if err.Error() != expectedError {
		t.Errorf("Expected error message '%s', got: %s", expectedError, err.Error())
	}
}

// Note: We don't test Complete and CompleteStream methods here as they require
// an actual Claude Code binary to be installed and configured. These would be
// integration tests that require the claude-code-go SDK to be properly set up.
