// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package search

import (
	"bufio"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/guild-framework/guild-core/pkg/gerror"
	"github.com/guild-framework/guild-core/tools"
)

// AgTool provides Silver Searcher (ag) functionality for agents
type AgTool struct {
	*tools.BaseTool
	workingDir string // Working directory for searches
}

// AgToolInput represents the input for ag searches
type AgToolInput struct {
	Pattern        string   `json:"pattern"`                   // Search pattern
	Path           string   `json:"path,omitempty"`            // Directory path to search (default: current working directory)
	FileTypes      []string `json:"file_types,omitempty"`      // File types to include (e.g., ["go", "js", "py"])
	IgnorePatterns []string `json:"ignore_patterns,omitempty"` // Patterns to ignore
	CaseSensitive  bool     `json:"case_sensitive,omitempty"`  // Case-sensitive search (default: false)
	WholeWord      bool     `json:"whole_word,omitempty"`      // Match whole words only
	Literal        bool     `json:"literal,omitempty"`         // Treat pattern as literal string, not regex
	MaxResults     int      `json:"max_results,omitempty"`     // Maximum number of results (default: 100)
	Context        int      `json:"context,omitempty"`         // Lines of context around matches
	Timeout        int      `json:"timeout,omitempty"`         // Timeout in seconds (default: 30)
}

// AgSearchResult represents a single search result
type AgSearchResult struct {
	File    string `json:"file"`    // File path
	Line    int    `json:"line"`    // Line number
	Column  int    `json:"column"`  // Column number (if available)
	Match   string `json:"match"`   // Matched text
	Context string `json:"context"` // Full line containing the match
	Before  string `json:"before"`  // Context lines before match
	After   string `json:"after"`   // Context lines after match
}

// AgToolResult represents the complete search results
type AgToolResult struct {
	Results   []AgSearchResult `json:"results"`
	Total     int              `json:"total"`
	Truncated bool             `json:"truncated"`
	Pattern   string           `json:"pattern"`
	Path      string           `json:"path"`
	Duration  string           `json:"duration"`
}

// NewAgTool creates a new Silver Searcher tool
func NewAgTool(workingDir string) *AgTool {
	if workingDir == "" {
		workingDir, _ = os.Getwd()
	}

	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"pattern": map[string]interface{}{
				"type":        "string",
				"description": "Search pattern (regex or literal string)",
			},
			"path": map[string]interface{}{
				"type":        "string",
				"description": "Directory path to search (default: current working directory)",
			},
			"file_types": map[string]interface{}{
				"type":        "array",
				"items":       map[string]interface{}{"type": "string"},
				"description": "File types to include (e.g., [\"go\", \"js\", \"py\"])",
			},
			"ignore_patterns": map[string]interface{}{
				"type":        "array",
				"items":       map[string]interface{}{"type": "string"},
				"description": "Patterns to ignore",
			},
			"case_sensitive": map[string]interface{}{
				"type":        "boolean",
				"description": "Case-sensitive search",
				"default":     false,
			},
			"whole_word": map[string]interface{}{
				"type":        "boolean",
				"description": "Match whole words only",
				"default":     false,
			},
			"literal": map[string]interface{}{
				"type":        "boolean",
				"description": "Treat pattern as literal string, not regex",
				"default":     false,
			},
			"max_results": map[string]interface{}{
				"type":        "integer",
				"description": "Maximum number of results",
				"default":     100,
			},
			"context": map[string]interface{}{
				"type":        "integer",
				"description": "Lines of context around matches",
				"default":     0,
			},
			"timeout": map[string]interface{}{
				"type":        "integer",
				"description": "Timeout in seconds",
				"default":     30,
			},
		},
		"required": []string{"pattern"},
	}

	examples := []string{
		`{"pattern": "function", "file_types": ["go", "js"]}`,
		`{"pattern": "TODO", "case_sensitive": true}`,
		`{"pattern": "import.*json", "file_types": ["py"], "context": 2}`,
		`{"pattern": "struct", "path": "./pkg", "whole_word": true}`,
		`{"pattern": "error", "ignore_patterns": ["*.test.go", "testdata"]}`,
	}

	baseTool := tools.NewBaseTool(
		"ag",
		"Search for patterns in files using The Silver Searcher (ag) - a fast code search tool",
		schema,
		"search",
		false,
		examples,
	)

	return &AgTool{
		BaseTool:   baseTool,
		workingDir: workingDir,
	}
}

