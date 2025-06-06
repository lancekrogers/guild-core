package registry

import (
	"sync"

	"github.com/guild-ventures/guild-core/pkg/gerror"
	"github.com/guild-ventures/guild-core/pkg/memory"
	"github.com/guild-ventures/guild-core/pkg/memory/vector"
)

// DefaultMemoryRegistry implements the MemoryRegistry interface
type DefaultMemoryRegistry struct {
	memoryStores       map[string]memory.Store
	vectorStores       map[string]vector.VectorStore
	chainManagers      map[string]memory.ChainManager
	defaultMemoryStore string
	defaultVectorStore string
	defaultChainManager string
	mu                 sync.RWMutex
}

// NewMemoryRegistry creates a new memory registry
func NewMemoryRegistry() MemoryRegistry {
	return &DefaultMemoryRegistry{
		memoryStores:  make(map[string]memory.Store),
		vectorStores:  make(map[string]vector.VectorStore),
		chainManagers: make(map[string]memory.ChainManager),
	}
}

// RegisterMemoryStore registers a memory store implementation
func (r *DefaultMemoryRegistry) RegisterMemoryStore(name string, store MemoryStore) error {
	if name == "" {
		return gerror.New(gerror.ErrCodeInvalidInput, "memory store name cannot be empty", nil).
			WithComponent("registry").
			WithOperation("RegisterMemoryStore")
	}
	if store == nil {
		return gerror.New(gerror.ErrCodeInvalidInput, "memory store cannot be nil", nil).
			WithComponent("registry").
			WithOperation("RegisterMemoryStore")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.memoryStores[name]; exists {
		return gerror.Newf(gerror.ErrCodeAlreadyExists, "memory store '%s' already registered", name).
			WithComponent("registry").
			WithOperation("RegisterMemoryStore").
			WithDetails("storeName", name)
	}

	// Convert the registry MemoryStore interface to the actual memory.Store interface
	memStore, ok := store.(memory.Store)
	if !ok {
		return gerror.New(gerror.ErrCodeInvalidFormat, "memory store does not implement the expected memory.Store interface", nil).
			WithComponent("registry").
			WithOperation("RegisterMemoryStore")
	}

	r.memoryStores[name] = memStore
	return nil
}

// GetMemoryStore retrieves a memory store by name
func (r *DefaultMemoryRegistry) GetMemoryStore(name string) (MemoryStore, error) {
	r.mu.RLock()
	store, exists := r.memoryStores[name]
	r.mu.RUnlock()

	if !exists {
		return nil, gerror.Newf(gerror.ErrCodeNotFound, "memory store '%s' not found", name).
			WithComponent("registry").
			WithOperation("GetMemoryStore").
			WithDetails("storeName", name)
	}

	return store, nil
}

// RegisterVectorStore registers a vector store implementation
func (r *DefaultMemoryRegistry) RegisterVectorStore(name string, store VectorStore) error {
	if name == "" {
		return gerror.New(gerror.ErrCodeInvalidInput, "vector store name cannot be empty", nil).
			WithComponent("registry").
			WithOperation("RegisterVectorStore")
	}
	if store == nil {
		return gerror.New(gerror.ErrCodeInvalidInput, "vector store cannot be nil", nil).
			WithComponent("registry").
			WithOperation("RegisterVectorStore")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.vectorStores[name]; exists {
		return gerror.Newf(gerror.ErrCodeAlreadyExists, "vector store '%s' already registered", name).
			WithComponent("registry").
			WithOperation("RegisterVectorStore").
			WithDetails("storeName", name)
	}

	// Convert the registry VectorStore interface to the actual vector.VectorStore interface
	vecStore, ok := store.(vector.VectorStore)
	if !ok {
		return gerror.New(gerror.ErrCodeInvalidFormat, "vector store does not implement the expected vector.VectorStore interface", nil).
			WithComponent("registry").
			WithOperation("RegisterVectorStore")
	}

	r.vectorStores[name] = vecStore
	return nil
}

// GetVectorStore retrieves a vector store by name
func (r *DefaultMemoryRegistry) GetVectorStore(name string) (VectorStore, error) {
	r.mu.RLock()
	store, exists := r.vectorStores[name]
	r.mu.RUnlock()

	if !exists {
		return nil, gerror.Newf(gerror.ErrCodeNotFound, "vector store '%s' not found", name).
			WithComponent("registry").
			WithOperation("GetVectorStore").
			WithDetails("storeName", name)
	}

	return store, nil
}

// GetDefaultMemoryStore returns the configured default memory store
func (r *DefaultMemoryRegistry) GetDefaultMemoryStore() (MemoryStore, error) {
	r.mu.RLock()
	defaultName := r.defaultMemoryStore
	r.mu.RUnlock()

	if defaultName == "" {
		return nil, gerror.New(gerror.ErrCodeMissingRequired, "no default memory store set", nil).
			WithComponent("registry").
			WithOperation("GetDefaultMemoryStore")
	}

	return r.GetMemoryStore(defaultName)
}

// GetDefaultVectorStore returns the configured default vector store
func (r *DefaultMemoryRegistry) GetDefaultVectorStore() (VectorStore, error) {
	r.mu.RLock()
	defaultName := r.defaultVectorStore
	r.mu.RUnlock()

	if defaultName == "" {
		return nil, gerror.New(gerror.ErrCodeMissingRequired, "no default vector store set", nil).
			WithComponent("registry").
			WithOperation("GetDefaultVectorStore")
	}

	return r.GetVectorStore(defaultName)
}

// SetDefaultMemoryStore sets the default memory store
func (r *DefaultMemoryRegistry) SetDefaultMemoryStore(name string) error {
	r.mu.RLock()
	_, exists := r.memoryStores[name]
	r.mu.RUnlock()

	if !exists {
		return gerror.Newf(gerror.ErrCodeNotFound, "memory store '%s' not registered", name).
			WithComponent("registry").
			WithOperation("SetDefaultMemoryStore").
			WithDetails("storeName", name)
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	r.defaultMemoryStore = name
	return nil
}

// SetDefaultVectorStore sets the default vector store
func (r *DefaultMemoryRegistry) SetDefaultVectorStore(name string) error {
	r.mu.RLock()
	_, exists := r.vectorStores[name]
	r.mu.RUnlock()

	if !exists {
		return gerror.Newf(gerror.ErrCodeNotFound, "vector store '%s' not registered", name).
			WithComponent("registry").
			WithOperation("SetDefaultVectorStore").
			WithDetails("storeName", name)
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	r.defaultVectorStore = name
	return nil
}

// ListMemoryStores returns all registered memory store names
func (r *DefaultMemoryRegistry) ListMemoryStores() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.memoryStores))
	for name := range r.memoryStores {
		names = append(names, name)
	}
	return names
}

// ListVectorStores returns all registered vector store names
func (r *DefaultMemoryRegistry) ListVectorStores() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.vectorStores))
	for name := range r.vectorStores {
		names = append(names, name)
	}
	return names
}

