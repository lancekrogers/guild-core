// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package code

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/lancekrogers/guild/pkg/gerror"
	"github.com/lancekrogers/guild/tools"
)

// SearchReplaceTool provides semantic code search and replace functionality
type SearchReplaceTool struct {
	*tools.BaseTool
}

// SearchReplaceParams represents the input parameters for search and replace
type SearchReplaceParams struct {
	Pattern       string   `json:"pattern"`                  // Search pattern (regex or literal)
	Replacement   string   `json:"replacement,omitempty"`    // Replacement text (empty for search-only)
	Files         []string `json:"files,omitempty"`          // File glob patterns to search
	Preview       bool     `json:"preview,omitempty"`        // Show preview before applying
	Regex         bool     `json:"regex,omitempty"`          // Use regex pattern matching
	CaseSensitive bool     `json:"case_sensitive,omitempty"` // Case sensitive search
	WholeWord     bool     `json:"whole_word,omitempty"`     // Match whole words only
	Recursive     bool     `json:"recursive,omitempty"`      // Search recursively in directories
	MaxResults    int      `json:"max_results,omitempty"`    // Maximum number of results (default: 100)
	Context       int      `json:"context,omitempty"`        // Number of context lines to show
}

// SearchReplaceResult represents the result of search and replace operation
type SearchReplaceResult struct {
	Pattern      string           `json:"pattern"`
	Replacement  string           `json:"replacement,omitempty"`
	TotalMatches int              `json:"total_matches"`
	FilesChanged int              `json:"files_changed"`
	Matches      []*SearchMatch   `json:"matches"`
	Preview      []*PreviewChange `json:"preview,omitempty"`
	Applied      bool             `json:"applied"`
	Errors       []string         `json:"errors,omitempty"`
	Summary      *SearchSummary   `json:"summary"`
}

// SearchMatch represents a single search match
type SearchMatch struct {
	File          string   `json:"file"`
	Line          int      `json:"line"`
	Column        int      `json:"column"`
	MatchText     string   `json:"match_text"`
	ContextBefore []string `json:"context_before,omitempty"`
	ContextAfter  []string `json:"context_after,omitempty"`
	Replaced      bool     `json:"replaced,omitempty"`
}

// PreviewChange represents a preview of what would be changed
type PreviewChange struct {
	File       string `json:"file"`
	Line       int    `json:"line"`
	Before     string `json:"before"`
	After      string `json:"after"`
	MatchStart int    `json:"match_start"`
	MatchEnd   int    `json:"match_end"`
}

// SearchSummary provides summary statistics
type SearchSummary struct {
	FilesSearched     int            `json:"files_searched"`
	FilesWithMatches  int            `json:"files_with_matches"`
	LanguageBreakdown map[string]int `json:"language_breakdown"`
	PatternType       string         `json:"pattern_type"` // literal, regex, semantic
	TimeElapsed       string         `json:"time_elapsed,omitempty"`
}

// NewSearchReplaceTool creates a new search and replace tool
func NewSearchReplaceTool() *SearchReplaceTool {
	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"pattern": map[string]interface{}{
				"type":        "string",
				"description": "Search pattern (literal text or regex)",
			},
			"replacement": map[string]interface{}{
				"type":        "string",
				"description": "Replacement text (omit for search-only)",
			},
			"files": map[string]interface{}{
				"type":        "array",
				"items":       map[string]interface{}{"type": "string"},
				"description": "File patterns to search (glob syntax)",
			},
			"preview": map[string]interface{}{
				"type":        "boolean",
				"description": "Show preview of changes before applying",
			},
			"regex": map[string]interface{}{
				"type":        "boolean",
				"description": "Use regular expressions for pattern matching",
			},
			"case_sensitive": map[string]interface{}{
				"type":        "boolean",
				"description": "Case sensitive search",
			},
			"whole_word": map[string]interface{}{
				"type":        "boolean",
				"description": "Match whole words only",
			},
			"recursive": map[string]interface{}{
				"type":        "boolean",
				"description": "Search recursively in directories",
			},
			"max_results": map[string]interface{}{
				"type":        "integer",
				"description": "Maximum number of results to return",
			},
			"context": map[string]interface{}{
				"type":        "integer",
				"description": "Number of context lines to show around matches",
			},
		},
		"required": []string{"pattern"},
	}

	examples := []string{
		`{"pattern": "TODO", "files": ["*.go"], "recursive": true}`,
		`{"pattern": "fmt.Println", "replacement": "log.Info", "files": ["src/**/*.go"], "preview": true}`,
		`{"pattern": "function\\s+(\\w+)", "replacement": "func $1", "regex": true}`,
		`{"pattern": "old_function_name", "replacement": "new_function_name", "whole_word": true}`,
		`{"pattern": "deprecated_method", "files": ["*.py"], "context": 3}`,
	}

	baseTool := tools.NewBaseTool(
		"search_replace",
		"Search for patterns in code and optionally replace them. Supports literal text, regex patterns, and semantic search with context.",
		schema,
		"code",
		false,
		examples,
	)

	return &SearchReplaceTool{
		BaseTool: baseTool,
	}
}

