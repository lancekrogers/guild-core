// Package events provides standardized event schemas for Guild services
package events

import (
	"fmt"
	"time"

	"github.com/lancekrogers/guild/pkg/events"
)

// Event Categories
const (
	// Service lifecycle events
	EventCategoryService = "service"
	
	// Data operation events
	EventCategoryData = "data"
	
	// Task management events
	EventCategoryTask = "task"
	
	// Agent events
	EventCategoryAgent = "agent"
	
	// Commission events
	EventCategoryCommission = "commission"
	
	// UI events
	EventCategoryUI = "ui"
	
	// System events
	EventCategorySystem = "system"
	
	// gRPC events
	EventCategoryGRPC = "grpc"
	
	// Corpus events
	EventCategoryCorpus = "corpus"
)

// Standard Event Types
const (
	// Service Lifecycle Events
	EventServiceStarting = "service.starting"
	EventServiceStarted  = "service.started"
	EventServiceStopping = "service.stopping"
	EventServiceStopped  = "service.stopped"
	EventServiceHealthy  = "service.healthy"
	EventServiceUnhealthy = "service.unhealthy"
	EventServiceError    = "service.error"
	
	// Data Events
	EventDataCreated     = "data.created"
	EventDataUpdated     = "data.updated"
	EventDataDeleted     = "data.deleted"
	EventDataQueried     = "data.queried"
	EventDataSynced      = "data.synced"
	EventDataCorrupted   = "data.corrupted"
	
	// Task Events
	EventTaskCreated     = "task.created"
	EventTaskAssigned    = "task.assigned"
	EventTaskStarted     = "task.started"
	EventTaskProgress    = "task.progress"
	EventTaskCompleted   = "task.completed"
	EventTaskFailed      = "task.failed"
	EventTaskCancelled   = "task.cancelled"
	EventTaskRetried     = "task.retried"
	
	// Agent Events
	EventAgentRegistered    = "agent.registered"
	EventAgentUnregistered  = "agent.unregistered"
	EventAgentStateChanged  = "agent.state.changed"
	EventAgentTaskReceived  = "agent.task.received"
	EventAgentTaskCompleted = "agent.task.completed"
	EventAgentError         = "agent.error"
	EventAgentHealthCheck   = "agent.health.check"
	
	// Commission Events
	EventCommissionCreated   = "commission.created"
	EventCommissionPlanned   = "commission.planned"
	EventCommissionStarted   = "commission.started"
	EventCommissionProgress  = "commission.progress"
	EventCommissionCompleted = "commission.completed"
	EventCommissionFailed    = "commission.failed"
	EventCommissionCancelled = "commission.cancelled"
	
	// UI Events
	EventUIConnected     = "ui.connected"
	EventUIDisconnected  = "ui.disconnected"
	EventUIStateChanged  = "ui.state.changed"
	EventUICommandIssued = "ui.command.issued"
	EventUIError         = "ui.error"
	
	// System Events
	EventSystemNotification = "system.notification"
	EventSystemWarning      = "system.warning"
	EventSystemError        = "system.error"
	EventSystemShutdown     = "system.shutdown"
	
	// gRPC Events
	EventGRPCRequest   = "grpc.request"
	EventGRPCResponse  = "grpc.response"
	EventGRPCStream    = "grpc.stream"
	EventGRPCError     = "grpc.error"
	
	// Corpus Events
	EventCorpusScanStarted   = "corpus.scan.started"
	EventCorpusScanProgress  = "corpus.scan.progress"
	EventCorpusScanCompleted = "corpus.scan.completed"
	EventCorpusFileIndexed   = "corpus.file.indexed"
	EventCorpusIndexError    = "corpus.index.error"
	EventCorpusSearch        = "corpus.search"
)

// Event Factories

// NewServiceEvent creates a service lifecycle event
func NewServiceEvent(serviceID, eventType string, data map[string]interface{}) events.CoreEvent {
	if data == nil {
		data = make(map[string]interface{})
	}
	data["service_id"] = serviceID
	data["timestamp"] = time.Now()
	
	return events.NewBaseEvent(
		fmt.Sprintf("%s-%s-%d", serviceID, eventType, time.Now().UnixNano()),
		eventType,
		EventCategoryService,
		data,
	)
}

