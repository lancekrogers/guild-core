package integration_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/guild-ventures/guild-core/pkg/config"
	"github.com/guild-ventures/guild-core/pkg/mcp/integration"
	"github.com/guild-ventures/guild-core/pkg/mcp/protocol"
	"github.com/guild-ventures/guild-core/pkg/mcp/tools"
	"github.com/guild-ventures/guild-core/pkg/registry"
)

func TestContextPropagation(t *testing.T) {
	ctx := context.Background()

	// Create test config
	mcpConfig := config.ExampleDevelopmentConfig()
	integrationConfig, err := mcpConfig.ToIntegrationConfig()
	require.NoError(t, err)

	// Create extended registry with MCP
	extendedRegistry, err := registry.NewExtendedComponentRegistry(ctx, integrationConfig)
	require.NoError(t, err)
	defer extendedRegistry.Close(ctx)

	// Verify MCP is enabled and started
	assert.True(t, extendedRegistry.IsMCPEnabled())

	// Test context propagation through tool registration
	t.Run("tool_registration_context", func(t *testing.T) {
		// Create a context with test values
		testCtx := context.WithValue(ctx, "test_key", "test_value")
		testCtx = context.WithValue(testCtx, "request_id", "test-req-123")

		// Get MCP server
		mcpServer := extendedRegistry.GetMCPExtension().GetMCPServer()
		require.NotNil(t, mcpServer)

		// Create a test tool that verifies context
		testTool := tools.NewBaseTool(
			"context-test-tool",
			"Context Test Tool",
			"Tool for testing context propagation",
			[]string{"test", "context"},
			protocol.CostProfile{
				ComputeCost:   0.001,
				FinancialCost: 0.0001,
			},
			[]protocol.ToolParameter{
				{
					Name:        "input",
					Type:        "string",
					Description: "Test input",
					Required:    true,
				},
			},
			[]protocol.ToolParameter{
				{
					Name:        "output",
					Type:        "string",
					Description: "Test output",
				},
			},
			func(execCtx context.Context, params map[string]interface{}) (interface{}, error) {
				// Verify context values are preserved
				if val := execCtx.Value("test_key"); val != "test_value" {
					t.Errorf("Expected context value 'test_value', got %v", val)
				}

				input, ok := params["input"].(string)
				if !ok {
					return nil, fmt.Errorf("input must be string")
				}

				return map[string]interface{}{
					"output": "processed: " + input,
				}, nil
			},
		)

		// Register tool through MCP registry
		toolRegistry := mcpServer.GetToolRegistry()
		err := toolRegistry.RegisterTool(testTool)
		require.NoError(t, err)

		// Execute tool to verify context propagation
		result, err := testTool.Execute(testCtx, map[string]interface{}{
			"input": "test input",
		})
		require.NoError(t, err)

		resultMap, ok := result.(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, "processed: test input", resultMap["output"])
	})
}

