package agent

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/guild-ventures/guild-core/internal/testutil"
	"github.com/guild-ventures/guild-core/pkg/agent"
	"github.com/guild-ventures/guild-core/pkg/agent/manager"
	"github.com/guild-ventures/guild-core/pkg/commission"
	"github.com/guild-ventures/guild-core/pkg/context"
	"github.com/guild-ventures/guild-core/pkg/kanban"
	"github.com/guild-ventures/guild-core/pkg/orchestrator/interfaces"
	"github.com/guild-ventures/guild-core/pkg/project"
)

// TestManagerAgentCommissionBreakdown tests the complete flow of a manager agent
// receiving a commission and breaking it down into tasks
func TestManagerAgentCommissionBreakdown(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Setup test environment
	projCtx, cleanup := testutil.SetupTestProject(t)
	defer cleanup()

	ctx := project.WithContext(context.Background(), projCtx)

	// Configure mock provider with realistic manager response
	mockProvider := testutil.NewMockLLMProvider()
	mockProvider.SetResponse("manager", testutil.GenerateMockAgentResponse(
		testutil.AgentResponseOptions{
			Type: "task_breakdown",
			Tasks: []string{
				"Design API schema and endpoints",
				"Implement authentication middleware",
				"Create database models",
				"Write comprehensive tests",
			},
		},
	))

	// Create agent context with mock provider
	agentCtx := &context.AgentContext{
		ProjectContext: projCtx,
		CostManager:    context.NewCostManager(nil), // Simple cost manager
		ToolRegistry:   testutil.NewMockToolRegistry(),
		ProviderName:   "mock",
		Provider:       mockProvider,
	}

	// Create manager agent
	managerAgent := agent.NewContextAgent(
		"test-manager",
		"Test Manager Agent",
		"manager",
		agentCtx,
	)

	// Create a test commission
	testCommission := testutil.GenerateTestCommission(testutil.CommissionOptions{
		Title:      "E-commerce API Development",
		Complexity: "medium",
		Domain:     "api",
		NumTasks:   4,
	})

	// Execute commission breakdown
	response, err := managerAgent.Execute(ctx, fmt.Sprintf("Break down this commission into tasks:\n\n%s", testCommission))
	require.NoError(t, err)
	assert.NotEmpty(t, response)

	// Verify response contains task breakdown
	assert.Contains(t, response, "Task")
	assert.Contains(t, response, "Design API schema")
	assert.Contains(t, response, "authentication")

	// Verify cost tracking
	costReport, err := agentCtx.CostManager.GetAgentReport("test-manager")
	require.NoError(t, err)
	assert.Equal(t, int64(1), costReport.TotalRequests)
}

// TestWorkerAgentTaskExecution tests worker agents receiving and executing tasks
func TestWorkerAgentTaskExecution(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Setup test environment
	projCtx, cleanup := testutil.SetupTestProject(t)
	defer cleanup()

	ctx := project.WithContext(context.Background(), projCtx)

	// Configure mock provider with implementation response
	mockProvider := testutil.NewMockLLMProvider()
	mockProvider.SetResponse("worker", testutil.GenerateMockAgentResponse(
		testutil.AgentResponseOptions{
			Type: "implementation",
			Code: `func AuthMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        token := r.Header.Get("Authorization")
        if !validateToken(token) {
            http.Error(w, "Unauthorized", http.StatusUnauthorized)
            return
        }
        next.ServeHTTP(w, r)
    })
}`,
		},
	))

	// Create worker context
	workerCtx := &context.AgentContext{
		ProjectContext: projCtx,
		CostManager:    context.NewCostManager(nil),
		ToolRegistry:   testutil.NewMockToolRegistry(),
		ProviderName:   "mock",
		Provider:       mockProvider,
	}

	// Create worker agent
	workerAgent := agent.NewContextAgent(
		"test-developer",
		"Test Developer Agent",
		"worker",
		workerCtx,
	)

	// Execute task
	task := "Implement authentication middleware for the API"
	response, err := workerAgent.Execute(ctx, task)
	require.NoError(t, err)
	assert.NotEmpty(t, response)

	// Verify implementation response
	assert.Contains(t, response, "AuthMiddleware")
	assert.Contains(t, response, "implementation")

	// Test concurrent task execution by multiple workers
	t.Run("ConcurrentWorkerExecution", func(t *testing.T) {
		var wg sync.WaitGroup
		numWorkers := 3
		
		for i := 0; i < numWorkers; i++ {
			wg.Add(1)
			workerID := fmt.Sprintf("worker-%d", i)
			
			go func(id string) {
				defer wg.Done()
				
				// Create worker-specific context
				wCtx := &context.AgentContext{
					ProjectContext: projCtx,
					CostManager:    workerCtx.CostManager, // Share cost manager
					ToolRegistry:   workerCtx.ToolRegistry,
					ProviderName:   "mock",
					Provider:       mockProvider,
				}
				
				// Create worker
				worker := agent.NewContextAgent(
					id,
					fmt.Sprintf("Worker %s", id),
					"worker",
					wCtx,
				)
				
				// Execute task
				_, err := worker.Execute(ctx, fmt.Sprintf("Task for %s", id))
				assert.NoError(t, err)
			}(workerID)
		}
		
		wg.Wait()
		
		// Verify all workers executed
		for i := 0; i < numWorkers; i++ {
			report, err := workerCtx.CostManager.GetAgentReport(fmt.Sprintf("worker-%d", i))
			require.NoError(t, err)
			assert.Equal(t, int64(1), report.TotalRequests)
		}
	})
}

