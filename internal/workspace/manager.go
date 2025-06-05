package workspace

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
)

// manager implements the Manager interface
type manager struct {
	mu         sync.RWMutex
	workspaces map[string]*workspace
	baseDir    string
	repoPath   string
}

// NewManager creates a new workspace manager
func NewManager(baseDir, repoPath string) (Manager, error) {
	// Ensure base directory exists
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create base directory: %w", err)
	}

	// Ensure workspaces subdirectory exists
	workspacesDir := filepath.Join(baseDir, "workspaces")
	if err := os.MkdirAll(workspacesDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create workspaces directory: %w", err)
	}

	return &manager{
		workspaces: make(map[string]*workspace),
		baseDir:    baseDir,
		repoPath:   repoPath,
	}, nil
}

// CreateWorkspace creates a new isolated workspace for an agent
func (m *manager) CreateWorkspace(ctx context.Context, opts CreateOptions) (Workspace, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Generate unique ID
	id := generateWorkspaceID(opts.AgentID)

	// Create workspace directory
	wsPath := workspacePath(m.baseDir, id)
	if err := os.MkdirAll(wsPath, 0755); err != nil {
		return nil, &WorkspaceError{
			Op:  "create",
			ID:  id,
			Err: err,
		}
	}

	// Determine branch name
	branchName := generateBranchName(opts.BranchPrefix, id)

	// Create workspace info
	info := &WorkspaceInfo{
		ID:           id,
		AgentID:      opts.AgentID,
		Path:         wsPath,
		Branch:       branchName,
		BaseBranch:   opts.BaseBranch,
		Status:       StatusActive,
		CreatedAt:    time.Now(),
		LastActivity: time.Now(),
	}

	// Create workspace instance
	ws := &workspace{
		info:     info,
		repoPath: m.repoPath,
	}

	// Create git worktree (will be implemented later)
	if err := m.createWorktree(ws, opts); err != nil {
		// Cleanup on failure
		os.RemoveAll(wsPath)
		return nil, &WorkspaceError{
			Op:  "create_worktree",
			ID:  id,
			Err: err,
		}
	}

	// Store workspace
	m.workspaces[id] = ws

	return ws, nil
}

// GetWorkspace retrieves a workspace by ID
func (m *manager) GetWorkspace(id string) (Workspace, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	ws, exists := m.workspaces[id]
	if !exists {
		return nil, &WorkspaceError{
			Op:      "get",
			ID:      id,
			Message: "workspace not found",
		}
	}

	return ws, nil
}

// ListWorkspaces returns all active workspaces
func (m *manager) ListWorkspaces() ([]Workspace, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	workspaces := make([]Workspace, 0, len(m.workspaces))
	for _, ws := range m.workspaces {
		workspaces = append(workspaces, ws)
	}

	return workspaces, nil
}

// CleanupWorkspace removes a workspace and its worktree
func (m *manager) CleanupWorkspace(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	ws, exists := m.workspaces[id]
	if !exists {
		return &WorkspaceError{
			Op:      "cleanup",
			ID:      id,
			Message: "workspace not found",
		}
	}

	// Cleanup the workspace
	if err := ws.Cleanup(); err != nil {
		return err
	}

	// Remove from map
	delete(m.workspaces, id)

	return nil
}

// CleanupInactive removes workspaces that have been inactive beyond the threshold
func (m *manager) CleanupInactive(threshold time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	toCleanup := []string{}

	// Identify inactive workspaces
	for id, ws := range m.workspaces {
		if now.Sub(ws.LastActivity()) > threshold && ws.Status() == StatusIdle {
			toCleanup = append(toCleanup, id)
		}
	}

	// Cleanup inactive workspaces
	var errs []error
	for _, id := range toCleanup {
		ws := m.workspaces[id]
		if err := ws.Cleanup(); err != nil {
			errs = append(errs, fmt.Errorf("failed to cleanup workspace %s: %w", id, err))
			continue
		}
		delete(m.workspaces, id)
	}

	if len(errs) > 0 {
		return fmt.Errorf("cleanup errors: %v", errs)
	}

	return nil
}

// createWorktree creates a git worktree for the workspace
func (m *manager) createWorktree(ws *workspace, opts CreateOptions) error {
	// This will be implemented when we add git operations
	// For now, return nil to allow basic structure to compile
	return nil
}

// generateWorkspaceID creates a unique ID for a workspace
func generateWorkspaceID(agentID string) string {
	return fmt.Sprintf("%s-%s", agentID, uuid.New().String()[:8])
}

// generateBranchName creates a branch name for the workspace
func generateBranchName(prefix, id string) string {
	if prefix == "" {
		prefix = "agent"
	}
	timestamp := time.Now().Format("20060102-150405")
	return fmt.Sprintf("%s/%s-%s", prefix, id, timestamp)
}