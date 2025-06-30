// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package worktree

import "context"

// Manager defines the interface for git worktree management
type Manager interface {
	// CreateWorktree creates a new isolated worktree for an agent
	CreateWorktree(ctx context.Context, req CreateWorktreeRequest) (*Worktree, error)
	
	// SyncWorktree synchronizes a worktree with its base branch
	SyncWorktree(ctx context.Context, worktreeID string) (*SyncResult, error)
	
	// RemoveWorktree removes a worktree and cleans up resources
	RemoveWorktree(ctx context.Context, worktreeID string) error
	
	// GetWorktree retrieves a worktree by ID
	GetWorktree(worktreeID string) *Worktree
	
	// GetActiveWorktrees returns all active worktrees
	GetActiveWorktrees() []*Worktree
	
	// GetWorktreesByAgent returns all worktrees for a specific agent
	GetWorktreesByAgent(agentID string) []*Worktree
	
	// GetStats returns usage statistics for worktrees
	GetStats(ctx context.Context) (*WorktreeStats, error)
	
	// Shutdown gracefully shuts down the worktree manager
	Shutdown(ctx context.Context) error
}

// Ensure WorktreeManager implements the Manager interface
var _ Manager = (*WorktreeManager)(nil)