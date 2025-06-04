package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/guild-ventures/guild-core/pkg/memory"
)

// SQLiteStoreAdapter implements memory.Store interface using SQLite repositories
// This bridges the gap between the existing memory.Store interface and our new repository pattern
type SQLiteStoreAdapter struct {
	registry StorageRegistry
}

// NewSQLiteStoreAdapter creates a new SQLite store adapter
// Following Guild's constructor pattern
func NewSQLiteStoreAdapter(registry StorageRegistry) memory.Store {
	return &SQLiteStoreAdapter{
		registry: registry,
	}
}

// Put stores a value with the given key, routing to appropriate repository based on bucket
func (s *SQLiteStoreAdapter) Put(ctx context.Context, bucket, key string, value []byte) error {
	switch bucket {
	case "tasks":
		return s.putTask(ctx, key, value)
	case "campaigns": 
		return s.putCampaign(ctx, key, value)
	case "commissions":
		return s.putCommission(ctx, key, value)
	case "agents":
		return s.putAgent(ctx, key, value)
	default:
		// For unknown buckets, we might need a generic key-value store
		// For now, return an error to identify what needs to be migrated
		return fmt.Errorf("unsupported bucket for SQLite storage: %s", bucket)
	}
}

// Get retrieves a value by key, routing to appropriate repository based on bucket
func (s *SQLiteStoreAdapter) Get(ctx context.Context, bucket, key string) ([]byte, error) {
	switch bucket {
	case "tasks":
		return s.getTask(ctx, key)
	case "campaigns":
		return s.getCampaign(ctx, key)
	case "commissions":
		return s.getCommission(ctx, key)
	case "agents":
		return s.getAgent(ctx, key)
	default:
		return nil, fmt.Errorf("unsupported bucket for SQLite storage: %s", bucket)
	}
}

// Delete removes a value by key, routing to appropriate repository based on bucket
func (s *SQLiteStoreAdapter) Delete(ctx context.Context, bucket, key string) error {
	switch bucket {
	case "tasks":
		return s.registry.GetTaskRepository().DeleteTask(ctx, key)
	case "campaigns":
		return s.registry.GetCampaignRepository().DeleteCampaign(ctx, key)
	case "commissions":
		return s.registry.GetCommissionRepository().DeleteCommission(ctx, key)
	case "agents":
		return s.registry.GetAgentRepository().DeleteAgent(ctx, key)
	default:
		return fmt.Errorf("unsupported bucket for SQLite storage: %s", bucket)
	}
}

// List returns all keys in a bucket
func (s *SQLiteStoreAdapter) List(ctx context.Context, bucket string) ([]string, error) {
	switch bucket {
	case "tasks":
		tasks, err := s.registry.GetTaskRepository().ListTasks(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list tasks: %w", err)
		}
		keys := make([]string, len(tasks))
		for i, task := range tasks {
			keys[i] = task.ID
		}
		return keys, nil
	case "campaigns":
		campaigns, err := s.registry.GetCampaignRepository().ListCampaigns(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list campaigns: %w", err)
		}
		keys := make([]string, len(campaigns))
		for i, campaign := range campaigns {
			keys[i] = campaign.ID
		}
		return keys, nil
	case "agents":
		agents, err := s.registry.GetAgentRepository().ListAgents(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list agents: %w", err)
		}
		keys := make([]string, len(agents))
		for i, agent := range agents {
			keys[i] = agent.ID
		}
		return keys, nil
	default:
		return nil, fmt.Errorf("unsupported bucket for SQLite storage: %s", bucket)
	}
}

// ListKeys returns keys with the given prefix in a bucket
func (s *SQLiteStoreAdapter) ListKeys(ctx context.Context, bucket, prefix string) ([]string, error) {
	allKeys, err := s.List(ctx, bucket)
	if err != nil {
		return nil, err
	}
	
	var filteredKeys []string
	for _, key := range allKeys {
		if strings.HasPrefix(key, prefix) {
			filteredKeys = append(filteredKeys, key)
		}
	}
	
	return filteredKeys, nil
}

// Close closes the store (no-op for SQLite adapter since database lifecycle is managed elsewhere)
func (s *SQLiteStoreAdapter) Close() error {
	// No-op: Database connection lifecycle is managed by the registry/database layer
	return nil
}

// Helper methods for each entity type

func (s *SQLiteStoreAdapter) putTask(ctx context.Context, key string, value []byte) error {
	var task Task
	if err := json.Unmarshal(value, &task); err != nil {
		return fmt.Errorf("failed to unmarshal task: %w", err)
	}
	
	// Set ID from key if not set
	if task.ID == "" {
		task.ID = key
	}
	
	// Try to get existing task to determine if this is create or update
	existing, err := s.registry.GetTaskRepository().GetTask(ctx, key)
	if err != nil {
		// Task doesn't exist, create it
		return s.registry.GetTaskRepository().CreateTask(ctx, &task)
	}
	
	// Task exists, update it
	task.CreatedAt = existing.CreatedAt // Preserve original creation time
	return s.registry.GetTaskRepository().UpdateTask(ctx, &task)
}

