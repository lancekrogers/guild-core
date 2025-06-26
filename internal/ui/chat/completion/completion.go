// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package completion

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/guild-ventures/guild-core/internal/ui/chat/messages"
	"github.com/guild-ventures/guild-core/pkg/agent"
	"github.com/guild-ventures/guild-core/pkg/config"
	"github.com/guild-ventures/guild-core/pkg/registry"
	"github.com/guild-ventures/guild-core/pkg/search"
	"github.com/guild-ventures/guild-core/pkg/suggestions"
)

// CompletionEngine provides intelligent command and agent auto-completion for core
type CompletionEngine struct {
	GuildConfig *config.GuildConfig
	ProjectRoot string
	Commands    map[string]messages.Command
	TaskIDs     []string                   // Cache of task IDs for completion
	registry    registry.ComponentRegistry // Access to kanban for task completion

	// NEW: Command processor integration
	CommandProcessor CommandProcessorInterface // Interface to get commands from command processor

	// NEW: File search integration
	FuzzyFinder *search.FuzzyFinder // For @file completions

	// NEW: Suggestion system integration
	SuggestionManager    suggestions.SuggestionManager
	ChatHandler          *agent.ChatSuggestionHandler
	ConversationHist     []suggestions.ChatMessage // Context cache
	LastSuggestionUpdate time.Time                 // Performance optimization
}

// CommandProcessorInterface defines the interface for getting commands from command processor
type CommandProcessorInterface interface {
	GetAvailableCommands() []messages.Command
}

// NewCompletionEngine creates a new completion engine with full functionality
func NewCompletionEngine(guildConfig *config.GuildConfig, ProjectRoot string) *CompletionEngine {
	engine := &CompletionEngine{
		GuildConfig:      guildConfig,
		ProjectRoot:      ProjectRoot,
		Commands:         make(map[string]messages.Command),
		ConversationHist: make([]suggestions.ChatMessage, 0),
	}

	// Initialize fuzzy finder for file completions
	if ProjectRoot != "" {
		config := search.FuzzyFinderConfig{
			WorkingDir:      ProjectRoot,
			ExcludePatterns: search.DefaultExcludePatterns(),
			MaxResults:      10,
			IndexTimeout:    5 * time.Second,
		}
		engine.FuzzyFinder = search.NewFuzzyFinder(config)
	}

	// Register built-in commands with medieval theming
	engine.registerCommands()
	return engine
}

// NewCompletionEngineWithSuggestions creates a completion engine with suggestion system integration
func NewCompletionEngineWithSuggestions(guildConfig *config.GuildConfig, ProjectRoot string,
	SuggestionManager suggestions.SuggestionManager, ChatHandler *agent.ChatSuggestionHandler,
) *CompletionEngine {
	engine := &CompletionEngine{
		GuildConfig:       guildConfig,
		ProjectRoot:       ProjectRoot,
		Commands:          make(map[string]messages.Command),
		SuggestionManager: SuggestionManager,
		ChatHandler:       ChatHandler,
		ConversationHist:  make([]suggestions.ChatMessage, 0),
	}

	// Register built-in commands with medieval theming
	engine.registerCommands()
	return engine
}

// registerCommands registers all available commands for completion
func (ce *CompletionEngine) registerCommands() {
	// If we have a command processor, get commands from it
	if ce.CommandProcessor != nil {
		commands := ce.CommandProcessor.GetAvailableCommands()
		for _, cmd := range commands {
			ce.Commands[cmd.Name] = cmd
		}
		return
	}

	// Fallback: use static commands if no command processor available
	commands := []messages.Command{
		{Name: "/help", Description: "Show available commands"},
		{Name: "/status", Description: "Show campaign status"},
		{Name: "/agents", Description: "List available agents"},
		{Name: "/prompt", Description: "Manage layered prompts"},
		{Name: "/tools", Description: "Manage Guild tools"},
		{Name: "/tool", Description: "Execute a tool directly"},
		{Name: "/test", Description: "Test rich content features"},
		{Name: "/export", Description: "Export chat session"},
		{Name: "/save", Description: "Quick save as markdown"},
		{Name: "/template", Description: "Template operations"},
		{Name: "/templates", Description: "Template management interface"},
		{Name: "/image", Description: "Display image with ASCII preview"},
		{Name: "/mermaid", Description: "Show Mermaid diagram help"},
		{Name: "/code", Description: "Code rendering features"},
		{Name: "/search", Description: "Search message history"},
		{Name: "/session", Description: "Session management"},
		{Name: "/exit", Description: "Exit Guild Chat"},
		{Name: "/quit", Description: "Exit Guild Chat"},
		{Name: "/clear", Description: "Clear chat history"},
		{Name: "/guilds", Description: "List all available guilds"},
		{Name: "/guild", Description: "Show current guild details or switch guild"},
		{Name: "/configrefresh", Description: "Reload configurations"},
		{Name: "/vim", Description: "Toggle vim mode for input"},
	}

	for _, cmd := range commands {
		ce.Commands[cmd.Name] = cmd
	}
}

