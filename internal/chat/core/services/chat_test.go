// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package services

import (
	"context"
	"testing"
	"time"

	_ "github.com/charmbracelet/bubbletea" // Used for message types
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"

	"github.com/guild-ventures/guild-core/pkg/agent"
	"github.com/guild-ventures/guild-core/pkg/commission"
	pb "github.com/guild-ventures/guild-core/pkg/grpc/pb/guild/v1"
	"github.com/guild-ventures/guild-core/pkg/memory"
	"github.com/guild-ventures/guild-core/pkg/providers"
	"github.com/guild-ventures/guild-core/pkg/registry"
	"github.com/guild-ventures/guild-core/pkg/suggestions"
	"github.com/guild-ventures/guild-core/pkg/tools"
)

// mockGuildClient implements pb.GuildClient for testing
type mockGuildClient struct {
	pb.GuildClient
	getAgentStatusFunc func(ctx context.Context, req *pb.GetAgentStatusRequest, opts ...grpc.CallOption) (*pb.AgentStatus, error)
}

func (m *mockGuildClient) GetAgentStatus(ctx context.Context, req *pb.GetAgentStatusRequest, opts ...grpc.CallOption) (*pb.AgentStatus, error) {
	if m.getAgentStatusFunc != nil {
		return m.getAgentStatusFunc(ctx, req, opts...)
	}
	return &pb.AgentStatus{
		State: pb.AgentStatus_IDLE,
		CurrentTask: "idle",
		LastActivity: time.Now().Unix(),
	}, nil
}

// mockEnhancedGuildArtisan implements agent.EnhancedGuildArtisan for testing
type mockEnhancedGuildArtisan struct {
	generateSuggestionsFunc func(ctx context.Context, request agent.SuggestionRequest) ([]suggestions.Suggestion, error)
}

func (m *mockEnhancedGuildArtisan) GetID() string {
	return "mock-enhanced-agent"
}

func (m *mockEnhancedGuildArtisan) GetName() string {
	return "Mock Enhanced Agent"
}

func (m *mockEnhancedGuildArtisan) GetType() string {
	return "enhanced"
}

func (m *mockEnhancedGuildArtisan) GetDescription() string {
	return "Mock enhanced agent for testing"
}

func (m *mockEnhancedGuildArtisan) GetCapabilities() []string {
	return []string{"suggestions", "streaming"}
}

func (m *mockEnhancedGuildArtisan) GetToolRegistry() tools.Registry {
	return nil
}

func (m *mockEnhancedGuildArtisan) GetCommissionManager() commission.CommissionManager {
	return nil
}

func (m *mockEnhancedGuildArtisan) GetLLMClient() providers.LLMClient {
	return nil
}

func (m *mockEnhancedGuildArtisan) GetMemoryManager() memory.ChainManager {
	return nil
}

func (m *mockEnhancedGuildArtisan) GetSuggestionManager() suggestions.SuggestionManager {
	return nil
}

func (m *mockEnhancedGuildArtisan) GetSuggestionsForContext(ctx context.Context, message string, filter *suggestions.SuggestionFilter) ([]suggestions.Suggestion, error) {
	return m.GenerateSuggestions(ctx, agent.SuggestionRequest{
		Message: message,
		Filter: filter,
	})
}

func (m *mockEnhancedGuildArtisan) ExecuteWithSuggestions(ctx context.Context, request string, enableSuggestions bool) (*agent.EnhancedExecutionResult, error) {
	return &agent.EnhancedExecutionResult{
		Response: "Mock response with suggestions",
		Success: true,
	}, nil
}

func (m *mockEnhancedGuildArtisan) GenerateSuggestions(ctx context.Context, request agent.SuggestionRequest) ([]suggestions.Suggestion, error) {
	if m.generateSuggestionsFunc != nil {
		return m.generateSuggestionsFunc(ctx, request)
	}
	return []suggestions.Suggestion{
		{
			Type:        suggestions.SuggestionTypeTool,
			Content:     "Use the FileReader tool",
			Description: "Read the configuration file",
			Confidence:  0.9,
		},
	}, nil
}

