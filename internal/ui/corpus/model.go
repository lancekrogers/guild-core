// Package corpus provides a Bubble Tea UI for the Guild corpus system.
package corpus

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/guild-ventures/guild-core/pkg/corpus"
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// UI modes
const (
	ModeList    = "list"    // Browsing document list
	ModeView    = "view"    // Viewing a document
	ModeSearch  = "search"  // Searching for documents
	ModeGraph   = "graph"   // Viewing graph visualization
	ModeTags    = "tags"    // Browsing by tag
	ModeCommand = "command" // Command input mode
	ModeHelp    = "help"    // Help screen
)

// UI layout constants
const (
	headerHeight = 3
	footerHeight = 3
)

// CorpusModel represents the UI state for the corpus browser.
type CorpusModel struct {
	// Config holds corpus configuration
	config struct {
		CorpusConfig corpus.Config
		CurrentUser  string
	}

	// Data state
	docs        []corpus.CorpusDoc  // All documents in the corpus
	docsByTag   map[string][]string // Documents grouped by tag
	allTags     []string            // All tags in the corpus
	currentDoc  corpus.CorpusDoc    // Currently viewed document
	graph       corpus.Graph        // Document relationship graph
	err         error               // Last error that occurred

	// UI state
	mode             string             // Current UI mode
	width            int                // Terminal width
	height           int                // Terminal height
	docList          list.Model         // Document list
	tagList          list.Model         // Tag list
	viewPort         viewport.Model     // Content viewer
	searchInput      textinput.Model    // Search input
	commandInput     textinput.Model    // Command input
	commandInputActive bool             // Whether command input is active
	graphView        *GraphView         // Graph visualization
	graphOffset      int                // Offset for graph scrolling
	helpView         help.Model         // Help model
	keys             keyMap             // Keyboard shortcuts
	ready            bool               // Whether the UI is initialized
}

// keyMap defines the key bindings for the UI.
type keyMap struct {
	// Navigation
	Up     key.Binding
	Down   key.Binding
	Left   key.Binding
	Right  key.Binding
	PageUp key.Binding
	PageDown key.Binding

	// Actions
	Enter    key.Binding
	Back     key.Binding
	Select   key.Binding
	Escape   key.Binding
	FollowLink key.Binding

	// Tabs/Views
	List   key.Binding
	Tags   key.Binding
	Graph  key.Binding
	Search key.Binding

	// Additional actions
	Refresh   key.Binding
	Backlinks key.Binding
	Command   key.Binding
	Help      key.Binding
	Quit      key.Binding
}

func defaultKeyMap() keyMap {
	return keyMap{
		// Navigation
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "move up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "move down"),
		),
		Left: key.NewBinding(
			key.WithKeys("left", "h"),
			key.WithHelp("←/h", "move left"),
		),
		Right: key.NewBinding(
			key.WithKeys("right", "l"),
			key.WithHelp("→/l", "move right"),
		),
		PageUp: key.NewBinding(
			key.WithKeys("pgup"),
			key.WithHelp("pgup", "page up"),
		),
		PageDown: key.NewBinding(
			key.WithKeys("pgdown"),
			key.WithHelp("pgdown", "page down"),
		),

		// Actions
		Enter: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "select/confirm"),
		),
		Back: key.NewBinding(
			key.WithKeys("backspace", "esc"),
			key.WithHelp("esc", "go back"),
		),
		Select: key.NewBinding(
			key.WithKeys("space"),
			key.WithHelp("space", "select"),
		),
		Escape: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "cancel"),
		),
		FollowLink: key.NewBinding(
			key.WithKeys("f"),
			key.WithHelp("f", "follow link"),
		),

		// Tabs/Views
		List: key.NewBinding(
			key.WithKeys("1", "l"),
			key.WithHelp("l", "document list"),
		),
		Tags: key.NewBinding(
			key.WithKeys("2", "t"),
			key.WithHelp("t", "tags view"),
		),
		Graph: key.NewBinding(
			key.WithKeys("3", "g"),
			key.WithHelp("g", "graph view"),
		),
		Search: key.NewBinding(
			key.WithKeys("/", "s"),
			key.WithHelp("s", "search"),
		),

		// Additional actions
		Refresh: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "refresh"),
		),
		Backlinks: key.NewBinding(
			key.WithKeys("b"),
			key.WithHelp("b", "show backlinks"),
		),
		Command: key.NewBinding(
			key.WithKeys(":"),
			key.WithHelp(":", "command mode"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "help"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
	}
}

// ShortHelp returns keybindings to be shown in the mini help view.
func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Up, k.Down, k.Enter, k.Help, k.Quit}
}

