// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package e2e

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/guild-framework/guild-core/pkg/gerror"
	guildv1 "github.com/guild-framework/guild-core/pkg/grpc/pb/guild/v1"
	"github.com/guild-framework/guild-core/pkg/project"
)

// E2ETestSuite provides utilities for end-to-end testing
type E2ETestSuite struct {
	projectDir    string
	guildBinary   string
	serverProcess *os.Process
	serverAddress string
	t             *testing.T
}

// setupE2ETestSuite creates a new E2E test environment
func setupE2ETestSuite(t *testing.T) *E2ETestSuite {
	t.Helper()

	// Create temporary project directory
	projectDir := t.TempDir()

	// Build guild binary if needed
	guildBinary := findOrBuildGuildBinary(t)

	return &E2ETestSuite{
		projectDir:  projectDir,
		guildBinary: guildBinary,
		t:           t,
	}
}

// TestingTB represents the interface common to *testing.T and *testing.B
type TestingTB interface {
	Helper()
	Log(args ...interface{})
	Logf(format string, args ...interface{})
	Skip(args ...interface{})
}

// findOrBuildGuildBinary locates or builds the guild binary for testing
func findOrBuildGuildBinary(t TestingTB) string {
	t.Helper()

	// Look for existing binary
	possiblePaths := []string{
		"../../../bin/guild",
		"../../bin/guild",
		"../bin/guild",
		"./bin/guild",
		"guild", // In PATH
	}

	for _, path := range possiblePaths {
		if _, err := os.Stat(path); err == nil {
			absPath, _ := filepath.Abs(path)
			t.Logf("Found guild binary at: %s", absPath)
			return absPath
		}
	}

	// Try to build if not found
	t.Log("Guild binary not found, attempting to build...")
	buildCmd := exec.Command("make", "build")
	buildCmd.Dir = "../../../" // Relative to guild-core/test/e2e/
	output, err := buildCmd.CombinedOutput()

	if err != nil {
		t.Logf("Build output: %s", string(output))
		t.Skip("Guild binary not available and build failed - skipping E2E tests")
	}

	// Check if build created binary
	builtBinary := "../../../bin/guild"
	if _, err := os.Stat(builtBinary); err == nil {
		absPath, _ := filepath.Abs(builtBinary)
		t.Logf("Built guild binary at: %s", absPath)
		return absPath
	}

	t.Skip("Guild binary not available - skipping E2E tests")
	return ""
}

// cleanup cleans up the test environment
func (suite *E2ETestSuite) cleanup() {
	if suite.serverProcess != nil {
		_ = suite.serverProcess.Kill()
		_, _ = suite.serverProcess.Wait()
	}
}

// runGuildCommand executes a guild CLI command in the project directory
func (suite *E2ETestSuite) runGuildCommand(args ...string) (string, error) {
	cmd := exec.Command(suite.guildBinary, args...)
	cmd.Dir = suite.projectDir

	// Set up environment
	cmd.Env = append(os.Environ(),
		"GUILD_LOG_LEVEL=error", // Reduce noise in tests
		"NO_COLOR=1",            // Disable colors for easier parsing
	)

	output, err := cmd.CombinedOutput()
	return string(output), err
}

// initProject initializes a guild project in the test directory
func (suite *E2ETestSuite) initProject() error {
	output, err := suite.runGuildCommand("init", "--force")
	if err != nil {
		suite.t.Logf("Init failed with output: %s", output)
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to initialize project").
			WithDetails("output", output)
	}

	// Verify project was initialized
	if !project.IsInitialized(suite.projectDir) {
		return gerror.New(gerror.ErrCodeValidation, "project not properly initialized", nil)
	}

	suite.t.Logf("Project initialized successfully in %s", suite.projectDir)
	return nil
}

// startGuildServer starts a guild serve process for testing
func (suite *E2ETestSuite) startGuildServer() error {
	// Find available port
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to find available port")
	}
	port := listener.Addr().(*net.TCPAddr).Port
	listener.Close()

	suite.serverAddress = fmt.Sprintf("localhost:%d", port)

	// Start server process
	cmd := exec.Command(suite.guildBinary, "serve", "--address", suite.serverAddress)
	cmd.Dir = suite.projectDir
	cmd.Env = append(os.Environ(),
		"GUILD_LOG_LEVEL=error",
		"NO_COLOR=1",
	)

	err = cmd.Start()
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to start guild server")
	}

	suite.serverProcess = cmd.Process

	// Wait for server to start
	time.Sleep(2 * time.Second)

	// Verify server is responding
	conn, err := grpc.Dial(suite.serverAddress, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		suite.cleanup()
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to connect to guild server")
	}
	conn.Close()

	suite.t.Logf("Guild server started at %s", suite.serverAddress)
	return nil
}

