package orchestrator

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/blockhead-consulting/guild/pkg/agent"
	"github.com/blockhead-consulting/guild/pkg/kanban"
	"github.com/blockhead-consulting/guild/pkg/orchestrator/mocks"
)

func setupDispatcherTest() (*TaskDispatcher, *mocks.MockKanbanManager, *mocks.MockAgentFactory, *EventBus, *mocks.MockEventHandler) {
	// Create mock event handler
	mockEventHandler := mocks.NewMockEventHandler()
	
	// Create event bus
	eventBus := NewEventBus()
	eventBus.SubscribeAll(mockEventHandler.GetHandlerFunc())
	
	// Create mock kanban manager
	mockKanbanManager := mocks.NewMockKanbanManager()
	
	// Create mock agent factory
	mockAgentFactory := mocks.NewMockAgentFactory()
	
	// Create task dispatcher
	dispatcher := NewTaskDispatcher(mockKanbanManager, mockAgentFactory, eventBus, 5)
	
	return dispatcher, mockKanbanManager, mockAgentFactory, eventBus, mockEventHandler
}

func createMockTask(id, title, description string, status string) *kanban.Task {
	return &kanban.Task{
		ID:          id,
		Title:       title,
		Description: description,
		Status:      status,
		Assignee:    "",
		Priority:    1,
		Tags:        []string{},
		Comments:    []*kanban.Comment{},
		History:     []*kanban.TaskHistory{},
		Created:     time.Now(),
		Updated:     time.Now(),
	}
}

func TestRegisterAgent(t *testing.T) {
	dispatcher, _, _, _, mockEventHandler := setupDispatcherTest()
	
	// Create mock agent
	mockAgent := mocks.NewMockAgent("agent1", "Test Agent", "worker")
	
	// Register agent
	dispatcher.RegisterAgent(mockAgent)
	
	// Wait for events to be processed
	time.Sleep(50 * time.Millisecond)
	
	// Check if agent is registered in the pool
	agents := dispatcher.GetAvailableAgents()
	if len(agents) != 1 {
		t.Fatalf("Expected 1 available agent, got %d", len(agents))
	}
	
	// Check if event was emitted
	events := mockEventHandler.FilterEventsByType(EventAgentAdded)
	if len(events) != 1 {
		t.Fatalf("Expected 1 agent added event, got %d", len(events))
	}
	
	// Check event data
	event := events[0]
	if event.Source != "dispatcher" {
		t.Errorf("Expected event source 'dispatcher', got '%s'", event.Source)
	}
	
	if agentID, ok := event.Data.(string); !ok || agentID != "agent1" {
		t.Errorf("Expected event data to be agent ID 'agent1', got '%v'", event.Data)
	}
}

func TestUnregisterAgent(t *testing.T) {
	dispatcher, _, _, _, mockEventHandler := setupDispatcherTest()
	
	// Create and register mock agents
	mockAgent1 := mocks.NewMockAgent("agent1", "Test Agent 1", "worker")
	mockAgent2 := mocks.NewMockAgent("agent2", "Test Agent 2", "worker")
	
	dispatcher.RegisterAgent(mockAgent1)
	dispatcher.RegisterAgent(mockAgent2)
	
	// Clear events
	mockEventHandler.Reset()
	
	// Unregister agent
	dispatcher.UnregisterAgent("agent1")
	
	// Wait for events to be processed
	time.Sleep(50 * time.Millisecond)
	
	// Check available agents
	agents := dispatcher.GetAvailableAgents()
	if len(agents) != 1 {
		t.Fatalf("Expected 1 available agent after unregistering, got %d", len(agents))
	}
	
	// Verify the remaining agent is agent2
	if agents[0].ID() != "agent2" {
		t.Errorf("Expected remaining agent to be 'agent2', got '%s'", agents[0].ID())
	}
	
	// Check if event was emitted
	events := mockEventHandler.FilterEventsByType(EventAgentRemoved)
	if len(events) != 1 {
		t.Fatalf("Expected 1 agent removed event, got %d", len(events))
	}
	
	// Check event data
	event := events[0]
	if agentID, ok := event.Data.(string); !ok || agentID != "agent1" {
		t.Errorf("Expected event data to be agent ID 'agent1', got '%v'", event.Data)
	}
}

