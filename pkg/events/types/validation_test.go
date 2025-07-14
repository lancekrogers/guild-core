// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package types

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lancekrogers/guild/pkg/events"
	"github.com/lancekrogers/guild/pkg/gerror"
)

func TestValidator_PropertyValidators(t *testing.T) {
	tests := []struct {
		name      string
		validator PropertyValidator
		value     interface{}
		wantErr   bool
	}{
		// Email validator tests
		{
			name:      "Valid email",
			validator: &EmailValidator{},
			value:     "test@example.com",
			wantErr:   false,
		},
		{
			name:      "Invalid email - no @",
			validator: &EmailValidator{},
			value:     "testexample.com",
			wantErr:   true,
		},
		{
			name:      "Invalid email - no domain",
			validator: &EmailValidator{},
			value:     "test@",
			wantErr:   true,
		},
		{
			name:      "Invalid email - not string",
			validator: &EmailValidator{},
			value:     123,
			wantErr:   true,
		},

		// URL validator tests
		{
			name:      "Valid HTTP URL",
			validator: &URLValidator{},
			value:     "http://example.com",
			wantErr:   false,
		},
		{
			name:      "Valid HTTPS URL",
			validator: &URLValidator{},
			value:     "https://example.com/path?query=value",
			wantErr:   false,
		},
		{
			name:      "Invalid URL - no protocol",
			validator: &URLValidator{},
			value:     "example.com",
			wantErr:   true,
		},
		{
			name:      "Invalid URL - not string",
			validator: &URLValidator{},
			value:     123,
			wantErr:   true,
		},

		// UUID validator tests
		{
			name:      "Valid UUID",
			validator: &UUIDValidator{},
			value:     "550e8400-e29b-41d4-a716-446655440000",
			wantErr:   false,
		},
		{
			name:      "Invalid UUID - wrong format",
			validator: &UUIDValidator{},
			value:     "550e8400-e29b-41d4-a716",
			wantErr:   true,
		},
		{
			name:      "Invalid UUID - not string",
			validator: &UUIDValidator{},
			value:     123,
			wantErr:   true,
		},

		// SemVer validator tests
		{
			name:      "Valid semver",
			validator: &SemVerValidator{},
			value:     "1.2.3",
			wantErr:   false,
		},
		{
			name:      "Valid semver with pre-release",
			validator: &SemVerValidator{},
			value:     "1.2.3-alpha.1",
			wantErr:   false,
		},
		{
			name:      "Valid semver with metadata",
			validator: &SemVerValidator{},
			value:     "1.2.3+build.123",
			wantErr:   false,
		},
		{
			name:      "Invalid semver",
			validator: &SemVerValidator{},
			value:     "1.2",
			wantErr:   true,
		},

		// Positive number validator tests
		{
			name:      "Valid positive int",
			validator: &PositiveNumberValidator{},
			value:     42,
			wantErr:   false,
		},
		{
			name:      "Valid positive float",
			validator: &PositiveNumberValidator{},
			value:     42.5,
			wantErr:   false,
		},
		{
			name:      "Invalid negative number",
			validator: &PositiveNumberValidator{},
			value:     -42,
			wantErr:   true,
		},
		{
			name:      "Invalid zero",
			validator: &PositiveNumberValidator{},
			value:     0,
			wantErr:   true,
		},
		{
			name:      "Invalid not a number",
			validator: &PositiveNumberValidator{},
			value:     "forty-two",
			wantErr:   true,
		},

		// Percentage validator tests
		{
			name:      "Valid percentage 0",
			validator: &PercentageValidator{},
			value:     0,
			wantErr:   false,
		},
		{
			name:      "Valid percentage 50",
			validator: &PercentageValidator{},
			value:     50.5,
			wantErr:   false,
		},
		{
			name:      "Valid percentage 100",
			validator: &PercentageValidator{},
			value:     100,
			wantErr:   false,
		},
		{
			name:      "Invalid percentage > 100",
			validator: &PercentageValidator{},
			value:     101,
			wantErr:   true,
		},
		{
			name:      "Invalid percentage < 0",
			validator: &PercentageValidator{},
			value:     -1,
			wantErr:   true,
		},

		// Enum validator tests
		{
			name:      "Valid enum value",
			validator: &EnumValidator{Values: []string{"active", "inactive", "pending"}},
			value:     "active",
			wantErr:   false,
		},
		{
			name:      "Invalid enum value",
			validator: &EnumValidator{Values: []string{"active", "inactive", "pending"}},
			value:     "deleted",
			wantErr:   true,
		},
		{
			name:      "Invalid enum - not string",
			validator: &EnumValidator{Values: []string{"active", "inactive", "pending"}},
			value:     123,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.validator.Validate(tt.value)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidator_EventValidation(t *testing.T) {
	registry := NewEventRegistry()
	validator := NewValidator(registry)
	ctx := context.Background()

	// Add custom validation rule for task events
	validator.AddRule("task.created", &TaskValidationRule{})

	tests := []struct {
		name    string
		event   events.CoreEvent
		wantErr bool
	}{
		{
			name: "Valid task event",
			event: events.NewBaseEvent("evt_123", "task.created", "test",
				map[string]interface{}{
					"task_id":  "task123",
					"name":     "Test Task",
					"status":   "pending",
					"priority": "high",
				}),
			wantErr: false,
		},
		{
			name: "Invalid task event - missing task_id",
			event: events.NewBaseEvent("evt_124", "task.created", "test",
				map[string]interface{}{
					"status": "pending",
				}),
			wantErr: true,
		},
		{
			name: "Invalid task event - invalid status",
			event: events.NewBaseEvent("evt_125", "task.created", "test",
				map[string]interface{}{
					"task_id": "task123",
					"status":  "invalid_status",
				}),
			wantErr: true,
		},
		{
			name: "Invalid event ID format",
			event: events.NewBaseEvent("invalid-id", "test.event", "test",
				map[string]interface{}{}),
			wantErr: true,
		},
		{
			name: "Invalid event type format",
			event: events.NewBaseEvent("evt_126", "InvalidType", "test",
				map[string]interface{}{}),
			wantErr: true,
		},
		{
			name: "Event with no source",
			event: events.NewBaseEvent("evt_127", "test.event", "",
				map[string]interface{}{}),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateEvent(ctx, tt.event)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidator_AgentValidation(t *testing.T) {
	registry := NewEventRegistry()
	validator := NewValidator(registry)
	ctx := context.Background()

	// Add agent validation rule
	validator.AddRule("agent.started", &AgentValidationRule{})

	tests := []struct {
		name    string
		event   events.CoreEvent
		wantErr bool
	}{
		{
			name: "Valid agent event",
			event: events.NewBaseEvent("evt_200", "agent.started", "test",
				map[string]interface{}{
					"agent_id":     "agent123",
					"capabilities": []interface{}{"code_review", "testing"},
				}),
			wantErr: false,
		},
		{
			name: "Invalid agent event - missing agent_id",
			event: events.NewBaseEvent("evt_201", "agent.started", "test",
				map[string]interface{}{
					"capabilities": []interface{}{"code_review"},
				}),
			wantErr: true,
		},
		{
			name: "Invalid agent event - invalid capabilities",
			event: events.NewBaseEvent("evt_202", "agent.started", "test",
				map[string]interface{}{
					"agent_id":     "agent123",
					"capabilities": []interface{}{"code_review", 123}, // number instead of string
				}),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateEvent(ctx, tt.event)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestSchemaEvolution_Migration(t *testing.T) {
	registry := NewEventRegistry()
	evolution := NewSchemaEvolution(registry)

	// Add migration from v1 to v2
	evolution.AddMigration("user.created", Migration{
		FromVersion: "1.0.0",
		ToVersion:   "2.0.0",
		Migrate: func(data map[string]interface{}) (map[string]interface{}, error) {
			// Split full_name into first_name and last_name
			if fullName, ok := data["full_name"].(string); ok {
				parts := splitName(fullName)
				data["first_name"] = parts[0]
				data["last_name"] = parts[1]
				delete(data, "full_name")
			}
			return data, nil
		},
	})

	// Create v1 event
	event := events.NewBaseEvent("evt_300", "user.created", "test",
		map[string]interface{}{
			"user_id":   "user123",
			"full_name": "John Doe",
			"email":     "john.doe@example.com",
		})
	event.WithMetadata("schema_version", "1.0.0")

	// Migrate to v2
	migrated, err := evolution.MigrateEvent(event, "2.0.0")
	require.NoError(t, err)
	assert.NotNil(t, migrated)

	// Check migrated data
	data := migrated.GetData()
	assert.Equal(t, "John", data["first_name"])
	assert.Equal(t, "Doe", data["last_name"])
	assert.Nil(t, data["full_name"])
	assert.Equal(t, "user123", data["user_id"])
	assert.Equal(t, "john.doe@example.com", data["email"])
	assert.Equal(t, "2.0.0", migrated.GetMetadata()["schema_version"])

	// Test migration when already at target version
	migrated2, err := evolution.MigrateEvent(migrated, "2.0.0")
	assert.NoError(t, err)
	assert.Equal(t, migrated.GetID(), migrated2.GetID())
}

func TestValidator_CustomPropertyValidators(t *testing.T) {
	registry := NewEventRegistry()
	validator := NewValidator(registry)

	// Add custom validator
	validator.AddPropertyValidator("phone", &phoneValidator{})

	// Test with schema that uses custom validator
	schema := &EventSchema{
		Type:     "contact.created",
		Version:  "1.0.0",
		Required: []string{"phone"},
		Properties: map[string]Property{
			"phone": {
				Type:     "string",
				Required: true,
				Metadata: map[string]interface{}{
					"validator": "phone",
				},
			},
		},
	}
	err := registry.RegisterSchema(schema)
	require.NoError(t, err)

	// This would require extending the validation to use custom validators
	// For now, just test that the validator was registered
	assert.NotNil(t, validator.validators["phone"])
}

// Helper functions

func splitName(fullName string) []string {
	parts := []string{"", ""}
	if fullName != "" {
		nameParts := []rune(fullName)
		for i, r := range nameParts {
			if r == ' ' {
				parts[0] = string(nameParts[:i])
				if i+1 < len(nameParts) {
					parts[1] = string(nameParts[i+1:])
				}
				break
			}
		}
		if parts[0] == "" {
			parts[0] = fullName
		}
	}
	return parts
}

// Custom validator for testing
type phoneValidator struct{}

func (v *phoneValidator) Validate(value interface{}) error {
	str, ok := value.(string)
	if !ok {
		return gerror.New(gerror.ErrCodeValidation, "value must be a string", nil)
	}

	// Simple phone validation (US format)
	if len(str) < 10 {
		return gerror.New(gerror.ErrCodeValidation, "invalid phone number", nil)
	}

	return nil
}
