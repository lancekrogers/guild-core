package mocks

import (
	"context"
	"sync"

	"github.com/guild-ventures/guild-core/pkg/agent"
	"github.com/guild-ventures/guild-core/pkg/kanban"
)

// MockAgent is a mock implementation of the agent.Agent interface
type MockAgent struct {
	id           string
	name         string
	agentType    string
	status       string
	currentTask  *kanban.Task
	tasks        []*kanban.Task
	taskHistory  []*kanban.Task
	mu           sync.Mutex
	executeError error
	state        *agent.State
}

// NewMockAgent creates a new mock agent
func NewMockAgent(id, name, agentType string) *MockAgent {
	return &MockAgent{
		id:          id,
		name:        name,
		agentType:   agentType,
		status:      agent.StatusIdle,
		tasks:       make([]*kanban.Task, 0),
		taskHistory: make([]*kanban.Task, 0),
		state: &agent.State{
			ID:          id,
			Name:        name,
			Type:        agentType,
			Status:      agent.StatusIdle,
			CurrentTask: "",
		},
	}
}

// ID returns the agent ID
func (m *MockAgent) ID() string {
	return m.id
}

// Name returns the agent name
func (m *MockAgent) Name() string {
	return m.name
}

// Type returns the agent type
func (m *MockAgent) Type() string {
	return m.agentType
}

// Status returns the agent status
func (m *MockAgent) Status() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.status
}

// GetState returns the agent state
func (m *MockAgent) GetState() *agent.State {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.state
}

// AssignTask assigns a task to the agent
func (m *MockAgent) AssignTask(ctx context.Context, task *kanban.Task) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.tasks = append(m.tasks, task)
	m.currentTask = task
	m.status = agent.StatusWorking
	m.state.Status = agent.StatusWorking
	m.state.CurrentTask = task.ID
	
	return nil
}

// Execute executes the agent's current task
func (m *MockAgent) Execute(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if m.executeError != nil {
		return m.executeError
	}
	
	// Simulate task execution
	if m.currentTask != nil {
		m.taskHistory = append(m.taskHistory, m.currentTask)
		m.currentTask = nil
	}
	
	m.status = agent.StatusIdle
	m.state.Status = agent.StatusIdle
	m.state.CurrentTask = ""
	
	return nil
}

// SetStatus sets the agent status
func (m *MockAgent) SetStatus(status string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.status = status
	m.state.Status = status
}

// SetExecuteError sets the error to be returned by Execute
func (m *MockAgent) SetExecuteError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.executeError = err
}

// GetTasks returns the agent's tasks
func (m *MockAgent) GetTasks() []*kanban.Task {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	tasks := make([]*kanban.Task, len(m.tasks))
	copy(tasks, m.tasks)
	
	return tasks
}

// GetTaskHistory returns the agent's task history
func (m *MockAgent) GetTaskHistory() []*kanban.Task {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	history := make([]*kanban.Task, len(m.taskHistory))
	copy(history, m.taskHistory)
	
	return history
}

// GetCurrentTask returns the agent's current task
func (m *MockAgent) GetCurrentTask() *kanban.Task {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	return m.currentTask
}