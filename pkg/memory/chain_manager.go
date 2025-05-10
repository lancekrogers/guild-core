package memory

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/google/uuid"
)

// BoltChainManager implements the ChainManager interface using a BoltDB store
type BoltChainManager struct {
	store Store
}

// NewBoltChainManager creates a new BoltChainManager
func NewBoltChainManager(store Store) *BoltChainManager {
	return &BoltChainManager{
		store: store,
	}
}

// CreateChain creates a new prompt chain
func (m *BoltChainManager) CreateChain(ctx context.Context, agentID, taskID string) (string, error) {
	if agentID == "" {
		return "", fmt.Errorf("agentID cannot be empty")
	}

	chainID := uuid.New().String()
	chain := &PromptChain{
		ID:        chainID,
		AgentID:   agentID,
		TaskID:    taskID,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
		Messages:  []Message{},
	}

	// Store the chain
	chainData, err := json.Marshal(chain)
	if err != nil {
		return "", fmt.Errorf("failed to marshal chain: %w", err)
	}

	if err := m.store.Put(ctx, "prompt_chains", chainID, chainData); err != nil {
		return "", fmt.Errorf("failed to store chain: %w", err)
	}

	// Index by agent
	if err := m.store.Put(ctx, "prompt_chains_by_agent", agentID+":"+chainID, []byte(chainID)); err != nil {
		return "", fmt.Errorf("failed to index chain by agent: %w", err)
	}

	// Index by task if provided
	if taskID != "" {
		if err := m.store.Put(ctx, "prompt_chains_by_task", taskID+":"+chainID, []byte(chainID)); err != nil {
			return "", fmt.Errorf("failed to index chain by task: %w", err)
		}
	}

	return chainID, nil
}

// GetChain retrieves a chain by ID
func (m *BoltChainManager) GetChain(ctx context.Context, chainID string) (*PromptChain, error) {
	if chainID == "" {
		return nil, fmt.Errorf("chainID cannot be empty")
	}

	data, err := m.store.Get(ctx, "prompt_chains", chainID)
	if err != nil {
		return nil, fmt.Errorf("failed to get chain: %w", err)
	}

	var chain PromptChain
	if err := json.Unmarshal(data, &chain); err != nil {
		return nil, fmt.Errorf("failed to unmarshal chain: %w", err)
	}

	return &chain, nil
}

// AddMessage adds a message to a chain
func (m *BoltChainManager) AddMessage(ctx context.Context, chainID string, message Message) error {
	if chainID == "" {
		return fmt.Errorf("chainID cannot be empty")
	}

	// Set timestamp if not provided
	if message.Timestamp.IsZero() {
		message.Timestamp = time.Now().UTC()
	}

	// Get the chain
	chain, err := m.GetChain(ctx, chainID)
	if err != nil {
		return fmt.Errorf("failed to get chain: %w", err)
	}

	// Add the message
	chain.Messages = append(chain.Messages, message)
	chain.UpdatedAt = time.Now().UTC()

	// Store the updated chain
	chainData, err := json.Marshal(chain)
	if err != nil {
		return fmt.Errorf("failed to marshal chain: %w", err)
	}

	if err := m.store.Put(ctx, "prompt_chains", chainID, chainData); err != nil {
		return fmt.Errorf("failed to update chain: %w", err)
	}

	return nil
}

// GetChainsByAgent retrieves all chains for an agent
func (m *BoltChainManager) GetChainsByAgent(ctx context.Context, agentID string) ([]*PromptChain, error) {
	if agentID == "" {
		return nil, fmt.Errorf("agentID cannot be empty")
	}

	// Get all keys with the agent prefix
	keys, err := m.store.ListKeys(ctx, "prompt_chains_by_agent", agentID+":")
	if err != nil {
		return nil, fmt.Errorf("failed to list chains by agent: %w", err)
	}

	var chains []*PromptChain
	for _, key := range keys {
		// Extract chain ID from the key
		chainIDBytes, err := m.store.Get(ctx, "prompt_chains_by_agent", key)
		if err != nil {
			continue // Skip this one if there's an error
		}

		// Get the chain
		chain, err := m.GetChain(ctx, string(chainIDBytes))
		if err != nil {
			continue // Skip this one if there's an error
		}

		chains = append(chains, chain)
	}

	// Sort by created time, newest first
	sort.Slice(chains, func(i, j int) bool {
		return chains[i].CreatedAt.After(chains[j].CreatedAt)
	})

	return chains, nil
}

