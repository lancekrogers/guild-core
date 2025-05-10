package mocks

import (
	"context"
	"sync"
	"time"

	"github.com/blockhead-consulting/Guild/pkg/memory"
	"github.com/blockhead-consulting/Guild/pkg/objective"
)

// MockObjectiveManager implements the objective.Manager interface for testing
type MockObjectiveManager struct {
	mu          sync.RWMutex
	objectives  map[string]*objective.Objective
	error       error
}

// NewMockObjectiveManager creates a new mock objective manager
func NewMockObjectiveManager() *MockObjectiveManager {
	return &MockObjectiveManager{
		objectives: make(map[string]*objective.Objective),
	}
}

// WithError configures the mock to return an error
func (m *MockObjectiveManager) WithError(err error) *MockObjectiveManager {
	m.error = err
	return m
}

// WithObjective adds an objective to the manager
func (m *MockObjectiveManager) WithObjective(obj *objective.Objective) *MockObjectiveManager {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.objectives[obj.ID] = obj
	return m
}

// Init implements objective.Manager.Init
func (m *MockObjectiveManager) Init(ctx context.Context) error {
	if m.error != nil {
		return m.error
	}
	return nil
}

// SaveObjective implements objective.Manager.SaveObjective
func (m *MockObjectiveManager) SaveObjective(ctx context.Context, obj *objective.Objective) error {
	if m.error != nil {
		return m.error
	}
	
	m.mu.Lock()
	defer m.mu.Unlock()
	
	obj.UpdatedAt = time.Now().UTC()
	m.objectives[obj.ID] = obj
	
	return nil
}

// GetObjective implements objective.Manager.GetObjective
func (m *MockObjectiveManager) GetObjective(ctx context.Context, objectiveID string) (*objective.Objective, error) {
	if m.error != nil {
		return nil, m.error
	}
	
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	obj, ok := m.objectives[objectiveID]
	if !ok {
		return nil, memory.ErrNotFound
	}
	
	return obj, nil
}

// ListObjectives implements objective.Manager.ListObjectives
func (m *MockObjectiveManager) ListObjectives(ctx context.Context) ([]*objective.Objective, error) {
	if m.error != nil {
		return nil, m.error
	}
	
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	var objectives []*objective.Objective
	for _, obj := range m.objectives {
		objectives = append(objectives, obj)
	}
	
	return objectives, nil
}

// DeleteObjective implements objective.Manager.DeleteObjective
func (m *MockObjectiveManager) DeleteObjective(ctx context.Context, objectiveID string) error {
	if m.error != nil {
		return m.error
	}
	
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if _, ok := m.objectives[objectiveID]; !ok {
		return memory.ErrNotFound
	}
	
	delete(m.objectives, objectiveID)
	
	return nil
}

// LoadObjectiveFromFile implements objective.Manager.LoadObjectiveFromFile
func (m *MockObjectiveManager) LoadObjectiveFromFile(ctx context.Context, filePath string) (*objective.Objective, error) {
	if m.error != nil {
		return nil, m.error
	}
	
	// Just return a new objective for testing purposes
	obj := objective.NewObjective("Test Objective", "Loaded from mock file")
	
	return obj, nil
}

// AddTask implements objective.Manager.AddTask
func (m *MockObjectiveManager) AddTask(ctx context.Context, objectiveID string, task objective.Task) error {
	if m.error != nil {
		return m.error
	}
	
	m.mu.Lock()
	defer m.mu.Unlock()
	
	obj, ok := m.objectives[objectiveID]
	if !ok {
		return memory.ErrNotFound
	}
	
	obj.Tasks = append(obj.Tasks, task)
	obj.UpdatedAt = time.Now().UTC()
	
	return nil
}

// UpdateTaskStatus implements objective.Manager.UpdateTaskStatus
func (m *MockObjectiveManager) UpdateTaskStatus(ctx context.Context, objectiveID string, taskID string, status string) error {
	if m.error != nil {
		return m.error
	}
	
	m.mu.Lock()
	defer m.mu.Unlock()
	
	obj, ok := m.objectives[objectiveID]
	if !ok {
		return memory.ErrNotFound
	}
	
	for i, task := range obj.Tasks {
		if task.ID == taskID {
			obj.Tasks[i].Status = status
			obj.UpdatedAt = time.Now().UTC()
			if status == "done" {
				now := time.Now().UTC()
				obj.Tasks[i].CompletedAt = &now
			}
			return nil
		}
	}
	
	return memory.ErrNotFound
}