# MCP Tool Bridge

The MCP Tool Bridge provides seamless integration between Guild's tool system and the Model Context Protocol (MCP) tool system. This allows Guild agents to use MCP tools and MCP systems to use Guild tools.

## Overview

The tool bridge solves the interface incompatibility between Guild and MCP tools:

- **Guild Tools**: Use a simple string-based input/output interface
- **MCP Tools**: Use structured parameter maps and return arbitrary data

The bridge provides adapters that translate between these interfaces, enabling full interoperability.

## Key Features

- **Bidirectional Tool Sharing**: Guild tools become available to MCP, and MCP tools become available to Guild
- **Automatic Cost Mapping**: MCP cost profiles are converted to Guild's Fibonacci scale (0, 1, 2, 3, 5, 8)
- **Capability Translation**: Tool categories and capabilities are mapped between systems
- **Schema Conversion**: JSON schemas are converted between formats automatically
- **Concurrent Safe**: Thread-safe operations for multi-agent environments

## Usage

### Basic Setup

```go
// Create registries
mcpRegistry := tools.NewMemoryRegistry()
guildRegistry := registry.NewToolRegistry()

// Create and start the bridge
bridge := NewToolBridge(mcpRegistry, guildRegistry)
ctx := context.Background()
if err := bridge.Start(ctx); err != nil {
    log.Fatal(err)
}
```

### Registering Guild Tools for MCP

```go
// Create a Guild tool
fileTool := fs.NewFileTool()

// Register it - becomes available in both registries
if err := bridge.RegisterGuildTool(fileTool); err != nil {
    log.Fatal(err)
}

// MCP can now use the Guild tool
mcpTool, _ := mcpRegistry.GetTool("guild_file")
result, _ := mcpTool.Execute(ctx, map[string]interface{}{
    "action": "read",
    "path": "/tmp/data.txt",
})
```

### Registering MCP Tools for Guild

```go
// Create an MCP tool
apiTool := tools.NewBaseTool(
    "weather_api",
    "Weather API",
    "Get weather information",
    []string{"api", "weather"},
    protocol.CostProfile{
        FinancialCost: 0.001,
        LatencyCost: time.Second,
    },
    parameters,
    returns,
    executorFunc,
)

// Register it - becomes available in both registries
if err := bridge.RegisterMCPTool(apiTool); err != nil {
    log.Fatal(err)
}

// Guild agents can now use the MCP tool
guildTool, _ := guildRegistry.GetTool("Weather API")
result, _ := guildTool.Execute(ctx, `{"location": "NYC"}`)
```

## Cost Mapping

MCP cost profiles are automatically mapped to Guild's Fibonacci scale:

| MCP Cost Profile | Guild Cost Magnitude | Description |
|-----------------|---------------------|-------------|
| $0, <100ms | 0 | Free tools |
| <$0.001, <1s | 1 | Very low cost |
| <$0.01, <5s | 2 | Low cost |
| <$0.1, <30s | 3 | Medium cost |
| <$1, <1min | 5 | High cost |
| >=$1, >=1min | 8 | Very high cost |

## Capability Mapping

The bridge automatically maps between Guild categories and MCP capabilities:

- `file` → `["file", "read", "write", "file_operations"]`
- `web` → `["web", "network", "http", "api"]`
- `code` → `["code", "execution", "analysis", "generation"]`
- `shell` → `["shell", "execution", "system", "command"]`

## Architecture

The bridge uses two adapter types:

1. **GuildToMCPAdapter**: Wraps Guild tools to implement the MCP Tool interface
2. **MCPToGuildAdapter**: Wraps MCP tools to implement the Guild Tool interface

These adapters handle:

- Parameter conversion (string ↔ map[string]interface{})
- Schema translation (Guild schema ↔ MCP parameters)
- Result formatting (ToolResult ↔ arbitrary data)
- Cost profile mapping
- Capability translation

## Testing

The implementation includes comprehensive tests:

```bash
go test ./pkg/mcp/integration -v
```

Tests cover:

- Adapter functionality
- Bidirectional tool registration
- Cost magnitude calculation
- Schema conversion
- Execution flow

## Future Enhancements

- Dynamic capability discovery
- Tool versioning support
- Performance metrics collection
- Advanced cost optimization
- Tool composition capabilities