func TestDispatchTasks(t *testing.T) {
	dispatcher, mockKanbanManager, _, _, mockEventHandler := setupDispatcherTest()
	
	// Create mock agents
	mockAgent1 := mocks.NewMockAgent("agent1", "Test Agent 1", "worker")
	mockAgent2 := mocks.NewMockAgent("agent2", "Test Agent 2", "worker")
	
	// Register agents
	dispatcher.RegisterAgent(mockAgent1)
	dispatcher.RegisterAgent(mockAgent2)
	
	// Create mock tasks
	task1 := createMockTask("task1", "Task 1", "Description 1", kanban.StatusTodo)
	task2 := createMockTask("task2", "Task 2", "Description 2", kanban.StatusTodo)
	
	// Add tasks to kanban manager
	mockKanbanManager.AddTasks(task1, task2)
	
	// Clear events
	mockEventHandler.Reset()
	
	// Dispatch tasks
	ctx := context.Background()
	err := dispatcher.DispatchTasks(ctx)
	if err != nil {
		t.Fatalf("DispatchTasks returned error: %v", err)
	}
	
	// Wait for events to be processed
	time.Sleep(50 * time.Millisecond)
	
	// Check active agents
	activeAgents := dispatcher.GetActiveAgents()
	if len(activeAgents) != 2 {
		t.Fatalf("Expected 2 active agents after dispatching, got %d", len(activeAgents))
	}
	
	// Check if task assignment events were emitted
	events := mockEventHandler.FilterEventsByType(EventTaskAssigned)
	if len(events) != 2 {
		t.Fatalf("Expected 2 task assigned events, got %d", len(events))
	}
	
	// Verify agent tasks
	agent1Tasks := mockAgent1.GetTasks()
	agent2Tasks := mockAgent2.GetTasks()
	
	if len(agent1Tasks) != 1 || len(agent2Tasks) != 1 {
		t.Fatalf("Expected each agent to have 1 task, got %d and %d", len(agent1Tasks), len(agent2Tasks))
	}
	
	// Verify task status was updated
	updatedTask1, _ := mockKanbanManager.GetTask(ctx, "task1")
	updatedTask2, _ := mockKanbanManager.GetTask(ctx, "task2")
	
	if updatedTask1.Status != kanban.StatusInProgress || updatedTask2.Status != kanban.StatusInProgress {
		t.Errorf("Expected tasks to be updated to in-progress status")
	}
}

func TestDispatchTasksWithNoAvailableTasks(t *testing.T) {
	dispatcher, mockKanbanManager, _, _, _ := setupDispatcherTest()
	
	// Create and register mock agent
	mockAgent := mocks.NewMockAgent("agent1", "Test Agent", "worker")
	dispatcher.RegisterAgent(mockAgent)
	
	// Don't add any tasks to the kanban manager
	
	// Dispatch tasks
	ctx := context.Background()
	err := dispatcher.DispatchTasks(ctx)
	if err != nil {
		t.Fatalf("DispatchTasks returned error: %v", err)
	}
	
	// Verify no agents were activated
	activeAgents := dispatcher.GetActiveAgents()
	if len(activeAgents) != 0 {
		t.Fatalf("Expected 0 active agents when no tasks available, got %d", len(activeAgents))
	}
}

func TestDispatchTasksWithKanbanError(t *testing.T) {
	dispatcher, mockKanbanManager, _, _, _ := setupDispatcherTest()
	
	// Set error in kanban manager
	mockKanbanManager.SetListError(errors.New("kanban error"))
	
	// Dispatch tasks
	ctx := context.Background()
	err := dispatcher.DispatchTasks(ctx)
	if err == nil {
		t.Fatalf("Expected error from DispatchTasks, got nil")
	}
}

func TestDispatchTasksWithNoAvailableAgents(t *testing.T) {
	dispatcher, mockKanbanManager, _, _, _ := setupDispatcherTest()
	
	// Create mock task
	task := createMockTask("task1", "Task 1", "Description 1", kanban.StatusTodo)
	
	// Add task to kanban manager
	mockKanbanManager.AddTasks(task)
	
	// Don't register any agents
	
	// Dispatch tasks
	ctx := context.Background()
	err := dispatcher.DispatchTasks(ctx)
	if err != nil {
		t.Fatalf("DispatchTasks returned error: %v", err)
	}
	
	// Verify no agents were activated
	activeAgents := dispatcher.GetActiveAgents()
	if len(activeAgents) != 0 {
		t.Fatalf("Expected 0 active agents when no agents available, got %d", len(activeAgents))
	}
	
	// Verify task status was not updated
	updatedTask, _ := mockKanbanManager.GetTask(ctx, "task1")
	if updatedTask.Status != kanban.StatusTodo {
		t.Errorf("Expected task to remain in todo status, got %s", updatedTask.Status)
	}
}

func TestStartAgent(t *testing.T) {
	dispatcher, _, _, _, mockEventHandler := setupDispatcherTest()
	
	// Create and register mock agent
	mockAgent := mocks.NewMockAgent("agent1", "Test Agent", "worker")
	mockAgent.SetStatus(agent.StatusWorking)
	
	dispatcher.RegisterAgent(mockAgent)
	
	// Activate the agent
	dispatcher.activeAgents["agent1"] = mockAgent
	
	// Clear events
	mockEventHandler.Reset()
	
	// Start agent
	ctx := context.Background()
	err := dispatcher.StartAgent(ctx, "agent1")
	if err != nil {
		t.Fatalf("StartAgent returned error: %v", err)
	}
	
	// Wait for events to be processed
	time.Sleep(50 * time.Millisecond)
	
	// Check if agent started event was emitted
	events := mockEventHandler.FilterEventsByType(EventAgentStarted)
	if len(events) != 1 {
		t.Fatalf("Expected 1 agent started event, got %d", len(events))
	}
	
	// Wait for agent execution to complete
	time.Sleep(100 * time.Millisecond)
	
	// Check if agent completed event was emitted
	completedEvents := mockEventHandler.FilterEventsByType(EventAgentCompleted)
	if len(completedEvents) != 1 {
		t.Fatalf("Expected 1 agent completed event, got %d", len(completedEvents))
	}
	
	// Verify agent was removed from active agents
	activeAgents := dispatcher.GetActiveAgents()
	if len(activeAgents) != 0 {
		t.Fatalf("Expected 0 active agents after completion, got %d", len(activeAgents))
	}
}

