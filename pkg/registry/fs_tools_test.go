// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package registry

import (
	"testing"

	"github.com/guild-ventures/guild-core/tools"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegisterFSTools(t *testing.T) {
	// Create a new tool registry
	toolRegistry := tools.NewToolRegistry()

	// Register FS tools
	err := RegisterFSTools(toolRegistry)
	require.NoError(t, err, "Failed to register FS tools")

	// Verify all expected tools are registered
	expectedTools := []string{"file", "glob", "grep"}

	for _, toolName := range expectedTools {
		tool, exists := toolRegistry.GetTool(toolName)
		assert.True(t, exists, "Tool %s should be registered", toolName)
		assert.NotNil(t, tool, "Tool %s should not be nil", toolName)
		assert.Equal(t, toolName, tool.Name(), "Tool name should match")
		assert.Equal(t, "filesystem", tool.Category(), "Tool category should be filesystem")
	}
}

func TestGetFSToolNames(t *testing.T) {
	names := GetFSToolNames()

	// Should have exactly 3 tools
	assert.Len(t, names, 3, "Should have 3 filesystem tools")

	// Check expected tools are present
	expectedTools := []string{"file", "glob", "grep"}
	assert.ElementsMatch(t, expectedTools, names, "Tool names should match expected")
}

func TestGetFSToolsByCategory(t *testing.T) {
	toolsByCategory := GetFSToolsByCategory()

	// Should have filesystem category
	fsTools, exists := toolsByCategory["filesystem"]
	assert.True(t, exists, "Should have filesystem category")

	// Should have 3 tools in filesystem category
	assert.Len(t, fsTools, 3, "Should have 3 tools in filesystem category")

	// Check expected tools are present
	expectedTools := []string{"file", "glob", "grep"}
	assert.ElementsMatch(t, expectedTools, fsTools, "Filesystem tools should match expected")
}

func TestFSToolsWithPkgRegistry(t *testing.T) {
	// Test with pkg registry interface
	registeredTools := make([]string, 0)

	// Create a mock registry that implements the RegisterTool interface
	mockRegistry := &mockPkgRegistry{
		registeredTools: &registeredTools,
	}

	// Register tools
	err := RegisterFSTools(mockRegistry)
	require.NoError(t, err, "Failed to register FS tools with pkg registry")

	// Verify all tools were registered
	expectedTools := []string{"file", "glob", "grep"}
	assert.ElementsMatch(t, expectedTools, registeredTools, "All FS tools should be registered")
}

// mockPkgRegistry implements the pkg registry interface for testing
type mockPkgRegistry struct {
	registeredTools *[]string
}

func (m *mockPkgRegistry) RegisterTool(tool tools.Tool) error {
	*m.registeredTools = append(*m.registeredTools, tool.Name())
	return nil
}
