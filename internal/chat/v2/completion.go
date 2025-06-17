// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package v2

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/guild-ventures/guild-core/internal/chat/v2/common"
	"github.com/guild-ventures/guild-core/pkg/config"
	"github.com/guild-ventures/guild-core/pkg/registry"
)

// CompletionEngine provides intelligent command and agent auto-completion for V2
type CompletionEngine struct {
	guildConfig *config.GuildConfig
	projectRoot string
	commands    map[string]Command
	taskIDs     []string                   // Cache of task IDs for completion
	registry    registry.ComponentRegistry // Access to kanban for task completion
}

// NewCompletionEngine creates a new completion engine with full functionality
func NewCompletionEngine(guildConfig *config.GuildConfig, projectRoot string) *CompletionEngine {
	engine := &CompletionEngine{
		guildConfig: guildConfig,
		projectRoot: projectRoot,
		commands:    make(map[string]Command),
	}

	// Register built-in commands with medieval theming
	engine.registerCommands()
	return engine
}

// registerCommands registers all available commands for completion
func (ce *CompletionEngine) registerCommands() {
	commands := []Command{
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
	}

	for _, cmd := range commands {
		ce.commands[cmd.Name] = cmd
	}
}

// Complete provides intelligent completion suggestions
func (ce *CompletionEngine) Complete(input string, cursorPos int) []common.CompletionResult {
	var results []common.CompletionResult

	// Handle different completion types based on input context
	if strings.HasPrefix(input, "/") {
		// Command completion
		results = append(results, ce.completeCommands(input)...)
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

	// If no results, provide helpful suggestions based on context
	if len(results) == 0 {
		results = ce.getHelpfulSuggestions(input)
	}

	return results
}

// completeCommands suggests command completions with smart sorting
func (ce *CompletionEngine) completeCommands(input string) []common.CompletionResult {
	var results []common.CompletionResult

	// First try exact prefix match
	for cmdName, cmd := range ce.commands {
		if strings.HasPrefix(cmdName, input) {
			results = append(results, common.CompletionResult{
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
		for cmdName, cmd := range ce.commands {
			if fuzzyMatch(cmdName, input) {
				results = append(results, common.CompletionResult{
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
func (ce *CompletionEngine) completeAgents(input string) []common.CompletionResult {
	var results []common.CompletionResult

	// Add @all for broadcast
	if fuzzyMatch("@all", input) {
		results = append(results, common.CompletionResult{
			Content: "@all",
			AgentID: "system",
			Metadata: map[string]string{
				"type":        "agent",
				"description": "Broadcast to all agents",
			},
		})
	}

	// Add configured agents
	if ce.guildConfig != nil {
		for _, agent := range ce.guildConfig.Agents {
			agentMention := "@" + agent.ID
			if fuzzyMatch(agentMention, input) {
				results = append(results, common.CompletionResult{
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
func (ce *CompletionEngine) completeArguments(input string) []common.CompletionResult {
	var results []common.CompletionResult

	// Common argument patterns
	args := []common.CompletionResult{
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
func (ce *CompletionEngine) completeFilePaths(input string) []common.CompletionResult {
	var results []common.CompletionResult

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

			results = append(results, common.CompletionResult{
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
func (ce *CompletionEngine) completeTaskIDs(input string) []common.CompletionResult {
	var results []common.CompletionResult

	// Use cached task IDs
	for _, taskID := range ce.taskIDs {
		if fuzzyMatch(taskID, input) {
			results = append(results, common.CompletionResult{
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

// getHelpfulSuggestions provides context-aware suggestions when no matches found
func (ce *CompletionEngine) getHelpfulSuggestions(input string) []common.CompletionResult {
	var results []common.CompletionResult

	// Suggest common starting patterns
	if input == "" {
		results = append(results,
			common.CompletionResult{Content: "/help", AgentID: "system", Metadata: map[string]string{"type": "suggestion", "description": "Show available commands"}},
			common.CompletionResult{Content: "@", AgentID: "system", Metadata: map[string]string{"type": "suggestion", "description": "Mention an agent"}},
			common.CompletionResult{Content: "/prompt", AgentID: "system", Metadata: map[string]string{"type": "suggestion", "description": "Manage prompts"}},
		)
	} else if strings.HasPrefix(input, "@") && len(input) == 1 {
		// Show all available agents when just @ is typed
		results = append(results, common.CompletionResult{
			Content: "@all",
			AgentID: "system",
			Metadata: map[string]string{
				"type":        "agent",
				"description": "Broadcast to all agents",
			},
		})

		if ce.guildConfig != nil {
			for _, agent := range ce.guildConfig.Agents {
				results = append(results, common.CompletionResult{
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
func (ce *CompletionEngine) getCommonGuildPaths(input string) []common.CompletionResult {
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

	var results []common.CompletionResult
	for _, path := range commonPaths {
		if fuzzyMatch(path, input) {
			results = append(results, common.CompletionResult{
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
	ce.taskIDs = taskIDs
}

// SetRegistry sets the component registry for advanced completions
func (ce *CompletionEngine) SetRegistry(reg registry.ComponentRegistry) {
	ce.registry = reg
}

// GetAllCommands returns all registered command names for testing/debugging
func (ce *CompletionEngine) GetAllCommands() []string {
	var commands []string
	for name := range ce.commands {
		commands = append(commands, name)
	}
	return commands
}

// GetAllAgents returns all registered agent IDs for testing/debugging
func (ce *CompletionEngine) GetAllAgents() []string {
	var agents []string
	if ce.guildConfig != nil {
		for _, agent := range ce.guildConfig.Agents {
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