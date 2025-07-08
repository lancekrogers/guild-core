// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package events

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/lancekrogers/guild/pkg/gerror"
)

// Legacy event type aliases for backward compatibility
type (
	// Integration package compatibility
	IntegrationEvent = BaseEvent

	// Orchestrator package compatibility
	OrchestratorEvent = BaseEvent

	// Corpus package compatibility
	CorpusEvent = BaseEvent

	// Scheduler package compatibility
	SchedulerEvent = AgentEvent
)

// ConversionContext provides context for event conversions
type ConversionContext struct {
	PreserveID        bool
	GenerateTimestamp bool
	DefaultSource     string
	Metadata          map[string]interface{}
}

// DefaultConversionContext returns sensible conversion defaults
func DefaultConversionContext() ConversionContext {
	return ConversionContext{
		PreserveID:        true,
		GenerateTimestamp: false,
		DefaultSource:     "unknown",
		Metadata:          make(map[string]interface{}),
	}
}

// FromJSON converts a JSON string to a CoreEvent
func FromJSON(jsonEvent string) (CoreEvent, error) {
	var raw map[string]interface{}
	if err := json.Unmarshal([]byte(jsonEvent), &raw); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeValidation, "invalid JSON event").
			WithComponent("events").
			WithOperation("FromJSON")
	}

	return FromMap(raw)
}

// ToJSON converts a CoreEvent to a JSON string
func ToJSON(event CoreEvent) (string, error) {
	data, err := json.Marshal(event)
	if err != nil {
		return "", gerror.Wrap(err, gerror.ErrCodeInternal, "failed to marshal event").
			WithComponent("events").
			WithOperation("ToJSON")
	}
	return string(data), nil
}

// FromMap converts a map to a CoreEvent
func FromMap(data map[string]interface{}) (CoreEvent, error) {
	event := &BaseEvent{
		Data:     make(map[string]interface{}),
		Metadata: make(map[string]interface{}),
	}

	// Extract required fields
	if id, ok := data["id"].(string); ok {
		event.ID = id
	}
	if eventType, ok := data["type"].(string); ok {
		event.Type = eventType
	}
	if source, ok := data["source"].(string); ok {
		event.Source = source
	}

	// Extract optional fields
	if target, ok := data["target"].(string); ok {
		event.Target = target
	}

	// Handle timestamp
	if ts, ok := data["timestamp"]; ok {
		switch t := ts.(type) {
		case string:
			if parsed, err := time.Parse(time.RFC3339, t); err == nil {
				event.Timestamp = parsed
			}
		case int64:
			event.Timestamp = time.Unix(t, 0)
		case float64:
			event.Timestamp = time.Unix(int64(t), 0)
		case time.Time:
			event.Timestamp = t
		}
	}

	// If no timestamp, use current time
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	// Extract data payload
	if eventData, ok := data["data"].(map[string]interface{}); ok {
		event.Data = eventData
	}

	// Extract metadata
	if metadata, ok := data["metadata"].(map[string]interface{}); ok {
		event.Metadata = metadata
	}

	// Copy any remaining fields to data
	for key, value := range data {
		switch key {
		case "id", "type", "source", "target", "timestamp", "data", "metadata":
			// Skip already processed fields
		default:
			event.Data[key] = value
		}
	}

	return event, nil
}

// ToMap converts a CoreEvent to a map
func ToMap(event CoreEvent) map[string]interface{} {
	result := make(map[string]interface{})

	result["id"] = event.GetID()
	result["type"] = event.GetType()
	result["source"] = event.GetSource()
	result["timestamp"] = event.GetTimestamp()

	if target := event.GetTarget(); target != "" {
		result["target"] = target
	}

	if data := event.GetData(); len(data) > 0 {
		result["data"] = data
	}

	if metadata := event.GetMetadata(); len(metadata) > 0 {
		result["metadata"] = metadata
	}

	return result
}

// FromProtobuf converts a gRPC protobuf event to a CoreEvent
// Note: This requires the actual protobuf definitions to be imported
func FromProtobufGeneric(pbData map[string]interface{}) (CoreEvent, error) {
	event := &BaseEvent{
		Data:     make(map[string]interface{}),
		Metadata: make(map[string]interface{}),
	}

	// Extract protobuf fields
	if id, ok := pbData["id"].(string); ok {
		event.ID = id
	}
	if eventType, ok := pbData["type"].(string); ok {
		event.Type = eventType
	}
	if source, ok := pbData["source"].(string); ok {
		event.Source = source
	}

	// Handle protobuf timestamp
	if ts, ok := pbData["timestamp"]; ok {
		if tsMap, ok := ts.(map[string]interface{}); ok {
			if seconds, ok := tsMap["seconds"].(int64); ok {
				if nanos, ok := tsMap["nanos"].(int32); ok {
					event.Timestamp = time.Unix(seconds, int64(nanos))
				} else {
					event.Timestamp = time.Unix(seconds, 0)
				}
			}
		}
	}

	// Handle protobuf Struct data
	if data, ok := pbData["data"]; ok {
		if dataMap, ok := data.(map[string]interface{}); ok {
			event.Data = dataMap
		}
	}

	return event, nil
}

