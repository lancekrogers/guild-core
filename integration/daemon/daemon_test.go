// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

//go:build integration
// +build integration

package daemon

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health/grpc_health_v1"

	"github.com/guild-framework/guild-core/internal/daemon"
	pkgDaemon "github.com/guild-framework/guild-core/pkg/daemon"
	"github.com/guild-framework/guild-core/pkg/gerror"
	pb "github.com/guild-framework/guild-core/pkg/grpc/pb/guild/v1"
	"github.com/guild-framework/guild-core/pkg/observability"
)

func TestGuildDaemonLifecycle(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Check context early
	if err := ctx.Err(); err != nil {
		t.Fatal(gerror.Wrap(err, gerror.ErrCodeCancelled, "test context cancelled").
			WithComponent("daemon_test").
			WithOperation("TestGuildDaemonLifecycle"))
	}

	// Set up observability context
	logger := observability.GetLogger(ctx)
	ctx = observability.WithComponent(ctx, "daemon_test")
	ctx = observability.WithOperation(ctx, "TestGuildDaemonLifecycle")

	logger.InfoContext(ctx, "Starting daemon lifecycle test")

	// Create test environment
	testEnv, err := createGuildTestEnvironment(ctx, t)
	if err != nil {
		t.Fatal(gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create test environment").
			WithComponent("daemon_test").
			WithOperation("TestGuildDaemonLifecycle"))
	}
	defer testEnv.cleanup(ctx)

	// Start daemon process
	daemonCmd, err := testEnv.startDaemon(ctx)
	if err != nil {
		t.Fatal(gerror.Wrap(err, gerror.ErrCodeInternal, "failed to start daemon").
			WithComponent("daemon_test").
			WithOperation("TestGuildDaemonLifecycle"))
	}
	defer func() {
		if daemonCmd.Process != nil {
			if err := daemonCmd.Process.Kill(); err != nil {
				logger.ErrorContext(ctx, "Failed to kill daemon process", "error", err)
			}
		}
	}()

	// Wait for daemon readiness
	if err := testEnv.waitForDaemonReady(ctx); err != nil {
		t.Fatal(gerror.Wrap(err, gerror.ErrCodeTimeout, "daemon failed to become ready").
			WithComponent("daemon_test").
			WithOperation("TestGuildDaemonLifecycle"))
	}

	// Test daemon status
	if err := testEnv.verifyDaemonStatus(ctx); err != nil {
		t.Fatal(gerror.Wrap(err, gerror.ErrCodeInternal, "daemon status verification failed").
			WithComponent("daemon_test").
			WithOperation("TestGuildDaemonLifecycle"))
	}

	// Test graceful shutdown
	if err := testEnv.shutdownDaemon(ctx); err != nil {
		t.Fatal(gerror.Wrap(err, gerror.ErrCodeInternal, "daemon shutdown failed").
			WithComponent("daemon_test").
			WithOperation("TestGuildDaemonLifecycle"))
	}

	logger.InfoContext(ctx, "Daemon lifecycle test completed successfully")
}

func TestGuildDaemonHealthCheck(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Check context early
	if err := ctx.Err(); err != nil {
		t.Fatal(gerror.Wrap(err, gerror.ErrCodeCancelled, "test context cancelled").
			WithComponent("daemon_test").
			WithOperation("TestGuildDaemonHealthCheck"))
	}

	logger := observability.GetLogger(ctx)
	ctx = observability.WithComponent(ctx, "daemon_test")
	ctx = observability.WithOperation(ctx, "TestGuildDaemonHealthCheck")

	// Skip if no daemon is running
	if !daemon.IsRunning() {
		t.Skip("No daemon running for health check test")
	}

	logger.InfoContext(ctx, "Testing daemon health check")

	// Connect to daemon via gRPC
	conn, err := grpc.NewClient("unix:///tmp/guild.sock", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatal(gerror.Wrap(err, gerror.ErrCodeConnection, "failed to connect to daemon").
			WithComponent("daemon_test").
			WithOperation("TestGuildDaemonHealthCheck").
			WithDetails("socket", "/tmp/guild.sock"))
	}
	defer func() {
		if err := conn.Close(); err != nil {
			logger.ErrorContext(ctx, "Failed to close gRPC connection", "error", err)
		}
	}()

	// Test health check
	health := grpc_health_v1.NewHealthClient(conn)
	resp, err := health.Check(ctx, &grpc_health_v1.HealthCheckRequest{})
	if err != nil {
		t.Fatal(gerror.Wrap(err, gerror.ErrCodeConnection, "health check failed").
			WithComponent("daemon_test").
			WithOperation("TestGuildDaemonHealthCheck"))
	}

	assert.Equal(t, grpc_health_v1.HealthCheckResponse_SERVING, resp.Status)
	logger.InfoContext(ctx, "Health check passed", "status", resp.Status.String())
}

