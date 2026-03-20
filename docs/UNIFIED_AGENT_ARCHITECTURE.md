# Unified Agent Architecture Documentation

## Overview

The Guild Framework uses a unified agent system located in `pkg/agents/` that consolidates all agent-related functionality into a cohesive, well-organized package structure. This document describes the architecture, design patterns, and usage guidelines for the unified agent system.

## Architecture Overview

```
pkg/agents/
├── core/                 # Core agent implementations and interfaces
│   ├── agent.go         # Base agent types and interfaces
│   ├── base_agent.go    # BaseAgent implementation
│   ├── tool_agent.go    # Tool-enabled agent extension
│   ├── executor/        # Task execution logic
│   ├── manager/         # Agent management and coordination
│   └── mocks/           # Mock implementations for testing
├── backstory/           # Agent personality and backstory system
│   ├── manager.go       # Backstory management
│   ├── wizard.go        # Interactive backstory creation
│   └── templates/       # Pre-defined agent templates
└── creation/            # Agent creation and initialization
    └── factory.go       # Agent factory implementations
```

## Core Components

### 1. Agent Interface (`pkg/agents/core/agent.go`)

The base `Agent` interface defines the contract all agents must implement:

```go
type Agent = interfaces.Agent

// Core methods every agent must implement:
- Execute(ctx context.Context, request string) (string, error)
- GetID() string
- GetName() string
- GetType() string
- GetCapabilities() []string
- GetDescription() string
```

### 2. GuildArtisan Interface

The `GuildArtisan` interface extends `Agent` with Guild-specific capabilities:

```go
type GuildArtisan interface {
    Agent
    GetToolRegistry() tools.Registry
    GetCommissionManager() commission.CommissionManager  
    GetLLMClient() providers.LLMClient
    GetMemoryManager() memory.ChainManager
}
```

### 3. ToolAgent Interface

For agents that can execute tools:

```go
type ToolAgent interface {
    Agent
    ExecuteWithTools(ctx context.Context, input string, availableTools []interfaces.ToolDefinition) (response string, toolCalls []interfaces.ToolCall, err error)
    ContinueWithToolResult(ctx context.Context, toolCallID string, result string) (string, error)
}
```

## Implementation Hierarchy

### BaseAgent

- Provides common functionality for all agents
- Manages agent configuration (ID, name, model, temperature, etc.)
- Foundation for specialized agent types

### WorkerAgent  

- Standard implementation of GuildArtisan
- Includes LLM client, memory manager, tool registry
- Supports cost tracking and tool execution
- Primary workhorse for task execution

### ManagerAgent

- Extends WorkerAgent
- Coordinates other agents
- Handles task delegation and orchestration

### BaseToolAgent

- Extends BaseAgent with tool execution capabilities
- Implements the ToolAgent interface
- Handles tool call parsing and execution flow

## Agent Creation and Management

### Agent Factory Pattern

The system uses a factory pattern for agent creation:

```go
// Example: Creating an enhanced agent
creator := agents.NewDefaultAgentCreator()
agentConfig, err := creator.CreateElenaGuildMaster(ctx)
```

### Backstory Integration

Agents can be enhanced with rich backstories:

```go
// Enhance an agent with a specialist template
initializer.EnhanceExistingAgent(ctx, agentID, "elena_guild_master", guildConfig, projectPath)
```

### Pre-defined Templates

The system includes several pre-defined agent templates:

1. **Elena the Guild Master** - Project coordination and management
2. **Marcus the Developer** - Code implementation specialist
3. **Vera the Tester** - Quality assurance and testing
4. **Other Specialists** - Various domain-specific agents

## Key Design Patterns

### 1. Dependency Injection

All agents receive their dependencies through constructors:

- LLM clients
- Memory managers
- Tool registries
- Commission managers

### 2. Context Propagation

Every operation accepts a context for:

- Cancellation support
- Deadline management
- Request tracing
- Observability

### 3. Interface Segregation

Small, focused interfaces allow flexibility:

- `Agent` - Core functionality
- `ToolAgent` - Tool execution
- `GuildArtisan` - Full Guild capabilities

### 4. Cost Management

Built-in cost tracking for:

- Token usage
- API calls
- Tool executions
- Resource consumption

## Usage Examples

### Creating a Basic Agent

