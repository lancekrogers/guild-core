// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package types

import (
	"context"
	"regexp"
	"strings"

	"github.com/lancekrogers/guild-core/pkg/events"
	"github.com/lancekrogers/guild-core/pkg/gerror"
)

// Validator provides event validation capabilities
type Validator struct {
	registry   *EventRegistry
	rules      map[string]ValidationRule
	validators map[string]PropertyValidator
}

// ValidationRule defines a validation rule for events
type ValidationRule interface {
	Validate(ctx context.Context, event events.CoreEvent) error
}

// PropertyValidator validates individual properties
type PropertyValidator interface {
	Validate(value interface{}) error
}

// NewValidator creates a new event validator
func NewValidator(registry *EventRegistry) *Validator {
	v := &Validator{
		registry:   registry,
		rules:      make(map[string]ValidationRule),
		validators: make(map[string]PropertyValidator),
	}

	// Register default validators
	v.registerDefaultValidators()

	return v
}

// AddRule adds a validation rule for an event type
func (v *Validator) AddRule(eventType string, rule ValidationRule) {
	v.rules[eventType] = rule
}

// AddPropertyValidator adds a property validator
func (v *Validator) AddPropertyValidator(name string, validator PropertyValidator) {
	v.validators[name] = validator
}

// ValidateEvent validates an event
func (v *Validator) ValidateEvent(ctx context.Context, event events.CoreEvent) error {
	if err := ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled")
	}

	// Schema validation
	if v.registry != nil {
		if err := v.registry.ValidateEvent(ctx, event); err != nil {
			return err
		}
	}

	// Type-specific rules
	if rule, exists := v.rules[event.GetType()]; exists {
		if err := rule.Validate(ctx, event); err != nil {
			return err
		}
	}

	// Common validations
	if err := v.validateCommon(event); err != nil {
		return err
	}

	return nil
}

// validateCommon performs common validations
func (v *Validator) validateCommon(event events.CoreEvent) error {
	// Validate event ID format
	if !isValidEventID(event.GetID()) {
		return gerror.New(gerror.ErrCodeValidation, "invalid event ID format", nil).
			WithDetails("id", event.GetID())
	}

	// Validate event type format
	if !isValidEventType(event.GetType()) {
		return gerror.New(gerror.ErrCodeValidation, "invalid event type format", nil).
			WithDetails("type", event.GetType())
	}

	// Validate source
	if event.GetSource() == "" {
		return gerror.New(gerror.ErrCodeValidation, "event source is required", nil)
	}

	// Validate timestamp
	if event.GetTimestamp().IsZero() {
		return gerror.New(gerror.ErrCodeValidation, "event timestamp is required", nil)
	}

	return nil
}

// registerDefaultValidators registers default property validators
func (v *Validator) registerDefaultValidators() {
	// String validators
	v.AddPropertyValidator("email", &EmailValidator{})
	v.AddPropertyValidator("url", &URLValidator{})
	v.AddPropertyValidator("uuid", &UUIDValidator{})
	v.AddPropertyValidator("semver", &SemVerValidator{})

	// Numeric validators
	v.AddPropertyValidator("positive", &PositiveNumberValidator{})
	v.AddPropertyValidator("percentage", &PercentageValidator{})

	// Custom validators
	v.AddPropertyValidator("task_status", &EnumValidator{
		Values: []string{"pending", "in_progress", "completed", "failed", "cancelled"},
	})
	v.AddPropertyValidator("priority", &EnumValidator{
		Values: []string{"low", "normal", "high", "critical"},
	})
	v.AddPropertyValidator("severity", &EnumValidator{
		Values: []string{"debug", "info", "warning", "error", "critical"},
	})
}

// Common property validators

// EmailValidator validates email addresses
type EmailValidator struct{}

func (v *EmailValidator) Validate(value interface{}) error {
	str, ok := value.(string)
	if !ok {
		return gerror.New(gerror.ErrCodeValidation, "value must be a string", nil)
	}

	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	if !emailRegex.MatchString(str) {
		return gerror.New(gerror.ErrCodeValidation, "invalid email format", nil).
			WithDetails("value", str)
	}

	return nil
}

// URLValidator validates URLs
type URLValidator struct{}