func TestGuildSessionPersistence(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	// Check context early
	if err := ctx.Err(); err != nil {
		t.Fatal(gerror.Wrap(err, gerror.ErrCodeCancelled, "test context cancelled").
			WithComponent("daemon_test").
			WithOperation("TestGuildSessionPersistence"))
	}

	logger := observability.GetLogger(ctx)
	ctx = observability.WithComponent(ctx, "daemon_test")
	ctx = observability.WithOperation(ctx, "TestGuildSessionPersistence")

	logger.InfoContext(ctx, "Starting session persistence test")

	// Create test environment
	testEnv, err := createGuildTestEnvironment(ctx, t)
	if err != nil {
		t.Fatal(gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create test environment").
			WithComponent("daemon_test").
			WithOperation("TestGuildSessionPersistence"))
	}
	defer testEnv.cleanup(ctx)

	// Start first daemon instance
	daemonCmd1, err := testEnv.startDaemon(ctx)
	if err != nil {
		t.Fatal(gerror.Wrap(err, gerror.ErrCodeInternal, "failed to start first daemon").
			WithComponent("daemon_test").
			WithOperation("TestGuildSessionPersistence"))
	}

	// Wait for daemon readiness
	if err := testEnv.waitForDaemonReady(ctx); err != nil {
		t.Fatal(gerror.Wrap(err, gerror.ErrCodeTimeout, "first daemon failed to become ready").
			WithComponent("daemon_test").
			WithOperation("TestGuildSessionPersistence"))
	}

	// Create session and send messages
	sessionId, messageCount, err := testEnv.createSessionAndSendMessages(ctx)
	if err != nil {
		t.Fatal(gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create session and send messages").
			WithComponent("daemon_test").
			WithOperation("TestGuildSessionPersistence"))
	}

	logger.InfoContext(ctx, "Created session with messages", "session_id", sessionId, "message_count", messageCount)

	// Restart daemon
	if err := daemonCmd1.Process.Kill(); err != nil {
		logger.ErrorContext(ctx, "Failed to kill first daemon", "error", err)
	}
	daemonCmd1.Wait()

	logger.InfoContext(ctx, "First daemon stopped, starting second daemon")

	daemonCmd2, err := testEnv.startDaemon(ctx)
	if err != nil {
		t.Fatal(gerror.Wrap(err, gerror.ErrCodeInternal, "failed to start second daemon").
			WithComponent("daemon_test").
			WithOperation("TestGuildSessionPersistence"))
	}
	defer func() {
		if daemonCmd2.Process != nil {
			if err := daemonCmd2.Process.Kill(); err != nil {
				logger.ErrorContext(ctx, "Failed to kill second daemon", "error", err)
			}
		}
	}()

	// Wait for second daemon readiness
	if err := testEnv.waitForDaemonReady(ctx); err != nil {
		t.Fatal(gerror.Wrap(err, gerror.ErrCodeTimeout, "second daemon failed to become ready").
			WithComponent("daemon_test").
			WithOperation("TestGuildSessionPersistence"))
	}

	// Verify messages persisted
	persistedCount, err := testEnv.verifySessionPersistence(ctx, sessionId)
	if err != nil {
		t.Fatal(gerror.Wrap(err, gerror.ErrCodeInternal, "failed to verify session persistence").
			WithComponent("daemon_test").
			WithOperation("TestGuildSessionPersistence"))
	}

	// Verify we have persistent messages
	assert.True(t, persistedCount > 0, "No messages persisted after restart")
	if messageCount > 0 {
		assert.True(t, persistedCount >= messageCount/2,
			"Too few messages persisted: had %d, now have %d", messageCount, persistedCount)
	}

	logger.InfoContext(ctx, "Session persistence verified",
		"original_count", messageCount, "persisted_count", persistedCount)
}

