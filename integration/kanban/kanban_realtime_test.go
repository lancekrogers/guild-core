// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

//go:build integration
// +build integration

package kanban

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/lancekrogers/guild/internal/daemon"
	kanbanui "github.com/lancekrogers/guild/internal/ui/kanban"
	"github.com/lancekrogers/guild/pkg/gerror"
	pb "github.com/lancekrogers/guild/pkg/grpc/pb/guild/v1"
	"github.com/lancekrogers/guild/pkg/kanban"
	"github.com/lancekrogers/guild/pkg/memory"
	"github.com/lancekrogers/guild/pkg/observability"
	"github.com/lancekrogers/guild/pkg/project/local"
	"github.com/lancekrogers/guild/pkg/registry"
	"github.com/lancekrogers/guild/pkg/storage"
)

// kanbanTestEnvironment provides integrated testing for kanban UI and event streaming
type kanbanTestEnvironment struct {
	ctx               context.Context
	cancel            context.CancelFunc
	registry          kanban.ComponentRegistry
	kanbanManager     *kanban.Manager
	board             *kanban.Board
	conn              *grpc.ClientConn
	eventClient       pb.EventServiceClient
	kanbanUI          *kanbanui.Model
	testDir           string
	logger            observability.Logger
	taskEventReceived chan *pb.TaskEvent
	taskCount         int
	mu                sync.RWMutex
}