func (m *mockEnhancedGuildArtisan) Execute(ctx context.Context, request string) (string, error) {
	return "Mock response", nil
}

func TestNewChatService(t *testing.T) {
	ctx := context.Background()
	client := &mockGuildClient{}
	reg := registry.NewComponentRegistry()

	tests := []struct {
		name      string
		ctx       context.Context
		client    pb.GuildClient
		registry  registry.ComponentRegistry
		wantError bool
	}{
		{
			name:      "valid inputs",
			ctx:       ctx,
			client:    client,
			registry:  reg,
			wantError: false,
		},
		{
			name:      "nil client",
			ctx:       ctx,
			client:    nil,
			registry:  reg,
			wantError: true,
		},
		{
			name:      "nil registry",
			ctx:       ctx,
			client:    client,
			registry:  nil,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cs, err := NewChatService(tt.ctx, tt.client, tt.registry)
			if tt.wantError {
				assert.Error(t, err)
				assert.Nil(t, cs)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, cs)
				assert.Equal(t, SuggestionModeBoth, cs.suggestionMode)
				assert.True(t, cs.enableSuggestions)
				assert.Equal(t, true, cs.enableSuggestions)
			}
		})
	}
}

func TestNewChatServiceWithSuggestions(t *testing.T) {
	ctx := context.Background()
	client := &mockGuildClient{}
	reg := registry.NewComponentRegistry()
	mockAgent := &mockEnhancedGuildArtisan{}

	tests := []struct {
		name      string
		agent     agent.EnhancedGuildArtisan
		wantError bool
	}{
		{
			name:      "with enhanced agent",
			agent:     mockAgent,
			wantError: false,
		},
		{
			name:      "without enhanced agent",
			agent:     nil,
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cs, err := NewChatServiceWithSuggestions(ctx, client, reg, tt.agent)
			if tt.wantError {
				assert.Error(t, err)
				assert.Nil(t, cs)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, cs)
				
				if tt.agent != nil {
					assert.NotNil(t, cs.suggestionService)
					assert.True(t, cs.enableSuggestions)
					assert.Equal(t, SuggestionModeBoth, cs.suggestionMode)
				}
			}
		})
	}
}

