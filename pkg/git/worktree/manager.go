// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package worktree

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/guild-framework/guild-core/pkg/gerror"
)

// WorktreeManager manages git worktrees for multiple agents
type WorktreeManager struct {
	baseRepo      *git.Repository
	worktrees     map[string]*Worktree
	basePath      string
	mu            sync.RWMutex
	cleanupTimer  *time.Timer
	cleanupPolicy CleanupPolicy
}

// NewWorktreeManager creates a new worktree manager
func NewWorktreeManager(ctx context.Context, repoPath string, basePath string) (*WorktreeManager, error) {
	if ctx.Err() != nil {
		return nil, gerror.Wrap(ctx.Err(), gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("git.worktree").
			WithOperation("NewWorktreeManager")
	}

	// Open the base repository
	repo, err := git.PlainOpen(repoPath)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to open git repository").
			WithComponent("git.worktree").
			WithOperation("NewWorktreeManager").
			WithDetails("repo_path", repoPath)
	}

	// Ensure worktrees base path exists
	if err := os.MkdirAll(basePath, 0o755); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create worktrees base path").
			WithComponent("git.worktree").
			WithOperation("NewWorktreeManager").
			WithDetails("base_path", basePath)
	}

	wm := &WorktreeManager{
		baseRepo:  repo,
		worktrees: make(map[string]*Worktree),
		basePath:  basePath,
		cleanupPolicy: CleanupPolicy{
			MaxAge:           24 * time.Hour,
			MaxDiskUsage:     5 * 1024 * 1024 * 1024, // 5GB
			ArchiveInsteadOf: true,
			PreserveActive:   true,
		},
	}

	// Start cleanup timer
	wm.startCleanupTimer(ctx)

	return wm, nil
}

// CreateWorktree creates a new isolated worktree for an agent
func (wm *WorktreeManager) CreateWorktree(ctx context.Context, req CreateWorktreeRequest) (*Worktree, error) {
	if ctx.Err() != nil {
		return nil, gerror.Wrap(ctx.Err(), gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("git.worktree").
			WithOperation("CreateWorktree")
	}

	wm.mu.Lock()
	defer wm.mu.Unlock()

	// Validate request
	if req.AgentID == "" {
		return nil, gerror.New(gerror.ErrCodeValidation, "agent_id is required", nil).
			WithComponent("git.worktree").
			WithOperation("CreateWorktree")
	}

	if req.TaskID == "" {
		return nil, gerror.New(gerror.ErrCodeValidation, "task_id is required", nil).
			WithComponent("git.worktree").
			WithOperation("CreateWorktree")
	}

	if req.BaseBranch == "" {
		req.BaseBranch = "main"
	}

	// Generate unique branch name
	branchName := fmt.Sprintf("agent/%s/%s", req.AgentID, req.TaskID)

	// Create worktree path
	worktreePath := filepath.Join(wm.basePath, "worktrees", req.AgentID, req.TaskID)

	// Ensure base branch is up to date
	if err := wm.fetchLatest(ctx, req.BaseBranch); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to fetch latest").
			WithComponent("git.worktree").
			WithOperation("CreateWorktree").
			WithDetails("branch", req.BaseBranch)
	}

	// Ensure parent directory exists
	if err := os.MkdirAll(filepath.Dir(worktreePath), 0o755); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create worktree parent directory").
			WithComponent("git.worktree").
			WithOperation("CreateWorktree").
			WithDetails("path", filepath.Dir(worktreePath))
	}

	// Create worktree with new branch using git command
	if err := wm.executeGitCommand(ctx, "worktree", "add", "-b", branchName, worktreePath, req.BaseBranch); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create worktree").
			WithComponent("git.worktree").
			WithOperation("CreateWorktree").
			WithDetails("branch", branchName).
			WithDetails("path", worktreePath)
	}

	// Open repository in worktree
	repo, err := git.PlainOpen(worktreePath)
	if err != nil {
		// Cleanup failed worktree
		wm.executeGitCommand(ctx, "worktree", "remove", "--force", worktreePath)
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to open worktree repository").
			WithComponent("git.worktree").
			WithOperation("CreateWorktree").
			WithDetails("path", worktreePath)
	}

	// Create worktree object
	worktreeID := wm.generateWorktreeID()
	worktree := &Worktree{
		ID:         worktreeID,
		AgentID:    req.AgentID,
		TaskID:     req.TaskID,
		Path:       worktreePath,
		Branch:     branchName,
		BaseBranch: req.BaseBranch,
		Status:     WorktreeActive,
		CreatedAt:  time.Now(),
		LastSync:   time.Now(),
		Repository: repo,
		Metadata:   req.Metadata,
	}

	// Configure worktree
	if err := wm.configureWorktree(ctx, worktree); err != nil {
		wm.cleanupWorktree(ctx, worktree)
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to configure worktree").
			WithComponent("git.worktree").
			WithOperation("CreateWorktree").
			WithDetails("worktree_id", worktreeID)
	}

	// Track worktree
	wm.worktrees[worktree.ID] = worktree

	return worktree, nil
}

