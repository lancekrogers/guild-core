// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

// Package shortcuts provides comprehensive keyboard shortcut management for Guild Framework UI
//
// This package implements the keyboard shortcut requirements identified in performance optimization,
// Agent 1 task, providing:
//   - Advanced keyboard shortcut system with context awareness
//   - Command palette with fuzzy search (VS Code style)
//   - Multi-modal shortcuts (global, chat, kanban, etc.)
//   - Customizable key bindings with conflict resolution
//
// The package follows Guild's architectural patterns:
//   - Context-first error handling with gerror
//   - Interface-driven design for testability
//   - Registry pattern for shortcut registration
//   - Observability integration
//
// Example usage:
//
//	// Create shortcut manager
//	manager := NewShortcutManager()
//
//	// Register custom shortcut
//	err := manager.RegisterShortcut(&Shortcut{
//		ID:          "custom_action",
//		Key:         "ctrl+alt+c",
//		Command:     "guild.custom.action",
//		Description: "Perform custom action",
//		Handler:     customHandler,
//	})
//
//	// Handle key press
//	cmd := manager.HandleKeyPress(ctx, "ctrl+shift+p")
//
//	// Open command palette
//	manager.ShowCommandPalette()
package shortcuts

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/guild-framework/guild-core/internal/ui"
	"github.com/guild-framework/guild-core/pkg/gerror"
	"go.uber.org/zap"
)

// Package version for compatibility tracking
const (
	Version     = "1.0.0"
	APIVersion  = "v1"
	PackageName = "shortcuts"
)

// ShortcutManager handles all keyboard shortcuts and command palette
type ShortcutManager struct {
	shortcuts      map[string]*Shortcut
	commandPalette *CommandPalette
	contexts       map[string]*ShortcutContext
	globalContext  *ShortcutContext
	currentContext string
	enabled        bool
	mu             sync.RWMutex
	logger         *zap.Logger
}

// Shortcut represents a keyboard shortcut
type Shortcut struct {
	ID          string          `json:"id"`
	Key         string          `json:"key"`         // Key combination (e.g., "ctrl+k")
	Command     string          `json:"command"`     // Command to execute
	Description string          `json:"description"` // User-friendly description
	Category    string          `json:"category"`    // Category for organization
	Context     string          `json:"context"`     // When this shortcut is active
	Handler     ShortcutHandler `json:"-"`           // Function to execute
	Enabled     bool            `json:"enabled"`     // Whether shortcut is active
	Priority    int             `json:"priority"`    // Resolution priority for conflicts
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

// ShortcutContext defines when shortcuts are active
type ShortcutContext struct {
	Name        string               `json:"name"`
	Description string               `json:"description"`
	Shortcuts   map[string]*Shortcut `json:"shortcuts"`
	Parent      *ShortcutContext     `json:"-"` // For inheritance
	Enabled     bool                 `json:"enabled"`
	CreatedAt   time.Time            `json:"created_at"`
}

// CommandPalette provides VS Code-style command searching
type CommandPalette struct {
	input    textinput.Model
	commands []*Command
	filtered []*Command
	selected int
	visible  bool
	width    int
	height   int
	theme    string
	mu       sync.RWMutex
}

// Command represents an executable command
type Command struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Category    string         `json:"category"`
	Keywords    []string       `json:"keywords"` // For search
	Handler     CommandHandler `json:"-"`
	Shortcut    string         `json:"shortcut,omitempty"`
	Icon        string         `json:"icon,omitempty"`
	Enabled     bool           `json:"enabled"`
	LastUsed    time.Time      `json:"last_used"`
	UsageCount  int            `json:"usage_count"`
}

// SearchResult represents a command search result with relevance scoring
type SearchResult struct {
	Command   *Command `json:"command"`
	Score     int      `json:"score"`
	MatchType string   `json:"match_type"`
}

// KeyBinding represents a key combination
type KeyBinding struct {
	Key       string    `json:"key"`
	Modifiers []string  `json:"modifiers"` // ctrl, alt, shift, cmd
	Context   string    `json:"context"`
	CreatedAt time.Time `json:"created_at"`
}

