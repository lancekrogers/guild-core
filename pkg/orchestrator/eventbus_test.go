package orchestrator

import (
	"sync"
	"testing"
	"time"

	"github.com/guild-ventures/guild-core/pkg/orchestrator/mocks"
)

func TestEventBusSubscribeAndPublish(t *testing.T) {
	// Create event bus
	eventBus := NewEventBus()
	
	// Create mock event handler
	mockHandler := mocks.NewMockEventHandler()
	
	// Subscribe to specific event type
	eventBus.Subscribe("test.event", mockHandler.GetHandlerFunc())
	
	// Create test event
	testEvent := Event{
		Type:   "test.event",
		Source: "test",
		Data:   "test data",
	}
	
	// Publish event
	eventBus.Publish(testEvent)
	
	// Wait for the event to be processed (since it's done in a goroutine)
	time.Sleep(50 * time.Millisecond)
	
	// Check if event was received
	events := mockHandler.GetEvents()
	if len(events) != 1 {
		t.Fatalf("Expected 1 event, got %d", len(events))
	}
	
	receivedEvent := events[0]
	if receivedEvent.Type != testEvent.Type {
		t.Errorf("Expected event type %s, got %s", testEvent.Type, receivedEvent.Type)
	}
	
	if receivedEvent.Source != testEvent.Source {
		t.Errorf("Expected event source %s, got %s", testEvent.Source, receivedEvent.Source)
	}
	
	if receivedEvent.Data != testEvent.Data {
		t.Errorf("Expected event data %v, got %v", testEvent.Data, receivedEvent.Data)
	}
}

func TestEventBusSubscribeAll(t *testing.T) {
	// Create event bus
	eventBus := NewEventBus()
	
	// Create mock event handler
	mockHandler := mocks.NewMockEventHandler()
	
	// Subscribe to all events
	eventBus.SubscribeAll(mockHandler.GetHandlerFunc())
	
	// Create and publish multiple events
	events := []Event{
		{Type: "event.type1", Source: "test", Data: "data1"},
		{Type: "event.type2", Source: "test", Data: "data2"},
		{Type: "event.type3", Source: "test", Data: "data3"},
	}
	
	for _, event := range events {
		eventBus.Publish(event)
	}
	
	// Wait for events to be processed
	time.Sleep(50 * time.Millisecond)
	
	// Check if all events were received
	receivedEvents := mockHandler.GetEvents()
	if len(receivedEvents) != len(events) {
		t.Fatalf("Expected %d events, got %d", len(events), len(receivedEvents))
	}
}

func TestEventBusUnsubscribe(t *testing.T) {
	// Create event bus
	eventBus := NewEventBus()
	
	// Create mock event handlers
	mockHandler1 := mocks.NewMockEventHandler()
	mockHandler2 := mocks.NewMockEventHandler()
	
	// Subscribe both handlers to the same event type
	handler1 := mockHandler1.GetHandlerFunc()
	handler2 := mockHandler2.GetHandlerFunc()
	
	eventBus.Subscribe("test.event", handler1)
	eventBus.Subscribe("test.event", handler2)
	
	// Create test event
	testEvent := Event{
		Type:   "test.event",
		Source: "test",
		Data:   "test data",
	}
	
	// Publish event (both handlers should receive it)
	eventBus.Publish(testEvent)
	
	// Wait for events to be processed
	time.Sleep(50 * time.Millisecond)
	
	// Check if both handlers received the event
	if len(mockHandler1.GetEvents()) != 1 || len(mockHandler2.GetEvents()) != 1 {
		t.Fatalf("Expected both handlers to receive 1 event")
	}
	
	// Reset handlers
	mockHandler1.Reset()
	mockHandler2.Reset()
	
	// Unsubscribe handler1
	eventBus.Unsubscribe("test.event", handler1)
	
	// Publish event again (only handler2 should receive it)
	eventBus.Publish(testEvent)
	
	// Wait for events to be processed
	time.Sleep(50 * time.Millisecond)
	
	// Check if only handler2 received the event
	if len(mockHandler1.GetEvents()) != 0 {
		t.Errorf("Expected handler1 to receive 0 events after unsubscribing, got %d", len(mockHandler1.GetEvents()))
	}
	
	if len(mockHandler2.GetEvents()) != 1 {
		t.Errorf("Expected handler2 to receive 1 event, got %d", len(mockHandler2.GetEvents()))
	}
}

