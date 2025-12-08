// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package promptchain

import (
	"context"
	"database/sql"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/guild-framework/guild-core/pkg/memory"
	"github.com/guild-framework/guild-core/pkg/storage"
)

// setupTestDB creates an in-memory SQLite database with the prompt chains schema
func setupTestDB(t *testing.T) (*sql.DB, storage.PromptChainRepository) {
	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)

	// Create the prompt chains schema
	schema := `
	CREATE TABLE prompt_chains (
		id TEXT PRIMARY KEY,
		agent_id TEXT NOT NULL,
		task_id TEXT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE prompt_chain_messages (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		chain_id TEXT NOT NULL REFERENCES prompt_chains(id) ON DELETE CASCADE,
		role TEXT NOT NULL CHECK (role IN ('system', 'user', 'assistant', 'tool')),
		content TEXT NOT NULL,
		name TEXT,
		timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		token_usage INTEGER DEFAULT 0,
		FOREIGN KEY (chain_id) REFERENCES prompt_chains(id)
	);

	CREATE INDEX idx_prompt_chains_agent ON prompt_chains(agent_id);
	CREATE INDEX idx_prompt_chains_task ON prompt_chains(task_id);
	CREATE INDEX idx_prompt_chain_messages_chain ON prompt_chain_messages(chain_id);
	CREATE INDEX idx_prompt_chain_messages_timestamp ON prompt_chain_messages(timestamp);
	`

	_, err = db.Exec(schema)
	require.NoError(t, err)

	repo := storage.DefaultPromptChainRepositoryFactory(db)
	return db, repo
}

func TestSQLiteChainManager_CreateChain(t *testing.T) {
	db, repo := setupTestDB(t)
	defer db.Close()

	manager := NewSQLiteChainManager(repo)
	ctx := context.Background()

	// Test creating a chain without task ID
	agentID := "test-agent-1"
	chainID, err := manager.CreateChain(ctx, agentID, "")
	assert.NoError(t, err)
	assert.NotEmpty(t, chainID)
	assert.Contains(t, chainID, "chain_"+agentID)

	// Test creating a chain with task ID
	taskID := "test-task-1"
	chainID2, err := manager.CreateChain(ctx, agentID, taskID)
	assert.NoError(t, err)
	assert.NotEmpty(t, chainID2)
	assert.NotEqual(t, chainID, chainID2)

	// Test with empty agent ID
	_, err = manager.CreateChain(ctx, "", taskID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "agentID cannot be empty")
}

