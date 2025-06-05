package registry

import (
	"sync"
)

// DefaultStorageRegistry implements StorageRegistry for the registry package
// This is a placeholder implementation to avoid circular imports
type DefaultStorageRegistry struct {
	taskRepo       TaskRepository
	campaignRepo   CampaignRepository
	commissionRepo CommissionRepository
	agentRepo      AgentRepository
	memoryStore    MemoryStore
	mu             sync.RWMutex
}

// NewStorageRegistry creates a new storage registry
func NewStorageRegistry() StorageRegistry {
	return &DefaultStorageRegistry{}
}

// RegisterTaskRepository registers a task repository implementation
func (r *DefaultStorageRegistry) RegisterTaskRepository(repo TaskRepository) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.taskRepo = repo
	return nil
}

// GetTaskRepository retrieves the registered task repository
func (r *DefaultStorageRegistry) GetTaskRepository() TaskRepository {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.taskRepo
}

// RegisterCampaignRepository registers a campaign repository implementation
func (r *DefaultStorageRegistry) RegisterCampaignRepository(repo CampaignRepository) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.campaignRepo = repo
	return nil
}

// GetCampaignRepository retrieves the registered campaign repository
func (r *DefaultStorageRegistry) GetCampaignRepository() CampaignRepository {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.campaignRepo
}

// RegisterCommissionRepository registers a commission repository implementation
func (r *DefaultStorageRegistry) RegisterCommissionRepository(repo CommissionRepository) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.commissionRepo = repo
	return nil
}

// GetCommissionRepository retrieves the registered commission repository
func (r *DefaultStorageRegistry) GetCommissionRepository() CommissionRepository {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.commissionRepo
}

// RegisterAgentRepository registers an agent repository implementation
func (r *DefaultStorageRegistry) RegisterAgentRepository(repo AgentRepository) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.agentRepo = repo
	return nil
}

// GetAgentRepository retrieves the registered agent repository
func (r *DefaultStorageRegistry) GetAgentRepository() AgentRepository {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.agentRepo
}

// GetMemoryStore returns the configured memory store adapter
func (r *DefaultStorageRegistry) GetMemoryStore() MemoryStore {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.memoryStore
}

// SetMemoryStore sets the memory store adapter (used internally by the registry)
func (r *DefaultStorageRegistry) SetMemoryStore(store MemoryStore) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.memoryStore = store
}

// Kanban repository methods - placeholder implementations for the default registry
// These return nil since the default registry doesn't support kanban operations

func (r *DefaultStorageRegistry) GetBoardRepository() KanbanBoardRepository {
	return nil
}

func (r *DefaultStorageRegistry) GetKanbanTaskRepository() KanbanTaskRepository {
	return nil
}

func (r *DefaultStorageRegistry) GetKanbanCampaignRepository() KanbanCampaignRepository {
	return nil
}

func (r *DefaultStorageRegistry) GetKanbanCommissionRepository() KanbanCommissionRepository {
	return nil
}