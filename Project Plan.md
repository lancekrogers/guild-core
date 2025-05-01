# Requirements

## 🧠 High-Level Goals

- Build a customizable agent framework in Go
- Support multi-agent collaboration via “Guilds”
- Use LLMs (OpenAI, Claude, DeepSeek, Ora, Ollama, etc.) as pluggable backends
- Automate recursive task planning and execution — your current manual LLM workflow
- Cache and inspect prompt chains
- Use ZeroMQ to allow other languages to integrate
- Allow for highly configurable project objectives using Markdown
- Model agent workflows in a kanban board
- Use zeroMQ for kanban board actions so that applications can easily be created to modify and monitor boards and task as agents work through them
- Human-in-the-loop model, agents move task to blocked when user input is needed for clarification, to review major changes, etc...
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

### 🕸 Guild Behavior

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

### 📊 Kanban Board

- Agent-specific boards for granular task tracking
- Manager agents view an aggregated Kanban across all agents in the guild

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
- Expose inspection via CLI or REST (grouped by guild, agent, and task)

### 🔄 I/O & Communication

- ZeroMQ as the primary internal communication bus:
  - Task creation and task board updates are sent over ZeroMQ
  - Agent-to-agent, agent-to-manager, and system notifications all use ZeroMQ
- Optional CLI + stdin fallback for standalone dev/test mode
- CLI dashboard displays:
  - Guild status
  - Per-agent Kanban board
  - Real-time task creation and updates (via subscribed ZeroMQ events)

## 🧪 Examples

- **Single-agent “email assistant”** — basic PromptChain + tool execution
- **Frontend/Backend/Reviewer guild** — build a Go + HTMX consulting website
- **Decentralized marketplace guild** — luxury items: blockchain backends, smart contracts, managers, overseer
- Use cases combining LLMs, tool execution, and ZeroMQ communication

---

## 🧠 User Workflow Statement

> ⚙️ **Overview**  
> The Guild Framework automates project execution using configurable AI agents and a Kanban-style task board.

---

### 📝 Step 1: Define the Project Spec

- 🧾 Write a **detailed markdown spec** describing the desired outcome, scope, constraints, and key context.

---

### 🧩 Step 2: Configure Agents

- 🧠 Assign **role-specific agents** (e.g., Frontend Dev, Planner, Reviewer).
- 🧰 Attach tools and APIs (email, codegen, web search, etc.).
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
