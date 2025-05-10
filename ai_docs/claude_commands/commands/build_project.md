## Guild Project Implementation Plan

@context
@lore_conventions

I'll help you build the Guild project systematically, tracking our progress in a document. Let's follow these steps:

1. **Initialize Project Structure**: Set up directory structure and core files
2. **Implement Core Interfaces**: Define primary interfaces for all components
3. **Build Components**: Implement each component in dependency order
4. **Create CLI**: Implement the command-line interface
5. **Add Integration Tests**: Create tests across components
6. **Documentation**: Complete user and developer documentation

I'll maintain a PROGRESS.md file tracking our implementation status, decisions, and next steps.

## Component Implementation Order

Based on dependencies between components, let's implement in this order:

1. **Providers**: LLM provider interfaces and implementations
2. **Memory**: Storage interfaces and implementations (BoltDB, Qdrant)
3. **Kanban**: Task tracking system with BoltDB backend
4. **Tools**: Tool interfaces and implementations
5. **Corpus**: Research corpus storage and retrieval system
6. **Objectives**: Objective parsing and management
7. **Agents**: Agent interfaces and implementations
8. **Orchestrator**: Multi-agent coordination and MCP
9. **CLI**: Command-line interface for user interaction

Let's start by initializing the progress tracking document and beginning implementation.

# Guild Component Implementation Details

This document provides a detailed breakdown of each component in the Guild framework, including specific implementation steps, dependencies, and key considerations.

## Implementation Order

Components are listed in recommended implementation order based on dependencies:

1. **Providers**: LLM provider interfaces and implementations
2. **Memory**: Storage interfaces and implementations
3. **Kanban**: Task tracking system
4. **Tools**: Tool interfaces and implementations
5. **Corpus**: Research corpus system
6. **Objectives**: Objective parsing and management
7. **Agents**: Agent interfaces and implementations
8. **Orchestrator**: Multi-agent coordination
9. **CLI**: Command-line interface

## 1. 🔌 Providers

**Purpose**: Interface with different LLM backends (OpenAI, Anthropic, Ollama, Ora)

### Implementation Steps

1. **Core Interfaces**

   - Define `Provider` interface in `pkg/providers/interface.go`
   - Create request/response structs
   - Define provider factory interface

2. **OpenAI Provider**

   - Implement OpenAI client in `pkg/providers/openai/client.go`
   - Support Chat Completions API
   - Implement streaming functionality
   - Add proper error handling and retries
   - Create cost calculation function

3. **Anthropic Provider**

   - Implement Anthropic client in `pkg/providers/anthropic/client.go`
   - Support Messages API
   - Implement streaming functionality
   - Add proper error handling and retries
   - Create cost calculation function

4. **Ollama Provider**

   - Implement Ollama client in `pkg/providers/ollama/client.go`
   - Support Generation API
   - Add local model detection
   - Implement health checks

5. **Ora Provider**

   - Implement Ora client in `pkg/providers/ora/client.go`
   - Support their unified API
   - Add automatic fallback handling

6. **Provider Factory**

   - Implement factory in `pkg/providers/factory.go`
   - Add registry for provider types
   - Support environment variable config

7. **Provider Tests**
   - Create mock provider in `pkg/providers/mock/provider.go`
   - Write unit tests for each provider
   - Create integration tests with test API keys

### Dependencies

- External provider SDKs or HTTP clients

### Key Considerations

- Error handling and retries
- Token counting and cost tracking
- Rate limiting
- Context management

## 2. 💾 Memory System

**Purpose**: Store and retrieve prompt chains, vector embeddings, and other persistent data

### Implementation Steps

1. **Memory Interfaces**

   - Define memory store interface in `pkg/memory/interface.go`
   - Create prompt chain data structures
   - Define vector store interface

2. **BoltDB Implementation**

   - Implement BoltDB store in `pkg/memory/boltdb/store.go`
   - Create bucket structure
   - Implement CRUD operations for prompt chains
   - Add indexing by task and agent

3. **Vector Store Abstraction**

   - Create vector interface in `pkg/memory/vector/interface.go`
   - Define embedding data structures
   - Create similarity search interface

