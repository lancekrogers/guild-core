package memory_test

import (
	"context"
	"errors"
	"strconv"
	"testing"
	"time"

	"github.com/guild-ventures/guild-core/pkg/memory"
	"github.com/guild-ventures/guild-core/pkg/memory/mocks"
)

// TestBoltChainManager_Implementation tests that BoltChainManager implements the ChainManager interface
func TestBoltChainManager_Implementation(t *testing.T) {
	var _ memory.ChainManager = &memory.BoltChainManager{}
}

// setupTestChainManager creates a test chain manager with a mock store
func setupTestChainManager(t *testing.T) (*memory.BoltChainManager, *mocks.MockStore) {
	mockStore := mocks.NewMockStore()
	initBuckets(mockStore)
	
	manager := memory.NewBoltChainManager(mockStore)
	return manager, mockStore
}

// initBuckets initializes the required buckets in the mock store
func initBuckets(mockStore *mocks.MockStore) {
	ctx := context.Background()
	mockStore.Put(ctx, "prompt_chains", "_init", []byte{})
	mockStore.Put(ctx, "prompt_chains_by_agent", "_init", []byte{})
	mockStore.Put(ctx, "prompt_chains_by_task", "_init", []byte{})
	
	// Reset call counters after initialization
	mockStore.PutCalls = 0
	mockStore.GetCalls = 0
	mockStore.DeleteCalls = 0
	mockStore.ListCalls = 0
}

// TestCreateChain tests the CreateChain method
func TestCreateChain(t *testing.T) {
	manager, mockStore := setupTestChainManager(t)
	ctx := context.Background()

	// Test with valid inputs
	agentID := "test-agent"
	taskID := "test-task"

	chainID, err := manager.CreateChain(ctx, agentID, taskID)
	if err != nil {
		t.Fatalf("Failed to create chain: %v", err)
	}

	if chainID == "" {
		t.Error("Expected non-empty chain ID")
	}

	// Verify store calls
	if mockStore.PutCalls != 3 { // One for the chain, one for agent index, one for task index
		t.Errorf("Expected 3 Put calls, got %d", mockStore.PutCalls)
	}

	// Test with empty agent ID
	_, err = manager.CreateChain(ctx, "", taskID)
	if err == nil {
		t.Error("Expected error with empty agent ID, got nil")
	}

	// Test with store error
	mockStore.Reset(); initBuckets(mockStore)
	mockStore.SetError("Put", errors.New("mock error"))

	_, err = manager.CreateChain(ctx, agentID, taskID)
	if err == nil {
		t.Error("Expected error with store error, got nil")
	}

	// Test with empty task ID (should still work, just not indexed by task)
	mockStore.Reset(); initBuckets(mockStore)
	chainID, err = manager.CreateChain(ctx, agentID, "")
	if err != nil {
		t.Fatalf("Failed to create chain with empty task ID: %v", err)
	}

	if mockStore.PutCalls != 2 { // One for the chain, one for agent index
		t.Errorf("Expected 2 Put calls with empty task ID, got %d", mockStore.PutCalls)
	}

	// Test with cancelled context
	cancelledCtx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err = manager.CreateChain(cancelledCtx, agentID, taskID)
	if err == nil {
		t.Error("Expected error with cancelled context, got nil")
	}
}

// TestGetChain tests the GetChain method
func TestGetChain(t *testing.T) {
	manager, mockStore := setupTestChainManager(t)
	ctx := context.Background()

	// Create a chain first
	agentID := "test-agent"
	taskID := "test-task"

	chainID, err := manager.CreateChain(ctx, agentID, taskID)
	if err != nil {
		t.Fatalf("Failed to create test chain: %v", err)
	}

	// Reset the mock store counter
	mockStore.GetCalls = 0

	// Get the chain
	chain, err := manager.GetChain(ctx, chainID)
	if err != nil {
		t.Fatalf("Failed to get chain: %v", err)
	}

	// Verify the chain properties
	if chain.ID != chainID {
		t.Errorf("Expected chain ID %s, got %s", chainID, chain.ID)
	}

	if chain.AgentID != agentID {
		t.Errorf("Expected agent ID %s, got %s", agentID, chain.AgentID)
	}

	if chain.TaskID != taskID {
		t.Errorf("Expected task ID %s, got %s", taskID, chain.TaskID)
	}

	if len(chain.Messages) != 0 {
		t.Errorf("Expected 0 messages, got %d", len(chain.Messages))
	}

	// Verify store calls
	if mockStore.GetCalls != 1 {
		t.Errorf("Expected 1 Get call, got %d", mockStore.GetCalls)
	}

	// Test with empty chain ID
	_, err = manager.GetChain(ctx, "")
	if err == nil {
		t.Error("Expected error with empty chain ID, got nil")
	}

	// Test with non-existent chain ID
	mockStore.SetError("Get", memory.ErrNotFound)
	_, err = manager.GetChain(ctx, "non-existent")
	if err == nil {
		t.Error("Expected error with non-existent chain ID, got nil")
	}

	// Test with cancelled context
	cancelledCtx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err = manager.GetChain(cancelledCtx, chainID)
	if err == nil {
		t.Error("Expected error with cancelled context, got nil")
	}
}

