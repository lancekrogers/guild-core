// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package git

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/lancekrogers/guild-core/pkg/gerror"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExecuteGitCommand(t *testing.T) {
	tests := []struct {
		name      string
		setupRepo func(t *testing.T) string
		args      []string
		wantErr   bool
		validate  func(t *testing.T, output string)
	}{
		{
			name: "successful command in git repo",
			setupRepo: func(t *testing.T) string {
				dir := t.TempDir()
				_, err := executeGitCommand(dir, "init")
				require.NoError(t, err)
				return dir
			},
			args:    []string{"status"},
			wantErr: false,
			validate: func(t *testing.T, output string) {
				assert.Contains(t, output, "On branch")
			},
		},
		{
			name: "command in non-git directory",
			setupRepo: func(t *testing.T) string {
				return t.TempDir()
			},
			args:    []string{"status"},
			wantErr: true,
		},
		{
			name: "invalid git command",
			setupRepo: func(t *testing.T) string {
				dir := t.TempDir()
				_, err := executeGitCommand(dir, "init")
				require.NoError(t, err)
				return dir
			},
			args:    []string{"invalid-command"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := tt.setupRepo(t)
			output, err := executeGitCommand(dir, tt.args...)

			if tt.wantErr {
				assert.Error(t, err)
				gerr, ok := err.(*gerror.GuildError)
				assert.True(t, ok, "error should be a GuildError")
				assert.Equal(t, gerror.ErrCodeInternal, gerr.Code)
			} else {
				assert.NoError(t, err)
				if tt.validate != nil {
					tt.validate(t, output)
				}
			}
		})
	}
}

func TestValidatePathWithBase(t *testing.T) {
	tests := []struct {
		name     string
		basePath string
		path     string
		wantErr  bool
		errCode  gerror.ErrorCode
	}{
		{
			name:     "empty path is valid",
			basePath: "/workspace",
			path:     "",
			wantErr:  false,
		},
		{
			name:     "subdirectory is valid",
			basePath: "/workspace",
			path:     "src/main.go",
			wantErr:  false,
		},
		{
			name:     "path traversal attempt",
			basePath: "/workspace",
			path:     "../../../etc/passwd",
			wantErr:  true,
			errCode:  gerror.ErrCodeInvalidInput,
		},
		{
			name:     "absolute path outside workspace",
			basePath: "/workspace",
			path:     "/etc/passwd",
			wantErr:  true,
			errCode:  gerror.ErrCodeInvalidInput,
		},
		{
			name:     "dot path is valid",
			basePath: "/workspace",
			path:     ".",
			wantErr:  false,
		},
		{
			name:     "double dot in middle of path",
			basePath: "/workspace",
			path:     "src/../pkg/main.go",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validatePathWithBase(tt.basePath, tt.path)

			if tt.wantErr {
				assert.Error(t, err)
				gerr, ok := err.(*gerror.GuildError)
				assert.True(t, ok, "error should be a GuildError")
				assert.Equal(t, tt.errCode, gerr.Code)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestFormatGitError(t *testing.T) {
	tests := []struct {
		name      string
		err       error
		operation string
		validate  func(t *testing.T, err error)
	}{
		{
			name:      "nil error returns nil",
			err:       nil,
			operation: "test",
			validate: func(t *testing.T, err error) {
				assert.Nil(t, err)
			},
		},
		{
			name:      "already a GuildError returns as-is",
			err:       gerror.New(gerror.ErrCodeNotFound, "test error", nil),
			operation: "test",
			validate: func(t *testing.T, err error) {
				gerr, ok := err.(*gerror.GuildError)
				assert.True(t, ok)
				assert.Equal(t, gerror.ErrCodeNotFound, gerr.Code)
			},
		},
		{
			name:      "regular error gets wrapped",
			err:       assert.AnError,
			operation: "commit",
			validate: func(t *testing.T, err error) {
				gerr, ok := err.(*gerror.GuildError)
				assert.True(t, ok)
				assert.Equal(t, gerror.ErrCodeInternal, gerr.Code)
				assert.Contains(t, gerr.Message, "git commit failed")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := formatGitError(tt.err, tt.operation)
			tt.validate(t, err)
		})
	}
}

func TestTruncateOutput(t *testing.T) {
	tests := []struct {
		name     string
		output   string
		maxLines int
		want     string
	}{
		{
			name:     "output within limit",
			output:   "line1\nline2\nline3",
			maxLines: 5,
			want:     "line1\nline2\nline3",
		},
		{
			name:     "output exceeds limit",
			output:   "line1\nline2\nline3\nline4\nline5",
			maxLines: 3,
			want:     "line1\nline2\nline3\n... truncated 2 lines ...",
		},
		{
			name:     "single line",
			output:   "single line",
			maxLines: 1,
			want:     "single line",
		},
		{
			name:     "empty output",
			output:   "",
			maxLines: 10,
			want:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncateOutput(tt.output, tt.maxLines)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestSanitizeGitOutput(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		output string
	}{
		{
			name:   "email addresses are sanitized",
			input:  "Author: John Doe <john@example.com>",
			output: "Author: John Doe &lt;john@example.com&gt;",
		},
		{
			name:   "multiple angle brackets",
			input:  "Merge: <abc123> <def456>",
			output: "Merge: &lt;abc123&gt; &lt;def456&gt;",
		},
		{
			name:   "no angle brackets",
			input:  "Simple commit message",
			output: "Simple commit message",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sanitizeGitOutput(tt.input)
			assert.Equal(t, tt.output, got)
		})
	}
}

func TestIsGitRepository(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(t *testing.T) string
		expected bool
	}{
		{
			name: "valid git repository",
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				_, err := executeGitCommand(dir, "init")
				require.NoError(t, err)
				return dir
			},
			expected: true,
		},
		{
			name: "non-git directory",
			setup: func(t *testing.T) string {
				return t.TempDir()
			},
			expected: false,
		},
		{
			name: "non-existent directory",
			setup: func(t *testing.T) string {
				return filepath.Join(t.TempDir(), "non-existent")
			},
			expected: false,
		},
		{
			name: "git subdirectory",
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				_, err := executeGitCommand(dir, "init")
				require.NoError(t, err)
				subdir := filepath.Join(dir, "subdir")
				require.NoError(t, os.MkdirAll(subdir, 0o755))
				return subdir
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := tt.setup(t)
			got := isGitRepository(dir)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestGetGitVersion(t *testing.T) {
	version, err := getGitVersion()
	// Should not error on systems with git installed
	if err != nil {
		// Check if git is not installed
		_, gitErr := executeGitCommand(".", "--version")
		if gitErr != nil {
			t.Skip("Git is not installed")
		}
		t.Fatalf("unexpected error: %v", err)
	}

	// Version should start with "git version"
	assert.Contains(t, version, "git version")
}
