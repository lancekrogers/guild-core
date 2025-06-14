# Guild Framework Demo Guide

> **⚠️ CURRENT STATE**: The Guild Framework is in active development with core functionality implemented but significant limitations. This guide reflects both working features and planned capabilities.

## Current Status

The Guild Framework has substantial infrastructure implemented but several core features have build issues or are incomplete.

### What Works Now
- ✅ **Project initialization** (`guild init`) - Fully functional with auto-detection
- ✅ **Chat interface** - 1,951-line production TUI with streaming, markdown rendering, and tool execution
- ✅ **Corpus scanning and indexing** (`guild corpus scan/query`) - RAG system working
- ✅ **Commission creation and refinement** - Markdown parsing and objective management
- ✅ **Prompt management** - 6-layer prompt system with template support
- ✅ **Multiple LLM providers** - OpenAI, Anthropic, Ollama, DeepSeek, Ora support
- ✅ **SQLite storage** - Full migration from BoltDB complete
- ✅ **Tool execution framework** - Safe workspace isolation implemented

### What Has Issues
- ❌ **gRPC services** - Build failures due to interface mismatches
- ❌ **Multi-agent orchestration** - Framework exists but integration issues
- ⚠️ **Campaign workflows** - Core implemented but some commands disabled
- ⚠️ **Kanban board** - Backend complete, UI has integration issues
- ❌ **Real-time monitoring** (`guild campaign watch`) - Depends on gRPC fixes

## Planned Demo Scenarios (Future Implementation)

The following sections describe the vision for Guild Framework demos once all features are implemented:

---

*[Original demo content follows as planned features...]*

## Overview

This guide provides comprehensive instructions for demonstrating the Guild Framework's multi-agent AI capabilities using the e-commerce platform example. The demos showcase how specialized AI agents work together to build complex software systems.

## Pre-Demo Setup

### Terminal Configuration

1. **Terminal Size**:
   - Width: 120-140 characters
   - Height: 40-50 lines
   - Font: Use a clear monospace font (SF Mono, Fira Code, etc.)

2. **Color Theme**:
   - Recommended: Monokai, Dracula, or Nord
   - Ensure good contrast for recording

3. **Multiple Terminals**:
   - Prepare 2-3 terminal windows/tabs
   - One for `guild campaign watch`
   - One for `guild chat`
   - One for showing output/logs

### Environment Setup

```bash
# Set up environment variables
export OPENAI_API_KEY="your-api-key"
export GUILD_CONFIG="examples/config/e-commerce-guild.yaml"

# Initialize Guild project (if not already done)
cd /path/to/demo/directory
guild init

# Verify setup
guild info
```

### Pre-Demo Checklist

- [ ] API keys configured and working
- [ ] Guild binary in PATH
- [ ] E-commerce commission file present
- [ ] Agent configuration loaded
- [ ] Terminal properly sized
- [ ] Recording software ready (if recording)

## Demo Scenarios

*Note: The rest of this document contains planned scenarios that cannot be executed with the current implementation.*

---

## Working Demo: Current Features

Here's what you can demonstrate with the current implementation:

### 1. Project Initialization Demo
```bash
./bin/guild init demo-project
cd demo-project
ls -la .guild/  # Show created structure
```

### 2. Chat Interface Demo (Main Feature)
```bash
../bin/guild chat
```

**Demonstrates**:
- Professional TUI with streaming responses
- Markdown rendering with syntax highlighting
- Multiple LLM provider support
- Tool execution capability
- Session persistence

### 3. Corpus Management Demo
```bash
../bin/guild corpus scan
../bin/guild corpus query "authentication patterns"
```

**Demonstrates**:
- Document indexing and RAG capabilities
- Vector search functionality
- Project-aware documentation retrieval

### 4. Commission System Demo
```bash
../bin/guild commission create "Build REST API"
../bin/guild commission refine [commission-file]
```

**Demonstrates**:
- Markdown-based commission parsing
- Objective hierarchy management
- Task breakdown capabilities

This provides a solid demonstration of the framework's foundation while being honest about current limitations.