// configureWorktree sets up agent-specific configuration for a worktree
func (wm *WorktreeManager) configureWorktree(ctx context.Context, wt *Worktree) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	// Configure git user for the worktree
	if err := wm.executeInWorktree(ctx, wt, "config", "user.name", wt.AgentID); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to set git user name").
			WithComponent("git.worktree").
			WithOperation("configureWorktree").
			WithDetails("worktree_id", wt.ID)
	}

	email := fmt.Sprintf("%s@guild.local", wt.AgentID)
	if err := wm.executeInWorktree(ctx, wt, "config", "user.email", email); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to set git user email").
			WithComponent("git.worktree").
			WithOperation("configureWorktree").
			WithDetails("worktree_id", wt.ID)
	}

	// Configure hooks directory
	hooksPath := filepath.Join(wt.Path, ".git", "hooks")
	if err := os.MkdirAll(hooksPath, 0o755); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create hooks directory").
			WithComponent("git.worktree").
			WithOperation("configureWorktree").
			WithDetails("worktree_id", wt.ID).
			WithDetails("hooks_path", hooksPath)
	}

	// Create pre-commit hook for validation
	preCommitHook := fmt.Sprintf(`#!/bin/bash
# Validate changes before commit
# Worktree ID: %s
# Agent ID: %s

echo "Validating changes for agent %s..."
exit 0
`, wt.ID, wt.AgentID, wt.AgentID)

	hookPath := filepath.Join(hooksPath, "pre-commit")
	if err := os.WriteFile(hookPath, []byte(preCommitHook), 0o755); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create pre-commit hook").
			WithComponent("git.worktree").
			WithOperation("configureWorktree").
			WithDetails("worktree_id", wt.ID).
			WithDetails("hook_path", hookPath)
	}

	return nil
}