// GetChainsByTask retrieves all chains for a task
func (m *BoltChainManager) GetChainsByTask(ctx context.Context, taskID string) ([]*PromptChain, error) {
	if taskID == "" {
		return nil, fmt.Errorf("taskID cannot be empty")
	}

	// Get all keys with the task prefix
	keys, err := m.store.ListKeys(ctx, "prompt_chains_by_task", taskID+":")
	if err != nil {
		return nil, fmt.Errorf("failed to list chains by task: %w", err)
	}

	var chains []*PromptChain
	for _, key := range keys {
		// Extract chain ID from the key
		chainIDBytes, err := m.store.Get(ctx, "prompt_chains_by_task", key)
		if err != nil {
			continue // Skip this one if there's an error
		}

		// Get the chain
		chain, err := m.GetChain(ctx, string(chainIDBytes))
		if err != nil {
			continue // Skip this one if there's an error
		}

		chains = append(chains, chain)
	}

	// Sort by created time, newest first
	sort.Slice(chains, func(i, j int) bool {
		return chains[i].CreatedAt.After(chains[j].CreatedAt)
	})

	return chains, nil
}

// BuildContext builds a context from chains for an agent and task
func (m *BoltChainManager) BuildContext(ctx context.Context, agentID, taskID string, maxTokens int) ([]Message, error) {
	var allMessages []Message
	
	// Get chains for this agent and task
	var chains []*PromptChain
	var err error
	
	if taskID != "" {
		// If taskID is provided, get chains for this specific task
		chains, err = m.GetChainsByTask(ctx, taskID)
		if err != nil {
			return nil, fmt.Errorf("failed to get chains by task: %w", err)
		}
	} else {
		// Otherwise, get all chains for this agent
		chains, err = m.GetChainsByAgent(ctx, agentID)
		if err != nil {
			return nil, fmt.Errorf("failed to get chains by agent: %w", err)
		}
	}
	
	// Extract messages from all chains, newest first
	for _, chain := range chains {
		for i := len(chain.Messages) - 1; i >= 0; i-- {
			allMessages = append(allMessages, chain.Messages[i])
		}
	}
	
	// Sort messages by timestamp, newest first
	sort.Slice(allMessages, func(i, j int) bool {
		return allMessages[i].Timestamp.After(allMessages[j].Timestamp)
	})
	
	// Limit the context to maxTokens if provided
	if maxTokens > 0 {
		var totalTokens int
		var contextMessages []Message
		
		for _, msg := range allMessages {
			// This is a simplified token count estimation
			// In practice, you'd use a proper tokenizer
			tokenCount := len(msg.Content) / 4
			if tokenCount == 0 {
				tokenCount = 1 // Ensure we count at least 1 token per message
			}
			
			if totalTokens + tokenCount > maxTokens {
				break
			}
			
			contextMessages = append(contextMessages, msg)
			totalTokens += tokenCount
		}
		
		allMessages = contextMessages
	}
	
	// Reverse again to get chronological order
	for i, j := 0, len(allMessages)-1; i < j; i, j = i+1, j-1 {
		allMessages[i], allMessages[j] = allMessages[j], allMessages[i]
	}
	
	return allMessages, nil
}

// DeleteChain deletes a chain
func (m *BoltChainManager) DeleteChain(ctx context.Context, chainID string) error {
	if chainID == "" {
		return fmt.Errorf("chainID cannot be empty")
	}
	
	// Get the chain first to retrieve agent and task IDs
	chain, err := m.GetChain(ctx, chainID)
	if err != nil {
		return fmt.Errorf("failed to get chain: %w", err)
	}
	
	// Delete the chain
	if err := m.store.Delete(ctx, "prompt_chains", chainID); err != nil {
		return fmt.Errorf("failed to delete chain: %w", err)
	}
	
	// Delete agent index
	if err := m.store.Delete(ctx, "prompt_chains_by_agent", chain.AgentID+":"+chainID); err != nil {
		// Log but don't fail
		fmt.Printf("warning: failed to delete agent index: %v\n", err)
	}
	
	// Delete task index if present
	if chain.TaskID != "" {
		if err := m.store.Delete(ctx, "prompt_chains_by_task", chain.TaskID+":"+chainID); err != nil {
			// Log but don't fail
			fmt.Printf("warning: failed to delete task index: %v\n", err)
		}
	}
	
	return nil
}