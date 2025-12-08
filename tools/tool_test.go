// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package tools_test

import (
	"context"
	"errors"
	"testing"

	"github.com/guild-framework/guild-core/tools"
	"github.com/guild-framework/guild-core/tools/mocks"
)

// TestBaseToolImplementation tests that BaseTool implements the Tool interface
func TestBaseToolImplementation(t *testing.T) {
	var _ tools.Tool = &tools.BaseTool{}
}

// TestNewBaseTool tests creating a new base tool
func TestNewBaseTool(t *testing.T) {
	name := "test-tool"
	description := "A tool for testing"
	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"param": map[string]interface{}{
				"type": "string",
			},
		},
	}
	category := "test"
	needsAuth := false
	examples := []string{"example input"}

	baseTool := tools.NewBaseTool(name, description, schema, category, needsAuth, examples)

	if baseTool.Name() != name {
		t.Errorf("Expected name '%s', got '%s'", name, baseTool.Name())
	}

	if baseTool.Description() != description {
		t.Errorf("Expected description '%s', got '%s'", description, baseTool.Description())
	}

	if baseTool.Category() != category {
		t.Errorf("Expected category '%s', got '%s'", category, baseTool.Category())
	}

	if baseTool.RequiresAuth() != needsAuth {
		t.Errorf("Expected requires auth %v, got %v", needsAuth, baseTool.RequiresAuth())
	}

	if len(baseTool.Examples()) != len(examples) {
		t.Errorf("Expected %d examples, got %d", len(examples), len(baseTool.Examples()))
	}

	if len(examples) > 0 && baseTool.Examples()[0] != examples[0] {
		t.Errorf("Expected example '%s', got '%s'", examples[0], baseTool.Examples()[0])
	}

	baseSchema := baseTool.Schema()
	if baseSchema["type"] != schema["type"] {
		t.Errorf("Expected schema type '%s', got '%s'", schema["type"], baseSchema["type"])
	}
}

// TestNewToolResult tests creating a new tool result
func TestNewToolResult(t *testing.T) {
	// Test successful result
	output := "test output"
	metadata := map[string]string{"key": "value"}
	extraData := map[string]interface{}{"extra": "data"}

	result := tools.NewToolResult(output, metadata, nil, extraData)

	if result.Output != output {
		t.Errorf("Expected output '%s', got '%s'", output, result.Output)
	}

	if result.Metadata["key"] != metadata["key"] {
		t.Errorf("Expected metadata key 'key' to have value '%s', got '%s'", metadata["key"], result.Metadata["key"])
	}

	if !result.Success {
		t.Error("Expected success to be true")
	}

	if result.Error != "" {
		t.Errorf("Expected empty error, got '%s'", result.Error)
	}

	if result.ExtraData["extra"] != extraData["extra"] {
		t.Errorf("Expected extra data key 'extra' to have value '%s', got '%s'", extraData["extra"], result.ExtraData["extra"])
	}

	// Test result with error
	err := errors.New("test error")
	result = tools.NewToolResult(output, metadata, err, extraData)

	if result.Success {
		t.Error("Expected success to be false")
	}

	if result.Error != "test error" {
		t.Errorf("Expected error 'test error', got '%s'", result.Error)
	}
}

// TestNewToolRegistry tests creating a new tool registry
func TestNewToolRegistry(t *testing.T) {
	registry := tools.NewToolRegistry()

	if registry == nil {
		t.Fatal("Expected non-nil registry")
	}

	tools := registry.ListTools()
	if len(tools) != 0 {
		t.Errorf("Expected empty registry, got %d tools", len(tools))
	}
}

// TestRegisterTool tests registering a tool
func TestRegisterTool(t *testing.T) {
	registry := tools.NewToolRegistry()
	mockTool := mocks.NewMockTool("test-tool", "A tool for testing")

	// Register the tool
	err := registry.RegisterTool(mockTool)
	if err != nil {
		t.Fatalf("Failed to register tool: %v", err)
	}

	// Verify the tool was registered
	tools := registry.ListTools()
	if len(tools) != 1 {
		t.Errorf("Expected 1 tool, got %d", len(tools))
	}

	if len(tools) > 0 && tools[0].Name() != "test-tool" {
		t.Errorf("Expected tool name 'test-tool', got '%s'", tools[0].Name())
	}

	// Try to register a nil tool
	err = registry.RegisterTool(nil)
	if err == nil {
		t.Error("Expected error registering nil tool, got nil")
	}

	// Try to register a tool with an empty name
	emptyNameTool := mocks.NewMockTool("", "Empty name tool")
	err = registry.RegisterTool(emptyNameTool)
	if err == nil {
		t.Error("Expected error registering tool with empty name, got nil")
	}

	// Try to register a tool with the same name
	duplicateTool := mocks.NewMockTool("test-tool", "Duplicate tool")
	err = registry.RegisterTool(duplicateTool)
	if err == nil {
		t.Error("Expected error registering duplicate tool, got nil")
	}
}

// TestGetTool tests getting a tool by name
func TestGetTool(t *testing.T) {
	registry := tools.NewToolRegistry()
	mockTool := mocks.NewMockTool("test-tool", "A tool for testing")

	// Register the tool
	err := registry.RegisterTool(mockTool)
	if err != nil {
		t.Fatalf("Failed to register tool: %v", err)
	}

	// Get the tool
	tool, exists := registry.GetTool("test-tool")
	if !exists {
		t.Fatal("Expected tool to exist")
	}

	if tool.Name() != "test-tool" {
		t.Errorf("Expected tool name 'test-tool', got '%s'", tool.Name())
	}

	// Try to get a non-existent tool
	_, exists = registry.GetTool("non-existent")
	if exists {
		t.Error("Expected non-existent tool to not exist")
	}
}

