package mocks

import (
	"context"

	"github.com/guild-ventures/guild-core/pkg/commission"
	"github.com/stretchr/testify/mock"
)

// MockObjectiveManager is a mock implementation of the objective.Manager type.
type MockObjectiveManager struct {
	mock.Mock
}

// GetObjective mocks the GetObjective method.
func (m *MockObjectiveManager) GetObjective(ctx context.Context, objectiveID string) (*objective.Objective, error) {
	args := m.Called(ctx, objectiveID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*objective.Objective), args.Error(1)
}

// CreateObjective mocks the CreateObjective method.
func (m *MockObjectiveManager) CreateObjective(ctx context.Context, obj *objective.Objective) (string, error) {
	args := m.Called(ctx, obj)
	return args.String(0), args.Error(1)
}

// UpdateObjective mocks the UpdateObjective method.
func (m *MockObjectiveManager) UpdateObjective(ctx context.Context, obj *objective.Objective) error {
	args := m.Called(ctx, obj)
	return args.Error(0)
}

// DeleteObjective mocks the DeleteObjective method.
func (m *MockObjectiveManager) DeleteObjective(ctx context.Context, objectiveID string) error {
	args := m.Called(ctx, objectiveID)
	return args.Error(0)
}

// ListObjectives mocks the ListObjectives method.
func (m *MockObjectiveManager) ListObjectives(ctx context.Context) ([]*objective.Objective, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*objective.Objective), args.Error(1)
}

// GenerateObjective mocks the GenerateObjective method.
func (m *MockObjectiveManager) GenerateObjective(ctx context.Context, prompt string) (*objective.Objective, error) {
	args := m.Called(ctx, prompt)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*objective.Objective), args.Error(1)
}

// ParseObjective mocks the ParseObjective method.
func (m *MockObjectiveManager) ParseObjective(content string) (*objective.Objective, error) {
	args := m.Called(content)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*objective.Objective), args.Error(1)
}