// ToProtobufGeneric converts a CoreEvent to a generic protobuf-compatible map
func ToProtobufGeneric(event CoreEvent) map[string]interface{} {
	result := make(map[string]interface{})

	result["id"] = event.GetID()
	result["type"] = event.GetType()
	result["source"] = event.GetSource()

	// Convert timestamp to protobuf format
	ts := event.GetTimestamp()
	result["timestamp"] = map[string]interface{}{
		"seconds": ts.Unix(),
		"nanos":   int32(ts.Nanosecond()),
	}

	// Convert data to protobuf Struct format
	if data := event.GetData(); len(data) > 0 {
		result["data"] = data
	}

	return result
}

// FromOrchestratorEvent converts legacy orchestrator events to CoreEvent
func FromOrchestratorEvent(legacyEvent interface{}) (CoreEvent, error) {
	// Handle different legacy event types through reflection or type assertion
	switch e := legacyEvent.(type) {
	case map[string]interface{}:
		return FromMap(e)
	default:
		// Try to marshal to JSON and back
		data, err := json.Marshal(legacyEvent)
		if err != nil {
			return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to convert legacy event").
				WithComponent("events").
				WithOperation("FromOrchestratorEvent")
		}
		return FromJSON(string(data))
	}
}

// FromKanbanEvent converts legacy kanban BoardEvent to TaskEvent
func FromKanbanEvent(boardID, taskID, eventType string, data map[string]string) *TaskEvent {
	// Convert string map to interface map
	eventData := make(map[string]interface{})
	for k, v := range data {
		eventData[k] = v
	}

	event := NewTaskEvent(eventType, taskID, eventData)
	event.WithBoard(boardID)

	// Generate ID from components
	event.ID = fmt.Sprintf("kanban_%s_%s_%d", boardID, taskID, time.Now().UnixNano())

	return event
}

// ToKanbanEvent converts a TaskEvent to legacy kanban format
func ToKanbanEvent(event *TaskEvent) (string, string, string, map[string]string, time.Time) {
	data := make(map[string]string)

	// Convert interface data to string data
	for k, v := range event.GetData() {
		if str, ok := v.(string); ok {
			data[k] = str
		} else {
			data[k] = fmt.Sprintf("%v", v)
		}
	}

	return event.BoardID, event.TaskID, event.GetType(), data, event.GetTimestamp()
}

// FromCorpusEvent converts legacy corpus events to MemoryEvent
func FromCorpusEvent(eventType, source string, data map[string]interface{}) *MemoryEvent {
	event := NewMemoryEvent(eventType, "corpus_operation", data)
	event.Source = source

	// Extract corpus-specific fields from data
	if docID, ok := data["document_id"].(string); ok {
		event.WithDocument(docID)
	}
	if corpusID, ok := data["corpus_id"].(string); ok {
		event.WithCorpus(corpusID)
	}
	if size, ok := data["size"].(int64); ok {
		event.WithSize(size)
	}

	return event
}

// FromSchedulerEvent converts legacy scheduler events to AgentEvent
func FromSchedulerEvent(taskID, agentID, eventType string, attributes map[string]interface{}) *AgentEvent {
	event := NewAgentEvent(eventType, agentID, attributes)
	event.WithCurrentTask(taskID)

	// Extract scheduler-specific fields
	if status, ok := attributes["status"].(string); ok {
		event.WithStatus(status)
	}
	if agentType, ok := attributes["agent_type"].(string); ok {
		event.WithAgentType(agentType)
	}

	return event
}

// EventConverter provides a configurable conversion interface
type EventConverter struct {
	context ConversionContext
}

// NewEventConverter creates a new event converter with the given context
func NewEventConverter(ctx ConversionContext) *EventConverter {
	return &EventConverter{context: ctx}
}

// Convert converts any supported event type to CoreEvent
func (c *EventConverter) Convert(input interface{}) (CoreEvent, error) {
	switch e := input.(type) {
	case CoreEvent:
		return e, nil
	case *BaseEvent:
		return e, nil
	case map[string]interface{}:
		return c.convertFromMap(e)
	case string:
		return FromJSON(e)
	default:
		return c.convertGeneric(input)
	}
}

// convertFromMap converts a map with conversion context
func (c *EventConverter) convertFromMap(data map[string]interface{}) (CoreEvent, error) {
	event, err := FromMap(data)
	if err != nil {
		return nil, err
	}

	baseEvent := event.(*BaseEvent)

	// Apply conversion context
	if !c.context.PreserveID {
		baseEvent.ID = fmt.Sprintf("converted_%d", time.Now().UnixNano())
	}

	if c.context.GenerateTimestamp {
		baseEvent.Timestamp = time.Now()
	}

	if baseEvent.Source == "" && c.context.DefaultSource != "" {
		baseEvent.Source = c.context.DefaultSource
	}

	// Add context metadata
	for k, v := range c.context.Metadata {
		baseEvent.WithMetadata(k, v)
	}

	return baseEvent, nil
}

// convertGeneric tries to convert unknown types through JSON marshaling
func (c *EventConverter) convertGeneric(input interface{}) (CoreEvent, error) {
	data, err := json.Marshal(input)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to marshal input for conversion").
			WithComponent("events").
			WithOperation("convertGeneric")
	}

	return FromJSON(string(data))
}

// ValidateConversion checks if a conversion was successful
func ValidateConversion(original interface{}, converted CoreEvent) error {
	if converted == nil {
		return gerror.New(gerror.ErrCodeValidation, "conversion resulted in nil event", nil)
	}

	if err := converted.(*BaseEvent).Validate(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeValidation, "converted event failed validation").
			WithComponent("events").
			WithOperation("ValidateConversion")
	}

	return nil
}
