// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package selector

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/guild-ventures/guild-core/pkg/config"
	"github.com/guild-ventures/guild-core/pkg/gerror"
	"github.com/guild-ventures/guild-core/pkg/paths"
	"github.com/guild-ventures/guild-core/pkg/project"
)

// Guild selector key bindings
type guildSelectorKeyMap struct {
	Up        key.Binding
	Down      key.Binding
	Select    key.Binding
	Quit      key.Binding
	CreateNew key.Binding
	Help      key.Binding
}

var guildSelectorKeys = guildSelectorKeyMap{
	Up: key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("↑/k", "up"),
	),
	Down: key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("↓/j", "down"),
	),
	Select: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "select"),
	),
	Quit: key.NewBinding(
		key.WithKeys("ctrl+c", "q"),
		key.WithHelp("q", "quit"),
	),
	CreateNew: key.NewBinding(
		key.WithKeys("n"),
		key.WithHelp("n", "new guild"),
	),
	Help: key.NewBinding(
		key.WithKeys("?"),
		key.WithHelp("?", "toggle help"),
	),
}

// Implement key.Map interface
func (k guildSelectorKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Up, k.Down, k.Select, k.CreateNew, k.Quit}
}

func (k guildSelectorKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Select},
		{k.CreateNew, k.Quit, k.Help},
	}
}

// GuildInfo represents information about a guild
type GuildInfo struct {
	Name        string
	Description string
	AgentCount  int
	Purpose     string
}

// GuildSelectorModel represents the guild selection UI
type GuildSelectorModel struct {
	// UI state
	guilds   []GuildInfo
	cursor   int
	width    int
	height   int
	help     help.Model
	showHelp bool
	err      error

	// Config state
	projectPath    string
	guildConfig    *config.GuildConfigFile
	campaignConfig *config.CampaignConfig
	lastSelected   string

	// Result
	selected string
	quit     bool

	// Context for operations
	ctx context.Context
}

// Style definitions
var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("205")).
			MarginBottom(1)

	selectedStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("205")).
			Background(lipgloss.Color("237"))

	normalStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252"))

	descStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("246")).
			Italic(true)

	borderStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("237")).
			Padding(1, 2)

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Bold(true)

	createNewStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("156")).
			Italic(true)
)

// NewGuildSelector creates a new guild selector model
func NewGuildSelector(ctx context.Context) (*GuildSelectorModel, error) {
	// Get project context
	projCtx, err := project.GetContext()
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to get project context").
			WithComponent("GuildSelector").
			WithOperation("NewGuildSelector")
	}

	m := &GuildSelectorModel{
		help:        help.New(),
		projectPath: projCtx.GetRootPath(),
		ctx:         ctx,
	}

	// Load guild configuration
	if err := m.loadGuilds(); err != nil {
		// If no guilds exist, we'll offer to create defaults
		if gerror.GetCode(err) == gerror.ErrCodeNotFound {
			m.guilds = []GuildInfo{} // Empty list, will show create option
		} else {
			return nil, err
		}
	}

	// Load campaign configuration to get last selected guild
	campaignConfig, err := config.LoadCampaignConfig(ctx, projCtx.GetRootPath())
	if err != nil && gerror.GetCode(err) != gerror.ErrCodeNotFound {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to load campaign config").
			WithComponent("GuildSelector").
			WithOperation("NewGuildSelector")
	}
	if campaignConfig != nil {
		m.campaignConfig = campaignConfig
		m.lastSelected = campaignConfig.LastSelectedGuild
		// Set cursor to last selected if it exists
		m.setCursorToGuild(m.lastSelected)
	}

	return m, nil
}

// loadGuilds loads the guild configuration and populates the guild list
func (m *GuildSelectorModel) loadGuilds() error {
	guildConfig, err := config.LoadGuildConfigFile(m.ctx, m.projectPath)
	if err != nil {
		return err
	}
	m.guildConfig = guildConfig

	// Convert to GuildInfo list
	m.guilds = make([]GuildInfo, 0, len(guildConfig.Guilds))
	for name, guild := range guildConfig.Guilds {
		m.guilds = append(m.guilds, GuildInfo{
			Name:        name,
			Description: guild.Description,
			Purpose:     guild.Purpose,
			AgentCount:  len(guild.Agents),
		})
	}

	// Sort guilds alphabetically
	sort.Slice(m.guilds, func(i, j int) bool {
		return m.guilds[i].Name < m.guilds[j].Name
	})

	return nil
}

