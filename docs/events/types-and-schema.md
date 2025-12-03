# Event Types and Schema System

## Overview

The event types and schema system provides:

- Schema-based event validation
- Type-safe event builders
- Event serialization (JSON, binary, compressed)
- Schema evolution and migration support
- Property validators (email, URL, UUID, etc.)

## Components

### Event Registry (`pkg/events/types/registry.go`)

- Manages event schemas and versions
- Provides event factories for type-safe creation
- Validates events against schemas
- Supports schema import/export

### Event Builders (`pkg/events/types/builders.go`)

- Type-safe builders for different event categories
- Automatic schema validation
- Fluent API for event construction

### Validation System (`pkg/events/types/validation.go`)

- Property validators (email, URL, UUID, semver, etc.)
- Event-specific validation rules
- Schema evolution support

### Serialization (`pkg/events/types/serialization.go`)

- Multiple formats: JSON, binary (gob), compressed
- Batch serialization for efficiency
- Streaming support for large datasets

## Usage Examples

### Register a Schema

```go
registry := NewEventRegistry()
schema := &EventSchema{
    Type:        "user.created",
    Version:     "1.0.0",
    Description: "User creation event",
    Required:    []string{"user_id", "email"},
    Properties: map[string]Property{
        "user_id": {Type: "string", Required: true},
        "email":   {Type: "string", Required: true},
    },
}
registry.RegisterSchema(schema)
```

### Build Type-Safe Events

```go
builder := NewTaskEventBuilder("task.created", registry)
event, err := builder.
    WithTaskID("task123").
    WithName("Implement feature").
    WithStatus("pending").
    WithPriority("high").
    Build()
```

### Serialize Events

```go
serializer := NewSerializer(FormatCompressed, registry)
data, err := serializer.Serialize(ctx, event)

// Deserialize
event, err = serializer.Deserialize(ctx, data)
```

### Schema Evolution

```go
evolution := NewSchemaEvolution(registry)
evolution.AddMigration("user.created", Migration{
    FromVersion: "1.0.0",
    ToVersion:   "2.0.0",
    Migrate: func(data map[string]interface{}) (map[string]interface{}, error) {
        // Migration logic
    },
})
```

## Built-in Event Types

- **Task Events**: task.created, task.updated, task.completed
- **Agent Events**: agent.started, agent.stopped, agent.error
- **System Events**: system.heartbeat, system.alert, system.metric
- **Commission Events**: commission.created, commission.updated
- **Memory Events**: memory.stored, memory.retrieved, memory.deleted
- **UI Events**: ui.interaction, ui.render, ui.error
