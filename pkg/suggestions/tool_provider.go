// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package suggestions

import (
	"context"
	"fmt"
	"strings"

	"github.com/lancekrogers/guild-core/tools"
)

// ToolSuggestionProvider suggests relevant tools based on context
type ToolSuggestionProvider struct {
	toolRegistry       *tools.ToolRegistry
	customKeywords     map[string][]string // Allow custom keyword mappings
	learningEnabled    bool                // Enable learning from usage patterns
	minConfidenceScore float64             // Configurable confidence threshold
}

// NewToolSuggestionProvider creates a new tool suggestion provider
func NewToolSuggestionProvider(toolRegistry *tools.ToolRegistry) *ToolSuggestionProvider {
	return &ToolSuggestionProvider{
		toolRegistry:       toolRegistry,
		customKeywords:     make(map[string][]string),
		learningEnabled:    true,
		minConfidenceScore: 0.3,
	}
}

// WithCustomKeywords adds custom keywords for specific tools
func (p *ToolSuggestionProvider) WithCustomKeywords(toolName string, keywords []string) *ToolSuggestionProvider {
	p.customKeywords[toolName] = keywords
	return p
}

// WithMinConfidence sets the minimum confidence score for suggestions
func (p *ToolSuggestionProvider) WithMinConfidence(score float64) *ToolSuggestionProvider {
	p.minConfidenceScore = score
	return p
}

// WithLearning enables or disables learning from usage patterns
func (p *ToolSuggestionProvider) WithLearning(enabled bool) *ToolSuggestionProvider {
	p.learningEnabled = enabled
	return p
}

// GetSuggestions returns tool suggestions based on context
func (p *ToolSuggestionProvider) GetSuggestions(ctx context.Context, context SuggestionContext) ([]Suggestion, error) {
	if p.toolRegistry == nil {
		return []Suggestion{}, nil
	}

	suggestions := make([]Suggestion, 0)

	// Get all available tools
	allTools := p.toolRegistry.ListTools()

	// Analyze context to determine which tools are most relevant
	currentMsg := strings.ToLower(context.CurrentMessage)
	recentContext := p.getRecentContext(context.ConversationHistory, 3)

	// Score each tool based on relevance
	for _, tool := range allTools {
		confidence := p.calculateToolRelevance(tool, currentMsg, recentContext, context)

		if confidence > p.minConfidenceScore { // Use configurable threshold
			suggestion := Suggestion{
				Type:        SuggestionTypeTool,
				Content:     tool.Name(),
				Display:     fmt.Sprintf("🔧 %s", tool.Name()),
				Description: tool.Description(),
				Confidence:  confidence,
				Priority:    p.calculatePriority(tool, context),
				Action: SuggestionAction{
					Type:   ActionTypeExecute,
					Target: tool.Name(),
					Parameters: map[string]interface{}{
						"tool_name": tool.Name(),
						"category":  tool.Category(),
						"schema":    tool.Schema(),
					},
					Preview: p.generatePreview(tool),
				},
				Tags: []string{tool.Category(), "tool"},
				Metadata: map[string]interface{}{
					"category":      tool.Category(),
					"requires_auth": tool.RequiresAuth(),
					"examples":      tool.Examples(),
				},
			}

			suggestions = append(suggestions, suggestion)
		}
	}

	return suggestions, nil
}

