// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package search

import (
	"strings"
	"testing"

	"github.com/lancekrogers/guild/tools"
)

// TestCraftSearchToolRegistry tests the search tool registry functionality
func TestCraftSearchToolRegistry(t *testing.T) {
	registry := tools.NewToolRegistry()
	workspacePath := "/tmp/test"

	err := RegisterSearchTools(registry, workspacePath)
	if err != nil {
		t.Fatalf("Failed to register search tools: %v", err)
	}

	// Check that ag tool is registered
	agTool, exists := registry.GetTool("ag")
	if !exists {
		t.Error("Expected ag tool to be registered")
	}

	if agTool == nil {
		t.Error("Expected ag tool to not be nil")
	}

	if agTool.Name() != "ag" {
		t.Errorf("Expected tool name to be 'ag', got '%s'", agTool.Name())
	}
}

// TestGuildSearchToolsInfo tests the search tools info structure
func TestGuildSearchToolsInfo(t *testing.T) {
	workspacePath := "/tmp/test"
	searchTools := GetSearchTools(workspacePath)

	if len(searchTools) == 0 {
		t.Fatal("Expected at least one search tool")
	}

	// Check ag tool info
	agToolInfo := searchTools[0]
	if agToolInfo.Tool.Name() != "ag" {
		t.Errorf("Expected first tool to be 'ag', got '%s'", agToolInfo.Tool.Name())
	}

	if agToolInfo.CostMagnitude != 0 {
		t.Errorf("Expected ag tool cost magnitude to be 0, got %d", agToolInfo.CostMagnitude)
	}

	expectedCapabilities := []string{"search", "text_search", "code_search", "pattern_matching", "file_filtering"}
	if len(agToolInfo.Capabilities) != len(expectedCapabilities) {
		t.Errorf("Expected %d capabilities, got %d", len(expectedCapabilities), len(agToolInfo.Capabilities))
	}

	for _, expected := range expectedCapabilities {
		found := false
		for _, capability := range agToolInfo.Capabilities {
			if capability == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected capability '%s' not found in %v", expected, agToolInfo.Capabilities)
		}
	}
}

// TestScribeAgToolCreation tests creating ag tool through registry functions
func TestScribeAgToolCreation(t *testing.T) {
	workspacePath := "/tmp/test"
	agTool := GetAgTool(workspacePath)

	if agTool == nil {
		t.Fatal("Expected ag tool to be created")
	}

	if agTool.Name() != "ag" {
		t.Errorf("Expected tool name to be 'ag', got '%s'", agTool.Name())
	}

	if agTool.workingDir != workspacePath {
		t.Errorf("Expected working directory to be '%s', got '%s'", workspacePath, agTool.workingDir)
	}
}

// TestJourneymanFileTypeSupport tests the supported file types
func TestJourneymanFileTypeSupport(t *testing.T) {
	fileTypes := GetSupportedFileTypes()

	if len(fileTypes) == 0 {
		t.Fatal("Expected supported file types to be returned")
	}

	// Check for common file types
	expectedTypes := []string{"go", "js", "py", "java", "html", "css", "json", "yaml"}
	for _, expected := range expectedTypes {
		found := false
		for _, fileType := range fileTypes {
			if fileType == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected file type '%s' not found in supported types", expected)
		}
	}
}

// TestJourneymanIgnorePatterns tests the common ignore patterns
func TestJourneymanIgnorePatterns(t *testing.T) {
	ignorePatterns := GetCommonIgnorePatterns()

	if len(ignorePatterns) == 0 {
		t.Fatal("Expected ignore patterns to be returned")
	}

	// Check for common ignore patterns
	expectedPatterns := []string{"node_modules", "*.min.js", ".git", "*.pyc", "build"}
	for _, expected := range expectedPatterns {
		found := false
		for _, pattern := range ignorePatterns {
			if pattern == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected ignore pattern '%s' not found in patterns", expected)
		}
	}
}

// MockCostAwareRegistry for testing cost-aware registration
type MockCostAwareRegistry struct {
	tools map[string]MockToolInfo
}

type MockToolInfo struct {
	tool          tools.Tool
	costMagnitude int
	capabilities  []string
}

func NewMockCostAwareRegistry() *MockCostAwareRegistry {
	return &MockCostAwareRegistry{
		tools: make(map[string]MockToolInfo),
	}
}

func (r *MockCostAwareRegistry) RegisterToolWithCost(name string, tool tools.Tool, costMagnitude int, capabilities []string) error {
	r.tools[name] = MockToolInfo{
		tool:          tool,
		costMagnitude: costMagnitude,
		capabilities:  capabilities,
	}
	return nil
}

// TestCraftCostAwareRegistration tests cost-aware tool registration
func TestCraftCostAwareRegistration(t *testing.T) {
	registry := NewMockCostAwareRegistry()
	workspacePath := "/tmp/test"

	err := RegisterSearchToolsWithCost(registry, workspacePath)
	if err != nil {
		t.Fatalf("Failed to register search tools with cost: %v", err)
	}

	// Check that ag tool is registered with correct cost information
	agToolInfo, exists := registry.tools["ag"]
	if !exists {
		t.Error("Expected ag tool to be registered")
	}

	if agToolInfo.costMagnitude != 0 {
		t.Errorf("Expected ag tool cost magnitude to be 0, got %d", agToolInfo.costMagnitude)
	}

	expectedCapabilities := []string{"search", "text_search", "code_search", "pattern_matching", "file_filtering"}
	if len(agToolInfo.capabilities) != len(expectedCapabilities) {
		t.Errorf("Expected %d capabilities, got %d", len(expectedCapabilities), len(agToolInfo.capabilities))
	}
}

// TestCraftInvalidCostRegistry tests registration with invalid registry
func TestCraftInvalidCostRegistry(t *testing.T) {
	// Test with a registry that doesn't support cost-aware registration
	invalidRegistry := tools.NewToolRegistry()
	workspacePath := "/tmp/test"

	err := RegisterSearchToolsWithCost(invalidRegistry, workspacePath)
	if err == nil {
		t.Error("Expected error when registering with invalid registry")
	}

	if !strings.Contains(err.Error(), "does not support cost-aware registration") {
		t.Errorf("Expected error about cost-aware registration, got: %v", err)
	}
}

// TestGuildValidateAgInstallation tests ag installation validation
func TestGuildValidateAgInstallation(t *testing.T) {
	err := ValidateAgInstallation()

	// This test will pass if ag is installed, or fail with a specific error if not
	// We can't assert the exact result since it depends on the system
	if err != nil {
		// If ag is not installed, error should mention installation instructions
		if !strings.Contains(err.Error(), "not installed") {
			t.Errorf("Expected error about ag not being installed, got: %v", err)
		}
		t.Logf("ag not installed on system (this is expected if ag is not available): %v", err)
	} else {
		t.Log("ag is installed and available on system")
	}
}

// TestGuildRegistryErrorHandling tests error handling in registry functions
func TestGuildRegistryErrorHandling(t *testing.T) {
	// Test with nil registry (should cause panic or error in real usage)
	// but we'll test the validation of arguments

	workspacePath := "/tmp/test"
	searchTools := GetSearchTools(workspacePath)

	// Ensure tools are created properly
	if len(searchTools) == 0 {
		t.Error("Expected search tools to be created even with potential registry issues")
	}

	// Test individual tool creation
	agTool := GetAgTool("")
	if agTool == nil {
		t.Error("Expected ag tool to be created even with empty workspace path")
	}
}
