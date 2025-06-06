// Package integration provides bridges between MCP and Guild systems
package integration

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/guild-ventures/guild-core/pkg/gerror"
	"github.com/guild-ventures/guild-core/pkg/mcp/protocol"
	"github.com/guild-ventures/guild-core/pkg/mcp/tools"
	"github.com/guild-ventures/guild-core/pkg/registry"
	basetools "github.com/guild-ventures/guild-core/tools"
)

// ToolBridge synchronizes tools between MCP and Guild registries
type ToolBridge struct {
	mcpRegistry   tools.Registry
	guildRegistry registry.ToolRegistry
	mu            sync.RWMutex
	
	// Adapters for converting between interfaces
	guildToMCP map[string]*GuildToMCPAdapter
	mcpToGuild map[string]*MCPToGuildAdapter
}

// NewToolBridge creates a new tool bridge
func NewToolBridge(mcpRegistry tools.Registry, guildRegistry registry.ToolRegistry) *ToolBridge {
	return &ToolBridge{
		mcpRegistry:   mcpRegistry,
		guildRegistry: guildRegistry,
		guildToMCP:    make(map[string]*GuildToMCPAdapter),
		mcpToGuild:    make(map[string]*MCPToGuildAdapter),
	}
}

// Start starts the tool bridge and performs initial synchronization
func (b *ToolBridge) Start(ctx context.Context) error {
	// Sync existing Guild tools to MCP
	if err := b.syncGuildToMCP(ctx); err != nil {
		return gerror.Wrap(err, gerror.Internal, "mcp_tool_bridge", "sync_tools", "failed to sync Guild tools to MCP")
	}
	
	// Sync existing MCP tools to Guild
	if err := b.syncMCPToGuild(ctx); err != nil {
		return gerror.Wrap(err, gerror.Internal, "mcp_tool_bridge", "sync_tools", "failed to sync MCP tools to Guild")
	}
	
	return nil
}

// Stop stops the tool bridge
func (b *ToolBridge) Stop(ctx context.Context) error {
	// Clean up any resources if needed
	return nil
}

// SyncAll synchronizes all tools between registries
func (b *ToolBridge) SyncAll(ctx context.Context) error {
	if err := b.syncGuildToMCP(ctx); err != nil {
		return err
	}
	return b.syncMCPToGuild(ctx)
}

// RegisterGuildTool registers a Guild tool and makes it available in MCP
func (b *ToolBridge) RegisterGuildTool(tool basetools.Tool) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	
	// Register with Guild registry first
	if err := b.guildRegistry.RegisterTool(tool.Name(), tool); err != nil {
		return gerror.Wrap(err, gerror.Internal, "mcp_tool_bridge", "sync_guild_to_mcp", "failed to register tool with Guild registry")
	}
	
	// Create adapter and register with MCP
	adapter := NewGuildToMCPAdapter(tool)
	b.guildToMCP[tool.Name()] = adapter
	
	if err := b.mcpRegistry.RegisterTool(adapter); err != nil {
		// Rollback Guild registration
		// Note: Guild registry doesn't support removal, so we just clean up our adapter
		delete(b.guildToMCP, tool.Name())
		return gerror.Wrap(err, gerror.Internal, "mcp_tool_bridge", "sync_tools", "failed to register tool with MCP registry")
	}
	
	return nil
}

// RegisterMCPTool registers an MCP tool and makes it available in Guild
func (b *ToolBridge) RegisterMCPTool(tool tools.Tool) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	
	// Register with MCP registry first
	if err := b.mcpRegistry.RegisterTool(tool); err != nil {
		return gerror.Wrap(err, gerror.Internal, "mcp_tool_bridge", "sync_tools", "failed to register tool with MCP registry")
	}
	
	// Create adapter and register with Guild
	adapter := NewMCPToGuildAdapter(tool)
	b.mcpToGuild[tool.ID()] = adapter
	
	// Determine cost magnitude based on MCP cost profile
	costMagnitude := b.calculateCostMagnitude(tool.GetCostProfile())
	
	if err := b.guildRegistry.RegisterToolWithCost(tool.Name(), adapter, costMagnitude, tool.Capabilities()); err != nil {
		// Rollback MCP registration
		b.mcpRegistry.DeregisterTool(tool.ID())
		delete(b.mcpToGuild, tool.ID())
		return gerror.Wrap(err, gerror.Internal, "mcp_tool_bridge", "sync_guild_to_mcp", "failed to register tool with Guild registry")
	}
	
	return nil
}

