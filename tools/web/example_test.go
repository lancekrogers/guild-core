package web

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/guild-ventures/guild-core/pkg/providers/interfaces"
	"github.com/guild-ventures/guild-core/tools"
)

// ExampleWebSearchTool demonstrates basic usage of the WebSearch tool
func ExampleWebSearchTool() {
	// Create the tool
	tool := NewWebSearchTool()
	
	// Example input
	input := `{"query": "artificial intelligence", "max_results": 3}`
	
	// Execute the search
	ctx := context.Background()
	result, err := tool.Execute(ctx, input)
	
	if err != nil {
		// Handle error
		return
	}
	
	if result.Success {
		// Process the search results
		// result.Output contains JSON with search results
		println("Search completed successfully")
	}
}

// ExampleWebFetchTool demonstrates basic usage of the WebFetch tool
func ExampleWebFetchTool() {
	// Create a mock AI provider for this example
	mockProvider := &MockAIProvider{}
	mockProvider.On("ChatCompletion", mock.Anything, mock.Anything).Return(
		&interfaces.ChatResponse{
			Choices: []interfaces.ChatChoice{
				{Message: interfaces.ChatMessage{Content: "Example analysis"}},
			},
		}, nil)
	
	// Create the tool
	tool := NewWebFetchTool(mockProvider)
	
	// Example input
	input := `{"url": "https://example.com", "prompt": "Summarize this page"}`
	
	// Execute the fetch and analysis
	ctx := context.Background()
	result, err := tool.Execute(ctx, input)
	
	if err != nil {
		// Handle error
		return
	}
	
	if result.Success {
		// Process the analyzed content
		// result.Output contains JSON with content and analysis
		println("Web fetch and analysis completed successfully")
	}
}

// TestToolsImplementInterface verifies that both tools implement the Tool interface correctly
func TestToolsImplementInterface(t *testing.T) {
	mockProvider := &MockAIProvider{}
	
	// Test WebSearch tool
	webSearchTool := NewWebSearchTool()
	var _ tools.Tool = webSearchTool
	
	assert.Equal(t, "web_search", webSearchTool.Name())
	assert.Equal(t, "web", webSearchTool.Category())
	assert.False(t, webSearchTool.RequiresAuth())
	assert.NotEmpty(t, webSearchTool.Description())
	assert.NotNil(t, webSearchTool.Schema())
	assert.NotEmpty(t, webSearchTool.Examples())
	
	// Test WebFetch tool
	webFetchTool := NewWebFetchTool(mockProvider)
	var _ tools.Tool = webFetchTool
	
	assert.Equal(t, "web_fetch", webFetchTool.Name())
	assert.Equal(t, "web", webFetchTool.Category())
	assert.False(t, webFetchTool.RequiresAuth())
	assert.NotEmpty(t, webFetchTool.Description())
	assert.NotNil(t, webFetchTool.Schema())
	assert.NotEmpty(t, webFetchTool.Examples())
}

// TestRegistryFunctions verifies the registry helper functions work correctly
func TestRegistryFunctions(t *testing.T) {
	// Test tool listing
	toolNames := ListWebTools()
	assert.Len(t, toolNames, 2)
	assert.Contains(t, toolNames, "web_search")
	assert.Contains(t, toolNames, "web_fetch")
	
	// Test tool info
	toolsInfo := GetWebToolsInfo()
	assert.Len(t, toolsInfo, 2)
	
	webSearchInfo := toolsInfo["web_search"]
	assert.Equal(t, "web_search", webSearchInfo.Name)
	assert.Equal(t, "web", webSearchInfo.Category)
	assert.Equal(t, 1, webSearchInfo.CostLevel)
	assert.False(t, webSearchInfo.RequiresAuth)
	assert.False(t, webSearchInfo.RequiresAI)
	
	webFetchInfo := toolsInfo["web_fetch"]
	assert.Equal(t, "web_fetch", webFetchInfo.Name)
	assert.Equal(t, "web", webFetchInfo.Category)
	assert.Equal(t, 2, webFetchInfo.CostLevel)
	assert.False(t, webFetchInfo.RequiresAuth)
	assert.True(t, webFetchInfo.RequiresAI)
	
	// Test individual tool getters
	mockProvider := &MockAIProvider{}
	
	searchTool := GetWebSearchTool()
	assert.NotNil(t, searchTool)
	assert.Equal(t, "web_search", searchTool.Name())
	
	fetchTool := GetWebFetchTool(mockProvider)
	assert.NotNil(t, fetchTool)
	assert.Equal(t, "web_fetch", fetchTool.Name())
}

// TestToolRegistration demonstrates how to register the web tools
func TestToolRegistration(t *testing.T) {
	// Create a tool registry
	toolRegistry := tools.NewToolRegistry()
	
	// Create a mock AI provider
	mockProvider := &MockAIProvider{}
	
	// Register the web tools
	err := RegisterWebTools(toolRegistry, mockProvider)
	assert.NoError(t, err)
	
	// Verify tools are registered
	searchTool, exists := toolRegistry.GetTool("web_search")
	assert.True(t, exists)
	assert.NotNil(t, searchTool)
	
	fetchTool, exists := toolRegistry.GetTool("web_fetch")
	assert.True(t, exists)
	assert.NotNil(t, fetchTool)
	
	// Test that we can list all tools
	allTools := toolRegistry.ListTools()
	toolNames := make([]string, len(allTools))
	for i, tool := range allTools {
		toolNames[i] = tool.Name()
	}
	assert.Contains(t, toolNames, "web_search")
	assert.Contains(t, toolNames, "web_fetch")
}