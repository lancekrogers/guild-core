package chat

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
)

// KeybindingAdapter provides platform-aware key bindings
type KeybindingAdapter struct {
	platform Platform
}

// NewKeybindingAdapter creates a new keybinding adapter for the current platform
func NewKeybindingAdapter() *KeybindingAdapter {
	return &KeybindingAdapter{
		platform: DetectPlatform(),
	}
}

// NewKeybindingAdapterForPlatform creates a keybinding adapter for a specific platform
func NewKeybindingAdapterForPlatform(platform Platform) *KeybindingAdapter {
	return &KeybindingAdapter{
		platform: platform,
	}
}

// GetChatKeyMap returns platform-specific key bindings
func (ka *KeybindingAdapter) GetChatKeyMap() chatKeyMap {
	if ka.platform.IsMacOS() {
		return ka.getMacOSKeyMap()
	}
	return ka.getDefaultKeyMap()
}

// getMacOSKeyMap returns macOS-specific key bindings using Alt/Option as primary modifier
func (ka *KeybindingAdapter) getMacOSKeyMap() chatKeyMap {
	return chatKeyMap{
		Submit: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("↵", "submit message"),
		),
		NewLine: key.NewBinding(
			key.WithKeys("alt+enter"),
			key.WithHelp("⌥↵", "new line"),
		),
		Quit: key.NewBinding(
			key.WithKeys("alt+q", "esc", "ctrl+d"),
			key.WithHelp("⌥Q/Esc", "quit"),
		),
		Help: key.NewBinding(
			key.WithKeys("alt+h"),
			key.WithHelp("⌥H", "toggle help"),
		),
		Prompt: key.NewBinding(
			key.WithKeys("alt+p"),
			key.WithHelp("⌥P", "prompt management"),
		),
		Status: key.NewBinding(
			key.WithKeys("alt+a"),
			key.WithHelp("⌥A", "agent status"),
		),
		Global: key.NewBinding(
			key.WithKeys("alt+g"),
			key.WithHelp("⌥G", "global view"),
		),
		ScrollUp: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "scroll up"),
		),
		ScrollDown: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "scroll down"),
		),
		PageUp: key.NewBinding(
			key.WithKeys("pgup", "alt+u"),
			key.WithHelp("PgUp/⌥U", "page up"),
		),
		PageDown: key.NewBinding(
			key.WithKeys("pgdown", "alt+d"),
			key.WithHelp("PgDn/⌥D", "page down"),
		),
		Home: key.NewBinding(
			key.WithKeys("home", "g"),
			key.WithHelp("Home/g", "go to start"),
		),
		End: key.NewBinding(
			key.WithKeys("end", "G"),
			key.WithHelp("End/G", "go to end"),
		),
		PrevHistory: key.NewBinding(
			key.WithKeys("alt+r"),
			key.WithHelp("⌥R", "previous command"),
		),
		NextHistory: key.NewBinding(
			key.WithKeys("alt+f"),
			key.WithHelp("⌥F", "next command"),
		),
		Clear: key.NewBinding(
			key.WithKeys("alt+l"),
			key.WithHelp("⌥L", "clear chat"),
		),
		ToggleViewMode: key.NewBinding(
			key.WithKeys("alt+t"),
			key.WithHelp("⌥T", "toggle view mode"),
		),
		CommandPalette: key.NewBinding(
			key.WithKeys("alt+k"),
			key.WithHelp("⌥K", "command palette"),
		),
		CancelTool: key.NewBinding(
			key.WithKeys("alt+x"),
			key.WithHelp("⌥X", "cancel tool"),
		),
		RetryCost: key.NewBinding(
			key.WithKeys("alt+r"),
			key.WithHelp("⌥R", "retry with consent"),
		),
		AcceptCost: key.NewBinding(
			key.WithKeys("alt+y"),
			key.WithHelp("⌥Y", "accept cost"),
		),
		SelectNext: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("Tab", "next item"),
		),
		SelectPrev: key.NewBinding(
			key.WithKeys("shift+tab"),
			key.WithHelp("⇧Tab", "previous item"),
		),
		Copy: key.NewBinding(
			key.WithKeys("alt+c"),
			key.WithHelp("⌥C", "copy"),
		),
		Paste: key.NewBinding(
			key.WithKeys("alt+v"),
			key.WithHelp("⌥V", "paste"),
		),
		Search: key.NewBinding(
			key.WithKeys("alt+/"),
			key.WithHelp("⌥/", "search"),
		),
		NextMatch: key.NewBinding(
			key.WithKeys("n"),
			key.WithHelp("n", "next match"),
		),
		PrevMatch: key.NewBinding(
			key.WithKeys("N"),
			key.WithHelp("N", "previous match"),
		),
		FuzzyFinder: key.NewBinding(
			key.WithKeys("alt+o"),
			key.WithHelp("⌥O", "fuzzy file finder"),
		),
		GlobalSearch: key.NewBinding(
			key.WithKeys("alt+shift+f"),
			key.WithHelp("⌥⇧F", "global search"),
		),
		ToggleVimMode: key.NewBinding(
			key.WithKeys("alt+shift+v"),
			key.WithHelp("⌥⇧V", "toggle vim mode"),
		),
	}
}

