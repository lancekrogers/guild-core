// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package git

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestFormatCommitHistory(t *testing.T) {
	tests := []struct {
		name    string
		commits []CommitInfo
		want    string
	}{
		{
			name: "multiple commits",
			commits: []CommitInfo{
				{ShortHash: "abc123", Subject: "Initial commit"},
				{ShortHash: "def456", Subject: "Add feature X"},
				{ShortHash: "ghi789", Subject: "Fix bug #123"},
			},
			want: `abc123 Initial commit
def456 Add feature X
ghi789 Fix bug #123`,
		},
		{
			name:    "empty commits",
			commits: []CommitInfo{},
			want:    "No commits found",
		},
		{
			name: "single commit",
			commits: []CommitInfo{
				{ShortHash: "abc123", Subject: "Only commit"},
			},
			want: "abc123 Only commit",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatCommitHistory(tt.commits)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestFormatCommitHistoryVerbose(t *testing.T) {
	tests := []struct {
		name    string
		commits []CommitInfo
		verify  func(t *testing.T, output string)
	}{
		{
			name: "full commit details",
			commits: []CommitInfo{
				{
					Hash:       "abc123def456",
					Author:     "Alice <alice@example.com>",
					AuthorDate: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
					Message:    "Initial commit\n\nThis is the first commit.",
				},
			},
			verify: func(t *testing.T, output string) {
				assert.Contains(t, output, "commit abc123def456")
				assert.Contains(t, output, "Author: Alice <alice@example.com>")
				assert.Contains(t, output, "Date:   Mon Jan 1 12:00:00 2024 +0000")
				assert.Contains(t, output, "    Initial commit")
				assert.Contains(t, output, "    This is the first commit.")
			},
		},
		{
			name:    "empty commits",
			commits: []CommitInfo{},
			verify: func(t *testing.T, output string) {
				assert.Equal(t, "No commits found", output)
			},
		},
		{
			name: "multiple commits with spacing",
			commits: []CommitInfo{
				{Hash: "abc123", Message: "First"},
				{Hash: "def456", Message: "Second"},
			},
			verify: func(t *testing.T, output string) {
				lines := strings.Split(output, "\n")
				// Should have blank lines between commits
				foundBlank := false
				for _, line := range lines {
					if line == "" && strings.Contains(output, "commit abc123") {
						foundBlank = true
						break
					}
				}
				assert.True(t, foundBlank, "Should have blank lines between commits")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatCommitHistoryVerbose(tt.commits)
			tt.verify(t, got)
		})
	}
}

func TestFormatGitDate(t *testing.T) {
	tests := []struct {
		name string
		time time.Time
		want string
	}{
		{
			name: "standard date",
			time: time.Date(2024, 1, 15, 14, 30, 45, 0, time.FixedZone("EST", -5*3600)),
			want: "Mon Jan 15 14:30:45 2024 -0500",
		},
		{
			name: "UTC date",
			time: time.Date(2024, 12, 25, 0, 0, 0, 0, time.UTC),
			want: "Wed Dec 25 00:00:00 2024 +0000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatGitDate(tt.time)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestFormatBlameOutput(t *testing.T) {
	tests := []struct {
		name      string
		blameInfo []BlameInfo
		want      string
	}{
		{
			name: "standard blame output",
			blameInfo: []BlameInfo{
				{Commit: "abc123def456", Author: "Alice", LineNumber: 1, LineContent: "package main"},
				{Commit: "def456ghi789", Author: "Bob", LineNumber: 2, LineContent: "import \"fmt\""},
				{Commit: "abc123def456", Author: "Alice", LineNumber: 10, LineContent: "func main() {"},
			},
			want: `abc123de Alice                 1: package main
def456gh Bob                   2: import "fmt"
abc123de Alice                10: func main() {`,
		},
		{
			name: "long author names are truncated",
			blameInfo: []BlameInfo{
				{Commit: "abc123def456", Author: "Very Long Author Name That Exceeds Limit", LineNumber: 1, LineContent: "code"},
			},
			want: `abc123de Very Long Author ... 1: code`,
		},
		{
			name:      "empty blame info",
			blameInfo: []BlameInfo{},
			want:      "No blame information available",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatBlameOutput(tt.blameInfo)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestFormatBlameOutputWithDates(t *testing.T) {
	tests := []struct {
		name      string
		blameInfo []BlameInfo
		verify    func(t *testing.T, output string)
	}{
		{
			name: "blame with dates",
			blameInfo: []BlameInfo{
				{
					Commit:      "abc123def456",
					Author:      "Alice",
					AuthorTime:  time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC),
					LineNumber:  1,
					LineContent: "package main",
				},
			},
			verify: func(t *testing.T, output string) {
				assert.Contains(t, output, "abc123de")
				assert.Contains(t, output, "Alice")
				assert.Contains(t, output, "2024-01-15")
				assert.Contains(t, output, "1) package main")
			},
		},
		{
			name:      "empty blame info",
			blameInfo: []BlameInfo{},
			verify: func(t *testing.T, output string) {
				assert.Equal(t, "No blame information available", output)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatBlameOutputWithDates(tt.blameInfo)
			tt.verify(t, got)
		})
	}
}

func TestFormatConflictList(t *testing.T) {
	tests := []struct {
		name      string
		conflicts []ConflictInfo
		verify    func(t *testing.T, output string)
	}{
		{
			name: "multiple conflicts",
			conflicts: []ConflictInfo{
				{
					File:          "main.go",
					ConflictCount: 2,
					ConflictBlocks: []ConflictBlock{
						{StartLine: 10, EndLine: 15},
						{StartLine: 25, EndLine: 30},
					},
				},
				{
					File:          "utils.go",
					ConflictCount: 1,
					ConflictBlocks: []ConflictBlock{
						{StartLine: 5, EndLine: 8},
					},
				},
			},
			verify: func(t *testing.T, output string) {
				assert.Contains(t, output, "Found 2 file(s) with conflicts:")
				assert.Contains(t, output, "main.go:")
				assert.Contains(t, output, "- 2 conflict block(s)")
				assert.Contains(t, output, "- Block 1: lines 10-15")
				assert.Contains(t, output, "- Block 2: lines 25-30")
				assert.Contains(t, output, "utils.go:")
				assert.Contains(t, output, "- 1 conflict block(s)")
				assert.Contains(t, output, "Total: 3 conflict(s) in 2 file(s)")
			},
		},
		{
			name:      "no conflicts",
			conflicts: []ConflictInfo{},
			verify: func(t *testing.T, output string) {
				assert.Equal(t, "No merge conflicts found", output)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatConflictList(tt.conflicts)
			tt.verify(t, got)
		})
	}
}

func TestFormatConflictDetails(t *testing.T) {
	tests := []struct {
		name     string
		conflict ConflictInfo
		verify   func(t *testing.T, output string)
	}{
		{
			name: "detailed conflict with base",
			conflict: ConflictInfo{
				File:          "main.go",
				ConflictCount: 1,
				ConflictBlocks: []ConflictBlock{
					{
						StartLine:  10,
						EndLine:    16,
						OurLines:   []string{"our version", "line 2"},
						BaseLines:  []string{"base version"},
						TheirLines: []string{"their version", "modified"},
					},
				},
			},
			verify: func(t *testing.T, output string) {
				assert.Contains(t, output, "Conflicts in main.go:")
				assert.Contains(t, output, "Total conflict blocks: 1")
				assert.Contains(t, output, "=== Conflict Block 1 (lines 10-16) ===")
				assert.Contains(t, output, "<<<<<<< OURS")
				assert.Contains(t, output, "our version")
				assert.Contains(t, output, "||||||| BASE")
				assert.Contains(t, output, "base version")
				assert.Contains(t, output, "=======")
				assert.Contains(t, output, "their version")
				assert.Contains(t, output, ">>>>>>> THEIRS")
			},
		},
		{
			name: "conflict without base",
			conflict: ConflictInfo{
				File:          "test.txt",
				ConflictCount: 1,
				ConflictBlocks: []ConflictBlock{
					{
						StartLine:  1,
						EndLine:    5,
						OurLines:   []string{"ours"},
						TheirLines: []string{"theirs"},
					},
				},
			},
			verify: func(t *testing.T, output string) {
				assert.Contains(t, output, "<<<<<<< OURS")
				assert.NotContains(t, output, "||||||| BASE")
				assert.Contains(t, output, "=======")
				assert.Contains(t, output, ">>>>>>> THEIRS")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatConflictDetails(tt.conflict)
			tt.verify(t, got)
		})
	}
}

func TestFormatConflictSummary(t *testing.T) {
	tests := []struct {
		name      string
		conflicts []ConflictInfo
		want      string
	}{
		{
			name:      "no conflicts",
			conflicts: []ConflictInfo{},
			want:      "✓ No merge conflicts",
		},
		{
			name: "single file with conflicts",
			conflicts: []ConflictInfo{
				{ConflictCount: 2},
			},
			want: "⚠️  1 file with 2 conflict block(s)",
		},
		{
			name: "multiple files with conflicts",
			conflicts: []ConflictInfo{
				{ConflictCount: 2},
				{ConflictCount: 1},
				{ConflictCount: 3},
			},
			want: "⚠️  3 files with 6 total conflict block(s)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatConflictSummary(tt.conflicts)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestCountConflictMarkers(t *testing.T) {
	tests := []struct {
		name      string
		conflicts []ConflictInfo
		want      int
	}{
		{
			name: "multiple conflicts",
			conflicts: []ConflictInfo{
				{OurMarkers: 2, TheirMarkers: 2},
				{OurMarkers: 1, TheirMarkers: 1},
			},
			want: 6,
		},
		{
			name:      "no conflicts",
			conflicts: []ConflictInfo{},
			want:      0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := countConflictMarkers(tt.conflicts)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestCountUniqueAuthors(t *testing.T) {
	tests := []struct {
		name      string
		blameInfo []BlameInfo
		want      int
	}{
		{
			name: "multiple authors",
			blameInfo: []BlameInfo{
				{Author: "Alice"},
				{Author: "Bob"},
				{Author: "Alice"},
				{Author: "Carol"},
				{Author: "Bob"},
			},
			want: 3,
		},
		{
			name: "single author",
			blameInfo: []BlameInfo{
				{Author: "Alice"},
				{Author: "Alice"},
			},
			want: 1,
		},
		{
			name:      "no authors",
			blameInfo: []BlameInfo{},
			want:      0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := countUniqueAuthors(tt.blameInfo)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestFindOldestCommit(t *testing.T) {
	tests := []struct {
		name      string
		blameInfo []BlameInfo
		want      string
	}{
		{
			name: "multiple commits with different times",
			blameInfo: []BlameInfo{
				{Commit: "abc123def456", AuthorTime: time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)},
				{Commit: "def456ghi789", AuthorTime: time.Date(2024, 1, 10, 0, 0, 0, 0, time.UTC)},
				{Commit: "ghi789jkl012", AuthorTime: time.Date(2024, 1, 20, 0, 0, 0, 0, time.UTC)},
			},
			want: "def456gh",
		},
		{
			name:      "empty blame info",
			blameInfo: []BlameInfo{},
			want:      "",
		},
		{
			name: "single commit",
			blameInfo: []BlameInfo{
				{Commit: "abc123def456", AuthorTime: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)},
			},
			want: "abc123de",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := findOldestCommit(tt.blameInfo)
			assert.Equal(t, tt.want, got)
		})
	}
}
