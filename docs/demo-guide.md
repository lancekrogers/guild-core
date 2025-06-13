# Guild Framework Demo Guide (PLANNED FEATURES)

> **⚠️ IMPORTANT**: This document describes planned demo scenarios that are **NOT YET IMPLEMENTED**. The Guild Framework is in active development and most features described here do not currently work.

## Current Status

The Guild Framework has significant functionality implemented but is not yet ready for the demos described in this document. 

### What Works Now
- Basic project initialization (`guild init`)
- Chat interface UI (single agent only)
- Corpus scanning and indexing
- Basic commission creation

### What Doesn't Work
- Multi-agent orchestration
- Campaign workflows
- Real-time task monitoring (`guild campaign watch`)
- Tool execution visualization
- Most commands shown in the demos below

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

## Alternative: Current Working Demo

For a demo with current functionality, you can:

1. Initialize a project: `./bin/guild init demo-project`
2. Start the chat interface: `./bin/guild chat`
3. Show the terminal UI and markdown rendering
4. Explain the vision for multi-agent orchestration

This provides a more honest demonstration of the current state while showing the potential of the framework.