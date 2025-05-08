# Guild Development with Claude Code

This guide explains how to effectively use Claude Code for developing the Guild framework. It covers setup, commands, workflows, and best practices for collaborating with Claude Code throughout the development process.

## Table of Contents

- [Setting Up Claude Code](#setting-up-claude-code)
- [Command System](#command-system)
- [Development Workflows](#development-workflows)
- [Best Practices](#best-practices)
- [Troubleshooting](#troubleshooting)
- [Common Tasks](#common-tasks)

## Setting Up Claude Code

### Initial Setup

1. **Install Claude Code**

   - Visit [https://console.anthropic.com/claude-code](https://console.anthropic.com/) to request access
   - Follow the installation instructions for your platform

2. **Project Configuration**

   - Clone the Guild repository

   ```bash
   git clone https://github.com/yourusername/guild.git
   cd guild
   ```

   - Create a `.claude` directory for commands

   ```bash
   mkdir -p .claude/commands
   ```

3. **Claude Code Authentication**
   - Follow the instructions for authenticating your Claude Code installation
   - Ensure Claude Code has access to the Guild repository

### Command Structure

Create the command structure in the `.claude` directory:

```
guild/
├── .claude/
│   ├── commands/
│   │   ├── README.md            # Command index
│   │   ├── context.md           # Project context
│   │   ├── implement_agent.md   # Agent implementation
│   │   ├── implement_memory.md  # Memory implementation
│   │   ├── ensure_tests.md      # Testing requirements
│   │   └── ...                  # Other commands
```

## Command System

Claude Code uses commands to provide context and instructions for specific tasks. Here are the core commands you should set up:

### Core Commands

| Command                | File                     | Purpose                                 |
| ---------------------- | ------------------------ | --------------------------------------- |
| `@context`             | `context.md`             | Load project context and specifications |
| `@implement_component` | `implement_component.md` | Generic component implementation        |
| `@ensure_tests`        | `ensure_tests.md`        | Test requirements and patterns          |
| `@review_code`         | `review_code.md`         | Code review guidelines                  |
| `@fix_issues`          | `fix_issues.md`          | Debugging and issue resolution          |

### Setting Up the Context Command

Create `.claude/commands/context.md`:

````markdown
## Project Context

Please review the following key resources to understand the Guild project:

1. First, read the Specs Index at ai_docs/specs_index.md
2. Then, check the AI Docs Index at ai_docs/README.md
3. Review the Guild lore at specs/naming_conventions_and_lore/lore.md

## Project Structure

Run these commands to understand the project structure:

```bash
git ls-files | grep -E '\.(go|md)$' | sort
tree -I "node_modules|.git|.idea|bin" --dirsfirst
go list -m all
```
````

## Implementation Principles

Remember:

1. Guild uses interface-first development
2. Follow Go concurrency patterns with goroutines and channels
3. Always propagate context.Context
4. Implement cost-aware decision making
5. ALL code must include comprehensive unit tests

````

### Component Implementation Commands

Create component-specific commands (example for agent implementation):

```markdown
## Implement Agent Component

Please help me implement the Agent component with these steps:

1. First, review the Agent specification at specs/features/agent-behavior.md
2. Then, check the implementation guide at ai_docs/architecture/agent_lifecycle.md
3. Follow the interface-first pattern from ai_docs/patterns/interface_first.md

## Implementation Requirements

The Agent component should:
1. Implement the Provider interface for LLM interactions
2. Support tools through a standardized interface
3. Maintain a personal Kanban board
4. Use cost-aware decision making
5. Execute tasks through prompt chains

## Implementation Approach

1. First, let's define the Agent interface (test-first)
2. Then implement a BasicAgent that satisfies this interface
3. Add tests to verify the implementation
4. Ensure error handling for all external operations
````

## Development Workflows

### Starting a New Component

1. **Initialize Context**

   - Start Claude Code
   - Load the context and review specifications

   ```
   @context
   ```

2. **Plan Implementation**

   - Discuss the component architecture with Claude Code

   ```
   Let's plan how to implement the [component] based on the specs.
   ```

3. **Write Tests First**

   ```
   @ensure_tests
   Let's start by writing tests for the [component] interface.
   ```

4. **Implement Component**

   ```
   @implement_[component]
   Now let's implement the concrete types that satisfy our tests.
   ```

5. **Review Implementation**
   ```
   @review_code
   Please review this implementation for adherence to our patterns and practices.
   ```

### Debugging and Issue Resolution

1. **Describe the Issue**

   ```
   I'm encountering an issue with [component]. Here's the error:
   [error message]
   ```

2. **Fix Issues**

   ```
   @fix_issues
   Here's the problematic code:
   [code snippet]
   ```

3. **Add Tests for the Fix**
   ```
   @ensure_tests
   Let's add tests to verify the fix works correctly.
   ```

### Code Review Workflow

1. **Request Review**

   ```
   @review_code
   Please review this implementation:
   [code snippet]
   ```

2. **Address Feedback**

   ```
   Let's address the feedback by refactoring this section:
   [code section]
   ```

3. **Verify Changes**
   ```
   Please verify the changes address the feedback and maintain test coverage.
   ```

## Best Practices

### Effective Communication with Claude Code

1. **Be Specific**

   - Provide clear, specific questions or requests
   - Refer to specific files or code sections

2. **Provide Context**

   - Use context commands to ensure Claude Code understands the project
   - Share relevant code snippets when discussing issues

3. **Use Technical Terminology**

   - Claude Code understands software development concepts
   - Use Go-specific terms when appropriate

4. **Iterative Development**
   - Break down large tasks into smaller steps
   - Build on previous interactions

### Code Generation Guidelines

1. **Test-First Approach**

   - Always ask for tests before implementation
   - Ensure tests cover normal and error cases

2. **Interface-First Development**

   - Define interfaces before concrete implementations
   - Review interfaces before proceeding to implementation

3. **Complete Implementation**

   - Ask Claude Code to complete entire files, not just fragments
   - Include error handling, documentation, and tests

4. **Code Review**
   - Have Claude Code review its own generated code
   - Ask for specific improvements

### Project Management

1. **Track Progress**

   - Create a Kanban board for development tasks
   - Mark completed components and outstanding tasks

2. **Component Status Tracking**

   - Maintain a list of components with implementation status
   - Note dependencies between components

3. **Documentation Updates**
   - Update documentation as components are implemented
   - Ensure AI docs reflect the actual implementation

## Troubleshooting

### Common Issues

1. **Context Limitations**

   - **Issue**: Claude Code may not retain full context of large codebase
   - **Solution**: Provide relevant file snippets in your messages

2. **Implementation Gaps**

   - **Issue**: Claude Code might miss implementation details
   - **Solution**: Use the review workflow to identify and fill gaps

3. **Go-Specific Patterns**

   - **Issue**: Generated code may not follow idiomatic Go
   - **Solution**: Reference the Go concurrency and interface patterns

4. **Inconsistent Naming**
   - **Issue**: Naming might deviate from Guild conventions
   - **Solution**: Reference the lore document for naming guidance

### Recovering Context

If Claude Code seems to have lost context:

```
@context
Let's review where we are with the [component] implementation. Here's what we've done so far:
[summary of progress]
```

## Common Tasks

### Setting Up a New Component

```
@context
@ensure_tests

I want to implement the [component] described in specs/features/[component].md.

Let's follow these steps:
1. Define the interfaces and data structures
2. Write tests for the interfaces
3. Implement concrete types
4. Verify with additional tests

Let's start with the interfaces.
```

### Implementing Agent Providers

```
@context
@implement_agent

I want to implement a new provider for [provider_name]. The provider should:
1. Implement the Provider interface
2. Handle request/response formatting
3. Support streaming responses
4. Calculate costs appropriately

Let's start with the interface implementation.
```

### Adding a New Tool

```
@context

I want to add a new tool for [purpose]. The tool should:
1. Implement the Tool interface
2. Execute [specific action]
3. Handle errors appropriately
4. Be testable with mocks

Let's implement this tool following our patterns.
```

### Implementing the Kanban System

```
@context
@implement_kanban

Let's implement the Kanban system according to specs/features/kanban_board.md. It should:
1. Support task state transitions
2. Persist tasks in BoltDB
3. Publish events via ZeroMQ
4. Handle task assignments

Let's start with the Kanban interfaces.
```

### Building the CLI

```
@context

I want to implement the Guild CLI commands. It should:
1. Use Cobra for command structure
2. Support all required subcommands
3. Provide interactive setup
4. Have clear help documentation

Let's implement the core CLI structure.
```

---

## Command Reference

| Command                   | Purpose                     | When to Use                             |
| ------------------------- | --------------------------- | --------------------------------------- |
| `@context`                | Load project context        | Start of session or when context needed |
| `@implement_agent`        | Agent implementation        | When working on the agent component     |
| `@implement_memory`       | Memory implementation       | When working on the memory system       |
| `@implement_kanban`       | Kanban implementation       | When working on the task system         |
| `@implement_tools`        | Tools implementation        | When working on the tools system        |
| `@implement_orchestrator` | Orchestrator implementation | When working on the guild coordinator   |
| `@ensure_tests`           | Testing requirements        | When writing tests or implementations   |
| `@review_code`            | Code review                 | After completing implementation         |
| `@fix_issues`             | Issue resolution            | When debugging problems                 |
| `@go_patterns`            | Go development patterns     | When needing guidance on Go patterns    |

---

Remember that Claude Code is a collaborative tool. It works best when you provide clear guidance, review its output critically, and iterate on implementation. With the right approach, Claude Code can significantly accelerate Guild development while maintaining high code quality.
