// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package routing

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/lancekrogers/guild/pkg/events"
	"github.com/lancekrogers/guild/pkg/gerror"
)

// Transformer provides event transformation capabilities
type Transformer struct {
	transforms []TransformFunc
}

// NewTransformer creates a new transformer
func NewTransformer(transforms ...TransformFunc) *Transformer {
	return &Transformer{
		transforms: transforms,
	}
}

// Transform applies all transformations to an event
func (t *Transformer) Transform(ctx context.Context, event events.CoreEvent) (events.CoreEvent, error) {
	if err := ctx.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled")
	}

	result := event
	for i, transform := range t.transforms {
		transformed, err := transform(ctx, result)
		if err != nil {
			return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "transformation failed").
				WithDetails("transform_index", i)
		}
		result = transformed
	}

	return result, nil
}

// Common transformers

// EnrichmentTransform adds data to events
func EnrichmentTransform(enrichments map[string]interface{}) TransformFunc {
	return func(ctx context.Context, event events.CoreEvent) (events.CoreEvent, error) {
		if err := ctx.Err(); err != nil {
			return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled")
		}

		// Clone event data
		data := make(map[string]interface{})
		for k, v := range event.GetData() {
			data[k] = v
		}

		// Add enrichments
		for k, v := range enrichments {
			data[k] = v
		}

		// Create new event with enriched data
		enriched := events.NewBaseEvent(
			event.GetID(),
			event.GetType(),
			event.GetSource(),
			data,
		)

		// Copy metadata
		if metadata := event.GetMetadata(); metadata != nil {
			for k, v := range metadata {
				enriched.WithMetadata(k, v)
			}
		}

		return enriched, nil
	}
}

// FilterTransform removes fields from events
func FilterTransform(fieldsToRemove []string) TransformFunc {
	return func(ctx context.Context, event events.CoreEvent) (events.CoreEvent, error) {
		if err := ctx.Err(); err != nil {
			return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled")
		}

		// Clone event data
		data := make(map[string]interface{})
		for k, v := range event.GetData() {
			keep := true
			for _, field := range fieldsToRemove {
				if k == field {
					keep = false
					break
				}
			}
			if keep {
				data[k] = v
			}
		}

		// Create new event with filtered data
		filtered := events.NewBaseEvent(
			event.GetID(),
			event.GetType(),
			event.GetSource(),
			data,
		)

		// Copy metadata
		if metadata := event.GetMetadata(); metadata != nil {
			for k, v := range metadata {
				filtered.WithMetadata(k, v)
			}
		}

		return filtered, nil
	}
}

// RenameFieldsTransform renames fields in events
func RenameFieldsTransform(fieldMap map[string]string) TransformFunc {
	return func(ctx context.Context, event events.CoreEvent) (events.CoreEvent, error) {
		if err := ctx.Err(); err != nil {
			return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled")
		}

		// Clone and rename fields
		data := make(map[string]interface{})
		for k, v := range event.GetData() {
			if newName, exists := fieldMap[k]; exists {
				data[newName] = v
			} else {
				data[k] = v
			}
		}

		// Create new event with renamed fields
		renamed := events.NewBaseEvent(
			event.GetID(),
			event.GetType(),
			event.GetSource(),
			data,
		)

		// Copy metadata
		if metadata := event.GetMetadata(); metadata != nil {
			for k, v := range metadata {
				renamed.WithMetadata(k, v)
			}
		}

		return renamed, nil
	}
}

// TypeConversionTransform changes the event type
func TypeConversionTransform(newType string) TransformFunc {
	return func(ctx context.Context, event events.CoreEvent) (events.CoreEvent, error) {
		if err := ctx.Err(); err != nil {
			return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled")
		}

		// Create new event with different type
		converted := events.NewBaseEvent(
			event.GetID(),
			newType,
			event.GetSource(),
			event.GetData(),
		)

		// Copy metadata and add original type
		if metadata := event.GetMetadata(); metadata != nil {
			for k, v := range metadata {
				converted.WithMetadata(k, v)
			}
		}
		converted.WithMetadata("original_type", event.GetType())

		return converted, nil
	}
}

// ConditionalTransform applies a transform based on a condition
func ConditionalTransform(condition func(events.CoreEvent) bool, transform TransformFunc) TransformFunc {
	return func(ctx context.Context, event events.CoreEvent) (events.CoreEvent, error) {
		if err := ctx.Err(); err != nil {
			return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled")
		}

		if condition(event) {
			return transform(ctx, event)
		}
		return event, nil
	}
}

