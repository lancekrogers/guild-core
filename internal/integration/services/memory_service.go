// Package services provides service wrappers for Guild components
package services

import (
	"context"
	"sync"
	"time"

	"github.com/lancekrogers/guild-core/pkg/events"
	"github.com/lancekrogers/guild-core/pkg/gerror"
	"github.com/lancekrogers/guild-core/pkg/memory"
	"github.com/lancekrogers/guild-core/pkg/memory/vector"
	"github.com/lancekrogers/guild-core/pkg/observability"
	"github.com/lancekrogers/guild-core/pkg/registry"
)

// MemoryService wraps the memory system to integrate with the service framework
type MemoryService struct {
	store        memory.Store
	chainManager memory.ChainManager
	vectorStore  vector.VectorStore
	registry     registry.ComponentRegistry
	eventBus     events.EventBus
	logger       observability.Logger

	// Service state
	started bool
	mu      sync.RWMutex

	// Metrics
	storeOps    uint64
	retrieveOps uint64
	searchOps   uint64
	chainOps    uint64
	avgLatency  time.Duration
	lastError   error
}

// MemoryServiceConfig configures the memory service
type MemoryServiceConfig struct {
	// Store configuration
	DatabasePath   string
	MaxConnections int
	ConnectTimeout time.Duration

	// Vector store configuration
	VectorStoreName string
	VectorDimension int

	// Performance tuning
	CacheSize     int
	FlushInterval time.Duration
}

// DefaultMemoryServiceConfig returns default configuration
func DefaultMemoryServiceConfig() MemoryServiceConfig {
	return MemoryServiceConfig{
		DatabasePath:    ".guild/memory.db",
		MaxConnections:  100,
		ConnectTimeout:  5 * time.Second,
		VectorStoreName: "milvus",
		VectorDimension: 384,
		CacheSize:       1000,
		FlushInterval:   100 * time.Millisecond,
	}
}

// NewMemoryService creates a new memory service wrapper
func NewMemoryService(
	registry registry.ComponentRegistry,
	eventBus events.EventBus,
	logger observability.Logger,
	config MemoryServiceConfig,
) (*MemoryService, error) {
	if registry == nil {
		return nil, gerror.New(gerror.ErrCodeValidation, "registry cannot be nil", nil).
			WithComponent("MemoryService")
	}
	if eventBus == nil {
		return nil, gerror.New(gerror.ErrCodeValidation, "event bus cannot be nil", nil).
			WithComponent("MemoryService")
	}
	if logger == nil {
		return nil, gerror.New(gerror.ErrCodeValidation, "logger cannot be nil", nil).
			WithComponent("MemoryService")
	}

	return &MemoryService{
		registry: registry,
		eventBus: eventBus,
		logger:   logger,
	}, nil
}

// Name returns the service name
func (s *MemoryService) Name() string {
	return "memory-service"
}

// Start initializes and starts the service
func (s *MemoryService) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.started {
		return gerror.New(gerror.ErrCodeAlreadyExists, "service already started", nil).
			WithComponent("MemoryService")
	}

	// Get memory components from registry
	memoryReg := s.registry.Memory()
	if memoryReg == nil {
		return gerror.New(gerror.ErrCodeInternal, "memory registry not available", nil).
			WithComponent("MemoryService")
	}

	// Get default memory store
	store, err := memoryReg.GetDefaultMemoryStore()
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to get memory store").
			WithComponent("MemoryService")
	}
	s.store = store

	// Get default chain manager
	chainManager, err := memoryReg.GetDefaultChainManager()
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to get chain manager").
			WithComponent("MemoryService")
	}
	s.chainManager = chainManager

	// Get default vector store
	vectorStore, err := memoryReg.GetDefaultVectorStore()
	if err != nil {
		// Vector store is optional, log warning but don't fail
		s.logger.WarnContext(ctx, "Vector store not available", "error", err)
	} else {
		s.vectorStore = vectorStore
	}

	s.started = true

	// Emit service started event
	if err := s.eventBus.Publish(ctx, events.NewBaseEvent(
		"memory-service-started",
		"service.started",
		"memory",
		map[string]interface{}{
			"has_vector_store": s.vectorStore != nil,
		},
	)); err != nil {
		s.logger.WarnContext(ctx, "Failed to publish service started event", "error", err)
	}

	s.logger.InfoContext(ctx, "Memory service started",
		"has_vector_store", s.vectorStore != nil)

	return nil
}

