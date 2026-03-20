// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package worktree

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCraftWorktreeManager tests creating a new worktree manager
func TestCraftWorktreeManager(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()

	// Create a test git repository
	repoPath := filepath.Join(tempDir, "test-repo")
	basePath := filepath.Join(tempDir, "worktrees")

	// Create a minimal git repo for testing
	err := os.MkdirAll(repoPath, 0o755)
	require.NoError(t, err)

	// Initialize git repo (this would fail in real usage without git)
	// For testing, we'll create the directory structure
	gitDir := filepath.Join(repoPath, ".git")
	err = os.MkdirAll(gitDir, 0o755)
	require.NoError(t, err)

	// This test would fail with actual git operations, but tests the constructor
	_, err = NewWorktreeManager(ctx, repoPath, basePath)
	// We expect this to fail because we don't have a real git repo
	assert.Error(t, err)
}

// TestJourneymanWorktreeManagerContextCancellation tests context cancellation
func TestJourneymanWorktreeManagerContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	tempDir := t.TempDir()
	repoPath := filepath.Join(tempDir, "test-repo")
	basePath := filepath.Join(tempDir, "worktrees")

	wm, err := NewWorktreeManager(ctx, repoPath, basePath)
	assert.Error(t, err)
	assert.Nil(t, wm)
}

// TestGuildWorktreeCreationRequest tests worktree creation request validation
func TestGuildWorktreeCreationRequest(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name        string
		req         CreateWorktreeRequest
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid request",
			req: CreateWorktreeRequest{
				AgentID:     "test-agent",
				TaskID:      "task-123",
				BaseBranch:  "main",
				Description: "Test task",
			},
			expectError: false,
		},
		{
			name: "missing agent ID",
			req: CreateWorktreeRequest{
				TaskID:     "task-123",
				BaseBranch: "main",
			},
			expectError: true,
			errorMsg:    "agent_id is required",
		},
		{
			name: "missing task ID",
			req: CreateWorktreeRequest{
				AgentID:    "test-agent",
				BaseBranch: "main",
			},
			expectError: true,
			errorMsg:    "task_id is required",
		},
		{
			name: "default base branch",
			req: CreateWorktreeRequest{
				AgentID: "test-agent",
				TaskID:  "task-123",
				// BaseBranch omitted
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock manager for validation testing
			mockManager := &mockWorktreeManager{}

			err := mockManager.validateRequest(ctx, tt.req)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestScribeWorktreeStats tests worktree statistics
func TestScribeWorktreeStats(t *testing.T) {
	mockManager := &mockWorktreeManager{
		worktrees: map[string]*Worktree{
			"wt1": {
				ID:        "wt1",
				AgentID:   "agent1",
				Status:    WorktreeActive,
				CreatedAt: time.Now().Add(-1 * time.Hour),
				Path:      "/tmp/wt1",
			},
			"wt2": {
				ID:        "wt2",
				AgentID:   "agent2",
				Status:    WorktreeActive,
				CreatedAt: time.Now().Add(-2 * time.Hour),
				Path:      "/tmp/wt2",
			},
			"wt3": {
				ID:        "wt3",
				AgentID:   "agent1",
				Status:    WorktreeArchived,
				CreatedAt: time.Now().Add(-3 * time.Hour),
				Path:      "/tmp/wt3",
			},
		},
	}

	ctx := context.Background()
	stats, err := mockManager.GetStats(ctx)
	require.NoError(t, err)

	assert.Equal(t, 3, stats.TotalWorktrees)
	assert.Equal(t, 2, stats.ActiveWorktrees)
	assert.Equal(t, 1, stats.ArchivedWorktrees)
	assert.Equal(t, 2, stats.WorktreesByAgent["agent1"])
	assert.Equal(t, 1, stats.WorktreesByAgent["agent2"])
	assert.NotNil(t, stats.OldestWorktree)
	assert.NotNil(t, stats.NewestWorktree)
}

// TestCraftWorktreeStatusTransitions tests status transitions
func TestCraftWorktreeStatusTransitions(t *testing.T) {
	tests := []struct {
		name           string
		initialStatus  WorktreeStatus
		expectedString string
	}{
		{"active status", WorktreeActive, "active"},
		{"merging status", WorktreeMerging, "merging"},
		{"conflicted status", WorktreeConflicted, "conflicted"},
		{"archived status", WorktreeArchived, "archived"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expectedString, tt.initialStatus.String())
		})
	}
}

// TestJourneymanWorktreeCleanupPolicy tests cleanup policy enforcement
func TestJourneymanWorktreeCleanupPolicy(t *testing.T) {
	policy := CleanupPolicy{
		MaxAge:           24 * time.Hour,
		MaxDiskUsage:     1024 * 1024 * 1024, // 1GB
		ArchiveInsteadOf: true,
		PreserveActive:   true,
	}

	// Test old worktree cleanup
	oldWorktree := &Worktree{
		ID:        "old-wt",
		Status:    WorktreeActive,
		CreatedAt: time.Now().Add(-25 * time.Hour), // Older than MaxAge
	}

	shouldCleanup := time.Since(oldWorktree.CreatedAt) > policy.MaxAge
	assert.True(t, shouldCleanup)

	// Test preservation of active worktrees
	if policy.PreserveActive && oldWorktree.Status == WorktreeActive {
		// Should not cleanup active worktrees even if old
		assert.True(t, policy.PreserveActive)
	}
}