// Complete provides intelligent completion suggestions enhanced with context-aware suggestions
func (ce *CompletionEngine) Complete(input string, cursorPos int) []CompletionResult {
	var results []CompletionResult

	// For empty input, provide helpful suggestions immediately
	if strings.TrimSpace(input) == "" {
		return ce.getHelpfulSuggestions(input)
	}

	// Get traditional completions first
	traditionalResults := ce.getTraditionalCompletions(input, cursorPos)
	results = append(results, traditionalResults...)

	// NEW: Add context-aware suggestions if suggestion system is available
	if ce.hasSuggestionSystem() && len(strings.TrimSpace(input)) > 2 {
		suggestionResults := ce.getSuggestions(input)
		results = append(results, suggestionResults...)
	}

	// Merge, rank, and deduplicate results
	results = ce.mergeAndRankResults(results, input)

	// If still no results, provide helpful suggestions based on context
	if len(results) == 0 {
		results = ce.getHelpfulSuggestions(input)
	}

	return results
}

// getTraditionalCompletions handles the original completion logic
func (ce *CompletionEngine) getTraditionalCompletions(input string, cursorPos int) []CompletionResult {
	var results []CompletionResult

	// Handle different completion types based on input context
	if strings.HasPrefix(input, "/") {
		// Command completion
		results = append(results, ce.completeCommands(input)...)
	} else if strings.HasPrefix(input, "@") && strings.Contains(input, ":") {
		// @file:path/to/file completion (like Claude Code)
		results = append(results, ce.completeFileSelection(input)...)
	} else if strings.HasPrefix(input, "@") {
		// Agent mention completion
		results = append(results, ce.completeAgents(input)...)
	} else if strings.Contains(input, "--") {
		// Command argument completion
		results = append(results, ce.completeArguments(input)...)
	} else if strings.Contains(strings.ToLower(input), "task") {
		// Task ID completion when context suggests tasks
		results = append(results, ce.completeTaskIDs(input)...)
	} else {
		// File path completion or general text
		results = append(results, ce.completeFilePaths(input)...)
	}

	return results
}

// completeCommands suggests command completions with smart sorting
func (ce *CompletionEngine) completeCommands(input string) []CompletionResult {
	var results []CompletionResult

	// First try exact prefix match
	for cmdName, cmd := range ce.Commands {
		if strings.HasPrefix(cmdName, input) {
			results = append(results, CompletionResult{
				Content: cmdName,
				AgentID: "system",
				Metadata: map[string]string{
					"type":        "command",
					"description": cmd.Description,
				},
			})
		}
	}

	// If no exact matches, try fuzzy matching
	if len(results) == 0 {
		for cmdName, cmd := range ce.Commands {
			if fuzzyMatch(cmdName, input) {
				results = append(results, CompletionResult{
					Content: cmdName,
					AgentID: "system",
					Metadata: map[string]string{
						"type":        "command",
						"description": cmd.Description,
					},
				})
			}
		}
	}

	// Sort by relevance (exact matches first, then by length)
	sort.Slice(results, func(i, j int) bool {
		// Exact prefix matches come first
		iExact := strings.HasPrefix(results[i].Content, input)
		jExact := strings.HasPrefix(results[j].Content, input)
		if iExact != jExact {
			return iExact
		}
		// Then sort by length (shorter = more relevant)
		return len(results[i].Content) < len(results[j].Content)
	})

	return results
}