func TestEventBusPublishJSON(t *testing.T) {
	// Create event bus
	eventBus := NewEventBus()
	
	// Create mock event handler
	mockHandler := mocks.NewMockEventHandler()
	
	// Subscribe to event type
	eventBus.Subscribe("json.test", mockHandler.GetHandlerFunc())
	
	// JSON event
	jsonEvent := `{"type":"json.test","source":"json","data":"json data"}`
	
	// Publish JSON event
	err := eventBus.PublishJSON(jsonEvent)
	if err != nil {
		t.Fatalf("PublishJSON returned error: %v", err)
	}
	
	// Wait for event to be processed
	time.Sleep(50 * time.Millisecond)
	
	// Check if event was received
	events := mockHandler.GetEvents()
	if len(events) != 1 {
		t.Fatalf("Expected 1 event, got %d", len(events))
	}
	
	receivedEvent := events[0]
	if receivedEvent.Type != "json.test" {
		t.Errorf("Expected event type 'json.test', got '%s'", receivedEvent.Type)
	}
	
	if receivedEvent.Source != "json" {
		t.Errorf("Expected event source 'json', got '%s'", receivedEvent.Source)
	}
	
	if receivedEvent.Data != "json data" {
		t.Errorf("Expected event data 'json data', got '%v'", receivedEvent.Data)
	}
}

func TestEventBusPublishJSONInvalid(t *testing.T) {
	// Create event bus
	eventBus := NewEventBus()
	
	// Invalid JSON
	invalidJSON := `{"type":"json.test","source":"json",data:"invalid"}`
	
	// Publish invalid JSON event
	err := eventBus.PublishJSON(invalidJSON)
	if err == nil {
		t.Fatalf("Expected error for invalid JSON, got nil")
	}
}

func TestEventBusConcurrentPublish(t *testing.T) {
	// Create event bus
	eventBus := NewEventBus()
	
	// Create mock event handler
	mockHandler := mocks.NewMockEventHandler()
	
	// Subscribe to all events
	eventBus.SubscribeAll(mockHandler.GetHandlerFunc())
	
	// Publish events concurrently
	const numEvents = 100
	var wg sync.WaitGroup
	wg.Add(numEvents)
	
	for i := 0; i < numEvents; i++ {
		go func(i int) {
			defer wg.Done()
			eventBus.Publish(Event{
				Type:   "concurrent.event",
				Source: "test",
				Data:   i,
			})
		}(i)
	}
	
	wg.Wait()
	
	// Wait for events to be processed
	time.Sleep(100 * time.Millisecond)
	
	// Check if all events were received
	receivedEvents := mockHandler.GetEvents()
	if len(receivedEvents) != numEvents {
		t.Fatalf("Expected %d events, got %d", numEvents, len(receivedEvents))
	}
}

func TestFilterHandler(t *testing.T) {
	// Create event bus
	eventBus := NewEventBus()
	
	// Create mock event handler
	mockHandler := mocks.NewMockEventHandler()
	
	// Create filter that only accepts events with even-numbered data
	filter := func(event Event) bool {
		if val, ok := event.Data.(int); ok {
			return val%2 == 0
		}
		return false
	}
	
	// Subscribe with filter
	eventBus.Subscribe("filter.test", FilterHandler(filter, mockHandler.GetHandlerFunc()))
	
	// Publish events with odd and even data
	for i := 0; i < 10; i++ {
		eventBus.Publish(Event{
			Type:   "filter.test",
			Source: "test",
			Data:   i,
		})
	}
	
	// Wait for events to be processed
	time.Sleep(50 * time.Millisecond)
	
	// Check if only events with even data were received
	receivedEvents := mockHandler.GetEvents()
	if len(receivedEvents) != 5 {
		t.Fatalf("Expected 5 events (even numbers), got %d", len(receivedEvents))
	}
	
	// Verify that all received events have even-numbered data
	for _, event := range receivedEvents {
		if val, ok := event.Data.(int); ok {
			if val%2 != 0 {
				t.Errorf("Received event with odd-numbered data: %v", val)
			}
		} else {
			t.Errorf("Received event with non-integer data: %v", event.Data)
		}
	}
}