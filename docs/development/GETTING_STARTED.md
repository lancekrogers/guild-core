# Getting Started with Guild Framework

Welcome to Guild! This guide will help you get productive with Guild in just 30 seconds.

## Table of Contents

1. [Quick Start (30 Seconds)](#quick-start-30-seconds)
2. [Prerequisites](#prerequisites)
3. [Installation Options](#installation-options)
4. [Basic Usage](#basic-usage)
5. [Available Commands](#available-commands)
6. [Advanced Configuration](#advanced-configuration)
7. [Developer Workflow](#developer-workflow)
8. [Troubleshooting](#troubleshooting)

## Quick Start (30 Seconds)

Get productive with Guild in 3 simple steps:

```bash
# 1. Install Guild (fast, no go vet)
cd guild-core && make install

# 2. Initialize your workspace with Elena
guild init my-project && cd my-project

# 3. Set API key and start chatting
export ANTHROPIC_API_KEY="your-key"
guild chat
```

That's it! You're now chatting with Elena, your AI assistant.

## Prerequisites

- Go 1.23 or higher
- Git
- SQLite
- At least one LLM API key:
  - Anthropic API key (recommended)
  - OpenAI API key
  - Or local Ollama installation

## Installation Options

### For Users (Fast Path)

```bash
# Clone the repository
git clone https://github.com/guild-ventures/guild-core.git
cd guild-core

# Fast install without go vet (30 seconds)
make install

# Verify installation
guild version
```

### For Developers (Full Validation)

```bash
# Clone the repository
git clone https://github.com/guild-ventures/guild-core.git
cd guild-core

# Full build with go vet validation
make build

# Run tests
make test
```

## Basic Usage

### Initialize a Guild Project

```bash
# Fast initialization with Elena agent
guild init my-project
cd my-project

# This creates a .guild/ directory with:
# - guild.yaml (configuration with Elena)
# - memory.db (SQLite database)
# - Various subdirectories for organization
```

### Set Up API Keys

```bash
# Required: Set at least one API key
export ANTHROPIC_API_KEY="your-anthropic-key"
# Optional:
export OPENAI_API_KEY="your-openai-key"
export OLLAMA_HOST="localhost:11434"  # For local models
```

### Start Chat Interface

```bash
# Start chatting immediately
guild chat
```

This opens a production-ready terminal UI where you can:

- Chat with Elena or other AI agents
- Use slash commands for advanced features
- See beautifully formatted responses with markdown support
- Execute tools and commands safely

### Scan Project Documentation

```bash
# Index your project's documentation
guild corpus scan

# Query the indexed documentation
guild corpus query "how does authentication work"
```

## Available Commands

### Core Commands (Production Ready)

- `guild init [path]` - Fast initialization with Elena agent
- `guild chat` - Production-ready interactive chat interface
- `guild setup-wizard` - Advanced TUI configuration interface
- `guild corpus scan` - Scan and index documentation
- `guild corpus query` - Query indexed documentation
- `guild commission` - Create commission documents
- `guild commission refine` - Refine commission documents
- `guild prompt` - Manage prompt templates

### Additional Commands

- `guild campaign` - Campaign management
- `guild cost` - Cost tracking tools
- `guild migrate` - Migration utilities
- `guild serve` - gRPC server (development)
- `guild agent start` - Agent management (development)

## Advanced Configuration

For users who want more control than the default Elena setup:

```bash
# Launch the interactive setup wizard
guild setup-wizard
```

The setup wizard provides:

- **Agent Selection**: Choose from multiple pre-configured agents
- **Custom Agents**: Create your own agent configurations
- **Provider Settings**: Configure multiple LLM providers
- **Advanced Options**: Token limits, temperature, and more
- **Import/Export**: Share configurations between projects

## Developer Workflow

If you're contributing to Guild or need full validation:

```bash
# Use the developer build workflow
make build    # Full build with go vet
make test     # Run comprehensive test suite
make clean    # Clean all artifacts

# Development helpers
make quick    # Fast build without validation
make dashboard # Show project status
```

### Testing Rules

**CRITICAL**: Never use `go test` directly!

```bash
# ❌ WRONG - Creates .test binaries
go test ./...

# ✅ CORRECT - Use make
make test
make unit-test
make integration
```

## Current Limitations

1. **Build Errors**: The project has build failures in several packages:
   - `pkg/grpc` - Interface mismatches with Campaign/Objectives API
   - `cmd/guild` - Some commands disabled due to gRPC dependencies
   - `internal/chat` - Build dependencies (though core functionality works)

2. **Integration Issues**: While most frameworks are implemented, integration is incomplete:
   - Multi-agent orchestration exists but has interface mismatches
   - Kanban board backend works but UI integration has issues
   - Campaign workflows partially implemented

3. **Disabled Features**: Some commands are commented out in main.go:
   - `guild serve` - gRPC server (build errors)
   - `guild agent start` - Agent management commands
   - `guild campaign watch` - Real-time monitoring

4. **Test Coverage**:
   - ~60% coverage (target: 80%+)
   - 8 test files disabled and need migration to internal test packages
   - Some integration tests failing due to interface changes

5. **Documentation Gaps**: Some documentation describes planned features as if implemented

## Troubleshooting

### Build Failures

If you encounter build errors:

```bash
# Clean build artifacts
make clean

# Try building specific components
make build-guild  # Build just the CLI

# Check specific errors
go build ./cmd/guild 2>&1 | head -20
```

### Git Submodule Issues

If working with Guild as a submodule:

```bash
# Fix git configuration
git config --file=../.git/modules/guild-core/config core.bare false
git config --file=../.git/modules/guild-core/config core.worktree ../../../guild-core
```

### API Key Issues

Ensure your API keys are correctly set:

```bash
# Check if keys are set
echo $ANTHROPIC_API_KEY
echo $OPENAI_API_KEY

# The chat interface will show an error if no valid keys are found
```

## Next Steps

1. **Use Chat Interface**: The chat functionality is the most mature feature
2. **Explore Code**: Look at `cmd/guild/chat.go` for the implementation
3. **Check Examples**: See `examples/commissions/` for commission templates
4. **Monitor Development**: This is a beta project with active development

## Contributing

Guild is in active development. Key areas needing work:

- Fixing build errors in gRPC package
- Implementing missing commands
- Improving test coverage
- Documentation updates

---

**Note**: This guide reflects the current state of the Guild Framework. Many planned features described in other documentation are not yet implemented. This document will be updated as features are completed.
