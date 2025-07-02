// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package utils

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/key"
)

// KeyBindings defines all keyboard shortcuts for Guild Chat
type KeyBindings struct {
	// Global shortcuts
	Quit           key.Binding
	Help           key.Binding
	CommandPalette key.Binding
	GlobalSearch   key.Binding

	// Navigation
	Submit     key.Binding
	NewLine    key.Binding
	ScrollUp   key.Binding
	ScrollDown key.Binding
	PageUp     key.Binding
	PageDown   key.Binding
	Home       key.Binding
	End        key.Binding

	// History and completion
	PrevHistory      key.Binding
	NextHistory      key.Binding
	TabComplete      key.Binding
	AcceptCompletion key.Binding
	CancelCompletion key.Binding

	// Search
	Search      key.Binding
	NextMatch   key.Binding
	PrevMatch   key.Binding
	ClearSearch key.Binding

	// View modes
	ToggleViewMode key.Binding
	FocusOutput    key.Binding
	FocusInput     key.Binding
	FocusStatus    key.Binding

	// Agent interaction
	AgentsList key.Binding
	PromptView key.Binding
	StatusView key.Binding

	// Tools and commands
	ToolsList   key.Binding
	ExecuteTool key.Binding
	CancelTool  key.Binding

	// Text editing
	Clear     key.Binding
	Copy      key.Binding
	Paste     key.Binding
	SelectAll key.Binding

	// Special features
	ToggleVimMode key.Binding
	Export        key.Binding
	ImportSession key.Binding

	// Developer shortcuts
	DebugInfo       key.Binding
	RefreshAgents   key.Binding
	ReconnectDaemon key.Binding

	// Chord shortcuts
	ChordPrefix    key.Binding
	MentionElena   key.Binding
	MentionMarcus  key.Binding
	MentionVera    key.Binding
	MentionAll     key.Binding
	
	// View shortcuts
	ViewChat   key.Binding
	ViewKanban key.Binding
	ViewCorpus key.Binding
	
	// Quick actions
	SaveSession  key.Binding
	UndoMessage  key.Binding
}

