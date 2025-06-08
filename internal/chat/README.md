# Chat Package

The `internal/chat` package provides Guild's terminal user interface (TUI) for interactive AI agent chat sessions.

## Architecture

The package is organized into several components:

### Core Files
- `chat.go` - Main chat model and TUI logic (to be moved from cmd/guild)
- `interface.go` - Public package interface and options
- `types.go` - Core type definitions (ChatModel, messages, etc.)

### Feature Modules  
- `keys.go` - Keyboard shortcuts and help system
- `completion.go` - Auto-completion engine for commands, agents, and files
- `commands.go` - Slash command processing (/help, /agents, etc.)

### Visual Components (to be extracted)
- `markdown.go` - Rich markdown rendering
- `status.go` - Agent status displays  
- `content.go` - Content formatting

### Event Handlers (to be extracted)
- `update.go` - Bubble Tea Update method (~800 lines)
- `view.go` - Bubble Tea View method (~600 lines)
- `events.go` - Message handling and state updates

## Usage

```go
import "github.com/guild-ventures/guild-core/internal/chat"

// Create and run chat interface
err := chat.Run(ctx, guildConfig, campaignID, sessionID)
```

## Status

Currently refactoring from a 4,500+ line monolithic file into a well-organized package.

### Progress
- [x] Extract type definitions
- [x] Extract key bindings  
- [x] Extract completion engine
- [x] Extract command processing
- [x] Create package structure
- [ ] Move main chat logic
- [ ] Extract Update method
- [ ] Extract View method
- [ ] Add comprehensive tests

### Line Count Reduction
- Original: 4,210 lines
- Current: 3,273 lines (22% reduction)
- Target: <500 lines per file