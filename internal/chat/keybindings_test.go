package chat

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestKeybindingAdapterCreation(t *testing.T) {
	// Test default creation
	adapter := NewKeybindingAdapter()
	assert.NotNil(t, adapter)
	assert.NotEqual(t, PlatformUnknown, adapter.platform)

	// Test platform-specific creation
	for _, platform := range []Platform{PlatformMacOS, PlatformLinux, PlatformWindows} {
		adapter := NewKeybindingAdapterForPlatform(platform)
		assert.NotNil(t, adapter)
		assert.Equal(t, platform, adapter.platform)
	}
}

func TestGetChatKeyMap(t *testing.T) {
	// Test that each platform returns a valid keymap
	for _, platform := range []Platform{PlatformMacOS, PlatformLinux, PlatformWindows} {
		t.Run(platform.String(), func(t *testing.T) {
			adapter := NewKeybindingAdapterForPlatform(platform)
			keyMap := adapter.GetChatKeyMap()

			// Verify all essential keybindings are present
			assert.NotNil(t, keyMap.Submit)
			assert.NotNil(t, keyMap.NewLine)
			assert.NotNil(t, keyMap.Quit)
			assert.NotNil(t, keyMap.Help)
			assert.NotNil(t, keyMap.Prompt)
			assert.NotNil(t, keyMap.Status)
			assert.NotNil(t, keyMap.Global)
			assert.NotNil(t, keyMap.ScrollUp)
			assert.NotNil(t, keyMap.ScrollDown)
			assert.NotNil(t, keyMap.Copy)
			assert.NotNil(t, keyMap.Paste)
			assert.NotNil(t, keyMap.Search)
			assert.NotNil(t, keyMap.CommandPalette)
			assert.NotNil(t, keyMap.ToggleVimMode)

			// Verify each keybinding has keys and help text
			assert.NotEmpty(t, keyMap.Submit.Keys())
			assert.NotEmpty(t, keyMap.Submit.Help().Key)
			assert.NotEmpty(t, keyMap.Submit.Help().Desc)
		})
	}
}

func TestPlatformSpecificKeybindings(t *testing.T) {
	t.Run("macOS uses Alt as primary modifier", func(t *testing.T) {
		adapter := NewKeybindingAdapterForPlatform(PlatformMacOS)
		keyMap := adapter.GetChatKeyMap()

		// Check primary commands use Alt
		assert.Contains(t, keyMap.Quit.Keys(), "alt+q")
		assert.Contains(t, keyMap.Help.Keys(), "alt+h")
		assert.Contains(t, keyMap.Prompt.Keys(), "alt+p")
		assert.Contains(t, keyMap.Status.Keys(), "alt+a")
		assert.Contains(t, keyMap.Clear.Keys(), "alt+l")
		assert.Contains(t, keyMap.Copy.Keys(), "alt+c")
		assert.Contains(t, keyMap.Paste.Keys(), "alt+v")

		// NewLine should use Alt+Enter
		assert.Contains(t, keyMap.NewLine.Keys(), "alt+enter")
	})

	t.Run("Linux uses Ctrl as primary modifier", func(t *testing.T) {
		adapter := NewKeybindingAdapterForPlatform(PlatformLinux)
		keyMap := adapter.GetChatKeyMap()

		// Check primary commands use Ctrl
		assert.Contains(t, keyMap.Quit.Keys(), "ctrl+q")
		assert.Contains(t, keyMap.Help.Keys(), "ctrl+h")
		assert.Contains(t, keyMap.Prompt.Keys(), "ctrl+p")
		assert.Contains(t, keyMap.Status.Keys(), "ctrl+a")
		assert.Contains(t, keyMap.Clear.Keys(), "ctrl+l")

		// Copy/Paste should support both standard and terminal variants
		assert.Contains(t, keyMap.Copy.Keys(), "ctrl+shift+c")
		assert.Contains(t, keyMap.Paste.Keys(), "ctrl+shift+v")

		// NewLine should support multiple options
		keys := keyMap.NewLine.Keys()
		assert.True(t, containsKey(keys, "shift+enter") || containsKey(keys, "ctrl+enter"))
	})

	t.Run("Windows uses Ctrl as primary modifier", func(t *testing.T) {
		adapter := NewKeybindingAdapterForPlatform(PlatformWindows)
		keyMap := adapter.GetChatKeyMap()

		// Should be similar to Linux
		assert.Contains(t, keyMap.Quit.Keys(), "ctrl+q")
		assert.Contains(t, keyMap.Help.Keys(), "ctrl+h")
	})
}

