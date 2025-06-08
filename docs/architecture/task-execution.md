# Task Execution Architecture

The Guild Framework implements a sophisticated task execution system that combines layered prompts, workspace isolation, and phased execution to enable AI agents to complete complex tasks reliably.

## Overview

Task execution in Guild follows a structured approach:

1. **Initialization** - Set up isolated workspace and context
2. **Planning** - Generate execution plan using layered prompts
3. **Execution** - Perform the actual work with tool usage
4. **Finalization** - Commit results and clean up resources

## Core Components

### Task Executor (`pkg/agent/executor/`)

The `TaskExecutor` interface defines the contract for task execution:

```go
type TaskExecutor interface {
    Execute(ctx context.Context, task *kanban.Task) (*ExecutionResult, error)
    GetProgress() float64
    GetStatus() ExecutionStatus
    Stop() error
}
```

### Execution Context

Each task execution has rich context:

```go
type ExecutionContext struct {
    WorkspaceDir string                 // Isolated workspace path
    ProjectRoot  string                 // Project root directory
    Objective    string                 // Parent objective/commission
    AgentID      string                 // Executing agent ID
    AgentType    string                 // Agent type (manager, worker)
    Capabilities []string               // Agent capabilities
    Tools        []string               // Available tools
    Metadata     map[string]interface{} // Additional context
}
```

### Execution Result

Detailed results are captured:

```go
type ExecutionResult struct {
    TaskID      string
    Status      ExecutionStatus
    StartTime   time.Time
    EndTime     time.Time
    Duration    time.Duration
    Output      string
    Artifacts   []Artifact
    ToolUsage   []ToolUsage
    Errors      []ExecutionError
    Metadata    map[string]interface{}
}
```

## Layered Prompt System

### Architecture

The prompt system uses composable layers to build context-aware prompts:

1. **Base Layer** - Agent role and capabilities
2. **Context Layer** - Project and commission information
3. **Task Layer** - Specific task requirements
4. **Tool Layer** - Available tools and usage
5. **Execution Layer** - Current step guidance

### Template Structure

Each layer is a Markdown template with Go template syntax:

```markdown
# Base Layer: Agent Role and Capabilities

You are {{.AgentName}}, a Guild Artisan (AI agent) with specialized capabilities.

## Your Role
{{.AgentRole}}

## Core Capabilities
{{range .Capabilities}}
- {{.}}
{{end}}
```

### Prompt Building

```go
// Build prompt with specific layers
promptBuilder := execution.NewCachedPromptBuilder()

// For planning phase - exclude execution layer
planningPrompt, err := promptBuilder.BuildPlanningPrompt(data)

// For execution phase - include all layers
executionPrompt, err := promptBuilder.BuildFullExecutionPrompt(data)
```

### Caching

Prompts are cached for performance:

- Cache key generated from layers + data
- 5-minute TTL by default
- LRU eviction when cache full

## Workspace Isolation

Each task executes in an isolated Git worktree:

### Benefits

- **No conflicts** between concurrent tasks
- **Clean environment** for each execution
- **Change tracking** via Git
- **Easy rollback** if needed

### Integration

```go
// Workspace created automatically in phaseInitialize
if e.workspaceManager != nil {
    opts := workspace.CreateOptions{
        AgentID:      e.agent.GetID(),
        BranchPrefix: "agent",
        WorkDir:      e.execContext.ProjectRoot,
    }

    ws, err := e.workspaceManager.CreateWorkspace(ctx, opts)
    // Workspace path now available in execution context
    e.execContext.WorkspaceDir = ws.Path()
}
```

## Execution Phases

### Phase 1: Initialize (10% progress)

- Create isolated workspace
- Set up execution environment
- Load project context
- Initialize available tools

```go
func (e *BasicTaskExecutor) phaseInitialize(ctx context.Context) error {
    // Create workspace
    workspace, err := e.workspaceManager.Create(e.agent.GetID(), e.currentTask.ID)

    // Update context with workspace
    e.execContext.WorkspaceDir = workspace.GetPath()

    // Initialize tools with workspace context
    // Load project-specific configuration

    return nil
}
```

### Phase 2: Plan (20% progress)

- Build planning prompt with context
- Query agent for execution plan
- Parse and validate plan
- Store plan in metadata

```go
func (e *BasicTaskExecutor) phasePlan(ctx context.Context) error {
    // Build layered prompt
    promptData := e.buildPromptData()
    planningPrompt, err := e.promptBuilder.BuildPlanningPrompt(promptData)

    // Add planning instructions
    fullPrompt := planningPrompt + planningInstructions

    // Get plan from agent
    plan, err := e.agent.Execute(ctx, fullPrompt)

    // Store for execution phase
    e.result.Metadata["execution_plan"] = plan

    return nil
}
```

