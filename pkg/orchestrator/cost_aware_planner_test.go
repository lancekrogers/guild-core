package orchestrator

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/guild-ventures/guild-core/pkg/config"
	"github.com/guild-ventures/guild-core/pkg/kanban"
	"github.com/guild-ventures/guild-core/pkg/objective"
	"github.com/guild-ventures/guild-core/pkg/registry"
)

func TestCostAwareTaskPlanner(t *testing.T) {
	// Setup test environment
	ctx := context.Background()
	
	// Create mock registry with test agents
	componentRegistry := createTestRegistry()
	
	// Create mock kanban board
	kanbanBoard := &MockKanbanBoard{
		tasks: make(map[string]*kanban.Task),
	}
	
	// Create mock manager agent
	managerAgent := &MockManagerAgent{
		responses: map[string]string{
			"planning": `TASK-001: Implement user authentication
Description: Create login and registration system
Capabilities: coding, security
Dependencies: none
Complexity: medium
PreferredCost: 2
---
TASK-002: Setup database
Description: Configure database tables and connections
Capabilities: database, configuration
Dependencies: none
Complexity: low
PreferredCost: 1
---
TASK-003: Create admin dashboard
Description: Build administrative interface
Capabilities: frontend, ui-design
Dependencies: TASK-001, TASK-002
Complexity: high
PreferredCost: 3`,
		},
	}
	
	// Create cost-aware planner
	planner := NewCostAwareTaskPlanner(managerAgent, kanbanBoard, componentRegistry, 5)
	
	t.Run("TestPlanTasks", func(t *testing.T) {
		// Create test objective
		obj := &objective.Objective{
			Title:       "Build User Management System",
			Description: "Complete user authentication and management system",
		}
		
		// Create test guild config
		guild := createTestGuildConfig()
		
		// Plan tasks
		tasks, err := planner.PlanTasks(ctx, obj, guild)
		require.NoError(t, err)
		assert.Len(t, tasks, 3)
		
		// Verify tasks have cost estimates
		for _, task := range tasks {
			assert.Contains(t, task.Metadata, "estimated_cost")
			assert.Contains(t, task.Metadata, "cheapest_agent")
		}
		
		// Verify first task
		assert.Equal(t, "TASK-001", tasks[0].ID)
		assert.Equal(t, "Implement user authentication", tasks[0].Title)
		assert.Equal(t, "coding, security", tasks[0].Metadata["capabilities"])
		assert.Equal(t, "medium", tasks[0].Metadata["complexity"])
	})
	
	t.Run("TestAssignTasksMinimizeCost", func(t *testing.T) {
		// Create test tasks
		tasks := createTestTasks()
		guild := createTestGuildConfig()
		
		options := AssignmentOptions{
			MaxCostMagnitude: 5,
			Strategy:         StrategyMinimizeCost,
			BalanceWorkload:  false,
		}
		
		// Assign tasks
		summary, err := planner.AssignTasksWithOptions(ctx, tasks, guild, options)
		require.NoError(t, err)
		
		// Verify assignments prioritize cost
		assert.Equal(t, 3, summary.TotalTasks)
		assert.True(t, summary.TotalCost <= 15) // Max 5 per task
		
		// Verify cheapest agents are selected
		for _, assignment := range summary.Assignments {
			assert.LessOrEqual(t, assignment.TotalCost, 5)
			assert.NotEmpty(t, assignment.Reason)
		}
	})
	
	t.Run("TestAssignTasksBalanced", func(t *testing.T) {
		tasks := createTestTasks()
		guild := createTestGuildConfig()
		
		options := AssignmentOptions{
			MaxCostMagnitude: 8,
			Strategy:         StrategyBalanced,
			BalanceWorkload:  true,
		}
		
		summary, err := planner.AssignTasksWithOptions(ctx, tasks, guild, options)
		require.NoError(t, err)
		
		// Verify workload distribution
		maxWorkload := 0
		minWorkload := 100
		for _, workload := range summary.AgentWorkloads {
			if workload > maxWorkload {
				maxWorkload = workload
			}
			if workload < minWorkload {
				minWorkload = workload
			}
		}
		
		// Workload should be reasonably balanced
		assert.LessOrEqual(t, maxWorkload-minWorkload, 2)
	})
	
	t.Run("TestAssignTasksCapabilityFirst", func(t *testing.T) {
		tasks := createTestTasks()
		guild := createTestGuildConfig()
		
		options := AssignmentOptions{
			MaxCostMagnitude: 3,
			Strategy:         StrategyCapabilityFirst,
			BalanceWorkload:  false,
		}
		
		summary, err := planner.AssignTasksWithOptions(ctx, tasks, guild, options)
		require.NoError(t, err)
		
		// Verify all assignments have proper capabilities
		for _, assignment := range summary.Assignments {
			assert.NotEmpty(t, assignment.AgentInfo.Capabilities)
		}
	})
	
	t.Run("TestBudgetConstraints", func(t *testing.T) {
		tasks := createTestTasks()
		guild := createTestGuildConfig()
		
		// Very low budget
		options := AssignmentOptions{
			MaxCostMagnitude: 1,
			Strategy:         StrategyMinimizeCost,
			BalanceWorkload:  false,
		}
		
		summary, err := planner.AssignTasksWithOptions(ctx, tasks, guild, options)
		
		// Should either succeed with low-cost agents or fail gracefully
		if err == nil {
			// If successful, all assignments should be within budget
			for _, assignment := range summary.Assignments {
				assert.LessOrEqual(t, assignment.TotalCost, 1)
			}
		} else {
			// Should have meaningful error message
			assert.Contains(t, err.Error(), "budget")
		}
	})
	
	t.Run("TestToolSelection", func(t *testing.T) {
		tasks := createTestTasks()
		guild := createTestGuildConfig()
		
		options := AssignmentOptions{
			MaxCostMagnitude: 8,
			Strategy:         StrategyBalanced,
			RequiredTools: map[string][]string{
				"TASK-001": {"file_operations", "execution"},
				"TASK-002": {"database", "configuration"},
			},
		}
		
		summary, err := planner.AssignTasksWithOptions(ctx, tasks, guild, options)
		require.NoError(t, err)
		
		// Verify tools are selected for tasks
		for _, assignment := range summary.Assignments {
			if len(assignment.Tools) > 0 {
				assert.GreaterOrEqual(t, assignment.TotalCost, assignment.AgentInfo.CostMagnitude)
				// Should be equal or greater because tools add cost
				expectedMinCost := assignment.AgentInfo.CostMagnitude
				assert.GreaterOrEqual(t, assignment.TotalCost, expectedMinCost)
			}
		}
	})
	
	t.Run("TestAssignmentSummary", func(t *testing.T) {
		tasks := createTestTasks()
		guild := createTestGuildConfig()
		
		options := AssignmentOptions{
			MaxCostMagnitude: 5,
			Strategy:         StrategyBalanced,
			BalanceWorkload:  true,
		}
		
		summary, err := planner.AssignTasksWithOptions(ctx, tasks, guild, options)
		require.NoError(t, err)
		
		// Verify summary statistics
		assert.Equal(t, len(tasks), summary.TotalTasks)
		assert.Greater(t, summary.TotalCost, 0)
		assert.Greater(t, summary.AverageCost, 0.0)
		assert.GreaterOrEqual(t, summary.BudgetUtilized, 0.0)
		assert.LessOrEqual(t, summary.BudgetUtilized, 100.0)
		
		// Verify cost breakdown
		assert.NotEmpty(t, summary.CostBreakdown)
		
		// Verify agent workloads
		assert.NotEmpty(t, summary.AgentWorkloads)
	})
}

