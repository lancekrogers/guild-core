// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package mocks

import (
	"sync"

	"github.com/lancekrogers/guild/pkg/orchestrator/interfaces"
)

// MockEventHandler is a mock implementation of the EventHandler function
type MockEventHandler struct {
	ReceivedEvents []interfaces.Event
	mu             sync.Mutex
	Handler        func(event interfaces.Event)
}

// NewMockEventHandler creates a new mock event handler
func NewMockEventHandler() *MockEventHandler {
	return &MockEventHandler{
		ReceivedEvents: make([]interfaces.Event, 0),
	}
}

// HandleEvent implements the EventHandler function
func (m *MockEventHandler) HandleEvent(event interfaces.Event) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.ReceivedEvents = append(m.ReceivedEvents, event)

	if m.Handler != nil {
		m.Handler(event)
	}
}

// GetHandlerFunc returns the event handler function
func (m *MockEventHandler) GetHandlerFunc() interfaces.EventHandler {
	return m.HandleEvent
}

// GetEvents returns all received events
func (m *MockEventHandler) GetEvents() []interfaces.Event {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Create a copy to avoid race conditions
	events := make([]interfaces.Event, len(m.ReceivedEvents))
	copy(events, m.ReceivedEvents)

	return events
}

// Reset clears all received events
func (m *MockEventHandler) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.ReceivedEvents = make([]interfaces.Event, 0)
}

// FilterEventsByType returns events of a specific type
func (m *MockEventHandler) FilterEventsByType(eventType string) []interfaces.Event {
	m.mu.Lock()
	defer m.mu.Unlock()

	var filtered []interfaces.Event
	for _, e := range m.ReceivedEvents {
		if string(e.Type) == eventType {
			filtered = append(filtered, e)
		}
	}

	return filtered
}