// NewDataEvent creates a data operation event
func NewDataEvent(entityType, entityID, operation string, data map[string]interface{}) events.CoreEvent {
	if data == nil {
		data = make(map[string]interface{})
	}
	data["entity_type"] = entityType
	data["entity_id"] = entityID
	data["operation"] = operation
	data["timestamp"] = time.Now()
	
	eventType := fmt.Sprintf("data.%s", operation)
	
	return events.NewBaseEvent(
		fmt.Sprintf("%s-%s-%s-%d", entityType, entityID, operation, time.Now().UnixNano()),
		eventType,
		EventCategoryData,
		data,
	)
}

// NewTaskEvent creates a task event
func NewTaskEvent(taskID, eventType string, data map[string]interface{}) events.CoreEvent {
	if data == nil {
		data = make(map[string]interface{})
	}
	data["task_id"] = taskID
	data["timestamp"] = time.Now()
	
	return events.NewBaseEvent(
		fmt.Sprintf("task-%s-%s-%d", taskID, eventType, time.Now().UnixNano()),
		eventType,
		EventCategoryTask,
		data,
	)
}

// NewAgentEvent creates an agent event
func NewAgentEvent(agentID, eventType string, data map[string]interface{}) events.CoreEvent {
	if data == nil {
		data = make(map[string]interface{})
	}
	data["agent_id"] = agentID
	data["timestamp"] = time.Now()
	
	return events.NewBaseEvent(
		fmt.Sprintf("agent-%s-%s-%d", agentID, eventType, time.Now().UnixNano()),
		eventType,
		EventCategoryAgent,
		data,
	)
}

// NewCommissionEvent creates a commission event
func NewCommissionEvent(commissionID, eventType string, data map[string]interface{}) events.CoreEvent {
	if data == nil {
		data = make(map[string]interface{})
	}
	data["commission_id"] = commissionID
	data["timestamp"] = time.Now()
	
	return events.NewBaseEvent(
		fmt.Sprintf("commission-%s-%s-%d", commissionID, eventType, time.Now().UnixNano()),
		eventType,
		EventCategoryCommission,
		data,
	)
}

// NewUIEvent creates a UI event
func NewUIEvent(sessionID, eventType string, data map[string]interface{}) events.CoreEvent {
	if data == nil {
		data = make(map[string]interface{})
	}
	data["session_id"] = sessionID
	data["timestamp"] = time.Now()
	
	return events.NewBaseEvent(
		fmt.Sprintf("ui-%s-%s-%d", sessionID, eventType, time.Now().UnixNano()),
		eventType,
		EventCategoryUI,
		data,
	)
}

// NewSystemEvent creates a system event
func NewSystemEvent(eventType, message string, severity string, data map[string]interface{}) events.CoreEvent {
	if data == nil {
		data = make(map[string]interface{})
	}
	data["message"] = message
	data["severity"] = severity
	data["timestamp"] = time.Now()
	
	return events.NewBaseEvent(
		fmt.Sprintf("system-%s-%d", eventType, time.Now().UnixNano()),
		eventType,
		EventCategorySystem,
		data,
	)
}

// Event Data Schemas