// Test helper functions

func createTestRegistry() registry.ComponentRegistry {
	mockRegistry := &MockComponentRegistry{
		agents: []registry.AgentInfo{
			{
				ID:            "cheap-coder",
				Name:          "Cheap Coder",
				Type:          "worker",
				Capabilities:  []string{"coding", "testing"},
				CostMagnitude: 1,
				Available:     true,
			},
			{
				ID:            "security-expert",
				Name:          "Security Expert",
				Type:          "specialist",
				Capabilities:  []string{"security", "coding"},
				CostMagnitude: 3,
				Available:     true,
			},
			{
				ID:            "ui-designer",
				Name:          "UI Designer",
				Type:          "specialist",
				Capabilities:  []string{"frontend", "ui-design"},
				CostMagnitude: 2,
				Available:     true,
			},
			{
				ID:            "database-admin",
				Name:          "Database Admin",
				Type:          "specialist",
				Capabilities:  []string{"database", "configuration"},
				CostMagnitude: 2,
				Available:     true,
			},
			{
				ID:            "senior-architect",
				Name:          "Senior Architect",
				Type:          "manager",
				Capabilities:  []string{"architecture", "planning", "coding"},
				CostMagnitude: 5,
				Available:     true,
			},
		},
		tools: []registry.ToolInfo{
			{
				Name:          "file_operations",
				Capabilities:  []string{"file_operations"},
				CostMagnitude: 0,
				Available:     true,
			},
			{
				Name:          "execution",
				Capabilities:  []string{"execution"},
				CostMagnitude: 0,
				Available:     true,
			},
			{
				Name:          "database",
				Capabilities:  []string{"database"},
				CostMagnitude: 1,
				Available:     true,
			},
		},
	}
	
	return mockRegistry
}

