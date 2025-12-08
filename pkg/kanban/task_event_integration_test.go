// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package kanban

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"

	"github.com/guild-framework/guild-core/pkg/comms"
	pb "github.com/guild-framework/guild-core/pkg/grpc/pb/guild/v1"
	"github.com/guild-framework/guild-core/pkg/orchestrator/interfaces"
)

// TestTaskEventFlow tests the complete task event flow end-to-end
func TestTaskEventFlow(t *testing.T) {
	ctx := context.Background()

	// Create mock event collectors
	grpcEvents := &MockGRPCEventService{}
	orchestratorEvents := &MockOrchestrator{}
	kanbanEventManager := NewEventManager(ctx, &MockPubSub{}, "test.")

	// Create task event publisher
	publisher := NewTaskEventPublisher(kanbanEventManager, grpcEvents, orchestratorEvents)

	// Create a test task
	task := NewTask("Test Task", "Test task for event flow")
	boardID := "test-board-123"
	createdBy := "test-user"

	// Test 1: Task Creation Event
	t.Run("TaskCreated", func(t *testing.T) {
		err := publisher.PublishTaskCreated(ctx, task, boardID, createdBy)
		require.NoError(t, err)

		// Verify gRPC event was published
		assert.Len(t, grpcEvents.PublishedEvents, 1)
		grpcEvent := grpcEvents.PublishedEvents[0]
		assert.Equal(t, "task.created", grpcEvent.Type)
		assert.Equal(t, "kanban-service", grpcEvent.Source)

		// Verify orchestrator event was published
		assert.Len(t, orchestratorEvents.PublishedEvents, 1)
		orchEvent := orchestratorEvents.PublishedEvents[0]
		assert.Equal(t, interfaces.EventTypeTaskCreated, orchEvent.Type)
		assert.Equal(t, task.ID, orchEvent.Data["task_id"])
		assert.Equal(t, boardID, orchEvent.Data["board_id"])

		// Clear events for next test
		grpcEvents.PublishedEvents = nil
		orchestratorEvents.PublishedEvents = nil
	})

	// Test 2: Task Move Event
	t.Run("TaskMoved", func(t *testing.T) {
		fromStatus := "backlog"
		toStatus := "in_progress"
		movedBy := "test-user"
		reason := "Starting work"

		err := publisher.PublishTaskMoved(ctx, task, boardID, fromStatus, toStatus, movedBy, reason)
		require.NoError(t, err)

		// Verify gRPC event was published
		assert.Len(t, grpcEvents.PublishedEvents, 1)
		grpcEvent := grpcEvents.PublishedEvents[0]
		assert.Equal(t, "task.moved", grpcEvent.Type)

		// Verify orchestrator event was published
		assert.Len(t, orchestratorEvents.PublishedEvents, 1)
		orchEvent := orchestratorEvents.PublishedEvents[0]
		assert.Equal(t, interfaces.EventTypeTaskStarted, orchEvent.Type)
		assert.Equal(t, fromStatus, orchEvent.Data["from_status"])
		assert.Equal(t, toStatus, orchEvent.Data["to_status"])

		// Clear events for next test
		grpcEvents.PublishedEvents = nil
		orchestratorEvents.PublishedEvents = nil
	})

	// Test 3: Task Update Event
	t.Run("TaskUpdated", func(t *testing.T) {
		updatedBy := "test-user"
		changes := map[string]string{
			"description": "old desc -> new desc",
			"priority":    "medium -> high",
		}

		err := publisher.PublishTaskUpdated(ctx, task, boardID, updatedBy, changes)
		require.NoError(t, err)

		// Verify gRPC event was published
		assert.Len(t, grpcEvents.PublishedEvents, 1)
		grpcEvent := grpcEvents.PublishedEvents[0]
		assert.Equal(t, "task.updated", grpcEvent.Type)

		// Clear events for next test
		grpcEvents.PublishedEvents = nil
		orchestratorEvents.PublishedEvents = nil
	})

	// Test 4: Task Completion Event
	t.Run("TaskCompleted", func(t *testing.T) {
		completedBy := "test-user"
		notes := "Task completed successfully"

		err := publisher.PublishTaskCompleted(ctx, task, boardID, completedBy, notes)
		require.NoError(t, err)

		// Verify gRPC event was published
		assert.Len(t, grpcEvents.PublishedEvents, 1)
		grpcEvent := grpcEvents.PublishedEvents[0]
		assert.Equal(t, "task.completed", grpcEvent.Type)

		// Verify orchestrator event was published
		assert.Len(t, orchestratorEvents.PublishedEvents, 1)
		orchEvent := orchestratorEvents.PublishedEvents[0]
		assert.Equal(t, interfaces.EventTypeTaskCompleted, orchEvent.Type)
		assert.Equal(t, completedBy, orchEvent.Data["completed_by"])

		// Clear events for next test
		grpcEvents.PublishedEvents = nil
		orchestratorEvents.PublishedEvents = nil
	})

	// Test 5: Task Blocking Event
	t.Run("TaskBlocked", func(t *testing.T) {
		blockedBy := "test-user"
		reason := "Waiting for dependencies"
		blockerIDs := []string{"blocker-1", "blocker-2"}

		err := publisher.PublishTaskBlocked(ctx, task, boardID, blockedBy, reason, blockerIDs)
		require.NoError(t, err)

		// Verify gRPC event was published
		assert.Len(t, grpcEvents.PublishedEvents, 1)
		grpcEvent := grpcEvents.PublishedEvents[0]
		assert.Equal(t, "task.blocked", grpcEvent.Type)

		// Clear events for next test
		grpcEvents.PublishedEvents = nil
		orchestratorEvents.PublishedEvents = nil
	})

	// Test 6: Task Unblocking Event
	t.Run("TaskUnblocked", func(t *testing.T) {
		unblockedBy := "test-user"
		reason := "Dependencies resolved"
		resolvedBlockerID := "blocker-1"

		err := publisher.PublishTaskUnblocked(ctx, task, boardID, unblockedBy, reason, resolvedBlockerID)
		require.NoError(t, err)

		// Verify gRPC event was published
		assert.Len(t, grpcEvents.PublishedEvents, 1)
		grpcEvent := grpcEvents.PublishedEvents[0]
		assert.Equal(t, "task.unblocked", grpcEvent.Type)

		// Clear events for next test
		grpcEvents.PublishedEvents = nil
		orchestratorEvents.PublishedEvents = nil
	})

	// Test 7: Task Deletion Event
	t.Run("TaskDeleted", func(t *testing.T) {
		deletedBy := "test-user"
		reason := "Task no longer needed"

		err := publisher.PublishTaskDeleted(ctx, task.ID, boardID, deletedBy, reason)
		require.NoError(t, err)

		// Verify gRPC event was published
		assert.Len(t, grpcEvents.PublishedEvents, 1)
		grpcEvent := grpcEvents.PublishedEvents[0]
		assert.Equal(t, "task.deleted", grpcEvent.Type)

		// Clear events for next test
		grpcEvents.PublishedEvents = nil
		orchestratorEvents.PublishedEvents = nil
	})
}