func (v *URLValidator) Validate(value interface{}) error {
	str, ok := value.(string)
	if !ok {
		return gerror.New(gerror.ErrCodeValidation, "value must be a string", nil)
	}

	urlRegex := regexp.MustCompile(`^(https?|ftp)://[^\s/$.?#].[^\s]*$`)
	if !urlRegex.MatchString(str) {
		return gerror.New(gerror.ErrCodeValidation, "invalid URL format", nil).
			WithDetails("value", str)
	}

	return nil
}

// UUIDValidator validates UUIDs
type UUIDValidator struct{}

func (v *UUIDValidator) Validate(value interface{}) error {
	str, ok := value.(string)
	if !ok {
		return gerror.New(gerror.ErrCodeValidation, "value must be a string", nil)
	}

	uuidRegex := regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)
	if !uuidRegex.MatchString(str) {
		return gerror.New(gerror.ErrCodeValidation, "invalid UUID format", nil).
			WithDetails("value", str)
	}

	return nil
}

// SemVerValidator validates semantic version strings
type SemVerValidator struct{}

func (v *SemVerValidator) Validate(value interface{}) error {
	str, ok := value.(string)
	if !ok {
		return gerror.New(gerror.ErrCodeValidation, "value must be a string", nil)
	}

	semverRegex := regexp.MustCompile(`^(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)(?:-((?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*)(?:\.(?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*))*))?(?:\+([0-9a-zA-Z-]+(?:\.[0-9a-zA-Z-]+)*))?$`)
	if !semverRegex.MatchString(str) {
		return gerror.New(gerror.ErrCodeValidation, "invalid semantic version format", nil).
			WithDetails("value", str)
	}

	return nil
}

// PositiveNumberValidator validates positive numbers
type PositiveNumberValidator struct{}

func (v *PositiveNumberValidator) Validate(value interface{}) error {
	switch num := value.(type) {
	case int, int8, int16, int32, int64:
		if num.(int) <= 0 {
			return gerror.New(gerror.ErrCodeValidation, "value must be positive", nil).
				WithDetails("value", num)
		}
	case uint, uint8, uint16, uint32, uint64:
		// Unsigned types are always positive
	case float32:
		if num <= 0 {
			return gerror.New(gerror.ErrCodeValidation, "value must be positive", nil).
				WithDetails("value", num)
		}
	case float64:
		if num <= 0 {
			return gerror.New(gerror.ErrCodeValidation, "value must be positive", nil).
				WithDetails("value", num)
		}
	default:
		return gerror.New(gerror.ErrCodeValidation, "value must be a number", nil)
	}

	return nil
}

// PercentageValidator validates percentage values (0-100)
type PercentageValidator struct{}

func (v *PercentageValidator) Validate(value interface{}) error {
	var num float64

	switch n := value.(type) {
	case int:
		num = float64(n)
	case int32:
		num = float64(n)
	case int64:
		num = float64(n)
	case float32:
		num = float64(n)
	case float64:
		num = n
	default:
		return gerror.New(gerror.ErrCodeValidation, "value must be a number", nil)
	}

	if num < 0 || num > 100 {
		return gerror.New(gerror.ErrCodeValidation, "percentage must be between 0 and 100", nil).
			WithDetails("value", num)
	}

	return nil
}

// EnumValidator validates against a set of allowed values
type EnumValidator struct {
	Values []string
}

func (v *EnumValidator) Validate(value interface{}) error {
	str, ok := value.(string)
	if !ok {
		return gerror.New(gerror.ErrCodeValidation, "value must be a string", nil)
	}

	for _, allowed := range v.Values {
		if str == allowed {
			return nil
		}
	}

	return gerror.New(gerror.ErrCodeValidation, "value not in allowed set", nil).
		WithDetails("value", str).
		WithDetails("allowed", strings.Join(v.Values, ", "))
}

// Validation rules for specific event types

// TaskValidationRule validates task events
type TaskValidationRule struct{}