// Callbacks for events
type (
	ShortcutHandler func(ctx context.Context) tea.Cmd
	CommandHandler  func(ctx context.Context, args map[string]interface{}) tea.Cmd
)

// NewShortcutManager creates a new shortcut manager with built-in shortcuts
func NewShortcutManager() *ShortcutManager {
	return NewShortcutManagerWithLogger(nil)
}

// NewShortcutManagerWithLogger creates a new shortcut manager with a specific logger
func NewShortcutManagerWithLogger(logger *zap.Logger) *ShortcutManager {
	if logger == nil {
		logger, _ = zap.NewDevelopment()
	}

	sm := &ShortcutManager{
		shortcuts: make(map[string]*Shortcut),
		contexts:  make(map[string]*ShortcutContext),
		enabled:   true,
		logger:    logger.Named("shortcut-manager"),
	}

	// Create global context
	sm.globalContext = &ShortcutContext{
		Name:        "global",
		Description: "Global shortcuts available everywhere",
		Shortcuts:   make(map[string]*Shortcut),
		Enabled:     true,
		CreatedAt:   time.Now(),
	}
	sm.contexts["global"] = sm.globalContext

	// Initialize command palette
	sm.commandPalette = NewCommandPalette()

	// Register default Guild shortcuts
	sm.registerDefaultShortcuts()
	sm.registerDefaultContexts()

	return sm
}

// RegisterShortcut registers a new keyboard shortcut
func (sm *ShortcutManager) RegisterShortcut(shortcut *Shortcut) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// Validate shortcut
	if err := sm.validateShortcut(shortcut); err != nil {
		return err
	}

	// Check for conflicts
	if err := sm.checkShortcutConflicts(shortcut); err != nil {
		return err
	}

	shortcut.CreatedAt = time.Now()
	shortcut.UpdatedAt = time.Now()

	sm.shortcuts[shortcut.ID] = shortcut

	// Add to appropriate context
	contextName := shortcut.Context
	if contextName == "" {
		contextName = "global"
	}

	context, exists := sm.contexts[contextName]
	if !exists {
		// Create context if it doesn't exist
		context = &ShortcutContext{
			Name:      contextName,
			Shortcuts: make(map[string]*Shortcut),
			Enabled:   true,
			CreatedAt: time.Now(),
		}
		sm.contexts[contextName] = context
	}

	context.Shortcuts[shortcut.Key] = shortcut

	sm.logger.Info("Shortcut registered",
		zap.String("id", shortcut.ID),
		zap.String("key", shortcut.Key),
		zap.String("context", contextName))

	return nil
}

// HandleKeyPress processes a key press and executes matching shortcuts
func (sm *ShortcutManager) HandleKeyPress(ctx context.Context, key string) tea.Cmd {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	if !sm.enabled {
		return nil
	}

	// Normalize key combination
	normalizedKey := sm.normalizeKey(key)

	// Check current context first
	if sm.currentContext != "" {
		if context, exists := sm.contexts[sm.currentContext]; exists && context.Enabled {
			if shortcut, exists := context.Shortcuts[normalizedKey]; exists && shortcut.Enabled {
				sm.logger.Debug("Executing context shortcut",
					zap.String("key", normalizedKey),
					zap.String("context", sm.currentContext),
					zap.String("command", shortcut.Command))
				return shortcut.Handler(ctx)
			}
		}
	}

	// Check global context
	if shortcut, exists := sm.globalContext.Shortcuts[normalizedKey]; exists && shortcut.Enabled {
		sm.logger.Debug("Executing global shortcut",
			zap.String("key", normalizedKey),
			zap.String("command", shortcut.Command))
		return shortcut.Handler(ctx)
	}

	return nil
}

// SetContext switches the active shortcut context
func (sm *ShortcutManager) SetContext(ctx context.Context, contextName string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if _, exists := sm.contexts[contextName]; !exists {
		return gerror.New(ui.ErrCodeUIContextNotFound, fmt.Sprintf("context '%s' not found", contextName), nil).
			WithComponent("shortcut-manager").
			WithOperation("SetContext")
	}

	previousContext := sm.currentContext
	sm.currentContext = contextName

	sm.logger.Info("Context switched",
		zap.String("previous", previousContext),
		zap.String("current", contextName))

	return nil
}

