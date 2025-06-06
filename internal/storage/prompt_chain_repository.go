package storage

import (
	"context"
	"database/sql"
	"time"

	"github.com/guild-ventures/guild-core/pkg/gerror"
	"github.com/guild-ventures/guild-core/internal/storage/db"
)

// promptChainRepository implements PromptChainRepository using SQLite
type promptChainRepository struct {
	db      *sql.DB
	queries *db.Queries
}

// newPromptChainRepository creates a new SQLite-based prompt chain repository (private constructor)
func newPromptChainRepository(database *sql.DB) PromptChainRepository {
	return &promptChainRepository{
		db:      database,
		queries: db.New(database),
	}
}

// DefaultPromptChainRepositoryFactory creates a prompt chain repository for registry use
func DefaultPromptChainRepositoryFactory(database *sql.DB) PromptChainRepository {
	return newPromptChainRepository(database)
}

// CreateChain creates a new prompt chain
func (r *promptChainRepository) CreateChain(ctx context.Context, chain *PromptChain) error {
	if chain == nil {
		return gerror.New(gerror.ErrCodeValidation, "CreateChain: chain cannot be nil", nil)
	}

	if chain.ID == "" {
		return gerror.New(gerror.ErrCodeMissingRequired, "CreateChain: chain ID cannot be empty", nil)
	}

	if chain.AgentID == "" {
		return gerror.New(gerror.ErrCodeMissingRequired, "CreateChain: agent ID cannot be empty", nil)
	}

	// Set timestamps if not provided
	now := time.Now()
	if chain.CreatedAt.IsZero() {
		chain.CreatedAt = now
	}
	if chain.UpdatedAt.IsZero() {
		chain.UpdatedAt = now
	}

	err := r.queries.CreatePromptChain(ctx, db.CreatePromptChainParams{
		ID:        chain.ID,
		AgentID:   chain.AgentID,
		TaskID:    chain.TaskID,
		CreatedAt: &chain.CreatedAt,
		UpdatedAt: &chain.UpdatedAt,
	})
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "CreateChain: failed to create prompt chain")
	}

	return nil
}

// GetChain retrieves a prompt chain by ID
func (r *promptChainRepository) GetChain(ctx context.Context, id string) (*PromptChain, error) {
	if id == "" {
		return nil, gerror.New(gerror.ErrCodeMissingRequired, "GetChain: id cannot be empty", nil)
	}

	// Get the chain metadata
	chainRow, err := r.queries.GetPromptChain(ctx, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, gerror.New(gerror.ErrCodeNotFound, "GetChain: prompt chain not found", nil)
		}
		return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "GetChain: failed to retrieve prompt chain")
	}

	// Get all messages for this chain
	messageRows, err := r.queries.GetPromptChainMessages(ctx, id)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "GetChain: failed to retrieve prompt chain messages")
	}

	// Convert to domain objects
	chain := &PromptChain{
		ID:        chainRow.ID,
		AgentID:   chainRow.AgentID,
		TaskID:    chainRow.TaskID,
		CreatedAt: safeTimeValue(chainRow.CreatedAt),
		UpdatedAt: safeTimeValue(chainRow.UpdatedAt),
		Messages:  make([]*PromptChainMessage, 0, len(messageRows)),
	}

	for _, msgRow := range messageRows {
		message := &PromptChainMessage{
			ID:         msgRow.ID,
			ChainID:    msgRow.ChainID,
			Role:       msgRow.Role,
			Content:    msgRow.Content,
			Name:       msgRow.Name,
			Timestamp:  safeTimeValue(msgRow.Timestamp),
			TokenUsage: safeInt32Value(msgRow.TokenUsage),
		}
		chain.Messages = append(chain.Messages, message)
	}

	return chain, nil
}

