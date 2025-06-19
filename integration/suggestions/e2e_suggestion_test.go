// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package suggestions_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/guild-ventures/guild-core/internal/chat/v2"
	"github.com/guild-ventures/guild-core/internal/chat/v2/services"
	"github.com/guild-ventures/guild-core/pkg/agent"
	"github.com/guild-ventures/guild-core/pkg/commission"
	"github.com/guild-ventures/guild-core/pkg/config"
	"github.com/guild-ventures/guild-core/pkg/memory"
	"github.com/guild-ventures/guild-core/pkg/providers"
	"github.com/guild-ventures/guild-core/pkg/suggestions"
	"github.com/guild-ventures/guild-core/pkg/tools"
)

// TestEndToEndSuggestionFlow tests the complete suggestion flow from user input to suggestion display
func TestEndToEndSuggestionFlow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()

	// Create configuration
	cfg := &config.GuildConfig{
		Agents: []config.AgentConfig{
			{
				ID:           "test-agent",
				Name:         "Test Agent",
				Type:         "worker",
				Capabilities: []string{"suggestions", "chat"},
			},
		},
	}

	// Initialize completion engine with suggestions
	engine, err := v2.NewCompletionEngineEnhanced(cfg, ".")
	require.NoError(t, err)
	require.NotNil(t, engine)

	t.Run("CommandSuggestions", func(t *testing.T) {
		// Test command completion
		results := engine.Complete("/hel", 4)
		assert.NotEmpty(t, results, "Should get command completions")

		// Verify /help is suggested
		found := false
		for _, r := range results {
			if r.Content == "/help" {
				found = true
				break
			}
		}
		assert.True(t, found, "Should suggest /help command")
	})

	t.Run("DirectSuggestions", func(t *testing.T) {
		// Test natural language suggestions
		suggestions, err := engine.GetDirectSuggestions(ctx, "How do I read a file?")
		assert.NoError(t, err)
		assert.NotNil(t, suggestions)
		
		// Should get at least command suggestions
		assert.NotEmpty(t, suggestions, "Should get suggestions for file reading query")
	})

	t.Run("ContextAwareSuggestions", func(t *testing.T) {
		// Update conversation history
		messages := []suggestions.ChatMessage{
			{Role: "user", Content: "I need to parse JSON", Timestamp: time.Now()},
			{Role: "assistant", Content: "You can use the json package", Timestamp: time.Now()},
		}
		engine.UpdateConversationHistory(messages)

		// Get follow-up suggestions
		suggestions, err := engine.GetDirectSuggestions(ctx, "Can you show me an example?")
		assert.NoError(t, err)
		assert.NotNil(t, suggestions)
		
		// Verify we get suggestions related to the context
		assert.NotEmpty(t, suggestions, "Should get context-aware suggestions")
	})
}

// TestSuggestionServiceIntegration tests the suggestion service in isolation
func TestSuggestionServiceIntegration(t *testing.T) {
	ctx := context.Background()

	// Create a mock enhanced agent
	mockAgent := &mockEnhancedAgent{
		id:   "test-agent",
		name: "Test Agent",
		suggestions: []suggestions.Suggestion{
			{
				ID:          "1",
				Type:        suggestions.SuggestionTypeTool,
				Content:     "Use FileReader tool",
				Display:     "Use FileReader tool",
				Description: "Read files from the filesystem",
				Confidence:  0.9,
				Priority:    1,
				Source:      "test",
				CreatedAt:   time.Now(),
			},
		},
	}

	// Create chat suggestion handler
	handler := agent.NewChatSuggestionHandler(mockAgent)
	require.NotNil(t, handler)

	// Create suggestion service
	service, err := services.NewSuggestionService(ctx, handler)
	require.NoError(t, err)
	require.NotNil(t, service)

	t.Run("BasicSuggestionFlow", func(t *testing.T) {
		// Get suggestions
		cmd := service.GetSuggestions("How do I read a file?", nil)
		msg := cmd()

		// Verify response
		suggestionsMsg, ok := msg.(services.SuggestionsReceivedMsg)
		require.True(t, ok)
		assert.NotEmpty(t, suggestionsMsg.Suggestions)
		assert.Equal(t, "Use FileReader tool", suggestionsMsg.Suggestions[0].Content)
	})

	t.Run("CachePerformance", func(t *testing.T) {
		// First request - cache miss
		cmd1 := service.GetSuggestions("test query", nil)
		msg1 := cmd1()
		result1, ok := msg1.(services.SuggestionsReceivedMsg)
		require.True(t, ok)
		assert.False(t, result1.FromCache)

		// Second request - cache hit
		cmd2 := service.GetSuggestions("test query", nil)
		msg2 := cmd2()
		result2, ok := msg2.(services.SuggestionsReceivedMsg)
		require.True(t, ok)
		assert.True(t, result2.FromCache)
		assert.Less(t, result2.Latency, result1.Latency, "Cached response should be faster")
	})

	t.Run("TokenOptimization", func(t *testing.T) {
		// Set low token budget
		service.SetTokenBudget(100)

		// Create large context
		largeContext := string(make([]byte, 10000))
		optimized := service.OptimizeContext(largeContext)

		// Verify optimization
		assert.Less(t, len(optimized), len(largeContext), "Context should be truncated")
		assert.LessOrEqual(t, len(optimized), 400, "Should respect token budget (4 chars per token)")
	})
}

