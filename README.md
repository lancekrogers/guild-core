# Guild: Collaborative AI Agent Framework

<div align="center">
  <strong>Orchestrate specialized AI agents working together in medieval-themed guilds</strong>
  <br>
  <br>

[![Go Version](https://img.shields.io/badge/Go-1.24-00ADD8.svg)](https://go.dev/dl/)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)]()
[![Status](https://img.shields.io/badge/Status-Pre--MVP-orange.svg)]()

</div>

---

## 🏰 Overview

Guild is an ambitious AI agent orchestration framework that coordinates specialized "artisans" (AI agents) working together on complex tasks. Inspired by medieval guilds where master craftspeople collaborated on projects, Guild provides multi-agent coordination, task management, and intelligent context handling.

> **Note**: Guild is in active development approaching MVP. This README reflects the current state of implementation. Comprehensive documentation updates will follow post-MVP completion.

## ✨ Current Features (Implemented)

### Core Systems

- ✅ **Advanced Chat Interface**: Production-ready TUI with 1,950 lines of code
- ✅ **Tool Execution System**: Complete with workspace isolation and safety features
- ✅ **6-Layer Prompt Architecture**: Dynamic prompt management with token optimization
- ✅ **Project Initialization**: Full `guild init` command with configuration templates
- ✅ **gRPC Infrastructure**: Bidirectional streaming for real-time agent communication
- ✅ **Registry Pattern**: Component discovery and dependency injection
- ✅ **Corpus/RAG System**: Knowledge management with vector storage
- ✅ **Medieval Theming**: Consistent terminology throughout (artisans, guilds, commissions)

### Available Commands

```bash
guild init [path]           # Initialize a new guild project
guild chat                  # Interactive chat with AI agents
guild corpus scan          # Scan and index project documentation
guild commission           # Create and refine work commissions
guild prompt               # Manage layered prompt system
guild agent start         # Start agent services
```

## 🚀 Quick Start

### Prerequisites

- Go 1.24.3 or higher
- Git

### Installation

```bash
# Clone the repository
git clone https://github.com/guild-ventures/guild-core.git
cd guild-core

# Install dependencies
task deps:install

# Build the CLI
task build

# Verify installation
./guild version
```

### Create Your First Guild Project

```bash
# Initialize a new project
guild init my-project
cd my-project

# Set up your API keys
export ANTHROPIC_API_KEY="your-key"
export OPENAI_API_KEY="your-key"

# Start chatting with agents
guild chat
```

This creates a `.guild/` directory with:

```
.guild/
├── guild.yaml          # Main guild configuration
├── config.yaml         # Project settings
├── corpus/            # Knowledge base
├── embeddings/        # Vector storage
├── agents/            # Agent definitions
├── objectives/        # Project goals
└── README.md          # Project documentation
```

## 🏗️ Architecture

Guild uses a sophisticated component-based architecture:

- **Agents** (Artisans): LLM-powered workers with specialized capabilities
- **Orchestrator**: Coordinates multiple agents working on shared objectives
- **Kanban Board**: Task tracking with state management
- **Memory Layer**: BoltDB for persistence, vector search for RAG
- **Tool System**: Extensible tool integration with safety controls
- **Prompt Layers**: 6-layer system for context and behavior management

## 🔨 Development

Guild uses [Taskfile](https://taskfile.dev/) for development workflows:

```bash
# Common tasks
task test              # Run all tests
task test:coverage     # Generate coverage report
task build            # Build the CLI
task clean            # Clean build artifacts

# Development helpers
task run CLI_ARGS="chat"     # Run commands directly
task test:analyze:lore       # Check medieval naming conventions
```

## 📚 Documentation

> **Note**: Documentation is being updated for the MVP release. Current docs may describe planned features.

- [GETTING_STARTED.md](GETTING_STARTED.md) - Will be updated post-MVP with demos
- [DEV.md](DEV.md) - Developer guidelines
- Additional documentation in `docs/` directory

## 🎯 Roadmap to MVP

### Immediate Priorities

1. Fix current build errors in orchestrator package
2. Complete chat → task assignment integration
3. Enable real-time progress tracking
4. Create end-to-end demo workflow

### Post-MVP Plans

- Comprehensive documentation with GIFs/demos
- Additional agent types and capabilities
- Advanced orchestration strategies
- Plugin system for custom tools
- Web UI for monitoring

## 🤝 Contributing

Guild is currently in pre-MVP development. Contribution guidelines will be established post-MVP. For now, please open issues for bugs or feature discussions.

## ⚖️ License

MIT License - see LICENSE file for details.

## 🙏 Acknowledgments

Guild draws inspiration from:

- [CrewAI](https://github.com/joaomdmoura/crewAI) - Multi-agent orchestration
- [LangChain](https://github.com/langchain-ai/langchain) - LLM application patterns
- [Aider](https://github.com/paul-gauthier/aider) - AI pair programming
- Medieval guild structures and craftsperson traditions

---

<div align="center">
  <i>Forging the future of AI agent collaboration, one artisan at a time</i>
</div>