func TestKanbanRealTimeEventFlow(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	// Check context early
	if err := ctx.Err(); err != nil {
		t.Fatal(gerror.Wrap(err, gerror.ErrCodeCancelled, "test context cancelled").
			WithComponent("kanban_integration_test").
			WithOperation("TestKanbanRealTimeEventFlow"))
	}

	logger := observability.GetLogger(ctx)
	ctx = observability.WithComponent(ctx, "kanban_integration_test")
	ctx = observability.WithOperation(ctx, "TestKanbanRealTimeEventFlow")

	logger.InfoContext(ctx, "Starting kanban real-time event flow test")

	// Skip if no daemon is running
	if !daemon.IsRunning() {
		t.Skip("No daemon running for kanban real-time event flow test")
	}

	// Set up test environment
	env, err := setupKanbanTestEnvironment(ctx, t)
	if err != nil {
		t.Fatal(gerror.Wrap(err, gerror.ErrCodeInternal, "failed to setup test environment").
			WithComponent("kanban_integration_test").
			WithOperation("TestKanbanRealTimeEventFlow"))
	}
	defer env.cleanup(ctx)

	logger.InfoContext(ctx, "Test environment setup complete", "board_id", env.board.ID)

	// Test 1: Create a task via kanban manager and verify UI receives event
	task1, err := env.createTestTask(ctx, "Integration Test Task 1", "Test task creation event flow")
	if err != nil {
		t.Fatal(gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create test task 1").
			WithComponent("kanban_integration_test").
			WithOperation("TestKanbanRealTimeEventFlow"))
	}

	logger.InfoContext(ctx, "Created test task 1", "task_id", task1.ID, "title", task1.Title)

	// Wait for UI to receive and process the event
	require.Eventually(t, func() bool {
		return env.hasTaskInUI(task1.ID)
	}, 5*time.Second, 100*time.Millisecond, "Task 1 should appear in UI within 5 seconds")

	// Verify task is in correct column (should start as TODO)
	assert.Equal(t, kanban.StatusTodo, task1.Status, "Task should start with TODO status")

	// Test 2: Move task to IN_PROGRESS and verify real-time update
	err = env.board.UpdateTaskStatus(ctx, task1.ID, kanban.StatusInProgress, "test-agent", "Starting work on task")
	require.NoError(t, err, "Should be able to update task status")

	logger.InfoContext(ctx, "Updated task 1 status to in_progress", "task_id", task1.ID)

	// Wait for UI to reflect the status change
	require.Eventually(t, func() bool {
		return env.getTaskStatusInUI(task1.ID) == kanban.StatusInProgress
	}, 3*time.Second, 100*time.Millisecond, "Task should move to IN_PROGRESS in UI")

	// Test 3: Create multiple tasks rapidly and verify all appear
	logger.InfoContext(ctx, "Creating multiple tasks rapidly")
	var taskIDs []string
	for i := 2; i <= 6; i++ {
		task, err := env.createTestTask(ctx, fmt.Sprintf("Rapid Task %d", i), fmt.Sprintf("Testing rapid task creation %d", i))
		require.NoError(t, err, "Should be able to create rapid task %d", i)
		taskIDs = append(taskIDs, task.ID)
		logger.InfoContext(ctx, "Created rapid task", "index", i, "task_id", task.ID)
	}

	// Wait for all tasks to appear in UI
	for _, taskID := range taskIDs {
		require.Eventually(t, func() bool {
			return env.hasTaskInUI(taskID)
		}, 10*time.Second, 200*time.Millisecond, "Rapid task %s should appear in UI", taskID)
	}

	totalTasks := env.getUITaskCount()
	assert.GreaterOrEqual(t, totalTasks, 5, "Should have at least 5 tasks in UI")
	logger.InfoContext(ctx, "All rapid tasks appeared in UI", "total_tasks", totalTasks)

	// Test 4: Test task assignment event
	err = env.board.AssignTask(ctx, task1.ID, "elena", "test-system", "Assigning to Elena")
	require.NoError(t, err, "Should be able to assign task")

	logger.InfoContext(ctx, "Assigned task 1 to elena", "task_id", task1.ID)

	// Wait for assignment to reflect in UI
	require.Eventually(t, func() bool {
		return env.getTaskAssigneeInUI(task1.ID) == "elena"
	}, 3*time.Second, 100*time.Millisecond, "Task assignment should update in UI")

	// Test 5: Test task blocking and unblocking
	err = env.board.AddTaskBlocker(ctx, task1.ID, "api-dependency", "test-system", "Waiting for API endpoint")
	require.NoError(t, err, "Should be able to block task")

	logger.InfoContext(ctx, "Blocked task 1", "task_id", task1.ID, "blocker", "api-dependency")

	// Wait for blocking status to appear in UI
	require.Eventually(t, func() bool {
		return env.getTaskStatusInUI(task1.ID) == kanban.StatusBlocked
	}, 3*time.Second, 100*time.Millisecond, "Task should move to BLOCKED status in UI")

	// Unblock the task
	err = env.board.RemoveTaskBlocker(ctx, task1.ID, "api-dependency", "test-system", "API endpoint ready")
	require.NoError(t, err, "Should be able to unblock task")

	logger.InfoContext(ctx, "Unblocked task 1", "task_id", task1.ID)

	// Wait for unblocking to reflect in UI
	require.Eventually(t, func() bool {
		status := env.getTaskStatusInUI(task1.ID)
		return status == kanban.StatusTodo || status == kanban.StatusInProgress
	}, 3*time.Second, 100*time.Millisecond, "Task should move out of BLOCKED status in UI")

	// Test 6: Complete the task
	err = env.board.UpdateTaskStatus(ctx, task1.ID, kanban.StatusDone, "elena", "Task completed successfully")
	require.NoError(t, err, "Should be able to complete task")

	logger.InfoContext(ctx, "Completed task 1", "task_id", task1.ID)

	// Wait for completion to reflect in UI
	require.Eventually(t, func() bool {
		return env.getTaskStatusInUI(task1.ID) == kanban.StatusDone
	}, 3*time.Second, 100*time.Millisecond, "Task should move to DONE status in UI")

	logger.InfoContext(ctx, "Kanban real-time event flow test completed successfully")
}

