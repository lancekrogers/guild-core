# Task Execution Flow

This document explains how tasks flow through the Guild system from creation to completion.

## Task Lifecycle States

1. **Creation**

   - Tasks are created from objectives
   - Initial metadata is assigned
   - Task is placed in "To Do" status

2. **Assignment**

   - Task is assigned to an agent
   - Agent is notified via event system
   - Task moves to "In Progress" status

3. **Execution**

   - Agent processes the task
   - Intermediate results are recorded
   - Prompt chain is built incrementally

4. **Blocking**

   - Task may be blocked awaiting human input
   - Status changes to "Blocked"
   - Human is notified via UI or CLI

5. **Resolution**

   - Human provides input
   - Task returns to "In Progress"
   - Agent continues execution

6. **Completion**
   - Task results are finalized
   - Status changes to "Done"
   - Dependent tasks are unblocked

## Event Flow

1. **Event Types**

   - `task_created`
   - `task_assigned`
   - `task_started`
   - `task_blocked`
   - `task_resumed`
   - `task_completed`
   - `task_failed`

2. **Event Structure**

   ```json
   {
     "type": "task_blocked",
     "task_id": "task-123",
     "agent_id": "agent-456",
     "timestamp": "2025-05-08T10:15:30Z",
     "data": {
       "reason": "Need clarification on API endpoint",
       "options": ["Option A", "Option B"]
     }
   }
   ```

3. **Subscription Pattern**
   - UI subscribes to events
   - Agents subscribe to relevant events
   - Manager agent subscribes to all events

## Task Data Flow

1. **Input Sources**

   - Objective markdown files
   - User commands
   - System-generated tasks

2. **State Storage**

   - BoltDB for task metadata
   - Prompt chains in BoltDB
   - Results in filesystem

3. **Output Destinations**
   - Task results in filesystem
   - Event logs
   - Kanban board UI

## Dependency Management

1. **Task Dependencies**

   - Tasks can depend on other tasks
   - Dependent tasks are blocked until dependencies complete
   - Interface tasks have special handling

2. **Dependency Resolution**
   - When a task completes, dependent tasks are evaluated
   - If all dependencies are met, task becomes unblocked
   - Interface changes may re-block dependent tasks

## Implementation Guidelines

```go
// Example task creation
func CreateTask(board Board, title, description string, agentID string) (Task, error) {
    // Implementation details...
}

// Example task state change
func (b *BoltBoard) Move(ctx context.Context, taskID string, status TaskStatus) error {
    // Implementation details...
}
```

## Related Documentation

- [../integration_guides/bolt_db_kanban.md](../integration_guides/bolt_db_kanban.md)
- [../integration_guides/agent_task_events.md](../integration_guides/agent_task_events.md)