```go
func createBasicAgent() *core.WorkerAgent {
    llmClient := providers.NewOpenAIClient(config)
    memoryManager := memory.NewChainManager()
    toolRegistry := tools.NewRegistry()
    
    return core.NewWorkerAgent(
        "agent-001",
        "Assistant",
        llmClient,
        memoryManager,
        toolRegistry,
        nil, // commission manager
        nil, // cost manager
    )
}
```

### Creating an Enhanced Agent Set

```go
func initializeAgents(ctx context.Context) error {
    initializer := agents.NewDefaultInitializer()
    
    // Create default agent set with Elena as manager
    guildConfig, err := initializer.CreateGuildConfigWithElena(ctx, "MyGuild")
    if err != nil {
        return err
    }
    
    // Initialize agents in project
    return initializer.InitializeDefaultAgents(ctx, projectPath)
}
```

### Executing with Tools

```go
func executeWithTools(ctx context.Context, agent core.ToolAgent) error {
    tools := []interfaces.ToolDefinition{
        {Type: "function", Function: interfaces.FunctionDefinition{
            Name: "search_code",
            Description: "Search for code patterns",
            Parameters: map[string]interface{}{
                "pattern": "string",
                "path": "string",
            },
        }},
    }
    
    response, toolCalls, err := agent.ExecuteWithTools(
        ctx,
        "Find all error handling in the codebase",
        tools,
    )
    
    // Process tool calls...
    return err
}
```

## Best Practices

### 1. Always Use Context

```go
// Good
response, err := agent.Execute(ctx, request)

// Bad - no context
response, err := agent.Execute(context.Background(), request)
```

### 2. Handle Cost Limits

```go
agent.SetCostBudget(core.CostTypeTokens, 10000)
if report := agent.GetCostReport(); report["tokens_used"].(float64) > 9000 {
    log.Warn("Approaching token limit")
}
```

### 3. Proper Error Handling

```go
response, err := agent.Execute(ctx, request)
if err != nil {
    if gerror.IsCode(err, gerror.ErrCodeRateLimit) {
        // Handle rate limiting
    }
    return gerror.Wrap(err, gerror.ErrCodeInternal, "agent execution failed").
        WithComponent("mycomponent").
        WithDetails("agent_id", agent.GetID())
}
```

### 4. Resource Cleanup

```go
// If agent has closeable resources
if closer, ok := agent.(io.Closer); ok {
    defer closer.Close()
}
```

## Testing

### Using Mocks

The system provides comprehensive mocks:

```go
func TestAgentExecution(t *testing.T) {
    mockAgent := mocks.NewMockAgent(t)
    mockAgent.On("Execute", mock.Anything, "test input").
        Return("test response", nil)
    
    // Test your code that uses the agent
}
```

### Integration Testing

```go
func TestAgentIntegration(t *testing.T) {
    // Create real agent with test dependencies
    agent := createTestAgent()
    
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    
    response, err := agent.Execute(ctx, "test request")
    require.NoError(t, err)
    assert.Contains(t, response, "expected content")
}
```

## Migration Guide

If you're migrating from the old `pkg/agent` structure:

1. Update imports:

   ```go
   // Old
   import "github.com/guild-framework/guild-core/pkg/agent"
   
   // New
   import "github.com/guild-framework/guild-core/pkg/agents/core"
   ```

2. Update type references:

   ```go
   // Old
   var agent agent.Agent
   
   // New
   var agent core.Agent
   ```

3. Update backstory imports:

   ```go
   // Old
   import "github.com/guild-framework/guild-core/pkg/backstory"
   
   // New
   import "github.com/guild-framework/guild-core/pkg/agents/backstory"
   ```

## Future Enhancements

1. **Multi-Agent Collaboration** - Enhanced coordination between agents
2. **Learning and Adaptation** - Agents that improve over time
3. **Custom Tool Development** - Easier tool creation and registration
4. **Performance Optimization** - Caching and request batching
5. **Advanced Cost Management** - Predictive cost modeling

## Conclusion

The unified agent architecture provides a solid foundation for building intelligent, cost-aware agents that can work independently or collaboratively. By following the patterns and practices outlined in this document, developers can create robust agent-based solutions that integrate seamlessly with the Guild Framework ecosystem.
