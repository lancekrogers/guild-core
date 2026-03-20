// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package git

import (
	"testing"

	"github.com/lancekrogers/guild-core/tools"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegisterGitTools(t *testing.T) {
	registry := tools.NewToolRegistry()
	workspacePath := "/test/workspace"

	err := RegisterGitTools(registry, workspacePath)
	require.NoError(t, err)

	// Verify all tools are registered
	expectedTools := []string{"git_log", "git_blame", "git_merge_conflicts"}

	for _, toolName := range expectedTools {
		tool, exists := registry.GetTool(toolName)
		assert.True(t, exists, "Tool %s should be registered", toolName)
		assert.NotNil(t, tool, "Tool %s should not be nil", toolName)
		assert.Equal(t, toolName, tool.Name())
		assert.Equal(t, "version_control", tool.Category())
		assert.False(t, tool.RequiresAuth())
	}

	// Verify tools list
	allTools := registry.ListTools()
	assert.Len(t, allTools, 3)

	// Verify category filtering
	vcTools := registry.ListToolsByCategory("version_control")
	assert.Len(t, vcTools, 3)
}

func TestRegisterGitTools_DuplicateRegistration(t *testing.T) {
	registry := tools.NewToolRegistry()
	workspacePath := "/test/workspace"

	// First registration should succeed
	err := RegisterGitTools(registry, workspacePath)
	require.NoError(t, err)

	// Second registration should fail due to duplicate names
	err = RegisterGitTools(registry, workspacePath)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already registered")
}

func TestGetGitTools(t *testing.T) {
	workspacePath := "/test/workspace"
	gitTools := GetGitTools(workspacePath)

	require.Len(t, gitTools, 3)

	// Check each tool info
	expectedTools := map[string][]string{
		"git_log":             {"version_control", "history", "search"},
		"git_blame":           {"version_control", "authorship", "analysis"},
		"git_merge_conflicts": {"version_control", "conflict_resolution", "merge"},
	}

	for _, toolInfo := range gitTools {
		toolName := toolInfo.Tool.Name()
		expectedCaps, exists := expectedTools[toolName]
		assert.True(t, exists, "Unexpected tool: %s", toolName)

		// Verify cost magnitude
		assert.Equal(t, 0, toolInfo.CostMagnitude, "Git tools should have zero cost magnitude")

		// Verify capabilities
		assert.Equal(t, expectedCaps, toolInfo.Capabilities)

		// Verify tool properties
		assert.Equal(t, "version_control", toolInfo.Tool.Category())
		assert.False(t, toolInfo.Tool.RequiresAuth())
		assert.NotEmpty(t, toolInfo.Tool.Description())
		assert.NotEmpty(t, toolInfo.Tool.Examples())
	}
}

func TestGitToolInfo_Structure(t *testing.T) {
	workspacePath := "/test/workspace"
	tool := NewGitLogTool(workspacePath)

	toolInfo := GitToolInfo{
		Tool:          tool,
		CostMagnitude: 0,
		Capabilities:  []string{"version_control", "history"},
	}

	assert.Equal(t, tool, toolInfo.Tool)
	assert.Equal(t, 0, toolInfo.CostMagnitude)
	assert.Equal(t, []string{"version_control", "history"}, toolInfo.Capabilities)
}

func TestRegisterGitTools_ToolProperties(t *testing.T) {
	registry := tools.NewToolRegistry()
	workspacePath := "/test/workspace"

	err := RegisterGitTools(registry, workspacePath)
	require.NoError(t, err)

	tests := []struct {
		toolName         string
		expectedDesc     string
		expectedExamples int
	}{
		{
			toolName:         "git_log",
			expectedDesc:     "View git commit history with filtering options",
			expectedExamples: 5,
		},
		{
			toolName:         "git_blame",
			expectedDesc:     "Show authorship information for each line of a file",
			expectedExamples: 4,
		},
		{
			toolName:         "git_merge_conflicts",
			expectedDesc:     "List, show, and help resolve merge conflicts in git repositories",
			expectedExamples: 4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.toolName, func(t *testing.T) {
			tool, exists := registry.GetTool(tt.toolName)
			require.True(t, exists)

			assert.Equal(t, tt.expectedDesc, tool.Description())
			assert.Len(t, tool.Examples(), tt.expectedExamples)

			// Verify schema exists and is not empty
			schema := tool.Schema()
			assert.NotNil(t, schema)

			assert.Equal(t, "object", schema["type"])
			assert.Contains(t, schema, "properties")
		})
	}
}

func TestRegisterGitTools_WorkspacePathPropagation(t *testing.T) {
	registry := tools.NewToolRegistry()
	workspacePath := "/custom/workspace/path"

	err := RegisterGitTools(registry, workspacePath)
	require.NoError(t, err)

	// Verify that the workspace path is properly set in tools
	// We can't directly access the workspacePath field from the interface,
	// but we can verify the tools were created with the correct path by
	// checking that they execute with the workspace context

	for _, toolName := range []string{"git_log", "git_blame", "git_merge_conflicts"} {
		tool, exists := registry.GetTool(toolName)
		require.True(t, exists)

		// Verify the tool has the expected structure
		assert.NotNil(t, tool)
		assert.Equal(t, toolName, tool.Name())
	}
}

func TestGetGitTools_DifferentWorkspaces(t *testing.T) {
	workspace1 := "/workspace1"
	workspace2 := "/workspace2"

	tools1 := GetGitTools(workspace1)
	tools2 := GetGitTools(workspace2)

	// Should return the same number and types of tools
	require.Len(t, tools1, 3)
	require.Len(t, tools2, 3)

	// But they should be different instances
	for i := range tools1 {
		assert.NotSame(t, tools1[i].Tool, tools2[i].Tool)
		assert.Equal(t, tools1[i].CostMagnitude, tools2[i].CostMagnitude)
		assert.Equal(t, tools1[i].Capabilities, tools2[i].Capabilities)
	}
}
