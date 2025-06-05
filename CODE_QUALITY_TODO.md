# Code Quality TODO List

Based on the analysis, here are the specific tasks that need to be completed to improve code quality:

## High Priority (Fix Immediately)

### 1. Error Wrapping in Critical Paths
- [ ] Fix error returns in `pkg/storage/database.go` - wrap all database errors
- [ ] Fix error returns in `pkg/kanban/board.go` - wrap all board operation errors
- [ ] Fix error returns in `pkg/orchestrator/*.go` - wrap orchestrator errors
- [ ] Fix error returns in `pkg/agent/manager/*.go` - wrap manager errors

### 2. Context Passing for I/O Operations
- [ ] Add context to `LoadBoard()` in `pkg/kanban/board.go`
- [ ] Add context to `ListBoards()` in `pkg/kanban/board.go` 
- [ ] Add context to `NewDatabase()` in `pkg/storage/database.go`
- [ ] Replace all `context.Background()` with passed context

## Medium Priority (Next Sprint)

### 3. Define Missing Interfaces
- [ ] Create `TaskExtractor` interface in `pkg/agent/manager`
- [ ] Create `IntelligentParser` interface in `pkg/agent/manager`
- [ ] Create `CostManager` interface in `pkg/agent`
- [ ] Create `EventBus` interface in `pkg/orchestrator`
- [ ] Create `ToolRegistry` interface in `pkg/tools`

### 4. Update Concrete Dependencies to Interfaces
- [ ] Change `*tools.ToolRegistry` to `tools.Registry` interface
- [ ] Change `*objective.Manager` to `objective.Repository` interface
- [ ] Change `*EventBus` to `EventBus` interface
- [ ] Change `*CostManager` to `CostManager` interface

### 5. Registry Pattern Implementation
- [ ] Implement `OrchestratorRegistry` in `pkg/registry`
- [ ] Implement `ParserRegistry` for managing parsers
- [ ] Update all constructors to use registry for dependencies
- [ ] Remove direct instantiation of components

## Low Priority (Future)

### 6. Create Domain-Specific Error Types
- [ ] Create `StorageError` type with operation context
- [ ] Create `KanbanError` type with board/task context
- [ ] Create `AgentError` type with agent context
- [ ] Update error returns to use typed errors

### 7. Improve Test Coverage
- [ ] Add tests for error wrapping
- [ ] Add tests for context cancellation
- [ ] Add tests for registry usage
- [ ] Add integration tests for full workflows

## Code Examples to Follow

### Error Wrapping Pattern
```go
// Always wrap errors with context
if err := operation(); err != nil {
    return fmt.Errorf("failed to perform operation on %s: %w", id, err)
}
```

### Context Passing Pattern
```go
// Always accept context as first parameter
func ProcessItem(ctx context.Context, id string) error {
    // Check context early
    if err := ctx.Err(); err != nil {
        return fmt.Errorf("context cancelled: %w", err)
    }
    // ... rest of function
}
```

### Interface Definition Pattern
```go
// Define interface in the package that uses it
type TaskProcessor interface {
    ProcessTask(ctx context.Context, task Task) error
}

// Implement in concrete type
type defaultTaskProcessor struct {
    deps Dependencies // interface, not concrete
}
```

### Registry Usage Pattern
```go
// Get dependencies from registry
func NewService(ctx context.Context, reg Registry) (Service, error) {
    repo, err := reg.GetRepository("task")
    if err != nil {
        return nil, fmt.Errorf("failed to get task repository: %w", err)
    }
    
    return &service{repository: repo}, nil
}
```

## Verification Steps

1. Run `grep -r "return err$" pkg/` to find unwrapped errors
2. Run `grep -r "context.Background()" pkg/` to find hardcoded contexts
3. Run `grep -r "New[A-Z]" pkg/` to find direct instantiations
4. Run tests after each change to ensure nothing breaks

## Success Metrics

- Zero instances of `return err` without wrapping
- Zero instances of `context.Background()` in library code
- All major components have interfaces defined
- All constructors use registry pattern
- Tests pass with >80% coverage