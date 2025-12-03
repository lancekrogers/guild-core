# Guild Development Guide

This document provides guidance for developing and contributing to the Guild framework.

## Development Environment Setup

### Prerequisites

- Go 1.23 or later
- Make (for build system)
- SQLite (for storage)
- Git

### Getting Started

1. Clone the repository:

   ```bash
   git clone https://github.com/guild-ventures/guild-core.git
   cd guild-core
   ```

2. Install dependencies:

   ```bash
   make deps
   ```

3. Build the project (Note: currently has build errors):

   ```bash
   make build
   ```

4. Run tests:

   ```bash
   make test
   ```

## Current Project Structure

```
guild-core/
├── cmd/                 # Command-line applications
│   ├── guild/           # Main Guild CLI (build errors)
│   ├── demos/           # Demo applications
│   └── rag_test/        # RAG testing utilities
├── pkg/                 # Public framework packages (can be imported)
│   ├── agent/           # Agent framework and execution
│   ├── campaign/        # Campaign management
│   ├── commission/      # Commission/objective system
│   ├── config/          # Configuration handling
│   ├── context/         # Context management
│   ├── corpus/          # Documentation indexing
│   ├── gerror/          # Error handling framework
│   ├── grpc/            # gRPC services (build errors)
│   ├── interfaces/      # Shared interfaces
│   ├── kanban/          # Task board management
│   ├── memory/          # Memory abstractions
│   ├── orchestrator/    # Multi-agent coordination
│   ├── project/         # Project management
│   ├── prompts/         # Prompt engineering system
│   ├── providers/       # LLM provider integrations
│   ├── registry/        # Component registry
│   ├── storage/         # Storage abstractions
│   ├── tools/           # Tool definitions
│   └── workspace/       # Workspace isolation
├── internal/            # Private application code
│   ├── chat/            # Chat UI implementation
│   ├── manager/         # Application manager
│   ├── memory/          # SQLite memory implementation
│   └── ui/              # Terminal UI components
├── db/                  # Database schema and migrations
│   ├── migrations/      # SQL migration files
│   └── queries/         # SQLC query definitions
├── docs/                # Documentation
├── examples/            # Example configurations
├── integration/         # Integration tests
├── proto/               # Protocol buffer definitions
└── tools/               # Development tools
```

## Key Components

### Agent System

The agent framework provides the core abstraction for AI-powered workers.

- **Core Interface**: `pkg/agent/interface.go`
- **Executor**: `pkg/agent/executor/` - Safe tool execution
- **Manager**: `pkg/agent/manager/` - Task routing and analysis

### Commission System (formerly Objectives)

Commissions define the goals and tasks for agents to work on.

- **Parser**: `pkg/commission/parser.go`
- **Models**: `pkg/commission/types.go`
- **Manager**: `pkg/commission/manager.go`

### Memory System

Guild uses SQLite for persistence and vector stores for RAG.

- **Interfaces**: `pkg/memory/interface.go`
- **RAG System**: `pkg/memory/rag/`
- **Vector Stores**: `pkg/memory/vector/`
- **SQL Implementation**: `internal/memory/`

### Registry Pattern

All components are accessible through a central registry.

- **Registry**: `pkg/registry/registry.go`
- **Interfaces**: `pkg/interfaces/`

## Build System

Guild uses Make as the primary build system:

```bash
# Build commands
make build              # Build all binaries
make build-guild        # Build just the CLI
make clean             # Clean all artifacts

# Test commands
make test              # Run all tests
make unit-test         # Unit tests only
make integration       # Integration tests
make coverage          # Generate coverage report

# Development helpers
make lint              # Run linters
make fmt               # Format code
make vet               # Run go vet
```

## Current Status & Known Issues

### Build Failures

The project currently has build failures in:

- `cmd/guild` - Main CLI
- `pkg/grpc` - gRPC services (Campaign interface mismatches)
- `internal/chat` - Chat UI

### Working Components

- Project initialization (`guild init`)
- Basic chat interface
- Corpus scanning and indexing
- Commission creation and refinement
- Provider integrations (Anthropic, OpenAI, Ollama)

### Development Priorities

1. Fix build errors in gRPC package
2. Complete Campaign/Commission interface alignment
3. Re-enable disabled tests
4. Improve test coverage

## Testing

### Test Organization

- Unit tests: Alongside source files (`*_test.go`)
- Integration tests: `integration/` directory
- Internal tests: `internal/*_test/` for testing unexported APIs

### Running Tests

```bash
# Never use go test directly!
make test              # All tests
make unit-test         # Unit tests only
make test PKG=./pkg/agent/...  # Specific package
```

## Code Standards

### Error Handling

Use the `gerror` package for all errors:

```go
import "github.com/guild-ventures/guild-core/pkg/gerror"

// Creating errors
err := gerror.New(gerror.ErrCodeInternal, "operation failed").
    WithComponent("agent").
    WithOperation("execute")

// Wrapping errors
wrapped := gerror.Wrap(err, gerror.ErrCodeInternal, "higher level failure")
```

### Medieval Naming

Maintain the medieval guild theme:

- Agents → Artisans
- Objectives → Commissions
- Task Board → Workshop Board
- Memory → Archives

### Code Organization

1. Define interfaces first
2. Use the registry pattern for component access
3. Keep public APIs in `pkg/`
4. Keep application code in `internal/`
5. Use meaningful package names

## Contributing

1. Check existing issues and discussions
2. Follow the code standards
3. Write tests for new functionality
4. Update documentation as needed
5. Ensure `make build` succeeds before submitting

---

**Note**: This guide reflects the current state of the codebase. Some features mentioned in other documentation may not be implemented yet.
