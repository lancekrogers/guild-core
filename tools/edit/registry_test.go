// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package edit

import (
	"testing"

	"github.com/lancekrogers/guild/tools"
)

func TestCraftEditToolsRegistry(t *testing.T) {
	registry := tools.NewToolRegistry()

	// Test registering all edit tools
	err := RegisterEditTools(registry)
	if err != nil {
		t.Fatalf("Failed to register edit tools: %v", err)
	}

	// Verify multi_edit tool is registered
	tool, exists := registry.GetTool("multi_edit")
	if !exists {
		t.Error("Expected multi_edit tool to be registered")
	}

	if tool == nil {
		t.Error("Expected multi_edit tool to not be nil")
	}

	// Verify tool properties
	if tool.Name() != "multi_edit" {
		t.Errorf("Expected tool name 'multi_edit', got '%s'", tool.Name())
	}

	if tool.Category() != "edit" {
		t.Errorf("Expected tool category 'edit', got '%s'", tool.Category())
	}

	// Verify other edit tools are also registered
	expectedTools := []string{"multi_edit", "apply_diff", "multi_refactor"}
	for _, toolName := range expectedTools {
		if _, exists := registry.GetTool(toolName); !exists {
			t.Errorf("Expected tool '%s' to be registered", toolName)
		}
	}
}

func TestCraftGetEditTools(t *testing.T) {
	editTools := GetEditTools()

	if len(editTools) == 0 {
		t.Error("Expected at least one edit tool")
	}

	// Find multi_edit tool in the list
	var multiEditTool *EditToolInfo
	for _, toolInfo := range editTools {
		if toolInfo.Tool.Name() == "multi_edit" {
			multiEditTool = &toolInfo
			break
		}
	}

	if multiEditTool == nil {
		t.Error("Expected to find multi_edit tool in edit tools list")
	}

	// Verify multi_edit tool properties
	if multiEditTool.CostMagnitude != 0 {
		t.Errorf("Expected multi_edit cost magnitude 0, got %d", multiEditTool.CostMagnitude)
	}

	expectedCapabilities := []string{"file_operations", "edit", "find_replace", "atomic"}
	if len(multiEditTool.Capabilities) != len(expectedCapabilities) {
		t.Errorf("Expected %d capabilities, got %d", len(expectedCapabilities), len(multiEditTool.Capabilities))
	}

	for i, cap := range expectedCapabilities {
		if i >= len(multiEditTool.Capabilities) || multiEditTool.Capabilities[i] != cap {
			t.Errorf("Expected capability '%s' at index %d, got '%s'", cap, i, multiEditTool.Capabilities[i])
		}
	}
}

func TestGuildEditToolsNoDuplicateRegistration(t *testing.T) {
	registry := tools.NewToolRegistry()

	// Register tools twice
	err := RegisterEditTools(registry)
	if err != nil {
		t.Fatalf("Failed first registration: %v", err)
	}

	err = RegisterEditTools(registry)
	if err == nil {
		t.Error("Expected error when registering tools twice")
	}
}
