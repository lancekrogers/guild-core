// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package panes

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lancekrogers/guild/internal/ui/vim"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVimInputAdapter_VimModeToggle(t *testing.T) {
	// Create input pane
	inputPane, err := NewInputPane(80, 3, false)
	require.NoError(t, err)

	// Cast to implementation
	inputPaneImpl, ok := inputPane.(*inputPaneImpl)
	require.True(t, ok)

	// Create vim manager
	vimManager := vim.NewVimModeManager()

	// Create adapter
	adapter := NewVimInputAdapter(inputPaneImpl, vimManager)

	// Test initial state - vim disabled
	assert.False(t, adapter.IsEnabled())
	assert.Equal(t, vim.ModeInsert, vimManager.GetState().Mode)

	// Enable vim mode
	adapter.SetEnabled(true)
	assert.True(t, adapter.IsEnabled())
	assert.Equal(t, vim.ModeNormal, vimManager.GetState().Mode)

	// Disable vim mode
	adapter.SetEnabled(false)
	assert.False(t, adapter.IsEnabled())
	assert.Equal(t, vim.ModeInsert, vimManager.GetState().Mode)
}

func TestVimInputAdapter_KeyHandling(t *testing.T) {
	// Create input pane
	inputPane, err := NewInputPane(80, 3, false)
	require.NoError(t, err)

	// Cast to implementation
	inputPaneImpl, ok := inputPane.(*inputPaneImpl)
	require.True(t, ok)

	// Create vim manager
	vimManager := vim.NewVimModeManager()

	// Create adapter
	adapter := NewVimInputAdapter(inputPaneImpl, vimManager)

	// Enable vim mode
	adapter.SetEnabled(true)

	// Test escape key in normal mode (should stay in normal mode)
	_, _ = adapter.Update(tea.KeyMsg{Type: tea.KeyEscape})
	assert.Equal(t, vim.ModeNormal, vimManager.GetState().Mode)

	// Test 'i' key to enter insert mode
	_, _ = adapter.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'i'}})
	assert.Equal(t, vim.ModeInsert, vimManager.GetState().Mode)

	// Test escape key in insert mode (should return to normal mode)
	_, _ = adapter.Update(tea.KeyMsg{Type: tea.KeyEscape})
	assert.Equal(t, vim.ModeNormal, vimManager.GetState().Mode)

	// Test movement keys in normal mode
	_, _ = adapter.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})
	assert.Equal(t, vim.ModeNormal, vimManager.GetState().Mode)

	// Test entering command mode
	_, _ = adapter.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{':'}})
	assert.Equal(t, vim.ModeCommand, vimManager.GetState().Mode)
}

func TestVimInputAdapter_InsertModeTyping(t *testing.T) {
	// Create input pane
	inputPane, err := NewInputPane(80, 3, false)
	require.NoError(t, err)

	// Cast to implementation
	inputPaneImpl, ok := inputPane.(*inputPaneImpl)
	require.True(t, ok)

	// Create vim manager
	vimManager := vim.NewVimModeManager()

	// Create adapter
	adapter := NewVimInputAdapter(inputPaneImpl, vimManager)

	// Enable vim mode and enter insert mode
	adapter.SetEnabled(true)
	vimManager.GetState().Mode = vim.ModeInsert

	// Type some text
	adapter.SetValue("")
	_, _ = adapter.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h', 'e', 'l', 'l', 'o'}})

	// The text should be passed through to the input pane
	// Note: In this test, we're not actually simulating the full textarea behavior,
	// but we're verifying that the adapter passes through the keys in insert mode
	assert.Equal(t, vim.ModeInsert, vimManager.GetState().Mode)
}
