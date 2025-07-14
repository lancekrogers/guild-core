// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package types

import (
	"context"
	"fmt"
	"time"

	"github.com/lancekrogers/guild/pkg/events"
	"github.com/lancekrogers/guild/pkg/gerror"
)

// TypedEventBuilder provides type-safe event creation with schema validation
type TypedEventBuilder struct {
	eventType string
	source    string
	data      map[string]interface{}
	metadata  map[string]interface{}
	registry  *EventRegistry
}

// NewTypedEventBuilder creates a new typed event builder
func NewTypedEventBuilder(eventType string, registry *EventRegistry) *TypedEventBuilder {
	return &TypedEventBuilder{
		eventType: eventType,
		source:    "guild",
		data:      make(map[string]interface{}),
		metadata:  make(map[string]interface{}),
		registry:  registry,
	}
}

// WithSource sets the event source
func (b *TypedEventBuilder) WithSource(source string) *TypedEventBuilder {
	b.source = source
	return b
}

// WithData adds data to the event
func (b *TypedEventBuilder) WithData(key string, value interface{}) *TypedEventBuilder {
	b.data[key] = value
	return b
}

// WithMetadata adds metadata to the event
func (b *TypedEventBuilder) WithMetadata(key string, value interface{}) *TypedEventBuilder {
	b.metadata[key] = value
	return b
}

// WithSchemaVersion sets the schema version
func (b *TypedEventBuilder) WithSchemaVersion(version string) *TypedEventBuilder {
	b.metadata["schema_version"] = version
	return b
}

// Build creates the event after validation
func (b *TypedEventBuilder) Build() (events.CoreEvent, error) {
	// Add schema version if not set
	if _, exists := b.metadata["schema_version"]; !exists {
		// Try to get latest schema version
		if b.registry != nil {
			if schema, err := b.registry.GetLatestSchema(b.eventType); err == nil {
				b.metadata["schema_version"] = schema.Version
			}
		}
	}

	// Create event using registry if available
	var event events.CoreEvent
	var err error

	if b.registry != nil {
		event, err = b.registry.CreateEvent(b.eventType, b.data)
		if err != nil {
			return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create event")
		}
	} else {
		// Fallback to base event
		event = events.NewBaseEvent(
			generateEventID(),
			b.eventType,
			b.source,
			b.data,
		)
	}

	// Cast to BaseEvent to set additional properties
	baseEvent, ok := event.(*events.BaseEvent)
	if ok {
		// Update source if it was set
		if b.source != "guild" {
			baseEvent.Source = b.source
		}

		// Add metadata
		for k, v := range b.metadata {
			baseEvent.WithMetadata(k, v)
		}
	}

	// Validate against schema if registry is available
	if b.registry != nil {
		if err := b.registry.ValidateEvent(context.Background(), event); err != nil {
			return nil, gerror.Wrap(err, gerror.ErrCodeValidation, "event validation failed")
		}
	}

	return event, nil
}

// TaskEventBuilder builds task events
type TaskEventBuilder struct {
	*TypedEventBuilder
}

// NewTaskEventBuilder creates a new task event builder
func NewTaskEventBuilder(eventType string, registry *EventRegistry) *TaskEventBuilder {
	return &TaskEventBuilder{
		TypedEventBuilder: NewTypedEventBuilder(eventType, registry),
	}
}

// WithTaskID sets the task ID
func (b *TaskEventBuilder) WithTaskID(taskID string) *TaskEventBuilder {
	b.WithData("task_id", taskID)
	return b
}

// WithName sets the task name
func (b *TaskEventBuilder) WithName(name string) *TaskEventBuilder {
	b.WithData("name", name)
	return b
}

// WithDescription sets the task description
func (b *TaskEventBuilder) WithDescription(description string) *TaskEventBuilder {
	b.WithData("description", description)
	return b
}

// WithStatus sets the task status
func (b *TaskEventBuilder) WithStatus(status string) *TaskEventBuilder {
	b.WithData("status", status)
	return b
}

// WithPriority sets the task priority
func (b *TaskEventBuilder) WithPriority(priority string) *TaskEventBuilder {
	b.WithData("priority", priority)
	return b
}

// WithAssignee sets the task assignee
func (b *TaskEventBuilder) WithAssignee(assignee string) *TaskEventBuilder {
	b.WithData("assignee", assignee)
	return b
}

// AgentEventBuilder builds agent events
type AgentEventBuilder struct {
	*TypedEventBuilder
}

// NewAgentEventBuilder creates a new agent event builder
func NewAgentEventBuilder(eventType string, registry *EventRegistry) *AgentEventBuilder {
	return &AgentEventBuilder{
		TypedEventBuilder: NewTypedEventBuilder(eventType, registry),
	}
}

// WithAgentID sets the agent ID
func (b *AgentEventBuilder) WithAgentID(agentID string) *AgentEventBuilder {
	b.WithData("agent_id", agentID)
	return b
}

// WithAgentName sets the agent name
func (b *AgentEventBuilder) WithAgentName(name string) *AgentEventBuilder {
	b.WithData("agent_name", name)
	return b
}

// WithCapabilities sets the agent capabilities
func (b *AgentEventBuilder) WithCapabilities(capabilities []string) *AgentEventBuilder {
	caps := make([]interface{}, len(capabilities))
	for i, c := range capabilities {
		caps[i] = c
	}
	b.WithData("capabilities", caps)
	return b
}