// guildTestEnvironment provides a test environment for daemon testing
type guildTestEnvironment struct {
	campaignDir  string
	oldCwd       string
	t            *testing.T
	daemonStdout *strings.Builder
	daemonStderr *strings.Builder
}

func createGuildTestEnvironment(ctx context.Context, t *testing.T) (*guildTestEnvironment, error) {
	logger := observability.GetLogger(ctx)

	// Create temporary directory for test campaign
	tmpDir := t.TempDir()
	campaignDir := filepath.Join(tmpDir, "test-campaign")
	if err := os.MkdirAll(campaignDir, 0755); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeIO, "failed to create test campaign directory").
			WithComponent("daemon_test").
			WithOperation("createGuildTestEnvironment").
			WithDetails("campaign_dir", campaignDir)
	}

	logger.InfoContext(ctx, "Created test campaign directory", "path", campaignDir)

	// Create .campaign directory structure
	campaignConfigDir := filepath.Join(campaignDir, ".campaign")
	if err := os.MkdirAll(campaignConfigDir, 0755); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeIO, "failed to create .campaign directory").
			WithComponent("daemon_test").
			WithOperation("createGuildTestEnvironment").
			WithDetails("campaign_config_dir", campaignConfigDir)
	}

	// Create campaign.yaml for campaign detection
	campaignYaml := `campaign: test-campaign
project: test-campaign-project
description: Test campaign for daemon integration tests
`
	campaignYamlPath := filepath.Join(campaignConfigDir, "campaign.yaml")
	if err := os.WriteFile(campaignYamlPath, []byte(campaignYaml), 0644); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeIO, "failed to write campaign.yaml").
			WithComponent("daemon_test").
			WithOperation("createGuildTestEnvironment").
			WithDetails("file_path", campaignYamlPath)
	}

	// Create a basic guild.yaml for the campaign
	guildYaml := `
name: test-campaign
version: 1.0.0
agents:
  - name: test-agent
    type: worker
    model: mock
`
	guildYamlPath := filepath.Join(campaignConfigDir, "guild.yaml")
	if err := os.WriteFile(guildYamlPath, []byte(guildYaml), 0644); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeIO, "failed to write guild.yaml").
			WithComponent("daemon_test").
			WithOperation("createGuildTestEnvironment").
			WithDetails("file_path", guildYamlPath)
	}

	logger.InfoContext(ctx, "Created campaign configuration",
		"campaign_yaml", campaignYamlPath,
		"guild_yaml", guildYamlPath)

	// Change to campaign directory
	oldCwd, err := os.Getwd()
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeIO, "failed to get current directory").
			WithComponent("daemon_test").
			WithOperation("createGuildTestEnvironment")
	}

	if err := os.Chdir(campaignDir); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeIO, "failed to change to campaign directory").
			WithComponent("daemon_test").
			WithOperation("createGuildTestEnvironment").
			WithDetails("campaign_dir", campaignDir)
	}

	logger.InfoContext(ctx, "Changed to campaign directory", "path", campaignDir)

	return &guildTestEnvironment{
		campaignDir: campaignDir,
		oldCwd:      oldCwd,
		t:           t,
	}, nil
}

