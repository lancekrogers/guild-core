# Guild Chat Keybindings Reference

## Critical Keybindings (FIXED)

### Exit Chat
- **Ctrl+Q** - Quit chat immediately
- **Esc** - Quit chat immediately  
- **Ctrl+D** - Quit chat immediately
- **/exit**, **/quit**, **/q** - Exit via command (now properly quits)

### Text Input
- **Enter** - Submit message
- **Shift+Enter** - Insert newline (multiline input) - FIXED
- **Ctrl+A** - Move to start of line (standard terminal behavior)
- **Ctrl+E** - Move to end of line (standard terminal behavior)

### Copy/Paste (Internal Clipboard)
- **Ctrl+Shift+C** - Copy current input or last message
- **Ctrl+Shift+V** - Paste from internal clipboard
- Note: Uses internal clipboard, not system clipboard

### Navigation
- **↑/k** - Scroll up in message history
- **↓/j** - Scroll down in message history
- **PgUp** - Page up
- **PgDn** - Page down
- **Home/g** - Go to start of chat
- **End/G** - Go to end of chat

### History
- **Ctrl+R** - Previous command from history
- **Ctrl+F** - Next command from history

### View Modes
- **Ctrl+H** - Show help
- **Ctrl+P** - Toggle prompt management view
- **Ctrl+A** - Toggle agent status view
- **Ctrl+G** - Toggle global stream view
- **Ctrl+T** - Toggle view mode

### Special Features
- **Ctrl+L** - Clear chat history
- **Ctrl+K** - Open command palette
- **Ctrl+O** - Fuzzy file finder
- **Ctrl+Shift+F** - Global search
- **Ctrl+/** - Search in chat
- **Tab** - Auto-complete commands

### Vim Mode
- **Ctrl+Alt+V** - Toggle vim mode on/off
- When enabled:
  - **i** - Enter insert mode (for typing)
  - **Esc** - Return to normal mode
  - **h/j/k/l** - Navigate left/down/up/right
  - **:q** - Quit (vim command)
  - **:w** - Save chat
  - **:wq** - Save and quit

## Commands

### Basic Commands
- **/help** or **/h** - Show help
- **/exit**, **/quit**, **/q** - Exit chat
- **/clear** or **/c** - Clear chat history
- **/status** or **/s** - Show guild status
- **/agents** or **/a** - List available agents

### Agent Communication
- **@agent_id message** - Send to specific agent
- **@all message** - Broadcast to all agents

### Session Management
- **/sessions** - List sessions
- **/session new [name]** - Create new session
- **/session rename <name>** - Rename current session
- **/session export [format]** - Export session (json/markdown/html)

## Troubleshooting

### If keys don't work:
1. Make sure you're not in vim mode (check status line)
2. Some terminals may intercept certain key combinations
3. Try the command version (e.g., /exit instead of Ctrl+Q)

### Terminal Compatibility:
- iTerm2, Terminal.app, Alacritty: All keybindings should work
- Some terminals may need configuration for Shift+Enter
- Ctrl+Shift combinations may conflict with terminal shortcuts

## Fixed Issues:
1. ✅ Shift+Enter now properly inserts newlines
2. ✅ Exit commands (/exit, /quit, /q) now properly quit
3. ✅ Esc, Ctrl+Q, Ctrl+D properly exit chat
4. ✅ Copy/Paste work with internal clipboard
5. ✅ Vim mode toggle (Ctrl+Alt+V) is connected