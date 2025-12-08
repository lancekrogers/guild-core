// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package web

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/guild-framework/guild-core/pkg/registry"
	"github.com/guild-framework/guild-core/tools"
)

func TestRegisterWebTools(t *testing.T) {
	toolRegistry := tools.NewToolRegistry()
	mockProvider := &MockAIProvider{}

	err := RegisterWebTools(toolRegistry, mockProvider)
	assert.NoError(t, err)

	// Verify both tools are registered
	webSearchTool, exists := toolRegistry.GetTool("web_search")
	assert.True(t, exists)
	assert.NotNil(t, webSearchTool)
	assert.Equal(t, "web_search", webSearchTool.Name())

	webFetchTool, exists := toolRegistry.GetTool("web_fetch")
	assert.True(t, exists)
	assert.NotNil(t, webFetchTool)
	assert.Equal(t, "web_fetch", webFetchTool.Name())
}

func TestRegisterWebToolsWithRegistry(t *testing.T) {
	mockRegistry := &MockToolRegistry{}
	mockProvider := &MockAIProvider{}

	// Set up expectations for tool registration
	mockRegistry.On("RegisterToolWithCost",
		"web_search",
		mock.AnythingOfType("*web.WebSearchTool"),
		1,
		[]string{"web", "search", "information_gathering"}).Return(nil)

	mockRegistry.On("RegisterToolWithCost",
		"web_fetch",
		mock.AnythingOfType("*web.WebFetchTool"),
		2,
		[]string{"web", "fetch", "analysis", "content_extraction"}).Return(nil)

	err := RegisterWebToolsWithRegistry(mockRegistry, mockProvider)
	assert.NoError(t, err)

	mockRegistry.AssertExpectations(t)
}

func TestGetWebSearchTool(t *testing.T) {
	tool := GetWebSearchTool()
	assert.NotNil(t, tool)
	assert.Equal(t, "web_search", tool.Name())
	assert.Equal(t, "web", tool.Category())
}

func TestGetWebFetchTool(t *testing.T) {
	mockProvider := &MockAIProvider{}
	tool := GetWebFetchTool(mockProvider)
	assert.NotNil(t, tool)
	assert.Equal(t, "web_fetch", tool.Name())
	assert.Equal(t, "web", tool.Category())
	assert.Equal(t, mockProvider, tool.aiProvider)
}

func TestListWebTools(t *testing.T) {
	tools := ListWebTools()
	assert.Len(t, tools, 2)
	assert.Contains(t, tools, "web_search")
	assert.Contains(t, tools, "web_fetch")
}

func TestGetWebToolsInfo(t *testing.T) {
	info := GetWebToolsInfo()
	assert.Len(t, info, 2)

	// Check WebSearch tool info
	webSearchInfo, exists := info["web_search"]
	assert.True(t, exists)
	assert.Equal(t, "web_search", webSearchInfo.Name)
	assert.Equal(t, "web", webSearchInfo.Category)
	assert.Equal(t, 1, webSearchInfo.CostLevel)
	assert.False(t, webSearchInfo.RequiresAuth)
	assert.False(t, webSearchInfo.RequiresAI)
	assert.Contains(t, webSearchInfo.Capabilities, "web")
	assert.Contains(t, webSearchInfo.Capabilities, "search")

	// Check WebFetch tool info
	webFetchInfo, exists := info["web_fetch"]
	assert.True(t, exists)
	assert.Equal(t, "web_fetch", webFetchInfo.Name)
	assert.Equal(t, "web", webFetchInfo.Category)
	assert.Equal(t, 2, webFetchInfo.CostLevel)
	assert.False(t, webFetchInfo.RequiresAuth)
	assert.True(t, webFetchInfo.RequiresAI)
	assert.Contains(t, webFetchInfo.Capabilities, "web")
	assert.Contains(t, webFetchInfo.Capabilities, "fetch")
	assert.Contains(t, webFetchInfo.Capabilities, "analysis")
}

// MockToolRegistry implements the ToolRegistry interface for testing
type MockToolRegistry struct {
	mock.Mock
}

func (m *MockToolRegistry) RegisterTool(name string, tool registry.Tool) error {
	args := m.Called(name, tool)
	return args.Error(0)
}

func (m *MockToolRegistry) GetTool(name string) (registry.Tool, error) {
	args := m.Called(name)
	return args.Get(0).(registry.Tool), args.Error(1)
}

func (m *MockToolRegistry) ListTools() []string {
	args := m.Called()
	return args.Get(0).([]string)
}

func (m *MockToolRegistry) GetToolsByCapability(capability string) []registry.Tool {
	args := m.Called(capability)
	return args.Get(0).([]registry.Tool)
}

func (m *MockToolRegistry) HasTool(name string) bool {
	args := m.Called(name)
	return args.Bool(0)
}

func (m *MockToolRegistry) GetUnderlyingRegistry() *tools.ToolRegistry {
	args := m.Called()
	return args.Get(0).(*tools.ToolRegistry)
}

func (m *MockToolRegistry) RegisterToolWithLegacyCost(tool registry.Tool, costPerUse float64) error {
	args := m.Called(tool, costPerUse)
	return args.Error(0)
}

func (m *MockToolRegistry) GetToolCost(toolName string) float64 {
	args := m.Called(toolName)
	return args.Get(0).(float64)
}

func (m *MockToolRegistry) SetToolCost(toolName string, cost float64) {
	m.Called(toolName, cost)
}

func (m *MockToolRegistry) GetToolsByCost(maxCost int) []registry.ToolInfo {
	args := m.Called(maxCost)
	return args.Get(0).([]registry.ToolInfo)
}

func (m *MockToolRegistry) GetCheapestToolByCapability(capability string) (*registry.ToolInfo, error) {
	args := m.Called(capability)
	return args.Get(0).(*registry.ToolInfo), args.Error(1)
}

func (m *MockToolRegistry) RegisterToolWithCost(name string, tool registry.Tool, costMagnitude int, capabilities []string) error {
	args := m.Called(name, tool, costMagnitude, capabilities)
	return args.Error(0)
}

func (m *MockToolRegistry) GetToolInfo(name string) (*registry.ToolInfo, error) {
	args := m.Called(name)
	return args.Get(0).(*registry.ToolInfo), args.Error(1)
}

func (m *MockToolRegistry) ListToolsWithMetadata() []registry.ToolInfo {
	args := m.Called()
	return args.Get(0).([]registry.ToolInfo)
}

func (m *MockToolRegistry) SetToolAvailability(name string, available bool) error {
	args := m.Called(name, available)
	return args.Error(0)
}

func (m *MockToolRegistry) RegisterBasicTools() error {
	args := m.Called()
	return args.Error(0)
}
