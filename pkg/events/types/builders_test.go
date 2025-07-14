// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package types

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTypedEventBuilder(t *testing.T) {
	registry := NewEventRegistry()

	// Register a schema
	schema := &EventSchema{
		Type:        "test.event",
		Version:     "1.0.0",
		Description: "Test event",
		Required:    []string{"required_field"},
		Properties: map[string]Property{
			"required_field": {
				Type:     "string",
				Required: true,
			},
			"optional_field": {
				Type:     "string",
				Required: false,
			},
		},
	}
	err := registry.RegisterSchema(schema)
	require.NoError(t, err)

	// Build valid event
	builder := NewTypedEventBuilder("test.event", registry)
	event, err := builder.
		WithSource("test-source").
		WithData("required_field", "value").
		WithData("optional_field", "optional").
		WithMetadata("custom", "metadata").
		Build()

	assert.NoError(t, err)
	assert.NotNil(t, event)
	assert.Equal(t, "test.event", event.GetType())
	assert.Equal(t, "test-source", event.GetSource())
	assert.Equal(t, "value", event.GetData()["required_field"])
	assert.Equal(t, "optional", event.GetData()["optional_field"])
	assert.Equal(t, "metadata", event.GetMetadata()["custom"])
	assert.Equal(t, "1.0.0", event.GetMetadata()["schema_version"])

	// Build invalid event (missing required field)
	builder2 := NewTypedEventBuilder("test.event", registry)
	_, err = builder2.
		WithSource("test-source").
		WithData("optional_field", "optional").
		Build()

	assert.Error(t, err)
}

func TestTaskEventBuilder(t *testing.T) {
	registry := NewEventRegistry()

	builder := NewTaskEventBuilder("task.created", registry)
	event, err := builder.
		WithTaskID("task123").
		WithName("Test Task").
		WithDescription("This is a test task").
		WithStatus("pending").
		WithPriority("high").
		WithAssignee("user123").
		Build()

	require.NoError(t, err)
	assert.NotNil(t, event)

	data := event.GetData()
	assert.Equal(t, "task123", data["task_id"])
	assert.Equal(t, "Test Task", data["name"])
	assert.Equal(t, "This is a test task", data["description"])
	assert.Equal(t, "pending", data["status"])
	assert.Equal(t, "high", data["priority"])
	assert.Equal(t, "user123", data["assignee"])
}

func TestAgentEventBuilder(t *testing.T) {
	registry := NewEventRegistry()

	builder := NewAgentEventBuilder("agent.started", registry)
	event, err := builder.
		WithAgentID("agent123").
		WithAgentName("Test Agent").
		WithCapabilities([]string{"code_review", "testing", "deployment"}).
		WithStatus("active").
		Build()

	require.NoError(t, err)
	assert.NotNil(t, event)

	data := event.GetData()
	assert.Equal(t, "agent123", data["agent_id"])
	assert.Equal(t, "Test Agent", data["agent_name"])
	assert.Equal(t, "active", data["status"])

	capabilities := data["capabilities"].([]interface{})
	assert.Len(t, capabilities, 3)
	assert.Contains(t, capabilities, "code_review")
	assert.Contains(t, capabilities, "testing")
	assert.Contains(t, capabilities, "deployment")
}

func TestSystemEventBuilder(t *testing.T) {
	registry := NewEventRegistry()

	metrics := map[string]interface{}{
		"cpu_usage":    45.5,
		"memory_usage": 78.2,
		"disk_usage":   60.0,
	}

	builder := NewSystemEventBuilder("system.alert", registry)
	event, err := builder.
		WithComponent("monitoring").
		WithSeverity("warning").
		WithMessage("High memory usage detected").
		WithMetrics(metrics).
		Build()

	require.NoError(t, err)
	assert.NotNil(t, event)

	data := event.GetData()
	assert.Equal(t, "monitoring", data["component"])
	assert.Equal(t, "warning", data["severity"])
	assert.Equal(t, "High memory usage detected", data["message"])
	assert.Equal(t, metrics, data["metrics"])
}

