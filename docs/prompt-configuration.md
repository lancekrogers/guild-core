# Guild Prompt Configuration Guide

This guide explains how to configure and use Guild's layered prompt management system.

## Overview

Guild provides a sophisticated 6-layer prompt system that allows you to customize AI agent behavior at different levels of granularity. The system follows a hierarchical override pattern where more specific layers take precedence over general ones.

## Prompt Layer Hierarchy

The prompt system consists of 6 layers, listed from lowest to highest priority:

1. **System Layer** - Base framework prompts
2. **Guild Layer** - Organization-wide prompts
3. **Project Layer** - Project-specific prompts
4. **Campaign Layer** - Campaign-specific prompts
5. **Objective Layer** - Task-specific prompts
6. **Runtime Layer** - Dynamic runtime prompts

Higher-numbered layers override lower-numbered layers when conflicts occur.

## Registry Integration

### Getting the Prompt Manager

```go
import (
    "github.com/guild-ventures/guild-core/pkg/registry"
)

// Get prompt manager from registry
reg := registry.NewComponentRegistry()
err := reg.Initialize(ctx, config)
if err != nil {
    return err
}

promptManager, err := reg.GetPromptManager()
if err != nil {
    return err
}
```

### Interface Methods

The prompt manager implements both `Manager` and `LayeredManager` interfaces:

```go
// Basic Manager interface methods
roles, err := promptManager.ListRoles(ctx)
domains, err := promptManager.ListDomains(ctx, "manager")
template, err := promptManager.GetPromptTemplate(ctx, "system", "analyzer")

// LayeredManager interface methods (some may return "not implemented")
layeredPrompt, err := promptManager.BuildLayeredPrompt(ctx, artisanID, sessionID, turnContext)
```

## Configuration in Guild YAML

### Basic Prompt Configuration

```yaml
# guild.yaml
prompts:
  default_format: "xml"  # or "markdown"
  layers:
    system:
      analyzer: |
        You are a system analyzer. Analyze the provided information systematically.
    guild:
      analyzer: |
        You are part of the Guild framework. Follow Guild conventions.
    project:
      analyzer: |
        You are working on project: {{ .ProjectName }}
        Focus on project-specific requirements.
```

### Campaign-Specific Prompts

```yaml
campaigns:
  e-commerce:
    prompts:
      campaign:
        developer: |
          You are building an e-commerce platform.
          Focus on scalability, security, and user experience.
        tester: |
          Test e-commerce functionality thoroughly.
          Pay special attention to payment flows and security.
```

## gRPC Integration

### Setting Custom Prompts

```go
import (
    promptspb "github.com/guild-ventures/guild-core/pkg/grpc/pb/prompts/v1"
)

client := promptspb.NewPromptsServiceClient(conn)

// Set a project-level prompt
_, err := client.SetPrompt(context.Background(), &promptspb.SetPromptRequest{
    PromptId: "developer-agent",
    Content:  "You are a skilled developer working on this project...",
    Layer:    "project",
    Format:   "xml",
})
```

### Getting Assembled Prompts

```go
// Get layered prompt
assembled, err := client.AssemblePrompt(context.Background(), &promptspb.AssemblePromptRequest{
    PromptId: "developer-agent",
    Context: map[string]string{
        "campaign":  "e-commerce",
        "objective": "implement-payment",
    },
})
```

## Format Support

### XML Format

```xml
<prompt layer="system">
    <role>analyzer</role>
    <instructions>
        You are a system analyzer. Your task is to...
    </instructions>
    <constraints>
        - Follow structured analysis
        - Provide clear reasoning
    </constraints>
</prompt>
```

### Markdown Format

```markdown
# System Analyzer Prompt

## Role
You are a system analyzer specialized in...

## Instructions
1. Analyze the provided information
2. Structure your response clearly
3. Provide actionable insights

## Context
- Layer: system
- Domain: analysis
```

## Advanced Features

### Token Optimization

```go
// Optimize prompts for token limits
optimized, err := client.OptimizePrompt(context.Background(), &promptspb.OptimizePromptRequest{
    PromptId:  "developer-agent",
    MaxTokens: 1000,
})

if optimized.TokensSaved > 0 {
    fmt.Printf("Saved %d tokens through optimization\n", optimized.TokensSaved)
}
```

### Dynamic Context

