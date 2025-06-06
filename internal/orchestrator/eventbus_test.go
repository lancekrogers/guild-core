package orchestrator

import (
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/guild-ventures/guild-core/internal/orchestrator/interfaces"
)

func TestEventBusSubscribe(t *testing.T) {
	eventBus := NewEventBus()
	
	// Track received events
	var receivedEvent Event
	eventReceived := make(chan bool, 1)
	
	// Subscribe to a specific event type
	eventBus.Subscribe("test.event", func(event Event) {
		receivedEvent = event
		eventReceived <- true
	})
	
	// Create test event
	testEvent := Event{
		Type:   EventType("test.event"),
		Source: "test",
		Data:   map[string]interface{}{"message": "test data"},
	}
	
	// Publish event
	eventBus.Publish(testEvent)
	
	// Wait for event to be received
	select {
	case <-eventReceived:
		// Event received
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Event not received within timeout")
	}
	
	// Verify event details
	if receivedEvent.Type != testEvent.Type {
		t.Errorf("Expected event type %s, got %s", testEvent.Type, receivedEvent.Type)
	}
	
	if receivedEvent.Source != testEvent.Source {
		t.Errorf("Expected event source %s, got %s", testEvent.Source, receivedEvent.Source)
	}
	
	// Compare data maps
	if msg, ok := receivedEvent.Data["message"]; !ok || msg != "test data" {
		t.Errorf("Expected event data message 'test data', got %v", receivedEvent.Data)
	}
}

func TestEventBusSubscribeAll(t *testing.T) {
	eventBus := NewEventBus()
	
	// Track all received events
	var receivedEvents []Event
	var mu sync.Mutex
	eventCount := make(chan int, 10)
	
	// Subscribe to all events
	eventBus.SubscribeAll(func(event Event) {
		mu.Lock()
		receivedEvents = append(receivedEvents, event)
		mu.Unlock()
		eventCount <- len(receivedEvents)
	})
	
	// Publish multiple events of different types
	events := []Event{
		{Type: EventType("event1"), Source: "test", Data: map[string]interface{}{"id": "1"}},
		{Type: EventType("event2"), Source: "test", Data: map[string]interface{}{"id": "2"}},
		{Type: EventType("event3"), Source: "test", Data: map[string]interface{}{"id": "3"}},
	}
	
	for _, event := range events {
		eventBus.Publish(event)
	}
	
	// Wait for all events to be received
	for i := 0; i < 3; i++ {
		select {
		case count := <-eventCount:
			if count > 3 {
				t.Fatal("Received more events than expected")
			}
		case <-time.After(100 * time.Millisecond):
			t.Fatal("Not all events received within timeout")
		}
	}
	
	// Verify all events were received
	mu.Lock()
	defer mu.Unlock()
	
	if len(receivedEvents) != 3 {
		t.Fatalf("Expected 3 events, received %d", len(receivedEvents))
	}
	
	// Verify all event types were received (order may vary due to goroutines)
	eventTypeMap := make(map[EventType]bool)
	for _, event := range receivedEvents {
		eventTypeMap[event.Type] = true
	}
	
	for i := 0; i < 3; i++ {
		expectedType := EventType(string("event") + string(rune('1'+i)))
		if !eventTypeMap[expectedType] {
			t.Errorf("Expected event type %s was not received", expectedType)
		}
	}
}

func TestEventBusUnsubscribe(t *testing.T) {
	eventBus := NewEventBus()
	
	// Track received events
	eventReceived := make(chan bool, 1)
	
	// Subscribe to an event
	handler := func(event Event) {
		eventReceived <- true
	}
	
	eventBus.Subscribe("test.event", handler)
	
	// Publish first event (should be received)
	eventBus.Publish(Event{
		Type:   EventType("test.event"),
		Source: "test",
		Data:   map[string]interface{}{"message": "first"},
	})
	
	// Wait for first event
	select {
	case <-eventReceived:
		// Event received as expected
	case <-time.After(100 * time.Millisecond):
		t.Fatal("First event not received")
	}
	
	// Unsubscribe
	eventBus.Unsubscribe("test.event", handler)
	
	// Publish second event (should not be received)
	eventBus.Publish(Event{
		Type:   EventType("test.event"),
		Source: "test",
		Data:   map[string]interface{}{"message": "second"},
	})
	
	// Wait to ensure event is not received
	select {
	case <-eventReceived:
		t.Fatal("Event received after unsubscribe")
	case <-time.After(50 * time.Millisecond):
		// No event received, as expected
	}
}

