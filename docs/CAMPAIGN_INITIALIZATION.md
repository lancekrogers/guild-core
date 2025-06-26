# Campaign Initialization Guide

## Overview

The `guild init` command creates a complete campaign structure that enables multi-agent AI orchestration for your project. This guide explains the enhanced initialization process and the structure it creates.

## Campaign Structure

When you run `guild init`, it creates the following directory structure:

```
project-root/
├── .campaign/                    # Campaign-specific configuration
│   ├── .hash                    # Unique campaign identifier (16 chars)
│   ├── campaign.yaml            # Campaign configuration
│   ├── socket-registry.yaml     # Daemon socket information
│   ├── agents/                  # Agent configurations
│   │   ├── elena-guild-master.yaml
│   │   ├── marcus-developer.yaml
│   │   └── vera-tester.yaml
│   ├── guilds/                  # Guild configurations
│   │   └── default-guild.yaml
│   ├── memory.db                # SQLite database
│   ├── prompts/                 # Custom prompt templates
│   ├── tools/                   # Project-specific tools
│   └── workspaces/              # Agent workspaces
├── commissions/                 # User-facing commission files
│   └── refined/                 # AI-refined commissions
├── corpus/                      # Project documentation
│   └── index/                   # Vector store indices
└── kanban/                      # Task tracking
```

## Key Components

### Campaign Hash

Each campaign receives a unique 16-character hash identifier generated from:
- Project path
- Current timestamp
- User information

This hash ensures unique socket paths and prevents conflicts between multiple campaigns.

### Campaign Configuration (campaign.yaml)

```yaml
campaign:
  hash: "a1b2c3d4e5f6g7h8"  # Unique identifier
  name: "guild-demo"
  project_name: "my-project"
  project_type: "go"
  created_at: "2025-01-26T10:30:00Z"
  version: "1.0.0"

daemon:
  socket_path: "/tmp/guild-a1b2c3d4e5f6g7h8.sock"
  log_level: "info"

storage:
  database: "memory.db"
  backend: "sqlite"

settings:
  auto_start_daemon: true
  session_timeout: "24h"
  max_agents: 10
```

### Default Agents

Guild creates three default agents:

1. **Elena (Guild Master)** - Manager agent who coordinates the team
2. **Marcus (Developer)** - Worker agent who implements features
3. **Vera (Tester)** - Specialist agent who ensures quality

Each agent has:
- Unique capabilities
- Appropriate tools
- Backstory and personality
- System prompt
- Provider configuration

### Project Type Adaptation

The initialization process detects your project type and adapts agents accordingly:

**Go Projects:**
- Marcus gains tools: `go_test`, `go_build`
- Marcus gains capabilities: `goroutines`, `channels`
- Expertise updated to Go-specific knowledge

**Python Projects:**
- Marcus gains tools: `pytest`, `pip`
- Marcus gains capabilities: `data_analysis`, `machine_learning`
- Expertise updated to Python frameworks

**JavaScript/TypeScript Projects:**
- Marcus gains tools: `npm`, `webpack`
- Marcus gains capabilities: `frontend`, `backend`
- Expertise updated to modern web development

**Rust Projects:**
- Marcus gains tools: `cargo`, `rustfmt`
- Marcus gains capabilities: `memory_safety`, `zero_cost_abstractions`
- Expertise updated to systems programming

### Guild Configuration

The `default-guild.yaml` defines how agents work together:

```yaml
guild:
  name: "my-project"
  description: "my-project Guild - Orchestrating AI agents for development"
  version: "1.0.0"

manager:
  default: "elena-guild-master"

agents:
  - elena-guild-master
  - marcus-developer
  - vera-tester

workflows:
  default: "collaborative"
  available:
    - collaborative
    - sequential
    - parallel

cost_optimization:
  enabled: true
  max_cost: 100.0
  alert_at: 80.0
  currency: "USD"
```

## Initialization Process

1. **Campaign Structure Creation**
   - Creates `.campaign/` directory tree
   - Creates user-facing directories (commissions, corpus, kanban)

2. **Project Type Detection**
   - Analyzes project files (go.mod, package.json, etc.)
   - Determines language and framework

3. **Provider Detection**
   - Checks for API keys in environment
   - Detects available AI providers

4. **Campaign Configuration**
   - Generates unique campaign hash
   - Creates campaign.yaml with project metadata

5. **Agent Creation**
   - Creates Elena, Marcus, and Vera configurations
   - Adapts agents to detected project type

6. **Guild Setup**
   - Creates default guild configuration
   - Sets up workflows and cost optimization

7. **Database Initialization**
   - Creates SQLite database
   - Runs migrations

8. **Socket Registry**
   - Creates socket-registry.yaml
   - Enables daemon communication

## Usage

After initialization:

```bash
# Start chatting with Elena
guild chat

# Start the daemon (optional, auto-starts with chat)
guild serve

# Check status
guild status
```

## Customization

### Adding Custom Agents

Create new agent YAML files in `.campaign/agents/`:

```yaml
id: "custom-agent"
name: "Custom Agent"
type: "specialist"
provider: "openai"
model: "gpt-4"
capabilities:
  - custom_capability
tools:
  - custom_tool
```

### Modifying Guild Configuration

Edit `.campaign/guilds/default-guild.yaml` to:
- Change the default manager
- Add/remove agents
- Adjust cost limits
- Change workflow preferences

### Custom Prompts

Add prompt templates to `.campaign/prompts/` for reusable instructions.

## Best Practices

1. **Version Control**
   - Commit `.campaign/` configuration files
   - Exclude `memory.db` and `workspaces/`

2. **Team Collaboration**
   - Share campaign configuration
   - Each developer uses their own API keys via environment variables

3. **Multiple Projects**
   - Each project gets its own campaign
   - Campaigns are isolated with unique socket paths

4. **Cost Management**
   - Set appropriate cost limits in guild configuration
   - Monitor usage with `guild status`

## Troubleshooting

### Campaign Already Exists

Use `--force` to reinitialize:
```bash
guild init --force
```

### Provider Detection Failed

Ensure API keys are set:
```bash
export ANTHROPIC_API_KEY="your-key"
export OPENAI_API_KEY="your-key"
```

### Database Issues

Remove and reinitialize:
```bash
rm -rf .campaign/memory.db
guild init
```

## Next Steps

1. Create your first commission: `guild commission "Build a REST API"`
2. Monitor progress: `guild kanban`
3. Search documentation: `guild corpus search "authentication"`
4. Customize agents for your workflow