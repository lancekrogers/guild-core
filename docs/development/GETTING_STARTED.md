# Getting Started with Guild Framework

Welcome to Guild! This guide reflects the current implementation status of the Guild Framework. Many features are still in development.

## Table of Contents

1. [Prerequisites](#prerequisites)
2. [Installation](#installation)
3. [Basic Usage](#basic-usage)
4. [Available Commands](#available-commands)
5. [Current Limitations](#current-limitations)
6. [Troubleshooting](#troubleshooting)

## Prerequisites

- Go 1.23 or higher
- Git
- SQLite
- At least one LLM API key:
  - Anthropic API key (recommended)
  - OpenAI API key
  - Or local Ollama installation

## Installation

```bash
# Clone the repository
git clone https://github.com/guild-ventures/guild-core.git
cd guild-core

# Install dependencies
make deps

# Build Guild (Note: Build currently has errors in some packages)
make build

# The build will fail but some binaries may be created
# Check if guild binary was created:
ls bin/guild
```

## Basic Usage

### Initialize a Guild Project

```bash
# Create a new guild project
./bin/guild init my-project
cd my-project

# This creates a .guild/ directory with:
# - guild.yaml (configuration)
# - memory.db (SQLite database)
# - Various subdirectories for corpus, campaigns, etc.
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

The chat interface is the most functional part of Guild currently:

```bash
# From your project directory
../bin/guild chat
```

This opens a terminal UI where you can:
- Chat with AI agents
- Use slash commands (some may not work)
- See formatted responses with markdown support

### Scan Project Documentation

```bash
# Index your project's documentation
../bin/guild corpus scan

# Query the indexed documentation
../bin/guild corpus query "how does authentication work"
```

## Available Commands

### Working Commands

- `guild init [path]` - Initialize a new guild project
- `guild chat` - Interactive chat interface (most functional)
- `guild corpus scan` - Scan and index documentation
- `guild corpus query` - Query indexed documentation
- `guild commission` - Create commission documents (basic functionality)
- `guild commission refine` - Refine commission documents
- `guild prompt` - Manage prompt templates

### Partially Working or Not Implemented

- `guild campaign` - Campaign management (limited functionality)
- `guild serve` - gRPC server (has build errors)
- `guild agent start` - Not implemented
- `guild migrate` - Migration utilities

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