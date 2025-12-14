// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package main

import (
	"context"
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/guild-framework/guild-core/internal/daemon"
	"github.com/guild-framework/guild-core/internal/daemonconn"
	kanbanui "github.com/guild-framework/guild-core/internal/ui/kanban"
	"github.com/guild-framework/guild-core/pkg/gerror"
	"github.com/guild-framework/guild-core/pkg/kanban"
	"github.com/guild-framework/guild-core/pkg/project/local"
	"github.com/guild-framework/guild-core/pkg/registry"
	"github.com/guild-framework/guild-core/pkg/storage"
)

var (
	kanbanNoDaemon bool // Don't auto-start the Guild server
)

// kanbanCmd represents the kanban command group
var kanbanCmd = &cobra.Command{
	Use:   "kanban",
	Short: "View and manage kanban boards",
	Long: `View and manage kanban boards for tracking work progress.

The kanban board shows tasks in 5 columns:
- TODO: Tasks ready to be worked on
- IN PROGRESS: Tasks currently being worked on
- BLOCKED: Tasks that are blocked by dependencies
- READY FOR REVIEW: Tasks completed and awaiting review
- DONE: Tasks that are complete

Use 'guild kanban list' to see all boards, or 'guild kanban view <board-id>' 
to view a specific board in the interactive UI.`,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

// kanbanListCmd lists all kanban boards
var kanbanListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all kanban boards",
	Long:  `List all kanban boards with their IDs, names, and task counts.`,
	RunE:  listKanbanBoards,
}

// kanbanViewCmd views a specific kanban board
var kanbanViewCmd = &cobra.Command{
	Use:   "view [board-id]",
	Short: "View a kanban board in interactive UI",
	Long: `Launch the interactive kanban board UI for viewing and managing tasks.

If no board-id is provided, this will show the main workshop board.
The interactive UI supports:
- Navigation with arrow keys or hjkl
- Task search with '/'
- Help with '?'
- Refresh with 'r'

Example:
  guild kanban view main-board
  guild kanban view demo-board`,
	RunE: viewKanbanBoard,
}

// kanbanCreateCmd creates a new kanban board
var kanbanCreateCmd = &cobra.Command{
	Use:   "create <name> [description]",
	Short: "Create a new kanban board",
	Long: `Create a new kanban board with the specified name and optional description.

Example:
  guild kanban create "My Project" "Project tracking board"
  guild kanban create backend-tasks`,
	Args: cobra.MinimumNArgs(1),
	RunE: createKanbanBoard,
}

func init() {
	// Register kanban subcommands
	kanbanCmd.AddCommand(kanbanListCmd)
	kanbanCmd.AddCommand(kanbanViewCmd)
	kanbanCmd.AddCommand(kanbanCreateCmd)

	// Add flags
	kanbanCmd.PersistentFlags().BoolVar(&kanbanNoDaemon, "no-daemon", false, "Don't auto-start the Guild server")
}

// initializeKanbanManager creates a properly initialized kanban manager with SQLite storage
func initializeKanbanManager(ctx context.Context) (*kanban.Manager, error) {
	// Initialize registry
	reg := registry.NewComponentRegistry()
	if err := reg.Initialize(ctx, registry.Config{}); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to initialize registry").
			WithComponent("cli").WithOperation("initializeKanbanManager")
	}

	// Get database path - check current directory for .guild setup
	dbPath := local.LocalDatabasePath(".")

	// Initialize SQLite storage
	_, _, err := storage.InitializeSQLiteStorageForRegistry(ctx, dbPath)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to initialize SQLite storage").
			WithComponent("cli").WithOperation("initializeKanbanManager").
			WithDetails("db_path", dbPath)
	}

	// Create kanban manager using registry
	kanbanRegistry := &kanbanComponentRegistry{componentReg: reg}
	kanbanMgr, err := kanban.NewManagerWithRegistry(ctx, kanbanRegistry)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create kanban manager").
			WithComponent("cli").WithOperation("initializeKanbanManager")
	}

	return kanbanMgr, nil
}

