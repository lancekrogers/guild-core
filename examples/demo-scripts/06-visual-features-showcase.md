# Visual Features Showcase Demo

This demo showcases all the rich visual enhancements in Guild's chat interface.

## Demo Duration: 5-7 minutes

## Pre-Demo Setup

```bash
# Initialize a new project
cd /tmp
mkdir visual-demo && cd visual-demo
guild init

# Start Guild chat
guild chat --campaign "visual-demo"
```

## Demo Script

### 1. Markdown Rendering Demo (1-2 minutes)

**User Input:**
```
/test markdown
```

**Expected Visual Output:**
- Headers with medieval styling (═══ Header ═══)
- Code blocks with syntax highlighting
- Lists with proper indentation
- Links with underline styling
- Bold and italic text rendering
- Tables with box-drawing characters

### 2. Syntax Highlighting Demo (1-2 minutes)

**User Input:**
```
@coder Create a Python function that calculates fibonacci numbers with memoization
```

**Expected Visual Output:**
- Python code with syntax highlighting
- Line numbers for code blocks > 5 lines
- Language detection and labeling
- Proper keyword coloring (gold for keywords, green for strings)
- Function definitions highlighted in blue

### 3. Agent Status Display Demo (1-2 minutes)

**User Input:**
```
Press Ctrl+A to view agent status panel
```

**Expected Visual Output:**
```
╭─ Guild Agent Status ─────────────────────────╮
│ Active Agents:                               │
│   🟢 @manager - Guild Master (idle)          │
│   🟡 @coder - Code Artisan (working: 45%)    │
│   🔴 @reviewer - Review Artisan (blocked)    │
│                                              │
│ Active Tools: 2                              │
│ Total Cost: $0.042                           │
│                                              │
│ Recent Activity:                             │
│   [14:32] Manager assigned task_123          │
│   [14:33] Coder started implementation       │
│   [14:34] Tool: file_writer executed         │
╰──────────────────────────────────────────────╯
```

### 4. Command Auto-Completion Demo (30 seconds)

**User Input:**
```
Type: /pr[TAB]
```

**Expected Behavior:**
- Auto-completes to `/prompt`
- Shows available subcommands

**User Input:**
```
Type: @co[TAB]
```

**Expected Behavior:**
- Shows completion options: @coder, @coordinator
- Displays agent capabilities inline

### 5. Multi-Agent Coordination View (1-2 minutes)

**User Input:**
```
@manager Create a REST API with authentication
```

**Expected Visual Output:**
- Real-time agent status updates
- Progress bars for long operations
- Tool execution indicators
- Animated status changes (⚪ → 🤔 → ⚙️ → ✅)

### 6. Error Display with Rich Formatting (30 seconds)

**User Input:**
```
@coder execute invalid-tool
```

**Expected Visual Output:**
```
╭─ Error ──────────────────────────────────────╮
│ ❌ Tool Execution Failed                     │
│                                              │
│ Error: Tool 'invalid-tool' not found         │
│                                              │
│ Available tools:                             │
│   • file_reader                              │
│   • file_writer                              │
│   • shell_executor                           │
│                                              │
│ Try: /tools list                             │
╰──────────────────────────────────────────────╯
```

## Key Visual Features to Highlight

1. **Medieval Theme Consistency**
   - Gold headers and keywords
   - Purple commands and inline code
   - Box-drawing borders
   - Guild terminology throughout

2. **Professional Code Display**
   - Syntax highlighting for 8+ languages
   - Line numbers for reference
   - Collapsible sections for long outputs
   - Diff rendering for file changes

3. **Real-Time Updates**
   - Agent status animations
   - Progress bars that update smoothly
   - Activity feed with timestamps
   - Cost tracking display

4. **Interactive Elements**
   - Tab completion with fuzzy matching
   - Command history (↑/↓ arrows)
   - Context-aware suggestions
   - Keyboard shortcuts overlay

## Post-Demo Talking Points

- "Notice how the medieval theme creates a unique, memorable experience"
- "The syntax highlighting makes code review natural in the chat interface"
- "Real-time agent status helps users understand what's happening behind the scenes"
- "Auto-completion reduces the learning curve for new users"
- "All visual enhancements maintain high performance even with large outputs"

## Troubleshooting

If visual features aren't showing:
1. Ensure terminal supports 256 colors
2. Check that Guild was built with visual features enabled
3. Try `/test visual` to verify component initialization
4. Check terminal width (minimum 80 columns recommended)

## Recording Tips

- Use a terminal with good color support (iTerm2, Windows Terminal)
- Set terminal size to 120x40 for optimal display
- Enable font ligatures for better code display
- Use a dark theme to match Guild's medieval aesthetic