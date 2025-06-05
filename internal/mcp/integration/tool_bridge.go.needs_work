// Package integration provides bridges between MCP and Guild systems
package integration

import (
	"context"
	"fmt"
	"sync"

	"github.com/guild-ventures/guild-core/pkg/mcp/protocol"
	"github.com/guild-ventures/guild-core/pkg/mcp/tools"
	"github.com/guild-ventures/guild-core/pkg/registry"
	guildtools "github.com/guild-ventures/guild-core/tools"
)

// ToolBridge synchronizes tools between MCP and Guild registries
type ToolBridge struct {
	mcpRegistry   tools.Registry
	guildRegistry registry.ToolRegistry
	mu            sync.RWMutex
	syncing       map[string]bool // Prevent sync loops
}

// NewToolBridge creates a new tool bridge
func NewToolBridge(mcpRegistry tools.Registry, guildRegistry registry.ToolRegistry) *ToolBridge {
	return &ToolBridge{
		mcpRegistry:   mcpRegistry,
		guildRegistry: guildRegistry,
		syncing:       make(map[string]bool),
	}
}

// SyncMCPToGuild synchronizes an MCP tool to the Guild registry
func (b *ToolBridge) SyncMCPToGuild(ctx context.Context, mcpTool tools.Tool) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	toolID := mcpTool.ID()
	if b.syncing[toolID] {
		return nil // Already syncing
	}
	b.syncing[toolID] = true
	defer delete(b.syncing, toolID)

	// Convert MCP tool to Guild tool
	guildTool := &MCPToolAdapter{
		mcpTool: mcpTool,
		bridge:  b,
	}

	// Register in Guild registry
	if err := b.guildRegistry.Register(ctx, guildTool); err != nil {
		return fmt.Errorf("failed to register MCP tool in Guild registry: %w", err)
	}

	return nil
}

// SyncGuildToMCP synchronizes a Guild tool to the MCP registry
func (b *ToolBridge) SyncGuildToMCP(ctx context.Context, guildTool guildtools.Tool) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	toolID := guildTool.ID()
	if b.syncing[toolID] {
		return nil // Already syncing
	}
	b.syncing[toolID] = true
	defer delete(b.syncing, toolID)

	// Convert Guild tool to MCP tool
	mcpTool := &GuildToolAdapter{
		guildTool: guildTool,
		bridge:    b,
	}

	// Register in MCP registry
	if err := b.mcpRegistry.RegisterTool(mcpTool); err != nil {
		return fmt.Errorf("failed to register Guild tool in MCP registry: %w", err)
	}

	return nil
}

// SyncAll synchronizes all tools bidirectionally
func (b *ToolBridge) SyncAll(ctx context.Context) error {
	// Sync MCP tools to Guild
	mcpTools := b.mcpRegistry.ListTools()
	for _, mcpTool := range mcpTools {
		if err := b.SyncMCPToGuild(ctx, mcpTool); err != nil {
			return fmt.Errorf("failed to sync MCP tool %s: %w", mcpTool.ID(), err)
		}
	}

	// Sync Guild tools to MCP
	guildTools, err := b.guildRegistry.ListTools(ctx)
	if err != nil {
		return fmt.Errorf("failed to list Guild tools: %w", err)
	}

	for _, guildTool := range guildTools {
		if err := b.SyncGuildToMCP(ctx, guildTool); err != nil {
			return fmt.Errorf("failed to sync Guild tool %s: %w", guildTool.ID(), err)
		}
	}

	return nil
}

// MCPToolAdapter adapts an MCP tool to Guild tool interface
type MCPToolAdapter struct {
	mcpTool tools.Tool
	bridge  *ToolBridge
}

// ID returns the tool ID
func (a *MCPToolAdapter) ID() string {
	return a.mcpTool.ID()
}

// Name returns the tool name
func (a *MCPToolAdapter) Name() string {
	return a.mcpTool.Name()
}