// syncGuildToMCP syncs existing Guild tools to MCP registry
func (b *ToolBridge) syncGuildToMCP(ctx context.Context) error {
	guildTools := b.guildRegistry.ListTools()
	
	for _, toolName := range guildTools {
		// Skip if already synced
		if _, exists := b.guildToMCP[toolName]; exists {
			continue
		}
		
		tool, err := b.guildRegistry.GetTool(toolName)
		if err != nil {
			continue // Skip tools that can't be retrieved
		}
		
		// Only sync base tools that implement the expected interface
		if baseTool, ok := tool.(basetools.Tool); ok {
			adapter := NewGuildToMCPAdapter(baseTool)
			b.guildToMCP[toolName] = adapter
			
			// Register with MCP, ignore errors for individual tools
			_ = b.mcpRegistry.RegisterTool(adapter)
		}
	}
	
	return nil
}

// syncMCPToGuild syncs existing MCP tools to Guild registry
func (b *ToolBridge) syncMCPToGuild(ctx context.Context) error {
	mcpTools := b.mcpRegistry.ListTools()
	
	for _, tool := range mcpTools {
		// Skip if already synced
		if _, exists := b.mcpToGuild[tool.ID()]; exists {
			continue
		}
		
		adapter := NewMCPToGuildAdapter(tool)
		b.mcpToGuild[tool.ID()] = adapter
		
		// Calculate cost magnitude from MCP cost profile
		costMagnitude := b.calculateCostMagnitude(tool.GetCostProfile())
		
		// Register with Guild, ignore errors for individual tools
		_ = b.guildRegistry.RegisterToolWithCost(tool.Name(), adapter, costMagnitude, tool.Capabilities())
	}
	
	return nil
}

// calculateCostMagnitude converts MCP cost profile to Guild's Fibonacci scale
func (b *ToolBridge) calculateCostMagnitude(profile protocol.CostProfile) int {
	// Free tools
	if profile.FinancialCost == 0 && profile.LatencyCost < time.Millisecond*100 {
		return 0
	}
	
	// Map based on financial cost and latency
	if profile.FinancialCost < 0.001 && profile.LatencyCost < time.Second {
		return 1 // Very low cost
	} else if profile.FinancialCost < 0.01 && profile.LatencyCost < time.Second*5 {
		return 2 // Low cost
	} else if profile.FinancialCost < 0.1 && profile.LatencyCost < time.Second*30 {
		return 3 // Medium cost
	} else if profile.FinancialCost < 1.0 && profile.LatencyCost < time.Minute {
		return 5 // High cost
	} else {
		return 8 // Very high cost
	}
}

// GuildToMCPAdapter adapts a Guild tool to the MCP Tool interface
type GuildToMCPAdapter struct {
	guildTool basetools.Tool
	id        string
}

// NewGuildToMCPAdapter creates a new adapter for Guild tools
func NewGuildToMCPAdapter(tool basetools.Tool) *GuildToMCPAdapter {
	return &GuildToMCPAdapter{
		guildTool: tool,
		id:        fmt.Sprintf("guild_%s", tool.Name()),
	}
}

// ID returns the unique tool identifier
func (a *GuildToMCPAdapter) ID() string {
	return a.id
}

// Name returns the human-readable tool name
func (a *GuildToMCPAdapter) Name() string {
	return a.guildTool.Name()
}

// Description returns the tool description
func (a *GuildToMCPAdapter) Description() string {
	return a.guildTool.Description()
}

// Capabilities returns the tool's capabilities
func (a *GuildToMCPAdapter) Capabilities() []string {
	// Map Guild categories to capabilities
	category := a.guildTool.Category()
	capabilities := []string{category}
	
	// Add common capabilities based on category
	switch category {
	case "file":
		capabilities = append(capabilities, "read", "write", "file_operations")
	case "web":
		capabilities = append(capabilities, "network", "http", "api")
	case "code":
		capabilities = append(capabilities, "execution", "analysis", "generation")
	case "shell":
		capabilities = append(capabilities, "execution", "system", "command")
	}
	
	return capabilities
}

