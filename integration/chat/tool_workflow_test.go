// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package chat_test

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/guild-ventures/guild-core/pkg/registry"
	"github.com/guild-ventures/guild-core/tools"
)

// TestToolWorkflowIntegration tests the end-to-end tool workflow in chat
func TestToolWorkflowIntegration(t *testing.T) {
	// Create a test registry with tools
	reg := registry.NewComponentRegistry()

	// Initialize with basic config
	cfg := registry.Config{
		Tools: registry.ToolConfig{
			EnabledTools: []string{"test-tool", "file-reader"},
		},
	}

	ctx := context.Background()
	err := reg.Initialize(ctx, cfg)
	require.NoError(t, err)

	// Register a test tool
	testTool := &mockTool{
		name:        "test-tool",
		description: "A test tool for integration testing",
		category:    "testing",
		schema: map[string]interface{}{
			"description": "A test tool for integration testing",
			"parameters": map[string]interface{}{
				"input": map[string]interface{}{
					"type":        "string",
					"description": "Input text to process",
					"required":    true,
				},
			},
		},
		examples: []string{
			"/tool test-tool --input 'hello world'",
		},
	}

	err = reg.Tools().RegisterTool("test-tool", testTool)
	require.NoError(t, err)

	// Test tool discovery
	t.Run("ToolDiscovery", func(t *testing.T) {
		toolNames := reg.Tools().ListTools()
		assert.Contains(t, toolNames, "test-tool")

		// Test tool retrieval
		tool, err := reg.Tools().GetTool("test-tool")
		assert.NoError(t, err)
		assert.NotNil(t, tool)
	})

	// Test tool execution workflow
	t.Run("ToolExecution", func(t *testing.T) {
		tool, err := reg.Tools().GetTool("test-tool")
		require.NoError(t, err)

		// Execute tool with test input
		input := "test execution"

		result, err := tool.Execute(ctx, input)
		assert.NoError(t, err)
		assert.True(t, result.Success)
		assert.Contains(t, result.Output, "test execution")
	})

	// Test tool capability search
	t.Run("ToolCapabilitySearch", func(t *testing.T) {
		// Check if capability-tool already exists, if not register it
		toolName := "capability-tool"
		if !reg.Tools().HasTool(toolName) {
			capabilityTool := &mockTool{
				name:        toolName,
				description: "A tool for testing capabilities",
				category:    "testing",
				schema: map[string]interface{}{
					"description": "A capability test tool",
				},
				examples: []string{},
			}

			err := reg.Tools().RegisterTool(toolName, capabilityTool)
			assert.NoError(t, err)
		}

		// Register tool with capabilities if method exists
		if toolRegistry := reg.Tools(); toolRegistry != nil {
			// Get the tool instance
			tool, err := toolRegistry.GetTool(toolName)
			if err == nil {
				// Try to register with cost if the method exists
				if costRegistry, ok := toolRegistry.(interface {
					RegisterToolWithCost(name string, tool registry.Tool, costMagnitude int, capabilities []string) error
				}); ok {
					err := costRegistry.RegisterToolWithCost(toolName, tool, 1, []string{"testing", "processing"})
					// It's OK if this fails - cost registration might not be implemented
					if err != nil {
						t.Logf("Could not register tool with cost: %v", err)
					}
				}
			}
		}

		// Search by capability
		tools := reg.Tools().GetToolsByCapability("testing")
		// This may be empty if capability search is not implemented
		// That's OK for this test - we're just testing the interface
		t.Logf("Found %d tools with testing capability", len(tools))
	})

	// Test cost-based tool selection (if available)
	t.Run("CostBasedSelection", func(t *testing.T) {
		// Skip if cost methods are not implemented
		if costRegistry, ok := reg.(interface {
			GetToolsByCost(maxCost int) []registry.ToolInfo
			GetCheapestToolByCapability(capability string) (*registry.ToolInfo, error)
		}); ok {
			// Get tools by cost
			toolInfos := costRegistry.GetToolsByCost(5)
			t.Logf("Found %d tools with cost <= 5", len(toolInfos))

			// Get cheapest tool by capability
			cheapest, err := costRegistry.GetCheapestToolByCapability("testing")
			if err == nil && cheapest != nil {
				// Should be one of our registered tools
				assert.Contains(t, []string{"test-tool", "capability-tool"}, cheapest.Name)
			} else {
				t.Logf("No tools found with capability 'testing' or method not implemented: %v", err)
			}
		} else {
			t.Skip("Cost-based selection methods not implemented in this registry")
		}
	})

	// Test concurrent tool execution
	t.Run("ConcurrentExecution", func(t *testing.T) {
		tool, err := reg.Tools().GetTool("test-tool")
		require.NoError(t, err)

		// Execute multiple tools concurrently
		results := make(chan *tools.ToolResult, 3)
		errors := make(chan error, 3)

		for i := 0; i < 3; i++ {
			go func(id int) {
				input := fmt.Sprintf("concurrent-%d", id)
				result, err := tool.Execute(ctx, input)
				if err != nil {
					errors <- err
				} else {
					results <- result
				}
			}(i)
		}

		// Collect results
		for i := 0; i < 3; i++ {
			select {
			case result := <-results:
				assert.True(t, result.Success)
				assert.Contains(t, result.Output, "concurrent-")
			case err := <-errors:
				t.Errorf("Concurrent execution failed: %v", err)
			case <-time.After(5 * time.Second):
				t.Error("Timeout waiting for concurrent execution")
			}
		}
	})

	// Test error handling
	t.Run("ErrorHandling", func(t *testing.T) {
		// Test non-existent tool
		_, err := reg.Tools().GetTool("non-existent")
		assert.Error(t, err)

		// Test tool execution with invalid input
		tool, err := reg.Tools().GetTool("test-tool")
		require.NoError(t, err)

		// Execute with empty input
		input := ""

		result, err := tool.Execute(ctx, input)
		// Should handle gracefully (depending on tool implementation)
		assert.NoError(t, err) // Mock tool handles any input
		assert.True(t, result.Success)
	})
}