// NewKeyBindings creates the default key bindings for Guild Chat
func NewKeyBindings() *KeyBindings {
	return &KeyBindings{
		// Global shortcuts
		Quit: key.NewBinding(
			key.WithKeys("ctrl+c", "ctrl+q"),
			key.WithHelp("ctrl+c/ctrl+q", "quit"),
		),
		Help: key.NewBinding(
			key.WithKeys("ctrl+h", "f1"),
			key.WithHelp("ctrl+h/f1", "help"),
		),
		CommandPalette: key.NewBinding(
			key.WithKeys("ctrl+p"),
			key.WithHelp("ctrl+p", "command palette"),
		),
		GlobalSearch: key.NewBinding(
			key.WithKeys("ctrl+shift+f"),
			key.WithHelp("ctrl+shift+f", "global search"),
		),

		// Navigation
		Submit: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "send message"),
		),
		NewLine: key.NewBinding(
			key.WithKeys("shift+enter"),
			key.WithHelp("shift+enter", "new line"),
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
			key.WithHelp("pgup/ctrl+u", "page up"),
		),
		PageDown: key.NewBinding(
			key.WithKeys("pgdown", "ctrl+d"),
			key.WithHelp("pgdown/ctrl+d", "page down"),
		),
		Home: key.NewBinding(
			key.WithKeys("home", "g"),
			key.WithHelp("home/g", "go to top"),
		),
		End: key.NewBinding(
			key.WithKeys("end", "G"),
			key.WithHelp("end/G", "go to bottom"),
		),

		// History and completion
		PrevHistory: key.NewBinding(
			key.WithKeys("up"),
			key.WithHelp("↑", "previous command"),
		),
		NextHistory: key.NewBinding(
			key.WithKeys("down"),
			key.WithHelp("↓", "next command"),
		),
		TabComplete: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("tab", "auto-complete"),
		),
		AcceptCompletion: key.NewBinding(
			key.WithKeys("enter", "tab"),
			key.WithHelp("enter/tab", "accept completion"),
		),
		CancelCompletion: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "cancel completion"),
		),

		// Search
		Search: key.NewBinding(
			key.WithKeys("ctrl+r", "/"),
			key.WithHelp("ctrl+r", "search history"),
		),
		NextMatch: key.NewBinding(
			key.WithKeys("n", "ctrl+n"),
			key.WithHelp("n", "next match"),
		),
		PrevMatch: key.NewBinding(
			key.WithKeys("N", "ctrl+shift+n"),
			key.WithHelp("N", "previous match"),
		),
		ClearSearch: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "clear search"),
		),

		// View modes
		ToggleViewMode: key.NewBinding(
			key.WithKeys("ctrl+t"),
			key.WithHelp("ctrl+t", "toggle view mode"),
		),
		FocusOutput: key.NewBinding(
			key.WithKeys("ctrl+1"),
			key.WithHelp("ctrl+1", "focus output"),
		),
		FocusInput: key.NewBinding(
			key.WithKeys("ctrl+2"),
			key.WithHelp("ctrl+2", "focus input"),
		),
		FocusStatus: key.NewBinding(
			key.WithKeys("ctrl+3"),
			key.WithHelp("ctrl+3", "focus status"),
		),

		// Agent interaction
		AgentsList: key.NewBinding(
			key.WithKeys("ctrl+a"),
			key.WithHelp("ctrl+a", "list agents"),
		),
		PromptView: key.NewBinding(
			key.WithKeys("ctrl+shift+p"),
			key.WithHelp("ctrl+shift+p", "view prompts"),
		),
		StatusView: key.NewBinding(
			key.WithKeys("ctrl+s"),
			key.WithHelp("ctrl+s", "view status"),
		),

		// Tools and commands
		ToolsList: key.NewBinding(
			key.WithKeys("ctrl+shift+t"),
			key.WithHelp("ctrl+shift+t", "list tools"),
		),
		ExecuteTool: key.NewBinding(
			key.WithKeys("ctrl+e"),
			key.WithHelp("ctrl+e", "execute tool"),
		),
		CancelTool: key.NewBinding(
			key.WithKeys("ctrl+x"),
			key.WithHelp("ctrl+x", "cancel tool"),
		),

		// Text editing
		Clear: key.NewBinding(
			key.WithKeys("ctrl+l"),
			key.WithHelp("ctrl+l", "clear input"),
		),
		Copy: key.NewBinding(
			key.WithKeys("ctrl+c"),
			key.WithHelp("ctrl+c", "copy"),
		),
		Paste: key.NewBinding(
			key.WithKeys("ctrl+v"),
			key.WithHelp("ctrl+v", "paste"),
		),
		SelectAll: key.NewBinding(
			key.WithKeys("ctrl+a"),
			key.WithHelp("ctrl+a", "select all"),
		),

		// Special features
		ToggleVimMode: key.NewBinding(
			key.WithKeys("ctrl+shift+v"),
			key.WithHelp("ctrl+shift+v", "toggle vim mode"),
		),
		Export: key.NewBinding(
			key.WithKeys("ctrl+shift+e"),
			key.WithHelp("ctrl+shift+e", "export chat"),
		),
		ImportSession: key.NewBinding(
			key.WithKeys("ctrl+shift+i"),
			key.WithHelp("ctrl+shift+i", "import session"),
		),

		// Developer shortcuts
		DebugInfo: key.NewBinding(
			key.WithKeys("ctrl+shift+d"),
			key.WithHelp("ctrl+shift+d", "debug info"),
		),
		RefreshAgents: key.NewBinding(
			key.WithKeys("ctrl+shift+r"),
			key.WithHelp("ctrl+shift+r", "refresh agents"),
		),
		ReconnectDaemon: key.NewBinding(
			key.WithKeys("ctrl+shift+c"),
			key.WithHelp("ctrl+shift+c", "reconnect daemon"),
		),

		// Chord shortcuts
		ChordPrefix: key.NewBinding(
			key.WithKeys("@"),
			key.WithHelp("@", "chord prefix"),
		),
		MentionElena: key.NewBinding(
			key.WithKeys("e"),
			key.WithHelp("@e", "mention Elena"),
		),
		MentionMarcus: key.NewBinding(
			key.WithKeys("m"),
			key.WithHelp("@m", "mention Marcus"),
		),
		MentionVera: key.NewBinding(
			key.WithKeys("v"),
			key.WithHelp("@v", "mention Vera"),
		),
		MentionAll: key.NewBinding(
			key.WithKeys("a"),
			key.WithHelp("@a", "mention all agents"),
		),
		
		// View shortcuts
		ViewChat: key.NewBinding(
			key.WithKeys("ctrl+1"),
			key.WithHelp("ctrl+1", "chat view"),
		),
		ViewKanban: key.NewBinding(
			key.WithKeys("ctrl+2"),
			key.WithHelp("ctrl+2", "kanban view"),
		),
		ViewCorpus: key.NewBinding(
			key.WithKeys("ctrl+3"),
			key.WithHelp("ctrl+3", "corpus view"),
		),
		
		// Quick actions
		SaveSession: key.NewBinding(
			key.WithKeys("ctrl+s"),
			key.WithHelp("ctrl+s", "save session"),
		),
		UndoMessage: key.NewBinding(
			key.WithKeys("ctrl+z"),
			key.WithHelp("ctrl+z", "undo last message"),
		),
	}
}