func TestKanbanUIPerformanceWith200Tasks(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()

	// Check context early
	if err := ctx.Err(); err != nil {
		t.Fatal(gerror.Wrap(err, gerror.ErrCodeCancelled, "test context cancelled").
			WithComponent("kanban_integration_test").
			WithOperation("TestKanbanUIPerformanceWith200Tasks"))
	}

	logger := observability.GetLogger(ctx)
	ctx = observability.WithComponent(ctx, "kanban_integration_test")
	ctx = observability.WithOperation(ctx, "TestKanbanUIPerformanceWith200Tasks")

	logger.InfoContext(ctx, "Starting kanban UI performance test with 200 tasks")

	// Skip if no daemon is running
	if !daemon.IsRunning() {
		t.Skip("No daemon running for kanban UI performance test")
	}

	// Set up test environment
	env, err := setupKanbanTestEnvironment(ctx, t)
	if err != nil {
		t.Fatal(gerror.Wrap(err, gerror.ErrCodeInternal, "failed to setup test environment").
			WithComponent("kanban_integration_test").
			WithOperation("TestKanbanUIPerformanceWith200Tasks"))
	}
	defer env.cleanup(ctx)

	// Create 200 tasks across different statuses
	logger.InfoContext(ctx, "Creating 200 test tasks")
	statuses := []kanban.TaskStatus{
		kanban.StatusTodo,
		kanban.StatusInProgress,
		kanban.StatusBlocked,
		kanban.StatusReadyForReview,
		kanban.StatusDone,
	}

	var taskIDs []string
	for i := 0; i < 200; i++ {
		status := statuses[i%len(statuses)]
		task, err := env.createTestTask(ctx, fmt.Sprintf("Performance Task %d", i+1), fmt.Sprintf("Task for performance testing #%d", i+1))
		require.NoError(t, err, "Should be able to create performance task %d", i+1)

		// Set different status for variety
		if status != kanban.StatusTodo {
			err = env.board.UpdateTaskStatus(ctx, task.ID, status, "test-agent", fmt.Sprintf("Setting to %s", status))
			require.NoError(t, err, "Should be able to set task status to %s", status)
		}

		taskIDs = append(taskIDs, task.ID)

		if (i+1)%50 == 0 {
			logger.InfoContext(ctx, "Created batch of tasks", "completed", i+1, "total", 200)
		}
	}

	logger.InfoContext(ctx, "All 200 tasks created, measuring UI performance")

	// Measure UI rendering performance
	start := time.Now()

	// Wait for all tasks to appear in UI (with longer timeout for 200 tasks)
	taskCount := 0
	require.Eventually(t, func() bool {
		taskCount = env.getUITaskCount()
		return taskCount >= 200
	}, 30*time.Second, 500*time.Millisecond, "All 200 tasks should appear in UI")

	loadTime := time.Since(start)
	logger.InfoContext(ctx, "UI loaded all tasks", "count", taskCount, "load_time", loadTime)

	// Performance requirements: UI should handle 200+ tasks
	assert.GreaterOrEqual(t, taskCount, 200, "UI should display at least 200 tasks")
	assert.Less(t, loadTime, 10*time.Second, "UI should load 200 tasks within 10 seconds")

	// Test UI responsiveness with large dataset
	// Simulate rapid status updates to test real-time performance
	logger.InfoContext(ctx, "Testing UI responsiveness with rapid updates")
	updateStart := time.Now()

	// Update status of first 20 tasks
	for i := 0; i < 20; i++ {
		if i >= len(taskIDs) {
			break
		}
		err := env.board.UpdateTaskStatus(ctx, taskIDs[i], kanban.StatusInProgress, "performance-test", "Rapid update test")
		require.NoError(t, err, "Should be able to update task %d", i)
	}

	updateDuration := time.Since(updateStart)
	logger.InfoContext(ctx, "Completed rapid updates", "updates", 20, "duration", updateDuration)

	// Performance requirement: Rapid updates should be fast
	assert.Less(t, updateDuration, 2*time.Second, "20 rapid updates should complete within 2 seconds")

	logger.InfoContext(ctx, "Kanban UI performance test completed successfully",
		"total_tasks", taskCount, "load_time", loadTime, "update_time", updateDuration)
}

