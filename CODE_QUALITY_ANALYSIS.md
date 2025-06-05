# Code Quality Analysis - Guild Core

This document provides a comprehensive analysis of code quality issues in the Guild Core codebase, focusing on context passing, error handling, and registry pattern implementation.

## 1. Context Passing Issues

### Critical Functions Missing Context

#### Storage Package
```go
// Current: pkg/storage/database.go
func NewDatabase(dbPath string) (*Database, error) {
    sqlDB, err := sql.Open("sqlite3", fmt.Sprintf("%s?_foreign_keys=on", dbPath))
    // Missing context for connection and ping
}

// Should be:
func NewDatabase(ctx context.Context, dbPath string) (*Database, error) {
    sqlDB, err := sql.Open("sqlite3", fmt.Sprintf("%s?_foreign_keys=on", dbPath))
    if err != nil {
        return nil, fmt.Errorf("failed to open database: %w", err)
    }
    
    // Use context for ping
    if err := sqlDB.PingContext(ctx); err != nil {
        return nil, fmt.Errorf("failed to ping database: %w", err)
    }
    // ...
}
```

#### Kanban Package
```go
// Current: pkg/kanban/board.go
func LoadBoard(registry ComponentRegistry, boardID string) (*Board, error) {
    // Uses context.Background() internally
    boardInterface, err := boardRepo.GetBoard(context.Background(), boardID)
}

// Should be:
func LoadBoard(ctx context.Context, registry ComponentRegistry, boardID string) (*Board, error) {
    boardInterface, err := boardRepo.GetBoard(ctx, boardID)
}
```

### Functions Using context.Background() Instead of Passed Context
- `pkg/kanban/board.go`: Multiple functions hardcode `context.Background()`
- `pkg/storage/init.go`: Some initialization functions create their own context

## 2. Error Handling Issues

### Missing Error Wrapping

#### Example 1: Storage Package
```go
// Current: pkg/storage/init.go:165
_, err := database.DB().Exec(schema)
return err  // No context

// Should be:
_, err := database.DB().Exec(schema)
if err != nil {
    return fmt.Errorf("failed to execute schema: %w", err)
}
return nil
```

#### Example 2: Agent Manager
```go
// Current: pkg/agent/manager/validator.go:32
if err := v.validateContext(refined.Context); err != nil {
    return err  // No wrapping
}

// Should be:
if err := v.validateContext(refined.Context); err != nil {
    return fmt.Errorf("context validation failed: %w", err)
}
```

### Using errors.New Instead of fmt.Errorf
```go
// Current: pkg/comms/channel/channel.go
return errors.New("publisher is closed")

// Should be:
return fmt.Errorf("publisher is closed")
```

### Error Context Best Practices
```go
// Good error wrapping pattern
func (s *Service) ProcessRequest(ctx context.Context, req *Request) error {
    if err := s.validate(req); err != nil {
        return fmt.Errorf("validation failed for request %s: %w", req.ID, err)
    }
    
    result, err := s.repository.Store(ctx, req)
    if err != nil {
        return fmt.Errorf("failed to store request %s: %w", req.ID, err)
    }
    
    return nil
}
```

## 3. Registry Pattern Violations

### Direct Instantiation Issues

#### Agent Package
```go
// Current: Direct instantiation
func (f *DefaultAgentFactory) CreateWorkerAgent(...) (*WorkerAgent, error) {
    agent := &WorkerAgent{
        id:       id,
        client:   client,
        tools:    f.toolRegistry,  // Direct struct dependency
        memory:   memoryManager,
    }
}

// Should be: Using registry
func (f *DefaultAgentFactory) CreateWorkerAgent(...) (Agent, error) {
    // Get dependencies from registry
    toolRegistry := f.registry.Tools()
    memoryRegistry := f.registry.Memory()
    
    agent := &WorkerAgent{
        id:       id,
        client:   client,
        tools:    toolRegistry,    // Interface dependency
        memory:   memoryManager,
    }
    
    // Register the agent
    f.registry.Agents().RegisterAgent(agent)
    
    return agent, nil
}
```