func (s *SQLiteStoreAdapter) getTask(ctx context.Context, key string) ([]byte, error) {
	task, err := s.registry.GetTaskRepository().GetTask(ctx, key)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return nil, memory.ErrNotFound
		}
		return nil, fmt.Errorf("failed to get task: %w", err)
	}
	
	return json.Marshal(task)
}

func (s *SQLiteStoreAdapter) putCampaign(ctx context.Context, key string, value []byte) error {
	var campaign Campaign
	if err := json.Unmarshal(value, &campaign); err != nil {
		return fmt.Errorf("failed to unmarshal campaign: %w", err)
	}
	
	// Set ID from key if not set
	if campaign.ID == "" {
		campaign.ID = key
	}
	
	// Try to get existing campaign to determine if this is create or update
	_, err := s.registry.GetCampaignRepository().GetCampaign(ctx, key)
	if err != nil {
		// Campaign doesn't exist, create it
		if campaign.CreatedAt.IsZero() {
			campaign.CreatedAt = time.Now()
		}
		if campaign.UpdatedAt.IsZero() {
			campaign.UpdatedAt = time.Now()
		}
		return s.registry.GetCampaignRepository().CreateCampaign(ctx, &campaign)
	}
	
	// Campaign exists, update status only (since repository only supports status updates)
	return s.registry.GetCampaignRepository().UpdateCampaignStatus(ctx, key, campaign.Status)
}

func (s *SQLiteStoreAdapter) getCampaign(ctx context.Context, key string) ([]byte, error) {
	campaign, err := s.registry.GetCampaignRepository().GetCampaign(ctx, key)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return nil, memory.ErrNotFound
		}
		return nil, fmt.Errorf("failed to get campaign: %w", err)
	}
	
	return json.Marshal(campaign)
}

func (s *SQLiteStoreAdapter) putCommission(ctx context.Context, key string, value []byte) error {
	var commission Commission
	if err := json.Unmarshal(value, &commission); err != nil {
		return fmt.Errorf("failed to unmarshal commission: %w", err)
	}
	
	// Set ID from key if not set
	if commission.ID == "" {
		commission.ID = key
	}
	
	// Try to get existing commission to determine if this is create or update
	_, err := s.registry.GetCommissionRepository().GetCommission(ctx, key)
	if err != nil {
		// Commission doesn't exist, create it
		if commission.CreatedAt.IsZero() {
			commission.CreatedAt = time.Now()
		}
		return s.registry.GetCommissionRepository().CreateCommission(ctx, &commission)
	}
	
	// Commission exists, update status only (since repository only supports status updates)
	return s.registry.GetCommissionRepository().UpdateCommissionStatus(ctx, key, commission.Status)
}

func (s *SQLiteStoreAdapter) getCommission(ctx context.Context, key string) ([]byte, error) {
	commission, err := s.registry.GetCommissionRepository().GetCommission(ctx, key)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return nil, memory.ErrNotFound
		}
		return nil, fmt.Errorf("failed to get commission: %w", err)
	}
	
	return json.Marshal(commission)
}

func (s *SQLiteStoreAdapter) putAgent(ctx context.Context, key string, value []byte) error {
	var agent Agent
	if err := json.Unmarshal(value, &agent); err != nil {
		return fmt.Errorf("failed to unmarshal agent: %w", err)
	}
	
	// Set ID from key if not set
	if agent.ID == "" {
		agent.ID = key
	}
	
	// Try to get existing agent to determine if this is create or update
	existing, err := s.registry.GetAgentRepository().GetAgent(ctx, key)
	if err != nil {
		// Agent doesn't exist, create it
		if agent.CreatedAt.IsZero() {
			agent.CreatedAt = time.Now()
		}
		return s.registry.GetAgentRepository().CreateAgent(ctx, &agent)
	}
	
	// Agent exists, update it
	agent.CreatedAt = existing.CreatedAt // Preserve original creation time
	return s.registry.GetAgentRepository().UpdateAgent(ctx, &agent)
}

func (s *SQLiteStoreAdapter) getAgent(ctx context.Context, key string) ([]byte, error) {
	agent, err := s.registry.GetAgentRepository().GetAgent(ctx, key)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return nil, memory.ErrNotFound
		}
		return nil, fmt.Errorf("failed to get agent: %w", err)
	}
	
	return json.Marshal(agent)
}