// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package commands

import (
	"strings"
	"sync"
)

// CompletionProvider handles tab completion for commands and arguments
type CompletionProvider interface {
	GetCompletions(input string, cursor int) []Completion
	AddCustomCompletion(pattern string, suggestions []string)
	UpdateContext(ctx CompletionContext)
}

// Completion represents a single completion suggestion
type Completion struct {
	Text        string // Text to insert
	Display     string // Text to show in menu
	Description string // Optional description
	Icon        string // Optional icon/prefix
}

// CompletionContext provides context for smart completions
type CompletionContext struct {
	CurrentAgent    string
	CurrentCommand  string
	WorkingDir      string
	RecentFiles     []string
	AvailableAgents []string
	AvailableTools  []string
}

// DefaultCompletionProvider implements CompletionProvider
type DefaultCompletionProvider struct {
	mu              sync.RWMutex
	context         CompletionContext
	customPatterns  map[string][]string
	commandRegistry map[string]CommandInfo
	recentItems     *RecentItemsCache
}

// CommandInfo stores information about a command
type CommandInfo struct {
	Name        string
	Description string
	Arguments   []ArgumentInfo
	Aliases     []string
}

// ArgumentInfo describes a command argument
type ArgumentInfo struct {
	Name        string
	Type        string // "agent", "file", "string", "choice"
	Required    bool
	Choices     []string
	Description string
}

// NewCompletionProvider creates a new completion provider
func NewCompletionProvider() *DefaultCompletionProvider {
	provider := &DefaultCompletionProvider{
		customPatterns:  make(map[string][]string),
		commandRegistry: make(map[string]CommandInfo),
		recentItems:     NewRecentItemsCache(100),
	}

	// Register default commands
	provider.registerDefaultCommands()

	return provider
}

// GetCompletions returns completion suggestions for the given input
func (p *DefaultCompletionProvider) GetCompletions(input string, cursor int) []Completion {
	p.mu.RLock()
	defer p.mu.RUnlock()

	// Extract the relevant portion for completion
	prefix := input[:cursor]

	// Determine completion type based on prefix
	switch {
	case strings.HasPrefix(prefix, "/"):
		return p.getCommandCompletions(prefix)
	case strings.HasPrefix(prefix, "@"):
		return p.getAgentCompletions(prefix)
	case strings.Contains(prefix, " "):
		return p.getArgumentCompletions(prefix)
	default:
		return p.getGeneralCompletions(prefix)
	}
}

// getCommandCompletions returns completions for commands starting with /
func (p *DefaultCompletionProvider) getCommandCompletions(prefix string) []Completion {
	var completions []Completion
	search := strings.ToLower(prefix[1:]) // Remove leading /

	for name, info := range p.commandRegistry {
		if strings.HasPrefix(strings.ToLower(name), search) {
			completions = append(completions, Completion{
				Text:        "/" + name,
				Display:     "/" + name,
				Description: info.Description,
				Icon:        "📋",
			})
		}

		// Also check aliases
		for _, alias := range info.Aliases {
			if strings.HasPrefix(strings.ToLower(alias), search) {
				completions = append(completions, Completion{
					Text:        "/" + alias,
					Display:     "/" + alias + " → " + name,
					Description: info.Description,
					Icon:        "🔗",
				})
			}
		}
	}

	// Add recent commands
	recentCommands := p.recentItems.GetRecent("command", search)
	for _, cmd := range recentCommands {
		if strings.HasPrefix(cmd, "/") {
			completions = append(completions, Completion{
				Text:        cmd,
				Display:     cmd + " (recent)",
				Description: "Recently used command",
				Icon:        "🕐",
			})
		}
	}

	return completions
}

