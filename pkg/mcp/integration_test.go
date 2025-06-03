package mcp_test

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/guild-ventures/guild-core/pkg/mcp/client"
	"github.com/guild-ventures/guild-core/pkg/mcp/protocol"
	"github.com/guild-ventures/guild-core/pkg/mcp/server"
	"github.com/guild-ventures/guild-core/pkg/mcp/transport"
	"github.com/guild-ventures/guild-core/pkg/registry"
)

func TestMCPIntegration(t *testing.T) {
	ctx := context.Background()

	// Create guild registry
	guildRegistry := registry.NewComponentRegistry()

	// Configure server
	serverConfig := &server.Config{
		ServerID:              "test-server",
		ServerName:            "Test MCP Server",
		Version:               "1.0.0",
		TransportConfig:       &transport.TransportConfig{
			Type: "memory", // Use memory transport for testing
		},
		MaxConcurrentRequests: 100,
		RequestTimeout:        30 * time.Second,
		EnableCostTracking:    true,
		EnableMetrics:         true,
	}

	// Create and start server
	mcpServer, err := server.NewServer(serverConfig, guildRegistry)
	require.NoError(t, err)

	err = mcpServer.Start(ctx)
	require.NoError(t, err)
	defer mcpServer.Stop(ctx)

	// Configure client
	clientConfig := &client.Config{
		ClientID:        "test-client",
		ClientName:      "Test MCP Client",
		Version:         "1.0.0",
		TransportConfig: &transport.TransportConfig{
			Type: "memory", // Use memory transport for testing
		},
		RequestTimeout: 10 * time.Second,
	}

	// Create and connect client
	mcpClient, err := client.NewClient(clientConfig)
	require.NoError(t, err)

	err = mcpClient.Connect(ctx)
	require.NoError(t, err)
	defer mcpClient.Disconnect(ctx)

	t.Run("ping", func(t *testing.T) {
		timestamp, err := mcpClient.Ping(ctx)
		require.NoError(t, err)
		assert.WithinDuration(t, time.Now(), timestamp, 5*time.Second)
	})

	t.Run("system_info", func(t *testing.T) {
		info, err := mcpClient.GetSystemInfo(ctx)
		require.NoError(t, err)
		assert.Equal(t, "test-server", info["server_id"])
		assert.Equal(t, "Test MCP Server", info["server_name"])
		assert.Equal(t, "1.0.0", info["version"])
	})

	t.Run("tool_lifecycle", func(t *testing.T) {
		// Define a test tool
		toolDef := &protocol.ToolDefinition{
			ID:          "test-calculator",
			Name:        "Calculator",
			Description: "Simple calculator tool",
			Capabilities: []string{"math", "calculation"},
			Parameters: []protocol.ToolParameter{
				{
					Name:        "operation",
					Type:        "string",
					Description: "Math operation to perform",
					Required:    true,
				},
				{
					Name:        "a",
					Type:        "number",
					Description: "First operand",
					Required:    true,
				},
				{
					Name:        "b",
					Type:        "number",
					Description: "Second operand",
					Required:    true,
				},
			},
			Returns: []protocol.ToolParameter{
				{
					Name:        "result",
					Type:        "number",
					Description: "Calculation result",
				},
			},
			CostProfile: protocol.CostProfile{
				ComputeCost:   0.001,
				MemoryCost:    1024,
				LatencyCost:   100 * time.Millisecond,
				FinancialCost: 0.0001,
			},
		}

		// Register tool
		err := mcpClient.RegisterTool(ctx, toolDef)
		require.NoError(t, err)

		// Discover tools
		query := &protocol.ToolQuery{
			RequiredCapabilities: []string{"math"},
			MaxCost:              1.0,
		}

		discovery, err := mcpClient.DiscoverTools(ctx, query)
		require.NoError(t, err)
		assert.Len(t, discovery.Tools, 1)
		assert.Equal(t, "test-calculator", discovery.Tools[0].ToolID)

		// Check tool health
		healthy, err := mcpClient.CheckToolHealth(ctx, "test-calculator")
		require.NoError(t, err)
		// Note: Health check might fail since we don't have a real executor
		_ = healthy

		// Deregister tool
		err = mcpClient.DeregisterTool(ctx, "test-calculator")
		require.NoError(t, err)

		// Verify tool is gone
		discovery, err = mcpClient.DiscoverTools(ctx, query)
		require.NoError(t, err)
		assert.Len(t, discovery.Tools, 0)
	})

	t.Run("cost_tracking", func(t *testing.T) {
		// Report some costs
		cost1 := &protocol.CostReport{
			OperationID:   "op-001",
			StartTime:     time.Now().Add(-1 * time.Hour),
			EndTime:       time.Now().Add(-1*time.Hour + 5*time.Second),
			ComputeCost:   0.01,
			MemoryCost:    2048,
			LatencyCost:   5 * time.Second,
			TokensCost:    100,
			APICallsCost:  1,
			FinancialCost: 0.001,
		}

		cost2 := &protocol.CostReport{
			OperationID:   "op-002",
			StartTime:     time.Now().Add(-30 * time.Minute),
			EndTime:       time.Now().Add(-30*time.Minute + 3*time.Second),
			ComputeCost:   0.005,
			MemoryCost:    1024,
			LatencyCost:   3 * time.Second,
			TokensCost:    50,
			APICallsCost:  1,
			FinancialCost: 0.0005,
		}

		err := mcpClient.ReportCost(ctx, cost1)
		require.NoError(t, err)

		err = mcpClient.ReportCost(ctx, cost2)
		require.NoError(t, err)

		// Query costs
		query := &protocol.CostQuery{
			StartTime: time.Now().Add(-2 * time.Hour),
			EndTime:   time.Now(),
			GroupBy:   "operation",
		}

		analysis, err := mcpClient.QueryCosts(ctx, query)
		require.NoError(t, err)
		
		assert.Equal(t, 0.015, analysis.TotalCost.ComputeCost)
		assert.Equal(t, int64(3072), analysis.TotalCost.MemoryCost)
		assert.Equal(t, 150, analysis.TotalCost.TokensCost)
		assert.Equal(t, 2, analysis.TotalCost.APICallsCost)
		assert.Equal(t, 0.0015, analysis.TotalCost.FinancialCost)
		assert.Len(t, analysis.Breakdown, 2)
	})

	t.Run("prompt_processing", func(t *testing.T) {
		prompt := &protocol.PromptMessage{
			Text:        "Hello, world!",
			HistoryID:   "conv-001",
			MaxTokens:   100,
			Temperature: 0.7,
			Parameters: map[string]interface{}{
				"model": "test-model",
			},
		}

		response, err := mcpClient.ProcessPrompt(ctx, prompt)
		require.NoError(t, err)
		assert.NotEmpty(t, response.Text)
		// Note: The actual response content will depend on the prompt processor implementation
	})
}

