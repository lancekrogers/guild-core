# Guild Development Guide

This document provides guidance for developing and contributing to the Guild framework.

## Development Environment Setup

### Prerequisites

- Go 1.24 or later
- [Task](https://taskfile.dev/) for running development tasks
- ZeroMQ library for communication (installed with `task deps:install`)

### Getting Started

1. Clone the repository:
   ```bash
   git clone https://github.com/guild-ventures/guild-core.git
   cd guild-core
   ```

2. Install dependencies:
   ```bash
   task deps:install
   ```

3. Build the project:
   ```bash
   task build
   ```

4. Run tests:
   ```bash
   task test
   ```

## Project Structure

```
guild/
├── cmd/                 # Command-line applications
│   └── guild/           # Guild CLI
├── pkg/                 # Core packages
│   ├── agent/           # Agent implementations
│   ├── comms/           # Communication protocols
│   ├── config/          # Configuration handling
│   ├── kanban/          # Task management
│   ├── memory/          # Storage interfaces
│   ├── objective/       # Objective parsing
│   ├── orchestrator/    # Guild coordination
│   └── providers/       # LLM providers
├── tools/               # Tool implementations
├── examples/            # Example guild configurations
└── docs/                # Documentation
```

## Key Components

### Agents

Agents are the core workers in Guild. They use LLMs to perform tasks and communicate with other agents.

- **Base Agent**: `pkg/agent/agent.go`
- **Worker Agent**: `pkg/agent/worker_agent.go`
- **Manager Agent**: `pkg/agent/manager_agent.go`

### Objective System

Objectives define the goals and tasks for the agents to work on.

- **Objective Models**: `pkg/objective/models.go`
- **Objective Parser**: `pkg/objective/parser.go`
- **Objective Manager**: `pkg/objective/manager.go`

### Memory System

Guild uses a memory system to store and retrieve information.

- **BoltDB Store**: `pkg/memory/boltdb/store.go`
- **Chain Manager**: `pkg/memory/chain_manager.go`
- **Vector Stores**: `pkg/memory/vector/`

### Kanban Board

Tasks are managed on a Kanban board with different columns representing task states.

- **Board**: `pkg/kanban/board.go`
- **Task Model**: `pkg/kanban/taskmodel.go`
- **Manager**: `pkg/kanban/manager.go`

### Guild Hall UI

The Guild Hall is our TUI (Terminal User Interface) built with Bubble Tea and styled according to medieval guild themes.

- **Components**: `pkg/ui/components/`
- **Objective UI**: `pkg/ui/objective/`
- **Style Guide**: `docs/ui/styleguide.md`

The UI follows Guild's lore and naming conventions:
- **Hall**: Main application views
- **Chamber**: Section/panel components
- **Workshop**: Interactive components
- **Ledger**: Data display components

### LLM Providers

Guild supports multiple LLM providers through a common interface.

- **Interface**: `pkg/providers/interfaces/llm.go`
- **Factory**: `pkg/providers/factory.go`
- **Implementations**: `pkg/providers/{anthropic,openai,ollama}/`

### Tools

Tools are capabilities that agents can use to interact with the environment.

- **Tool Interface**: `tools/tool.go`
- **Shell Tool**: `tools/shell/shell_tool.go`
- **Scraper Tool**: `tools/scraper/scraper.go`

## Development Workflow

### Adding a New Feature

1. Create a feature branch:
   ```bash
   git checkout -b feature/my-new-feature
   ```

2. Implement the feature and add tests.

3. Run all tests:
   ```bash
   task test
   ```

4. Format and lint your code:
   ```bash
   task lint
   ```

5. Submit a pull request.

### Debugging

To run Guild with more verbose output:

```bash
task run CLI_ARGS="--debug objective list"
```

## Testing

### Running Tests

- Run all tests:
  ```bash
  task test
  ```

- Run unit tests only:
  ```bash
  task test:unit
  ```

- Run tests for a specific package:
  ```bash
  task test:packages PACKAGE="./pkg/objective"
  ```

### Test Coverage

Guild includes comprehensive test coverage tools:

- Generate basic coverage report:
  ```bash
  task test:coverage
  ```

- Generate coverage for working packages only:
  ```bash
  task test:coverage:working
  ```

- Generate detailed coverage by package:
  ```bash
  task test:coverage:detailed
  ```

- Generate a coverage badge for your README:
  ```bash
  task test:coverage:badge
  ```

### Test Verification and Analysis

Guild provides advanced test verification tools that help maintain code quality:

- Identify untested functions in a package:
  ```bash
  task test:verify PACKAGE="./pkg/objective"
  ```

- Verify test coverage for all working packages:
  ```bash
  task test:verify:all
  ```

- Analyze test patterns and quality:
  ```bash
  task test:analyze
  ```

### Guild Lore and Naming Conventions

Guild follows specific naming conventions based on medieval guild terminology. These tools help ensure tests adhere to these conventions:

- Check adherence to Guild naming conventions:
  ```bash
  task test:analyze:lore
  ```

- Lint tests for naming compliance:
  ```bash
  task test:lint:naming
  ```

- Generate a comprehensive test quality report that includes coverage, verification, and lore compliance:
  ```bash
  task test:report
  ```

#### Test Naming Conventions

Guild uses themed test names according to the following conventions:

| Test Type | Convention | Description |
|-----------|------------|-------------|
| Unit tests | `TestCraft<FunctionName>` | Tests for individual functions |
| Integration tests | `TestGuild<FeatureName>` | Tests for integrated components |
| Mock tests | `TestJourneyman<MockName>` | Tests using mock objects |
| Error tests | `TestApprentice<ErrorCase>` | Tests for error conditions |
| Benchmarks | `BenchmarkMaster<FunctionName>` | Performance benchmarks |

### Writing Tests

- Place test files in the same directory as the code being tested with a `_test.go` suffix
- Use Go's standard testing package
- Follow Guild naming conventions for test functions
- Use table-driven tests where appropriate
- Create mock implementations for interfaces in `mocks/` subdirectories
- Test both success and error cases
- Add thorough test comments explaining what each test verifies

## UI Development

The Guild Hall UI is built using [Bubble Tea](https://github.com/charmbracelet/bubbletea), a Go framework for building terminal user interfaces. All UI components follow Guild's medieval theme and lore.

### Setting Up UI Environment

```bash
# Install UI dependencies (Bubble Tea, Bubbles, and Lipgloss)
task ui:dev:setup
```

### Creating New UI Components

Guild UI components follow a metaphor-based naming convention:

- **Hall**: Main views that contain multiple components
- **Chamber**: Panels or sections within a Hall
- **Workshop**: Interactive areas where users perform actions
- **Ledger**: Components that display structured data

To scaffold a new component:

```bash
# Generate a new UI component
task ui:scaffold COMPONENT=task_list TYPE=ledger

# This will create:
# - pkg/ui/components/task_list/model.go
# - pkg/ui/components/task_list/task_list_test.go
```

Generated components follow Guild's naming and styling conventions automatically.

### UI Development Workflow

1. Use `task ui:scaffold` to create new components
2. Run the UI in development mode with hot reloading:
   ```bash
   task ui:dev:run
   ```
3. Test the UI components with Guild lore verification:
   ```bash
   task ui:test
   ```
4. Check your component against the style guide:
   ```bash
   task ui:styleguide
   ```

### UI Documentation and Style Guide

The UI style guide defines the visual language and interaction patterns:

```bash
# Generate or update the UI style guide
task ui:styleguide

# Generate Guild Hall UI documentation
task ui:docs:generate
```

The generated documentation can be found in the `docs/ui/` directory.

### UI Testing Conventions

UI tests should follow Guild's test naming conventions:

- `TestCraft<Component>`: Tests for component creation
- `TestGuild<Component>`: Tests for component integration
- `TestJourneyman<Component>`: Tests for mock interactions
- `TestApprentice<Component>`: Tests for error handling

## Building and Running

### Building

```bash
task build
```

This will create the Guild binary in the `bin/` directory.

### Running

```bash
task run CLI_ARGS="objective list"
```

Or directly:

```bash
./bin/guild objective list
```

## Releasing

### Creating a New Release

1. Update version in `cmd/guild/main.go`.
2. Tag the release:
   ```bash
   git tag -a v0.1.0 -m "Release v0.1.0"
   git push origin v0.1.0
   ```

3. Build binaries:
   ```bash
   task build
   ```

## Roadmap

Current priorities for Guild development:

1. **Core Features**:
   - Complete the agent communication system
   - Enhance the objective parsing and processing
   - Improve the memory system with better context retrieval

2. **User Experience**:
   - Develop the CLI further with more commands
   - Add a TUI (Terminal User Interface) for objective management
   - Improve error handling and user feedback

3. **Integrations**:
   - Add more LLM providers
   - Develop additional tools
   - Create integration points for external systems

## Troubleshooting

### Common Issues

#### ZeroMQ Dependency Issues

If you encounter issues with ZeroMQ:

```bash
# On macOS
brew install zeromq

# On Ubuntu/Debian
sudo apt-get install libzmq3-dev

# On Windows with Chocolatey
choco install zeromq
```

#### Import Cycles

If you encounter import cycle errors:

1. Use interface packages to break dependencies
2. Move shared types to common packages
3. Use dependency injection to manage references

## Using Claude Code for Development

For more information on using Claude Code for Guild development, please see the guide in the `ai_docs/Getting Started with Guild Claude Code Guide.md` file. The document covers:

- Setting up Claude Code for Guild development
- Using commands to provide context to Claude
- Development workflows and best practices
- Troubleshooting and common tasks

## Getting Help

If you need help or want to discuss Guild development:

- Open an issue on GitHub
- Check the documentation in the `docs/` directory