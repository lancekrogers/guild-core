package mocks

import (
	"context"
	"sync"
	"time"

	"github.com/guild-ventures/guild-core/pkg/memory"
	"github.com/guild-ventures/guild-core/internal/commission"
)

// MockCommissionManager implements the objective.Manager interface for testing
type MockCommissionManager struct {
	mu          sync.RWMutex
	objectives  map[string]*commission.Commission
	error       error
}

// NewMockCommissionManager creates a new mock objective manager
func NewMockCommissionManager() *MockCommissionManager {
	return &MockCommissionManager{
		objectives: make(map[string]*commission.Commission),
	}
}

// WithError configures the mock to return an error
func (m *MockCommissionManager) WithError(err error) *MockCommissionManager {
	m.error = err
	return m
}

// WithObjective adds an objective to the manager
func (m *MockCommissionManager) WithObjective(obj *commission.Commission) *MockCommissionManager {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.objectives[obj.ID] = obj
	return m
}

// Init implements objective.Manager.Init
func (m *MockCommissionManager) Init(ctx context.Context) error {
	if m.error != nil {
		return m.error
	}
	return nil
}

// SaveObjective implements objective.Manager.SaveObjective
func (m *MockCommissionManager) SaveObjective(ctx context.Context, obj *commission.Commission) error {
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
func (m *MockCommissionManager) GetObjective(ctx context.Context, objectiveID string) (*commission.Commission, error) {
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
func (m *MockCommissionManager) ListObjectives(ctx context.Context) ([]*commission.Commission, error) {
	if m.error != nil {
		return nil, m.error
	}
	
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	var objectives []*commission.Commission
	for _, obj := range m.objectives {
		objectives = append(objectives, obj)
	}
	
	return objectives, nil
}

// DeleteObjective implements objective.Manager.DeleteObjective
func (m *MockCommissionManager) DeleteObjective(ctx context.Context, objectiveID string) error {
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
func (m *MockCommissionManager) LoadObjectiveFromFile(ctx context.Context, filePath string) (*commission.Commission, error) {
	if m.error != nil {
		return nil, m.error
	}
	
	// Just return a new objective for testing purposes
	obj := commission.NewCommission("Test Objective", "Loaded from mock file")
	
	return obj, nil
}

// AddTask implements objective.Manager.AddTask
func (m *MockCommissionManager) AddTask(ctx context.Context, objectiveID string, task *commission.CommissionTask) error {
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
func (m *MockCommissionManager) UpdateTaskStatus(ctx context.Context, objectiveID string, taskID string, status string) error {
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