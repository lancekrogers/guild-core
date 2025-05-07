# Agent Lifecycle

Describes the lifecycle of an agent within the Guild runtime.

## Lifecycle Phases

1. **Init**: Agent is created and tools are bound.
2. **Execute**: Agent performs a task with context.
3. **Report**: Task result is logged and emitted.
4. **Idle**: Agent polls for next task or shuts down.

## Error Handling

- Fails with retries.
- May trigger re-plan or rollback.