// TestAddMessage tests the AddMessage method
func TestAddMessage(t *testing.T) {
	manager, mockStore := setupTestChainManager(t)
	ctx := context.Background()

	// Create a chain first
	agentID := "test-agent"
	taskID := "test-task"

	chainID, err := manager.CreateChain(ctx, agentID, taskID)
	if err != nil {
		t.Fatalf("Failed to create test chain: %v", err)
	}

	// Reset mock store counters only, not the buckets
	mockStore.PutCalls = 0
	mockStore.GetCalls = 0
	mockStore.DeleteCalls = 0
	mockStore.ListCalls = 0

	// Add a message
	message := memory.Message{
		Role:      "user",
		Content:   "test message",
		Timestamp: time.Now().UTC(),
	}

	err = manager.AddMessage(ctx, chainID, message)
	if err != nil {
		t.Fatalf("Failed to add message: %v", err)
	}

	// Verify store calls
	expectedCalls := 2 // One Get, one Put
	actualCalls := mockStore.GetCalls + mockStore.PutCalls
	if actualCalls != expectedCalls {
		t.Errorf("Expected %d store calls, got %d", expectedCalls, actualCalls)
	}

	// Get the chain and verify the message was added
	chain, err := manager.GetChain(ctx, chainID)
	if err != nil {
		t.Fatalf("Failed to get chain after adding message: %v", err)
	}

	if len(chain.Messages) != 1 {
		t.Fatalf("Expected 1 message, got %d", len(chain.Messages))
	}

	if chain.Messages[0].Content != message.Content {
		t.Errorf("Expected message content %s, got %s", message.Content, chain.Messages[0].Content)
	}

	// Test with empty chain ID
	err = manager.AddMessage(ctx, "", message)
	if err == nil {
		t.Error("Expected error with empty chain ID, got nil")
	}

	// Test with non-existent chain ID
	mockStore.Reset(); initBuckets(mockStore)
	mockStore.SetError("Get", memory.ErrNotFound)
	err = manager.AddMessage(ctx, "non-existent", message)
	if err == nil {
		t.Error("Expected error with non-existent chain ID, got nil")
	}

	// Test with message without timestamp (should auto-set)
	mockStore.Reset(); initBuckets(mockStore)
	
	// Re-create the chain after reset
	chainID2, err := manager.CreateChain(ctx, "agent-123", "task-456")
	if err != nil {
		t.Fatalf("Failed to re-create chain: %v", err)
	}
	
	messageNoTimestamp := memory.Message{
		Role:    "user",
		Content: "message without timestamp",
	}

	err = manager.AddMessage(ctx, chainID2, messageNoTimestamp)
	if err != nil {
		t.Fatalf("Failed to add message without timestamp: %v", err)
	}

	// Get the chain and verify the message timestamp was set
	chain, err = manager.GetChain(ctx, chainID2)
	if err != nil {
		t.Fatalf("Failed to get chain after adding message without timestamp: %v", err)
	}

	if chain.Messages[0].Timestamp.IsZero() {
		t.Error("Expected timestamp to be set automatically, got zero value")
	}

	// Test with cancelled context
	cancelledCtx, cancel := context.WithCancel(context.Background())
	cancel()

	err = manager.AddMessage(cancelledCtx, chainID, message)
	if err == nil {
		t.Error("Expected error with cancelled context, got nil")
	}
}