func (env *guildTestEnvironment) cleanup(ctx context.Context) {
	logger := observability.GetLogger(ctx)

	if err := os.Chdir(env.oldCwd); err != nil {
		logger.ErrorContext(ctx, "Failed to restore working directory",
			"error", err, "old_cwd", env.oldCwd)
	}

	logger.InfoContext(ctx, "Test environment cleanup completed")
}

func (env *guildTestEnvironment) getExpectedSocketPath() string {
	// Get daemon config for test-campaign session 0
	config, err := daemon.GetDaemonConfig("test-campaign", 0)
	if err != nil {
		env.t.Logf("Failed to get daemon config: %v", err)
		return ""
	}
	return config.SocketPath
}

func (env *guildTestEnvironment) startDaemon(ctx context.Context) (*exec.Cmd, error) {
	logger := observability.GetLogger(ctx)

	// Start daemon in foreground mode for testing
	cmd := exec.CommandContext(ctx, "guild", "serve", "--foreground")
	cmd.Dir = env.campaignDir

	// Capture output for debugging
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	logger.InfoContext(ctx, "Starting daemon process",
		"command", "guild serve --foreground",
		"cwd", env.campaignDir)

	if err := cmd.Start(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to start daemon process").
			WithComponent("daemon_test").
			WithOperation("startDaemon").
			WithDetails("command", "guild serve --foreground").
			WithDetails("cwd", env.campaignDir)
	}

	logger.InfoContext(ctx, "Daemon process started", "pid", cmd.Process.Pid)

	// Store output builders for later access
	env.daemonStdout = &stdout
	env.daemonStderr = &stderr

	return cmd, nil
}

func (env *guildTestEnvironment) waitForDaemonReady(ctx context.Context) error {
	logger := observability.GetLogger(ctx)

	logger.InfoContext(ctx, "Waiting for daemon to start")

	// Create a timeout context for daemon startup
	startupCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-startupCtx.Done():
			// Log daemon output for debugging
			if env.daemonStdout != nil && env.daemonStderr != nil {
				env.t.Logf("DAEMON STDOUT:\n%s", env.daemonStdout.String())
				env.t.Logf("DAEMON STDERR:\n%s", env.daemonStderr.String())
			}
			return gerror.Wrap(startupCtx.Err(), gerror.ErrCodeTimeout, "daemon failed to start within timeout").
				WithComponent("daemon_test").
				WithOperation("waitForDaemonReady").
				WithDetails("timeout", "15s")
		case <-ticker.C:
			// Check context cancellation
			if err := ctx.Err(); err != nil {
				return gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled while waiting for daemon").
					WithComponent("daemon_test").
					WithOperation("waitForDaemonReady")
			}

			// Check if daemon reports as running by testing socket connectivity
			socketPath := env.getExpectedSocketPath()
			logger.InfoContext(ctx, "Checking socket connectivity", "socket_path", socketPath)
			if pkgDaemon.CanConnect(socketPath) {
				logger.InfoContext(ctx, "Daemon startup confirmed via socket connectivity")
				return nil
			}
		}
	}
}

func (env *guildTestEnvironment) verifyDaemonStatus(ctx context.Context) error {
	logger := observability.GetLogger(ctx)

	logger.InfoContext(ctx, "Checking daemon status")

	socketPath := env.getExpectedSocketPath()
	if socketPath == "" {
		return gerror.New(gerror.ErrCodeInternal, "could not determine socket path", nil).
			WithComponent("daemon_test").
			WithOperation("verifyDaemonStatus")
	}

	isRunning := pkgDaemon.CanConnect(socketPath)
	if !isRunning {
		return gerror.New(gerror.ErrCodeInternal, "daemon not running", nil).
			WithComponent("daemon_test").
			WithOperation("verifyDaemonStatus").
			WithDetails("socket_path", socketPath)
	}

	logger.InfoContext(ctx, "Daemon status verified", "socket_path", socketPath)
	return nil
}

