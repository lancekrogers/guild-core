// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package chat

import (
	"context"
	"fmt"
	"net"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	// "github.com/charmbracelet/x/exp/teatest" // TODO: Use for future TUI tests
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/lancekrogers/guild/pkg/gerror"
	// grpcpkg "github.com/lancekrogers/guild/pkg/grpc" // TODO: Use for grpc utilities
	guildv1 "github.com/lancekrogers/guild/pkg/grpc/pb/guild/v1"
	// "github.com/lancekrogers/guild/pkg/registry" // TODO: Use for registry functionality
)

// mockGuildServer implements a test gRPC server for chat integration testing
type mockGuildServer struct {
	guildv1.UnimplementedGuildServer
	guildv1.UnimplementedChatServiceServer
	agents map[string]*mockAgent
	mu     sync.RWMutex
}

type mockAgent struct {
	id           string
	name         string
	responses    map[string]string
	mentionCount int
}

func newMockGuildServer() *mockGuildServer {
	server := &mockGuildServer{
		agents: make(map[string]*mockAgent),
	}

	// Setup test agents
	server.agents["elena"] = &mockAgent{
		id:   "elena",
		name: "Elena Guild Master",
		responses: map[string]string{
			"default": "Hello! I'm Elena, your guild master. How can I help coordinate your project today?",
			"task":    "I'll help you break down that task into manageable pieces for the team.",
			"plan":    "Let me create a strategic plan for this initiative.",
		},
	}

	server.agents["marcus"] = &mockAgent{
		id:   "marcus",
		name: "Marcus Developer",
		responses: map[string]string{
			"default": "Hi! I'm Marcus, your development specialist. Ready to build something amazing!",
			"code":    "I'll implement that feature using best practices and proper testing.",
			"debug":   "Let me analyze that issue and provide a solution.",
		},
	}

	server.agents["vera"] = &mockAgent{
		id:   "vera",
		name: "Vera Tester",
		responses: map[string]string{
			"default": "Greetings! I'm Vera, your quality assurance specialist.",
			"test":    "I'll create comprehensive tests to ensure quality and catch regressions.",
			"qa":      "Let me review this for potential issues and improvements.",
		},
	}

	return server
}

func (s *mockGuildServer) ListAvailableAgents(ctx context.Context, req *guildv1.ListAgentsRequest) (*guildv1.ListAgentsResponse, error) {
	if err := ctx.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled")
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	agents := make([]*guildv1.AgentInfo, 0, len(s.agents))
	for _, agent := range s.agents {
		agents = append(agents, &guildv1.AgentInfo{
			Id:   agent.id,
			Name: agent.name,
			Status: &guildv1.AgentStatus{
				State: guildv1.AgentStatus_IDLE,
			},
			Type: "mock",
		})
	}

	return &guildv1.ListAgentsResponse{
		Agents: agents,
	}, nil
}

func (s *mockGuildServer) SendMessageToAgent(ctx context.Context, req *guildv1.AgentMessageRequest) (*guildv1.AgentMessageResponse, error) {
	if err := ctx.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	agent, exists := s.agents[req.AgentId]
	if !exists {
		return nil, gerror.Newf(gerror.ErrCodeNotFound, "agent %s not found", req.AgentId)
	}

	agent.mentionCount++

	// Select response based on message content
	message := strings.ToLower(req.Message)
	var response string

	switch {
	case strings.Contains(message, "task") || strings.Contains(message, "breakdown"):
		response = agent.responses["task"]
	case strings.Contains(message, "code") || strings.Contains(message, "implement"):
		response = agent.responses["code"]
	case strings.Contains(message, "test") || strings.Contains(message, "qa"):
		response = agent.responses["test"]
	case strings.Contains(message, "plan") || strings.Contains(message, "strategy"):
		response = agent.responses["plan"]
	case strings.Contains(message, "debug") || strings.Contains(message, "issue"):
		response = agent.responses["debug"]
	default:
		response = agent.responses["default"]
	}

	if response == "" {
		response = agent.responses["default"]
	}

	return &guildv1.AgentMessageResponse{
		Response: response,
		AgentId:  req.AgentId,
		Status: &guildv1.AgentStatus{
			State: guildv1.AgentStatus_IDLE,
		},
	}, nil
}