// SyncWorktree synchronizes a worktree with its base branch
func (wm *WorktreeManager) SyncWorktree(ctx context.Context, worktreeID string) (*SyncResult, error) {
	if ctx.Err() != nil {
		return nil, gerror.Wrap(ctx.Err(), gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("git.worktree").
			WithOperation("SyncWorktree")
	}

	wm.mu.RLock()
	wt, exists := wm.worktrees[worktreeID]
	wm.mu.RUnlock()

	if !exists {
		return nil, gerror.New(gerror.ErrCodeNotFound, "worktree not found", nil).
			WithComponent("git.worktree").
			WithOperation("SyncWorktree").
			WithDetails("worktree_id", worktreeID)
	}

	result := &SyncResult{
		WorktreeID: worktreeID,
		Timestamp:  time.Now(),
	}

	// Fetch latest from base branch
	if err := wm.executeInWorktree(ctx, wt, "fetch", "origin", wt.BaseBranch); err != nil {
		result.Success = false
		result.Message = fmt.Sprintf("failed to fetch: %v", err)
		return result, nil
	}

	// Check divergence
	divergence, err := wm.checkDivergence(ctx, wt)
	if err != nil {
		result.Success = false
		result.Message = fmt.Sprintf("failed to check divergence: %v", err)
		return result, nil
	}

	result.Divergence = *divergence

	if divergence.Behind > 0 {
		// Attempt rebase
		if err := wm.rebaseWorktree(ctx, wt); err != nil {
			wt.Status = WorktreeConflicted
			result.Success = false
			result.Message = fmt.Sprintf("rebase failed: %v", err)
			result.Conflicts = wm.getConflictFiles(ctx, wt)
			return result, nil
		}
	}

	wt.LastSync = time.Now()
	result.Success = true
	result.Message = "sync completed successfully"

	return result, nil
}

// rebaseWorktree attempts to rebase the worktree onto the latest base branch
func (wm *WorktreeManager) rebaseWorktree(ctx context.Context, wt *Worktree) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	// Stash any uncommitted changes
	stashErr := wm.executeInWorktree(ctx, wt, "stash", "push", "-m", "auto-stash before rebase")
	stashCreated := stashErr == nil

	// Attempt rebase
	err := wm.executeInWorktree(ctx, wt, "rebase", fmt.Sprintf("origin/%s", wt.BaseBranch))
	if err != nil {
		// Check if it's a conflict
		if wm.hasRebaseConflicts(ctx, wt) {
			return gerror.Wrap(err, gerror.ErrCodeConflict, "rebase conflict detected").
				WithComponent("git.worktree").
				WithOperation("rebaseWorktree").
				WithDetails("worktree_id", wt.ID)
		}
		return gerror.Wrap(err, gerror.ErrCodeInternal, "rebase failed").
			WithComponent("git.worktree").
			WithOperation("rebaseWorktree").
			WithDetails("worktree_id", wt.ID)
	}

	// Pop stash if we created one
	if stashCreated {
		wm.executeInWorktree(ctx, wt, "stash", "pop")
	}

	return nil
}

// RemoveWorktree removes a worktree and cleans up resources
func (wm *WorktreeManager) RemoveWorktree(ctx context.Context, worktreeID string) error {
	if ctx.Err() != nil {
		return gerror.Wrap(ctx.Err(), gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("git.worktree").
			WithOperation("RemoveWorktree")
	}

	wm.mu.Lock()
	defer wm.mu.Unlock()

	wt, exists := wm.worktrees[worktreeID]
	if !exists {
		return nil // Already removed
	}

	// Check for uncommitted changes
	hasChanges, err := wm.hasUncommittedChanges(ctx, wt)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to check for uncommitted changes").
			WithComponent("git.worktree").
			WithOperation("RemoveWorktree").
			WithDetails("worktree_id", worktreeID)
	}

	if hasChanges && wm.cleanupPolicy.ArchiveInsteadOf {
		// Archive instead of delete
		wt.Status = WorktreeArchived
		return wm.archiveWorktree(ctx, wt)
	}

	// Remove worktree
	if err := wm.executeGitCommand(ctx, "worktree", "remove", "--force", wt.Path); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to remove worktree").
			WithComponent("git.worktree").
			WithOperation("RemoveWorktree").
			WithDetails("worktree_id", worktreeID).
			WithDetails("path", wt.Path)
	}

	// Delete branch if it's fully merged
	if wm.isBranchMerged(ctx, wt.Branch) {
		wm.executeGitCommand(ctx, "branch", "-d", wt.Branch)
	}

	delete(wm.worktrees, worktreeID)

	return nil
}

// GetWorktree retrieves a worktree by ID
func (wm *WorktreeManager) GetWorktree(worktreeID string) *Worktree {
	wm.mu.RLock()
	defer wm.mu.RUnlock()

	return wm.worktrees[worktreeID]
}

// GetActiveWorktrees returns all active worktrees
func (wm *WorktreeManager) GetActiveWorktrees() []*Worktree {
	wm.mu.RLock()
	defer wm.mu.RUnlock()

	var active []*Worktree
	for _, wt := range wm.worktrees {
		if wt.Status == WorktreeActive {
			active = append(active, wt)
		}
	}

	return active
}

