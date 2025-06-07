package mocks

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/guild-ventures/guild-core/pkg/memory"
)

// MockChainManager implements the memory.ChainManager interface for testing
type MockChainManager struct {
	mu        sync.RWMutex
	chains    map[string]*memory.PromptChain
	chainMap  map[string][]string // Maps agentID/taskID to chainIDs
	nextID    int
	buildCtx  []memory.Message
	Error     error
}

// NewMockChainManager creates a new mock chain manager
func NewMockChainManager() *MockChainManager {
	return &MockChainManager{
		chains:   make(map[string]*memory.PromptChain),
		chainMap: make(map[string][]string),
	}
}

// WithError configures the mock to return an error
func (m *MockChainManager) WithError(err error) *MockChainManager {
	m.Error = err
	return m
}

// WithBuildContext sets messages to return from BuildContext
func (m *MockChainManager) WithBuildContext(messages []memory.Message) *MockChainManager {
	m.buildCtx = messages
	return m
}

// CreateChain implements memory.ChainManager.CreateChain
func (m *MockChainManager) CreateChain(ctx context.Context, agentID, taskID string) (string, error) {
	// Check context cancellation
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	default:
		// Continue execution
	}

	// Return error if configured
	if m.Error != nil {
		return "", m.Error
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Create a new chain ID
	m.nextID++
	chainID := fmt.Sprintf("chain-%d", m.nextID)

	// Create a new chain
	chain := &memory.PromptChain{
		ID:        chainID,
		AgentID:   agentID,
		TaskID:    taskID,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
		Messages:  []memory.Message{},
	}

	// Store the chain
	m.chains[chainID] = chain

	// Map the chain to agent/task
	key := agentID
	if taskID != "" {
		key += ":" + taskID
	}
	m.chainMap[key] = append(m.chainMap[key], chainID)

	return chainID, nil
}

// GetChain implements memory.ChainManager.GetChain
func (m *MockChainManager) GetChain(ctx context.Context, chainID string) (*memory.PromptChain, error) {
	// Check context cancellation
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
		// Continue execution
	}

	// Return error if configured
	if m.Error != nil {
		return nil, m.Error
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	chain, ok := m.chains[chainID]
	if !ok {
		return nil, memory.ErrNotFound
	}

	return chain, nil
}

// AddMessage implements memory.ChainManager.AddMessage
func (m *MockChainManager) AddMessage(ctx context.Context, chainID string, message memory.Message) error {
	// Check context cancellation
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		// Continue execution
	}

	// Return error if configured
	if m.Error != nil {
		return m.Error
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	chain, ok := m.chains[chainID]
	if !ok {
		return memory.ErrNotFound
	}

	// Set timestamp if not provided
	if message.Timestamp.IsZero() {
		message.Timestamp = time.Now().UTC()
	}

	// Add the message
	chain.Messages = append(chain.Messages, message)
	chain.UpdatedAt = time.Now().UTC()

	return nil
}

// GetChainsByAgent implements memory.ChainManager.GetChainsByAgent
func (m *MockChainManager) GetChainsByAgent(ctx context.Context, agentID string) ([]*memory.PromptChain, error) {
	// Check context cancellation
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
		// Continue execution
	}

	// Return error if configured
	if m.Error != nil {
		return nil, m.Error
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	// Get chain IDs for this agent
	chainIDs := m.chainMap[agentID]

	// Get chains
	var chains []*memory.PromptChain
	for _, chainID := range chainIDs {
		chain, ok := m.chains[chainID]
		if ok {
			chains = append(chains, chain)
		}
	}

	return chains, nil
}

// GetChainsByTask implements memory.ChainManager.GetChainsByTask
func (m *MockChainManager) GetChainsByTask(ctx context.Context, taskID string) ([]*memory.PromptChain, error) {
	// Check context cancellation
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
		// Continue execution
	}

	// Return error if configured
	if m.Error != nil {
		return nil, m.Error
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	// Get chains for this task
	var chains []*memory.PromptChain
	for _, chain := range m.chains {
		if chain.TaskID == taskID {
			chains = append(chains, chain)
		}
	}

	return chains, nil
}

// BuildContext implements memory.ChainManager.BuildContext
func (m *MockChainManager) BuildContext(ctx context.Context, agentID, taskID string, maxTokens int) ([]memory.Message, error) {
	// Check context cancellation
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
		// Continue execution
	}

	// Return error if configured
	if m.Error != nil {
		return nil, m.Error
	}

	// Return predefined context if available
	if len(m.buildCtx) > 0 {
		return m.buildCtx, nil
	}

	// Build context from scratch
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Get chains for this agent/task
	key := agentID
	if taskID != "" {
		key += ":" + taskID
	}
	chainIDs := m.chainMap[key]

	// Collect messages from all chains
	var allMessages []memory.Message
	for _, chainID := range chainIDs {
		chain, ok := m.chains[chainID]
		if ok {
			allMessages = append(allMessages, chain.Messages...)
		}
	}

	// Return all messages (in a real implementation, would respect maxTokens)
	return allMessages, nil
}

// DeleteChain implements memory.ChainManager.DeleteChain
func (m *MockChainManager) DeleteChain(ctx context.Context, chainID string) error {
	// Check context cancellation
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		// Continue execution
	}

	// Return error if configured
	if m.Error != nil {
		return m.Error
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	chain, ok := m.chains[chainID]
	if !ok {
		return memory.ErrNotFound
	}

	// Remove the chain
	delete(m.chains, chainID)

	// Remove from chainMap
	key := chain.AgentID
	if chain.TaskID != "" {
		key += ":" + chain.TaskID
	}
	chainIDs := m.chainMap[key]
	for i, id := range chainIDs {
		if id == chainID {
			// Remove this ID from the slice
			m.chainMap[key] = append(chainIDs[:i], chainIDs[i+1:]...)
			break
		}
	}

	return nil
}
