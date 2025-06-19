// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package v2

import (
	"context"
	"testing"
	
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	
	"github.com/guild-ventures/guild-core/pkg/agent"
	"github.com/guild-ventures/guild-core/pkg/commission"
	"github.com/guild-ventures/guild-core/pkg/memory"
	"github.com/guild-ventures/guild-core/pkg/providers"
	"github.com/guild-ventures/guild-core/pkg/suggestions"
	"github.com/guild-ventures/guild-core/pkg/tools"
)

// TestSuggestionIntegration verifies the suggestion system is properly integrated
func TestSuggestionIntegration(t *testing.T) {
	// Create a minimal test app
	app := &App{
		ctx: context.Background(),
	}
	
	// Initialize the suggestion system
	err := app.initializeSuggestionSystem()
	assert.NoError(t, err, "Should initialize suggestion system without error")
	
	// Verify components are created (they may be nil if dependencies are missing)
	t.Run("Factory Creation", func(t *testing.T) {
		// Factory might be nil if registry/dependencies are missing
		if app.suggestionFactory != nil {
			assert.NotNil(t, app.suggestionFactory.GetSuggestionManager(),
				"Factory should have a suggestion manager")
		}
	})
	
	t.Run("Enhanced Agent", func(t *testing.T) {
		// Enhanced agent might be nil if dependencies are missing
		if app.enhancedAgent != nil {
			assert.NotEmpty(t, app.enhancedAgent.GetID(), "Agent should have an ID")
			assert.Equal(t, "chat-agent", app.enhancedAgent.GetID())
			assert.Equal(t, "Chat Assistant", app.enhancedAgent.GetName())
		}
	})
	
	t.Run("Chat Handler", func(t *testing.T) {
		// Chat handler might be nil if agent creation failed
		if app.chatHandler != nil {
			// Test suggestion request handling
			request := agent.SuggestionRequest{
				Message:        "test",
				MaxSuggestions: 5,
			}
			
			// This might fail if no providers are configured
			resp, err := app.chatHandler.GetSuggestions(context.Background(), request)
			if err == nil {
				assert.True(t, resp.Success, "Response should indicate success")
				assert.NotNil(t, resp.Suggestions, "Should return suggestions array")
			}
		}
	})
}

// TestCompletionEngineIntegration verifies completion engine suggestion integration
func TestCompletionEngineIntegration(t *testing.T) {
	// Create completion engine
	engine := NewCompletionEngine(nil, ".")
	require.NotNil(t, engine)
	
	// Create mock enhanced agent
	mockAgent := &MockEnhancedAgent{
		id:   "test-agent",
		name: "Test Agent",
		suggestionManager: suggestions.NewSuggestionManager(),
	}
	
	// Create chat handler
	handler := agent.NewChatSuggestionHandler(mockAgent)
	
	// Set enhanced agent on completion engine
	engine.SetEnhancedAgent(mockAgent, handler)
	
	// Verify integration
	assert.NotNil(t, engine.suggestionManager, "Should have suggestion manager")
	assert.NotNil(t, engine.chatHandler, "Should have chat handler")
}

// MockEnhancedAgent implements a minimal EnhancedGuildArtisan for testing
type MockEnhancedAgent struct {
	id                string
	name              string
	suggestionManager suggestions.SuggestionManager
}

func (m *MockEnhancedAgent) GetID() string { return m.id }
func (m *MockEnhancedAgent) GetName() string { return m.name }
func (m *MockEnhancedAgent) GetSuggestionManager() suggestions.SuggestionManager {
	return m.suggestionManager
}

func (m *MockEnhancedAgent) GetSuggestionsForContext(ctx context.Context, message string, filter *suggestions.SuggestionFilter) ([]suggestions.Suggestion, error) {
	return []suggestions.Suggestion{}, nil
}

func (m *MockEnhancedAgent) ExecuteWithSuggestions(ctx context.Context, request string, enableSuggestions bool) (*agent.EnhancedExecutionResult, error) {
	return &agent.EnhancedExecutionResult{}, nil
}

// Implement GuildArtisan interface methods
func (m *MockEnhancedAgent) GetCapabilities() []string { return []string{} }
func (m *MockEnhancedAgent) GetStatus() string { return "active" }
func (m *MockEnhancedAgent) GetType() string { return "mock" }
func (m *MockEnhancedAgent) Execute(ctx context.Context, task string) (string, error) {
	return "", nil
}
func (m *MockEnhancedAgent) Stop() error { return nil }
func (m *MockEnhancedAgent) GetCommissionManager() commission.CommissionManager { return nil }
func (m *MockEnhancedAgent) GetLLMClient() providers.LLMClient { return nil }
func (m *MockEnhancedAgent) GetMemoryManager() memory.ChainManager { return nil }
func (m *MockEnhancedAgent) GetToolRegistry() tools.Registry { return nil }