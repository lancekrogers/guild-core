# Guild Agent Framework Project Overview

## Core Concept

Guild is a comprehensive agent framework built in Go that enables multi-agent collaboration through "Guilds." These guilds coordinate autonomous agents to complete complex objectives using shared memory, cost heuristics, and human-in-the-loop coordination.

## Architecture

- **Agent**: A configurable worker with a role, personality, goals, and tools
- **Guild**: A collection of agents that coordinate around a shared objective
- **Tool**: Interface for extending agent capabilities (e.g., email, file I/O, web search)

### Memory & Storage

- **In-memory cache**: Stores prompts, decisions, and outputs during execution
- **BoltDB**: Provides structured persistence for Kanban boards, tasks, objectives, and other records
- **Qdrant**: Serves as a vector database for embeddings-based lookups (prompt-chain similarity, objective retrieval)
- **PromptChain**: Logged set of prompt-response pairs per agent (for replay/study)

### Model Support

The framework supports interchangeable LLM backends:

- OpenAI
- Claude (Anthropic)
- DeepSeek
- Ora API
- Ollama (local)

## Agent & Guild Behavior

### Agent Behavior

- Execute tasks via LLM API
- Access tools via interfaces
- React to feedback from other agents or humans
- Recursively break down tasks based on prompts and goals
- Maintain personal Kanban board of tasks

### Guild Behavior

- Coordinate task execution among agents
- Ask users for input when needed
- Aggregate results toward a shared guild-level goal
- Execute agent actions concurrently using goroutines and channels

### Cost-Aware Behavior

- Agents inspect configured costs when selecting tools or planning prompt chains
- Default to lower-cost tools when functionally equivalent
- Log high-cost actions with reasoning justification
- Use costs for work reassignment and rebalancing

## Tools & Assistants

### Code Assistants

- Claude Code terminal: Interactive code generation and editing
- Aider: Open-source CLI code assistant for scoped refactoring or implementation tasks

### Command-Line Tools

Agents can be configured with custom command-line tools to:

- Replace repetitive LLM calls
- Standardize transformations
- Reduce prompt token costs

### Tool Configuration

```yaml
tools:
  - name: tree2scaffold
    cmd: "./bin/tree2scaffold"
    context_description: "Convert a folder into a scaffold config before asking LLM to write code."
  - name: aider
    cmd: "aider --yes --message '{{task}}'"
    context_description: "Refactor code with scoped assistant."
```

## Meta-Coordination Protocol (MCP) & RAG

The system uses two mechanisms for optimization:

### MCP (Meta-Coordination Protocol)

Manager agents:

- Detect repeated prompt chains or inefficient LLM use
- Flag repetitive patterns as tool candidates
- Recommend CLI tools for automation
- Apply heuristics for task cost, similarity, and outcome

### RAG (Retrieval-Augmented Generation)

Used by agents to:

- Recover past prompt chains and outputs
- Retrieve project objectives or known tool usage
- Load examples or summaries from memory instead of re-generating

### Optimization Loop

1. Agents complete repeated high-cost tasks
2. Manager detects the pattern
3. Flagged as tool candidate
4. User builds CLI tool
5. Tool added to agent config
6. Future prompts invoke CLI instead of LLM

## Objectives System

Guild uses a directory-based Markdown objective system to encode complex project goals.

### Directory Structure

```
/objectives
├── README.md                      # High-level vision & constraints
├── infrastructure/
│   ├── overview.md
│   ├── kanban-system.md
│   └── qdrant-integration.md
├── agents/
│   ├── roles.md
│   ├── planner.md
│   └── coder.md
```

### Markdown Format

```markdown
# 🧠 Goal

State the specific outcome this task or subcomponent should achieve.

# 📂 Context

Summarize the component's role in the system.

# 🔧 Requirements

- List of specs, interfaces, outputs
- Clear enough to be translated into subtasks

# 📌 Tags

- kanban
- task-engine
- zmq

# 🔗 Related

- [../agents/manager.md](../agents/manager.md)
```

## Kanban Task System

Guild's task system is modeled after a Kanban workflow:

### Task Properties

- `id`: unique identifier
- `title`: short summary
- `description`: full prompt or spec
- `status`: To Do, In Progress, Blocked, Done
- `agent`: assigned worker
- `prompt_chain`: optional prompt history

### Status Workflow

- `To Do`
- `In Progress`
- `Blocked` (human input needed)
- `Done`

### ZeroMQ Integration

ZeroMQ serves as the internal messaging and coordination bus, with events like:

- `task_created`
- `task_updated`
- `task_moved`
- `task_completed`
- `task_blocked`

## Memory & Retrieval

### PromptChain

Each agent maintains a history of prompt-response pairs per task:

- Stored in-memory during execution
- Archived to disk (BoltDB) upon task completion
- Indexed by task ID, agent ID, timestamp, and outcome

### Vector Store (Qdrant)

Used for semantic memory and fast similarity search, indexing:

- Past prompt chains
- Markdown objective files
- Final task outputs or summaries

## Configuration

Guild uses YAML or JSON configuration files:

```yaml
agents:
  - name: planner
    model: claude-3-opus
    tools:
      - tree2scaffold
      - search-codebase

  - name: implementer
    model: ollama:gemma-2b
    tools:
      - make
      - aider

guilds:
  - name: guild-dev
    agents:
      - planner
      - implementer
```

### Cost Configuration

```yaml
costs:
  cli_tools:
    default: 0
  local_models:
    gemma-2b: 1
    llama3-70b: 3
  api_models:
    openai-gpt-4: 60 # cost per 1M tokens
    claude-opus: 45
    deepseek-coder: 25
```

## User Workflow

1. **Define Objectives in Markdown**
   - Create a hierarchical tree of markdown files
   - Structure each file as a prompt with goals, context, requirements
2. **Configure Secure Local Agents**
   - Define agents, models, and tools in guild.yaml
   - Configure cost settings to enforce policy (e.g., offline-only)
3. **Generate and Manage Kanban**
   - Guild parses objectives into a Kanban board per agent
   - Monitor progress via dashboard or ZeroMQ events
4. **Human-in-the-Loop Review**
   - Agents mark tasks as Blocked when they need human judgment
   - Users can edit specs, approve outputs, or add notes
5. **Continuous Optimization via MCP + RAG**
   - System flags repetitive tasks as tool candidates
   - Agents use memory to restore context and prefer efficient tools

## Git Workflow & Concurrency

Guild agents operate concurrently using Git branches as isolated sandboxes:

### Workflow

- Manager creates a repo with main branch
- Each agent gets its own working branch (agent/{n})
- Agents only work in their designated branch
- When tasks complete, agents commit, push, and notify the manager
- Manager attempts to merge changes into main
- Kanban board tracks merge status

### Interface Handling

- Tasks marked as "interface" get special treatment
- Dependent tasks are blocked until interfaces are complete
- When interfaces change, dependent tasks return to Blocked status
- Agents refresh context via RAG when working on interface-dependent tasks

## Planned Enhancements

- Model routing by task complexity
- Versioned specs + live updates
- Permissioning & sandboxing
- Retry & failure handling
- Agent personality/role templates

## Example Use Cases

- **Single-Agent Email Assistant**: Uses Claude 3 Sonnet to suggest email replies
- **Dev Guild: Go + HTMX Website Builder**: Employs planner, implementer, and reviewer agents
- **Decentralized Marketplace**: Builds blockchain-based exchange with manager, contract-dev, and frontend agents
- **Legal Documentation Automation**: Privacy-preserving system for patent drafting using local models