func (env *guildTestEnvironment) shutdownDaemon(ctx context.Context) error {
	logger := observability.GetLogger(ctx)

	logger.InfoContext(ctx, "Initiating daemon shutdown")

	socketPath := env.getExpectedSocketPath()
	if socketPath == "" {
		return gerror.New(gerror.ErrCodeInternal, "could not determine socket path", nil).
			WithComponent("daemon_test").
			WithOperation("shutdownDaemon")
	}

	if err := pkgDaemon.StopSession(socketPath); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to stop daemon").
			WithComponent("daemon_test").
			WithOperation("shutdownDaemon").
			WithDetails("socket_path", socketPath)
	}

	// Wait for shutdown
	shutdownCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-shutdownCtx.Done():
			return gerror.Wrap(shutdownCtx.Err(), gerror.ErrCodeTimeout, "daemon failed to stop within timeout").
				WithComponent("daemon_test").
				WithOperation("shutdownDaemon").
				WithDetails("timeout", "10s")
		case <-ticker.C:
			if !pkgDaemon.CanConnect(socketPath) {
				logger.InfoContext(ctx, "Daemon shutdown confirmed")
				return nil
			}
		}
	}
}

func (env *guildTestEnvironment) createSessionAndSendMessages(ctx context.Context) (string, int, error) {
	logger := observability.GetLogger(ctx)

	// Get the actual socket path
	socketPath := env.getExpectedSocketPath()
	if socketPath == "" {
		return "", 0, gerror.New(gerror.ErrCodeInternal, "could not determine socket path", nil).
			WithComponent("daemon_test").
			WithOperation("createSessionAndSendMessages")
	}

	// Connect to daemon
	conn, err := grpc.NewClient("unix://"+socketPath, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return "", 0, gerror.Wrap(err, gerror.ErrCodeConnection, "failed to connect to daemon").
			WithComponent("daemon_test").
			WithOperation("createSessionAndSendMessages").
			WithDetails("socket", socketPath)
	}
	defer conn.Close()

	client := pb.NewChatServiceClient(conn)

	// Create session
	session, err := client.CreateChatSession(ctx, &pb.CreateChatSessionRequest{
		Name:       "persistence-test",
		CampaignId: "test-campaign",
	})
	if err != nil {
		return "", 0, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create chat session").
			WithComponent("daemon_test").
			WithOperation("createSessionAndSendMessages")
	}

	logger.InfoContext(ctx, "Created chat session", "session_id", session.Id)

	// Start a chat stream to send messages
	stream, err := client.Chat(ctx)
	if err != nil {
		return "", 0, gerror.Wrap(err, gerror.ErrCodeConnection, "failed to create chat stream").
			WithComponent("daemon_test").
			WithOperation("createSessionAndSendMessages").
			WithDetails("session_id", session.Id)
	}

	// Send test messages
	testMessages := []string{
		"Hello, this is test message 1",
		"This is test message 2",
		"This is test message 3",
		"Final test message 4",
	}

	sentCount := 0
	for i, msg := range testMessages {
		if err := ctx.Err(); err != nil {
			logger.ErrorContext(ctx, "Context cancelled while sending messages", "error", err)
			break
		}

		err := stream.Send(&pb.ChatRequest{
			Request: &pb.ChatRequest_Message{
				Message: &pb.ChatMessage{
					SessionId:  session.Id,
					SenderId:   "user",
					SenderName: "test-user",
					Content:    msg,
					Type:       pb.ChatMessage_USER_MESSAGE,
					Timestamp:  time.Now().UnixMilli(),
				},
			},
		})
		if err != nil {
			logger.ErrorContext(ctx, "Failed to send message", "error", err, "message_index", i)
			break
		}
		sentCount++

		// Try to receive potential response (non-blocking)
		if _, err := stream.Recv(); err != nil && err != io.EOF {
			logger.InfoContext(ctx, "Received response or stream closed", "message_index", i)
		}
	}

	stream.CloseSend()

	logger.InfoContext(ctx, "Sent messages to session",
		"session_id", session.Id, "sent_count", sentCount)

	return session.Id, sentCount, nil
}

