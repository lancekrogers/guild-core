package integration

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/guild-ventures/guild-core/pkg/mcp/protocol"
	"github.com/guild-ventures/guild-core/pkg/mcp/tools"
	"github.com/guild-ventures/guild-core/pkg/registry"
	"github.com/guild-ventures/guild-core/tools/fs"
	"github.com/guild-ventures/guild-core/tools/shell"
)

// Example demonstrates how to use the ToolBridge to synchronize tools
// between Guild and MCP registries
func Example() {
	// Create the registries
	mcpRegistry := tools.NewMemoryRegistry()
	guildRegistry := registry.NewToolRegistry()

	// Create the tool bridge
	bridge := NewToolBridge(mcpRegistry, guildRegistry)

	// Register some Guild tools
	fileTool := fs.NewFileTool("/tmp")
	shellTool := shell.NewShellTool(shell.ShellToolOptions{
		WorkingDir: "/tmp",
	})

	// Register Guild tools - they'll be available in both registries
	if err := bridge.RegisterGuildTool(fileTool); err != nil {
		log.Printf("Failed to register file tool: %v", err)
	}

	if err := bridge.RegisterGuildTool(shellTool); err != nil {
		log.Printf("Failed to register shell tool: %v", err)
	}

	// Start the bridge to sync existing tools
	ctx := context.Background()
	if err := bridge.Start(ctx); err != nil {
		log.Printf("Failed to start bridge: %v", err)
	}

	// Now both registries have access to the tools
	// Guild can use MCP tools:
	guildTools := guildRegistry.ListTools()
	fmt.Printf("Guild has %d tools available\n", len(guildTools))

	// MCP can use Guild tools:
	mcpTools := mcpRegistry.ListTools()
	fmt.Printf("MCP has %d tools available\n", len(mcpTools))

	// Example: Execute a Guild tool through MCP
	mcpFileTool, err := mcpRegistry.GetTool("guild_file")
	if err == nil {
		result, err := mcpFileTool.Execute(ctx, map[string]interface{}{
			"action": "read",
			"path":   "/tmp/test.txt",
		})
		if err != nil {
			fmt.Printf("Error executing tool: %v\n", err)
		} else {
			fmt.Printf("Tool result: %v\n", result)
		}
	}
}

// ExampleMCPToolRegistration shows how to register an MCP tool
// and make it available to Guild agents
func ExampleMCPToolRegistration() {
	// Create registries and bridge
	mcpRegistry := tools.NewMemoryRegistry()
	guildRegistry := registry.NewToolRegistry()
	bridge := NewToolBridge(mcpRegistry, guildRegistry)

	// Create an MCP tool (e.g., an API client tool)
	apiTool := tools.NewBaseTool(
		"weather_api",
		"Weather API",
		"Get weather information for a location",
		[]string{"api", "weather", "network"},
		protocol.CostProfile{
			FinancialCost: 0.001, // $0.001 per call
			LatencyCost:   time.Second,
		},
		[]protocol.ToolParameter{
			{
				Name:        "location",
				Type:        "string",
				Description: "City name or coordinates",
				Required:    true,
			},
			{
				Name:        "units",
				Type:        "string",
				Description: "Temperature units (celsius/fahrenheit)",
				Required:    false,
				Default:     "celsius",
			},
		},
		[]protocol.ToolParameter{
			{
				Name:        "temperature",
				Type:        "number",
				Description: "Current temperature",
				Required:    true,
			},
			{
				Name:        "description",
				Type:        "string",
				Description: "Weather description",
				Required:    true,
			},
		},
		func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
			// Mock implementation
			location := params["location"].(string)
			units := "celsius"
			if u, ok := params["units"].(string); ok {
				units = u
			}

			return map[string]interface{}{
				"temperature": 22.5,
				"description": fmt.Sprintf("Sunny in %s", location),
				"units":       units,
			}, nil
		},
	)

	// Register the MCP tool - it becomes available to Guild agents
	if err := bridge.RegisterMCPTool(apiTool); err != nil {
		log.Printf("Failed to register weather API tool: %v", err)
		return
	}

	// Now Guild agents can use the weather tool
	guildTool, err := guildRegistry.GetTool("Weather API")
	if err != nil {
		log.Printf("Failed to get tool from Guild registry: %v", err)
		return
	}

	// Guild agents would use it like this:
	ctx := context.Background()
	result, err := guildTool.Execute(ctx, `{"location": "San Francisco", "units": "fahrenheit"}`)
	if err != nil {
		log.Printf("Failed to execute tool: %v", err)
		return
	}

	fmt.Printf("Weather result: %s\n", result.Output)

	// The tool is also tracked with cost information
	// Note: In a real implementation, you'd access cost info through the registry
	fmt.Printf("Tool registered successfully with cost tracking\n")
}
