# Guild Event System Documentation

## Overview

The Guild Framework uses an event-driven architecture to enable loose coupling between components and provide comprehensive observability. All major operations emit events that can be consumed by other services, logged, or used for monitoring.

## Event Categories

### Service Events (`service.*`)

Lifecycle and health events for Guild services.

| Event Type | Description | Data Fields |
|------------|-------------|-------------|
| `service.starting` | Service is beginning startup | `service_id`, `service_name`, `timestamp` |
| `service.started` | Service has started successfully | `service_id`, `service_name`, `timestamp`, `startup_duration` |
| `service.stopping` | Service is beginning shutdown | `service_id`, `service_name`, `timestamp` |
| `service.stopped` | Service has stopped | `service_id`, `service_name`, `timestamp`, `uptime_seconds` |
| `service.healthy` | Service health check passed | `service_id`, `service_name`, `timestamp` |
| `service.unhealthy` | Service health check failed | `service_id`, `service_name`, `timestamp`, `error` |
| `service.error` | Service encountered an error | `service_id`, `service_name`, `timestamp`, `error`, `severity` |

### Data Events (`data.*`)

CRUD and data operation events.

| Event Type | Description | Data Fields |
|------------|-------------|-------------|
| `data.created` | Entity was created | `entity_type`, `entity_id`, `timestamp`, `created_by` |
| `data.updated` | Entity was updated | `entity_type`, `entity_id`, `timestamp`, `updated_by`, `changes` |
| `data.deleted` | Entity was deleted | `entity_type`, `entity_id`, `timestamp`, `deleted_by` |
| `data.queried` | Data was queried | `entity_type`, `query`, `result_count`, `timestamp` |
| `data.synced` | Data was synchronized | `entity_type`, `sync_count`, `timestamp` |
| `data.corrupted` | Data corruption detected | `entity_type`, `entity_id`, `timestamp`, `details` |

### Task Events (`task.*`)

Task lifecycle and progress events.

| Event Type | Description | Data Fields |
|------------|-------------|-------------|
| `task.created` | Task was created | `task_id`, `task_type`, `timestamp`, `created_by` |
| `task.assigned` | Task was assigned | `task_id`, `assigned_to`, `timestamp` |
| `task.started` | Task execution started | `task_id`, `assigned_to`, `timestamp` |
| `task.progress` | Task progress update | `task_id`, `progress`, `message`, `timestamp` |
| `task.completed` | Task completed successfully | `task_id`, `duration`, `result`, `timestamp` |
| `task.failed` | Task failed | `task_id`, `error`, `timestamp` |
| `task.cancelled` | Task was cancelled | `task_id`, `cancelled_by`, `timestamp` |
| `task.retried` | Task retry attempted | `task_id`, `retry_count`, `timestamp` |

### Agent Events (`agent.*`)

Agent lifecycle and activity events.

| Event Type | Description | Data Fields |
|------------|-------------|-------------|
| `agent.registered` | Agent registered | `agent_id`, `agent_name`, `agent_type`, `capabilities`, `timestamp` |
| `agent.unregistered` | Agent unregistered | `agent_id`, `agent_name`, `timestamp` |
| `agent.state.changed` | Agent state changed | `agent_id`, `old_state`, `new_state`, `timestamp` |
| `agent.task.received` | Agent received task | `agent_id`, `task_id`, `timestamp` |
| `agent.task.completed` | Agent completed task | `agent_id`, `task_id`, `duration`, `timestamp` |
| `agent.error` | Agent error | `agent_id`, `error`, `task_id`, `timestamp` |
| `agent.health.check` | Agent health checked | `agent_id`, `status`, `timestamp` |

### Commission Events (`commission.*`)

Commission (high-level task) events.