// Execute runs the search and replace tool with the given input
func (t *SearchReplaceTool) Execute(ctx context.Context, input string) (*tools.ToolResult, error) {
	var params SearchReplaceParams
	if err := json.Unmarshal([]byte(input), &params); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInvalidInput, "invalid input").
			WithComponent("search_replace_tool").
			WithOperation("execute")
	}

	// Validate required parameters
	if params.Pattern == "" {
		return nil, gerror.New(gerror.ErrCodeInvalidInput, "pattern is required", nil).
			WithComponent("search_replace_tool").
			WithOperation("execute")
	}

	// Set defaults
	if params.MaxResults == 0 {
		params.MaxResults = 100
	}
	if len(params.Files) == 0 {
		params.Files = []string{"*"}
	}

	// Perform search
	result, err := t.performSearch(ctx, params)
	if err != nil {
		return nil, err
	}

	// If replacement is specified and not preview mode, apply changes
	if params.Replacement != "" && !params.Preview && len(result.Matches) > 0 {
		err = t.applyReplacements(ctx, params, result)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("Failed to apply replacements: %v", err))
		} else {
			result.Applied = true
		}
	}

	// Format output
	output := t.formatResult(result, params)

	metadata := map[string]string{
		"pattern":       params.Pattern,
		"total_matches": fmt.Sprintf("%d", result.TotalMatches),
		"files_changed": fmt.Sprintf("%d", result.FilesChanged),
		"applied":       fmt.Sprintf("%t", result.Applied),
	}

	extraData := map[string]interface{}{
		"result": result,
	}

	return tools.NewToolResult(output, metadata, nil, extraData), nil
}

// performSearch performs the search operation
func (t *SearchReplaceTool) performSearch(ctx context.Context, params SearchReplaceParams) (*SearchReplaceResult, error) {
	result := &SearchReplaceResult{
		Pattern:     params.Pattern,
		Replacement: params.Replacement,
		Summary: &SearchSummary{
			LanguageBreakdown: make(map[string]int),
		},
	}

	// Compile regex pattern if needed
	var pattern *regexp.Regexp
	var err error

	if params.Regex {
		flags := ""
		if !params.CaseSensitive {
			flags = "(?i)"
		}
		pattern, err = regexp.Compile(flags + params.Pattern)
		if err != nil {
			return nil, gerror.Wrap(err, gerror.ErrCodeInvalidInput, "invalid regex pattern").
				WithComponent("search_replace_tool").
				WithOperation("perform_search")
		}
		result.Summary.PatternType = "regex"
	} else {
		// For literal patterns, escape special regex characters
		escapedPattern := regexp.QuoteMeta(params.Pattern)
		if params.WholeWord {
			escapedPattern = `\b` + escapedPattern + `\b`
		}
		flags := ""
		if !params.CaseSensitive {
			flags = "(?i)"
		}
		pattern, err = regexp.Compile(flags + escapedPattern)
		if err != nil {
			return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to compile literal pattern").
				WithComponent("search_replace_tool").
				WithOperation("perform_search")
		}
		result.Summary.PatternType = "literal"
	}

	// Find files to search
	files, err := t.findFiles(params.Files, params.Recursive)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to find files").
			WithComponent("search_replace_tool").
			WithOperation("perform_search")
	}

	result.Summary.FilesSearched = len(files)

	// Search each file
	matchCount := 0
	filesWithMatches := 0

	for _, file := range files {
		if matchCount >= params.MaxResults {
			break
		}

		matches, err := t.searchFile(file, pattern, params.Context)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("Failed to search %s: %v", file, err))
			continue
		}

		if len(matches) > 0 {
			filesWithMatches++

			// Limit matches to respect MaxResults
			remainingSlots := params.MaxResults - matchCount
			if remainingSlots > 0 {
				if len(matches) > remainingSlots {
					matches = matches[:remainingSlots]
				}
				result.Matches = append(result.Matches, matches...)
				matchCount += len(matches)
			}

			// Update language breakdown
			language := string(DetectLanguage(file))
			result.Summary.LanguageBreakdown[language]++
		}

		// Generate preview if replacement is specified
		if params.Replacement != "" && len(matches) > 0 {
			previews, err := t.generatePreviews(file, pattern, params.Replacement, matches)
			if err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("Failed to generate preview for %s: %v", file, err))
			} else {
				result.Preview = append(result.Preview, previews...)
			}
		}
	}

	result.TotalMatches = matchCount
	result.Summary.FilesWithMatches = filesWithMatches

	return result, nil
}