```go
// Set runtime context
turnContext := map[string]interface{}{
    "current_task": "implement authentication",
    "complexity":   "high",
    "deadline":     "2025-06-15",
}

// Context gets injected into prompt templates
prompt, err := promptManager.BuildLayeredPrompt(ctx, artisanID, sessionID, turnContext)
```

## Testing Prompt Configuration

### Unit Testing

```go
func TestPromptConfiguration(t *testing.T) {
    reg := registry.NewComponentRegistry()
    config := registry.Config{
        // ... prompt configuration
    }
    
    err := reg.Initialize(context.Background(), config)
    require.NoError(t, err)
    
    manager, err := reg.GetPromptManager()
    require.NoError(t, err)
    
    // Test prompt retrieval
    roles, err := manager.ListRoles(ctx)
    assert.NoError(t, err)
    assert.Contains(t, roles, "analyzer")
}
```

### Integration Testing

```go
func TestPromptGRPCIntegration(t *testing.T) {
    // Start gRPC server with registry
    server := grpc.NewServerWithRegistry(port, registry)
    
    // Create client and test prompt operations
    client := promptspb.NewPromptsServiceClient(conn)
    
    // Test setting and getting prompts
    _, err := client.SetPrompt(ctx, &promptspb.SetPromptRequest{...})
    assert.NoError(t, err)
}
```

## File-Based Prompt Storage

### Directory Structure

```
.guild/
├── prompts/
│   ├── system/
│   │   ├── analyzer.xml
│   │   └── developer.xml
│   ├── guild/
│   │   └── conventions.md
│   ├── project/
│   │   └── requirements.md
│   └── campaign/
│       └── e-commerce/
│           ├── developer.xml
│           └── tester.xml
```

### Loading from Files

```go
// Load prompts from .guild/prompts directory
err := promptManager.LoadFromDirectory(ctx, ".guild/prompts")
if err != nil {
    return fmt.Errorf("failed to load prompts: %w", err)
}
```

## Best Practices

### Prompt Design

1. **Layer Appropriately**: Put general behavior in system/guild layers, specific requirements in project/campaign layers
2. **Use Clear Role Definitions**: Start each prompt with a clear role statement
3. **Provide Context**: Include relevant project or task context
4. **Set Constraints**: Define clear boundaries and expectations
5. **Test Combinations**: Verify how layers combine in your specific use case

### Performance Optimization

1. **Cache Assembled Prompts**: Avoid re-assembling identical prompts
2. **Use Token Optimization**: Enable automatic token optimization for large prompts
3. **Monitor Usage**: Track prompt performance and token consumption
4. **Lazy Loading**: Load prompts only when needed

### Security Considerations

1. **Validate Inputs**: Sanitize user-provided prompt content
2. **Limit Permissions**: Restrict who can modify system and guild layers
3. **Audit Changes**: Log prompt modifications for security review
4. **Isolate Contexts**: Ensure proper isolation between campaigns and projects

## Migration Guide

### From Legacy System

If migrating from a previous prompt system:

1. **Identify Current Prompts**: Catalog existing prompts and their usage
2. **Map to Layers**: Determine appropriate layer for each prompt
3. **Convert Format**: Transform to XML or Markdown format
4. **Test Integration**: Verify prompts work with new layered system
5. **Update Code**: Replace direct prompt usage with registry calls

### Registry Integration

Replace direct prompt manager creation:

```go
// Old approach
manager := layered.NewLayeredPromptManager(store, formatter)

// New approach
manager, err := registry.GetPromptManager()
```

## Troubleshooting

### Common Issues

1. **"Prompt Manager Not Initialized"**: Ensure registry is initialized before calling GetPromptManager()
2. **"Layer Not Found"**: Verify layer names match exactly (case-sensitive)
3. **"Format Not Supported"**: Check that format is either "xml" or "markdown"
4. **"Template Parse Error"**: Validate template syntax and variable references

### Debug Information

Enable debug logging to troubleshoot prompt issues:

```go
// Enable prompt debugging
logger := observability.NewGuildLogger("prompts", "debug")
promptManager.SetLogger(logger)
```

### Performance Issues

If experiencing slow prompt assembly:

1. Check prompt cache configuration
2. Verify template complexity
3. Monitor token optimization effectiveness
4. Consider prompt size reduction
