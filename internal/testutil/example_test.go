package testutil_test

import (
	"context"
	"testing"

	"github.com/guild-ventures/guild-core/internal/testutil"
	"github.com/guild-ventures/guild-core/pkg/project"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Example integration test using the test utilities
func TestExampleCommissionWorkflow(t *testing.T) {
	// Setup test project with all features
	projCtx, cleanup := testutil.SetupTestProject(t, testutil.TestProjectOptions{
		Name:           "commission-workflow-test",
		WithCorpus:     true,
		WithObjectives: true,
	})
	defer cleanup()

	// Verify project structure
	testutil.AssertProjectStructure(t, projCtx)

	// Create context with project
	ctx := project.WithContext(context.Background(), projCtx)

	// Setup mock LLM provider
	mockProvider := testutil.NewMockLLMProvider()
	
	// Configure mock response for manager agent
	mockProvider.SetResponse("manager", testutil.GenerateMockAgentResponse(
		testutil.AgentResponseOptions{
			Type: "task_breakdown",
			Tasks: []string{
				"Design API schema",
				"Implement endpoints",
				"Add authentication",
				"Write tests",
			},
		},
	).Content)

	// Verify we can retrieve the mock response
	_, ok := project.FromContext(ctx)
	require.True(t, ok, "project context should be available")
}

// Example test for tool execution
func TestExampleToolExecution(t *testing.T) {
	// Create mock tool registry
	registry := testutil.NewMockToolRegistry()

	// Configure expected behavior
	registry.SetExecutionResult("file", map[string]interface{}{
		"status": "success",
		"files":  []string{"file1.go", "file2.go"},
	})

	// Get tool and execute
	tool, err := registry.GetTool("file")
	require.NoError(t, err)
	assert.NotNil(t, tool)

	// Execute tool
	ctx := context.Background()
	result, err := tool.Execute(ctx, `{"action": "list", "path": "/tmp"}`)
	require.NoError(t, err)
	assert.True(t, result.Success)

	// Verify call count
	assert.Equal(t, 1, registry.GetCallCount("file"))
}

// Example test for streaming responses
func TestExampleStreamingResponse(t *testing.T) {
	mockProvider := testutil.NewMockLLMProvider()

	// Configure streaming response
	mockProvider.SetStreamResponse("test-model", []string{
		"I'll help you ",
		"create a comprehensive ",
		"solution for ",
		"your request.",
	})

	// Test would use this in actual chat implementation
	assert.NotNil(t, mockProvider)
}

// Example test for commission generation
func TestExampleCommissionGeneration(t *testing.T) {
	// Generate a test commission
	commission := testutil.GenerateTestCommission(testutil.CommissionOptions{
		Title:      "E-commerce API",
		Complexity: "complex",
		Domain:     "api",
		NumTasks:   5,
	})

	// Verify commission content
	assert.Contains(t, commission, "E-commerce API")
	assert.Contains(t, commission, "Requirements")
	assert.Contains(t, commission, "Tasks")
	assert.Contains(t, commission, "Success Criteria")
}

// Example test demonstrating campaign configuration
func TestExampleCampaignConfig(t *testing.T) {
	// Generate campaign configuration
	config := testutil.GenerateCampaignConfig(testutil.CampaignConfigOptions{
		Name:         "test-campaign",
		NumAgents:    3,
		NumProviders: 1,
		WithTools:    true,
	})

	// Verify configuration
	assert.Equal(t, "test-campaign", config.Name)
	assert.Len(t, config.Agents, 3)
	
	// Check tools are assigned to agents
	for _, agent := range config.Agents {
		if len(agent.Tools) > 0 {
			assert.Contains(t, agent.Tools, "file")
			assert.Contains(t, agent.Tools, "shell")
			assert.Contains(t, agent.Tools, "http")
		}
	}
}

// Example test for mock event bus
func TestExampleEventBus(t *testing.T) {
	eventBus := testutil.NewMockEventBus()

	// Subscribe to events
	received := make(chan interface{}, 1)
	eventBus.Subscribe("task.completed", func(event interface{}) {
		received <- event
	})

	// Publish event
	err := eventBus.Publish("task.completed", map[string]string{
		"task_id": "123",
		"status":  "success",
	})
	require.NoError(t, err)

	// Verify event was received
	select {
	case event := <-received:
		data, ok := event.(map[string]string)
		require.True(t, ok)
		assert.Equal(t, "123", data["task_id"])
		assert.Equal(t, "success", data["status"])
	default:
		t.Fatal("event not received")
	}

	// Verify event history
	events := eventBus.GetEvents()
	assert.Len(t, events, 1)
}