// completeAgents suggests agent completions
func (ce *CompletionEngine) completeAgents(input string) []CompletionResult {
	var results []CompletionResult

	// Add @all for broadcast
	if fuzzyMatch("@all", input) {
		results = append(results, CompletionResult{
			Content: "@all",
			AgentID: "system",
			Metadata: map[string]string{
				"type":        "agent",
				"description": "Broadcast to all agents",
			},
		})
	}

	// Add configured agents
	if ce.GuildConfig != nil {
		for _, agent := range ce.GuildConfig.Agents {
			agentMention := "@" + agent.ID
			if fuzzyMatch(agentMention, input) {
				results = append(results, CompletionResult{
					Content: agentMention,
					AgentID: agent.ID,
					Metadata: map[string]string{
						"type":        "agent",
						"description": fmt.Sprintf("%s - %s", agent.Name, strings.Join(agent.Capabilities, ", ")),
					},
				})
			}
		}
	}

	return results
}

// completeArguments suggests command argument completions
func (ce *CompletionEngine) completeArguments(input string) []CompletionResult {
	var results []CompletionResult

	// Common argument patterns
	args := []CompletionResult{
		{Content: "--path", AgentID: "system", Metadata: map[string]string{"type": "argument", "description": "File or directory path"}},
		{Content: "--layer", AgentID: "system", Metadata: map[string]string{"type": "argument", "description": "Prompt layer name"}},
		{Content: "--text", AgentID: "system", Metadata: map[string]string{"type": "argument", "description": "Text content"}},
		{Content: "--timeout", AgentID: "system", Metadata: map[string]string{"type": "argument", "description": "Timeout in seconds"}},
		{Content: "--command", AgentID: "system", Metadata: map[string]string{"type": "argument", "description": "Shell command to execute"}},
	}

	for _, arg := range args {
		if fuzzyMatch(arg.Content, strings.ToLower(input)) {
			results = append(results, arg)
		}
	}

	return results
}

// completeFilePaths suggests file path completions with real filesystem scanning
func (ce *CompletionEngine) completeFilePaths(input string) []CompletionResult {
	var results []CompletionResult

	// Determine the directory to scan based on input
	var dirToScan string
	var searchPrefix string

	if strings.Contains(input, "/") {
		// Input contains path separator - extract directory and filename prefix
		lastSlash := strings.LastIndex(input, "/")
		dirToScan = input[:lastSlash+1]
		searchPrefix = input[lastSlash+1:]
	} else {
		// No path separator - scan current directory
		dirToScan = "."
		searchPrefix = input
	}

	// Read directory contents
	entries, err := os.ReadDir(dirToScan)
	if err != nil {
		// Fallback to common Guild paths if directory read fails
		return ce.getCommonGuildPaths(input)
	}

	// Add matching entries
	for _, entry := range entries {
		name := entry.Name()

		// Skip hidden files unless explicitly searching for them
		if strings.HasPrefix(name, ".") && !strings.HasPrefix(searchPrefix, ".") {
			continue
		}

		// Check if name matches search prefix
		if fuzzyMatch(name, searchPrefix) {
			fullPath := filepath.Join(dirToScan, name)
			displayPath := fullPath
			if dirToScan == "." {
				displayPath = name
			}

			fileType := "file"
			if entry.IsDir() {
				fileType = "directory"
				displayPath += "/"
			}

			results = append(results, CompletionResult{
				Content: displayPath,
				AgentID: "system",
				Metadata: map[string]string{
					"type":        "file",
					"description": fmt.Sprintf("%s (%s)", name, fileType),
				},
			})
		}
	}

	// Sort by exact match first, then alphabetically
	sort.Slice(results, func(i, j int) bool {
		iExact := strings.HasPrefix(results[i].Content, input)
		jExact := strings.HasPrefix(results[j].Content, input)
		if iExact != jExact {
			return iExact
		}
		return results[i].Content < results[j].Content
	})

	return results
}

// completeTaskIDs suggests task ID completions
func (ce *CompletionEngine) completeTaskIDs(input string) []CompletionResult {
	var results []CompletionResult

	// Use cached task IDs
	for _, taskID := range ce.TaskIDs {
		if fuzzyMatch(taskID, input) {
			results = append(results, CompletionResult{
				Content: taskID,
				AgentID: "system",
				Metadata: map[string]string{
					"type":        "task",
					"description": "Task ID",
				},
			})
		}
	}

	return results
}

