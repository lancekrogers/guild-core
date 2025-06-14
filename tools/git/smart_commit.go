package git

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/guild-ventures/guild-core/pkg/gerror"
	"github.com/guild-ventures/guild-core/pkg/providers/interfaces"
	"github.com/guild-ventures/guild-core/tools"
)

// SmartCommitTool creates git commits with AI-generated messages
type SmartCommitTool struct {
	*tools.BaseTool
	llmProvider interfaces.AIProvider
}

// SmartCommitInput represents input for smart git commits
type SmartCommitInput struct {
	Files             []string          `json:"files,omitempty"`               // Files to commit (empty for all staged)
	Message           string            `json:"message,omitempty"`             // Override AI message
	Conventional      bool              `json:"conventional,omitempty"`        // Use conventional commit format
	MaxMessageLength  int               `json:"max_message_length,omitempty"`  // Max first line length
	IncludeDetails    bool              `json:"include_details,omitempty"`     // Include detailed description
	AutoStage         bool              `json:"auto_stage,omitempty"`          // Auto-stage files before commit
	Push              bool              `json:"push,omitempty"`                // Push after commit
	Amend             bool              `json:"amend,omitempty"`               // Amend last commit
	AnalyzeContext    bool              `json:"analyze_context,omitempty"`     // Include file context in analysis
	Environment       map[string]string `json:"environment,omitempty"`         // Environment variables for git
}

// SmartCommitResult represents the result of a smart commit
type SmartCommitResult struct {
	CommitHash       string            `json:"commit_hash"`
	Message          string            `json:"message"`
	GeneratedMessage bool              `json:"generated_message"`
	FilesCommitted   []string          `json:"files_committed"`
	FilesStaged      []string          `json:"files_staged,omitempty"`
	DiffAnalysis     *DiffAnalysis     `json:"diff_analysis,omitempty"`
	Pushed           bool              `json:"pushed,omitempty"`
	Amended          bool              `json:"amended,omitempty"`
	Stats            *CommitStats      `json:"stats"`
	Suggestions      []string          `json:"suggestions,omitempty"`
	Metadata         map[string]string `json:"metadata"`
}

// DiffAnalysis represents analysis of the changes being committed
type DiffAnalysis struct {
	ChangeType        string            `json:"change_type"`        // feat, fix, docs, style, refactor, test, chore
	Scope             string            `json:"scope,omitempty"`    // Component/module affected
	BreakingChange    bool              `json:"breaking_change"`
	LinesAdded        int               `json:"lines_added"`
	LinesDeleted      int               `json:"lines_deleted"`
	FilesChanged      int               `json:"files_changed"`
	Languages         []string          `json:"languages"`
	TestsIncluded     bool              `json:"tests_included"`
	DocsUpdated       bool              `json:"docs_updated"`
	Summary           string            `json:"summary"`
	KeyChanges        []string          `json:"key_changes"`
}

// CommitStats represents statistics about the commit
type CommitStats struct {
	Insertions    int               `json:"insertions"`
	Deletions     int               `json:"deletions"`
	FilesChanged  int               `json:"files_changed"`
	Duration      time.Duration     `json:"duration"`
	MessageLength int               `json:"message_length"`
}

// NewSmartCommitTool creates a new smart commit tool
func NewSmartCommitTool(llmProvider interfaces.AIProvider) *SmartCommitTool {
	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"files": map[string]interface{}{
				"type":        "array",
				"items":       map[string]interface{}{"type": "string"},
				"description": "Files to commit (empty for all staged files)",
			},
			"message": map[string]interface{}{
				"type":        "string",
				"description": "Override AI-generated message with custom message",
			},
			"conventional": map[string]interface{}{
				"type":        "boolean",
				"default":     true,
				"description": "Use conventional commit format (type: description)",
			},
			"max_message_length": map[string]interface{}{
				"type":        "integer",
				"default":     72,
				"description": "Maximum length for commit message first line",
			},
			"include_details": map[string]interface{}{
				"type":        "boolean",
				"default":     false,
				"description": "Include detailed description in commit message",
			},
			"auto_stage": map[string]interface{}{
				"type":        "boolean",
				"default":     false,
				"description": "Automatically stage files before committing",
			},
			"push": map[string]interface{}{
				"type":        "boolean",
				"default":     false,
				"description": "Push commit to remote after creating",
			},
			"amend": map[string]interface{}{
				"type":        "boolean",
				"default":     false,
				"description": "Amend the last commit instead of creating new one",
			},
			"analyze_context": map[string]interface{}{
				"type":        "boolean",
				"default":     true,
				"description": "Analyze file context for better message generation",
			},
		},
	}

	examples := []string{
		`{"auto_stage": true, "conventional": true}`,
		`{"files": ["src/main.go", "README.md"], "include_details": true}`,
		`{"message": "fix: resolve authentication bug", "push": true}`,
		`{"amend": true, "message": "docs: update installation guide"}`,
	}

	baseTool := tools.NewBaseTool(
		"smart_commit",
		"Create git commits with AI-generated messages and intelligent analysis",
		schema,
		"git",
		false,
		examples,
	)

	return &SmartCommitTool{
		BaseTool:    baseTool,
		llmProvider: llmProvider,
	}
}