// GetMemoryStore returns the underlying memory.Store for direct access
func (r *DefaultMemoryRegistry) GetActualMemoryStore(name string) (memory.Store, error) {
	r.mu.RLock()
	store, exists := r.memoryStores[name]
	r.mu.RUnlock()

	if !exists {
		return nil, gerror.Newf(gerror.ErrCodeNotFound, "memory store '%s' not found", name).
			WithComponent("registry").
			WithOperation("GetMemoryStore").
			WithDetails("storeName", name)
	}

	return store, nil
}

// GetVectorStore returns the underlying vector.VectorStore for direct access
func (r *DefaultMemoryRegistry) GetActualVectorStore(name string) (vector.VectorStore, error) {
	r.mu.RLock()
	store, exists := r.vectorStores[name]
	r.mu.RUnlock()

	if !exists {
		return nil, gerror.Newf(gerror.ErrCodeNotFound, "vector store '%s' not found", name).
			WithComponent("registry").
			WithOperation("GetVectorStore").
			WithDetails("storeName", name)
	}

	return store, nil
}

// RegisterChainManager registers a chain manager implementation
func (r *DefaultMemoryRegistry) RegisterChainManager(name string, manager ChainManager) error {
	if name == "" {
		return gerror.New(gerror.ErrCodeInvalidInput, "chain manager name cannot be empty", nil).
			WithComponent("registry").
			WithOperation("RegisterChainManager")
	}
	if manager == nil {
		return gerror.New(gerror.ErrCodeInvalidInput, "chain manager cannot be nil", nil).
			WithComponent("registry").
			WithOperation("RegisterChainManager")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.chainManagers[name]; exists {
		return gerror.Newf(gerror.ErrCodeAlreadyExists, "chain manager '%s' already registered", name).
			WithComponent("registry").
			WithOperation("RegisterChainManager").
			WithDetails("managerName", name)
	}

	// Convert the registry ChainManager interface to the actual memory.ChainManager interface
	chainMgr, ok := manager.(memory.ChainManager)
	if !ok {
		return gerror.New(gerror.ErrCodeInvalidFormat, "chain manager does not implement the expected memory.ChainManager interface", nil).
			WithComponent("registry").
			WithOperation("RegisterChainManager")
	}

	r.chainManagers[name] = chainMgr
	return nil
}

// GetChainManager retrieves a chain manager by name
func (r *DefaultMemoryRegistry) GetChainManager(name string) (ChainManager, error) {
	r.mu.RLock()
	manager, exists := r.chainManagers[name]
	r.mu.RUnlock()

	if !exists {
		return nil, gerror.Newf(gerror.ErrCodeNotFound, "chain manager '%s' not found", name).
			WithComponent("registry").
			WithOperation("GetChainManager").
			WithDetails("managerName", name)
	}

	return manager, nil
}

// GetDefaultChainManager returns the configured default chain manager
func (r *DefaultMemoryRegistry) GetDefaultChainManager() (ChainManager, error) {
	r.mu.RLock()
	defaultName := r.defaultChainManager
	r.mu.RUnlock()

	if defaultName == "" {
		return nil, gerror.New(gerror.ErrCodeMissingRequired, "no default chain manager set", nil).
			WithComponent("registry").
			WithOperation("GetDefaultChainManager")
	}

	return r.GetChainManager(defaultName)
}

// SetDefaultChainManager sets the default chain manager
func (r *DefaultMemoryRegistry) SetDefaultChainManager(name string) error {
	r.mu.RLock()
	_, exists := r.chainManagers[name]
	r.mu.RUnlock()

	if !exists {
		return gerror.Newf(gerror.ErrCodeNotFound, "chain manager '%s' not registered", name).
			WithComponent("registry").
			WithOperation("SetDefaultChainManager").
			WithDetails("managerName", name)
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	r.defaultChainManager = name
	return nil
}

// ListChainManagers returns all registered chain manager names
func (r *DefaultMemoryRegistry) ListChainManagers() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.chainManagers))
	for name := range r.chainManagers {
		names = append(names, name)
	}
	return names
}

// GetActualChainManager returns the underlying memory.ChainManager for direct access
func (r *DefaultMemoryRegistry) GetActualChainManager(name string) (memory.ChainManager, error) {
	r.mu.RLock()
	manager, exists := r.chainManagers[name]
	r.mu.RUnlock()

	if !exists {
		return nil, gerror.Newf(gerror.ErrCodeNotFound, "chain manager '%s' not found", name).
			WithComponent("registry").
			WithOperation("GetChainManager").
			WithDetails("managerName", name)
	}

	return manager, nil
}