// getAgentCompletions returns completions for agents starting with @
func (p *DefaultCompletionProvider) getAgentCompletions(prefix string) []Completion {
	var completions []Completion
	search := strings.ToLower(prefix[1:]) // Remove leading @

	for _, agent := range p.context.AvailableAgents {
		if strings.HasPrefix(strings.ToLower(agent), search) {
			completions = append(completions, Completion{
				Text:        "@" + agent,
				Display:     "@" + agent,
				Description: "AI Agent",
				Icon:        "🤖",
			})
		}
	}

	// Add current agent if matches
	if p.context.CurrentAgent != "" && strings.HasPrefix(strings.ToLower(p.context.CurrentAgent), search) {
		completions = append(completions, Completion{
			Text:        "@" + p.context.CurrentAgent,
			Display:     "@" + p.context.CurrentAgent + " (current)",
			Description: "Currently selected agent",
			Icon:        "⭐",
		})
	}

	return completions
}

// getArgumentCompletions returns completions for command arguments
func (p *DefaultCompletionProvider) getArgumentCompletions(input string) []Completion {
	parts := strings.Fields(input)
	if len(parts) < 2 {
		return nil
	}

	// Get the command (first part)
	cmd := parts[0]
	if strings.HasPrefix(cmd, "/") {
		cmd = cmd[1:]
	}

	// Find command info
	cmdInfo, exists := p.commandRegistry[cmd]
	if !exists {
		// Check aliases
		for _, info := range p.commandRegistry {
			for _, alias := range info.Aliases {
				if alias == cmd {
					cmdInfo = info
					exists = true
					break
				}
			}
			if exists {
				break
			}
		}
	}

	if !exists {
		return nil
	}

	// Determine which argument we're completing
	argIndex := len(parts) - 2
	if argIndex >= len(cmdInfo.Arguments) {
		return nil
	}

	argInfo := cmdInfo.Arguments[argIndex]
	lastPart := parts[len(parts)-1]

	// Generate completions based on argument type
	switch argInfo.Type {
	case "agent":
		return p.getAgentCompletions("@" + lastPart)
	case "file":
		return p.getFileCompletions(lastPart)
	case "choice":
		return p.getChoiceCompletions(lastPart, argInfo.Choices)
	default:
		return nil
	}
}

// getFileCompletions returns file path completions
func (p *DefaultCompletionProvider) getFileCompletions(prefix string) []Completion {
	// This will be implemented in filepath_completion.go
	var completions []Completion

	// For now, add recent files
	for _, file := range p.context.RecentFiles {
		if strings.HasPrefix(file, prefix) {
			completions = append(completions, Completion{
				Text:        file,
				Display:     file,
				Description: "Recent file",
				Icon:        "📄",
			})
		}
	}

	return completions
}

// getChoiceCompletions returns completions from a fixed set of choices
func (p *DefaultCompletionProvider) getChoiceCompletions(prefix string, choices []string) []Completion {
	var completions []Completion
	search := strings.ToLower(prefix)

	for _, choice := range choices {
		if strings.HasPrefix(strings.ToLower(choice), search) {
			completions = append(completions, Completion{
				Text:        choice,
				Display:     choice,
				Description: "",
				Icon:        "▪",
			})
		}
	}

	return completions
}

// getGeneralCompletions returns general completions when no prefix is detected
func (p *DefaultCompletionProvider) getGeneralCompletions(prefix string) []Completion {
	var completions []Completion

	// Suggest command prefix
	completions = append(completions, Completion{
		Text:        "/",
		Display:     "/ - Commands",
		Description: "Start typing a command",
		Icon:        "📋",
	})

	// Suggest agent prefix
	completions = append(completions, Completion{
		Text:        "@",
		Display:     "@ - Agents",
		Description: "Select an AI agent",
		Icon:        "🤖",
	})

	return completions
}

// AddCustomCompletion adds custom completion patterns
func (p *DefaultCompletionProvider) AddCustomCompletion(pattern string, suggestions []string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.customPatterns[pattern] = suggestions
}

// UpdateContext updates the completion context
func (p *DefaultCompletionProvider) UpdateContext(ctx CompletionContext) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.context = ctx
}

