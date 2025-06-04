package workspace_test

import (
	"context"
	"fmt"
	"log"

	"github.com/guild-ventures/guild-core/pkg/workspace"
)

func Example_basicUsage() {
	// Create a workspace manager
	manager, err := workspace.NewManager("/tmp/guild/workspaces", "/path/to/project")
	if err != nil {
		log.Fatal(err)
	}

	// Create a workspace for an agent
	opts := workspace.CreateOptions{
		AgentID:      "research-agent-1",
		BaseBranch:   "main",
		BranchPrefix: "research",
	}

	ctx := context.Background()
	ws, err := manager.CreateWorkspace(ctx, opts)
	if err != nil {
		log.Fatal(err)
	}
	defer ws.Cleanup()

	// Agent works in the isolated workspace
	fmt.Printf("Agent working in: %s\n", ws.Path())
	fmt.Printf("On branch: %s\n", ws.Branch())
	fmt.Printf("Status: %s\n", ws.Status())
}

func Example_multipleAgents() {
	// Create a workspace manager
	manager, err := workspace.NewManager("/tmp/guild/workspaces", "/path/to/project")
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()

	// Create workspaces for multiple agents
	agents := []string{"research-agent", "code-agent", "test-agent"}
	workspaces := make([]workspace.Workspace, 0, len(agents))

	for _, agentID := range agents {
		opts := workspace.CreateOptions{
			AgentID:      agentID,
			BaseBranch:   "main",
			BranchPrefix: agentID,
		}

		ws, err := manager.CreateWorkspace(ctx, opts)
		if err != nil {
			log.Printf("Failed to create workspace for %s: %v", agentID, err)
			continue
		}
		workspaces = append(workspaces, ws)
	}

	// Each agent can work independently
	for _, ws := range workspaces {
		fmt.Printf("Workspace %s at %s on branch %s\n", 
			ws.ID(), ws.Path(), ws.Branch())
	}

	// Cleanup all workspaces when done
	for _, ws := range workspaces {
		if err := ws.Cleanup(); err != nil {
			log.Printf("Failed to cleanup workspace %s: %v", ws.ID(), err)
		}
	}
}

func Example_withConfiguration() {
	// Use custom configuration
	config := workspace.DefaultConfig()
	config.BaseDir = "/custom/workspace/dir"
	config.MaxWorkspaces = 5
	config.PreserveOnError = true

	// Validate configuration
	if err := config.Validate(); err != nil {
		log.Fatal(err)
	}

	// Configuration would be used when creating the manager
	// (This would be implemented in a future iteration)
	fmt.Printf("Max workspaces: %d\n", config.MaxWorkspaces)
	fmt.Printf("Auto cleanup: %v\n", config.AutoCleanup)
}