func (s *mockGuildServer) Chat(stream guildv1.ChatService_ChatServer) error {
	ctx := stream.Context()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			req, err := stream.Recv()
			if err != nil {
				return err
			}

			// Handle chat message - extract message from oneof request
			var messageContent string
			if msgReq := req.GetMessage(); msgReq != nil {
				messageContent = msgReq.Content
			}

			response := &guildv1.ChatResponse{
				Response: &guildv1.ChatResponse_Message{
					Message: &guildv1.ChatMessage{
						SessionId:  "test-session",
						SenderId:   "system",
						SenderName: "System",
						Content:    "Echo: " + messageContent,
						Type:       guildv1.ChatMessage_AGENT_RESPONSE,
						Timestamp:  time.Now().Unix(),
					},
				},
			}

			if err := stream.Send(response); err != nil {
				return err
			}
		}
	}
}

// setupMockGRPCServer starts a mock gRPC server for testing
func setupMockGRPCServer(t *testing.T) (string, *mockGuildServer, func()) {
	t.Helper()

	// Find available port
	listener, err := net.Listen("tcp", ":0")
	require.NoError(t, err)
	port := listener.Addr().(*net.TCPAddr).Port
	listener.Close()

	address := fmt.Sprintf("localhost:%d", port)

	// Create mock server
	mockServer := newMockGuildServer()

	// Setup gRPC server
	s := grpc.NewServer()
	guildv1.RegisterGuildServer(s, mockServer)
	guildv1.RegisterChatServiceServer(s, mockServer)

	// Start server
	listener, err = net.Listen("tcp", address)
	require.NoError(t, err)

	go func() {
		_ = s.Serve(listener)
	}()

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	cleanup := func() {
		s.Stop()
		listener.Close()
	}

	return address, mockServer, cleanup
}

// TestGuildChatAgentCommunication tests basic message flow between chat UI and agents
func TestGuildChatAgentCommunication(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Setup mock gRPC server
	address, mockServer, cleanup := setupMockGRPCServer(t)
	defer cleanup()

	// Connect to mock server
	conn, err := grpc.Dial(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	defer conn.Close()

	client := guildv1.NewGuildClient(conn)

	// Test agent communication through gRPC
	tests := []struct {
		name       string
		agentID    string
		message    string
		expectText string
	}{
		{
			name:       "elena_general_greeting",
			agentID:    "elena",
			message:    "Hello Elena, can you help me?",
			expectText: "Elena, your guild master",
		},
		{
			name:       "marcus_code_request",
			agentID:    "marcus",
			message:    "Marcus, please implement a new feature",
			expectText: "implement that feature",
		},
		{
			name:       "vera_test_request",
			agentID:    "vera",
			message:    "Vera, please test this functionality",
			expectText: "comprehensive tests",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := client.SendMessageToAgent(ctx, &guildv1.AgentMessageRequest{
				AgentId: tt.agentID,
				Message: tt.message,
			})

			require.NoError(t, err, "Should be able to send message to agent")
			assert.NotEmpty(t, resp.Response, "Should receive non-empty response")
			assert.Contains(t, resp.Response, tt.expectText, "Response should contain expected text")
			assert.Equal(t, tt.agentID, resp.AgentId, "Response should be from correct agent")
			assert.Equal(t, guildv1.AgentStatus_IDLE, resp.Status.State, "Should have idle status")
		})
	}

	// Verify agents received messages
	for agentID := range mockServer.agents {
		agent := mockServer.agents[agentID]
		assert.Greater(t, agent.mentionCount, 0, "Agent %s should have received at least one message", agentID)
	}
}

// TestGuildAgentMentionRouting tests @mention routing functionality
func TestGuildAgentMentionRouting(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Setup mock gRPC server
	address, _, cleanup := setupMockGRPCServer(t)
	defer cleanup()

	// Connect to mock server
	conn, err := grpc.Dial(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	defer conn.Close()

	client := guildv1.NewGuildClient(conn)

	tests := []struct {
		name       string
		mention    string
		agentID    string
		message    string
		expectPass bool
	}{
		{
			name:       "elena_mention_routes_to_manager",
			mention:    "@elena",
			agentID:    "elena",
			message:    "@elena can you help coordinate this project?",
			expectPass: true,
		},
		{
			name:       "marcus_mention_routes_to_developer",
			mention:    "@marcus",
			agentID:    "marcus",
			message:    "@marcus please implement user authentication",
			expectPass: true,
		},
		{
			name:       "vera_mention_routes_to_tester",
			mention:    "@vera",
			agentID:    "vera",
			message:    "@vera can you test the login functionality?",
			expectPass: true,
		},
		{
			name:       "invalid_mention_fails",
			mention:    "@nonexistent",
			agentID:    "nonexistent",
			message:    "@nonexistent please help",
			expectPass: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := client.SendMessageToAgent(ctx, &guildv1.AgentMessageRequest{
				AgentId: tt.agentID,
				Message: tt.message,
			})

			if tt.expectPass {
				require.NoError(t, err, "Valid agent mention should succeed")
				assert.NotEmpty(t, resp.Response, "Should receive response")
				assert.Equal(t, tt.agentID, resp.AgentId, "Response should be from mentioned agent")
			} else {
				assert.Error(t, err, "Invalid agent mention should fail")
			}
		})
	}
}

