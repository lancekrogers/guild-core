package chat

import (
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/lipgloss"
)

// newChatKeyMap creates the default key bindings for the chat interface
// This function is deprecated in favor of the KeybindingAdapter
// but kept for backwards compatibility
func newChatKeyMap() chatKeyMap {
	adapter := NewKeybindingAdapter()
	return adapter.GetChatKeyMap()
}

// Short returns a list of key bindings for the short help view
func (k chatKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{
		k.Submit,
		k.NewLine,
		k.Quit,
		k.Help,
		k.Prompt,
		k.Status,
	}
}

// FullHelp returns a list of key bindings for the full help view
func (k chatKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Submit, k.NewLine, k.Quit, k.Help, k.Clear},
		{k.Prompt, k.Status, k.Global, k.CommandPalette},
		{k.ScrollUp, k.ScrollDown, k.PageUp, k.PageDown},
		{k.Home, k.End, k.PrevHistory, k.NextHistory},
		{k.Search, k.NextMatch, k.PrevMatch, k.ToggleViewMode},
		{k.Copy, k.Paste, k.FuzzyFinder, k.GlobalSearch},
		{k.ToggleVimMode},
	}
}

// Help styles for consistent appearance
var (
	helpTitleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("141")).
			Bold(true).
			MarginBottom(1)

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))

	helpKeyStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("141")).
			Bold(true)

	helpDescStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("244"))
)