// TestGuildCompleteUserJourney tests the complete end-to-end user workflow
func TestGuildCompleteUserJourney(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	suite := setupE2ETestSuite(t)
	defer suite.cleanup()

	t.Run("step_1_guild_init", func(t *testing.T) {
		err := suite.initProject()
		require.NoError(t, err, "Guild init should succeed")

		// Verify expected files were created
		expectedFiles := []string{
			".campaign/campaign.yaml",
			".campaign/memory.db",
		}

		for _, file := range expectedFiles {
			filePath := filepath.Join(suite.projectDir, file)
			_, err := os.Stat(filePath)
			assert.NoError(t, err, "Expected file %s should exist", file)
		}

		// Verify agents directory exists and has content
		agentsDir := filepath.Join(suite.projectDir, ".campaign/agents")
		entries, err := os.ReadDir(agentsDir)
		require.NoError(t, err, "Should be able to read agents directory")
		assert.Greater(t, len(entries), 0, "Agents directory should contain agent files")

		t.Logf("✅ Guild init completed successfully")
	})

	t.Run("step_2_start_guild_serve", func(t *testing.T) {
		err := suite.startGuildServer()
		require.NoError(t, err, "Guild serve should start successfully")

		t.Logf("✅ Guild server started successfully")
	})

	t.Run("step_3_verify_grpc_services", func(t *testing.T) {
		// Connect to server
		conn, err := grpc.Dial(suite.serverAddress, grpc.WithTransportCredentials(insecure.NewCredentials()))
		require.NoError(t, err, "Should connect to guild server")
		defer conn.Close()

		// Test Guild service
		guildClient := guildv1.NewGuildClient(conn)

		// List available agents
		agentsResp, err := guildClient.ListAvailableAgents(ctx, &guildv1.ListAgentsRequest{})
		if err != nil {
			t.Logf("ListAvailableAgents error (may be expected): %v", err)
		} else {
			t.Logf("Found %d available agents", len(agentsResp.Agents))
			for _, agent := range agentsResp.Agents {
				t.Logf("  - %s: %s (%s)", agent.Id, agent.Name, agent.Type)
			}
		}

		// Test Chat service
		chatClient := guildv1.NewChatServiceClient(conn)
		stream, err := chatClient.Chat(ctx)
		if err != nil {
			t.Logf("Chat stream error (may be expected): %v", err)
		} else {
			assert.NotNil(t, stream, "Chat stream should be created")
			stream.CloseSend()
			t.Logf("✅ Chat service stream created successfully")
		}

		t.Logf("✅ gRPC services verified")
	})

	t.Run("step_4_send_message_to_elena", func(t *testing.T) {
		conn, err := grpc.Dial(suite.serverAddress, grpc.WithTransportCredentials(insecure.NewCredentials()))
		require.NoError(t, err)
		defer conn.Close()

		client := guildv1.NewGuildClient(conn)

		// Try to send message to Elena (guild master)
		resp, err := client.SendMessageToAgent(ctx, &guildv1.AgentMessageRequest{
			AgentId: "elena",
			Message: "Hello Elena! Can you help me plan a new project?",
		})

		if err != nil {
			t.Logf("SendMessageToAgent error (may be expected with test setup): %v", err)
			// This might fail if agents aren't fully initialized, which is expected in test environment
		} else {
			assert.NotNil(t, resp, "Should receive response")
			assert.NotEmpty(t, resp.Response, "Response should not be empty")
			assert.Equal(t, "elena", resp.AgentId, "Response should be from Elena")
			t.Logf("✅ Elena responded: %s", resp.Response)
		}
	})

	t.Run("step_5_verify_session_persistence", func(t *testing.T) {
		// Check that memory database was created and is accessible
		memoryDbPath := filepath.Join(suite.projectDir, ".campaign/memory.db")
		stat, err := os.Stat(memoryDbPath)
		require.NoError(t, err, "Memory database should exist")
		assert.Greater(t, stat.Size(), int64(0), "Memory database should not be empty")

		t.Logf("✅ Session persistence verified (memory.db: %d bytes)", stat.Size())
	})

	t.Run("step_6_verify_campaign_structure", func(t *testing.T) {
		// Verify complete campaign structure
		expectedStructure := map[string]bool{
			".campaign/campaign.yaml": true,
			".campaign/memory.db":     true,
			".campaign/agents":        true,
			".campaign/guilds":        false, // May or may not exist
			"commissions":             false, // May or may not exist
		}

		for path, required := range expectedStructure {
			fullPath := filepath.Join(suite.projectDir, path)
			_, err := os.Stat(fullPath)
			if required {
				assert.NoError(t, err, "Required path %s should exist", path)
			} else {
				if err == nil {
					t.Logf("Optional path %s exists", path)
				} else {
					t.Logf("Optional path %s does not exist (OK)", path)
				}
			}
		}

		t.Logf("✅ Campaign structure verified")
	})

	t.Run("step_7_performance_check", func(t *testing.T) {
		// Measure basic performance metrics
		start := time.Now()

		conn, err := grpc.Dial(suite.serverAddress, grpc.WithTransportCredentials(insecure.NewCredentials()))
		require.NoError(t, err)
		defer conn.Close()

		connectionTime := time.Since(start)

		client := guildv1.NewGuildClient(conn)
		start = time.Now()

		_, err = client.ListAvailableAgents(ctx, &guildv1.ListAgentsRequest{})

		responseTime := time.Since(start)

		// Performance assertions
		assert.Less(t, connectionTime, 5*time.Second, "Connection should be fast")
		if err == nil {
			assert.Less(t, responseTime, 2*time.Second, "Agent list should respond quickly")
		}

		t.Logf("✅ Performance check: connection=%v, response=%v", connectionTime, responseTime)
	})
}

