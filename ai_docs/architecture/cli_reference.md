## Guild CLI Reference

@context
@lore_conventions

This command provides a consolidated reference for the Guild CLI structure, commands, and implementation guidelines, with special emphasis on Bubble Tea integration and corpus system functionality.

### Guild CLI Overview

The Guild CLI follows these design principles:

- Medieval guild terminology for names and descriptions where appropriate
- Hierarchical command structure using Cobra
- Rich terminal UI built with Bubble Tea
- Human-in-the-loop capabilities for agent interaction
- Consistent navigation and interaction patterns

### Primary Command Structure

```
guild
├── init                # Initialize a new Guild project directory
├── objective           # Manage individual objectives
│   ├── create          # Create a new objective
│   ├── add-context     # Add context to objective
│   ├── regenerate      # Rebuild docs/specs from objective
│   ├── suggest         # Get improvement suggestions
│   └── ready           # Mark objective as ready
├── objectives          # Objectives dashboard (plural)
│   ├── list            # List all objectives
│   ├── status          # Show objectives status
│   └── filter          # Filter by status/tag
├── corpus              # Research corpus management
│   ├── view            # Browse corpus content
│   ├── add             # Add document to corpus
│   ├── search          # Search corpus content
│   ├── stats           # Show corpus statistics
│   └── graph           # Generate/view graph visualization
├── run                 # Execute agents against objectives
│   ├── agent           # Run a single agent
│   └── guild           # Run a full guild of agents
├── status              # Check system status
│   ├── tasks           # Show task board
│   ├── agents          # Show agent status
│   └── guilds          # Show guild status
├── tool                # Tool management
│   ├── register        # Register new tool
│   ├── list            # List available tools
│   └── run             # Execute a tool directly
└── config              # Configure Guild
    ├── list            # List configuration
    ├── set             # Set configuration value
    ├── model           # Configure LLM models
    └── corpus          # Configure corpus settings
```

### Bubble Tea Implementation Strategy

All CLI commands that provide interactive functionality should use Bubble Tea:

1. **Command Layer**:

   - Basic Cobra commands for entry points
   - Parameter parsing and validation
   - Launches appropriate Bubble Tea model

2. **Model Layer**:

   - Each interactive feature has its own Bubble Tea model
   - Models maintain state and handle events
   - Follow MVC pattern within models

3. **View Components**:
   - Reusable UI components in pkg/ui/components
   - Consistent styling and theming
   - Support for different terminal sizes

### Objective System Commands

The objective system implements these specific commands with Bubble Tea:

#### Single Objective Management

```
guild objective [objectivePath]
```

- Opens an interactive Bubble Tea UI for working with an objective
- With path: Opens that specific objective file
- Without path: Offers to create a new objective
- Displays current status and iteration count
- Provides command interface within the UI

```go
// Implementation pattern
func runObjectiveUI(objectivePath string) error {
    model := objective_ui.NewModel(objectivePath)
    p := tea.NewProgram(model, tea.WithAltScreen())
    _, err := p.Run()
    return err
}
```

#### Create New Objective

```
guild objective create
```

- Starts interactive objective creation process with Bubble Tea UI
- Prompts for natural language description
- Generates structured markdown objective
- Opens for review and editing

#### Add Context to Objective

```
guild objective add-context "<text>"
```

- Adds user-provided context to current objective
- Supports document references with @spec/path or @ai_docs/path notation
- Updates objective content with new context

### Corpus System Commands

The corpus system implements these commands with Bubble Tea:

#### View Corpus Content

```
guild corpus view [docPath]
```

- Opens an interactive Bubble Tea UI for browsing the corpus
- Without path: Shows corpus directory structure
- With path: Opens the specified document
- Supports navigation via links between documents
- Tracks document views for user activity logging

```go
// Implementation pattern
func runCorpusViewUI(docPath string) error {
    model := corpus_ui.NewViewModel(docPath)
    p := tea.NewProgram(model, tea.WithAltScreen())
    _, err := p.Run()
    return err
}
```

#### Add to Corpus

```
guild corpus add
```

- Opens an interactive UI for adding content to the corpus
- Supports various source types (URL, YouTube, text input)
- Processes content through appropriate tools
- Validates against corpus size limits
- Generates appropriate metadata

