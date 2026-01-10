// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package commands

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/lancekrogers/guild-core/pkg/commission"
	"github.com/lancekrogers/guild-core/pkg/gerror"
)

// MockCommissionManager is a mock implementation of commission.Manager
type MockCommissionManager struct {
	mock.Mock
}

func (m *MockCommissionManager) CreateCommission(ctx context.Context, comm commission.Commission) (*commission.Commission, error) {
	args := m.Called(ctx, comm)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*commission.Commission), args.Error(1)
}

func (m *MockCommissionManager) GetCommission(ctx context.Context, id string) (*commission.Commission, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*commission.Commission), args.Error(1)
}

func (m *MockCommissionManager) UpdateCommission(ctx context.Context, comm commission.Commission) error {
	args := m.Called(ctx, comm)
	return args.Error(0)
}

func (m *MockCommissionManager) DeleteCommission(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockCommissionManager) ListCommissions(ctx context.Context) ([]*commission.Commission, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*commission.Commission), args.Error(1)
}

func (m *MockCommissionManager) SaveCommission(ctx context.Context, comm *commission.Commission) error {
	args := m.Called(ctx, comm)
	return args.Error(0)
}

func (m *MockCommissionManager) LoadCommissionFromFile(ctx context.Context, path string) (*commission.Commission, error) {
	args := m.Called(ctx, path)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*commission.Commission), args.Error(1)
}

func (m *MockCommissionManager) GetCommissionsByTag(ctx context.Context, tag string) ([]*commission.Commission, error) {
	args := m.Called(ctx, tag)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*commission.Commission), args.Error(1)
}

func (m *MockCommissionManager) SetCommission(ctx context.Context, commissionID string) error {
	args := m.Called(ctx, commissionID)
	return args.Error(0)
}

func TestNewCommissionCommand(t *testing.T) {
	mockManager := &MockCommissionManager{}
	cmd := NewCommissionCommand(mockManager)

	assert.NotNil(t, cmd)
	assert.Equal(t, mockManager, cmd.manager)
}

func TestExecute_ShowHelp(t *testing.T) {
	mockManager := &MockCommissionManager{}
	cmd := NewCommissionCommand(mockManager)
	ctx := context.Background()

	// Test with no args
	teaCmd := cmd.Execute(ctx, []string{})
	require.NotNil(t, teaCmd)

	result := teaCmd()
	cmdResult, ok := result.(CommissionCommandResult)
	require.True(t, ok)
	assert.Equal(t, "help", cmdResult.Type)
	assert.Contains(t, cmdResult.Message, "Commission Commands")

	// Test with explicit help
	teaCmd = cmd.Execute(ctx, []string{"help"})
	result = teaCmd()
	cmdResult, ok = result.(CommissionCommandResult)
	require.True(t, ok)
	assert.Equal(t, "help", cmdResult.Type)
}

func TestExecute_StartNew(t *testing.T) {
	mockManager := &MockCommissionManager{}
	cmd := NewCommissionCommand(mockManager)
	ctx := context.Background()

	teaCmd := cmd.Execute(ctx, []string{"new"})
	require.NotNil(t, teaCmd)

	result := teaCmd()
	cmdResult, ok := result.(CommissionCommandResult)
	require.True(t, ok)
	assert.Equal(t, "start_new", cmdResult.Type)
	assert.Contains(t, cmdResult.Message, "Starting new commission creation with Elena")

	data, ok := cmdResult.Data.(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "planning", data["mode"])
	assert.Equal(t, "elena-guild-master", data["agent"])
}

func TestExecute_ListCommissions(t *testing.T) {
	mockManager := &MockCommissionManager{}
	cmd := NewCommissionCommand(mockManager)
	ctx := context.Background()

	// Test empty list
	mockManager.On("ListCommissions", ctx).Return([]*commission.Commission{}, nil).Once()

	teaCmd := cmd.Execute(ctx, []string{"list"})
	result := teaCmd()
	cmdResult, ok := result.(CommissionCommandResult)
	require.True(t, ok)
	assert.Equal(t, "list", cmdResult.Type)
	assert.Contains(t, cmdResult.Message, "No commissions found")

	// Test with commissions
	testCommissions := []*commission.Commission{
		{
			ID:          "comm_123",
			Title:       "Test Commission",
			Description: "A test commission",
			Status:      commission.CommissionStatusActive,
		},
		{
			ID:          "comm_456",
			Title:       "Another Commission",
			Description: "",
			Status:      commission.CommissionStatusCompleted,
		},
	}

	mockManager.On("ListCommissions", ctx).Return(testCommissions, nil).Once()

	teaCmd = cmd.Execute(ctx, []string{"list"})
	result = teaCmd()
	cmdResult, ok = result.(CommissionCommandResult)
	require.True(t, ok)
	assert.Equal(t, "list", cmdResult.Type)
	assert.Contains(t, cmdResult.Message, "Available Commissions")
	assert.Contains(t, cmdResult.Message, "Test Commission")
	assert.Contains(t, cmdResult.Message, "Another Commission")

	// Test with error
	mockManager.On("ListCommissions", ctx).Return(nil, errors.New("storage error")).Once()

	teaCmd = cmd.Execute(ctx, []string{"list"})
	result = teaCmd()
	cmdResult, ok = result.(CommissionCommandResult)
	require.True(t, ok)
	assert.Equal(t, "error", cmdResult.Type)
	assert.NotNil(t, cmdResult.Error)
}