// listKanbanBoards lists all available kanban boards
func listKanbanBoards(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Auto-start daemon unless --no-daemon flag is set
	if !kanbanNoDaemon {
		if !daemon.IsReachable(ctx) {
			fmt.Println("🚀 Starting Guild server...")
			if err := daemon.EnsureRunning(ctx); err != nil {
				return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to start Guild server").
					WithComponent("cli").
					WithOperation("kanban.list.daemon_start")
			}
			// Wait up to ~5s for the daemon to be reachable
			for i := 0; i < 10; i++ {
				if daemon.IsReachable(ctx) {
					break
				}
				time.Sleep(500 * time.Millisecond)
			}
		}
	}

	// Check if server is reachable
	if !daemon.IsReachable(ctx) {
		return gerror.New(gerror.ErrCodeConnection, "Guild server is not reachable", nil).
			WithComponent("cli").
			WithOperation("kanban.list").
			WithDetails("help", "Start the daemon first: 'guild serve --foreground', then run 'guild kanban list --no-daemon'")
	}

	// Initialize kanban manager with storage
	kanbanMgr, err := initializeKanbanManager(ctx)
	if err != nil {
		return err
	}

	// List boards
	boards, err := kanbanMgr.ListBoards(ctx)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to list boards").
			WithComponent("cli").WithOperation("listKanbanBoards")
	}

	if len(boards) == 0 {
		fmt.Println("No kanban boards found.")
		fmt.Println("Create one with: guild kanban create <name>")
		return nil
	}

	fmt.Printf("📋 Found %d kanban board(s):\n\n", len(boards))

	for _, board := range boards {
		// Get task counts for each status
		tasks, err := board.GetAllTasks(ctx)
		if err != nil {
			fmt.Printf("⚠️  %s (%s) - Error loading tasks: %v\n", board.Name, board.ID, err)
			continue
		}

		// Count by status
		statusCounts := make(map[kanban.TaskStatus]int)
		for _, task := range tasks {
			statusCounts[task.Status]++
		}

		total := len(tasks)
		todo := statusCounts[kanban.StatusTodo]
		inProgress := statusCounts[kanban.StatusInProgress]
		blocked := statusCounts[kanban.StatusBlocked]
		review := statusCounts[kanban.StatusReadyForReview]
		done := statusCounts[kanban.StatusDone]

		fmt.Printf("🏗️  %s (%s)\n", board.Name, board.ID)
		fmt.Printf("    %s\n", board.Description)
		fmt.Printf("    📊 Total: %d | TODO: %d | In Progress: %d | Blocked: %d | Review: %d | Done: %d\n",
			total, todo, inProgress, blocked, review, done)
		fmt.Printf("    🕒 Created: %s\n", board.CreatedAt.Format("2006-01-02 15:04"))
		fmt.Println()
	}

	fmt.Println("Use 'guild kanban view <board-id>' to open a board in the interactive UI.")
	return nil
}

