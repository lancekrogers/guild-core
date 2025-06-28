// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package chat

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"

	"github.com/lancekrogers/guild/internal/daemonconn"
	"github.com/lancekrogers/guild/internal/ui/chat/common/types"
	"github.com/lancekrogers/guild/pkg/config"
	pb "github.com/lancekrogers/guild/pkg/grpc/pb/guild/v1"
	"github.com/lancekrogers/guild/pkg/registry"
)

// MockSessionService implements pb.SessionServiceServer for testing
type MockSessionService struct {
	pb.UnimplementedSessionServiceServer
	sessions []*pb.Session
	messages []*pb.Message
}

func (m *MockSessionService) ListSessions(ctx context.Context, req *pb.ListSessionsRequest) (*pb.ListSessionsResponse, error) {
	return &pb.ListSessionsResponse{
		Sessions:   m.sessions,
		TotalCount: int64(len(m.sessions)),
	}, nil
}

func (m *MockSessionService) CreateSession(ctx context.Context, req *pb.CreateSessionRequest) (*pb.Session, error) {
	session := &pb.Session{
		Id:   "test-session-123",
		Name: req.Name,
	}
	if req.CampaignId != nil {
		session.CampaignId = req.CampaignId
	}
	m.sessions = append(m.sessions, session)
	return session, nil
}

func (m *MockSessionService) StreamMessages(req *pb.StreamMessagesRequest, stream pb.SessionService_StreamMessagesServer) error {
	// Send any existing messages
	for _, msg := range m.messages {
		if msg.SessionId == req.SessionId {
			if err := stream.Send(msg); err != nil {
				return err
			}
		}
	}
	return nil
}

// MockChatService implements pb.ChatServiceServer for testing
type MockChatService struct {
	pb.UnimplementedChatServiceServer
}

func (m *MockChatService) Chat(stream pb.ChatService_ChatServer) error {
	// Echo back messages for testing
	for {
		req, err := stream.Recv()
		if err != nil {
			return err
		}

		if msg := req.GetMessage(); msg != nil {
			// Echo back as agent response
			resp := &pb.ChatResponse{
				Response: &pb.ChatResponse_Message{
					Message: &pb.ChatMessage{
						SessionId:  msg.SessionId,
						SenderId:   "test-agent",
						SenderName: "Test Agent",
						Content:    "Echo: " + msg.Content,
						Type:       pb.ChatMessage_AGENT_RESPONSE,
						Timestamp:  time.Now().Unix(),
						Metadata:   map[string]string{"test": "true"},
					},
				},
			}
			if err := stream.Send(resp); err != nil {
				return err
			}
		}
	}
}

// MockGuildService implements pb.GuildServer for testing
type MockGuildService struct {
	pb.UnimplementedGuildServer
}

// createTestServer creates a test gRPC server with mock services
func createTestServer() (*grpc.Server, *bufconn.Listener) {
	lis := bufconn.Listen(1024 * 1024)
	server := grpc.NewServer()

	// Register mock services
	pb.RegisterSessionServiceServer(server, &MockSessionService{})
	pb.RegisterChatServiceServer(server, &MockChatService{})
	pb.RegisterGuildServer(server, &MockGuildService{})

	go func() {
		if err := server.Serve(lis); err != nil {
			// Server stopped
		}
	}()

	return server, lis
}

// bufDialer creates a dialer for bufconn
func bufDialer(listener *bufconn.Listener) func(context.Context, string) (net.Conn, error) {
	return func(ctx context.Context, url string) (net.Conn, error) {
		return listener.Dial()
	}
}

// TestChatApp_DaemonConnection tests successful daemon connection
func TestChatApp_DaemonConnection(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Create test server
	server, listener := createTestServer()
	defer server.Stop()

	// Create test app
	guildConfig := &config.GuildConfig{
		Name: "test-guild",
	}
	registry := registry.NewComponentRegistry()

	app := NewApp(ctx, guildConfig, registry)
	app.SetCampaignID("test-campaign")

	// Don't use connection manager for this test - we're simulating direct connection
	app.connManager = nil

	// Create connection to test server
	conn, err := grpc.DialContext(ctx, "bufnet",
		grpc.WithContextDialer(bufDialer(listener)),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)
	defer conn.Close()

	// Manually set up the connection (simulating successful daemon discovery)
	app.grpcConn = conn
	app.guildClient = pb.NewGuildClient(conn)
	app.chatClient = pb.NewChatServiceClient(conn)
	app.sessionClient = pb.NewSessionServiceClient(conn)
	app.connectionStatus = true
	app.connectionInfo = &daemonconn.ConnectionInfo{
		Address: "test-server",
		Type:    "tcp",
	}

	// Test that the app recognizes daemon connection
	assert.True(t, app.isConnectedToDaemon())
	assert.False(t, app.directMode)
}

// TestChatApp_DirectModeFallback tests fallback to direct mode
func TestChatApp_DirectModeFallback(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Create test app without daemon
	guildConfig := &config.GuildConfig{
		Name: "test-guild",
	}
	registry := registry.NewComponentRegistry()

	app := NewApp(ctx, guildConfig, registry)
	app.SetCampaignID("test-campaign")

	// Initialize with no daemon available (should trigger direct mode)
	app.connManager = daemonconn.NewManager(ctx)
	app.enableDirectMode()

	// Test that direct mode is enabled
	assert.True(t, app.directMode)
	assert.False(t, app.connectionStatus)
	assert.False(t, app.isConnectedToDaemon())
}

