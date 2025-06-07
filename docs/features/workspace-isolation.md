# Workspace Isolation

The Guild Framework provides workspace isolation for agents using Git worktrees, ensuring that multiple agents can work on tasks concurrently without interfering with each other.

## Overview

Each agent task executes in its own isolated Git worktree, providing:
- **Complete isolation** between concurrent agent tasks
- **Clean working environment** for each task
- **Automatic change tracking** via Git
- **Safe experimentation** without affecting the main codebase

## Architecture

### Components

1. **Workspace Manager** (`pkg/workspace/`)
   - Manages lifecycle of agent workspaces
   - Tracks active workspaces
   - Handles cleanup and resource management

2. **Git Integration** (`pkg/workspace/git_integration.go`)
   - Creates Git worktrees for each task
   - Manages branches and commits
   - Provides Git operations (diff, commit, status)

3. **Task Executor Integration** (`pkg/agent/executor/`)
   - Automatically creates workspaces during task initialization
   - Commits changes during task finalization
   - Stores workspace metadata in execution results

### Directory Structure

```
.guild/
└── workspaces/
    ├── agent-001-20240115-143022/   # Isolated worktree
    ├── agent-002-20240115-143045/   # Another agent's workspace
    └── agent-001-20240115-144512/   # Same agent, different task
```

### Branch Naming Convention

Each workspace gets its own Git branch:
```
agent/{agentID}-{timestamp}
```

Example: `agent/worker-001-20240115-143022`

## Usage

### Basic Workspace Creation

```go
import "github.com/guild-ventures/guild-core/pkg/workspace"

// Create a Git-aware workspace manager
manager, err := workspace.NewGitManager(".guild", "/path/to/repo")
if err != nil {
    log.Fatal(err)
}

// Create workspace for an agent task
opts := workspace.CreateOptions{
    AgentID:      "worker-001",
    BranchPrefix: "agent",
    WorkDir:      ".guild",
    BaseBranch:   "main",
}

ws, err := manager.CreateWorkspace(context.Background(), opts)
if err != nil {
    log.Fatal(err)
}

// Use the workspace
fmt.Printf("Working in: %s\n", ws.Path())
fmt.Printf("On branch: %s\n", ws.Branch())

// Clean up when done
defer ws.Cleanup()
```

### Integration with Task Executor

The task executor automatically manages workspaces:

```go
executor, err := executor.NewBasicTaskExecutor(
    agent,
    kanbanBoard,
    toolRegistry,
    execContext,
    workspaceManager, // Provide workspace manager
)

// Workspace is created automatically during Execute()
result, err := executor.Execute(ctx, task)
```

### Git Operations

When using Git workspaces, additional operations are available:

```go
// Type assert to GitWorkspace for Git operations
if gitWs, ok := ws.(*workspace.GitWorkspace); ok {
    // Check for uncommitted changes
    gitWs.UpdateGitInfo()
    info := gitWs.GetGitInfo()

    if info.IsDirty {
        // Get diff of changes
        diff, _ := gitWs.GetDiff()
        fmt.Printf("Changes:\n%s\n", diff)

        // Commit changes
        err := gitWs.CommitChanges("Task completed: implemented feature X")
        if err != nil {
            log.Printf("Failed to commit: %v", err)
        }
    }
}
```

## Workspace Lifecycle

### 1. Creation Phase
- Git worktree created from base branch (main/master)
- New branch created for the workspace
- Workspace registered with manager

### 2. Active Phase
- Agent works in isolated directory
- All file operations are contained
- Changes tracked by Git automatically

### 3. Finalization Phase
- Uncommitted changes detected
- Optional auto-commit with task metadata
- Workspace info stored in execution results

### 4. Cleanup Phase
- Worktree removed from filesystem
- Branch optionally deleted
- Resources freed

## Configuration

### Workspace Manager Configuration

```yaml
workspace:
  base_dir: ".guild/workspaces"
  max_workspaces: 10
  cleanup_on_exit: true
  cleanup_branches: true
  base_branch: "main"
```

### Environment Variables

- `GUILD_WORKSPACE_DIR`: Override default workspace directory
- `GUILD_WORKSPACE_CLEANUP`: Enable/disable automatic cleanup

## Best Practices

1. **Always Clean Up**: Use `defer ws.Cleanup()` to ensure workspaces are removed
2. **Commit Regularly**: Commit changes at logical checkpoints during task execution
3. **Use Descriptive Messages**: Include task ID and agent ID in commit messages
4. **Handle Conflicts**: Be prepared for merge conflicts when integrating changes back
5. **Monitor Disk Usage**: Workspaces consume disk space; implement retention policies

## Error Handling

Common errors and solutions:

### Workspace Creation Fails
```go
ws, err := manager.CreateWorkspace(ctx, opts)
if err != nil {
    if strings.Contains(err.Error(), "not a git repository") {
        // Not in a Git repo - use basic workspace instead
        basicManager := workspace.NewManager(baseDir, "")
        ws, err = basicManager.CreateWorkspace(ctx, opts)
    }
}
```

### Commit Fails
```go
err := gitWs.CommitChanges("Task complete")
if err != nil {
    // Log but continue - changes still in worktree
    log.Printf("Warning: failed to commit: %v", err)
    // Could attempt to stash or create patch
}
```

## Testing

### Unit Tests
```bash
go test ./pkg/workspace/...
```

### Integration Tests
```go
func TestWorkspaceIsolation(t *testing.T) {
    // Create test repository
    repoPath, cleanup := setupTestRepo(t)
    defer cleanup()

    // Create manager
    manager, err := workspace.NewGitManager(".guild", repoPath)
    require.NoError(t, err)

    // Create multiple workspaces
    ws1, _ := manager.CreateWorkspace(ctx, opts1)
    ws2, _ := manager.CreateWorkspace(ctx, opts2)

    // Verify isolation
    assert.NotEqual(t, ws1.Path(), ws2.Path())
    assert.NotEqual(t, ws1.Branch(), ws2.Branch())
}
```

## Limitations

1. **Git Requirement**: Full isolation requires Git; falls back to basic directories otherwise
2. **Disk Space**: Each worktree is a full checkout; can consume significant space
3. **Performance**: Creating worktrees has overhead; consider pooling for high-frequency tasks
4. **Branch Proliferation**: Many branches created; implement cleanup strategies

## Future Enhancements

- [ ] Worktree pooling for performance
- [ ] Shallow clones for large repositories
- [ ] Integration with cloud storage for artifacts
- [ ] Distributed workspace support across machines
- [ ] Workspace templates and presets
