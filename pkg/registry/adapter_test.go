package registry

import (
	"context"
	"testing"
	"time"

	"github.com/guild-ventures/guild-core/pkg/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSQLiteStorageRegistryAdapters(t *testing.T) {
	ctx := context.Background()

	// Initialize SQLite storage for tests
	storageReg, _, err := storage.InitializeSQLiteStorageForTests(ctx)
	require.NoError(t, err)

	// Create SQLite storage registry with adapters
	sqliteReg := &SQLiteStorageRegistry{
		registry: storageReg,
	}

	t.Run("Campaign repository adapter", func(t *testing.T) {
		campaignRepo := sqliteReg.GetCampaignRepository()
		require.NotNil(t, campaignRepo, "Campaign repository adapter should not be nil")

		// Test create
		campaign := &Campaign{
			ID:        "test-campaign",
			Name:      "Test Campaign",
			Status:    "active",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		err := campaignRepo.CreateCampaign(ctx, campaign)
		require.NoError(t, err)

		// Test get
		retrieved, err := campaignRepo.GetCampaign(ctx, "test-campaign")
		require.NoError(t, err)
		assert.Equal(t, campaign.ID, retrieved.ID)
		assert.Equal(t, campaign.Name, retrieved.Name)
	})

	t.Run("Commission repository adapter", func(t *testing.T) {
		commissionRepo := sqliteReg.GetCommissionRepository()
		require.NotNil(t, commissionRepo, "Commission repository adapter should not be nil")

		// First create a campaign for the foreign key
		campaignRepo := sqliteReg.GetCampaignRepository()
		campaign := &Campaign{
			ID:        "test-campaign-2",
			Name:      "Test Campaign 2",
			Status:    "active",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		err := campaignRepo.CreateCampaign(ctx, campaign)
		require.NoError(t, err)

		// Test create commission
		commission := &Commission{
			ID:         "test-commission",
			CampaignID: "test-campaign-2",
			Title:      "Test Commission",
			Status:     "pending",
			CreatedAt:  time.Now(),
		}
		err = commissionRepo.CreateCommission(ctx, commission)
		require.NoError(t, err)

		// Test get
		retrieved, err := commissionRepo.GetCommission(ctx, "test-commission")
		require.NoError(t, err)
		assert.Equal(t, commission.ID, retrieved.ID)
		assert.Equal(t, commission.Title, retrieved.Title)
	})

	t.Run("Task repository adapter", func(t *testing.T) {
		taskRepo := sqliteReg.GetTaskRepository()
		require.NotNil(t, taskRepo, "Task repository adapter should not be nil")

		// First create a campaign and commission for the foreign key
		campaignRepo := sqliteReg.GetCampaignRepository()
		campaign := &Campaign{
			ID:        "test-campaign-3",
			Name:      "Test Campaign 3",
			Status:    "active",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		err := campaignRepo.CreateCampaign(ctx, campaign)
		require.NoError(t, err)

		commissionRepo := sqliteReg.GetCommissionRepository()
		commission := &Commission{
			ID:         "test-commission-2",
			CampaignID: "test-campaign-3",
			Title:      "Test Commission 2",
			Status:     "pending",
			CreatedAt:  time.Now(),
		}
		err = commissionRepo.CreateCommission(ctx, commission)
		require.NoError(t, err)

		// Test create task
		desc := "Test task description"
		task := &StorageTask{
			ID:           "test-task",
			CommissionID: "test-commission-2",
			Title:        "Test Task",
			Description:  &desc,
			Status:       "todo",
			Column:       "backlog",
			StoryPoints:  3,
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}
		err = taskRepo.CreateTask(ctx, task)
		require.NoError(t, err)

		// Test get
		retrieved, err := taskRepo.GetTask(ctx, "test-task")
		require.NoError(t, err)
		assert.Equal(t, task.ID, retrieved.ID)
		assert.Equal(t, task.Title, retrieved.Title)
	})

	t.Run("Agent repository adapter", func(t *testing.T) {
		agentRepo := sqliteReg.GetAgentRepository()
		require.NotNil(t, agentRepo, "Agent repository adapter should not be nil")

		// Test create agent
		provider := "openai"
		model := "gpt-4"
		agent := &StorageAgent{
			ID:            "test-agent",
			Name:          "Test Agent",
			Type:          "worker",
			Provider:      &provider,
			Model:         &model,
			CostMagnitude: 3,
			CreatedAt:     time.Now(),
		}
		err := agentRepo.CreateAgent(ctx, agent)
		require.NoError(t, err)

		// Test get
		retrieved, err := agentRepo.GetAgent(ctx, "test-agent")
		require.NoError(t, err)
		assert.Equal(t, agent.ID, retrieved.ID)
		assert.Equal(t, agent.Name, retrieved.Name)
	})
}