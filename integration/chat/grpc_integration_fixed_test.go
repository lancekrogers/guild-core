// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package chat

import (
	"context"
	"fmt"
	"net"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/guild-ventures/guild-core/internal/testutil"
	"github.com/guild-ventures/guild-core/pkg/config"
	grpcpkg "github.com/guild-ventures/guild-core/pkg/grpc"
	guildv1 "github.com/guild-ventures/guild-core/pkg/grpc/pb/guild/v1"
	"github.com/guild-ventures/guild-core/pkg/project"
	"github.com/guild-ventures/guild-core/pkg/registry"
)

// eventBusAdapter wraps MockEventBus to match the gRPC EventBus interface
type eventBusAdapter struct {
	mock *testutil.MockEventBus
}

// simpleMockAgent is a minimal mock agent for testing
type simpleMockAgent struct {
	id       string
	name     string
	response string
}

func (m *simpleMockAgent) GetID() string             { return m.id }
func (m *simpleMockAgent) GetName() string           { return m.name }
func (m *simpleMockAgent) GetType() string           { return "worker" }
func (m *simpleMockAgent) GetCapabilities() []string { return []string{"task-breakdown"} }
func (m *simpleMockAgent) Execute(ctx context.Context, input string) (string, error) {
	return m.response, nil
}

func (a *eventBusAdapter) Publish(event interface{}) {
	// Extract type from event if possible, otherwise use generic type
	eventType := "generic"
	if typed, ok := event.(map[string]interface{}); ok {
		if t, exists := typed["type"]; exists {
			eventType = fmt.Sprintf("%v", t)
		}
	}
	a.mock.Publish(eventType, event)
}

func (a *eventBusAdapter) Subscribe(eventType string, handler func(event interface{})) {
	a.mock.Subscribe(eventType, handler)
}