func (r *TaskValidationRule) Validate(ctx context.Context, event events.CoreEvent) error {
	data := event.GetData()

	// Task ID is required
	if _, ok := data["task_id"].(string); !ok {
		return gerror.New(gerror.ErrCodeValidation, "task_id is required", nil)
	}

	// If status is provided, validate it
	if status, ok := data["status"].(string); ok {
		validator := &EnumValidator{
			Values: []string{"pending", "in_progress", "completed", "failed", "cancelled"},
		}
		if err := validator.Validate(status); err != nil {
			return err
		}
	}

	// If priority is provided, validate it
	if priority, ok := data["priority"].(string); ok {
		validator := &EnumValidator{
			Values: []string{"low", "normal", "high", "critical"},
		}
		if err := validator.Validate(priority); err != nil {
			return err
		}
	}

	return nil
}

// AgentValidationRule validates agent events
type AgentValidationRule struct{}

func (r *AgentValidationRule) Validate(ctx context.Context, event events.CoreEvent) error {
	data := event.GetData()

	// Agent ID is required
	if _, ok := data["agent_id"].(string); !ok {
		return gerror.New(gerror.ErrCodeValidation, "agent_id is required", nil)
	}

	// Validate capabilities if present
	if caps, ok := data["capabilities"].([]interface{}); ok {
		for i, cap := range caps {
			if _, ok := cap.(string); !ok {
				return gerror.New(gerror.ErrCodeValidation, "capability must be a string", nil).
					WithDetails("index", i)
			}
		}
	}

	return nil
}

// Helper functions

func isValidEventID(id string) bool {
	// Event ID should follow a specific pattern
	return strings.HasPrefix(id, "evt_") && len(id) > 4
}

func isValidEventType(eventType string) bool {
	// Event type should be in format category.action
	parts := strings.Split(eventType, ".")
	if len(parts) < 2 {
		return false
	}

	// Each part should be lowercase with optional underscores
	for _, part := range parts {
		if !regexp.MustCompile(`^[a-z][a-z0-9_]*$`).MatchString(part) {
			return false
		}
	}

	return true
}

// SchemaEvolution handles schema versioning and migration
type SchemaEvolution struct {
	registry   *EventRegistry
	migrations map[string][]Migration
}

// Migration defines a schema migration
type Migration struct {
	FromVersion string
	ToVersion   string
	Migrate     func(data map[string]interface{}) (map[string]interface{}, error)
}

// NewSchemaEvolution creates a new schema evolution handler
func NewSchemaEvolution(registry *EventRegistry) *SchemaEvolution {
	return &SchemaEvolution{
		registry:   registry,
		migrations: make(map[string][]Migration),
	}
}

// AddMigration adds a migration for an event type
func (se *SchemaEvolution) AddMigration(eventType string, migration Migration) {
	se.migrations[eventType] = append(se.migrations[eventType], migration)
}

// MigrateEvent migrates an event to the latest schema version
func (se *SchemaEvolution) MigrateEvent(event events.CoreEvent, targetVersion string) (events.CoreEvent, error) {
	// Get current version from metadata
	currentVersion := "1.0.0"
	if metadata := event.GetMetadata(); metadata != nil {
		if v, ok := metadata["schema_version"].(string); ok {
			currentVersion = v
		}
	}

	// If already at target version, return as-is
	if currentVersion == targetVersion {
		return event, nil
	}

	// Find migration path
	migrations, exists := se.migrations[event.GetType()]
	if !exists {
		return nil, gerror.New(gerror.ErrCodeNotFound, "no migrations found for event type", nil).
			WithDetails("type", event.GetType())
	}

	// Apply migrations in sequence
	data := event.GetData()
	for _, migration := range migrations {
		if migration.FromVersion == currentVersion {
			migratedData, err := migration.Migrate(data)
			if err != nil {
				return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "migration failed").
					WithDetails("from", migration.FromVersion).
					WithDetails("to", migration.ToVersion)
			}
			data = migratedData
			currentVersion = migration.ToVersion

			if currentVersion == targetVersion {
				break
			}
		}
	}

	// Create new event with migrated data
	newEvent := events.NewBaseEvent(
		event.GetID(),
		event.GetType(),
		event.GetSource(),
		data,
	)

	// Copy metadata and update version
	if metadata := event.GetMetadata(); metadata != nil {
		for k, v := range metadata {
			newEvent.WithMetadata(k, v)
		}
	}
	newEvent.WithMetadata("schema_version", targetVersion)

	return newEvent, nil
}
