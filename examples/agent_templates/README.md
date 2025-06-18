# Agent Template Examples

This directory contains examples of using the lightweight agent template system to quickly generate agent configurations without requiring extensive backstory details.

## Quick Start

The agent template system allows you to create agents with just the essential fields:

```go
generator := setup.NewAgentTemplateGenerator()

// Use a built-in template
template, _ := generator.GetTemplate("claude-code-developer")
err := generator.GenerateAgentFile(projectPath, template)

// Or create a custom minimal agent
custom := generator.CreateCustomTemplate(
    "my-agent",           // ID
    "My Custom Agent",    // Name
    "worker",            // Type
    "openai",            // Provider
    "gpt-4",             // Model
    "Handles tasks",     // Description
    []string{"coding"},  // Capabilities
)
err := generator.GenerateAgentFile(projectPath, custom)
```

## Built-in Templates

The system includes pre-configured templates for common providers:

### Claude Code Templates
- `claude-code-manager` - Strategic manager for coordinating tasks
- `claude-code-developer` - Full-stack developer
- `claude-code-architect` - System architect for complex designs

### Ollama Templates (Local Models)
- `ollama-coder` - Local coding assistant (privacy-sensitive)
- `ollama-analyst` - Local data analyst

### OpenAI Templates
- `openai-developer` - Versatile GPT-4 developer
- `openai-creative` - Creative specialist

### Anthropic Templates
- `anthropic-researcher` - Deep research specialist

### Generic Templates
- `generic-manager` - Basic manager (requires provider/model)
- `generic-worker` - Basic worker (requires provider/model)

## Quick Setup

For the fastest setup, use the quick setup method:

```go
generator := setup.NewAgentTemplateGenerator()

// Creates a manager and worker with specified models
err := generator.QuickSetup(
    projectPath,
    "openai",           // Provider
    "gpt-4",           // Manager model
    "gpt-3.5-turbo",   // Worker model
)
```

This creates:
- `.guild/agents/manager.yml` - A basic manager agent
- `.guild/agents/worker-1.yml` - A basic worker agent

## Generated File Example

A minimal agent configuration looks like:

```yaml
id: my-agent
name: My Custom Agent
type: worker
provider: openai
model: gpt-4
description: Handles general development tasks
capabilities:
  - coding
  - testing
  - documentation
max_tokens: 3000      # Auto-set based on type
temperature: 0.4      # Auto-set based on type
system_prompt: |
  You are My Custom Agent, a worker agent. Handles general development tasks.
  Your capabilities include: coding, testing, documentation.
  Approach tasks methodically and communicate clearly.
```

## Optional Backstory Fields

You can optionally add personality without the full backstory system:

```go
template := setup.AgentTemplate{
    ID:           "wise-coder",
    Name:         "Wise Coder",
    Type:         "specialist",
    Provider:     "anthropic",
    Model:        "claude-3-opus",
    Description:  "Experienced coding specialist",
    Capabilities: []string{"coding", "architecture", "mentoring"},
    
    // Optional - adds flavor without complexity
    Experience:   "15 years building distributed systems",
    Expertise:    "Microservices, event-driven architecture",
    Philosophy:   "Simple solutions to complex problems",
}
```

## Integration with Guild Config

The generated agent files integrate seamlessly with the guild configuration:

```yaml
# .guild/guild.yaml
name: My Development Guild
agents:
  - !include agents/manager.yml
  - !include agents/worker-1.yml
  - !include agents/wise-coder.yml
```

## Benefits

1. **Minimal Configuration**: Only requires essential fields
2. **Smart Defaults**: Automatically sets appropriate values for token limits and temperature
3. **Optional Complexity**: Can add backstory elements only when needed
4. **Provider Templates**: Pre-configured for common providers
5. **Quick Setup**: Get running with two agents in seconds

## Migration from Rich Backstories

If you have existing agents with rich backstories, they continue to work unchanged. The lightweight templates are an alternative, not a replacement. You can:

- Use lightweight templates for quick prototypes
- Use rich backstories for production agents with personality
- Mix both approaches in the same guild
- Start lightweight and add backstory details later