// TestListTools tests listing all tools
func TestListTools(t *testing.T) {
	registry := tools.NewToolRegistry()

	// Register multiple tools
	tool1 := mocks.NewMockTool("tool1", "Tool 1")
	tool2 := mocks.NewMockTool("tool2", "Tool 2")
	tool3 := mocks.NewMockTool("tool3", "Tool 3")

	registry.RegisterTool(tool1)
	registry.RegisterTool(tool2)
	registry.RegisterTool(tool3)

	// List all tools
	tools := registry.ListTools()
	if len(tools) != 3 {
		t.Errorf("Expected 3 tools, got %d", len(tools))
	}

	// Check if all tools are present
	toolNames := map[string]bool{}
	for _, tool := range tools {
		toolNames[tool.Name()] = true
	}

	if !toolNames["tool1"] || !toolNames["tool2"] || !toolNames["tool3"] {
		t.Error("Not all tools were returned")
	}
}

// TestListToolsByCategory tests listing tools by category
func TestListToolsByCategory(t *testing.T) {
	registry := tools.NewToolRegistry()

	// Register tools with different categories
	tool1 := mocks.NewMockTool("tool1", "Tool 1").WithCategory("category1")
	tool2 := mocks.NewMockTool("tool2", "Tool 2").WithCategory("category1")
	tool3 := mocks.NewMockTool("tool3", "Tool 3").WithCategory("category2")

	registry.RegisterTool(tool1)
	registry.RegisterTool(tool2)
	registry.RegisterTool(tool3)

	// List tools by category
	category1Tools := registry.ListToolsByCategory("category1")
	if len(category1Tools) != 2 {
		t.Errorf("Expected 2 tools in category1, got %d", len(category1Tools))
	}

	category2Tools := registry.ListToolsByCategory("category2")
	if len(category2Tools) != 1 {
		t.Errorf("Expected 1 tool in category2, got %d", len(category2Tools))
	}

	category3Tools := registry.ListToolsByCategory("category3")
	if len(category3Tools) != 0 {
		t.Errorf("Expected 0 tools in category3, got %d", len(category3Tools))
	}
}

// TestExecuteTool tests executing a tool
func TestExecuteTool(t *testing.T) {
	registry := tools.NewToolRegistry()
	ctx := context.Background()

	// Create a mock tool with a specific result
	mockTool := mocks.NewMockTool("test-tool", "A tool for testing")
	mockTool.WithResult(&tools.ToolResult{
		Output:  "Tool executed successfully",
		Success: true,
	})

	// Register the tool
	err := registry.RegisterTool(mockTool)
	if err != nil {
		t.Fatalf("Failed to register tool: %v", err)
	}

	// Execute the tool
	result, err := registry.ExecuteTool(ctx, "test-tool", "test input")
	if err != nil {
		t.Fatalf("Failed to execute tool: %v", err)
	}

	if !result.Success {
		t.Error("Expected successful execution")
	}

	if result.Output != "Tool executed successfully" {
		t.Errorf("Expected output 'Tool executed successfully', got '%s'", result.Output)
	}

	// Verify the tool was executed with the correct input
	if mockTool.LastInputValue != "test input" {
		t.Errorf("Expected input 'test input', got '%s'", mockTool.LastInputValue)
	}

	// Try to execute a non-existent tool
	_, err = registry.ExecuteTool(ctx, "non-existent", "test input")
	if err == nil {
		t.Error("Expected error executing non-existent tool, got nil")
	}

	// Try to execute a tool that returns an error
	mockTool.WithError(errors.New("test error"))
	_, err = registry.ExecuteTool(ctx, "test-tool", "test input")
	if err == nil {
		t.Error("Expected error from tool execution, got nil")
	}

	// Test with cancelled context
	cancelledCtx, cancel := context.WithCancel(context.Background())
	cancel()
	mockTool.WithError(nil) // Reset error
	_, err = registry.ExecuteTool(cancelledCtx, "test-tool", "test input")
	if err == nil {
		t.Error("Expected error with cancelled context, got nil")
	}
}

// TestExecuteToolWithParams tests executing a tool with parameters
func TestExecuteToolWithParams(t *testing.T) {
	registry := tools.NewToolRegistry()
	ctx := context.Background()

	// Create a mock tool
	mockTool := mocks.NewMockTool("test-tool", "A tool for testing")
	mockTool.WithResult(&tools.ToolResult{
		Output:  "Tool executed successfully with params",
		Success: true,
	})

	// Register the tool
	err := registry.RegisterTool(mockTool)
	if err != nil {
		t.Fatalf("Failed to register tool: %v", err)
	}

	// Execute the tool with parameters
	params := map[string]interface{}{
		"param1": "value1",
		"param2": 123,
		"param3": true,
	}
	result, err := registry.ExecuteToolWithParams(ctx, "test-tool", params)
	if err != nil {
		t.Fatalf("Failed to execute tool with params: %v", err)
	}

	if !result.Success {
		t.Error("Expected successful execution")
	}

	if result.Output != "Tool executed successfully with params" {
		t.Errorf("Expected output 'Tool executed successfully with params', got '%s'", result.Output)
	}

	// Verify the tool was executed with the correct parameters
	if mockTool.LastParamsValue["param1"] != "value1" {
		t.Errorf("Expected param1 'value1', got '%v'", mockTool.LastParamsValue["param1"])
	}

	if mockTool.LastParamsValue["param2"] != float64(123) {
		t.Errorf("Expected param2 123, got %v", mockTool.LastParamsValue["param2"])
	}

	if mockTool.LastParamsValue["param3"] != true {
		t.Errorf("Expected param3 true, got %v", mockTool.LastParamsValue["param3"])
	}
}
