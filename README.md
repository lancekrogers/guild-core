# Guild: Collaborative AI Agent Framework

<div align="center">
  <strong>Orchestrate agents working together in guilds</strong>
  <br>
  <br>

[![Go Version](https://img.shields.io/badge/Go-1.24-00ADD8.svg)](https://go.dev/dl/)
[![License](https://img.shields.io/badge/License-Proprietary-red.svg)]()

</div>

---

## 🧠 Overview

Guild is a Go-based framework for orchestrating multiple AI agents that work together on complex tasks. Inspired by the historical tradition of medieval guilds — associations of skilled craftspeople who governed their trade — Guild provides a structured approach to agentic workflow management.

**Key Features:**

- 🤖 **Multi-Agent Coordination**: Organize agents with different specializations in collaborative guilds
- 📋 **Kanban Task Management**: Track tasks through their lifecycle with a familiar board interface
- 💾 **Memory & RAG**: Store and retrieve context with BoltDB and vector search via Qdrant
- 🔌 **Multiple LLM Providers**: Support for OpenAI, Anthropic, Ollama (local), and Ora
- 🛠️ **Tool Integration**: Seamless integration with CLI tools and external services
- 💰 **Treasury Management**: Built-in cost tracking, budgeting, and optimization for both API and local models
- 🧩 **Human-in-the-Loop**: Block tasks for human input when needed

## 🚀 Installation

### Using Go

```bash
# Clone the repository
git clone https://github.com/yourusername/guild.git
cd guild

# Install dependencies
go mod download

# Build the CLI
go build -o bin/guild cmd/guild/main.go

# Add to your PATH
export PATH=$PATH:$(pwd)/bin
```

### Using Binaries

Binary releases will be available in the future.

## 🚀 Quick Start

### Create a New Guild Project

```bash
# Initialize a new Guild project
guild init my-project

# Change to the project directory
cd my-project
```

This creates a `.guild/` directory in your project, similar to how Git creates `.git/`. The `.guild/` directory contains all project-local configuration and data:

```
.guild/
├── config.yaml        # Project-specific configuration
├── corpus/           # Knowledge base and documentation
├── embeddings/       # Vector embeddings for RAG
├── agents/           # Agent configurations
├── objectives/       # Project objectives
├── memory/           # BoltDB storage
└── cache/           # Temporary files and caches
```

**Important**: The `.guild/` directory should be added to your `.gitignore` file as it contains machine-specific data and potentially sensitive information.

### Configure Agents and Models

Edit the `guild.yaml` file to configure your agents and models:

```yaml
agents:
  - name: planner
    provider: anthropic
    model: claude-3-opus
    tools:
      - search-web
      - file-reader

  - name: implementer
    provider: ollama
    model: llama3-8b
    tools:
      - file-writer
      - aider

guilds:
  - name: content-team
    agents:
      - planner
      - implementer
    manager: planner
    objectives_path: objectives

costs:
  api_models:
    claude-3-opus: 45
    gpt-4: 60
  local_models:
    llama3-8b: 1
  cli_tools:
    default: 0
```

### Define Objectives

Create an objective file in the `objectives` directory:

```markdown
# 🧠 Goal

Create a technical blog post about AI agent frameworks.

# 📂 Context

This is the first in a series of technical blog posts about AI tools and frameworks.
The target audience is technical practitioners in the AI space.

# 🔧 Requirements

- Overview of agent frameworks
- Comparison of key frameworks
- Code examples in Python or JavaScript
- Focus on practical implementation
- Target length: 1500-2000 words

# 📌 Tags

- ai
- agents
- tutorial
- technical

# 🔗 Related

- None
```

### Run the Guild

```bash
# Run the guild with the defined objective
guild run content-team

# View the Kanban board
guild kanban

# Interact with blocked tasks
guild task unblock task-123 --input "Focus more on practical examples"
```

## 🏗️ Architecture

Guild is built on a component-based architecture inspired by medieval guild structures:

- **Agents** (Artisans): LLM-powered workers that execute specialized tasks
- **Guilds**: Teams of artisans working together on shared objectives
- **Kanban** (Workshop Board): Task tracking system with state management
- **Memory** (Archives): Persistence layer with vector search capabilities
- **Tools** (Implements): External capabilities that agents can use
- **Objectives** (Commissions): Structured goals defined in markdown
- **Guild Hall UI**: Medieval-themed terminal UI for objective management

## 📚 Documentation

Comprehensive documentation is available in the `docs` directory:

- [Project Structure](docs/project-structure.md) - Understanding the `.guild/` directory
- [User Guide](docs/user-guide.md)
- [Architecture](docs/architecture.md)
- [Configuration](docs/configuration.md)
- [Agent Types](docs/agent-types.md)
- [Tool Integration](docs/tool-integration.md)
- [API Reference](docs/api-reference.md)

## 🧩 Examples

Guild includes several example projects:

- [Content Creation](examples/content-guild)
- [Software Development](examples/dev-guild)
- [Data Analysis](examples/analysis-guild)

## 🛠️ Development

### Building from Source

```bash
# Clone the repository
git clone https://github.com/yourusername/guild.git
cd guild

# Using Taskfile (recommended)
# Install Task if you don't have it: https://taskfile.dev/installation/
task build      # Build the CLI
task test       # Run tests
task version    # Display version

# Manual build
go build -o bin/guild cmd/guild/main.go
```

### Using Taskfile

Guild uses [Taskfile](https://taskfile.dev/) to simplify common development tasks:

```bash
# Install dependencies (including ZeroMQ)
task deps:install

# Build the project
task build

# Run the application
task run CLI_ARGS="--help"

# Run tests
task test                     # Run all tests
task test:unit                # Run unit tests only
task test:packages            # Run tests for specific packages

# Test Analysis & Quality
task test:coverage            # Generate basic coverage report
task test:coverage:working    # Coverage for working packages only
task test:coverage:detailed   # Detailed coverage by package
task test:coverage:badge      # Generate coverage badge
task test:verify              # Identify untested functions
task test:verify:all          # Verify all working packages
task test:analyze             # Analyze test patterns and quality
task test:analyze:lore        # Check adherence to Guild naming conventions
task test:lint:naming         # Lint tests for naming compliance
task test:report              # Generate comprehensive test quality report

# Clean build artifacts
task clean

# Show version
task version

# Objective commands
task objective:create CLI_ARGS="My new objective"
task objective:list
task objective:view CLI_ARGS="objective-id"
task objective:ui      # Launch the Guild Hall UI

# Agent commands
task agent CLI_ARGS="start agent-id"

# Guild Hall UI Development
task ui:dev:setup           # Install Guild Hall UI dependencies (Bubble Tea)
task ui:dev:run             # Run Guild Hall in craftsman's workshop mode (hot reloading)
task ui:scaffold COMPONENT=task_list TYPE=ledger  # Craft a new Guild Hall component
task ui:test                # Verify Guild Hall components for lore compliance
task ui:styleguide          # Generate Guild Hall style manuscript
task ui:docs:generate       # Scribe Guild Hall documentation
task ui:demo                # Visit the Guild Hall demonstration
```

### Project Structure

```
guild/
├── cmd/                 # Command-line applications
│   └── guild/           # Guild CLI
├── pkg/                 # Core packages
│   ├── agent/           # Agent implementations
│   ├── comms/           # Communication protocols
│   ├── config/          # Configuration handling
│   ├── kanban/          # Task management
│   ├── memory/          # Storage interfaces
│   ├── objective/       # Objective parsing
│   ├── orchestrator/    # Guild coordination
│   └── providers/       # LLM providers
├── tools/               # Tool implementations
├── examples/            # Example guild configurations
└── docs/                # Documentation
```

## ⚖️ License

This project is proprietary software intended for private business use. A decision to open source may be made in the future.

## 💬 Contributing

As Guild is currently a private project, contributions are by invitation only. If Guild becomes open source in the future, contribution guidelines will be published.

## 🙏 Acknowledgments

Guild draws inspiration from several projects and concepts:

- [CrewAI](https://github.com/joaomdmoura/crewAI)
- [LangChain](https://github.com/langchain-ai/langchain)
- [AutoGen](https://github.com/microsoft/autogen)
- Medieval guild structures and collaboration models

---

<div align="center">
  <i>Built with ❤️ for the age of agentic workflows</i>
</div>
 later
- BoltDB for storage
- (Optional) Qdrant for vector search