#### Orchestrator Package
```go
// Current: Direct instantiation
func NewCommissionIntegrationService(registry ComponentRegistry) (*CommissionIntegrationService, error) {
    eventBus := NewEventBus()  // Direct instantiation
    kanbanManager, err := kanban.NewManagerWithRegistry(kanbanAdapter)
}

// Should be: Using registry
func NewCommissionIntegrationService(registry ComponentRegistry) (*CommissionIntegrationService, error) {
    eventBus := registry.Orchestrator().GetEventBus()
    kanbanManager := registry.Orchestrator().GetKanbanManager()
}
```

### Missing Interfaces

Components without interfaces:
- `TaskExtractor` (pkg/agent/manager)
- `IntelligentParser` (pkg/agent/manager)
- `CostManager` (pkg/agent)
- `EventBus` (pkg/orchestrator)
- `TaskDispatcher` (pkg/orchestrator)

### Concrete Type Dependencies

```go
// Current: Concrete type dependency
type ManagerAgent struct {
    toolRegistry *tools.ToolRegistry  // Concrete type
    objectives   *objective.Manager    // Concrete type
}

// Should be: Interface dependency
type ManagerAgent struct {
    toolRegistry tools.Registry        // Interface
    objectives   objective.Repository  // Interface
}
```

## 4. Recommended Fixes

### Phase 1: Error Handling (Immediate)
1. Replace all `errors.New` with `fmt.Errorf`
2. Add error wrapping to all error returns
3. Create domain-specific error types

### Phase 2: Context Propagation (Short-term)
1. Add context parameter to all I/O operations
2. Remove hardcoded `context.Background()` calls
3. Use context for timeouts and cancellation

### Phase 3: Registry Pattern (Medium-term)
1. Define interfaces for all major components
2. Update constructors to use registry
3. Replace concrete dependencies with interfaces
4. Implement missing registries

### Phase 4: Testing and Validation (Ongoing)
1. Add tests for error wrapping
2. Add tests for context cancellation
3. Add tests for registry usage

## 5. Example Refactoring

### Before: Direct instantiation with poor error handling
```go
func CreateTaskProcessor() (*TaskProcessor, error) {
    db, err := storage.NewDatabase("tasks.db")
    if err != nil {
        return nil, err
    }
    
    validator := NewValidator()
    executor := NewExecutor(db)
    
    return &TaskProcessor{
        db:        db,
        validator: validator,
        executor:  executor,
    }, nil
}
```

### After: Registry pattern with proper error handling
```go
func CreateTaskProcessor(ctx context.Context, reg registry.ComponentRegistry) (TaskProcessor, error) {
    // Get dependencies from registry
    db, err := reg.Storage().GetDatabase()
    if err != nil {
        return nil, fmt.Errorf("failed to get database from registry: %w", err)
    }
    
    validator, err := reg.GetValidator("task")
    if err != nil {
        return nil, fmt.Errorf("failed to get task validator: %w", err)
    }
    
    executor, err := reg.GetExecutor("task")
    if err != nil {
        return nil, fmt.Errorf("failed to get task executor: %w", err)
    }
    
    processor := &taskProcessor{
        db:        db,
        validator: validator,
        executor:  executor,
    }
    
    // Register the processor
    if err := reg.RegisterProcessor("task", processor); err != nil {
        return nil, fmt.Errorf("failed to register task processor: %w", err)
    }
    
    return processor, nil
}
```

## 6. Priority Actions

1. **Immediate (This Week)**
   - Fix error wrapping in storage package
   - Add context to database operations
   - Fix error wrapping in kanban package

2. **Short-term (Next 2 Weeks)**
   - Define interfaces for major components
   - Update agent factory to use registry
   - Fix context passing in orchestrator

3. **Medium-term (Next Month)**
   - Complete registry pattern migration
   - Add comprehensive error types
   - Update all constructors to use registry

## 7. Benefits of These Changes

1. **Better Debugging**: Wrapped errors provide full context
2. **Graceful Shutdown**: Context allows proper cancellation
3. **Testability**: Interface dependencies enable mocking
4. **Maintainability**: Registry pattern reduces coupling
5. **Consistency**: Standard patterns across codebase