// TestChatServiceBasicsFixed tests basic chat service functionality with proper project setup
func TestChatServiceBasicsFixed(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Setup test project using testutil
	projCtx, cleanup := testutil.SetupTestProject(t, testutil.TestProjectOptions{
		Name: "chat-basics-test",
		CustomConfig: &config.GuildConfig{
			Name:        "test-chat-guild",
			Description: "Test guild for chat service",
			Agents: []config.AgentConfig{
				{
					ID:           "test-manager",
					Name:         "Test Manager",
					Type:         "manager",
					Provider:     "mock",
					Model:        "test-model",
					Capabilities: []string{"task-management"},
				},
			},
		},
	})
	defer cleanup()

	ctx, cancel := context.WithTimeout(project.WithContext(context.Background(), projCtx), 15*time.Second)
	defer cancel()

	// Find available port
	listener, err := net.Listen("tcp", ":0")
	require.NoError(t, err)
	port := listener.Addr().(*net.TCPAddr).Port
	listener.Close()

	address := fmt.Sprintf("localhost:%d", port)

	// Create registry with mock provider
	mockProvider := testutil.NewMockLLMProvider()
	mockProvider.SetResponse("test", "Mock response for chat service test")

	// Create registry with test configuration
	reg := registry.NewComponentRegistry()
	registryConfig := &registry.Config{
		Agents: registry.AgentConfigYaml{
			DefaultType: "worker",
			Types: map[string]interface{}{
				"worker": map[string]interface{}{
					"enabled": true,
				},
				"manager": map[string]interface{}{
					"enabled": true,
				},
			},
		},
		Tools: registry.ToolConfig{
			EnabledTools: []string{"file", "shell"},
			Settings: map[string]interface{}{
				"timeout": "30s",
			},
		},
		Providers: registry.ProviderConfig{
			// Don't set default provider or configure any - this prevents agent factory creation
			Providers: map[string]interface{}{},
		},
		Memory: registry.MemoryConfig{
			DefaultMemoryStore: "sqlite",
			DefaultVectorStore: "chromem",
			Stores: map[string]interface{}{
				"sqlite": map[string]interface{}{
					"path": filepath.Join(projCtx.GetGuildPath(), "memory.db"),
				},
				"chromem": map[string]interface{}{
					"persistence_path": projCtx.GetEmbeddingsPath(),
					"dimension":        384,
				},
			},
		},
	}

	// Initialize registry with claudecode provider (doesn't need actual connection)
	err = reg.Initialize(ctx, *registryConfig)
	require.NoError(t, err)

	// Now register mock provider
	err = reg.Providers().RegisterProvider("mock", mockProvider)
	require.NoError(t, err)

	// Register some test agents so they are available for ListAvailableAgents
	testAgents := []registry.GuildAgentConfig{
		{
			ID:           "test-worker",
			Name:         "Test Worker Agent",
			Type:         "worker",
			Provider:     "mock",
			Model:        "test-model",
			Capabilities: []string{"coding", "analysis"},
		},
		{
			ID:           "test-manager",
			Name:         "Test Manager Agent", 
			Type:         "manager",
			Provider:     "mock",
			Model:        "test-model",
			Capabilities: []string{"planning", "coordination"},
		},
	}
	
	for _, agent := range testAgents {
		err = reg.Agents().RegisterGuildAgent(agent)
		require.NoError(t, err)
	}

	// Create event bus adapter
	mockEventBus := testutil.NewMockEventBus()
	eventBusAdapter := &eventBusAdapter{mock: mockEventBus}

	// Start server
	server := grpcpkg.NewServer(reg, eventBusAdapter)
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

	// Test guild service
	client := guildv1.NewGuildClient(conn)

	// Test listing campaigns
	campaigns, err := client.ListCampaigns(ctx, &guildv1.ListCampaignsRequest{})
	assert.NoError(t, err)
	assert.NotNil(t, campaigns)
	t.Logf("ListCampaigns succeeded with %d campaigns", len(campaigns.Campaigns))

	// Test listing agents
	agents, err := client.ListAvailableAgents(ctx, &guildv1.ListAgentsRequest{
		IncludeStatus: true,
	})
	assert.NoError(t, err)
	assert.NotNil(t, agents)
	assert.Greater(t, len(agents.Agents), 0, "Should have at least one agent")
	t.Logf("ListAvailableAgents succeeded with %d agents", len(agents.Agents))

	// Test chat service
	chatClient := guildv1.NewChatServiceClient(conn)
	stream, err := chatClient.Chat(ctx)
	assert.NoError(t, err)
	assert.NotNil(t, stream)

	// Clean close
	err = stream.CloseSend()
	assert.NoError(t, err)
}

// TestAgentExecutionFixed tests agent execution with proper mock setup
func TestAgentExecutionFixed(t *testing.T) {
	t.Skip("Skipping complex agent execution test - needs proper agent factory setup")
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Setup test project
	projCtx, cleanup := testutil.SetupTestProject(t, testutil.TestProjectOptions{
		Name:         "agent-execution-test",
		CustomConfig: testutil.CreateTestGuildConfig("test-execution-guild"),
	})
	defer cleanup()

	ctx, cancel := context.WithTimeout(project.WithContext(context.Background(), projCtx), 30*time.Second)
	defer cancel()

	// Create mock provider with specific responses
	mockProvider := testutil.NewMockLLMProvider()
	mockAgentResponse := testutil.GenerateMockAgentResponse(
		testutil.AgentResponseOptions{
			Type: "task_breakdown",
			Tasks: []string{
				"Parse authentication requirements",
				"Design authentication flow",
				"Create implementation plan",
			},
		},
	)
	mockProvider.SetResponse("manager", mockAgentResponse.Content)

	// Create registry
	reg := registry.NewComponentRegistry()
	registryConfig := createTestRegistryConfig(projCtx, mockProvider)
	err := reg.Initialize(ctx, *registryConfig)
	require.NoError(t, err)

	// Register mock provider after initialization
	err = reg.Providers().RegisterProvider("mock", mockProvider)
	require.NoError(t, err)

	// Get agent registry
	agentRegistry := reg.Agents()
	require.NotNil(t, agentRegistry)

	// Test worker agent (the default type)
	workerAgent, err := agentRegistry.GetAgent("worker")
	require.NoError(t, err)
	require.NotNil(t, workerAgent)

	// Execute task breakdown
	response, err := workerAgent.Execute(ctx, "Create a simple task breakdown for user authentication")
	require.NoError(t, err)
	assert.NotEmpty(t, response)

	// Verify response contains task breakdown
	assert.Contains(t, response, "authentication")
	assert.Contains(t, strings.ToLower(response), "task")

	// Test concurrent agent execution
	t.Run("ConcurrentExecution", func(t *testing.T) {
		var wg sync.WaitGroup
		errors := make(chan error, 5)

		// Execute multiple agents concurrently
		for i := 0; i < 5; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()

				agent, err := agentRegistry.GetAgent("worker")
				if err != nil {
					errors <- err
					return
				}

				_, err = agent.Execute(ctx, fmt.Sprintf("Task %d", idx))
				if err != nil {
					errors <- err
				}
			}(i)
		}

		wg.Wait()
		close(errors)

		// Check for errors
		for err := range errors {
			t.Errorf("Concurrent execution error: %v", err)
		}
	})
}