| Event Type | Description | Data Fields |
|------------|-------------|-------------|
| `commission.created` | Commission created | `commission_id`, `title`, `description`, `timestamp` |
| `commission.planned` | Commission planning complete | `commission_id`, `task_count`, `estimated_duration`, `timestamp` |
| `commission.started` | Commission execution started | `commission_id`, `timestamp` |
| `commission.progress` | Commission progress update | `commission_id`, `progress`, `completed_tasks`, `total_tasks`, `timestamp` |
| `commission.completed` | Commission completed | `commission_id`, `duration`, `result`, `timestamp` |
| `commission.failed` | Commission failed | `commission_id`, `error`, `timestamp` |
| `commission.cancelled` | Commission cancelled | `commission_id`, `cancelled_by`, `reason`, `timestamp` |

### UI Events (`ui.*`)

User interface interaction events.

| Event Type | Description | Data Fields |
|------------|-------------|-------------|
| `ui.connected` | UI client connected | `session_id`, `user_id`, `timestamp` |
| `ui.disconnected` | UI client disconnected | `session_id`, `user_id`, `duration`, `timestamp` |
| `ui.state.changed` | UI state changed | `session_id`, `component`, `old_state`, `new_state`, `timestamp` |
| `ui.command.issued` | UI command issued | `session_id`, `command`, `parameters`, `timestamp` |
| `ui.error` | UI error occurred | `session_id`, `error`, `component`, `timestamp` |

### System Events (`system.*`)

System-wide notifications and alerts.

| Event Type | Description | Data Fields |
|------------|-------------|-------------|
| `system.notification` | System notification | `message`, `severity`, `source`, `timestamp` |
| `system.warning` | System warning | `message`, `component`, `details`, `timestamp` |
| `system.error` | System error | `error`, `component`, `stack_trace`, `timestamp` |
| `system.shutdown` | System shutting down | `reason`, `graceful`, `timestamp` |

### gRPC Events (`grpc.*`)

gRPC server and client events.

| Event Type | Description | Data Fields |
|------------|-------------|-------------|
| `grpc.request` | gRPC request received | `method`, `client`, `timestamp` |
| `grpc.response` | gRPC response sent | `method`, `status_code`, `duration`, `timestamp` |
| `grpc.stream` | gRPC stream event | `method`, `event_type`, `message_count`, `timestamp` |
| `grpc.error` | gRPC error | `method`, `error`, `status_code`, `timestamp` |

### Corpus Events (`corpus.*`)

Document corpus scanning and indexing events.

| Event Type | Description | Data Fields |
|------------|-------------|-------------|
| `corpus.scan.started` | Corpus scan started | `base_path`, `file_patterns`, `timestamp` |
| `corpus.scan.progress` | Scan progress update | `processed_files`, `total_files`, `current_path`, `timestamp` |
| `corpus.scan.completed` | Corpus scan completed | `files_processed`, `files_indexed`, `duration`, `timestamp` |
| `corpus.file.indexed` | File indexed | `file_path`, `file_type`, `size`, `timestamp` |
| `corpus.index.error` | Indexing error | `file_path`, `error`, `timestamp` |
| `corpus.search` | Corpus search performed | `query`, `result_count`, `duration`, `timestamp` |

## Event Flow Diagrams

### Service Startup Flow

```
┌─────────────┐     ┌──────────────┐     ┌─────────────┐
│   Bootstrap │────▶│service.starting│────▶│ Service Init│
└─────────────┘     └──────────────┘     └─────────────┘
                                                  │
                                                  ▼
                    ┌──────────────┐     ┌─────────────┐
                    │service.started│◀────│Service Ready│
                    └──────────────┘     └─────────────┘
```

### Task Execution Flow

```
┌─────────────┐     ┌──────────────┐     ┌─────────────┐
│  Task Queue │────▶│task.created  │────▶│  Scheduler  │
└─────────────┘     └──────────────┘     └─────────────┘
                                                  │
                                          ┌───────▼───────┐
                                          │task.assigned  │
                                          └───────┬───────┘
                                                  │
┌─────────────┐     ┌──────────────┐     ┌───────▼───────┐
│   Agent     │◀────│task.started  │◀────│Agent Manager │
└─────────────┘     └──────────────┘     └───────────────┘
       │
       ├────────────▶ task.progress (multiple times)
       │
       └────────────▶ task.completed / task.failed
```

