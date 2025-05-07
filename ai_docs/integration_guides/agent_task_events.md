# Agent Task Event Protocol

This document outlines the event system used for task state changes.

## Message Format

```json
{
  "type": "task_update",
  "task_id": "1234",
  "state": "done",
  "agent": "planner"
}
```

## Events

- `task_created`
- `task_blocked`
- `task_unblocked`
- `task_completed`