func TestRegistryPatternIntegration(t *testing.T) {
	ctx := context.Background()

	// Create test config
	mcpConfig := config.ExampleDevelopmentConfig()
	integrationConfig, err := mcpConfig.ToIntegrationConfig()
	require.NoError(t, err)

	// Create extended registry
	extendedRegistry, err := registry.NewExtendedComponentRegistry(ctx, integrationConfig)
	require.NoError(t, err)
	defer extendedRegistry.Close(ctx)

	t.Run("mcp_server_registration", func(t *testing.T) {
		// Verify MCP server is registered and accessible
		mcpExtension := extendedRegistry.GetMCPExtension()
		require.NotNil(t, mcpExtension)

		mcpServer := mcpExtension.GetMCPServer()
		require.NotNil(t, mcpServer)

		// Verify server configuration
		serverConfig := mcpServer.GetConfig()
		require.NotNil(t, serverConfig)
		assert.Equal(t, "dev-mcp-server", serverConfig.ServerID)
	})

	t.Run("tool_registry_bridge", func(t *testing.T) {
		// Get the tool bridge
		toolBridge := extendedRegistry.GetMCPExtension().GetToolBridge()
		require.NotNil(t, toolBridge)

		// Create test tools in MCP registry
		mcpServer := extendedRegistry.GetMCPExtension().GetMCPServer()
		mcpToolRegistry := mcpServer.GetToolRegistry()

		testTool := tools.NewBaseTool(
			"bridge-test-tool",
			"Bridge Test Tool",
			"Tool for testing registry bridge",
			[]string{"test", "bridge"},
			protocol.CostProfile{},
			[]protocol.ToolParameter{},
			[]protocol.ToolParameter{},
			func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
				return "bridge test result", nil
			},
		)

		// Register in MCP registry
		err := mcpToolRegistry.RegisterTool(testTool)
		require.NoError(t, err)

		// Sync tools to Guild registry
		err = extendedRegistry.SyncMCPTools(ctx)
		require.NoError(t, err)

		// Verify tool appears in both registries
		mcpTools := mcpToolRegistry.ListTools()
		assert.Len(t, mcpTools, 1)
		assert.Equal(t, "bridge-test-tool", mcpTools[0].ID())

		// Note: Full Guild registry integration would require actual Guild tool registry implementation
		// This test verifies the bridge mechanism is in place
	})

	t.Run("registry_lifecycle", func(t *testing.T) {
		// Test proper startup and shutdown
		assert.True(t, extendedRegistry.IsMCPEnabled())

		// Create a new registry to test lifecycle
		newConfig := config.ExampleDevelopmentConfig()
		newIntegrationConfig, err := newConfig.ToIntegrationConfig()
		require.NoError(t, err)

		newRegistry, err := registry.NewExtendedComponentRegistry(ctx, newIntegrationConfig)
		require.NoError(t, err)

		// Verify it starts properly
		assert.True(t, newRegistry.IsMCPEnabled())

		// Test shutdown
		err = newRegistry.Close(ctx)
		assert.NoError(t, err)
	})
}

func TestInterfaceFirstDesign(t *testing.T) {
	ctx := context.Background()

	// Create config
	mcpConfig := config.ExampleDevelopmentConfig()
	integrationConfig, err := mcpConfig.ToIntegrationConfig()
	require.NoError(t, err)

	t.Run("interface_compliance", func(t *testing.T) {
		// Verify MCP components implement expected interfaces
		extendedRegistry, err := registry.NewExtendedComponentRegistry(ctx, integrationConfig)
		require.NoError(t, err)
		defer extendedRegistry.Close(ctx)

		// Verify ExtendedComponentRegistry implements MCPRegistry interface
		var mcpRegistry registry.MCPRegistry = extendedRegistry
		assert.NotNil(t, mcpRegistry)

		// Verify interface methods work
		assert.True(t, mcpRegistry.IsMCPEnabled())

		extension := mcpRegistry.GetMCPExtension()
		assert.NotNil(t, extension)

		// Test interface methods
		err = mcpRegistry.SyncMCPTools(ctx)
		assert.NoError(t, err)
	})

	t.Run("dependency_injection", func(t *testing.T) {
		// Test that components properly use dependency injection
		extension, err := integration.NewMCPRegistryExtension(integrationConfig, registry.New())
		require.NoError(t, err)

		// Verify dependencies are properly injected
		server := extension.GetMCPServer()
		require.NotNil(t, server)

		toolBridge := extension.GetToolBridge()
		require.NotNil(t, toolBridge)

		// Cleanup
		err = extension.Shutdown(ctx)
		assert.NoError(t, err)
	})
}