4. **Qdrant Implementation**

   - Implement Qdrant client in `pkg/memory/vector/qdrant.go`
   - Add collection management
   - Implement embedding storage
   - Create similarity search functions

5. **Embeddings Generation**

   - Implement OpenAI embeddings in `pkg/memory/vector/openai_embedder.go`
   - Add local embedding option

6. **Chain Manager**

   - Create chain manager in `pkg/memory/chain_manager.go`
   - Add prompt chain creation and updates
   - Implement context building from chains

7. **RAG System**

   - Implement retriever in `pkg/memory/rag/retriever.go`
   - Create context enhancement functions
   - Add relevance scoring

8. **Memory Tests**
   - Create mock stores
   - Test persistence operations
   - Test vector search functionality
   - Benchmark RAG performance

### Dependencies

- BoltDB
- Qdrant
- OpenAI (for embeddings)

### Key Considerations

- Performance optimization
- Efficient indexing
- Context window management
- Token optimization

## 3. 📋 Kanban System

**Purpose**: Track and manage tasks throughout their lifecycle

### Implementation Steps

1. **Kanban Interfaces**

   - Define task model in `pkg/kanban/taskmodel.go`
   - Create board interface
   - Define task status transitions

2. **Task Implementation**

   - Implement task data structure
   - Add validation functions
   - Create serialization methods

3. **BoltDB Board**

   - Implement board in `pkg/kanban/board.go`
   - Create bucket structure
   - Add CRUD operations for tasks
   - Implement status transitions

4. **Event System**

   - Define events in `pkg/kanban/events.go`
   - Create ZeroMQ integration in `pkg/comms/transport/zeromq/pubsub.go`
   - Add message serialization (JSON/Protobuf)
   - Implement event publishing with ZeroMQ
   - Create event subscription system

5. **Board Manager**

   - Create manager in `pkg/kanban/manager.go`
   - Add multi-board support
   - Implement board creation and retrieval

6. **Dependency Tracking**

   - Implement task dependency tracking
   - Add blocking/unblocking logic
   - Create dependency resolution

7. **Kanban Tests**
   - Create mock board
   - Test task transitions
   - Test event publishing
   - Verify persistence

### Dependencies

- BoltDB
- ZeroMQ
- Memory system

### Key Considerations

- Concurrent task modifications
- Event propagation
- State transition validation
- Task history

## 4. 🛠️ Tools System

**Purpose**: Provide interfaces for agents to use external tools and CLI commands

### Implementation Steps

1. **Tool Interfaces**

   - Define tool interface in `tools/tool.go`
   - Create tool registry
   - Define tool result structure

2. **CLI Tool Wrapper**

   - Implement CLI tool in `tools/cli/tool.go`
   - Add command execution
   - Implement templating for arguments
   - Add working directory management

3. **Common Tools**

   - Implement file tools in `tools/fs/tool.go`
   - Create HTTP tools in `tools/http/tool.go`
   - Add search tools

4. **Code Assistants**

   - Implement Aider integration in `tools/code/aider.go`
   - Add Claude Code integration in `tools/code/claude_code.go`

5. **Tool Factory**

   - Create factory in `tools/factory.go`
   - Add tool registration
   - Implement configuration loading

6. **Tool Cost Tracking**

   - Add cost calculation to tools
   - Implement usage tracking
   - Create cost optimization

7. **Tool Tests**
   - Create mock tools
   - Test command execution
   - Verify result parsing
   - Test error handling

### Dependencies

- External CLI tools
- Process execution

### Key Considerations

- Security (command injection)
- Error handling
- Resource management
- Tool isolation

## 5. 📝 Corpus System

**Purpose**: Store and organize research findings, summaries, and generated insights in a structured format

### Implementation Steps

1. **Corpus Models**

   - Define corpus document model in `pkg/corpus/models.go`
   - Create configuration structs
   - Define interfaces for storage and retrieval

2. **Storage Implementation**

   - Implement markdown storage in `pkg/corpus/storage.go`
   - Add size limit enforcement
   - Implement directory organization
   - Create metadata handling

3. **Link Management**

   - Implement wikilink parser in `pkg/corpus/links.go`
   - Add autolink functionality
   - Create graph generation
   - Implement graph storage in JSON

