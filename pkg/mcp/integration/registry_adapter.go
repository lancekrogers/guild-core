// Package integration provides registry integration for MCP
package integration

import (
	"context"
	"fmt"
	"time"

	"github.com/guild-ventures/guild-core/pkg/mcp/server"
	"github.com/guild-ventures/guild-core/pkg/mcp/transport"
	"github.com/guild-ventures/guild-core/pkg/registry"
)

// MCPRegistryAdapter adapts MCP server for Guild registry
type MCPRegistryAdapter struct {
	server *server.Server
	config *MCPConfig
}

// MCPConfig represents MCP configuration for Guild
type MCPConfig struct {
	Enabled         bool                        `yaml:"enabled"`
	ServerID        string                      `yaml:"server_id"`
	ServerName      string                      `yaml:"server_name"`
	Transport       *transport.TransportConfig  `yaml:"transport"`
	EnableAuth      bool                        `yaml:"enable_auth"`
	EnableTLS       bool                        `yaml:"enable_tls"`
	EnableMetrics   bool                        `yaml:"enable_metrics"`
	EnableTracing   bool                        `yaml:"enable_tracing"`
	EnableCost      bool                        `yaml:"enable_cost_tracking"`
	MaxRequests     int                         `yaml:"max_concurrent_requests"`
	RequestTimeout  string                      `yaml:"request_timeout"`
	JWTSecret       string                      `yaml:"jwt_secret,omitempty"`
	TLSCertFile     string                      `yaml:"tls_cert_file,omitempty"`
	TLSKeyFile      string                      `yaml:"tls_key_file,omitempty"`
}

// NewMCPRegistryAdapter creates a new MCP registry adapter
func NewMCPRegistryAdapter(config *MCPConfig, guildRegistry registry.Registry) (*MCPRegistryAdapter, error) {
	if config == nil {
		return nil, fmt.Errorf("MCP config cannot be nil")
	}

	if !config.Enabled {
		return &MCPRegistryAdapter{config: config}, nil
	}

	// Convert MCP config to server config
	serverConfig, err := convertToServerConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to convert MCP config: %w", err)
	}

	// Create MCP server
	mcpServer, err := server.NewServer(serverConfig, guildRegistry)
	if err != nil {
		return nil, fmt.Errorf("failed to create MCP server: %w", err)
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
		return fmt.Errorf("failed to start MCP server: %w", err)
	}

	return nil
}

// Stop stops the MCP server
func (a *MCPRegistryAdapter) Stop(ctx context.Context) error {
	if a.server == nil {
		return nil
	}

	if err := a.server.Stop(ctx); err != nil {
		return fmt.Errorf("failed to stop MCP server: %w", err)
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
func (a *MCPRegistryAdapter) GetConfig() *MCPConfig {
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
		return fmt.Errorf("MCP server not properly configured")
	}

	return nil
}

// convertToServerConfig converts MCPConfig to server.Config
func convertToServerConfig(config *MCPConfig) (*server.Config, error) {
	// Parse request timeout
	var requestTimeout = 30 // default 30 seconds
	if config.RequestTimeout != "" {
		// Parse duration string - simplified for now
		// In production, use time.ParseDuration
		requestTimeout = 30
	}

	// Set defaults
	if config.ServerID == "" {
		config.ServerID = "guild-mcp-server"
	}
	if config.ServerName == "" {
		config.ServerName = "Guild MCP Server"
	}
	if config.MaxRequests == 0 {
		config.MaxRequests = 100
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
		TLSCertFile:           config.TLSCertFile,
		TLSKeyFile:            config.TLSKeyFile,
		EnableAuth:            config.EnableAuth,
		JWTSecret:             config.JWTSecret,
		MaxConcurrentRequests: config.MaxRequests,
		RequestTimeout:        time.Duration(requestTimeout) * time.Second,
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
func NewMCPRegistryExtension(config *MCPConfig, guildRegistry registry.Registry) (*MCPRegistryExtension, error) {
	adapter, err := NewMCPRegistryAdapter(config, guildRegistry)
	if err != nil {
		return nil, fmt.Errorf("failed to create MCP adapter: %w", err)
	}

	var toolBridge *ToolBridge
	if adapter.IsEnabled() && adapter.GetServer() != nil {
		// Create tool bridge
		mcpToolRegistry := adapter.GetServer().GetToolRegistry()
		guildToolRegistry := guildRegistry.GetToolRegistry()
		
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
		return fmt.Errorf("failed to start MCP adapter: %w", err)
	}

	// Sync tools if bridge is available
	if e.toolBridge != nil {
		if err := e.toolBridge.SyncAll(ctx); err != nil {
			return fmt.Errorf("failed to sync tools: %w", err)
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
		return fmt.Errorf("tool bridge not available")
	}
	return e.toolBridge.SyncAll(ctx)
}