// calculateToolRelevance calculates how relevant a tool is to the current context
func (p *ToolSuggestionProvider) calculateToolRelevance(tool tools.Tool, currentMsg string, recentContext string, context SuggestionContext) float64 {
	relevance := 0.0

	toolName := strings.ToLower(tool.Name())
	toolDesc := strings.ToLower(tool.Description())
	category := strings.ToLower(tool.Category())

	// Check for direct name matches
	if strings.Contains(currentMsg, toolName) {
		relevance = 0.9
	}

	// Check custom keywords first
	if customKeywords, exists := p.customKeywords[tool.Name()]; exists {
		for _, keyword := range customKeywords {
			if strings.Contains(currentMsg, strings.ToLower(keyword)) ||
				strings.Contains(recentContext, strings.ToLower(keyword)) {
				relevance = maxFloat(relevance, 0.8)
			}
		}
	}

	// Category-based relevance
	categoryKeywords := p.getCategoryKeywords(category)
	for _, keyword := range categoryKeywords {
		if strings.Contains(currentMsg, keyword) || strings.Contains(recentContext, keyword) {
			relevance = maxFloat(relevance, 0.7)
		}
	}

	// Smart description analysis - extract key action words
	actionWords := p.extractActionWords(toolDesc)
	for _, actionWord := range actionWords {
		if strings.Contains(currentMsg, actionWord) || strings.Contains(recentContext, actionWord) {
			relevance = maxFloat(relevance, 0.6)
		}
	}

	// Description keyword matching with improved algorithm
	descWords := p.extractSignificantWords(toolDesc)
	msgWords := p.extractSignificantWords(currentMsg)
	contextWords := p.extractSignificantWords(recentContext)

	matchScore := p.calculateWordMatchScore(descWords, msgWords, contextWords)
	if matchScore > 0 {
		relevance = maxFloat(relevance, matchScore*0.7)
	}

	// Example-based matching - check if examples match the context
	if p.matchesExamples(tool, currentMsg, context) {
		relevance = maxFloat(relevance, 0.75)
	}

	// Boost relevance for specific patterns
	if p.hasSpecificToolPatterns(tool, currentMsg, context) {
		relevance *= 1.3
		if relevance > 1.0 {
			relevance = 1.0
		}
	}

	// Project context boost
	if p.isToolRelevantToProject(tool, context.ProjectContext) {
		relevance *= 1.2
		if relevance > 1.0 {
			relevance = 1.0
		}
	}

	// Learning boost - if enabled, boost tools that were useful in similar contexts
	if p.learningEnabled && p.wasUsefulInSimilarContext(tool, context) {
		relevance *= 1.15
		if relevance > 1.0 {
			relevance = 1.0
		}
	}

	return relevance
}

// getCategoryKeywords returns keywords associated with a tool category
func (p *ToolSuggestionProvider) getCategoryKeywords(category string) []string {
	keywords := map[string][]string{
		"file":   {"file", "read", "write", "edit", "create", "delete", "find", "search", "glob", "grep"},
		"git":    {"git", "commit", "branch", "merge", "diff", "log", "blame", "history", "version", "repository"},
		"code":   {"parse", "analyze", "refactor", "ast", "syntax", "metrics", "dependencies", "code quality"},
		"web":    {"fetch", "scrape", "download", "http", "api", "url", "website", "search online"},
		"shell":  {"execute", "command", "bash", "shell", "script", "run", "terminal"},
		"lsp":    {"completion", "definition", "references", "hover", "symbols", "rename", "format"},
		"edit":   {"modify", "change", "update", "replace", "refactor", "multi-edit", "diff"},
		"search": {"find", "locate", "search", "grep", "ag", "ripgrep", "pattern"},
		"dev":    {"test", "testing", "unit test", "integration", "coverage", "tdd"},
		"corpus": {"documentation", "index", "knowledge", "corpus", "scan", "analyze docs"},
	}

	return keywords[category]
}

// hasSpecificToolPatterns checks for specific patterns that indicate a tool is needed
func (p *ToolSuggestionProvider) hasSpecificToolPatterns(tool tools.Tool, currentMsg string, context SuggestionContext) bool {
	toolName := tool.Name()

	patterns := map[string][]string{
		"glob":                {"find files", "list files", "*.go", "*.js", "*.py", "file pattern"},
		"grep":                {"search for", "find in files", "grep", "search content"},
		"git_log":             {"git history", "commit history", "who changed", "recent commits"},
		"git_blame":           {"who wrote", "blame", "author of", "who modified"},
		"git_merge_conflicts": {"merge conflict", "resolve conflict", "conflict markers"},
		"http":                {"fetch url", "download", "api call", "http request"},
		"webfetch":            {"read website", "fetch page", "scrape", "get content from"},
		"websearch":           {"search online", "google", "search web", "find information about"},
		"multi_edit":          {"multiple files", "bulk edit", "refactor across", "change everywhere"},
		"ast":                 {"parse code", "syntax tree", "code structure", "analyze syntax"},
		"code_metrics":        {"code quality", "complexity", "metrics", "code analysis"},
		"test_runner":         {"run tests", "execute tests", "test suite", "unit tests"},
	}

	if patterns[toolName] != nil {
		for _, pattern := range patterns[toolName] {
			if strings.Contains(currentMsg, pattern) {
				return true
			}
		}
	}

	return false
}