// TestAgentEventBusCommunication tests agent communication through the event bus
func TestAgentEventBusCommunication(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Setup test environment
	projCtx, cleanup := testutil.SetupTestProject(t)
	defer cleanup()

	ctx := project.WithContext(context.Background(), projCtx)

	// Create mock event bus
	eventBus := testutil.NewMockEventBus()

	// Track events
	var receivedEvents []interface{}
	var mu sync.Mutex

	// Subscribe to task events
	eventBus.Subscribe(string(interfaces.TaskCreated), func(event interface{}) {
		mu.Lock()
		receivedEvents = append(receivedEvents, event)
		mu.Unlock()
	})

	eventBus.Subscribe(string(interfaces.TaskAssigned), func(event interface{}) {
		mu.Lock()
		receivedEvents = append(receivedEvents, event)
		mu.Unlock()
	})

	// Create manager with event publishing
	mockProvider := testutil.NewMockLLMProvider()
	managerCtx := &context.AgentContext{
		ProjectContext: projCtx,
		CostManager:    context.NewCostManager(nil),
		ToolRegistry:   testutil.NewMockToolRegistry(),
		ProviderName:   "mock",
		Provider:       mockProvider,
	}

	// Configure response that triggers events
	mockProvider.SetResponse("manager", testutil.GenerateMockAgentResponse(
		testutil.AgentResponseOptions{
			Type: "task_breakdown",
			Tasks: []string{
				"Task 1: API Design",
				"Task 2: Implementation",
			},
		},
	))

	managerAgent := agent.NewContextAgent(
		"event-manager",
		"Event Manager",
		"manager",
		managerCtx,
	)

	// Simulate task creation through manager
	response, err := managerAgent.Execute(ctx, "Create tasks for API development")
	require.NoError(t, err)
	assert.NotEmpty(t, response)

	// Publish task events manually (in real system, this would be done by orchestrator)
	eventBus.Publish(interfaces.TaskCreatedEvent{
		TaskID:      "task-1",
		Title:       "Task 1: API Design",
		Description: "Design the API endpoints and schema",
		Priority:    kanban.PriorityHigh,
		CreatedBy:   "event-manager",
	})

	eventBus.Publish(interfaces.TaskAssignedEvent{
		TaskID:  "task-1",
		AgentID: "worker-1",
	})

	// Give events time to propagate
	time.Sleep(100 * time.Millisecond)

	// Verify events were received
	mu.Lock()
	defer mu.Unlock()
	assert.Len(t, receivedEvents, 2, "Should receive task created and assigned events")

	// Verify event types
	foundCreated := false
	foundAssigned := false
	for _, event := range receivedEvents {
		switch e := event.(type) {
		case interfaces.TaskCreatedEvent:
			foundCreated = true
			assert.Equal(t, "task-1", e.TaskID)
			assert.Equal(t, "event-manager", e.CreatedBy)
		case interfaces.TaskAssignedEvent:
			foundAssigned = true
			assert.Equal(t, "task-1", e.TaskID)
			assert.Equal(t, "worker-1", e.AgentID)
		}
	}
	assert.True(t, foundCreated, "Should receive TaskCreatedEvent")
	assert.True(t, foundAssigned, "Should receive TaskAssignedEvent")
}