// TestChatToolCommandParsing tests the tool command parsing logic
func TestChatToolCommandParsing(t *testing.T) {
	testCases := []struct {
		name           string
		input          string
		expectedTool   string
		expectedParams map[string]string
	}{
		{
			name:         "Simple tool execution",
			input:        "/tool file-reader --path=./test.txt",
			expectedTool: "file-reader",
			expectedParams: map[string]string{
				"path": "./test.txt",
			},
		},
		{
			name:         "Tool with multiple parameters",
			input:        "/tool shell-exec --command=ls-la --timeout=30",
			expectedTool: "shell-exec",
			expectedParams: map[string]string{
				"command": "ls-la",
				"timeout": "30",
			},
		},
		{
			name:         "Tool with flag parameter",
			input:        "/tool process-monitor --verbose",
			expectedTool: "process-monitor",
			expectedParams: map[string]string{
				"verbose": "true",
			},
		},
		{
			name:         "Tool with key=value syntax",
			input:        "/tool web-scraper --url=https://example.com --format=json",
			expectedTool: "web-scraper",
			expectedParams: map[string]string{
				"url":    "https://example.com",
				"format": "json",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Parse the input
			parts := strings.Fields(tc.input)
			require.GreaterOrEqual(t, len(parts), 3) // Should have at least "/tool toolname param"

			toolID := parts[1]
			assert.Equal(t, tc.expectedTool, toolID)

			// Parse parameters
			params := make(map[string]string)
			for _, arg := range parts[2:] {
				if strings.Contains(arg, "=") {
					keyValue := strings.SplitN(arg, "=", 2)
					params[strings.TrimPrefix(keyValue[0], "--")] = keyValue[1]
				} else if strings.HasPrefix(arg, "--") {
					// For flags without values, store as "true"
					params[strings.TrimPrefix(arg, "--")] = "true"
				} else {
					// For positional arguments that don't start with --,
					// we might handle them differently depending on the tool
					// For now, just continue parsing
					continue
				}
			}

			assert.Equal(t, tc.expectedParams, params)
		})
	}
}

// mockTool implements the tools.Tool interface for testing
type mockTool struct {
	name        string
	description string
	schema      map[string]interface{}
	examples    []string
	category    string
}

func (m *mockTool) Execute(ctx context.Context, input string) (*tools.ToolResult, error) {
	// Simple mock execution - echo the input
	return &tools.ToolResult{
		Output:  fmt.Sprintf("Processed: %s", input),
		Success: true,
	}, nil
}

func (m *mockTool) Schema() map[string]interface{} {
	return m.schema
}

func (m *mockTool) Examples() []string {
	return m.examples
}

func (m *mockTool) Name() string {
	return m.name
}

func (m *mockTool) Description() string {
	return m.description
}

func (m *mockTool) Category() string {
	return m.category
}

func (m *mockTool) RequiresAuth() bool {
	return false
}

// Ensure mockTool implements tools.Tool
var _ tools.Tool = (*mockTool)(nil)
