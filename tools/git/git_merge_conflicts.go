package git

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/guild-ventures/guild-core/pkg/gerror"
	"github.com/guild-ventures/guild-core/tools"
)

// GitMergeConflictsInput represents the input parameters for merge conflicts tool
type GitMergeConflictsInput struct {
	Action   string `json:"action"`
	File     string `json:"file,omitempty"`
	Strategy string `json:"strategy,omitempty"`
	Preview  bool   `json:"preview,omitempty"`
}

// GitMergeConflictsTool implements git merge conflict detection and resolution
type GitMergeConflictsTool struct {
	*tools.BaseTool
	workspacePath string
}

// NewGitMergeConflictsTool creates a new git merge conflicts tool
func NewGitMergeConflictsTool(workspacePath string) *GitMergeConflictsTool {
	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"action": map[string]interface{}{
				"type":        "string",
				"description": "Action to perform: list, show, or resolve",
				"default":     "list",
				"enum":        []string{"list", "show", "resolve"},
			},
			"file": map[string]interface{}{
				"type":        "string",
				"description": "File path (required for show/resolve actions)",
			},
			"strategy": map[string]interface{}{
				"type":        "string",
				"description": "Resolution strategy: ours, theirs, or manual (required for resolve)",
				"enum":        []string{"ours", "theirs", "manual"},
			},
			"preview": map[string]interface{}{
				"type":        "boolean",
				"description": "Preview resolution without applying it",
				"default":     false,
			},
		},
	}

	examples := []string{
		`{"action": "list"}`,
		`{"action": "show", "file": "src/main.go"}`,
		`{"action": "resolve", "file": "README.md", "strategy": "ours"}`,
		`{"action": "resolve", "file": "pkg/config.go", "strategy": "theirs", "preview": true}`,
	}

	baseTool := tools.NewBaseTool(
		"git_merge_conflicts",
		"List, show, and help resolve merge conflicts in git repositories",
		schema,
		"version_control",
		false,
		examples,
	)

	return &GitMergeConflictsTool{
		BaseTool:      baseTool,
		workspacePath: workspacePath,
	}
}

// Execute runs the merge conflicts tool with the specified action
func (t *GitMergeConflictsTool) Execute(ctx context.Context, input string) (*tools.ToolResult, error) {
	// Parse input
	var params GitMergeConflictsInput
	if err := json.Unmarshal([]byte(input), &params); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInvalidInput, "invalid input format").
			WithComponent("tools.git").
			WithOperation("execute_merge_conflicts")
	}

	// Set default action
	if params.Action == "" {
		params.Action = "list"
	}

	// Verify workspace is a git repository
	if !isGitRepository(t.workspacePath) {
		return nil, gerror.New(gerror.ErrCodeInvalidInput, "workspace is not a git repository", nil).
			WithComponent("tools.git").
			WithOperation("execute_merge_conflicts")
	}

	// Route to appropriate handler based on action
	switch params.Action {
	case "list":
		return t.listConflicts()
	case "show":
		if params.File == "" {
			return nil, gerror.New(gerror.ErrCodeInvalidInput, "file parameter required for show action", nil).
				WithComponent("tools.git").
				WithOperation("execute_merge_conflicts")
		}
		return t.showConflict(params.File)
	case "resolve":
		if params.File == "" || params.Strategy == "" {
			return nil, gerror.New(gerror.ErrCodeInvalidInput, "file and strategy parameters required for resolve action", nil).
				WithComponent("tools.git").
				WithOperation("execute_merge_conflicts")
		}
		return t.resolveConflict(params.File, params.Strategy, params.Preview)
	default:
		return nil, gerror.New(gerror.ErrCodeInvalidInput, fmt.Sprintf("unknown action: %s", params.Action), nil).
			WithComponent("tools.git").
			WithOperation("execute_merge_conflicts")
	}
}

