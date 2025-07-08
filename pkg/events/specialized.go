// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package events

import (
	"time"
)

// TaskEvent represents events related to task operations in the kanban system
type TaskEvent struct {
	*BaseEvent
	TaskID      string `json:"task_id"`
	BoardID     string `json:"board_id,omitempty"`
	Status      string `json:"status,omitempty"`
	Priority    string `json:"priority,omitempty"`
	AssignedTo  string `json:"assigned_to,omitempty"`
	Description string `json:"description,omitempty"`
}

// NewTaskEvent creates a new task-related event
func NewTaskEvent(eventType, taskID string, data map[string]interface{}) *TaskEvent {
	base := NewBaseEvent("", eventType, "kanban", data)
	return &TaskEvent{
		BaseEvent: base,
		TaskID:    taskID,
	}
}

// WithBoard sets the board ID for the task event
func (e *TaskEvent) WithBoard(boardID string) *TaskEvent {
	e.BoardID = boardID
	return e
}

// WithStatus sets the task status
func (e *TaskEvent) WithStatus(status string) *TaskEvent {
	e.Status = status
	return e
}

// WithAssignment sets the assigned agent
func (e *TaskEvent) WithAssignment(agentID string) *TaskEvent {
	e.AssignedTo = agentID
	return e
}

// AgentEvent represents events related to agent operations
type AgentEvent struct {
	*BaseEvent
	AgentID      string   `json:"agent_id"`
	AgentType    string   `json:"agent_type,omitempty"`
	Status       string   `json:"status,omitempty"`
	CurrentTask  string   `json:"current_task,omitempty"`
	Capabilities []string `json:"capabilities,omitempty"`
}

// NewAgentEvent creates a new agent-related event
func NewAgentEvent(eventType, agentID string, data map[string]interface{}) *AgentEvent {
	base := NewBaseEvent("", eventType, "orchestrator", data)
	return &AgentEvent{
		BaseEvent: base,
		AgentID:   agentID,
	}
}

// WithAgentType sets the agent type
func (e *AgentEvent) WithAgentType(agentType string) *AgentEvent {
	e.AgentType = agentType
	return e
}

// WithStatus sets the agent status
func (e *AgentEvent) WithStatus(status string) *AgentEvent {
	e.Status = status
	return e
}

// WithCurrentTask sets the current task
func (e *AgentEvent) WithCurrentTask(taskID string) *AgentEvent {
	e.CurrentTask = taskID
	return e
}

// SystemEvent represents system-level events
type SystemEvent struct {
	*BaseEvent
	ComponentID string `json:"component_id"`
	Severity    string `json:"severity"`
	Message     string `json:"message,omitempty"`
	ErrorCode   string `json:"error_code,omitempty"`
}

// NewSystemEvent creates a new system-level event
func NewSystemEvent(eventType, componentID, severity string, data map[string]interface{}) *SystemEvent {
	base := NewBaseEvent("", eventType, "system", data)
	return &SystemEvent{
		BaseEvent:   base,
		ComponentID: componentID,
		Severity:    severity,
	}
}

// WithMessage sets the system event message
func (e *SystemEvent) WithMessage(message string) *SystemEvent {
	e.Message = message
	return e
}

// WithErrorCode sets the error code
func (e *SystemEvent) WithErrorCode(errorCode string) *SystemEvent {
	e.ErrorCode = errorCode
	return e
}

// CommissionEvent represents events related to commission operations
type CommissionEvent struct {
	*BaseEvent
	CommissionID   string            `json:"commission_id"`
	Title          string            `json:"title,omitempty"`
	Status         string            `json:"status,omitempty"`
	Progress       float64           `json:"progress,omitempty"`
	AssignedAgents []string          `json:"assigned_agents,omitempty"`
	EstimatedCost  float64           `json:"estimated_cost,omitempty"`
	ActualCost     float64           `json:"actual_cost,omitempty"`
	Tags           map[string]string `json:"tags,omitempty"`
}

// NewCommissionEvent creates a new commission-related event
func NewCommissionEvent(eventType, commissionID string, data map[string]interface{}) *CommissionEvent {
	base := NewBaseEvent("", eventType, "commission", data)
	return &CommissionEvent{
		BaseEvent:    base,
		CommissionID: commissionID,
	}
}

// WithTitle sets the commission title
func (e *CommissionEvent) WithTitle(title string) *CommissionEvent {
	e.Title = title
	return e
}

// WithStatus sets the commission status
func (e *CommissionEvent) WithStatus(status string) *CommissionEvent {
	e.Status = status
	return e
}

// WithProgress sets the commission progress
func (e *CommissionEvent) WithProgress(progress float64) *CommissionEvent {
	e.Progress = progress
	return e
}

// UIEvent represents user interface events
type UIEvent struct {
	*BaseEvent
	ComponentID string            `json:"component_id"`
	Action      string            `json:"action"`
	UserID      string            `json:"user_id,omitempty"`
	SessionID   string            `json:"session_id,omitempty"`
	Params      map[string]string `json:"params,omitempty"`
}

// NewUIEvent creates a new UI-related event
func NewUIEvent(eventType, componentID, action string, data map[string]interface{}) *UIEvent {
	base := NewBaseEvent("", eventType, "ui", data)
	return &UIEvent{
		BaseEvent:   base,
		ComponentID: componentID,
		Action:      action,
	}
}

