# BoltDB Kanban System

The Kanban system persists task metadata and state transitions using BoltDB.

## Schema

- `Tasks` bucket stores serialized task states.
- States: `todo`, `in_progress`, `blocked`, `done`.

## Indexing

- Tasks are indexed by task ID and agent ID.
- Blocking metadata used to enforce interface-first flow.