// Execute runs the smart commit tool
func (t *SmartCommitTool) Execute(ctx context.Context, input string) (*tools.ToolResult, error) {
	var params SmartCommitInput
	if err := json.Unmarshal([]byte(input), &params); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInvalidInput, "invalid input").
			WithComponent("smart_commit_tool").
			WithOperation("execute")
	}

	// Set defaults
	if params.MaxMessageLength == 0 {
		params.MaxMessageLength = 72
	}

	startTime := time.Now()

	// Auto-stage files if requested
	if params.AutoStage {
		stagedFiles, err := t.stageFiles(params.Files)
		if err != nil {
			return nil, err
		}
		params.Files = stagedFiles
	}

	// Get diff for analysis
	diff, err := t.getDiff(params.Files, params.Amend)
	if err != nil {
		return nil, err
	}

	if strings.TrimSpace(diff) == "" {
		return nil, gerror.New(gerror.ErrCodeInvalidInput, "no changes to commit", nil).
			WithComponent("smart_commit_tool").
			WithOperation("execute")
	}

	// Analyze the diff
	analysis := t.analyzeDiff(diff, params.AnalyzeContext)

	// Generate or use provided message
	message := params.Message
	generatedMessage := false
	if message == "" {
		var err error
		message, err = t.generateCommitMessage(ctx, diff, analysis, params)
		if err != nil {
			return nil, err
		}
		generatedMessage = true
	}

	// Create the commit
	commitHash, err := t.createCommit(message, params)
	if err != nil {
		return nil, err
	}

	// Push if requested
	pushed := false
	if params.Push && !params.Amend {
		if err := t.pushCommit(); err != nil {
			// Don't fail the entire operation if push fails
			analysis.Summary += " (push failed: " + err.Error() + ")"
		} else {
			pushed = true
		}
	}

	duration := time.Since(startTime)

	// Generate suggestions
	suggestions := t.generateSuggestions(analysis, message, params)

	result := &SmartCommitResult{
		CommitHash:       commitHash,
		Message:          message,
		GeneratedMessage: generatedMessage,
		FilesCommitted:   params.Files,
		DiffAnalysis:     analysis,
		Pushed:           pushed,
		Amended:          params.Amend,
		Stats: &CommitStats{
			Insertions:    analysis.LinesAdded,
			Deletions:     analysis.LinesDeleted,
			FilesChanged:  analysis.FilesChanged,
			Duration:      duration,
			MessageLength: len(strings.Split(message, "\n")[0]),
		},
		Suggestions: suggestions,
		Metadata: map[string]string{
			"conventional_format": fmt.Sprintf("%t", params.Conventional),
			"generated_message":   fmt.Sprintf("%t", generatedMessage),
			"change_type":         analysis.ChangeType,
			"scope":               analysis.Scope,
		},
	}

	// Convert result to ToolResult
	resultJSON, _ := json.Marshal(result)
	metadata := map[string]string{
		"commit_hash":  commitHash,
		"change_type":  analysis.ChangeType,
		"files_count":  fmt.Sprintf("%d", analysis.FilesChanged),
		"lines_added":  fmt.Sprintf("%d", analysis.LinesAdded),
		"lines_deleted": fmt.Sprintf("%d", analysis.LinesDeleted),
		"generated":    fmt.Sprintf("%t", generatedMessage),
	}

	return tools.NewToolResult(string(resultJSON), metadata, nil, nil), nil
}

