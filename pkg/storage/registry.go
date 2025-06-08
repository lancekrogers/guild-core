package storage

import (
	"sync"
)

// DefaultStorageRegistry implements StorageRegistry following Guild's registry pattern
type DefaultStorageRegistry struct {
	taskRepo        TaskRepository
	campaignRepo    CampaignRepository
	commissionRepo  CommissionRepository
	boardRepo       BoardRepository
	agentRepo       AgentRepository
	promptChainRepo PromptChainRepository
	memoryStore     interface{} // Can be memory.Store or MemoryStoreAdapter
	mu              sync.RWMutex
}

// newStorageRegistry creates a new storage registry (private constructor)
// Following Guild's constructor pattern
func newStorageRegistry() StorageRegistry {
	return &DefaultStorageRegistry{}
}

// DefaultStorageRegistryFactory creates a storage registry for registry use
func DefaultStorageRegistryFactory() StorageRegistry {
	return newStorageRegistry()
}

// RegisterTaskRepository registers a task repository
func (r *DefaultStorageRegistry) RegisterTaskRepository(repo TaskRepository) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.taskRepo = repo
}

// RegisterCampaignRepository registers a campaign repository
func (r *DefaultStorageRegistry) RegisterCampaignRepository(repo CampaignRepository) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.campaignRepo = repo
}

// RegisterCommissionRepository registers a commission repository
func (r *DefaultStorageRegistry) RegisterCommissionRepository(repo CommissionRepository) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.commissionRepo = repo
}

// RegisterBoardRepository registers a board repository
func (r *DefaultStorageRegistry) RegisterBoardRepository(repo BoardRepository) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.boardRepo = repo
}

// RegisterAgentRepository registers an agent repository
func (r *DefaultStorageRegistry) RegisterAgentRepository(repo AgentRepository) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.agentRepo = repo
}

// RegisterPromptChainRepository registers a prompt chain repository
func (r *DefaultStorageRegistry) RegisterPromptChainRepository(repo PromptChainRepository) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.promptChainRepo = repo
}

// GetTaskRepository returns the registered task repository
func (r *DefaultStorageRegistry) GetTaskRepository() TaskRepository {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.taskRepo
}

// GetCampaignRepository returns the registered campaign repository
func (r *DefaultStorageRegistry) GetCampaignRepository() CampaignRepository {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.campaignRepo
}

// GetCommissionRepository returns the registered commission repository
func (r *DefaultStorageRegistry) GetCommissionRepository() CommissionRepository {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.commissionRepo
}

// GetBoardRepository returns the registered board repository
func (r *DefaultStorageRegistry) GetBoardRepository() BoardRepository {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.boardRepo
}

// GetAgentRepository returns the registered agent repository
func (r *DefaultStorageRegistry) GetAgentRepository() AgentRepository {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.agentRepo
}

// GetPromptChainRepository returns the registered prompt chain repository
func (r *DefaultStorageRegistry) GetPromptChainRepository() PromptChainRepository {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.promptChainRepo
}

// RegisterMemoryStore registers a memory store
func (r *DefaultStorageRegistry) RegisterMemoryStore(store interface{}) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.memoryStore = store
}

// GetMemoryStore returns the registered memory store
func (r *DefaultStorageRegistry) GetMemoryStore() interface{} {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.memoryStore
}