// Execute runs the ag tool with the given input
func (t *AgTool) Execute(ctx context.Context, input string) (*tools.ToolResult, error) {
	var params AgToolInput
	if err := json.Unmarshal([]byte(input), &params); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInvalidInput, "invalid input").
			WithComponent("ag_tool").
			WithOperation("execute")
	}

	// Validate pattern
	if params.Pattern == "" {
		return nil, gerror.New(gerror.ErrCodeInvalidInput, "pattern is required", nil).
			WithComponent("ag_tool").
			WithOperation("execute")
	}

	// Check if ag is installed
	if !t.isAgInstalled() {
		return tools.NewToolResult("", map[string]string{
			"error": "ag_not_installed",
		}, gerror.New(gerror.ErrCodeNotFound, "The Silver Searcher (ag) is not installed. Please install it using: brew install the_silver_searcher (macOS) or apt-get install silversearcher-ag (Ubuntu)", nil).
			WithComponent("ag_tool").
			WithOperation("execute"), nil), nil
	}

	// Set defaults
	if params.Path == "" {
		params.Path = t.workingDir
	}
	if params.MaxResults == 0 {
		params.MaxResults = 100
	}
	if params.Timeout == 0 {
		params.Timeout = 30
	}

	// Convert to absolute path
	searchPath, err := filepath.Abs(params.Path)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInvalidInput, "invalid path").
			WithComponent("ag_tool").
			WithOperation("execute")
	}

	// Check if path exists
	if _, err := os.Stat(searchPath); os.IsNotExist(err) {
		return nil, gerror.Newf(gerror.ErrCodeNotFound, "search path does not exist: %s", searchPath).
			WithComponent("ag_tool").
			WithOperation("execute")
	}

	startTime := time.Now()

	// Execute ag search
	result, err := t.executeAgSearch(ctx, params, searchPath)
	if err != nil {
		return tools.NewToolResult("", map[string]string{
			"pattern": params.Pattern,
			"path":    searchPath,
		}, err, nil), nil
	}

	result.Duration = time.Since(startTime).String()

	// Convert result to JSON
	resultJSON, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to marshal results").
			WithComponent("ag_tool").
			WithOperation("execute")
	}

	metadata := map[string]string{
		"pattern":   params.Pattern,
		"path":      searchPath,
		"total":     strconv.Itoa(result.Total),
		"truncated": strconv.FormatBool(result.Truncated),
		"duration":  result.Duration,
	}

	return tools.NewToolResult(string(resultJSON), metadata, nil, map[string]interface{}{
		"structured_results": result,
	}), nil
}

// isAgInstalled checks if ag is installed and available
func (t *AgTool) isAgInstalled() bool {
	cmd := exec.Command("ag", "--version")
	return cmd.Run() == nil
}

