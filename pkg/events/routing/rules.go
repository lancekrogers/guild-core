// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package routing

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/guild-framework/guild-core/pkg/events"
	"github.com/guild-framework/guild-core/pkg/gerror"
)

// EventTypeRule matches events by type
type EventTypeRule struct {
	Types    []string
	Patterns []string
}

// Matches checks if the event type matches
func (r *EventTypeRule) Matches(ctx context.Context, event events.CoreEvent) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled")
	}

	eventType := event.GetType()

	// Check exact matches
	for _, t := range r.Types {
		if eventType == t {
			return true, nil
		}
	}

	// Check patterns
	for _, pattern := range r.Patterns {
		if matchesPattern(eventType, pattern) {
			return true, nil
		}
	}

	return false, nil
}

// SourceRule matches events by source
type SourceRule struct {
	Sources  []string
	Patterns []string
}

// Matches checks if the event source matches
func (r *SourceRule) Matches(ctx context.Context, event events.CoreEvent) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled")
	}

	source := event.GetSource()

	// Check exact matches
	for _, s := range r.Sources {
		if source == s {
			return true, nil
		}
	}

	// Check patterns
	for _, pattern := range r.Patterns {
		if matchesPattern(source, pattern) {
			return true, nil
		}
	}

	return false, nil
}

// DataRule matches events by data content
type DataRule struct {
	FieldName string
	Operator  string // eq, ne, gt, lt, gte, lte, contains, regex
	Value     interface{}
	regex     *regexp.Regexp
}

// Matches checks if the event data matches
func (r *DataRule) Matches(ctx context.Context, event events.CoreEvent) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled")
	}

	data := event.GetData()
	if data == nil {
		return false, nil
	}

	fieldValue, exists := getNestedField(data, r.FieldName)
	if !exists {
		return false, nil
	}

	switch r.Operator {
	case "eq", "equals":
		return compareEqual(fieldValue, r.Value), nil
	case "ne", "not_equals":
		return !compareEqual(fieldValue, r.Value), nil
	case "gt", "greater_than":
		return compareGreater(fieldValue, r.Value), nil
	case "lt", "less_than":
		return compareLess(fieldValue, r.Value), nil
	case "gte", "greater_or_equal":
		return !compareLess(fieldValue, r.Value), nil
	case "lte", "less_or_equal":
		return !compareGreater(fieldValue, r.Value), nil
	case "contains":
		return compareContains(fieldValue, r.Value), nil
	case "regex":
		if r.regex == nil {
			pattern, ok := r.Value.(string)
			if !ok {
				return false, gerror.New(gerror.ErrCodeValidation, "regex value must be string", nil)
			}
			regex, err := regexp.Compile(pattern)
			if err != nil {
				return false, gerror.Wrap(err, gerror.ErrCodeValidation, "invalid regex pattern")
			}
			r.regex = regex
		}
		str, ok := fieldValue.(string)
		if !ok {
			return false, nil
		}
		return r.regex.MatchString(str), nil
	default:
		return false, gerror.New(gerror.ErrCodeValidation, "unknown operator", nil).
			WithDetails("operator", r.Operator)
	}
}

// MetadataRule matches events by metadata
type MetadataRule struct {
	Key      string
	Operator string
	Value    interface{}
}

// Matches checks if the event metadata matches
func (r *MetadataRule) Matches(ctx context.Context, event events.CoreEvent) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled")
	}

	metadata := event.GetMetadata()
	if metadata == nil {
		return false, nil
	}

	value, exists := metadata[r.Key]
	if !exists && r.Operator != "exists" {
		return false, nil
	}

	switch r.Operator {
	case "exists":
		return exists, nil
	case "eq", "equals":
		return compareEqual(value, r.Value), nil
	case "ne", "not_equals":
		return !compareEqual(value, r.Value), nil
	default:
		return false, gerror.New(gerror.ErrCodeValidation, "unknown operator", nil).
			WithDetails("operator", r.Operator)
	}
}

// CompositeRule combines multiple rules
type CompositeRule struct {
	Rules    []RoutingRule
	Operator string // "and", "or", "not"
}

