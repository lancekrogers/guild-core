# 🗙️ Objectives as Hierarchical Markdown Plans

Guild uses a directory-based Markdown objective system to encode complex project goals in a way that is both human-readable and machine-parsable. This forms the foundation for task decomposition, prompt planning, and agent guidance.

---

## 📂 Directory-Based Planning

- Root: `/objectives/`
- Each folder = subsystem, topic, or component
- Each `.md` file = self-contained prompt or task definition
- Folders group files by:

  - Agent role
  - System type
  - Tool class
  - Workflow stage

### Example Layout

```text
/objectives
├── README.md                      # High-level vision & constraints
├── infrastructure/
│   ├── overview.md
│   ├── kanban-system.md
│   └── qdrant-integration.md
├── agents/
│   ├── roles.md
│   ├── planner.md
│   └── coder.md
├── cli-tools/
│   ├── aider.md
│   └── tree2scaffold.md
```

---

## 📄 Markdown Prompt Format

Each file should be readable by both LLMs and humans. Use the following sections:

```markdown
# 🧠 Goal

State the specific outcome this task or subcomponent should achieve.

# 📂 Context

Summarize the component’s role in the system. Include links, constraints, or architectural notes.

# 🔧 Requirements

- List of specs, interfaces, outputs
- Clear enough to be translated into subtasks

# 📌 Tags

- kanban
- task-engine
- zmq
- persistence

# 🔗 Related

- [../agents/manager.md](../agents/manager.md)
- [../infrastructure/qdrant-integration.md](../infrastructure/qdrant-integration.md)
```

---

## 🤖 Agent Usage

- Agents traverse objective files via links and tags
- They build context chains for long-running tasks
- Objective prompts can be used for:

  - Task planning
  - Cost-aware decision making
  - Prompt rehydration

- Tasks are traced back to objective files for auditing and iteration
