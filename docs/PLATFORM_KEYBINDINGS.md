# Platform-Aware Keybindings

The Guild Framework chat interface now features intelligent platform-aware keybindings that adapt to your operating system, providing an optimal terminal experience on macOS, Linux, and Windows.

## Overview

The keybinding system automatically detects your platform and adjusts modifier keys to avoid conflicts with terminal emulators and system shortcuts.

### Platform Detection

- **macOS**: Uses Alt/Option (⌥) as the primary modifier
- **Linux**: Uses Ctrl as the primary modifier
- **Windows**: Uses Ctrl as the primary modifier

## Why Platform-Specific Keybindings?

### macOS Considerations

- Many terminal emulators on macOS intercept Ctrl+key combinations for their own use
- The Cmd (⌘) key is often not detectable by terminal applications
- Alt/Option (⌥) provides the most reliable modifier key for custom shortcuts
- Avoids conflicts with system shortcuts like Cmd+Q (quit) and Cmd+C (copy)

### Linux/Windows Considerations

- Ctrl is the standard modifier key for terminal applications
- Most users expect Ctrl-based shortcuts in CLI tools
- Terminal copy/paste often requires Ctrl+Shift+C/V

## Keybinding Reference

### Essential Commands

| Action | macOS | Linux/Windows |
|--------|-------|---------------|
| Submit Message | Enter | Enter |
| New Line | ⌥↵ (Alt+Enter) | Shift+Enter |
| Quit | ⌥Q / Esc | Ctrl+Q / Esc |
| Help | ⌥H | Ctrl+H |

### Navigation

| Action | macOS | Linux/Windows |
|--------|-------|---------------|
| Scroll Up | ↑ / k | ↑ / k |
| Scroll Down | ↓ / j | ↓ / j |
| Page Up | PgUp / ⌥U | PgUp / Ctrl+U |
| Page Down | PgDn / ⌥D | PgDn / Ctrl+D |
| Go to Start | Home / g | Home / g |
| Go to End | End / G | End / G |

### Features

| Action | macOS | Linux/Windows |
|--------|-------|---------------|
| Command Palette | ⌥K | Ctrl+K |
| Agent Status | ⌥A | Ctrl+A |
| Prompt Management | ⌥P | Ctrl+P |
| Global View | ⌥G | Ctrl+G |
| Clear Chat | ⌥L | Ctrl+L |
| Toggle View Mode | ⌥T | Ctrl+T |

### Text Operations

| Action | macOS | Linux/Windows |
|--------|-------|---------------|
| Copy | ⌥C | Ctrl+Shift+C |
| Paste | ⌥V | Ctrl+Shift+V |
| Search | ⌥/ | Ctrl+/ |
| Next Match | n | n |
| Previous Match | N | N |

### Advanced Features

| Action | macOS | Linux/Windows |
|--------|-------|---------------|
| Fuzzy File Finder | ⌥O | Ctrl+O |
| Global Search | ⌥⇧F | Ctrl+Shift+F |
| Toggle Vim Mode | ⌥⇧V | Ctrl+Alt+V |

### History

| Action | macOS | Linux/Windows |
|--------|-------|---------------|
| Previous Command | ⌥R | Ctrl+R |
| Next Command | ⌥F | Ctrl+F |

## Terminal Configuration Tips

### macOS Terminal.app

- Alt/Option keys work by default
- For Alt+Enter: Ensure "Use Option as Meta key" is enabled in Terminal preferences

### iTerm2 (macOS)

- Go to Preferences → Profiles → Keys
- Set "Option Key" to "Esc+" for proper Alt key behavior
- Alt+Enter should work without additional configuration

### GNOME Terminal (Linux)

- Alt keys may be used for menu access
- Disable menu shortcuts: Edit → Preferences → General → uncheck "Enable menu access keys"

### Windows Terminal

- Alt keys work by default
- Ctrl+Shift+C/V is the standard for copy/paste

## Implementation Details

The platform detection and keybinding system is implemented in:

- `internal/chat/platform.go` - Platform detection
- `internal/chat/keybindings.go` - Keybinding adapter
- `internal/chat/chat_keys.go` - Integration with chat interface

### Key Features

1. **Automatic Detection**: Uses `runtime.GOOS` to detect the platform
2. **Consistent API**: Same keybinding structure across platforms
3. **Smart Formatting**: Platform-specific display of shortcuts (e.g., ⌥Q vs Ctrl+Q)
4. **Backwards Compatible**: Existing code continues to work with the new system

### Testing

The implementation includes comprehensive tests for:

- Platform detection
- Keybinding generation
- Format conversion
- Help text generation

Run tests with:

```bash
go test ./internal/chat -v -run "TestPlatform|TestKeybinding"
```

## Troubleshooting

### macOS: Alt/Option key not working

1. Check terminal emulator settings for Option/Alt key behavior
2. Some terminals may need "Use Option as Meta key" enabled
3. Try using Esc instead of Alt as a fallback

### Linux: Ctrl shortcuts intercepted by terminal

1. Check if menu access keys are enabled
2. Consider using Alt as an alternative modifier
3. Some terminals reserve Ctrl+Shift for their own use

### Copy/Paste issues

1. Terminal copy/paste often differs from system copy/paste
2. Try both Ctrl+C/V and Ctrl+Shift+C/V variants
3. On macOS, Cmd+C/V works for system clipboard

## Future Enhancements

- User-configurable keybindings
- Keybinding profiles for different terminal emulators
- Visual keybinding editor
- Import/export keybinding configurations
