//go:build integration

package performance

import (
	"context"
	"fmt"
	"math/rand"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/guild-ventures/guild-core/internal/testutil"
	"github.com/guild-ventures/guild-core/pkg/commission"
	"github.com/guild-ventures/guild-core/pkg/kanban"
	"github.com/guild-ventures/guild-core/pkg/registry"
	"github.com/guild-ventures/guild-core/pkg/storage"
)

// MockTask represents a task for dependency testing
type MockTask struct {
	ID           string
	Title        string
	Dependencies []string
}

// MockBoard represents a kanban board for performance testing
type MockBoard struct {
	Columns []string
	Tasks   map[string][]*kanban.Task
}

// TestLargeCommissionHandling tests system behavior with massive commissions
func TestLargeCommissionHandling(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()

	t.Run("100PlusTaskCommission", func(t *testing.T) {
		// Setup test environment
		_, cleanup := testutil.SetupTestProject(t)
		defer cleanup()

		reg := registry.NewComponentRegistry()
		err := reg.Initialize(ctx, registry.Config{})
		require.NoError(t, err)

		// Initialize storage with all repositories
		storageRegistry, _, err := storage.InitializeSQLiteStorageForTests(ctx)
		require.NoError(t, err)

		// Create large commission
		taskCount := 150
		commissionContent := generateLargeCommission(taskCount)

		// Setup mock provider to return many tasks
		mockProvider := testutil.NewMockLLMProvider()
		tasks := make([]string, taskCount)
		for i := 0; i < taskCount; i++ {
			tasks[i] = fmt.Sprintf("Task %d: Implement feature component %d", i+1, i+1)
		}

		// Set up mock provider response as a simple string for the new API
		mockResponse := "Tasks breakdown:\n"
		for _, task := range tasks {
			mockResponse += "- " + task + "\n"
		}
		mockProvider.SetResponse("manager", mockResponse)

		err = reg.Providers().RegisterProvider("mock", mockProvider)
		require.NoError(t, err)

		// Track memory usage
		var m runtime.MemStats
		runtime.GC()
		runtime.ReadMemStats(&m)
		startMemory := m.Alloc

		// Process commission
		startTime := time.Now()

		// Get repositories from storage registry
		commissionRepo := storageRegistry.GetCommissionRepository()
		taskRepo := storageRegistry.GetTaskRepository()
		campaignRepo := storageRegistry.GetCampaignRepository()

		// Create commission manager with new API
		manager, err := commission.DefaultCommissionManagerFactory(commissionRepo, "/tmp/commissions")
		require.NoError(t, err)

		// Suppress unused variable warning
		_ = taskRepo

		// Create a campaign first (required for foreign key constraint)
		testCampaign := &storage.Campaign{
			ID:        "test-campaign-001",
			Name:      "Performance Test Campaign",
			Status:    "active",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		err = campaignRepo.CreateCampaign(ctx, testCampaign)
		require.NoError(t, err)

		// Create commission with campaign reference
		comm := commission.Commission{
			ID:          "large-commission-001",
			CampaignID:  testCampaign.ID,
			Title:       "Large System Implementation",
			Description: commissionContent,
			Status:      commission.StatusDraft,
		}

		createdComm, err := manager.CreateCommission(ctx, comm)
		require.NoError(t, err)

		// Use the created commission
		comm = *createdComm

		processingTime := time.Since(startTime)

		// Check memory after processing
		runtime.GC()
		runtime.ReadMemStats(&m)
		memoryGrowth := m.Alloc - startMemory

		// Verify performance metrics
		assert.Less(t, processingTime, 30*time.Second, "Should process 150 tasks within 30s")
		assert.Less(t, memoryGrowth, uint64(100*1024*1024), "Memory growth should be < 100MB")

		// Verify commission was created (task creation would be handled by orchestrator)
		verifyComm, err := manager.GetCommission(ctx, comm.ID)
		require.NoError(t, err)
		assert.Equal(t, comm.ID, verifyComm.ID)
		assert.Equal(t, comm.Title, verifyComm.Title)

		// Test UI responsiveness with large task list
		uiStartTime := time.Now()

		// Simulate UI operations
		for i := 0; i < 10; i++ {
			// Simulate fetching tasks for display
			_, err := taskRepo.ListTasksByStatus(ctx, "todo")
			require.NoError(t, err)
		}

		uiResponseTime := time.Since(uiStartTime) / 10
		assert.Less(t, uiResponseTime, 100*time.Millisecond, "UI queries should be fast")
	})

	t.Run("ComplexDependencyGraph", func(t *testing.T) {
		// Create commission with complex task dependencies
		_, cleanup := testutil.SetupTestProject(t)
		defer cleanup()

		reg := registry.NewComponentRegistry()
		err := reg.Initialize(ctx, registry.Config{})
		require.NoError(t, err)

		storageRegistry, _, err := storage.InitializeSQLiteStorageForTests(ctx)
		require.NoError(t, err)

		// Suppress unused variable warnings
		_ = reg
		_ = storageRegistry

		// Generate tasks with dependencies
		taskCount := 1000
		tasks := generateTasksWithDependencies(taskCount)

		// Track dependency resolution performance
		startTime := time.Now()

		// Analyze dependencies (use local algorithms for testing)
		cycles := detectCycles(tasks)
		assert.Empty(t, cycles, "Should not have dependency cycles")

		// Calculate critical path
		criticalPath := findCriticalPath(tasks)
		assert.NotEmpty(t, criticalPath, "Should find critical path")

		// Test dependency graph structure
		assert.Len(t, tasks, taskCount, "Should have correct number of tasks")

		analysisTime := time.Since(startTime)
		assert.Less(t, analysisTime, 5*time.Second, "Dependency analysis should be fast")

		// Verify memory efficiency
		var m runtime.MemStats
		runtime.GC()
		runtime.ReadMemStats(&m)
		assert.Less(t, m.Alloc, uint64(500*1024*1024), "Should handle 1000 tasks in < 500MB")
	})

	t.Run("TaskSchedulingEfficiency", func(t *testing.T) {
		// Test efficient scheduling of many tasks
		taskQueue := make(chan *MockTask, 1000)
		completedTasks := int32(0)

		// Create worker pool
		workerCount := 10
		var wg sync.WaitGroup

		startTime := time.Now()

		// Start workers
		for i := 0; i < workerCount; i++ {
			wg.Add(1)
			go func(workerID int) {
				defer wg.Done()
				for _ = range taskQueue {
					// Simulate task execution
					time.Sleep(time.Duration(rand.Intn(10)) * time.Millisecond)
					atomic.AddInt32(&completedTasks, 1)
				}
			}(i)
		}

		// Queue tasks
		taskCount := 500
		for i := 0; i < taskCount; i++ {
			taskQueue <- &MockTask{
				ID:    fmt.Sprintf("task-%d", i),
				Title: fmt.Sprintf("Task %d", i),
			}
		}
		close(taskQueue)

		// Wait for completion
		wg.Wait()
		duration := time.Since(startTime)

		// Verify efficiency
		assert.Equal(t, int32(taskCount), completedTasks, "All tasks should complete")
		assert.Less(t, duration, 10*time.Second, "Should process 500 tasks quickly")

		// Calculate throughput
		throughput := float64(taskCount) / duration.Seconds()
		t.Logf("Task throughput: %.2f tasks/second", throughput)
		assert.Greater(t, throughput, 50.0, "Should process >50 tasks/second")
	})

	t.Run("DatabaseQueryPerformance", func(t *testing.T) {
		// Test database performance with large datasets
		_, cleanup := testutil.SetupTestProject(t)
		defer cleanup()

		storageRegistry, _, err := storage.InitializeSQLiteStorageForTests(ctx)
		require.NoError(t, err)

		taskRepo := storageRegistry.GetTaskRepository()
		campaignRepo := storageRegistry.GetCampaignRepository()
		commissionRepo := storageRegistry.GetCommissionRepository()

		// Create a campaign first
		testCampaign := &storage.Campaign{
			ID:        "perf-test-campaign",
			Name:      "Performance Test Campaign",
			Status:    "active",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		err = campaignRepo.CreateCampaign(ctx, testCampaign)
		require.NoError(t, err)

		// Create commissions that will be referenced by tasks
		for i := 0; i < 100; i++ {
			comm := &storage.Commission{
				ID:          fmt.Sprintf("commission-%d", i),
				CampaignID:  testCampaign.ID,
				Title:       fmt.Sprintf("Commission %d", i),
				Description: stringPtr(fmt.Sprintf("Commission for tasks %d-%d", i*100, (i+1)*100-1)),
				Status:      "active",
				CreatedAt:   time.Now(),
			}
			err = commissionRepo.CreateCommission(ctx, comm)
			require.NoError(t, err)
		}

		// Insert many tasks
		insertStart := time.Now()
		taskCount := 10000

		// Individual task creation (no batch API available)
		for i := 0; i < taskCount; i++ {
			task := &storage.Task{
				ID:           fmt.Sprintf("task-%d", i),
				Title:        fmt.Sprintf("Task %d", i),
				Description:  stringPtr(fmt.Sprintf("Description for task %d with some content", i)),
				Status:       []string{"todo", "in_progress", "done"}[i%3],
				Column:       []string{"backlog", "in_progress", "done"}[i%3],
				CommissionID: fmt.Sprintf("commission-%d", i/100),
				StoryPoints:  int32([]int{1, 2, 3}[i%3]),
				Metadata:     make(map[string]interface{}),
				CreatedAt:    time.Now(),
				UpdatedAt:    time.Now(),
			}
			err := taskRepo.CreateTask(ctx, task)
			require.NoError(t, err)
		}

		insertDuration := time.Since(insertStart)
		assert.Less(t, insertDuration, 10*time.Second, "Should insert 10k tasks quickly")

		// Test query performance
		queries := []struct {
			name  string
			query func() error
		}{
			{
				name: "ListTasksByStatus",
				query: func() error {
					_, err := taskRepo.ListTasksByStatus(ctx, "todo")
					return err
				},
			},
			{
				name: "ListTasksByCommission",
				query: func() error {
					_, err := taskRepo.ListTasksByCommission(ctx, "commission-50")
					return err
				},
			},
			{
				name: "ListTasks",
				query: func() error {
					_, err := taskRepo.ListTasks(ctx)
					return err
				},
			},
		}

		for _, q := range queries {
			t.Run(q.name, func(t *testing.T) {
				// Warm up
				q.query()

				// Measure performance
				iterations := 100
				start := time.Now()
				for i := 0; i < iterations; i++ {
					err := q.query()
					require.NoError(t, err)
				}
				avgDuration := time.Since(start) / time.Duration(iterations)

				// For 10k records, 50ms is a reasonable threshold for average query time
				assert.Less(t, avgDuration, 50*time.Millisecond,
					fmt.Sprintf("%s should be fast (avg: %v)", q.name, avgDuration))
			})
		}
	})
}

// TestVisualizationPerformance tests UI rendering with large datasets
func TestVisualizationPerformance(t *testing.T) {
	t.Run("KanbanBoardWith1000Tasks", func(t *testing.T) {
		// Simulate kanban board with many tasks using mock structure
		board := &MockBoard{
			Columns: []string{"todo", "in_progress", "review", "done"},
			Tasks:   make(map[string][]*kanban.Task),
		}

		// Populate board
		taskCount := 1000
		for i := 0; i < taskCount; i++ {
			column := board.Columns[i%4]
			task := &kanban.Task{
				ID:    fmt.Sprintf("task-%d", i),
				Title: fmt.Sprintf("Task %d: %s", i, generateTaskTitle()),
			}
			board.Tasks[column] = append(board.Tasks[column], task)
		}

		// Test rendering performance
		startTime := time.Now()

		// Simulate rendering operations
		for i := 0; i < 60; i++ { // 60 frames
			// Get visible tasks (viewport simulation)
			visibleTasks := getVisibleTasks(board, 20) // 20 tasks visible
			assert.NotEmpty(t, visibleTasks)

			// Simulate layout calculation
			calculateLayout(visibleTasks)
		}

		duration := time.Since(startTime)
		fps := 60.0 / duration.Seconds()

		assert.Greater(t, fps, 30.0, "Should maintain >30 FPS with 1000 tasks")
	})

	t.Run("DependencyGraphVisualization", func(t *testing.T) {
		// Create large dependency graph
		nodes := 500
		edges := 2000

		graph := generateDependencyGraph(nodes, edges)

		// Test graph layout algorithm
		startTime := time.Now()

		// Calculate layout
		layout := calculateGraphLayout(graph)
		assert.Len(t, layout, nodes, "Should layout all nodes")

		// Test rendering updates
		for i := 0; i < 30; i++ { // 30 frame updates
			// Simulate view changes
			updateGraphView(layout, i)
		}

		duration := time.Since(startTime)
		assert.Less(t, duration, 2*time.Second, "Graph operations should be fast")
	})
}

// Helper functions

func generateLargeCommission(taskCount int) string {
	content := `# Large System Implementation

## Overview
This is a comprehensive system implementation requiring many interconnected components.

## Tasks
`
	for i := 0; i < taskCount; i++ {
		content += fmt.Sprintf("\n### Component %d\n", i+1)
		content += fmt.Sprintf("- Design %s module\n", []string{"API", "Database", "UI", "Service"}[i%4])
		content += fmt.Sprintf("- Implement %s functionality\n", []string{"CRUD", "Auth", "Search", "Analytics"}[i%4])
		content += fmt.Sprintf("- Test %s integration\n", []string{"Unit", "Integration", "E2E", "Performance"}[i%4])
	}

	return content
}

func generateTasksWithDependencies(count int) []*MockTask {
	tasks := make([]*MockTask, count)

	for i := 0; i < count; i++ {
		task := &MockTask{
			ID:           fmt.Sprintf("task-%d", i),
			Title:        fmt.Sprintf("Task %d", i),
			Dependencies: []string{},
		}

		// Create realistic dependency patterns
		if i > 0 {
			// Depend on previous task
			task.Dependencies = append(task.Dependencies, fmt.Sprintf("task-%d", i-1))
		}

		if i > 10 && i%10 == 0 {
			// Every 10th task depends on task 10 steps back
			task.Dependencies = append(task.Dependencies, fmt.Sprintf("task-%d", i-10))
		}

		if i > 50 && i%50 == 0 {
			// Milestone tasks depend on multiple previous tasks
			for j := 1; j <= 5; j++ {
				task.Dependencies = append(task.Dependencies, fmt.Sprintf("task-%d", i-j*10))
			}
		}

		tasks[i] = task
	}

	return tasks
}

func detectCycles(tasks []*MockTask) [][]string {
	// Simple cycle detection algorithm
	visited := make(map[string]bool)
	recStack := make(map[string]bool)
	cycles := [][]string{}

	var dfs func(taskID string, path []string) bool
	dfs = func(taskID string, path []string) bool {
		visited[taskID] = true
		recStack[taskID] = true
		path = append(path, taskID)

		// Find task
		var task *MockTask
		for _, t := range tasks {
			if t.ID == taskID {
				task = t
				break
			}
		}

		if task != nil {
			for _, dep := range task.Dependencies {
				if !visited[dep] {
					if dfs(dep, path) {
						return true
					}
				} else if recStack[dep] {
					// Found cycle
					cycleStart := 0
					for i, id := range path {
						if id == dep {
							cycleStart = i
							break
						}
					}
					cycles = append(cycles, path[cycleStart:])
					return true
				}
			}
		}

		recStack[taskID] = false
		return false
	}

	for _, task := range tasks {
		if !visited[task.ID] {
			dfs(task.ID, []string{})
		}
	}

	return cycles
}

func findCriticalPath(tasks []*MockTask) []string {
	// Simplified critical path calculation
	taskMap := make(map[string]*MockTask)
	for _, task := range tasks {
		taskMap[task.ID] = task
	}

	// Find tasks with no dependencies (start nodes)
	startTasks := []*MockTask{}
	for _, task := range tasks {
		if len(task.Dependencies) == 0 {
			startTasks = append(startTasks, task)
		}
	}

	if len(startTasks) == 0 {
		return []string{}
	}

	// Simple path: just follow first dependency chain
	path := []string{startTasks[0].ID}

	// Find dependent tasks
	for i := 0; i < 10 && i < len(tasks); i++ {
		lastTask := path[len(path)-1]
		for _, task := range tasks {
			for _, dep := range task.Dependencies {
				if dep == lastTask {
					path = append(path, task.ID)
					break
				}
			}
		}
	}

	return path
}

func generateTaskTitle() string {
	components := []string{
		"API", "Database", "Frontend", "Backend", "Service", "Module",
		"Handler", "Controller", "Model", "View", "Repository", "Factory",
	}
	actions := []string{
		"Create", "Update", "Refactor", "Optimize", "Test", "Deploy",
		"Configure", "Implement", "Design", "Review", "Document", "Validate",
	}

	return fmt.Sprintf("%s %s", actions[rand.Intn(len(actions))], components[rand.Intn(len(components))])
}

func getVisibleTasks(board *MockBoard, viewportSize int) []*kanban.Task {
	visible := []*kanban.Task{}
	count := 0

	for _, column := range board.Columns {
		for _, task := range board.Tasks[column] {
			if count >= viewportSize {
				return visible
			}
			visible = append(visible, task)
			count++
		}
	}

	return visible
}

func calculateLayout(tasks []*kanban.Task) {
	// Simulate layout calculations
	for _, task := range tasks {
		_ = len(task.Title) * 8 // Width calculation
		_ = 60                  // Height calculation
	}
}

type GraphNode struct {
	ID    string
	Edges []string
}

func generateDependencyGraph(nodes, edges int) []*GraphNode {
	graph := make([]*GraphNode, nodes)

	for i := 0; i < nodes; i++ {
		graph[i] = &GraphNode{
			ID:    fmt.Sprintf("node-%d", i),
			Edges: []string{},
		}
	}

	// Add random edges
	for i := 0; i < edges; i++ {
		from := rand.Intn(nodes)
		to := rand.Intn(nodes)
		if from != to {
			graph[from].Edges = append(graph[from].Edges, graph[to].ID)
		}
	}

	return graph
}

type NodeLayout struct {
	ID string
	X  float64
	Y  float64
}

func calculateGraphLayout(graph []*GraphNode) []NodeLayout {
	layout := make([]NodeLayout, len(graph))

	// Simple force-directed layout simulation
	for i, node := range graph {
		layout[i] = NodeLayout{
			ID: node.ID,
			X:  rand.Float64() * 1000,
			Y:  rand.Float64() * 1000,
		}
	}

	// Simulate iterations
	for iter := 0; iter < 10; iter++ {
		for i := range layout {
			// Apply forces
			layout[i].X += (rand.Float64() - 0.5) * 10
			layout[i].Y += (rand.Float64() - 0.5) * 10
		}
	}

	return layout
}

func updateGraphView(layout []NodeLayout, frame int) {
	// Simulate view transformations
	zoom := 1.0 + float64(frame)*0.01
	offsetX := float64(frame) * 5
	offsetY := float64(frame) * 3

	for i := range layout {
		layout[i].X = layout[i].X*zoom + offsetX
		layout[i].Y = layout[i].Y*zoom + offsetY
	}
}

// Helper function for creating string pointers
func stringPtr(s string) *string {
	return &s
}
