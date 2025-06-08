package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	kanbanui "github.com/guild-ventures/guild-core/internal/ui/kanban"
	"github.com/guild-ventures/guild-core/pkg/gerror"
	"github.com/guild-ventures/guild-core/pkg/kanban"
	"github.com/guild-ventures/guild-core/pkg/registry"
	"github.com/guild-ventures/guild-core/pkg/storage"
)

// kanbanDemoCmd demonstrates the 5-column kanban board UI
var kanbanDemoCmd = &cobra.Command{
	Use:   "kanban-demo",
	Short: "Demonstrate the 5-column kanban board UI",
	Long: `Launches an interactive kanban board UI that demonstrates:
- 5-column layout (TODO, IN PROGRESS, BLOCKED, READY FOR REVIEW, DONE)
- Viewport-based rendering for handling 1000+ tasks
- Smooth scrolling and navigation
- Task search and filtering
- 60fps performance target

This demo creates a test board with sample tasks to showcase the UI.`,
	RunE: runKanbanDemo,
}

var (
	numTasks      int
	interactiveUI bool
	seedData      bool
)

func init() {
	kanbanDemoCmd.Flags().IntVar(&numTasks, "tasks", 100, "Number of sample tasks to generate")
	kanbanDemoCmd.Flags().BoolVar(&interactiveUI, "ui", true, "Launch interactive UI")
	kanbanDemoCmd.Flags().BoolVar(&seedData, "seed", true, "Seed with sample data")
}

func runKanbanDemo(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Initialize registry with SQLite storage
	reg := registry.NewComponentRegistry()
	if err := reg.Initialize(ctx, registry.Config{}); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to initialize registry").
			WithComponent("cli").WithOperation("runKanbanDemo")
	}

	// Create temporary SQLite database for demo
	tempDB := fmt.Sprintf("/tmp/guild-kanban-demo-%d.db", time.Now().Unix())

	// Initialize SQLite storage
	_, _, err := storage.InitializeSQLiteStorageForRegistry(ctx, tempDB)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to create store").
			WithComponent("cli").WithOperation("runKanbanDemo").
			WithDetails("temp_db", tempDB)
	}
	// No need to close - SQLite is managed by registry

	// Create kanban manager using registry
	kanbanRegistry := &kanbanComponentRegistry{componentReg: reg}
	kanbanMgr, err := kanban.NewManagerWithRegistry(ctx, kanbanRegistry)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create kanban manager").
			WithComponent("cli").WithOperation("runKanbanDemo")
	}

	// Create demo board
	boardID := "demo-board"
	boardName := "E-commerce Platform Development"
	boardDesc := "Demo board showcasing 5-column kanban with 1000+ tasks"

	board, err := kanbanMgr.CreateBoard(ctx, boardName, boardDesc)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to create board").
			WithComponent("cli").WithOperation("runKanbanDemo").
			WithDetails("board_name", boardName)
	}
	board.ID = boardID

	// Seed with sample tasks if requested
	if seedData {
		if err := seedDemoTasks(ctx, board, numTasks); err != nil {
			return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to seed tasks").
				WithComponent("cli").WithOperation("runKanbanDemo").
				WithDetails("num_tasks", fmt.Sprintf("%d", numTasks))
		}

		fmt.Printf("✅ Created demo board with %d tasks\n", numTasks)
	}

	// Launch UI if requested
	if interactiveUI {
		fmt.Println("🚀 Launching kanban board UI...")
		fmt.Println("   Press ? for help, q to quit")

		// Create and run the UI
		model := kanbanui.New(ctx, kanbanMgr, board.ID)
		p := tea.NewProgram(model, tea.WithAltScreen())

		if _, err := p.Run(); err != nil {
			return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to run UI").
				WithComponent("cli").WithOperation("runKanbanDemo")
		}
	} else {
		// Just display stats
		displayBoardStats(ctx, board)
	}

	return nil
}