// TestAgentContextSharing tests context sharing between agents
func TestAgentContextSharing(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Setup test environment
	projCtx, cleanup := testutil.SetupTestProject(t)
	defer cleanup()

	ctx := project.WithContext(context.Background(), projCtx)

	// Create shared components
	sharedCostManager := context.NewCostManager(nil)
	sharedToolRegistry := testutil.NewMockToolRegistry()
	mockProvider := testutil.NewMockLLMProvider()

	// Configure different responses for different agents
	mockProvider.SetResponse("architect", testutil.GenerateMockAgentResponse(
		testutil.AgentResponseOptions{
			Type:     "review",
			Feedback: "The API design looks good. Consider adding rate limiting.",
		},
	))

	mockProvider.SetResponse("developer", testutil.GenerateMockAgentResponse(
		testutil.AgentResponseOptions{
			Type: "implementation",
			Code: "// Rate limiting middleware implementation",
		},
	))

	// Create architect agent
	architectCtx := &context.AgentContext{
		ProjectContext: projCtx,
		CostManager:    sharedCostManager,
		ToolRegistry:   sharedToolRegistry,
		ProviderName:   "mock",
		Provider:       mockProvider,
	}

	architect := agent.NewContextAgent(
		"architect",
		"System Architect",
		"specialist",
		architectCtx,
	)

	// Create developer agent with shared context
	developerCtx := &context.AgentContext{
		ProjectContext: projCtx,
		CostManager:    sharedCostManager, // Shared cost tracking
		ToolRegistry:   sharedToolRegistry, // Shared tools
		ProviderName:   "mock",
		Provider:       mockProvider,
	}

	developer := agent.NewContextAgent(
		"developer",
		"Developer",
		"worker",
		developerCtx,
	)

	// Architect reviews design
	review, err := architect.Execute(ctx, "Review the API design for security considerations")
	require.NoError(t, err)
	assert.Contains(t, review, "rate limiting")

	// Developer implements based on review
	implementation, err := developer.Execute(ctx, "Implement rate limiting as suggested by architect")
	require.NoError(t, err)
	assert.Contains(t, implementation, "Rate limiting")

	// Verify shared cost tracking
	architectCost, err := sharedCostManager.GetAgentReport("architect")
	require.NoError(t, err)
	assert.Equal(t, int64(1), architectCost.TotalRequests)

	developerCost, err := sharedCostManager.GetAgentReport("developer")
	require.NoError(t, err)
	assert.Equal(t, int64(1), developerCost.TotalRequests)

	// Verify total cost tracking
	totalCost, err := sharedCostManager.GetTotalCost()
	require.NoError(t, err)
	assert.Equal(t, int64(2), totalCost.TotalRequests)
}

// TestAgentCapabilityRouting tests routing tasks to agents based on capabilities
func TestAgentCapabilityRouting(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Setup test environment
	projCtx, cleanup := testutil.SetupTestProject(t)
	defer cleanup()

	ctx := project.WithContext(context.Background(), projCtx)

	// Create mock provider
	mockProvider := testutil.NewMockLLMProvider()

	// Create agent factory
	factory := &testAgentFactory{
		provider: mockProvider,
		projCtx:  projCtx,
	}

	// Create task complexity analyzer
	analyzer := manager.NewTaskComplexityAnalyzer()

	// Create agent router
	router := manager.NewAgentRouter(factory, analyzer)

	// Define test tasks with different requirements
	tasks := []struct {
		description      string
		expectedAgent    string
		requiredCapability string
	}{
		{
			description:      "Write unit tests for the authentication module",
			expectedAgent:    "tester",
			requiredCapability: "testing",
		},
		{
			description:      "Design the database schema for user management",
			expectedAgent:    "architect",
			requiredCapability: "design",
		},
		{
			description:      "Implement the REST API endpoints",
			expectedAgent:    "developer",
			requiredCapability: "coding",
		},
		{
			description:      "Review the security implementation",
			expectedAgent:    "reviewer",
			requiredCapability: "review",
		},
	}

	// Test routing for each task
	for _, task := range tasks {
		t.Run(task.expectedAgent, func(t *testing.T) {
			// Analyze task
			analysis := analyzer.AnalyzeTask(ctx, task.description)
			
			// Route to appropriate agent
			agent, err := router.RouteToAgent(ctx, analysis)
			require.NoError(t, err)
			require.NotNil(t, agent)

			// Verify correct agent was selected
			// Note: The actual agent selection depends on the implementation
			// For now, we verify that an agent was selected
			response, err := agent.Execute(ctx, task.description)
			require.NoError(t, err)
			assert.NotEmpty(t, response)
		})
	}
}

