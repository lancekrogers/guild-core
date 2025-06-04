package workspace

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
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
		return fmt.Errorf("workspace path error: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("workspace path is not a directory: %s", w.info.Path)
	}
	return nil
}

// workspacePath constructs the full path for a workspace
func workspacePath(baseDir, id string) string {
	return filepath.Join(baseDir, "workspaces", id)
}