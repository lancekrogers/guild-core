package mocks

import (
	"context"
	"sync"
	"time"

	"github.com/guild-ventures/guild-core/pkg/commission"
	"github.com/guild-ventures/guild-core/pkg/memory"
)

// MockCommissionManager implements the objective.Manager interface for testing
type MockCommissionManager struct {
	mu         sync.RWMutex
	objectives map[string]*commission.Commission
	error      error
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

// Additional methods to implement CommissionManager interface

// CreateCommission implements commission.CommissionManager.CreateCommission
func (m *MockCommissionManager) CreateCommission(ctx context.Context, commission commission.Commission) (*commission.Commission, error) {
	if m.error != nil {
		return nil, m.error
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Create a copy and store it
	newCommission := commission
	newCommission.CreatedAt = time.Now().UTC()
	newCommission.UpdatedAt = time.Now().UTC()

	m.objectives[newCommission.ID] = &newCommission
	return &newCommission, nil
}

// GetCommission implements commission.CommissionManager.GetCommission
func (m *MockCommissionManager) GetCommission(ctx context.Context, id string) (*commission.Commission, error) {
	return m.GetObjective(ctx, id)
}

// UpdateCommission implements commission.CommissionManager.UpdateCommission
func (m *MockCommissionManager) UpdateCommission(ctx context.Context, commission commission.Commission) error {
	if m.error != nil {
		return m.error
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.objectives[commission.ID]; !ok {
		return memory.ErrNotFound
	}

	commission.UpdatedAt = time.Now().UTC()
	m.objectives[commission.ID] = &commission
	return nil
}

// DeleteCommission implements commission.CommissionManager.DeleteCommission
func (m *MockCommissionManager) DeleteCommission(ctx context.Context, id string) error {
	return m.DeleteObjective(ctx, id)
}

// ListCommissions implements commission.CommissionManager.ListCommissions
func (m *MockCommissionManager) ListCommissions(ctx context.Context) ([]*commission.Commission, error) {
	return m.ListObjectives(ctx)
}

// SaveCommission implements commission.CommissionManager.SaveCommission
func (m *MockCommissionManager) SaveCommission(ctx context.Context, commission *commission.Commission) error {
	return m.SaveObjective(ctx, commission)
}

// LoadCommissionFromFile implements commission.CommissionManager.LoadCommissionFromFile
func (m *MockCommissionManager) LoadCommissionFromFile(ctx context.Context, path string) (*commission.Commission, error) {
	return m.LoadObjectiveFromFile(ctx, path)
}

// GetCommissionsByTag implements commission.CommissionManager.GetCommissionsByTag
func (m *MockCommissionManager) GetCommissionsByTag(ctx context.Context, tag string) ([]*commission.Commission, error) {
	if m.error != nil {
		return nil, m.error
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []*commission.Commission
	for _, obj := range m.objectives {
		for _, objTag := range obj.Tags {
			if objTag == tag {
				result = append(result, obj)
				break
			}
		}
	}

	return result, nil
}

// SetCommission implements commission.CommissionManager.SetCommission
func (m *MockCommissionManager) SetCommission(ctx context.Context, commissionID string) error {
	if m.error != nil {
		return m.error
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	if _, ok := m.objectives[commissionID]; !ok {
		return memory.ErrNotFound
	}

	// For mock purposes, just verify the commission exists
	return nil
}
