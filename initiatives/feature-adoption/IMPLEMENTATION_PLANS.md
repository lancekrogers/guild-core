# Implementation Plans for Validated Features

## Feature 1: General Confirmation Framework

### Overview
Add a reusable confirmation system for dangerous operations to prevent accidental data loss and build user trust.

### Current State
- Only exists in `corpus.go` for document deletion
- No session persistence
- No "don't ask again" functionality

### Implementation Plan

#### 1. Core Confirmation Service (4 hours)
```go
// pkg/confirmation/service.go
package confirmation

type ConfirmationService struct {
    sessionStore map[string]bool // Track "don't ask again" preferences
    mu          sync.RWMutex
}

type ConfirmationRequest struct {
    ID          string   // Unique operation ID
    Message     string   // What to confirm
    Details     []string // Additional context
    Destructive bool     // Is this a dangerous operation?
    SessionKey  string   // For "don't ask again" functionality
}

func (s *ConfirmationService) RequiresConfirmation(req ConfirmationRequest) bool
func (s *ConfirmationService) Confirm(req ConfirmationRequest) (bool, error)
func (s *ConfirmationService) SetSessionPreference(sessionKey string, dontAskAgain bool)
```

#### 2. Chat UI Integration (2 hours)
```go
// internal/ui/chat/confirmation.go
type ConfirmationDialog struct {
    tea.Model
    request     ConfirmationRequest
    response    chan bool
    dontAskAgain bool
}

// Renders:
// ⚠️  Confirmation Required
// ────────────────────────
// This operation will overwrite 3 files:
//   • src/main.go
//   • src/config.go  
//   • src/utils.go
//
// Continue? [y/N] 
// [ ] Don't ask again this session
```

#### 3. Integration Points (2 hours)
- File operations in tools package
- Commission state changes
- High-cost LLM operations
- Kanban board deletions

### Testing Strategy
- Unit tests for service logic
- Integration tests with chat UI
- E2E tests for file operations

### Success Metrics
- Zero accidental file overwrites
- 90% user satisfaction with confirmation flow
- No performance impact on operations

---

## Feature 2: Task Dependency Resolution

### Overview
Enhance the orchestrator to automatically resolve task dependencies and execute tasks in optimal order.

### Current State
```go
type Task struct {
    Dependencies []string // Exists but not used
}
```

### Implementation Plan

#### 1. Dependency Graph Builder (6 hours)
```go
// pkg/orchestrator/dependencies/graph.go
package dependencies

type DependencyGraph struct {
    nodes map[string]*TaskNode
    edges map[string][]string
}

type TaskNode struct {
    Task         *Task
    Dependencies []*TaskNode
    Dependents   []*TaskNode
    Status       TaskStatus
}

func (g *DependencyGraph) AddTask(task *Task) error
func (g *DependencyGraph) ValidateNoCycles() error
func (g *DependencyGraph) GetExecutionOrder() [][]*Task // Returns batches
func (g *DependencyGraph) GetReadyTasks() []*Task
```

#### 2. Enhanced Dispatcher (4 hours)
```go
// pkg/orchestrator/dispatcher.go updates
type taskDispatcher struct {
    // ... existing fields ...
    depGraph *dependencies.DependencyGraph
}

func (d *taskDispatcher) DispatchWithDependencies(ctx context.Context) error {
    // 1. Build dependency graph
    // 2. Validate no cycles
    // 3. Get initial ready tasks
    // 4. Dispatch ready tasks in parallel
    // 5. Update graph on completion
    // 6. Get next batch of ready tasks
    // 7. Repeat until all complete
}
```

#### 3. Event Integration (2 hours)
```go
// Emit new events
- TaskDependencyMet
- TaskBlocked
- DependencyCycleDetected
```

#### 4. Kanban UI Updates (2 hours)
- Show task dependencies visually
- Indicate blocked tasks
- Display dependency chains

### Example Usage
```yaml
commission:
  tasks:
    - id: design-api
      title: Design REST API
      
    - id: implement-api
      title: Implement API endpoints
      dependencies: [design-api]
      
    - id: write-tests
      title: Write API tests
      dependencies: [design-api]
      
    - id: deploy
      title: Deploy to staging
      dependencies: [implement-api, write-tests]
```

### Testing Strategy
- Unit tests for graph algorithms
- Cycle detection tests
- Parallel execution tests
- Integration with existing dispatcher

### Success Metrics
- 30% faster commission completion for dependent tasks
- Zero dependency deadlocks
- Improved resource utilization

---

## Implementation Timeline

### Week 1: Confirmation Framework
- Day 1-2: Core service implementation
- Day 3: Chat UI integration
- Day 4: Integration with operations
- Day 5: Testing and polish

### Week 2: Dependency Resolution  
- Day 1-2: Dependency graph implementation
- Day 3: Dispatcher enhancement
- Day 4: Event and UI integration
- Day 5: Testing and documentation

## Risk Mitigation

### Confirmation Framework Risks
- **Risk**: UI complexity in TUI environment
- **Mitigation**: Start with simple y/n, add features incrementally

### Dependency Resolution Risks
- **Risk**: Complex dependency cycles
- **Mitigation**: Comprehensive cycle detection with clear error messages
- **Risk**: Performance impact on large graphs
- **Mitigation**: Optimize for common case (< 20 tasks)

## Alternative Approach

Given the focus on shipping in 3-4 weeks, consider:
1. **Skip these features entirely** - Guild works without them
2. **Focus only on integration** - Make existing features work better
3. **Add features post-launch** - Based on user feedback

## Recommendation

**Focus on integration over new features**. These enhancements are nice-to-have but not critical for launch. The time would be better spent:
1. Fixing the 9 failing test packages
2. Creating real LLM agent implementations  
3. Connecting chat UI to orchestrator
4. Writing documentation and examples

If time permits after core integration, the confirmation framework (3 days) would be the highest value addition.