// WithUser sets the user ID
func (e *UIEvent) WithUser(userID string) *UIEvent {
	e.UserID = userID
	return e
}

// WithSession sets the session ID
func (e *UIEvent) WithSession(sessionID string) *UIEvent {
	e.SessionID = sessionID
	return e
}

// WithParams sets UI event parameters
func (e *UIEvent) WithParams(params map[string]string) *UIEvent {
	e.Params = params
	return e
}

// PerformanceEvent represents performance monitoring events
type PerformanceEvent struct {
	*BaseEvent
	MetricName  string  `json:"metric_name"`
	Value       float64 `json:"value"`
	Unit        string  `json:"unit"`
	Threshold   float64 `json:"threshold,omitempty"`
	ComponentID string  `json:"component_id"`
	Operation   string  `json:"operation,omitempty"`
}

// NewPerformanceEvent creates a new performance-related event
func NewPerformanceEvent(metricName string, value float64, unit, componentID string) *PerformanceEvent {
	base := NewBaseEvent("", EventTypePerformanceMetric, "performance", map[string]interface{}{
		"metric_name":  metricName,
		"value":        value,
		"unit":         unit,
		"component_id": componentID,
	})

	return &PerformanceEvent{
		BaseEvent:   base,
		MetricName:  metricName,
		Value:       value,
		Unit:        unit,
		ComponentID: componentID,
	}
}

// WithThreshold sets the performance threshold
func (e *PerformanceEvent) WithThreshold(threshold float64) *PerformanceEvent {
	e.Threshold = threshold
	return e
}

// WithOperation sets the operation name
func (e *PerformanceEvent) WithOperation(operation string) *PerformanceEvent {
	e.Operation = operation
	return e
}

// IsAlert returns true if the performance value exceeds the threshold
func (e *PerformanceEvent) IsAlert() bool {
	return e.Threshold > 0 && e.Value > e.Threshold
}

// MemoryEvent represents memory/corpus operation events
type MemoryEvent struct {
	*BaseEvent
	OperationType string `json:"operation_type"`
	DocumentID    string `json:"document_id,omitempty"`
	CorpusID      string `json:"corpus_id,omitempty"`
	Size          int64  `json:"size,omitempty"`
	VectorDims    int    `json:"vector_dims,omitempty"`
	Query         string `json:"query,omitempty"`
	ResultCount   int    `json:"result_count,omitempty"`
}

// NewMemoryEvent creates a new memory/corpus event
func NewMemoryEvent(eventType, operationType string, data map[string]interface{}) *MemoryEvent {
	base := NewBaseEvent("", eventType, "memory", data)
	return &MemoryEvent{
		BaseEvent:     base,
		OperationType: operationType,
	}
}

// WithDocument sets the document ID
func (e *MemoryEvent) WithDocument(documentID string) *MemoryEvent {
	e.DocumentID = documentID
	return e
}

// WithCorpus sets the corpus ID
func (e *MemoryEvent) WithCorpus(corpusID string) *MemoryEvent {
	e.CorpusID = corpusID
	return e
}

// WithSize sets the operation size
func (e *MemoryEvent) WithSize(size int64) *MemoryEvent {
	e.Size = size
	return e
}

// EventBuilder provides a fluent interface for building events
type EventBuilder struct {
	event *BaseEvent
}

// NewEventBuilder creates a new event builder
func NewEventBuilder(eventType, source string) *EventBuilder {
	return &EventBuilder{
		event: NewBaseEvent("", eventType, source, nil),
	}
}

// WithID sets the event ID
func (b *EventBuilder) WithID(id string) *EventBuilder {
	b.event.ID = id
	return b
}

// WithTarget sets the event target
func (b *EventBuilder) WithTarget(target string) *EventBuilder {
	b.event.Target = target
	return b
}

// WithTimestamp sets the event timestamp
func (b *EventBuilder) WithTimestamp(timestamp time.Time) *EventBuilder {
	b.event.Timestamp = timestamp
	return b
}

// WithData adds data to the event
func (b *EventBuilder) WithData(key string, value interface{}) *EventBuilder {
	if b.event.Data == nil {
		b.event.Data = make(map[string]interface{})
	}
	b.event.Data[key] = value
	return b
}

// WithMetadata adds metadata to the event
func (b *EventBuilder) WithMetadata(key string, value interface{}) *EventBuilder {
	if b.event.Metadata == nil {
		b.event.Metadata = make(map[string]interface{})
	}
	b.event.Metadata[key] = value
	return b
}

// Build returns the constructed event
func (b *EventBuilder) Build() *BaseEvent {
	return b.event
}

// AsTaskEvent converts to a TaskEvent (adds task-specific fields as empty)
func (b *EventBuilder) AsTaskEvent() *TaskEvent {
	return &TaskEvent{
		BaseEvent: b.event,
	}
}

// AsAgentEvent converts to an AgentEvent (adds agent-specific fields as empty)
func (b *EventBuilder) AsAgentEvent() *AgentEvent {
	return &AgentEvent{
		BaseEvent: b.event,
	}
}

// AsSystemEvent converts to a SystemEvent (adds system-specific fields as empty)
func (b *EventBuilder) AsSystemEvent() *SystemEvent {
	return &SystemEvent{
		BaseEvent: b.event,
	}
}
