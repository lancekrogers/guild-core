// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package tools

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBaseTool_HealthCheck(t *testing.T) {
	// Create a basic tool
	baseTool := NewBaseTool(
		"test-tool",
		"A test tool",
		map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"input": map[string]interface{}{"type": "string"},
			},
		},
		"test",
		false,
		[]string{"example input"},
	)

	// BaseTool health check should always return nil
	err := baseTool.HealthCheck()
	assert.NoError(t, err, "BaseTool health check should not return error")
}

// MockHealthyTool is a mock tool that is always healthy
type MockHealthyTool struct {
	*BaseTool
}

func (m *MockHealthyTool) HealthCheck() error {
	return nil
}

// MockUnhealthyTool is a mock tool that is always unhealthy
type MockUnhealthyTool struct {
	*BaseTool
	healthError error
}

func (m *MockUnhealthyTool) HealthCheck() error {
	return m.healthError
}

func TestToolRegistry_HealthCheck(t *testing.T) {
	registry := NewToolRegistry()

	// Register a healthy tool
	healthyTool := &MockHealthyTool{
		BaseTool: NewBaseTool("healthy", "Always healthy", nil, "test", false, nil),
	}
	err := registry.RegisterTool(healthyTool)
	require.NoError(t, err)

	// Register an unhealthy tool
	unhealthyTool := &MockUnhealthyTool{
		BaseTool:    NewBaseTool("unhealthy", "Always unhealthy", nil, "test", false, nil),
		healthError: assert.AnError,
	}
	err = registry.RegisterTool(unhealthyTool)
	require.NoError(t, err)

	// Test health check for healthy tool
	tool, exists := registry.GetTool("healthy")
	require.True(t, exists)
	err = tool.HealthCheck()
	assert.NoError(t, err, "Healthy tool should not return error")

	// Test health check for unhealthy tool
	tool, exists = registry.GetTool("unhealthy")
	require.True(t, exists)
	err = tool.HealthCheck()
	assert.Error(t, err, "Unhealthy tool should return error")
	assert.Equal(t, assert.AnError, err)
}

func TestHealthCheckAllTools(t *testing.T) {
	registry := NewToolRegistry()

	// Register multiple tools
	tools := []Tool{
		&MockHealthyTool{
			BaseTool: NewBaseTool("tool1", "Tool 1", nil, "test", false, nil),
		},
		&MockHealthyTool{
			BaseTool: NewBaseTool("tool2", "Tool 2", nil, "test", false, nil),
		},
		&MockUnhealthyTool{
			BaseTool:    NewBaseTool("tool3", "Tool 3", nil, "test", false, nil),
			healthError: assert.AnError,
		},
	}

	for _, tool := range tools {
		err := registry.RegisterTool(tool)
		require.NoError(t, err)
	}

	// Check health of all tools
	healthStatus := make(map[string]bool)
	for _, tool := range registry.ListTools() {
		err := tool.HealthCheck()
		healthStatus[tool.Name()] = err == nil
	}

	// Verify results
	assert.True(t, healthStatus["tool1"], "tool1 should be healthy")
	assert.True(t, healthStatus["tool2"], "tool2 should be healthy")
	assert.False(t, healthStatus["tool3"], "tool3 should be unhealthy")
}
