// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package types

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lancekrogers/guild-core/pkg/events"
)

func TestEventRegistry_RegisterSchema(t *testing.T) {
	registry := NewEventRegistry()

	schema := &EventSchema{
		Type:        "test.event",
		Version:     "1.0.0",
		Description: "Test event",
		Required:    []string{"field1"},
		Properties: map[string]Property{
			"field1": {
				Type:        "string",
				Description: "Field 1",
				Required:    true,
			},
			"field2": {
				Type:        "number",
				Description: "Field 2",
				Required:    false,
			},
		},
	}

	// Register schema
	err := registry.RegisterSchema(schema)
	assert.NoError(t, err)

	// Register duplicate should fail
	err = registry.RegisterSchema(schema)
	assert.Error(t, err)

	// Register with missing type should fail
	err = registry.RegisterSchema(&EventSchema{Version: "1.0.0"})
	assert.Error(t, err)

	// Register with missing version should fail
	err = registry.RegisterSchema(&EventSchema{Type: "test.event"})
	assert.Error(t, err)
}

func TestEventRegistry_GetSchema(t *testing.T) {
	registry := NewEventRegistry()

	schema := &EventSchema{
		Type:        "test.event",
		Version:     "1.0.0",
		Description: "Test event",
	}

	err := registry.RegisterSchema(schema)
	require.NoError(t, err)

	// Get existing schema
	retrieved, err := registry.GetSchema("test.event", "1.0.0")
	assert.NoError(t, err)
	assert.Equal(t, schema.Type, retrieved.Type)
	assert.Equal(t, schema.Version, retrieved.Version)

	// Get non-existent schema
	_, err = registry.GetSchema("non.existent", "1.0.0")
	assert.Error(t, err)
}

