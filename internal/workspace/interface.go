// Package workspace provides git worktree isolation for AI agents
package workspace

import (
	"context"
	"time"
)

// Workspace represents an isolated git worktree for agent operations
type Workspace interface {
	// ID returns the unique identifier for this workspace
	ID() string

	// Path returns the filesystem path to the workspace directory
	Path() string

	// Branch returns the git branch name for this workspace
	Branch() string

	// Status returns the current status of the workspace
	Status() WorkspaceStatus

	// LastActivity returns the timestamp of the last activity in this workspace
	LastActivity() time.Time

	// Cleanup removes the workspace from disk
	Cleanup() error
}

// Manager handles creation and lifecycle of workspaces
type Manager interface {
	// CreateWorkspace creates a new isolated workspace for an agent
	CreateWorkspace(ctx context.Context, opts CreateOptions) (Workspace, error)

	// GetWorkspace retrieves a workspace by ID
	GetWorkspace(id string) (Workspace, error)

	// ListWorkspaces returns all active workspaces
	ListWorkspaces() ([]Workspace, error)

	// CleanupWorkspace removes a workspace and its worktree
	CleanupWorkspace(id string) error

	// CleanupInactive removes workspaces that have been inactive beyond the threshold
	CleanupInactive(threshold time.Duration) error
}

// CreateOptions configures workspace creation
type CreateOptions struct {
	// AgentID is the identifier of the agent requesting the workspace
	AgentID string

	// BaseBranch is the branch to create the workspace from (default: main)
	BaseBranch string

	// BranchPrefix is the prefix for the new branch name
	BranchPrefix string

	// RepoPath is the path to the source repository
	RepoPath string

	// WorkDir is the base directory for creating workspaces
	WorkDir string
}

// WorkspaceStatus represents the current state of a workspace
type WorkspaceStatus string

const (
	// StatusActive indicates the workspace is in use
	StatusActive WorkspaceStatus = "active"

	// StatusIdle indicates the workspace exists but is not actively used
	StatusIdle WorkspaceStatus = "idle"

	// StatusCleaning indicates the workspace is being cleaned up
	StatusCleaning WorkspaceStatus = "cleaning"

	// StatusError indicates the workspace is in an error state
	StatusError WorkspaceStatus = "error"
)