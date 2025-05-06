# Requirements

## 🧑‍🌾 High-Level Goals

- Build a customizable agent framework in Go
- Support multi-agent collaboration via “Guilds”
- Use LLMs (OpenAI, Claude, DeepSeek, Ora, Ollama, etc.) as pluggable backends
- Automate recursive task planning and execution — your current manual LLM workflow
- Cache and inspect prompt chains
- Use ZeroMQ to allow other languages to integrate
- Allow for highly configurable project objectives using Markdown
- Model agent workflows in a kanban board
- Use ZeroMQ for kanban board actions so that applications can easily be created to modify and monitor boards and tasks as agents work through them
- Human-in-the-loop model: agents move tasks to blocked when user input is needed for clarification or review
- Rebuild your Golang fluency while building something genuinely useful

## ✅ Core Requirements by Category

### 🧱 Core Concepts

- **Agent**: A configurable worker with a role, personality, goals, and tools
- **Guild**: A collection of agents that coordinate around a shared objective
- **Tool**: Interface for extending agent capabilities (e.g., email, file I/O, web search)
- **Memory & VectorStore**:
  - **In-memory cache** of prompts, decisions, and outputs
  - **BoltDB** for structured persistence (Kanban, tasks, objectives, standard records)
  - **Qdrant** as the vector database for embeddings-based lookups (prompt-chain similarity, objective retrieval)
- **PromptChain**: A logged set of prompt–response pairs per agent (for replay/study)

### ⚙️ Configuration

- YAML or JSON–based config for:
  - Defining agents (roles, specialization, model config)
  - Guild composition
  - Goals for agents or guilds
  - Tools each agent can access
  - Configurable models and providers
  - API key management via `.env` or config file + env overrides

### 🧠 Agent Behavior

- Execute tasks via LLM API
- Access tools via interfaces (email, HTTP, file, search, etc.)
- React to feedback from other agents or humans (event loop)
- Recursively break down tasks based on prompts and goals
- Maintain each agent’s personal Kanban board of tasks
- Move tasks through the board as they’re completed
- Refactor or rewrite task specs dynamically when appropriate

### 🛨 Guild Behavior

- Coordinate task execution among agents
- Ask the user for input when needed
- Aggregate results toward a shared guild-level goal
- Concurrent execution of agent actions using goroutines and channels

### 🎯 Guild Objective Configuration

- Objectives authored in Markdown, broken down by component and agent type
- Parser reads headers & tags to generate tasks per agent/component
- Bullet-point details under each header become the task description

### 📋 Tasks Management

- Tasks organized by guild, agent type, and individual agent
- Tasks stored in **BoltDB** and presented in Kanban style
- Each agent sees a personalized Kanban view (To Do, In Progress, Done)
- In-progress tasks carry their associated PromptChain until completion
- Completed PromptChains can be archived or reviewed via the vector store
- Status transitions governed by `task.Engine` package

### 📊 Kanban Board

- Agent-specific boards for granular task tracking
- Manager agents view an aggregated Kanban across all agents in the guild
- Task updates and transitions published over ZeroMQ using a defined message protocol (task ID, action, agent, etc.)

### 🔌 Model Abstraction

Support interchangeable LLM backends:

- ✅ OpenAI
- ✅ Claude (Anthropic)
- ✅ DeepSeek
- ✅ Ora API
- ✅ Ollama (local)

### 💾 PromptChain & Memory

- **In-memory store** with optional persistence
- **BoltDB** for structured records (tasks, objectives, agent state)
- **Qdrant** for vector embeddings:
  - Store and index embeddings of prompts, responses, and objectives
  - Perform semantic lookups (e.g., “find similar past prompts” or “retrieve related objectives”)
- Support streaming prompt updates via channel
- Expose inspection via CLI or REST (grouped by guild, agent, and task)

### ↺ I/O & Communication