func TestMCPConcurrency(t *testing.T) {
	ctx := context.Background()

	// Create guild registry
	guildRegistry := registry.NewComponentRegistry()

	// Configure server for high concurrency
	serverConfig := &server.Config{
		ServerID:              "concurrent-server",
		ServerName:            "Concurrent Test Server",
		Version:               "1.0.0",
		TransportConfig:       &transport.TransportConfig{
			Type: "memory",
		},
		MaxConcurrentRequests: 1000,
		RequestTimeout:        5 * time.Second,
		EnableCostTracking:    true,
	}

	mcpServer, err := server.NewServer(serverConfig, guildRegistry)
	require.NoError(t, err)

	err = mcpServer.Start(ctx)
	require.NoError(t, err)
	defer mcpServer.Stop(ctx)

	// Create multiple clients
	numClients := 10
	clients := make([]*client.Client, numClients)

	for i := 0; i < numClients; i++ {
		clientConfig := &client.Config{
			ClientID:        fmt.Sprintf("client-%d", i),
			ClientName:      fmt.Sprintf("Test Client %d", i),
			Version:         "1.0.0",
			TransportConfig: &transport.TransportConfig{
				Type: "memory",
			},
			RequestTimeout: 5 * time.Second,
		}

		mcpClient, err := client.NewClient(clientConfig)
		require.NoError(t, err)

		err = mcpClient.Connect(ctx)
		require.NoError(t, err)
		defer mcpClient.Disconnect(ctx)

		clients[i] = mcpClient
	}

	// Test concurrent tool registration
	t.Run("concurrent_tool_registration", func(t *testing.T) {
		done := make(chan bool, numClients)
		errors := make(chan error, numClients)

		for i, mcpClient := range clients {
			go func(clientIdx int, client *client.Client) {
				toolDef := &protocol.ToolDefinition{
					ID:          fmt.Sprintf("tool-%d", clientIdx),
					Name:        fmt.Sprintf("Tool %d", clientIdx),
					Description: "Concurrent test tool",
					Capabilities: []string{"test"},
					CostProfile: protocol.CostProfile{
						ComputeCost:   0.001,
						FinancialCost: 0.0001,
					},
				}

				if err := client.RegisterTool(ctx, toolDef); err != nil {
					errors <- err
					return
				}

				done <- true
			}(i, mcpClient)
		}

		// Wait for all registrations to complete
		for i := 0; i < numClients; i++ {
			select {
			case <-done:
				// Success
			case err := <-errors:
				t.Errorf("Tool registration failed: %v", err)
			case <-time.After(10 * time.Second):
				t.Error("Timeout waiting for tool registration")
			}
		}
	})

	// Test concurrent cost reporting
	t.Run("concurrent_cost_reporting", func(t *testing.T) {
		done := make(chan bool, numClients*10) // 10 reports per client
		errors := make(chan error, numClients*10)

		for i, mcpClient := range clients {
			go func(clientIdx int, client *client.Client) {
				for j := 0; j < 10; j++ {
					cost := &protocol.CostReport{
						OperationID:   fmt.Sprintf("op-%d-%d", clientIdx, j),
						StartTime:     time.Now(),
						EndTime:       time.Now().Add(time.Millisecond * 100),
						ComputeCost:   0.001,
						FinancialCost: 0.0001,
					}

					if err := client.ReportCost(ctx, cost); err != nil {
						errors <- err
						return
					}

					done <- true
				}
			}(i, mcpClient)
		}

		// Wait for all reports to complete
		expectedReports := numClients * 10
		for i := 0; i < expectedReports; i++ {
			select {
			case <-done:
				// Success
			case err := <-errors:
				t.Errorf("Cost reporting failed: %v", err)
			case <-time.After(15 * time.Second):
				t.Error("Timeout waiting for cost reporting")
			}
		}
	})
}

