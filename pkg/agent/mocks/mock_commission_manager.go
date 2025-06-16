// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package mocks

import (
	"context"
	"sync"
	"time"

	"github.com/guild-ventures/guild-core/pkg/commission"
	"github.com/guild-ventures/guild-core/pkg/memory"
)

// MockCommissionManager implements the commission.Manager interface for testing
type MockCommissionManager struct {
	mu          sync.RWMutex
	commissions map[string]*commission.Commission
	error       error
}

// NewMockCommissionManager creates a new mock commission manager
func NewMockCommissionManager() *MockCommissionManager {
	return &MockCommissionManager{
		commissions: make(map[string]*commission.Commission),
	}
}

// WithError configures the mock to return an error
func (m *MockCommissionManager) WithError(err error) *MockCommissionManager {
	m.error = err
	return m
}

// WithCommission adds a commission to the manager
func (m *MockCommissionManager) WithCommission(obj *commission.Commission) *MockCommissionManager {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.commissions[obj.ID] = obj
	return m
}

// Init implements commission.Manager.Init
func (m *MockCommissionManager) Init(ctx context.Context) error {
	if m.error != nil {
		return m.error
	}
	return nil
}

// SaveCommission implements commission.Manager.SaveCommission
func (m *MockCommissionManager) SaveCommission(ctx context.Context, obj *commission.Commission) error {
	if m.error != nil {
		return m.error
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	obj.UpdatedAt = time.Now().UTC()
	m.commissions[obj.ID] = obj

	return nil
}

// GetCommission implements commission.Manager.GetCommission
func (m *MockCommissionManager) GetCommission(ctx context.Context, commissionID string) (*commission.Commission, error) {
	if m.error != nil {
		return nil, m.error
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	obj, ok := m.commissions[commissionID]
	if !ok {
		return nil, memory.ErrNotFound
	}

	return obj, nil
}

// ListCommissions implements commission.Manager.ListCommissions
func (m *MockCommissionManager) ListCommissions(ctx context.Context) ([]*commission.Commission, error) {
	if m.error != nil {
		return nil, m.error
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	var commissions []*commission.Commission
	for _, obj := range m.commissions {
		commissions = append(commissions, obj)
	}

	return commissions, nil
}

// DeleteCommission implements commission.Manager.DeleteCommission
func (m *MockCommissionManager) DeleteCommission(ctx context.Context, commissionID string) error {
	if m.error != nil {
		return m.error
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.commissions[commissionID]; !ok {
		return memory.ErrNotFound
	}

	delete(m.commissions, commissionID)

	return nil
}

// LoadCommissionFromFile implements commission.Manager.LoadCommissionFromFile
func (m *MockCommissionManager) LoadCommissionFromFile(ctx context.Context, filePath string) (*commission.Commission, error) {
	if m.error != nil {
		return nil, m.error
	}

	// Just return a new commission for testing purposes
	obj := commission.NewCommission("Test Commission", "Loaded from mock file")

	return obj, nil
}

// AddTask implements commission.Manager.AddTask
func (m *MockCommissionManager) AddTask(ctx context.Context, commissionID string, task *commission.CommissionTask) error {
	if m.error != nil {
		return m.error
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	obj, ok := m.commissions[commissionID]
	if !ok {
		return memory.ErrNotFound
	}

	obj.Tasks = append(obj.Tasks, task)
	obj.UpdatedAt = time.Now().UTC()

	return nil
}

// UpdateTaskStatus implements commission.Manager.UpdateTaskStatus
func (m *MockCommissionManager) UpdateTaskStatus(ctx context.Context, commissionID string, taskID string, status string) error {
	if m.error != nil {
		return m.error
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	obj, ok := m.commissions[commissionID]
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

	m.commissions[newCommission.ID] = &newCommission
	return &newCommission, nil
}

// UpdateCommission implements commission.CommissionManager.UpdateCommission
func (m *MockCommissionManager) UpdateCommission(ctx context.Context, commission commission.Commission) error {
	if m.error != nil {
		return m.error
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.commissions[commission.ID]; !ok {
		return memory.ErrNotFound
	}

	commission.UpdatedAt = time.Now().UTC()
	m.commissions[commission.ID] = &commission
	return nil
}

// GetCommissionsByTag implements commission.CommissionManager.GetCommissionsByTag
func (m *MockCommissionManager) GetCommissionsByTag(ctx context.Context, tag string) ([]*commission.Commission, error) {
	if m.error != nil {
		return nil, m.error
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []*commission.Commission
	for _, obj := range m.commissions {
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

	if _, ok := m.commissions[commissionID]; !ok {
		return memory.ErrNotFound
	}

	// For mock purposes, just verify the commission exists
	return nil
}
