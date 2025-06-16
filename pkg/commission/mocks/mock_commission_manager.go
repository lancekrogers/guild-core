// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package mocks

import (
	"context"

	"github.com/stretchr/testify/mock"

	"github.com/guild-ventures/guild-core/pkg/commission"
)

// MockCommissionManager is a mock implementation of the commission.Manager type.
type MockCommissionManager struct {
	mock.Mock
}

// GetCommission mocks the GetCommission method.
func (m *MockCommissionManager) GetCommission(ctx context.Context, commissionID string) (*commission.Commission, error) {
	args := m.Called(ctx, commissionID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*commission.Commission), args.Error(1)
}

// CreateCommission mocks the CreateCommission method.
func (m *MockCommissionManager) CreateCommission(ctx context.Context, obj *commission.Commission) (string, error) {
	args := m.Called(ctx, obj)
	return args.String(0), args.Error(1)
}

// UpdateCommission mocks the UpdateCommission method.
func (m *MockCommissionManager) UpdateCommission(ctx context.Context, obj *commission.Commission) error {
	args := m.Called(ctx, obj)
	return args.Error(0)
}

// DeleteCommission mocks the DeleteCommission method.
func (m *MockCommissionManager) DeleteCommission(ctx context.Context, commissionID string) error {
	args := m.Called(ctx, commissionID)
	return args.Error(0)
}

// ListCommissions mocks the ListCommissions method.
func (m *MockCommissionManager) ListCommissions(ctx context.Context) ([]*commission.Commission, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*commission.Commission), args.Error(1)
}

// GenerateCommission mocks the GenerateCommission method.
func (m *MockCommissionManager) GenerateCommission(ctx context.Context, prompt string) (*commission.Commission, error) {
	args := m.Called(ctx, prompt)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*commission.Commission), args.Error(1)
}

// ParseCommission mocks the ParseCommission method.
func (m *MockCommissionManager) ParseCommission(content string) (*commission.Commission, error) {
	args := m.Called(content)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*commission.Commission), args.Error(1)
}