4. **UI Components**

   - Create corpus browser UI in `pkg/ui/corpus/browser.go`
   - Implement document viewer in `pkg/ui/corpus/viewer.go`
   - Add search interface in `pkg/ui/corpus/search.go`
   - Create graph visualization in `pkg/ui/corpus/graph.go`

5. **CLI Commands**

   - Implement corpus commands in `cmd/guild/corpus_cmd.go`
   - Add subcommands for viewing, adding, searching
   - Create statistics dashboard
   - Implement graph visualization command

6. **Tool Integration**

   - Add corpus write capability to tools
   - Implement size checking middleware
   - Create document processing pipeline
   - Add user activity tracking

7. **Corpus Tests**
   - Test storage and retrieval
   - Test link extraction and graph building
   - Test size limit enforcement
   - Test UI components

### Dependencies

- Filesystem access
- BubbleTea for UI
- Markdown parser

### Key Considerations

- Document format consistency
- Size limit enforcement
- Link validation
- User permissions
- Graph performance

## 6. 📄 Objectives System

**Purpose**: Parse and manage markdown objectives that define tasks for agents

### Implementation Steps

1. **Check Existing Implementation**

   - Look for existing objective-related files
   - Review existing code in `pkg/objective/` if it exists

2. **Prompt System**

   - Create prompt system in `internal/prompts/manager.go`
   - Implement objective prompt loader in `internal/prompts/objective/loader.go`
   - Add markdown prompt files in `internal/prompts/objective/markdown/`:
     - `creation.md` - Objective creation prompt
     - `ai_docs_gen.md` - AI docs generation prompt
     - `specs_gen.md` - Specs generation prompt
     - `refinement.md` - Objective refinement prompt
     - `suggestion.md` - Improvement suggestions prompt

3. **Generator Package**

   - Create generator interfaces in `pkg/generator/interface.go`
   - Implement objective generator in `pkg/generator/objective/generator.go`
   - Add methods for objective creation, documentation generation, etc.
   - Connect with providers for LLM access

4. **Objective Models**

   - Define objective model in `pkg/objective/models.go`
   - Create parser interface
   - Define objective hierarchy
   - Add lifecycle management (status tracking, versioning)

5. **Markdown Parser**

   - Implement parser in `pkg/objective/parser.go`
   - Add section extraction
   - Create tag parsing
   - Implement link resolution
   - Support both new and existing objectives

6. **Objective Manager**

   - Create manager in `pkg/objective/manager.go`
   - Add objective loading from filesystem
   - Implement objective indexing
   - Create search functions
   - Add status tracking

7. **Task Generation**

   - Implement task breakdown in `pkg/objective/task_generator.go`
   - Add task creation from objectives
   - Create dependency mapping

8. **UI Components**

   - Create Bubble Tea UI models in `pkg/ui/objective/model.go`
   - Implement view in `pkg/ui/objective/view.go`
   - Add update logic in `pkg/ui/objective/update.go`
   - Create dashboard in `pkg/ui/objective/dashboard.go`
   - Add editor components in `pkg/ui/objective/editor.go`

9. **CLI Integration**

   - Add command for individual objectives in `cmd/guild/objective_cmd.go`
   - Create dashboard command in `cmd/guild/objectives_cmd.go`
   - Implement option flags and commands

10. **Objective Tests**
    - Test markdown parser
    - Test objective models
    - Test prompt system
    - Test generators
    - Test UI components
    - Test CLI commands

### Dependencies

- Providers (for LLM access)
- Memory system (for storing objectives)
- Kanban (for creating tasks from objectives)
- Bubble Tea (for UI components)

### Key Considerations

- Markdown parsing accuracy
- Flexible objective formats
- UI responsiveness
- Proper integration with LLM providers
- Status tracking and versioning
- Filesystem organization

## 7. 🤖 Agents System

**Purpose**: Create and manage autonomous agents that execute tasks and use tools

### Implementation Steps

1. **Agent Interfaces**

   - Define agent interface in `pkg/agent/agent.go`
   - Create agent factory
   - Define agent state and context

2. **Agent Implementation**

   - Implement base agent in `pkg/agent/base_agent.go`
   - Add execution logic
   - Implement tool access
   - Create state management

