# AI Docs Index - Guide for Claude Code

This index helps Claude Code navigate the Guild project documentation to efficiently implement features and understand the system architecture.

## 🧭 How to Use This Guide

1. **For implementation tasks**, start with the relevant section in this index
2. **For architectural questions**, refer to the specs directory first
3. **For code patterns**, check the patterns section in this index
4. **For integration help**, see the integration guides section

## 📚 Core Documentation

### System Architecture

| Document              | Description                                         | Location                                                                           |
| --------------------- | --------------------------------------------------- | ---------------------------------------------------------------------------------- |
| System Overview       | High-level architecture and component relationships | [specs/architecture/architecture.md](../specs/architecture/architecture.md)        |
| Agent Lifecycle       | How agents are created, run, and terminated         | [ai_docs/architecture/agent_lifecycle.md](architecture/agent_lifecycle.md)         |
| Guild Runtime         | Guild execution flow and coordination               | [ai_docs/architecture/guild_runtime.md](architecture/guild_runtime.md)             |
| Task Execution Flow   | How tasks flow through the system                   | [ai_docs/architecture/task_execution_flow.md](architecture/task_execution_flow.md) |
| Coordination Protocol | Meta-coordination protocol (MCP)                    | [specs/architecture/coordination.md](../specs/architecture/coordination.md)        |

### Core Components

| Component     | Requirements                                                            | Implementation Guide                                                                 |
| ------------- | ----------------------------------------------------------------------- | ------------------------------------------------------------------------------------ |
| Agents        | [specs/features/agent-behavior.md](../specs/features/agent-behavior.md) | [ai_docs/architecture/agent_lifecycle.md](architecture/agent_lifecycle.md)           |
| Kanban Board  | [specs/features/kanban_board.md](../specs/features/kanban_board.md)     | [ai_docs/integration_guides/bolt_db_kanban.md](integration_guides/bolt_db_kanban.md) |
| Memory System | [specs/features/memory.md](../specs/features/memory.md)                 | [ai_docs/patterns/prompt_chain_memory.md](patterns/prompt_chain_memory.md)           |
| Objectives    | [specs/features/objectives.md](../specs/features/objectives.md)         | [ai_docs/architecture/task_execution_flow.md](architecture/task_execution_flow.md)   |
| Tools         | [specs/features/tools.md](../specs/features/tools.md)                   | _Implementation guide needed_                                                        |

## 💻 Implementation Patterns

### Go Patterns

| Pattern                     | Description                                        | Location                                                                   | [48;76;141;2736;2538t |
| --------------------------- | -------------------------------------------------- | -------------------------------------------------------------------------- | --------------------- |
| Go Concurrency              | Goroutines, channels, and synchronization patterns | [ai_docs/patterns/go_concurrency.md](patterns/go_concurrency.md)           |
| Interface-First Development | Designing with interfaces before implementation    | [ai_docs/patterns/interface_first.md](patterns/interface_first.md)         |
| Prompt Chain Memory         | Managing and storing prompt chains                 | [ai_docs/patterns/prompt_chain_memory.md](patterns/prompt_chain_memory.md) |

### External API Documentation

| API         | Description                                 | Location                                                   |
| ----------- | ------------------------------------------- | ---------------------------------------------------------- |
| Claude Code | Using Claude Code for development           | [ai_docs/api_docs/claude_code.md](api_docs/claude_code.md) |
| OpenAI SDK  | Integrating with OpenAI API                 | [ai_docs/api_docs/openai_sdk.md](api_docs/openai_sdk.md)   |
| ZeroMQ      | Messaging system for internal communication | [ai_docs/api_docs/zeromq.md](api_docs/zeromq.md)           |

## 🔌 Integration Guides

| Integration         | Description                                | Location                                                                                       |
| ------------------- | ------------------------------------------ | ---------------------------------------------------------------------------------------------- |
| Agent Task Events   | Publishing and subscribing to agent events | [ai_docs/integration_guides/agent_task_events.md](integration_guides/agent_task_events.md)     |
| BoltDB Kanban       | Implementing Kanban with BoltDB            | [ai_docs/integration_guides/bolt_db_kanban.md](integration_guides/bolt_db_kanban.md)           |
| Qdrant Vector Store | Vector storage for semantic memory         | [ai_docs/integration_guides/qdrant_vector_store.md](integration_guides/qdrant_vector_store.md) |

## 🚀 Implementation Tasks

This section maps common implementation tasks to the relevant documentation:

### Implementing an Agent

1. Review requirements: [specs/features/agent-behavior.md](../specs/features/agent-behavior.md)
2. Understand the lifecycle: [ai_docs/architecture/agent_lifecycle.md](architecture/agent_lifecycle.md)
3. Learn concurrency patterns: [ai_docs/patterns/go_concurrency.md](patterns/go_concurrency.md)
4. Implement provider integration:
   - OpenAI: [ai_docs/api_docs/openai_sdk.md](api_docs/openai_sdk.md)
   - Claude: [ai_docs/api_docs/claude_code.md](api_docs/claude_code.md)
5. Review the interface: [pkg/agent/agent.go](../pkg/agent/agent.go)

### Building the Kanban System

1. Review requirements: [specs/features/kanban_board.md](../specs/features/kanban_board.md)
2. Understand BoltDB integration: [ai_docs/integration_guides/bolt_db_kanban.md](integration_guides/bolt_db_kanban.md)
3. Learn event system: [ai_docs/integration_guides/agent_task_events.md](integration_guides/agent_task_events.md)
4. Review the interfaces:
   - [pkg/kanban/board.go](../pkg/kanban/board.go)
   - [pkg/kanban/taskmodel.go](../pkg/kanban/taskmodel.go)

### Implementing Memory and RAG

1. Review requirements: [specs/features/memory.md](../specs/features/memory.md)
2. Understand prompt chains: [ai_docs/patterns/prompt_chain_memory.md](patterns/prompt_chain_memory.md)
3. Learn vector store integration: [ai_docs/integration_guides/qdrant_vector_store.md](integration_guides/qdrant_vector_store.md)
4. Review the interfaces:
   - [pkg/memory/interface.go](../pkg/memory/interface.go)
   - [pkg/memory/vector/interface.go](../pkg/memory/vector/interface.go)

### Building a Tool

1. Review requirements: [specs/features/tools.md](../specs/features/tools.md)
2. Understand the tool interface: [tools/tool.go](../tools/tool.go)
3. See example tools:
   - [tools/cryptocurrency/trade.go](../tools/cryptocurrency/trade.go)
   - [tools/scraper/scraper.go](../tools/scraper/scraper.go)

## 📋 Current Status and Missing Documentation

The following documentation would be helpful but is currently not available:

1. CLI Command implementation guide
2. Tool creation workflow
3. Testing strategy and patterns
4. Deployment and packaging guide

## 🛠️ How to Ask for Help

When asking Claude Code for help with Guild implementation:

1. **Be specific** about which component you're working on
2. **Reference relevant docs** from this index
3. **Include current code** if you're debugging or extending
4. **Specify interfaces** that you're implementing

Example query:

```
I'm implementing the BoltDB storage for the Kanban system described in ai_docs/integration_guides/bolt_db_kanban.md.
This needs to implement the interface in pkg/kanban/board.go. Here's my current code:

[your code]

Can you help me implement the Move method to change a task's status?
```

This approach will help Claude Code provide more accurate and contextual assistance for your Guild project.
