package storage

import (
	"context"
	"testing"

	"github.com/guild-ventures/guild-core/pkg/memory"
	"github.com/guild-ventures/guild-core/pkg/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStoragePackageOnly(t *testing.T) {
	ctx := context.Background()

	t.Run("SQLite storage initializes for tests", func(t *testing.T) {
		// Test the storage initialization directly
		storageReg, memoryStoreAdapter, err := storage.InitializeSQLiteStorageForTests(ctx)
		require.NoError(t, err, "Storage initialization should succeed")
		require.NotNil(t, storageReg, "Storage registry should be initialized")
		require.NotNil(t, memoryStoreAdapter, "Memory store adapter should be available")

		// Verify repositories are available
		taskRepo := storageReg.GetTaskRepository()
		require.NotNil(t, taskRepo, "Task repository should be available")

		campaignRepo := storageReg.GetCampaignRepository()
		require.NotNil(t, campaignRepo, "Campaign repository should be available")

		commissionRepo := storageReg.GetCommissionRepository()
		require.NotNil(t, commissionRepo, "Commission repository should be available")

		agentRepo := storageReg.GetAgentRepository()
		require.NotNil(t, agentRepo, "Agent repository should be available")
	})

	t.Run("Task repository CRUD operations", func(t *testing.T) {
		// Initialize storage
		storageReg, _, err := storage.InitializeSQLiteStorageForTests(ctx)
		require.NoError(t, err)

		// Create required parent records first (respecting foreign key constraints)
		campaignRepo := storageReg.GetCampaignRepository()
		commissionRepo := storageReg.GetCommissionRepository()
		taskRepo := storageReg.GetTaskRepository()
		
		require.NotNil(t, campaignRepo)
		require.NotNil(t, commissionRepo)
		require.NotNil(t, taskRepo)

		// Create campaign first
		campaign := &storage.Campaign{
			ID:     "test-campaign-1",
			Name:   "Test Campaign",
			Status: "active",
		}
		err = campaignRepo.CreateCampaign(ctx, campaign)
		require.NoError(t, err, "Should create campaign successfully")

		// Create commission second
		commission := &storage.Commission{
			ID:         "test-commission-1",
			CampaignID: "test-campaign-1",
			Title:      "Test Commission",
			Status:     "pending",
		}
		err = commissionRepo.CreateCommission(ctx, commission)
		require.NoError(t, err, "Should create commission successfully")

		// Now create task with valid commission reference
		task := &storage.Task{
			ID:           "test-task-1",
			CommissionID: "test-commission-1",
			Title:        "Test Task",
			Description:  stringPtr("Test task description"),
			Status:       "todo",
			Column:       "backlog",
			StoryPoints:  3,
		}

		err = taskRepo.CreateTask(ctx, task)
		require.NoError(t, err, "Should create task successfully")

		// Retrieve the task
		retrievedTask, err := taskRepo.GetTask(ctx, "test-task-1")
		require.NoError(t, err, "Should retrieve task successfully")
		require.NotNil(t, retrievedTask, "Retrieved task should not be nil")

		assert.Equal(t, "test-task-1", retrievedTask.ID)
		assert.Equal(t, "Test Task", retrievedTask.Title)
		assert.Equal(t, "todo", retrievedTask.Status)
		assert.Equal(t, int32(3), retrievedTask.StoryPoints)

		// List tasks
		tasks, err := taskRepo.ListTasks(ctx)
		require.NoError(t, err, "Should list tasks successfully")
		assert.Len(t, tasks, 1, "Should have exactly one task")
		assert.Equal(t, "test-task-1", tasks[0].ID)

		// Update task status
		err = taskRepo.UpdateTaskStatus(ctx, "test-task-1", "in_progress")
		require.NoError(t, err, "Should update task status")

		// Verify update
		updatedTask, err := taskRepo.GetTask(ctx, "test-task-1")
		require.NoError(t, err)
		assert.Equal(t, "in_progress", updatedTask.Status)

		// Delete task
		err = taskRepo.DeleteTask(ctx, "test-task-1")
		require.NoError(t, err, "Should delete task successfully")

		// Verify deletion
		_, err = taskRepo.GetTask(ctx, "test-task-1")
		assert.Error(t, err, "Should not find deleted task")
	})

	t.Run("Memory store adapter works", func(t *testing.T) {
		// Initialize storage
		storageReg, memoryStoreAdapter, err := storage.InitializeSQLiteStorageForTests(ctx)
		require.NoError(t, err)
		require.NotNil(t, storageReg)
		require.NotNil(t, memoryStoreAdapter)

		// Cast to memory store interface
		memStore := memoryStoreAdapter.(memory.Store)

		// Create required parent records first for foreign key constraints
		campaignRepo := storageReg.GetCampaignRepository()
		commissionRepo := storageReg.GetCommissionRepository()

		// Create campaign
		campaign := &storage.Campaign{
			ID:     "adapter-campaign-1",
			Name:   "Adapter Test Campaign",
			Status: "active",
		}
		err = campaignRepo.CreateCampaign(ctx, campaign)
		require.NoError(t, err)

		// Create commission
		commission := &storage.Commission{
			ID:         "adapter-commission-1",
			CampaignID: "adapter-campaign-1",
			Title:      "Adapter Test Commission",
			Status:     "pending",
		}
		err = commissionRepo.CreateCommission(ctx, commission)
		require.NoError(t, err)

		// Test Put operation with valid task JSON
		testData := []byte(`{
			"id":"test-task-1",
			"commission_id":"adapter-commission-1",
			"title":"Test Task",
			"description":"Test task description",
			"status":"todo",
			"column":"backlog",
			"story_points":3
		}`)
		err = memStore.Put(ctx, "tasks", "test-task-1", testData)
		require.NoError(t, err, "Should put data successfully")

		// Test Get operation
		retrievedData, err := memStore.Get(ctx, "tasks", "test-task-1")
		require.NoError(t, err, "Should get data successfully")
		require.NotNil(t, retrievedData, "Retrieved data should not be nil")

		// Test List operation
		keys, err := memStore.List(ctx, "tasks")
		require.NoError(t, err, "Should list keys successfully")
		assert.Contains(t, keys, "test-task-1", "Should contain our test key")

		// Test Delete operation
		err = memStore.Delete(ctx, "tasks", "test-task-1")
		require.NoError(t, err, "Should delete successfully")

		// Verify deletion
		_, err = memStore.Get(ctx, "tasks", "test-task-1")
		assert.Error(t, err, "Should not find deleted item")
	})
}

// Helper function to create string pointers
func stringPtr(s string) *string {
	return &s
}