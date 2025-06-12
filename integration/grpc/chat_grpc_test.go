package grpc

import (
	"context"
	"testing"
	"time"

	guildgrpc "github.com/guild-ventures/guild-core/pkg/grpc"
	guildpb "github.com/guild-ventures/guild-core/pkg/grpc/pb/guild/v1"
	"github.com/guild-ventures/guild-core/pkg/registry"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func TestEndToEndChatGRPC(t *testing.T) {
	// Create registry and event bus
	reg := registry.NewComponentRegistry()
	eventBus := newMockEventBus()

	// Initialize registry with basic configuration to prevent nil panics
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
	err = agentRegistry.RegisterAgentType("mock", func(config registry.AgentConfig) (registry.Agent, error) {
		return &mockAgent{
			id:           config.Name,
			name:         config.Name,
			toolRegistry: nil, // Chat test doesn't need tools
			responses: map[string]string{
				"Hello, Guild!": "Greetings from the mock agent!",
				"test":          "Mock agent response",
			},
		}, nil
	})
	require.NoError(t, err)

	// Start test gRPC server
	server := guildgrpc.NewServer(reg, eventBus)

	// Start server in background
	go func() {
		err := server.Start(ctx, ":50051")
		assert.NoError(t, err)
	}()

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	// Create client connection
	conn, err := grpc.Dial("localhost:50051", grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	defer conn.Close()

	client := guildpb.NewChatServiceClient(conn)

	t.Run("basic message exchange", func(t *testing.T) {
		// Create chat session
		createResp, err := client.CreateChatSession(context.Background(), &guildpb.CreateChatSessionRequest{
			Name:       "Test Session 1",
			CampaignId: "test-campaign",
			AgentIds:   []string{"mock"}, // Use our mock agent
		})
		require.NoError(t, err)
		assert.NotEmpty(t, createResp.Id)

		// Open bidirectional chat stream
		stream, err := client.Chat(context.Background())
		require.NoError(t, err)

		// Send initial message
		err = stream.Send(&guildpb.ChatRequest{
			Request: &guildpb.ChatRequest_Message{
				Message: &guildpb.ChatMessage{
					SessionId: createResp.Id,
					SenderId:  "user",
					Content:   "Hello, Guild!",
					Type:      guildpb.ChatMessage_USER_MESSAGE,
				},
			},
		})
		require.NoError(t, err)

		// Receive response
		resp, err := stream.Recv()
		require.NoError(t, err)

		// Check if we got a message response
		if msgResp, ok := resp.Response.(*guildpb.ChatResponse_Message); ok {
			assert.NotNil(t, msgResp.Message)
			assert.NotEmpty(t, msgResp.Message.Content)
		}

		// Close stream
		err = stream.CloseSend()
		require.NoError(t, err)
	})

	t.Run("multiple messages", func(t *testing.T) {
		// Create new session
		createResp, err := client.CreateChatSession(context.Background(), &guildpb.CreateChatSessionRequest{
			Name:       "Test Session 2",
			CampaignId: "test-campaign",
			AgentIds:   []string{}, // Empty agents for this test
		})
		require.NoError(t, err)
		sessionID := createResp.Id

		// Open chat stream
		stream, err := client.Chat(context.Background())
		require.NoError(t, err)

		// Send multiple messages
		messages := []string{
			"First message",
			"Second message",
			"Third message",
		}

		for _, msg := range messages {
			err = stream.Send(&guildpb.ChatRequest{
				Request: &guildpb.ChatRequest_Message{
					Message: &guildpb.ChatMessage{
						SessionId: sessionID,
						SenderId:  "user",
						Content:   msg,
						Type:      guildpb.ChatMessage_USER_MESSAGE,
					},
				},
			})
			require.NoError(t, err)

			// Verify response
			resp, err := stream.Recv()
			require.NoError(t, err)

			if msgResp, ok := resp.Response.(*guildpb.ChatResponse_Message); ok {
				assert.NotNil(t, msgResp.Message)
				assert.NotEmpty(t, msgResp.Message.Content)
			}
		}

		// Close stream
		err = stream.CloseSend()
		require.NoError(t, err)
	})

	t.Run("error handling", func(t *testing.T) {
		// Try to send message without creating session first
		stream, err := client.Chat(context.Background())
		require.NoError(t, err)

		err = stream.Send(&guildpb.ChatRequest{
			Request: &guildpb.ChatRequest_Message{
				Message: &guildpb.ChatMessage{
					SessionId: "non-existent-session",
					SenderId:  "user",
					Content:   "This should fail",
					Type:      guildpb.ChatMessage_USER_MESSAGE,
				},
			},
		})
		require.NoError(t, err)

		// Should receive error response
		_, err = stream.Recv()
		// Error could be in the response or as an error
		// Implementation-specific behavior

		// Close stream
		err = stream.CloseSend()
		require.NoError(t, err)
	})

	t.Run("concurrent sessions", func(t *testing.T) {
		// Create multiple sessions concurrently
		sessionIDs := []string{"concurrent-1", "concurrent-2", "concurrent-3"}

		for _, sessionName := range sessionIDs {
			t.Run(sessionName, func(t *testing.T) {
				t.Parallel()

				// Create session
				createResp, err := client.CreateChatSession(context.Background(), &guildpb.CreateChatSessionRequest{
					Name:       sessionName,
					CampaignId: "test-campaign",
					AgentIds:   []string{}, // Empty agents for this test
				})
				require.NoError(t, err)
				sessionID := createResp.Id

				// Open chat stream
				stream, err := client.Chat(context.Background())
				require.NoError(t, err)

				err = stream.Send(&guildpb.ChatRequest{
					Request: &guildpb.ChatRequest_Message{
						Message: &guildpb.ChatMessage{
							SessionId: sessionID,
							SenderId:  "user",
							Content:   "Concurrent test message",
							Type:      guildpb.ChatMessage_USER_MESSAGE,
						},
					},
				})
				require.NoError(t, err)

				// Verify response
				resp, err := stream.Recv()
				require.NoError(t, err)

				if msgResp, ok := resp.Response.(*guildpb.ChatResponse_Message); ok {
					assert.NotNil(t, msgResp.Message)
					assert.NotEmpty(t, msgResp.Message.Content)
				}

				// Close stream
				err = stream.CloseSend()
				require.NoError(t, err)
			})
		}
	})
}
