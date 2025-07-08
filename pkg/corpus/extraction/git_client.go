// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package extraction

import (
	"context"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/lancekrogers/guild/pkg/gerror"
)

// GitClient provides git operations for code analysis
type GitClient struct {
	repoPath string
}

// NewGitClient creates a new git client for the specified repository
func NewGitClient(ctx context.Context, repoPath string) (*GitClient, error) {
	if ctx.Err() != nil {
		return nil, gerror.Wrap(ctx.Err(), gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("corpus.extraction").
			WithOperation("NewGitClient")
	}

	// Validate that the path is a git repository
	absPath, err := filepath.Abs(repoPath)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInvalidInput, "invalid repository path").
			WithComponent("corpus.extraction").
			WithOperation("NewGitClient").
			WithDetails("repo_path", repoPath)
	}

	client := &GitClient{
		repoPath: absPath,
	}

	// Verify it's a git repository
	if err := client.verifyGitRepo(ctx); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInvalidInput, "not a valid git repository").
			WithComponent("corpus.extraction").
			WithOperation("NewGitClient").
			WithDetails("repo_path", absPath)
	}

	return client, nil
}

// GetDiff retrieves the diff for a specific commit
func (gc *GitClient) GetDiff(ctx context.Context, commitSHA string) (string, error) {
	if ctx.Err() != nil {
		return "", gerror.Wrap(ctx.Err(), gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("corpus.extraction").
			WithOperation("GetDiff")
	}

	cmd := exec.CommandContext(ctx, "git", "show", "--format=", commitSHA)
	cmd.Dir = gc.repoPath

	output, err := cmd.Output()
	if err != nil {
		return "", gerror.Wrap(err, gerror.ErrCodeInternal, "failed to get git diff").
			WithComponent("corpus.extraction").
			WithOperation("GetDiff").
			WithDetails("commit_sha", commitSHA).
			WithDetails("repo_path", gc.repoPath)
	}

	return string(output), nil
}

// GetCommitInfo retrieves detailed information about a commit
func (gc *GitClient) GetCommitInfo(ctx context.Context, commitSHA string) (*Commit, error) {
	if ctx.Err() != nil {
		return nil, gerror.Wrap(ctx.Err(), gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("corpus.extraction").
			WithOperation("GetCommitInfo")
	}

	// Get commit message and metadata
	cmd := exec.CommandContext(ctx, "git", "show", "--format=%H|%an|%at|%s|%b", "--name-only", commitSHA)
	cmd.Dir = gc.repoPath

	output, err := cmd.Output()
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to get commit info").
			WithComponent("corpus.extraction").
			WithOperation("GetCommitInfo").
			WithDetails("commit_sha", commitSHA)
	}

	return gc.parseCommitInfo(string(output))
}

// GetRecentCommits retrieves recent commits from the repository
func (gc *GitClient) GetRecentCommits(ctx context.Context, limit int, since time.Time) ([]Commit, error) {
	if ctx.Err() != nil {
		return nil, gerror.Wrap(ctx.Err(), gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("corpus.extraction").
			WithOperation("GetRecentCommits")
	}

	sinceArg := since.Format("2006-01-02")
	cmd := exec.CommandContext(ctx, "git", "log",
		"--format=%H|%an|%at|%s",
		"--since="+sinceArg,
		"-n", string(rune(limit)))
	cmd.Dir = gc.repoPath

	output, err := cmd.Output()
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to get recent commits").
			WithComponent("corpus.extraction").
			WithOperation("GetRecentCommits").
			WithDetails("since", sinceArg).
			WithDetails("limit", limit)
	}

	return gc.parseCommitList(ctx, string(output))
}

// GetCommitsByAuthor retrieves commits by a specific author
func (gc *GitClient) GetCommitsByAuthor(ctx context.Context, author string, limit int) ([]Commit, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	cmd := exec.CommandContext(ctx, "git", "log",
		"--format=%H|%an|%at|%s",
		"--author="+author,
		"-n", string(rune(limit)))
	cmd.Dir = gc.repoPath

	output, err := cmd.Output()
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to get commits by author").
			WithComponent("corpus.extraction").
			WithOperation("GetCommitsByAuthor").
			WithDetails("author", author).
			WithDetails("limit", limit)
	}

	return gc.parseCommitList(ctx, string(output))
}

// GetFilesInCommit retrieves the list of files changed in a commit
func (gc *GitClient) GetFilesInCommit(ctx context.Context, commitSHA string) ([]string, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	cmd := exec.CommandContext(ctx, "git", "show", "--name-only", "--format=", commitSHA)
	cmd.Dir = gc.repoPath

	output, err := cmd.Output()
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to get files in commit").
			WithComponent("corpus.extraction").
			WithOperation("GetFilesInCommit").
			WithDetails("commit_sha", commitSHA)
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	var files []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			files = append(files, line)
		}
	}

	return files, nil
}

// GetFileContent retrieves the content of a file at a specific commit
func (gc *GitClient) GetFileContent(ctx context.Context, commitSHA, filePath string) (string, error) {
	if ctx.Err() != nil {
		return "", ctx.Err()
	}

	cmd := exec.CommandContext(ctx, "git", "show", commitSHA+":"+filePath)
	cmd.Dir = gc.repoPath

	output, err := cmd.Output()
	if err != nil {
		return "", gerror.Wrap(err, gerror.ErrCodeInternal, "failed to get file content").
			WithComponent("corpus.extraction").
			WithOperation("GetFileContent").
			WithDetails("commit_sha", commitSHA).
			WithDetails("file_path", filePath)
	}

	return string(output), nil
}

