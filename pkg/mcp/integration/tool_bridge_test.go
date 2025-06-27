// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package integration

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lancekrogers/guild/pkg/mcp/protocol"
	"github.com/lancekrogers/guild/pkg/mcp/tools"
	"github.com/lancekrogers/guild/pkg/registry"
	basetools "github.com/lancekrogers/guild/tools"
)

// MockGuildTool implements the Guild Tool interface
type MockGuildTool struct {
	name     string
	executed bool
}

func (m *MockGuildTool) Name() string { return m.name }

func (m *MockGuildTool) Description() string { return "Mock Guild tool for testing" }

func (m *MockGuildTool) Category() string { return "test" }

func (m *MockGuildTool) RequiresAuth() bool { return false }

func (m *MockGuildTool) Examples() []string { return []string{`{"input": "test"}`} }

func (m *MockGuildTool) Schema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"input": map[string]interface{}{
				"type":        "string",
				"description": "Test input",
			},
		},
		"required": []interface{}{"input"},
	}
}

func (m *MockGuildTool) Execute(ctx context.Context, input string) (*basetools.ToolResult, error) {
	m.executed = true
	var params map[string]interface{}
	if err := json.Unmarshal([]byte(input), &params); err != nil {
		return nil, err
	}

	output := "Executed with: " + params["input"].(string)
	return basetools.NewToolResult(output, nil, nil, nil), nil
}

// MockMCPTool implements the MCP Tool interface
type MockMCPTool struct {
	id       string
	name     string
	executed bool
}

func (m *MockMCPTool) ID() string { return m.id }

func (m *MockMCPTool) Name() string { return m.name }

func (m *MockMCPTool) Description() string { return "Mock MCP tool for testing" }

func (m *MockMCPTool) Capabilities() []string { return []string{"test", "mock"} }

func (m *MockMCPTool) HealthCheck() error { return nil }

func (m *MockMCPTool) Execute(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	m.executed = true
	return map[string]interface{}{
		"output":  "MCP executed with: " + params["input"].(string),
		"success": true,
	}, nil
}

func (m *MockMCPTool) GetCostProfile() protocol.CostProfile {
	return protocol.CostProfile{
		FinancialCost: 0.01,
		LatencyCost:   time.Millisecond * 200,
	}
}

func (m *MockMCPTool) GetParameters() []protocol.ToolParameter {
	return []protocol.ToolParameter{
		{
			Name:        "input",
			Type:        "string",
			Description: "Test input parameter",
			Required:    true,
		},
	}
}

func (m *MockMCPTool) GetReturns() []protocol.ToolParameter {
	return []protocol.ToolParameter{
		{
			Name:        "output",
			Type:        "string",
			Description: "Test output",
			Required:    true,
		},
	}
}

func TestGuildToMCPAdapter(t *testing.T) {
	// Create a mock Guild tool
	guildTool := &MockGuildTool{name: "test_guild_tool"}

	// Create adapter
	adapter := NewGuildToMCPAdapter(guildTool)

	// Test interface methods
	assert.Equal(t, "guild_test_guild_tool", adapter.ID())
	assert.Equal(t, "test_guild_tool", adapter.Name())
	assert.Equal(t, "Mock Guild tool for testing", adapter.Description())
	assert.Contains(t, adapter.Capabilities(), "test")

	// Test parameter conversion
	params := adapter.GetParameters()
	require.Len(t, params, 1)
	assert.Equal(t, "input", params[0].Name)
	assert.Equal(t, "string", params[0].Type)
	assert.True(t, params[0].Required)

	// Test execution
	ctx := context.Background()
	result, err := adapter.Execute(ctx, map[string]interface{}{"input": "hello"})
	require.NoError(t, err)
	assert.True(t, guildTool.executed)

	resultMap, ok := result.(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "Executed with: hello", resultMap["output"])
	assert.True(t, resultMap["success"].(bool))
}

func TestMCPToGuildAdapter(t *testing.T) {
	// Create a mock MCP tool
	mcpTool := &MockMCPTool{id: "mcp_test", name: "test_mcp_tool"}

	// Create adapter
	adapter := NewMCPToGuildAdapter(mcpTool)

	// Test interface methods
	assert.Equal(t, "test_mcp_tool", adapter.Name())
	assert.Equal(t, "Mock MCP tool for testing", adapter.Description())
	assert.Equal(t, "test", adapter.Category())
	assert.True(t, adapter.RequiresAuth()) // Because financial cost > 0

	// Test schema generation
	schema := adapter.Schema()
	props, ok := schema["properties"].(map[string]interface{})
	require.True(t, ok)
	require.Contains(t, props, "input")

	required, ok := schema["required"].([]string)
	require.True(t, ok)
	assert.Contains(t, required, "input")

	// Test execution
	ctx := context.Background()
	input := `{"input": "world"}`
	result, err := adapter.Execute(ctx, input)
	require.NoError(t, err)
	assert.True(t, mcpTool.executed)
	assert.Equal(t, "MCP executed with: world", result.Output)
	assert.True(t, result.Success)
}

func TestToolBridge(t *testing.T) {
	t.Skip("ToolBridge tests disabled - registry methods need to be implemented")

	// Create registries
	mcpRegistry := tools.NewMemoryRegistry()
	guildRegistry := registry.NewToolRegistry()

	// Create bridge
	bridge := NewToolBridge(mcpRegistry, guildRegistry)

	// Test registering a Guild tool
	guildTool := &MockGuildTool{name: "bridge_test_guild"}
	err := bridge.RegisterGuildTool(guildTool)
	require.NoError(t, err)

	// Test registering an MCP tool
	mcpToolOrig := &MockMCPTool{id: "bridge_mcp", name: "bridge_test_mcp"}
	err = bridge.RegisterMCPTool(mcpToolOrig)
	require.NoError(t, err)

	mcpTools := mcpRegistry.ListTools()
	assert.NotEmpty(t, mcpTools)
}

func TestCostMagnitudeCalculation(t *testing.T) {
	bridge := &ToolBridge{}

	tests := []struct {
		name     string
		profile  protocol.CostProfile
		expected int
	}{
		{
			name: "free tool",
			profile: protocol.CostProfile{
				FinancialCost: 0,
				LatencyCost:   time.Millisecond * 50,
			},
			expected: 0,
		},
		{
			name: "very low cost",
			profile: protocol.CostProfile{
				FinancialCost: 0.0005,
				LatencyCost:   time.Millisecond * 500,
			},
			expected: 1,
		},
		{
			name: "low cost",
			profile: protocol.CostProfile{
				FinancialCost: 0.005,
				LatencyCost:   time.Second * 2,
			},
			expected: 2,
		},
		{
			name: "medium cost",
			profile: protocol.CostProfile{
				FinancialCost: 0.05,
				LatencyCost:   time.Second * 10,
			},
			expected: 3,
		},
		{
			name: "high cost",
			profile: protocol.CostProfile{
				FinancialCost: 0.5,
				LatencyCost:   time.Second * 45,
			},
			expected: 5,
		},
		{
			name: "very high cost",
			profile: protocol.CostProfile{
				FinancialCost: 2.0,
				LatencyCost:   time.Minute * 2,
			},
			expected: 8,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := bridge.calculateCostMagnitude(tt.profile)
			assert.Equal(t, tt.expected, result)
		})
	}
}