// Matches checks if the composite rule matches
func (r *CompositeRule) Matches(ctx context.Context, event events.CoreEvent) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled")
	}

	if len(r.Rules) == 0 {
		return true, nil
	}

	switch strings.ToLower(r.Operator) {
	case "and":
		for _, rule := range r.Rules {
			matches, err := rule.Matches(ctx, event)
			if err != nil {
				return false, err
			}
			if !matches {
				return false, nil
			}
		}
		return true, nil

	case "or":
		for _, rule := range r.Rules {
			matches, err := rule.Matches(ctx, event)
			if err != nil {
				return false, err
			}
			if matches {
				return true, nil
			}
		}
		return false, nil

	case "not":
		if len(r.Rules) != 1 {
			return false, gerror.New(gerror.ErrCodeValidation, "NOT operator requires exactly one rule", nil)
		}
		matches, err := r.Rules[0].Matches(ctx, event)
		return !matches, err

	default:
		return false, gerror.New(gerror.ErrCodeValidation, "unknown operator", nil).
			WithDetails("operator", r.Operator)
	}
}

// TimeWindowRule matches events within a time window
type TimeWindowRule struct {
	StartTime  string // HH:MM format
	EndTime    string // HH:MM format
	DaysOfWeek []string
	TimeZone   string
}

// Matches checks if the event is within the time window
func (r *TimeWindowRule) Matches(ctx context.Context, event events.CoreEvent) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled")
	}

	// TODO: Implement time window matching logic
	// For now, always match
	return true, nil
}

// FrequencyRule limits event frequency
type FrequencyRule struct {
	MaxEvents  int
	TimeWindow time.Duration
	counter    map[string][]time.Time
	mu         sync.RWMutex
}

// Matches checks if the event is within frequency limits
func (r *FrequencyRule) Matches(ctx context.Context, event events.CoreEvent) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if r.counter == nil {
		r.counter = make(map[string][]time.Time)
	}

	key := event.GetType() + ":" + event.GetSource()
	now := time.Now()
	cutoff := now.Add(-r.TimeWindow)

	// Remove old entries
	var validTimes []time.Time
	for _, t := range r.counter[key] {
		if t.After(cutoff) {
			validTimes = append(validTimes, t)
		}
	}

	// Check if we're within limits
	if len(validTimes) >= r.MaxEvents {
		r.counter[key] = validTimes
		return false, nil
	}

	// Add current event
	r.counter[key] = append(validTimes, now)
	return true, nil
}

// Helper functions

func matchesPattern(str, pattern string) bool {
	// Simple wildcard matching
	if pattern == "*" {
		return true
	}

	// Convert wildcard pattern to regex
	regexPattern := "^" + strings.ReplaceAll(
		strings.ReplaceAll(pattern, ".", "\\."),
		"*", ".*",
	) + "$"

	matched, _ := regexp.MatchString(regexPattern, str)
	return matched
}

func getNestedField(data map[string]interface{}, fieldPath string) (interface{}, bool) {
	parts := strings.Split(fieldPath, ".")
	current := data

	for i, part := range parts {
		value, exists := current[part]
		if !exists {
			return nil, false
		}

		// If this is the last part, return the value
		if i == len(parts)-1 {
			return value, true
		}

		// Otherwise, try to navigate deeper
		next, ok := value.(map[string]interface{})
		if !ok {
			return nil, false
		}
		current = next
	}

	return nil, false
}

func compareEqual(a, b interface{}) bool {
	// Handle nil cases
	if a == nil || b == nil {
		return a == b
	}

	// Try direct comparison
	if a == b {
		return true
	}

	// Handle numeric comparisons
	aFloat, aOk := toFloat64(a)
	bFloat, bOk := toFloat64(b)
	if aOk && bOk {
		return aFloat == bFloat
	}

	// String comparison
	return toString(a) == toString(b)
}

func compareGreater(a, b interface{}) bool {
	aFloat, aOk := toFloat64(a)
	bFloat, bOk := toFloat64(b)
	if aOk && bOk {
		return aFloat > bFloat
	}
	return false
}

func compareLess(a, b interface{}) bool {
	aFloat, aOk := toFloat64(a)
	bFloat, bOk := toFloat64(b)
	if aOk && bOk {
		return aFloat < bFloat
	}
	return false
}

func compareContains(a, b interface{}) bool {
	aStr := toString(a)
	bStr := toString(b)
	return strings.Contains(aStr, bStr)
}

func toFloat64(v interface{}) (float64, bool) {
	switch val := v.(type) {
	case float64:
		return val, true
	case float32:
		return float64(val), true
	case int:
		return float64(val), true
	case int32:
		return float64(val), true
	case int64:
		return float64(val), true
	case uint:
		return float64(val), true
	case uint32:
		return float64(val), true
	case uint64:
		return float64(val), true
	default:
		return 0, false
	}
}

func toString(v interface{}) string {
	return fmt.Sprintf("%v", v)
}
