// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package git

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/lancekrogers/guild-core/pkg/gerror"
	"github.com/lancekrogers/guild-core/tools"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewGitBlameTool(t *testing.T) {
	workspacePath := "/test/workspace"
	tool := NewGitBlameTool(workspacePath)

	assert.NotNil(t, tool)
	assert.Equal(t, "git_blame", tool.Name())
	assert.Equal(t, "Show authorship information for each line of a file", tool.Description())
	assert.Equal(t, "version_control", tool.Category())
	assert.False(t, tool.RequiresAuth())
	assert.Equal(t, workspacePath, tool.workspacePath)

	// Check schema
	schema := tool.Schema()
	assert.Equal(t, "object", schema["type"])

	// Check required fields
	required, ok := schema["required"].([]string)
	require.True(t, ok)
	assert.Contains(t, required, "file")

	// Check properties
	props, ok := schema["properties"].(map[string]interface{})
	require.True(t, ok)
	assert.Contains(t, props, "file")
	assert.Contains(t, props, "line_start")
	assert.Contains(t, props, "line_end")
	assert.Contains(t, props, "ignore_whitespace")
	assert.Contains(t, props, "show_email")
	assert.Contains(t, props, "show_date")

	// Check examples
	examples := tool.Examples()
	assert.Len(t, examples, 4)
}

