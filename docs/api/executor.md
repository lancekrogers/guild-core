# Task Executor API Reference

## Package: `github.com/guild-ventures/guild-core/pkg/agent/executor`

The executor package provides task execution capabilities for Guild agents, including workspace isolation, layered prompts, and phased execution.

## Interfaces

### TaskExecutor

The main interface for task execution.

```go
type TaskExecutor interface {
    // Execute runs the task execution loop for a given task
    Execute(ctx context.Context, task *kanban.Task) (*ExecutionResult, error)
    
    // GetProgress returns the current execution progress (0.0 to 1.0)
    GetProgress() float64
    
    // GetStatus returns the current execution status
    GetStatus() ExecutionStatus
    
    // Stop gracefully stops the execution
    Stop() error
}
```

## Types

### ExecutionStatus

Represents the current state of task execution.

```go
type ExecutionStatus string

const (
    StatusInitializing ExecutionStatus = "initializing"
    StatusRunning      ExecutionStatus = "running"
    StatusPaused       ExecutionStatus = "paused"
    StatusCompleted    ExecutionStatus = "completed"
    StatusFailed       ExecutionStatus = "failed"
    StatusStopped      ExecutionStatus = "stopped"
)
```

### ExecutionContext

Provides context for task execution.

```go
type ExecutionContext struct {
    WorkspaceDir string                 // Isolated workspace directory
    ProjectRoot  string                 // Project root directory
    Objective    string                 // Parent objective description
    AgentID      string                 // Executing agent ID
    AgentType    string                 // Agent type (manager, worker, etc)
    Capabilities []string               // Agent capabilities
    Tools        []string               // Available tools
    Metadata     map[string]interface{} // Additional context
}
```

### ExecutionResult

Contains the outcome of a task execution.

```go
type ExecutionResult struct {
    TaskID      string                 `json:"task_id"`
    Status      ExecutionStatus        `json:"status"`
    StartTime   time.Time              `json:"start_time"`
    EndTime     time.Time              `json:"end_time"`
    Duration    time.Duration          `json:"duration"`
    Output      string                 `json:"output"`
    Artifacts   []Artifact             `json:"artifacts"`
    ToolUsage   []ToolUsage            `json:"tool_usage"`
    Errors      []ExecutionError       `json:"errors,omitempty"`
    Metadata    map[string]interface{} `json:"metadata,omitempty"`
}
```

### Artifact

Represents a file or resource created during execution.

```go
type Artifact struct {
    Name        string    `json:"name"`
    Type        string    `json:"type"`
    Path        string    `json:"path"`
    Size        int64     `json:"size"`
    CreatedAt   time.Time `json:"created_at"`
    Description string    `json:"description,omitempty"`
}
```

### ToolUsage

Tracks which tools were used during execution.

```go
type ToolUsage struct {
    ToolName    string                 `json:"tool_name"`
    Invocations int                    `json:"invocations"`
    TotalTime   time.Duration          `json:"total_time"`
    Results     []map[string]interface{} `json:"results,omitempty"`
}
```

### ExecutionError

Represents an error that occurred during execution.

```go
type ExecutionError struct {
    Phase     string    `json:"phase"`
    Error     string    `json:"error"`
    Timestamp time.Time `json:"timestamp"`
    Retryable bool      `json:"retryable"`
}
```

## Implementations

### BasicTaskExecutor

The standard implementation of TaskExecutor.

```go
func NewBasicTaskExecutor(
    agent agent.Agent,
    kanbanBoard *kanban.Board,
    toolRegistry *tools.ToolRegistry,
    execContext *ExecutionContext,
    workspaceManager workspace.Manager,
) (*BasicTaskExecutor, error)
```

**Parameters:**
- `agent`: The AI agent that will execute the task
- `kanbanBoard`: Kanban board for task status updates (can be nil)
- `toolRegistry`: Registry of available tools (can be nil)
- `execContext`: Execution context with project information
- `workspaceManager`: Manager for workspace isolation (can be nil)

**Returns:**
- `*BasicTaskExecutor`: The executor instance
- `error`: Creation error, if any

## Usage Examples

### Basic Task Execution

```go
// Create execution context
execContext := &executor.ExecutionContext{
    ProjectRoot:  "/path/to/project",
    Objective:    "Implement user authentication",
    AgentID:      "worker-001",
    AgentType:    "worker",
    Capabilities: []string{"coding", "testing"},
    Tools:        []string{"file_system", "shell"},
}

// Create executor
exec, err := executor.NewBasicTaskExecutor(
    agent,
    kanbanBoard,
    toolRegistry,
    execContext,
    workspaceManager,
)
if err != nil {
    log.Fatal(err)
}

// Execute task
result, err := exec.Execute(ctx, task)
if err != nil {
    log.Printf("Execution failed: %v", err)
}

// Check results
fmt.Printf("Status: %s\n", result.Status)
fmt.Printf("Duration: %v\n", result.Duration)
fmt.Printf("Artifacts: %d\n", len(result.Artifacts))
```

