package main

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/guild-ventures/guild-core/pkg/config"
	"github.com/guild-ventures/guild-core/pkg/gerror"
	"github.com/guild-ventures/guild-core/pkg/kanban"
	"github.com/guild-ventures/guild-core/pkg/commission"
	"github.com/guild-ventures/guild-core/pkg/orchestrator"
	"github.com/guild-ventures/guild-core/pkg/registry"
	"github.com/guild-ventures/guild-core/tools"
)

// orchestratorCmd demonstrates the cost-aware orchestrator in action
var orchestratorCmd = &cobra.Command{
	Use:   "orchestrator-demo",
	Short: "Demonstrate cost-aware task planning and assignment",
	Long: `Demonstrates the Guild's intelligent orchestrator that uses cost optimization
for task planning and agent assignment.

The orchestrator:
1. Decomposes objectives into tasks using a manager agent
2. Assigns tasks to the most cost-effective agents
3. Selects optimal tools based on cost considerations
4. Provides detailed cost analysis and workload balancing

Examples:
  guild orchestrator-demo --objective "Build user auth system" --max-cost 5
  guild orchestrator-demo --strategy balanced --show-alternatives
  guild orchestrator-demo --strategy minimize-cost --max-cost 3
  guild orchestrator-demo --list-strategies`,
	RunE: runOrchestratorDemo,
}

var (
	orchestratorObjectiveText      string
	orchestratorMaxCostFilter      int
	orchestratorAssignmentStrategy string
	orchestratorShowAlternatives   bool
	orchestratorShowWorkload       bool
	orchestratorListStrategies     bool
	orchestratorOutputJSON         bool
)

func init() {
	orchestratorCmd.Flags().StringVar(&orchestratorObjectiveText, "objective", "Build a web application with user authentication", "Objective to plan and assign")
	orchestratorCmd.Flags().IntVar(&orchestratorMaxCostFilter, "max-cost", 8, "Maximum cost magnitude filter for agents (0-8)")
	orchestratorCmd.Flags().StringVar(&orchestratorAssignmentStrategy, "strategy", "balanced", "Assignment strategy: minimize-cost, balanced, capability-first")
	orchestratorCmd.Flags().BoolVar(&orchestratorShowAlternatives, "show-alternatives", false, "Show alternative agent assignments")
	orchestratorCmd.Flags().BoolVar(&orchestratorShowWorkload, "show-workload", true, "Show agent workload distribution")
	orchestratorCmd.Flags().BoolVar(&orchestratorListStrategies, "list-strategies", false, "List available assignment strategies")
	orchestratorCmd.Flags().BoolVar(&orchestratorOutputJSON, "json", false, "Output results in JSON format")
}