// isToolRelevantToProject checks if a tool is relevant to the current project
func (p *ToolSuggestionProvider) isToolRelevantToProject(tool tools.Tool, projectContext ProjectContext) bool {
	category := tool.Category()

	// Git tools are relevant if we're in a git repository
	if category == "git" && projectContext.ProjectType != "" {
		return true
	}

	// Code tools are relevant based on language
	if category == "code" || category == "lsp" {
		if projectContext.Language != "" {
			return true
		}
	}

	// Dev tools are relevant if we have a test framework
	if category == "dev" && (strings.Contains(projectContext.Framework, "test") ||
		projectContext.Language != "") {
		return true
	}

	return false
}

// calculatePriority calculates the priority of a tool suggestion
func (p *ToolSuggestionProvider) calculatePriority(tool tools.Tool, context SuggestionContext) int {
	priority := 5 // Base priority

	category := tool.Category()

	// Category-based priority adjustments
	categoryPriorities := map[string]int{
		"file":   7, // File operations are commonly needed
		"git":    6, // Git operations are frequent
		"code":   7, // Code analysis is important
		"search": 8, // Search tools are very useful
		"edit":   7, // Edit tools are essential
		"lsp":    6, // LSP tools for code intelligence
		"web":    5, // Web tools are less common
		"shell":  4, // Shell is powerful but dangerous
	}

	if catPriority, exists := categoryPriorities[category]; exists {
		priority = catPriority
	}

	// Boost priority if tool was recently used (would need usage tracking)
	// This is a placeholder for future enhancement

	// Boost priority for frequently needed tools
	frequentTools := map[string]int{
		"glob":       2,
		"grep":       2,
		"multi_edit": 1,
		"git_log":    1,
	}

	if boost, exists := frequentTools[tool.Name()]; exists {
		priority += boost
	}

	// Cap priority at 10
	if priority > 10 {
		priority = 10
	}

	return priority
}

// generatePreview generates a preview of how to use the tool
func (p *ToolSuggestionProvider) generatePreview(tool tools.Tool) string {
	examples := tool.Examples()
	if len(examples) > 0 {
		// Return the first example as preview
		return fmt.Sprintf("Example: %s", examples[0])
	}

	// Generate a basic preview based on schema
	schema := tool.Schema()
	if props, ok := schema["properties"].(map[string]interface{}); ok {
		var params []string
		for param := range props {
			params = append(params, param)
		}
		if len(params) > 0 {
			return fmt.Sprintf("Parameters: %s", strings.Join(params, ", "))
		}
	}

	return fmt.Sprintf("Tool: %s", tool.Name())
}