func TestEventRegistry_ValidateEvent(t *testing.T) {
	registry := NewEventRegistry()
	ctx := context.Background()

	// Register schema with validation rules
	schema := &EventSchema{
		Type:        "user.created",
		Version:     "1.0.0",
		Description: "User created event",
		Required:    []string{"user_id", "email"},
		Properties: map[string]Property{
			"user_id": {
				Type:     "string",
				Required: true,
			},
			"email": {
				Type:     "string",
				Required: true,
			},
			"age": {
				Type:     "integer",
				Required: false,
			},
			"status": {
				Type:     "string",
				Required: false,
				Enum:     []interface{}{"active", "inactive", "pending"},
			},
		},
	}

	err := registry.RegisterSchema(schema)
	require.NoError(t, err)

	tests := []struct {
		name    string
		event   events.CoreEvent
		wantErr bool
	}{
		{
			name: "Valid event",
			event: events.NewBaseEvent("evt_123", "user.created", "test",
				map[string]interface{}{
					"user_id": "user123",
					"email":   "test@example.com",
					"age":     25,
					"status":  "active",
				}),
			wantErr: false,
		},
		{
			name: "Missing required field",
			event: events.NewBaseEvent("evt_124", "user.created", "test",
				map[string]interface{}{
					"user_id": "user123",
					// email is missing
				}),
			wantErr: true,
		},
		{
			name: "Invalid enum value",
			event: events.NewBaseEvent("evt_125", "user.created", "test",
				map[string]interface{}{
					"user_id": "user123",
					"email":   "test@example.com",
					"status":  "invalid_status",
				}),
			wantErr: true,
		},
		{
			name: "Wrong type",
			event: events.NewBaseEvent("evt_126", "user.created", "test",
				map[string]interface{}{
					"user_id": "user123",
					"email":   "test@example.com",
					"age":     "twenty-five", // should be integer
				}),
			wantErr: true,
		},
		{
			name: "No schema (should pass)",
			event: events.NewBaseEvent("evt_127", "unregistered.event", "test",
				map[string]interface{}{
					"any": "data",
				}),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Add version metadata for events with schema
			if tt.event.GetType() == "user.created" {
				tt.event.(*events.BaseEvent).WithMetadata("schema_version", "1.0.0")
			}

			err := registry.ValidateEvent(ctx, tt.event)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestEventRegistry_Factory(t *testing.T) {
	registry := NewEventRegistry()

	// Register factory
	factory := func(data map[string]interface{}) (events.CoreEvent, error) {
		return events.NewBaseEvent("custom_id", "custom.event", "factory", data), nil
	}

	err := registry.RegisterFactory("custom.event", factory)
	assert.NoError(t, err)

	// Create event using factory
	event, err := registry.CreateEvent("custom.event", map[string]interface{}{
		"field": "value",
	})
	assert.NoError(t, err)
	assert.Equal(t, "custom_id", event.GetID())
	assert.Equal(t, "custom.event", event.GetType())
	assert.Equal(t, "factory", event.GetSource())

	// Create event without factory (fallback)
	event, err = registry.CreateEvent("no.factory", map[string]interface{}{
		"field": "value",
	})
	assert.NoError(t, err)
	assert.Equal(t, "no.factory", event.GetType())
	assert.Equal(t, "registry", event.GetSource())
}

func TestEventRegistry_Versions(t *testing.T) {
	registry := NewEventRegistry()

	// Register multiple versions
	versions := []string{"1.0.0", "1.1.0", "2.0.0"}
	for _, v := range versions {
		err := registry.RegisterSchema(&EventSchema{
			Type:    "versioned.event",
			Version: v,
		})
		require.NoError(t, err)
	}

	// Get versions
	retrieved := registry.GetVersions("versioned.event")
	assert.ElementsMatch(t, versions, retrieved)

	// Get latest version
	latest, err := registry.GetLatestSchema("versioned.event")
	assert.NoError(t, err)
	assert.Equal(t, "2.0.0", latest.Version)

	// Get versions for non-existent type
	retrieved = registry.GetVersions("non.existent")
	assert.Empty(t, retrieved)
}

func TestEventRegistry_ExportImport(t *testing.T) {
	// Create registry without built-in types for testing
	registry1 := &EventRegistry{
		schemas:   make(map[string]*EventSchema),
		factories: make(map[string]EventFactory),
		versions:  make(map[string][]string),
	}

	// Register some schemas
	schemas := []*EventSchema{
		{
			Type:        "export.test1",
			Version:     "1.0.0",
			Description: "Test schema 1",
		},
		{
			Type:        "export.test2",
			Version:     "1.0.0",
			Description: "Test schema 2",
		},
	}

	for _, schema := range schemas {
		err := registry1.RegisterSchema(schema)
		require.NoError(t, err)
	}

	// Export schemas
	exported, err := registry1.ExportSchemas()
	assert.NoError(t, err)
	assert.NotEmpty(t, exported)

	// Import into new registry (also without built-in types)
	registry2 := &EventRegistry{
		schemas:   make(map[string]*EventSchema),
		factories: make(map[string]EventFactory),
		versions:  make(map[string][]string),
	}
	err = registry2.ImportSchemas(exported)
	assert.NoError(t, err)

	// Verify imported schemas
	for _, schema := range schemas {
		imported, err := registry2.GetSchema(schema.Type, schema.Version)
		assert.NoError(t, err)
		assert.NotNil(t, imported)
		assert.Equal(t, schema.Type, imported.Type)
		assert.Equal(t, schema.Version, imported.Version)
		assert.Equal(t, schema.Description, imported.Description)
	}
}

func TestEventRegistry_NestedValidation(t *testing.T) {
	registry := NewEventRegistry()
	ctx := context.Background()

	// Register schema with nested objects
	schema := &EventSchema{
		Type:     "order.created",
		Version:  "1.0.0",
		Required: []string{"order_id", "customer"},
		Properties: map[string]Property{
			"order_id": {
				Type:     "string",
				Required: true,
			},
			"customer": {
				Type:     "object",
				Required: true,
				Properties: map[string]Property{
					"id": {
						Type:     "string",
						Required: true,
					},
					"email": {
						Type:     "string",
						Required: true,
					},
					"address": {
						Type:     "object",
						Required: false,
						Properties: map[string]Property{
							"street": {Type: "string"},
							"city":   {Type: "string"},
							"zip":    {Type: "string"},
						},
					},
				},
			},
			"items": {
				Type:     "array",
				Required: false,
				Items: &Property{
					Type: "object",
					Properties: map[string]Property{
						"product_id": {Type: "string", Required: true},
						"quantity":   {Type: "integer", Required: true},
						"price":      {Type: "number", Required: true},
					},
				},
			},
		},
	}

	err := registry.RegisterSchema(schema)
	require.NoError(t, err)

	// Valid nested event
	validEvent := events.NewBaseEvent("evt_200", "order.created", "test",
		map[string]interface{}{
			"order_id": "order123",
			"customer": map[string]interface{}{
				"id":    "cust123",
				"email": "customer@example.com",
				"address": map[string]interface{}{
					"street": "123 Main St",
					"city":   "Boston",
					"zip":    "02101",
				},
			},
			"items": []interface{}{
				map[string]interface{}{
					"product_id": "prod1",
					"quantity":   2,
					"price":      19.99,
				},
				map[string]interface{}{
					"product_id": "prod2",
					"quantity":   1,
					"price":      39.99,
				},
			},
		})

	validEvent.WithMetadata("schema_version", "1.0.0")
	err = registry.ValidateEvent(ctx, validEvent)
	assert.NoError(t, err)

	// Invalid nested event (missing required nested field)
	invalidEvent := events.NewBaseEvent("evt_201", "order.created", "test",
		map[string]interface{}{
			"order_id": "order124",
			"customer": map[string]interface{}{
				"id": "cust124",
				// email is missing
			},
		})

	invalidEvent.WithMetadata("schema_version", "1.0.0")
	err = registry.ValidateEvent(ctx, invalidEvent)
	assert.Error(t, err)
}
