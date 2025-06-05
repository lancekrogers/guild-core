package memory_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/guild-ventures/guild-core/pkg/memory"
	"github.com/guild-ventures/guild-core/pkg/storage"
)

// TestPromptChainIntegration tests the full integration of the prompt chain system
func TestPromptChainIntegration(t *testing.T) {
	ctx := context.Background()

	// Initialize SQLite storage for testing
	storageReg, _, err := storage.InitializeSQLiteStorageForTests(ctx)
	require.NoError(t, err)

	// Get the prompt chain repository
	promptChainRepo := storageReg.GetPromptChainRepository()
	require.NotNil(t, promptChainRepo)

	// Create SQLite chain manager
	chainManager := memory.NewSQLiteChainManager(promptChainRepo)

	// Test 1: Create a chain for an agent
	agentID := "test-agent-integration"
	taskID := "test-task-integration"

	chainID, err := chainManager.CreateChain(ctx, agentID, taskID)
	assert.NoError(t, err)
	assert.NotEmpty(t, chainID)

	// Test 2: Add messages to simulate a conversation
	messages := []memory.Message{
		{
			Role:       "system",
			Content:    "You are a helpful AI assistant.",
			Timestamp:  time.Now(),
			TokenUsage: 8,
		},
		{
			Role:       "user",
			Content:    "Hello! Can you help me understand how the Guild Framework works?",
			Timestamp:  time.Now().Add(1 * time.Second),
			TokenUsage: 14,
		},
		{
			Role:       "assistant",
			Content:    "Of course! The Guild Framework is an AI agent orchestration system that uses a medieval guild metaphor...",
			Timestamp:  time.Now().Add(2 * time.Second),
			TokenUsage: 25,
		},
	}

	for _, msg := range messages {
		err = chainManager.AddMessage(ctx, chainID, msg)
		assert.NoError(t, err)
	}

	// Test 3: Retrieve the chain and verify messages
	chain, err := chainManager.GetChain(ctx, chainID)
	assert.NoError(t, err)
	assert.NotNil(t, chain)
	assert.Equal(t, chainID, chain.ID)
	assert.Equal(t, agentID, chain.AgentID)
	assert.Equal(t, taskID, chain.TaskID)
	assert.Len(t, chain.Messages, 3)

	// Verify message order and content
	for i, msg := range chain.Messages {
		assert.Equal(t, messages[i].Role, msg.Role)
		assert.Equal(t, messages[i].Content, msg.Content)
		assert.Equal(t, messages[i].TokenUsage, msg.TokenUsage)
	}

	// Test 4: Build context with token limit
	contextMessages, err := chainManager.BuildContext(ctx, agentID, taskID, 40)
	assert.NoError(t, err)
	assert.True(t, len(contextMessages) > 0)

	// Calculate total tokens in context
	totalTokens := 0
	for _, msg := range contextMessages {
		totalTokens += msg.TokenUsage
	}
	assert.LessOrEqual(t, totalTokens, 40)

	// Test 5: Get chains by agent
	agentChains, err := chainManager.GetChainsByAgent(ctx, agentID)
	assert.NoError(t, err)
	assert.Len(t, agentChains, 1)
	assert.Equal(t, chainID, agentChains[0].ID)

	// Test 6: Get chains by task
	taskChains, err := chainManager.GetChainsByTask(ctx, taskID)
	assert.NoError(t, err)
	assert.Len(t, taskChains, 1)
	assert.Equal(t, chainID, taskChains[0].ID)

	// Test 7: Delete the chain
	err = chainManager.DeleteChain(ctx, chainID)
	assert.NoError(t, err)

	// Verify chain is deleted
	_, err = chainManager.GetChain(ctx, chainID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

// TestPromptChainConcurrency tests concurrent access to the prompt chain system
func TestPromptChainConcurrency(t *testing.T) {
	t.Skip("Skipping concurrency test - SQLite in-memory databases have connection isolation issues")
	
	ctx := context.Background()

	// Initialize SQLite storage for testing
	storageReg, _, err := storage.InitializeSQLiteStorageForTests(ctx)
	require.NoError(t, err)

	// Get the prompt chain repository
	promptChainRepo := storageReg.GetPromptChainRepository()
	require.NotNil(t, promptChainRepo)

	// Create SQLite chain manager
	chainManager := memory.NewSQLiteChainManager(promptChainRepo)

	// Create multiple chains concurrently
	numAgents := 5
	numMessagesPerAgent := 10
	
	type result struct {
		agentID string
		chainID string
		err     error
	}
	
	results := make(chan result, numAgents)

	// Launch goroutines to create chains and add messages
	for i := 0; i < numAgents; i++ {
		go func(agentNum int) {
			agentID := fmt.Sprintf("agent-%d", agentNum)
			chainID, err := chainManager.CreateChain(ctx, agentID, "")
			if err != nil {
				results <- result{agentID: agentID, err: err}
				return
			}

			// Add messages
			for j := 0; j < numMessagesPerAgent; j++ {
				msg := memory.Message{
					Role:       "user",
					Content:    fmt.Sprintf("Message %d from agent %d", j, agentNum),
					Timestamp:  time.Now(),
					TokenUsage: 5,
				}
				if err := chainManager.AddMessage(ctx, chainID, msg); err != nil {
					results <- result{agentID: agentID, chainID: chainID, err: err}
					return
				}
			}

			results <- result{agentID: agentID, chainID: chainID}
		}(i)
	}

	// Collect results
	chainIDs := make(map[string]string)
	for i := 0; i < numAgents; i++ {
		res := <-results
		assert.NoError(t, res.err)
		if res.err == nil {
			chainIDs[res.agentID] = res.chainID
		}
	}

	// Verify all chains were created and have the correct number of messages
	for agentID, chainID := range chainIDs {
		chain, err := chainManager.GetChain(ctx, chainID)
		assert.NoError(t, err)
		assert.Equal(t, agentID, chain.AgentID)
		assert.Len(t, chain.Messages, numMessagesPerAgent)
	}
}