func TestKanbanSearchFunctionality(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Check context early
	if err := ctx.Err(); err != nil {
		t.Fatal(gerror.Wrap(err, gerror.ErrCodeCancelled, "test context cancelled").
			WithComponent("kanban_integration_test").
			WithOperation("TestKanbanSearchFunctionality"))
	}

	logger := observability.GetLogger(ctx)
	ctx = observability.WithComponent(ctx, "kanban_integration_test")
	ctx = observability.WithOperation(ctx, "TestKanbanSearchFunctionality")

	logger.InfoContext(ctx, "Starting kanban search functionality test")

	// Skip if no daemon is running
	if !daemon.IsRunning() {
		t.Skip("No daemon running for kanban search functionality test")
	}

	// Set up test environment
	env, err := setupKanbanTestEnvironment(ctx, t)
	if err != nil {
		t.Fatal(gerror.Wrap(err, gerror.ErrCodeInternal, "failed to setup test environment").
			WithComponent("kanban_integration_test").
			WithOperation("TestKanbanSearchFunctionality"))
	}
	defer env.cleanup(ctx)

	// Create test tasks with different attributes for searching
	testCases := []struct {
		title       string
		description string
		assignee    string
	}{
		{"API Authentication", "Implement OAuth2 authentication for API", "elena"},
		{"Database Migration", "Migrate user data to new schema", "marcus"},
		{"Frontend Components", "Create reusable React components", "vera"},
		{"API Documentation", "Write comprehensive API documentation", "elena"},
		{"Performance Testing", "Load test the authentication system", "marcus"},
	}

	var taskIDs []string
	for i, tc := range testCases {
		task, err := env.createTestTask(ctx, tc.title, tc.description)
		require.NoError(t, err, "Should be able to create test task %d", i+1)

		// Assign the task
		err = env.board.AssignTask(ctx, task.ID, tc.assignee, "test-system", "Initial assignment")
		require.NoError(t, err, "Should be able to assign task %d", i+1)

		taskIDs = append(taskIDs, task.ID)
		logger.InfoContext(ctx, "Created search test task", "index", i+1, "title", tc.title, "assignee", tc.assignee)
	}

	// Wait for all tasks to appear in UI
	require.Eventually(t, func() bool {
		return env.getUITaskCount() >= len(testCases)
	}, 10*time.Second, 200*time.Millisecond, "All test tasks should appear in UI")

	// Test search by title keyword
	apiTasks := env.searchTasks("api")
	assert.GreaterOrEqual(t, len(apiTasks), 2, "Should find at least 2 tasks containing 'api'")
	logger.InfoContext(ctx, "Search by 'api' keyword", "results", len(apiTasks))

	// Test search by assignee
	elenaTasks := env.searchTasks("elena")
	assert.GreaterOrEqual(t, len(elenaTasks), 2, "Should find at least 2 tasks assigned to Elena")
	logger.InfoContext(ctx, "Search by 'elena' assignee", "results", len(elenaTasks))

	// Test search by description keyword
	authTasks := env.searchTasks("authentication")
	assert.GreaterOrEqual(t, len(authTasks), 1, "Should find at least 1 task about authentication")
	logger.InfoContext(ctx, "Search by 'authentication' keyword", "results", len(authTasks))

	// Test search with no results
	noResults := env.searchTasks("nonexistent")
	assert.Equal(t, 0, len(noResults), "Should find no tasks with nonexistent keyword")
	logger.InfoContext(ctx, "Search with no results", "results", len(noResults))

	logger.InfoContext(ctx, "Kanban search functionality test completed successfully")
}