func runOrchestratorDemo(cmd *cobra.Command, args []string) error {
	if orchestratorListStrategies {
		return showAssignmentStrategies()
	}

	ctx := context.Background()

	// Create and initialize registry
	componentRegistry := registry.NewComponentRegistry()
	config := registry.Config{
		Agents: registry.AgentConfigYaml{DefaultType: "worker"},
		Tools:  registry.ToolConfig{EnabledTools: []string{"shell", "git", "http_client"}},
	}

	if err := componentRegistry.Initialize(ctx, config); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to initialize registry").
			WithComponent("cli").
			WithOperation("runOrchestratorDemo")
	}

	// Register demo agents and tools
	if err := registerOrchestratorDemoAgents(componentRegistry); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to register demo agents").
			WithComponent("cli").WithOperation("runOrchestratorDemo")
	}

	if err := registerOrchestratorDemoTools(componentRegistry); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to register demo tools").
			WithComponent("cli").WithOperation("runOrchestratorDemo")
	}

	// Create mock kanban board
	kanbanBoard := &OrchestratorMockKanbanBoard{
		tasks: make(map[string]*kanban.Task),
	}

	// Create mock manager agent
	managerAgent := &OrchestratorMockManagerAgent{}

	// Create cost-aware planner
	planner := orchestrator.DefaultCostAwareTaskPlannerFactory(
		managerAgent,
		kanbanBoard,
		componentRegistry,
		orchestratorMaxCostFilter,
	)

	// Create objective
	obj := &commission.Commission{
		Title:       "Development Project",
		Description: orchestratorObjectiveText,
	}

	// Create guild config
	guild := createOrchestratorGuildConfig()

	// Plan tasks
	fmt.Printf("🏰 Guild Orchestrator - Cost-Aware Task Planning\n")
	fmt.Printf("═══════════════════════════════════════════════\n\n")

	fmt.Printf("📋 Objective: %s\n", orchestratorObjectiveText)
	fmt.Printf("🔍 Max Cost Filter: %d (excluding agents above this cost)\n", orchestratorMaxCostFilter)
	fmt.Printf("🎯 Strategy: %s\n\n", orchestratorAssignmentStrategy)

	// Plan tasks using the orchestrator
	tasks, err := planner.PlanTasks(ctx, obj, guild)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to plan tasks").
			WithComponent("cli").WithOperation("runOrchestratorDemo").WithDetails("objective", orchestratorObjectiveText)
	}

	fmt.Printf("✅ Planned %d tasks:\n", len(tasks))
	for i, task := range tasks {
		estimated := task.Metadata["estimated_cost"]
		cheapest := task.Metadata["cheapest_agent"]
		fmt.Printf("   %d. %s (Est. cost: %s, Cheapest: %s)\n",
			i+1, task.Title, estimated, cheapest)
	}
	fmt.Printf("\n")

	// Assign tasks with selected strategy
	strategyEnum := parseAssignmentStrategy(orchestratorAssignmentStrategy)

	// Apply cost filter to exclude expensive agents if requested
	maxCostPerTask := min(8, orchestratorMaxCostFilter) // Cap at max Fibonacci value

	options := orchestrator.AssignmentOptions{
		MaxCostMagnitude: maxCostPerTask,
		Strategy:         strategyEnum,
		BalanceWorkload:  orchestratorAssignmentStrategy == "balanced",
		RequiredTools: map[string][]string{
			"TASK-001": {"file_operations", "execution"},
			"TASK-002": {"database", "configuration"},
			"TASK-003": {"file_operations", "network"},
		},
	}

	summary, err := planner.AssignTasksWithOptions(ctx, tasks, guild, options)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to assign tasks").
			WithComponent("cli").WithOperation("runOrchestratorDemo").WithDetails("strategy", orchestratorAssignmentStrategy)
	}

	// Display results
	if orchestratorOutputJSON {
		return outputAssignmentJSON(summary)
	}

	return displayAssignmentResults(summary, orchestratorShowAlternatives, orchestratorShowWorkload)
}

func parseAssignmentStrategy(strategy string) orchestrator.CostAssignmentStrategy {
	switch strategy {
	case "minimize-cost":
		return orchestrator.StrategyMinimizeCost
	case "capability-first":
		return orchestrator.StrategyCapabilityFirst
	default:
		return orchestrator.StrategyBalanced
	}
}

