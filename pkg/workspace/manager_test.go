package workspace

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewManager(t *testing.T) {
	tmpDir := t.TempDir()
	repoPath := "/tmp/test-repo"

	manager, err := NewManager(tmpDir, repoPath)
	require.NoError(t, err)
	require.NotNil(t, manager)

	// Check that workspaces directory was created
	workspacesDir := filepath.Join(tmpDir, "workspaces")
	_, err = os.Stat(workspacesDir)
	assert.NoError(t, err)
}

func TestCreateWorkspace(t *testing.T) {
	tmpDir := t.TempDir()
	repoPath := "/tmp/test-repo"

	manager, err := NewManager(tmpDir, repoPath)
	require.NoError(t, err)

	opts := CreateOptions{
		AgentID:      "test-agent",
		BaseBranch:   "main",
		BranchPrefix: "feature",
		RepoPath:     repoPath,
		WorkDir:      tmpDir,
	}

	ctx := context.Background()
	ws, err := manager.CreateWorkspace(ctx, opts)
	require.NoError(t, err)
	require.NotNil(t, ws)

	// Verify workspace properties
	assert.Contains(t, ws.ID(), "test-agent")
	assert.Contains(t, ws.Branch(), "feature/")
	assert.Equal(t, StatusActive, ws.Status())
	assert.NotEmpty(t, ws.Path())

	// Verify workspace directory exists
	_, err = os.Stat(ws.Path())
	assert.NoError(t, err)
}

func TestGetWorkspace(t *testing.T) {
	tmpDir := t.TempDir()
	repoPath := "/tmp/test-repo"

	manager, err := NewManager(tmpDir, repoPath)
	require.NoError(t, err)

	// Create a workspace
	opts := CreateOptions{
		AgentID:      "test-agent",
		BaseBranch:   "main",
		BranchPrefix: "feature",
		RepoPath:     repoPath,
		WorkDir:      tmpDir,
	}

	ctx := context.Background()
	created, err := manager.CreateWorkspace(ctx, opts)
	require.NoError(t, err)

	// Get the workspace
	retrieved, err := manager.GetWorkspace(created.ID())
	require.NoError(t, err)
	assert.Equal(t, created.ID(), retrieved.ID())

	// Try to get non-existent workspace
	_, err = manager.GetWorkspace("non-existent")
	assert.Error(t, err)
}

func TestListWorkspaces(t *testing.T) {
	tmpDir := t.TempDir()
	repoPath := "/tmp/test-repo"

	manager, err := NewManager(tmpDir, repoPath)
	require.NoError(t, err)

	// Initially empty
	workspaces, err := manager.ListWorkspaces()
	require.NoError(t, err)
	assert.Empty(t, workspaces)

	// Create multiple workspaces
	ctx := context.Background()
	for i := 0; i < 3; i++ {
		opts := CreateOptions{
			AgentID:      "test-agent",
			BaseBranch:   "main",
			BranchPrefix: "feature",
			RepoPath:     repoPath,
			WorkDir:      tmpDir,
		}
		_, err := manager.CreateWorkspace(ctx, opts)
		require.NoError(t, err)
	}

	// List should now have 3
	workspaces, err = manager.ListWorkspaces()
	require.NoError(t, err)
	assert.Len(t, workspaces, 3)
}

func TestCleanupWorkspace(t *testing.T) {
	tmpDir := t.TempDir()
	repoPath := "/tmp/test-repo"

	manager, err := NewManager(tmpDir, repoPath)
	require.NoError(t, err)

	// Create a workspace
	opts := CreateOptions{
		AgentID:      "test-agent",
		BaseBranch:   "main",
		BranchPrefix: "feature",
		RepoPath:     repoPath,
		WorkDir:      tmpDir,
	}

	ctx := context.Background()
	ws, err := manager.CreateWorkspace(ctx, opts)
	require.NoError(t, err)

	wsPath := ws.Path()
	wsID := ws.ID()

	// Cleanup the workspace
	err = manager.CleanupWorkspace(wsID)
	require.NoError(t, err)

	// Verify it's gone from manager
	_, err = manager.GetWorkspace(wsID)
	assert.Error(t, err)

	// Verify directory is removed
	_, err = os.Stat(wsPath)
	assert.True(t, os.IsNotExist(err))
}

func TestCleanupInactive(t *testing.T) {
	tmpDir := t.TempDir()
	repoPath := "/tmp/test-repo"

	manager, err := NewManager(tmpDir, repoPath)
	require.NoError(t, err)

	// Create workspaces
	ctx := context.Background()
	opts := CreateOptions{
		AgentID:      "test-agent",
		BaseBranch:   "main",
		BranchPrefix: "feature",
		RepoPath:     repoPath,
		WorkDir:      tmpDir,
	}

	// Create active workspace
	active, err := manager.CreateWorkspace(ctx, opts)
	require.NoError(t, err)

	// Create idle workspace (we'll need to modify this when we implement status updates)
	idle, err := manager.CreateWorkspace(ctx, opts)
	require.NoError(t, err)

	// For now, manually set to idle for testing
	idleWs := idle.(*workspace)
	idleWs.info.Status = StatusIdle
	idleWs.info.LastActivity = time.Now().Add(-2 * time.Hour)

	// Cleanup with 1 hour threshold
	err = manager.CleanupInactive(1 * time.Hour)
	require.NoError(t, err)

	// Active should still exist
	_, err = manager.GetWorkspace(active.ID())
	assert.NoError(t, err)

	// Idle should be gone
	_, err = manager.GetWorkspace(idle.ID())
	assert.Error(t, err)
}