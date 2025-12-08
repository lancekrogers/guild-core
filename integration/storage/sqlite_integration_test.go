// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

//go:build integration
// +build integration

package storage

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/guild-framework/guild-core/pkg/memory"
	"github.com/guild-framework/guild-core/pkg/storage"
)

// TestSQLiteStorageIntegration tests the SQLite storage implementation directly
// This replaces the complex adapter-based test with a straightforward approach
func TestSQLiteStorageIntegration(t *testing.T) {
	ctx := context.Background()

	t.Run("SQLite storage initializes correctly", func(t *testing.T) {
		// Initialize SQLite storage
		storageReg, memStore, err := storage.InitializeSQLiteStorageForTests(ctx)
		require.NoError(t, err)
		require.NotNil(t, storageReg)
		require.NotNil(t, memStore)

		// Verify all repositories are available
		assert.NotNil(t, storageReg.GetCampaignRepository())
		assert.NotNil(t, storageReg.GetCommissionRepository())
		assert.NotNil(t, storageReg.GetBoardRepository())
		assert.NotNil(t, storageReg.GetTaskRepository())
		assert.NotNil(t, storageReg.GetAgentRepository())
		assert.NotNil(t, storageReg.GetPromptChainRepository())
	})

	t.Run("Campaign → Commission → Board → Task hierarchy", func(t *testing.T) {
		// Initialize SQLite storage
		storageReg, _, err := storage.InitializeSQLiteStorageForTests(ctx)
		require.NoError(t, err)

		campaignRepo := storageReg.GetCampaignRepository()
		commissionRepo := storageReg.GetCommissionRepository()
		boardRepo := storageReg.GetBoardRepository()
		taskRepo := storageReg.GetTaskRepository()

		// 1. Create Campaign
		campaign := &storage.Campaign{
			ID:     "test-campaign-1",
			Name:   "Test Campaign",
			Status: "active",
		}
		err = campaignRepo.CreateCampaign(ctx, campaign)
		require.NoError(t, err)

		// 2. Create Commission linked to Campaign
		commission := &storage.Commission{
			ID:         "test-commission-1",
			CampaignID: campaign.ID,
			Title:      "Test Commission",
			Status:     "pending",
		}
		err = commissionRepo.CreateCommission(ctx, commission)
		require.NoError(t, err)

		// 3. Create Board linked to Commission
		board := &storage.Board{
			ID:           "test-board-1",
			CommissionID: commission.ID,
			Name:         "Test Board",
			Status:       "active",
		}
		err = boardRepo.CreateBoard(ctx, board)
		require.NoError(t, err)

		// 4. Create Tasks linked to Commission (and optionally Board)
		task1 := &storage.Task{
			ID:           "test-task-1",
			CommissionID: commission.ID,
			BoardID:      &board.ID,
			Title:        "First Test Task",
			Status:       "todo",
			StoryPoints:  3,
		}
		err = taskRepo.CreateTask(ctx, task1)
		require.NoError(t, err)

		task2 := &storage.Task{
			ID:           "test-task-2",
			CommissionID: commission.ID,
			BoardID:      &board.ID,
			Title:        "Second Test Task",
			Status:       "in_progress",
			StoryPoints:  5,
		}
		err = taskRepo.CreateTask(ctx, task2)
		require.NoError(t, err)

		// 5. Verify the complete hierarchy by reading back

		// Get campaign
		retrievedCampaign, err := campaignRepo.GetCampaign(ctx, campaign.ID)
		require.NoError(t, err)
		assert.Equal(t, campaign.Name, retrievedCampaign.Name)
		assert.Equal(t, campaign.Status, retrievedCampaign.Status)

		// Get commission and verify foreign key
		retrievedCommission, err := commissionRepo.GetCommission(ctx, commission.ID)
		require.NoError(t, err)
		assert.Equal(t, commission.Title, retrievedCommission.Title)
		assert.Equal(t, campaign.ID, retrievedCommission.CampaignID)

		// Get board and verify foreign key
		retrievedBoard, err := boardRepo.GetBoard(ctx, board.ID)
		require.NoError(t, err)
		assert.Equal(t, board.Name, retrievedBoard.Name)
		assert.Equal(t, commission.ID, retrievedBoard.CommissionID)

		// Get tasks and verify foreign keys
		retrievedTask1, err := taskRepo.GetTask(ctx, task1.ID)
		require.NoError(t, err)
		assert.Equal(t, task1.Title, retrievedTask1.Title)
		assert.Equal(t, commission.ID, retrievedTask1.CommissionID)
		assert.Equal(t, board.ID, *retrievedTask1.BoardID)

		retrievedTask2, err := taskRepo.GetTask(ctx, task2.ID)
		require.NoError(t, err)
		assert.Equal(t, task2.Title, retrievedTask2.Title)
		assert.Equal(t, commission.ID, retrievedTask2.CommissionID)
		assert.Equal(t, board.ID, *retrievedTask2.BoardID)

		// 6. Test relationship queries

		// List tasks by commission
		commissionTasks, err := taskRepo.ListTasksByCommission(ctx, commission.ID)
		require.NoError(t, err)
		assert.Len(t, commissionTasks, 2)

		// List tasks by board
		boardTasks, err := taskRepo.ListTasksByBoard(ctx, board.ID)
		require.NoError(t, err)
		assert.Len(t, boardTasks, 2)

		// List commissions by campaign
		campaignCommissions, err := commissionRepo.ListCommissionsByCampaign(ctx, campaign.ID)
		require.NoError(t, err)
		assert.Len(t, campaignCommissions, 1)
		assert.Equal(t, commission.ID, campaignCommissions[0].ID)
	})

	t.Run("Task events and history tracking", func(t *testing.T) {
		// Initialize SQLite storage
		storageReg, _, err := storage.InitializeSQLiteStorageForTests(ctx)
		require.NoError(t, err)

		// Create required parent records for foreign key constraints
		campaignRepo := storageReg.GetCampaignRepository()
		commissionRepo := storageReg.GetCommissionRepository()
		taskRepo := storageReg.GetTaskRepository()

		// Create campaign first
		campaign := &storage.Campaign{
			ID:     "events-campaign",
			Name:   "Events Test Campaign",
			Status: "active",
		}
		err = campaignRepo.CreateCampaign(ctx, campaign)
		require.NoError(t, err)

		// Create commission linked to campaign
		commission := &storage.Commission{
			ID:         "events-commission",
			CampaignID: campaign.ID,
			Title:      "Events Test Commission",
			Status:     "pending",
		}
		err = commissionRepo.CreateCommission(ctx, commission)
		require.NoError(t, err)

		// Create a task with valid commission reference
		task := &storage.Task{
			ID:           "test-task-events",
			CommissionID: commission.ID,
			Title:        "Task for Event Testing",
			Status:       "todo",
			StoryPoints:  2,
		}
		err = taskRepo.CreateTask(ctx, task)
		require.NoError(t, err)

		// Record some events
		event1 := &storage.TaskEvent{
			TaskID:    task.ID,
			EventType: "created",
			NewValue:  strPtr("todo"),
			Reason:    strPtr("Task created"),
			CreatedAt: time.Now(),
		}
		err = taskRepo.RecordTaskEvent(ctx, event1)
		require.NoError(t, err)

		event2 := &storage.TaskEvent{
			TaskID:    task.ID,
			EventType: "status_changed",
			OldValue:  strPtr("todo"),
			NewValue:  strPtr("in_progress"),
			Reason:    strPtr("Agent started working"),
			CreatedAt: time.Now().Add(1 * time.Minute),
		}
		err = taskRepo.RecordTaskEvent(ctx, event2)
		require.NoError(t, err)

		// Get task history
		history, err := taskRepo.GetTaskHistory(ctx, task.ID)
		require.NoError(t, err)
		assert.Len(t, history, 2)

		// Verify events are in chronological order
		assert.Equal(t, "created", history[0].EventType)
		assert.Equal(t, "status_changed", history[1].EventType)
		assert.Equal(t, "Task created", *history[0].Reason)
		assert.Equal(t, "Agent started working", *history[1].Reason)
	})

	t.Run("Memory store adapter works", func(t *testing.T) {
		// Initialize SQLite storage
		_, memStoreAdapter, err := storage.InitializeSQLiteStorageForTests(ctx)
		require.NoError(t, err)
		require.NotNil(t, memStoreAdapter)

		// Cast to memory store interface
		memStore := memStoreAdapter.(memory.Store)

		// Store some data as JSON bytes
		testData := []byte(`{"key1":"value1","key2":42,"key3":["a","b","c"]}`)

		err = memStore.Put(ctx, "test-bucket", "test-key", testData)
		require.NoError(t, err)

		// Retrieve the data
		retrievedData, err := memStore.Get(ctx, "test-bucket", "test-key")
		require.NoError(t, err)
		assert.NotNil(t, retrievedData)
		assert.Contains(t, string(retrievedData), "value1")
		assert.Contains(t, string(retrievedData), "42")

		// Test non-existent key
		_, err = memStore.Get(ctx, "test-bucket", "non-existent")
		assert.Error(t, err) // Should return error for non-existent key

		// Test list keys
		keys, err := memStore.List(ctx, "test-bucket")
		require.NoError(t, err)
		assert.Contains(t, keys, "test-key")

		// Test delete
		err = memStore.Delete(ctx, "test-bucket", "test-key")
		require.NoError(t, err)

		// Verify deletion
		_, err = memStore.Get(ctx, "test-bucket", "test-key")
		assert.Error(t, err) // Should return error for deleted key
	})

	t.Run("Agent repository CRUD operations", func(t *testing.T) {
		// Initialize SQLite storage
		storageReg, _, err := storage.InitializeSQLiteStorageForTests(ctx)
		require.NoError(t, err)

		agentRepo := storageReg.GetAgentRepository()

		// Create an agent
		agent := &storage.Agent{
			ID:       "test-agent-1",
			Name:     "Test Agent",
			Type:     "worker",
			Provider: strPtr("openai"),
			Model:    strPtr("gpt-4"),
			Capabilities: map[string]interface{}{
				"coding":    true,
				"reasoning": true,
				"languages": []string{"go", "python"},
			},
			Tools: map[string]interface{}{
				"file_operations": true,
				"shell_commands":  false,
			},
			CostMagnitude: 3,
		}

		err = agentRepo.CreateAgent(ctx, agent)
		require.NoError(t, err)

		// Get the agent
		retrievedAgent, err := agentRepo.GetAgent(ctx, agent.ID)
		require.NoError(t, err)
		assert.Equal(t, agent.Name, retrievedAgent.Name)
		assert.Equal(t, agent.Type, retrievedAgent.Type)
		assert.Equal(t, agent.Provider, retrievedAgent.Provider)
		assert.Equal(t, agent.Model, retrievedAgent.Model)
		assert.Equal(t, agent.CostMagnitude, retrievedAgent.CostMagnitude)

		// Verify capabilities JSON marshaling
		assert.True(t, retrievedAgent.Capabilities["coding"].(bool))
		assert.Contains(t, retrievedAgent.Capabilities["languages"], "go")

		// List agents
		agents, err := agentRepo.ListAgents(ctx)
		require.NoError(t, err)
		assert.Len(t, agents, 1)
		assert.Equal(t, agent.ID, agents[0].ID)

		// Update agent
		agent.CostMagnitude = 4
		err = agentRepo.UpdateAgent(ctx, agent)
		require.NoError(t, err)

		// Verify update
		updatedAgent, err := agentRepo.GetAgent(ctx, agent.ID)
		require.NoError(t, err)
		assert.Equal(t, int32(4), updatedAgent.CostMagnitude)

		// Delete agent
		err = agentRepo.DeleteAgent(ctx, agent.ID)
		require.NoError(t, err)

		// Verify deletion
		_, err = agentRepo.GetAgent(ctx, agent.ID)
		assert.Error(t, err) // Should return not found error
	})
}

// Helper function to create string pointers
func strPtr(s string) *string {
	return &s
}
