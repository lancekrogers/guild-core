package mocks

import (
	"sync"

	"github.com/blockhead-consulting/Guild/pkg/orchestrator"
)

// MockEventHandler is a mock implementation of the EventHandler function
type MockEventHandler struct {
	ReceivedEvents []orchestrator.Event
	mu             sync.Mutex
	Handler        func(event orchestrator.Event)
}

// NewMockEventHandler creates a new mock event handler
func NewMockEventHandler() *MockEventHandler {
	return &MockEventHandler{
		ReceivedEvents: make([]orchestrator.Event, 0),
	}
}

// HandleEvent implements the EventHandler function
func (m *MockEventHandler) HandleEvent(event orchestrator.Event) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.ReceivedEvents = append(m.ReceivedEvents, event)
	
	if m.Handler != nil {
		m.Handler(event)
	}
}

// GetHandlerFunc returns the event handler function
func (m *MockEventHandler) GetHandlerFunc() orchestrator.EventHandler {
	return m.HandleEvent
}

// GetEvents returns all received events
func (m *MockEventHandler) GetEvents() []orchestrator.Event {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	// Create a copy to avoid race conditions
	events := make([]orchestrator.Event, len(m.ReceivedEvents))
	copy(events, m.ReceivedEvents)
	
	return events
}

// Reset clears all received events
func (m *MockEventHandler) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.ReceivedEvents = make([]orchestrator.Event, 0)
}

// FilterEventsByType returns events of a specific type
func (m *MockEventHandler) FilterEventsByType(eventType string) []orchestrator.Event {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	var filtered []orchestrator.Event
	for _, e := range m.ReceivedEvents {
		if e.Type == eventType {
			filtered = append(filtered, e)
		}
	}
	
	return filtered
}