// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package git

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseGitLog(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []CommitInfo
	}{
		{
			name: "simple log output",
			input: `abc123 Initial commit
def456 Add feature X
ghi789 Fix bug #123`,
			want: []CommitInfo{
				{Hash: "abc123", ShortHash: "abc123", Message: "Initial commit", Subject: "Initial commit"},
				{Hash: "def456", ShortHash: "def456", Message: "Add feature X", Subject: "Add feature X"},
				{Hash: "ghi789", ShortHash: "ghi789", Message: "Fix bug #123", Subject: "Fix bug #123"},
			},
		},
		{
			name:  "empty output",
			input: "",
			want:  []CommitInfo{},
		},
		{
			name: "output with empty lines",
			input: `abc123 First commit

def456 Second commit`,
			want: []CommitInfo{
				{Hash: "abc123", ShortHash: "abc123", Message: "First commit", Subject: "First commit"},
				{Hash: "def456", ShortHash: "def456", Message: "Second commit", Subject: "Second commit"},
			},
		},
		{
			name:  "single commit",
			input: `abc123 Single commit`,
			want: []CommitInfo{
				{Hash: "abc123", ShortHash: "abc123", Message: "Single commit", Subject: "Single commit"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseGitLog(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestParseGitLogVerbose(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		verify func(t *testing.T, commits []CommitInfo)
	}{
		{
			name: "full commit details",
			input: `commit abc123
Author: Alice <alice@example.com>
Date:   Mon Jan 1 12:00:00 2024 +0000

    Initial commit
    
    This is the first commit in the repository.

commit def456
Author: Bob <bob@example.com>
Date:   Tue Jan 2 13:00:00 2024 +0000

    Add feature X`,
			verify: func(t *testing.T, commits []CommitInfo) {
				require.Len(t, commits, 2)

				assert.Equal(t, "abc123", commits[0].Hash)
				assert.Equal(t, "abc123", commits[0].ShortHash)
				assert.Equal(t, "Alice <alice@example.com>", commits[0].Author)
				assert.Equal(t, "Initial commit", commits[0].Subject)
				assert.Contains(t, commits[0].Message, "This is the first commit")

				assert.Equal(t, "def456", commits[1].Hash)
				assert.Equal(t, "Bob <bob@example.com>", commits[1].Author)
				assert.Equal(t, "Add feature X", commits[1].Subject)
			},
		},
		{
			name:  "empty input",
			input: "",
			verify: func(t *testing.T, commits []CommitInfo) {
				assert.Empty(t, commits)
			},
		},
		{
			name: "commit without body",
			input: `commit abc123
Author: Alice <alice@example.com>
Date:   Mon Jan 1 12:00:00 2024 +0000

    Single line commit`,
			verify: func(t *testing.T, commits []CommitInfo) {
				require.Len(t, commits, 1)
				assert.Equal(t, "Single line commit", commits[0].Subject)
				assert.Equal(t, "Single line commit", commits[0].Message)
				assert.Empty(t, commits[0].Body)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseGitLogVerbose(tt.input)
			tt.verify(t, got)
		})
	}
}

func TestParseGitDate(t *testing.T) {
	tests := []struct {
		name    string
		dateStr string
		wantErr bool
		verify  func(t *testing.T, tm time.Time)
	}{
		{
			name:    "standard git date format",
			dateStr: "Mon Jan 2 15:04:05 2006 -0700",
			wantErr: false,
			verify: func(t *testing.T, tm time.Time) {
				assert.Equal(t, 2006, tm.Year())
				assert.Equal(t, time.January, tm.Month())
				assert.Equal(t, 2, tm.Day())
			},
		},
		{
			name:    "alternative date format",
			dateStr: "2024-01-15 10:30:45 +0000",
			wantErr: false,
			verify: func(t *testing.T, tm time.Time) {
				assert.Equal(t, 2024, tm.Year())
				assert.Equal(t, time.January, tm.Month())
				assert.Equal(t, 15, tm.Day())
			},
		},
		{
			name:    "invalid date format",
			dateStr: "not a date",
			wantErr: true,
		},
		{
			name:    "empty string",
			dateStr: "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseGitDate(tt.dateStr)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if tt.verify != nil {
					tt.verify(t, got)
				}
			}
		})
	}
}

func TestParseGitBlame(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []BlameInfo
	}{
		{
			name: "standard blame output",
			input: `abc123 (Alice 2024-01-01 12:00:00 +0000   1) package main
abc123 (Alice 2024-01-01 12:00:00 +0000   2) 
def456 (Bob   2024-01-02 13:00:00 +0000   3) import "fmt"
def456 (Bob   2024-01-02 13:00:00 +0000   4) 
ghi789 (Carol 2024-01-03 14:00:00 +0000   5) func main() {
ghi789 (Carol 2024-01-03 14:00:00 +0000   6)     fmt.Println("Hello")
ghi789 (Carol 2024-01-03 14:00:00 +0000   7) }`,
			want: []BlameInfo{
				{Commit: "abc123", Author: "Alice", LineNumber: 1, LineContent: "package main"},
				{Commit: "abc123", Author: "Alice", LineNumber: 2, LineContent: ""},
				{Commit: "def456", Author: "Bob", LineNumber: 3, LineContent: "import \"fmt\""},
				{Commit: "def456", Author: "Bob", LineNumber: 4, LineContent: ""},
				{Commit: "ghi789", Author: "Carol", LineNumber: 5, LineContent: "func main() {"},
				{Commit: "ghi789", Author: "Carol", LineNumber: 6, LineContent: "    fmt.Println(\"Hello\")"},
				{Commit: "ghi789", Author: "Carol", LineNumber: 7, LineContent: "}"},
			},
		},
		{
			name:  "author names with spaces",
			input: `abc123 (John Doe 2024-01-01 12:00:00 +0000   1) // Comment`,
			want: []BlameInfo{
				{Commit: "abc123", Author: "John Doe", LineNumber: 1, LineContent: "// Comment"},
			},
		},
		{
			name:  "empty input",
			input: "",
			want:  []BlameInfo{},
		},
		{
			name: "malformed lines are skipped",
			input: `invalid line
abc123 (Alice 2024-01-01 12:00:00 +0000   1) valid line
another invalid line`,
			want: []BlameInfo{
				{Commit: "abc123", Author: "Alice", LineNumber: 1, LineContent: "valid line"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseGitBlame(tt.input)
			assert.Equal(t, len(tt.want), len(got))

			for i := range tt.want {
				assert.Equal(t, tt.want[i].Commit, got[i].Commit)
				assert.Equal(t, tt.want[i].Author, got[i].Author)
				assert.Equal(t, tt.want[i].LineNumber, got[i].LineNumber)
				assert.Equal(t, tt.want[i].LineContent, got[i].LineContent)
			}
		})
	}
}

func TestParseConflictedFiles(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{
			name: "multiple conflicted files",
			input: `src/main.go
pkg/utils/helper.go
README.md`,
			want: []string{"src/main.go", "pkg/utils/helper.go", "README.md"},
		},
		{
			name:  "empty output",
			input: "",
			want:  []string{},
		},
		{
			name: "output with empty lines",
			input: `file1.go

file2.go`,
			want: []string{"file1.go", "file2.go"},
		},
		{
			name:  "single file",
			input: "conflict.txt",
			want:  []string{"conflict.txt"},
		},
		{
			name: "files with spaces in names",
			input: `my file.txt
another file.go`,
			want: []string{"my file.txt", "another file.go"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseConflictedFiles(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestParseConflictMarkers(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		verify func(t *testing.T, info ConflictInfo)
	}{
		{
			name: "simple 2-way conflict",
			input: `line before
<<<<<<< HEAD
our version
=======
their version
>>>>>>> feature-branch
line after`,
			verify: func(t *testing.T, info ConflictInfo) {
				assert.Equal(t, 1, info.ConflictCount)
				assert.Equal(t, 1, info.OurMarkers)
				assert.Equal(t, 1, info.TheirMarkers)
				require.Len(t, info.ConflictBlocks, 1)

				block := info.ConflictBlocks[0]
				assert.Equal(t, 2, block.StartLine)
				assert.Equal(t, 6, block.EndLine)
				assert.Equal(t, []string{"our version"}, block.OurLines)
				assert.Equal(t, []string{"their version"}, block.TheirLines)
				assert.Empty(t, block.BaseLines)
			},
		},
		{
			name: "3-way conflict with base",
			input: `<<<<<<< HEAD
current
||||||| merged common ancestors
base
=======
incoming
>>>>>>> branch`,
			verify: func(t *testing.T, info ConflictInfo) {
				assert.Equal(t, 1, info.ConflictCount)
				require.Len(t, info.ConflictBlocks, 1)

				block := info.ConflictBlocks[0]
				assert.Equal(t, []string{"current"}, block.OurLines)
				assert.Equal(t, []string{"base"}, block.BaseLines)
				assert.Equal(t, []string{"incoming"}, block.TheirLines)
			},
		},
		{
			name: "multiple conflicts",
			input: `<<<<<<< HEAD
first our
=======
first their
>>>>>>> branch
middle content
<<<<<<< HEAD
second our
=======
second their
>>>>>>> branch`,
			verify: func(t *testing.T, info ConflictInfo) {
				assert.Equal(t, 2, info.ConflictCount)
				assert.Equal(t, 2, info.OurMarkers)
				assert.Equal(t, 2, info.TheirMarkers)
				require.Len(t, info.ConflictBlocks, 2)
			},
		},
		{
			name: "no conflicts",
			input: `normal file
without any
conflict markers`,
			verify: func(t *testing.T, info ConflictInfo) {
				assert.Equal(t, 0, info.ConflictCount)
				assert.Empty(t, info.ConflictBlocks)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseConflictMarkers(tt.input)
			tt.verify(t, got)
		})
	}
}

func TestParseGitStatus(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  map[string]string
	}{
		{
			name: "various file states",
			input: `?? untracked.txt
A  added.go
M  modified.py
D  deleted.md
R  renamed.js
C  copied.rb
UU conflicted.txt`,
			want: map[string]string{
				"untracked.txt":  "untracked",
				"added.go":       "added",
				"modified.py":    "modified",
				"deleted.md":     "deleted",
				"renamed.js":     "renamed",
				"copied.rb":      "copied",
				"conflicted.txt": "both modified",
			},
		},
		{
			name:  "empty status",
			input: "",
			want:  map[string]string{},
		},
		{
			name: "files with spaces",
			input: `?? "my file.txt"
M  "another file.go"`,
			want: map[string]string{
				`"my file.txt"`:     "untracked",
				`"another file.go"`: "modified",
			},
		},
		{
			name: "conflict states",
			input: `UU both_modified.txt
AA both_added.txt
DD both_deleted.txt`,
			want: map[string]string{
				"both_modified.txt": "both modified",
				"both_added.txt":    "both added",
				"both_deleted.txt":  "both deleted",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseGitStatus(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}