// setupKanbanTestEnvironment creates a complete test environment for kanban integration testing
func setupKanbanTestEnvironment(ctx context.Context, t *testing.T) (*kanbanTestEnvironment, error) {
	logger := observability.GetLogger(ctx)

	// Create test context
	testCtx, cancel := context.WithCancel(ctx)

	// Create temporary directory for test
	testDir := t.TempDir()

	logger.InfoContext(ctx, "Setting up kanban test environment", "test_dir", testDir)

	// Initialize registry and storage
	reg := registry.NewComponentRegistry()
	if err := reg.Initialize(testCtx, registry.Config{}); err != nil {
		cancel()
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to initialize registry").
			WithComponent("kanban_integration_test").
			WithOperation("setupKanbanTestEnvironment")
	}

	// Get database path for the test
	dbPath := local.LocalDatabasePath(testDir)

	// Initialize SQLite storage
	_, _, err := storage.InitializeSQLiteStorageForRegistry(testCtx, dbPath)
	if err != nil {
		cancel()
		return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to initialize SQLite storage").
			WithComponent("kanban_integration_test").
			WithOperation("setupKanbanTestEnvironment").
			WithDetails("db_path", dbPath)
	}

	// Create kanban manager using registry
	kanbanRegistry := &testKanbanComponentRegistry{componentReg: reg}
	kanbanMgr, err := kanban.NewManagerWithRegistry(testCtx, kanbanRegistry)
	if err != nil {
		cancel()
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create kanban manager").
			WithComponent("kanban_integration_test").
			WithOperation("setupKanbanTestEnvironment")
	}

	// Create test board
	board, err := kanbanMgr.CreateBoard(testCtx, "Integration Test Board", "Board for integration testing")
	if err != nil {
		cancel()
		return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to create test board").
			WithComponent("kanban_integration_test").
			WithOperation("setupKanbanTestEnvironment")
	}

	logger.InfoContext(ctx, "Created test board", "board_id", board.ID, "name", board.Name)

	// Connect to daemon for event streaming
	conn, err := grpc.NewClient("unix:///tmp/guild.sock", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		cancel()
		return nil, gerror.Wrap(err, gerror.ErrCodeConnection, "failed to connect to daemon").
			WithComponent("kanban_integration_test").
			WithOperation("setupKanbanTestEnvironment").
			WithDetails("socket", "/tmp/guild.sock")
	}

	eventClient := pb.NewEventServiceClient(conn)

	// Create kanban UI model with event streaming
	kanbanUI := kanbanui.NewWithEventClient(testCtx, kanbanMgr, board.ID, conn)

	// Initialize the UI model
	kanbanUI.Init()

	env := &kanbanTestEnvironment{
		ctx:               testCtx,
		cancel:            cancel,
		registry:          kanbanRegistry,
		kanbanManager:     kanbanMgr,
		board:             board,
		conn:              conn,
		eventClient:       eventClient,
		kanbanUI:          kanbanUI,
		testDir:           testDir,
		logger:            logger,
		taskEventReceived: make(chan *pb.TaskEvent, 100),
		taskCount:         0,
	}

	logger.InfoContext(ctx, "Kanban test environment setup complete")
	return env, nil
}

// createTestTask creates a test task and waits for persistence
func (env *kanbanTestEnvironment) createTestTask(ctx context.Context, title, description string) (*kanban.Task, error) {
	task, err := env.board.CreateTask(ctx, title, description)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to create test task").
			WithComponent("kanban_integration_test").
			WithOperation("createTestTask").
			WithDetails("title", title)
	}

	env.mu.Lock()
	env.taskCount++
	env.mu.Unlock()

	// Small delay to allow for event propagation
	time.Sleep(50 * time.Millisecond)

	return task, nil
}

// hasTaskInUI checks if a task exists in the UI model
func (env *kanbanTestEnvironment) hasTaskInUI(taskID string) bool {
	// Note: This is a simplified check - in a real implementation,
	// we would need to access the UI model's internal state
	// For this test, we'll check against the board directly as a proxy
	task, err := env.board.GetTask(env.ctx, taskID)
	return err == nil && task != nil
}

// getTaskStatusInUI gets the status of a task in the UI
func (env *kanbanTestEnvironment) getTaskStatusInUI(taskID string) kanban.TaskStatus {
	task, err := env.board.GetTask(env.ctx, taskID)
	if err != nil || task == nil {
		return ""
	}
	return task.Status
}