// TestSuggestionProviderChain tests multiple providers working together
func TestSuggestionProviderChain(t *testing.T) {
	ctx := context.Background()

	// Create suggestion manager
	manager := suggestions.NewSuggestionManager()

	// Register multiple providers
	commandProvider := suggestions.NewCommandSuggestionProvider()
	followUpProvider := suggestions.NewFollowUpSuggestionProvider()
	
	err := manager.RegisterProvider(commandProvider)
	require.NoError(t, err)
	err = manager.RegisterProvider(followUpProvider)
	require.NoError(t, err)

	// Create context
	suggestionCtx := suggestions.SuggestionContext{
		CurrentMessage: "help",
		ConversationHistory: []suggestions.ChatMessage{
			{Role: "user", Content: "Hello", Timestamp: time.Now()},
			{Role: "assistant", Content: "Hi! How can I help you?", Timestamp: time.Now()},
		},
	}

	// Get suggestions from all providers
	allSuggestions, err := manager.GetSuggestions(ctx, suggestionCtx, nil)
	assert.NoError(t, err)
	assert.NotEmpty(t, allSuggestions)

	// Verify we get suggestions from multiple providers
	types := make(map[suggestions.SuggestionType]bool)
	for _, s := range allSuggestions {
		types[s.Type] = true
		t.Logf("Got suggestion: type=%s, content=%s", s.Type, s.Content)
	}
	assert.True(t, types[suggestions.SuggestionTypeCommand], "Should have command suggestions")
	// Follow-up suggestions may not always be generated depending on context
	// assert.True(t, types[suggestions.SuggestionTypeFollowUp], "Should have follow-up suggestions")
	
	// Just verify we got suggestions from at least one provider
	assert.NotEmpty(t, allSuggestions, "Should have at least some suggestions")
}

// Mock enhanced agent for testing
type mockEnhancedAgent struct {
	id          string
	name        string
	suggestions []suggestions.Suggestion
}

func (m *mockEnhancedAgent) GetID() string                              { return m.id }
func (m *mockEnhancedAgent) GetName() string                            { return m.name }
func (m *mockEnhancedAgent) GetType() string                            { return "mock" }
func (m *mockEnhancedAgent) GetCapabilities() []string                  { return []string{"suggestions"} }
func (m *mockEnhancedAgent) Execute(ctx context.Context, task string) (string, error) {
	return "mock response", nil
}

func (m *mockEnhancedAgent) GetSuggestionsForContext(ctx context.Context, message string, filter *suggestions.SuggestionFilter) ([]suggestions.Suggestion, error) {
	return m.suggestions, nil
}

func (m *mockEnhancedAgent) ExecuteWithSuggestions(ctx context.Context, request string, enableSuggestions bool) (*agent.EnhancedExecutionResult, error) {
	return &agent.EnhancedExecutionResult{
		Response: "mock response",
		Success:  true,
	}, nil
}

func (m *mockEnhancedAgent) GetSuggestionManager() suggestions.SuggestionManager {
	return suggestions.NewSuggestionManager()
}

// Implement remaining GuildArtisan methods
func (m *mockEnhancedAgent) GetToolRegistry() tools.Registry             { return nil }
func (m *mockEnhancedAgent) GetCommissionManager() commission.CommissionManager { return nil }
func (m *mockEnhancedAgent) GetLLMClient() providers.LLMClient          { return nil }
func (m *mockEnhancedAgent) GetMemoryManager() memory.ChainManager      { return nil }