// getRecentContext extracts recent conversation context
func (p *ToolSuggestionProvider) getRecentContext(history []ChatMessage, limit int) string {
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

// extractActionWords extracts action words from tool description
func (p *ToolSuggestionProvider) extractActionWords(description string) []string {
	// Common action verbs in tool descriptions
	actionVerbs := []string{
		"search", "find", "create", "read", "write", "edit", "delete", "fetch",
		"analyze", "parse", "format", "compile", "execute", "run", "test",
		"commit", "merge", "diff", "blame", "log", "checkout", "branch",
		"download", "upload", "scrape", "query", "filter", "sort", "group",
		"refactor", "rename", "extract", "inline", "move", "copy", "replace",
	}

	words := strings.Fields(strings.ToLower(description))
	actions := []string{}

	for _, word := range words {
		// Check if word is an action verb
		for _, action := range actionVerbs {
			if strings.HasPrefix(word, action) {
				actions = append(actions, action)
			}
		}
	}

	return actions
}

// extractSignificantWords extracts significant words for matching
func (p *ToolSuggestionProvider) extractSignificantWords(text string) []string {
	// Skip common words
	commonWords := map[string]bool{
		"the": true, "a": true, "an": true, "and": true, "or": true,
		"in": true, "on": true, "at": true, "to": true, "for": true,
		"of": true, "with": true, "by": true, "from": true, "is": true,
		"are": true, "was": true, "were": true, "been": true, "be": true,
		"have": true, "has": true, "had": true, "do": true, "does": true,
		"did": true, "will": true, "would": true, "should": true, "could": true,
		"may": true, "might": true, "must": true, "can": true, "this": true,
		"that": true, "these": true, "those": true, "i": true, "you": true,
		"he": true, "she": true, "it": true, "we": true, "they": true,
		"over": true, // Add "over" to common words
	}

	words := strings.Fields(strings.ToLower(text))
	significant := []string{}

	for _, word := range words {
		// Clean punctuation
		word = strings.Trim(word, ".,!?;:()[]{}\"'")

		// Skip if too short or common
		if len(word) > 2 && !commonWords[word] {
			significant = append(significant, word)
		}
	}

	return significant
}

// calculateWordMatchScore calculates match score between word sets
func (p *ToolSuggestionProvider) calculateWordMatchScore(descWords, msgWords, contextWords []string) float64 {
	if len(descWords) == 0 {
		return 0
	}

	matches := 0
	for _, descWord := range descWords {
		for _, msgWord := range msgWords {
			if descWord == msgWord {
				matches++
				break
			}
		}
		for _, ctxWord := range contextWords {
			if descWord == ctxWord {
				matches++
				break
			}
		}
	}

	return float64(matches) / float64(len(descWords))
}

// matchesExamples checks if tool examples match the context
func (p *ToolSuggestionProvider) matchesExamples(tool tools.Tool, currentMsg string, context SuggestionContext) bool {
	examples := tool.Examples()
	if len(examples) == 0 {
		return false
	}

	// Check if current message resembles any example
	for _, example := range examples {
		// Extract significant words from example
		exampleWords := p.extractSignificantWords(example)
		msgWords := p.extractSignificantWords(currentMsg)

		// Calculate similarity
		matchCount := 0
		for _, exWord := range exampleWords {
			for _, msgWord := range msgWords {
				if exWord == msgWord {
					matchCount++
				}
			}
		}

		// If we have enough matching words, consider it a match
		// Use a minimum of 2 words or 30% of example words
		minMatches := 2
		if float64(len(exampleWords))*0.3 > 2 {
			minMatches = int(float64(len(exampleWords)) * 0.3)
		}

		if matchCount >= minMatches && len(exampleWords) > 0 {
			return true
		}
	}

	return false
}

// wasUsefulInSimilarContext checks if tool was useful in similar contexts (placeholder)
func (p *ToolSuggestionProvider) wasUsefulInSimilarContext(tool tools.Tool, context SuggestionContext) bool {
	// This is a placeholder for future implementation
	// Would integrate with usage analytics to track which tools were accepted/used
	// in similar contexts (same project type, similar keywords, etc.)
	return false
}

// UpdateContext updates the provider's context (no-op for stateless provider)
func (p *ToolSuggestionProvider) UpdateContext(ctx context.Context, context SuggestionContext) error {
	// This provider is stateless
	return nil
}

// SupportedTypes returns the suggestion types this provider handles
func (p *ToolSuggestionProvider) SupportedTypes() []SuggestionType {
	return []SuggestionType{SuggestionTypeTool}
}

// GetMetadata returns provider metadata
func (p *ToolSuggestionProvider) GetMetadata() ProviderMetadata {
	return ProviderMetadata{
		Name:        "ToolSuggestionProvider",
		Version:     "1.0.0",
		Description: "Suggests relevant tools from the tool registry based on conversation context",
		Capabilities: []string{
			"tool_matching",
			"category_analysis",
			"context_relevance",
			"example_preview",
		},
	}
}