func TestGitBlameTool_Execute(t *testing.T) {
	tmpDir := t.TempDir()
	setupTestBlameRepo(t, tmpDir)

	tool := NewGitBlameTool(tmpDir)
	ctx := context.Background()

	tests := []struct {
		name    string
		input   GitBlameInput
		wantErr bool
		errCode gerror.ErrorCode
		verify  func(t *testing.T, result *tools.ToolResult)
	}{
		{
			name: "basic blame",
			input: GitBlameInput{
				File: "test.txt",
			},
			wantErr: false,
			verify: func(t *testing.T, result *tools.ToolResult) {
				assert.NotEmpty(t, result.Output)
				assert.Equal(t, tmpDir, result.Metadata["workspace_path"])
				assert.Equal(t, "test.txt", result.Metadata["file"])
				assert.Contains(t, result.Metadata, "unique_authors")
			},
		},
		{
			name: "blame with line range",
			input: GitBlameInput{
				File:      "test.txt",
				LineStart: 1,
				LineEnd:   2,
			},
			wantErr: false,
			verify: func(t *testing.T, result *tools.ToolResult) {
				assert.NotEmpty(t, result.Output)
			},
		},
		{
			name: "blame with email and date",
			input: GitBlameInput{
				File:      "test.txt",
				ShowEmail: true,
				ShowDate:  true,
			},
			wantErr: false,
			verify: func(t *testing.T, result *tools.ToolResult) {
				assert.NotEmpty(t, result.Output)
			},
		},
		{
			name:  "missing file parameter",
			input: GitBlameInput{
				// File is required but missing
			},
			wantErr: true,
			errCode: gerror.ErrCodeInvalidInput,
		},
		{
			name: "non-existent file",
			input: GitBlameInput{
				File: "non-existent.txt",
			},
			wantErr: true,
			errCode: gerror.ErrCodeNotFound,
		},
		{
			name: "path traversal attempt",
			input: GitBlameInput{
				File: "../../../etc/passwd",
			},
			wantErr: true,
			errCode: gerror.ErrCodeInvalidInput,
		},
		{
			name: "invalid line range - only start",
			input: GitBlameInput{
				File:      "test.txt",
				LineStart: 5,
				// LineEnd missing
			},
			wantErr: true,
			errCode: gerror.ErrCodeInvalidInput,
		},
		{
			name: "invalid line range - start > end",
			input: GitBlameInput{
				File:      "test.txt",
				LineStart: 10,
				LineEnd:   5,
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

func TestGitBlameTool_Execute_UntrackedFile(t *testing.T) {
	tmpDir := t.TempDir()
	setupTestBlameRepo(t, tmpDir)

	// Create an untracked file
	untrackedFile := filepath.Join(tmpDir, "untracked.txt")
	require.NoError(t, os.WriteFile(untrackedFile, []byte("untracked content"), 0o644))

	tool := NewGitBlameTool(tmpDir)
	ctx := context.Background()

	input := GitBlameInput{File: "untracked.txt"}
	inputJSON, _ := json.Marshal(input)

	result, err := tool.Execute(ctx, string(inputJSON))
	assert.Error(t, err)
	assert.Nil(t, result)

	gerr, ok := err.(*gerror.GuildError)
	assert.True(t, ok)
	// Should get error about git command failure (may include "not tracked" message)
	assert.Contains(t, gerr.Message, "git command failed")
}

func TestGitBlameTool_Execute_NonGitDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	tool := NewGitBlameTool(tmpDir)
	ctx := context.Background()

	input := GitBlameInput{File: "test.txt"}
	inputJSON, _ := json.Marshal(input)

	result, err := tool.Execute(ctx, string(inputJSON))
	assert.Error(t, err)
	assert.Nil(t, result)

	gerr, ok := err.(*gerror.GuildError)
	assert.True(t, ok)
	assert.Equal(t, gerror.ErrCodeInvalidInput, gerr.Code)
	assert.Contains(t, gerr.Message, "not a git repository")
}

func TestGitBlameTool_Execute_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	setupTestBlameRepo(t, tmpDir)
	tool := NewGitBlameTool(tmpDir)
	ctx := context.Background()

	result, err := tool.Execute(ctx, "invalid json")
	assert.Error(t, err)
	assert.Nil(t, result)

	gerr, ok := err.(*gerror.GuildError)
	assert.True(t, ok)
	assert.Equal(t, gerror.ErrCodeInvalidInput, gerr.Code)
}

func TestGitBlameTool_EstimateCost(t *testing.T) {
	tool := NewGitBlameTool("/test")

	tests := []struct {
		name   string
		params map[string]interface{}
		want   float64
	}{
		{
			name:   "no line range",
			params: map[string]interface{}{"file": "test.txt"},
			want:   2.0, // Default medium cost
		},
		{
			name: "small line range",
			params: map[string]interface{}{
				"line_start": 1.0,
				"line_end":   30.0,
			},
			want: 1.0,
		},
		{
			name: "medium line range",
			params: map[string]interface{}{
				"line_start": 1.0,
				"line_end":   150.0,
			},
			want: 2.0,
		},
		{
			name: "large line range",
			params: map[string]interface{}{
				"line_start": 1.0,
				"line_end":   500.0,
			},
			want: 3.0,
		},
		{
			name: "integer values",
			params: map[string]interface{}{
				"line_start": 10,
				"line_end":   40,
			},
			want: 1.0,
		},
		{
			name: "only start line specified",
			params: map[string]interface{}{
				"line_start": 10.0,
			},
			want: 2.0, // Falls back to default
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tool.EstimateCost(tt.params)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestGitBlameTool_ShowDate(t *testing.T) {
	tmpDir := t.TempDir()
	setupTestBlameRepo(t, tmpDir)
	tool := NewGitBlameTool(tmpDir)
	ctx := context.Background()

	input := GitBlameInput{
		File:     "test.txt",
		ShowDate: true,
	}
	inputJSON, _ := json.Marshal(input)

	result, err := tool.Execute(ctx, string(inputJSON))
	assert.NoError(t, err)
	assert.NotNil(t, result)

	// When ShowDate is true, the output should include date formatting
	assert.NotEmpty(t, result.Output)
	// The actual date format verification would depend on the specific blame output
}

// Helper function to set up a test git repository with blame-able content
func setupTestBlameRepo(t *testing.T, dir string) {
	t.Helper()

	// Initialize repository
	_, err := executeGitCommand(dir, "init")
	require.NoError(t, err)

	// Configure git user
	_, err = executeGitCommand(dir, "config", "user.email", "test@example.com")
	require.NoError(t, err)
	_, err = executeGitCommand(dir, "config", "user.name", "Test Author")
	require.NoError(t, err)

	// Create a test file with multiple lines
	testFile := filepath.Join(dir, "test.txt")
	content := `line 1
line 2
line 3
line 4
line 5`
	require.NoError(t, os.WriteFile(testFile, []byte(content), 0o644))

	_, err = executeGitCommand(dir, "add", "test.txt")
	require.NoError(t, err)

	_, err = executeGitCommand(dir, "commit", "-m", "Initial commit with test file")
	require.NoError(t, err)

	// Modify the file to create more blame history
	_, err = executeGitCommand(dir, "config", "user.name", "Second Author")
	require.NoError(t, err)

	modifiedContent := `line 1 modified
line 2
line 3 modified
line 4
line 5
line 6 added`
	require.NoError(t, os.WriteFile(testFile, []byte(modifiedContent), 0o644))

	_, err = executeGitCommand(dir, "add", "test.txt")
	require.NoError(t, err)

	_, err = executeGitCommand(dir, "commit", "-m", "Modify test file")
	require.NoError(t, err)
}
