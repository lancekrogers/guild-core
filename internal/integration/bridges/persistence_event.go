package bridges

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/lancekrogers/guild/pkg/events"
	"github.com/lancekrogers/guild/pkg/gerror"
	"github.com/lancekrogers/guild/pkg/observability"
	"github.com/lancekrogers/guild/pkg/storage"
)

// PersistenceEventBridge connects persistence operations to the event system
type PersistenceEventBridge struct {
	eventBus events.EventBus
	storage  storage.StorageRegistry
	logger   observability.Logger

	// Configuration
	config PersistenceEventConfig

	// State
	started bool
	mu      sync.RWMutex

	// Metrics
	eventsPublished uint64
	errors          uint64
}

// PersistenceEventConfig configures the persistence-event bridge
type PersistenceEventConfig struct {
	// EmitCRUDEvents enables events for Create/Read/Update/Delete operations
	EmitCRUDEvents bool

	// EmitQueryEvents enables events for query operations
	EmitQueryEvents bool

	// EmitTransactionEvents enables events for transaction boundaries
	EmitTransactionEvents bool

	// IncludePayload includes entity data in events
	IncludePayload bool

	// SensitiveFields to exclude from event payloads
	SensitiveFields []string
}

// DefaultPersistenceEventConfig returns default configuration
func DefaultPersistenceEventConfig() PersistenceEventConfig {
	return PersistenceEventConfig{
		EmitCRUDEvents:        true,
		EmitQueryEvents:       false,
		EmitTransactionEvents: true,
		IncludePayload:        true,
		SensitiveFields:       []string{"password", "token", "secret", "key"},
	}
}

// NewPersistenceEventBridge creates a new persistence-event bridge
func NewPersistenceEventBridge(eventBus events.EventBus, storage storage.StorageRegistry, logger observability.Logger, config PersistenceEventConfig) *PersistenceEventBridge {
	return &PersistenceEventBridge{
		eventBus: eventBus,
		storage:  storage,
		logger:   logger,
		config:   config,
	}
}

// Name returns the service name
func (b *PersistenceEventBridge) Name() string {
	return "persistence-event-bridge"
}

// Start initializes and starts the bridge
func (b *PersistenceEventBridge) Start(ctx context.Context) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.started {
		return gerror.New(gerror.ErrCodeAlreadyExists, "bridge already started", nil).
			WithComponent("persistence_event_bridge")
	}

	// TODO: Wrap storage with event-emitting decorators
	// For now, we'll implement manual event emission in application code

	b.started = true
	b.logger.InfoContext(ctx, "Persistence-event bridge started",
		"emit_crud", b.config.EmitCRUDEvents,
		"emit_query", b.config.EmitQueryEvents,
		"emit_transaction", b.config.EmitTransactionEvents)

	return nil
}

// Stop gracefully shuts down the bridge
func (b *PersistenceEventBridge) Stop(ctx context.Context) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if !b.started {
		return gerror.New(gerror.ErrCodeValidation, "bridge not started", nil).
			WithComponent("persistence_event_bridge")
	}

	b.started = false
	b.logger.InfoContext(ctx, "Persistence-event bridge stopped",
		"events_published", b.eventsPublished,
		"errors", b.errors)

	return nil
}

// Health checks if the bridge is healthy
func (b *PersistenceEventBridge) Health(ctx context.Context) error {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if !b.started {
		return gerror.New(gerror.ErrCodeResourceExhausted, "bridge not started", nil).
			WithComponent("persistence_event_bridge")
	}

	// Check if storage is available
	if b.storage == nil {
		return gerror.New(gerror.ErrCodeResourceExhausted, "storage not available", nil).
			WithComponent("persistence_event_bridge")
	}

	return nil
}

// Ready checks if the bridge is ready
func (b *PersistenceEventBridge) Ready(ctx context.Context) error {
	return b.Health(ctx)
}

// EmitEntityCreated emits an event when an entity is created
func (b *PersistenceEventBridge) EmitEntityCreated(ctx context.Context, entityType string, entityID string, entity interface{}) error {
	if !b.config.EmitCRUDEvents {
		return nil
	}

	return b.emitEvent(ctx, "persistence.entity.created", PersistenceEventData{
		EntityType: entityType,
		EntityID:   entityID,
		Operation:  "create",
		Timestamp:  time.Now(),
		Payload:    b.sanitizePayload(entity),
	})
}

// EmitEntityUpdated emits an event when an entity is updated
func (b *PersistenceEventBridge) EmitEntityUpdated(ctx context.Context, entityType string, entityID string, entity interface{}, changes map[string]interface{}) error {
	if !b.config.EmitCRUDEvents {
		return nil
	}

	return b.emitEvent(ctx, "persistence.entity.updated", PersistenceEventData{
		EntityType: entityType,
		EntityID:   entityID,
		Operation:  "update",
		Timestamp:  time.Now(),
		Payload:    b.sanitizePayload(entity),
		Changes:    b.sanitizePayload(changes),
	})
}