// stageFiles stages the specified files or all changes if no files specified
func (t *SmartCommitTool) stageFiles(files []string) ([]string, error) {
	var cmd *exec.Cmd
	if len(files) == 0 {
		cmd = exec.Command("git", "add", ".")
	} else {
		args := append([]string{"add"}, files...)
		cmd = exec.Command("git", args...)
	}

	if err := cmd.Run(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to stage files").
			WithComponent("smart_commit_tool").
			WithOperation("stageFiles")
	}

	// Get list of staged files
	cmd = exec.Command("git", "diff", "--cached", "--name-only")
	output, err := cmd.Output()
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to get staged files").
			WithComponent("smart_commit_tool").
			WithOperation("stageFiles")
	}

	stagedFiles := strings.Fields(string(output))
	return stagedFiles, nil
}

// getDiff gets the diff for the changes to be committed
func (t *SmartCommitTool) getDiff(files []string, amend bool) (string, error) {
	var cmd *exec.Cmd
	
	if amend {
		cmd = exec.Command("git", "diff", "HEAD~1")
	} else if len(files) > 0 {
		args := append([]string{"diff", "--cached"}, files...)
		cmd = exec.Command("git", args...)
	} else {
		cmd = exec.Command("git", "diff", "--cached")
	}

	output, err := cmd.Output()
	if err != nil {
		return "", gerror.Wrap(err, gerror.ErrCodeInternal, "failed to get diff").
			WithComponent("smart_commit_tool").
			WithOperation("getDiff")
	}

	return string(output), nil
}

// analyzeDiff analyzes the diff to understand the type of changes
func (t *SmartCommitTool) analyzeDiff(diff string, includeContext bool) *DiffAnalysis {
	analysis := &DiffAnalysis{
		KeyChanges: []string{},
		Languages:  []string{},
	}

	lines := strings.Split(diff, "\n")
	var currentFile string
	languageSet := make(map[string]bool)

	for _, line := range lines {
		// File headers
		if strings.HasPrefix(line, "+++") || strings.HasPrefix(line, "---") {
			if strings.HasPrefix(line, "+++ b/") {
				currentFile = strings.TrimPrefix(line, "+++ b/")
				analysis.FilesChanged++
				
				// Detect language
				if lang := detectLanguageFromFile(currentFile); lang != "" {
					languageSet[lang] = true
				}
			}
		}

		// Count line changes
		if strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "+++") {
			analysis.LinesAdded++
		} else if strings.HasPrefix(line, "-") && !strings.HasPrefix(line, "---") {
			analysis.LinesDeleted++
		}

		// Detect change patterns
		if strings.Contains(line, "test") || strings.Contains(line, "spec") {
			analysis.TestsIncluded = true
		}
		if strings.Contains(currentFile, "README") || strings.Contains(currentFile, ".md") {
			analysis.DocsUpdated = true
		}
		if strings.Contains(line, "BREAKING CHANGE") {
			analysis.BreakingChange = true
		}
	}

	// Convert language set to slice
	for lang := range languageSet {
		analysis.Languages = append(analysis.Languages, lang)
	}

	// Determine change type
	analysis.ChangeType = t.determineChangeType(diff, analysis)
	analysis.Scope = t.determineScope(diff, analysis)
	analysis.Summary = t.createSummary(analysis)

	return analysis
}

// generateCommitMessage uses AI to generate an appropriate commit message
func (t *SmartCommitTool) generateCommitMessage(ctx context.Context, diff string, analysis *DiffAnalysis, params SmartCommitInput) (string, error) {
	if t.llmProvider == nil {
		// Fallback to simple message if no LLM provider
		return t.generateFallbackMessage(analysis), nil
	}

	prompt := t.buildCommitPrompt(diff, analysis, params)
	
	// Create a simple chat request
	req := interfaces.ChatRequest{
		Model: "gpt-3.5-turbo", // Default model
		Messages: []interfaces.ChatMessage{
			{
				Role:    "user",
				Content: prompt,
			},
		},
		MaxTokens:   150,
		Temperature: 0.3,
	}
	
	response, err := t.llmProvider.ChatCompletion(ctx, req)
	if err != nil {
		// Fallback to simple message on LLM failure
		return t.generateFallbackMessage(analysis), nil
	}

	var message string
	if len(response.Choices) > 0 {
		message = strings.TrimSpace(response.Choices[0].Message.Content)
	} else {
		return t.generateFallbackMessage(analysis), nil
	}
	
	// Validate and clean up the message
	message = t.cleanupMessage(message, params.MaxMessageLength, params.Conventional)
	
	return message, nil
}