- ZeroMQ as the primary internal communication bus:
  - Task creation and task board updates are sent over ZeroMQ
  - Agent-to-agent, agent-to-manager, and system notifications all use ZeroMQ
  - Message format defined in `kanban.proto` or equivalent
- Optional CLI + stdin fallback for standalone dev/test mode
- CLI dashboard displays:
  - Guild status
  - Per-agent Kanban board
  - Real-time task creation and updates (via subscribed ZeroMQ events)

### 🛠️ External Coding Assistants

- Support local CLI-based coding agents as Guild tools:
- Claude Code terminal tool (included with Claude Max subscription)
- Aider (open-source AI coding assistant)
- Agents can invoke these tools to:
- Generate or refactor code in a local working directory
- Execute shell commands or review diffs as needed
- Capture output for inclusion in prompt chains or follow-up actions
- Each CLI tool is registered as a CodeRunner-compatible tool and spawned via subprocess
- Users can configure:
- Tool path (claude, aider, etc.)
- Working directory per task or agent
- Optional STDIN/STDOUT piping, TTY emulation if needed

### 🧰 CLI Tooling Environment (Agent-Specific)

- Each agent can be configured with a set of available command-line tools in its environment
- Tools include:
- Purpose-built CLI utilities (e.g., tree2scaffold, goreleaser, buf, protoc, make)
- Custom shell scripts
- Code assistants like Claude Code and Aider
- Each tool entry includes:
- name: Tool identifier
- cmd: The CLI command or entry point
- context_description: A natural language explanation for the LLM describing when and why this tool should be used instead of using tokens
- args: Optional templated CLI arguments (e.g., using task variables)
- working_dir: Optional path override for execution context

Example YAML config:

```yaml
tools:
  - name: tree2scaffold
    cmd: "./bin/tree2scaffold"
    context_description: "Use this tool to convert a folder structure into a scaffold config before asking the LLM to write code."
  - name: goreleaser
    cmd: "goreleaser release --clean"
    context_description: "Run this to create a new release build for Go-based CLI tools."
  - name: aider
    cmd: "aider --yes --message '{{task}}'"
    context_description: "Invoke this assistant when working on complex code refactors that are already well-scoped."
```

- During prompt generation, the agent includes tool descriptions when relevant to nudge the LLM toward invoking them
- Runtime logic determines when a task should invoke a tool vs. route through the LLM
- Output from tools can optionally be logged, summarized, or fed back into the agent’s memory

## 🧪 Examples

- **Single-agent “email assistant”** — basic PromptChain + tool execution
- **Frontend/Backend/Reviewer guild** — build a Go + HTMX consulting website
- **Decentralized marketplace guild** — luxury items: blockchain backends, smart contracts, managers, overseer
- Use cases combining LLMs, tool execution, and ZeroMQ communication

---

### 🗺️ Objective as a Hierarchical Project Plan

Guild objectives are authored as a structured hierarchy of Markdown files that reflect a natural project planning process. This enables both human contributors and LLM agents to understand, traverse, and act on the project’s goals from high-level vision to granular implementation steps.

🧱 Structure
• The root of the project objective is a directory tree (e.g., /objectives/) containing Markdown files and subfolders.
• Each file corresponds to a node in a mind map—a discrete sub-objective with:
• A clear goal
• Contextual details
• Linked dependencies
• Tags for traversal
• Folders group related objectives into systems, roles, tools, or workflows.
• Agents can navigate the tree recursively or follow explicit links between files.

📁 Example Hierarchy

```bash
/objectives
├── README.md                      # Global vision and constraints
├── infrastructure/
│   ├── overview.md
│   ├── kanban-system.md
│   └── qdrant-integration.md
├── agents/
│   ├── roles.md
│   ├── planner.md
│   └── coder.md
├── cli-tools/
│   ├── aider.md
│   └── tree2scaffold.md

```

### 📝 Document Format