// getDefaultKeyMap returns the default key bindings for Linux/Windows
func (ka *KeybindingAdapter) getDefaultKeyMap() chatKeyMap {
	return chatKeyMap{
		Submit: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("Enter", "submit message"),
		),
		NewLine: key.NewBinding(
			key.WithKeys("shift+enter", "ctrl+enter"),
			key.WithHelp("Shift+Enter", "new line"),
		),
		Quit: key.NewBinding(
			key.WithKeys("ctrl+q", "esc", "ctrl+d"),
			key.WithHelp("Ctrl+Q/Esc", "quit"),
		),
		Help: key.NewBinding(
			key.WithKeys("ctrl+h"),
			key.WithHelp("Ctrl+H", "toggle help"),
		),
		Prompt: key.NewBinding(
			key.WithKeys("ctrl+p"),
			key.WithHelp("Ctrl+P", "prompt management"),
		),
		Status: key.NewBinding(
			key.WithKeys("ctrl+a"),
			key.WithHelp("Ctrl+A", "agent status"),
		),
		Global: key.NewBinding(
			key.WithKeys("ctrl+g"),
			key.WithHelp("Ctrl+G", "global view"),
		),
		ScrollUp: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "scroll up"),
		),
		ScrollDown: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "scroll down"),
		),
		PageUp: key.NewBinding(
			key.WithKeys("pgup", "ctrl+u"),
			key.WithHelp("PgUp/Ctrl+U", "page up"),
		),
		PageDown: key.NewBinding(
			key.WithKeys("pgdown", "ctrl+d"),
			key.WithHelp("PgDn/Ctrl+D", "page down"),
		),
		Home: key.NewBinding(
			key.WithKeys("home", "g"),
			key.WithHelp("Home/g", "go to start"),
		),
		End: key.NewBinding(
			key.WithKeys("end", "G"),
			key.WithHelp("End/G", "go to end"),
		),
		PrevHistory: key.NewBinding(
			key.WithKeys("ctrl+r"),
			key.WithHelp("Ctrl+R", "previous command"),
		),
		NextHistory: key.NewBinding(
			key.WithKeys("ctrl+f"),
			key.WithHelp("Ctrl+F", "next command"),
		),
		Clear: key.NewBinding(
			key.WithKeys("ctrl+l"),
			key.WithHelp("Ctrl+L", "clear chat"),
		),
		ToggleViewMode: key.NewBinding(
			key.WithKeys("ctrl+t"),
			key.WithHelp("Ctrl+T", "toggle view mode"),
		),
		CommandPalette: key.NewBinding(
			key.WithKeys("ctrl+k"),
			key.WithHelp("Ctrl+K", "command palette"),
		),
		CancelTool: key.NewBinding(
			key.WithKeys("ctrl+x"),
			key.WithHelp("Ctrl+X", "cancel tool"),
		),
		RetryCost: key.NewBinding(
			key.WithKeys("ctrl+r"),
			key.WithHelp("Ctrl+R", "retry with consent"),
		),
		AcceptCost: key.NewBinding(
			key.WithKeys("ctrl+y"),
			key.WithHelp("Ctrl+Y", "accept cost"),
		),
		SelectNext: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("Tab", "next item"),
		),
		SelectPrev: key.NewBinding(
			key.WithKeys("shift+tab"),
			key.WithHelp("Shift+Tab", "previous item"),
		),
		Copy: key.NewBinding(
			key.WithKeys("ctrl+shift+c", "ctrl+c"),
			key.WithHelp("Ctrl+Shift+C", "copy"),
		),
		Paste: key.NewBinding(
			key.WithKeys("ctrl+shift+v", "ctrl+v"),
			key.WithHelp("Ctrl+Shift+V", "paste"),
		),
		Search: key.NewBinding(
			key.WithKeys("ctrl+/"),
			key.WithHelp("Ctrl+/", "search"),
		),
		NextMatch: key.NewBinding(
			key.WithKeys("n"),
			key.WithHelp("n", "next match"),
		),
		PrevMatch: key.NewBinding(
			key.WithKeys("N"),
			key.WithHelp("N", "previous match"),
		),
		FuzzyFinder: key.NewBinding(
			key.WithKeys("ctrl+o"),
			key.WithHelp("Ctrl+O", "fuzzy file finder"),
		),
		GlobalSearch: key.NewBinding(
			key.WithKeys("ctrl+shift+f"),
			key.WithHelp("Ctrl+Shift+F", "global search"),
		),
		ToggleVimMode: key.NewBinding(
			key.WithKeys("ctrl+alt+v"),
			key.WithHelp("Ctrl+Alt+V", "toggle vim mode"),
		),
	}
}