### With Progress Monitoring

```go
// Start execution in background
go func() {
    result, err := exec.Execute(ctx, task)
    // Handle completion
}()

// Monitor progress
ticker := time.NewTicker(1 * time.Second)
defer ticker.Stop()

for {
    select {
    case <-ticker.C:
        progress := exec.GetProgress()
        status := exec.GetStatus()
        fmt.Printf("Progress: %.0f%% [%s]\n", progress*100, status)
        
        if status == executor.StatusCompleted || 
           status == executor.StatusFailed {
            return
        }
    case <-ctx.Done():
        exec.Stop()
        return
    }
}
```

### Without Workspace Isolation

```go
// Create executor without workspace manager
exec, err := executor.NewBasicTaskExecutor(
    agent,
    kanbanBoard,
    toolRegistry,
    execContext,
    nil, // No workspace manager
)

// Task will execute in project root directory
result, err := exec.Execute(ctx, task)
```

### Handling Execution Errors

```go
result, err := exec.Execute(ctx, task)
if err != nil {
    // Execution failed, but result may contain partial progress
    if result != nil {
        fmt.Printf("Failed at: %s\n", result.Status)
        fmt.Printf("Errors:\n")
        for _, execErr := range result.Errors {
            fmt.Printf("  [%s] %s (retryable: %v)\n",
                execErr.Phase,
                execErr.Error,
                execErr.Retryable,
            )
        }
    }
    return err
}
```

### Accessing Workspace Information

```go
result, err := exec.Execute(ctx, task)
if err == nil {
    // Get workspace details from metadata
    if wsPath, ok := result.Metadata["workspace_path"].(string); ok {
        fmt.Printf("Task executed in: %s\n", wsPath)
    }
    
    if wsBranch, ok := result.Metadata["workspace_branch"].(string); ok {
        fmt.Printf("Git branch: %s\n", wsBranch)
    }
    
    if plan, ok := result.Metadata["execution_plan"].(string); ok {
        fmt.Printf("Execution plan:\n%s\n", plan)
    }
}
```

## Integration with Prompts

The executor uses the layered prompt system from `internal/prompts/agent/execution/`:

### Available Prompt Layers

1. **base_layer.md** - Agent identity and capabilities
2. **context_layer.md** - Project and commission context
3. **task_layer.md** - Specific task requirements
4. **tool_layer.md** - Available tools and usage
5. **execution_layer.md** - Current execution phase guidance

### Customizing Prompts

Prompts can be customized by modifying the template files:

```bash
internal/prompts/agent/execution/
├── base_layer.md
├── context_layer.md
├── task_layer.md
├── tool_layer.md
└── execution_layer.md
```

## Thread Safety

The BasicTaskExecutor is thread-safe and can be safely accessed from multiple goroutines:

- Progress and status queries are read-locked
- State updates are write-locked
- Stop() can be called from any goroutine

## Performance Considerations

### Prompt Caching

Prompts are cached for 5 minutes by default:
- Reduces template parsing overhead
- Improves response time for repeated tasks
- Cache size limited to 100 entries

### Workspace Creation

Workspace creation has overhead:
- Git worktree creation: ~100-500ms
- Consider pooling for high-frequency tasks
- Can disable with nil workspace manager

### Memory Usage

- Execution logs are kept in memory
- Large artifacts should be written to disk
- Consider result size for long-running tasks

## Error Handling

The executor follows these error handling principles:

1. **Graceful degradation** - Continue when possible
2. **Error collection** - All errors stored in result
3. **Partial results** - Return what was completed
4. **Retryable flags** - Indicate which errors can retry

## Testing

### Unit Testing

```go
func TestExecutor(t *testing.T) {
    // Create mock dependencies
    mockAgent := &mockAgent{id: "test-agent"}
    execContext := &executor.ExecutionContext{
        AgentID: "test-agent",
    }
    
    // Create executor
    exec, err := executor.NewBasicTaskExecutor(
        mockAgent,
        nil, // No kanban board
        nil, // No tools
        execContext,
        nil, // No workspace
    )
    require.NoError(t, err)
    
    // Create test task
    task := &kanban.Task{
        ID:    "test-task",
        Title: "Test Task",
    }
    
    // Execute
    result, err := exec.Execute(context.Background(), task)
    assert.NoError(t, err)
    assert.Equal(t, executor.StatusCompleted, result.Status)
}
```

### Integration Testing

See `pkg/agent/executor/task_executor_test.go` for comprehensive tests.