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

Summarize the component's role in the system. Include links, constraints, or architectural notes.

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

---

## 🔄 Objective Lifecycle

The lifecycle of objectives in Guild is highly adaptable and can originate from various starting points:

### Multiple Entry Points

- **Empty start**: User begins with just a conversation with an agent to develop an objective from scratch
- **Partial draft**: User has a general idea and some initial content to refine
- **Pre-populated structure**: User creates a complete directory structure with markdown files organizing different aspects of the project
- **Fully detailed plan**: User provides a comprehensive objective with all sections already filled out

### Flexible Development Process

- The objective can evolve through iterative conversation with agents
- A user can modify existing markdown files and directories at any point
- The system supports both top-down (start with high-level and break down) and bottom-up (assemble from components) approaches

### Contextual Adaptation

- For complex projects (like compiling research into a book), a user might provide extensive source materials upfront
- The objective can include external references like web links, YouTube videos, PDFs, etc.
- Agents adapt their guidance based on the completeness of the provided information

### Interactive Refinement

- The default agent works with the user to refine objectives through conversation
- If information is insufficient, the agent proactively asks clarifying questions
- The agent provides feedback on what's missing or could be improved

### Path to Implementation

- When an objective reaches sufficient clarity, the agent confirms with the user
- Only after user confirmation does the agent break down the objective into `/ai_docs` and `/specs`
- This ensures human oversight of the planning process before execution begins

---

## 📋 Integration with Kanban

- Each objective can be decomposed into tasks in the Kanban system
- Tasks maintain references back to their source objectives
- Completed tasks feed back into the objective's status tracking
- The objective serves as the contextual anchor for all derived tasks
