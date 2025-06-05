package memory

import (
	"context"
	"time"
	"fmt"

	"github.com/guild-ventures/guild-core/pkg/gerror"
	"github.com/guild-ventures/guild-core/pkg/storage"
)

// sqliteChainManager implements ChainManager using SQLite storage
type sqliteChainManager struct {
	repo storage.PromptChainRepository
}

// NewSQLiteChainManager creates a new SQLite-based chain manager
func NewSQLiteChainManager(repo storage.PromptChainRepository) ChainManager {
	return &sqliteChainManager{
		repo: repo,
	}
}

// CreateChain creates a new prompt chain
func (m *sqliteChainManager) CreateChain(ctx context.Context, agentID, taskID string) (string, error) {
	if agentID == "" {
		return "", gerror.New(gerror.ErrCodeMissingRequired, "CreateChain: agentID cannot be empty", nil)
	}

	// Generate a unique chain ID
	chainID := fmt.Sprintf("chain_%s_%d", agentID, time.Now().UnixNano())

	chain := &storage.PromptChain{
		ID:        chainID,
		AgentID:   agentID,
		TaskID:    nil,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Set task ID if provided
	if taskID != "" {
		chain.TaskID = &taskID
	}

	err := m.repo.CreateChain(ctx, chain)
	if err != nil {
		return "", gerror.Wrap(err, gerror.ErrCodeStorage, "CreateChain: failed to create chain")
	}

	return chainID, nil
}

// GetChain retrieves a chain by ID
func (m *sqliteChainManager) GetChain(ctx context.Context, chainID string) (*PromptChain, error) {
	if chainID == "" {
		return nil, gerror.New(gerror.ErrCodeMissingRequired, "GetChain: chainID cannot be empty", nil)
	}

	storageChain, err := m.repo.GetChain(ctx, chainID)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "GetChain: failed to retrieve chain")
	}

	return m.convertStorageChainToDomain(storageChain), nil
}

// AddMessage adds a message to a chain
func (m *sqliteChainManager) AddMessage(ctx context.Context, chainID string, message Message) error {
	if chainID == "" {
		return gerror.New(gerror.ErrCodeMissingRequired, "AddMessage: chainID cannot be empty", nil)
	}

	if message.Role == "" {
		return gerror.New(gerror.ErrCodeMissingRequired, "AddMessage: message role cannot be empty", nil)
	}

	if message.Content == "" {
		return gerror.New(gerror.ErrCodeMissingRequired, "AddMessage: message content cannot be empty", nil)
	}

	storageMessage := &storage.PromptChainMessage{
		ChainID:    chainID,
		Role:       message.Role,
		Content:    message.Content,
		Name:       nil,
		Timestamp:  message.Timestamp,
		TokenUsage: int32(message.TokenUsage),
	}

	// Set name if provided
	if message.Name != "" {
		storageMessage.Name = &message.Name
	}

	// Set timestamp if not provided
	if storageMessage.Timestamp.IsZero() {
		storageMessage.Timestamp = time.Now()
	}

	err := m.repo.AddMessage(ctx, chainID, storageMessage)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "AddMessage: failed to add message")
	}

	return nil
}

// GetChainsByAgent retrieves all chains for an agent
func (m *sqliteChainManager) GetChainsByAgent(ctx context.Context, agentID string) ([]*PromptChain, error) {
	if agentID == "" {
		return nil, gerror.New(gerror.ErrCodeMissingRequired, "GetChainsByAgent: agentID cannot be empty", nil)
	}

	storageChains, err := m.repo.GetChainsByAgent(ctx, agentID)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "GetChainsByAgent: failed to retrieve chains")
	}

	chains := make([]*PromptChain, 0, len(storageChains))
	for _, storageChain := range storageChains {
		// For list operations, we don't need to load all messages
		chain := &PromptChain{
			ID:        storageChain.ID,
			AgentID:   storageChain.AgentID,
			TaskID:    "",
			CreatedAt: storageChain.CreatedAt,
			UpdatedAt: storageChain.UpdatedAt,
			Messages:  []Message{},
		}

		if storageChain.TaskID != nil {
			chain.TaskID = *storageChain.TaskID
		}

		chains = append(chains, chain)
	}

	return chains, nil
}

