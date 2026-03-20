// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package eventbus

import (
	"container/list"
	"sync"
	"time"
)

// DeadLetterEvent represents an event that failed processing
type DeadLetterEvent struct {
	Event     Event
	Error     string
	Timestamp time.Time
	Retries   int
}

// DeadLetterQueue stores events that failed processing
type DeadLetterQueue struct {
	mu       sync.RWMutex
	events   *list.List
	eventMap map[string]*list.Element
	maxSize  int
}

// NewDeadLetterQueue creates a new dead letter queue
func NewDeadLetterQueue(maxSize int) *DeadLetterQueue {
	return &DeadLetterQueue{
		events:   list.New(),
		eventMap: make(map[string]*list.Element),
		maxSize:  maxSize,
	}
}

// Add adds an event to the dead letter queue
func (dlq *DeadLetterQueue) Add(event *DeadLetterEvent) {
	dlq.mu.Lock()
	defer dlq.mu.Unlock()

	// Check if event already exists
	if elem, exists := dlq.eventMap[event.Event.GetID()]; exists {
		// Update existing event
		elem.Value = event
		return
	}

	// Add new event
	elem := dlq.events.PushBack(event)
	dlq.eventMap[event.Event.GetID()] = elem

	// Enforce max size (remove oldest)
	if dlq.events.Len() > dlq.maxSize {
		oldest := dlq.events.Front()
		if oldest != nil {
			oldEvent := oldest.Value.(*DeadLetterEvent)
			delete(dlq.eventMap, oldEvent.Event.GetID())
			dlq.events.Remove(oldest)
		}
	}
}

// Remove removes an event from the dead letter queue
func (dlq *DeadLetterQueue) Remove(eventID string) bool {
	dlq.mu.Lock()
	defer dlq.mu.Unlock()

	elem, exists := dlq.eventMap[eventID]
	if !exists {
		return false
	}

	delete(dlq.eventMap, eventID)
	dlq.events.Remove(elem)
	return true
}

// Get retrieves an event from the dead letter queue
func (dlq *DeadLetterQueue) Get(eventID string) *DeadLetterEvent {
	dlq.mu.RLock()
	defer dlq.mu.RUnlock()

	elem, exists := dlq.eventMap[eventID]
	if !exists {
		return nil
	}

	return elem.Value.(*DeadLetterEvent)
}

// GetAll returns all events in the dead letter queue
func (dlq *DeadLetterQueue) GetAll() []*DeadLetterEvent {
	dlq.mu.RLock()
	defer dlq.mu.RUnlock()

	events := make([]*DeadLetterEvent, 0, dlq.events.Len())
	for elem := dlq.events.Front(); elem != nil; elem = elem.Next() {
		events = append(events, elem.Value.(*DeadLetterEvent))
	}

	return events
}

// GetOldest returns the oldest N events from the queue
func (dlq *DeadLetterQueue) GetOldest(n int) []*DeadLetterEvent {
	dlq.mu.RLock()
	defer dlq.mu.RUnlock()

	events := make([]*DeadLetterEvent, 0, n)
	count := 0

	for elem := dlq.events.Front(); elem != nil && count < n; elem = elem.Next() {
		events = append(events, elem.Value.(*DeadLetterEvent))
		count++
	}

	return events
}

// GetByAge returns events older than the specified duration
func (dlq *DeadLetterQueue) GetByAge(age time.Duration) []*DeadLetterEvent {
	dlq.mu.RLock()
	defer dlq.mu.RUnlock()

	cutoff := time.Now().Add(-age)
	events := make([]*DeadLetterEvent, 0)

	for elem := dlq.events.Front(); elem != nil; elem = elem.Next() {
		event := elem.Value.(*DeadLetterEvent)
		if event.Timestamp.Before(cutoff) {
			events = append(events, event)
		}
	}

	return events
}

// Size returns the current size of the dead letter queue
func (dlq *DeadLetterQueue) Size() int {
	dlq.mu.RLock()
	defer dlq.mu.RUnlock()
	return dlq.events.Len()
}

// Clear removes all events from the dead letter queue
func (dlq *DeadLetterQueue) Clear() {
	dlq.mu.Lock()
	defer dlq.mu.Unlock()

	dlq.events.Init()
	dlq.eventMap = make(map[string]*list.Element)
}

// Stats returns statistics about the dead letter queue
func (dlq *DeadLetterQueue) Stats() map[string]interface{} {
	dlq.mu.RLock()
	defer dlq.mu.RUnlock()

	stats := make(map[string]interface{})
	stats["size"] = dlq.events.Len()
	stats["max_size"] = dlq.maxSize

	// Calculate age distribution
	now := time.Now()
	var oldestAge, newestAge time.Duration

	if dlq.events.Len() > 0 {
		oldest := dlq.events.Front().Value.(*DeadLetterEvent)
		newest := dlq.events.Back().Value.(*DeadLetterEvent)
		oldestAge = now.Sub(oldest.Timestamp)
		newestAge = now.Sub(newest.Timestamp)
	}

	stats["oldest_age"] = oldestAge
	stats["newest_age"] = newestAge

	// Count by retry attempts
	retryCounts := make(map[int]int)
	for elem := dlq.events.Front(); elem != nil; elem = elem.Next() {
		event := elem.Value.(*DeadLetterEvent)
		retryCounts[event.Retries]++
	}
	stats["retry_distribution"] = retryCounts

	return stats
}