Each Markdown file is structured as a prompt-friendly spec, readable and actionable by both LLMs and humans:

```markdown
# 🧠 Goal

Implement a persistent, ZeroMQ-connected Kanban system to coordinate agent task execution.

# 🗂️ Context

This component enables agents and external UIs to view and mutate task state through a shared event bus and BoltDB backend.

# 🔧 Requirements

- Store all task transitions in BoltDB
- Emit task change events via ZeroMQ
- Support external CLI task manipulation

# 📌 Tags

kanban, task-engine, zmq, persistence

# 🔗 Related

- ../agents/manager.md
- ../infrastructure/qdrant-integration.md
```

## 🤖 Agent Usage

    • Agents reference objective files to derive their scope and constraints
    • Linked files act as context chains, reducing token usage while preserving intent
    • Completed tasks are traceable back to the objective nodes they fulfill
    • Enables agents to reason locally within a node, or globally by traversing the graph

---

### 💰 Cost-Aware Agent & Guild Behavior

To minimize unnecessary LLM usage and prioritize efficient workflows, Guild managers and agents operate under a cost-aware model. Each tool or action is assigned a configurable numeric cost that informs agent decision-making and planning.

This enables:

- ⚖️ Cost-based task assignment by manager agents
- 🧠 Smart tool selection by agents to avoid expensive LLM calls when not necessary
- 🔧 Customizable environment tuning by users to reflect real-world compute or API costs

⸻

📐 Cost Scoring System

Each tool, model, or subprocess is given a relative cost score.

- Local CLI tools (e.g., make, tree2scaffold)
- Default cost: 0
- Treated as free by agents
- Local LLMs (e.g., Ollama, Mistral, LLaMA)
- Default cost: 1 for small models
- Up to 3 for large or GPU-intensive models
- Remote/API LLMs (e.g., GPT-4, Claude)
- Cost reflects real token pricing
- User-defined in config (e.g., cost per million tokens)

⸻

⚙️ Cost Configuration

Example cost config in guild.yaml or equivalent:

```yaml
costs:
  cli_tools:
    default: 0

  local_models:
    gemma-2b: 1
    llama3-70b: 3
    mistral: 2

  api_models:
    openai-gpt-4: 60 # Cost per 1M tokens (USD-based or normalized unit)
    claude-opus: 45
    deepseek-coder: 25
```

- All values are user-overridable
- Costs can be documented in the tool/model registry

⸻

🧠 Agent Behavior

- Agents inspect costs when selecting tools or planning prompt chains
- Whenever possible, they will:
- Prefer lower-cost options that achieve the same goal
- Justify the use of higher-cost tools in their reasoning output
- Manager agents can reassign tasks or adjust agent roles to reduce projected cost

⸻

🧾 Optional: Cost Logging

- Estimated cost per task can be tracked and logged
- Enables:
- Agent/guild usage breakdown
- Future optimization via dashboards
- User visibility into cumulative token/API usage

### 🧩 Cost Strategy Interface (Go)

To support cost-aware decisions at runtime, Guild defines a pluggable cost estimation interface:

```go
type CostEstimator interface {
    EstimateToolCost(toolName string) int
    EstimateModelCost(modelName string, tokens int) int
}
```

    • EstimateToolCost: Returns static or dynamic cost for CLI tools or internal operations
    • EstimateModelCost: Computes cost based on model name and token count (used in API calls)

This allows:
• Agents to compare potential actions (e.g., call LLM vs. run local tool)
• Managers to optimize task distribution
• CLI dashboards to expose real-time or historical cost breakdowns

Default implementations read from the user’s config (guild.yaml), but this interface can also support runtime pricing APIs or adaptive heuristics in the future.

---

🧠 Meta-Coordination & Optimization Protocol (MCP + RAG)

