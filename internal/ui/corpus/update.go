package corpus

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/guild-ventures/guild-core/pkg/corpus"
)

// Update handles UI events and state transitions
func (m CorpusModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Prioritize common commands
		if key.Matches(msg, m.keys.Quit) {
			return m, tea.Quit
		}

		// Handle the command input when it's active
		if m.commandInputActive {
			switch {
			case key.Matches(msg, m.keys.Enter):
				cmd := m.executeCommand(m.commandInput.Value())
				m.commandInput.Reset()
				m.commandInputActive = false
				return m, cmd
			case key.Matches(msg, m.keys.Escape):
				m.commandInput.Reset()
				m.commandInputActive = false
				return m, nil
			default:
				var cmd tea.Cmd
				m.commandInput, cmd = m.commandInput.Update(msg)
				return m, cmd
			}
		}

		// If command input isn't active, check if ':' is pressed to activate it
		if msg.String() == ":" && !m.commandInputActive {
			m.commandInputActive = true
			m.commandInput.Focus()
			return m, textinput.Blink
		}

		// Mode-specific keyboard handling
		switch m.mode {
		case ModeList:
			return m.handleListModeUpdate(msg)
		case ModeView:
			return m.handleViewModeUpdate(msg)
		case ModeSearch:
			return m.handleSearchModeUpdate(msg)
		case ModeGraph:
			return m.handleGraphModeUpdate(msg)
		case ModeTags:
			return m.handleTagsModeUpdate(msg)
		}

	case corpus.CorpusDoc:
		// Handle a loaded document
		m.currentDoc = msg
		m.mode = ModeView
		m.viewPort.SetContent(msg.Body)
		m.viewPort.GotoTop()
		// Track the user viewing this document
		cfg := m.configToCorpusConfig()
		go corpus.TrackUserView(m.ctx, m.config.GetUser(), msg.FilePath, cfg)
		return m, nil

	case corpus.Graph:
		// Handle a loaded graph
		m.graph = msg
		return m, nil

	case errMsg:
		m.err = msg
		return m, nil

	case windowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m.handleWindowResize(msg.Width, msg.Height)
	}

	return m, tea.Batch(cmds...)
}

// handleListModeUpdate handles keyboard events in list mode
func (m CorpusModel) handleListModeUpdate(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch {
	case key.Matches(msg, m.keys.Enter):
		selected, ok := m.docList.SelectedItem().(CorpusItem)
		if ok {
			return m, loadDocument(selected.doc.FilePath)
		}
	case key.Matches(msg, m.keys.Search):
		m.mode = ModeSearch
		m.searchInput.Focus()
		return m, textinput.Blink
	case key.Matches(msg, m.keys.Tags):
		cfg := m.configToCorpusConfig()
		return m, loadTags(cfg)
	case key.Matches(msg, m.keys.Graph):
		m.mode = ModeGraph
		cfg := m.configToCorpusConfig()
		return m, loadGraph(cfg)
	default:
		m.docList, cmd = m.docList.Update(msg)
		return m, cmd
	}

	return m, nil
}

// handleViewModeUpdate handles keyboard events in document view mode
func (m CorpusModel) handleViewModeUpdate(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch {
	case key.Matches(msg, m.keys.Back):
		m.mode = ModeList
		return m, nil
	case key.Matches(msg, m.keys.FollowLink):
		// Extract the link under cursor if any and follow it
		link := m.getLinkUnderCursor()
		if link != "" {
			// Try to load the document with the given title
			cfg := m.configToCorpusConfig()
			return m, loadDocumentByTitle(link, cfg)
		}
	default:
		m.viewPort, cmd = m.viewPort.Update(msg)
		return m, cmd
	}

	return m, nil
}

// handleSearchModeUpdate handles keyboard events in search mode
func (m CorpusModel) handleSearchModeUpdate(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch {
	case key.Matches(msg, m.keys.Enter):
		query := m.searchInput.Value()
		m.searchInput.Reset()
		m.mode = ModeList
		cfg := m.configToCorpusConfig()
		return m, search(query, cfg)
	case key.Matches(msg, m.keys.Escape):
		m.searchInput.Reset()
		m.mode = ModeList
		return m, nil
	default:
		m.searchInput, cmd = m.searchInput.Update(msg)
		return m, cmd
	}
}

// handleGraphModeUpdate handles keyboard events in graph visualization mode
func (m CorpusModel) handleGraphModeUpdate(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Back):
		m.mode = ModeList
		return m, nil
	case key.Matches(msg, m.keys.Select):
		// Select and open the document at current cursor position in graph
		if doc := m.getDocAtGraphCursor(); doc != nil {
			return m, loadDocument(doc.FilePath)
		}
	default:
		// Handle graph navigation keys
		// TODO: Implement graph cursor movement
		return m, nil
	}

	return m, nil
}