func TestCommissionEventBuilder(t *testing.T) {
	registry := NewEventRegistry()

	builder := NewCommissionEventBuilder("commission.updated", registry)
	event, err := builder.
		WithCommissionID("comm123").
		WithTitle("Build Feature X").
		WithObjective("Implement the new feature X with full test coverage").
		WithProgress(75.5).
		WithStatus("in_progress").
		Build()

	require.NoError(t, err)
	assert.NotNil(t, event)

	data := event.GetData()
	assert.Equal(t, "comm123", data["commission_id"])
	assert.Equal(t, "Build Feature X", data["title"])
	assert.Equal(t, "Implement the new feature X with full test coverage", data["objective"])
	assert.Equal(t, 75.5, data["progress"])
	assert.Equal(t, "in_progress", data["status"])
}

func TestMemoryEventBuilder(t *testing.T) {
	registry := NewEventRegistry()

	embedding := []float64{0.1, 0.2, 0.3, 0.4, 0.5}

	builder := NewMemoryEventBuilder("memory.stored", registry)
	event, err := builder.
		WithOperation("store").
		WithMemoryType("conversation").
		WithContent("User asked about event system architecture").
		WithEmbedding(embedding).
		WithSimilarity(0.95).
		Build()

	require.NoError(t, err)
	assert.NotNil(t, event)

	data := event.GetData()
	assert.Equal(t, "store", data["operation"])
	assert.Equal(t, "conversation", data["memory_type"])
	assert.Equal(t, "User asked about event system architecture", data["content"])
	assert.Equal(t, 0.95, data["similarity"])

	embeddingData := data["embedding"].([]interface{})
	assert.Len(t, embeddingData, 5)
	assert.Equal(t, 0.1, embeddingData[0])
}

func TestUIEventBuilder(t *testing.T) {
	registry := NewEventRegistry()

	uiData := map[string]interface{}{
		"theme":    "dark",
		"language": "en",
		"viewport": map[string]interface{}{
			"width":  1920,
			"height": 1080,
		},
	}

	builder := NewUIEventBuilder("ui.interaction", registry)
	event, err := builder.
		WithComponent("settings_panel").
		WithAction("click").
		WithUserID("user123").
		WithSessionID("session456").
		WithUIData(uiData).
		Build()

	require.NoError(t, err)
	assert.NotNil(t, event)

	data := event.GetData()
	assert.Equal(t, "settings_panel", data["component"])
	assert.Equal(t, "click", data["action"])
	assert.Equal(t, "user123", data["user_id"])
	assert.Equal(t, "session456", data["session_id"])
	assert.Equal(t, "dark", data["theme"])
	assert.Equal(t, "en", data["language"])
	assert.NotNil(t, data["viewport"])
}

func TestEventBuilderWithoutRegistry(t *testing.T) {
	// Test building without registry (no validation)
	builder := NewTypedEventBuilder("test.event", nil)
	event, err := builder.
		WithSource("test").
		WithData("any_field", "any_value").
		Build()

	require.NoError(t, err)
	assert.NotNil(t, event)
	assert.Equal(t, "test.event", event.GetType())
	assert.Equal(t, "any_value", event.GetData()["any_field"])
}

func TestEventBuilderValidation(t *testing.T) {
	registry := NewEventRegistry()

	// Register schema with specific validation
	schema := &EventSchema{
		Type:     "validated.event",
		Version:  "1.0.0",
		Required: []string{"email", "status"},
		Properties: map[string]Property{
			"email": {
				Type:     "string",
				Required: true,
			},
			"status": {
				Type:     "string",
				Required: true,
				Enum:     []interface{}{"active", "inactive"},
			},
		},
	}
	err := registry.RegisterSchema(schema)
	require.NoError(t, err)

	// Invalid event - missing required fields
	builder := NewTypedEventBuilder("validated.event", registry)
	_, err = builder.Build()
	assert.Error(t, err)

	// Invalid event - wrong enum value
	builder2 := NewTypedEventBuilder("validated.event", registry)
	_, err = builder2.
		WithData("email", "test@example.com").
		WithData("status", "invalid_status").
		Build()
	assert.Error(t, err)

	// Valid event
	builder3 := NewTypedEventBuilder("validated.event", registry)
	event, err := builder3.
		WithData("email", "test@example.com").
		WithData("status", "active").
		Build()
	assert.NoError(t, err)
	assert.NotNil(t, event)
}