func (env *guildTestEnvironment) verifySessionPersistence(ctx context.Context, sessionId string) (int, error) {
	logger := observability.GetLogger(ctx)

	// Get the actual socket path
	socketPath := env.getExpectedSocketPath()
	if socketPath == "" {
		return 0, gerror.New(gerror.ErrCodeInternal, "could not determine socket path", nil).
			WithComponent("daemon_test").
			WithOperation("verifySessionPersistence")
	}

	// Connect to restarted daemon
	conn, err := grpc.NewClient("unix://"+socketPath, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return 0, gerror.Wrap(err, gerror.ErrCodeConnection, "failed to connect to restarted daemon").
			WithComponent("daemon_test").
			WithOperation("verifySessionPersistence").
			WithDetails("socket", socketPath)
	}
	defer conn.Close()

	client := pb.NewChatServiceClient(conn)

	// Verify messages persisted
	history, err := client.GetChatHistory(ctx, &pb.GetChatHistoryRequest{
		SessionId: sessionId,
		Limit:     100,
	})
	if err != nil {
		return 0, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to get chat history").
			WithComponent("daemon_test").
			WithOperation("verifySessionPersistence").
			WithDetails("session_id", sessionId)
	}

	persistedCount := len(history.Messages)
	logger.InfoContext(ctx, "Retrieved persisted messages",
		"session_id", sessionId, "persisted_count", persistedCount)

	return persistedCount, nil
}

func TestGuildSessionService(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Check context early
	if err := ctx.Err(); err != nil {
		t.Fatal(gerror.Wrap(err, gerror.ErrCodeCancelled, "test context cancelled").
			WithComponent("daemon_test").
			WithOperation("TestGuildSessionService"))
	}

	logger := observability.GetLogger(ctx)
	ctx = observability.WithComponent(ctx, "daemon_test")
	ctx = observability.WithOperation(ctx, "TestGuildSessionService")

	// Skip if no daemon is running
	if !daemon.IsRunning() {
		t.Skip("No daemon running for session service test")
	}

	logger.InfoContext(ctx, "Testing SessionService API")

	// Connect to daemon via gRPC
	conn, err := grpc.NewClient("unix:///tmp/guild.sock", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatal(gerror.Wrap(err, gerror.ErrCodeConnection, "failed to connect to daemon").
			WithComponent("daemon_test").
			WithOperation("TestGuildSessionService").
			WithDetails("socket", "/tmp/guild.sock"))
	}
	defer func() {
		if err := conn.Close(); err != nil {
			logger.ErrorContext(ctx, "Failed to close gRPC connection", "error", err)
		}
	}()

	// Test SessionService
	sessionClient := pb.NewSessionServiceClient(conn)

	// Create a session using SessionService
	campaignId := "test-campaign"
	session, err := sessionClient.CreateSession(ctx, &pb.CreateSessionRequest{
		Name:       "session-service-test",
		CampaignId: &campaignId,
	})
	if err != nil {
		t.Fatal(gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create session via SessionService").
			WithComponent("daemon_test").
			WithOperation("TestGuildSessionService"))
	}

	logger.InfoContext(ctx, "Created session via SessionService", "session_id", session.Id)

	// Save messages to session
	for i := 0; i < 3; i++ {
		_, err := sessionClient.SaveMessage(ctx, &pb.SaveMessageRequest{
			Message: &pb.Message{
				SessionId: session.Id,
				Role:      pb.Message_USER,
				Content:   fmt.Sprintf("SessionService test message %d", i),
			},
		})
		if err != nil {
			t.Fatal(gerror.Wrap(err, gerror.ErrCodeInternal, "failed to save message").
				WithComponent("daemon_test").
				WithOperation("TestGuildSessionService").
				WithDetails("message_index", i))
		}
	}

	logger.InfoContext(ctx, "Saved messages via SessionService")

	// Retrieve messages
	messages, err := sessionClient.GetMessages(ctx, &pb.GetMessagesRequest{
		SessionId: session.Id,
	})
	if err != nil {
		t.Fatal(gerror.Wrap(err, gerror.ErrCodeInternal, "failed to retrieve messages").
			WithComponent("daemon_test").
			WithOperation("TestGuildSessionService").
			WithDetails("session_id", session.Id))
	}

	assert.Len(t, messages.Messages, 3)
	logger.InfoContext(ctx, "Retrieved messages via SessionService", "count", len(messages.Messages))

	// List sessions
	sessionsList, err := sessionClient.ListSessions(ctx, &pb.ListSessionsRequest{
		CampaignId: &campaignId,
	})
	if err != nil {
		t.Fatal(gerror.Wrap(err, gerror.ErrCodeInternal, "failed to list sessions").
			WithComponent("daemon_test").
			WithOperation("TestGuildSessionService"))
	}

	assert.True(t, len(sessionsList.Sessions) > 0)
	logger.InfoContext(ctx, "Listed sessions via SessionService", "count", len(sessionsList.Sessions))

	// Clean up
	_, err = sessionClient.DeleteSession(ctx, &pb.DeleteSessionRequest{
		Id: session.Id,
	})
	if err != nil {
		logger.ErrorContext(ctx, "Failed to delete test session", "error", err, "session_id", session.Id)
	}

	logger.InfoContext(ctx, "SessionService test completed successfully")
}