// Description returns the tool description
func (a *MCPToolAdapter) Description() string {
	return a.mcpTool.Description()
}

// Execute executes the tool
func (a *MCPToolAdapter) Execute(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	return a.mcpTool.Execute(ctx, args)
}

// Validate validates tool parameters (if Guild tools have this method)
func (a *MCPToolAdapter) Validate(args map[string]interface{}) error {
	// Convert MCP parameters to validation
	params := a.mcpTool.GetParameters()
	for _, param := range params {
		if param.Required {
			if _, exists := args[param.Name]; !exists {
				return fmt.Errorf("required parameter %s missing", param.Name)
			}
		}
	}
	return nil
}

// GetCapabilities returns tool capabilities
func (a *MCPToolAdapter) GetCapabilities() []string {
	return a.mcpTool.Capabilities()
}

// GuildToolAdapter adapts a Guild tool to MCP tool interface
type GuildToolAdapter struct {
	guildTool guildtools.Tool
	bridge    *ToolBridge
}

// ID returns the tool ID
func (a *GuildToolAdapter) ID() string {
	return a.guildTool.ID()
}

// Name returns the tool name
func (a *GuildToolAdapter) Name() string {
	return a.guildTool.Name()
}

// Description returns the tool description
func (a *GuildToolAdapter) Description() string {
	return a.guildTool.Description()
}

// Capabilities returns capabilities (convert from Guild tool capabilities)
func (a *GuildToolAdapter) Capabilities() []string {
	// If Guild tool has GetCapabilities method, use it
	if capTool, ok := a.guildTool.(interface{ GetCapabilities() []string }); ok {
		return capTool.GetCapabilities()
	}
	// Otherwise, derive from tool type or return generic capabilities
	return []string{"guild-tool"}
}

// Execute executes the tool
func (a *GuildToolAdapter) Execute(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	return a.guildTool.Execute(ctx, params)
}

// HealthCheck checks if the tool is healthy
func (a *GuildToolAdapter) HealthCheck() error {
	// If Guild tool has health check, use it
	if healthTool, ok := a.guildTool.(interface{ HealthCheck() error }); ok {
		return healthTool.HealthCheck()
	}
	// Otherwise, assume healthy
	return nil
}

// GetCostProfile returns the cost profile
func (a *GuildToolAdapter) GetCostProfile() protocol.CostProfile {
	// If Guild tool has cost profile, convert it
	if costTool, ok := a.guildTool.(interface{ GetCostProfile() interface{} }); ok {
		// Convert Guild cost profile to MCP cost profile
		_ = costTool.GetCostProfile()
		// Return default for now - would need to know Guild's cost structure
	}
	
	return protocol.CostProfile{
		ComputeCost:   0.001,
		MemoryCost:    1024,
		FinancialCost: 0.0001,
	}
}

// GetParameters returns parameter definitions
func (a *GuildToolAdapter) GetParameters() []protocol.ToolParameter {
	// If Guild tool has parameter definitions, convert them
	if paramTool, ok := a.guildTool.(interface{ GetParameters() []interface{} }); ok {
		guildParams := paramTool.GetParameters()
		var mcpParams []protocol.ToolParameter
		
		// Convert Guild parameters to MCP parameters
		for _, param := range guildParams {
			// This would need to know Guild's parameter structure
			// For now, return basic parameter
			mcpParams = append(mcpParams, protocol.ToolParameter{
				Name:        "input",
				Type:        "object",
				Description: "Tool input parameters",
				Required:    false,
			})
		}
		
		return mcpParams
	}
	
	// Default parameters
	return []protocol.ToolParameter{
		{
			Name:        "input",
			Type:        "object",
			Description: "Tool input parameters",
			Required:    false,
		},
	}
}

// GetReturns returns return value definitions
func (a *GuildToolAdapter) GetReturns() []protocol.ToolParameter {
	return []protocol.ToolParameter{
		{
			Name:        "result",
			Type:        "object",
			Description: "Tool execution result",
		},
	}
}