// FormatKeyBinding formats a key binding for display based on platform
func (ka *KeybindingAdapter) FormatKeyBinding(keys ...string) string {
	var formatted []string
	for _, k := range keys {
		formatted = append(formatted, ka.formatSingleKey(k))
	}
	return strings.Join(formatted, "/")
}

// formatSingleKey formats a single key for display
func (ka *KeybindingAdapter) formatSingleKey(key string) string {
	if ka.platform.IsMacOS() {
		// Handle special keys first
		key = strings.ReplaceAll(key, "esc", "Esc")
		key = strings.ReplaceAll(key, "tab", "Tab")
		key = strings.ReplaceAll(key, "enter", "Enter")
		key = strings.ReplaceAll(key, "space", "Space")
		
		// Replace modifiers with macOS symbols
		key = strings.ReplaceAll(key, "ctrl+", "⌃")
		key = strings.ReplaceAll(key, "alt+", "⌥")
		key = strings.ReplaceAll(key, "cmd+", "⌘")
		key = strings.ReplaceAll(key, "shift+", "⇧")
		
		// Work with runes to handle Unicode properly
		runes := []rune(key)
		lastSymbolRuneIndex := -1
		
		// Find the last modifier symbol
		for i, r := range runes {
			if r == '⌃' || r == '⌥' || r == '⌘' || r == '⇧' {
				lastSymbolRuneIndex = i
			}
		}
		
		if lastSymbolRuneIndex >= 0 && lastSymbolRuneIndex < len(runes)-1 {
			// Capitalize the part after the last symbol
			afterSymbol := string(runes[lastSymbolRuneIndex+1:])
			if len(afterSymbol) == 1 {
				key = string(runes[:lastSymbolRuneIndex+1]) + strings.ToUpper(afterSymbol)
			}
		} else if len(key) == 1 {
			// Single character with no modifiers
			key = strings.ToUpper(key)
		}
	} else {
		// Format for Linux/Windows
		key = strings.ReplaceAll(key, "ctrl+", "Ctrl+")
		key = strings.ReplaceAll(key, "alt+", "Alt+")
		key = strings.ReplaceAll(key, "shift+", "Shift+")
		
		// Handle special keys first (before capitalization)
		key = strings.ReplaceAll(key, "esc", "Esc")
		key = strings.ReplaceAll(key, "tab", "Tab")
		key = strings.ReplaceAll(key, "enter", "Enter")
		key = strings.ReplaceAll(key, "space", "Space")
		
		// Capitalize the key part
		parts := strings.Split(key, "+")
		if len(parts) > 1 {
			// Only capitalize if it's a single letter
			lastPart := parts[len(parts)-1]
			if len(lastPart) == 1 {
				parts[len(parts)-1] = strings.ToUpper(lastPart)
			}
			key = strings.Join(parts, "+")
		} else if len(key) == 1 {
			key = strings.ToUpper(key)
		}
	}
	
	return key
}

// GetPlatformHelpText returns platform-specific help text
func (ka *KeybindingAdapter) GetPlatformHelpText() string {
	platform := ka.platform.String()
	modifier := ka.platform.GetModifierDisplay()
	
	return fmt.Sprintf("Platform: %s | Primary Modifier: %s", platform, modifier)
}