// seedDemoTasks creates sample tasks for the demo
func seedDemoTasks(ctx context.Context, board *kanban.Board, count int) error {
	// Task prefixes and types
	prefixes := []string{"FE", "BE", "API", "DB", "TEST", "SEC", "DOCS", "INFRA"}
	titles := []string{
		"Implement user authentication",
		"Create product catalog",
		"Build shopping cart",
		"Setup payment gateway",
		"Design checkout flow",
		"Add search functionality",
		"Implement order tracking",
		"Create admin dashboard",
		"Setup CI/CD pipeline",
		"Write API documentation",
		"Add unit tests",
		"Perform security audit",
		"Optimize database queries",
		"Implement caching layer",
		"Add monitoring alerts",
	}

	agents := []string{
		"frontend-dev",
		"backend-dev",
		"fullstack-dev",
		"devops-eng",
		"qa-engineer",
		"security-expert",
		"db-admin",
		"tech-lead",
	}

	// Distribution of tasks across columns
	statusDistribution := map[kanban.TaskStatus]float64{
		kanban.StatusTodo:           0.40, // 40% in TODO
		kanban.StatusInProgress:     0.15, // 15% in progress
		kanban.StatusBlocked:        0.05, // 5% blocked
		kanban.StatusReadyForReview: 0.10, // 10% ready for review
		kanban.StatusDone:           0.30, // 30% done
	}

	// Create tasks
	for i := 0; i < count; i++ {
		prefix := prefixes[rand.Intn(len(prefixes))]
		taskID := fmt.Sprintf("%s-%03d", prefix, i+1)
		title := titles[rand.Intn(len(titles))]

		task := kanban.NewTask(
			fmt.Sprintf("%s: %s", taskID, title),
			fmt.Sprintf("Implementation details for %s", title),
		)

		task.ID = taskID

		// Assign status based on distribution
		r := rand.Float64()
		cumulative := 0.0
		for status, prob := range statusDistribution {
			cumulative += prob
			if r <= cumulative {
				task.Status = status
				break
			}
		}

		// Set priority
		priorityRand := rand.Float64()
		if priorityRand < 0.2 {
			task.Priority = kanban.PriorityHigh
		} else if priorityRand < 0.6 {
			task.Priority = kanban.PriorityMedium
		} else {
			task.Priority = kanban.PriorityLow
		}

		// Assign to agent
		if task.Status != kanban.StatusTodo && rand.Float64() < 0.8 {
			task.AssignedTo = agents[rand.Intn(len(agents))]
		}

		// Set progress for in-progress tasks
		if task.Status == kanban.StatusInProgress {
			task.Progress = rand.Intn(90) + 10 // 10-99%
		} else if task.Status == kanban.StatusDone {
			task.Progress = 100
		}

		// Add some blocked reasons
		if task.Status == kanban.StatusBlocked {
			blockers := []string{
				"Waiting for API specs",
				"Need AWS credentials",
				"Database migration pending",
				"Security review required",
				"Dependency not ready",
			}
			blockerReason := blockers[rand.Intn(len(blockers))]
			task.AddBlocker(fmt.Sprintf("BLOCK-%d", rand.Intn(100)), "system", blockerReason)
			task.Metadata["blocker_reason"] = blockerReason
		}

		// Set created time to simulate age
		task.CreatedAt = time.Now().Add(-time.Duration(rand.Intn(30*24)) * time.Hour) // Up to 30 days old

		// Create the task first
		createdTask, err := board.CreateTask(ctx, title, fmt.Sprintf("Implementation details for %s", title))
		if err != nil {
			return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to create task").
				WithComponent("cli").WithOperation("seedDemoTasks").
				WithDetails("task_id", taskID)
		}

		// Update with our generated properties
		createdTask.Status = task.Status
		createdTask.Priority = task.Priority
		createdTask.AssignedTo = task.AssignedTo
		createdTask.Progress = task.Progress
		createdTask.CreatedAt = task.CreatedAt
		createdTask.Blockers = task.Blockers
		// Preserve board_id in metadata
		if createdTask.Metadata == nil {
			createdTask.Metadata = make(map[string]string)
		}
		for k, v := range task.Metadata {
			createdTask.Metadata[k] = v
		}

		// Save the updated task
		if err := board.UpdateTask(ctx, createdTask); err != nil {
			return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to update task").
				WithComponent("cli").WithOperation("seedDemoTasks").
				WithDetails("task_id", taskID)
		}
	}

	return nil
}

// displayBoardStats shows statistics about the board
func displayBoardStats(ctx context.Context, board *kanban.Board) {
	fmt.Printf("\n📊 Board Statistics\n")
	fmt.Printf("══════════════════\n")

	statuses := []kanban.TaskStatus{
		kanban.StatusTodo,
		kanban.StatusInProgress,
		kanban.StatusBlocked,
		kanban.StatusReadyForReview,
		kanban.StatusDone,
	}

	total := 0
	for _, status := range statuses {
		tasks, err := board.GetTasksByStatus(ctx, status)
		if err != nil {
			log.Printf("Error getting tasks for status %s: %v", status, err)
			continue
		}

		count := len(tasks)
		total += count

		// Calculate percentage
		percentage := float64(count) / float64(numTasks) * 100

		// Create a simple bar chart
		barLength := int(percentage / 2) // Scale to fit
		bar := ""
		for i := 0; i < barLength; i++ {
			bar += "█"
		}

		fmt.Printf("%-20s: %4d tasks (%5.1f%%) %s\n", status, count, percentage, bar)
	}

	fmt.Printf("%-20s: %4d tasks\n", "TOTAL", total)
	fmt.Printf("\n✨ Demo board created successfully!\n")
}