// findFiles finds files matching the given patterns
func (t *SearchReplaceTool) findFiles(patterns []string, recursive bool) ([]string, error) {
	var files []string
	seen := make(map[string]bool)

	for _, pattern := range patterns {
		if recursive && strings.Contains(pattern, "**") {
			// Handle recursive glob patterns
			matches, err := filepath.Glob(pattern)
			if err != nil {
				continue
			}
			for _, match := range matches {
				if !seen[match] {
					info, err := os.Stat(match)
					if err == nil && !info.IsDir() {
						files = append(files, match)
						seen[match] = true
					}
				}
			}
		} else if recursive {
			// Walk directory tree for patterns like "*.go"
			err := filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
				if err != nil || info.IsDir() {
					return err
				}

				matched, err := filepath.Match(pattern, filepath.Base(path))
				if err != nil {
					return err
				}

				if matched && !seen[path] {
					files = append(files, path)
					seen[path] = true
				}

				return nil
			})
			if err != nil {
				return nil, err
			}
		} else {
			// Non-recursive glob
			matches, err := filepath.Glob(pattern)
			if err != nil {
				continue
			}
			for _, match := range matches {
				if !seen[match] {
					info, err := os.Stat(match)
					if err == nil && !info.IsDir() {
						files = append(files, match)
						seen[match] = true
					}
				}
			}
		}
	}

	return files, nil
}

// searchFile searches for patterns in a single file
func (t *SearchReplaceTool) searchFile(filename string, pattern *regexp.Regexp, contextLines int) ([]*SearchMatch, error) {
	content, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(content), "\n")
	var matches []*SearchMatch

	for lineNum, line := range lines {
		lineMatches := pattern.FindAllStringIndex(line, -1)
		for _, match := range lineMatches {
			searchMatch := &SearchMatch{
				File:      filename,
				Line:      lineNum + 1,  // 1-based line numbers
				Column:    match[0] + 1, // 1-based column numbers
				MatchText: line[match[0]:match[1]],
			}

			// Add context lines if requested
			if contextLines > 0 {
				start := lineNum - contextLines
				if start < 0 {
					start = 0
				}
				end := lineNum + contextLines + 1
				if end > len(lines) {
					end = len(lines)
				}

				// Context before
				for i := start; i < lineNum; i++ {
					searchMatch.ContextBefore = append(searchMatch.ContextBefore, lines[i])
				}

				// Context after
				for i := lineNum + 1; i < end; i++ {
					searchMatch.ContextAfter = append(searchMatch.ContextAfter, lines[i])
				}
			}

			matches = append(matches, searchMatch)
		}
	}

	return matches, nil
}

// generatePreviews generates preview of what replacements would look like
func (t *SearchReplaceTool) generatePreviews(filename string, pattern *regexp.Regexp, replacement string, matches []*SearchMatch) ([]*PreviewChange, error) {
	content, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(content), "\n")
	var previews []*PreviewChange

	for _, match := range matches {
		if match.Line-1 >= len(lines) {
			continue
		}

		lineContent := lines[match.Line-1]

		// Apply the replacement to this line
		newLine := pattern.ReplaceAllString(lineContent, replacement)

		// Find the position of the match in the original line
		matchIndices := pattern.FindStringIndex(lineContent)
		var matchStart, matchEnd int
		if len(matchIndices) >= 2 {
			matchStart = matchIndices[0]
			matchEnd = matchIndices[1]
		}

		preview := &PreviewChange{
			File:       filename,
			Line:       match.Line,
			Before:     lineContent,
			After:      newLine,
			MatchStart: matchStart,
			MatchEnd:   matchEnd,
		}

		previews = append(previews, preview)
	}

	return previews, nil
}