// handleTagsModeUpdate handles keyboard events in tags browsing mode
func (m CorpusModel) handleTagsModeUpdate(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch {
	case key.Matches(msg, m.keys.Back):
		m.mode = ModeList
		return m, nil
	case key.Matches(msg, m.keys.Enter):
		selected, ok := m.tagList.SelectedItem().(TagItem)
		if ok {
			// Filter documents by selected tag
			m.mode = ModeList
			cfg := m.configToCorpusConfig()
			return m, filterByTag(selected.tag, cfg)
		}
	default:
		m.tagList, cmd = m.tagList.Update(msg)
		return m, cmd
	}

	return m, nil
}

// handleWindowResize adjusts all UI components for a new window size
func (m CorpusModel) handleWindowResize(width, height int) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	// Update viewport dimensions
	m.viewPort.Width = width
	m.viewPort.Height = height - headerHeight - footerHeight - 1
	if m.viewPort.Height < 0 {
		m.viewPort.Height = 0
	}

	// Update list dimensions
	docListHeight := height - headerHeight - footerHeight
	if docListHeight < 0 {
		docListHeight = 0
	}
	m.docList.SetSize(width, docListHeight)
	m.tagList.SetSize(width, docListHeight)

	// Update search input width
	m.searchInput.Width = width - 2

	// Update command input width
	m.commandInput.Width = width - 2

	return m, tea.Batch(cmds...)
}

// executeCommand processes commands entered into the command input
func (m CorpusModel) executeCommand(cmd string) tea.Cmd {
	cmd = strings.TrimSpace(cmd)
	if cmd == "" {
		return nil
	}

	parts := strings.SplitN(cmd, " ", 2)
	command := parts[0]
	args := ""
	if len(parts) > 1 {
		args = parts[1]
	}

	switch command {
	case "search", "s":
		m.mode = ModeList
		cfg := m.configToCorpusConfig()
		return search(args, cfg)
	case "list", "ls":
		m.mode = ModeList
		cfg := m.configToCorpusConfig()
		return listDocuments(cfg)
	case "tag", "tags":
		m.mode = ModeTags
		cfg := m.configToCorpusConfig()
		return loadTags(cfg)
	case "graph", "g":
		m.mode = ModeGraph
		cfg := m.configToCorpusConfig()
		return loadGraph(cfg)
	case "new", "create":
		// Open an editor to create a new document
		// Will be implemented when integrating with CLI
		return nil
	case "help", "h":
		// Show help in the viewport
		m.mode = ModeView
		m.viewPort.SetContent(getHelpText())
		m.viewPort.GotoTop()
		return nil
	default:
		return func() tea.Msg {
			return errMsg{err: fmt.Errorf("unknown command: %s", command)}
		}
	}
}

// Tea commands
func loadDocument(path string) tea.Cmd {
	return func() tea.Msg {
		// Use background context since we can't access m.ctx here
		doc, err := corpus.Load(context.Background(), path)
		if err != nil {
			return errMsg{err: err}
		}
		return *doc
	}
}

func loadDocumentByTitle(title string, cfg corpus.Config) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		paths, err := corpus.List(ctx, cfg)
		if err != nil {
			return errMsg{err: err}
		}

		normalizedTitle := strings.ToLower(title)
		for _, path := range paths {
			// Load the document to check its title
			doc, err := corpus.Load(ctx, path)
			if err != nil {
				continue // Skip documents that can't be loaded
			}

			if strings.ToLower(doc.Title) == normalizedTitle {
				return loadDocument(path)()
			}
		}

		return errMsg{err: fmt.Errorf("document not found: %s", title)}
	}
}

func listDocuments(cfg corpus.Config) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		paths, err := corpus.List(ctx, cfg)
		if err != nil {
			return errMsg{err: err}
		}

		items := make([]list.Item, 0, len(paths))
		for _, path := range paths {
			// Load each document
			doc, err := corpus.Load(ctx, path)
			if err != nil {
				continue // Skip documents that can't be loaded
			}
			items = append(items, CorpusItem{doc: *doc})
		}

		return listItemsMsg{items: items}
	}
}