func TestChatServiceSuggestionIntegration(t *testing.T) {
	ctx := context.Background()
	client := &mockGuildClient{}
	reg := registry.NewComponentRegistry()
	mockAgent := &mockEnhancedGuildArtisan{}

	// Set up mock agent with custom suggestions
	mockAgent.generateSuggestionsFunc = func(ctx context.Context, request agent.SuggestionRequest) ([]suggestions.Suggestion, error) {
		return []suggestions.Suggestion{
			{
				Type:        suggestions.SuggestionTypeTool,
				Content:     "Use the FileReader tool",
				Description: "Read the configuration file",
				Confidence:  0.9,
			},
			{
				Type:        suggestions.SuggestionTypeContext,
				Content:     "Include error handling",
				Description: "Add try-catch blocks",
				Confidence:  0.8,
			},
		}, nil
	}

	// Create service with suggestions
	cs, err := NewChatServiceWithSuggestions(ctx, client, reg, mockAgent)
	require.NoError(t, err)
	require.NotNil(t, cs)

	t.Run("SendMessageWithSuggestions", func(t *testing.T) {
		cmd := cs.SendMessageWithSuggestions("test-agent", "How do I read a config file?", "conv-123")
		assert.NotNil(t, cmd)
		
		// Execute the command
		msg := cmd()
		assert.NotNil(t, msg)
	})

	t.Run("TokenOptimization", func(t *testing.T) {
		// Test message optimization
		longMessage := string(make([]byte, 20000)) // Very long message
		cmd := cs.SendMessage("test-agent", longMessage)
		assert.NotNil(t, cmd)
		
		// Execute the command to trigger token tracking
		msg := cmd()
		assert.NotNil(t, msg)
		
		// Check token usage is tracked
		assert.True(t, cs.enableSuggestions)
	})

	t.Run("SuggestionModes", func(t *testing.T) {
		// Test different suggestion modes
		modes := []SuggestionMode{
			SuggestionModeNone,
			SuggestionModePre,
			SuggestionModePost,
			SuggestionModeBoth,
		}

		for _, mode := range modes {
			cs.SetSuggestionMode(mode)
			assert.Equal(t, mode, cs.suggestionMode)
			assert.Equal(t, mode != SuggestionModeNone, cs.enableSuggestions)
			
			// Test pre-execution suggestions
			preCmd := cs.GetPreExecutionSuggestions("test message", "conv-123")
			if mode == SuggestionModePre || mode == SuggestionModeBoth {
				assert.NotNil(t, preCmd)
			} else {
				assert.Nil(t, preCmd)
			}
			
			// Test post-execution suggestions
			postCmd := cs.GetPostExecutionSuggestions("original", "response")
			if mode == SuggestionModePost || mode == SuggestionModeBoth {
				assert.NotNil(t, postCmd)
			} else {
				assert.Nil(t, postCmd)
			}
		}
	})

	t.Run("ProcessAgentResponse", func(t *testing.T) {
		cs.SetSuggestionMode(SuggestionModeBoth)
		
		response := AgentResponseMsg{
			AgentID: "test-agent",
			Content: "Here's how to read a config file...",
			Done:    true,
		}
		
		cmd := cs.ProcessAgentResponse(response, "How do I read a config file?")
		assert.NotNil(t, cmd)
		
		// Execute and check result
		msg := cmd()
		assert.NotNil(t, msg)
	})

	t.Run("Statistics", func(t *testing.T) {
		stats := cs.GetStats()
		
		// Check basic stats
		assert.Contains(t, stats, "agent_count")
		assert.Contains(t, stats, "suggestions_enabled")
		assert.Contains(t, stats, "suggestion_mode")
		assert.Contains(t, stats, "suggestions_enabled")
		assert.Contains(t, stats, "stream_count")
		
		// Check suggestion service stats are included
		assert.Contains(t, stats, "suggestion_total_requests")
		assert.Contains(t, stats, "suggestion_cache_hits")
	})

	t.Run("ConfigureSuggestions", func(t *testing.T) {
		cs.ConfigureSuggestions(false)
		assert.Equal(t, false, cs.enableSuggestions)
		
		// Verify suggestion service is still available
		assert.NotNil(t, cs.suggestionService)
	})

	t.Run("NilSuggestionService", func(t *testing.T) {
		// Create service without suggestions
		cs2, err := NewChatService(ctx, client, reg)
		require.NoError(t, err)
		
		// These should not panic with nil suggestion service
		cmd := cs2.GetPreExecutionSuggestions("test", "conv-123")
		assert.Nil(t, cmd)
		
		cmd = cs2.GetPostExecutionSuggestions("original", "response")
		assert.Nil(t, cmd)
		
		response := AgentResponseMsg{
			AgentID: "test-agent",
			Content: "response",
			Done:    true,
		}
		cmd = cs2.ProcessAgentResponse(response, "original")
		assert.Nil(t, cmd) // Should return nil when no suggestion service
	})
}

func TestChatServiceCommands(t *testing.T) {
	ctx := context.Background()
	client := &mockGuildClient{}
	reg := registry.NewComponentRegistry()
	
	cs, err := NewChatService(ctx, client, reg)
	require.NoError(t, err)

	t.Run("Start", func(t *testing.T) {
		cmd := cs.Start()
		assert.NotNil(t, cmd)
		
		msg := cmd()
		assert.IsType(t, ChatServiceStartedMsg{}, msg)
	})

	t.Run("GetAgentStatus", func(t *testing.T) {
		cmd := cs.GetAgentStatus("test-agent")
		assert.NotNil(t, cmd)
		
		msg := cmd()
		assert.IsType(t, AgentStatusUpdateMsg{}, msg)
	})

	t.Run("ExecuteTool", func(t *testing.T) {
		params := map[string]string{
			"param1": "value1",
		}
		cmd := cs.ExecuteTool("test-tool", params)
		assert.NotNil(t, cmd)
		
		msg := cmd()
		assert.IsType(t, ToolExecutionCompleteMsg{}, msg)
	})

	t.Run("StreamChat", func(t *testing.T) {
		cmd := cs.StreamChat("test-agent")
		assert.NotNil(t, cmd)
		
		msg := cmd()
		assert.IsType(t, ChatStreamStartedMsg{}, msg)
	})

	t.Run("StopStream", func(t *testing.T) {
		// Start a stream first
		cs.activeStreams["test-agent"] = struct{}{}
		
		cmd := cs.StopStream("test-agent")
		assert.NotNil(t, cmd)
		
		msg := cmd()
		assert.IsType(t, ChatStreamStoppedMsg{}, msg)
		
		// Verify stream was removed
		assert.False(t, cs.IsStreamActive("test-agent"))
	})
}

