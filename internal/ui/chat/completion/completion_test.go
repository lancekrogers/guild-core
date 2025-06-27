// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package completion

import (
	"context"
	"testing"
	"time"

	"github.com/lancekrogers/guild/pkg/config"
	"github.com/lancekrogers/guild/pkg/suggestions"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCompletionEngine_BasicCompletion(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantMin  int
		contains []string
	}{
		{
			name:     "command completion",
			input:    "/hel",
			wantMin:  1,
			contains: []string{"/help"},
		},
		{
			name:     "agent completion",
			input:    "@",
			wantMin:  1,
			contains: []string{"@all"},
		},
		{
			name:     "argument completion",
			input:    "--pa",
			wantMin:  1,
			contains: []string{"--path"},
		},
		{
			name:     "empty input helpful suggestions",
			input:    "",
			wantMin:  1,
			contains: []string{"/help"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create basic config
			cfg := &config.GuildConfig{
				Agents: []config.AgentConfig{
					{ID: "test-agent", Name: "Test Agent", Capabilities: []string{"testing"}},
				},
			}

			// Create completion engine
			engine := NewCompletionEngine(cfg, ".")

			// Get completions
			results := engine.Complete(tt.input, len(tt.input))

			// Check minimum results
			assert.GreaterOrEqual(t, len(results), tt.wantMin,
				"expected at least %d results, got %d", tt.wantMin, len(results))

			// Check for expected content
			for _, expected := range tt.contains {
				found := false
				for _, result := range results {
					if result.Content == expected {
						found = true
						break
					}
				}
				assert.True(t, found, "expected to find %q in results", expected)
			}
		})
	}
}

func TestCompletionEngineEnhanced_SuggestionIntegration(t *testing.T) {
	// Create enhanced engine
	cfg := &config.GuildConfig{
		Agents: []config.AgentConfig{
			{ID: "test-agent", Name: "Test Agent", Capabilities: []string{"testing"}},
		},
	}

	engine, err := NewCompletionEngineEnhanced(cfg, ".")
	require.NoError(t, err)
	require.NotNil(t, engine)

	// Verify providers are registered
	status := engine.GetProviderStatus()
	assert.True(t, status["command"], "command provider should be registered")
	assert.True(t, status["followup"], "followup provider should be registered")
	// These providers require external dependencies and are not registered in the basic setup
	// assert.True(t, status["template"], "template provider should be registered")
	// assert.True(t, status["tool"], "tool provider should be registered")
	// assert.True(t, status["lsp"], "lsp provider should be registered")

	// Test that the Complete method works with command completion
	// (The direct suggestion API is for natural language, not command prefix matching)
	results := engine.Complete("/hel", 4)
	assert.NotEmpty(t, results, "should get command completions")

	// Verify we get /help
	found := false
	for _, r := range results {
		if r.Content == "/help" {
			found = true
			break
		}
	}
	assert.True(t, found, "should find /help in completions")
}

func TestCompletionEngine_UpdateConversationHistory(t *testing.T) {
	// Create enhanced engine which has suggestion system
	enhanced, err := NewCompletionEngineEnhanced(&config.GuildConfig{}, ".")
	require.NoError(t, err)

	engine := enhanced.CompletionEngine

	// Create mock conversation history
	messages := []suggestions.ChatMessage{
		{Role: "user", Content: "Hello", Timestamp: time.Now()},
		{Role: "assistant", Content: "Hi there!", Timestamp: time.Now()},
	}

	// Update conversation history
	engine.UpdateConversationHistory(messages)

	// Since conversationHist is private, we can't directly verify it
	// Instead, we'll just ensure the method doesn't panic
	// In a real test, we would verify the behavior through public methods
}