// ShowCommandPalette shows the command palette
func (sm *ShortcutManager) ShowCommandPalette() tea.Cmd {
	sm.commandPalette.Show()
	return func() tea.Msg {
		return CommandPaletteToggleMsg{Visible: true}
	}
}

// HideCommandPalette hides the command palette
func (sm *ShortcutManager) HideCommandPalette() tea.Cmd {
	sm.commandPalette.Hide()
	return func() tea.Msg {
		return CommandPaletteToggleMsg{Visible: false}
	}
}

// GetCommandPalette returns the command palette for UI integration
func (sm *ShortcutManager) GetCommandPalette() *CommandPalette {
	return sm.commandPalette
}

// ListShortcuts returns all registered shortcuts
func (sm *ShortcutManager) ListShortcuts() []*Shortcut {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	shortcuts := make([]*Shortcut, 0, len(sm.shortcuts))
	for _, shortcut := range sm.shortcuts {
		shortcuts = append(shortcuts, shortcut)
	}
	return shortcuts
}

// ListContexts returns all available contexts
func (sm *ShortcutManager) ListContexts() []*ShortcutContext {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	contexts := make([]*ShortcutContext, 0, len(sm.contexts))
	for _, context := range sm.contexts {
		contexts = append(contexts, context)
	}
	return contexts
}

// registerDefaultShortcuts registers built-in Guild shortcuts
func (sm *ShortcutManager) registerDefaultShortcuts() {
	// Command palette
	sm.RegisterShortcut(&Shortcut{
		ID:          "command_palette",
		Key:         "ctrl+shift+p",
		Command:     "guild.command.palette",
		Description: "Open command palette",
		Category:    "navigation",
		Context:     "global",
		Handler:     sm.openCommandPalette,
		Enabled:     true,
		Priority:    100,
	})

	// Quick navigation
	sm.RegisterShortcut(&Shortcut{
		ID:          "quick_open",
		Key:         "ctrl+p",
		Command:     "guild.quick.open",
		Description: "Quick open file/commission",
		Category:    "navigation",
		Context:     "global",
		Handler:     sm.quickOpen,
		Enabled:     true,
		Priority:    90,
	})

	// Agent communication shortcuts
	sm.RegisterShortcut(&Shortcut{
		ID:          "mention_agent_1",
		Key:         "ctrl+1",
		Command:     "guild.core.mention.1",
		Description: "Mention Agent 1",
		Category:    "agent",
		Context:     "chat",
		Handler:     sm.mentionAgent("agent-1"),
		Enabled:     true,
		Priority:    80,
	})

	sm.RegisterShortcut(&Shortcut{
		ID:          "mention_agent_2",
		Key:         "ctrl+2",
		Command:     "guild.core.mention.2",
		Description: "Mention Agent 2",
		Category:    "agent",
		Context:     "chat",
		Handler:     sm.mentionAgent("agent-2"),
		Enabled:     true,
		Priority:    80,
	})

	sm.RegisterShortcut(&Shortcut{
		ID:          "mention_agent_3",
		Key:         "ctrl+3",
		Command:     "guild.core.mention.3",
		Description: "Mention Agent 3",
		Category:    "agent",
		Context:     "chat",
		Handler:     sm.mentionAgent("agent-3"),
		Enabled:     true,
		Priority:    80,
	})

	// View switching
	sm.RegisterShortcut(&Shortcut{
		ID:          "switch_to_chat",
		Key:         "ctrl+shift+c",
		Command:     "guild.view.chat",
		Description: "Switch to chat view",
		Category:    "view",
		Context:     "global",
		Handler:     sm.switchToView("chat"),
		Enabled:     true,
		Priority:    70,
	})

	sm.RegisterShortcut(&Shortcut{
		ID:          "switch_to_kanban",
		Key:         "ctrl+shift+k",
		Command:     "guild.view.kanban",
		Description: "Switch to kanban view",
		Category:    "view",
		Context:     "global",
		Handler:     sm.switchToView("kanban"),
		Enabled:     true,
		Priority:    70,
	})

	// Task management
	sm.RegisterShortcut(&Shortcut{
		ID:          "new_commission",
		Key:         "ctrl+n",
		Command:     "guild.commission.new",
		Description: "Create new commission",
		Category:    "task",
		Context:     "global",
		Handler:     sm.newCommission,
		Enabled:     true,
		Priority:    60,
	})

	sm.RegisterShortcut(&Shortcut{
		ID:          "focus_search",
		Key:         "ctrl+f",
		Command:     "guild.search.focus",
		Description: "Focus search input",
		Category:    "navigation",
		Context:     "global",
		Handler:     sm.focusSearch,
		Enabled:     true,
		Priority:    60,
	})

	// Developer shortcuts
	sm.RegisterShortcut(&Shortcut{
		ID:          "toggle_debug",
		Key:         "ctrl+shift+d",
		Command:     "guild.debug.toggle",
		Description: "Toggle debug mode",
		Category:    "developer",
		Context:     "global",
		Handler:     sm.toggleDebug,
		Enabled:     true,
		Priority:    50,
	})

	sm.RegisterShortcut(&Shortcut{
		ID:          "performance_profile",
		Key:         "ctrl+shift+r",
		Command:     "guild.performance.profile",
		Description: "Start performance profiling",
		Category:    "developer",
		Context:     "global",
		Handler:     sm.startProfiling,
		Enabled:     true,
		Priority:    50,
	})

	// Help
	sm.RegisterShortcut(&Shortcut{
		ID:          "show_shortcuts",
		Key:         "ctrl+?",
		Command:     "guild.help.shortcuts",
		Description: "Show keyboard shortcuts",
		Category:    "help",
		Context:     "global",
		Handler:     sm.showShortcutsHelp,
		Enabled:     true,
		Priority:    40,
	})

	// Escape key for universal cancel
	sm.RegisterShortcut(&Shortcut{
		ID:          "escape",
		Key:         "esc",
		Command:     "guild.general.escape",
		Description: "Cancel current operation",
		Category:    "general",
		Context:     "global",
		Handler:     sm.handleEscape,
		Enabled:     true,
		Priority:    200, // Highest priority
	})
}