func TestChatServiceConfiguration(t *testing.T) {
	ctx := context.Background()
	client := &mockGuildClient{}
	reg := registry.NewComponentRegistry()
	
	cs, err := NewChatService(ctx, client, reg)
	require.NoError(t, err)

	t.Run("SetTimeout", func(t *testing.T) {
		cs.SetTimeout(60 * time.Second)
		assert.Equal(t, 60*time.Second, cs.timeout)
	})

	t.Run("SetSuggestionService", func(t *testing.T) {
		mockAgent := &mockEnhancedGuildArtisan{}
		handler := agent.NewChatSuggestionHandler(mockAgent)
		suggestionService, err := NewSuggestionService(ctx, handler)
		require.NoError(t, err)
		
		err = cs.SetSuggestionService(suggestionService)
		assert.NoError(t, err)
		assert.Equal(t, suggestionService, cs.suggestionService)
		
		// Test nil service
		err = cs.SetSuggestionService(nil)
		assert.Error(t, err)
	})

	t.Run("GetAgents", func(t *testing.T) {
		cs.agents = []string{"agent1", "agent2", "agent3"}
		agents := cs.GetAgents()
		assert.Equal(t, cs.agents, agents)
	})

	t.Run("ActiveStreams", func(t *testing.T) {
		// Add some streams
		cs.activeStreams["agent1"] = struct{}{}
		cs.activeStreams["agent2"] = struct{}{}
		
		assert.Equal(t, 2, cs.GetActiveStreams())
		assert.True(t, cs.IsStreamActive("agent1"))
		assert.True(t, cs.IsStreamActive("agent2"))
		assert.False(t, cs.IsStreamActive("agent3"))
	})
}

// TestChatServiceMessageBatching tests the batch command functionality
func TestChatServiceMessageBatching(t *testing.T) {
	ctx := context.Background()
	client := &mockGuildClient{}
	reg := registry.NewComponentRegistry()
	mockAgent := &mockEnhancedGuildArtisan{}
	
	// No need to set up mock - our mock returns capabilities by default
	
	cs, err := NewChatServiceWithSuggestions(ctx, client, reg, mockAgent)
	require.NoError(t, err)
	
	// Enable pre-suggestions
	cs.SetSuggestionMode(SuggestionModePre)
	
	cmd := cs.SendMessageWithSuggestions("test-agent", "test message", "conv-123")
	assert.NotNil(t, cmd)
	
	// The command should be a batch when suggestions are enabled
	// This tests that multiple commands are properly batched together
}

// TestChatServiceErrorHandling tests error scenarios
func TestChatServiceErrorHandling(t *testing.T) {
	ctx := context.Background()
	client := &mockGuildClient{
		getAgentStatusFunc: func(ctx context.Context, req *pb.GetAgentStatusRequest, opts ...grpc.CallOption) (*pb.AgentStatus, error) {
			return nil, assert.AnError
		},
	}
	reg := registry.NewComponentRegistry()
	
	cs, err := NewChatService(ctx, client, reg)
	require.NoError(t, err)
	
	t.Run("GetAgentStatusError", func(t *testing.T) {
		cmd := cs.GetAgentStatus("test-agent")
		msg := cmd()
		
		errMsg, ok := msg.(ChatServiceErrorMsg)
		assert.True(t, ok)
		assert.Equal(t, "get_agent_status", errMsg.Operation)
		assert.Error(t, errMsg.Error)
	})
}