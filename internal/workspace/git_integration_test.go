package workspace

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupTestRepo creates a test git repository
func setupTestRepo(t *testing.T) (string, func()) {
	tmpDir, err := os.MkdirTemp("", "guild-git-test-*")
	require.NoError(t, err)

	// Initialize git repo
	cmd := exec.Command("git", "init")
	cmd.Dir = tmpDir
	require.NoError(t, cmd.Run())

	// Configure git
	cmd = exec.Command("git", "config", "user.email", "test@guild.local")
	cmd.Dir = tmpDir
	require.NoError(t, cmd.Run())

	cmd = exec.Command("git", "config", "user.name", "Guild Test")
	cmd.Dir = tmpDir
	require.NoError(t, cmd.Run())

	// Create initial commit
	readmePath := filepath.Join(tmpDir, "README.md")
	err = os.WriteFile(readmePath, []byte("# Test Repo\n"), 0644)
	require.NoError(t, err)

	cmd = exec.Command("git", "add", ".")
	cmd.Dir = tmpDir
	require.NoError(t, cmd.Run())

	cmd = exec.Command("git", "commit", "-m", "Initial commit")
	cmd.Dir = tmpDir
	require.NoError(t, cmd.Run())

	cleanup := func() {
		os.RemoveAll(tmpDir)
	}

	return tmpDir, cleanup
}

func TestGitManager_CreateWorkspace(t *testing.T) {
	repoPath, cleanup := setupTestRepo(t)
	defer cleanup()

	// Create git manager
	workDir := filepath.Join(repoPath, ".guild")
	manager, err := NewGitManager(workDir, repoPath)
	require.NoError(t, err)

	// Create workspace
	opts := CreateOptions{
		AgentID:      "test-agent",
		BranchPrefix: "agent",
		WorkDir:      workDir,
		BaseBranch:   "main",
	}

	ws, err := manager.CreateWorkspace(context.Background(), opts)
	require.NoError(t, err)
	assert.NotNil(t, ws)

	// Verify workspace
	assert.DirExists(t, ws.Path())
	assert.Contains(t, ws.Branch(), "agent/test-agent")
	assert.Equal(t, StatusActive, ws.Status())

	// Verify it's a git worktree
	gitWs, ok := ws.(*GitWorkspace)
	assert.True(t, ok)
	assert.NotNil(t, gitWs.GetGitInfo())

	// Cleanup
	err = ws.Cleanup()
	assert.NoError(t, err)
}

func TestGitWorkspace_CommitChanges(t *testing.T) {
	repoPath, cleanup := setupTestRepo(t)
	defer cleanup()

	// Create workspace
	workDir := filepath.Join(repoPath, ".guild")
	manager, err := NewGitManager(workDir, repoPath)
	require.NoError(t, err)

	opts := CreateOptions{
		AgentID:      "commit-test",
		BranchPrefix: "agent",
		WorkDir:      workDir,
	}

	ws, err := manager.CreateWorkspace(context.Background(), opts)
	require.NoError(t, err)

	gitWs := ws.(*GitWorkspace)

	// Make changes
	testFile := filepath.Join(ws.Path(), "test.txt")
	err = os.WriteFile(testFile, []byte("Test content\n"), 0644)
	require.NoError(t, err)

	// Check dirty status
	gitWs.UpdateGitInfo()
	assert.True(t, gitWs.GetGitInfo().IsDirty)

	// Commit changes
	err = gitWs.CommitChanges("Test commit")
	assert.NoError(t, err)

	// Check clean status
	gitWs.UpdateGitInfo()
	assert.False(t, gitWs.GetGitInfo().IsDirty)
	assert.NotEmpty(t, gitWs.GetGitInfo().CommitHash)

	// Cleanup
	err = ws.Cleanup()
	assert.NoError(t, err)
}

func TestGitWorkspace_GetDiff(t *testing.T) {
	repoPath, cleanup := setupTestRepo(t)
	defer cleanup()

	// Create workspace
	workDir := filepath.Join(repoPath, ".guild")
	manager, err := NewGitManager(workDir, repoPath)
	require.NoError(t, err)

	opts := CreateOptions{
		AgentID:      "diff-test",
		BranchPrefix: "agent",
		WorkDir:      workDir,
	}

	ws, err := manager.CreateWorkspace(context.Background(), opts)
	require.NoError(t, err)

	gitWs := ws.(*GitWorkspace)

	// Modify README
	readmePath := filepath.Join(ws.Path(), "README.md")
	err = os.WriteFile(readmePath, []byte("# Test Repo\n\nModified content\n"), 0644)
	require.NoError(t, err)

	// Get diff
	diff, err := gitWs.GetDiff()
	assert.NoError(t, err)
	assert.Contains(t, diff, "Modified content")
	assert.Contains(t, diff, "README.md")

	// Cleanup
	err = ws.Cleanup()
	assert.NoError(t, err)
}