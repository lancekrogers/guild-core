// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package git

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/lancekrogers/guild/pkg/gerror"
	"github.com/lancekrogers/guild/tools"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewGitMergeConflictsTool(t *testing.T) {
	workspacePath := "/test/workspace"
	tool := NewGitMergeConflictsTool(workspacePath)

	assert.NotNil(t, tool)
	assert.Equal(t, "git_merge_conflicts", tool.Name())
	assert.Equal(t, "List, show, and help resolve merge conflicts in git repositories", tool.Description())
	assert.Equal(t, "version_control", tool.Category())
	assert.False(t, tool.RequiresAuth())
	assert.Equal(t, workspacePath, tool.workspacePath)

	// Check schema
	schema := tool.Schema()
	assert.Equal(t, "object", schema["type"])

	// Check properties
	props, ok := schema["properties"].(map[string]interface{})
	require.True(t, ok)
	assert.Contains(t, props, "action")
	assert.Contains(t, props, "file")
	assert.Contains(t, props, "strategy")
	assert.Contains(t, props, "preview")

	// Check action enum
	actionProp, ok := props["action"].(map[string]interface{})
	require.True(t, ok)
	enum, ok := actionProp["enum"].([]string)
	require.True(t, ok)
	assert.Contains(t, enum, "list")
	assert.Contains(t, enum, "show")
	assert.Contains(t, enum, "resolve")

	// Check examples
	examples := tool.Examples()
	assert.Len(t, examples, 4)
}

