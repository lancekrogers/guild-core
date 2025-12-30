# Guild Visual Components Architecture

## Overview

Guild's chat interface provides rich visual enhancements that create a professional development experience while maintaining the medieval theme. All visual components are implemented in the `internal/chat/` package.

## Component Architecture

### Core Visual Components

#### 1. Markdown Renderer (`markdown_renderer.go`)

- **Purpose**: Renders markdown content with syntax highlighting
- **Dependencies**:
  - `github.com/charmbracelet/glamour/v2` - Markdown rendering
  - `github.com/alecthomas/chroma/v2` - Syntax highlighting
- **Features**:
  - Medieval-themed styling with gold headers
  - Code block syntax highlighting for 8+ languages
  - Line numbers for code blocks > 5 lines
  - Performance caching for repeated content
  - Graceful degradation on errors

#### 2. Content Formatter (`content_formatter.go`)

- **Purpose**: High-level content formatting for different message types
- **Features**:
  - Agent response formatting with markdown support
  - System message formatting with importance detection
  - Error message formatting with red borders
  - Tool output formatting with orange theme
  - Content type detection (Plain, Markdown, Code, JSON, YAML)
  - Language inference for unmarked code blocks
  - Collapsible sections for long outputs

#### 3. Agent Status Display (`agent_status.go`, `status_display.go`)

- **Purpose**: Real-time agent activity monitoring
- **Components**:
  - `AgentStatusTracker` - Monitors agent states and activities
  - `StatusDisplay` - Renders status information
- **Features**:
  - Real-time status updates (idle, thinking, working, blocked)
  - Progress bars for long operations
  - Tool execution tracking
  - Cost accumulation display
  - Activity feed with timestamps

#### 4. Agent Indicators (`agent_indicators.go`)

- **Purpose**: Visual indicators for agent states
- **Features**:
  - Animated status indicators (⚪ Idle, 🤔 Thinking, ⚙️ Working, ✅ Done)
  - Multi-agent coordination display
  - Progress visualization
  - Time estimates

#### 5. Command Completion (`chat_completion.go`)

- **Purpose**: Intelligent auto-completion for commands and agents
- **Features**:
  - Tab completion for commands (/help, /status, etc.)
  - Agent name completion with fuzzy matching
  - File path completion
  - Task ID completion from kanban board
  - Context-aware suggestions

#### 6. Command History (`history.go`)

- **Purpose**: Persistent command history across sessions
- **Features**:
  - Arrow key navigation (↑/↓)
  - Persistent storage in `.guild/chat_history.txt`
  - Search through history
  - Session isolation

## Integration Architecture

### View Rendering Pipeline

1. **Message Reception**

   ```
   gRPC Stream → Message Handler → View Update
   ```

2. **Content Processing**

   ```
   Raw Content → Content Formatter → Markdown Renderer → Terminal Display
   ```

3. **Visual Enhancement Flow**

   ```go
   // In updateMessagesView() method
   if m.contentFormatter != nil {
       formatted = m.contentFormatter.FormatMessage(msgType, content, metadata)
   } else {
       formatted = m.safeFormatContent(msgType, content, agentID)
   }
   ```

### Component Initialization

All visual components are initialized in `newChatModel()`:

```go
// Initialize rich content rendering
markdownRenderer, _ := NewMarkdownRenderer(chatWidth)
contentFormatter := NewContentFormatter(markdownRenderer, chatWidth)

// Initialize command completion
completionEngine := NewCompletionEngine(guildConfig, projectRoot)
commandHistory := NewCommandHistory(projectRoot + "/.guild/chat_history.txt")

// Initialize agent status systems
statusTracker := NewAgentStatusTracker(guildConfig)
statusDisplay := NewStatusDisplay(statusTracker, width/4, height/3)
agentIndicators := NewAgentIndicators()
```

## Medieval Theme Standards

### Color Palette

```go
const (
    Background = "#1a1a1a"  // Castle stone
    Foreground = "#d4d4d4"  // Parchment
    Purple     = "#b794f6"  // Commands
    Gold       = "#f6e05e"  // Keywords/Headers
    Blue       = "#63b3ed"  // Types/Links
    Green      = "#68d391"  // Strings
    Red        = "#fc8181"  // Errors
    Orange     = "#f6ad55"  // Warnings
)
```

### Visual Elements

- Headers: `═══ Header ═══`
- Borders: `╭─╮│╰─╯`
- Lists: `•` (bullet points)
- Tasks: `[✓]` (completed), `[ ]` (pending)
- Quotes: `│` (indent token)

## Performance Considerations

1. **Caching**: Markdown renderer caches rendered content
2. **Throttling**: Status updates are throttled to prevent flicker
3. **Lazy Loading**: Large outputs are rendered progressively
4. **Terminal Detection**: Adapts to terminal width automatically

## Testing

Visual components are tested in `internal/chat_test/`:

- `markdown_renderer_test.go` - Markdown rendering tests
- `content_formatter_test.go` - Content formatting tests

Run tests with:

```bash
go test ./internal/chat_test/ -v
```

## Future Enhancements

1. **Theme System**: Support multiple themes beyond medieval
2. **Custom Highlighting**: User-defined syntax highlighting rules
3. **Visual Diff**: Side-by-side diff rendering for code changes
4. **Chart Support**: ASCII charts for metrics visualization
5. **Plugin System**: Allow custom visual components

## Integration Points

Visual components integrate with:

- **gRPC Services**: Receive formatted messages
- **Kanban Board**: Display task status
- **Tool System**: Show tool execution progress
- **Provider System**: Display model selection
- **Cost Tracking**: Show accumulated costs