// Execute executes the tool with given parameters
func (a *GuildToMCPAdapter) Execute(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	// Convert params to JSON string for Guild tool
	inputJSON, err := json.Marshal(params)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.Internal, "mcp_tool_bridge", "convert_schema", "failed to marshal parameters")
	}
	
	// Execute Guild tool
	result, err := a.guildTool.Execute(ctx, string(inputJSON))
	if err != nil {
		return nil, err
	}
	
	// Return the result as a structured response
	return map[string]interface{}{
		"output":   result.Output,
		"success":  result.Success,
		"error":    result.Error,
		"metadata": result.Metadata,
	}, nil
}

// HealthCheck checks if the tool is healthy
func (a *GuildToMCPAdapter) HealthCheck() error {
	// Simple health check - try to get schema
	if schema := a.guildTool.Schema(); schema == nil {
		return gerror.New(gerror.InvalidArgument, "mcp_tool_bridge", "validate_schema", "tool %s has no schema", a.guildTool.Name())
	}
	return nil
}

// GetCostProfile returns the tool's cost profile
func (a *GuildToMCPAdapter) GetCostProfile() protocol.CostProfile {
	// Map Guild tool categories to approximate cost profiles
	profile := protocol.CostProfile{
		FinancialCost: 0, // Default to free
		LatencyCost:   time.Millisecond * 100, // Default 100ms
	}
	
	// Adjust based on category
	switch a.guildTool.Category() {
	case "web", "api":
		profile.LatencyCost = time.Second
		profile.FinancialCost = 0.001 // Small API cost
	case "shell", "execution":
		profile.LatencyCost = time.Millisecond * 500
		profile.ComputeCost = 0.1
	}
	
	return profile
}

// GetParameters returns the tool's parameter definitions
func (a *GuildToMCPAdapter) GetParameters() []protocol.ToolParameter {
	schema := a.guildTool.Schema()
	if schema == nil {
		return nil
	}
	
	// Extract parameters from JSON schema
	var params []protocol.ToolParameter
	
	if properties, ok := schema["properties"].(map[string]interface{}); ok {
		required := make(map[string]bool)
		if reqList, ok := schema["required"].([]interface{}); ok {
			for _, req := range reqList {
				if reqStr, ok := req.(string); ok {
					required[reqStr] = true
				}
			}
		}
		
		for name, prop := range properties {
			if propMap, ok := prop.(map[string]interface{}); ok {
				param := protocol.ToolParameter{
					Name:        name,
					Type:        getStringValue(propMap, "type"),
					Description: getStringValue(propMap, "description"),
					Required:    required[name],
					Default:     propMap["default"],
				}
				params = append(params, param)
			}
		}
	}
	
	return params
}

// GetReturns returns the tool's return value definitions
func (a *GuildToMCPAdapter) GetReturns() []protocol.ToolParameter {
	// Guild tools don't define return schemas, so we provide a standard output
	return []protocol.ToolParameter{
		{
			Name:        "output",
			Type:        "string",
			Description: "The tool's output",
			Required:    true,
		},
		{
			Name:        "success",
			Type:        "boolean",
			Description: "Whether the tool execution was successful",
			Required:    true,
		},
		{
			Name:        "error",
			Type:        "string",
			Description: "Error message if execution failed",
			Required:    false,
		},
		{
			Name:        "metadata",
			Type:        "object",
			Description: "Additional metadata from the tool",
			Required:    false,
		},
	}
}

// MCPToGuildAdapter adapts an MCP tool to the Guild Tool interface
type MCPToGuildAdapter struct {
	mcpTool tools.Tool
}

// NewMCPToGuildAdapter creates a new adapter for MCP tools
func NewMCPToGuildAdapter(tool tools.Tool) *MCPToGuildAdapter {
	return &MCPToGuildAdapter{
		mcpTool: tool,
	}
}

// Name returns the name of the tool
func (a *MCPToGuildAdapter) Name() string {
	return a.mcpTool.Name()
}

