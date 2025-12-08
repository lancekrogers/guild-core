// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package mocks

import (
	"context"

	"github.com/stretchr/testify/mock"

	"github.com/guild-framework/guild-core/tools"
)

// MockToolRegistry is a mock implementation of the tools.ToolRegistry type.
type MockToolRegistry struct {
	mock.Mock
}

// RegisterTool mocks the RegisterTool method.
func (m *MockToolRegistry) RegisterTool(tool tools.Tool) error {
	args := m.Called(tool)
	return args.Error(0)
}

// GetTool mocks the GetTool method.
func (m *MockToolRegistry) GetTool(name string) (tools.Tool, error) {
	args := m.Called(name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(tools.Tool), args.Error(1)
}

// ListTools mocks the ListTools method.
func (m *MockToolRegistry) ListTools() []tools.Tool {
	args := m.Called()
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).([]tools.Tool)
}

// ExecuteTool mocks the ExecuteTool method.
func (m *MockToolRegistry) ExecuteTool(ctx context.Context, name string, input string) (*tools.ToolResult, error) {
	args := m.Called(ctx, name, input)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*tools.ToolResult), args.Error(1)
}

// ExecuteToolWithParams mocks the ExecuteToolWithParams method.
func (m *MockToolRegistry) ExecuteToolWithParams(ctx context.Context, name string, params map[string]interface{}) (*tools.ToolResult, error) {
	args := m.Called(ctx, name, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*tools.ToolResult), args.Error(1)
}
