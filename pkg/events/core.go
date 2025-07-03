// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

// Package events provides a unified event system for Guild components
package events

import (
	"time"
)

// CoreEvent defines the minimal event interface that all Guild events must implement
type CoreEvent interface {
	GetID() string
	GetType() string
	GetSource() string
	GetTimestamp() time.Time
	GetData() map[string]interface{}
	
	// Optional fields that may be empty
	GetTarget() string
	GetMetadata() map[string]interface{}
}

// BaseEvent provides the standard implementation of CoreEvent
type BaseEvent struct {
	ID        string                 `json:"id"`
	Type      string                 `json:"type"`
	Source    string                 `json:"source"`
	Target    string                 `json:"target,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
	Data      map[string]interface{} `json:"data"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// GetID returns the event identifier
func (e *BaseEvent) GetID() string {
	return e.ID
}

// GetType returns the event type
func (e *BaseEvent) GetType() string {
	return e.Type
}

// GetSource returns the event source component
func (e *BaseEvent) GetSource() string {
	return e.Source
}

// GetTarget returns the event target component (may be empty)
func (e *BaseEvent) GetTarget() string {
	return e.Target
}

// GetTimestamp returns when the event occurred
func (e *BaseEvent) GetTimestamp() time.Time {
	return e.Timestamp
}

// GetData returns the event payload data
func (e *BaseEvent) GetData() map[string]interface{} {
	if e.Data == nil {
		return make(map[string]interface{})
	}
	return e.Data
}

// GetMetadata returns additional event metadata (may be empty)
func (e *BaseEvent) GetMetadata() map[string]interface{} {
	if e.Metadata == nil {
		return make(map[string]interface{})
	}
	return e.Metadata
}

// NewBaseEvent creates a new BaseEvent with the required fields
func NewBaseEvent(id, eventType, source string, data map[string]interface{}) *BaseEvent {
	if data == nil {
		data = make(map[string]interface{})
	}
	
	return &BaseEvent{
		ID:        id,
		Type:      eventType,
		Source:    source,
		Timestamp: time.Now(),
		Data:      data,
		Metadata:  make(map[string]interface{}),
	}
}

// WithTarget sets the target component for the event
func (e *BaseEvent) WithTarget(target string) *BaseEvent {
	e.Target = target
	return e
}

// WithMetadata adds metadata to the event
func (e *BaseEvent) WithMetadata(key string, value interface{}) *BaseEvent {
	if e.Metadata == nil {
		e.Metadata = make(map[string]interface{})
	}
	e.Metadata[key] = value
	return e
}

// WithData adds data to the event payload
func (e *BaseEvent) WithData(key string, value interface{}) *BaseEvent {
	if e.Data == nil {
		e.Data = make(map[string]interface{})
	}
	e.Data[key] = value
	return e
}

// Clone creates a deep copy of the event
func (e *BaseEvent) Clone() *BaseEvent {
	clone := &BaseEvent{
		ID:        e.ID,
		Type:      e.Type,
		Source:    e.Source,
		Target:    e.Target,
		Timestamp: e.Timestamp,
	}
	
	// Deep copy data
	if e.Data != nil {
		clone.Data = make(map[string]interface{})
		for k, v := range e.Data {
			clone.Data[k] = v
		}
	}
	
	// Deep copy metadata
	if e.Metadata != nil {
		clone.Metadata = make(map[string]interface{})
		for k, v := range e.Metadata {
			clone.Metadata[k] = v
		}
	}
	
	return clone
}

// Validate checks if the event has all required fields
func (e *BaseEvent) Validate() error {
	if e.ID == "" {
		return ErrMissingEventID
	}
	if e.Type == "" {
		return ErrMissingEventType
	}
	if e.Source == "" {
		return ErrMissingEventSource
	}
	if e.Timestamp.IsZero() {
		return ErrMissingEventTimestamp
	}
	return nil
}