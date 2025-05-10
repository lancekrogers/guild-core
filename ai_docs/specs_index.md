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

| Feature           | Specification                                                                             | Implementation Notes                                        |
| ----------------- | ----------------------------------------------------------------------------------------- | ----------------------------------------------------------- |
| Agent Behavior    | [specs/features/agent-behavior.md](../specs/features/agent-behavior.md)                   | Agent lifecycle, task execution, cost-aware decision making |
| Kanban Board      | [specs/features/kanban_board.md](../specs/features/kanban_board.md)                       | Task states, ZeroMQ events, board management                |
| Memory System     | [specs/features/memory.md](../specs/features/memory.md)                                   | PromptChain, vector store, RAG, context restoration         |
| Objectives System | [specs/features/objectives/objectives.md](../specs/features/objectives/objectives.md)     | Markdown objective format, directory structure, parsing     |
| Objective UI      | [specs/features/objectives/objective_ui.md](../specs/features/objectives/objective_ui.md) | Interactive UI, dashboard, command processing               |
| Tools             | [specs/features/tools.md](../specs/features/tools.md)                                     | Tool integration, CLI execution, configuration format       |
| Configuration     | [specs/features/configuration.md](../specs/features/configuration.md)                     | YAML format, environment integration, cost settings         |
| Corpus System     | [specs/features/corpus_system.md](../specs/features/corpus_system.md)                     | Knowledge management, corpus organization                   |

### Component Specifications

| Component         | Specification                                                                     | Implementation Notes                               |
| ----------------- | --------------------------------------------------------------------------------- | -------------------------------------------------- |
| CLI Commands      | [specs/components/cli_command_specs.md](../specs/components/cli_command_specs.md) | Command structure, flags, help documentation       |
| Manager Task Loop | [specs/components/manager_task_loop.md](../specs/components/manager_task_loop.md) | Manager agent execution cycle, task prioritization |

### Examples and Workflows

| Document                                                                                                                        | Description            | Key Points                                      |
| ------------------------------------------------------------------------------------------------------------------------------- | ---------------------- | ----------------------------------------------- |
| [specs/examples_use_cases_and_user_workflows/examples.md](../specs/examples_use_cases_and_user_workflows/examples.md)           | Extended examples      | Detailed example implementations and use cases  |
| [specs/examples_use_cases_and_user_workflows/user_workflow.md](../specs/examples_use_cases_and_user_workflows/user_workflow.md) | Detailed user workflow | Step-by-step guide for patent attorney use case |

### Project Context and Terminology

| Document                                                                                  | Description                 | Key Points                                                  |
| ----------------------------------------------------------------------------------------- | --------------------------- | ----------------------------------------------------------- |
| [specs/naming_conventions_and_lore/lore.md](../specs/naming_conventions_and_lore/lore.md) | Guild lore and naming       | Conceptual mapping, naming conventions, directory structure |
| [specs/CLAUDE_PROJECT_INSTRUCT.md](../specs/CLAUDE_PROJECT_INSTRUCT.md)                   | Project overview for Claude | Comprehensive project description and core concepts         |

### Future Enhancements

| Document                                                                          | Description          | Key Points                                                  |
| --------------------------------------------------------------------------------- | -------------------- | ----------------------------------------------------------- |
| [specs/refactors/enhancements-index.md](../specs/refactors/enhancements-index.md) | Planned enhancements | Model routing, spec versioning, permissions, error handling |

### Tools

| Document                                                                | Description       | Key Points                             |
| ----------------------------------------------------------------------- | ----------------- | -------------------------------------- |
| [specs/tools/youtube_ingestion.md](../specs/tools/youtube_ingestion.md) | YouTube Ingestion | YouTube content processing and storage |

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

5. **Objectives System** (from objectives.md and objective_ui.md)

   - Parses markdown files into structured objectives
   - Organizes tasks in hierarchical structure
   - Links related objectives and tasks
   - Provides interactive UI for objective creation and management
   - Tracks objective status and progress
   - Generates AI docs and specs from objectives

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

## 🎯 Objective System Implementation

The objective system is a key component that requires particular attention to implement correctly:

### Core Components

1. **Prompt Management System** (internal/prompts)

   - Centralized prompt management in internal/prompts/manager.go
   - Objective-specific prompts in internal/prompts/objective/markdown/
   - Templating and rendering for LLM prompts

2. **Generator Package** (pkg/generator)

   - Interface definitions in pkg/generator/interface.go
   - Objective generator implementation in pkg/generator/objective/generator.go
   - LLM integration for content generation

3. **Objective Models** (pkg/objective)

   - Data structures in pkg/objective/models.go
   - Parser implementation in pkg/objective/parser.go
   - Lifecycle management in pkg/objective/lifecycle.go

4. **UI Components** (pkg/ui/objective)

   - Bubble Tea UI models in pkg/ui/objective/model.go
   - View implementation in pkg/ui/objective/view.go
   - Update logic in pkg/ui/objective/update.go
   - Dashboard in pkg/ui/objective/dashboard.go

5. **CLI Commands** (cmd/guild)
   - Individual objective management in cmd/guild/objective_cmd.go
   - Dashboard overview in cmd/guild/objectives_cmd.go

### Implementation Order

When implementing the objective system, follow this sequence:

1. Check existing code to avoid duplication
2. Implement internal prompt system first
3. Build generator package with LLM integration
4. Enhance or implement objective models
5. Create UI components
6. Add CLI commands

This ensures that dependencies are properly satisfied and each component builds on the previous one.