// TestGetChainsByAgent tests the GetChainsByAgent method
func TestGetChainsByAgent(t *testing.T) {
	manager, mockStore := setupTestChainManager(t)
	ctx := context.Background()

	// Create multiple chains for the same agent
	agentID := "test-agent"
	numChains := 3

	for i := 0; i < numChains; i++ {
		taskID := "task-" + strconv.Itoa(i)
		_, err := manager.CreateChain(ctx, agentID, taskID)
		if err != nil {
			t.Fatalf("Failed to create test chain %d: %v", i, err)
		}
	}

	// Reset mock store counters only, not the buckets
	mockStore.PutCalls = 0
	mockStore.GetCalls = 0
	mockStore.DeleteCalls = 0
	mockStore.ListCalls = 0

	// Get chains by agent
	chains, err := manager.GetChainsByAgent(ctx, agentID)
	if err != nil {
		t.Fatalf("Failed to get chains by agent: %v", err)
	}

	if len(chains) != numChains {
		t.Errorf("Expected %d chains, got %d", numChains, len(chains))
	}

	// Verify the chains are for the correct agent
	for _, chain := range chains {
		if chain.AgentID != agentID {
			t.Errorf("Expected agent ID %s, got %s", agentID, chain.AgentID)
		}
	}

	// Test with empty agent ID
	_, err = manager.GetChainsByAgent(ctx, "")
	if err == nil {
		t.Error("Expected error with empty agent ID, got nil")
	}

	// Test with non-existent agent ID
	mockStore.Reset(); initBuckets(mockStore)
	mockStore.SetError("ListKeys", nil) // No error, just empty result
	chains, err = manager.GetChainsByAgent(ctx, "non-existent")
	if err != nil {
		t.Fatalf("Failed to get chains for non-existent agent: %v", err)
	}

	if len(chains) != 0 {
		t.Errorf("Expected 0 chains for non-existent agent, got %d", len(chains))
	}

	// Test with cancelled context
	cancelledCtx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err = manager.GetChainsByAgent(cancelledCtx, agentID)
	if err == nil {
		t.Error("Expected error with cancelled context, got nil")
	}
}

// TestGetChainsByTask tests the GetChainsByTask method
func TestGetChainsByTask(t *testing.T) {
	manager, mockStore := setupTestChainManager(t)
	ctx := context.Background()

	// Create multiple chains for the same task but different agents
	taskID := "test-task"
	numChains := 3

	for i := 0; i < numChains; i++ {
		agentID := "agent-" + strconv.Itoa(i)
		_, err := manager.CreateChain(ctx, agentID, taskID)
		if err != nil {
			t.Fatalf("Failed to create test chain %d: %v", i, err)
		}
	}

	// Reset mock store counters only, not the buckets
	mockStore.PutCalls = 0
	mockStore.GetCalls = 0
	mockStore.DeleteCalls = 0
	mockStore.ListCalls = 0

	// Get chains by task
	chains, err := manager.GetChainsByTask(ctx, taskID)
	if err != nil {
		t.Fatalf("Failed to get chains by task: %v", err)
	}

	if len(chains) != numChains {
		t.Errorf("Expected %d chains, got %d", numChains, len(chains))
	}

	// Verify the chains are for the correct task
	for _, chain := range chains {
		if chain.TaskID != taskID {
			t.Errorf("Expected task ID %s, got %s", taskID, chain.TaskID)
		}
	}

	// Test with empty task ID
	_, err = manager.GetChainsByTask(ctx, "")
	if err == nil {
		t.Error("Expected error with empty task ID, got nil")
	}

	// Test with non-existent task ID
	mockStore.Reset(); initBuckets(mockStore)
	mockStore.SetError("ListKeys", nil) // No error, just empty result
	chains, err = manager.GetChainsByTask(ctx, "non-existent")
	if err != nil {
		t.Fatalf("Failed to get chains for non-existent task: %v", err)
	}

	if len(chains) != 0 {
		t.Errorf("Expected 0 chains for non-existent task, got %d", len(chains))
	}

	// Test with cancelled context
	cancelledCtx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err = manager.GetChainsByTask(cancelledCtx, taskID)
	if err == nil {
		t.Error("Expected error with cancelled context, got nil")
	}
}