// buildCommitPrompt creates a prompt for AI commit message generation
func (t *SmartCommitTool) buildCommitPrompt(diff string, analysis *DiffAnalysis, params SmartCommitInput) string {
	prompt := "Generate a concise git commit message for the following changes.\n\n"
	
	if params.Conventional {
		prompt += "Use conventional commit format: type(scope): description\n"
		prompt += "Types: feat, fix, docs, style, refactor, test, chore, ci, build, perf\n\n"
	}
	
	prompt += fmt.Sprintf("Maximum first line length: %d characters\n", params.MaxMessageLength)
	
	if analysis != nil {
		prompt += fmt.Sprintf("\nChange Analysis:\n")
		prompt += fmt.Sprintf("- Files changed: %d\n", analysis.FilesChanged)
		prompt += fmt.Sprintf("- Lines added: %d, deleted: %d\n", analysis.LinesAdded, analysis.LinesDeleted)
		prompt += fmt.Sprintf("- Languages: %s\n", strings.Join(analysis.Languages, ", "))
		prompt += fmt.Sprintf("- Tests included: %t\n", analysis.TestsIncluded)
		prompt += fmt.Sprintf("- Docs updated: %t\n", analysis.DocsUpdated)
		if analysis.BreakingChange {
			prompt += "- BREAKING CHANGE detected\n"
		}
	}
	
	// Truncate diff if too long
	if len(diff) > 2000 {
		diff = diff[:2000] + "\n... (truncated)"
	}
	
	prompt += fmt.Sprintf("\nGit diff:\n```\n%s\n```\n", diff)
	
	if params.IncludeDetails {
		prompt += "\nProvide a detailed description after the first line, separated by a blank line."
	}
	
	prompt += "\nCommit message:"
	
	return prompt
}

// generateFallbackMessage creates a simple message when AI is unavailable
func (t *SmartCommitTool) generateFallbackMessage(analysis *DiffAnalysis) string {
	if analysis == nil {
		return "chore: update files"
	}
	
	changeType := analysis.ChangeType
	if changeType == "" {
		changeType = "chore"
	}
	
	scope := ""
	if analysis.Scope != "" {
		scope = fmt.Sprintf("(%s)", analysis.Scope)
	}
	
	description := "update files"
	if len(analysis.KeyChanges) > 0 {
		description = analysis.KeyChanges[0]
	} else if analysis.FilesChanged == 1 {
		description = "update file"
	} else {
		description = fmt.Sprintf("update %d files", analysis.FilesChanged)
	}
	
	return fmt.Sprintf("%s%s: %s", changeType, scope, description)
}

// cleanupMessage cleans and validates the generated message
func (t *SmartCommitTool) cleanupMessage(message string, maxLength int, conventional bool) string {
	lines := strings.Split(message, "\n")
	if len(lines) == 0 {
		return "chore: update files"
	}
	
	firstLine := strings.TrimSpace(lines[0])
	
	// Remove any markdown formatting or quotes
	firstLine = strings.Trim(firstLine, "`\"'")
	
	// Ensure conventional format if required
	if conventional && !regexp.MustCompile(`^(feat|fix|docs|style|refactor|test|chore|ci|build|perf)(\(.+\))?: .+`).MatchString(firstLine) {
		if !strings.Contains(firstLine, ":") {
			firstLine = "chore: " + firstLine
		}
	}
	
	// Truncate if too long
	if len(firstLine) > maxLength {
		firstLine = firstLine[:maxLength-3] + "..."
	}
	
	// Rebuild message
	result := firstLine
	if len(lines) > 1 {
		for i, line := range lines[1:] {
			if i == 0 && strings.TrimSpace(line) == "" {
				result += "\n"
			} else if strings.TrimSpace(line) != "" {
				result += "\n" + line
			}
		}
	}
	
	return result
}