// registerDefaultContexts creates default shortcut contexts
func (sm *ShortcutManager) registerDefaultContexts() {
	contexts := []*ShortcutContext{
		{
			Name:        "chat",
			Description: "Chat interface shortcuts",
			Shortcuts:   make(map[string]*Shortcut),
			Enabled:     true,
			CreatedAt:   time.Now(),
		},
		{
			Name:        "kanban",
			Description: "Kanban board shortcuts",
			Shortcuts:   make(map[string]*Shortcut),
			Enabled:     true,
			CreatedAt:   time.Now(),
		},
		{
			Name:        "modal",
			Description: "Modal dialog shortcuts",
			Shortcuts:   make(map[string]*Shortcut),
			Enabled:     true,
			CreatedAt:   time.Now(),
		},
		{
			Name:        "search",
			Description: "Search interface shortcuts",
			Shortcuts:   make(map[string]*Shortcut),
			Enabled:     true,
			CreatedAt:   time.Now(),
		},
	}

	for _, context := range contexts {
		sm.contexts[context.Name] = context
	}
}

// Shortcut handlers
func (sm *ShortcutManager) openCommandPalette(ctx context.Context) tea.Cmd {
	return sm.ShowCommandPalette()
}

func (sm *ShortcutManager) quickOpen(ctx context.Context) tea.Cmd {
	return func() tea.Msg {
		return QuickOpenMsg{Type: "file"}
	}
}

func (sm *ShortcutManager) mentionAgent(agentID string) ShortcutHandler {
	return func(ctx context.Context) tea.Cmd {
		return func() tea.Msg {
			return AgentMentionMsg{AgentID: agentID}
		}
	}
}

func (sm *ShortcutManager) switchToView(view string) ShortcutHandler {
	return func(ctx context.Context) tea.Cmd {
		return func() tea.Msg {
			return ViewSwitchMsg{View: view}
		}
	}
}

func (sm *ShortcutManager) newCommission(ctx context.Context) tea.Cmd {
	return func() tea.Msg {
		return NewCommissionMsg{}
	}
}

func (sm *ShortcutManager) focusSearch(ctx context.Context) tea.Cmd {
	return func() tea.Msg {
		return FocusSearchMsg{}
	}
}

