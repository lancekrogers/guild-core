// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package git

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/guild-framework/guild-core/pkg/gerror"
	"github.com/guild-framework/guild-core/tools"
)

// GitBlameInput represents the input parameters for git blame
type GitBlameInput struct {
	File             string `json:"file"`
	LineStart        int    `json:"line_start,omitempty"`
	LineEnd          int    `json:"line_end,omitempty"`
	IgnoreWhitespace bool   `json:"ignore_whitespace,omitempty"`
	ShowEmail        bool   `json:"show_email,omitempty"`
	ShowDate         bool   `json:"show_date,omitempty"`
}

// GitBlameTool implements git blame functionality
type GitBlameTool struct {
	*tools.BaseTool
	workspacePath string
}

// NewGitBlameTool creates a new git blame tool
func NewGitBlameTool(workspacePath string) *GitBlameTool {
	schema := map[string]interface{}{
		"type":     "object",
		"required": []string{"file"},
		"properties": map[string]interface{}{
			"file": map[string]interface{}{
				"type":        "string",
				"description": "Path to file to blame (relative to workspace root)",
			},
			"line_start": map[string]interface{}{
				"type":        "integer",
				"description": "Start line number (requires line_end)",
			},
			"line_end": map[string]interface{}{
				"type":        "integer",
				"description": "End line number (requires line_start)",
			},
			"ignore_whitespace": map[string]interface{}{
				"type":        "boolean",
				"description": "Ignore whitespace changes (default: true)",
				"default":     true,
			},
			"show_email": map[string]interface{}{
				"type":        "boolean",
				"description": "Show author email addresses",
				"default":     false,
			},
			"show_date": map[string]interface{}{
				"type":        "boolean",
				"description": "Show commit dates in output",
				"default":     true,
			},
		},
	}

	examples := []string{
		`{"file": "main.go"}`,
		`{"file": "pkg/agent/core.go", "line_start": 100, "line_end": 150}`,
		`{"file": "README.md", "ignore_whitespace": false, "show_email": true}`,
		`{"file": "cmd/guild/chat.go", "show_date": true}`,
	}

	baseTool := tools.NewBaseTool(
		"git_blame",
		"Show authorship information for each line of a file",
		schema,
		"version_control",
		false,
		examples,
	)

	return &GitBlameTool{
		BaseTool:      baseTool,
		workspacePath: workspacePath,
	}
}