// setCursorToGuild sets the cursor to the specified guild if it exists
func (m *GuildSelectorModel) setCursorToGuild(guildName string) {
	for i, guild := range m.guilds {
		if guild.Name == guildName {
			m.cursor = i
			return
		}
	}
}

// Init initializes the guild selector
func (m *GuildSelectorModel) Init() tea.Cmd {
	return nil
}

// Update handles messages
func (m *GuildSelectorModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.help.Width = msg.Width

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, guildSelectorKeys.Quit):
			m.quit = true
			return m, tea.Quit

		case key.Matches(msg, guildSelectorKeys.Up):
			if m.cursor > 0 {
				m.cursor--
			}

		case key.Matches(msg, guildSelectorKeys.Down):
			// maxCursor is the index of the "Create New Guild" option
			maxCursor := len(m.guilds) // "Create New Guild" is at position len(guilds)
			if len(m.guilds) == 0 {
				maxCursor = 0 // When no guilds, "Create New Guild" is at position 0
			}
			if m.cursor < maxCursor {
				m.cursor++
			}

		case key.Matches(msg, guildSelectorKeys.Select):
			if len(m.guilds) == 0 || m.cursor == len(m.guilds) {
				// Create new guild
				return m, m.createDefaultGuild
			}
			// Select existing guild
			m.selected = m.guilds[m.cursor].Name
			return m, m.saveSelection

		case key.Matches(msg, guildSelectorKeys.CreateNew):
			return m, m.createDefaultGuild

		case key.Matches(msg, guildSelectorKeys.Help):
			m.showHelp = !m.showHelp
		}

	case guildCreatedMsg:
		// Reload guilds after creation
		if err := m.loadGuilds(); err != nil {
			m.err = err
			return m, nil
		}
		// Select the newly created guild
		m.setCursorToGuild(string(msg))
		m.selected = string(msg)
		return m, m.saveSelection

	case selectionSavedMsg:
		return m, tea.Quit

	case errMsg:
		m.err = msg.error
		return m, nil
	}

	return m, nil
}

// View renders the guild selector
func (m *GuildSelectorModel) View() string {
	if m.err != nil {
		return errorStyle.Render(fmt.Sprintf("Error: %v\n\nPress 'q' to quit.", m.err))
	}

	var b strings.Builder

	// Title
	b.WriteString(titleStyle.Render("🏰 Select Guild"))
	b.WriteString("\n\n")

	// Show guilds or create option
	if len(m.guilds) == 0 {
		b.WriteString("No guilds found. Press ")
		b.WriteString(createNewStyle.Render("'n'"))
		b.WriteString(" to create a default guild.\n\n")

		// Show create option
		if m.cursor == 0 {
			b.WriteString(selectedStyle.Render("→ Create Default Guild"))
		} else {
			b.WriteString(normalStyle.Render("  Create Default Guild"))
		}
	} else {
		// List guilds
		for i, guild := range m.guilds {
			cursor := "  "
			style := normalStyle
			if i == m.cursor {
				cursor = "→ "
				style = selectedStyle
			}

			// Guild name with agent count
			line := fmt.Sprintf("%s%s (%d agents)", cursor, guild.Name, guild.AgentCount)
			b.WriteString(style.Render(line))

			// Show if it's the last selected
			if guild.Name == m.lastSelected {
				b.WriteString(descStyle.Render(" [last selected]"))
			}
			b.WriteString("\n")

			// Description on next line, indented
			if i == m.cursor && guild.Description != "" {
				b.WriteString(descStyle.Render(fmt.Sprintf("    %s\n", guild.Purpose)))
			}
		}

		// Add create new option at the end
		b.WriteString("\n")
		cursor := "  "
		style := createNewStyle
		if m.cursor == len(m.guilds) {
			cursor = "→ "
			style = selectedStyle
		}
		b.WriteString(style.Render(fmt.Sprintf("%sCreate New Guild", cursor)))
	}

	// Help
	b.WriteString("\n\n")
	if m.showHelp {
		b.WriteString(m.help.View(guildSelectorKeys))
	} else {
		b.WriteString(descStyle.Render("Press ? for help"))
	}

	// Wrap in border
	content := borderStyle.Render(b.String())

	// Center vertically
	if m.height > 0 {
		lines := strings.Count(content, "\n") + 1
		padding := (m.height - lines) / 2
		if padding > 0 {
			content = strings.Repeat("\n", padding) + content
		}
	}

	return content
}

