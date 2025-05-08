# Specs Index for Claude Code

This index guides Claude Code in understanding the specifications for the Guild framework.

## 🧭 How to Use This Guide

1. **For implementation questions**, refer to the relevant spec section first
2. **For architectural decisions**, check the architecture specs
3. **For feature requirements**, look at the feature specs
4. **For project context**, review the lore and naming conventions

## 📚 Core Specifications

### Architecture

| Document                                                                    | Description                   | Key Points                                           |
| --------------------------------------------------------------------------- | ----------------------------- | ---------------------------------------------------- |
| [specs/architecture/architecture.md](../specs/architecture/architecture.md) | Overall system design         | Core components, model abstraction, high-level goals |
| [specs/architecture/coordination.md](../specs/architecture/coordination.md) | Meta-coordination protocol    | MCP, RAG implementation, optimization loop           |
| [specs/agent-git-workflow.md](../specs/agent-git-workflow.md)               | Git-based agent collaboration | Branch model, merge flow, interface blocking         |

### Feature Specifications

| Feature           | Specification                                                           | Implementation Notes                                        |
| ----------------- | ----------------------------------------------------------------------- | ----------------------------------------------------------- |
| Agent Behavior    | [specs/features/agent-behavior.md](../specs/features/agent-behavior.md) | Agent lifecycle, task execution, cost-aware decision making |
| Kanban Board      | [specs/features/kanban.md](../specs/features/kanban_board.md)           | Task states, ZeroMQ events, board management                |
| Memory System     | [specs/features/memory.md](../specs/features/memory.md)                 | PromptChain, vector store, RAG, context restoration         |
| Objectives System | [specs/features/objectives.md](../specs/features/objectives.md)         | Markdown objective format, directory structure, parsing     |
| Tools             | [specs/features/tools.md](../specs/features/tools.md)                   | Tool integration, CLI execution, configuration format       |
| Configuration     | [specs/features/configuration.md](../specs/features/configuration.md)   | YAML format, environment integration, cost settings         |

### Examples and Workflows

| Document                                                                                                                        | Description                  | Key Points                                                |
| ------------------------------------------------------------------------------------------------------------------------------- | ---------------------------- | --------------------------------------------------------- |
| [specs/examples.md](../specs/examples.md)                                                                                       | Example guild configurations | Single-agent assistant, dev guild, marketplace examples   |
| [specs/user_workflow.md](../specs/user_workflow.md)                                                                             | User workflow guide          | 5-step process, objective definition, agent configuration |
| [specs/examples_use_cases_and_user_workflows/examples.md](../specs/examples_use_cases_and_user_workflows/examples.md)           | Extended examples            | Detailed example implementations and use cases            |
| [specs/examples_use_cases_and_user_workflows/user_workflow.md](../specs/examples_use_cases_and_user_workflows/user_workflow.md) | Detailed user workflow       | Step-by-step guide for patent attorney use case           |

### Project Context and Terminology

| Document                                                                                  | Description                 | Key Points                                                  |
| ----------------------------------------------------------------------------------------- | --------------------------- | ----------------------------------------------------------- |
| [specs/naming_conventions_and_lore/lore.md](../specs/naming_conventions_and_lore/lore.md) | Guild lore and naming       | Conceptual mapping, naming conventions, directory structure |
| [specs/CLAUDE_PROJECT_INSTRUCT.md](../specs/CLAUDE_PROJECT_INSTRUCT.md)                   | Project overview for Claude | Comprehensive project description and core concepts         |

### Future Enhancements

| Document                                                                          | Description          | Key Points                                                  |
| --------------------------------------------------------------------------------- | -------------------- | ----------------------------------------------------------- |
| [specs/refactors/enhancements-index.md](../specs/refactors/enhancements-index.md) | Planned enhancements | Model routing, spec versioning, permissions, error handling |

## 🧩 Component Relationships

The Guild framework consists of these key components, as defined in the specs:

1. **Agents** (from agent-behavior.md)

   - Execute tasks via LLM API
   - Access tools via interfaces
   - Maintain personal Kanban boards

2. **Orchestrators/Guilds** (from architecture.md, coordination.md)

   - Coordinate task execution among agents
   - Implement the Meta-Coordination Protocol (MCP)
   - Manage cost-aware resource allocation

3. **Kanban System** (from kanban_board.md)

   - Tracks tasks through their lifecycle
   - Publishes events via ZeroMQ
   - Persists in BoltDB

4. **Memory System** (from memory.md)

   - Stores prompt chains in BoltDB
   - Maintains vector embeddings in Qdrant
   - Provides RAG for context restoration

5. **Objectives System** (from objectives.md)

   - Parses markdown files into structured objectives
   - Organizes tasks in hierarchical structure
   - Links related objectives and tasks

6. **Tools System** (from tools.md)
   - Wraps CLI tools for agent use
   - Provides standardized interface for all tools
   - Enables cost optimization through tool reuse

## 🔄 Relationship Between Specs and AI Docs

- **Specs**: Define WHAT should be built and WHY (requirements, architecture decisions, feature specifications)
- **AI Docs**: Explain HOW to implement the specs (implementation patterns, integration guides, API usage)

When implementing any component, Claude Code should:

1. First review the relevant spec to understand requirements
2. Then check AI docs for implementation guidance
3. Ask questions if the spec/docs don't provide sufficient clarity

## 💻 Implementation Guidance

When implementing Guild components:

1. **Follow Interface-First Development**

   - Define interfaces before implementation
   - Keep interfaces in separate files
   - Use composition over inheritance

2. **Apply Go Concurrency Patterns**

   - Use goroutines for agent tasks
   - Communicate via channels
   - Ensure context propagation

3. **Prioritize Error Handling**

   - Implement graceful error recovery
   - Use semantic error types
   - Ensure proper resource cleanup

4. **Ensure Cost Awareness**

   - Implement cost tracking
   - Prefer low-cost operations
   - Log justifications for high-cost actions

5. **Enable Human-in-the-Loop**
   - Block tasks for human input when needed
   - Provide clear context for decisions
   - Maintain audit trail for human interventions