// AddMessage adds a message to a prompt chain
func (r *promptChainRepository) AddMessage(ctx context.Context, chainID string, message *PromptChainMessage) error {
	if chainID == "" {
		return gerror.New(gerror.ErrCodeMissingRequired, "AddMessage: chainID cannot be empty", nil)
	}

	if message == nil {
		return gerror.New(gerror.ErrCodeValidation, "AddMessage: message cannot be nil", nil)
	}

	if message.Role == "" {
		return gerror.New(gerror.ErrCodeMissingRequired, "AddMessage: message role cannot be empty", nil)
	}

	if message.Content == "" {
		return gerror.New(gerror.ErrCodeMissingRequired, "AddMessage: message content cannot be empty", nil)
	}

	// Set timestamp if not provided
	if message.Timestamp.IsZero() {
		message.Timestamp = time.Now()
	}

	// Set chain ID
	message.ChainID = chainID

	err := r.queries.AddPromptChainMessage(ctx, db.AddPromptChainMessageParams{
		ChainID:    chainID,
		Role:       message.Role,
		Content:    message.Content,
		Name:       message.Name,
		Timestamp:  &message.Timestamp,
		TokenUsage: safeInt64Pointer(int64(message.TokenUsage)),
	})
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "AddMessage: failed to add message to prompt chain")
	}

	// Update the chain's updated_at timestamp
	now := time.Now()
	_, updateErr := r.db.ExecContext(ctx, "UPDATE prompt_chains SET updated_at = ? WHERE id = ?", &now, chainID)
	if updateErr != nil {
		return gerror.Wrap(updateErr, gerror.ErrCodeStorage, "AddMessage: failed to update chain timestamp")
	}

	return nil
}

// GetChainsByAgent retrieves all chains for a specific agent
func (r *promptChainRepository) GetChainsByAgent(ctx context.Context, agentID string) ([]*PromptChain, error) {
	if agentID == "" {
		return nil, gerror.New(gerror.ErrCodeMissingRequired, "GetChainsByAgent: agentID cannot be empty", nil)
	}

	chainRows, err := r.queries.GetPromptChainsByAgent(ctx, agentID)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "GetChainsByAgent: failed to retrieve chains")
	}

	chains := make([]*PromptChain, 0, len(chainRows))
	for _, chainRow := range chainRows {
		chain := &PromptChain{
			ID:        chainRow.ID,
			AgentID:   chainRow.AgentID,
			TaskID:    chainRow.TaskID,
			CreatedAt: safeTimeValue(chainRow.CreatedAt),
			UpdatedAt: safeTimeValue(chainRow.UpdatedAt),
		}
		chains = append(chains, chain)
	}

	return chains, nil
}

// GetChainsByTask retrieves all chains for a specific task
func (r *promptChainRepository) GetChainsByTask(ctx context.Context, taskID string) ([]*PromptChain, error) {
	if taskID == "" {
		return nil, gerror.New(gerror.ErrCodeMissingRequired, "GetChainsByTask: taskID cannot be empty", nil)
	}

	chainRows, err := r.queries.GetPromptChainsByTask(ctx, &taskID)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "GetChainsByTask: failed to retrieve chains")
	}

	chains := make([]*PromptChain, 0, len(chainRows))
	for _, chainRow := range chainRows {
		chain := &PromptChain{
			ID:        chainRow.ID,
			AgentID:   chainRow.AgentID,
			TaskID:    chainRow.TaskID,
			CreatedAt: safeTimeValue(chainRow.CreatedAt),
			UpdatedAt: safeTimeValue(chainRow.UpdatedAt),
		}
		chains = append(chains, chain)
	}

	return chains, nil
}

// DeleteChain deletes a prompt chain and all its messages
func (r *promptChainRepository) DeleteChain(ctx context.Context, id string) error {
	if id == "" {
		return gerror.New(gerror.ErrCodeMissingRequired, "DeleteChain: id cannot be empty", nil)
	}

	// Delete all messages first (will be handled by CASCADE in the schema)
	err := r.queries.DeletePromptChainMessages(ctx, id)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "DeleteChain: failed to delete prompt chain messages")
	}

	// Delete the chain
	err = r.queries.DeletePromptChain(ctx, id)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "DeleteChain: failed to delete prompt chain")
	}

	return nil
}

// Helper functions for safe type conversions between database and domain models

// safeTimeValue safely converts a *time.Time to time.Time, returning zero time if nil
func safeTimeValue(t *time.Time) time.Time {
	if t == nil {
		return time.Time{}
	}
	return *t
}

// safeStringValue safely converts a *string to string, returning empty string if nil
func safeStringValue(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// safeInt32Value safely converts a *int64 to int32, returning 0 if nil
func safeInt32Value(i *int64) int32 {
	if i == nil {
		return 0
	}
	return int32(*i)
}

// safeStringPointer safely converts a string to *string, returning nil if empty
func safeStringPointer(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// safeInt64Pointer safely converts an int64 to *int64, returning nil if zero
func safeInt64Pointer(i int64) *int64 {
	if i == 0 {
		return nil
	}
	return &i
}