func TestMCPErrorHandling(t *testing.T) {
	ctx := context.Background()

	// Create guild registry
	guildRegistry := registry.NewComponentRegistry()

	serverConfig := &server.Config{
		ServerID:        "error-test-server",
		ServerName:      "Error Test Server",
		Version:         "1.0.0",
		TransportConfig: &transport.TransportConfig{
			Type: "memory",
		},
		RequestTimeout: 5 * time.Second,
	}

	mcpServer, err := server.NewServer(serverConfig, guildRegistry)
	require.NoError(t, err)

	err = mcpServer.Start(ctx)
	require.NoError(t, err)
	defer mcpServer.Stop(ctx)

	clientConfig := &client.Config{
		ClientID:        "error-test-client",
		TransportConfig: &transport.TransportConfig{
			Type: "memory",
		},
		RequestTimeout: 5 * time.Second,
	}

	mcpClient, err := client.NewClient(clientConfig)
	require.NoError(t, err)

	err = mcpClient.Connect(ctx)
	require.NoError(t, err)
	defer mcpClient.Disconnect(ctx)

	t.Run("tool_not_found", func(t *testing.T) {
		healthy, err := mcpClient.CheckToolHealth(ctx, "nonexistent-tool")
		assert.Error(t, err)
		assert.False(t, healthy)
	})

	t.Run("duplicate_tool_registration", func(t *testing.T) {
		toolDef := &protocol.ToolDefinition{
			ID:          "duplicate-tool",
			Name:        "Duplicate Tool",
			Description: "Tool to test duplicate registration",
		}

		// First registration should succeed
		err := mcpClient.RegisterTool(ctx, toolDef)
		require.NoError(t, err)

		// Second registration should fail
		err = mcpClient.RegisterTool(ctx, toolDef)
		assert.Error(t, err)

		// Cleanup
		mcpClient.DeregisterTool(ctx, "duplicate-tool")
	})

	t.Run("invalid_tool_execution", func(t *testing.T) {
		// Try to execute a non-existent tool
		req := &protocol.ToolExecutionRequest{
			ToolID:     "nonexistent-tool",
			Parameters: map[string]interface{}{},
		}

		_, err := mcpClient.ExecuteTool(ctx, req)
		assert.Error(t, err)
	})
}

// Helper function to set up memory transport for testing
func setupMemoryTransport() *transport.TransportConfig {
	return &transport.TransportConfig{
		Type: "memory",
		Config: map[string]interface{}{
			"buffer_size": 1000,
		},
	}
}