// Stop gracefully shuts down the service
func (s *MemoryService) Stop(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.started {
		return gerror.New(gerror.ErrCodeValidation, "service not started", nil).
			WithComponent("MemoryService")
	}

	// Close memory store
	if s.store != nil {
		if err := s.store.Close(); err != nil {
			s.logger.ErrorContext(ctx, "Failed to close memory store", "error", err)
		}
	}

	// Close vector store if available
	if s.vectorStore != nil {
		if err := s.vectorStore.Close(); err != nil {
			s.logger.ErrorContext(ctx, "Failed to close vector store", "error", err)
		}
	}

	// Emit service stopped event
	if err := s.eventBus.Publish(ctx, events.NewBaseEvent(
		"memory-service-stopped",
		"service.stopped",
		"memory",
		map[string]interface{}{
			"store_ops":    s.storeOps,
			"retrieve_ops": s.retrieveOps,
			"search_ops":   s.searchOps,
			"chain_ops":    s.chainOps,
			"avg_latency":  s.avgLatency.Milliseconds(),
		},
	)); err != nil {
		s.logger.WarnContext(ctx, "Failed to publish service stopped event", "error", err)
	}

	s.started = false
	s.store = nil
	s.chainManager = nil
	s.vectorStore = nil

	s.logger.InfoContext(ctx, "Memory service stopped",
		"total_operations", s.storeOps+s.retrieveOps+s.searchOps+s.chainOps)

	return nil
}

// Health checks if the service is healthy
func (s *MemoryService) Health(ctx context.Context) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.started {
		return gerror.New(gerror.ErrCodeResourceExhausted, "service not started", nil).
			WithComponent("MemoryService")
	}

	// Check memory store is available
	if s.store == nil {
		return gerror.New(gerror.ErrCodeInternal, "memory store not available", nil).
			WithComponent("MemoryService")
	}

	// Try a simple operation to verify connectivity
	testCtx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()

	if _, err := s.store.List(testCtx, "health-check"); err != nil {
		s.lastError = err
		return gerror.Wrap(err, gerror.ErrCodeInternal, "memory store health check failed").
			WithComponent("MemoryService")
	}

	// Check vector store if available
	if s.vectorStore != nil {
		// Try a simple query to verify vector store is working
		if _, err := s.vectorStore.QueryEmbeddings(testCtx, "health check", 1); err != nil {
			// Vector store issues are warnings, not failures
			s.logger.WarnContext(ctx, "Vector store unhealthy", "error", err)
		}
	}

	return nil
}

// Ready checks if the service is ready to handle requests
func (s *MemoryService) Ready(ctx context.Context) error {
	if err := s.Health(ctx); err != nil {
		return err
	}

	// Additional readiness checks
	// For memory service, healthy == ready

	return nil
}

// Store stores a value with event emission
func (s *MemoryService) Store(ctx context.Context, bucket, key string, value []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.started {
		return gerror.New(gerror.ErrCodeValidation, "service not started", nil).
			WithComponent("MemoryService")
	}

	start := time.Now()

	// Store in memory
	err := s.store.Put(ctx, bucket, key, value)

	duration := time.Since(start)
	s.updateMetrics(duration)

	// Emit event
	event := events.NewBaseEvent(
		"memory-store-"+key,
		"memory.stored",
		"memory",
		map[string]interface{}{
			"bucket":   bucket,
			"key":      key,
			"size":     len(value),
			"success":  err == nil,
			"duration": duration.Milliseconds(),
			"error":    errorString(err),
		},
	)

	if pubErr := s.eventBus.Publish(ctx, event); pubErr != nil {
		s.logger.WarnContext(ctx, "Failed to publish store event", "error", pubErr)
	}

	s.storeOps++

	if err != nil {
		s.lastError = err
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to store value").
			WithComponent("MemoryService").
			WithDetails("bucket", bucket).
			WithDetails("key", key)
	}

	return nil
}

