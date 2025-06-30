// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package worktree

import (
	"time"

	"github.com/go-git/go-git/v5"
)

// WorktreeStatus represents the current status of a worktree
type WorktreeStatus int

const (
	WorktreeActive WorktreeStatus = iota
	WorktreeMerging
	WorktreeConflicted
	WorktreeArchived
)

// String returns the string representation of WorktreeStatus
func (ws WorktreeStatus) String() string {
	switch ws {
	case WorktreeActive:
		return "active"
	case WorktreeMerging:
		return "merging"
	case WorktreeConflicted:
		return "conflicted"
	case WorktreeArchived:
		return "archived"
	default:
		return "unknown"
	}
}

// Worktree represents an isolated git worktree for agent work
type Worktree struct {
	ID          string                 `json:"id"`
	AgentID     string                 `json:"agent_id"`
	Path        string                 `json:"path"`
	Branch      string                 `json:"branch"`
	BaseBranch  string                 `json:"base_branch"`
	Status      WorktreeStatus         `json:"status"`
	CreatedAt   time.Time              `json:"created_at"`
	LastSync    time.Time              `json:"last_sync"`
	Repository  *git.Repository        `json:"-"`
	Metadata    map[string]interface{} `json:"metadata"`
	TaskID      string                 `json:"task_id"`
}

// CreateWorktreeRequest contains parameters for creating a new worktree
type CreateWorktreeRequest struct {
	AgentID     string                 `json:"agent_id"`
	TaskID      string                 `json:"task_id"`
	BaseBranch  string                 `json:"base_branch"`
	Description string                 `json:"description"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// WorktreeStats provides statistics about worktree usage
type WorktreeStats struct {
	TotalWorktrees   int            `json:"total_worktrees"`
	ActiveWorktrees  int            `json:"active_worktrees"`
	ArchivedWorktrees int           `json:"archived_worktrees"`
	WorktreesByAgent map[string]int `json:"worktrees_by_agent"`
	DiskUsage        int64          `json:"disk_usage_bytes"`
	OldestWorktree   *time.Time     `json:"oldest_worktree"`
	NewestWorktree   *time.Time     `json:"newest_worktree"`
}

// Divergence represents how far ahead/behind a worktree is from its base branch
type Divergence struct {
	Ahead  int `json:"ahead"`
	Behind int `json:"behind"`
}

// SyncResult contains the result of a worktree sync operation
type SyncResult struct {
	WorktreeID string    `json:"worktree_id"`
	Success    bool      `json:"success"`
	Divergence Divergence `json:"divergence"`
	Conflicts  []string  `json:"conflicts"`
	Timestamp  time.Time `json:"timestamp"`
	Message    string    `json:"message"`
}

// CleanupPolicy defines how worktrees should be cleaned up
type CleanupPolicy struct {
	MaxAge           time.Duration `json:"max_age"`
	MaxDiskUsage     int64         `json:"max_disk_usage"`
	ArchiveInsteadOf bool          `json:"archive_instead_of_delete"`
	PreserveActive   bool          `json:"preserve_active"`
}