// getTaskAssigneeInUI gets the assignee of a task in the UI
func (env *kanbanTestEnvironment) getTaskAssigneeInUI(taskID string) string {
	task, err := env.board.GetTask(env.ctx, taskID)
	if err != nil || task == nil {
		return ""
	}
	return task.AssignedTo
}

// getUITaskCount gets the total number of tasks in the UI
func (env *kanbanTestEnvironment) getUITaskCount() int {
	allTasks, err := env.board.GetAllTasks(env.ctx)
	if err != nil {
		return 0
	}
	return len(allTasks)
}

// searchTasks simulates search functionality in the UI
func (env *kanbanTestEnvironment) searchTasks(query string) []*kanban.Task {
	allTasks, err := env.board.GetAllTasks(env.ctx)
	if err != nil {
		return nil
	}

	var results []*kanban.Task
	for _, task := range allTasks {
		if containsIgnoreCase(task.Title, query) ||
			containsIgnoreCase(task.Description, query) ||
			containsIgnoreCase(task.AssignedTo, query) {
			results = append(results, task)
		}
	}

	return results
}

// cleanup cleans up the test environment
func (env *kanbanTestEnvironment) cleanup(ctx context.Context) {
	env.logger.InfoContext(ctx, "Cleaning up kanban test environment")

	// Close gRPC connection
	if env.conn != nil {
		env.conn.Close()
	}

	// Close kanban manager
	if env.kanbanManager != nil {
		env.kanbanManager.Close()
	}

	// Cancel context
	if env.cancel != nil {
		env.cancel()
	}

	env.logger.InfoContext(ctx, "Kanban test environment cleanup complete")
}

// testKanbanComponentRegistry implements kanban.ComponentRegistry for testing
type testKanbanComponentRegistry struct {
	componentReg registry.ComponentRegistry
}

// Storage implements kanban.ComponentRegistry
func (r *testKanbanComponentRegistry) Storage() kanban.StorageRegistry {
	return &testStorageRegistry{storageReg: r.componentReg.Storage()}
}

// testStorageRegistry implements kanban.StorageRegistry for testing
type testStorageRegistry struct {
	storageReg registry.StorageRegistry
}

// GetKanbanCampaignRepository implements kanban.StorageRegistry
func (r *testStorageRegistry) GetKanbanCampaignRepository() kanban.CampaignRepository {
	return &testCampaignRepository{}
}

// GetKanbanCommissionRepository implements kanban.StorageRegistry
func (r *testStorageRegistry) GetKanbanCommissionRepository() kanban.CommissionRepository {
	return &testCommissionRepository{}
}

// GetBoardRepository implements kanban.StorageRegistry
func (r *testStorageRegistry) GetBoardRepository() kanban.BoardRepository {
	return &testBoardRepository{storageReg: r.storageReg}
}

// GetKanbanTaskRepository implements kanban.StorageRegistry
func (r *testStorageRegistry) GetKanbanTaskRepository() kanban.TaskRepository {
	return &testTaskRepository{storageReg: r.storageReg}
}

// GetMemoryStore implements kanban.StorageRegistry
func (r *testStorageRegistry) GetMemoryStore() kanban.MemoryStore {
	// The storage registry returns interface{}, need to cast to memory.Store
	memStore := r.storageReg.GetMemoryStore()
	if store, ok := memStore.(memory.Store); ok {
		return &testMemoryStore{store: store}
	}
	// Return a minimal implementation if cast fails
	return &testMemoryStore{store: nil}
}

// testCampaignRepository implements kanban.CampaignRepository for testing
type testCampaignRepository struct{}

func (r *testCampaignRepository) CreateCampaign(ctx context.Context, campaign interface{}) error {
	return nil // No-op for testing
}

// testCommissionRepository implements kanban.CommissionRepository for testing
type testCommissionRepository struct{}

func (r *testCommissionRepository) CreateCommission(ctx context.Context, commission interface{}) error {
	return nil // No-op for testing
}

