# 🧠 Goal

Define the manager agent's core responsibilities in Guild, including the task loop, objective change detection, and task scheduling heuristics.

# 📂 Context

The manager agent orchestrates Guild workflows by:

- Parsing objectives from `/objectives/`
- Decomposing them into agent-specific tasks
- Scheduling and tracking task execution
- Reacting to updates in objective markdown files

This behavior is critical for evolving, multi-agent coordination.

# 🔧 Requirements

## 🌀 Task Loop

1. **Ingest Objective**

   - Load root objective directory path from `guild.yaml`
   - Parse all `*.md` files into an internal graph
   - Extract `# Goal`, `# Requirements`, `# Related` sections

2. **Plan Tasks**

   - Break down `Requirements` into task queue
   - Assign tags/labels based on filename and tags field
   - Map task → agent based on role and tool compatibility

3. **Dispatch Tasks**

   - Send tasks to agents via ZeroMQ or Go channels
   - Monitor in-progress and blocked tasks via Kanban state

4. **Track and Merge**

   - Log completed tasks and outputs
   - Trigger post-task commands (e.g., run tests, re-prime agents)
   - Merge partial outputs into `specs/` or `ai_docs/` if required

5. **Loop**
   - Recheck `/objectives/` for changes
   - Sleep/retry loop with exponential backoff or event-driven change

---

## 🛎️ Change Detection

- Monitor objective file timestamps + checksums
- Detect added/removed/modified `.md` files
- Trigger re-plan only if:
  - New file added with `# Goal` and `# Requirements`
  - A `# Requirements` list changes in size/content
  - A `# Related` link to an unprocessed task is added

---

## 🧠 Task Scheduling Heuristics

- Prefer to schedule leaf tasks first (no dependencies)
- Use Kanban state to avoid overwhelming agents
- Interface tasks (API contracts, shared modules) block dependent tasks
- Cost-aware: avoid scheduling expensive model tasks unless required
- Use tags like `urgent`, `blocked`, `infra` to prioritize queues

---

# 📌 Tags

- manager-agent
- scheduler
- objective-tracker
- orchestration

# 🔗 Related

- `/specs/components/cli_command_specs.md`
- `/specs/features/kanban_board.md`

# 🧪 Self-Validation

- Modify an objective file → verify re-plan triggers
- Block a task → confirm manager waits and reschedules dependents
- Log scheduling decisions for each round