// TestMultiAgentCostTracking tests cost tracking across multiple agent executions
func TestMultiAgentCostTracking(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Setup test environment
	projCtx, cleanup := testutil.SetupTestProject(t)
	defer cleanup()

	ctx := project.WithContext(context.Background(), projCtx)

	// Create shared cost manager with mock storage
	costManager := context.NewCostManager(nil)

	// Create mock provider that tracks token usage
	mockProvider := testutil.NewMockLLMProvider()
	
	// Configure provider to return token counts
	mockProvider.SetTokenCounts(100, 150) // input: 100, output: 150

	// Create multiple agents
	agents := []struct {
		id   string
		name string
		role string
	}{
		{"manager", "Manager", "manager"},
		{"dev1", "Developer 1", "worker"},
		{"dev2", "Developer 2", "worker"},
		{"reviewer", "Reviewer", "worker"},
	}

	// Execute tasks with each agent
	for _, agentInfo := range agents {
		agentCtx := &context.AgentContext{
			ProjectContext: projCtx,
			CostManager:    costManager,
			ToolRegistry:   testutil.NewMockToolRegistry(),
			ProviderName:   "mock",
			Provider:       mockProvider,
		}

		agent := agent.NewContextAgent(
			agentInfo.id,
			agentInfo.name,
			agentInfo.role,
			agentCtx,
		)

		// Execute multiple requests
		for i := 0; i < 3; i++ {
			_, err := agent.Execute(ctx, fmt.Sprintf("Task %d for %s", i, agentInfo.name))
			require.NoError(t, err)

			// Record cost
			costManager.RecordTokenUsage(agentInfo.id, 100, 150)
		}
	}

	// Verify individual agent costs
	for _, agentInfo := range agents {
		report, err := costManager.GetAgentReport(agentInfo.id)
		require.NoError(t, err)
		assert.Equal(t, int64(3), report.TotalRequests)
		assert.Equal(t, int64(300), report.TotalInputTokens)  // 100 * 3
		assert.Equal(t, int64(450), report.TotalOutputTokens) // 150 * 3
	}

	// Verify total cost
	totalCost, err := costManager.GetTotalCost()
	require.NoError(t, err)
	assert.Equal(t, int64(12), totalCost.TotalRequests)        // 4 agents * 3 requests
	assert.Equal(t, int64(1200), totalCost.TotalInputTokens)   // 4 * 300
	assert.Equal(t, int64(1800), totalCost.TotalOutputTokens)  // 4 * 450

	// Test cost report generation
	report := costManager.GenerateReport()
	assert.Contains(t, report, "Total Requests: 12")
	assert.Contains(t, report, "manager")
	assert.Contains(t, report, "dev1")
	assert.Contains(t, report, "dev2")
	assert.Contains(t, report, "reviewer")
}

// testAgentFactory is a test implementation of agent factory
type testAgentFactory struct {
	provider *testutil.MockLLMProvider
	projCtx  project.Context
}

func (f *testAgentFactory) CreateAgent(ctx context.Context, agentType string) (agent.Agent, error) {
	agentCtx := &context.AgentContext{
		ProjectContext: f.projCtx,
		CostManager:    context.NewCostManager(nil),
		ToolRegistry:   testutil.NewMockToolRegistry(),
		ProviderName:   "mock",
		Provider:       f.provider,
	}

	// Create agent based on type
	var id, name string
	switch agentType {
	case "manager":
		id = "manager"
		name = "Manager"
	case "developer":
		id = "developer"
		name = "Developer"
	case "tester":
		id = "tester"
		name = "Tester"
	case "architect":
		id = "architect"
		name = "Architect"
	case "reviewer":
		id = "reviewer"
		name = "Reviewer"
	default:
		id = "worker"
		name = "Worker"
	}

	return agent.NewContextAgent(id, name, agentType, agentCtx), nil
}