func TestUniversalKeybindings(t *testing.T) {
	// Test that some keybindings remain the same across platforms
	for _, platform := range []Platform{PlatformMacOS, PlatformLinux, PlatformWindows} {
		t.Run(platform.String(), func(t *testing.T) {
			adapter := NewKeybindingAdapterForPlatform(platform)
			keyMap := adapter.GetChatKeyMap()

			// These should be universal
			assert.Contains(t, keyMap.Submit.Keys(), "enter")
			assert.Contains(t, keyMap.Quit.Keys(), "esc")
			assert.Contains(t, keyMap.SelectNext.Keys(), "tab")
			assert.Contains(t, keyMap.SelectPrev.Keys(), "shift+tab")
			assert.Contains(t, keyMap.ScrollUp.Keys(), "up")
			assert.Contains(t, keyMap.ScrollDown.Keys(), "down")
			assert.Contains(t, keyMap.NextMatch.Keys(), "n")
			assert.Contains(t, keyMap.PrevMatch.Keys(), "N")
		})
	}
}

func TestHelpTextFormatting(t *testing.T) {
	t.Run("macOS uses symbols in help text", func(t *testing.T) {
		adapter := NewKeybindingAdapterForPlatform(PlatformMacOS)
		keyMap := adapter.GetChatKeyMap()

		// Check for macOS symbols in help text
		assert.Contains(t, keyMap.Quit.Help().Key, "⌥")
		assert.Contains(t, keyMap.NewLine.Help().Key, "⌥")
		assert.Contains(t, keyMap.Copy.Help().Key, "⌥")
		assert.Contains(t, keyMap.GlobalSearch.Help().Key, "⇧") // Shift symbol
	})

	t.Run("Linux/Windows use text in help", func(t *testing.T) {
		for _, platform := range []Platform{PlatformLinux, PlatformWindows} {
			adapter := NewKeybindingAdapterForPlatform(platform)
			keyMap := adapter.GetChatKeyMap()

			// Should use text, not symbols
			assert.Contains(t, keyMap.Quit.Help().Key, "Ctrl")
			assert.NotContains(t, keyMap.Quit.Help().Key, "⌥")
			assert.NotContains(t, keyMap.Quit.Help().Key, "⌘")
		}
	})
}

func TestFormatKeyBinding(t *testing.T) {
	tests := []struct {
		name     string
		platform Platform
		input    []string
		expected string
	}{
		// macOS formatting
		{"macOS ctrl", PlatformMacOS, []string{"ctrl+a"}, "⌃A"},
		{"macOS alt", PlatformMacOS, []string{"alt+q"}, "⌥Q"},
		{"macOS cmd", PlatformMacOS, []string{"cmd+c"}, "⌘C"},
		{"macOS shift", PlatformMacOS, []string{"shift+tab"}, "⇧Tab"},
		{"macOS multiple", PlatformMacOS, []string{"alt+q", "esc"}, "⌥Q/Esc"},
		{"macOS complex", PlatformMacOS, []string{"alt+shift+f"}, "⌥⇧F"},

		// Linux formatting
		{"Linux ctrl", PlatformLinux, []string{"ctrl+a"}, "Ctrl+A"},
		{"Linux alt", PlatformLinux, []string{"alt+q"}, "Alt+Q"},
		{"Linux shift", PlatformLinux, []string{"shift+tab"}, "Shift+Tab"},
		{"Linux multiple", PlatformLinux, []string{"ctrl+q", "esc"}, "Ctrl+Q/Esc"},

		// Windows formatting (same as Linux)
		{"Windows ctrl", PlatformWindows, []string{"ctrl+a"}, "Ctrl+A"},
		{"Windows multiple", PlatformWindows, []string{"ctrl+q", "esc"}, "Ctrl+Q/Esc"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adapter := NewKeybindingAdapterForPlatform(tt.platform)
			result := adapter.FormatKeyBinding(tt.input...)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestBackwardsCompatibility(t *testing.T) {
	// Test that the deprecated newChatKeyMap still works
	keyMap := newChatKeyMap()
	
	// Should return a valid keymap
	assert.NotNil(t, keyMap)
	assert.NotNil(t, keyMap.Submit)
	assert.NotNil(t, keyMap.Quit)
	assert.NotEmpty(t, keyMap.Submit.Keys())
	
	// Should use platform-specific bindings
	platform := DetectPlatform()
	if platform.IsMacOS() {
		assert.Contains(t, keyMap.Quit.Keys(), "alt+q")
	} else {
		assert.Contains(t, keyMap.Quit.Keys(), "ctrl+q")
	}
}

// Helper function
func containsKey(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}