func displayAssignmentResults(summary *orchestrator.AssignmentSummary, showAlternatives, showWorkload bool) error {
	fmt.Printf("🎯 Task Assignment Results\n")
	fmt.Printf("═══════════════════════════\n\n")

	// Assignment details
	for _, assignment := range summary.Assignments {
		costIcon := getOrchestratorCostIcon(assignment.TotalCost)
		fmt.Printf("%s Task: %s\n", costIcon, assignment.TaskID)
		fmt.Printf("   Agent: %s (%s)\n", assignment.AgentInfo.Name, assignment.AgentInfo.Type)
		fmt.Printf("   Cost: %d | Reason: %s\n", assignment.TotalCost, assignment.Reason)

		if len(assignment.Tools) > 0 {
			fmt.Printf("   Tools: ")
			for i, tool := range assignment.Tools {
				if i > 0 { fmt.Printf(", ") }
				fmt.Printf("%s(%d)", tool.Name, tool.CostMagnitude)
			}
			fmt.Printf("\n")
		}

		if orchestratorShowAlternatives && len(assignment.Alternatives) > 0 {
			fmt.Printf("   Alternatives: ")
			for i, alt := range assignment.Alternatives {
				if i > 0 { fmt.Printf(", ") }
				fmt.Printf("%s(%d)", alt.Name, alt.CostMagnitude)
			}
			fmt.Printf("\n")
		}
		fmt.Printf("\n")
	}

	// Summary statistics
	fmt.Printf("📊 Summary Statistics\n")
	fmt.Printf("════════════════════\n")
	fmt.Printf("Total Tasks: %d\n", summary.TotalTasks)
	fmt.Printf("Total Cost: %d\n", summary.TotalCost)
	fmt.Printf("Average Cost: %.1f\n", summary.AverageCost)
	// Removed confusing budget utilization metric
	fmt.Printf("\n")

	// Cost breakdown
	fmt.Printf("💰 Cost Breakdown by Agent Type\n")
	fmt.Printf("═══════════════════════════════\n")
	for agentType, cost := range summary.CostBreakdown {
		percentage := float64(cost) / float64(summary.TotalCost) * 100
		fmt.Printf("%s: %d (%.1f%%)\n", agentType, cost, percentage)
	}
	fmt.Printf("\n")

	// Workload distribution
	if orchestratorShowWorkload {
		fmt.Printf("👥 Agent Workload Distribution\n")
		fmt.Printf("══════════════════════════════\n")
		for agentID, workload := range summary.AgentWorkloads {
			fmt.Printf("%s: %d tasks\n", agentID, workload)
		}
		fmt.Printf("\n")
	}

	// Cost efficiency analysis based on average cost
	if summary.AverageCost <= 2 {
		fmt.Printf("💡 Cost Profile: ECONOMICAL - Using mostly low-cost agents\n")
	} else if summary.AverageCost <= 4 {
		fmt.Printf("💡 Cost Profile: BALANCED - Good mix of agent costs\n")
	} else {
		fmt.Printf("💡 Cost Profile: PREMIUM - Using high-cost agents for quality\n")
	}

	return nil
}

func outputAssignmentJSON(summary *orchestrator.AssignmentSummary) error {
	data, err := json.MarshalIndent(summary, "", "  ")
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to marshal summary to JSON").
			WithComponent("cli").WithOperation("outputAssignmentJSON")
	}

	fmt.Println(string(data))
	return nil
}

func showAssignmentStrategies() error {
	fmt.Printf("🎯 Available Assignment Strategies\n")
	fmt.Printf("═════════════════════════════════\n\n")

	strategies := []struct {
		name        string
		description string
		use_case    string
	}{
		{
			name:        "minimize-cost",
			description: "Always selects the cheapest capable agent for each task",
			use_case:    "Cost-sensitive projects, development/testing phases",
		},
		{
			name:        "balanced",
			description: "Balances cost optimization with workload distribution",
			use_case:    "Production environments, team coordination",
		},
		{
			name:        "capability-first",
			description: "Prioritizes best capability match, then optimizes cost",
			use_case:    "Complex tasks, quality-critical projects",
		},
	}

	for _, strategy := range strategies {
		fmt.Printf("🔹 **%s**\n", strategy.name)
		fmt.Printf("   Description: %s\n", strategy.description)
		fmt.Printf("   Best for: %s\n\n", strategy.use_case)
	}

	fmt.Printf("💡 Usage: --strategy [minimize-cost|balanced|capability-first]\n")
	return nil
}

// Helper functions and mock implementations