func loadTags(cfg corpus.Config) tea.Cmd {
	return func() tea.Msg {
		// Get all document paths
		ctx := context.Background()
		paths, err := corpus.List(ctx, cfg)
		if err != nil {
			return errMsg{err: err}
		}

		// Extract and count unique tags
		tagCounts := make(map[string]int)
		for _, path := range paths {
			// Load the document
			doc, err := corpus.Load(ctx, path)
			if err != nil {
				continue // Skip documents that can't be loaded
			}
			for _, tag := range doc.Tags {
				tagCounts[tag]++
			}
		}

		// Create list items for each tag
		items := make([]list.Item, 0, len(tagCounts))
		for tag, count := range tagCounts {
			items = append(items, TagItem{
				tag:   tag,
				count: count,
			})
		}

		return tagListItemsMsg{items: items}
	}
}

func filterByTag(tag string, cfg corpus.Config) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		paths, err := corpus.List(ctx, cfg)
		if err != nil {
			return errMsg{err: err}
		}

		// Filter documents by tag
		var filtered []corpus.CorpusDoc
		for _, path := range paths {
			// Load the document
			doc, err := corpus.Load(ctx, path)
			if err != nil {
				continue // Skip documents that can't be loaded
			}

			for _, docTag := range doc.Tags {
				if docTag == tag {
					filtered = append(filtered, *doc)
					break
				}
			}
		}

		// Create list items for filtered docs
		items := make([]list.Item, len(filtered))
		for i, doc := range filtered {
			items[i] = CorpusItem{doc: doc}
		}

		return listItemsMsg{items: items}
	}
}

func search(query string, cfg corpus.Config) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		paths, err := corpus.List(ctx, cfg)
		if err != nil {
			return errMsg{err: err}
		}

		query = strings.ToLower(query)
		var filtered []corpus.CorpusDoc
		for _, path := range paths {
			// Load the document
			doc, err := corpus.Load(ctx, path)
			if err != nil {
				continue // Skip documents that can't be loaded
			}

			// Check title
			if strings.Contains(strings.ToLower(doc.Title), query) {
				filtered = append(filtered, *doc)
				continue
			}

			// Check tags
			for _, tag := range doc.Tags {
				if strings.Contains(strings.ToLower(tag), query) {
					filtered = append(filtered, *doc)
					break
				}
			}

			// Check content (slower)
			if strings.Contains(strings.ToLower(doc.Body), query) {
				filtered = append(filtered, *doc)
			}
		}

		// Create list items for filtered docs
		items := make([]list.Item, len(filtered))
		for i, doc := range filtered {
			items[i] = CorpusItem{doc: doc}
		}

		return listItemsMsg{items: items}
	}
}

func loadGraph(cfg corpus.Config) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		graph, err := corpus.BuildGraph(ctx, cfg)
		if err != nil {
			return errMsg{err: err}
		}
		return *graph
	}
}

// Helper methods

// getLinkUnderCursor extracts a wikilink at the cursor position in the viewport
func (m CorpusModel) getLinkUnderCursor() string {
	// Get the current line under cursor
	line := m.getCurrentLine()
	if line == "" {
		return ""
	}

	// Find wikilinks in the current line
	links := corpus.ExtractLinks(line)
	if len(links) == 0 {
		return ""
	}

	// For now, return the first link found
	// TODO: Implement more precise cursor positioning to find the exact link under cursor
	return links[0]
}

// getCurrentLine gets the current line of text in the viewport at cursor position
func (m CorpusModel) getCurrentLine() string {
	content := m.viewPort.View()
	lines := strings.Split(content, "\n")

	// Find the line at cursor position (approximation)
	cursorY := m.viewPort.YOffset + m.viewPort.YPosition
	if cursorY < 0 || cursorY >= len(lines) {
		return ""
	}

	return lines[cursorY]
}

// getDocAtGraphCursor returns the document at the current cursor position in graph view
func (m CorpusModel) getDocAtGraphCursor() *corpus.CorpusDoc {
	// This is a placeholder
	// TODO: Implement actual graph cursor navigation and selection
	return nil
}

// getHelpText returns the help documentation for the corpus UI
func getHelpText() string {
	return `# Corpus Browser Help

## Navigation Keys
- Arrow keys: Navigate lists and viewport content
- Enter: Select item or follow link
- Esc: Go back or cancel action

## Mode Keys
- s: Switch to search mode
- g: Switch to graph view
- t: Show tags list
- l: Return to document list

## Commands
Type ':' followed by a command:

- :search <query>  - Search for documents
- :list            - List all documents
- :tags            - Browse by tags
- :graph           - View document graph
- :new             - Create a new document
- :help            - Show this help

## Document View
When viewing a document:
- Press Enter with cursor on a [[wikilink]] to follow it
- Press Esc to return to the list view

## Guild Research Corpus
The corpus is a knowledge repository for storing research findings,
summaries and generated insights. Documents are stored as Markdown
files with YAML frontmatter for metadata.
`
}