// FullHelp returns keybindings for the expanded help view.
func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Left, k.Right},
		{k.PageUp, k.PageDown, k.Enter, k.Back},
		{k.List, k.Tags, k.Graph, k.Search},
		{k.FollowLink, k.Refresh, k.Command, k.Help, k.Quit},
	}
}

// CorpusItem represents an item in the document list.
type CorpusItem struct {
	doc corpus.CorpusDoc // The document
}

func (i CorpusItem) Title() string {
	return i.doc.Title
}

func (i CorpusItem) Description() string {
	if len(i.doc.Tags) > 0 {
		return strings.Join(i.doc.Tags, ", ")
	}
	return filepath.Base(i.doc.FilePath)
}

func (i CorpusItem) FilterValue() string {
	return i.doc.Title + " " + strings.Join(i.doc.Tags, " ")
}

// TagItem represents an item in the tag list.
type TagItem struct {
	tag   string // Tag name
	count int    // Number of documents with this tag
}

func (i TagItem) Title() string {
	return i.tag
}

func (i TagItem) Description() string {
	return fmt.Sprintf("%d documents", i.count)
}

func (i TagItem) FilterValue() string {
	return i.tag
}

// NewModel creates a new corpus UI model.
func NewModel(cfg corpus.Config, user string) CorpusModel {
	// Initialize the search input
	searchInput := textinput.New()
	searchInput.Placeholder = "Search documents..."
	searchInput.CharLimit = 100
	searchInput.Width = 30

	// Initialize the command input
	commandInput := textinput.New()
	commandInput.Placeholder = "Enter command..."
	commandInput.CharLimit = 100
	commandInput.Width = 30

	// Document list with custom styling
	docDelegate := list.NewDefaultDelegate()
	docDelegate.Styles.SelectedTitle = docDelegate.Styles.SelectedTitle.
		Foreground(lipgloss.Color("170")).
		BorderForeground(lipgloss.Color("170"))
	docDelegate.Styles.SelectedDesc = docDelegate.Styles.SelectedDesc.
		Foreground(lipgloss.Color("241"))

	docList := list.New([]list.Item{}, docDelegate, 0, 0)
	docList.Title = "Guild Corpus Documents"
	docList.SetShowStatusBar(false)
	docList.SetFilteringEnabled(false)
	docList.SetShowHelp(false)

	// Tag list with custom styling
	tagDelegate := list.NewDefaultDelegate()
	tagDelegate.Styles.SelectedTitle = tagDelegate.Styles.SelectedTitle.
		Foreground(lipgloss.Color("170")).
		BorderForeground(lipgloss.Color("170"))

	tagList := list.New([]list.Item{}, tagDelegate, 0, 0)
	tagList.Title = "Document Tags"
	tagList.SetShowStatusBar(false)
	tagList.SetFilteringEnabled(false)
	tagList.SetShowHelp(false)

	// Viewport for document viewing
	viewPort := viewport.New(80, 20)
	viewPort.Style = lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62"))
	viewPort.SetContent("Loading corpus...")

	// Create the model
	return CorpusModel{
		config: struct {
			CorpusConfig corpus.Config
			CurrentUser  string
		}{
			CorpusConfig: cfg,
			CurrentUser:  user,
		},
		docs:        []corpus.CorpusDoc{},
		docsByTag:   make(map[string][]string),
		allTags:     []string{},
		mode:        ModeList,
		docList:     docList,
		tagList:     tagList,
		viewPort:    viewPort,
		searchInput: searchInput,
		commandInput: commandInput,
		keys:        defaultKeyMap(),
		ready:       false,
	}
}

// Init initializes the model.
func (m CorpusModel) Init() tea.Cmd {
	return tea.Batch(
		listDocuments(m.config.CorpusConfig),
		loadTags(m.config.CorpusConfig),
		loadGraph(m.config.CorpusConfig),
	)
}

// getSelectedDoc returns the currently selected document.
func (m CorpusModel) getSelectedDoc() *corpus.CorpusDoc {
	if m.mode == ModeList && len(m.docList.Items()) > 0 {
		idx := m.docList.Index()
		if idx >= 0 && idx < len(m.docList.Items()) {
			item := m.docList.Items()[idx].(CorpusItem)
			return &item.doc
		}
	} else if m.mode == ModeGraph && m.graphView != nil {
		return m.graphView.GetSelectedDoc()
	}
	return nil
}

// Message types used in the update cycle
type windowSizeMsg struct {
	Width  int
	Height int
}

type errMsg struct {
	err error
}

func (e errMsg) Error() string {
	return e.err.Error()
}

type listItemsMsg struct {
	items []list.Item
}

type tagListItemsMsg struct {
	items []list.Item
}