// completeFileSelection handles @file: completions like Claude Code
func (ce *CompletionEngine) completeFileSelection(input string) []CompletionResult {
	var results []CompletionResult

	if ce.FuzzyFinder == nil {
		return results
	}

	// Parse @file:pattern format
	if !strings.Contains(input, ":") {
		// Show @file: suggestion if just typing @file
		if fuzzyMatch("@file:", input) {
			results = append(results, CompletionResult{
				Content: "@file:",
				AgentID: "system",
				Metadata: map[string]string{
					"type":        "file",
					"description": "Select file to include in message",
				},
			})
		}
		return results
	}

	// Extract the file pattern after @file:
	parts := strings.SplitN(input, ":", 2)
	if len(parts) != 2 {
		return results
	}

	filePattern := parts[1]

	// Use fuzzy finder to search for files
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	fileResults, err := ce.FuzzyFinder.Search(ctx, filePattern)
	if err != nil {
		return results
	}

	// Convert file results to completion results
	for _, fileResult := range fileResults {
		// Format as @file:path/to/file
		content := "@file:" + fileResult.Path
		description := fmt.Sprintf("%s (%s)", fileResult.Name, fileResult.Path)
		if fileResult.IsDirectory {
			description += " [directory]"
		}

		results = append(results, CompletionResult{
			Content: content,
			AgentID: "system",
			Metadata: map[string]string{
				"type":        "file",
				"description": description,
				"file_path":   fileResult.AbsPath,
				"file_size":   fmt.Sprintf("%d", fileResult.Size),
			},
		})
	}

	return results
}

// getHelpfulSuggestions provides context-aware suggestions when no matches found
func (ce *CompletionEngine) getHelpfulSuggestions(input string) []CompletionResult {
	var results []CompletionResult

	// Suggest common starting patterns
	if input == "" {
		results = append(results,
			CompletionResult{Content: "/help", AgentID: "system", Metadata: map[string]string{"type": "suggestion", "description": "Show available commands"}},
			CompletionResult{Content: "@", AgentID: "system", Metadata: map[string]string{"type": "suggestion", "description": "Mention an agent"}},
			CompletionResult{Content: "@file:", AgentID: "system", Metadata: map[string]string{"type": "suggestion", "description": "Include file in message"}},
			CompletionResult{Content: "/prompt", AgentID: "system", Metadata: map[string]string{"type": "suggestion", "description": "Manage prompts"}},
		)
	} else if strings.HasPrefix(input, "@") && len(input) == 1 {
		// Show all available agents and @file: when just @ is typed
		results = append(results, CompletionResult{
			Content: "@all",
			AgentID: "system",
			Metadata: map[string]string{
				"type":        "agent",
				"description": "Broadcast to all agents",
			},
		})

		results = append(results, CompletionResult{
			Content: "@file:",
			AgentID: "system",
			Metadata: map[string]string{
				"type":        "file",
				"description": "Include file in message",
			},
		})

		if ce.GuildConfig != nil {
			for _, agent := range ce.GuildConfig.Agents {
				results = append(results, CompletionResult{
					Content: "@" + agent.ID,
					AgentID: agent.ID,
					Metadata: map[string]string{
						"type":        "agent",
						"description": agent.Name,
					},
				})
			}
		}
	}

	return results
}

// getCommonGuildPaths returns common Guild project paths as fallback
func (ce *CompletionEngine) getCommonGuildPaths(input string) []CompletionResult {
	commonPaths := []string{
		".guild/",
		".guild/commissions/",
		".guild/campaigns/",
		".guild/archives/",
		"pkg/",
		"cmd/",
		"internal/",
		"docs/",
		"examples/",
	}

	var results []CompletionResult
	for _, path := range commonPaths {
		if fuzzyMatch(path, input) {
			results = append(results, CompletionResult{
				Content: path,
				AgentID: "system",
				Metadata: map[string]string{
					"type":        "file",
					"description": "Common Guild path",
				},
			})
		}
	}

	return results
}

// UpdateTaskCache updates the cached task IDs for completion
func (ce *CompletionEngine) UpdateTaskCache(taskIDs []string) {
	ce.TaskIDs = taskIDs
}

