# 📋 Kanban Board & Tasks

Guild's task system is modeled after a Kanban workflow. Tasks are created from objectives, assigned to agents, and moved through states as work progresses.

---

## 📆 Task Engine

- Tasks originate from parsed objectives or direct user input
- Each task has:

  - `id`: unique identifier
  - `title`: short summary
  - `description`: full prompt or spec
  - `status`: To Do, In Progress, Blocked, Done
  - `agent`: assigned worker
  - `prompt_chain`: optional prompt history

Stored persistently in **BoltDB** for durability.

---

## 📑 Agent Boards

- Each agent has a personalized Kanban view

  - Can be queried or rendered via CLI/UI
  - Filters tasks by role, type, or priority

- Manager agents maintain aggregate boards for all agents

Task views:

- `To Do`
- `In Progress`
- `Blocked` (human input needed)
- `Done`

---

## ♻️ ZeroMQ Integration

ZeroMQ is used as the internal messaging and coordination bus.

### Events:

- `task_created`
- `task_updated`
- `task_moved`
- `task_completed`
- `task_blocked`

### Protocol:

Messages are published as structured JSON or Protobuf objects, including:

- `task_id`
- `agent_id`
- `action`
- `timestamp`
- `changes`

This enables:

- Live dashboards
- Event replay
- Cross-language integration