func (sm *ShortcutManager) toggleDebug(ctx context.Context) tea.Cmd {
	return func() tea.Msg {
		return ToggleDebugMsg{}
	}
}

func (sm *ShortcutManager) startProfiling(ctx context.Context) tea.Cmd {
	return func() tea.Msg {
		return StartProfilingMsg{}
	}
}

func (sm *ShortcutManager) showShortcutsHelp(ctx context.Context) tea.Cmd {
	return func() tea.Msg {
		return ShowHelpMsg{Type: "shortcuts"}
	}
}

func (sm *ShortcutManager) handleEscape(ctx context.Context) tea.Cmd {
	return func() tea.Msg {
		return EscapeMsg{}
	}
}

// Helper methods
func (sm *ShortcutManager) validateShortcut(shortcut *Shortcut) error {
	if shortcut.ID == "" {
		return gerror.New(gerror.ErrCodeValidation, "shortcut ID cannot be empty", nil).
			WithComponent("shortcut-manager").
			WithOperation("validateShortcut")
	}

	if shortcut.Key == "" {
		return gerror.New(gerror.ErrCodeValidation, "shortcut key cannot be empty", nil).
			WithComponent("shortcut-manager").
			WithOperation("validateShortcut")
	}

	if shortcut.Handler == nil {
		return gerror.New(gerror.ErrCodeValidation, "shortcut handler cannot be nil", nil).
			WithComponent("shortcut-manager").
			WithOperation("validateShortcut")
	}

	return nil
}

func (sm *ShortcutManager) checkShortcutConflicts(shortcut *Shortcut) error {
	contextName := shortcut.Context
	if contextName == "" {
		contextName = "global"
	}

	context, exists := sm.contexts[contextName]
	if !exists {
		return nil // No conflicts in non-existent context
	}

	if existing, exists := context.Shortcuts[shortcut.Key]; exists {
		if existing.Priority > shortcut.Priority {
			return gerror.New(gerror.ErrCodeConflict,
				fmt.Sprintf("shortcut conflict: key '%s' already bound to '%s' with higher priority",
					shortcut.Key, existing.ID), nil).
				WithComponent("shortcut-manager").
				WithOperation("checkShortcutConflicts").
				WithDetails("existing_id", existing.ID).
				WithDetails("existing_priority", fmt.Sprintf("%d", existing.Priority)).
				WithDetails("new_priority", fmt.Sprintf("%d", shortcut.Priority))
		}
	}

	return nil
}

func (sm *ShortcutManager) normalizeKey(key string) string {
	// Convert to lowercase and standardize modifiers
	key = strings.ToLower(key)
	key = strings.ReplaceAll(key, "command", "cmd")
	key = strings.ReplaceAll(key, "control", "ctrl")
	key = strings.ReplaceAll(key, "option", "alt")

	// Sort modifiers for consistency
	parts := strings.Split(key, "+")
	if len(parts) > 1 {
		modifiers := parts[:len(parts)-1]
		mainKey := parts[len(parts)-1]

		sort.Strings(modifiers)
		return strings.Join(modifiers, "+") + "+" + mainKey
	}

	return key
}

// NewCommandPalette creates a new command palette
func NewCommandPalette() *CommandPalette {
	input := textinput.New()
	input.Placeholder = "Type a command..."
	input.CharLimit = 256

	cp := &CommandPalette{
		input:    input,
		commands: make([]*Command, 0),
		filtered: make([]*Command, 0),
		visible:  false,
		width:    80,
		height:   15,
		theme:    "default",
	}

	// Register all available commands
	cp.registerCommands()

	return cp
}

// Show displays the command palette
func (cp *CommandPalette) Show() {
	cp.mu.Lock()
	defer cp.mu.Unlock()

	cp.visible = true
	cp.input.Focus()
	cp.selected = 0
	cp.filterCommands("")
}

// Hide conceals the command palette
func (cp *CommandPalette) Hide() {
	cp.mu.Lock()
	defer cp.mu.Unlock()

	cp.visible = false
	cp.input.Blur()
	cp.input.SetValue("")
}

