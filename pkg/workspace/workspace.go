// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package workspace

import (
	"os"
	"path/filepath"
	"time"

	"github.com/guild-ventures/guild-core/pkg/gerror"
)

// workspace implements the Workspace interface
type workspace struct {
	info     *WorkspaceInfo
	gitInfo  *GitInfo
	repoPath string
}

// NewWorkspace creates a new workspace instance
func NewWorkspace(info *WorkspaceInfo, repoPath string) Workspace {
	return &workspace{
		info:     info,
		repoPath: repoPath,
		gitInfo:  &GitInfo{},
	}
}

// ID returns the unique identifier for this workspace
func (w *workspace) ID() string {
	return w.info.ID
}

// Path returns the filesystem path to the workspace directory
func (w *workspace) Path() string {
	return w.info.Path
}

// Branch returns the git branch name for this workspace
func (w *workspace) Branch() string {
	return w.info.Branch
}

// Status returns the current status of the workspace
func (w *workspace) Status() WorkspaceStatus {
	return w.info.Status
}

// LastActivity returns the timestamp of the last activity in this workspace
func (w *workspace) LastActivity() time.Time {
	return w.info.LastActivity
}

// Cleanup removes the workspace from disk
func (w *workspace) Cleanup() error {
	// Update status
	w.info.Status = StatusCleaning

	// Remove the worktree
	if err := w.removeWorktree(); err != nil {
		w.info.Status = StatusError
		return &WorkspaceError{
			Op:  "cleanup",
			ID:  w.info.ID,
			Err: err,
		}
	}

	// Remove the workspace directory
	if err := os.RemoveAll(w.info.Path); err != nil {
		w.info.Status = StatusError
		return &WorkspaceError{
			Op:  "cleanup",
			ID:  w.info.ID,
			Err: err,
		}
	}

	return nil
}

// removeWorktree removes the git worktree
func (w *workspace) removeWorktree() error {
	// This will be implemented when we add git operations
	// For now, return nil to allow basic structure to compile
	return nil
}

// updateActivity updates the last activity timestamp
func (w *workspace) updateActivity() {
	w.info.LastActivity = time.Now()
}

// validatePath ensures the workspace path exists and is accessible
func (w *workspace) validatePath() error {
	info, err := os.Stat(w.info.Path)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "workspace path error").
			WithComponent("workspace").
			WithOperation("validatePath").
			WithDetails("path", w.info.Path)
	}
	if !info.IsDir() {
		return gerror.New(gerror.ErrCodeValidation, "workspace path is not a directory", nil).
			WithComponent("workspace").
			WithOperation("validatePath").
			WithDetails("path", w.info.Path)
	}
	return nil
}

// workspacePath constructs the full path for a workspace
func workspacePath(baseDir, id string) string {
	return filepath.Join(baseDir, "workspaces", id)
}
