# Vim Mode Usage Guide

## Overview

The Guild chat interface now supports vim-style key bindings for power users. When vim mode is enabled, you can navigate and edit text using familiar vim commands.

## Enabling Vim Mode

To toggle vim mode on/off, use the `/vim` command in the chat:

```
/vim
```

When vim mode is enabled, you'll see a status indicator showing the current mode (NORMAL, INSERT, VISUAL, or COMMAND).

## Vim Modes

### Normal Mode
- Default mode when vim is enabled
- Navigate and perform commands
- Press `i` to enter Insert mode
- Press `v` to enter Visual mode
- Press `:` to enter Command mode

### Insert Mode
- Type text normally
- Press `ESC` to return to Normal mode

### Visual Mode
- Select text (limited functionality in current implementation)
- Press `ESC` to return to Normal mode

### Command Mode
- Enter vim commands
- Press `ESC` to cancel
- Press `Enter` to execute command

## Key Bindings

### Normal Mode Navigation
- `h` - Move cursor left (moves to start of line)
- `l` - Move cursor right (moves to end of line)
- `j` - Navigate down in history
- `k` - Navigate up in history
- `w` - Move forward by word (moves to end)
- `b` - Move backward by word (moves to start)
- `0` - Move to start of line
- `$` - Move to end of line

### Mode Switching
- `i` - Enter Insert mode
- `v` - Enter Visual mode
- `:` - Enter Command mode
- `ESC` - Return to Normal mode

### Editing Commands
- `x` - Delete character (simplified implementation)
- `o` - Insert new line and enter Insert mode (if multiline enabled)

### Command Mode Commands
- `:w` or `:write` - Save chat (not applicable for input)
- `:q` or `:quit` - Quit application
- `:wq` - Save and quit

## Implementation Notes

The vim mode integration with the chat input pane has some limitations:

1. **Cursor Movement**: Due to limitations in the underlying textarea component, precise cursor movement is simplified. `h` moves to start, `l` moves to end.

2. **Visual Mode**: Visual selection is not fully implemented in the current version.

3. **History Navigation**: `j` and `k` navigate through command history rather than moving lines (since the input is typically single-line).

4. **Insert Mode**: When in Insert mode, all keys except `ESC` are passed through to normal text input.

## Development Status

This is an initial implementation of vim mode. Future enhancements may include:
- Better cursor positioning
- Full visual mode support
- More vim commands
- Registers and yanking
- Search within input
- Macros

## Example Workflow

1. Start the chat interface
2. Type `/vim` to enable vim mode
3. You'll be in NORMAL mode by default
4. Press `i` to enter INSERT mode and type your message
5. Press `ESC` to return to NORMAL mode
6. Use `j`/`k` to navigate through history
7. Press `i` again to edit
8. Press `Enter` in INSERT mode to send the message