func TestSQLiteChainManager_GetChain(t *testing.T) {
	db, repo := setupTestDB(t)
	defer db.Close()

	manager := NewSQLiteChainManager(repo)
	ctx := context.Background()

	// Create a chain first
	agentID := "test-agent-2"
	taskID := "test-task-2"
	chainID, err := manager.CreateChain(ctx, agentID, taskID)
	require.NoError(t, err)

	// Get the chain
	chain, err := manager.GetChain(ctx, chainID)
	assert.NoError(t, err)
	assert.NotNil(t, chain)
	assert.Equal(t, chainID, chain.ID)
	assert.Equal(t, agentID, chain.AgentID)
	assert.Equal(t, taskID, chain.TaskID)
	assert.Empty(t, chain.Messages)

	// Test with non-existent chain
	_, err = manager.GetChain(ctx, "non-existent-chain")
	assert.Error(t, err)

	// Test with empty chain ID
	_, err = manager.GetChain(ctx, "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "chainID cannot be empty")
}

func TestSQLiteChainManager_AddMessage(t *testing.T) {
	db, repo := setupTestDB(t)
	defer db.Close()

	manager := NewSQLiteChainManager(repo)
	ctx := context.Background()

	// Create a chain
	agentID := "test-agent-3"
	chainID, err := manager.CreateChain(ctx, agentID, "")
	require.NoError(t, err)

	// Add a user message
	userMsg := memory.Message{
		Role:       "user",
		Content:    "Hello, how are you?",
		Name:       "test-user",
		Timestamp:  time.Now(),
		TokenUsage: 5,
	}
	err = manager.AddMessage(ctx, chainID, userMsg)
	assert.NoError(t, err)

	// Add an assistant message
	assistantMsg := memory.Message{
		Role:       "assistant",
		Content:    "I'm doing well, thank you!",
		Timestamp:  time.Now(),
		TokenUsage: 7,
	}
	err = manager.AddMessage(ctx, chainID, assistantMsg)
	assert.NoError(t, err)

	// Verify messages were added
	chain, err := manager.GetChain(ctx, chainID)
	require.NoError(t, err)
	assert.Len(t, chain.Messages, 2)
	assert.Equal(t, "user", chain.Messages[0].Role)
	assert.Equal(t, userMsg.Content, chain.Messages[0].Content)
	assert.Equal(t, "test-user", chain.Messages[0].Name)
	assert.Equal(t, "assistant", chain.Messages[1].Role)
	assert.Equal(t, assistantMsg.Content, chain.Messages[1].Content)

	// Test validation errors
	err = manager.AddMessage(ctx, "", userMsg)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "chainID cannot be empty")

	err = manager.AddMessage(ctx, chainID, memory.Message{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "message role cannot be empty")

	err = manager.AddMessage(ctx, chainID, memory.Message{Role: "user"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "message content cannot be empty")
}

func TestSQLiteChainManager_GetChainsByAgent(t *testing.T) {
	db, repo := setupTestDB(t)
	defer db.Close()

	manager := NewSQLiteChainManager(repo)
	ctx := context.Background()

	// Create chains for different agents
	agent1 := "test-agent-4"
	agent2 := "test-agent-5"

	// Create 3 chains for agent1
	for i := 0; i < 3; i++ {
		_, err := manager.CreateChain(ctx, agent1, "")
		require.NoError(t, err)
		time.Sleep(10 * time.Millisecond) // Ensure different timestamps
	}

	// Create 2 chains for agent2
	for i := 0; i < 2; i++ {
		_, err := manager.CreateChain(ctx, agent2, "")
		require.NoError(t, err)
		time.Sleep(10 * time.Millisecond)
	}

	// Get chains for agent1
	chains1, err := manager.GetChainsByAgent(ctx, agent1)
	assert.NoError(t, err)
	assert.Len(t, chains1, 3)
	for _, chain := range chains1 {
		assert.Equal(t, agent1, chain.AgentID)
	}

	// Get chains for agent2
	chains2, err := manager.GetChainsByAgent(ctx, agent2)
	assert.NoError(t, err)
	assert.Len(t, chains2, 2)
	for _, chain := range chains2 {
		assert.Equal(t, agent2, chain.AgentID)
	}

	// Test with non-existent agent
	chains3, err := manager.GetChainsByAgent(ctx, "non-existent-agent")
	assert.NoError(t, err)
	assert.Empty(t, chains3)

	// Test with empty agent ID
	_, err = manager.GetChainsByAgent(ctx, "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "agentID cannot be empty")
}

func TestSQLiteChainManager_GetChainsByTask(t *testing.T) {
	db, repo := setupTestDB(t)
	defer db.Close()

	manager := NewSQLiteChainManager(repo)
	ctx := context.Background()

	// Create chains for different tasks
	agentID := "test-agent-6"
	task1 := "test-task-1"
	task2 := "test-task-2"

	// Create 2 chains for task1
	for i := 0; i < 2; i++ {
		_, err := manager.CreateChain(ctx, agentID, task1)
		require.NoError(t, err)
	}

	// Create 1 chain for task2
	_, err := manager.CreateChain(ctx, agentID, task2)
	require.NoError(t, err)

	// Create 1 chain without task
	_, err = manager.CreateChain(ctx, agentID, "")
	require.NoError(t, err)

	// Get chains for task1
	chains1, err := manager.GetChainsByTask(ctx, task1)
	assert.NoError(t, err)
	assert.Len(t, chains1, 2)
	for _, chain := range chains1 {
		assert.Equal(t, task1, chain.TaskID)
	}

	// Get chains for task2
	chains2, err := manager.GetChainsByTask(ctx, task2)
	assert.NoError(t, err)
	assert.Len(t, chains2, 1)
	assert.Equal(t, task2, chains2[0].TaskID)

	// Test with empty task ID
	_, err = manager.GetChainsByTask(ctx, "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "taskID cannot be empty")
}

func TestSQLiteChainManager_BuildContext(t *testing.T) {
	db, repo := setupTestDB(t)
	defer db.Close()

	manager := NewSQLiteChainManager(repo)
	ctx := context.Background()

	// Create a chain with messages
	agentID := "test-agent-7"
	taskID := "test-task-7"
	chainID, err := manager.CreateChain(ctx, agentID, taskID)
	require.NoError(t, err)

	// Add multiple messages
	messages := []memory.Message{
		{Role: "system", Content: "You are a helpful assistant.", TokenUsage: 5},
		{Role: "user", Content: "What is 2+2?", TokenUsage: 3},
		{Role: "assistant", Content: "2+2 equals 4.", TokenUsage: 4},
		{Role: "user", Content: "What is 3+3?", TokenUsage: 3},
		{Role: "assistant", Content: "3+3 equals 6.", TokenUsage: 4},
	}

	for i, msg := range messages {
		msg.Timestamp = time.Now().Add(time.Duration(i) * time.Minute)
		err = manager.AddMessage(ctx, chainID, msg)
		require.NoError(t, err)
	}

	// Test building context without token limit
	contextMsgs, err := manager.BuildContext(ctx, agentID, taskID, 0)
	assert.NoError(t, err)
	assert.Len(t, contextMsgs, 5)

	// Verify messages are in chronological order
	for i := 0; i < len(contextMsgs)-1; i++ {
		assert.True(t, contextMsgs[i].Timestamp.Before(contextMsgs[i+1].Timestamp))
	}

	// Test building context with token limit
	contextMsgs2, err := manager.BuildContext(ctx, agentID, taskID, 10)
	assert.NoError(t, err)
	assert.True(t, len(contextMsgs2) < len(messages))

	// Test with empty agent ID
	_, err = manager.BuildContext(ctx, "", taskID, 0)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "agentID cannot be empty")
}

func TestSQLiteChainManager_DeleteChain(t *testing.T) {
	db, repo := setupTestDB(t)
	defer db.Close()

	manager := NewSQLiteChainManager(repo)
	ctx := context.Background()

	// Create a chain
	agentID := "test-agent-8"
	chainID, err := manager.CreateChain(ctx, agentID, "")
	require.NoError(t, err)

	// Add a message
	err = manager.AddMessage(ctx, chainID, memory.Message{
		Role:    "user",
		Content: "Test message",
	})
	require.NoError(t, err)

	// Verify chain exists
	chain, err := manager.GetChain(ctx, chainID)
	require.NoError(t, err)
	assert.NotNil(t, chain)

	// Delete the chain
	err = manager.DeleteChain(ctx, chainID)
	assert.NoError(t, err)

	// Verify chain no longer exists
	_, err = manager.GetChain(ctx, chainID)
	assert.Error(t, err)

	// Test deleting non-existent chain (should not error)
	err = manager.DeleteChain(ctx, "non-existent-chain")
	assert.NoError(t, err)

	// Test with empty chain ID
	err = manager.DeleteChain(ctx, "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "chainID cannot be empty")
}

func TestSQLiteChainManager_Integration(t *testing.T) {
	db, repo := setupTestDB(t)
	defer db.Close()

	manager := NewSQLiteChainManager(repo)
	ctx := context.Background()

	// Simulate a conversation between user and agent
	agentID := "conversational-agent"
	taskID := "chat-task-1"

	// Create chain
	chainID, err := manager.CreateChain(ctx, agentID, taskID)
	require.NoError(t, err)

	// Simulate conversation
	conversation := []memory.Message{
		{Role: "system", Content: "You are a helpful math tutor.", TokenUsage: 6},
		{Role: "user", Content: "Can you help me with algebra?", TokenUsage: 7},
		{Role: "assistant", Content: "Of course! I'd be happy to help you with algebra. What specific topic would you like to work on?", TokenUsage: 20},
		{Role: "user", Content: "I need help with quadratic equations.", TokenUsage: 8},
		{Role: "assistant", Content: "Great! Quadratic equations are equations of the form ax² + bx + c = 0. Let's start with a simple example.", TokenUsage: 25},
	}

	// Add messages with realistic timestamps
	for i, msg := range conversation {
		msg.Timestamp = time.Now().Add(time.Duration(i*30) * time.Second)
		err = manager.AddMessage(ctx, chainID, msg)
		require.NoError(t, err)
	}

	// Build context for the agent
	context, err := manager.BuildContext(ctx, agentID, taskID, 100)
	assert.NoError(t, err)
	assert.Len(t, context, 5)

	// Calculate total tokens
	totalTokens := 0
	for _, msg := range context {
		totalTokens += msg.TokenUsage
	}
	assert.Equal(t, 66, totalTokens)

	// Test context with lower token limit
	limitedContext, err := manager.BuildContext(ctx, agentID, taskID, 50)
	assert.NoError(t, err)
	assert.True(t, len(limitedContext) < 5)
}
