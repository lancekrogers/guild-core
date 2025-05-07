# 🏗️ Architecture

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

## 🧱 Core Concepts

- **Agent**: A configurable worker with a role, personality, goals, and tools
- **Guild**: A collection of agents that coordinate around a shared objective
- **Tool**: Interface for extending agent capabilities (e.g., email, file I/O, web search)
- **Memory & VectorStore**:

  - **In-memory cache** of prompts, decisions, and outputs
  - **BoltDB** for structured persistence (Kanban, tasks, objectives, standard records)
  - **Qdrant** as the vector database for embeddings-based lookups (prompt-chain similarity, objective retrieval)

- **PromptChain**: A logged set of prompt–response pairs per agent (for replay/study)

## 🔌 Model Abstraction

Support interchangeable LLM backends:

- ✅ OpenAI
- ✅ Claude (Anthropic)
- ✅ DeepSeek
- ✅ Ora API
- ✅ Ollama (local)