// Description returns a description of what the tool does
func (a *MCPToGuildAdapter) Description() string {
	return a.mcpTool.Description()
}

// Schema returns the JSON schema for the tool's input parameters
func (a *MCPToGuildAdapter) Schema() map[string]interface{} {
	// Convert MCP parameters to JSON schema
	properties := make(map[string]interface{})
	required := []string{}
	
	for _, param := range a.mcpTool.GetParameters() {
		propDef := map[string]interface{}{
			"type":        param.Type,
			"description": param.Description,
		}
		if param.Default != nil {
			propDef["default"] = param.Default
		}
		properties[param.Name] = propDef
		
		if param.Required {
			required = append(required, param.Name)
		}
	}
	
	schema := map[string]interface{}{
		"type":       "object",
		"properties": properties,
	}
	
	if len(required) > 0 {
		schema["required"] = required
	}
	
	return schema
}

// Execute runs the tool with the given input and returns the result
func (a *MCPToGuildAdapter) Execute(ctx context.Context, input string) (*basetools.ToolResult, error) {
	// Parse input JSON to map
	var params map[string]interface{}
	if err := json.Unmarshal([]byte(input), &params); err != nil {
		return nil, gerror.Wrap(err, gerror.InvalidArgument, "mcp_tool_bridge", "execute", "failed to parse input JSON")
	}
	
	// Execute MCP tool
	result, err := a.mcpTool.Execute(ctx, params)
	if err != nil {
		return basetools.NewToolResult("", nil, err, nil), nil
	}
	
	// Convert result to string output
	var output string
	switch v := result.(type) {
	case string:
		output = v
	case map[string]interface{}:
		// Extract output field if present
		if out, ok := v["output"].(string); ok {
			output = out
		} else {
			// Otherwise, marshal the whole result
			jsonOut, _ := json.MarshalIndent(v, "", "  ")
			output = string(jsonOut)
		}
	default:
		// Marshal any other type
		jsonOut, _ := json.MarshalIndent(result, "", "  ")
		output = string(jsonOut)
	}
	
	// Extract metadata if available
	metadata := make(map[string]string)
	if resultMap, ok := result.(map[string]interface{}); ok {
		if meta, ok := resultMap["metadata"].(map[string]interface{}); ok {
			for k, v := range meta {
				if str, ok := v.(string); ok {
					metadata[k] = str
				}
			}
		}
	}
	
	return basetools.NewToolResult(output, metadata, nil, nil), nil
}

// Examples returns a list of example inputs for the tool
func (a *MCPToGuildAdapter) Examples() []string {
	// Generate examples from parameter schema
	params := a.mcpTool.GetParameters()
	if len(params) == 0 {
		return []string{`{}`}
	}
	
	example := make(map[string]interface{})
	for _, param := range params {
		switch param.Type {
		case "string":
			example[param.Name] = fmt.Sprintf("example_%s", param.Name)
		case "number", "integer":
			example[param.Name] = 42
		case "boolean":
			example[param.Name] = true
		case "array":
			example[param.Name] = []interface{}{"item1", "item2"}
		case "object":
			example[param.Name] = map[string]interface{}{"key": "value"}
		}
	}
	
	jsonExample, _ := json.MarshalIndent(example, "", "  ")
	return []string{string(jsonExample)}
}

// Category returns the category of the tool
func (a *MCPToGuildAdapter) Category() string {
	// Map first capability to category
	caps := a.mcpTool.Capabilities()
	if len(caps) > 0 {
		// Common capability to category mappings
		switch caps[0] {
		case "file_operations", "read", "write":
			return "file"
		case "network", "http", "api":
			return "web"
		case "execution", "system", "command":
			return "shell"
		case "analysis", "generation":
			return "code"
		default:
			return caps[0]
		}
	}
	return "general"
}

// RequiresAuth returns whether the tool requires authentication
func (a *MCPToGuildAdapter) RequiresAuth() bool {
	// Check if financial cost > 0 as a proxy for requiring auth
	profile := a.mcpTool.GetCostProfile()
	return profile.FinancialCost > 0
}

// Helper function to safely extract string values from interface maps
func getStringValue(m map[string]interface{}, key string) string {
	if val, ok := m[key].(string); ok {
		return val
	}
	return ""
}