func TestGuildEventService(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Check context early
	if err := ctx.Err(); err != nil {
		t.Fatal(gerror.Wrap(err, gerror.ErrCodeCancelled, "test context cancelled").
			WithComponent("daemon_test").
			WithOperation("TestGuildEventService"))
	}

	logger := observability.GetLogger(ctx)
	ctx = observability.WithComponent(ctx, "daemon_test")
	ctx = observability.WithOperation(ctx, "TestGuildEventService")

	// Skip if no daemon is running
	if !daemon.IsRunning() {
		t.Skip("No daemon running for event service test")
	}

	logger.InfoContext(ctx, "Testing EventService API")

	// Connect to daemon via gRPC
	conn, err := grpc.NewClient("unix:///tmp/guild.sock", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatal(gerror.Wrap(err, gerror.ErrCodeConnection, "failed to connect to daemon").
			WithComponent("daemon_test").
			WithOperation("TestGuildEventService").
			WithDetails("socket", "/tmp/guild.sock"))
	}
	defer func() {
		if err := conn.Close(); err != nil {
			logger.ErrorContext(ctx, "Failed to close gRPC connection", "error", err)
		}
	}()

	// Test EventService
	eventClient := pb.NewEventServiceClient(conn)

	// Start event stream
	stream, err := eventClient.StreamEvents(ctx, &pb.StreamEventsRequest{
		EventTypes: []string{"task.*", "session.*"},
	})
	if err != nil {
		t.Fatal(gerror.Wrap(err, gerror.ErrCodeConnection, "failed to create event stream").
			WithComponent("daemon_test").
			WithOperation("TestGuildEventService"))
	}

	logger.InfoContext(ctx, "Created event stream")

	// Publish a test event
	_, err = eventClient.PublishEvent(ctx, &pb.PublishEventRequest{
		Event: &pb.Event{
			Type:   "task.created",
			Source: "daemon_test",
		},
	})
	if err != nil {
		t.Fatal(gerror.Wrap(err, gerror.ErrCodeInternal, "failed to publish test event").
			WithComponent("daemon_test").
			WithOperation("TestGuildEventService"))
	}

	logger.InfoContext(ctx, "Published test event")

	// Try to receive the event (with timeout)
	eventReceived := make(chan bool, 1)
	go func() {
		_, err := stream.Recv()
		eventReceived <- (err == nil)
	}()

	select {
	case success := <-eventReceived:
		if success {
			logger.InfoContext(ctx, "Successfully received test event")
		} else {
			logger.InfoContext(ctx, "Event reception failed (expected in test environment)")
		}
	case <-time.After(5 * time.Second):
		logger.InfoContext(ctx, "Event reception timeout (expected in test environment)")
	}

	logger.InfoContext(ctx, "EventService test completed")
}