// viewKanbanBoard opens the interactive kanban board UI
func viewKanbanBoard(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Auto-start daemon unless --no-daemon flag is set
	if !kanbanNoDaemon {
		if !daemon.IsReachable(ctx) {
			fmt.Println("🚀 Starting Guild server...")
			if err := daemon.EnsureRunning(ctx); err != nil {
				return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to start Guild server").
					WithComponent("cli").
					WithOperation("kanban.view.daemon_start")
			}
			// Wait up to ~5s for the daemon to be reachable
			for i := 0; i < 10; i++ {
				if daemon.IsReachable(ctx) {
					break
				}
				time.Sleep(500 * time.Millisecond)
			}
		}
	}

	// Check if server is reachable
	if !daemon.IsReachable(ctx) {
		return gerror.New(gerror.ErrCodeConnection, "Guild server is not reachable", nil).
			WithComponent("cli").
			WithOperation("kanban.view").
			WithDetails("help", "Start: 'guild serve --foreground', then 'guild kanban view --no-daemon' ")
	}

	// Determine board ID
	boardID := "main-board" // Default board
	if len(args) > 0 {
		boardID = args[0]
	}

	// Initialize kanban manager with storage
	kanbanMgr, err := initializeKanbanManager(ctx)
	if err != nil {
		return err
	}

	// Check if board exists, create it if it doesn't
	board, err := kanbanMgr.GetBoard(ctx, boardID)
	if err != nil {
		// Try to create the default board if it doesn't exist
		if boardID == "main-board" {
			fmt.Println("Creating main workshop board...")
			board, err = kanbanMgr.CreateBoard(ctx, "Main Workshop Board", "Central board for tracking all guild work")
			if err != nil {
				return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to create main board").
					WithComponent("cli").WithOperation("viewKanbanBoard")
			}
		} else {
			return gerror.Wrapf(err, gerror.ErrCodeNotFound, "board not found: %s", boardID).
				WithComponent("cli").WithOperation("viewKanbanBoard")
		}
	}

	fmt.Printf("🚀 Opening kanban board: %s\n", board.Name)
	fmt.Println("   Press ? for help, q to quit")

	// Try to connect to daemon for event streaming
	var model *kanbanui.Model
	conn, _, err := daemonconn.Discover(ctx)
	if err != nil {
		// No daemon connection, use basic model without events
		fmt.Println("   ⚠️  Running without event stream (daemon not available)")
		model = kanbanui.New(ctx, kanbanMgr, board.ID)
	} else {
		// Use model with event streaming
		fmt.Println("   🟢 Connected to event stream for real-time updates")
		model = kanbanui.NewWithEventClient(ctx, kanbanMgr, board.ID, conn)
	}

	p := tea.NewProgram(model, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to run kanban UI").
			WithComponent("cli").WithOperation("viewKanbanBoard")
	}

	// Clean up connection if exists
	if conn != nil {
		conn.Close()
	}

	return nil
}

// createKanbanBoard creates a new kanban board
func createKanbanBoard(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Auto-start daemon unless --no-daemon flag is set
	if !kanbanNoDaemon {
		if !daemon.IsReachable(ctx) {
			fmt.Println("🚀 Starting Guild server...")
			if err := daemon.EnsureRunning(ctx); err != nil {
				return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to start Guild server").
					WithComponent("cli").
					WithOperation("kanban.create.daemon_start")
			}
			for i := 0; i < 10; i++ {
				if daemon.IsReachable(ctx) {
					break
				}
				time.Sleep(500 * time.Millisecond)
			}
		}
	}

	// Check if server is reachable
	if !daemon.IsReachable(ctx) {
		return gerror.New(gerror.ErrCodeConnection, "Guild server is not reachable", nil).
			WithComponent("cli").
			WithOperation("kanban.create").
			WithDetails("help", "Start: 'guild serve --foreground', then 'guild kanban create --no-daemon' ")
	}

	name := args[0]
	description := "Kanban board for tracking tasks"
	if len(args) > 1 {
		description = args[1]
	}

	// Initialize kanban manager with storage
	kanbanMgr, err := initializeKanbanManager(ctx)
	if err != nil {
		return err
	}

	// Create the board
	board, err := kanbanMgr.CreateBoard(ctx, name, description)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to create board").
			WithComponent("cli").WithOperation("createKanbanBoard").
			WithDetails("board_name", name)
	}

	fmt.Printf("✅ Created kanban board: %s (%s)\n", board.Name, board.ID)
	fmt.Printf("   Description: %s\n", board.Description)
	fmt.Printf("   View it with: guild kanban view %s\n", board.ID)

	return nil
}

// Note: kanban registry adapters are defined in campaign.go