// GetChainsByTask retrieves all chains for a task
func (m *sqliteChainManager) GetChainsByTask(ctx context.Context, taskID string) ([]*PromptChain, error) {
	if taskID == "" {
		return nil, gerror.New(gerror.ErrCodeMissingRequired, "GetChainsByTask: taskID cannot be empty", nil)
	}

	storageChains, err := m.repo.GetChainsByTask(ctx, taskID)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "GetChainsByTask: failed to retrieve chains")
	}

	chains := make([]*PromptChain, 0, len(storageChains))
	for _, storageChain := range storageChains {
		// For list operations, we don't need to load all messages
		chain := &PromptChain{
			ID:        storageChain.ID,
			AgentID:   storageChain.AgentID,
			TaskID:    "",
			CreatedAt: storageChain.CreatedAt,
			UpdatedAt: storageChain.UpdatedAt,
			Messages:  []Message{},
		}

		if storageChain.TaskID != nil {
			chain.TaskID = *storageChain.TaskID
		}

		chains = append(chains, chain)
	}

	return chains, nil
}

// BuildContext builds a context from chains for an agent and task
func (m *sqliteChainManager) BuildContext(ctx context.Context, agentID, taskID string, maxTokens int) ([]Message, error) {
	if agentID == "" {
		return nil, gerror.New(gerror.ErrCodeMissingRequired, "BuildContext: agentID cannot be empty", nil)
	}

	// Get chains for the agent
	chains, err := m.GetChainsByAgent(ctx, agentID)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "BuildContext: failed to retrieve chains")
	}

	// Filter by task if specified
	if taskID != "" {
		filteredChains := make([]*PromptChain, 0)
		for _, chain := range chains {
			if chain.TaskID == taskID {
				filteredChains = append(filteredChains, chain)
			}
		}
		chains = filteredChains
	}

	// Collect messages from all chains
	allMessages := make([]Message, 0)
	totalTokens := 0

	for _, chain := range chains {
		// Load full chain with messages
		fullChain, err := m.GetChain(ctx, chain.ID)
		if err != nil {
			// Log error but continue with other chains
			continue
		}

		// Add messages in reverse order (newest first) until token limit
		for i := len(fullChain.Messages) - 1; i >= 0; i-- {
			message := fullChain.Messages[i]
			
			// Check token limit
			if maxTokens > 0 && totalTokens+message.TokenUsage > maxTokens {
				break
			}

			allMessages = append([]Message{message}, allMessages...)
			totalTokens += message.TokenUsage
		}

		// Break if we've hit the token limit
		if maxTokens > 0 && totalTokens >= maxTokens {
			break
		}
	}

	return allMessages, nil
}

// DeleteChain deletes a chain
func (m *sqliteChainManager) DeleteChain(ctx context.Context, chainID string) error {
	if chainID == "" {
		return gerror.New(gerror.ErrCodeMissingRequired, "DeleteChain: chainID cannot be empty", nil)
	}

	err := m.repo.DeleteChain(ctx, chainID)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "DeleteChain: failed to delete chain")
	}

	return nil
}

// convertStorageChainToDomain converts a storage PromptChain to a domain PromptChain
func (m *sqliteChainManager) convertStorageChainToDomain(storageChain *storage.PromptChain) *PromptChain {
	chain := &PromptChain{
		ID:        storageChain.ID,
		AgentID:   storageChain.AgentID,
		TaskID:    "",
		CreatedAt: storageChain.CreatedAt,
		UpdatedAt: storageChain.UpdatedAt,
		Messages:  make([]Message, 0, len(storageChain.Messages)),
	}

	if storageChain.TaskID != nil {
		chain.TaskID = *storageChain.TaskID
	}

	// Convert messages
	for _, storageMsg := range storageChain.Messages {
		message := Message{
			Role:       storageMsg.Role,
			Content:    storageMsg.Content,
			Name:       "",
			Timestamp:  storageMsg.Timestamp,
			TokenUsage: int(storageMsg.TokenUsage),
		}

		if storageMsg.Name != nil {
			message.Name = *storageMsg.Name
		}

		chain.Messages = append(chain.Messages, message)
	}

	return chain
}