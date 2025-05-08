## Guild Project Implementation Plan

@context

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
5. **Agents**: Agent interfaces and implementations
6. **Objectives**: Objective parsing and management
7. **Orchestrator**: Multi-agent coordination and MCP
8. **CLI**: Command-line interface for user interaction

Let's start by initializing the progress tracking document and beginning implementation.

# Guild Component Implementation Details

This document provides a detailed breakdown of each component in the Guild framework, including specific implementation steps, dependencies, and key considerations.

## Implementation Order

Components are listed in recommended implementation order based on dependencies:

1. **Providers**: LLM provider interfaces and implementations
2. **Memory**: Storage interfaces and implementations
3. **Kanban**: Task tracking system
4. **Tools**: Tool interfaces and implementations
5. **Objectives**: Objective parsing and management
6. **Agents**: Agent interfaces and implementations
7. **Orchestrator**: Multi-agent coordination
8. **CLI**: Command-line interface

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
   - Create ZeroMQ integration
   - Implement event publishing

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

## 5. 📝 Objectives System

**Purpose**: Parse and manage markdown objectives that define tasks for agents

### Implementation Steps

1. **Objective Interfaces**

   - Define objective model in `pkg/objective/objective.go`
   - Create parser interface
   - Define objective hierarchy

2. **Markdown Parser**

   - Implement parser in `pkg/objective/markdown/parser.go`
   - Add section extraction
   - Create tag parsing
   - Implement link resolution

3. **Objective Manager**

   - Create manager in `pkg/objective/manager.go`
   - Add objective loading from filesystem
   - Implement objective indexing
   - Create search functions

4. **Task Generation**

   - Implement task breakdown in `pkg/objective/task_generator.go`
   - Add task creation from objectives
   - Create dependency mapping

5. **Template System**

   - Implement templates in `pkg/objective/template/template.go`
   - Add template rendering
   - Create template library

6. **Objective Tests**
   - Test markd