func TestConfigurationIntegration(t *testing.T) {
	t.Run("config_validation", func(t *testing.T) {
		// Test valid config
		validConfig := config.ExampleDevelopmentConfig()
		err := validConfig.Validate()
		assert.NoError(t, err)

		// Test invalid config
		invalidConfig := &config.MCPConfig{
			Enabled:    true,
			ServerID:   "", // Invalid: empty server ID
			ServerName: "Test Server",
		}
		err = invalidConfig.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "server_id is required")
	})

	t.Run("config_conversion", func(t *testing.T) {
		// Test config conversion
		guildConfig := config.ExampleNATSConfig()
		integrationConfig, err := guildConfig.ToIntegrationConfig()
		require.NoError(t, err)

		assert.Equal(t, guildConfig.Enabled, integrationConfig.Enabled)
		assert.Equal(t, guildConfig.ServerID, integrationConfig.ServerID)
		assert.Equal(t, guildConfig.Transport.Type, integrationConfig.Transport.Type)
	})

	t.Run("default_configs", func(t *testing.T) {
		// Test default configurations
		defaultConfig := config.DefaultMCPConfig()
		assert.False(t, defaultConfig.Enabled) // Should be disabled by default

		prodConfig := config.ProductionMCPConfig()
		assert.True(t, prodConfig.Enabled)
		assert.True(t, prodConfig.Security.EnableTLS)
		assert.Equal(t, "nats", prodConfig.Transport.Type)
	})
}

func TestErrorHandling(t *testing.T) {
	ctx := context.Background()

	t.Run("nil_config_handling", func(t *testing.T) {
		// Test handling of nil config
		extendedRegistry, err := registry.NewExtendedComponentRegistry(ctx, nil)
		require.NoError(t, err)
		defer extendedRegistry.Close(ctx)

		// Should not be enabled with nil config
		assert.False(t, extendedRegistry.IsMCPEnabled())
	})

	t.Run("disabled_mcp_handling", func(t *testing.T) {
		// Test disabled MCP configuration
		disabledConfig := &integration.MCPConfig{
			Enabled: false,
		}

		extendedRegistry, err := registry.NewExtendedComponentRegistry(ctx, disabledConfig)
		require.NoError(t, err)
		defer extendedRegistry.Close(ctx)

		assert.False(t, extendedRegistry.IsMCPEnabled())

		// Operations should be safe even when disabled
		err = extendedRegistry.SyncMCPTools(ctx)
		assert.Error(t, err) // Should error because MCP is not configured

		err = extendedRegistry.StopMCP(ctx)
		assert.NoError(t, err) // Should not error - nothing to stop
	})
}

func TestConcurrency(t *testing.T) {
	ctx := context.Background()

	// Create test registry
	mcpConfig := config.ExampleDevelopmentConfig()
	integrationConfig, err := mcpConfig.ToIntegrationConfig()
	require.NoError(t, err)

	extendedRegistry, err := registry.NewExtendedComponentRegistry(ctx, integrationConfig)
	require.NoError(t, err)
	defer extendedRegistry.Close(ctx)

	t.Run("concurrent_tool_operations", func(t *testing.T) {
		mcpServer := extendedRegistry.GetMCPExtension().GetMCPServer()
		toolRegistry := mcpServer.GetToolRegistry()

		// Test concurrent tool registration
		const numTools = 10
		done := make(chan bool, numTools)
		errors := make(chan error, numTools)

		for i := 0; i < numTools; i++ {
			go func(index int) {
				tool := tools.NewBaseTool(
					fmt.Sprintf("concurrent-tool-%d", index),
					fmt.Sprintf("Concurrent Tool %d", index),
					"Tool for concurrency testing",
					[]string{"test", "concurrent"},
					protocol.CostProfile{},
					[]protocol.ToolParameter{},
					[]protocol.ToolParameter{},
					func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
						return fmt.Sprintf("result-%d", index), nil
					},
				)

				if err := toolRegistry.RegisterTool(tool); err != nil {
					errors <- err
					return
				}

				done <- true
			}(i)
		}

		// Wait for all registrations
		for i := 0; i < numTools; i++ {
			select {
			case <-done:
				// Success
			case err := <-errors:
				t.Errorf("Concurrent registration failed: %v", err)
			case <-time.After(5 * time.Second):
				t.Error("Timeout waiting for concurrent registration")
			}
		}

		// Verify all tools were registered
		tools := toolRegistry.ListTools()
		assert.Len(t, tools, numTools)
	})
}