func TestGitMergeConflictsTool_Execute_ListAction(t *testing.T) {
	tmpDir := t.TempDir()
	setupBasicRepo(t, tmpDir)

	tool := NewGitMergeConflictsTool(tmpDir)
	ctx := context.Background()

	tests := []struct {
		name    string
		input   GitMergeConflictsInput
		wantErr bool
		verify  func(t *testing.T, result *tools.ToolResult)
	}{
		{
			name: "list with no conflicts",
			input: GitMergeConflictsInput{
				Action: "list",
			},
			wantErr: false,
			verify: func(t *testing.T, result *tools.ToolResult) {
				assert.Contains(t, result.Output, "No merge conflicts found")
				assert.Equal(t, "0", result.Metadata["conflict_count"])
			},
		},
		{
			name:  "default action is list",
			input: GitMergeConflictsInput{
				// Action not specified, should default to "list"
			},
			wantErr: false,
			verify: func(t *testing.T, result *tools.ToolResult) {
				assert.Contains(t, result.Output, "No merge conflicts found")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inputJSON, err := json.Marshal(tt.input)
			require.NoError(t, err)

			result, err := tool.Execute(ctx, string(inputJSON))

			if tt.wantErr {
				assert.Error(t, err)
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

func TestGitMergeConflictsTool_Execute_ShowAction(t *testing.T) {
	tmpDir := t.TempDir()
	setupConflictedRepo(t, tmpDir)

	tool := NewGitMergeConflictsTool(tmpDir)
	ctx := context.Background()

	tests := []struct {
		name    string
		input   GitMergeConflictsInput
		wantErr bool
		errCode gerror.ErrorCode
		verify  func(t *testing.T, result *tools.ToolResult)
	}{
		{
			name: "show conflicts in file",
			input: GitMergeConflictsInput{
				Action: "show",
				File:   "conflict.txt",
			},
			wantErr: false,
			verify: func(t *testing.T, result *tools.ToolResult) {
				assert.NotEmpty(t, result.Output)
				assert.Equal(t, "conflict.txt", result.Metadata["file"])
				assert.Equal(t, tmpDir, result.Metadata["workspace_path"])
			},
		},
		{
			name: "show action without file parameter",
			input: GitMergeConflictsInput{
				Action: "show",
				// File missing
			},
			wantErr: true,
			errCode: gerror.ErrCodeInvalidInput,
		},
		{
			name: "show non-existent file",
			input: GitMergeConflictsInput{
				Action: "show",
				File:   "non-existent.txt",
			},
			wantErr: true,
			errCode: gerror.ErrCodeNotFound,
		},
		{
			name: "path traversal attempt",
			input: GitMergeConflictsInput{
				Action: "show",
				File:   "../../../etc/passwd",
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

func TestGitMergeConflictsTool_Execute_ResolveAction(t *testing.T) {
	tmpDir := t.TempDir()
	setupConflictedRepo(t, tmpDir)

	tool := NewGitMergeConflictsTool(tmpDir)
	ctx := context.Background()

	tests := []struct {
		name    string
		input   GitMergeConflictsInput
		wantErr bool
		errCode gerror.ErrorCode
		verify  func(t *testing.T, result *tools.ToolResult)
	}{
		{
			name: "resolve action without file parameter",
			input: GitMergeConflictsInput{
				Action: "resolve",
				// File missing
			},
			wantErr: true,
			errCode: gerror.ErrCodeInvalidInput,
		},
		{
			name: "resolve action without strategy parameter",
			input: GitMergeConflictsInput{
				Action: "resolve",
				File:   "test.txt",
				// Strategy missing
			},
			wantErr: true,
			errCode: gerror.ErrCodeInvalidInput,
		},
		{
			name: "resolve with manual strategy - no conflicts",
			input: GitMergeConflictsInput{
				Action:   "resolve",
				File:     "conflict.txt",
				Strategy: "manual",
			},
			wantErr: false,
			verify: func(t *testing.T, result *tools.ToolResult) {
				assert.Contains(t, result.Output, "has no merge conflicts")
				assert.Equal(t, "conflict.txt", result.Metadata["file"])
			},
		},
		{
			name: "resolve with unknown strategy - no conflicts",
			input: GitMergeConflictsInput{
				Action:   "resolve",
				File:     "conflict.txt",
				Strategy: "unknown",
			},
			wantErr: false,
			verify: func(t *testing.T, result *tools.ToolResult) {
				// Should return early with no conflicts message, not validate strategy
				assert.Contains(t, result.Output, "has no merge conflicts")
			},
		},
		{
			name: "resolve with preview mode - no conflicts",
			input: GitMergeConflictsInput{
				Action:   "resolve",
				File:     "conflict.txt",
				Strategy: "ours",
				Preview:  true,
			},
			wantErr: false,
			verify: func(t *testing.T, result *tools.ToolResult) {
				assert.Contains(t, result.Output, "has no merge conflicts")
				assert.Equal(t, "conflict.txt", result.Metadata["file"])
			},
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

func TestGitMergeConflictsTool_Execute_InvalidAction(t *testing.T) {
	tmpDir := t.TempDir()
	setupBasicRepo(t, tmpDir)
	tool := NewGitMergeConflictsTool(tmpDir)
	ctx := context.Background()

	input := GitMergeConflictsInput{
		Action: "invalid",
	}
	inputJSON, _ := json.Marshal(input)

	result, err := tool.Execute(ctx, string(inputJSON))
	assert.Error(t, err)
	assert.Nil(t, result)

	gerr, ok := err.(*gerror.GuildError)
	assert.True(t, ok)
	assert.Equal(t, gerror.ErrCodeInvalidInput, gerr.Code)
	assert.Contains(t, gerr.Message, "unknown action")
}

func TestGitMergeConflictsTool_Execute_NonGitDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	tool := NewGitMergeConflictsTool(tmpDir)
	ctx := context.Background()

	input := GitMergeConflictsInput{Action: "list"}
	inputJSON, _ := json.Marshal(input)

	result, err := tool.Execute(ctx, string(inputJSON))
	assert.Error(t, err)
	assert.Nil(t, result)

	gerr, ok := err.(*gerror.GuildError)
	assert.True(t, ok)
	assert.Equal(t, gerror.ErrCodeInvalidInput, gerr.Code)
	assert.Contains(t, gerr.Message, "not a git repository")
}

func TestGitMergeConflictsTool_Execute_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	setupBasicRepo(t, tmpDir)
	tool := NewGitMergeConflictsTool(tmpDir)
	ctx := context.Background()

	result, err := tool.Execute(ctx, "invalid json")
	assert.Error(t, err)
	assert.Nil(t, result)

	gerr, ok := err.(*gerror.GuildError)
	assert.True(t, ok)
	assert.Equal(t, gerror.ErrCodeInvalidInput, gerr.Code)
}

func TestGitMergeConflictsTool_EstimateCost(t *testing.T) {
	tool := NewGitMergeConflictsTool("/test")

	tests := []struct {
		name   string
		params map[string]interface{}
		want   float64
	}{
		{
			name:   "list action",
			params: map[string]interface{}{"action": "list"},
			want:   1.0,
		},
		{
			name:   "show action",
			params: map[string]interface{}{"action": "show"},
			want:   2.0,
		},
		{
			name:   "resolve action",
			params: map[string]interface{}{"action": "resolve"},
			want:   3.0,
		},
		{
			name: "resolve with preview",
			params: map[string]interface{}{
				"action":  "resolve",
				"preview": true,
			},
			want: 2.0,
		},
		{
			name:   "unknown action",
			params: map[string]interface{}{"action": "unknown"},
			want:   1.0,
		},
		{
			name:   "default when not specified",
			params: map[string]interface{}{},
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

// Helper function to set up a basic git repository
func setupBasicRepo(t *testing.T, dir string) {
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
}

// Helper function to set up a repository with conflicts
func setupConflictedRepo(t *testing.T, dir string) {
	t.Helper()

	setupBasicRepo(t, dir)

	// Create a file with conflict markers to simulate a merge conflict
	conflictFile := filepath.Join(dir, "conflict.txt")
	conflictContent := `normal line
<<<<<<< HEAD
our version of the change
=======
their version of the change
>>>>>>> feature-branch
another normal line`

	require.NoError(t, os.WriteFile(conflictFile, []byte(conflictContent), 0644))
}