func registerOrchestratorDemoAgents(componentRegistry registry.ComponentRegistry) error {
	agentRegistry, ok := componentRegistry.Agents().(*registry.DefaultAgentRegistry)
	if !ok {
		return gerror.New(gerror.ErrCodeInternal, "failed to get agent registry", nil).
			WithComponent("cli").
			WithOperation("registerOrchestratorDemoAgents")
	}

	demoAgents := []registry.GuildAgentConfig{
		{
			ID:            "shell-automator",
			Name:          "Shell Automator",
			Type:          "worker",
			Provider:      "local",
			Model:         "",
			Capabilities:  []string{"file_operations", "shell", "automation"},
			CostMagnitude: 0,
		},
		{
			ID:            "junior-dev",
			Name:          "Junior Developer",
			Type:          "worker",
			Provider:      "anthropic",
			Model:         "claude-3-haiku",
			Capabilities:  []string{"coding", "testing", "documentation"},
			CostMagnitude: 1,
		},
		{
			ID:            "full-stack-dev",
			Name:          "Full Stack Developer",
			Type:          "worker",
			Provider:      "openai",
			Model:         "gpt-3.5-turbo",
			Capabilities:  []string{"frontend", "backend", "database", "api", "ui-design"},
			CostMagnitude: 2,
		},
		{
			ID:            "security-specialist",
			Name:          "Security Specialist",
			Type:          "specialist",
			Provider:      "anthropic",
			Model:         "claude-3-sonnet",
			Capabilities:  []string{"security", "authentication", "encryption", "coding", "testing"},
			CostMagnitude: 3,
		},
		{
			ID:            "senior-architect",
			Name:          "Senior Architect",
			Type:          "manager",
			Provider:      "openai",
			Model:         "gpt-4-turbo",
			Capabilities:  []string{"architecture", "planning", "system-design"},
			CostMagnitude: 5,
		},
		{
			ID:            "expert-consultant",
			Name:          "Expert Consultant",
			Type:          "manager",
			Provider:      "anthropic",
			Model:         "claude-3-opus",
			Capabilities:  []string{"strategy", "optimization", "research"},
			CostMagnitude: 8,
		},
	}

	for _, agent := range demoAgents {
		if err := agentRegistry.RegisterGuildAgent(agent); err != nil {
			return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to register agent").
				WithComponent("cli").WithOperation("runOrchestratorDemo").WithDetails("agent_id", agent.ID)
		}
	}

	return nil
}

func registerOrchestratorDemoTools(componentRegistry registry.ComponentRegistry) error {
	toolRegistry, ok := componentRegistry.Tools().(*registry.DefaultToolRegistry)
	if !ok {
		return gerror.New(gerror.ErrCodeInternal, "failed to get tool registry", nil).
			WithComponent("cli").
			WithOperation("registerOrchestratorDemoTools")
	}

	demoTools := []struct {
		name         string
		costMagnitude int
		capabilities []string
	}{
		{"shell", 0, []string{"execution", "file_operations", "automation"}},
		{"git", 0, []string{"version_control", "collaboration"}},
		{"file_system", 0, []string{"file_operations", "read", "write"}},
		{"database", 1, []string{"database", "configuration", "persistence"}},
		{"http_client", 1, []string{"network", "api", "web"}},
		{"docker", 2, []string{"containers", "deployment", "isolation"}},
		{"security_scanner", 3, []string{"security", "analysis", "vulnerability"}},
		{"ai_assistant", 5, []string{"analysis", "optimization", "intelligence"}},
	}

	for _, tool := range demoTools {
		mockTool := &OrchestratorDemoTool{name: tool.name}
		if err := toolRegistry.RegisterToolWithCost(tool.name, mockTool, tool.costMagnitude, tool.capabilities); err != nil {
			return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to register tool").
				WithComponent("cli").WithOperation("registerOrchestratorDemoTools").WithDetails("tool_name", tool.name)
		}
	}

	return nil
}

func createOrchestratorGuildConfig() *config.GuildConfig {
	return &config.GuildConfig{
		Name: "Development Guild",
		Agents: []config.AgentConfig{
			{ID: "shell-automator", Name: "Shell Automator", Type: "worker", Capabilities: []string{"file_operations", "shell"}},
			{ID: "junior-dev", Name: "Junior Developer", Type: "worker", Capabilities: []string{"coding", "testing"}},
			{ID: "full-stack-dev", Name: "Full Stack Developer", Type: "worker", Capabilities: []string{"frontend", "backend"}},
			{ID: "security-specialist", Name: "Security Specialist", Type: "specialist", Capabilities: []string{"security", "authentication"}},
		},
	}
}