// registerDefaultCommands registers the built-in commands
func (p *DefaultCompletionProvider) registerDefaultCommands() {
	// Agent commands
	p.commandRegistry["agent"] = CommandInfo{
		Name:        "agent",
		Description: "Manage AI agents",
		Arguments: []ArgumentInfo{
			{Name: "action", Type: "choice", Required: true, Choices: []string{"list", "select", "info"}},
			{Name: "agent", Type: "agent", Required: false},
		},
	}

	// Campaign commands
	p.commandRegistry["campaign"] = CommandInfo{
		Name:        "campaign",
		Description: "View campaign information",
		Arguments: []ArgumentInfo{
			{Name: "action", Type: "choice", Required: true, Choices: []string{"info", "list"}},
		},
	}

	// Prompt commands
	p.commandRegistry["prompt"] = CommandInfo{
		Name:        "prompt",
		Description: "Manage layered prompts",
		Arguments: []ArgumentInfo{
			{Name: "action", Type: "choice", Required: true, Choices: []string{"list", "set", "delete", "show"}},
			{Name: "layer", Type: "choice", Required: false, Choices: []string{"base", "context", "task", "style", "constraints", "examples"}},
		},
	}

	// Tool commands
	p.commandRegistry["tool"] = CommandInfo{
		Name:        "tool",
		Description: "Manage and execute tools",
		Arguments: []ArgumentInfo{
			{Name: "action", Type: "choice", Required: true, Choices: []string{"list", "search", "info", "execute"}},
			{Name: "tool", Type: "string", Required: false},
		},
	}

	// Help command
	p.commandRegistry["help"] = CommandInfo{
		Name:        "help",
		Description: "Show help information",
		Arguments:   []ArgumentInfo{},
		Aliases:     []string{"?", "h"},
	}

	// Clear command
	p.commandRegistry["clear"] = CommandInfo{
		Name:        "clear",
		Description: "Clear the chat screen",
		Arguments:   []ArgumentInfo{},
		Aliases:     []string{"cls"},
	}

	// Exit command
	p.commandRegistry["exit"] = CommandInfo{
		Name:        "exit",
		Description: "Exit the chat",
		Arguments:   []ArgumentInfo{},
		Aliases:     []string{"quit", "q"},
	}
}

// RecentItemsCache maintains a cache of recently used items
type RecentItemsCache struct {
	mu       sync.RWMutex
	items    map[string][]string
	maxItems int
}

// NewRecentItemsCache creates a new recent items cache
func NewRecentItemsCache(maxItems int) *RecentItemsCache {
	return &RecentItemsCache{
		items:    make(map[string][]string),
		maxItems: maxItems,
	}
}

// Add adds an item to the recent cache
func (r *RecentItemsCache) Add(category, item string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Initialize category if needed
	if r.items[category] == nil {
		r.items[category] = make([]string, 0, r.maxItems)
	}

	// Remove if already exists
	items := r.items[category]
	for i, existing := range items {
		if existing == item {
			items = append(items[:i], items[i+1:]...)
			break
		}
	}

	// Add to front
	r.items[category] = append([]string{item}, items...)

	// Trim to max size
	if len(r.items[category]) > r.maxItems {
		r.items[category] = r.items[category][:r.maxItems]
	}
}

// GetRecent returns recent items matching the prefix
func (r *RecentItemsCache) GetRecent(category, prefix string) []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	items, exists := r.items[category]
	if !exists {
		return nil
	}

	var matches []string
	for _, item := range items {
		if strings.HasPrefix(item, prefix) {
			matches = append(matches, item)
		}
	}

	return matches
}

// FuzzyMatch performs fuzzy string matching
func FuzzyMatch(pattern, text string) bool {
	pattern = strings.ToLower(pattern)
	text = strings.ToLower(text)

	patternIdx := 0
	for _, ch := range text {
		if patternIdx < len(pattern) && ch == rune(pattern[patternIdx]) {
			patternIdx++
		}
	}

	return patternIdx == len(pattern)
}

// RegisterCommand registers a new command with the completion provider
func (p *DefaultCompletionProvider) RegisterCommand(cmd CommandInfo) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.commandRegistry[cmd.Name] = cmd
}
