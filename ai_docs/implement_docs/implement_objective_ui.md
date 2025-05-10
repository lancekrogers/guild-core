## Implement Objective UI with Bubble Tea

@context
@lore_conventions

This guide focuses specifically on implementing the Objective UI components using the Bubble Tea TUI framework.

### 1. Check Existing UI Components

```bash
# Find any existing Bubble Tea components
find ./pkg -type f -name "*.go" | xargs grep -l "bubble\|tea\|lipgloss" | sort

# Check for existing UI models
find ./pkg -type f -name "*.go" | xargs grep -l "Model\|Init\|Update\|View" | sort

# Check for existing objective UI files
find ./pkg -type f -name "*.go" | xargs grep -l "objective.*ui\|ui.*objective" | sort
```

### 2. Understand UI Requirements

Review the UI specification at `specs/features/objectives/objective_ui.md` and note the key requirements:

- Individual objective editing
- Objective dashboard
- Status tracking
- Command processing

### 3. Implementation Structure

The Objective UI should follow this structure:

```
pkg/
└── ui/
    └── objective/
        ├── model.go       # Bubble Tea model and state
        ├── view.go        # UI rendering functions
        ├── update.go      # Event handling logic
        ├── commands.go    # UI command handling
        ├── dashboard.go   # Objectives dashboard view
        ├── editor.go      # Single objective editor
        └── components/    # Reusable UI components
            ├── input.go     # Text input component
            ├── preview.md   # Markdown preview component
            ├── status.go    # Status display component
            └── list.go      # Objective list component
```

### 4. Core Implementation Steps

#### A. Define Model

In `model.go`:

1. Define the Bubble Tea model structure:

   - Include state for editing mode, dashboard mode
   - Add fields for current objective, list of objectives
   - Include UI state like cursor position, scroll position, etc.
   - Add fields for user input and feedback messages

2. Implement `Init()` function to initialize the model

#### B. Implement View

In `view.go`:

1. Create rendering functions for different UI states:

   - Dashboard view
   - Editor view
   - Command input view
   - Preview view

2. Use Lipgloss for styling:
   - Define consistent styles for different UI elements
   - Create borders, padding, and colors
   - Handle terminal resizing

#### C. Handle Events

In `update.go`:

1. Implement the `Update()` function:

   - Handle keyboard events
   - Process command messages
   - Update model state
   - Switch between views

2. Add message types for:
   - Command execution results
   - LLM generation completion
   - File operations
   - Status changes

#### D. Create Dashboard

In `dashboard.go`:

1. Implement the objectives dashboard:

   - List all objectives with status
   - Show modification dates
   - Display completion percentages
   - Add filtering options

2. Create navigation functions:
   - Select objectives
   - Sort and filter
   - Open selected objective

#### E. Build Editor

In `editor.go`:

1. Create the objective editor:

   - Display current content
   - Allow editing sections
   - Show previews
   - Execute commands

2. Implement editing functions:
   - Add context
   - Generate stubs
   - Refine content
   - Mark as ready

### 5. UI Components

In `components/`:

1. Create reusable components:

   - Text input with history
   - Markdown preview with syntax highlighting
   - Status display
   - Command palette

2. Make components consistent:
   - Similar styling
   - Consistent keyboard shortcuts
   - Clear state transitions

### 6. Testing

For each UI component:

1. Create unit tests:

   - Test model updates
   - Verify view rendering
   - Check event handling

2. Create mock objects:

   - Mock objective store
   - Mock generators
   - Mock file system

3. Test full UI flows:
   - Creation
   - Editing
   - Dashboard navigation

### 7. Integration with CLI

Connect UI to CLI commands:

1. In `cmd/guild/objective_cmd.go`:

   - Initialize the UI model
   - Start Bubble Tea program
   - Handle command-line flags

2. Connect dashboard to `cmd/guild/objectives_cmd.go`

### Implementation Tips

1. **Start Simple**: Begin with a basic model that can be rendered
2. **Add Incrementally**: Build features one by one
3. **Test Interactively**: Use `go run ./cmd/guild objective` to test
4. **Use Debugger**: Bubble Tea can be debugged with a logger
5. **Consider Viewports**: Use bubbles/viewport for scrolling content
6. **Manage State**: Keep UI state in model, business logic in services

### Resources

- [Bubble Tea Examples](https://github.com/charmbracelet/bubbletea/tree/master/examples)
- [Lipgloss Documentation](https://github.com/charmbracelet/lipgloss)
- [Bubble Tea Components](https://github.com/charmbracelet/bubbles)