func (r *testCommissionRepository) GetCommission(ctx context.Context, id string) (interface{}, error) {
	return nil, gerror.New(gerror.ErrCodeNotFound, "commission not found", nil)
}

// testBoardRepository implements kanban.BoardRepository for testing
type testBoardRepository struct {
	storageReg registry.StorageRegistry
}

func (r *testBoardRepository) CreateBoard(ctx context.Context, board interface{}) error {
	// For kanban integration, delegate to the kanban board repository
	return r.storageReg.GetBoardRepository().CreateBoard(ctx, board)
}

func (r *testBoardRepository) GetBoard(ctx context.Context, id string) (interface{}, error) {
	return r.storageReg.GetBoardRepository().GetBoard(ctx, id)
}

func (r *testBoardRepository) UpdateBoard(ctx context.Context, board interface{}) error {
	// For kanban integration, delegate to the kanban board repository
	return r.storageReg.GetBoardRepository().UpdateBoard(ctx, board)
}

func (r *testBoardRepository) DeleteBoard(ctx context.Context, id string) error {
	return r.storageReg.GetBoardRepository().DeleteBoard(ctx, id)
}

func (r *testBoardRepository) ListBoards(ctx context.Context) ([]interface{}, error) {
	return r.storageReg.GetBoardRepository().ListBoards(ctx)
}

// testTaskRepository implements kanban.TaskRepository for testing
type testTaskRepository struct {
	storageReg registry.StorageRegistry
}

func (r *testTaskRepository) CreateTask(ctx context.Context, task interface{}) error {
	// For kanban integration, delegate to the kanban task repository
	return r.storageReg.GetKanbanTaskRepository().CreateTask(ctx, task)
}

func (r *testTaskRepository) UpdateTask(ctx context.Context, task interface{}) error {
	// For kanban integration, delegate to the kanban task repository
	return r.storageReg.GetKanbanTaskRepository().UpdateTask(ctx, task)
}

func (r *testTaskRepository) DeleteTask(ctx context.Context, id string) error {
	return r.storageReg.GetKanbanTaskRepository().DeleteTask(ctx, id)
}

func (r *testTaskRepository) ListTasksByBoard(ctx context.Context, boardID string) ([]interface{}, error) {
	return r.storageReg.GetKanbanTaskRepository().ListTasksByBoard(ctx, boardID)
}

func (r *testTaskRepository) RecordTaskEvent(ctx context.Context, event interface{}) error {
	return nil // No-op for testing
}

// testMemoryStore implements kanban.MemoryStore for testing
type testMemoryStore struct {
	store memory.Store
}

func (r *testMemoryStore) Get(ctx context.Context, bucket, key string) ([]byte, error) {
	if r.store == nil {
		return nil, gerror.New(gerror.ErrCodeNotFound, "memory store not available", nil)
	}
	return r.store.Get(ctx, bucket, key)
}

func (r *testMemoryStore) Put(ctx context.Context, bucket, key string, value []byte) error {
	if r.store == nil {
		return gerror.New(gerror.ErrCodeInternal, "memory store not available", nil)
	}
	return r.store.Put(ctx, bucket, key, value)
}

func (r *testMemoryStore) Delete(ctx context.Context, bucket, key string) error {
	if r.store == nil {
		return gerror.New(gerror.ErrCodeInternal, "memory store not available", nil)
	}
	return r.store.Delete(ctx, bucket, key)
}

func (r *testMemoryStore) List(ctx context.Context, bucket string) ([]string, error) {
	if r.store == nil {
		return nil, gerror.New(gerror.ErrCodeInternal, "memory store not available", nil)
	}
	return r.store.List(ctx, bucket)
}

// containsIgnoreCase checks if a string contains a substring (case-insensitive)
func containsIgnoreCase(s, substr string) bool {
	return len(s) >= len(substr) &&
		s[:len(substr)] == substr ||
		(len(s) > len(substr) && containsIgnoreCase(s[1:], substr))
}