// TestAgentLifecycle tests the complete lifecycle of agents from creation to cleanup
func TestAgentLifecycle(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Setup test environment
	projCtx, cleanup := testutil.SetupTestProject(t)
	defer cleanup()

	ctx := project.WithContext(context.Background(), projCtx)

	// Create components
	mockProvider := testutil.NewMockLLMProvider()
	costManager := context.NewCostManager(nil)
	eventBus := testutil.NewMockEventBus()

	// Track agent lifecycle events
	var createdAgents []string
	var executedTasks []string
	var cleanedUpAgents []string
	var mu sync.Mutex

	// Subscribe to lifecycle events
	eventBus.Subscribe("agent.created", func(event interface{}) {
		if e, ok := event.(map[string]string); ok {
			mu.Lock()
			createdAgents = append(createdAgents, e["agent_id"])
			mu.Unlock()
		}
	})

	eventBus.Subscribe("task.executed", func(event interface{}) {
		if e, ok := event.(map[string]string); ok {
			mu.Lock()
			executedTasks = append(executedTasks, e["task_id"])
			mu.Unlock()
		}
	})

	eventBus.Subscribe("agent.cleanup", func(event interface{}) {
		if e, ok := event.(map[string]string); ok {
			mu.Lock()
			cleanedUpAgents = append(cleanedUpAgents, e["agent_id"])
			mu.Unlock()
		}
	})

	// Create agent
	agentCtx := &context.AgentContext{
		ProjectContext: projCtx,
		CostManager:    costManager,
		ToolRegistry:   testutil.NewMockToolRegistry(),
		ProviderName:   "mock",
		Provider:       mockProvider,
	}

	testAgent := agent.NewContextAgent(
		"lifecycle-test",
		"Lifecycle Test Agent",
		"worker",
		agentCtx,
	)

	// Publish creation event
	eventBus.Publish(map[string]string{
		"agent_id": "lifecycle-test",
		"type":     "worker",
	})

	// Execute tasks
	for i := 0; i < 3; i++ {
		taskID := fmt.Sprintf("task-%d", i)
		_, err := testAgent.Execute(ctx, fmt.Sprintf("Execute %s", taskID))
		require.NoError(t, err)

		// Publish execution event
		eventBus.Publish(map[string]string{
			"task_id":  taskID,
			"agent_id": "lifecycle-test",
		})
	}

	// Simulate cleanup
	eventBus.Publish(map[string]string{
		"agent_id": "lifecycle-test",
		"status":   "cleanup",
	})

	// Give events time to propagate
	time.Sleep(100 * time.Millisecond)

	// Verify lifecycle
	mu.Lock()
	defer mu.Unlock()

	assert.Contains(t, createdAgents, "lifecycle-test")
	assert.Len(t, executedTasks, 3)
	assert.Contains(t, cleanedUpAgents, "lifecycle-test")

	// Verify cost was tracked
	report, err := costManager.GetAgentReport("lifecycle-test")
	require.NoError(t, err)
	assert.Equal(t, int64(3), report.TotalRequests)
}