// IsVisible returns whether the command palette is currently shown
func (cp *CommandPalette) IsVisible() bool {
	cp.mu.RLock()
	defer cp.mu.RUnlock()
	return cp.visible
}

// UpdateInput updates the search input
func (cp *CommandPalette) UpdateInput(msg tea.Msg) tea.Cmd {
	cp.mu.Lock()
	defer cp.mu.Unlock()

	var cmd tea.Cmd
	cp.input, cmd = cp.input.Update(msg)

	// Update filtered commands based on input
	cp.filterCommands(cp.input.Value())

	return cmd
}

// SelectNext moves selection to the next command
func (cp *CommandPalette) SelectNext() {
	cp.mu.Lock()
	defer cp.mu.Unlock()

	if len(cp.filtered) > 0 {
		cp.selected = (cp.selected + 1) % len(cp.filtered)
	}
}

// SelectPrevious moves selection to the previous command
func (cp *CommandPalette) SelectPrevious() {
	cp.mu.Lock()
	defer cp.mu.Unlock()

	if len(cp.filtered) > 0 {
		cp.selected = (cp.selected - 1 + len(cp.filtered)) % len(cp.filtered)
	}
}

// ExecuteSelected executes the currently selected command
func (cp *CommandPalette) ExecuteSelected(ctx context.Context) tea.Cmd {
	cp.mu.Lock()

	var command *Command
	if len(cp.filtered) > 0 && cp.selected < len(cp.filtered) {
		command = cp.filtered[cp.selected]
		command.LastUsed = time.Now()
		command.UsageCount++
	}

	cp.mu.Unlock()

	if command != nil {
		cp.Hide() // Hide after releasing lock to avoid deadlock

		if command.Handler != nil {
			return command.Handler(ctx, nil)
		}
	}

	return nil
}

// GetFilteredCommands returns currently filtered commands
func (cp *CommandPalette) GetFilteredCommands() []*Command {
	cp.mu.RLock()
	defer cp.mu.RUnlock()

	return cp.filtered
}

// GetSelectedIndex returns the currently selected command index
func (cp *CommandPalette) GetSelectedIndex() int {
	cp.mu.RLock()
	defer cp.mu.RUnlock()

	return cp.selected
}

// registerCommands registers all available commands
func (cp *CommandPalette) registerCommands() {
	commands := []*Command{
		{
			ID:          "guild.commission.new",
			Name:        "New Commission",
			Description: "Create a new commission for agent work",
			Category:    "Commission",
			Keywords:    []string{"new", "create", "commission", "project", "task"},
			Shortcut:    "Ctrl+N",
			Icon:        "📝",
			Enabled:     true,
		},
		{
			ID:          "guild.core.status",
			Name:        "Show Agent Status",
			Description: "Display current status of all agents",
			Category:    "Agent",
			Keywords:    []string{"agent", "status", "active", "busy", "online"},
			Icon:        "👥",
			Enabled:     true,
		},
		{
			ID:          "guild.session.export",
			Name:        "Export Session",
			Description: "Export current chat session",
			Category:    "Session",
			Keywords:    []string{"export", "save", "session", "download", "backup"},
			Icon:        "💾",
			Enabled:     true,
		},
		{
			ID:          "guild.performance.monitor",
			Name:        "Performance Monitor",
			Description: "Open performance monitoring dashboard",
			Category:    "Developer",
			Keywords:    []string{"performance", "monitor", "stats", "profile", "debug"},
			Shortcut:    "Ctrl+Shift+R",
			Icon:        "📊",
			Enabled:     true,
		},
		{
			ID:          "guild.theme.switch",
			Name:        "Switch Theme",
			Description: "Change UI theme",
			Category:    "Appearance",
			Keywords:    []string{"theme", "color", "dark", "light", "appearance"},
			Icon:        "🎨",
			Enabled:     true,
		},
		{
			ID:          "guild.shortcuts.help",
			Name:        "Keyboard Shortcuts",
			Description: "Show all available keyboard shortcuts",
			Category:    "Help",
			Keywords:    []string{"help", "shortcuts", "keys", "commands", "hotkeys"},
			Shortcut:    "Ctrl+?",
			Icon:        "⌨️",
			Enabled:     true,
		},
		{
			ID:          "guild.view.chat",
			Name:        "Switch to Chat",
			Description: "Switch to chat interface",
			Category:    "Navigation",
			Keywords:    []string{"chat", "conversation", "talk", "message"},
			Shortcut:    "Ctrl+Shift+C",
			Icon:        "💬",
			Enabled:     true,
		},
		{
			ID:          "guild.view.kanban",
			Name:        "Switch to Kanban",
			Description: "Switch to kanban board view",
			Category:    "Navigation",
			Keywords:    []string{"kanban", "board", "tasks", "workflow"},
			Shortcut:    "Ctrl+Shift+K",
			Icon:        "📋",
			Enabled:     true,
		},
		{
			ID:          "guild.search.global",
			Name:        "Global Search",
			Description: "Search across all content",
			Category:    "Search",
			Keywords:    []string{"search", "find", "lookup", "query"},
			Shortcut:    "Ctrl+F",
			Icon:        "🔍",
			Enabled:     true,
		},
		{
			ID:          "guild.debug.toggle",
			Name:        "Toggle Debug Mode",
			Description: "Enable or disable debug information",
			Category:    "Developer",
			Keywords:    []string{"debug", "developer", "console", "logs"},
			Shortcut:    "Ctrl+Shift+D",
			Icon:        "🐛",
			Enabled:     true,
		},
	}

	cp.commands = commands
	cp.filtered = commands
}

