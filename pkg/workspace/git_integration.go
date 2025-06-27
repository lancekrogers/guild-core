// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package workspace

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/lancekrogers/guild/pkg/gerror"
)

// GitManager extends Manager with git worktree support
type GitManager struct {
	Manager
	repoPath string
}

// NewGitManager creates a new git-aware workspace manager
func NewGitManager(baseDir, repoPath string) (*GitManager, error) {
	// Validate git repository
	if err := validateGitRepository(repoPath); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInvalidInput, "invalid git repository").
			WithComponent("workspace").
			WithOperation("new_git_manager")
	}

	// Create base manager
	baseMgr, err := NewManager(baseDir, repoPath)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create base manager").
			WithComponent("workspace").
			WithOperation("new_git_manager")
	}

	return &GitManager{
		Manager:  baseMgr,
		repoPath: repoPath,
	}, nil
}

// CreateWorkspace creates a new git worktree workspace
func (m *GitManager) CreateWorkspace(ctx context.Context, opts CreateOptions) (Workspace, error) {
	// Generate workspace info
	id := generateWorkspaceID(opts.AgentID)
	workspacePath := filepath.Join(opts.WorkDir, "workspaces", id)

	// Generate branch name with proper prefix
	branchPrefix := opts.BranchPrefix
	if branchPrefix == "" {
		branchPrefix = "agent"
	}
	branchName := fmt.Sprintf("%s/%s-%s", branchPrefix, opts.AgentID, id)

	// Create worktree
	baseBranch := opts.BaseBranch
	if baseBranch == "" {
		baseBranch = detectDefaultBranch(m.repoPath)
	}

	cmd := exec.Command("git", "worktree", "add", "-b", branchName, workspacePath, baseBranch)
	cmd.Dir = m.repoPath

	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, fmt.Sprintf("failed to create worktree: %s", string(output))).
			WithComponent("workspace").
			WithOperation("create_workspace")
	}

	// Create workspace info
	info := &WorkspaceInfo{
		ID:           id,
		AgentID:      opts.AgentID,
		Path:         workspacePath,
		Branch:       branchName,
		Status:       StatusActive,
		CreatedAt:    time.Now(),
		LastActivity: time.Now(),
	}

	// Create git-enabled workspace
	ws := &GitWorkspace{
		workspace: &workspace{
			info:     info,
			repoPath: m.repoPath,
			gitInfo: &GitInfo{
				BranchName: branchName,
				RemoteURL:  "", // TODO: Get remote URL
				IsDirty:    false,
			},
		},
	}

	// Update git info
	ws.UpdateGitInfo()

	// Track in manager (if it has internal tracking)
	if mgr, ok := m.Manager.(*manager); ok {
		mgr.mu.Lock()
		mgr.workspaces[id] = ws.workspace
		mgr.mu.Unlock()
	}

	return ws, nil
}

// GitWorkspace extends workspace with git operations
type GitWorkspace struct {
	*workspace
}

// GetGitInfo returns the git information for this workspace
func (w *GitWorkspace) GetGitInfo() *GitInfo {
	return w.gitInfo
}

// UpdateGitInfo updates git-specific information
func (w *GitWorkspace) UpdateGitInfo() error {
	// Get current commit SHA
	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = w.info.Path

	output, err := cmd.Output()
	if err == nil {
		w.gitInfo.CommitHash = strings.TrimSpace(string(output))
	}

	// Check for uncommitted changes
	cmd = exec.Command("git", "status", "--porcelain")
	cmd.Dir = w.info.Path

	output, err = cmd.Output()
	if err == nil {
		w.gitInfo.IsDirty = len(output) > 0
	}

	return nil
}

// Cleanup removes the workspace and its worktree
func (w *GitWorkspace) Cleanup() error {
	// Remove worktree
	cmd := exec.Command("git", "worktree", "remove", w.info.Path, "--force")
	cmd.Dir = w.repoPath

	if err := cmd.Run(); err != nil {
		// Fallback to manual removal
		if err := os.RemoveAll(w.info.Path); err != nil {
			return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to remove workspace").
				WithComponent("workspace").
				WithOperation("remove_workspace")
		}
	}

	// Delete branch
	cmd = exec.Command("git", "branch", "-D", w.info.Branch)
	cmd.Dir = w.repoPath
	cmd.Run() // Ignore errors

	// Prune worktrees
	cmd = exec.Command("git", "worktree", "prune")
	cmd.Dir = w.repoPath
	cmd.Run()

	w.info.Status = StatusCleaning
	return nil
}

// CommitChanges commits any uncommitted changes in the workspace
func (w *GitWorkspace) CommitChanges(message string) error {
	// Stage all changes
	cmd := exec.Command("git", "add", "-A")
	cmd.Dir = w.info.Path
	if err := cmd.Run(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to stage changes").
			WithComponent("workspace").
			WithOperation("stage_changes")
	}

	// Check if there are changes to commit
	cmd = exec.Command("git", "diff", "--cached", "--quiet")
	cmd.Dir = w.info.Path
	if err := cmd.Run(); err == nil {
		// No changes to commit
		return nil
	}

	// Commit changes
	cmd = exec.Command("git", "commit", "-m", message)
	cmd.Dir = w.info.Path
	output, err := cmd.CombinedOutput()
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, fmt.Sprintf("failed to commit: %s", string(output))).
			WithComponent("workspace").
			WithOperation("commit_changes")
	}

	w.UpdateGitInfo()
	return nil
}

// GetDiff returns the diff of uncommitted changes
func (w *GitWorkspace) GetDiff() (string, error) {
	cmd := exec.Command("git", "diff", "HEAD")
	cmd.Dir = w.info.Path

	output, err := cmd.Output()
	if err != nil {
		return "", gerror.Wrap(err, gerror.ErrCodeInternal, "failed to get diff").
			WithComponent("workspace").
			WithOperation("get_diff")
	}

	return string(output), nil
}

// validateGitRepository checks if the path is a valid git repository
func validateGitRepository(path string) error {
	cmd := exec.Command("git", "rev-parse", "--git-dir")
	cmd.Dir = path

	if err := cmd.Run(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInvalidInput, "not a git repository").
			WithComponent("workspace").
			WithOperation("validate_git_repository")
	}

	// Check worktree support
	cmd = exec.Command("git", "worktree", "list")
	cmd.Dir = path

	if err := cmd.Run(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInvalidInput, "git worktree not supported").
			WithComponent("workspace").
			WithOperation("validate_git_repository")
	}

	return nil
}

// detectDefaultBranch detects the default branch of the repository
func detectDefaultBranch(repoPath string) string {
	// Try to get the default branch from origin
	cmd := exec.Command("git", "symbolic-ref", "refs/remotes/origin/HEAD")
	cmd.Dir = repoPath

	output, err := cmd.Output()
	if err == nil {
		parts := strings.Split(strings.TrimSpace(string(output)), "/")
		if len(parts) > 0 {
			return parts[len(parts)-1]
		}
	}

	// Check common branch names
	for _, branch := range []string{"main", "master"} {
		cmd = exec.Command("git", "rev-parse", "--verify", branch)
		cmd.Dir = repoPath
		if err := cmd.Run(); err == nil {
			return branch
		}
	}

	return "main"
}
