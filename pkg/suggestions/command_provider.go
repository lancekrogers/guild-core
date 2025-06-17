// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package suggestions

import (
	"context"
	"fmt"
	"strings"
)

// CommandSuggestionProvider suggests relevant commands based on context
type CommandSuggestionProvider struct {
	commands []CommandDefinition
}

// CommandDefinition defines a command that can be suggested
type CommandDefinition struct {
	Name        string
	Description string
	Category    string
	Keywords    []string
	Patterns    []string // Patterns that trigger this command
	Priority    int
}

// NewCommandSuggestionProvider creates a new command suggestion provider
func NewCommandSuggestionProvider() *CommandSuggestionProvider {
	return &CommandSuggestionProvider{
		commands: getDefaultCommands(),
	}
}

// GetSuggestions returns command suggestions based on context
func (p *CommandSuggestionProvider) GetSuggestions(ctx context.Context, context SuggestionContext) ([]Suggestion, error) {
	suggestions := make([]Suggestion, 0)
	
	// Analyze current message for command intent
	currentMsg := strings.ToLower(context.CurrentMessage)
	
	// Check recent conversation for context
	recentContext := p.getRecentContext(context.ConversationHistory, 3)
	
	for _, cmd := range p.commands {
		confidence := p.calculateCommandConfidence(cmd, currentMsg, recentContext, context)
		
		if confidence > 0.3 { // Threshold for showing suggestion
			suggestion := Suggestion{
				Type:        SuggestionTypeCommand,
				Content:     cmd.Name,
				Display:     fmt.Sprintf("/%s", cmd.Name),
				Description: cmd.Description,
				Confidence:  confidence,
				Priority:    cmd.Priority,
				Action: SuggestionAction{
					Type:   ActionTypeExecute,
					Target: cmd.Name,
				},
				Tags: append([]string{cmd.Category}, cmd.Keywords...),
			}
			
			suggestions = append(suggestions, suggestion)
		}
	}
	
	return suggestions, nil
}

// UpdateContext updates the provider's context (no-op for stateless provider)
func (p *CommandSuggestionProvider) UpdateContext(ctx context.Context, context SuggestionContext) error {
	// This provider is stateless
	return nil
}

// SupportedTypes returns the suggestion types this provider handles
func (p *CommandSuggestionProvider) SupportedTypes() []SuggestionType {
	return []SuggestionType{SuggestionTypeCommand}
}

// GetMetadata returns provider metadata
func (p *CommandSuggestionProvider) GetMetadata() ProviderMetadata {
	return ProviderMetadata{
		Name:        "CommandSuggestionProvider",
		Version:     "1.0.0",
		Description: "Suggests relevant commands based on conversation context",
		Capabilities: []string{
			"context_analysis",
			"pattern_matching",
			"keyword_detection",
		},
	}
}

// calculateCommandConfidence calculates how relevant a command is to the current context
func (p *CommandSuggestionProvider) calculateCommandConfidence(cmd CommandDefinition, currentMsg string, recentContext string, context SuggestionContext) float64 {
	confidence := 0.0
	
	// Check for exact command mention
	if strings.Contains(currentMsg, cmd.Name) {
		confidence = 0.9
		return confidence
	}
	
	// Check for pattern matches
	for _, pattern := range cmd.Patterns {
		if strings.Contains(currentMsg, pattern) {
			confidence = maxFloat(confidence, 0.8)
		}
		if strings.Contains(recentContext, pattern) {
			confidence = maxFloat(confidence, 0.6)
		}
	}
	
	// Check for keyword matches
	keywordMatches := 0
	for _, keyword := range cmd.Keywords {
		if strings.Contains(currentMsg, keyword) {
			keywordMatches++
		}
		if strings.Contains(recentContext, keyword) {
			keywordMatches++
		}
	}
	
	if keywordMatches > 0 {
		keywordConfidence := float64(keywordMatches) / float64(len(cmd.Keywords)*2) // *2 for current + recent
		confidence = maxFloat(confidence, keywordConfidence*0.7)
	}
	
	// Boost confidence based on project context
	if p.isRelevantToProject(cmd, context.ProjectContext) {
		confidence *= 1.2
		if confidence > 1.0 {
			confidence = 1.0
		}
	}
	
	// Adjust by command priority
	confidence *= (1.0 + float64(cmd.Priority)*0.1)
	if confidence > 1.0 {
		confidence = 1.0
	}
	
	return confidence
}

// getRecentContext extracts recent conversation context
func (p *CommandSuggestionProvider) getRecentContext(history []ChatMessage, limit int) string {
	if len(history) == 0 {
		return ""
	}
	
	start := len(history) - limit
	if start < 0 {
		start = 0
	}
	
	var context strings.Builder
	for i := start; i < len(history); i++ {
		context.WriteString(strings.ToLower(history[i].Content))
		context.WriteString(" ")
	}
	
	return context.String()
}

