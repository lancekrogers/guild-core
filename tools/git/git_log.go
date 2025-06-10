package git

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/guild-ventures/guild-core/pkg/gerror"
	"github.com/guild-ventures/guild-core/tools"
)

// GitLogInput represents the input parameters for git log
type GitLogInput struct {
	MaxCommits int    `json:"max_commits,omitempty"`
	Path       string `json:"path,omitempty"`
	Author     string `json:"author,omitempty"`
	Since      string `json:"since,omitempty"`
	Until      string `json:"until,omitempty"`
	Grep       string `json:"grep,omitempty"`
	OneLine    bool   `json:"one_line,omitempty"`
	ShowDiff   bool   `json:"show_diff,omitempty"`
}

// GitLogTool implements git log functionality
type GitLogTool struct {
	*tools.BaseTool
}

// NewGitLogTool creates a new git log tool
func NewGitLogTool() *GitLogTool {
	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"max_commits": map[string]interface{}{
				"type":        "integer",
				"description": "Maximum number of commits to show (default: 20)",
				"default":     20,
			},
			"path": map[string]interface{}{
				"type":        "string",
				"description": "Filter commits affecting this path",
			},
			"author": map[string]interface{}{
				"type":        "string",
				"description": "Filter by commit author",
			},
			"since": map[string]interface{}{
				"type":        "string",
				"description": "Show commits since date (e.g., '2 weeks ago')",
			},
			"until": map[string]interface{}{
				"type":        "string",
				"description": "Show commits until date",
			},
			"grep": map[string]interface{}{
				"type":        "string",
				"description": "Filter commits by message content",
			},
			"one_line": map[string]interface{}{
				"type":        "boolean",
				"description": "Show compact one-line format (default: true)",
				"default":     true,
			},
			"show_diff": map[string]interface{}{
				"type":        "boolean",
				"description": "Include diff output for each commit",
				"default":     false,
			},
		},
	}

	examples := []string{
		`{"max_commits": 10}`,
		`{"author": "john.doe@example.com", "since": "1 week ago"}`,
		`{"path": "pkg/agent", "max_commits": 5}`,
		`{"grep": "fix", "one_line": true}`,
		`{"show_diff": true, "max_commits": 3}`,
	}

	baseTool := tools.NewBaseTool(
		"git_log",
		"View git commit history with filtering options",
		schema,
		"version_control",
		false,
		examples,
	)

	return &GitLogTool{
		BaseTool: baseTool,
	}
}

// Execute runs the git log command with the specified parameters
func (t *GitLogTool) Execute(ctx context.Context, input string) (*tools.ToolResult, error) {
	// Parse input
	var params GitLogInput
	if err := json.Unmarshal([]byte(input), &params); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInvalidInput, "invalid input format").
			WithComponent("tools.git").
			WithOperation("execute_log")
	}

	// Get workspace from context
	gitWs, err := getWorkspaceFromContext(ctx)
	if err != nil {
		return nil, err
	}

	// Set defaults
	if params.MaxCommits == 0 {
		params.MaxCommits = 20
	}

	// Build git log command
	args := []string{"log"}

	// Format options
	if params.OneLine || !params.ShowDiff {
		args = append(args, "--oneline")
		if !params.ShowDiff {
			args = append(args, "--graph")
		}
	} else {
		args = append(args, "--format=medium")
	}

	// Limit commits
	args = append(args, fmt.Sprintf("-n%d", params.MaxCommits))

	// Author filter
	if params.Author != "" {
		args = append(args, fmt.Sprintf("--author=%s", params.Author))
	}

	// Date filters
	if params.Since != "" {
		args = append(args, fmt.Sprintf("--since=%s", params.Since))
	}
	if params.Until != "" {
		args = append(args, fmt.Sprintf("--until=%s", params.Until))
	}

	// Message grep
	if params.Grep != "" {
		args = append(args, fmt.Sprintf("--grep=%s", params.Grep))
	}

	// Show diff if requested
	if params.ShowDiff {
		args = append(args, "-p", "--stat")
	}

	// Path filter (must come after --)
	if params.Path != "" {
		// Validate path is within workspace
		if err := validatePath(gitWs, params.Path); err != nil {
			return nil, err
		}
		args = append(args, "--", params.Path)
	}

	// Execute command
	output, err := executeGitCommand(gitWs.Path(), args...)
	if err != nil {
		// Check if it's just an empty repository
		if strings.Contains(err.Error(), "does not have any commits yet") {
			return tools.NewToolResult(
				"No commits found in repository",
				map[string]string{
					"workspace": gitWs.ID(),
					"branch":    gitWs.Branch(),
				},
				nil,
				nil,
			), nil
		}
		return nil, formatGitError(err, "log")
	}

	// Parse and format output
	var formattedOutput string
	metadata := map[string]string{
		"workspace": gitWs.ID(),
		"branch":    gitWs.Branch(),
	}

	if params.OneLine && !params.ShowDiff {
		// Parse simple format
		commits := parseGitLog(output)
		formattedOutput = formatCommitHistory(commits)
		metadata["commit_count"] = strconv.Itoa(len(commits))
	} else if params.ShowDiff {
		// For diff output, truncate if too large
		formattedOutput = truncateOutput(output, 500)
		metadata["truncated"] = "true"
	} else {
		// Parse verbose format
		commits := parseGitLogVerbose(output)
		formattedOutput = formatCommitHistoryVerbose(commits)
		metadata["commit_count"] = strconv.Itoa(len(commits))
	}

	// Sanitize output
	formattedOutput = sanitizeGitOutput(formattedOutput)

	return tools.NewToolResult(formattedOutput, metadata, nil, nil), nil
}

// EstimateCost estimates the cost of running git log
func (t *GitLogTool) EstimateCost(params map[string]interface{}) float64 {
	// Git operations are local, use Fibonacci scale based on commit count
	maxCommits := 20
	if val, ok := params["max_commits"].(float64); ok {
		maxCommits = int(val)
	} else if val, ok := params["max_commits"].(int); ok {
		maxCommits = val
	}

	// Fibonacci scale for cost estimation
	if maxCommits <= 10 {
		return 1.0 // Fibonacci(1)
	} else if maxCommits <= 50 {
		return 2.0 // Fibonacci(2)
	} else if maxCommits <= 100 {
		return 3.0 // Fibonacci(3)
	} else {
		return 5.0 // Fibonacci(4)
	}
}