Guild implements a system-level strategy for minimizing redundant LLM usage, improving tool adoption, and maintaining high-quality results at low cost. This behavior is governed by two cooperating mechanisms:
• MCP (Meta-Coordination Protocol) – used by manager agents to detect inefficiencies, flag opportunities for optimization, and guide multi-agent behavior
• RAG (Retrieval-Augmented Generation) – used by all agents to semantically retrieve past context, prompt chains, and decisions

Together, these enable:
• Automatic restoration of agent context after resets or token window exhaustion
• Cost-based planning by default
• A persistent memory of past decisions, tool usage, and mistakes

---

### 🧭 MCP – Coordination & Oversight Layer

MCP allows manager agents to:
• Detect repeated prompt chains or patterns of high-cost tool use
• Flag such patterns as tool_candidate entries in memory (BoltDB or Qdrant)
• Recommend creation of a CLI tool to automate the pattern
• Monitor overall guild behavior and enforce global rules (e.g., avoid GPT-4 for formatting tasks)

Logged Examples:

```json
{
  "type": "tool_candidate",
  "task_signature": "format markdown tables",
  "example_prompt": "...",
  "estimated_cost_per_use": 8,
  "recommended_tool_name": "format-md"
}
```

### 🔍 RAG – Prompt Recovery & Memory Lookup

RAG is used by agents to:
• Retrieve relevant past prompt chains, outputs, and tool usage
• Rehydrate prior decisions without consuming context tokens
• Load relevant tool descriptions, usage patterns, or Markdown objectives

Sources of Memory:
• Qdrant embeddings of:
• Prior tasks and prompt chains
• Markdown objectives
• Completed outputs tagged with purpose
• Symbolic lookups (e.g., by tag or task type)

---

🔄 Cost-Aware Context Restoration

When an agent begins or resumes a task: 1. RAG retrieves task-relevant prompt chain history 2. MCP-injected metadata informs the agent of:
• Available tools
• Their cost vs. LLM alternatives
• Prior usage patterns 3. The agent uses this restored context to:
• Prefer efficient tool invocation
• Avoid previously expensive or redundant decisions
• Justify higher-cost actions when no alternative exists

---

📊 CLI Tool Optimization Loop

Guild enables an optimization cycle: 1. Agents log repeated high-cost tasks 2. MCP flags these as tool candidates 3. The user builds a CLI tool 4. The tool is added to the agent config 5. Future tasks use the new tool instead of the LLM

This enables compounding efficiency gains over time.

---

## 🧠 User Workflow Statement

> ⚙️ **Overview**  
> The Guild Framework automates project execution using configurable AI agents and a Kanban-style task board.

---

### 📝 Step 1: Define the Project Spec

- 📟 Write a **detailed markdown spec** describing the desired outcome, scope, constraints, and key context.

---

### 🧹 Step 2: Configure Agents

- 🧠 Assign **role-specific agents** (e.g., Frontend Dev, Planner, Reviewer).
- 🛠️ Attach tools and APIs (email, codegen, web search, etc.).
- 🤖 Specify model backends like `OpenAI`, `Claude`, `Ollama`, or local models.

---

### 🗂️ Step 3: Generate the Kanban Board

- 🧱 The system **decomposes the spec** into structured tasks.
- 📌 Tasks are organized into a **Kanban board** (`To Do`, `In Progress`, `Blocked`, `Done`).
- 🧑‍💻 Agents **autonomously pick up tasks** aligned with their configuration.

---

### 🚧 Step 4: Human-in-the-Loop Interaction

- 🛑 Tasks that need clarification or input are marked **Blocked**.
- 👤 The user reviews blocked tasks, answers agent queries, or updates task specs.

---

### 📈 Step 5: Real-Time Monitoring & Adjustment

- 🖥️ Track task status and agent output in real time.
- 🧹 Adjust agents, tools, and the board dynamically as the project evolves.

---

> ✅ **Result**: AI agents collaborate under tight constraints, optimizing for quality and cost, while keeping the user in control of high-leverage decisions.