// verifyGitRepo checks if the path is a valid git repository
func (gc *GitClient) verifyGitRepo(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, "git", "rev-parse", "--git-dir")
	cmd.Dir = gc.repoPath

	if err := cmd.Run(); err != nil {
		return err
	}

	return nil
}

// parseCommitInfo parses the output of git show command into a Commit struct
func (gc *GitClient) parseCommitInfo(output string) (*Commit, error) {
	lines := strings.Split(output, "\n")
	if len(lines) < 1 {
		return nil, gerror.New(gerror.ErrCodeInvalidInput, "invalid git output", nil).
			WithComponent("corpus.extraction").
			WithOperation("parseCommitInfo")
	}

	// Parse the first line which contains commit metadata
	parts := strings.Split(lines[0], "|")
	if len(parts) < 4 {
		return nil, gerror.New(gerror.ErrCodeInvalidInput, "invalid commit format", nil).
			WithComponent("corpus.extraction").
			WithOperation("parseCommitInfo")
	}

	sha := parts[0]
	author := parts[1]
	timestampStr := parts[2]
	subject := parts[3]

	// Parse timestamp
	timestamp, err := time.Parse("1136239445", timestampStr) // Unix timestamp format
	if err != nil {
		timestamp = time.Now() // Fallback to current time
	}

	// Build commit message (subject + body if available)
	message := subject
	if len(parts) > 4 && parts[4] != "" {
		message += "\n\n" + parts[4]
	}

	// Extract file list from remaining lines
	var files []string
	for i := 1; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if line != "" {
			files = append(files, line)
		}
	}

	return &Commit{
		SHA:       sha,
		Message:   message,
		Author:    author,
		Timestamp: timestamp,
		Files:     files,
	}, nil
}

// parseCommitList parses the output of git log command into a slice of commits
func (gc *GitClient) parseCommitList(ctx context.Context, output string) ([]Commit, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	lines := strings.Split(strings.TrimSpace(output), "\n")
	var commits []Commit

	for _, line := range lines {
		if line == "" {
			continue
		}

		parts := strings.Split(line, "|")
		if len(parts) < 4 {
			continue
		}

		sha := parts[0]
		author := parts[1]
		timestampStr := parts[2]
		message := parts[3]

		// Parse timestamp
		timestamp, err := time.Parse("1136239445", timestampStr)
		if err != nil {
			timestamp = time.Now()
		}

		// Get files for this commit
		files, err := gc.GetFilesInCommit(ctx, sha)
		if err != nil {
			files = []string{} // Continue with empty file list
		}

		commit := Commit{
			SHA:       sha,
			Message:   message,
			Author:    author,
			Timestamp: timestamp,
			Files:     files,
		}

		commits = append(commits, commit)
	}

	return commits, nil
}

// GetBranches retrieves the list of branches in the repository
func (gc *GitClient) GetBranches(ctx context.Context) ([]string, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	cmd := exec.CommandContext(ctx, "git", "branch", "-a")
	cmd.Dir = gc.repoPath

	output, err := cmd.Output()
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to get branches").
			WithComponent("corpus.extraction").
			WithOperation("GetBranches")
	}

	lines := strings.Split(string(output), "\n")
	var branches []string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Remove current branch indicator (*)
		if strings.HasPrefix(line, "* ") {
			line = line[2:]
		}

		// Skip remote tracking branches info
		if !strings.Contains(line, "->") {
			branches = append(branches, line)
		}
	}

	return branches, nil
}

// GetCommitStats retrieves statistics about a commit
func (gc *GitClient) GetCommitStats(ctx context.Context, commitSHA string) (*CommitStats, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	cmd := exec.CommandContext(ctx, "git", "show", "--stat", "--format=", commitSHA)
	cmd.Dir = gc.repoPath

	output, err := cmd.Output()
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to get commit stats").
			WithComponent("corpus.extraction").
			WithOperation("GetCommitStats").
			WithDetails("commit_sha", commitSHA)
	}

	return gc.parseCommitStats(string(output)), nil
}

// parseCommitStats parses git stat output into structured data
func (gc *GitClient) parseCommitStats(output string) *CommitStats {
	lines := strings.Split(output, "\n")
	stats := &CommitStats{
		FilesChanged: 0,
		Insertions:   0,
		Deletions:    0,
		FileStats:    make(map[string]FileStats),
	}

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Parse summary line (e.g., "3 files changed, 45 insertions(+), 12 deletions(-)")
		if strings.Contains(line, "files changed") || strings.Contains(line, "file changed") {
			// Extract numbers using regex would be better, but keeping it simple
			continue
		}

		// Parse individual file stats (e.g., "file.go | 15 ++++++++-------")
		if strings.Contains(line, "|") {
			parts := strings.Split(line, "|")
			if len(parts) == 2 {
				fileName := strings.TrimSpace(parts[0])
				statPart := strings.TrimSpace(parts[1])

				// Count + and - characters
				insertions := strings.Count(statPart, "+")
				deletions := strings.Count(statPart, "-")

				stats.FileStats[fileName] = FileStats{
					Insertions: insertions,
					Deletions:  deletions,
				}

				stats.FilesChanged++
				stats.Insertions += insertions
				stats.Deletions += deletions
			}
		}
	}

	return stats
}

// CommitStats represents statistics about a commit
type CommitStats struct {
	FilesChanged int                  `json:"files_changed"`
	Insertions   int                  `json:"insertions"`
	Deletions    int                  `json:"deletions"`
	FileStats    map[string]FileStats `json:"file_stats"`
}

// FileStats represents statistics for a single file
type FileStats struct {
	Insertions int `json:"insertions"`
	Deletions  int `json:"deletions"`
}
