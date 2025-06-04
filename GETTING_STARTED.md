# Getting Started with Guild Framework

Welcome to Guild! This guide will walk you through setting up your first AI agent team and running a campaign to complete complex objectives.

## Table of Contents

1. [Installation](#installation)
2. [Quick Start](#quick-start)
3. [Configuring Your First Guild](#configuring-your-first-guild)
4. [Creating Objectives](#creating-objectives)
5. [Running Campaigns](#running-campaigns)
6. [Interacting with Agents](#interacting-with-agents)
7. [Review Workflow](#review-workflow)
8. [Advanced Features](#advanced-features)

## Installation

```bash
# Clone the repository
git clone https://github.com/guild-ventures/guild-core.git
cd guild-core

# Install dependencies
task deps:install

# Build Guild
task build

# Verify installation
./bin/guild version
```

## Quick Start

![Guild Quick Start Demo](docs/gifs/quick-start.gif)
*Starting your first campaign in under 2 minutes*

```bash
# Initialize a Guild project
guild init my-project
cd my-project

# Commission your first strategic work
guild commission "Build a web API with authentication" --assign

# Monitor the workshop
guild workshop
```

## Configuring Your First Guild

Guild uses a team of specialized AI agents to complete objectives. Each agent has specific capabilities, tools, and cost considerations.

### Basic Guild Configuration

![Guild Configuration Demo](docs/gifs/configure-guild.gif)
*Setting up a development team with specialized agents*

Create a `guild.yaml` file in your project root:

```yaml
name: "Development Team"
description: "A balanced team for full-stack development"

manager:
  default: "project-manager"
  settings:
    model: "claude-3-sonnet-20240229"
    provider: "anthropic"

agents:
  - id: "backend-dev"
    role: "backend_developer"
    capabilities: ["python", "api", "database", "testing"]
    tools:
      - "shell"
      - "file_system"
      - "http_client"
    model: "claude-3-sonnet-20240229"
    provider: "anthropic"
    cost_magnitude: 0.75
    context_window: 200000
    
  - id: "frontend-dev"
    role: "frontend_developer"
    capabilities: ["javascript", "react", "css", "ui/ux"]
    tools:
      - "file_system"
      - "web_scraper"
    model: "gpt-4"
    provider: "openai"
    cost_magnitude: 1.0
    context_window: 128000
    
  - id: "researcher"
    role: "research_analyst"
    capabilities: ["research", "documentation", "analysis"]
    tools:
      - "corpus"
      - "web_scraper"
      - "file_system"
    model: "claude-3-haiku-20240307"
    provider: "anthropic"
    cost_magnitude: 0.25  # Lower cost for research tasks
    context_window: 200000
```

### Understanding Agent Configuration

- **capabilities**: Skills the agent possesses (used for task assignment)
- **tools**: Available tools the agent can use
- **cost_magnitude**: Relative cost factor (lower = cheaper, preferred for suitable tasks)
- **context_window**: Maximum tokens before context management kicks in

## Commissioning Strategic Work

![Commission Creation Demo](docs/gifs/create-commission.gif)
*Commissioning and coordinating agent collaboration*

### Simple Commission

```bash
guild commission "Build User Authentication" --assign
```

### Hierarchical Objectives

Create structured objectives for complex projects:

```markdown
# E-commerce Platform

## Backend Services
### User Management
- [ ] Design user database schema
- [ ] Implement registration endpoint
- [ ] Create login with JWT tokens
- [ ] Add password reset functionality

### Product Catalog
- [ ] Design product data model
- [ ] Create CRUD endpoints
- [ ] Implement search functionality
- [ ] Add category management

## Frontend Application
### User Interface
- [ ] Design responsive layout
- [ ] Create product grid component
- [ ] Build shopping cart UI
- [ ] Implement checkout flow
```

### Monitoring Commissions

![Commission Monitoring Demo](docs/gifs/monitor-commission.gif)
*Watching the Guild workshop coordinate agents*

```bash
# Monitor all active commissions and agent assignments
guild workshop

# Check specific commission progress
guild commission status

# List all commissions
guild commission list
```

## Running Campaigns

![Campaign Execution Demo](docs/gifs/run-campaign.gif)
*Watching agents collaborate on tasks in real-time*

### Creating a Campaign

```bash
# Create a campaign from objectives
guild campaign create \
  --objective objectives/backend-services.md \
  --objective objectives/frontend-app.md \
  --name "E-commerce Development"
```

### Monitoring Progress

```bash
# Start the campaign
guild campaign start --id campaign-123

# Watch real-time progress
guild campaign watch --id campaign-123
```

### Kanban Board View

![Kanban Board Demo](docs/gifs/kanban-board.gif)
*ASCII kanban board showing task progress*

```
╔════════════════════════════════════════════════════════════════╗
║                 Campaign: E-commerce Development                 ║
╠══════════════╦══════════════╦══════════════╦══════════════════╣
║     TODO     ║  IN PROGRESS ║    REVIEW    ║      DONE        ║
╠══════════════╬══════════════╬══════════════╬══════════════════╣
║ [BE-001]     ║ [FE-002]     ║ [BE-003]     ║ [BE-004]         ║
║ User Schema  ║ Layout Design║ JWT Login    ║ Registration API ║
║ @backend-dev ║ @frontend    ║ @backend-dev ║ @backend-dev     ║
║              ║ 🧠 45%       ║ ⏳ Pending   ║ ✅ Completed     ║
╚══════════════╩══════════════╩══════════════╩══════════════════╝
```

## Interacting with Agents

![Agent Chat Demo](docs/gifs/agent-chat.gif)
*Direct communication with agents during task execution*

### Chat Commands

```bash
# Open the Guild chat interface
guild chat

# Direct message to an agent
> @backend-dev How's the user schema coming along?

# Block an agent's current task for input
> @block.frontend-dev

# Natural language commands
> @backend-dev get started with the login endpoint
```

### Slash Commands

| Command | Description |
|---------|-------------|
| `/status` | Show campaign and agent status |
| `/pause <agent-id>` | Pause a specific agent |
| `/resume <agent-id>` | Resume a paused agent |
| `/block <task-id>` | Block a specific task |
| `/approve <task-id>` | Approve a task in review |
| `/help` | Show all available commands |

## Review Workflow

![Review Workflow Demo](docs/gifs/review-workflow.gif)
*Reviewing and approving agent work*

### Reviewing Tasks

When a task moves to review, Guild creates a markdown file:

```bash
# View tasks in review
ls .guild/kanban/campaign-123/review/

# Edit a review file
$EDITOR .guild/kanban/campaign-123/review/BE-003.md
```

### Review File Format

```markdown
# Task: Implement JWT Login
**ID**: BE-003
**Assigned To**: backend-dev
**Status**: review

## Work Completed
- Created `/api/auth/login` endpoint
- Implemented JWT token generation
- Added refresh token support

## Code Changes
- `src/auth/login.py`: Login endpoint implementation
- `src/auth/tokens.py`: JWT token utilities
- `tests/auth/test_login.py`: Unit tests

## Review Notes
<!-- Add your review comments here -->
Looks good! Please add rate limiting before deployment.

## Next Action
move_to: in_progress  # or 'done' if approved
```

### Approving Work

```bash
# Approve and move to done
guild chat
> /approve BE-003

# Or request changes
> @backend-dev Please add rate limiting to the login endpoint
```

## Advanced Features

### RAG Integration

![RAG Integration Demo](docs/gifs/rag-integration.gif)
*How agents decide what to store for future reference*

Agents automatically determine what information to store in the RAG system:
- 💾 Stored in RAG: Reusable project knowledge
- 🧠 Context only: Temporary conversation data
- 📊 Context meter: Visual indicator of remaining context

### Context Window Management

![Context Management Demo](docs/gifs/context-management.gif)
*Automatic context window handling*

Watch as agents manage their context windows:
- Warning indicators when approaching limits
- Automatic summarization or truncation
- Seamless continuation of long tasks

### Cost Optimization

![Cost Optimization Demo](docs/gifs/cost-optimization.gif)
*Task assignment based on agent capabilities and cost*

Guild automatically assigns tasks to the most cost-effective capable agent:
1. Filters agents by required capabilities
2. Checks tool availability
3. Selects lowest cost_magnitude agent
4. Balances workload across team

### Adding MCP Tools

```yaml
agents:
  - id: "database-expert"
    tools:
      - "shell"
      - "mcp://localhost:3000/postgres-tools"
      - "mcp://localhost:3001/migration-tools"
```

## Troubleshooting

### Common Issues

1. **Import Cycle Errors**
   ```bash
   # Check for build issues
   go build ./...
   
   # Use the build task
   task build
   ```

2. **Agent Not Responding**
   ```bash
   # Check agent status
   guild chat
   > /status
   
   # Restart specific agent
   > /restart backend-dev
   ```

3. **Task Stuck in Review**
   ```bash
   # Force approve
   guild task approve --id BE-003 --force
   ```

## Best Practices

1. **Start Small**: Begin with 2-3 agents and simple objectives
2. **Refine Iteratively**: Use the manager agent to improve objectives
3. **Monitor Context**: Keep an eye on context usage indicators
4. **Review Regularly**: Don't let tasks pile up in review
5. **Cost Awareness**: Configure cost_magnitude appropriately

## Example Projects

### 1. Simple API
- 2 agents (backend, tester)
- Single objective file
- Basic CRUD operations

### 2. Full-Stack Application  
- 5 agents (backend, frontend, designer, tester, devops)
- Hierarchical objectives
- Complete development lifecycle

### 3. Research Project
- 3 agents (researcher, analyst, writer)
- Low-cost models for research
- High-quality model for final output

## Next Steps

1. Join our Discord community
2. Explore advanced agent configurations
3. Contribute your own tools via MCP
4. Share your Guild configurations

Happy building with Guild! 🏰