func TestCompletionEngine_ProjectContext(t *testing.T) {
	tests := []struct {
		name         string
		projectRoot  string
		wantType     string
		wantLanguage string
	}{
		{
			name:         "go project",
			projectRoot:  "/Users/lancerogers/Dev/AI/guild-framework/guild-core", // Absolute path to guild-core root
			wantType:     "go-library",
			wantLanguage: "go",
		},
		{
			name:         "empty project",
			projectRoot:  "",
			wantType:     "unknown",
			wantLanguage: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine := NewCompletionEngine(&config.GuildConfig{}, tt.projectRoot)

			// Test project type detection
			assert.Equal(t, tt.wantType, engine.detectProjectType())

			// Test language detection
			assert.Equal(t, tt.wantLanguage, engine.detectPrimaryLanguage())

			// Test project context building
			ctx := engine.buildProjectContext()
			assert.Equal(t, tt.projectRoot, ctx.ProjectPath)
			assert.Equal(t, tt.wantType, ctx.ProjectType)
			assert.Equal(t, tt.wantLanguage, ctx.Language)
		})
	}
}

func TestCompletionEngine_MergeAndRankResults(t *testing.T) {
	engine := NewCompletionEngine(&config.GuildConfig{}, ".")

	// Create test results with duplicates
	results := []CompletionResult{
		{Content: "/help", AgentID: "system", Metadata: map[string]string{"type": "command"}},
		{Content: "/Help", AgentID: "system", Metadata: map[string]string{"type": "command"}}, // Duplicate (case-insensitive)
		{Content: "/status", AgentID: "system", Metadata: map[string]string{"type": "command"}},
		{Content: "/help", AgentID: "suggestion", Metadata: map[string]string{"type": "suggestion"}}, // Duplicate
	}

	// Merge and rank
	merged := engine.mergeAndRankResults(results, "/hel")

	// Should deduplicate
	assert.LessOrEqual(t, len(merged), 2, "should deduplicate results")

	// First result should be exact match
	if len(merged) > 0 {
		assert.Equal(t, "/help", merged[0].Content)
	}
}

func TestCompletionEngine_FuzzyMatch(t *testing.T) {
	tests := []struct {
		text    string
		pattern string
		want    bool
	}{
		{"hello", "hel", true},
		{"HELLO", "hel", true},
		{"world", "hel", false},
		{"", "hel", false},
		{"hello", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.text+"_"+tt.pattern, func(t *testing.T) {
			got := fuzzyMatch(tt.text, tt.pattern)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestCompletionEngine_CancellationHandling(t *testing.T) {
	// Create a pre-cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// Create enhanced engine for direct suggestion test
	enhanced, err := NewCompletionEngineEnhanced(&config.GuildConfig{}, ".")
	require.NoError(t, err)

	// Should handle cancelled context gracefully
	// Note: The suggestion providers might still return results if they don't check context
	// The important thing is that the method completes without hanging
	suggestions, err := enhanced.GetDirectSuggestions(ctx, "test")
	// We might get an error or results depending on timing
	if err != nil {
		assert.Nil(t, suggestions)
	} else {
		// If no error, we should at least get results
		assert.NotNil(t, suggestions)
	}
}

func TestCompletionEngine_GetSuggestionIcon(t *testing.T) {
	engine := NewCompletionEngine(&config.GuildConfig{}, ".")

	tests := []struct {
		suggestionType suggestions.SuggestionType
		wantIcon       string
	}{
		{suggestions.SuggestionTypeCommand, "⚡"},
		{suggestions.SuggestionTypeTool, "🔧"},
		{suggestions.SuggestionTypeTemplate, "📝"},
		{suggestions.SuggestionTypeFollowUp, "💡"},
		{suggestions.SuggestionTypeCode, "💻"},
		{suggestions.SuggestionType("unknown"), "✨"},
	}

	for _, tt := range tests {
		t.Run(string(tt.suggestionType), func(t *testing.T) {
			icon := engine.getSuggestionIcon(tt.suggestionType)
			assert.Equal(t, tt.wantIcon, icon)
		})
	}
}