// TestBuildContext tests the BuildContext method
func TestBuildContext(t *testing.T) {
	manager, mockStore := setupTestChainManager(t)
	ctx := context.Background()

	// Create a chain
	agentID := "test-agent"
	taskID := "test-task"

	chainID, err := manager.CreateChain(ctx, agentID, taskID)
	if err != nil {
		t.Fatalf("Failed to create test chain: %v", err)
	}

	// Add several messages
	messages := []memory.Message{
		{
			Role:    "system",
			Content: "You are a helpful assistant",
		},
		{
			Role:    "user",
			Content: "Hello",
		},
		{
			Role:    "assistant",
			Content: "Hi there! How can I help you?",
		},
		{
			Role:    "user",
			Content: "Tell me a joke",
		},
	}

	for _, msg := range messages {
		if err := manager.AddMessage(ctx, chainID, msg); err != nil {
			t.Fatalf("Failed to add message: %v", err)
		}
	}

	// Reset mock store counters only, not the buckets
	mockStore.PutCalls = 0
	mockStore.GetCalls = 0
	mockStore.DeleteCalls = 0
	mockStore.ListCalls = 0

	// Build context with no token limit
	contextMessages, err := manager.BuildContext(ctx, agentID, taskID, 0)
	if err != nil {
		t.Fatalf("Failed to build context: %v", err)
	}

	if len(contextMessages) != len(messages) {
		t.Errorf("Expected %d messages in context, got %d", len(messages), len(contextMessages))
	}

	// Verify messages are in the right order (chronological)
	for i, msg := range messages {
		if contextMessages[i].Content != msg.Content {
			t.Errorf("Expected message content '%s' at position %d, got '%s'", 
				msg.Content, i, contextMessages[i].Content)
		}
	}

	// Test with token limit (simulate limiting context to first 2 messages)
	// In a real situation, this would depend on the content length
	mockStore.Reset(); initBuckets(mockStore)
	contextMessages, err = manager.BuildContext(ctx, agentID, taskID, 10)
	if err != nil {
		t.Fatalf("Failed to build context with token limit: %v", err)
	}

	if len(contextMessages) > len(messages) {
		t.Errorf("Expected fewer messages with token limit, got %d", len(contextMessages))
	}

	// Test with empty agent ID
	_, err = manager.BuildContext(ctx, "", taskID, 0)
	if err == nil {
		t.Error("Expected error with empty agent ID, got nil")
	}

	// Test with non-existent agent and task
	mockStore.Reset(); initBuckets(mockStore)
	mockStore.SetError("ListKeys", nil) // No error, just empty result
	contextMessages, err = manager.BuildContext(ctx, "non-existent", "non-existent", 0)
	if err != nil {
		t.Fatalf("Failed to build context for non-existent agent/task: %v", err)
	}

	if len(contextMessages) != 0 {
		t.Errorf("Expected 0 messages for non-existent agent/task, got %d", len(contextMessages))
	}

	// Test with cancelled context
	cancelledCtx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err = manager.BuildContext(cancelledCtx, agentID, taskID, 0)
	if err == nil {
		t.Error("Expected error with cancelled context, got nil")
	}
}

// TestDeleteChain tests the DeleteChain method
func TestDeleteChain(t *testing.T) {
	manager, mockStore := setupTestChainManager(t)
	ctx := context.Background()

	// Create a chain
	agentID := "test-agent"
	taskID := "test-task"

	chainID, err := manager.CreateChain(ctx, agentID, taskID)
	if err != nil {
		t.Fatalf("Failed to create test chain: %v", err)
	}

	// Reset mock store counters only, not the buckets
	mockStore.PutCalls = 0
	mockStore.GetCalls = 0
	mockStore.DeleteCalls = 0
	mockStore.ListCalls = 0

	// Delete the chain
	err = manager.DeleteChain(ctx, chainID)
	if err != nil {
		t.Fatalf("Failed to delete chain: %v", err)
	}

	// Verify store calls
	expectedCalls := 4 // One Get, three Delete
	actualCalls := mockStore.GetCalls + mockStore.DeleteCalls
	if actualCalls != expectedCalls {
		t.Errorf("Expected %d store calls, got %d", expectedCalls, actualCalls)
	}

	// Try to get the deleted chain
	mockStore.Reset(); initBuckets(mockStore)
	mockStore.SetError("Get", memory.ErrNotFound)
	_, err = manager.GetChain(ctx, chainID)
	if err == nil {
		t.Error("Expected error when getting deleted chain, got nil")
	}

	// Test with empty chain ID
	err = manager.DeleteChain(ctx, "")
	if err == nil {
		t.Error("Expected error with empty chain ID, got nil")
	}

	// Test with non-existent chain ID
	mockStore.Reset(); initBuckets(mockStore)
	mockStore.SetError("Get", memory.ErrNotFound)
	err = manager.DeleteChain(ctx, "non-existent")
	if err == nil {
		t.Error("Expected error with non-existent chain ID, got nil")
	}

	// Test with cancelled context
	cancelledCtx, cancel := context.WithCancel(context.Background())
	cancel()

	err = manager.DeleteChain(cancelledCtx, chainID)
	if err == nil {
		t.Error("Expected error with cancelled context, got nil")
	}
}