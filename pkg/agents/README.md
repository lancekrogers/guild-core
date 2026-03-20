# Enhanced Agent System

This package provides the enhanced agent creation system for Guild Framework, featuring **Elena the Guild Master** and a sophisticated backstory system that makes agents feel like real professionals rather than generic AI tools.

## Overview

The enhanced agent system brings together:

- **Rich Medieval Backstories**: Each agent has a detailed professional background, experience, and personality
- **Intelligent Provider Mapping**: Agents are automatically assigned to optimal LLM providers
- **Personality-Enhanced Prompts**: The backstory system enhances prompts with personality and context
- **Specialist Templates**: Pre-built specialist agents with domain expertise
- **Seamless Integration**: Works with existing Guild configuration system

## Key Features

### Elena the Guild Master

Elena is the flagship agent of this system - a wise and empathetic project coordinator who brings grace and expertise to team leadership.

**Key Characteristics:**

- 18 years of experience leading digital artisan teams
- Rich backstory including previous roles and achievements
- Sophisticated personality with high empathy (10/10) and wisdom (9/10)
- Strategic thinking combined with nurturing leadership style
- Optimized for Claude Code provider for best management responses

**Usage:**

```go
creator := agents.NewDefaultAgentCreator()
elena, err := creator.CreateElenaGuildMaster(ctx)
```

### Enhanced Default Agents

The system includes three core agents:

1. **Elena the Guild Master** (Manager)
   - Provider: Claude Code
   - Focus: Project coordination, team leadership, strategic planning
   - Personality: Empathetic, strategic, nurturing

2. **Marcus the Code Artisan** (Developer)  
   - Provider: Claude Code
   - Focus: Software development, system design, code quality
   - Personality: Precise, creative, collaborative, mentoring

3. **Vera the Quality Guardian** (Specialist)
   - Provider: Anthropic
   - Focus: Quality assurance, testing, bug detection
   - Personality: Meticulous, protective, systematic, analytical

### Specialist Templates

The system leverages Guild's existing specialist template system, including:

- **Security Sentinel** (Sir Gareth the Vigilant)
- **Performance Artisan** (Master Thane Swiftforge)  
- **Frontend Artist** (Lady Aria Dreamweaver)
- **Code Sage** (Elder Kodrin the Wise)
- **Data Mystic** (Oracle Pythia Numberweaver)

## Quick Start

### Create Enhanced Guild

```go
// Create new guild with Elena and enhanced agents
initializer := agents.NewAgentInitializer(promptRegistry)
guildConfig, err := initializer.CreateGuildConfigWithElena(ctx, "my-guild")
```

### Upgrade Existing Guild

```go
// Add Elena to existing guild and enhance agents
err := initializer.UpgradeExistingGuild(ctx, existingConfig, projectPath)
```

### Initialize Project with Enhanced Agents

```go
// Create and save default enhanced agents to project
err := initializer.InitializeDefaultAgents(ctx, projectPath)
```

### Generate Personality-Enhanced Prompts

```go
// Get enhanced prompt with personality and context
enhancedPrompt, err := initializer.GeneratePersonalityPrompt(
    ctx, "elena-guild-master", basePrompt, turnContext)
```

## Integration with Guild Framework

### Backstory System Integration

The enhanced agents integrate seamlessly with Guild's existing backstory system:

- **BackstoryManager**: Manages agent personalities and contexts
- **Layered Prompts**: Enhances prompts with personality layers
- **Specialist Templates**: Reuses existing specialist configurations

### Provider Optimization

Agents are intelligently mapped to optimal providers:

```go
// Elena and Marcus prefer Claude Code for best performance
elena.Provider = "claude_code"
marcus.Provider = "claude_code"

// Vera uses Anthropic for analytical testing work  
vera.Provider = "anthropic"
```

### Configuration Compatibility

Enhanced agents are fully compatible with existing Guild configuration:

- Standard YAML configuration files
- Existing capability and tool systems
- Current gRPC integration
- Backstory and personality enhancement layers

## Architecture

### Key Components

1. **DefaultAgentCreator**: Creates enhanced agents with rich backstories
2. **AgentInitializer**: Manages initialization and integration with existing systems
3. **Backstory Integration**: Leverages existing BackstoryManager for personality enhancement
4. **Provider Mapping**: Intelligent assignment of agents to optimal LLM providers

### File Structure

```
pkg/agents/
├── defaults.go           # Enhanced agent creation
├── initialization.go     # Integration with Guild systems
├── interface.go          # Public interfaces
├── example_usage.go      # Usage examples
├── defaults_test.go      # Comprehensive tests
└── README.md            # This documentation
```

## Examples

See `example_usage.go` for complete examples including:

- Creating enhanced guilds from scratch
- Upgrading existing simple guilds
- Generating personality-enhanced prompts
- Using specialist templates

## Testing

Run tests with:

```bash
go test -v ./pkg/agents/...
```

The test suite covers:

- Agent creation and validation
- Backstory and personality verification
- Provider mapping logic
- Context cancellation handling
- Integration with specialist templates

## Medieval Theme

The enhanced agent system maintains Guild's medieval theme throughout:

- **Guild Ranks**: "Guild Master", "Master Artisan", "Master Guardian"
- **Specialties**: "Team Orchestration", "Code Craftsmanship", "Quality Guardianship"
- **Tools**: "Staff of Coordination", "Hammer of Benchmarks", "Shield of Protection"
- **Philosophy**: Rich philosophical statements about their craft and values

## Future Enhancements

The system is designed for future expansion:

- Additional specialist templates
- Domain-specific agent variants
- Enhanced personality evolution based on interactions
- Advanced provider optimization algorithms
- Integration with Guild's learning and memory systems

## Integration Points

This system integrates with:

- **pkg/backstory**: For personality management
- **pkg/config**: For agent configuration
- **pkg/prompts/layered**: For prompt enhancement  
- **pkg/providers**: For LLM provider optimization
- **internal/ui/init**: For project initialization

The enhanced agent system transforms Guild from a functional framework into an immersive experience where users collaborate with skilled professionals who have rich backgrounds, personalities, and expertise.