// WithStatus sets the agent status
func (b *AgentEventBuilder) WithStatus(status string) *AgentEventBuilder {
	b.WithData("status", status)
	return b
}

// SystemEventBuilder builds system events
type SystemEventBuilder struct {
	*TypedEventBuilder
}

// NewSystemEventBuilder creates a new system event builder
func NewSystemEventBuilder(eventType string, registry *EventRegistry) *SystemEventBuilder {
	return &SystemEventBuilder{
		TypedEventBuilder: NewTypedEventBuilder(eventType, registry),
	}
}

// WithComponent sets the system component
func (b *SystemEventBuilder) WithComponent(component string) *SystemEventBuilder {
	b.WithData("component", component)
	return b
}

// WithSeverity sets the event severity
func (b *SystemEventBuilder) WithSeverity(severity string) *SystemEventBuilder {
	b.WithData("severity", severity)
	return b
}

// WithMessage sets the event message
func (b *SystemEventBuilder) WithMessage(message string) *SystemEventBuilder {
	b.WithData("message", message)
	return b
}

// WithMetrics sets system metrics
func (b *SystemEventBuilder) WithMetrics(metrics map[string]interface{}) *SystemEventBuilder {
	b.WithData("metrics", metrics)
	return b
}

// CommissionEventBuilder builds commission events
type CommissionEventBuilder struct {
	*TypedEventBuilder
}

// NewCommissionEventBuilder creates a new commission event builder
func NewCommissionEventBuilder(eventType string, registry *EventRegistry) *CommissionEventBuilder {
	return &CommissionEventBuilder{
		TypedEventBuilder: NewTypedEventBuilder(eventType, registry),
	}
}

// WithCommissionID sets the commission ID
func (b *CommissionEventBuilder) WithCommissionID(commissionID string) *CommissionEventBuilder {
	b.WithData("commission_id", commissionID)
	return b
}

// WithTitle sets the commission title
func (b *CommissionEventBuilder) WithTitle(title string) *CommissionEventBuilder {
	b.WithData("title", title)
	return b
}

// WithObjective sets the commission objective
func (b *CommissionEventBuilder) WithObjective(objective string) *CommissionEventBuilder {
	b.WithData("objective", objective)
	return b
}

// WithProgress sets the commission progress
func (b *CommissionEventBuilder) WithProgress(progress float64) *CommissionEventBuilder {
	b.WithData("progress", progress)
	return b
}

// WithStatus sets the commission status
func (b *CommissionEventBuilder) WithStatus(status string) *CommissionEventBuilder {
	b.WithData("status", status)
	return b
}

// MemoryEventBuilder builds memory events
type MemoryEventBuilder struct {
	*TypedEventBuilder
}

// NewMemoryEventBuilder creates a new memory event builder
func NewMemoryEventBuilder(eventType string, registry *EventRegistry) *MemoryEventBuilder {
	return &MemoryEventBuilder{
		TypedEventBuilder: NewTypedEventBuilder(eventType, registry),
	}
}

// WithOperation sets the memory operation type
func (b *MemoryEventBuilder) WithOperation(operation string) *MemoryEventBuilder {
	b.WithData("operation", operation)
	return b
}

// WithMemoryType sets the memory type
func (b *MemoryEventBuilder) WithMemoryType(memoryType string) *MemoryEventBuilder {
	b.WithData("memory_type", memoryType)
	return b
}

// WithContent sets the memory content
func (b *MemoryEventBuilder) WithContent(content string) *MemoryEventBuilder {
	b.WithData("content", content)
	return b
}

// WithEmbedding sets the memory embedding
func (b *MemoryEventBuilder) WithEmbedding(embedding []float64) *MemoryEventBuilder {
	emb := make([]interface{}, len(embedding))
	for i, v := range embedding {
		emb[i] = v
	}
	b.WithData("embedding", emb)
	return b
}

// WithSimilarity sets the similarity score
func (b *MemoryEventBuilder) WithSimilarity(similarity float64) *MemoryEventBuilder {
	b.WithData("similarity", similarity)
	return b
}

// UIEventBuilder builds UI events
type UIEventBuilder struct {
	*TypedEventBuilder
}

// NewUIEventBuilder creates a new UI event builder
func NewUIEventBuilder(eventType string, registry *EventRegistry) *UIEventBuilder {
	return &UIEventBuilder{
		TypedEventBuilder: NewTypedEventBuilder(eventType, registry),
	}
}

// WithComponent sets the UI component
func (b *UIEventBuilder) WithComponent(component string) *UIEventBuilder {
	b.WithData("component", component)
	return b
}

// WithAction sets the UI action
func (b *UIEventBuilder) WithAction(action string) *UIEventBuilder {
	b.WithData("action", action)
	return b
}

// WithUserID sets the user ID
func (b *UIEventBuilder) WithUserID(userID string) *UIEventBuilder {
	b.WithData("user_id", userID)
	return b
}

// WithSessionID sets the session ID
func (b *UIEventBuilder) WithSessionID(sessionID string) *UIEventBuilder {
	b.WithData("session_id", sessionID)
	return b
}

// WithData adds UI-specific data
func (b *UIEventBuilder) WithUIData(data map[string]interface{}) *UIEventBuilder {
	for k, v := range data {
		b.WithData(k, v)
	}
	return b
}

// generateEventID creates a unique event ID
func generateEventID() string {
	return fmt.Sprintf("evt_%d_%d", time.Now().UnixNano(), time.Now().Nanosecond())
}