// TestChatApp_SessionPersistence tests session loading functionality
func TestChatApp_SessionPersistence(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Create test server with session data
	server, listener := createTestServer()
	defer server.Stop()

	// Create test app
	guildConfig := &config.GuildConfig{
		Name: "test-guild",
	}
	registry := registry.NewComponentRegistry()

	app := NewApp(ctx, guildConfig, registry)
	app.SetCampaignID("test-campaign")

	// Set up connection
	conn, err := grpc.DialContext(ctx, "bufnet",
		grpc.WithContextDialer(bufDialer(listener)),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)
	defer conn.Close()

	app.sessionClient = pb.NewSessionServiceClient(conn)

	// Test session creation (no existing sessions)
	err = app.loadSessionFromDaemon()
	assert.NoError(t, err)
	assert.NotEmpty(t, app.config.SessionID)
}

// TestChatApp_MessageSending tests message sending in both modes
func TestChatApp_MessageSending(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Test direct mode message sending
	t.Run("DirectMode", func(t *testing.T) {
		guildConfig := &config.GuildConfig{Name: "test-guild"}
		registry := registry.NewComponentRegistry()
		app := NewApp(ctx, guildConfig, registry)

		app.enableDirectMode()

		// Send message in direct mode
		err := app.sendMessage(ctx, "test message")
		assert.NoError(t, err)

		// Verify message was added (enableDirectMode adds 2 system messages first)
		assert.GreaterOrEqual(t, len(app.messages), 4) // 2 system messages + user message + system response
		
		// Find the user message (should be after the system messages)
		var userMsgIndex int
		for i, msg := range app.messages {
			if msg.Type == types.MsgUser {
				userMsgIndex = i
				break
			}
		}
		
		assert.Equal(t, "test message", app.messages[userMsgIndex].Content)
		assert.Equal(t, types.MsgUser, app.messages[userMsgIndex].Type)
	})

	// Test daemon mode message sending
	t.Run("DaemonMode", func(t *testing.T) {
		server, listener := createTestServer()
		defer server.Stop()

		guildConfig := &config.GuildConfig{Name: "test-guild"}
		registry := registry.NewComponentRegistry()
		app := NewApp(ctx, guildConfig, registry)
		app.SetSessionID("test-session")

		// Set up daemon connection
		conn, err := grpc.DialContext(ctx, "bufnet",
			grpc.WithContextDialer(bufDialer(listener)),
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		)
		require.NoError(t, err)
		defer conn.Close()

		app.grpcConn = conn
		app.chatClient = pb.NewChatServiceClient(conn)
		app.connectionStatus = true
		app.connectionInfo = &daemonconn.ConnectionInfo{
			Address: "test-server",
			Type:    "tcp",
		}

		// Send message via daemon (this would normally be async)
		err = app.sendMessageViaDaemon(ctx, "daemon test message")
		assert.NoError(t, err)
	})
}

// TestChatApp_Reconnection tests automatic reconnection behavior
func TestChatApp_Reconnection(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	guildConfig := &config.GuildConfig{Name: "test-guild"}
	registry := registry.NewComponentRegistry()
	app := NewApp(ctx, guildConfig, registry)

	// Start with daemon connected
	server, listener := createTestServer()

	conn, err := grpc.DialContext(ctx, "bufnet",
		grpc.WithContextDialer(bufDialer(listener)),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)

	// Set up connection without connection manager
	app.connManager = nil
	app.grpcConn = conn
	app.connectionStatus = true
	app.connectionInfo = &daemonconn.ConnectionInfo{Address: "test", Type: "tcp"}

	// Verify connected
	assert.True(t, app.isConnectedToDaemon())

	// Simulate daemon disconnect
	server.Stop()
	conn.Close()

	// Trigger direct mode fallback
	app.enableDirectMode()

	// Verify fallback to direct mode
	assert.True(t, app.directMode)
	assert.False(t, app.isConnectedToDaemon())
}

// TestChatApp_MessageConversion tests message format conversion
func TestChatApp_MessageConversion(t *testing.T) {
	guildConfig := &config.GuildConfig{Name: "test-guild"}
	registry := registry.NewComponentRegistry()
	app := NewApp(context.Background(), guildConfig, registry)

	// Test ChatMessage conversion
	chatMsg := &pb.ChatMessage{
		SessionId:  "test-session",
		SenderId:   "agent-1",
		SenderName: "Test Agent",
		Content:    "Hello world",
		Type:       pb.ChatMessage_AGENT_RESPONSE,
		Timestamp:  time.Now().Unix(),
		Metadata:   map[string]string{"test": "value"},
	}

	converted := app.convertChatMessage(chatMsg)

	assert.Equal(t, types.MsgAgent, converted.Type)
	assert.Equal(t, "Hello world", converted.Content)
	assert.Equal(t, "agent-1", converted.AgentID)
	assert.Equal(t, "value", converted.Metadata["test"])
}

// TestDaemonConnection_EnvironmentOverride tests GUILD_DAEMON_ADDR override
func TestDaemonConnection_EnvironmentOverride(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	// Test with environment override (will fail since no server running)
	t.Setenv("GUILD_DAEMON_ADDR", "localhost:9999")

	conn, info, err := daemonconn.Discover(ctx)

	// Should either fail with connection error or succeed with mock connection
	if err != nil {
		assert.Contains(t, err.Error(), "failed to connect to daemon at override address")
	} else {
		assert.NotNil(t, conn)
		assert.NotNil(t, info)
		assert.Equal(t, "localhost:9999", info.Address)
		assert.Equal(t, "tcp", info.Type)
		conn.Close()
	}
}