func getOrchestratorCostIcon(cost int) string {
	switch {
	case cost == 0: return "🆓"
	case cost <= 2: return "💚"
	case cost <= 4: return "💛"
	case cost <= 6: return "🧡"
	case cost <= 8: return "❤️"
	default: return "💜"
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Mock implementations

type OrchestratorMockKanbanBoard struct {
	tasks   map[string]*kanban.Task
	counter int
}

func (m *OrchestratorMockKanbanBoard) CreateTask(ctx context.Context, title, description string) (*kanban.Task, error) {
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

func (m *OrchestratorMockKanbanBoard) UpdateTask(ctx context.Context, task *kanban.Task) error {
	m.tasks[task.ID] = task
	return nil
}

func (m *OrchestratorMockKanbanBoard) GetTask(ctx context.Context, taskID string) (*kanban.Task, error) {
	if task, exists := m.tasks[taskID]; exists {
		return task, nil
	}
	return nil, gerror.New(gerror.ErrCodeNotFound, "task not found", nil).
		WithComponent("cli").WithOperation("GetTask").WithDetails("task_id", taskID)
}

func (m *OrchestratorMockKanbanBoard) ListTasksByStatus(ctx context.Context, boardID string, status kanban.TaskStatus) ([]*kanban.Task, error) {
	var tasks []*kanban.Task
	for _, task := range m.tasks {
		if task.Status == status {
			tasks = append(tasks, task)
		}
	}
	return tasks, nil
}

func (m *OrchestratorMockKanbanBoard) UpdateTaskStatus(ctx context.Context, taskID, status, assignee, comment string) error {
	if task, exists := m.tasks[taskID]; exists {
		task.Status = kanban.TaskStatus(status)
		task.AssignedTo = assignee
		return nil
	}
	return gerror.New(gerror.ErrCodeNotFound, "task not found", nil).
		WithComponent("cli").WithOperation("UpdateTaskStatus").WithDetails("task_id", taskID)
}

type OrchestratorMockManagerAgent struct{}

func (m *OrchestratorMockManagerAgent) Execute(ctx context.Context, request string) (string, error) {
	// Return a realistic task breakdown
	return `TASK-001: Setup project structure
Description: Initialize repository and basic project structure
Capabilities: file_operations, shell
Dependencies: none
Complexity: low
PreferredCost: 0
---
TASK-002: Implement user authentication
Description: Create login, registration, and session management
Capabilities: coding, security
Dependencies: TASK-001
Complexity: medium
PreferredCost: 2
---
TASK-003: Build frontend interface
Description: Create user interface components and pages
Capabilities: frontend, ui-design
Dependencies: TASK-001
Complexity: medium
PreferredCost: 2
---
TASK-004: Setup database layer
Description: Configure database and create data models
Capabilities: database, backend
Dependencies: TASK-001
Complexity: medium
PreferredCost: 2
---
TASK-005: Security audit and testing
Description: Review security implementation and run tests
Capabilities: security, testing
Dependencies: TASK-002, TASK-003, TASK-004
Complexity: high
PreferredCost: 3`, nil
}

func (m *OrchestratorMockManagerAgent) GetID() string   { return "orchestrator-manager" }
func (m *OrchestratorMockManagerAgent) GetName() string { return "Orchestrator Manager" }

type OrchestratorDemoTool struct {
	name string
}

func (d *OrchestratorDemoTool) Name() string        { return d.name }
func (d *OrchestratorDemoTool) Description() string { return fmt.Sprintf("Demo %s tool", d.name) }
func (d *OrchestratorDemoTool) Category() string    { return "demo" }
func (d *OrchestratorDemoTool) Schema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"input": map[string]interface{}{
				"type": "string",
				"description": "Input for the tool",
			},
		},
	}
}
func (d *OrchestratorDemoTool) Execute(ctx context.Context, input string) (*tools.ToolResult, error) {
	return &tools.ToolResult{
		Output:  fmt.Sprintf("Demo result from %s", d.name),
		Success: true,
	}, nil
}
func (d *OrchestratorDemoTool) Examples() []string    { return []string{fmt.Sprintf("example for %s", d.name)} }
func (d *OrchestratorDemoTool) RequiresAuth() bool    { return false }
