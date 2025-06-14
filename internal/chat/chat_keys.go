package chat

import (
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/lipgloss"
)

// newChatKeyMap creates the default key bindings for the chat interface
func newChatKeyMap() chatKeyMap {
	return chatKeyMap{
		Submit: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "submit message"),
		),
		NewLine: key.NewBinding(
			key.WithKeys("shift+enter"),
			key.WithHelp("shift+enter", "new line"),
		),
		Quit: key.NewBinding(
			key.WithKeys("ctrl+q", "esc", "ctrl+d"),
			key.WithHelp("ctrl+q/esc", "quit"),
		),
		Help: key.NewBinding(
			key.WithKeys("ctrl+h"),
			key.WithHelp("ctrl+h", "toggle help"),
		),
		Prompt: key.NewBinding(
			key.WithKeys("ctrl+p"),
			key.WithHelp("ctrl+p", "prompt management"),
		),
		Status: key.NewBinding(
			key.WithKeys("ctrl+a"),
			key.WithHelp("ctrl+a", "agent status"),
		),
		Global: key.NewBinding(
			key.WithKeys("ctrl+g"),
			key.WithHelp("ctrl+g", "global view"),
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
			key.WithKeys("pgup"),
			key.WithHelp("pgup", "page up"),
		),
		PageDown: key.NewBinding(
			key.WithKeys("pgdown"),
			key.WithHelp("pgdn", "page down"),
		),
		Home: key.NewBinding(
			key.WithKeys("home", "g"),
			key.WithHelp("home/g", "go to start"),
		),
		End: key.NewBinding(
			key.WithKeys("end", "G"),
			key.WithHelp("end/G", "go to end"),
		),
		PrevHistory: key.NewBinding(
			key.WithKeys("ctrl+r"),
			key.WithHelp("ctrl+r", "previous command"),
		),
		NextHistory: key.NewBinding(
			key.WithKeys("ctrl+f"),
			key.WithHelp("ctrl+f", "next command"),
		),
		Clear: key.NewBinding(
			key.WithKeys("ctrl+l"),
			key.WithHelp("ctrl+l", "clear chat"),
		),
		ToggleViewMode: key.NewBinding(
			key.WithKeys("ctrl+t"),
			key.WithHelp("ctrl+t", "toggle view mode"),
		),
		CommandPalette: key.NewBinding(
			key.WithKeys("ctrl+k"),
			key.WithHelp("ctrl+k", "command palette"),
		),
		CancelTool: key.NewBinding(
			key.WithKeys("ctrl+x"),
			key.WithHelp("ctrl+x", "cancel tool"),
		),
		RetryCost: key.NewBinding(
			key.WithKeys("ctrl+r"),
			key.WithHelp("ctrl+r", "retry with consent"),
		),
		AcceptCost: key.NewBinding(
			key.WithKeys("ctrl+y"),
			key.WithHelp("ctrl+y", "accept cost"),
		),
		SelectNext: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("tab", "next item"),
		),
		SelectPrev: key.NewBinding(
			key.WithKeys("shift+tab"),
			key.WithHelp("shift+tab", "previous item"),
		),
		Copy: key.NewBinding(
			key.WithKeys("ctrl+shift+c"),
			key.WithHelp("ctrl+shift+c", "copy"),
		),
		Paste: key.NewBinding(
			key.WithKeys("ctrl+shift+v"),
			key.WithHelp("ctrl+shift+v", "paste"),
		),
		Search: key.NewBinding(
			key.WithKeys("ctrl+/"),
			key.WithHelp("ctrl+/", "search"),
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
			key.WithHelp("ctrl+o", "fuzzy file finder"),
		),
		GlobalSearch: key.NewBinding(
			key.WithKeys("ctrl+shift+f"),
			key.WithHelp("ctrl+shift+f", "global search"),
		),
		ToggleVimMode: key.NewBinding(
			key.WithKeys("ctrl+alt+v"),
			key.WithHelp("ctrl+alt+v", "toggle vim mode"),
		),
	}
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