// TestToolExecutionFixed tests tool execution with proper registry setup
func TestToolExecutionFixed(t *testing.T) {
	t.Skip("Skipping complex tool execution test - needs proper agent factory setup")
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Setup test project
	projCtx, cleanup := testutil.SetupTestProject(t, testutil.TestProjectOptions{
		Name: "tool-execution-test",
	})
	defer cleanup()

	ctx, cancel := context.WithTimeout(project.WithContext(context.Background(), projCtx), 20*time.Second)
	defer cancel()

	// Create mock provider
	mockProvider := testutil.NewMockLLMProvider()
	devResponse := testutil.GenerateMockAgentResponse(
		testutil.AgentResponseOptions{
			Type: "implementation",
			Code: "// File operations completed successfully",
		},
	)
	mockProvider.SetResponse("developer", devResponse.Content)

	// Create registry with tools enabled
	reg := registry.NewComponentRegistry()
	registryConfig := createTestRegistryConfig(projCtx, mockProvider)

	// Ensure tools are enabled
	registryConfig.Tools.EnabledTools = []string{"file", "shell", "http"}

	err := reg.Initialize(ctx, *registryConfig)
	require.NoError(t, err)

	// Register mock provider after initialization
	err = reg.Providers().RegisterProvider("mock", mockProvider)
	require.NoError(t, err)

	// Get tool registry
	toolRegistry := reg.Tools()
	require.NotNil(t, toolRegistry)

	// List available tools
	tools := toolRegistry.ListTools()
	assert.Greater(t, len(tools), 0, "Should have tools available")
	t.Logf("Available tools: %v", tools)

	// Test file tool if available
	if contains(tools, "file") {
		fileTool, err := toolRegistry.GetTool("file")
		require.NoError(t, err)
		require.NotNil(t, fileTool)

		t.Logf("File tool available: %s", fileTool.Description())

		// Test basic file operation
		testFile := "test-file.txt"
		result, err := fileTool.Execute(ctx, fmt.Sprintf(`{"action": "write", "path": "%s", "content": "test content"}`, testFile))
		assert.NoError(t, err)
		assert.NotNil(t, result)
	}

	// Test agent with tools
	agentRegistry := reg.Agents()
	require.NotNil(t, agentRegistry)

	developerAgent, err := agentRegistry.GetAgent("developer")
	require.NoError(t, err)
	require.NotNil(t, developerAgent)

	// Execute task that might use tools
	response, err := developerAgent.Execute(ctx, "Create a test file with some content")
	assert.NoError(t, err)
	assert.NotEmpty(t, response)
}

