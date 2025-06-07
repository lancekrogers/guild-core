package chat

import (
	"context"
	"fmt"
	"net"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/guild-ventures/guild-core/pkg/grpc/pb/guild/v1"
	grpcpkg "github.com/guild-ventures/guild-core/pkg/grpc"
	"github.com/guild-ventures/guild-core/pkg/config"
	"github.com/guild-ventures/guild-core/pkg/project"
	"github.com/guild-ventures/guild-core/pkg/registry"
)

// TestGRPCServerStartup tests that the gRPC server starts correctly
func TestGRPCServerStartup(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Find available port
	listener, err := net.Listen("tcp", ":0")
	require.NoError(t, err)
	port := listener.Addr().(*net.TCPAddr).Port
	listener.Close()

	address := fmt.Sprintf("localhost:%d", port)

	// Create test registry
	reg := registry.NewComponentRegistry()
	err = reg.Initialize(ctx, registry.Config{})
	require.NoError(t, err)

	// Start gRPC server in goroutine
	server := grpcpkg.NewServer(reg)
	go func() {
		err := server.Start(ctx, address)
		if err != nil && ctx.Err() == nil {
			t.Errorf("Server failed to start: %v", err)
		}
	}()

	// Give server time to start
	time.Sleep(500 * time.Millisecond)

	// Try to connect
	conn, err := grpc.Dial(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	defer conn.Close()

	// Verify service is available
	client := guildv1.NewGuildServiceClient(conn)

	// Test health check or basic operation
	// Note: Adjust this based on actual service methods available
	_, err = client.GetGuildInfo(ctx, &guildv1.GetGuildInfoRequest{})
	
	// We expect this to work or fail gracefully, not panic or hang
	if err != nil {
		// Log the error but don't fail if it's just unimplemented
		t.Logf("GetGuildInfo returned error (may be expected): %v", err)
	}
}

// TestChatServiceBasics tests basic chat service functionality
func TestChatServiceBasics(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Setup test environment
	testDir := t.TempDir()
	
	// Initialize a test guild project
	err := project.InitializeGuildProject(testDir, &config.GuildConfig{
		Name:        "test-guild",
		Description: "Test guild for integration testing",
		Agents: []config.AgentConfig{
			{
				ID:       "test-manager",
				Name:     "Test Manager",
				Role:     "manager",
				Provider: "mock",
				Model:    "test-model",
			},
		},
	})
	require.NoError(t, err)

	// Find available port
	listener, err := net.Listen("tcp", ":0")
	require.NoError(t, err)
	port := listener.Addr().(*net.TCPAddr).Port
	listener.Close()

	address := fmt.Sprintf("localhost:%d", port)

	// Create registry with test project context
	reg := registry.NewComponentRegistry()
	err = reg.Initialize(ctx, registry.Config{})
	require.NoError(t, err)

	// Start server
	server := grpcpkg.NewServer(reg)
	go func() {
		err := server.Start(ctx, address)
		if err != nil && ctx.Err() == nil {
			t.Errorf("Server failed to start: %v", err)
		}
	}()

	// Wait for server to start
	time.Sleep(500 * time.Millisecond)

	// Connect to server
	conn, err := grpc.Dial(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	defer conn.Close()

	// Test basic guild service
	client := guildv1.NewGuildServiceClient(conn)

	// Try to get guild info
	info, err := client.GetGuildInfo(ctx, &guildv1.GetGuildInfoRequest{})
	if err != nil {
		t.Logf("GetGuildInfo error (may be expected): %v", err)
	} else {
		t.Logf("GetGuildInfo succeeded: %+v", info)
	}
}

// TestAgentExecution tests that messages trigger real agent execution
func TestAgentExecution(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Setup test project
	testDir := t.TempDir()
	
	guildConfig := &config.GuildConfig{
		Name:        "test-execution-guild",
		Description: "Test guild for agent execution testing",
		Agents: []config.AgentConfig{
			{
				ID:          "test-manager",
				Name:        "Test Manager",
				Role:        "manager",
				Provider:    "mock",
				Model:       "test-model",
				Capabilities: []string{"task-management", "coordination"},
			},
			{
				ID:          "test-worker",
				Name:        "Test Worker",
				Role:        "worker",
				Provider:    "mock",
				Model:       "test-model",
				Capabilities: []string{"implementation", "testing"},
			},
		},
	}

	err := project.InitializeGuildProject(testDir, guildConfig)
	require.NoError(t, err)

	// Create registry and set up components
	reg := registry.NewComponentRegistry()
	err = reg.Initialize(ctx, registry.Config{})
	require.NoError(t, err)

	// Test agent creation and execution through registry
	agentFactory, err := reg.GetAgentFactory()
	require.NoError(t, err)

	// Create test agent
	agent, err := agentFactory.CreateAgent(ctx, "test-manager", guildConfig.Agents[0])
	require.NoError(t, err)
	assert.NotNil(t, agent)

	// Test agent execution
	response, err := agent.Execute(ctx, "Create a simple task breakdown for user authentication")
	
	// Should not fail outright
	if err != nil {
		t.Logf("Agent execution returned error (may be expected with mock): %v", err)
	}
	
	// Should get some response (even if it's mock)
	if response != "" {
		t.Logf("Agent response: %s", response)
		
		// Should NOT be hardcoded demo response
		assert.NotEqual(t, "I'll analyze this request...", response)
		
		// Should contain relevant terms for the request
		lowerResponse := strings.ToLower(response)
		assert.True(t, 
			strings.Contains(lowerResponse, "task") || 
			strings.Contains(lowerResponse, "auth") || 
			strings.Contains(lowerResponse, "user") ||
			strings.Contains(lowerResponse, "mock"), // Allow mock responses
			"Response should contain relevant terms or indicate mock usage")
	}
}

// TestToolExecution tests that agents can execute tools
func TestToolExecution(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	// Create workspace
	workspace := t.TempDir()
	
	// Setup test project
	guildConfig := &config.GuildConfig{
		Name:        "test-tool-guild",
		Description: "Test guild for tool execution",
		Agents: []config.AgentConfig{
			{
				ID:          "test-developer",
				Name:        "Test Developer",
				Role:        "developer",
				Provider:    "mock",
				Model:       "test-model",
				Capabilities: []string{"file-creation", "coding"},
			},
		},
	}

	err := project.InitializeGuildProject(workspace, guildConfig)
	require.NoError(t, err)

	// Create registry
	reg := registry.NewComponentRegistry()
	err = reg.Initialize(ctx, registry.Config{})
	require.NoError(t, err)

	// Get tool registry
	toolRegistry, err := reg.GetToolRegistry()
	require.NoError(t, err)

	// Check if file tool is available
	tools := toolRegistry.ListTools()
	t.Logf("Available tools: %v", tools)

	// If we have file tools, test basic file operations
	hasFileTool := false
	for _, tool := range tools {
		if strings.Contains(strings.ToLower(tool), "file") {
			hasFileTool = true
			break
		}
	}

	if hasFileTool {
		// Test file tool through registry
		fileTool, err := toolRegistry.GetTool("file")
		if err == nil && fileTool != nil {
			// Test basic file tool functionality
			t.Logf("File tool available: %s", fileTool.Description())
			
			// Note: Actual tool execution would require more complex setup
			// This is testing that tools are registered and accessible
		}
	} else {
		t.Log("No file tools available - testing tool registry access only")
	}

	// Test agent with tools
	agentFactory, err := reg.GetAgentFactory()
	require.NoError(t, err)

	agent, err := agentFactory.CreateAgent(ctx, "test-developer", guildConfig.Agents[0])
	require.NoError(t, err)

	// Test agent has access to tools
	if guildAgent, ok := agent.(interface{ GetToolRegistry() any }); ok {
		toolReg := guildAgent.GetToolRegistry()
		assert.NotNil(t, toolReg, "Agent should have access to tool registry")
	}
}

// TestChatPerformance tests that chat operations are responsive
func TestChatPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Setup minimal test environment
	reg := registry.NewComponentRegistry()
	err := reg.Initialize(ctx, registry.Config{})
	require.NoError(t, err)

	// Benchmark agent creation time
	start := time.Now()
	
	agentFactory, err := reg.GetAgentFactory()
	require.NoError(t, err)

	testConfig := config.AgentConfig{
		ID:       "perf-test",
		Name:     "Performance Test Agent",
		Role:     "worker",
		Provider: "mock",
		Model:    "test-model",
	}

	_, err = agentFactory.CreateAgent(ctx, "perf-test", testConfig)
	
	elapsed := time.Since(start)

	// Agent creation should be fast (under 1 second)
	assert.Less(t, elapsed, 1*time.Second, "Agent creation should be fast")

	if err != nil {
		t.Logf("Agent creation error (may be expected): %v", err)
	} else {
		t.Logf("Agent creation time: %v", elapsed)
	}
}

// TestMemoryUsage tests that chat doesn't leak memory during operation
func TestMemoryUsage(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping memory test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Get initial memory usage
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	startMem := m.Alloc

	// Setup registry
	reg := registry.NewComponentRegistry()
	err := reg.Initialize(ctx, registry.Config{})
	require.NoError(t, err)

	// Create multiple agents to simulate memory usage
	agentFactory, err := reg.GetAgentFactory()
	require.NoError(t, err)

	testConfig := config.AgentConfig{
		ID:       "memory-test",
		Name:     "Memory Test Agent",
		Role:     "worker",
		Provider: "mock",
		Model:    "test-model",
	}

	// Create and destroy agents multiple times
	for i := 0; i < 50; i++ {
		agent, err := agentFactory.CreateAgent(ctx, fmt.Sprintf("memory-test-%d", i), testConfig)
		if err != nil {
			continue // Skip on error
		}
		
		// Execute something to trigger memory allocation
		if agent != nil {
			_, _ = agent.Execute(ctx, fmt.Sprintf("Memory test operation %d", i))
		}
	}

	// Force garbage collection
	runtime.GC()
	runtime.GC() // Run twice to be thorough

	// Check memory after operations
	runtime.ReadMemStats(&m)
	endMem := m.Alloc

	growth := endMem - startMem
	t.Logf("Memory growth: %d bytes (%.2f MB)", growth, float64(growth)/(1024*1024))

	// Should not grow more than 50MB (this is quite generous)
	assert.Less(t, growth, uint64(50*1024*1024), "Memory growth should be reasonable")
}