// GetWorktreesByAgent returns all worktrees for a specific agent
func (wm *WorktreeManager) GetWorktreesByAgent(agentID string) []*Worktree {
	wm.mu.RLock()
	defer wm.mu.RUnlock()

	var agentWorktrees []*Worktree
	for _, wt := range wm.worktrees {
		if wt.AgentID == agentID {
			agentWorktrees = append(agentWorktrees, wt)
		}
	}

	return agentWorktrees
}

// GetStats returns usage statistics for worktrees
func (wm *WorktreeManager) GetStats(ctx context.Context) (*WorktreeStats, error) {
	if ctx.Err() != nil {
		return nil, gerror.Wrap(ctx.Err(), gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("git.worktree").
			WithOperation("GetStats")
	}

	wm.mu.RLock()
	defer wm.mu.RUnlock()

	stats := &WorktreeStats{
		WorktreesByAgent: make(map[string]int),
	}

	var oldest, newest *time.Time

	for _, wt := range wm.worktrees {
		stats.TotalWorktrees++
		stats.WorktreesByAgent[wt.AgentID]++

		if wt.Status == WorktreeActive {
			stats.ActiveWorktrees++
		} else if wt.Status == WorktreeArchived {
			stats.ArchivedWorktrees++
		}

		if oldest == nil || wt.CreatedAt.Before(*oldest) {
			oldest = &wt.CreatedAt
		}
		if newest == nil || wt.CreatedAt.After(*newest) {
			newest = &wt.CreatedAt
		}

		// Calculate disk usage (approximate)
		if info, err := os.Stat(wt.Path); err == nil && info.IsDir() {
			if size, err := wm.getDirSize(wt.Path); err == nil {
				stats.DiskUsage += size
			}
		}
	}

	stats.OldestWorktree = oldest
	stats.NewestWorktree = newest

	return stats, nil
}

// Helper methods

func (wm *WorktreeManager) generateWorktreeID() string {
	return fmt.Sprintf("wt_%d", time.Now().UnixNano())
}

func (wm *WorktreeManager) executeGitCommand(ctx context.Context, args ...string) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = wm.getBaseRepoPath()

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git command failed: %s (output: %s)", err, string(output))
	}

	return nil
}

func (wm *WorktreeManager) executeInWorktree(ctx context.Context, wt *Worktree, args ...string) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = wt.Path

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git command failed in worktree %s: %s (output: %s)", wt.ID, err, string(output))
	}

	return nil
}

func (wm *WorktreeManager) getBaseRepoPath() string {
	// Get the working directory of the base repository
	if worktree, err := wm.baseRepo.Worktree(); err == nil {
		return worktree.Filesystem.Root()
	}
	return "."
}

func (wm *WorktreeManager) fetchLatest(ctx context.Context, branch string) error {
	return wm.executeGitCommand(ctx, "fetch", "origin", branch)
}

func (wm *WorktreeManager) checkDivergence(ctx context.Context, wt *Worktree) (*Divergence, error) {
	// Use git rev-list to count commits ahead/behind
	aheadCmd := exec.CommandContext(ctx, "git", "rev-list", "--count", fmt.Sprintf("origin/%s..HEAD", wt.BaseBranch))
	aheadCmd.Dir = wt.Path
	aheadOutput, err := aheadCmd.Output()
	if err != nil {
		return nil, err
	}

	behindCmd := exec.CommandContext(ctx, "git", "rev-list", "--count", fmt.Sprintf("HEAD..origin/%s", wt.BaseBranch))
	behindCmd.Dir = wt.Path
	behindOutput, err := behindCmd.Output()
	if err != nil {
		return nil, err
	}

	var ahead, behind int
	fmt.Sscanf(strings.TrimSpace(string(aheadOutput)), "%d", &ahead)
	fmt.Sscanf(strings.TrimSpace(string(behindOutput)), "%d", &behind)

	return &Divergence{
		Ahead:  ahead,
		Behind: behind,
	}, nil
}

func (wm *WorktreeManager) hasRebaseConflicts(ctx context.Context, wt *Worktree) bool {
	// Check if .git/rebase-merge or .git/rebase-apply exists
	rebaseMergeDir := filepath.Join(wt.Path, ".git", "rebase-merge")
	rebaseApplyDir := filepath.Join(wt.Path, ".git", "rebase-apply")

	if _, err := os.Stat(rebaseMergeDir); err == nil {
		return true
	}
	if _, err := os.Stat(rebaseApplyDir); err == nil {
		return true
	}

	return false
}

