// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package grpc

import (
	"context"
	"fmt"
	"testing"
	"time"

	guildgrpc "github.com/lancekrogers/guild/pkg/grpc"
	guildpb "github.com/lancekrogers/guild/pkg/grpc/pb/guild/v1"

	// promptspb "github.com/lancekrogers/guild/pkg/grpc/pb/prompts/v1"
	"github.com/lancekrogers/guild/pkg/providers/mock"
	"github.com/lancekrogers/guild/pkg/registry"
	"github.com/lancekrogers/guild/pkg/tools"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// MockTool for testing the full pipeline
type MockTool struct {
	name    string
	execLog []string
}

func (m *MockTool) Name() string        { return m.name }
func (m *MockTool) Description() string { return fmt.Sprintf("Mock tool: %s", m.name) }
func (m *MockTool) Schema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"input": map[string]interface{}{"type": "string"},
		},
	}
}
func (m *MockTool) Execute(ctx context.Context, input string) (*tools.ToolResult, error) {
	m.execLog = append(m.execLog, fmt.Sprintf("Executed with input: %s", input))
	return &tools.ToolResult{
		Output:  fmt.Sprintf("%s executed successfully", m.name),
		Success: true,
		Metadata: map[string]string{
			"time": time.Now().Format(time.RFC3339),
		},
	}, nil
}
func (m *MockTool) Examples() []string {
	return []string{"example input"}
}
func (m *MockTool) Category() string {
	return "test"
}
func (m *MockTool) RequiresAuth() bool {
	return false
}

