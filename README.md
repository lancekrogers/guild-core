# Guild: Collaborative AI Agent Framework

<div align="center">
  <img src="docs/images/readme_banner.png" alt="Guild Framework Banner" width="100%">
  <br>
  <br>
  <strong>Orchestrate specialized AI agents working together in medieval-themed guilds</strong>
  <br>
  <br>

[![Go Version](https://img.shields.io/badge/Go-1.23+-00ADD8.svg)](https://go.dev/dl/)
[![License](https://img.shields.io/badge/License-Custom-blue.svg)](LICENSE)
[![Status](https://img.shields.io/badge/Status-Beta-yellow.svg)]()

</div>

---

## 🏰 Overview

Guild is an ambitious AI agent orchestration framework that coordinates specialized "artisans" (AI agents) working together on complex tasks. Inspired by medieval guilds where master craftspeople collaborated on projects, Guild provides multi-agent coordination, task management, and intelligent context handling.

> **Note**: Guild is in active development approaching MVP. This README reflects the current state of implementation. Comprehensive documentation updates will follow post-MVP completion.

## ✨ Current Features (Implemented)

### Core Systems

- ✅ **Advanced Chat Interface**: Production-ready TUI
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
guild chat                  # Interactive chat with AI agents (TUI)
guild corpus scan          # Scan and index project documentation
guild corpus query         # Query indexed documentation
guild commission           # Create and refine work commissions
guild commission refine    # Refine existing commission documents
guild prompt               # Manage layered prompt system
guild campaign             # Manage campaign workflows
guild serve                # Start gRPC server (build errors present)
guild agent start         # Start agent services (not implemented)
guild cost                 # Cost tracking tools
guild migrate              # Migrate from global to project configuration
```

## 🚀 Quick Start

### Prerequisites

- Go 1.24 or higher
- Git
- SQLite (for storage)

### Installation

```bash
# Clone the repository
git clone https://github.com/guild-ventures/guild-core.git
cd guild-core

# Install dependencies
make deps

# Build the CLI (Note: some packages have build errors)
make build

# Verify installation
./bin/guild version
```

### Create Your First Guild Project

```bash
# Initialize a new project
./bin/guild init my-project
cd my-project

# Set up your API keys (at least one required)
export ANTHROPIC_API_KEY="your-key"
# export OPENAI_API_KEY="your-key"      # Optional
# export OLLAMA_HOST="localhost:11434"  # Optional for local models

# Start chatting with agents
../bin/guild chat
```

This creates a `.guild/` directory with:

```
.guild/
├── guild.yaml          # Main guild configuration
├── memory.db          # SQLite database for state
├── corpus/            # Knowledge base documents
├── objectives/        # Commission documents
│   └── refined/      # Refined commission outputs
├── kanban/           # Task board state
├── archives/         # Agent memory
├── campaigns/        # Campaign definitions
└── prompts/         # Custom prompt templates
```

## 🏗️ Architecture

Guild uses a sophisticated component-based architecture:

- **Agents** (Artisans): LLM-powered workers with specialized capabilities
- **Orchestrator**: Coordinates multiple agents working on shared objectives
- **Kanban Board**: Task tracking with SQLite state management
- **Memory Layer**: SQLite for persistence, ChromemGo for vector search
- **Tool System**: Extensible tool integration with workspace isolation
- **Prompt Layers**: 6-layer system for context and behavior management
- **Registry Pattern**: Dynamic component discovery and dependency injection

## 🔨 Development

### 🚨 CRITICAL: Test Execution Rules

**NEVER run `go test` directly** - it creates `.test` binaries that pollute the repository!

```bash
# ❌ WRONG - Creates .test binaries
go test ./...

# ✅ CORRECT - Use make or task
make test              # Run all tests properly
make unit-test         # Run unit tests with dashboard
make integration       # Run integration tests
task test              # Alternative using Taskfile
```

### Development Workflow

Guild uses both Makefile and [Taskfile](https://taskfile.dev/) for development:

```bash
# Build commands (ALWAYS use make/task)
make build            # Build with visual progress
task build            # Alternative build command

# Test commands (NEVER use go test directly)
make test              # Run all tests
make unit-test         # Unit tests with dashboard
make coverage          # Generate coverage report
task test:coverage     # Alternative coverage command

# Development helpers
task run CLI_ARGS="chat"     # Run commands directly
task test:analyze:lore       # Check medieval naming conventions
make clean            # Clean ALL artifacts including .test files
```

## 📚 Documentation

> **Note**: Documentation is being updated for the MVP release. Current docs may describe planned features.

- [GETTING_STARTED.md](docs/development/GETTING_STARTED.md) - Will be updated post-MVP with demos
- [DEV.md](docs/development/DEV.md) - Developer guidelines
- Additional documentation in `docs/` directory

## 🎯 Current Status & Known Issues

### Working Features

- ✅ Project initialization (`guild init`)
- ✅ Interactive chat interface (`guild chat`)
- ✅ Corpus scanning and indexing
- ✅ Commission creation and refinement
- ✅ Basic agent framework

### Known Issues

- ⚠️ gRPC server has build errors (`pkg/grpc` package)
- ⚠️ Some agent features not fully implemented
- ⚠️ Test coverage needs improvement
- ⚠️ Some commands show as available but aren't fully implemented

### Post-MVP Plans

- Comprehensive documentation with GIFs/demos
- Additional agent types and capabilities
- Advanced orchestration strategies
- Plugin system for custom tools
- Web UI for monitoring

## 🔧 Troubleshooting

### Git Submodule Issues

If you encounter `fatal: this operation must be run in a work tree` when running git commands:

```bash
# Run the fix script
./scripts/fix-git-config.sh
```

Or manually fix:

```bash
git config --file=../.git/modules/guild-core/config core.bare false
git config --file=../.git/modules/guild-core/config core.worktree ../../../guild-core
```

This issue can occur when pre-commit hooks reset the git submodule configuration.

### Pre-commit Hook Issues

If pre-commit hooks fail due to VCS issues:

```bash
# Commit without pre-commit hooks (when necessary)
git commit --no-verify -m "Your commit message"
```

## 🤝 Contributing

Guild is currently in pre-MVP development. Contribution guidelines will be established post-MVP. For now, please open issues for bugs or feature discussions.

## ⚖️ License

Custom License - see [LICENSE](LICENSE) file for details.

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