func createTestGuildConfig() *config.GuildConfig {
	return &config.GuildConfig{
		Name: "Test Guild",
		Agents: []config.AgentConfig{
			{
				ID:           "cheap-coder",
				Name:         "Cheap Coder",
				Type:         "worker",
				Capabilities: []string{"coding", "testing"},
			},
			{
				ID:           "security-expert",
				Name:         "Security Expert",
				Type:         "specialist",
				Capabilities: []string{"security", "coding"},
			},
		},
	}
}

func createTestTasks() []*kanban.Task {
	return []*kanban.Task{
		{
			ID:          "TASK-001",
			Title:       "Implement authentication",
			Description: "Create user login system",
			Status:      kanban.StatusTodo,
			Metadata: map[string]string{
				"capabilities": "coding, security",
				"complexity":   "medium",
			},
		},
		{
			ID:          "TASK-002",
			Title:       "Setup database",
			Description: "Configure database connections",
			Status:      kanban.StatusTodo,
			Metadata: map[string]string{
				"capabilities": "database, configuration",
				"complexity":   "low",
			},
		},
		{
			ID:          "TASK-003",
			Title:       "Create UI",
			Description: "Build user interface",
			Status:      kanban.StatusTodo,
			Metadata: map[string]string{
				"capabilities": "frontend, ui-design",
				"complexity":   "high",
			},
		},
	}
}

// Mock implementations

type MockComponentRegistry struct {
	agents []registry.AgentInfo
	tools  []registry.ToolInfo
}

func (m *MockComponentRegistry) GetAgentsByCost(maxCost int) []registry.AgentInfo {
	var result []registry.AgentInfo
	for _, agent := range m.agents {
		if agent.CostMagnitude <= maxCost {
			result = append(result, agent)
		}
	}
	return result
}

func (m *MockComponentRegistry) GetCheapestAgentByCapability(capability string) (*registry.AgentInfo, error) {
	var cheapest *registry.AgentInfo
	minCost := 999
	
	for _, agent := range m.agents {
		for _, cap := range agent.Capabilities {
			if cap == capability && agent.CostMagnitude < minCost {
				minCost = agent.CostMagnitude
				agentCopy := agent
				cheapest = &agentCopy
				break
			}
		}
	}
	
	if cheapest == nil {
		return nil, registry.ErrComponentNotFound
	}
	return cheapest, nil
}

func (m *MockComponentRegistry) GetAgentsByCapability(capability string) []registry.AgentInfo {
	var result []registry.AgentInfo
	for _, agent := range m.agents {
		for _, cap := range agent.Capabilities {
			if cap == capability {
				result = append(result, agent)
				break
			}
		}
	}
	return result
}

