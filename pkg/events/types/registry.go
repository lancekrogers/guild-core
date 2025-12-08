// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package types

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"sync"

	"github.com/guild-framework/guild-core/pkg/events"
	"github.com/guild-framework/guild-core/pkg/gerror"
)

// EventSchema defines the schema for an event type
type EventSchema struct {
	Type        string                 `json:"type"`
	Version     string                 `json:"version"`
	Description string                 `json:"description"`
	Required    []string               `json:"required"`
	Properties  map[string]Property    `json:"properties"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// Property defines a property in an event schema
type Property struct {
	Type        string                 `json:"type"`
	Description string                 `json:"description"`
	Required    bool                   `json:"required"`
	Default     interface{}            `json:"default,omitempty"`
	Enum        []interface{}          `json:"enum,omitempty"`
	Properties  map[string]Property    `json:"properties,omitempty"` // For nested objects
	Items       *Property              `json:"items,omitempty"`      // For arrays
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// EventRegistry manages event types and schemas
type EventRegistry struct {
	mu        sync.RWMutex
	schemas   map[string]*EventSchema
	factories map[string]EventFactory
	versions  map[string][]string // type -> versions
}

// EventFactory creates events of a specific type
type EventFactory func(data map[string]interface{}) (events.CoreEvent, error)

// NewEventRegistry creates a new event registry
func NewEventRegistry() *EventRegistry {
	registry := &EventRegistry{
		schemas:   make(map[string]*EventSchema),
		factories: make(map[string]EventFactory),
		versions:  make(map[string][]string),
	}

	// Register built-in event types
	registry.registerBuiltinTypes()

	return registry
}

// RegisterSchema registers an event schema
func (r *EventRegistry) RegisterSchema(schema *EventSchema) error {
	if schema == nil {
		return gerror.New(gerror.ErrCodeValidation, "schema is required", nil)
	}

	if schema.Type == "" {
		return gerror.New(gerror.ErrCodeValidation, "schema type is required", nil)
	}

	if schema.Version == "" {
		return gerror.New(gerror.ErrCodeValidation, "schema version is required", nil)
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	key := fmt.Sprintf("%s:%s", schema.Type, schema.Version)

	if _, exists := r.schemas[key]; exists {
		return gerror.New(gerror.ErrCodeAlreadyExists, "schema already registered", nil).
			WithDetails("type", schema.Type).
			WithDetails("version", schema.Version)
	}

	r.schemas[key] = schema
	r.versions[schema.Type] = append(r.versions[schema.Type], schema.Version)

	return nil
}

// RegisterFactory registers an event factory
func (r *EventRegistry) RegisterFactory(eventType string, factory EventFactory) error {
	if eventType == "" {
		return gerror.New(gerror.ErrCodeValidation, "event type is required", nil)
	}

	if factory == nil {
		return gerror.New(gerror.ErrCodeValidation, "factory is required", nil)
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.factories[eventType]; exists {
		return gerror.New(gerror.ErrCodeAlreadyExists, "factory already registered", nil).
			WithDetails("type", eventType)
	}

	r.factories[eventType] = factory

	return nil
}

// GetSchema retrieves a schema by type and version
func (r *EventRegistry) GetSchema(eventType, version string) (*EventSchema, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	key := fmt.Sprintf("%s:%s", eventType, version)
	schema, exists := r.schemas[key]

	if !exists {
		return nil, gerror.New(gerror.ErrCodeNotFound, "schema not found", nil).
			WithDetails("type", eventType).
			WithDetails("version", version)
	}

	return schema, nil
}

// GetLatestSchema retrieves the latest version of a schema
func (r *EventRegistry) GetLatestSchema(eventType string) (*EventSchema, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	versions, exists := r.versions[eventType]
	if !exists || len(versions) == 0 {
		return nil, gerror.New(gerror.ErrCodeNotFound, "no schema versions found", nil).
			WithDetails("type", eventType)
	}

	// Assume versions are in order (should implement proper version comparison)
	latestVersion := versions[len(versions)-1]
	key := fmt.Sprintf("%s:%s", eventType, latestVersion)

	return r.schemas[key], nil
}

// CreateEvent creates an event using the registered factory
func (r *EventRegistry) CreateEvent(eventType string, data map[string]interface{}) (events.CoreEvent, error) {
	r.mu.RLock()
	factory, exists := r.factories[eventType]
	r.mu.RUnlock()

	if !exists {
		// Fall back to generic event creation
		return events.NewBaseEvent(
			generateEventID(),
			eventType,
			"registry",
			data,
		), nil
	}

	return factory(data)
}

// ValidateEvent validates an event against its schema
func (r *EventRegistry) ValidateEvent(ctx context.Context, event events.CoreEvent) error {
	if err := ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled")
	}

	// Extract version from metadata if available
	version := "1.0.0" // default version
	if metadata := event.GetMetadata(); metadata != nil {
		if v, ok := metadata["schema_version"].(string); ok {
			version = v
		}
	}

	schema, err := r.GetSchema(event.GetType(), version)
	if err != nil {
		// If no schema found, allow the event (backward compatibility)
		return nil
	}

	return r.validateData(event.GetData(), schema)
}

// validateData validates data against a schema
func (r *EventRegistry) validateData(data map[string]interface{}, schema *EventSchema) error {
	// Check required fields
	for _, required := range schema.Required {
		if _, exists := data[required]; !exists {
			return gerror.New(gerror.ErrCodeValidation, "missing required field", nil).
				WithDetails("field", required)
		}
	}

	// Validate each property
	for name, prop := range schema.Properties {
		value, exists := data[name]

		if prop.Required && !exists {
			return gerror.New(gerror.ErrCodeValidation, "missing required property", nil).
				WithDetails("property", name)
		}

		if exists {
			if err := r.validateProperty(name, value, &prop); err != nil {
				return err
			}
		}
	}

	return nil
}

// validateProperty validates a single property value
func (r *EventRegistry) validateProperty(name string, value interface{}, prop *Property) error {
	// Check type
	valueType := getType(value)
	if prop.Type != "" && valueType != prop.Type {
		return gerror.New(gerror.ErrCodeValidation, "property type mismatch", nil).
			WithDetails("property", name).
			WithDetails("expected", prop.Type).
			WithDetails("actual", valueType)
	}

	// Check enum values
	if len(prop.Enum) > 0 {
		found := false
		for _, enumValue := range prop.Enum {
			if reflect.DeepEqual(value, enumValue) {
				found = true
				break
			}
		}
		if !found {
			return gerror.New(gerror.ErrCodeValidation, "value not in enum", nil).
				WithDetails("property", name).
				WithDetails("value", value)
		}
	}

	// Validate nested objects
	if prop.Type == "object" && prop.Properties != nil {
		if objData, ok := value.(map[string]interface{}); ok {
			for subName, subProp := range prop.Properties {
				if subValue, exists := objData[subName]; exists {
					if err := r.validateProperty(fmt.Sprintf("%s.%s", name, subName), subValue, &subProp); err != nil {
						return err
					}
				} else if subProp.Required {
					return gerror.New(gerror.ErrCodeValidation, "missing required nested property", nil).
						WithDetails("property", fmt.Sprintf("%s.%s", name, subName))
				}
			}
		}
	}

	// Validate array items
	if prop.Type == "array" && prop.Items != nil {
		if arr, ok := value.([]interface{}); ok {
			for i, item := range arr {
				if err := r.validateProperty(fmt.Sprintf("%s[%d]", name, i), item, prop.Items); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

// GetEventTypes returns all registered event types
func (r *EventRegistry) GetEventTypes() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	types := make([]string, 0, len(r.versions))
	for eventType := range r.versions {
		types = append(types, eventType)
	}

	return types
}

// GetVersions returns all versions for an event type
func (r *EventRegistry) GetVersions(eventType string) []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	versions := r.versions[eventType]
	result := make([]string, len(versions))
	copy(result, versions)

	return result
}

// ExportSchemas exports all schemas as JSON
func (r *EventRegistry) ExportSchemas() ([]byte, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	schemas := make([]*EventSchema, 0, len(r.schemas))
	for _, schema := range r.schemas {
		schemas = append(schemas, schema)
	}

	return json.MarshalIndent(schemas, "", "  ")
}

// ImportSchemas imports schemas from JSON
func (r *EventRegistry) ImportSchemas(data []byte) error {
	var schemas []*EventSchema
	if err := json.Unmarshal(data, &schemas); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeValidation, "failed to unmarshal schemas")
	}

	for _, schema := range schemas {
		if err := r.RegisterSchema(schema); err != nil {
			return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to register schema").
				WithDetails("type", schema.Type)
		}
	}

	return nil
}

// registerBuiltinTypes registers the built-in event types
func (r *EventRegistry) registerBuiltinTypes() {
	// Task events
	r.RegisterSchema(&EventSchema{
		Type:        "task.created",
		Version:     "1.0.0",
		Description: "Emitted when a new task is created",
		Required:    []string{"task_id", "name"},
		Properties: map[string]Property{
			"task_id": {
				Type:        "string",
				Description: "Unique identifier for the task",
				Required:    true,
			},
			"name": {
				Type:        "string",
				Description: "Name of the task",
				Required:    true,
			},
			"description": {
				Type:        "string",
				Description: "Detailed description of the task",
				Required:    false,
			},
			"status": {
				Type:        "string",
				Description: "Current status of the task",
				Required:    false,
				Default:     "pending",
				Enum:        []interface{}{"pending", "in_progress", "completed", "failed"},
			},
			"priority": {
				Type:        "string",
				Description: "Priority level of the task",
				Required:    false,
				Default:     "normal",
				Enum:        []interface{}{"low", "normal", "high", "critical"},
			},
		},
	})

	// Agent events
	r.RegisterSchema(&EventSchema{
		Type:        "agent.started",
		Version:     "1.0.0",
		Description: "Emitted when an agent starts",
		Required:    []string{"agent_id"},
		Properties: map[string]Property{
			"agent_id": {
				Type:        "string",
				Description: "Unique identifier for the agent",
				Required:    true,
			},
			"agent_name": {
				Type:        "string",
				Description: "Name of the agent",
				Required:    false,
			},
			"capabilities": {
				Type:        "array",
				Description: "List of agent capabilities",
				Required:    false,
				Items: &Property{
					Type: "string",
				},
			},
		},
	})

	// System events
	r.RegisterSchema(&EventSchema{
		Type:        "system.heartbeat",
		Version:     "1.0.0",
		Description: "System heartbeat event",
		Required:    []string{},
		Properties: map[string]Property{
			"timestamp": {
				Type:        "string",
				Description: "Heartbeat timestamp",
				Required:    false,
			},
			"metrics": {
				Type:        "object",
				Description: "System metrics",
				Required:    false,
				Properties: map[string]Property{
					"cpu_usage": {
						Type:        "number",
						Description: "CPU usage percentage",
					},
					"memory_usage": {
						Type:        "number",
						Description: "Memory usage percentage",
					},
				},
			},
		},
	})

	// Register factories for built-in types
	r.RegisterFactory("task.created", func(data map[string]interface{}) (events.CoreEvent, error) {
		return events.NewTaskEvent("task.created",
			getStringValue(data, "task_id"),
			data,
		), nil
	})

	r.RegisterFactory("agent.started", func(data map[string]interface{}) (events.CoreEvent, error) {
		return events.NewAgentEvent("agent.started",
			getStringValue(data, "agent_id"),
			data,
		), nil
	})

	r.RegisterFactory("system.heartbeat", func(data map[string]interface{}) (events.CoreEvent, error) {
		return events.NewSystemEvent("system.heartbeat",
			"system",
			"info",
			data,
		), nil
	})
}

// Helper functions

func getType(value interface{}) string {
	if value == nil {
		return "null"
	}

	switch value.(type) {
	case string:
		return "string"
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return "integer"
	case float32, float64:
		return "number"
	case bool:
		return "boolean"
	case []interface{}:
		return "array"
	case map[string]interface{}:
		return "object"
	default:
		return "unknown"
	}
}

func getStringValue(data map[string]interface{}, key string) string {
	if value, ok := data[key].(string); ok {
		return value
	}
	return ""
}