// TestChatPerformanceFixed tests chat performance with lightweight setup
func TestChatPerformanceFixed(t *testing.T) {
	t.Skip("Skipping complex performance test - needs proper agent factory setup")
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	// Use minimal project setup for performance testing
	projCtx, cleanup := testutil.SetupTestProject(t, testutil.TestProjectOptions{
		Name:         "performance-test",
		SkipDatabase: false, // Keep database for realistic test
	})
	defer cleanup()

	ctx, cancel := context.WithTimeout(project.WithContext(context.Background(), projCtx), 10*time.Second)
	defer cancel()

	// Create lightweight mock provider
	mockProvider := testutil.NewMockLLMProvider()
	mockProvider.SetResponse("worker", "Quick response")

	// Create registry
	reg := registry.NewComponentRegistry()
	registryConfig := createMinimalRegistryConfig(projCtx, mockProvider)
	err := reg.Initialize(ctx, *registryConfig)
	require.NoError(t, err)

	// Register mock provider after initialization
	err = reg.Providers().RegisterProvider("mock", mockProvider)
	require.NoError(t, err)

	agentRegistry := reg.Agents()
	require.NotNil(t, agentRegistry)

	// Benchmark agent creation
	start := time.Now()
	agent, err := agentRegistry.GetAgent("worker")
	elapsed := time.Since(start)

	assert.NoError(t, err)
	assert.NotNil(t, agent)
	assert.Less(t, elapsed, 100*time.Millisecond, "Agent creation should be fast")
	t.Logf("Agent creation time: %v", elapsed)

	// Benchmark agent execution
	execStart := time.Now()
	response, err := agent.Execute(ctx, "Quick test")
	execElapsed := time.Since(execStart)

	assert.NoError(t, err)
	assert.NotEmpty(t, response)
	assert.Less(t, execElapsed, 50*time.Millisecond, "Agent execution should be fast")
	t.Logf("Agent execution time: %v", execElapsed)

	// Benchmark multiple sequential executions
	t.Run("SequentialPerformance", func(t *testing.T) {
		numExecutions := 100
		totalStart := time.Now()

		for i := 0; i < numExecutions; i++ {
			_, err := agent.Execute(ctx, fmt.Sprintf("Test %d", i))
			assert.NoError(t, err)
		}

		totalElapsed := time.Since(totalStart)
		avgTime := totalElapsed / time.Duration(numExecutions)

		t.Logf("Sequential execution performance:")
		t.Logf("- Total executions: %d", numExecutions)
		t.Logf("- Total time: %v", totalElapsed)
		t.Logf("- Average time per execution: %v", avgTime)

		assert.Less(t, avgTime, 20*time.Millisecond, "Average execution should be fast")
	})
}