func TestStartAgentWithError(t *testing.T) {
	dispatcher, _, _, _, mockEventHandler := setupDispatcherTest()
	
	// Create and register mock agent with execution error
	mockAgent := mocks.NewMockAgent("agent1", "Test Agent", "worker")
	mockAgent.SetStatus(agent.StatusWorking)
	mockAgent.SetExecuteError(errors.New("execution error"))
	
	dispatcher.RegisterAgent(mockAgent)
	
	// Activate the agent
	dispatcher.activeAgents["agent1"] = mockAgent
	
	// Clear events
	mockEventHandler.Reset()
	
	// Start agent
	ctx := context.Background()
	err := dispatcher.StartAgent(ctx, "agent1")
	if err != nil {
		t.Fatalf("StartAgent returned error: %v", err)
	}
	
	// Wait for execution to complete
	time.Sleep(100 * time.Millisecond)
	
	// Check if agent failed event was emitted
	events := mockEventHandler.FilterEventsByType(EventAgentFailed)
	if len(events) != 1 {
		t.Fatalf("Expected 1 agent failed event, got %d", len(events))
	}
	
	// Verify error message in event metadata
	event := events[0]
	metadata, ok := event.Metadata.(map[string]string)
	if !ok {
		t.Fatalf("Expected metadata to be map[string]string, got %T", event.Metadata)
	}
	
	errorMsg, exists := metadata["error"]
	if !exists || errorMsg != "execution error" {
		t.Errorf("Expected error message 'execution error', got '%s'", errorMsg)
	}
}

func TestStartNonExistentAgent(t *testing.T) {
	dispatcher, _, _, _, _ := setupDispatcherTest()
	
	// Try to start non-existent agent
	ctx := context.Background()
	err := dispatcher.StartAgent(ctx, "nonexistent")
	if err == nil {
		t.Fatalf("Expected error when starting non-existent agent, got nil")
	}
}

func TestGetActiveAndAvailableAgents(t *testing.T) {
	dispatcher, _, _, _, _ := setupDispatcherTest()
	
	// Create mock agents
	activeAgent := mocks.NewMockAgent("active1", "Active Agent", "worker")
	activeAgent.SetStatus(agent.StatusWorking)
	
	idleAgent1 := mocks.NewMockAgent("idle1", "Idle Agent 1", "worker")
	idleAgent2 := mocks.NewMockAgent("idle2", "Idle Agent 2", "worker")
	busyAgent := mocks.NewMockAgent("busy1", "Busy Agent", "worker")
	busyAgent.SetStatus(agent.StatusWorking)
	
	// Register all agents
	dispatcher.RegisterAgent(activeAgent)
	dispatcher.RegisterAgent(idleAgent1)
	dispatcher.RegisterAgent(idleAgent2)
	dispatcher.RegisterAgent(busyAgent)
	
	// Set active agent
	dispatcher.activeAgents["active1"] = activeAgent
	
	// Get active agents
	activeAgents := dispatcher.GetActiveAgents()
	if len(activeAgents) != 1 {
		t.Fatalf("Expected 1 active agent, got %d", len(activeAgents))
	}
	
	if activeAgents[0].ID() != "active1" {
		t.Errorf("Expected active agent ID 'active1', got '%s'", activeAgents[0].ID())
	}
	
	// Get available agents (should be idle agents only)
	availableAgents := dispatcher.GetAvailableAgents()
	if len(availableAgents) != 2 {
		t.Fatalf("Expected 2 available agents, got %d", len(availableAgents))
	}
	
	// Verify available agent IDs
	var foundIdle1, foundIdle2 bool
	for _, agent := range availableAgents {
		if agent.ID() == "idle1" {
			foundIdle1 = true
		} else if agent.ID() == "idle2" {
			foundIdle2 = true
		}
	}
	
	if !foundIdle1 || !foundIdle2 {
		t.Errorf("Missing expected idle agents in available agents list")
	}
}

func TestDispatcherRunContextCancellation(t *testing.T) {
	dispatcher, _, _, _, _ := setupDispatcherTest()
	
	// Create cancellable context
	ctx, cancel := context.WithCancel(context.Background())
	
	// Run dispatcher in goroutine
	errCh := make(chan error)
	go func() {
		err := dispatcher.Run(ctx, 50*time.Millisecond)
		errCh <- err
	}()
	
	// Wait briefly for Run to start
	time.Sleep(100 * time.Millisecond)
	
	// Cancel context
	cancel()
	
	// Wait for Run to return
	select {
	case err := <-errCh:
		if err != context.Canceled {
			t.Fatalf("Expected context.Canceled error, got %v", err)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatalf("Timeout waiting for Run to return after context cancellation")
	}
}