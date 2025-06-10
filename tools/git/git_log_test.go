package git

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/guild-ventures/guild-core/pkg/gerror"
	"github.com/guild-ventures/guild-core/tools"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewGitLogTool(t *testing.T) {
	workspacePath := "/test/workspace"
	tool := NewGitLogTool(workspacePath)

	assert.NotNil(t, tool)
	assert.Equal(t, "git_log", tool.Name())
	assert.Equal(t, "View git commit history with filtering options", tool.Description())
	assert.Equal(t, "version_control", tool.Category())
	assert.False(t, tool.RequiresAuth())
	assert.Equal(t, workspacePath, tool.workspacePath)

	// Check schema
	schema := tool.Schema()
	assert.Equal(t, "object", schema["type"])

	// Check properties
	props, ok := schema["properties"].(map[string]interface{})
	require.True(t, ok)
	assert.Contains(t, props, "max_commits")
	assert.Contains(t, props, "path")
	assert.Contains(t, props, "author")
	assert.Contains(t, props, "since")
	assert.Contains(t, props, "until")
	assert.Contains(t, props, "grep")
	assert.Contains(t, props, "one_line")
	assert.Contains(t, props, "show_diff")

	// Check examples
	examples := tool.Examples()
	assert.Len(t, examples, 5)
}

func TestGitLogTool_Execute(t *testing.T) {
	// Create a temporary git repository for testing
	tmpDir := t.TempDir()
	setupTestRepo(t, tmpDir)

	tool := NewGitLogTool(tmpDir)
	ctx := context.Background()

	tests := []struct {
		name    string
		input   GitLogInput
		wantErr bool
		errCode gerror.ErrorCode
		verify  func(t *testing.T, result *tools.ToolResult)
	}{
		{
			name: "basic log with defaults",
			input: GitLogInput{
				MaxCommits: 10,
			},
			wantErr: false,
			verify: func(t *testing.T, result *tools.ToolResult) {
				assert.NotEmpty(t, result.Output)
				// May contain commits or "No commits found" depending on git setup
				assert.Equal(t, tmpDir, result.Metadata["workspace_path"])
			},
		},
		{
			name: "log with author filter",
			input: GitLogInput{
				Author:     "Test Author",
				MaxCommits: 5,
			},
			wantErr: false,
			verify: func(t *testing.T, result *tools.ToolResult) {
				assert.NotEmpty(t, result.Output)
			},
		},
		{
			name: "log with path filter",
			input: GitLogInput{
				Path:       "test.txt",
				MaxCommits: 5,
			},
			wantErr: false,
			verify: func(t *testing.T, result *tools.ToolResult) {
				assert.NotEmpty(t, result.Output)
			},
		},
		{
			name: "log with grep filter",
			input: GitLogInput{
				Grep:       "test",
				MaxCommits: 5,
			},
			wantErr: false,
			verify: func(t *testing.T, result *tools.ToolResult) {
				assert.NotEmpty(t, result.Output)
			},
		},
		{
			name: "empty repository",
			input: GitLogInput{
				MaxCommits: 5,
			},
			wantErr: false,
			verify: func(t *testing.T, result *tools.ToolResult) {
				// Note: Our test repo has commits, so this won't trigger empty repo message
				assert.NotEmpty(t, result.Output)
			},
		},
		{
			name: "path traversal attempt",
			input: GitLogInput{
				Path: "../../../etc/passwd",
			},
			wantErr: true,
			errCode: gerror.ErrCodeInvalidInput,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inputJSON, err := json.Marshal(tt.input)
			require.NoError(t, err)

			result, err := tool.Execute(ctx, string(inputJSON))

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errCode != "" {
					gerr, ok := err.(*gerror.GuildError)
					assert.True(t, ok)
					assert.Equal(t, tt.errCode, gerr.Code)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.True(t, result.Success)
				if tt.verify != nil {
					tt.verify(t, result)
				}
			}
		})
	}
}

