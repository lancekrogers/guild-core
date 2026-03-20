# Agent Status Tracking System

The agent status tracking system provides real-time monitoring and visualization of agent activities within the Guild Framework chat interface.

## Components

### 1. StatusTracker (`tracker.go`)

The core tracking component that maintains agent state and activity history.

**Features:**

- Agent registration and lifecycle management
- Status updates with reason tracking
- Activity logging with configurable history size
- Statistics collection and reporting
- Inactive agent purging

**Usage:**

```go
tracker := status.NewStatusTracker(ctx)

// Register an agent
tracker.RegisterAgent(status.AgentInfo{
    ID:     "agent-1",
    Name:   "Task Manager",
    Type:   "manager",
    Status: status.StatusIdle,
})

// Update status
tracker.UpdateAgentStatus("agent-1", status.StatusWorking, "Processing commission")

// Get statistics
stats := tracker.GetStats()
fmt.Printf("Active agents: %d/%d\n", stats.ActiveAgents, stats.TotalAgents)
```

### 2. AgentDisplay (`display.go`)

Formatting component for rendering agent status in various formats.

**Features:**

- Multiple display formats (detailed, compact, summary)
- Status-based color coding using lipgloss
- Icon mapping for visual status indicators
- Agent grouping by status

**Usage:**

```go
display := status.NewAgentDisplay()

// Format single agent
formatted := display.FormatAgentCompact(agentInfo)

// Format agent list
agents, _ := tracker.GetAllAgents()
list := display.FormatAgentList(agents)
```

### 3. IndicatorManager (`indicators.go`)

Manages animated indicators for active agents.

**Indicator Types:**

- **Spinner**: Rotating animation for general activity
- **Pulse**: Pulsing dot for thinking/processing
- **Progress**: Progress bar for task execution
- **Dots**: Animated dots for loading states

**Usage:**

```go
indicators := status.NewIndicatorManager()

// Set indicator for working agent
indicators.SetIndicator("agent-1", status.IndicatorProgress, status.StatusWorking)

// Update animations (call periodically)
indicators.Update()

// Get current frame
frame := indicators.GetIndicator("agent-1")
```

### 4. StatusIntegration (`integration.go`)

Connects the status tracking system to the UI and orchestrator events.

**Features:**

- Orchestrator event handling
- Bubble Tea message processing
- Automatic UI updates
- Status pane integration

**Usage:**

```go
integration, _ := status.NewStatusIntegration(ctx, statusPane)

// Handle orchestrator events
integration.HandleOrchestratorEvent(event)

// Process UI updates
cmd := integration.Update(timeMsg)
```

## Agent Status Types

- **Idle** (🟢): Agent is available but not working
- **Thinking** (🤔): Agent is processing or planning
- **Working** (⚙️): Agent is actively executing tasks
- **Error** (🔴): Agent encountered an error
- **Offline** (⚫): Agent is not available
- **Starting** (🔵): Agent is initializing
- **Stopping** (🟠): Agent is shutting down

## Integration with Chat UI

The status tracking system integrates with the chat UI through the StatusPane:

1. **Real-time Updates**: Agent status changes are immediately reflected in the UI
2. **Animation Support**: Active agents display animated indicators
3. **Summary View**: Compact summary in the status bar
4. **Detailed View**: Expanded view with agent details and activity

## Architecture Decisions

1. **Modular Design**: Each component has a single responsibility
2. **Interface-First**: All major components are defined by interfaces
3. **Context Propagation**: All I/O operations accept context.Context
4. **Thread Safety**: Concurrent access handled with sync.RWMutex
5. **Error Handling**: Consistent use of gerror framework

## Performance Considerations

- Activity history is limited to prevent memory growth
- Animations update at configurable frame rates
- Statistics are cached and updated only on state changes
- Inactive agents can be purged to free resources

## Testing

The package includes comprehensive tests and examples:

- Unit tests for each component
- Integration tests for UI interaction
- Example usage in `example_test.go`

## Future Enhancements

1. **Persistence**: Save agent history to database
2. **Metrics Export**: Prometheus/OpenTelemetry integration
3. **Custom Indicators**: Plugin system for custom animations
4. **Agent Groups**: Support for team/guild tracking
5. **Performance Profiling**: Agent efficiency metrics
