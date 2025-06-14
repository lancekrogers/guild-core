# Installation Guide

> **⚠️ BUILD STATUS**: Guild is in active development with build errors in some packages. Core functionality works but full build may fail.

## Prerequisites

- Go 1.23 or later (required for latest features)
- Git
- SQLite3 (for database functionality)
- Make (required for proper building)
- At least one LLM API key:
  - Anthropic API key (recommended)
  - OpenAI API key
  - Or local Ollama installation

## Installing Guild

### From Source (Recommended)

```bash
# Clone the repository
git clone https://github.com/guild-ventures/guild-core.git
cd guild-core

# Install dependencies
make deps

# Build the CLI (may show errors but often succeeds)
make build

# Check if guild binary was created
ls -la bin/guild

# If make build fails, try building just the CLI
go build -o bin/guild cmd/guild/main.go

# Test the installation
./bin/guild --help
```

### Current Build Issues

The following packages currently have build errors:
- `pkg/grpc` - Interface mismatches with Campaign/Objectives
- `cmd/guild` - Some commands disabled due to gRPC issues  
- `internal/chat` - Build dependencies

**Workaround**: The main guild binary often builds successfully despite these errors.

### Using Go Install (Not Recommended)

```bash
# This may fail due to current build issues
go install github.com/guild-ventures/guild-core/cmd/guild@latest
```

**Note**: `go install` is not recommended currently due to build errors. Use the source installation method above.

## Setup and Verification

### 1. Set API Keys

```bash
# Required: Set at least one API key
export ANTHROPIC_API_KEY="your-anthropic-key"
# Optional:
export OPENAI_API_KEY="your-openai-key"
export OLLAMA_HOST="localhost:11434"  # For local models
```

### 2. Initialize a Project

```bash
# Create a test project
./bin/guild init test-project
cd test-project

# Check that .guild directory was created
ls -la .guild/
```

### 3. Test Core Functionality

```bash
# Test chat interface (main feature)
../bin/guild chat

# Test corpus scanning
../bin/guild corpus scan

# Test help system
../bin/guild --help
```

## Next Steps

- [Getting Started Guide](../development/GETTING_STARTED.md) - Current state and limitations
- [Demo Guide](../demo-guide.md) - What works now vs planned features
- [Current Status](../CURRENT_STATUS.md) - Comprehensive status overview