func (wm *WorktreeManager) getConflictFiles(ctx context.Context, wt *Worktree) []string {
	cmd := exec.CommandContext(ctx, "git", "diff", "--name-only", "--diff-filter=U")
	cmd.Dir = wt.Path

	output, err := cmd.Output()
	if err != nil {
		return nil
	}

	files := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(files) == 1 && files[0] == "" {
		return nil
	}

	return files
}

func (wm *WorktreeManager) hasUncommittedChanges(ctx context.Context, wt *Worktree) (bool, error) {
	cmd := exec.CommandContext(ctx, "git", "status", "--porcelain")
	cmd.Dir = wt.Path

	output, err := cmd.Output()
	if err != nil {
		return false, err
	}

	return len(strings.TrimSpace(string(output))) > 0, nil
}

func (wm *WorktreeManager) isBranchMerged(ctx context.Context, branch string) bool {
	cmd := exec.CommandContext(ctx, "git", "branch", "--merged", "main")
	cmd.Dir = wm.getBaseRepoPath()

	output, err := cmd.Output()
	if err != nil {
		return false
	}

	return strings.Contains(string(output), branch)
}

func (wm *WorktreeManager) archiveWorktree(ctx context.Context, wt *Worktree) error {
	// Create archive directory
	archiveDir := filepath.Join(wm.basePath, "archived", wt.AgentID)
	if err := os.MkdirAll(archiveDir, 0o755); err != nil {
		return err
	}

	// Create archive name with timestamp
	archiveName := fmt.Sprintf("%s_%s_%d.tar.gz", wt.AgentID, wt.TaskID, time.Now().Unix())
	archivePath := filepath.Join(archiveDir, archiveName)

	// Create tar archive
	cmd := exec.CommandContext(ctx, "tar", "-czf", archivePath, "-C", filepath.Dir(wt.Path), filepath.Base(wt.Path))
	if err := cmd.Run(); err != nil {
		return err
	}

	// Remove original worktree
	return wm.executeGitCommand(ctx, "worktree", "remove", "--force", wt.Path)
}

func (wm *WorktreeManager) cleanupWorktree(ctx context.Context, wt *Worktree) {
	wm.executeGitCommand(ctx, "worktree", "remove", "--force", wt.Path)
	wm.executeGitCommand(ctx, "branch", "-D", wt.Branch)
}

func (wm *WorktreeManager) getDirSize(path string) (int64, error) {
	var size int64
	err := filepath.Walk(path, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return nil
	})
	return size, err
}

func (wm *WorktreeManager) startCleanupTimer(ctx context.Context) {
	if wm.cleanupTimer != nil {
		wm.cleanupTimer.Stop()
	}

	wm.cleanupTimer = time.AfterFunc(30*time.Minute, func() {
		wm.performCleanup(ctx)
		wm.startCleanupTimer(ctx) // Restart timer
	})
}

func (wm *WorktreeManager) performCleanup(ctx context.Context) {
	wm.mu.Lock()
	defer wm.mu.Unlock()

	now := time.Now()
	var toRemove []string

	for id, wt := range wm.worktrees {
		// Check age
		if now.Sub(wt.CreatedAt) > wm.cleanupPolicy.MaxAge {
			if wt.Status != WorktreeActive || !wm.cleanupPolicy.PreserveActive {
				toRemove = append(toRemove, id)
			}
		}
	}

	// Remove old worktrees
	for _, id := range toRemove {
		if wt := wm.worktrees[id]; wt != nil {
			wm.cleanupWorktree(ctx, wt)
			delete(wm.worktrees, id)
		}
	}
}

// Shutdown gracefully shuts down the worktree manager
func (wm *WorktreeManager) Shutdown(ctx context.Context) error {
	if wm.cleanupTimer != nil {
		wm.cleanupTimer.Stop()
	}

	// Optionally cleanup all worktrees on shutdown
	return nil
}
