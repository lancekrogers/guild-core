# Guild: Collaborative AI Agent Framework

<div align="center">
  <img src="docs/images/readme_banner.png" alt="Guild Framework Banner" width="100%">
  <br><br>
  <strong>Orchestrate specialized AI agents working together in medieval-themed guilds</strong>
  <br><br>

[![Go Version](https://img.shields.io/badge/Go-1.23+-00ADD8.svg)](https://go.dev/dl/)
[![License](https://img.shields.io/badge/License-Angry%20Goat-red.svg)](LICENSE)
[![Status](https://img.shields.io/badge/Status-Pre--MVP-orange.svg)]()

</div>

---

> ⚠️ **Guild is under active development and not yet ready for public use.**
>
> This project is being shared early to demonstrate ongoing work on a deeply modular AI orchestration framework. Many systems are incomplete or experimental, and the codebase is **not yet stable or usable out-of-the-box**.
>
> If you're curious, feel free to follow development or get in touch — but **this is not a production-ready tool**, nor is it currently suitable for integration into your projects.

---

## 🏰 What Is Guild?

**Guild** is a collaborative, agentic framework built in Go for orchestrating specialized AI agents to work together toward complex goals — inspired by medieval guilds of skilled craftspeople.

It provides:

- Configurable agents with roles, tools, and goals
- Shared memory and structured task decomposition
- Kanban-style task coordination
- A command-line interface for managing projects, agents, and tasks

The goal is to provide a **developer-first, deeply modular system** for building and scaling multi-agent AI workflows — one that balances fun, clarity, and serious engineering.

---

## 🎬 Visual Demo: Real-time Kanban Board

Guild's kanban board provides real-time visual coordination for multi-agent workflows. Watch tasks flow through columns as agents collaborate, with events streaming at sub-200ms latency:

### Quick Demo: Task Creation and Real-time Updates

![Guild Kanban Quick Demo](demo-assets/quick-demo-optimized.gif)

*Create tasks and watch them appear instantly across all connected kanban views*

### Multi-Agent Workflow Coordination  

![Guild Kanban Multi-Agent Workflow](demo-assets/multi-task-workflow-optimized.gif)

*Multiple agents working simultaneously with automatic task blocking and dependency resolution*

### Performance at Scale

![Guild Kanban Performance Showcase](demo-assets/performance-showcase-optimized.gif)

*Smooth 30 FPS rendering with 200+ tasks and real-time search capabilities*

**Key Features Demonstrated:**

- ⚡ **Real-time Updates**: Sub-200ms latency from agent action to UI display
- 🔄 **Event Streaming**: Live task status changes across all connected views  
- 🚫 **Smart Blocking**: Automatic task blocking with human review workflow
- 🔍 **Instant Search**: Filter 200+ tasks with real-time results
- 📊 **Performance**: 30 FPS rendering and >5k events/second throughput
- 👥 **Multi-Agent**: Visual coordination of parallel agent work

Try the interactive demo yourself:

```bash
# Record your own kanban demo
./scripts/record-kanban-demo.sh quick-demo

# Or try the full performance showcase  
./scripts/record-kanban-demo.sh performance-showcase
```

> **See the complete demo commission example**: [examples/kanban-demo-commission.md](examples/kanban-demo-commission.md) shows a realistic 30-task development workflow demonstrating how agents collaborate through the kanban board to build a task tracking API.

---

## 🔧 Project Status

Guild is in **pre-MVP development**. Core infrastructure is in place, but many commands and systems are incomplete or non-functional. The project is being opened publicly to:

- Share development progress
- Get feedback and issue reports
- Allow early supporters to follow along
- Begin building a contributor community (later)

---

## 📜 License

Guild is **not open source**.

It is licensed under a custom [Angry Goat License](LICENSE), which means:

- ✅ Personal, educational, and non-commercial use is allowed
- ❌ Commercial use (SaaS, resale, internal enterprise, etc.) is **prohibited** without a commercial license
- 🔒 Redistribution, forking, and repackaging are not allowed
- 💼 If you'd like to use Guild commercially, email [lance@blockhead.consulting](mailto:lance@blockhead.consulting)

> This is a **source-available project**, not an open source one.

---

## 🧭 What’s Next?

I'm working toward a stable MVP that will include:

- A multi-agent TUI chat
- Multi-agent task decomposition
- Local/remote model provider integration
- Workspace-safe tool execution
- Agent memory + human knowledge base integration
- Kanban agent task tracking
- Cost aware task assignment
- Themed project structure, making multi-agentic development easy and intuitive
- Layered system prompts

If you’d like to follow development, feel free to:

- ⭐ Star the repo
- Watch for updates
- Open issues for feedback or discussion
- Contact: [lance@blockhead.consulting](mailto:lance@blockhead.consulting)

---

## 🏗️ Architecture Overview

Guild is built as a modular, extensible framework for orchestrating multiple AI agents working together on complex tasks. The architecture emphasizes:

**Core Components:**

- **Agent Framework**: Flexible system for defining AI agents with specialized capabilities, tools, and behaviors
- **Orchestration Engine**: Coordinates multiple agents, managing task dependencies and inter-agent communication via gRPC
- **Memory Layer**: SQLite-based persistence with vector search capabilities for maintaining context across sessions
- **Tool Execution System**: Sandboxed environments for agents to safely execute code and interact with external systems
- **Prompt Management**: Multi-layer prompt system enabling dynamic behavior customization while maintaining efficiency
- **Task Management**: Kanban-style board for tracking work items and managing human review requirements

**Technology Stack:**

- **Go**: Chosen for performance, strong concurrency primitives, and type safety
- **SQLite**: Production-ready relational database for state management
- **gRPC**: High-performance RPC framework for inter-component communication
- **Bubble Tea**: Modern TUI framework for rich terminal interfaces

---

## 🎯 Technical Philosophy

**Why Go?**
Guild is built in Go rather than Python (the typical choice for AI projects) for several deliberate reasons:

- **Performance**: Go's compiled nature and efficient runtime make it ideal for orchestrating multiple concurrent agents
- **Type Safety**: Strong typing catches errors at compile time, crucial for a complex multi-agent system
- **Concurrency**: Go's goroutines and channels provide elegant patterns for coordinating parallel agent operations
- **Single Binary**: Easy deployment without dependency management headaches

**Developer-First Design**

- **Modularity**: Every major system is behind an interface, allowing easy extension and testing
- **Registry Pattern**: Components are discoverable and loosely coupled through a central registry
- **Progressive Disclosure**: Start simple with one agent, scale up to complex multi-agent workflows
- **Local-First**: Everything runs on your machine - no cloud dependencies unless you want them

**The Guild Metaphor**
While not yet consistently implemented, the medieval guild theme serves a purpose beyond aesthetics:

- Makes complex multi-agent concepts more intuitive
- Provides a consistent mental model for system components
- Adds personality to what could be a dry technical framework

The goal is serious engineering wrapped in an approachable, memorable package.

---

## 🙏 Acknowledgments

Guild draws inspiration from:

- [CrewAI](https://github.com/joaomdmoura/crewAI) - Multi-agent orchestration patterns
- [LangChain](https://github.com/langchain-ai/langchain) - LLM application framework concepts
- [Aider](https://github.com/paul-gauthier/aider) - AI pair programming interactions
- Medieval guild structures and master-apprentice relationships

---

<div align="center">
  <i>Forging the future of agentic collaboration — one artisan at a time.</i>
</div>