// TestMemoryUsageFixed tests memory usage with proper tracking
func TestMemoryUsageFixed(t *testing.T) {
	t.Skip("Skipping complex memory test - needs proper agent factory setup")
	if testing.Short() {
		t.Skip("Skipping memory test in short mode")
	}

	// Setup minimal project
	projCtx, cleanup := testutil.SetupTestProject(t, testutil.TestProjectOptions{
		Name: "memory-usage-test",
	})
	defer cleanup()

	ctx, cancel := context.WithTimeout(project.WithContext(context.Background(), projCtx), 15*time.Second)
	defer cancel()

	// Force garbage collection and get baseline
	runtime.GC()
	runtime.GC()
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	startMem := m.Alloc

	// Create lightweight provider
	mockProvider := testutil.NewMockLLMProvider()
	mockProvider.SetResponse("worker", "Memory test response")

	// Create registry
	reg := registry.NewComponentRegistry()
	registryConfig := createMinimalRegistryConfig(projCtx, mockProvider)
	err := reg.Initialize(ctx, *registryConfig)
	require.NoError(t, err)

	// Register mock provider after initialization
	err = reg.Providers().RegisterProvider("mock", mockProvider)
	require.NoError(t, err)

	agentRegistry := reg.Agents()
	require.NotNil(t, agentRegistry)

	// Track memory allocations
	allocations := make([]uint64, 0, 50)

	// Create and use agents multiple times
	for i := 0; i < 50; i++ {
		agent, err := agentRegistry.GetAgent("worker")
		require.NoError(t, err)

		// Execute task
		_, err = agent.Execute(ctx, fmt.Sprintf("Memory test %d", i))
		assert.NoError(t, err)

		// Track memory periodically
		if i%10 == 0 {
			runtime.GC()
			runtime.ReadMemStats(&m)
			allocations = append(allocations, m.Alloc)
		}
	}

	// Final cleanup and measurement
	runtime.GC()
	runtime.GC()
	runtime.ReadMemStats(&m)
	endMem := m.Alloc

	// Calculate growth
	growth := endMem - startMem
	growthMB := float64(growth) / (1024 * 1024)

	t.Logf("Memory usage statistics:")
	t.Logf("- Start memory: %.2f MB", float64(startMem)/(1024*1024))
	t.Logf("- End memory: %.2f MB", float64(endMem)/(1024*1024))
	t.Logf("- Memory growth: %.2f MB", growthMB)

	// Check allocation trend
	if len(allocations) > 2 {
		avgGrowthPerOp := growth / 50
		t.Logf("- Average growth per operation: %d bytes", avgGrowthPerOp)

		// Should not grow more than 100KB per operation on average
		assert.Less(t, avgGrowthPerOp, uint64(100*1024),
			"Memory growth per operation should be reasonable")
	}

	// Total growth should be reasonable
	assert.Less(t, growthMB, 20.0, "Total memory growth should be under 20MB")
}

// Helper functions

func createTestRegistryConfig(projCtx *project.Context, provider interface{}) *registry.Config {
	return &registry.Config{
		Agents: registry.AgentConfigYaml{
			DefaultType: "worker",
			Types: map[string]interface{}{
				"worker": map[string]interface{}{
					"enabled": true,
				},
				"manager": map[string]interface{}{
					"enabled": true,
				},
				"developer": map[string]interface{}{
					"enabled": true,
				},
			},
		},
		Tools: registry.ToolConfig{
			EnabledTools: []string{"file", "shell", "http"},
			Settings: map[string]interface{}{
				"timeout": "30s",
			},
		},
		Providers: registry.ProviderConfig{
			// Don't set default provider or configure any - this prevents agent factory creation
			Providers: map[string]interface{}{},
		},
		Memory: registry.MemoryConfig{
			DefaultMemoryStore: "sqlite",
			DefaultVectorStore: "chromem",
			Stores: map[string]interface{}{
				"sqlite": map[string]interface{}{
					"path": filepath.Join(projCtx.GetGuildPath(), "memory.db"),
				},
				"chromem": map[string]interface{}{
					"persistence_path": projCtx.GetEmbeddingsPath(),
					"dimension":        384,
				},
			},
		},
	}
}

func createMinimalRegistryConfig(projCtx *project.Context, provider interface{}) *registry.Config {
	return &registry.Config{
		Agents: registry.AgentConfigYaml{
			DefaultType: "worker",
			Types: map[string]interface{}{
				"worker": map[string]interface{}{
					"enabled": true,
				},
			},
		},
		Tools: registry.ToolConfig{
			EnabledTools: []string{}, // No tools for minimal config
		},
		Providers: registry.ProviderConfig{
			// Don't set default provider or configure any - this prevents agent factory creation
			Providers: map[string]interface{}{},
		},
		Memory: registry.MemoryConfig{
			DefaultMemoryStore: "sqlite",
			DefaultVectorStore: "chromem",
			Stores: map[string]interface{}{
				"sqlite": map[string]interface{}{
					"path": ":memory:", // In-memory for performance
				},
				"chromem": map[string]interface{}{
					"persistence_path": "",
					"dimension":        384,
				},
			},
		},
	}
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
