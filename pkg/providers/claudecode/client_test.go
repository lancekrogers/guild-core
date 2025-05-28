package claudecode

import (
	"context"
	"testing"

	"github.com/guild-ventures/guild-core/pkg/providers/interfaces"
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
		name        string
		binPath     string
		expected    string
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
	}{
		{
			name:           "Coding focused model",
			model:          "coding-focused",
			expectedPrompt: "You are an expert software engineer focused on writing clean, efficient code.",
		},
		{
			name:           "Debugging focused model",
			model:          "debugging-focused",
			expectedPrompt: "You are an expert debugger focused on identifying and fixing code issues.",
		},
		{
			name:           "Review focused model",
			model:          "review-focused",
			expectedPrompt: "You are an expert code reviewer focused on best practices and quality.",
		},
		{
			name:           "Unknown model",
			model:          "unknown",
			expectedPrompt: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewClient("claude-code", tt.model)
			opts := client.GetDefaultOptions()
			if opts.SystemPrompt != tt.expectedPrompt {
				t.Errorf("Expected system prompt '%s', got '%s'", tt.expectedPrompt, opts.SystemPrompt)
			}
		})
	}
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

	if err.Error() != "embedding generation not supported by Claude Code provider" {
		t.Errorf("Expected specific error message, got: %s", err.Error())
	}
}

// Note: We don't test Complete and CompleteStream methods here as they require
// an actual Claude Code binary to be installed and configured. These would be
// integration tests that require the claude-code-go SDK to be properly set up.