// Selected returns the selected guild name
func (m *GuildSelectorModel) Selected() string {
	return m.selected
}

// Quit returns whether the user quit
func (m *GuildSelectorModel) Quit() bool {
	return m.quit
}

// Message types
type (
	guildCreatedMsg   string
	selectionSavedMsg struct{}
	errMsg            struct{ error }
)

// createDefaultGuild creates a default guild configuration
func (m *GuildSelectorModel) createDefaultGuild() tea.Msg {
	// Check if we need to create guild.yml first
	guildConfigPath := filepath.Join(m.projectPath, ".guild", "guild.yml")
	if _, err := os.Stat(guildConfigPath); os.IsNotExist(err) {
		// Create initial guild config
		m.guildConfig = &config.GuildConfigFile{
			Guilds: make(map[string]config.GuildDefinition),
		}
	}

	// Determine which provider is available
	provider := "claude"
	agentPrefix := "claude"

	// Check if Ollama is available by looking at the main guild.yaml
	mainConfigPath := filepath.Join(m.projectPath, paths.DefaultCampaignDir, "guild.yaml")
	if data, err := os.ReadFile(mainConfigPath); err == nil {
		if strings.Contains(string(data), "ollama:") {
			// Ollama is configured, create Ollama-based guild
			provider = "ollama"
			agentPrefix = "local"
		}
	}

	// Create default guild
	defaultGuildName := fmt.Sprintf("default-%s-guild", provider)
	defaultGuild := config.GuildDefinition{
		Purpose:     fmt.Sprintf("General purpose development guild using %s", provider),
		Description: fmt.Sprintf("A versatile guild for software development tasks using %s models", provider),
		Agents: []string{
			fmt.Sprintf("%s-manager", agentPrefix),
			fmt.Sprintf("%s-developer", agentPrefix),
			fmt.Sprintf("%s-tester", agentPrefix),
		},
		Coordination: &config.CoordinationSettings{
			MaxParallelTasks: 3,
			ReviewRequired:   false,
			AutoHandoff:      true,
		},
	}

	// Add guild to config
	if err := m.guildConfig.AddGuild(defaultGuildName, defaultGuild); err != nil {
		return errMsg{err}
	}

	// Save the configuration
	if err := config.SaveGuildConfigFile(m.ctx, m.projectPath, m.guildConfig); err != nil {
		return errMsg{err}
	}

	// Also ensure we have the corresponding agents in guild.yaml
	// This would typically be done by the init command, but we'll add basic ones
	if err := m.ensureDefaultAgents(provider, agentPrefix); err != nil {
		return errMsg{err}
	}

	return guildCreatedMsg(defaultGuildName)
}

// ensureDefaultAgents ensures the default agents exist in guild.yaml
func (m *GuildSelectorModel) ensureDefaultAgents(provider, agentPrefix string) error {
	// This is a simplified version - in production, you'd want to properly
	// update the guild.yaml file with the agent definitions
	// For now, we assume the agents exist or will be created by the system
	return nil
}

// saveSelection saves the selected guild to campaign.yml
func (m *GuildSelectorModel) saveSelection() tea.Msg {
	// Initialize campaign config if it doesn't exist
	if m.campaignConfig == nil {
		m.campaignConfig = &config.CampaignConfig{
			Name:        "default-campaign",
			Description: "Default campaign",
		}
	}

	// Update last selected guild
	if err := m.campaignConfig.UpdateLastSelectedGuild(m.ctx, m.projectPath, m.selected); err != nil {
		return errMsg{err}
	}

	return selectionSavedMsg{}
}

// RunGuildSelector runs the guild selector UI and returns the selected guild
func RunGuildSelector(ctx context.Context) (string, error) {
	model, err := NewGuildSelector(ctx)
	if err != nil {
		return "", err
	}

	p := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		return "", gerror.Wrap(err, gerror.ErrCodeInternal, "failed to run guild selector").
			WithComponent("GuildSelector").
			WithOperation("RunGuildSelector")
	}

	if model.Quit() {
		return "", gerror.New(gerror.ErrCodeCancelled, "user cancelled guild selection", nil).
			WithComponent("GuildSelector").
			WithOperation("RunGuildSelector")
	}

	if model.Selected() == "" {
		return "", gerror.New(gerror.ErrCodeInternal, "no guild selected", nil).
			WithComponent("GuildSelector").
			WithOperation("RunGuildSelector")
	}

	return model.Selected(), nil
}