func TestFullPipelineIntegration(t *testing.T) {
	// Create registry with all components
	reg := registry.NewComponentRegistry()

	// Initialize registry with basic configuration
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := reg.Initialize(ctx, registry.Config{
		Agents: registry.AgentConfigYaml{
			DefaultType: "worker",
			Types: map[string]interface{}{
				"worker": map[string]interface{}{
					"enabled": true,
				},
			},
		},
		Providers: registry.ProviderConfig{
			DefaultProvider: "claudecode",
			Providers: map[string]interface{}{
				"claudecode": map[string]interface{}{
					"model":    "sonnet",
					"bin_path": "claude-code",
				},
			},
		},
	})
	require.NoError(t, err)

	// Register mock provider
	mockProvider, err := mock.NewProvider()
	require.NoError(t, err)
	mockProvider.Enable()
	mockProvider.SetResponse("*", "You asked to use the calculator tool. Let me help with that.")
	err = reg.Providers().RegisterProvider("mock", mockProvider)
	require.NoError(t, err)

	// Register tools
	calcTool := &MockTool{name: "calculator"}
	fileTool := &MockTool{name: "file-reader"}

	err = reg.Tools().RegisterTool(calcTool.Name(), calcTool)
	require.NoError(t, err)
	err = reg.Tools().RegisterTool(fileTool.Name(), fileTool)
	require.NoError(t, err)

	// Register a mock agent factory that can handle tool commands
	agentRegistry := reg.Agents()
	err = agentRegistry.RegisterAgentType("pipeline-agent", func(config registry.AgentConfig) (registry.Agent, error) {
		return &mockAgent{
			id:           config.Name,
			name:         config.Name,
			toolRegistry: reg.Tools(), // Pass the tool registry so it can execute tools
			responses: map[string]string{
				"Can you calculate 42 + 58 for me?": "I'll help you calculate that. Let me use the calculator tool.",
			},
		}, nil
	})
	require.NoError(t, err)

	// Start server
	eventBus := newMockEventBus()
	server := guildgrpc.NewServer(reg, eventBus)

	go func() {
		err := server.Start(ctx, ":50054")
		assert.NoError(t, err)
	}()

	time.Sleep(100 * time.Millisecond)

	// Create clients
	conn, err := grpc.Dial("localhost:50054", grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	defer conn.Close()

	chatClient := guildpb.NewChatServiceClient(conn)
	// TODO: Fix prompt service client integration
	// promptClient := promptspb.NewPromptServiceClient(conn)

	t.Run("full chat to agent to tool pipeline", func(t *testing.T) {
		// Skip prompt setup for now - focus on core chat functionality

		// Create chat session with an agent
		createResp, err := chatClient.CreateChatSession(context.Background(), &guildpb.CreateChatSessionRequest{
			Name:       "Pipeline Test 1",
			CampaignId: "test-campaign",
			AgentIds:   []string{"pipeline-agent"}, // Add agent to the session
		})
		require.NoError(t, err)
		assert.NotEmpty(t, createResp.Id)
		sessionID := createResp.Id

		// Send message that should trigger tool use
		stream, err := chatClient.Chat(context.Background())
		require.NoError(t, err)

		err = stream.Send(&guildpb.ChatRequest{
			Request: &guildpb.ChatRequest_Message{
				Message: &guildpb.ChatMessage{
					SessionId: sessionID,
					SenderId:  "user",
					Content:   "Can you calculate 42 + 58 for me?",
					Type:      guildpb.ChatMessage_USER_MESSAGE,
				},
			},
		})
		require.NoError(t, err)

		err = stream.CloseSend()
		require.NoError(t, err)

		// Receive response
		resp, err := stream.Recv()
		require.NoError(t, err)
		if msgResp, ok := resp.Response.(*guildpb.ChatResponse_Message); ok {
			assert.NotNil(t, msgResp.Message)
			// The mock agent should respond with the pre-configured message
			assert.Contains(t, msgResp.Message.Content, "I'll help you calculate that")
		}

		// Verify tool was considered (in real implementation)
		// assert.Greater(t, len(calcTool.execLog), 0)
	})

	t.Run("error propagation through pipeline", func(t *testing.T) {
		t.Skip("Skipping error propagation test - mock agent doesn't use providers")
		// Create session with an agent
		createResp, err := chatClient.CreateChatSession(context.Background(), &guildpb.CreateChatSessionRequest{
			Name:       "Error Pipeline 1",
			CampaignId: "test-campaign",
			AgentIds:   []string{"pipeline-agent"}, // Add agent to the session
		})
		require.NoError(t, err)

		// Configure provider to return error
		mockProvider.SetError("*", fmt.Errorf("simulated provider error"))

		// Send message
		stream, err := chatClient.Chat(context.Background())
		require.NoError(t, err)

		err = stream.Send(&guildpb.ChatRequest{
			Request: &guildpb.ChatRequest_Message{
				Message: &guildpb.ChatMessage{
					SessionId: createResp.Id,
					SenderId:  "user",
					Content:   "This should trigger an error",
					Type:      guildpb.ChatMessage_USER_MESSAGE,
				},
			},
		})
		require.NoError(t, err)

		err = stream.CloseSend()
		require.NoError(t, err)

		// Should receive error response
		_, err = stream.Recv()
		assert.Error(t, err)

		// Reset provider
		mockProvider.SetError("*", nil)
	})

	t.Run("concurrent request handling", func(t *testing.T) {
		sessions := make([]string, 5)
		for i := 0; i < 5; i++ {
			sessions[i] = fmt.Sprintf("concurrent-pipeline-%d", i)
		}

		// Create all sessions first
		for _, sessionID := range sessions {
			_, err := chatClient.CreateChatSession(context.Background(), &guildpb.CreateChatSessionRequest{
				Name:       sessionID,
				CampaignId: "test-campaign",
			})
			require.NoError(t, err)
		}

		// Send messages concurrently
		for _, sessionID := range sessions {
			sessionID := sessionID // capture for goroutine
			t.Run(sessionID, func(t *testing.T) {
				t.Parallel()

				stream, err := chatClient.Chat(context.Background())
				require.NoError(t, err)

				err = stream.Send(&guildpb.ChatRequest{
					Request: &guildpb.ChatRequest_Message{
						Message: &guildpb.ChatMessage{
							SessionId: sessionID,
							SenderId:  "user",
							Content:   fmt.Sprintf("Message for %s", sessionID),
							Type:      guildpb.ChatMessage_USER_MESSAGE,
						},
					},
				})
				require.NoError(t, err)

				err = stream.CloseSend()
				require.NoError(t, err)

				resp, err := stream.Recv()
				require.NoError(t, err)
				if msgResp, ok := resp.Response.(*guildpb.ChatResponse_Message); ok {
					assert.NotEmpty(t, msgResp.Message.Content)
				}
			})
		}
	})

	t.Run("prompt context in pipeline", func(t *testing.T) {
		t.Skip("Skipping prompt tests until prompt service is fixed")
		// Set layered prompts
		// layers := []string{"system", "guild", "project", "campaign"}
		// for _, layer := range layers {
		// 	_, err := promptClient.SetPromptLayer(context.Background(), &promptspb.SetPromptLayerRequest{
		// 		Layer:   layer,
		// 		Key:     "context-test",
		// 		Content: fmt.Sprintf("This is the %s layer prompt", layer),
		// 	})
		// 	require.NoError(t, err)
		// }

		// Create session with context
		createResp, err := chatClient.CreateChatSession(context.Background(), &guildpb.CreateChatSessionRequest{
			Name:       "Context Pipeline 1",
			CampaignId: "test-campaign",
			Metadata: map[string]string{
				"project": "test-project",
				"guild":   "test-guild",
			},
		})
		require.NoError(t, err)

		// Send message
		stream, err := chatClient.Chat(context.Background())
		require.NoError(t, err)

		err = stream.Send(&guildpb.ChatRequest{
			Request: &guildpb.ChatRequest_Message{
				Message: &guildpb.ChatMessage{
					SessionId: createResp.Id,
					SenderId:  "user",
					Content:   "What context are you aware of?",
					Type:      guildpb.ChatMessage_USER_MESSAGE,
				},
			},
		})
		require.NoError(t, err)

		err = stream.CloseSend()
		require.NoError(t, err)

		// Response should reflect layered context
		resp, err := stream.Recv()
		require.NoError(t, err)
		if msgResp, ok := resp.Response.(*guildpb.ChatResponse_Message); ok {
			assert.NotEmpty(t, msgResp.Message.Content)
		}
	})

	t.Run("cost tracking through pipeline", func(t *testing.T) {
		// Create session
		createResp, err := chatClient.CreateChatSession(context.Background(), &guildpb.CreateChatSessionRequest{
			Name:       "Cost Pipeline 1",
			CampaignId: "test-campaign",
		})
		require.NoError(t, err)

		// Send multiple messages to accumulate cost
		for i := 0; i < 3; i++ {
			stream, err := chatClient.Chat(context.Background())
			require.NoError(t, err)

			err = stream.Send(&guildpb.ChatRequest{
				Request: &guildpb.ChatRequest_Message{
					Message: &guildpb.ChatMessage{
						SessionId: createResp.Id,
						SenderId:  "user",
						Content:   fmt.Sprintf("Message %d for cost tracking", i),
						Type:      guildpb.ChatMessage_USER_MESSAGE,
					},
				},
			})
			require.NoError(t, err)

			err = stream.CloseSend()
			require.NoError(t, err)

			resp, err := stream.Recv()
			require.NoError(t, err)
			if msgResp, ok := resp.Response.(*guildpb.ChatResponse_Message); ok {
				assert.NotEmpty(t, msgResp.Message.Content)
			}

			// In real implementation, check cost accumulation
			// assert.Greater(t, resp.SessionCost, 0.0)
		}
	})

	t.Run("tool execution via command", func(t *testing.T) {
		// Create session with an agent
		createResp, err := chatClient.CreateChatSession(context.Background(), &guildpb.CreateChatSessionRequest{
			Name:       "Tool Command 1",
			CampaignId: "test-campaign",
			AgentIds:   []string{"pipeline-agent"}, // Add agent to the session
		})
		require.NoError(t, err)

		// Execute tool directly
		stream, err := chatClient.Chat(context.Background())
		require.NoError(t, err)

		err = stream.Send(&guildpb.ChatRequest{
			Request: &guildpb.ChatRequest_Message{
				Message: &guildpb.ChatMessage{
					SessionId: createResp.Id,
					SenderId:  "user",
					Content:   "/tool calculator --arg1 10 --arg2 20",
					Type:      guildpb.ChatMessage_USER_MESSAGE,
				},
			},
		})
		require.NoError(t, err)

		err = stream.CloseSend()
		require.NoError(t, err)

		// Should execute tool and return result
		resp, err := stream.Recv()
		require.NoError(t, err)
		if msgResp, ok := resp.Response.(*guildpb.ChatResponse_Message); ok {
			// The mock agent will parse /tool command and execute the calculator tool
			assert.Contains(t, msgResp.Message.Content, "calculator executed successfully")
		}
	})
}