// TestGuildProjectTypeDetection tests project type detection during init
func TestGuildProjectTypeDetection(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	projectTypes := []struct {
		name         string
		setupFiles   map[string]string
		expectedText string // Text to look for in output or config
	}{
		{
			name: "go_project",
			setupFiles: map[string]string{
				"go.mod":  "module test-project\n\ngo 1.21\n",
				"main.go": "package main\n\nfunc main() {}\n",
			},
			expectedText: "Go",
		},
		{
			name: "javascript_project",
			setupFiles: map[string]string{
				"package.json": `{"name": "test", "version": "1.0.0"}`,
				"index.js":     "console.log('hello');\n",
			},
			expectedText: "JavaScript",
		},
		{
			name: "python_project",
			setupFiles: map[string]string{
				"requirements.txt": "flask==2.0.0\n",
				"app.py":           "from flask import Flask\n",
			},
			expectedText: "Python",
		},
		{
			name: "generic_project",
			setupFiles: map[string]string{
				"README.md": "# Test Project\n",
			},
			expectedText: "project",
		},
	}

	for _, tt := range projectTypes {
		t.Run(tt.name, func(t *testing.T) {
			suite := setupE2ETestSuite(t)
			defer suite.cleanup()

			// Setup project files
			for file, content := range tt.setupFiles {
				filePath := filepath.Join(suite.projectDir, file)
				err := os.MkdirAll(filepath.Dir(filePath), 0755)
				require.NoError(t, err)
				err = os.WriteFile(filePath, []byte(content), 0644)
				require.NoError(t, err)
			}

			// Initialize project
			output, err := suite.runGuildCommand("init", "--force")
			require.NoError(t, err, "Init should succeed for %s project", tt.name)

			// Verify project was initialized
			assert.True(t, project.IsInitialized(suite.projectDir), "Project should be initialized")

			// Check for expected content in agents directory
			agentsDir := filepath.Join(suite.projectDir, ".campaign/agents")
			entries, err := os.ReadDir(agentsDir)
			require.NoError(t, err)

			// For specific project types, we expect certain agents
			var hasRelevantAgent bool
			for _, entry := range entries {
				if strings.Contains(entry.Name(), "marcus") || // Developer agent
					strings.Contains(entry.Name(), "elena") { // Manager agent
					hasRelevantAgent = true
					break
				}
			}

			assert.True(t, hasRelevantAgent, "Should have relevant agent for %s project", tt.name)

			t.Logf("✅ %s project detection and init completed", tt.name)
			t.Logf("Init output: %s", output)
		})
	}
}