// Execute runs the git blame command with the specified parameters
func (t *GitBlameTool) Execute(ctx context.Context, input string) (*tools.ToolResult, error) {
	// Parse input
	var params GitBlameInput
	if err := json.Unmarshal([]byte(input), &params); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInvalidInput, "invalid input format").
			WithComponent("tools.git").
			WithOperation("execute_blame")
	}

	// Validate required parameters
	if params.File == "" {
		return nil, gerror.New(gerror.ErrCodeInvalidInput, "file parameter is required", nil).
			WithComponent("tools.git").
			WithOperation("execute_blame")
	}

	// Verify workspace is a git repository
	if !isGitRepository(t.workspacePath) {
		return nil, gerror.New(gerror.ErrCodeInvalidInput, "workspace is not a git repository", nil).
			WithComponent("tools.git").
			WithOperation("execute_blame")
	}

	// Validate file path is within workspace
	if err := validatePathWithBase(t.workspacePath, params.File); err != nil {
		return nil, err
	}

	// Check if file exists
	filePath := filepath.Join(t.workspacePath, params.File)
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil, gerror.New(gerror.ErrCodeNotFound, fmt.Sprintf("file not found: %s", params.File), nil).
			WithComponent("tools.git").
			WithOperation("execute_blame")
	}

	// Validate line range parameters
	if (params.LineStart > 0 && params.LineEnd == 0) || (params.LineStart == 0 && params.LineEnd > 0) {
		return nil, gerror.New(gerror.ErrCodeInvalidInput, "both line_start and line_end must be specified together", nil).
			WithComponent("tools.git").
			WithOperation("execute_blame")
	}

	if params.LineStart > params.LineEnd && params.LineEnd > 0 {
		return nil, gerror.New(gerror.ErrCodeInvalidInput, "line_start must be less than or equal to line_end", nil).
			WithComponent("tools.git").
			WithOperation("execute_blame")
	}

	// Set defaults
	if params.IgnoreWhitespace {
		params.IgnoreWhitespace = true // Default is true
	}

	// Build git blame command
	args := []string{"blame"}

	// Use porcelain format for easier parsing
	args = append(args, "--porcelain")

	// Ignore whitespace if requested
	if params.IgnoreWhitespace {
		args = append(args, "-w")
	}

	// Line range if specified
	if params.LineStart > 0 && params.LineEnd > 0 {
		args = append(args, fmt.Sprintf("-L%d,%d", params.LineStart, params.LineEnd))
	}

	// Add the file path
	args = append(args, params.File)

	// Execute command
	output, err := executeGitCommand(t.workspacePath, args...)
	if err != nil {
		// Check for common error cases
		errStr := err.Error()
		if strings.Contains(errStr, "no such path") {
			return nil, gerror.New(gerror.ErrCodeNotFound, fmt.Sprintf("file not found in git history: %s", params.File), nil).
				WithComponent("tools.git").
				WithOperation("execute_blame")
		}
		if strings.Contains(errStr, "not under version control") {
			return nil, gerror.New(gerror.ErrCodeInvalidInput, fmt.Sprintf("file is not tracked by git: %s", params.File), nil).
				WithComponent("tools.git").
				WithOperation("execute_blame")
		}
		return nil, formatGitError(err, "blame")
	}

	// For porcelain format, we need to get the actual file content separately
	// to show the blame with line content
	// Note: For now, we'll use simple git blame format instead

	// Parse blame information
	// For now, use simple git blame format instead of porcelain for better readability
	// Re-run with human-readable format
	args = []string{"blame"}

	if params.IgnoreWhitespace {
		args = append(args, "-w")
	}

	if params.LineStart > 0 && params.LineEnd > 0 {
		args = append(args, fmt.Sprintf("-L%d,%d", params.LineStart, params.LineEnd))
	}

	// Add formatting options
	if !params.ShowEmail {
		args = append(args, "--no-show-email")
	}

	args = append(args, params.File)

	output, err = executeGitCommand(t.workspacePath, args...)
	if err != nil {
		return nil, formatGitError(err, "blame")
	}

	// Parse the human-readable output
	blameInfo := parseGitBlame(output)

	// Format output based on options
	var formattedOutput string
	if params.ShowDate {
		formattedOutput = formatBlameOutputWithDates(blameInfo)
	} else {
		formattedOutput = formatBlameOutput(blameInfo)
	}

	// Prepare metadata
	metadata := map[string]string{
		"workspace_path": t.workspacePath,
		"file":           params.File,
		"unique_authors": fmt.Sprintf("%d", countUniqueAuthors(blameInfo)),
	}

	if len(blameInfo) > 0 {
		metadata["oldest_commit"] = findOldestCommit(blameInfo)
		metadata["total_lines"] = fmt.Sprintf("%d", len(blameInfo))
	}

	// Sanitize output
	formattedOutput = sanitizeGitOutput(formattedOutput)

	return tools.NewToolResult(formattedOutput, metadata, nil, nil), nil
}

// EstimateCost estimates the cost of running git blame
func (t *GitBlameTool) EstimateCost(params map[string]interface{}) float64 {
	// Cost depends on file size and line range
	// Since we don't know file size upfront, use line range as proxy

	lineStart := 0
	lineEnd := 0

	if val, ok := params["line_start"].(float64); ok {
		lineStart = int(val)
	} else if val, ok := params["line_start"].(int); ok {
		lineStart = val
	}

	if val, ok := params["line_end"].(float64); ok {
		lineEnd = int(val)
	} else if val, ok := params["line_end"].(int); ok {
		lineEnd = val
	}

	// If line range specified, use that for cost estimation
	if lineStart > 0 && lineEnd > 0 {
		lines := lineEnd - lineStart + 1
		if lines <= 50 {
			return 1.0 // Small range
		} else if lines <= 200 {
			return 2.0 // Medium range
		} else {
			return 3.0 // Large range
		}
	}

	// Without line range, assume medium cost
	return 2.0
}