// listConflicts lists all files with merge conflicts
func (t *GitMergeConflictsTool) listConflicts() (*tools.ToolResult, error) {
	// Get files with merge conflicts using git diff
	output, err := executeGitCommand(t.workspacePath, "diff", "--name-only", "--diff-filter=U")
	if err != nil {
		// Check if we're in a merge state at all
		mergeHeadPath := filepath.Join(t.workspacePath, ".git", "MERGE_HEAD")
		if _, statErr := os.Stat(mergeHeadPath); os.IsNotExist(statErr) {
			return tools.NewToolResult(
				"No merge in progress",
				map[string]string{
					"workspace_path": t.workspacePath,
					"conflict_count": "0",
				},
				nil,
				nil,
			), nil
		}
		return nil, formatGitError(err, "diff")
	}

	// Parse conflicted files
	files := parseConflictedFiles(output)

	if len(files) == 0 {
		return tools.NewToolResult(
			"No merge conflicts found",
			map[string]string{
				"workspace_path": t.workspacePath,
				"conflict_count": "0",
			},
			nil,
			nil,
		), nil
	}

	// Analyze each conflict
	conflicts := []ConflictInfo{}
	for _, file := range files {
		info, err := t.analyzeConflict(file)
		if err != nil {
			// Skip files we can't analyze
			continue
		}
		conflicts = append(conflicts, info)
	}

	// Format output
	formattedOutput := formatConflictList(conflicts)

	// Calculate total conflict markers
	totalMarkers := countConflictMarkers(conflicts)

	metadata := map[string]string{
		"workspace_path":   t.workspacePath,
		"conflict_count":   fmt.Sprintf("%d", len(conflicts)),
		"total_markers":    fmt.Sprintf("%d", totalMarkers),
		"conflicted_files": strings.Join(files, ","),
	}

	return tools.NewToolResult(formattedOutput, metadata, nil, nil), nil
}

// showConflict shows detailed conflict information for a specific file
func (t *GitMergeConflictsTool) showConflict(file string) (*tools.ToolResult, error) {
	// Validate file path
	if err := validatePathWithBase(t.workspacePath, file); err != nil {
		return nil, err
	}

	// Read file content
	filePath := filepath.Join(t.workspacePath, file)
	content, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, gerror.New(gerror.ErrCodeNotFound, fmt.Sprintf("file not found: %s", file), nil).
				WithComponent("tools.git").
				WithOperation("show_conflict")
		}
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to read file").
			WithComponent("tools.git").
			WithOperation("show_conflict")
	}

	// Analyze conflict markers
	conflictInfo := parseConflictMarkers(string(content))
	conflictInfo.File = file

	if conflictInfo.ConflictCount == 0 {
		return tools.NewToolResult(
			fmt.Sprintf("No conflict markers found in %s", file),
			map[string]string{
				"workspace_path": t.workspacePath,
				"file":           file,
			},
			nil,
			nil,
		), nil
	}

	// Format detailed output
	formattedOutput := formatConflictDetails(conflictInfo)

	metadata := map[string]string{
		"workspace_path": t.workspacePath,
		"file":           file,
		"conflict_count": fmt.Sprintf("%d", conflictInfo.ConflictCount),
		"our_markers":    fmt.Sprintf("%d", conflictInfo.OurMarkers),
		"their_markers":  fmt.Sprintf("%d", conflictInfo.TheirMarkers),
	}

	return tools.NewToolResult(formattedOutput, metadata, nil, nil), nil
}