// TestTaskEventLatency tests that events are published with low latency
func TestTaskEventLatency(t *testing.T) {
	ctx := context.Background()

	// Create mock event collectors
	grpcEvents := &MockGRPCEventService{}
	orchestratorEvents := &MockOrchestrator{}
	kanbanEventManager := NewEventManager(ctx, &MockPubSub{}, "test.")

	// Create task event publisher
	publisher := NewTaskEventPublisher(kanbanEventManager, grpcEvents, orchestratorEvents)

	// Create a test task
	task := NewTask("Latency Test Task", "Test task for latency measurement")
	boardID := "test-board-latency"

	// Measure latency of task creation event
	start := time.Now()
	err := publisher.PublishTaskCreated(ctx, task, boardID, "test-user")
	duration := time.Since(start)

	require.NoError(t, err)
	assert.Less(t, duration.Milliseconds(), int64(10), "Event publishing should complete in <10ms")

	// Verify events were published
	assert.Len(t, grpcEvents.PublishedEvents, 1)
	assert.Len(t, orchestratorEvents.PublishedEvents, 1)
}

// TestConcurrentEventPublishing tests event publishing under concurrent load
func TestConcurrentEventPublishing(t *testing.T) {
	ctx := context.Background()

	// Create mock event collectors
	grpcEvents := &MockGRPCEventService{}
	orchestratorEvents := &MockOrchestrator{}
	kanbanEventManager := NewEventManager(ctx, &MockPubSub{}, "test.")

	// Create task event publisher
	publisher := NewTaskEventPublisher(kanbanEventManager, grpcEvents, orchestratorEvents)

	const numConcurrentEvents = 100
	const numGoroutines = 10

	var wg sync.WaitGroup
	errChan := make(chan error, numConcurrentEvents)

	// Launch concurrent goroutines publishing events
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()

			for j := 0; j < numConcurrentEvents/numGoroutines; j++ {
				task := NewTask("Concurrent Task", "Test concurrent event publishing")
				task.ID = task.ID + "-" + string(rune(goroutineID)) + "-" + string(rune(j))

				if err := publisher.PublishTaskCreated(ctx, task, "test-board", "test-user"); err != nil {
					errChan <- err
				}
			}
		}(i)
	}

	wg.Wait()
	close(errChan)

	// Check for any errors
	var errors []error
	for err := range errChan {
		errors = append(errors, err)
	}

	assert.Empty(t, errors, "No errors should occur during concurrent event publishing")

	// Verify we got the expected number of events
	assert.Len(t, grpcEvents.PublishedEvents, numConcurrentEvents)
	assert.Len(t, orchestratorEvents.PublishedEvents, numConcurrentEvents)
}

