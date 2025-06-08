// Package integration provides registry integration for MCP
package integration

import (
	"context"
	"time"

	"github.com/guild-ventures/guild-core/pkg/gerror"
	mcpconfig "github.com/guild-ventures/guild-core/pkg/mcp/config"
	"github.com/guild-ventures/guild-core/pkg/mcp/server"
	"github.com/guild-ventures/guild-core/pkg/mcp/transport"
	"github.com/guild-ventures/guild-core/pkg/registry"
)

// MCPRegistryAdapter adapts MCP server for Guild registry
type MCPRegistryAdapter struct {
	server *server.Server
	config *mcpconfig.MCPConfig
}

// NewMCPRegistryAdapter creates a new MCP registry adapter
func NewMCPRegistryAdapter(config *mcpconfig.MCPConfig, guildRegistry registry.ComponentRegistry) (*MCPRegistryAdapter, error) {
	if config == nil {
		return nil, gerror.New(gerror.ErrCodeInvalidInput, "mcp_integration", nil).WithComponent("new_mcp_registry_adapter").WithOperation("MCP config cannot be nil")
	}

	if !config.Enabled {
		return &MCPRegistryAdapter{config: config}, nil
	}

	// Convert MCP config to server config
	serverConfig, err := convertToServerConfig(config)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "mcp_integration").WithComponent("new_mcp_registry_adapter").WithOperation("failed to convert MCP config")
	}

	// Create MCP server
	mcpServer, err := server.NewServer(serverConfig, guildRegistry)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "mcp_integration").WithComponent("new_mcp_registry_adapter").WithOperation("failed to create MCP server")
	}

	return &MCPRegistryAdapter{
		server: mcpServer,
		config: config,
	}, nil
}

// Start starts the MCP server
func (a *MCPRegistryAdapter) Start(ctx context.Context) error {
	if !a.config.Enabled || a.server == nil {
		return nil // MCP disabled
	}

	if err := a.server.Start(ctx); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "mcp_integration").WithComponent("start").WithOperation("failed to start MCP server")
	}

	return nil
}

// Stop stops the MCP server
func (a *MCPRegistryAdapter) Stop(ctx context.Context) error {
	if a.server == nil {
		return nil
	}

	if err := a.server.Stop(ctx); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "mcp_integration").WithComponent("shutdown").WithOperation("failed to stop MCP server")
	}

	return nil
}

// GetServer returns the underlying MCP server
func (a *MCPRegistryAdapter) GetServer() *server.Server {
	return a.server
}

// IsEnabled returns whether MCP is enabled
func (a *MCPRegistryAdapter) IsEnabled() bool {
	return a.config.Enabled
}

// GetConfig returns the MCP configuration
func (a *MCPRegistryAdapter) GetConfig() *mcpconfig.MCPConfig {
	return a.config
}

// Health checks the health of the MCP server
func (a *MCPRegistryAdapter) Health(ctx context.Context) error {
	if !a.config.Enabled || a.server == nil {
		return nil
	}

	// Perform health check - could ping server or check components
	// For now, just verify server is configured
	if a.server.GetConfig() == nil {
		return gerror.New(gerror.ErrCodeInternal, "mcp_integration", nil).WithComponent("get_status").WithOperation("MCP server not properly configured")
	}

	return nil
}

// convertToServerConfig converts MCPConfig to server.Config
func convertToServerConfig(config *mcpconfig.MCPConfig) (*server.Config, error) {
	// Set defaults
	if config.ServerID == "" {
		config.ServerID = "guild-mcp-server"
	}
	if config.ServerName == "" {
		config.ServerName = "Guild MCP Server"
	}

	// Default transport config if not provided
	if config.Transport == nil {
		config.Transport = &transport.TransportConfig{
			Type:    "memory", // Default to memory for testing
			Address: "memory://",
		}
	}

	return &server.Config{
		ServerID:              config.ServerID,
		ServerName:            config.ServerName,
		Version:               "1.0.0",
		TransportConfig:       config.Transport,
		EnableTLS:             config.EnableTLS,
		TLSCertFile:           "", // TODO: Add to MCPConfig if needed
		TLSKeyFile:            "", // TODO: Add to MCPConfig if needed
		EnableAuth:            config.EnableAuth,
		JWTSecret:             "",               // TODO: Add to MCPConfig if needed
		MaxConcurrentRequests: 100,              // Default value
		RequestTimeout:        30 * time.Second, // Default 30 seconds
		EnableMetrics:         config.EnableMetrics,
		EnableTracing:         config.EnableTracing,
		EnableCostTracking:    config.EnableCost,
	}, nil
}

// MCPRegistryExtension extends the Guild registry with MCP support
type MCPRegistryExtension struct {
	adapter    *MCPRegistryAdapter
	toolBridge *ToolBridge
}

// NewMCPRegistryExtension creates a new MCP registry extension
func NewMCPRegistryExtension(config *mcpconfig.MCPConfig, guildRegistry registry.ComponentRegistry) (*MCPRegistryExtension, error) {
	adapter, err := NewMCPRegistryAdapter(config, guildRegistry)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "mcp_integration").WithComponent("default_mcp_factory").WithOperation("failed to create MCP adapter")
	}

	var toolBridge *ToolBridge
	if adapter.IsEnabled() && adapter.GetServer() != nil {
		// Create tool bridge
		mcpToolRegistry := adapter.GetServer().GetToolRegistry()
		guildToolRegistry := guildRegistry.Tools()

		if guildToolRegistry != nil {
			toolBridge = NewToolBridge(mcpToolRegistry, guildToolRegistry)
		}
	}

	return &MCPRegistryExtension{
		adapter:    adapter,
		toolBridge: toolBridge,
	}, nil
}

// Initialize initializes the MCP extension
func (e *MCPRegistryExtension) Initialize(ctx context.Context) error {
	// Start MCP server
	if err := e.adapter.Start(ctx); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "mcp_integration").WithComponent("sync_with_registry").WithOperation("failed to start MCP adapter")
	}

	// Sync tools if bridge is available
	if e.toolBridge != nil {
		if err := e.toolBridge.SyncAll(ctx); err != nil {
			return gerror.Wrap(err, gerror.ErrCodeInternal, "mcp_integration").WithComponent("sync_with_registry").WithOperation("failed to sync tools")
		}
	}

	return nil
}

// Shutdown shuts down the MCP extension
func (e *MCPRegistryExtension) Shutdown(ctx context.Context) error {
	return e.adapter.Stop(ctx)
}

// GetMCPServer returns the MCP server
func (e *MCPRegistryExtension) GetMCPServer() *server.Server {
	return e.adapter.GetServer()
}

// GetToolBridge returns the tool bridge
func (e *MCPRegistryExtension) GetToolBridge() *ToolBridge {
	return e.toolBridge
}

// SyncTools manually syncs tools between registries
func (e *MCPRegistryExtension) SyncTools(ctx context.Context) error {
	if e.toolBridge == nil {
		return gerror.New(gerror.ErrCodeInternal, "mcp_integration", nil).WithComponent("handle_tool_request").WithOperation("tool bridge not available")
	}
	return e.toolBridge.SyncAll(ctx)
}