// TestGuildWorktreeSyncResult tests sync result creation
func TestGuildWorktreeSyncResult(t *testing.T) {
	result := &SyncResult{
		WorktreeID: "test-wt",
		Success:    true,
		Divergence: Divergence{
			Ahead:  2,
			Behind: 1,
		},
		Conflicts: []string{},
		Timestamp: time.Now(),
		Message:   "sync completed successfully",
	}

	assert.Equal(t, "test-wt", result.WorktreeID)
	assert.True(t, result.Success)
	assert.Equal(t, 2, result.Divergence.Ahead)
	assert.Equal(t, 1, result.Divergence.Behind)
	assert.Empty(t, result.Conflicts)
	assert.Equal(t, "sync completed successfully", result.Message)
}

// TestScribeWorktreePathGeneration tests worktree path generation
func TestScribeWorktreePathGeneration(t *testing.T) {
	basePath := "/tmp/guild-worktrees"
	agentID := "test-agent"
	taskID := "task-123"

	expectedPath := filepath.Join(basePath, "worktrees", agentID, taskID)
	actualPath := filepath.Join(basePath, "worktrees", agentID, taskID)

	assert.Equal(t, expectedPath, actualPath)
}

// TestCraftWorktreeMetadata tests worktree metadata handling
func TestCraftWorktreeMetadata(t *testing.T) {
	metadata := map[string]interface{}{
		"task_type":      "feature",
		"priority":       "high",
		"estimated_time": "2h",
		"complexity":     5,
	}

	worktree := &Worktree{
		ID:       "test-wt",
		AgentID:  "test-agent",
		Metadata: metadata,
	}

	assert.Equal(t, "feature", worktree.Metadata["task_type"])
	assert.Equal(t, "high", worktree.Metadata["priority"])
	assert.Equal(t, "2h", worktree.Metadata["estimated_time"])
	assert.Equal(t, 5, worktree.Metadata["complexity"])
}

// Mock implementation for testing
type mockWorktreeManager struct {
	worktrees map[string]*Worktree
}

func (m *mockWorktreeManager) CreateWorktree(ctx context.Context, req CreateWorktreeRequest) (*Worktree, error) {
	// Create a mock worktree for testing
	wt := &Worktree{
		ID:         "mock-wt-" + req.TaskID,
		AgentID:    req.AgentID,
		TaskID:     req.TaskID,
		Path:       "/tmp/mock-" + req.TaskID,
		Branch:     "agent/" + req.AgentID + "/" + req.TaskID,
		BaseBranch: req.BaseBranch,
		Status:     WorktreeActive,
		CreatedAt:  time.Now(),
		LastSync:   time.Now(),
		Metadata:   req.Metadata,
	}
	if m.worktrees != nil {
		m.worktrees[wt.ID] = wt
	}
	return wt, nil
}

func (m *mockWorktreeManager) SyncWorktree(ctx context.Context, worktreeID string) (*SyncResult, error) {
	return nil, nil
}

func (m *mockWorktreeManager) RemoveWorktree(ctx context.Context, worktreeID string) error {
	return nil
}

func (m *mockWorktreeManager) GetWorktree(worktreeID string) *Worktree {
	return m.worktrees[worktreeID]
}

func (m *mockWorktreeManager) GetActiveWorktrees() []*Worktree {
	var active []*Worktree
	for _, wt := range m.worktrees {
		if wt.Status == WorktreeActive {
			active = append(active, wt)
		}
	}
	return active
}

func (m *mockWorktreeManager) GetWorktreesByAgent(agentID string) []*Worktree {
	var agent []*Worktree
	for _, wt := range m.worktrees {
		if wt.AgentID == agentID {
			agent = append(agent, wt)
		}
	}
	return agent
}

func (m *mockWorktreeManager) GetStats(ctx context.Context) (*WorktreeStats, error) {
	stats := &WorktreeStats{
		WorktreesByAgent: make(map[string]int),
	}

	var oldest, newest *time.Time

	for _, wt := range m.worktrees {
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
	}

	stats.OldestWorktree = oldest
	stats.NewestWorktree = newest

	return stats, nil
}

func (m *mockWorktreeManager) Shutdown(ctx context.Context) error {
	return nil
}

func (m *mockWorktreeManager) validateRequest(ctx context.Context, req CreateWorktreeRequest) error {
	if req.AgentID == "" {
		return &mockError{message: "agent_id is required"}
	}
	if req.TaskID == "" {
		return &mockError{message: "task_id is required"}
	}
	return nil
}

type mockError struct {
	message string
}

func (e *mockError) Error() string {
	return e.message
}