// createCommit creates the actual git commit
func (t *SmartCommitTool) createCommit(message string, params SmartCommitInput) (string, error) {
	var cmd *exec.Cmd
	
	if params.Amend {
		cmd = exec.Command("git", "commit", "--amend", "-m", message)
	} else {
		cmd = exec.Command("git", "commit", "-m", message)
	}
	
	// Set environment variables if provided
	if len(params.Environment) > 0 {
		env := cmd.Env
		for key, value := range params.Environment {
			env = append(env, fmt.Sprintf("%s=%s", key, value))
		}
		cmd.Env = env
	}
	
	if err := cmd.Run(); err != nil {
		return "", gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create commit").
			WithComponent("smart_commit_tool").
			WithOperation("createCommit")
	}
	
	// Get the commit hash
	cmd = exec.Command("git", "rev-parse", "HEAD")
	output, err := cmd.Output()
	if err != nil {
		return "", gerror.Wrap(err, gerror.ErrCodeInternal, "failed to get commit hash").
			WithComponent("smart_commit_tool").
			WithOperation("createCommit")
	}
	
	return strings.TrimSpace(string(output)), nil
}

// pushCommit pushes the commit to the remote
func (t *SmartCommitTool) pushCommit() error {
	cmd := exec.Command("git", "push")
	if err := cmd.Run(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to push commit").
			WithComponent("smart_commit_tool").
			WithOperation("pushCommit")
	}
	return nil
}

// Helper functions for analysis
func (t *SmartCommitTool) determineChangeType(diff string, analysis *DiffAnalysis) string {
	diff = strings.ToLower(diff)
	
	if strings.Contains(diff, "test") || strings.Contains(diff, "spec") {
		return "test"
	}
	if strings.Contains(diff, "readme") || strings.Contains(diff, ".md") || strings.Contains(diff, "doc") {
		return "docs"
	}
	if strings.Contains(diff, "fix") || strings.Contains(diff, "bug") || strings.Contains(diff, "error") {
		return "fix"
	}
	if strings.Contains(diff, "style") || strings.Contains(diff, "format") || strings.Contains(diff, "lint") {
		return "style"
	}
	if strings.Contains(diff, "refactor") || strings.Contains(diff, "rename") || strings.Contains(diff, "move") {
		return "refactor"
	}
	if analysis.LinesAdded > analysis.LinesDeleted*2 {
		return "feat"
	}
	
	return "chore"
}

func (t *SmartCommitTool) determineScope(diff string, analysis *DiffAnalysis) string {
	// Extract common directory or component names
	if len(analysis.Languages) == 1 {
		return analysis.Languages[0]
	}
	
	// Look for common patterns in file paths
	if strings.Contains(diff, "/api/") || strings.Contains(diff, "api.") {
		return "api"
	}
	if strings.Contains(diff, "/ui/") || strings.Contains(diff, "/frontend/") {
		return "ui"
	}
	if strings.Contains(diff, "/auth/") || strings.Contains(diff, "auth.") {
		return "auth"
	}
	if strings.Contains(diff, "/test/") || strings.Contains(diff, "test.") {
		return "test"
	}
	
	return ""
}

func (t *SmartCommitTool) createSummary(analysis *DiffAnalysis) string {
	if analysis.FilesChanged == 1 {
		return "Modified 1 file"
	}
	return fmt.Sprintf("Modified %d files", analysis.FilesChanged)
}

func (t *SmartCommitTool) generateSuggestions(analysis *DiffAnalysis, message string, params SmartCommitInput) []string {
	var suggestions []string
	
	if analysis.TestsIncluded && !strings.Contains(message, "test") {
		suggestions = append(suggestions, "Consider mentioning test changes in commit message")
	}
	
	if analysis.BreakingChange && !strings.Contains(message, "BREAKING") {
		suggestions = append(suggestions, "Add 'BREAKING CHANGE' to message footer for breaking changes")
	}
	
	if len(analysis.Languages) > 1 {
		suggestions = append(suggestions, "Consider splitting multi-language changes into separate commits")
	}
	
	if analysis.FilesChanged > 10 {
		suggestions = append(suggestions, "Large commit - consider splitting into smaller, focused commits")
	}
	
	return suggestions
}

func detectLanguageFromFile(filename string) string {
	ext := filepath.Ext(filename)
	switch ext {
	case ".go":
		return "go"
	case ".js", ".jsx":
		return "javascript"
	case ".ts", ".tsx":
		return "typescript"
	case ".py":
		return "python"
	case ".rs":
		return "rust"
	case ".java":
		return "java"
	case ".rb":
		return "ruby"
	case ".php":
		return "php"
	case ".c", ".h":
		return "c"
	case ".cpp", ".hpp", ".cc":
		return "cpp"
	case ".cs":
		return "csharp"
	default:
		return ""
	}
}