// Mock implementations for testing

type MockGRPCEventService struct {
	PublishedEvents []*pb.Event
	mu              sync.Mutex
}

func (m *MockGRPCEventService) PublishEvent(ctx context.Context, req *pb.PublishEventRequest, opts ...grpc.CallOption) (*pb.PublishEventResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.PublishedEvents = append(m.PublishedEvents, req.Event)

	return &pb.PublishEventResponse{
		Success: true,
		Message: "Event published successfully",
	}, nil
}

func (m *MockGRPCEventService) StreamEvents(ctx context.Context, req *pb.StreamEventsRequest, opts ...grpc.CallOption) (grpc.ServerStreamingClient[pb.Event], error) {
	// Not implemented for this test
	return nil, nil
}

type MockOrchestrator struct {
	PublishedEvents []interfaces.Event
	mu              sync.Mutex
}

func (m *MockOrchestrator) Publish(event interfaces.Event) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.PublishedEvents = append(m.PublishedEvents, event)
}

func (m *MockOrchestrator) Subscribe(eventType interfaces.EventType, handler interfaces.EventHandler) {
	// Not implemented for this test
}

func (m *MockOrchestrator) SubscribeAll(handler interfaces.EventHandler) {
	// Not implemented for this test
}

type MockPubSub struct{}

func (m *MockPubSub) Publish(ctx context.Context, topic string, data []byte) error {
	return nil
}

func (m *MockPubSub) PublishMessage(ctx context.Context, msg *comms.Message) error {
	return nil
}

func (m *MockPubSub) Subscribe(ctx context.Context, topicPattern string) error {
	return nil
}

func (m *MockPubSub) Unsubscribe(ctx context.Context, topicPattern string) error {
	return nil
}

func (m *MockPubSub) Receive(ctx context.Context) (*comms.Message, error) {
	// Block to simulate receiving - for testing we won't actually receive
	<-ctx.Done()
	return nil, ctx.Err()
}

func (m *MockPubSub) Close() error {
	return nil
}

// TestTaskEventMetadata tests that event metadata is properly preserved
func TestTaskEventMetadata(t *testing.T) {
	ctx := context.Background()

	// Create mock event collectors
	grpcEvents := &MockGRPCEventService{}
	orchestratorEvents := &MockOrchestrator{}
	kanbanEventManager := NewEventManager(ctx, &MockPubSub{}, "test.")

	// Create task event publisher
	publisher := NewTaskEventPublisher(kanbanEventManager, grpcEvents, orchestratorEvents)

	// Create a test task with metadata
	task := NewTask("Metadata Test Task", "Test task with metadata")
	task.Metadata["custom_field"] = "custom_value"
	task.Metadata["priority_level"] = "high"
	task.AssignedTo = "test-assignee"

	err := publisher.PublishTaskCreated(ctx, task, "test-board", "test-user")
	require.NoError(t, err)

	// Verify gRPC event contains metadata
	assert.Len(t, grpcEvents.PublishedEvents, 1)
	grpcEvent := grpcEvents.PublishedEvents[0]

	// Check that event data contains task information
	eventData := grpcEvent.Data.AsMap()
	assert.Equal(t, task.ID, eventData["task_id"])
	assert.Equal(t, "test-board", eventData["board_id"])
	assert.Equal(t, task.Title, eventData["title"])
	assert.Equal(t, task.AssignedTo, eventData["assignee"])

	// Verify orchestrator event contains metadata
	assert.Len(t, orchestratorEvents.PublishedEvents, 1)
	orchEvent := orchestratorEvents.PublishedEvents[0]
	assert.Equal(t, task.ID, orchEvent.Data["task_id"])
	assert.Equal(t, "test-board", orchEvent.Data["board_id"])
	assert.Equal(t, task.AssignedTo, orchEvent.Data["assignee"])
}