3. **Agent Types**

   - Implement manager agent in `pkg/agent/manager_agent.go`
   - Create worker agent in `pkg/agent/worker_agent.go`
   - Add specialized agents as needed

4. **Agent Configuration**

   - Define configuration in `pkg/agent/config.go`
   - Add loading from YAML
   - Implement validation

5. **Prompt Management**

   - Create agent prompts in `pkg/agent/prompts/`
   - Add prompt chain management
   - Implement context building

6. **Agent Tests**
   - Create mock agents
   - Test execution flows
   - Verify tool usage
   - Test error handling

### Dependencies

- Providers (for LLM access)
- Memory system (for storing state)
- Kanban (for task management)
- Tools (for execution capabilities)
- Corpus (for knowledge access)

### Key Considerations

- Context management
- Tool selection
- Error recovery
- Cost optimization
- Corpus access control

## 8. 🧠 Orchestrator System

**Purpose**: Coordinate multiple agents to complete complex objectives and implement MCP

### Implementation Steps

1. **Orchestrator Interfaces**

   - Define orchestrator interface in `pkg/orchestrator/orchestrator.go`
   - Create dispatcher interface
   - Define event bus interface

2. **Dispatcher Implementation**

   - Implement dispatcher in `pkg/orchestrator/dispatcher.go`
   - Add agent assignment
   - Create task prioritization
   - Implement dependency resolution

3. **Event Bus Implementation**

   - Create event bus in `pkg/orchestrator/eventbus.go`
   - Add event publishing
   - Implement subscription management
   - Create event handler registration

4. **MCP Implementation**

   - Implement MCP in `pkg/orchestrator/mcp.go`
   - Add pattern detection
   - Create optimization suggestions
   - Implement tool candidate detection

5. **Runner Implementation**

   - Create runner in `pkg/orchestrator/runner.go`
   - Add concurrent execution
   - Implement resource management
   - Create progress tracking

6. **Integration with Git**

   - Implement Git workflow in `pkg/orchestrator/git_workflow.go`
   - Add branch management
   - Create merge handling
   - Implement conflict resolution

7. **Orchestrator Tests**
   - Test dispatcher logic
   - Verify event propagation
   - Test MCP pattern detection
   - Validate Git integration

### Dependencies

- Agents (for execution)
- Kanban (for task tracking)
- Memory (for state preservation)
- Tools (for Git operations)
- Corpus (for knowledge integration)

### Key Considerations

- Concurrency management
- Resource allocation
- Deadlock prevention
- Git branch isolation
- Merge conflict resolution

## 9. 🖥️ CLI System

**Purpose**: Provide a command-line interface for user interaction with Guild

### Implementation Steps

1. **CLI Framework**

   - Set up Cobra in `cmd/guild/main.go`
   - Define root command
   - Create command structure
   - Add global flags

2. **Bubble Tea Integration**

   - Create shared UI components in `pkg/ui/components/`
   - Implement consistent styling in `pkg/ui/style.go`
   - Add reusable models for common patterns
   - Create helper functions for Bubble Tea programs

3. **Core Commands**

   - Implement init in `cmd/guild/init.go`
   - Create run in `cmd/guild/run.go`
   - Add status in `cmd/guild/status.go`
   - Implement list in `cmd/guild/list.go`

4. **Component Commands**

   - Create agent commands in `cmd/guild/agent.go`
   - Add objective commands in `cmd/guild/objective.go`
   - Implement task commands in `cmd/guild/task.go`
   - Create tool commands in `cmd/guild/tool.go`
   - Add corpus commands in `cmd/guild/corpus_cmd.go`

5. **UI Components**

   - Implement dashboard components
   - Create interactive editors
   - Add progress indicators
   - Implement corpus browser and graph visualization

6. **Configuration**

   - Implement config load/save
   - Add environment variable support
   - Create profile management
   - Implement validation

7. **CLI Tests**
   - Test command execution
   - Verify flag parsing
   - Test output formatting
   - Validate error handling
   - Test UI components

### Dependencies

- All components (for functionality)
- Cobra (for CLI framework)
- Bubble Tea (for TUI)
- Lipgloss (for styling)

### Key Considerations

- User experience
- Command consistency
- Help documentation
- Error reporting
- Configuration management
- Terminal compatibility