### Commission Lifecycle

```
commission.created ──▶ commission.planned ──▶ commission.started
                                                      │
                                                      ▼
                                              commission.progress
                                                      │
                                   ┌──────────────────┴──────────────────┐
                                   ▼                                     ▼
                           commission.completed                   commission.failed
```

## Integration Patterns

### 1. Event Logging Bridge

All events can be automatically logged through the EventLoggerBridge:

```go
// Configure event logging
config := bridges.EventLoggerConfig{
    LogLevel:         bridges.LogLevelInfo,
    IncludeEventData: true,
    BufferSize:       1000,
}
bridge := bridges.NewEventLoggerBridge(eventBus, logger, config)
```

### 2. Persistence Event Bridge

Automatically emit events for all database operations:

```go
// Enable persistence events
config := bridges.PersistenceEventConfig{
    EmitCRUDEvents:  true,
    EmitQueryEvents: false,
    IncludePayload:  false,
}
```

### 3. UI Event Bridge

Connect UI state changes to the event system:

```go
// Configure UI events
config := bridges.UIEventConfig{
    BatchEvents:   true,
    BatchInterval: 100 * time.Millisecond,
    MaxBatchSize:  50,
}
```

## Best Practices

### 1. Event Naming

- Use dot notation for hierarchical event types
- Start with the category, followed by subcategories
- Be specific but not overly verbose
- Examples: `task.created`, `agent.state.changed`, `commission.progress`

### 2. Event Data

- Always include `timestamp`
- Include relevant IDs (task_id, agent_id, etc.)
- Keep payloads reasonable in size
- Sensitive data should be omitted or redacted
- Use consistent field names across similar events

### 3. Event Handling

- Handlers should be idempotent
- Don't assume event ordering
- Handle errors gracefully
- Keep handlers fast (offload heavy work)
- Use context for cancellation

### 4. Performance

- Events are asynchronous by default
- Use batching for high-frequency events
- Consider event filtering at source
- Monitor event queue sizes

## Debugging Guide

### Enabling Event Tracing

```bash
# Enable all event logging
export GUILD_EVENT_LOG_LEVEL=debug

# Enable specific event categories
export GUILD_EVENT_FILTER="task.*,agent.*"

# Write events to file
export GUILD_EVENT_LOG_FILE=".guild/logs/events.log"
```

### Common Issues

1. **Missing Events**
   - Check service is started and healthy
   - Verify event bus subscription
   - Check event filters

2. **Event Storms**
   - Enable event batching
   - Increase buffer sizes
   - Add rate limiting

3. **Event Order**
   - Events are not guaranteed to be ordered
   - Use timestamps for sequencing
   - Consider event sourcing for critical flows

### Event Metrics

Monitor these key metrics:

- Events published per second
- Event processing latency
- Event queue depth
- Failed event handlers
- Event payload sizes

## Example: Subscribing to Events

```go
// Subscribe to all task events
eventBus.Subscribe(ctx, "task.*", func(ctx context.Context, event events.Event) error {
    switch event.Type() {
    case events.EventTaskCreated:
        // Handle task creation
    case events.EventTaskCompleted:
        // Handle task completion
    }
    return nil
})

// Subscribe to specific event
eventBus.Subscribe(ctx, events.EventAgentStateChanged, func(ctx context.Context, event events.Event) error {
    data := event.Data().(map[string]interface{})
    agentID := data["agent_id"].(string)
    newState := data["new_state"].(string)
    
    // Handle state change
    return nil
})
```

## Future Enhancements

1. **Event Versioning**: Support for event schema evolution
2. **Event Replay**: Ability to replay events for debugging
3. **Event Filtering**: Advanced filtering capabilities
4. **Event Correlation**: Track related events across services
5. **Event Analytics**: Built-in event analysis tools