// SetRegistry sets the component registry for advanced completions
func (ce *CompletionEngine) SetRegistry(reg registry.ComponentRegistry) {
	ce.registry = reg
}

// GetAllCommands returns all registered command names for testing/debugging
func (ce *CompletionEngine) GetAllCommands() []string {
	var commands []string
	for name := range ce.Commands {
		commands = append(commands, name)
	}
	return commands
}

// GetAllAgents returns all registered agent IDs for testing/debugging
func (ce *CompletionEngine) GetAllAgents() []string {
	var agents []string
	if ce.GuildConfig != nil {
		for _, agent := range ce.GuildConfig.Agents {
			agents = append(agents, "@"+agent.ID)
		}
	}
	return agents
}

// fuzzyMatch performs a simple fuzzy matching algorithm
func fuzzyMatch(text, pattern string) bool {
	if pattern == "" {
		return true
	}

	text = strings.ToLower(text)
	pattern = strings.ToLower(pattern)

	// Simple contains match for now
	// TODO: Implement more sophisticated fuzzy matching
	return strings.Contains(text, pattern)
}

// =====================================================
// NEW: Suggestion System Integration Methods
// =====================================================

// hasSuggestionSystem checks if the suggestion system is available
func (ce *CompletionEngine) hasSuggestionSystem() bool {
	return ce.SuggestionManager != nil && ce.ChatHandler != nil
}

// getSuggestions gets context-aware suggestions from the suggestion system
func (ce *CompletionEngine) getSuggestions(input string) []CompletionResult {
	if !ce.hasSuggestionSystem() {
		return []CompletionResult{}
	}

	// Performance optimization: avoid too frequent suggestion requests
	now := time.Now()
	if now.Sub(ce.LastSuggestionUpdate) < 300*time.Millisecond {
		return []CompletionResult{}
	}
	ce.LastSuggestionUpdate = now

	// Build suggestion request
	request := agent.SuggestionRequest{
		Message:        input,
		MaxSuggestions: 3, // Limit for performance in real-time
		MinConfidence:  0.5,
		Filter: &suggestions.SuggestionFilter{
			MaxResults: 3,
		},
	}

	// Get suggestions with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	response, err := ce.ChatHandler.GetSuggestions(ctx, request)
	if err != nil {
		// Graceful fallback - don't block completions if suggestions fail
		return []CompletionResult{}
	}

	// Convert suggestions to completion results
	return ce.convertSuggestionsToCompletions(response.Suggestions)
}

// convertSuggestionsToCompletions converts suggestion results to completion format
func (ce *CompletionEngine) convertSuggestionsToCompletions(suggestions []suggestions.Suggestion) []CompletionResult {
	results := make([]CompletionResult, 0, len(suggestions))

	for _, suggestion := range suggestions {
		// Use Display if available, otherwise Content
		content := suggestion.Content
		if suggestion.Display != "" {
			content = suggestion.Display
		}

		result := CompletionResult{
			Content: content,
			AgentID: "suggestion-system",
		}

		// Add metadata for suggestion tracking
		if result.Metadata == nil {
			result.Metadata = make(map[string]string)
		}
		result.Metadata["type"] = "suggestion"
		result.Metadata["suggestion_id"] = suggestion.ID
		result.Metadata["suggestion_source"] = suggestion.Source
		result.Metadata["description"] = suggestion.Description
		result.Metadata["icon"] = ce.getSuggestionIcon(suggestion.Type)
		result.Metadata["category"] = string(suggestion.Type)

		results = append(results, result)
	}

	return results
}

// getSuggestionIcon returns an appropriate icon for the suggestion type
func (ce *CompletionEngine) getSuggestionIcon(suggestionType suggestions.SuggestionType) string {
	switch suggestionType {
	case suggestions.SuggestionTypeCommand:
		return "⚡" // Command suggestions
	case suggestions.SuggestionTypeTool:
		return "🔧" // Tool suggestions
	case suggestions.SuggestionTypeTemplate:
		return "📝" // Template suggestions
	case suggestions.SuggestionTypeFollowUp:
		return "💡" // Follow-up suggestions
	case suggestions.SuggestionTypeCode:
		return "💻" // Code suggestions
	default:
		return "✨" // Generic suggestion
	}
}