func TestEventBusJSON(t *testing.T) {
	
	// Original event with complex data
	originalEvent := Event{
		Type:   EventType("json.test"),
		Source: "test",
		Data: map[string]interface{}{
			"string":  "test",
			"number":  42,
			"boolean": true,
			"array":   []interface{}{"a", "b", "c"},
			"object":  map[string]interface{}{"key": "value"},
		},
	}
	
	// Convert to JSON
	jsonData, err := json.Marshal(originalEvent)
	if err != nil {
		t.Fatalf("Failed to convert event to JSON: %v", err)
	}
	
	// Parse JSON back
	var parsedEvent interfaces.Event
	if err := json.Unmarshal([]byte(jsonData), &parsedEvent); err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}
	
	// Verify basic fields
	if string(parsedEvent.Type) != string(originalEvent.Type) {
		t.Errorf("Expected type %s, got %s", originalEvent.Type, parsedEvent.Type)
	}
	
	if parsedEvent.Source != originalEvent.Source {
		t.Errorf("Expected source %s, got %s", originalEvent.Source, parsedEvent.Source)
	}
	
	// Verify data fields
	if str, ok := parsedEvent.Data["string"].(string); !ok || str != "test" {
		t.Errorf("Expected string 'test', got %v", parsedEvent.Data["string"])
	}
	
	if num, ok := parsedEvent.Data["number"].(float64); !ok || num != 42 {
		t.Errorf("Expected number 42, got %v", parsedEvent.Data["number"])
	}
}

func TestEventBusConcurrency(t *testing.T) {
	eventBus := NewEventBus()
	
	// Track events
	var receivedCount int
	var mu sync.Mutex
	done := make(chan bool)
	
	// Subscribe to events
	eventBus.Subscribe("concurrent.event", func(event Event) {
		mu.Lock()
		receivedCount++
		mu.Unlock()
		
		// Simulate some processing
		time.Sleep(time.Microsecond)
	})
	
	// Publish events concurrently
	numGoroutines := 10
	eventsPerGoroutine := 100
	expectedTotal := numGoroutines * eventsPerGoroutine
	
	var wg sync.WaitGroup
	wg.Add(numGoroutines)
	
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < eventsPerGoroutine; j++ {
				eventBus.Publish(Event{
					Type:   EventType("concurrent.event"),
					Source: "test",
					Data:   map[string]interface{}{"goroutine": id, "event": j},
				})
			}
		}(i)
	}
	
	// Wait for all publishers to finish
	wg.Wait()
	
	// Give handlers time to process
	go func() {
		time.Sleep(500 * time.Millisecond)
		done <- true
	}()
	
	// Wait for processing or timeout
	select {
	case <-done:
		// Check if all events were received
		mu.Lock()
		defer mu.Unlock()
		if receivedCount != expectedTotal {
			t.Errorf("Expected %d events, received %d", expectedTotal, receivedCount)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Timeout waiting for events to be processed")
	}
}

func TestEventBusMultipleHandlers(t *testing.T) {
	eventBus := NewEventBus()
	
	// Track which handlers were called
	handler1Called := false
	handler2Called := false
	handler3Called := false
	
	var wg sync.WaitGroup
	wg.Add(3)
	
	// Subscribe multiple handlers to the same event
	eventBus.Subscribe("multi.event", func(event Event) {
		handler1Called = true
		wg.Done()
	})
	
	eventBus.Subscribe("multi.event", func(event Event) {
		handler2Called = true
		wg.Done()
	})
	
	eventBus.Subscribe("multi.event", func(event Event) {
		handler3Called = true
		wg.Done()
	})
	
	// Publish event
	eventBus.Publish(Event{
		Type:   EventType("multi.event"),
		Source: "test",
		Data:   map[string]interface{}{"test": true},
	})
	
	// Wait for handlers
	done := make(chan bool)
	go func() {
		wg.Wait()
		done <- true
	}()
	
	select {
	case <-done:
		// All handlers completed
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Not all handlers were called within timeout")
	}
	
	// Verify all handlers were called
	if !handler1Called || !handler2Called || !handler3Called {
		t.Errorf("Not all handlers were called: h1=%v, h2=%v, h3=%v", 
			handler1Called, handler2Called, handler3Called)
	}
}