// VimKeyBindings creates Vim-style key bindings
func NewVimKeyBindings() *KeyBindings {
	keys := NewKeyBindings()

	// Override with Vim-style bindings
	keys.ScrollUp = key.NewBinding(
		key.WithKeys("k"),
		key.WithHelp("k", "scroll up"),
	)
	keys.ScrollDown = key.NewBinding(
		key.WithKeys("j"),
		key.WithHelp("j", "scroll down"),
	)
	keys.Home = key.NewBinding(
		key.WithKeys("gg"),
		key.WithHelp("gg", "go to top"),
	)
	keys.End = key.NewBinding(
		key.WithKeys("G"),
		key.WithHelp("G", "go to bottom"),
	)
	keys.PageUp = key.NewBinding(
		key.WithKeys("ctrl+u"),
		key.WithHelp("ctrl+u", "page up"),
	)
	keys.PageDown = key.NewBinding(
		key.WithKeys("ctrl+d"),
		key.WithHelp("ctrl+d", "page down"),
	)
	keys.Search = key.NewBinding(
		key.WithKeys("/"),
		key.WithHelp("/", "search"),
	)
	keys.NextMatch = key.NewBinding(
		key.WithKeys("n"),
		key.WithHelp("n", "next match"),
	)
	keys.PrevMatch = key.NewBinding(
		key.WithKeys("N"),
		key.WithHelp("N", "previous match"),
	)

	return keys
}

// FullHelp returns all key bindings for the help view
func (k KeyBindings) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		// Navigation
		{k.Submit, k.NewLine, k.ScrollUp, k.ScrollDown},
		{k.PageUp, k.PageDown, k.Home, k.End},

		// History and completion
		{k.PrevHistory, k.NextHistory, k.TabComplete},
		{k.AcceptCompletion, k.CancelCompletion},

		// Search
		{k.Search, k.NextMatch, k.PrevMatch, k.ClearSearch},

		// View modes
		{k.ToggleViewMode, k.FocusOutput, k.FocusInput, k.FocusStatus},

		// Agent interaction
		{k.AgentsList, k.PromptView, k.StatusView},

		// Tools and commands
		{k.ToolsList, k.ExecuteTool, k.CancelTool},

		// Text editing
		{k.Clear, k.Copy, k.Paste, k.SelectAll},

		// Global shortcuts
		{k.CommandPalette, k.GlobalSearch, k.Help, k.Quit},

		// Special features
		{k.ToggleVimMode, k.Export, k.ImportSession},

		// Developer shortcuts
		{k.DebugInfo, k.RefreshAgents, k.ReconnectDaemon},
	}
}

// ShortHelp returns key bindings to be shown in the mini help view
func (k KeyBindings) ShortHelp() []key.Binding {
	return []key.Binding{
		k.Submit,
		k.TabComplete,
		k.CommandPalette,
		k.Search,
		k.Help,
		k.Quit,
	}
}

