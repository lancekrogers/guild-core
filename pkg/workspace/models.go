package workspace

import (
	"time"
)

// WorkspaceInfo contains metadata about a workspace
type WorkspaceInfo struct {
	ID           string
	AgentID      string
	Path         string
	Branch       string
	BaseBranch   string
	Status       WorkspaceStatus
	CreatedAt    time.Time
	LastActivity time.Time
}

// GitInfo contains git-related information for a workspace
type GitInfo struct {
	CommitHash   string
	BranchName   string
	RemoteURL    string
	IsDirty      bool
	TrackedFiles int
	UntrackedFiles int
}

// WorkspaceMetrics tracks usage statistics
type WorkspaceMetrics struct {
	TotalCreated   int
	ActiveCount    int
	CleanedCount   int
	AverageLifespan time.Duration
	LastCleanup    time.Time
}

// Error types for workspace operations
type WorkspaceError struct {
	Op      string
	ID      string
	Err     error
	Message string
}

func (e *WorkspaceError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	if e.Err != nil {
		return e.Op + ": " + e.Err.Error()
	}
	return e.Op + ": workspace error"
}

func (e *WorkspaceError) Unwrap() error {
	return e.Err
}