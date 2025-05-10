## Module Structure

@context

This command outlines the expected module structure for the Guild project. Follow this structure when implementing new components.

### Standard Go Project Layout

Guild follows the standard Go project layout with these key directories:

```
guild/
├── cmd/                    # Command-line applications
│   └── guild/              # Main guild CLI application
│       ├── main.go
│       └── [command].go    # Specific command implementations
│
├── pkg/                    # Public libraries that can be imported by other projects
│   ├── agent/              # Agent components
│   ├── kanban/             # Kanban task system
│   ├── memory/             # Memory and persistence
│   ├── objective/          # Objective parsing and management
│   ├── orchestrator/       # Multi-agent coordination
│   ├── providers/          # LLM provider integrations
│   ├── generator/          # Content generation system
│   │   ├── interface.go    # Generator interfaces
│   │   └── objective/      # Objective-specific generators
│   └── ui/                 # UI components
│       └── objective/      # Objective UI with Bubble Tea
│
├── internal/               # Private application code
│   └── prompts/            # System prompts
│       ├── manager.go      # Prompt manager
│       └── objective/      # Objective-specific prompts
│           ├── loader.go
│           └── markdown/   # Markdown prompt files
│
├── tools/                  # Tool definitions and implementations
│
├── ai_docs/                # Documentation for AI assistants
│
├── specs/                  # Project specifications
│
└── .claude/                # Claude Code commands
```

### Module Import Structure

Follow these import path conventions:

```go
// Public component imports
import (
    "github.com/blockhead-consulting/guild/pkg/agent"
    "github.com/blockhead-consulting/guild/pkg/objective"
    "github.com/blockhead-consulting/guild/pkg/generator"
    // etc.
)

// Internal imports (only within the project)
import (
    "github.com/blockhead-consulting/guild/internal/prompts"
    // etc.
)
```

### Package Organization Guidelines

1. **Interface Separation**

   - Define interfaces in `interface.go` files
   - Keep implementation files separate
   - Example:
     ```
     pkg/generator/interface.go     # Interfaces
     pkg/generator/objective/generator.go  # Implementation
     ```

2. **Type and Model Separation**

   - Define types in `models.go` or `types.go` files
   - Keep business logic in separate files
   - Example:
     ```
     pkg/objective/models.go    # Data structures
     pkg/objective/parser.go    # Business logic
     ```

3. **Testing Organization**

   - Create test files alongside implementation files
   - Put test fixtures in `testdata/` directories
   - Include both unit and integration tests
   - Example:
     ```
     pkg/objective/parser.go
     pkg/objective/parser_test.go
     pkg/objective/testdata/sample_objective.md
     ```

4. **Package Documentation**
   - Include package-level documentation in a `doc.go` file
   - Example:
     ```go
     // Package objective provides functionality for parsing and
     // managing markdown objectives in the Guild framework.
     package objective
     ```

### Objective System Implementation Structure

Follow this structure specifically for the objective system implementation:

```
internal/prompts/
└── objective/
    ├── loader.go             # Prompt loader
    └── markdown/
        ├── creation.md       # Objective creation prompt
        ├── ai_docs_gen.md    # AI docs generation prompt
        ├── specs_gen.md      # Specs generation prompt
        ├── refinement.md     # Objective refinement prompt
        └── suggestion.md     # Improvement suggestions prompt

pkg/generator/
├── interface.go              # Generator interfaces
└── objective/
    ├── generator.go          # Objective generator implementation
    └── generator_test.go     # Tests for objective generator

pkg/objective/
├── models.go                 # Objective data models
├── parser.go                 # Markdown parser
├── lifecycle.go              # Lifecycle management
└── manager.go                # Objective management

pkg/ui/objective/
├── model.go                  # Bubble Tea model
├── view.go                   # UI rendering
├── update.go                 # Event handling
└── dashboard.go              # Objectives dashboard

cmd/guild/
├── objective_cmd.go          # Command for single objective
└── objectives_cmd.go         # Command for objectives dashboard
```

When adding new functionality, ensure it follows this structure to maintain consistency across the project.
