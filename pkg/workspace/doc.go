// Package workspace provides git worktree isolation for AI agents working on the same repository.
//
// The workspace package addresses the challenge of multiple AI agents working concurrently
// on the same codebase without conflicts. It creates isolated git worktrees for each agent,
// allowing them to:
//
//   - Work on separate branches without interference
//   - Make commits and changes in isolation
//   - Test changes without affecting other agents
//   - Coordinate merges through a controlled process
//
// # Architecture
//
// The package follows a manager pattern where a central Manager creates and tracks
// individual Workspace instances. Each workspace corresponds to a git worktree with:
//
//   - Unique filesystem location
//   - Dedicated git branch
//   - Activity tracking for cleanup
//   - Status monitoring
//
// # Usage
//
// Basic usage:
//
//	manager, err := workspace.NewManager("/var/guild/workspaces", "/path/to/repo")
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	opts := workspace.CreateOptions{
//	    AgentID:      "research-agent-1",
//	    BaseBranch:   "main",
//	    BranchPrefix: "research",
//	}
//
//	ws, err := manager.CreateWorkspace(context.Background(), opts)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer ws.Cleanup()
//
//	// Agent can now work in ws.Path() directory
//	fmt.Printf("Working in: %s on branch: %s\n", ws.Path(), ws.Branch())
//
// # Lifecycle Management
//
// Workspaces support automatic cleanup of inactive instances:
//
//	// Clean up workspaces inactive for more than 2 hours
//	err := manager.CleanupInactive(2 * time.Hour)
//
// # Integration with Guild Framework
//
// This package integrates with the Guild agent system where:
//   - Each agent (Artisan) gets its own workspace
//   - The orchestrator manages workspace lifecycle
//   - Workspaces are cleaned up after task completion
//   - Failed tasks trigger workspace preservation for debugging
package workspace