func TestExecute_ShowStatus(t *testing.T) {
	mockManager := &MockCommissionManager{}
	cmd := NewCommissionCommand(mockManager)
	ctx := context.Background()

	teaCmd := cmd.Execute(ctx, []string{"status"})
	result := teaCmd()
	cmdResult, ok := result.(CommissionCommandResult)
	require.True(t, ok)
	assert.Equal(t, "status", cmdResult.Type)
	assert.Contains(t, cmdResult.Message, "No active commission")
}

func TestExecute_TriggerRefinement(t *testing.T) {
	mockManager := &MockCommissionManager{}
	cmd := NewCommissionCommand(mockManager)
	ctx := context.Background()

	// Test without ID
	teaCmd := cmd.Execute(ctx, []string{"refine"})
	result := teaCmd()
	cmdResult, ok := result.(CommissionCommandResult)
	require.True(t, ok)
	assert.Equal(t, "error", cmdResult.Type)
	assert.Contains(t, cmdResult.Message, "specify a commission ID")

	// Test with valid ID
	testCommission := &commission.Commission{
		ID:    "comm_123",
		Title: "Test Commission",
	}

	mockManager.On("GetCommission", ctx, "comm_123").Return(testCommission, nil).Once()

	teaCmd = cmd.Execute(ctx, []string{"refine", "comm_123"})
	result = teaCmd()
	cmdResult, ok = result.(CommissionCommandResult)
	require.True(t, ok)
	assert.Equal(t, "refine", cmdResult.Type)
	assert.Contains(t, cmdResult.Message, "Starting refinement")
	assert.Equal(t, testCommission, cmdResult.Data)

	// Test with invalid ID
	mockManager.On("GetCommission", ctx, "invalid").Return(nil, gerror.New(gerror.ErrCodeNotFound, "not found", nil)).Once()

	teaCmd = cmd.Execute(ctx, []string{"refine", "invalid"})
	result = teaCmd()
	cmdResult, ok = result.(CommissionCommandResult)
	require.True(t, ok)
	assert.Equal(t, "error", cmdResult.Type)
	assert.NotNil(t, cmdResult.Error)
}

func TestExecute_ResumeCommission(t *testing.T) {
	mockManager := &MockCommissionManager{}
	cmd := NewCommissionCommand(mockManager)
	ctx := context.Background()

	// Test with valid commission ID
	testCommission := &commission.Commission{
		ID:    "comm_123",
		Title: "Test Commission",
	}

	mockManager.On("GetCommission", ctx, "comm_123").Return(testCommission, nil).Once()

	teaCmd := cmd.Execute(ctx, []string{"comm_123"})
	result := teaCmd()
	cmdResult, ok := result.(CommissionCommandResult)
	require.True(t, ok)
	assert.Equal(t, "resume", cmdResult.Type)
	assert.Contains(t, cmdResult.Message, "Resuming commission")
	assert.Equal(t, testCommission, cmdResult.Data)

	// Test with invalid ID and suggestions
	mockManager.On("GetCommission", ctx, "comm").Return(nil, errors.New("not found")).Once()
	mockManager.On("ListCommissions", ctx).Return([]*commission.Commission{
		{ID: "comm_123", Title: "Test Commission"},
		{ID: "comm_456", Title: "Another Commission"},
	}, nil).Once()

	teaCmd = cmd.Execute(ctx, []string{"comm"})
	result = teaCmd()
	cmdResult, ok = result.(CommissionCommandResult)
	require.True(t, ok)
	assert.Equal(t, "error", cmdResult.Type)
	assert.Contains(t, cmdResult.Message, "not found")
	assert.Contains(t, cmdResult.Message, "Did you mean")
}

func TestGetCompletions(t *testing.T) {
	mockManager := &MockCommissionManager{}
	cmd := NewCommissionCommand(mockManager)
	ctx := context.Background()

	// Set up test commissions
	testCommissions := []*commission.Commission{
		{ID: "comm_123", Title: "Test Commission"},
		{ID: "comm_456", Title: "Another Commission"},
	}

	// Set up mock to return commissions for all calls
	mockManager.On("ListCommissions", ctx).Return(testCommissions, nil)

	// Test base completions
	completions := cmd.GetCompletions(ctx, []string{})
	assert.Contains(t, completions, "new")
	assert.Contains(t, completions, "list")
	assert.Contains(t, completions, "status")
	assert.Contains(t, completions, "refine")
	assert.Contains(t, completions, "help")

	// Test partial command completion
	completions = cmd.GetCompletions(ctx, []string{"li"})
	assert.Contains(t, completions, "list")
	assert.NotContains(t, completions, "new")

	// Test commission ID completion

	completions = cmd.GetCompletions(ctx, []string{"comm"})
	assert.Contains(t, completions, "comm_123")
	assert.Contains(t, completions, "comm_456")

	// Test refine completion
	completions = cmd.GetCompletions(ctx, []string{"refine", "comm"})
	assert.Contains(t, completions, "comm_123")
	assert.Contains(t, completions, "comm_456")
}