// isRelevantToProject checks if a command is relevant to the current project
func (p *CommandSuggestionProvider) isRelevantToProject(cmd CommandDefinition, project ProjectContext) bool {
	// Language-specific commands
	if project.Language != "" {
		switch project.Language {
		case "go":
			return strings.Contains(cmd.Name, "go") || cmd.Category == "golang"
		case "python":
			return strings.Contains(cmd.Name, "py") || cmd.Category == "python"
		case "javascript", "typescript":
			return strings.Contains(cmd.Name, "js") || strings.Contains(cmd.Name, "ts") || cmd.Category == "javascript"
		}
	}
	
	// Framework-specific commands
	if project.Framework != "" && strings.Contains(strings.ToLower(cmd.Description), strings.ToLower(project.Framework)) {
		return true
	}
	
	return false
}

// getDefaultCommands returns the default set of commands
func getDefaultCommands() []CommandDefinition {
	return []CommandDefinition{
		// Guild-specific commands
		{
			Name:        "help",
			Description: "Show available commands and how to use Guild",
			Category:    "guild",
			Keywords:    []string{"help", "commands", "usage", "how to"},
			Patterns:    []string{"how do i", "what can", "show me"},
			Priority:    10,
		},
		{
			Name:        "chat",
			Description: "Start or continue a chat session",
			Category:    "guild",
			Keywords:    []string{"chat", "conversation", "talk", "discuss"},
			Patterns:    []string{"let's chat", "start chat", "open chat"},
			Priority:    9,
		},
		{
			Name:        "init",
			Description: "Initialize a new Guild project",
			Category:    "guild",
			Keywords:    []string{"init", "initialize", "setup", "create", "new"},
			Patterns:    []string{"new project", "initialize", "setup guild"},
			Priority:    8,
		},
		{
			Name:        "commission",
			Description: "Create or refine a commission document",
			Category:    "guild",
			Keywords:    []string{"commission", "task", "objective", "goal"},
			Patterns:    []string{"create commission", "new task", "define objective"},
			Priority:    7,
		},
		{
			Name:        "campaign",
			Description: "Manage campaigns and projects",
			Category:    "guild",
			Keywords:    []string{"campaign", "project", "workspace"},
			Patterns:    []string{"new campaign", "switch project", "list campaigns"},
			Priority:    7,
		},
		
		// Development commands
		{
			Name:        "build",
			Description: "Build the current project",
			Category:    "development",
			Keywords:    []string{"build", "compile", "make"},
			Patterns:    []string{"build project", "compile code", "make build"},
			Priority:    6,
		},
		{
			Name:        "test",
			Description: "Run tests for the current project",
			Category:    "development",
			Keywords:    []string{"test", "testing", "unit test", "check"},
			Patterns:    []string{"run tests", "test code", "check tests"},
			Priority:    6,
		},
		{
			Name:        "debug",
			Description: "Start debugging session",
			Category:    "development",
			Keywords:    []string{"debug", "debugger", "breakpoint", "step"},
			Patterns:    []string{"debug this", "start debugger", "set breakpoint"},
			Priority:    5,
		},
		
		// File and search commands
		{
			Name:        "search",
			Description: "Search for files, code, or content",
			Category:    "navigation",
			Keywords:    []string{"search", "find", "locate", "grep"},
			Patterns:    []string{"search for", "find in", "where is"},
			Priority:    8,
		},
		{
			Name:        "open",
			Description: "Open a file or directory",
			Category:    "navigation",
			Keywords:    []string{"open", "edit", "view", "show"},
			Patterns:    []string{"open file", "show me", "edit file"},
			Priority:    7,
		},
		
		// Git commands
		{
			Name:        "git",
			Description: "Git operations and version control",
			Category:    "vcs",
			Keywords:    []string{"git", "commit", "push", "pull", "branch"},
			Patterns:    []string{"git commit", "push changes", "create branch"},
			Priority:    6,
		},
		
		// Template commands
		{
			Name:        "template",
			Description: "Use or manage templates",
			Category:    "productivity",
			Keywords:    []string{"template", "snippet", "boilerplate"},
			Patterns:    []string{"use template", "apply template", "create template"},
			Priority:    5,
		},
		{
			Name:        "export",
			Description: "Export chat or content to various formats",
			Category:    "productivity",
			Keywords:    []string{"export", "save", "download", "output"},
			Patterns:    []string{"export chat", "save as", "download conversation"},
			Priority:    4,
		},
	}
}

// Helper function
func maxFloat(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}