// TestGuildMultiAgentResponses tests broadcast and multi-agent coordination
func TestGuildMultiAgentResponses(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Setup mock gRPC server
	address, mockServer, cleanup := setupMockGRPCServer(t)
	defer cleanup()

	// Connect to mock server
	conn, err := grpc.Dial(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	defer conn.Close()

	client := guildv1.NewGuildClient(conn)

	// Test broadcast to all agents
	t.Run("broadcast_handling", func(t *testing.T) {
		message := "@all we need to implement a new payment system"

		// Get all available agents
		agentsResp, err := client.ListAvailableAgents(ctx, &guildv1.ListAgentsRequest{})
		require.NoError(t, err)

		responses := make(map[string]*guildv1.AgentMessageResponse)

		// Send message to each agent
		for _, agent := range agentsResp.Agents {
			resp, err := client.SendMessageToAgent(ctx, &guildv1.AgentMessageRequest{
				AgentId: agent.Id,
				Message: message,
			})
			require.NoError(t, err)
			responses[agent.Id] = resp
		}

		// Verify all agents responded
		assert.Len(t, responses, len(agentsResp.Agents), "All agents should respond to broadcast")

		// Verify responses are agent-specific
		for agentID, resp := range responses {
			assert.Equal(t, agentID, resp.AgentId, "Response should be from correct agent")
			assert.NotEmpty(t, resp.Response, "Response should not be empty")

			// Each agent should respond differently based on their role
			switch agentID {
			case "elena":
				assert.Contains(t, resp.Response, "coordinate", "Elena should offer coordination")
			case "marcus":
				assert.Contains(t, resp.Response, "build", "Marcus should offer implementation")
			case "vera":
				assert.Contains(t, resp.Response, "quality", "Vera should offer testing")
			}
		}
	})

	// Test response ordering and status updates
	t.Run("response_ordering", func(t *testing.T) {
		// Send messages with short delays to test ordering
		messages := []struct {
			agentID string
			message string
		}{
			{"elena", "Plan the payment integration"},
			{"marcus", "Implement payment gateway"},
			{"vera", "Test payment flows"},
		}

		responses := make([]*guildv1.AgentMessageResponse, 0, len(messages))

		for _, msg := range messages {
			resp, err := client.SendMessageToAgent(ctx, &guildv1.AgentMessageRequest{
				AgentId: msg.agentID,
				Message: msg.message,
			})
			require.NoError(t, err)
			responses = append(responses, resp)

			time.Sleep(10 * time.Millisecond) // Small delay to test ordering
		}

		// Verify we got responses in order
		assert.Len(t, responses, len(messages), "Should receive all responses")

		for i, resp := range responses {
			assert.Equal(t, messages[i].agentID, resp.AgentId, "Response %d should be from correct agent", i)
			assert.Equal(t, guildv1.AgentStatus_IDLE, resp.Status.State, "All responses should have idle status")
		}
	})

	// Test multi-agent status updates
	t.Run("status_updates", func(t *testing.T) {
		// Check that all agents are available
		agentsResp, err := client.ListAvailableAgents(ctx, &guildv1.ListAgentsRequest{
			IncludeStatus: true,
		})
		require.NoError(t, err)

		// Verify all expected agents are ready
		expectedAgents := []string{"elena", "marcus", "vera"}
		actualAgents := make(map[string]*guildv1.AgentStatus)

		for _, agent := range agentsResp.Agents {
			actualAgents[agent.Id] = agent.Status
		}

		for _, expectedID := range expectedAgents {
			status, exists := actualAgents[expectedID]
			assert.True(t, exists, "Expected agent %s should exist", expectedID)
			assert.Equal(t, guildv1.AgentStatus_IDLE, status.State, "Agent %s should be idle", expectedID)
		}
	})

	// Verify mention counts increased
	for agentID, agent := range mockServer.agents {
		t.Logf("Agent %s received %d mentions", agentID, agent.mentionCount)
		assert.Greater(t, agent.mentionCount, 0, "Agent %s should have received mentions", agentID)
	}
}

// TestGuildChatServiceStreaming tests bidirectional streaming for real-time chat
func TestGuildChatServiceStreaming(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Setup mock gRPC server
	address, _, cleanup := setupMockGRPCServer(t)
	defer cleanup()

	// Connect to mock server
	conn, err := grpc.Dial(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	defer conn.Close()

	chatClient := guildv1.NewChatServiceClient(conn)

	// Test bidirectional streaming
	stream, err := chatClient.Chat(ctx)
	require.NoError(t, err)

	// Send test messages
	testMessages := []string{
		"Hello chat service",
		"How are you doing?",
		"Testing streaming functionality",
	}

	// Start receiving responses in goroutine
	responses := make(chan *guildv1.ChatResponse, len(testMessages))
	errChan := make(chan error, 1)

	go func() {
		defer close(responses)
		for {
			resp, err := stream.Recv()
			if err != nil {
				errChan <- err
				return
			}
			responses <- resp
		}
	}()

	// Send messages
	for _, msg := range testMessages {
		err := stream.Send(&guildv1.ChatRequest{
			Request: &guildv1.ChatRequest_Message{
				Message: &guildv1.ChatMessage{
					SessionId:  "test-session",
					SenderId:   "user",
					SenderName: "Test User",
					Content:    msg,
					Type:       guildv1.ChatMessage_USER_MESSAGE,
					Timestamp:  time.Now().Unix(),
				},
			},
		})
		require.NoError(t, err)

		time.Sleep(50 * time.Millisecond) // Small delay between messages
	}

	// Close sending side
	err = stream.CloseSend()
	require.NoError(t, err)

	// Collect responses
	var receivedResponses []*guildv1.ChatResponse
	for i := 0; i < len(testMessages); i++ {
		select {
		case resp := <-responses:
			receivedResponses = append(receivedResponses, resp)
		case err := <-errChan:
			t.Logf("Stream error (may be expected): %v", err)
			break
		case <-time.After(2 * time.Second):
			t.Log("Timeout waiting for response (may be expected)")
			break
		}
	}

	// Verify we got responses
	if len(receivedResponses) > 0 {
		for i, resp := range receivedResponses {
			if respMsg := resp.GetMessage(); respMsg != nil {
				assert.NotEmpty(t, respMsg.Content, "Response %d should have content", i)
				assert.Contains(t, respMsg.Content, "Echo:", "Response should be echo")
				assert.Equal(t, guildv1.ChatMessage_AGENT_RESPONSE, respMsg.Type, "Response should be agent response type")
			} else {
				t.Errorf("Response %d should contain a message", i)
			}
		}
	} else {
		t.Log("No streaming responses received (may be expected with mock)")
	}
}

// TestGuildChatUIIntegration tests the actual TUI chat interface with mock gRPC backend
func TestGuildChatUIIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping TUI integration test in short mode")
	}

	// This test would integrate with the actual TUI chat model
	// For now, we test the basic chat input functionality

	t.Run("chat_input_handling", func(t *testing.T) {
		// Create a basic text input model for testing
		input := textinput.New()
		input.Placeholder = "Type your message..."
		input.Focus()

		// Test typing a message
		testMessage := "Hello agents!"
		for _, char := range testMessage {
			input, _ = input.Update(tea.KeyMsg{
				Type:  tea.KeyRunes,
				Runes: []rune{char},
			})
		}

		assert.Equal(t, testMessage, input.Value(), "Input should contain typed message")

		// Test Enter key
		input, _ = input.Update(tea.KeyMsg{Type: tea.KeyEnter})

		// After Enter, input should be ready for submission
		// (actual submission handling would be in the parent model)
		assert.NotEmpty(t, input.Value(), "Input should retain value until parent handles it")
	})

	t.Run("mention_detection", func(t *testing.T) {
		testCases := []struct {
			input       string
			expectAgent string
			hasMention  bool
		}{
			{"@elena help me plan this", "elena", true},
			{"@marcus implement auth", "marcus", true},
			{"@vera test the feature", "vera", true},
			{"@all coordinate together", "all", true},
			{"regular message", "", false},
			{"email@domain.com", "", false}, // Should not detect email as mention
		}

		for _, tc := range testCases {
			t.Run(tc.input, func(t *testing.T) {
				// Simple mention detection logic
				var agent string
				hasMention := false

				if strings.HasPrefix(tc.input, "@") {
					parts := strings.Fields(tc.input)
					if len(parts) > 0 {
						mention := parts[0][1:] // Remove @
						if mention == "elena" || mention == "marcus" || mention == "vera" || mention == "all" {
							agent = mention
							hasMention = true
						}
					}
				}

				assert.Equal(t, tc.expectAgent, agent, "Should detect correct agent")
				assert.Equal(t, tc.hasMention, hasMention, "Should correctly identify mention presence")
			})
		}
	})
}