// EmitEntityDeleted emits an event when an entity is deleted
func (b *PersistenceEventBridge) EmitEntityDeleted(ctx context.Context, entityType string, entityID string) error {
	if !b.config.EmitCRUDEvents {
		return nil
	}

	return b.emitEvent(ctx, "persistence.entity.deleted", PersistenceEventData{
		EntityType: entityType,
		EntityID:   entityID,
		Operation:  "delete",
		Timestamp:  time.Now(),
	})
}

// EmitQueryExecuted emits an event when a query is executed
func (b *PersistenceEventBridge) EmitQueryExecuted(ctx context.Context, queryType string, query interface{}, resultCount int, duration time.Duration) error {
	if !b.config.EmitQueryEvents {
		return nil
	}

	return b.emitEvent(ctx, "persistence.query.executed", QueryEventData{
		QueryType:   queryType,
		Query:       query,
		ResultCount: resultCount,
		Duration:    duration,
		Timestamp:   time.Now(),
	})
}

// EmitTransactionStarted emits an event when a transaction starts
func (b *PersistenceEventBridge) EmitTransactionStarted(ctx context.Context, txID string) error {
	if !b.config.EmitTransactionEvents {
		return nil
	}

	return b.emitEvent(ctx, "persistence.transaction.started", TransactionEventData{
		TransactionID: txID,
		State:         "started",
		Timestamp:     time.Now(),
	})
}

// EmitTransactionCommitted emits an event when a transaction is committed
func (b *PersistenceEventBridge) EmitTransactionCommitted(ctx context.Context, txID string, duration time.Duration) error {
	if !b.config.EmitTransactionEvents {
		return nil
	}

	return b.emitEvent(ctx, "persistence.transaction.committed", TransactionEventData{
		TransactionID: txID,
		State:         "committed",
		Duration:      duration,
		Timestamp:     time.Now(),
	})
}

// EmitTransactionRolledBack emits an event when a transaction is rolled back
func (b *PersistenceEventBridge) EmitTransactionRolledBack(ctx context.Context, txID string, reason string, duration time.Duration) error {
	if !b.config.EmitTransactionEvents {
		return nil
	}

	return b.emitEvent(ctx, "persistence.transaction.rolledback", TransactionEventData{
		TransactionID: txID,
		State:         "rolledback",
		Reason:        reason,
		Duration:      duration,
		Timestamp:     time.Now(),
	})
}

// emitEvent publishes an event to the event bus
func (b *PersistenceEventBridge) emitEvent(ctx context.Context, eventType string, data interface{}) error {
	// Generate a unique event ID
	eventID := fmt.Sprintf("persistence-%d-%s", time.Now().UnixNano(), eventType)
	
	// Convert data to map[string]interface{}
	dataMap := make(map[string]interface{})
	dataMap["payload"] = data
	
	event := events.NewBaseEvent(eventID, eventType, "persistence-bridge", dataMap).
		WithMetadata("bridge", "persistence-event")

	if err := b.eventBus.Publish(ctx, event); err != nil {
		b.mu.Lock()
		b.errors++
		b.mu.Unlock()

		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to publish event").
			WithComponent("persistence_event_bridge").
			WithDetails("event_type", eventType)
	}

	b.mu.Lock()
	b.eventsPublished++
	b.mu.Unlock()

	return nil
}

// sanitizePayload removes sensitive fields from payloads
func (b *PersistenceEventBridge) sanitizePayload(payload interface{}) interface{} {
	if !b.config.IncludePayload || payload == nil {
		return nil
	}

	// TODO: Implement proper sanitization logic
	// For now, return payload as-is
	return payload
}

// GetMetrics returns bridge metrics
func (b *PersistenceEventBridge) GetMetrics() PersistenceEventMetrics {
	b.mu.RLock()
	defer b.mu.RUnlock()

	return PersistenceEventMetrics{
		EventsPublished: b.eventsPublished,
		Errors:          b.errors,
		Running:         b.started,
	}
}

// PersistenceEventMetrics contains bridge metrics
type PersistenceEventMetrics struct {
	EventsPublished uint64
	Errors          uint64
	Running         bool
}

// Event data structures

// PersistenceEventData represents a persistence operation event
type PersistenceEventData struct {
	EntityType string      `json:"entity_type"`
	EntityID   string      `json:"entity_id"`
	Operation  string      `json:"operation"`
	Timestamp  time.Time   `json:"timestamp"`
	Payload    interface{} `json:"payload,omitempty"`
	Changes    interface{} `json:"changes,omitempty"`
}

// QueryEventData represents a query execution event
type QueryEventData struct {
	QueryType   string        `json:"query_type"`
	Query       interface{}   `json:"query"`
	ResultCount int           `json:"result_count"`
	Duration    time.Duration `json:"duration"`
	Timestamp   time.Time     `json:"timestamp"`
}

// TransactionEventData represents a transaction event
type TransactionEventData struct {
	TransactionID string        `json:"transaction_id"`
	State         string        `json:"state"`
	Reason        string        `json:"reason,omitempty"`
	Duration      time.Duration `json:"duration,omitempty"`
	Timestamp     time.Time     `json:"timestamp"`
}
