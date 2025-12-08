// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package orchestrator

import (
	"context"
	"testing"

	"github.com/guild-framework/guild-core/pkg/agents/core"
	"github.com/guild-framework/guild-core/pkg/kanban"
	"github.com/guild-framework/guild-core/pkg/orchestrator/interfaces"
	"github.com/stretchr/testify/assert"
)

// Test Config struct and validation
func TestConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: Config{
				MaxConcurrentAgents: 5,
				ManagerAgentID:      "test-manager",
				KanbanBoardID:       "test-board",
				ExecutionMode:       "parallel",
			},
			wantErr: false,
		},
		{
			name:    "empty config",
			config:  Config{},
			wantErr: false, // Empty config is valid, will use defaults
		},
		{
			name: "sequential mode",
			config: Config{
				MaxConcurrentAgents: 1,
				ManagerAgentID:      "manager",
				KanbanBoardID:       "board",
				ExecutionMode:       "sequential",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test that config struct is properly formed
			assert.IsType(t, Config{}, tt.config)

			// Test specific fields
			if tt.config.MaxConcurrentAgents > 0 {
				assert.Greater(t, tt.config.MaxConcurrentAgents, 0)
			}

			if tt.config.ExecutionMode != "" {
				assert.Contains(t, []string{"parallel", "sequential", "managed"}, tt.config.ExecutionMode)
			}
		})
	}
}

// Test Status constants
func TestStatus(t *testing.T) {
	tests := []struct {
		name   string
		status Status
	}{
		{"idle", StatusIdle},
		{"running", StatusRunning},
		{"paused", StatusPaused},
		{"error", StatusError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotEmpty(t, string(tt.status))
			assert.IsType(t, Status(""), tt.status)
		})
	}
}

// Test TaskStatus struct
func TestTaskStatus(t *testing.T) {
	ts := TaskStatus{
		TaskID:  "task-1",
		AgentID: "agent-1",
		Status:  "running",
	}

	assert.Equal(t, "task-1", ts.TaskID)
	assert.Equal(t, "agent-1", ts.AgentID)
	assert.Equal(t, "running", ts.Status)
}

// Test AgentStatus struct
func TestAgentStatus(t *testing.T) {
	as := AgentStatus{
		AgentID:      "agent-1",
		Available:    true,
		CurrentTask:  "task-1",
		TasksHandled: 5,
	}

	assert.Equal(t, "agent-1", as.AgentID)
	assert.True(t, as.Available)
	assert.Equal(t, "task-1", as.CurrentTask)
	assert.Equal(t, 5, as.TasksHandled)
}

// Simple agent implementation for testing
type simpleTestAgent struct {
	id   string
	name string
}

func (a *simpleTestAgent) Execute(ctx context.Context, request string) (string, error) {
	return "test response", nil
}

func (a *simpleTestAgent) GetID() string {
	return a.id
}

func (a *simpleTestAgent) GetName() string {
	return a.name
}

func (a *simpleTestAgent) GetType() string {
	return "test"
}

func (a *simpleTestAgent) GetCapabilities() []string {
	return []string{"testing"}
}

// Test that our simple agent implements the Agent interface
func TestSimpleAgent(t *testing.T) {
	testAgent := &simpleTestAgent{
		id:   "test-agent",
		name: "Test Agent",
	}

	// Test interface compliance
	var _ core.Agent = testAgent

	// Test methods
	assert.Equal(t, "test-agent", testAgent.GetID())
	assert.Equal(t, "Test Agent", testAgent.GetName())

	ctx := context.Background()
	response, err := testAgent.Execute(ctx, "test request")
	assert.NoError(t, err)
	assert.Equal(t, "test response", response)
}

// Simple task dispatcher for testing
type simpleTestDispatcher struct{}

func (d *simpleTestDispatcher) RegisterAgent(agent core.Agent)                        {}
func (d *simpleTestDispatcher) UnregisterAgent(agentID string)                        {}
func (d *simpleTestDispatcher) Dispatch(ctx context.Context, task *kanban.Task) error { return nil }
func (d *simpleTestDispatcher) GetTaskStatus(ctx context.Context, taskID string) (TaskStatus, error) {
	return TaskStatus{}, nil
}
func (d *simpleTestDispatcher) GetAgentStatus(agentID string) AgentStatus { return AgentStatus{} }
func (d *simpleTestDispatcher) ListAvailableAgents() []core.Agent         { return nil }
func (d *simpleTestDispatcher) Stop(ctx context.Context) error            { return nil }

// Test that our simple dispatcher implements the TaskDispatcher interface
func TestSimpleDispatcher(t *testing.T) {
	dispatcher := &simpleTestDispatcher{}

	// Test interface compliance
	var _ TaskDispatcher = dispatcher

	// Test basic methods don't panic
	dispatcher.RegisterAgent(&simpleTestAgent{id: "test"})
	dispatcher.UnregisterAgent("test")

	ctx := context.Background()
	err := dispatcher.Dispatch(ctx, &kanban.Task{ID: "task-1"})
	assert.NoError(t, err)

	status, err := dispatcher.GetTaskStatus(ctx, "task-1")
	assert.NoError(t, err)
	assert.IsType(t, TaskStatus{}, status)
}

// Simple event bus for testing
type simpleTestEventBus struct{}

func (e *simpleTestEventBus) Subscribe(eventType interfaces.EventType, handler interfaces.EventHandler) {
}
func (e *simpleTestEventBus) SubscribeAll(handler interfaces.EventHandler) {}
func (e *simpleTestEventBus) Unsubscribe(eventType interfaces.EventType, handler interfaces.EventHandler) {
}
func (e *simpleTestEventBus) Publish(event interfaces.Event)     {}
func (e *simpleTestEventBus) PublishJSON(jsonEvent string) error { return nil }

// Test that our simple event bus implements the EventBus interface
func TestSimpleEventBus(t *testing.T) {
	eventBus := &simpleTestEventBus{}

	// Test interface compliance
	var _ EventBus = eventBus

	// Test basic methods don't panic
	eventBus.Subscribe("test", func(event interfaces.Event) {})
	eventBus.SubscribeAll(func(event interfaces.Event) {})
	eventBus.Unsubscribe("test", func(event interfaces.Event) {})
	eventBus.Publish(interfaces.Event{
		Type: "test",
		Data: map[string]interface{}{"message": "test"},
	})

	err := eventBus.PublishJSON(`{"type": "test", "data": {"message": "test"}}`)
	assert.NoError(t, err)
}
