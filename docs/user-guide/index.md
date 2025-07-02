# Guild Framework User Guide

Welcome to Guild - the multi-agent orchestration framework that transforms how you build software. This guide will help you master Guild's powerful features and integrate it into your development workflow.

## Table of Contents

1. [Getting Started](#getting-started)
2. [Core Concepts](#core-concepts)
3. [Installation & Setup](#installation)
4. [Your First Commission](#first-commission)
5. [Working with Artisans](#artisans)
6. [Advanced Features](#advanced)
7. [Best Practices](#best-practices)
8. [Troubleshooting](#troubleshooting)

## Getting Started

### What is Guild?

Guild is a multi-agent AI orchestration framework that enables teams of specialized AI agents (called Artisans) to work together on complex software projects. Unlike single-agent tools, Guild coordinates multiple artisans with different capabilities, allowing them to collaborate like a real development team.

### Key Features

- **Multi-Artisan Orchestration**: Multiple specialized artisans working in parallel
- **Commission-Based Workflow**: Define projects as "commissions" for your guild
- **Visual Task Tracking**: Built-in kanban board for monitoring progress
- **Knowledge Management**: Artisans learn from your codebase and documentation
- **Tool Permissions**: Fine-grained control over what each artisan can do
- **Session Persistence**: Resume conversations seamlessly

### System Requirements

- **OS**: macOS, Linux, Windows (WSL2)
- **Memory**: 4GB RAM minimum, 8GB recommended
- **Storage**: 1GB free space
- **Dependencies**: Git, Go 1.21+ (optional for development)

## Core Concepts

### The Guild Metaphor

Guild uses a medieval guild metaphor to make multi-agent systems intuitive:

- **Guild**: Your team of AI artisans working together
- **Artisans**: Individual AI agents with specialized skills
- **Commissions**: Projects or tasks given to the guild
- **Workshop**: The kanban board where work is tracked
- **Archives**: The knowledge base that artisans learn from

### Artisan Roles

#### Elena - The Foreman
Elena is your project manager and coordinator. She:
- Plans and breaks down projects
- Assigns tasks to other artisans
- Monitors progress and resolves blockers
- Communicates status updates

#### Marcus - The Codesmith
Marcus is your senior developer. He:
- Implements features and fixes bugs
- Writes clean, maintainable code
- Reviews and refactors existing code
- Handles complex technical challenges

#### Vera - The Inspector
Vera is your QA engineer. She:
- Writes and runs tests
- Validates code quality
- Identifies edge cases
- Ensures requirements are met

### Commission Workflow

1. **Define**: Describe your project to Elena
2. **Plan**: Elena creates a detailed plan
3. **Execute**: Artisans work on assigned tasks
4. **Review**: Human reviews blocked tasks
5. **Complete**: Delivered working software

## Installation & Setup

### Quick Start (30 seconds)

```bash
# Install Guild
curl -sSL https://guild.dev/install.sh | sh

# Initialize your first project
guild init my-project
cd my-project

# Start chatting
guild chat
```

### Detailed Installation

#### macOS
```bash
# Using Homebrew
brew tap guild-framework/guild
brew install guild

# Or download directly
curl -L https://github.com/guild-framework/releases/latest/download/guild-darwin-arm64.tar.gz | tar xz
sudo mv guild /usr/local/bin/
```

#### Linux
```bash
# Download and install
curl -L https://github.com/guild-framework/releases/latest/download/guild-linux-amd64.tar.gz | tar xz
sudo mv guild /usr/local/bin/

# Add to PATH if needed
echo 'export PATH=$PATH:/usr/local/bin' >> ~/.bashrc
source ~/.bashrc
```

#### Configuration

Guild stores configuration in `.guild/guild.yaml`:

```yaml
# API Keys
providers:
  openai:
    api_key: ${OPENAI_API_KEY}
  anthropic:
    api_key: ${ANTHROPIC_API_KEY}

# Artisan Configuration  
artisans:
  elena:
    model: gpt-4
    temperature: 0.7
  marcus:
    model: claude-3-opus
    temperature: 0.3
  vera:
    model: gpt-4
    temperature: 0.2

# Performance Settings
performance:
  max_concurrent_artisans: 3
  cache_enabled: true
  
# Budget Controls
budget:
  monthly_limit: 100.00
  alert_threshold: 80%
```

## Your First Commission

### Step 1: Start a Chat

```bash
guild chat
```

You'll see Elena's welcome message:

```
Elena: Good morrow! I'm Elena, foreman of this guild. I'm here to help you plan and execute your software projects. What would you like to build today?
```

### Step 2: Describe Your Project

```
You: I need a REST API for a todo list application with user authentication
```

### Step 3: Interactive Planning

Elena will ask clarifying questions:

```
Elena: Excellent! I'll help you build a todo list API. Let me ask a few questions to understand your requirements better:

1. What framework would you prefer? (Express, Fastify, Gin, etc.)
2. What database will you use? (PostgreSQL, MongoDB, SQLite)
3. Do you need JWT or session-based authentication?
4. Any specific features beyond basic CRUD?
```

### Step 4: Review the Plan

After gathering requirements, Elena presents a plan:

```
Elena: Here's the commission plan for your todo list API:

## Commission: Todo List REST API

### Overview
Build a REST API with user authentication, supporting CRUD operations for todos with PostgreSQL storage and JWT authentication.

### Task Breakdown
1. Set up project structure and dependencies (Marcus)
2. Design database schema (Marcus)
3. Implement authentication endpoints (Marcus)
4. Create todo CRUD endpoints (Marcus)
5. Write API tests (Vera)
6. Add API documentation (Elena)

Shall I proceed with this plan?
```

### Step 5: Watch Progress

Type `yes` to start. In another terminal, monitor progress:

```bash
guild kanban
```

## Working with Artisans

### Artisan Communication

#### Direct Messages
Use @ mentions to talk to specific artisans:

```
@marcus can you explain the authentication implementation?
@vera what test coverage do we have?
@elena what's the project status?
```

#### Artisan Capabilities

Each artisan has specialized tools:

**Elena's Tools**:
- Project planning
- Task breakdown
- Progress tracking
- Documentation

**Marcus's Tools**:
- Code generation
- File manipulation
- Git operations
- Database queries

**Vera's Tools**:
- Test execution
- Code analysis
- Coverage reports
- Validation

### Managing Artisan Work

#### Viewing Active Tasks
```
/task list
```

#### Checking Artisan Status
```
/artisan status
```

#### Pausing/Resuming Work
```
/task pause TASK-001
/task resume TASK-001
```

## Advanced Features

### Corpus Integration

Guild's corpus system helps artisans learn from your documentation:

#### Adding Knowledge
```
/corpus add pattern "Always use dependency injection for testability"
/corpus add example ./examples/di-pattern.go
```

#### Searching Knowledge
```
/search authentication patterns
/corpus stats
```

### Custom Artisans

Create specialized artisans for your needs:

```yaml
# .guild/artisans/sophia.yaml
name: sophia
role: database-expert
model: gpt-4
temperature: 0.3
system_prompt: |
  You are Sophia, a database architecture expert.
  Focus on schema design, query optimization, and data modeling.
tools:
  - read_file
  - write_file
  - execute_sql
```

### Parallel Execution

Guild automatically parallelizes independent tasks:

```
Elena: I've identified 3 independent tasks that can run in parallel:
- Marcus: Implement user endpoints
- Marcus: Implement todo endpoints  
- Vera: Set up test framework

This will reduce completion time from 6 hours to 2 hours.
```

### Session Management

#### Saving Sessions
```
/session save my-important-project
```

#### Loading Sessions
```
guild chat --session my-important-project
```

#### Exporting Conversations
```
/export markdown > project-history.md
/export json > project-data.json
```

## Best Practices

### 1. Clear Commissions

**Good**:
```
Build a REST API for todo management with:
- User registration and JWT auth
- CRUD operations for todos
- PostgreSQL database
- Input validation
- OpenAPI documentation
```

**Too Vague**:
```
Make a todo app
```

### 2. Incremental Development

Break large projects into smaller commissions:

1. First commission: Basic CRUD API
2. Second commission: Add authentication
3. Third commission: Add advanced features

### 3. Review Blocked Tasks

When artisans encounter blockers:

```bash
# Check blocked tasks
ls .guild/kanban/review/

# Review and resolve
$EDITOR .guild/kanban/review/API-005.md
```

### 4. Leverage the Corpus

Feed Guild your existing documentation:

```bash
# Index your docs
guild corpus index ./docs

# Add architecture decisions
/corpus add decision "Use event sourcing for audit logs"
```

### 5. Monitor Costs

Keep track of API usage:

```bash
# View cost dashboard
guild cost

# Set budget alerts
guild config set budget.daily_limit 10.00
```

## Troubleshooting

### Common Issues

#### "No API key found"
```bash
export OPENAI_API_KEY="your-key-here"
# Or add to .guild/guild.yaml
```

#### "Artisan not responding"
```bash
# Check artisan status
guild artisan status

# Restart specific artisan
guild artisan restart marcus
```

#### "Git conflicts in worktree"
```bash
# Guild handles most conflicts automatically
# For manual resolution:
cd .guild/worktrees/marcus/TASK-001
git status
git merge --abort  # If needed
```

### Debug Mode

Enable detailed logging:

```bash
guild chat --debug
```

### Getting Help

- **Documentation**: https://docs.guild.dev
- **Discord**: https://discord.gg/guild-framework
- **GitHub Issues**: https://github.com/guild-framework/guild/issues

## Appendix

### Keyboard Shortcuts

| Shortcut | Action |
|----------|--------|
| `Ctrl+P` | Command palette |
| `Ctrl+K` | Clear chat |
| `Ctrl+S` | Save session |
| `@` + `e/m/v` | Mention artisan |
| `Ctrl+1/2/3` | Switch views |

### Command Reference

| Command | Description |
|---------|-------------|
| `/commission new` | Start new project |
| `/task list` | Show all tasks |
| `/corpus add` | Add knowledge |
| `/search` | Search corpus |
| `/export` | Export chat |
| `/help` | Show all commands |
