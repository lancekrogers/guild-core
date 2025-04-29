# Requirements

## 🧠 High-Level Goals

- Build a customizable agent framework in Go
- Support multi-agent collaboration via “Guilds”
- Use LLMs (OpenAI, Claude, DeepSeek, Ora, Ollama, etc.) as pluggable backends
- Automate recursive task planning and execution — your current manual LLM workflow
- Cache and inspect prompt chains
- Use ZeroMQ to allow other languages to integrate
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

- Use ZeroMQ for cross-language task/agent message passing
- Optional CLI + stdin fallback for standalone usage
- CLI dashboard for real-time guild status

## 🧪 Examples

- **Single-agent “email assistant”** — basic PromptChain + tool execution
- **Frontend/Backend/Reviewer guild** — build a Go + HTMX consulting website
- **Decentralized marketplace guild** — luxury items: blockchain backends, smart contracts, managers, overseer
- Use cases combining LLMs, tool execution, and ZeroMQ communication