#### Search Corpus

```
guild corpus search <query>
```

- Opens interactive search UI
- Displays results with preview snippets
- Supports filtering by tags, date, and source
- Allows opening documents from search results

#### Corpus Statistics

```
guild corpus stats
```

- Shows interactive dashboard of corpus statistics
- Displays size, document count, tag cloud
- Shows recent additions and popular documents
- Visualizes corpus growth over time

#### Graph Visualization

```
guild corpus graph
```

- Generates and displays interactive graph of corpus documents
- Shows connections between documents
- Supports filtering and highlighting
- Provides navigation via graph nodes

### Corpus System Configuration

Configure corpus settings with:

```
guild config corpus
```

- Opens interactive configuration UI for corpus settings
- Set storage location
- Configure size limits
- Manage integration with tools

### UI Components Implementation

Bubble Tea components for the CLI should be organized in packages:

```
pkg/ui/
├── components/         # Shared UI components
│   ├── input.go        # Text input with history
│   ├── list.go         # List with selection
│   ├── modal.go        # Modal dialog
│   ├── help.go         # Help/keybinding display
│   └── style.go        # Common styles
├── objective/          # Objective-specific UI
│   ├── model.go        # Objective UI model
│   ├── view.go         # Rendering functions
│   └── update.go       # Event handling
└── corpus/             # Corpus-specific UI
    ├── model.go        # Corpus UI model
    ├── view.go         # Rendering functions
    ├── update.go       # Event handling
    └── graph.go        # Graph visualization
```

### Common UI Patterns

All Bubble Tea UIs should follow these patterns:

1. **Model Structure**

   ```go
   type Model struct {
       width, height int           // Terminal dimensions
       state         string        // Current UI state (e.g., "viewing", "editing")
       input         textinput.Model // Text input component
       list          list.Model     // List component if needed
       viewport      viewport.Model // Content viewport
       help          help.Model    // Help component
       keymap        keymap        // Key bindings
       // Feature-specific fields...
   }
   ```

2. **Init Method**

   ```go
   func (m Model) Init() tea.Cmd {
       return tea.Batch(
           textinput.Blink,
           // Other initialization commands...
       )
   }
   ```

3. **Update Method**

   ```go
   func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
       var cmds []tea.Cmd

       switch msg := msg.(type) {
       case tea.KeyMsg:
           // Handle key presses

       case tea.WindowSizeMsg:
           // Handle window resize

       // Handle custom message types...
       }

       // Update sub-components

       return m, tea.Batch(cmds...)
   }
   ```

4. **View Method**
   ```go
   func (m Model) View() string {
       // Render UI based on state
       switch m.state {
       case "viewing":
           // Render view state

       case "editing":
           // Render edit state
       }

       // Combine components and return
   }
   ```

### Corpus-Specific Implementation

When implementing corpus features:

1. **Document Management**

   - Follow the CorpusDoc structure from pkg/corpus
   - Check size limits before adding content
   - Generate appropriate metadata

2. **Link Processing**

   - Parse and render Obsidian-style [[wikilinks]]
   - Support navigation between linked documents
   - Generate graph data for visualization

3. **User Activity Tracking**

   - Log document views
   - Track navigation patterns
   - Use for relevance algorithms

4. **Graph Visualization**
   - Implement using Bubble Tea and appropriate terminal graphics
   - Support zooming and panning
   - Allow selection and navigation via graph

### Implementation Guidelines

When implementing CLI commands:

1. **Error Handling**

   - Provide clear, context-rich error messages
   - Handle errors at the appropriate level
   - Offer recovery suggestions when possible

2. **Progress Feedback**

   - Show progress for long-running operations
   - Provide real-time updates when possible
   - Use spinners or progress bars for extended operations

3. **Command Documentation**

   - Include detailed examples in help text
   - Document all flags and options
   - Provide context-sensitive help

4. **Configuration Integration**
   - Use consistent methods to access configuration
   - Respect user preferences from config
   - Provide reasonable defaults

These guidelines ensure the CLI commands are implemented consistently and provide a good user experience while adhering to the Guild project's naming conventions and architectural principles.