// filterCommands performs fuzzy search on commands
func (cp *CommandPalette) filterCommands(query string) {
	if query == "" {
		// Show all commands sorted by usage
		cp.filtered = make([]*Command, len(cp.commands))
		copy(cp.filtered, cp.commands)

		sort.Slice(cp.filtered, func(i, j int) bool {
			return cp.filtered[i].UsageCount > cp.filtered[j].UsageCount
		})

		cp.selected = 0
		return
	}

	query = strings.ToLower(query)
	results := make([]*SearchResult, 0)

	for _, cmd := range cp.commands {
		if !cmd.Enabled {
			continue
		}

		score := 0
		matchType := ""

		// Exact name match (highest score)
		if strings.ToLower(cmd.Name) == query {
			score += 100
			matchType = "exact_name"
		} else if strings.Contains(strings.ToLower(cmd.Name), query) {
			score += 50
			matchType = "partial_name"
		}

		// Command ID match
		if strings.Contains(strings.ToLower(cmd.ID), query) {
			score += 40
			if matchType == "" {
				matchType = "id"
			}
		}

		// Description match
		if strings.Contains(strings.ToLower(cmd.Description), query) {
			score += 20
			if matchType == "" {
				matchType = "description"
			}
		}

		// Keyword match
		for _, keyword := range cmd.Keywords {
			if strings.Contains(strings.ToLower(keyword), query) {
				score += 30
				if matchType == "" {
					matchType = "keyword"
				}
				break
			}
		}

		// Category match
		if strings.Contains(strings.ToLower(cmd.Category), query) {
			score += 15
			if matchType == "" {
				matchType = "category"
			}
		}

		// Boost score based on usage
		score += cmd.UsageCount * 2

		// Boost recently used commands
		if time.Since(cmd.LastUsed) < time.Hour {
			score += 10
		}

		if score > 0 {
			results = append(results, &SearchResult{
				Command:   cmd,
				Score:     score,
				MatchType: matchType,
			})
		}
	}

	// Sort by score (descending)
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	// Extract commands from results
	cp.filtered = make([]*Command, len(results))
	for i, result := range results {
		cp.filtered[i] = result.Command
	}

	cp.selected = 0
}

// Message types for Bubble Tea integration
type CommandPaletteToggleMsg struct {
	Visible bool
}

type AgentMentionMsg struct {
	AgentID string
}

type ViewSwitchMsg struct {
	View string
}

type QuickOpenMsg struct {
	Type string // "file", "commission", etc.
}

type NewCommissionMsg struct{}

type FocusSearchMsg struct{}

type ToggleDebugMsg struct{}

type StartProfilingMsg struct{}

type ShowHelpMsg struct {
	Type string // "shortcuts", "general", etc.
}

type EscapeMsg struct{}

type ShortcutExecuteMsg struct {
	ShortcutID string
	Command    string
}
