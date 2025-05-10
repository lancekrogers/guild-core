# 🧠 Goal

Design a task management system for Guild based on Kanban principles to coordinate multi-agent workflows, track task states, and support dynamic updates via Go channels and BoltDB.

---

# 📂 Context

Guild agents work asynchronously and in parallel. A durable, inspectable task system is needed to:

- Track task lifecycle from planning to completion
- Support human-in-the-loop states (`Blocked`)
- Allow real-time monitoring and rehydration of agent state
- Coordinate between agents via the manager and event bus

---

# 🧱 Kanban Model

Each task has the following structure:

```yaml
task:
  id: string              # Unique identifier
  title: string           # Short human-readable label
  description: string     # Full prompt or plan
  status: enum            # To Do | In Progress | Blocked | Done
  agent: string           # Assigned agent (optional)
  tags: []string          # Used for routing, UX filtering
  prompt_chain: []Prompt  # Optional prompt history
  created_at: timestamp
  updated_at: timestamp
```

Tasks are stored in **BoltDB** with secondary indexes for:
- Agent
- Status
- Tags
- Objective source

---

# 📆 Task Lifecycle

Tasks move through the following states:

- `To Do`: Unstarted, ready to be assigned
- `In Progress`: Being executed by an agent
- `Blocked`: Requires human input or another task
- `Done`: Completed and optionally validated

Transition rules:
- Only `To Do` → `In Progress` or `Blocked`
- `Blocked` can become `To Do` again
- `Done` is final unless marked `Reopened`

---

# 👤 Agent Boards

Each agent gets a filtered view of its task queue:

- CLI: `guild monitor --agent coder`
- JSON API: `/tasks?agent=reviewer&status=todo`
- Grouped by status, tag, or source file

The **manager agent** maintains a global view and coordination logic.

---

# 🧠 Blocking Logic

Some tasks (e.g., interface definitions) **block dependent tasks** until marked `Done`. 

- Blocking defined in task metadata or inferred via objective analysis
- Manager agent enforces dependency locks
- Downstream agents wait or are assigned alternate work

---

# 📡 Event System

Uses Go channels for the internal message bus with a pub/sub pattern for live updates and coordination.

### 📤 Emitted Events

- `task_created`
- `task_updated`
- `task_blocked`
- `task_unblocked`
- `task_completed`

### 📦 Message Format

```json
{
  "type": "task_event",
  "task_id": "abc-123",
  "agent": "planner",
  "action": "task_updated",
  "timestamp": "2025-05-07T14:21:00Z",
  "payload": {
    "status": "in_progress",
    "previous_status": "todo"
  }
}
```

Events are published and consumed through the channel-based pub/sub system to support:

- Live dashboards
- CLI tools
- Replay + auditing

> **Future Extension**: For distributed deployment across multiple machines, ZeroMQ integration is planned for a future version. See `/specs/horizon/zeromq_integration.md` for details.

---

# 🔧 CLI Integration

Commands:
```bash
guild monitor            # View global kanban state
guild monitor --agent implementer
guild tasks --json       # Dump current task queue
```

Outputs:
- Filterable by agent, tag, or objective
- Supports TUI, JSON, and Web dashboard adapters

---

# 🧪 Self-Validation

- Manager can reconstruct Kanban state from event log
- Task transitions follow valid paths
- Blocked tasks pause downstream execution
- Monitor view reflects real-time status updates