### Phase 3: Execute (70% progress)

- Execute plan steps
- Use tools as needed
- Track progress
- Generate artifacts

```go
func (e *BasicTaskExecutor) phaseExecute(ctx context.Context) error {
    // Parse execution plan
    steps := e.parseExecutionPlan()

    for i, step := range steps {
        // Update progress
        progress := 0.2 + (0.5 * float64(i+1) / float64(len(steps)))
        e.updateProgress(progress, step.Name)

        // Execute step with tools
        result, err := e.executeStep(ctx, step)

        // Track artifacts
        e.collectArtifacts(result)
    }

    return nil
}
```

### Phase 4: Finalize (90% progress)

- Commit workspace changes
- Collect all artifacts
- Generate execution report
- Clean up temporary resources

```go
func (e *BasicTaskExecutor) phaseFinalize(ctx context.Context) error {
    // Check for uncommitted changes
    if gitWs, ok := e.workspace.(*workspace.GitWorkspace); ok {
        if gitWs.GetGitInfo().IsDirty {
            // Auto-commit changes
            commitMsg := fmt.Sprintf("Task %s: Completed by %s",
                e.currentTask.ID, e.agent.GetID())
            gitWs.CommitChanges(commitMsg)
        }
    }

    // Store workspace info in results
    e.result.Metadata["workspace_path"] = e.workspace.Path()
    e.result.Metadata["workspace_branch"] = e.workspace.Branch()

    return nil
}
```

## State Management

### Execution States

```go
const (
    StatusInitializing ExecutionStatus = "initializing"
    StatusRunning      ExecutionStatus = "running"
    StatusPaused       ExecutionStatus = "paused"
    StatusCompleted    ExecutionStatus = "completed"
    StatusFailed       ExecutionStatus = "failed"
    StatusStopped      ExecutionStatus = "stopped"
)
```

### Kanban Integration

Task status is synchronized with the kanban board:

```go
// Update kanban when starting
e.updateTaskStatus(ctx, kanban.StatusInProgress, "Starting execution")

// Update on completion
e.updateTaskStatus(ctx, kanban.StatusReadyForReview, "Task completed")

// Update on failure
e.updateTaskStatus(ctx, kanban.StatusBlocked, "Execution failed")
```

## Error Handling

### Graceful Degradation

```go
// Workspace creation fails - continue without isolation
if err := createWorkspace(); err != nil {
    log.Printf("Warning: workspace creation failed: %v", err)
    e.execContext.WorkspaceDir = e.execContext.ProjectRoot
}

// Tool execution fails - try alternatives
if err := primaryTool.Execute(); err != nil {
    if fallbackTool != nil {
        result, err = fallbackTool.Execute()
    }
}
```

### Error Collection

All errors are collected in the execution result:

```go
type ExecutionError struct {
    Phase     string    // Which phase failed
    Error     string    // Error message
    Timestamp time.Time // When it occurred
    Retryable bool      // Can it be retried
}
```

## Monitoring and Observability

### Progress Tracking

Real-time progress updates:

```go
e.updateProgress(0.5, "Processing files")
// Emits progress event for UI updates
```

### Execution Logs

Structured logging throughout:

```go
e.addExecutionLog("Created execution plan", map[string]interface{}{
    "plan_length": len(plan),
    "agent_id":    e.agent.GetID(),
})
```

### Metrics Collection

Key metrics tracked:

- Execution duration
- Tool usage frequency
- Artifact generation
- Error rates

## Best Practices

1. **Always use context**: Pass context through all phases for cancellation
2. **Update progress frequently**: Keep UI responsive with progress updates
3. **Commit early and often**: Make logical commits during execution
4. **Handle errors gracefully**: Log but continue when possible
5. **Clean up resources**: Use defer for cleanup operations

## Example Usage

```go
// Create executor with all dependencies
executor, err := executor.NewBasicTaskExecutor(
    agent,
    kanbanBoard,
    toolRegistry,
    execContext,
    workspaceManager,
)

// Execute task
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
defer cancel()

result, err := executor.Execute(ctx, task)
if err != nil {
    log.Printf("Task failed: %v", err)
    // Result still contains partial progress
}

// Check results
fmt.Printf("Task %s completed in %v\n", result.TaskID, result.Duration)
fmt.Printf("Generated %d artifacts\n", len(result.Artifacts))
fmt.Printf("Workspace: %s\n", result.Metadata["workspace_path"])
```

## Future Enhancements

- [ ] Streaming execution updates via gRPC
- [ ] Checkpoint/resume for long-running tasks
- [ ] Parallel step execution where possible
- [ ] Tool usage optimization based on history
- [ ] Execution replay for debugging