// applyReplacements applies the replacements to files
func (t *SearchReplaceTool) applyReplacements(ctx context.Context, params SearchReplaceParams, result *SearchReplaceResult) error {
	// Group matches by file
	fileMatches := make(map[string][]*SearchMatch)
	for _, match := range result.Matches {
		fileMatches[match.File] = append(fileMatches[match.File], match)
	}

	// Compile the pattern again for replacement
	var pattern *regexp.Regexp
	var err error

	if params.Regex {
		flags := ""
		if !params.CaseSensitive {
			flags = "(?i)"
		}
		pattern, err = regexp.Compile(flags + params.Pattern)
	} else {
		escapedPattern := regexp.QuoteMeta(params.Pattern)
		if params.WholeWord {
			escapedPattern = `\b` + escapedPattern + `\b`
		}
		flags := ""
		if !params.CaseSensitive {
			flags = "(?i)"
		}
		pattern, err = regexp.Compile(flags + escapedPattern)
	}

	if err != nil {
		return err
	}

	// Apply replacements to each file
	for filename, matches := range fileMatches {
		err := t.replaceInFile(filename, pattern, params.Replacement)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("Failed to replace in %s: %v", filename, err))
			continue
		}

		result.FilesChanged++

		// Mark matches as replaced
		for _, match := range matches {
			match.Replaced = true
		}
	}

	return nil
}

// replaceInFile performs the actual replacement in a file
func (t *SearchReplaceTool) replaceInFile(filename string, pattern *regexp.Regexp, replacement string) error {
	// Read the file
	content, err := os.ReadFile(filename)
	if err != nil {
		return err
	}

	// Perform the replacement
	newContent := pattern.ReplaceAllString(string(content), replacement)

	// Write back to file
	return os.WriteFile(filename, []byte(newContent), 0644)
}

// formatResult formats the search and replace result for output
func (t *SearchReplaceTool) formatResult(result *SearchReplaceResult, params SearchReplaceParams) string {
	var output strings.Builder

	// Header
	if params.Replacement != "" {
		if result.Applied {
			output.WriteString(fmt.Sprintf("Search and Replace Applied\n"))
		} else if params.Preview {
			output.WriteString(fmt.Sprintf("Search and Replace Preview\n"))
		} else {
			output.WriteString(fmt.Sprintf("Search and Replace (Dry Run)\n"))
		}
		output.WriteString(fmt.Sprintf("Pattern: %s → %s\n", result.Pattern, result.Replacement))
	} else {
		output.WriteString(fmt.Sprintf("Search Results\n"))
		output.WriteString(fmt.Sprintf("Pattern: %s\n", result.Pattern))
	}

	// Summary
	output.WriteString(fmt.Sprintf("Found %d matches in %d files (searched %d files)\n",
		result.TotalMatches, result.Summary.FilesWithMatches, result.Summary.FilesSearched))

	if result.Applied {
		output.WriteString(fmt.Sprintf("Modified %d files\n", result.FilesChanged))
	}

	// Language breakdown
	if len(result.Summary.LanguageBreakdown) > 1 {
		output.WriteString("\nLanguage breakdown:\n")
		for lang, count := range result.Summary.LanguageBreakdown {
			output.WriteString(fmt.Sprintf("  %s: %d files\n", lang, count))
		}
	}

	// Show matches/previews
	if len(result.Preview) > 0 && params.Preview {
		output.WriteString("\nPreview of changes:\n")
		for i, preview := range result.Preview {
			if i >= 10 { // Limit preview output
				output.WriteString(fmt.Sprintf("... and %d more changes\n", len(result.Preview)-10))
				break
			}
			output.WriteString(fmt.Sprintf("\n%s:%d\n", preview.File, preview.Line))
			output.WriteString(fmt.Sprintf("- %s\n", preview.Before))
			output.WriteString(fmt.Sprintf("+ %s\n", preview.After))
		}
	} else if len(result.Matches) > 0 {
		output.WriteString("\nMatches:\n")
		currentFile := ""
		matchCount := 0

		for _, match := range result.Matches {
			if matchCount >= 20 { // Limit output
				output.WriteString(fmt.Sprintf("... and %d more matches\n", result.TotalMatches-20))
				break
			}

			if match.File != currentFile {
				if currentFile != "" {
					output.WriteString("\n")
				}
				output.WriteString(fmt.Sprintf("%s:\n", match.File))
				currentFile = match.File
			}

			status := ""
			if match.Replaced {
				status = " [REPLACED]"
			}

			output.WriteString(fmt.Sprintf("  Line %d:%d: %s%s\n",
				match.Line, match.Column, match.MatchText, status))

			// Show context if available
			if len(match.ContextBefore) > 0 || len(match.ContextAfter) > 0 {
				for _, line := range match.ContextBefore {
					output.WriteString(fmt.Sprintf("    %s\n", line))
				}
				if len(match.ContextAfter) > 0 {
					for _, line := range match.ContextAfter {
						output.WriteString(fmt.Sprintf("    %s\n", line))
					}
				}
			}

			matchCount++
		}
	}

	// Show errors if any
	if len(result.Errors) > 0 {
		output.WriteString("\nWarnings/Errors:\n")
		for _, err := range result.Errors {
			output.WriteString(fmt.Sprintf("- %s\n", err))
		}
	}

	return output.String()
}