// mergeAndRankResults combines traditional completions with suggestions and ranks them
func (ce *CompletionEngine) mergeAndRankResults(results []CompletionResult, input string) []CompletionResult {
	if len(results) == 0 {
		return results
	}

	// Deduplicate by content
	seen := make(map[string]bool)
	deduplicated := make([]CompletionResult, 0, len(results))

	for _, result := range results {
		key := strings.ToLower(result.Content)
		if !seen[key] {
			seen[key] = true
			deduplicated = append(deduplicated, result)
		}
	}

	// Sort by relevance: exact matches first, then alphabetically
	sort.Slice(deduplicated, func(i, j int) bool {
		a, b := deduplicated[i], deduplicated[j]

		// Exact matches go first
		inputLower := strings.ToLower(input)
		aExact := strings.HasPrefix(strings.ToLower(a.Content), inputLower)
		bExact := strings.HasPrefix(strings.ToLower(b.Content), inputLower)

		if aExact && !bExact {
			return true
		}
		if !aExact && bExact {
			return false
		}

		// Then sort alphabetically for consistent ordering
		return a.Content < b.Content
	})

	// Limit total results for performance
	if len(deduplicated) > 8 {
		deduplicated = deduplicated[:8]
	}

	return deduplicated
}

// UpdateConversationHistory updates the conversation context for better suggestions
func (ce *CompletionEngine) UpdateConversationHistory(messages []suggestions.ChatMessage) {
	if !ce.hasSuggestionSystem() {
		return
	}

	// Keep recent history for context (last 10 messages for performance)
	if len(messages) > 10 {
		ce.ConversationHist = messages[len(messages)-10:]
	} else {
		ce.ConversationHist = messages
	}
}

// buildProjectContext creates project context for suggestions
func (ce *CompletionEngine) buildProjectContext() suggestions.ProjectContext {
	return suggestions.ProjectContext{
		ProjectPath: ce.ProjectRoot,
		ProjectType: ce.detectProjectType(),
		Language:    ce.detectPrimaryLanguage(),
	}
}

// detectProjectType analyzes the project to determine its type
func (ce *CompletionEngine) detectProjectType() string {
	if ce.ProjectRoot == "" {
		return "unknown"
	}

	// Check for common project indicators
	if _, err := os.Stat(filepath.Join(ce.ProjectRoot, "go.mod")); err == nil {
		return "go-library"
	}
	if _, err := os.Stat(filepath.Join(ce.ProjectRoot, "package.json")); err == nil {
		return "javascript"
	}
	if _, err := os.Stat(filepath.Join(ce.ProjectRoot, "Cargo.toml")); err == nil {
		return "rust"
	}
	if _, err := os.Stat(filepath.Join(ce.ProjectRoot, "requirements.txt")); err == nil {
		return "python"
	}

	return "general"
}

// detectPrimaryLanguage analyzes the project to determine the primary language
func (ce *CompletionEngine) detectPrimaryLanguage() string {
	if ce.ProjectRoot == "" {
		return ""
	}

	// Simple language detection based on file extensions
	if _, err := os.Stat(filepath.Join(ce.ProjectRoot, "go.mod")); err == nil {
		return "go"
	}
	if _, err := os.Stat(filepath.Join(ce.ProjectRoot, "package.json")); err == nil {
		return "javascript"
	}
	if _, err := os.Stat(filepath.Join(ce.ProjectRoot, "Cargo.toml")); err == nil {
		return "rust"
	}
	if _, err := os.Stat(filepath.Join(ce.ProjectRoot, "requirements.txt")); err == nil {
		return "python"
	}

	return ""
}

// SetCommandProcessor sets the command processor to get live commands from
func (ce *CompletionEngine) SetCommandProcessor(processor CommandProcessorInterface) {
	ce.CommandProcessor = processor
	// Re-register commands to get latest from processor
	ce.registerCommands()
}

// SetEnhancedAgent configures the completion engine with an enhanced agent
func (ce *CompletionEngine) SetEnhancedAgent(agent agent.EnhancedGuildArtisan, handler *agent.ChatSuggestionHandler) {
	if agent != nil {
		ce.SuggestionManager = agent.GetSuggestionManager()
		ce.ChatHandler = handler
	}
}
