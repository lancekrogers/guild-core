# Workspace Package

The workspace package provides git worktree isolation for AI agents working concurrently on the same codebase.

## Overview

When multiple AI agents work on the same repository, they need isolation to prevent conflicts. This package creates separate git worktrees for each agent, allowing them to:

- Work on independent branches
- Make commits without affecting others
- Test changes in isolation
- Coordinate merges through controlled processes

## Current Status

✅ **Implemented:**
- Basic workspace interface and models
- Manager for creating and tracking workspaces
- Lifecycle management (create, get, list, cleanup)
- Automatic cleanup of inactive workspaces
- Configuration structure
- Comprehensive test coverage

⏳ **TODO (Git Integration):**
- Git worktree creation and management
- Branch creation and switching
- Commit operations
- Status tracking (clean/dirty state)
- Merge coordination

## Architecture

```
┌─────────────┐
│   Manager   │ ← Central coordinator
└──────┬──────┘
       │ Creates & manages
       ▼
┌─────────────┐     ┌─────────────┐
│ Workspace 1 │     │ Workspace 2 │
├─────────────┤     ├─────────────┤
│ Agent: A1   │     │ Agent: A2   │
│ Branch: f/1 │     │ Branch: f/2 │
│ Path: /ws/1 │     │ Path: /ws/2 │
└─────────────┘     └─────────────┘
```

## Usage

```go
// Create manager
manager, err := workspace.NewManager("/var/guild/workspaces", "/path/to/repo")

// Create workspace for agent
opts := workspace.CreateOptions{
    AgentID:      "research-agent-1",
    BaseBranch:   "main",
    BranchPrefix: "research",
}

ws, err := manager.CreateWorkspace(ctx, opts)
defer ws.Cleanup()

// Agent works in isolated directory
fmt.Printf("Working in: %s\n", ws.Path())
```

## Integration Points

This package will integrate with:

- **Agent System**: Each agent gets an isolated workspace
- **Orchestrator**: Manages workspace lifecycle with tasks
- **Memory System**: Tracks workspace history and metrics
- **Campaign System**: Coordinates multi-agent work

## Next Steps

1. Implement git worktree operations using go-git library
2. Add branch protection and merge strategies
3. Implement workspace state persistence
4. Add metrics and monitoring
5. Create integration with orchestrator

## Design Decisions

- **File-based isolation**: Each workspace is a separate directory
- **Git worktrees**: Leverages git's built-in worktree feature
- **Automatic cleanup**: Prevents disk space issues
- **Branch naming**: Follows pattern `{prefix}/{agent-id}-{timestamp}`
- **Concurrent access**: Thread-safe manager with mutex protection