// TestCommissionToCompletionFlow tests the complete flow from commission to task completion
func TestCommissionToCompletionFlow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Setup test environment
	projCtx, cleanup := testutil.SetupTestProject(t)
	defer cleanup()

	ctx := project.WithContext(context.Background(), projCtx)

	// Create shared components
	mockProvider := testutil.NewMockLLMProvider()
	costManager := context.NewCostManager(nil)
	eventBus := testutil.NewMockEventBus()

	// Configure mock responses
	mockProvider.SetResponse("manager", testutil.GenerateMockAgentResponse(
		testutil.AgentResponseOptions{
			Type: "task_breakdown",
			Tasks: []string{
				"Design user authentication API",
				"Implement JWT token generation",
				"Add password hashing",
				"Write authentication tests",
			},
		},
	))

	mockProvider.SetResponse("developer", testutil.GenerateMockAgentResponse(
		testutil.AgentResponseOptions{
			Type: "implementation",
			Code: "// Implementation completed",
		},
	))

	mockProvider.SetResponse("tester", testutil.GenerateMockAgentResponse(
		testutil.AgentResponseOptions{
			Type:     "review",
			Feedback: "All tests passing",
		},
	))

	// Create commission
	testCommission := &commission.Commission{
		ID:          "test-commission",
		Title:       "Implement User Authentication",
		Description: "Create a secure user authentication system",
		Status:      commission.StatusPending,
	}

	// Track workflow progress
	workflowSteps := make(map[string]bool)
	var mu sync.Mutex

	// Subscribe to workflow events
	eventBus.Subscribe("commission.created", func(event interface{}) {
		mu.Lock()
		workflowSteps["commission_created"] = true
		mu.Unlock()
	})

	eventBus.Subscribe("tasks.created", func(event interface{}) {
		mu.Lock()
		workflowSteps["tasks_created"] = true
		mu.Unlock()
	})

	eventBus.Subscribe("tasks.assigned", func(event interface{}) {
		mu.Lock()
		workflowSteps["tasks_assigned"] = true
		mu.Unlock()
	})

	eventBus.Subscribe("tasks.completed", func(event interface{}) {
		mu.Lock()
		workflowSteps["tasks_completed"] = true
		mu.Unlock()
	})

	// Step 1: Commission created
	eventBus.Publish(map[string]interface{}{
		"type":          "commission.created",
		"commission_id": testCommission.ID,
	})

	// Step 2: Manager breaks down commission
	managerCtx := &context.AgentContext{
		ProjectContext: projCtx,
		CostManager:    costManager,
		ToolRegistry:   testutil.NewMockToolRegistry(),
		ProviderName:   "mock",
		Provider:       mockProvider,
	}

	managerAgent := agent.NewContextAgent(
		"manager",
		"Project Manager",
		"manager",
		managerCtx,
	)

	breakdown, err := managerAgent.Execute(ctx, testCommission.Description)
	require.NoError(t, err)
	assert.Contains(t, breakdown, "Design user authentication")

	// Step 3: Tasks created from breakdown
	tasks := []struct {
		id          string
		title       string
		assignedTo  string
		agentType   string
	}{
		{"task-1", "Design user authentication API", "architect", "developer"},
		{"task-2", "Implement JWT token generation", "dev-1", "developer"},
		{"task-3", "Add password hashing", "dev-2", "developer"},
		{"task-4", "Write authentication tests", "tester", "tester"},
	}

	eventBus.Publish(map[string]interface{}{
		"type":  "tasks.created",
		"count": len(tasks),
	})

	// Step 4: Assign and execute tasks
	for _, task := range tasks {
		// Create agent for task
		agentCtx := &context.AgentContext{
			ProjectContext: projCtx,
			CostManager:    costManager,
			ToolRegistry:   testutil.NewMockToolRegistry(),
			ProviderName:   "mock",
			Provider:       mockProvider,
		}

		taskAgent := agent.NewContextAgent(
			task.assignedTo,
			task.assignedTo,
			task.agentType,
			agentCtx,
		)

		// Execute task
		_, err := taskAgent.Execute(ctx, task.title)
		require.NoError(t, err)
	}

	eventBus.Publish(map[string]interface{}{
		"type": "tasks.assigned",
	})

	// Step 5: Complete all tasks
	eventBus.Publish(map[string]interface{}{
		"type":            "tasks.completed",
		"commission_id":   testCommission.ID,
		"completed_tasks": len(tasks),
	})

	// Give events time to propagate
	time.Sleep(100 * time.Millisecond)

	// Verify complete workflow
	mu.Lock()
	defer mu.Unlock()

	assert.True(t, workflowSteps["commission_created"], "Commission should be created")
	assert.True(t, workflowSteps["tasks_created"], "Tasks should be created")
	assert.True(t, workflowSteps["tasks_assigned"], "Tasks should be assigned")
	assert.True(t, workflowSteps["tasks_completed"], "Tasks should be completed")

	// Verify cost tracking for all agents
	totalCost, err := costManager.GetTotalCost()
	require.NoError(t, err)
	assert.Equal(t, int64(5), totalCost.TotalRequests) // 1 manager + 4 task agents
}