func (m *MockComponentRegistry) GetToolsByCost(maxCost int) []registry.ToolInfo {
	var result []registry.ToolInfo
	for _, tool := range m.tools {
		if tool.CostMagnitude <= maxCost {
			result = append(result, tool)
		}
	}
	return result
}

func (m *MockComponentRegistry) GetCheapestToolByCapability(capability string) (*registry.ToolInfo, error) {
	var cheapest *registry.ToolInfo
	minCost := 999
	
	for _, tool := range m.tools {
		for _, cap := range tool.Capabilities {
			if cap == capability && tool.CostMagnitude < minCost {
				minCost = tool.CostMagnitude
				toolCopy := tool
				cheapest = &toolCopy
				break
			}
		}
	}
	
	if cheapest == nil {
		return nil, registry.ErrComponentNotFound
	}
	return cheapest, nil
}

// Stub methods for ComponentRegistry interface
func (m *MockComponentRegistry) Agents() registry.AgentRegistry { return nil }
func (m *MockComponentRegistry) Tools() registry.ToolRegistry { return nil }
func (m *MockComponentRegistry) Providers() registry.ProviderRegistry { return nil }
func (m *MockComponentRegistry) Memory() registry.MemoryRegistry { return nil }
func (m *MockComponentRegistry) Project() registry.ProjectRegistry { return nil }
func (m *MockComponentRegistry) Prompts() *registry.PromptRegistry { return nil }
func (m *MockComponentRegistry) Initialize(ctx context.Context, config registry.Config) error { return nil }
func (m *MockComponentRegistry) Shutdown(ctx context.Context) error { return nil }

type MockKanbanBoard struct {
	tasks   map[string]*kanban.Task
	counter int
}

func (m *MockKanbanBoard) CreateTask(ctx context.Context, title, description string) (*kanban.Task, error) {
	m.counter++
	task := &kanban.Task{
		ID:          fmt.Sprintf("TASK-%03d", m.counter),
		Title:       title,
		Description: description,
		Status:      kanban.StatusTodo,
		Metadata:    make(map[string]string),
	}
	m.tasks[task.ID] = task
	return task, nil
}

func (m *MockKanbanBoard) UpdateTask(ctx context.Context, task *kanban.Task) error {
	m.tasks[task.ID] = task
	return nil
}

func (m *MockKanbanBoard) GetTask(ctx context.Context, taskID string) (*kanban.Task, error) {
	if task, exists := m.tasks[taskID]; exists {
		return task, nil
	}
	return nil, fmt.Errorf("task not found")
}

func (m *MockKanbanBoard) ListTasksByStatus(ctx context.Context, boardID string, status kanban.TaskStatus) ([]*kanban.Task, error) {
	var tasks []*kanban.Task
	for _, task := range m.tasks {
		if task.Status == status {
			tasks = append(tasks, task)
		}
	}
	return tasks, nil
}

func (m *MockKanbanBoard) UpdateTaskStatus(ctx context.Context, taskID, status, assignee, comment string) error {
	if task, exists := m.tasks[taskID]; exists {
		task.Status = kanban.TaskStatus(status)
		task.AssignedTo = assignee
		if task.Metadata == nil {
			task.Metadata = make(map[string]string)
		}
		task.Metadata["comment"] = comment
		return nil
	}
	return fmt.Errorf("task not found")
}

type MockManagerAgent struct {
	responses map[string]string
}

func (m *MockManagerAgent) Execute(ctx context.Context, request string) (string, error) {
	// Simple response mapping based on request content
	if strings.Contains(request, "decompose") || strings.Contains(request, "break down") {
		return m.responses["planning"], nil
	}
	if strings.Contains(request, "assign") || strings.Contains(request, "assignment") {
		return "TASK-001: cheap-coder\nTASK-002: database-admin\nTASK-003: ui-designer", nil
	}
	return "Mock response", nil
}

func (m *MockManagerAgent) GetID() string { return "mock-manager" }
func (m *MockManagerAgent) GetName() string { return "Mock Manager" }