// resolveConflict attempts to resolve conflicts using the specified strategy
func (t *GitMergeConflictsTool) resolveConflict(file string, strategy string, preview bool) (*tools.ToolResult, error) {
	// Validate file path
	if err := validatePathWithBase(t.workspacePath, file); err != nil {
		return nil, err
	}

	// Check if file has conflicts
	output, err := executeGitCommand(t.workspacePath, "diff", "--name-only", "--diff-filter=U", file)
	if err != nil {
		return nil, formatGitError(err, "diff")
	}

	if strings.TrimSpace(output) == "" {
		return tools.NewToolResult(
			fmt.Sprintf("File %s has no merge conflicts", file),
			map[string]string{
				"workspace_path": t.workspacePath,
				"file":           file,
			},
			nil,
			nil,
		), nil
	}

	var resolvedContent string
	var metadata map[string]string

	switch strategy {
	case "ours":
		// Use --ours strategy
		if !preview {
			if _, err := executeGitCommand(t.workspacePath, "checkout", "--ours", file); err != nil {
				return nil, formatGitError(err, "checkout --ours")
			}
			// Stage the resolved file
			if _, err := executeGitCommand(t.workspacePath, "add", file); err != nil {
				return nil, formatGitError(err, "add")
			}
			resolvedContent = fmt.Sprintf("✓ Resolved %s using 'ours' strategy (keeping current branch changes)", file)
		} else {
			// Show what would happen
			content, _ := executeGitCommand(t.workspacePath, "show", fmt.Sprintf(":%d:%s", 2, file))
			resolvedContent = fmt.Sprintf("Preview: Would use 'ours' strategy for %s\n\nContent that would be kept:\n%s", file, truncateOutput(content, 50))
		}

	case "theirs":
		// Use --theirs strategy
		if !preview {
			if _, err := executeGitCommand(t.workspacePath, "checkout", "--theirs", file); err != nil {
				return nil, formatGitError(err, "checkout --theirs")
			}
			// Stage the resolved file
			if _, err := executeGitCommand(t.workspacePath, "add", file); err != nil {
				return nil, formatGitError(err, "add")
			}
			resolvedContent = fmt.Sprintf("✓ Resolved %s using 'theirs' strategy (keeping incoming changes)", file)
		} else {
			// Show what would happen
			content, _ := executeGitCommand(t.workspacePath, "show", fmt.Sprintf(":%d:%s", 3, file))
			resolvedContent = fmt.Sprintf("Preview: Would use 'theirs' strategy for %s\n\nContent that would be kept:\n%s", file, truncateOutput(content, 50))
		}

	case "manual":
		// Provide guidance for manual resolution
		resolvedContent = fmt.Sprintf(`Manual resolution required for %s

To resolve manually:
1. Open the file in your editor
2. Look for conflict markers:
   <<<<<<< HEAD (your changes)
   ...
   ======= 
   ...
   >>>>>>> branch-name (incoming changes)
3. Edit the file to keep the desired changes
4. Remove all conflict markers
5. Save the file
6. Run: git add %s
7. Continue with your merge/rebase`, file, file)

	default:
		return nil, gerror.New(gerror.ErrCodeInvalidInput, fmt.Sprintf("unknown strategy: %s", strategy), nil).
			WithComponent("tools.git").
			WithOperation("resolve_conflict")
	}

	metadata = map[string]string{
		"workspace_path": t.workspacePath,
		"file":           file,
		"strategy":       strategy,
		"preview":        fmt.Sprintf("%v", preview),
	}

	if !preview && strategy != "manual" {
		metadata["resolved"] = "true"

		// Check if all conflicts are resolved
		remainingConflicts, _ := executeGitCommand(t.workspacePath, "diff", "--name-only", "--diff-filter=U")
		if strings.TrimSpace(remainingConflicts) == "" {
			resolvedContent += "\n\n✅ All conflicts have been resolved! You can now commit the changes."
			metadata["all_resolved"] = "true"
		} else {
			conflictCount := len(strings.Split(strings.TrimSpace(remainingConflicts), "\n"))
			resolvedContent += fmt.Sprintf("\n\n⚠️  %d conflict(s) remaining in other files", conflictCount)
			metadata["remaining_conflicts"] = fmt.Sprintf("%d", conflictCount)
		}
	}

	return tools.NewToolResult(resolvedContent, metadata, nil, nil), nil
}

// analyzeConflict analyzes a single file for conflict information
func (t *GitMergeConflictsTool) analyzeConflict(file string) (ConflictInfo, error) {
	filePath := filepath.Join(t.workspacePath, file)
	content, err := os.ReadFile(filePath)
	if err != nil {
		return ConflictInfo{}, err
	}

	info := parseConflictMarkers(string(content))
	info.File = file
	return info, nil
}

// EstimateCost estimates the cost of merge conflict operations
func (t *GitMergeConflictsTool) EstimateCost(params map[string]interface{}) float64 {
	// Merge conflict operations are generally quick
	action := "list"
	if val, ok := params["action"].(string); ok {
		action = val
	}

	switch action {
	case "list":
		return 1.0 // Quick operation
	case "show":
		return 2.0 // Reading and parsing file
	case "resolve":
		// Resolution might involve more work
		if preview, ok := params["preview"].(bool); ok && preview {
			return 2.0 // Just preview
		}
		return 3.0 // Actual resolution
	default:
		return 1.0
	}
}