// GetContextualHelp returns help for a specific context
func (k KeyBindings) GetContextualHelp(context string) [][]key.Binding {
	switch context {
	case "input":
		return [][]key.Binding{
			{k.Submit, k.NewLine, k.TabComplete},
			{k.PrevHistory, k.NextHistory, k.Clear},
			{k.Copy, k.Paste, k.SelectAll},
		}
	case "output":
		return [][]key.Binding{
			{k.ScrollUp, k.ScrollDown, k.PageUp, k.PageDown},
			{k.Home, k.End, k.Search},
			{k.NextMatch, k.PrevMatch, k.ClearSearch},
		}
	case "completion":
		return [][]key.Binding{
			{k.AcceptCompletion, k.CancelCompletion},
			{k.TabComplete},
		}
	case "search":
		return [][]key.Binding{
			{k.NextMatch, k.PrevMatch, k.ClearSearch},
		}
	default:
		return k.FullHelp()
	}
}

// IsQuitKey checks if a key event is a quit command
func (k KeyBindings) IsQuitKey(keyStr string) bool {
	// Simplified key matching - check against known quit keys
	quitKeys := []string{"ctrl+c", "ctrl+q"}
	for _, quitKey := range quitKeys {
		if keyStr == quitKey {
			return true
		}
	}
	return false
}

// IsSubmitKey checks if a key event is a submit command
func (k KeyBindings) IsSubmitKey(keyStr string) bool {
	return keyStr == "enter"
}

// IsHelpKey checks if a key event is a help command
func (k KeyBindings) IsHelpKey(keyStr string) bool {
	return keyStr == "ctrl+h" || keyStr == "f1"
}

// KeyContext represents different contexts where keys might behave differently
type KeyContext int

const (
	ContextGlobal KeyContext = iota
	ContextInput
	ContextOutput
	ContextCompletion
	ContextSearch
	ContextCommandPalette
	ContextHelp
)

// GetKeysForContext returns relevant key bindings for a specific context
func (k KeyBindings) GetKeysForContext(context KeyContext) map[string]key.Binding {
	keys := make(map[string]key.Binding)

	// Global keys are always available
	keys["quit"] = k.Quit
	keys["help"] = k.Help
	keys["command_palette"] = k.CommandPalette

	switch context {
	case ContextInput:
		keys["submit"] = k.Submit
		keys["new_line"] = k.NewLine
		keys["tab_complete"] = k.TabComplete
		keys["prev_history"] = k.PrevHistory
		keys["next_history"] = k.NextHistory
		keys["clear"] = k.Clear
		keys["copy"] = k.Copy
		keys["paste"] = k.Paste

	case ContextOutput:
		keys["scroll_up"] = k.ScrollUp
		keys["scroll_down"] = k.ScrollDown
		keys["page_up"] = k.PageUp
		keys["page_down"] = k.PageDown
		keys["home"] = k.Home
		keys["end"] = k.End
		keys["search"] = k.Search
		keys["next_match"] = k.NextMatch
		keys["prev_match"] = k.PrevMatch

	case ContextCompletion:
		keys["accept_completion"] = k.AcceptCompletion
		keys["cancel_completion"] = k.CancelCompletion
		keys["tab_complete"] = k.TabComplete

	case ContextSearch:
		keys["next_match"] = k.NextMatch
		keys["prev_match"] = k.PrevMatch
		keys["clear_search"] = k.ClearSearch

	case ContextCommandPalette:
		keys["submit"] = k.Submit
		keys["cancel_completion"] = k.CancelCompletion
		keys["scroll_up"] = k.ScrollUp
		keys["scroll_down"] = k.ScrollDown
	}

	return keys
}

// FormatKeyBinding formats a key binding for display
func FormatKeyBinding(binding key.Binding) string {
	help := binding.Help()
	if help.Key == "" {
		return ""
	}

	return fmt.Sprintf("%-15s %s", help.Key, help.Desc)
}

// FormatKeyBindings formats multiple key bindings for display
func FormatKeyBindings(bindings []key.Binding) []string {
	var formatted []string
	for _, binding := range bindings {
		if formattedBinding := FormatKeyBinding(binding); formattedBinding != "" {
			formatted = append(formatted, formattedBinding)
		}
	}
	return formatted
}