// executeAgSearch performs the actual ag search
func (t *AgTool) executeAgSearch(ctx context.Context, params AgToolInput, searchPath string) (*AgToolResult, error) {
	// Build ag command arguments
	args := []string{}

	// Add pattern as first argument
	args = append(args, params.Pattern)

	// Add search path
	args = append(args, searchPath)

	// File type filtering
	for _, fileType := range params.FileTypes {
		args = append(args, "--"+fileType)
	}

	// Ignore patterns
	for _, pattern := range params.IgnorePatterns {
		args = append(args, "--ignore", pattern)
	}

	// Case sensitivity
	if params.CaseSensitive {
		args = append(args, "--case-sensitive")
	} else {
		args = append(args, "--ignore-case")
	}

	// Whole word matching
	if params.WholeWord {
		args = append(args, "--word-regexp")
	}

	// Literal pattern
	if params.Literal {
		args = append(args, "--literal")
	}

	// Context lines
	if params.Context > 0 {
		args = append(args, "--context", strconv.Itoa(params.Context))
	}

	// Output format: line numbers, column numbers, and file names
	args = append(args, "--line-number", "--column", "--nogroup", "--nocolor")

	// Create context with timeout
	timeout := time.Duration(params.Timeout) * time.Second
	execCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Execute ag command
	cmd := exec.CommandContext(execCtx, "ag", args...)
	cmd.Dir = searchPath

	output, err := cmd.Output()
	if err != nil {
		// Handle exit code 1 (no matches found) as success
		if exitErr, ok := err.(*exec.ExitError); ok {
			if exitErr.ExitCode() == 1 {
				// No matches found
				return &AgToolResult{
					Results:   []AgSearchResult{},
					Total:     0,
					Truncated: false,
					Pattern:   params.Pattern,
					Path:      searchPath,
				}, nil
			}
		}

		// Handle timeout
		if execCtx.Err() == context.DeadlineExceeded {
			return nil, gerror.Newf(gerror.ErrCodeInternal, "ag search timed out after %d seconds", params.Timeout).
				WithComponent("ag_tool").
				WithOperation("execute_ag_search")
		}

		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "ag command failed").
			WithComponent("ag_tool").
			WithOperation("execute_ag_search")
	}

	// Parse ag output
	results, err := t.parseAgOutput(string(output), params.MaxResults)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to parse ag output").
			WithComponent("ag_tool").
			WithOperation("execute_ag_search")
	}

	truncated := len(results) >= params.MaxResults

	return &AgToolResult{
		Results:   results,
		Total:     len(results),
		Truncated: truncated,
		Pattern:   params.Pattern,
		Path:      searchPath,
	}, nil
}

// parseAgOutput parses the output from ag command
func (t *AgTool) parseAgOutput(output string, maxResults int) ([]AgSearchResult, error) {
	var results []AgSearchResult
	scanner := bufio.NewScanner(strings.NewReader(output))

	for scanner.Scan() && len(results) < maxResults {
		line := scanner.Text()
		if line == "" {
			continue
		}

		result, err := t.parseAgLine(line)
		if err != nil {
			// Log error but continue processing other lines
			continue
		}

		results = append(results, result)
	}

	if err := scanner.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "error reading ag output").
			WithComponent("ag_tool").
			WithOperation("parse_ag_output")
	}

	return results, nil
}

// parseAgLine parses a single line of ag output
// Expected format: filename:line:column:match_text
func (t *AgTool) parseAgLine(line string) (AgSearchResult, error) {
	// Split by colon, but be careful about Windows paths and colons in content
	parts := strings.SplitN(line, ":", 4)
	if len(parts) < 4 {
		return AgSearchResult{}, gerror.Newf(gerror.ErrCodeInvalidFormat, "invalid ag output format: %s", line).
			WithComponent("ag_tool").
			WithOperation("parse_ag_line")
	}

	// Parse line number
	lineNum, err := strconv.Atoi(parts[1])
	if err != nil {
		return AgSearchResult{}, gerror.Wrap(err, gerror.ErrCodeInvalidFormat, "invalid line number").
			WithComponent("ag_tool").
			WithOperation("parse_ag_line")
	}

	// Parse column number
	column, err := strconv.Atoi(parts[2])
	if err != nil {
		return AgSearchResult{}, gerror.Wrap(err, gerror.ErrCodeInvalidFormat, "invalid column number").
			WithComponent("ag_tool").
			WithOperation("parse_ag_line")
	}

	// Extract file path relative to working directory
	filePath := parts[0]
	if relPath, err := filepath.Rel(t.workingDir, filePath); err == nil {
		filePath = relPath
	}

	// The rest is the matched content
	matchContent := parts[3]

	return AgSearchResult{
		File:    filePath,
		Line:    lineNum,
		Column:  column,
		Match:   strings.TrimSpace(matchContent),
		Context: matchContent,
	}, nil
}

// GetCapabilities returns the capabilities of the ag tool
func (t *AgTool) GetCapabilities() []string {
	return []string{"search", "text_search", "code_search", "pattern_matching", "file_filtering"}
}