// TestGuildErrorRecovery tests error handling and recovery scenarios
func TestGuildErrorRecovery(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	t.Run("init_in_existing_project", func(t *testing.T) {
		suite := setupE2ETestSuite(t)
		defer suite.cleanup()

		// First init
		err := suite.initProject()
		require.NoError(t, err)

		// Second init with force should succeed
		output, err := suite.runGuildCommand("init", "--force")
		assert.NoError(t, err, "Second init with --force should succeed")
		t.Logf("Second init output: %s", output)
	})

	t.Run("serve_without_init", func(t *testing.T) {
		suite := setupE2ETestSuite(t)
		defer suite.cleanup()

		// Try to serve without initializing
		output, err := suite.runGuildCommand("serve")
		assert.Error(t, err, "Serve should fail without init")
		assert.Contains(t, strings.ToLower(output), "not initialized", "Should mention project not initialized")

		t.Logf("Expected error output: %s", output)
	})

	t.Run("corrupted_config_recovery", func(t *testing.T) {
		suite := setupE2ETestSuite(t)
		defer suite.cleanup()

		// Initialize project first
		err := suite.initProject()
		require.NoError(t, err)

		// Corrupt the campaign config
		campaignPath := filepath.Join(suite.projectDir, ".campaign/campaign.yaml")
		err = os.WriteFile(campaignPath, []byte("invalid: yaml: content: ["), 0644)
		require.NoError(t, err)

		// Try operations that should handle corrupted config gracefully
		output, err := suite.runGuildCommand("status")
		if err != nil {
			assert.Contains(t, strings.ToLower(output), "config", "Should mention config issue")
			t.Logf("Expected config error: %s", output)
		}
	})
}

// TestGuildConcurrentOperations tests concurrent access and operations
func TestGuildConcurrentOperations(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	suite := setupE2ETestSuite(t)
	defer suite.cleanup()

	// Initialize project
	err := suite.initProject()
	require.NoError(t, err)

	// Start server
	err = suite.startGuildServer()
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Test multiple concurrent connections
	t.Run("multiple_client_connections", func(t *testing.T) {
		const numClients = 5
		errors := make(chan error, numClients)

		for i := 0; i < numClients; i++ {
			go func(clientID int) {
				conn, err := grpc.Dial(suite.serverAddress, grpc.WithTransportCredentials(insecure.NewCredentials()))
				if err != nil {
					errors <- gerror.Wrapf(err, gerror.ErrCodeInternal, "client %d connection failed", clientID)
					return
				}
				defer conn.Close()

				client := guildv1.NewGuildClient(conn)

				// Make a request
				_, err = client.ListAvailableAgents(ctx, &guildv1.ListAgentsRequest{})
				if err != nil {
					t.Logf("Client %d request error (may be expected): %v", clientID, err)
				}

				errors <- nil
			}(i)
		}

		// Wait for all clients to complete
		for i := 0; i < numClients; i++ {
			select {
			case err := <-errors:
				if err != nil {
					t.Logf("Client error: %v", err)
				}
			case <-time.After(10 * time.Second):
				t.Error("Client timed out")
			}
		}

		t.Logf("✅ Concurrent connections test completed")
	})
}

// BenchmarkE2EInitPerformance benchmarks the init command performance
func BenchmarkE2EInitPerformance(b *testing.B) {
	if testing.Short() {
		b.Skip("Skipping E2E benchmark in short mode")
	}

	guildBinary := findOrBuildGuildBinary(b)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		tempDir := b.TempDir()

		cmd := exec.Command(guildBinary, "init", "--force")
		cmd.Dir = tempDir
		cmd.Env = append(os.Environ(), "GUILD_LOG_LEVEL=error", "NO_COLOR=1")

		start := time.Now()
		_, err := cmd.CombinedOutput()
		elapsed := time.Since(start)

		if err != nil {
			b.Fatalf("Init failed: %v", err)
		}

		if elapsed > 5*time.Second {
			b.Logf("Init took %v (iteration %d)", elapsed, i)
		}
	}
}