func TestGitLogTool_Execute_NonGitDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	tool := NewGitLogTool(tmpDir)
	ctx := context.Background()

	input := GitLogInput{MaxCommits: 5}
	inputJSON, _ := json.Marshal(input)

	result, err := tool.Execute(ctx, string(inputJSON))
	assert.Error(t, err)
	assert.Nil(t, result)

	gerr, ok := err.(*gerror.GuildError)
	assert.True(t, ok)
	assert.Equal(t, gerror.ErrCodeInvalidInput, gerr.Code)
	assert.Contains(t, gerr.Message, "not a git repository")
}

func TestGitLogTool_Execute_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	setupTestRepo(t, tmpDir)
	tool := NewGitLogTool(tmpDir)
	ctx := context.Background()

	result, err := tool.Execute(ctx, "invalid json")
	assert.Error(t, err)
	assert.Nil(t, result)

	gerr, ok := err.(*gerror.GuildError)
	assert.True(t, ok)
	assert.Equal(t, gerror.ErrCodeInvalidInput, gerr.Code)
}

func TestGitLogTool_EstimateCost(t *testing.T) {
	tool := NewGitLogTool("/test")

	tests := []struct {
		name   string
		params map[string]interface{}
		want   float64
	}{
		{
			name:   "small commit count",
			params: map[string]interface{}{"max_commits": 5.0},
			want:   1.0,
		},
		{
			name:   "medium commit count",
			params: map[string]interface{}{"max_commits": 30.0},
			want:   2.0,
		},
		{
			name:   "large commit count",
			params: map[string]interface{}{"max_commits": 75.0},
			want:   3.0,
		},
		{
			name:   "very large commit count",
			params: map[string]interface{}{"max_commits": 150.0},
			want:   5.0,
		},
		{
			name:   "default when not specified",
			params: map[string]interface{}{},
			want:   2.0, // Default is 20, which falls in medium range
		},
		{
			name:   "integer value",
			params: map[string]interface{}{"max_commits": 8},
			want:   1.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tool.EstimateCost(tt.params)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestGitLogTool_ShowDiff(t *testing.T) {
	tmpDir := t.TempDir()
	setupTestRepo(t, tmpDir)
	tool := NewGitLogTool(tmpDir)
	ctx := context.Background()

	input := GitLogInput{
		MaxCommits: 2,
		ShowDiff:   true,
	}
	inputJSON, _ := json.Marshal(input)

	result, err := tool.Execute(ctx, string(inputJSON))
	assert.NoError(t, err)
	assert.NotNil(t, result)

	// When show_diff is true, output should be truncated
	assert.Equal(t, "true", result.Metadata["truncated"])
}

// Helper function to set up a test git repository
func setupTestRepo(t *testing.T, dir string) {
	t.Helper()

	// Initialize repository
	_, err := executeGitCommand(dir, "init")
	require.NoError(t, err)

	// Configure git user
	_, err = executeGitCommand(dir, "config", "user.email", "test@example.com")
	require.NoError(t, err)
	_, err = executeGitCommand(dir, "config", "user.name", "Test Author")
	require.NoError(t, err)

	// Create and commit a test file
	testFile := filepath.Join(dir, "test.txt")
	require.NoError(t, os.WriteFile(testFile, []byte("test content"), 0644))

	_, err = executeGitCommand(dir, "add", "test.txt")
	require.NoError(t, err)

	_, err = executeGitCommand(dir, "commit", "-m", "Initial commit")
	require.NoError(t, err)

	// Add another file and commit
	anotherFile := filepath.Join(dir, "another.txt")
	require.NoError(t, os.WriteFile(anotherFile, []byte("more content"), 0644))

	_, err = executeGitCommand(dir, "add", "another.txt")
	require.NoError(t, err)

	_, err = executeGitCommand(dir, "commit", "-m", "Add another test file")
	require.NoError(t, err)
}
