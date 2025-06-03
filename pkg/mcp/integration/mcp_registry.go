// Package integration provides MCP integration for the Guild registry system
package integration

import (
	"context"
	"fmt"

	mcpconfig "github.com/guild-ventures/guild-core/pkg/mcp/config"
	"github.com/guild-ventures/guild-core/pkg/mcp/transport"
	"github.com/guild-ventures/guild-core/pkg/registry"
)

// MCPRegistry extends the component registry with MCP support
type MCPRegistry interface {
	// GetMCPExtension returns the MCP extension
	GetMCPExtension() *MCPRegistryExtension
	
	// StartMCP starts the MCP server
	StartMCP(ctx context.Context) error
	
	// StopMCP stops the MCP server
	StopMCP(ctx context.Context) error
	
	// SyncMCPTools synchronizes tools between MCP and Guild registries
	SyncMCPTools(ctx context.Context) error
	
	// IsMCPEnabled returns whether MCP is enabled
	IsMCPEnabled() bool
}

// MCPRegistryMixin provides MCP functionality for registries
type MCPRegistryMixin struct {
	mcpExtension *MCPRegistryExtension
	mcpConfig    *mcpconfig.MCPConfig
}

// NewMCPRegistryMixin creates a new MCP registry mixin
func NewMCPRegistryMixin(config *mcpconfig.MCPConfig, baseRegistry registry.ComponentRegistry) (*MCPRegistryMixin, error) {
	if config == nil {
		// MCP disabled
		return &MCPRegistryMixin{
			mcpConfig: &mcpconfig.MCPConfig{Enabled: false},
		}, nil
	}

	extension, err := NewMCPRegistryExtension(config, baseRegistry)
	if err != nil {
		return nil, fmt.Errorf("failed to create MCP extension: %w", err)
	}

	return &MCPRegistryMixin{
		mcpExtension: extension,
		mcpConfig:    config,
	}, nil
}

// GetMCPExtension returns the MCP extension
func (m *MCPRegistryMixin) GetMCPExtension() *MCPRegistryExtension {
	return m.mcpExtension
}

// StartMCP starts the MCP server
func (m *MCPRegistryMixin) StartMCP(ctx context.Context) error {
	if m.mcpExtension == nil {
		return fmt.Errorf("MCP not configured")
	}

	return m.mcpExtension.Initialize(ctx)
}

// StopMCP stops the MCP server
func (m *MCPRegistryMixin) StopMCP(ctx context.Context) error {
	if m.mcpExtension == nil {
		return nil // MCP not configured, nothing to stop
	}

	return m.mcpExtension.Shutdown(ctx)
}

// SyncMCPTools synchronizes tools between registries
func (m *MCPRegistryMixin) SyncMCPTools(ctx context.Context) error {
	if m.mcpExtension == nil {
		return fmt.Errorf("MCP not configured")
	}

	return m.mcpExtension.SyncTools(ctx)
}

// IsMCPEnabled returns whether MCP is enabled
func (m *MCPRegistryMixin) IsMCPEnabled() bool {
	return m.mcpConfig != nil && m.mcpConfig.Enabled
}

// ExtendedComponentRegistry extends ComponentRegistry with MCP support
type ExtendedComponentRegistry struct {
	registry.ComponentRegistry
	*MCPRegistryMixin
}

// NewExtendedComponentRegistry creates a new extended component registry with MCP support
func NewExtendedComponentRegistry(ctx context.Context, mcpConfig *mcpconfig.MCPConfig) (*ExtendedComponentRegistry, error) {
	// Create base component registry
	baseRegistry := registry.NewComponentRegistry()

	// Create MCP mixin
	mcpMixin, err := NewMCPRegistryMixin(mcpConfig, baseRegistry)
	if err != nil {
		return nil, fmt.Errorf("failed to create MCP mixin: %w", err)
	}

	extended := &ExtendedComponentRegistry{
		ComponentRegistry: baseRegistry,
		MCPRegistryMixin:  mcpMixin,
	}

	// Initialize MCP if enabled
	if mcpMixin.IsMCPEnabled() {
		if err := mcpMixin.StartMCP(ctx); err != nil {
			return nil, fmt.Errorf("failed to start MCP: %w", err)
		}
	}

	return extended, nil
}

// Close extends the base Close method to include MCP cleanup
func (r *ExtendedComponentRegistry) Close(ctx context.Context) error {
	// Stop MCP first
	if err := r.StopMCP(ctx); err != nil {
		// Log error but continue with base cleanup
		fmt.Printf("Warning: failed to stop MCP: %v\n", err)
	}

	// Call base Close if it exists
	if closer, ok := r.ComponentRegistry.(interface{ Close(context.Context) error }); ok {
		return closer.Close(ctx)
	}

	return nil
}

// Helper functions for easy MCP integration

// WithMCP returns a configuration option to enable MCP
func WithMCP(config *mcpconfig.MCPConfig) func(*ExtendedComponentRegistry) error {
	return func(r *ExtendedComponentRegistry) error {
		if config == nil {
			return nil
		}

		// Update MCP configuration
		if r.MCPRegistryMixin != nil && r.MCPRegistryMixin.mcpConfig != nil {
			*r.MCPRegistryMixin.mcpConfig = *config
		}

		return nil
	}
}

// WithMCPDefaults returns a default MCP configuration
func WithMCPDefaults() *mcpconfig.MCPConfig {
	return &mcpconfig.MCPConfig{
		Enabled:        true,
		ServerID:       "guild-mcp-server",
		ServerName:     "Guild MCP Server",
		EnableAuth:     false,
		EnableTLS:      false,
		EnableMetrics:  true,
		EnableTracing:  false,
		EnableCost:     true,
		Transport: &transport.TransportConfig{
			Type:    "memory",
			Address: "memory://default",
		},
	}
}

// Example usage helper
func ExampleMCPRegistry(ctx context.Context) (*ExtendedComponentRegistry, error) {
	config := WithMCPDefaults()
	return NewExtendedComponentRegistry(ctx, config)
}