// JSONPathTransform extracts and restructures data using JSON paths
func JSONPathTransform(mapping map[string]string) TransformFunc {
	return func(ctx context.Context, event events.CoreEvent) (events.CoreEvent, error) {
		if err := ctx.Err(); err != nil {
			return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled")
		}

		// Marshal event data to JSON for path extraction
		jsonData, err := json.Marshal(event.GetData())
		if err != nil {
			return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to marshal event data")
		}

		var dataMap map[string]interface{}
		if err := json.Unmarshal(jsonData, &dataMap); err != nil {
			return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to unmarshal event data")
		}

		// Extract fields using paths
		newData := make(map[string]interface{})
		for newField, path := range mapping {
			value, found := extractJSONPath(dataMap, path)
			if found {
				newData[newField] = value
			}
		}

		// Create new event with extracted data
		transformed := events.NewBaseEvent(
			event.GetID(),
			event.GetType(),
			event.GetSource(),
			newData,
		)

		// Copy metadata
		if metadata := event.GetMetadata(); metadata != nil {
			for k, v := range metadata {
				transformed.WithMetadata(k, v)
			}
		}

		return transformed, nil
	}
}

// AggregateTransform combines multiple events into one
type AggregateTransform struct {
	window    time.Duration
	keyFunc   func(events.CoreEvent) string
	aggregate func([]events.CoreEvent) (events.CoreEvent, error)
	buffer    map[string][]events.CoreEvent
	mu        sync.RWMutex
}

// NewAggregateTransform creates a new aggregate transformer
func NewAggregateTransform(window time.Duration, keyFunc func(events.CoreEvent) string, aggregate func([]events.CoreEvent) (events.CoreEvent, error)) *AggregateTransform {
	return &AggregateTransform{
		window:    window,
		keyFunc:   keyFunc,
		aggregate: aggregate,
		buffer:    make(map[string][]events.CoreEvent),
	}
}

// Transform aggregates events
func (at *AggregateTransform) Transform(ctx context.Context, event events.CoreEvent) (events.CoreEvent, error) {
	if err := ctx.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled")
	}

	key := at.keyFunc(event)

	at.mu.Lock()
	defer at.mu.Unlock()

	// Add event to buffer
	at.buffer[key] = append(at.buffer[key], event)

	// Check if we should aggregate
	if len(at.buffer[key]) == 1 {
		// First event, schedule aggregation
		go func() {
			time.Sleep(at.window)
			at.flush(key)
		}()
		return nil, nil // Don't emit yet
	}

	return nil, nil // Added to buffer, don't emit
}

// flush aggregates buffered events
func (at *AggregateTransform) flush(key string) {
	at.mu.Lock()
	defer at.mu.Unlock()

	events := at.buffer[key]
	if len(events) == 0 {
		return
	}

	delete(at.buffer, key)

	// Aggregate events
	if aggregated, err := at.aggregate(events); err == nil {
		// TODO: Emit aggregated event
		_ = aggregated
	}
}

// Helper functions

func extractJSONPath(data map[string]interface{}, path string) (interface{}, bool) {
	parts := strings.Split(path, ".")
	current := data

	for i, part := range parts {
		// Handle array index notation
		if strings.Contains(part, "[") && strings.Contains(part, "]") {
			// TODO: Implement array index extraction
			return nil, false
		}

		value, exists := current[part]
		if !exists {
			return nil, false
		}

		if i == len(parts)-1 {
			return value, true
		}

		next, ok := value.(map[string]interface{})
		if !ok {
			return nil, false
		}
		current = next
	}

	return nil, false
}

// ChainTransform chains multiple transforms together
func ChainTransform(transforms ...TransformFunc) TransformFunc {
	return func(ctx context.Context, event events.CoreEvent) (events.CoreEvent, error) {
		result := event
		for _, transform := range transforms {
			transformed, err := transform(ctx, result)
			if err != nil {
				return nil, err
			}
			result = transformed
		}
		return result, nil
	}
}

// DebugTransform logs event details for debugging
func DebugTransform(prefix string) TransformFunc {
	return func(ctx context.Context, event events.CoreEvent) (events.CoreEvent, error) {
		if err := ctx.Err(); err != nil {
			return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled")
		}

		fmt.Printf("[%s] Event: ID=%s, Type=%s, Source=%s, Data=%v\n",
			prefix, event.GetID(), event.GetType(), event.GetSource(), event.GetData())

		return event, nil
	}
}