// Retrieve retrieves a value with event emission
func (s *MemoryService) Retrieve(ctx context.Context, bucket, key string) ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.started {
		return nil, gerror.New(gerror.ErrCodeValidation, "service not started", nil).
			WithComponent("MemoryService")
	}

	start := time.Now()

	// Retrieve from memory
	value, err := s.store.Get(ctx, bucket, key)

	duration := time.Since(start)
	s.updateMetrics(duration)

	// Emit event
	event := events.NewBaseEvent(
		"memory-retrieve-"+key,
		"memory.retrieved",
		"memory",
		map[string]interface{}{
			"bucket":   bucket,
			"key":      key,
			"found":    err == nil,
			"size":     len(value),
			"duration": duration.Milliseconds(),
			"error":    errorString(err),
		},
	)

	if pubErr := s.eventBus.Publish(ctx, event); pubErr != nil {
		s.logger.WarnContext(ctx, "Failed to publish retrieve event", "error", pubErr)
	}

	s.retrieveOps++

	if err != nil {
		if err == memory.ErrNotFound {
			return nil, err // Don't wrap ErrNotFound
		}
		s.lastError = err
		return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to retrieve value").
			WithComponent("MemoryService").
			WithDetails("bucket", bucket).
			WithDetails("key", key)
	}

	return value, nil
}

// Search performs vector search if available
func (s *MemoryService) Search(ctx context.Context, query string, limit int) ([]vector.EmbeddingMatch, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.started {
		return nil, gerror.New(gerror.ErrCodeValidation, "service not started", nil).
			WithComponent("MemoryService")
	}

	if s.vectorStore == nil {
		return nil, gerror.New(gerror.ErrCodeNotFound, "vector store not available", nil).
			WithComponent("MemoryService")
	}

	start := time.Now()

	// Perform search
	results, err := s.vectorStore.QueryEmbeddings(ctx, query, limit)

	duration := time.Since(start)
	s.updateMetrics(duration)

	// Emit search event
	event := events.NewBaseEvent(
		"memory-search",
		"memory.searched",
		"memory",
		map[string]interface{}{
			"query":        query,
			"limit":        limit,
			"result_count": len(results),
			"duration":     duration.Milliseconds(),
			"success":      err == nil,
			"error":        errorString(err),
		},
	)

	if pubErr := s.eventBus.Publish(ctx, event); pubErr != nil {
		s.logger.WarnContext(ctx, "Failed to publish search event", "error", pubErr)
	}

	s.searchOps++

	if err != nil {
		s.lastError = err
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "search failed").
			WithComponent("MemoryService")
	}

	return results, nil
}

// CreateChain creates a new prompt chain
func (s *MemoryService) CreateChain(ctx context.Context, agentID, taskID string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.started {
		return "", gerror.New(gerror.ErrCodeValidation, "service not started", nil).
			WithComponent("MemoryService")
	}

	chainID, err := s.chainManager.CreateChain(ctx, agentID, taskID)
	if err != nil {
		s.lastError = err
		return "", gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create chain").
			WithComponent("MemoryService")
	}

	// Emit event
	event := events.NewBaseEvent(
		chainID,
		"memory.chain.created",
		"memory",
		map[string]interface{}{
			"chain_id": chainID,
			"agent_id": agentID,
			"task_id":  taskID,
		},
	)

	if pubErr := s.eventBus.Publish(ctx, event); pubErr != nil {
		s.logger.WarnContext(ctx, "Failed to publish chain created event", "error", pubErr)
	}

	s.chainOps++

	return chainID, nil
}

// GetChainManager returns the chain manager (for compatibility)
func (s *MemoryService) GetChainManager() memory.ChainManager {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.chainManager
}

// GetStore returns the memory store (for compatibility)
func (s *MemoryService) GetStore() memory.Store {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.store
}

// GetVectorStore returns the vector store (for compatibility)
func (s *MemoryService) GetVectorStore() vector.VectorStore {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.vectorStore
}

// updateMetrics updates service metrics
func (s *MemoryService) updateMetrics(duration time.Duration) {
	// Simple moving average for latency
	if s.avgLatency == 0 {
		s.avgLatency = duration
	} else {
		s.avgLatency = (s.avgLatency*9 + duration) / 10
	}
}

// errorString returns error string or empty if nil
func errorString(err error) string {
	if err != nil {
		return err.Error()
	}
	return ""
}