// ServiceEventData represents common service event data
type ServiceEventData struct {
	ServiceID   string                 `json:"service_id"`
	ServiceName string                 `json:"service_name"`
	Status      string                 `json:"status"`
	Timestamp   time.Time              `json:"timestamp"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// TaskEventData represents common task event data
type TaskEventData struct {
	TaskID      string                 `json:"task_id"`
	TaskType    string                 `json:"task_type"`
	Status      string                 `json:"status"`
	Progress    float64                `json:"progress,omitempty"`
	AssignedTo  string                 `json:"assigned_to,omitempty"`
	Error       string                 `json:"error,omitempty"`
	Timestamp   time.Time              `json:"timestamp"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// AgentEventData represents common agent event data
type AgentEventData struct {
	AgentID     string                 `json:"agent_id"`
	AgentName   string                 `json:"agent_name"`
	AgentType   string                 `json:"agent_type"`
	State       string                 `json:"state"`
	TaskID      string                 `json:"task_id,omitempty"`
	Error       string                 `json:"error,omitempty"`
	Timestamp   time.Time              `json:"timestamp"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// CommissionEventData represents common commission event data
type CommissionEventData struct {
	CommissionID   string                 `json:"commission_id"`
	Title          string                 `json:"title"`
	Status         string                 `json:"status"`
	Progress       float64                `json:"progress,omitempty"`
	TaskCount      int                    `json:"task_count,omitempty"`
	CompletedTasks int                    `json:"completed_tasks,omitempty"`
	Error          string                 `json:"error,omitempty"`
	Timestamp      time.Time              `json:"timestamp"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
}

// DataEventData represents common data operation event data
type DataEventData struct {
	EntityType  string                 `json:"entity_type"`
	EntityID    string                 `json:"entity_id"`
	Operation   string                 `json:"operation"`
	Success     bool                   `json:"success"`
	Error       string                 `json:"error,omitempty"`
	ChangedBy   string                 `json:"changed_by,omitempty"`
	Timestamp   time.Time              `json:"timestamp"`
	OldValue    interface{}            `json:"old_value,omitempty"`
	NewValue    interface{}            `json:"new_value,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// UIEventData represents common UI event data
type UIEventData struct {
	SessionID   string                 `json:"session_id"`
	UserID      string                 `json:"user_id,omitempty"`
	EventType   string                 `json:"event_type"`
	Component   string                 `json:"component,omitempty"`
	Action      string                 `json:"action,omitempty"`
	Value       interface{}            `json:"value,omitempty"`
	Timestamp   time.Time              `json:"timestamp"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// Event Validation

// ValidateEventType checks if an event type is valid
func ValidateEventType(eventType string) bool {
	validTypes := []string{
		EventServiceStarting, EventServiceStarted, EventServiceStopping, EventServiceStopped,
		EventServiceHealthy, EventServiceUnhealthy, EventServiceError,
		EventDataCreated, EventDataUpdated, EventDataDeleted, EventDataQueried,
		EventDataSynced, EventDataCorrupted,
		EventTaskCreated, EventTaskAssigned, EventTaskStarted, EventTaskProgress,
		EventTaskCompleted, EventTaskFailed, EventTaskCancelled, EventTaskRetried,
		EventAgentRegistered, EventAgentUnregistered, EventAgentStateChanged,
		EventAgentTaskReceived, EventAgentTaskCompleted, EventAgentError, EventAgentHealthCheck,
		EventCommissionCreated, EventCommissionPlanned, EventCommissionStarted,
		EventCommissionProgress, EventCommissionCompleted, EventCommissionFailed, EventCommissionCancelled,
		EventUIConnected, EventUIDisconnected, EventUIStateChanged, EventUICommandIssued, EventUIError,
		EventSystemNotification, EventSystemWarning, EventSystemError, EventSystemShutdown,
		EventGRPCRequest, EventGRPCResponse, EventGRPCStream, EventGRPCError,
		EventCorpusScanStarted, EventCorpusScanProgress, EventCorpusScanCompleted,
		EventCorpusFileIndexed, EventCorpusIndexError, EventCorpusSearch,
	}
	
	for _, valid := range validTypes {
		if eventType == valid {
			return true
		}
	}
	return false
}

// GetEventCategory returns the category for an event type
func GetEventCategory(eventType string) string {
	switch {
	case hasPrefix(eventType, "service."):
		return EventCategoryService
	case hasPrefix(eventType, "data."):
		return EventCategoryData
	case hasPrefix(eventType, "task."):
		return EventCategoryTask
	case hasPrefix(eventType, "agent."):
		return EventCategoryAgent
	case hasPrefix(eventType, "commission."):
		return EventCategoryCommission
	case hasPrefix(eventType, "ui."):
		return EventCategoryUI
	case hasPrefix(eventType, "system."):
		return EventCategorySystem
	case hasPrefix(eventType, "grpc."):
		return EventCategoryGRPC
	case hasPrefix(eventType, "corpus."):
		return EventCategoryCorpus
	default:
		return "unknown"
	}
}

func hasPrefix(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}

// Event Versioning Support

// EventVersion represents the schema version for events
type EventVersion string

const (
	EventVersionV1 EventVersion = "v1"
	EventVersionV2 EventVersion = "v2" // Future use
)

// VersionedEvent adds version information to events
type VersionedEvent struct {
	Version EventVersion           `json:"version"`
	Type    string                 `json:"type"`
	Data    map[string]interface{} `json:"data"`
}

// ToVersionedEvent converts a standard event to a versioned event
func ToVersionedEvent(event events.CoreEvent) VersionedEvent {
	return VersionedEvent{
		Version: EventVersionV1,
		Type:    event.GetType(),
		Data:    event.GetData().(map[string]interface{}),
	}
}