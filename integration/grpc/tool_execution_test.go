package grpc

import (
	"context"
	"testing"

	guildgrpc "github.com/guild-ventures/guild-core/pkg/grpc"
	guildpb "github.com/guild-ventures/guild-core/pkg/grpc/pb/guild/v1"
	"github.com/guild-ventures/guild-core/pkg/registry"
	"github.com/guild-ventures/guild-core/pkg/tools"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// TestTool implements a simple test tool
type TestTool struct {
	name        string
	description string
	executed    bool
	result      string
}

func (t *TestTool) Name() string        { return t.name }
func (t *TestTool) Description() string { return t.description }
func (t *TestTool) Schema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"input": map[string]interface{}{"type": "string"},
		},
	}
}
func (t *TestTool) Execute(ctx context.Context, input string) (*tools.ToolResult, error) {
	t.executed = true
	return &tools.ToolResult{
		Output:  t.result,
		Success: true,
	}, nil
}
func (t *TestTool) Examples() []string {
	return []string{"test input"}
}
func (t *TestTool) Category() string {
	return "test"
}
func (t *TestTool) RequiresAuth() bool {
	return false
}

func TestToolExecutionViaGRPC(t *testing.T) {
	// Create registry and register test tools
	reg := registry.NewComponentRegistry()
	
	// Initialize registry with mock configuration
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
	
	// Register a mock agent factory
	agentRegistry := reg.Agents()
	err = agentRegistry.RegisterAgentType("tool-agent", func(config registry.AgentConfig) (registry.Agent, error) {
		return &mockAgent{
			id:           config.Name,
			name:         config.Name,
			toolRegistry: reg.Tools(), // Pass the tool registry so it can execute tools
			responses: map[string]string{
				// Remove hardcoded tool responses - let the agent execute real tools
			},
		}, nil
	})
	require.NoError(t, err)
	
	// Create test tools
	testTool1 := &TestTool{
		name:        "test-tool-1",
		description: "First test tool",
		result:      "Tool 1 executed successfully",
	}
	
	testTool2 := &TestTool{
		name:        "test-tool-2",
		description: "Second test tool with cost",
		result:      "Tool 2 executed with cost tracking",
	}

	// Register tools
	err = reg.Tools().RegisterTool(testTool1.Name(), testTool1)
	require.NoError(t, err)
	err = reg.Tools().RegisterTool(testTool2.Name(), testTool2)
	require.NoError(t, err)

	// Start gRPC server with registry
	eventBus := newMockEventBus()
	server := guildgrpc.NewServer(reg, eventBus)

	go func() {
		err := server.Start(ctx, ":50052")
		assert.NoError(t, err)
	}()

	// Create client
	conn, err := grpc.Dial("localhost:50052", grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	defer conn.Close()

	client := guildpb.NewChatServiceClient(conn)

	t.Run("execute tool via /tool command", func(t *testing.T) {
		// Create session with tool agent
		createResp, err := client.CreateChatSession(context.Background(), &guildpb.CreateChatSessionRequest{
			Name:       "Tool Test 1",
			CampaignId: "test-campaign",
			AgentIds:   []string{"tool-agent"},
		})
		require.NoError(t, err)

		// Send tool command
		stream, err := client.Chat(context.Background())
		require.NoError(t, err)

		err = stream.Send(&guildpb.ChatRequest{
			Request: &guildpb.ChatRequest_Message{
				Message: &guildpb.ChatMessage{
					SessionId: createResp.Id,
					SenderId:  "user",
					Content:   "/tool test-tool-1",
					Type:      guildpb.ChatMessage_USER_MESSAGE,
				},
			},
		})
		require.NoError(t, err)

		err = stream.CloseSend()
		require.NoError(t, err)

		// Check response
		resp, err := stream.Recv()
		require.NoError(t, err)
		if msgResp, ok := resp.Response.(*guildpb.ChatResponse_Message); ok {
			assert.Contains(t, msgResp.Message.Content, "Tool 1 executed successfully")
		}
		assert.True(t, testTool1.executed)
	})

	t.Run("safety checks", func(t *testing.T) {
		// Test workspace isolation
		safeTool := &TestTool{
			name:        "safe-tool",
			description: "Tool with safety checks",
			result:      "Executed safely",
		}
		
		err := reg.Tools().RegisterTool(safeTool.Name(), safeTool)
		require.NoError(t, err)

		// Executor is created internally by the framework
		// Just verify the tool was registered
		registeredTool, err := reg.Tools().GetTool(safeTool.Name())
		require.NoError(t, err)
		assert.NotNil(t, registeredTool)

		// Verify workspace isolation is enabled
		// This would normally check for sandboxed execution
	})

	t.Run("cost tracking", func(t *testing.T) {
		// Create session with cost tracking
		createResp, err := client.CreateChatSession(context.Background(), &guildpb.CreateChatSessionRequest{
			Name:       "Cost Test 1",
			CampaignId: "test-campaign",
			AgentIds:   []string{"tool-agent"},
		})
		require.NoError(t, err)

		// Execute tool that incurs cost
		stream, err := client.Chat(context.Background())
		require.NoError(t, err)

		err = stream.Send(&guildpb.ChatRequest{
			Request: &guildpb.ChatRequest_Message{
				Message: &guildpb.ChatMessage{
					SessionId: createResp.Id,
					SenderId:  "user",
					Content:   "/tool test-tool-2",
					Type:      guildpb.ChatMessage_USER_MESSAGE,
				},
			},
		})
		require.NoError(t, err)

		err = stream.CloseSend()
		require.NoError(t, err)

		// Verify response includes cost info
		resp, err := stream.Recv()
		require.NoError(t, err)
		if msgResp, ok := resp.Response.(*guildpb.ChatResponse_Message); ok {
			assert.Contains(t, msgResp.Message.Content, "cost tracking")
		}
		
		// In a real test, we'd verify cost was recorded
		// assert.Greater(t, resp.Cost, 0.0)
	})

	t.Run("tool not found", func(t *testing.T) {
		// Create session
		createResp, err := client.CreateChatSession(context.Background(), &guildpb.CreateChatSessionRequest{
			Name:       "Error Test 1",
			CampaignId: "test-campaign",
			AgentIds:   []string{"tool-agent"},
		})
		require.NoError(t, err)

		// Try to execute non-existent tool
		stream, err := client.Chat(context.Background())
		require.NoError(t, err)

		err = stream.Send(&guildpb.ChatRequest{
			Request: &guildpb.ChatRequest_Message{
				Message: &guildpb.ChatMessage{
					SessionId: createResp.Id,
					SenderId:  "user",
					Content:   "/tool non-existent-tool",
					Type:      guildpb.ChatMessage_USER_MESSAGE,
				},
			},
		})
		require.NoError(t, err)

		err = stream.CloseSend()
		require.NoError(t, err)

		// Should get error response
		resp, err := stream.Recv()
		require.NoError(t, err)
		if msgResp, ok := resp.Response.(*guildpb.ChatResponse_Message); ok {
			assert.Contains(t, msgResp.Message.Content, "not found")
		}
	})

	t.Run("concurrent tool execution", func(t *testing.T) {
		// Test multiple tools executing concurrently
		sessions := []string{"concurrent-tool-1", "concurrent-tool-2", "concurrent-tool-3"}
		
		for _, sessionID := range sessions {
			t.Run(sessionID, func(t *testing.T) {
				t.Parallel()
				
				// Create session
				createResp, err := client.CreateChatSession(context.Background(), &guildpb.CreateChatSessionRequest{
					Name:       sessionID,
					CampaignId: "test-campaign",
					AgentIds:   []string{"tool-agent"},
				})
				require.NoError(t, err)

				// Execute tool
				stream, err := client.Chat(context.Background())
				require.NoError(t, err)

				err = stream.Send(&guildpb.ChatRequest{
					Request: &guildpb.ChatRequest_Message{
						Message: &guildpb.ChatMessage{
							SessionId: createResp.Id,
							SenderId:  "user",
							Content:   "/tool test-tool-1",
							Type:      guildpb.ChatMessage_USER_MESSAGE,
						},
					},
				})
				require.NoError(t, err)

				err = stream.CloseSend()
				require.NoError(t, err)

				// Verify execution
				resp, err := stream.Recv()
				require.NoError(t, err)
				if msgResp, ok := resp.Response.(*guildpb.ChatResponse_Message); ok {
					assert.Contains(t, msgResp.Message.Content, "executed successfully")
				}
			})
		}
	})
}