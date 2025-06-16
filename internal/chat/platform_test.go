// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package chat

import (
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/guild-ventures/guild-core/pkg/config"
)

func TestPlatformDetection(t *testing.T) {
	platform := DetectPlatform()

	// Verify we get a valid platform
	assert.NotEqual(t, PlatformUnknown, platform, "Platform should be detected")

	// Verify platform matches runtime.GOOS
	switch runtime.GOOS {
	case "darwin":
		assert.Equal(t, PlatformMacOS, platform)
		assert.True(t, platform.IsMacOS())
		assert.False(t, platform.IsLinux())
		assert.False(t, platform.IsWindows())
	case "linux":
		assert.Equal(t, PlatformLinux, platform)
		assert.False(t, platform.IsMacOS())
		assert.True(t, platform.IsLinux())
		assert.False(t, platform.IsWindows())
	case "windows":
		assert.Equal(t, PlatformWindows, platform)
		assert.False(t, platform.IsMacOS())
		assert.False(t, platform.IsLinux())
		assert.True(t, platform.IsWindows())
	}
}

func TestPlatformModifierKeys(t *testing.T) {
	tests := []struct {
		platform           Platform
		expectedPrimary    string
		expectedDisplay    string
		expectedSecondary  string
		expectedSecDisplay string
	}{
		{
			platform:           PlatformMacOS,
			expectedPrimary:    "alt",
			expectedDisplay:    "⌥",
			expectedSecondary:  "cmd",
			expectedSecDisplay: "⌘",
		},
		{
			platform:           PlatformLinux,
			expectedPrimary:    "ctrl",
			expectedDisplay:    "Ctrl",
			expectedSecondary:  "alt",
			expectedSecDisplay: "Alt",
		},
		{
			platform:           PlatformWindows,
			expectedPrimary:    "ctrl",
			expectedDisplay:    "Ctrl",
			expectedSecondary:  "alt",
			expectedSecDisplay: "Alt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.platform.String(), func(t *testing.T) {
			assert.Equal(t, tt.expectedPrimary, tt.platform.GetModifierKey())
			assert.Equal(t, tt.expectedDisplay, tt.platform.GetModifierDisplay())
			assert.Equal(t, tt.expectedSecondary, tt.platform.GetSecondaryModifierKey())
			assert.Equal(t, tt.expectedSecDisplay, tt.platform.GetSecondaryModifierDisplay())
		})
	}
}

func TestKeybindingAdapter(t *testing.T) {
	// Test macOS keybindings
	t.Run("macOS keybindings", func(t *testing.T) {
		adapter := NewKeybindingAdapterForPlatform(PlatformMacOS)
		keyMap := adapter.GetChatKeyMap()

		// Check that Alt/Option is used as primary modifier
		assert.Contains(t, keyMap.Quit.Keys(), "alt+q")
		assert.Contains(t, keyMap.Help.Keys(), "alt+h")
		assert.Contains(t, keyMap.Prompt.Keys(), "alt+p")
		assert.Contains(t, keyMap.Copy.Keys(), "alt+c")
		assert.Contains(t, keyMap.Paste.Keys(), "alt+v")

		// Check help text uses macOS symbols
		assert.Contains(t, keyMap.Quit.Help().Key, "⌥")
		assert.Contains(t, keyMap.NewLine.Help().Key, "⌥")
	})

	// Test Linux/Windows keybindings
	t.Run("Linux keybindings", func(t *testing.T) {
		adapter := NewKeybindingAdapterForPlatform(PlatformLinux)
		keyMap := adapter.GetChatKeyMap()

		// Check that Ctrl is used as primary modifier
		assert.Contains(t, keyMap.Quit.Keys(), "ctrl+q")
		assert.Contains(t, keyMap.Help.Keys(), "ctrl+h")
		assert.Contains(t, keyMap.Prompt.Keys(), "ctrl+p")

		// Check help text uses standard notation
		assert.Contains(t, keyMap.Quit.Help().Key, "Ctrl")
		assert.NotContains(t, keyMap.Help.Help().Key, "⌥")
	})
}

func TestKeybindingFormatting(t *testing.T) {
	tests := []struct {
		name     string
		platform Platform
		input    []string
		expected string
	}{
		{
			name:     "macOS single key",
			platform: PlatformMacOS,
			input:    []string{"alt+q"},
			expected: "⌥Q",
		},
		{
			name:     "macOS multiple keys",
			platform: PlatformMacOS,
			input:    []string{"alt+q", "esc"},
			expected: "⌥Q/Esc",
		},
		{
			name:     "Linux single key",
			platform: PlatformLinux,
			input:    []string{"ctrl+q"},
			expected: "Ctrl+Q",
		},
		{
			name:     "Linux multiple keys",
			platform: PlatformLinux,
			input:    []string{"ctrl+q", "esc"},
			expected: "Ctrl+Q/Esc",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adapter := NewKeybindingAdapterForPlatform(tt.platform)
			result := adapter.FormatKeyBinding(tt.input...)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestPlatformHelpText(t *testing.T) {
	adapter := NewKeybindingAdapter()
	helpText := adapter.GetPlatformHelpText()

	// Should contain platform name
	assert.Contains(t, helpText, "Platform:")

	// Should contain modifier key info
	assert.Contains(t, helpText, "Primary Modifier:")

	// Should not be empty
	assert.NotEmpty(t, helpText)
}

func TestChatModelWithPlatformKeybindings(t *testing.T) {
	// Create a minimal guild config for testing
	cfg := &config.GuildConfig{
		Name:        "test-guild",
		Description: "Test guild for platform keybindings",
		Manager: config.ManagerConfig{
			Default: "test-manager",
		},
		Agents: []config.AgentConfig{},
	}

	// Create chat model
	model := NewChatModel(cfg, nil, nil, "test-campaign")

	// Verify keybinding adapter is initialized
	require.NotNil(t, model.keyAdapter)

	// Verify keys are set based on platform
	platform := DetectPlatform()
	if platform.IsMacOS() {
		assert.Contains(t, model.keys.Quit.Keys(), "alt+q")
	} else {
		assert.Contains(t, model.keys.Quit.Keys(), "ctrl+q")
	}

	// Test help text includes platform info
	helpText := model.getHelpText()
	assert.Contains(t, helpText, "Platform:")
}

func TestNewLineKeybinding(t *testing.T) {
	// Test that newline keybinding works on all platforms
	tests := []struct {
		platform Platform
		expected []string
	}{
		{
			platform: PlatformMacOS,
			expected: []string{"alt+enter"},
		},
		{
			platform: PlatformLinux,
			expected: []string{"shift+enter", "ctrl+enter"},
		},
		{
			platform: PlatformWindows,
			expected: []string{"shift+enter", "ctrl+enter"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.platform.String(), func(t *testing.T) {
			adapter := NewKeybindingAdapterForPlatform(tt.platform)
			keyMap := adapter.GetChatKeyMap()

			keys := keyMap.NewLine.Keys()
			for _, expectedKey := range tt.expected {
				assert.Contains(t, keys, expectedKey, "NewLine should contain %s on %s", expectedKey, tt.platform)
			}
		})
	}
}