// CreateHelpText creates formatted help text for key bindings
func (k KeyBindings) CreateHelpText(context string) string {
	bindings := k.GetContextualHelp(context)

	var sections []string
	sectionTitles := []string{
		"Navigation",
		"History & Completion",
		"Search",
		"View Modes",
		"Agent Interaction",
		"Tools & Commands",
		"Text Editing",
		"Global Shortcuts",
		"Special Features",
		"Developer",
	}

	for i, section := range bindings {
		if i < len(sectionTitles) {
			title := sectionTitles[i]
			var lines []string
			lines = append(lines, title+":")
			lines = append(lines, strings.Repeat("-", len(title)+1))

			for _, binding := range section {
				if formatted := FormatKeyBinding(binding); formatted != "" {
					lines = append(lines, "  "+formatted)
				}
			}

			sections = append(sections, strings.Join(lines, "\n"))
		}
	}

	return strings.Join(sections, "\n\n")
}

// ChordManager handles multi-key chord sequences
type ChordManager struct {
	activeChord   string
	chordTimeout  time.Duration
	lastKeyTime   time.Time
	chordBindings map[string]func() tea.Cmd
}

// NewChordManager creates a new chord manager
func NewChordManager() *ChordManager {
	cm := &ChordManager{
		chordTimeout:  2 * time.Second,
		chordBindings: make(map[string]func() tea.Cmd),
	}
	
	// Register default chord bindings
	cm.RegisterChordBindings()
	
	return cm
}

// RegisterChordBindings registers all chord sequences
func (cm *ChordManager) RegisterChordBindings() {
	// Agent mention chords
	cm.chordBindings["@e"] = func() tea.Cmd {
		return func() tea.Msg {
			return InsertTextMsg{Text: "@elena "}
		}
	}
	cm.chordBindings["@m"] = func() tea.Cmd {
		return func() tea.Msg {
			return InsertTextMsg{Text: "@marcus "}
		}
	}
	cm.chordBindings["@v"] = func() tea.Cmd {
		return func() tea.Msg {
			return InsertTextMsg{Text: "@vera "}
		}
	}
	cm.chordBindings["@a"] = func() tea.Cmd {
		return func() tea.Msg {
			return InsertTextMsg{Text: "@all "}
		}
	}
	
	// Window management chords
	cm.chordBindings["gw"] = func() tea.Cmd {
		return func() tea.Msg {
			return SwitchWindowMsg{Window: "next"}
		}
	}
	cm.chordBindings["gW"] = func() tea.Cmd {
		return func() tea.Msg {
			return SwitchWindowMsg{Window: "prev"}
		}
	}
}

// HandleKey processes a key press for chord sequences
func (cm *ChordManager) HandleKey(key string) (tea.Cmd, bool) {
	now := time.Now()
	
	// Check if chord has timed out
	if cm.activeChord != "" && now.Sub(cm.lastKeyTime) > cm.chordTimeout {
		cm.activeChord = ""
	}
	
	// Build potential chord
	potentialChord := cm.activeChord + key
	
	// Check if this completes a chord
	if action, exists := cm.chordBindings[potentialChord]; exists {
		cm.activeChord = ""
		return action(), true
	}
	
	// Check if this could be the start of a chord
	for chord := range cm.chordBindings {
		if strings.HasPrefix(chord, potentialChord) {
			cm.activeChord = potentialChord
			cm.lastKeyTime = now
			return ShowChordPromptCmd(potentialChord), true
		}
	}
	
	// Not a chord
	cm.activeChord = ""
	return nil, false
}

// IsChordActive returns true if a chord sequence is in progress
func (cm *ChordManager) IsChordActive() bool {
	return cm.activeChord != ""
}

// GetActiveChord returns the current chord sequence
func (cm *ChordManager) GetActiveChord() string {
	return cm.activeChord
}

// CancelChord cancels the current chord sequence
func (cm *ChordManager) CancelChord() {
	cm.activeChord = ""
}

// Message types for chord actions
type InsertTextMsg struct {
	Text string
}

type SwitchWindowMsg struct {
	Window string
}

// ShowChordPromptCmd shows the chord sequence in progress
func ShowChordPromptCmd(chord string) tea.Cmd {
	return func() tea.Msg {
		return ChordPromptMsg{Chord: chord}
	}
}

type ChordPromptMsg struct {
	Chord string
}
