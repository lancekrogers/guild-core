package orchestrator

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/guild-ventures/guild-core/pkg/agent"
	"github.com/guild-ventures/guild-core/pkg/config"
	"github.com/guild-ventures/guild-core/pkg/gerror"
	"github.com/guild-ventures/guild-core/internal/kanban"
	"github.com/guild-ventures/guild-core/internal/commission"
	"github.com/guild-ventures/guild-core/pkg/registry"
)

// CostAwareTaskPlanner implements TaskPlanner with cost optimization
type CostAwareTaskPlanner struct {
	managerAgent      agent.Agent
	kanbanBoard       KanbanManager
	componentRegistry registry.ComponentRegistry
	maxCostFilter     int  // Maximum cost magnitude filter (exclude agents above this)
	balanceWorkload   bool // Whether to balance workload across agents
}

// CostAssignmentStrategy defines how cost optimization is applied
type CostAssignmentStrategy int

const (
	// StrategyMinimizeCost - Always choose the cheapest capable agent
	StrategyMinimizeCost CostAssignmentStrategy = iota
	
	// StrategyBalanced - Balance cost optimization with workload distribution
	StrategyBalanced
	
	// StrategyCapabilityFirst - Choose best capability match, then optimize cost
	StrategyCapabilityFirst
)

// AssignmentOptions configure cost-aware assignment behavior
type AssignmentOptions struct {
	MaxCostMagnitude int                    // Maximum allowed cost magnitude
	Strategy         CostAssignmentStrategy // Assignment strategy
	BalanceWorkload  bool                   // Whether to balance agent workloads
	PreferredAgents  []string               // Agent IDs to prefer (lower cost tie-breaking)
	RequiredTools    map[string][]string    // Task ID -> required tool capabilities
}

// TaskAssignmentResult contains assignment details with cost information
type TaskAssignmentResult struct {
	TaskID        string                 `json:"task_id"`
	AgentID       string                 `json:"agent_id"`
	AgentInfo     *registry.AgentInfo    `json:"agent_info"`
	Tools         []registry.ToolInfo    `json:"tools"`
	TotalCost     int                    `json:"total_cost"`
	Reason        string                 `json:"reason"`
	Alternatives  []registry.AgentInfo   `json:"alternatives"`
}

// AssignmentSummary provides overview of cost-optimized assignments
type AssignmentSummary struct {
	TotalTasks       int                    `json:"total_tasks"`
	TotalCost        int                    `json:"total_cost"`
	AverageCost      float64                `json:"average_cost"`
	CostEfficiency   string                 `json:"cost_efficiency"`
	Assignments      []TaskAssignmentResult `json:"assignments"`
	AgentWorkloads   map[string]int         `json:"agent_workloads"`
	CostBreakdown    map[string]int         `json:"cost_breakdown"`
}

// NewCostAwareTaskPlanner creates a new cost-optimized task planner
func newCostAwareTaskPlanner(
	managerAgent agent.Agent, 
	kanbanBoard KanbanManager, 
	componentRegistry registry.ComponentRegistry,
	maxCostFilter int,
) *CostAwareTaskPlanner {
	return &CostAwareTaskPlanner{
		managerAgent:      managerAgent,
		kanbanBoard:       kanbanBoard,
		componentRegistry: componentRegistry,
		maxCostFilter:     maxCostFilter,
		balanceWorkload:   true,
	}
}

// DefaultCostAwareTaskPlannerFactory creates a cost-aware planner factory for registry use
func DefaultCostAwareTaskPlannerFactory(
	managerAgent agent.Agent, 
	kanbanBoard KanbanManager, 
	componentRegistry registry.ComponentRegistry,
	maxCostFilter int,
) *CostAwareTaskPlanner {
	return newCostAwareTaskPlanner(managerAgent, kanbanBoard, componentRegistry, maxCostFilter)
}

// PlanTasks decomposes an objective into tasks (enhanced with cost awareness)
func (p *CostAwareTaskPlanner) PlanTasks(ctx context.Context, obj *commission.Commission, guild *config.GuildConfig) ([]*kanban.Task, error) {
	// Build a planning prompt with cost information
	prompt := p.buildCostAwarePlanningPrompt(obj, guild)
	
	// Execute the planning request
	response, err := p.managerAgent.Execute(ctx, prompt)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeAgent, "manager agent failed to plan tasks").
			WithComponent("orchestrator").
			WithOperation("PlanTasks")
	}
	
	// Parse the response into tasks
	tasks, err := p.parseTasksFromResponse(response)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeOrchestration, "failed to parse tasks from response").
			WithComponent("orchestrator").
			WithOperation("PlanTasks")
	}
	
	// Enhance tasks with cost estimates
	if err := p.enhanceTasksWithCostEstimates(ctx, tasks, guild); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeOrchestration, "failed to enhance tasks with cost estimates").
			WithComponent("orchestrator").
			WithOperation("PlanTasks")
	}
	
	// Add tasks to kanban board
	for _, task := range tasks {
		// Create task on board
		createdTask, err := p.kanbanBoard.CreateTask(ctx, task.Title, task.Description)
		if err != nil {
			return nil, gerror.Wrap(err, gerror.ErrCodeOrchestration, "failed to create task on kanban board").
				WithComponent("orchestrator").
				WithOperation("PlanTasks")
		}
		
		// Update task with parsed data and cost information
		createdTask.Status = task.Status
		createdTask.Metadata = task.Metadata
		createdTask.Dependencies = task.Dependencies
		
		// Update the task
		if err := p.kanbanBoard.UpdateTask(ctx, createdTask); err != nil {
			return nil, gerror.Wrap(err, gerror.ErrCodeOrchestration, "failed to update task").
				WithComponent("orchestrator").
				WithOperation("PlanTasks")
		}
		
		// Update our reference
		task.ID = createdTask.ID
	}
	
	return tasks, nil
}

// AssignTasks assigns tasks using cost-aware optimization
func (p *CostAwareTaskPlanner) AssignTasks(ctx context.Context, tasks []*kanban.Task, guild *config.GuildConfig) error {
	options := AssignmentOptions{
		MaxCostMagnitude: p.maxCostFilter,
		Strategy:         StrategyBalanced,
		BalanceWorkload:  p.balanceWorkload,
		RequiredTools:    make(map[string][]string),
	}
	
	summary, err := p.AssignTasksWithOptions(ctx, tasks, guild, options)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeOrchestration, "failed to assign tasks with cost optimization").
			WithComponent("CostAwareTaskPlanner").
			WithOperation("assignTasks")
	}
	
	// Apply assignments to kanban board
	for _, assignment := range summary.Assignments {
		task, err := p.kanbanBoard.GetTask(ctx, assignment.TaskID)
		if err != nil {
			continue // Skip if task not found
		}
		
		// Update task assignment with cost information
		task.AssignedTo = assignment.AgentID
		task.Metadata["assigned_to"] = assignment.AgentID
		task.Metadata["assignment_cost"] = fmt.Sprintf("%d", assignment.TotalCost)
		task.Metadata["assignment_reason"] = assignment.Reason
		task.Metadata["agent_name"] = assignment.AgentInfo.Name
		task.Metadata["agent_type"] = assignment.AgentInfo.Type
		
		if err := p.kanbanBoard.UpdateTask(ctx, task); err != nil {
			return gerror.Wrap(err, gerror.ErrCodeOrchestration, "failed to update task assignment").
				WithComponent("orchestrator").
				WithOperation("AssignTasks")
		}
	}
	
	return nil
}

// AssignTasksWithOptions performs cost-aware assignment with detailed configuration
func (p *CostAwareTaskPlanner) AssignTasksWithOptions(
	ctx context.Context, 
	tasks []*kanban.Task, 
	guild *config.GuildConfig, 
	options AssignmentOptions,
) (*AssignmentSummary, error) {
	
	summary := &AssignmentSummary{
		TotalTasks:     len(tasks),
		Assignments:    make([]TaskAssignmentResult, 0, len(tasks)),
		AgentWorkloads: make(map[string]int),
		CostBreakdown:  make(map[string]int),
	}
	
	// Sort tasks by complexity and dependencies for optimal assignment order
	sortedTasks := p.sortTasksByPriority(tasks)
	
	for _, task := range sortedTasks {
		assignment, err := p.assignSingleTask(ctx, task, guild, options, summary.AgentWorkloads)
		if err != nil {
			return nil, gerror.Wrap(err, gerror.ErrCodeOrchestration, "failed to assign task").
				WithComponent("orchestrator").
				WithOperation("AssignTasksWithOptions").
				WithDetails("task_id", task.ID)
		}
		
		// Update summary
		summary.Assignments = append(summary.Assignments, *assignment)
		summary.TotalCost += assignment.TotalCost
		summary.AgentWorkloads[assignment.AgentID]++
		summary.CostBreakdown[assignment.AgentInfo.Type] += assignment.TotalCost
	}
	
	// Calculate summary statistics
	if summary.TotalTasks > 0 {
		summary.AverageCost = float64(summary.TotalCost) / float64(summary.TotalTasks)
	}
	// Calculate cost efficiency description instead of percentage
	if summary.AverageCost <= 2 {
		summary.CostEfficiency = "Economical"
	} else if summary.AverageCost <= 4 {
		summary.CostEfficiency = "Balanced"
	} else {
		summary.CostEfficiency = "Premium"
	}
	
	return summary, nil
}

// assignSingleTask assigns a single task using cost optimization
func (p *CostAwareTaskPlanner) assignSingleTask(
	ctx context.Context,
	task *kanban.Task,
	guild *config.GuildConfig,
	options AssignmentOptions,
	currentWorkloads map[string]int,
) (*TaskAssignmentResult, error) {
	
	// Extract required capabilities from task metadata
	capabilitiesStr, _ := task.Metadata["capabilities"]
	requiredCapabilities := p.parseCapabilities(capabilitiesStr)
	
	if len(requiredCapabilities) == 0 {
		return nil, gerror.New(gerror.ErrCodeValidation, "task has no required capabilities specified", nil).
			WithComponent("orchestrator").
			WithOperation("assignSingleTask").
			WithDetails("task_id", task.ID)
	}
	
	// Get agents that can handle the primary capability
	primaryCapability := requiredCapabilities[0]
	
	var assignedAgent *registry.AgentInfo
	var reason string
	var alternatives []registry.AgentInfo
	
	switch options.Strategy {
	case StrategyMinimizeCost:
		agent, err := p.componentRegistry.GetCheapestAgentByCapability(primaryCapability)
		if err != nil {
			return nil, gerror.Wrap(err, gerror.ErrCodeAgent, "no agent found for capability").
				WithComponent("orchestrator").
				WithOperation("assignSingleTask").
				WithDetails("capability", primaryCapability)
		}
		if agent.CostMagnitude > options.MaxCostMagnitude {
			return nil, gerror.New(gerror.ErrCodeValidation, "cheapest agent exceeds budget", nil).
				WithComponent("orchestrator").
				WithOperation("assignSingleTask").
				WithDetails("agent_cost", agent.CostMagnitude).
				WithDetails("max_budget", options.MaxCostMagnitude)
		}
		assignedAgent = agent
		reason = fmt.Sprintf("Cheapest agent for capability '%s'", primaryCapability)
		
	case StrategyBalanced:
		candidates := p.componentRegistry.GetAgentsByCost(options.MaxCostMagnitude)
		suitableAgents := p.filterAgentsByCapabilities(candidates, requiredCapabilities)
		
		if len(suitableAgents) == 0 {
			return nil, gerror.New(gerror.ErrCodeNoAvailableAgent, "no suitable agents found for capabilities within budget", nil).
				WithComponent("orchestrator").
				WithOperation("assignSingleTask").
				WithDetails("required_capabilities", fmt.Sprintf("%v", requiredCapabilities)).
				WithDetails("max_budget", options.MaxCostMagnitude)
		}
		
		// Apply workload balancing
		if options.BalanceWorkload {
			assignedAgent = p.selectBalancedAgent(suitableAgents, currentWorkloads)
			reason = "Balanced cost and workload optimization"
		} else {
			assignedAgent = &suitableAgents[0] // Already sorted by cost
			reason = "Cost-optimized selection"
		}
		
		alternatives = suitableAgents[1:] // Store alternatives for reporting
		
	case StrategyCapabilityFirst:
		allAgents := p.componentRegistry.GetAgentsByCapability(primaryCapability)
		if len(allAgents) == 0 {
			return nil, gerror.New(gerror.ErrCodeAgent, "no agent found for capability", nil).
				WithComponent("orchestrator").
				WithOperation("assignSingleTask").
				WithDetails("capability", primaryCapability)
		}
		
		// Find the best capability match within cost filter
		for _, agent := range allAgents {
			if agent.CostMagnitude <= options.MaxCostMagnitude &&
			   p.hasAllCapabilities(agent.Capabilities, requiredCapabilities) {
				assignedAgent = &agent
				reason = "Best capability match within cost limit"
				break
			}
		}
		
		if assignedAgent == nil {
			return nil, gerror.New(gerror.ErrCodeNoAvailableAgent, "no agent with required capabilities within cost filter", nil).
				WithComponent("orchestrator").
				WithOperation("assignSingleTask").
				WithDetails("cost_filter", options.MaxCostMagnitude)
		}
	}
	
	// Find required tools for the task
	requiredToolCaps, exists := options.RequiredTools[task.ID]
	if !exists {
		// Default tool requirements based on task complexity
		complexity, _ := task.Metadata["complexity"]
		requiredToolCaps = p.getDefaultToolRequirements(complexity)
	}
	
	// Select optimal tools
	tools, toolCost := p.selectOptimalTools(requiredToolCaps, options.MaxCostMagnitude-assignedAgent.CostMagnitude)
	
	result := &TaskAssignmentResult{
		TaskID:       task.ID,
		AgentID:      assignedAgent.ID,
		AgentInfo:    assignedAgent,
		Tools:        tools,
		TotalCost:    assignedAgent.CostMagnitude + toolCost,
		Reason:       reason,
		Alternatives: alternatives,
	}
	
	return result, nil
}

// selectOptimalTools chooses the best tools within remaining budget
func (p *CostAwareTaskPlanner) selectOptimalTools(requiredCapabilities []string, remainingBudget int) ([]registry.ToolInfo, int) {
	var selectedTools []registry.ToolInfo
	totalCost := 0
	
	for _, capability := range requiredCapabilities {
		tool, err := p.componentRegistry.GetCheapestToolByCapability(capability)
		if err != nil {
			continue // Tool not available
		}
		
		if tool.CostMagnitude <= remainingBudget {
			selectedTools = append(selectedTools, *tool)
			totalCost += tool.CostMagnitude
			remainingBudget -= tool.CostMagnitude
		}
	}
	
	return selectedTools, totalCost
}

// Helper methods

func (p *CostAwareTaskPlanner) buildCostAwarePlanningPrompt(obj *commission.Commission, guild *config.GuildConfig) string {
	var prompt strings.Builder
	
	prompt.WriteString("You are the manager agent for the ")
	prompt.WriteString(guild.Name)
	prompt.WriteString(" guild. Your task is to decompose the following objective into concrete, actionable tasks with cost optimization in mind.\n\n")
	
	prompt.WriteString("## Objective\n")
	prompt.WriteString(obj.Format())
	prompt.WriteString("\n\n")
	
	prompt.WriteString("## Available Agents and Capabilities (with Cost Information)\n")
	agents := p.componentRegistry.GetAgentsByCost(p.maxCostFilter)
	for _, agent := range agents {
		costIcon := p.getCostIcon(agent.CostMagnitude)
		prompt.WriteString(fmt.Sprintf("- **%s** (%s) %s Cost: %d | Capabilities: %s\n", 
			agent.Name, agent.ID, costIcon, agent.CostMagnitude, strings.Join(agent.Capabilities, ", ")))
	}
	prompt.WriteString("\n")
	
	prompt.WriteString("## Cost Considerations\n")
	if p.maxCostFilter < 8 {
		prompt.WriteString(fmt.Sprintf("Note: Agents with cost magnitude > %d are excluded from consideration.\n", p.maxCostFilter))
	}
	prompt.WriteString("Consider cost-effectiveness when assigning tasks.\n\n")
	
	prompt.WriteString("## Instructions\n")
	prompt.WriteString("Break down the objective into specific tasks optimized for cost efficiency. For each task, provide:\n")
	prompt.WriteString("1. A unique task ID (e.g., TASK-001)\n")
	prompt.WriteString("2. A clear, concise title\n")
	prompt.WriteString("3. A detailed description\n")
	prompt.WriteString("4. Required capabilities (prioritize cheaper agents when possible)\n")
	prompt.WriteString("5. Dependencies on other tasks (if any)\n")
	prompt.WriteString("6. Estimated complexity (low, medium, high)\n")
	prompt.WriteString("7. Preferred cost level (0=free, 1=cheap, 2=mid, 3=expensive)\n\n")
	
	prompt.WriteString("Format your response as follows:\n")
	prompt.WriteString("```\n")
	prompt.WriteString("TASK-001: [Title]\n")
	prompt.WriteString("Description: [Detailed description]\n")
	prompt.WriteString("Capabilities: [capability1, capability2]\n")
	prompt.WriteString("Dependencies: [TASK-XXX, TASK-YYY] or none\n")
	prompt.WriteString("Complexity: [low|medium|high]\n")
	prompt.WriteString("PreferredCost: [0|1|2|3]\n")
	prompt.WriteString("---\n")
	prompt.WriteString("```\n")
	
	return prompt.String()
}

func (p *CostAwareTaskPlanner) enhanceTasksWithCostEstimates(ctx context.Context, tasks []*kanban.Task, guild *config.GuildConfig) error {
	for _, task := range tasks {
		// Extract capabilities
		capabilitiesStr, _ := task.Metadata["capabilities"]
		capabilities := p.parseCapabilities(capabilitiesStr)
		
		if len(capabilities) > 0 {
			// Find cheapest agent for estimation
			agent, err := p.componentRegistry.GetCheapestAgentByCapability(capabilities[0])
			if err == nil {
				task.Metadata["estimated_cost"] = fmt.Sprintf("%d", agent.CostMagnitude)
				task.Metadata["cheapest_agent"] = agent.Name
			}
		}
		
		// Extract preferred cost from planning
		if preferredCost, exists := task.Metadata["preferred_cost"]; exists {
			task.Metadata["preferred_cost"] = preferredCost
		}
	}
	
	return nil
}

func (p *CostAwareTaskPlanner) parseCapabilities(capabilitiesStr string) []string {
	if capabilitiesStr == "" {
		return nil
	}
	
	capabilities := strings.Split(capabilitiesStr, ",")
	for i := range capabilities {
		capabilities[i] = strings.TrimSpace(capabilities[i])
	}
	
	return capabilities
}

func (p *CostAwareTaskPlanner) filterAgentsByCapabilities(agents []registry.AgentInfo, requiredCapabilities []string) []registry.AgentInfo {
	var suitable []registry.AgentInfo
	
	for _, agent := range agents {
		if p.hasAllCapabilities(agent.Capabilities, requiredCapabilities) {
			suitable = append(suitable, agent)
		}
	}
	
	return suitable
}

func (p *CostAwareTaskPlanner) hasAllCapabilities(agentCapabilities, requiredCapabilities []string) bool {
	for _, required := range requiredCapabilities {
		found := false
		for _, agentCap := range agentCapabilities {
			if agentCap == required {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

func (p *CostAwareTaskPlanner) selectBalancedAgent(agents []registry.AgentInfo, currentWorkloads map[string]int) *registry.AgentInfo {
	// Sort by workload first, then by cost
	sort.Slice(agents, func(i, j int) bool {
		workloadI := currentWorkloads[agents[i].ID]
		workloadJ := currentWorkloads[agents[j].ID]
		
		if workloadI != workloadJ {
			return workloadI < workloadJ // Prefer less loaded agent
		}
		
		return agents[i].CostMagnitude < agents[j].CostMagnitude // Then prefer cheaper
	})
	
	return &agents[0]
}

func (p *CostAwareTaskPlanner) sortTasksByPriority(tasks []*kanban.Task) []*kanban.Task {
	sortedTasks := make([]*kanban.Task, len(tasks))
	copy(sortedTasks, tasks)
	
	// Sort by dependencies first, then by complexity
	sort.Slice(sortedTasks, func(i, j int) bool {
		// Tasks with no dependencies come first
		iDeps := len(sortedTasks[i].Dependencies)
		jDeps := len(sortedTasks[j].Dependencies)
		
		if iDeps != jDeps {
			return iDeps < jDeps
		}
		
		// Then by complexity (high complexity first to get expensive agents early)
		iComplexity := p.getComplexityScore(sortedTasks[i].Metadata["complexity"])
		jComplexity := p.getComplexityScore(sortedTasks[j].Metadata["complexity"])
		
		return iComplexity > jComplexity
	})
	
	return sortedTasks
}

func (p *CostAwareTaskPlanner) getComplexityScore(complexity string) int {
	switch complexity {
	case "high":
		return 3
	case "medium":
		return 2
	case "low":
		return 1
	default:
		return 2 // Default to medium
	}
}

func (p *CostAwareTaskPlanner) getDefaultToolRequirements(complexity string) []string {
	switch complexity {
	case "high":
		return []string{"file_operations", "execution", "analysis"}
	case "medium":
		return []string{"file_operations", "execution"}
	case "low":
		return []string{"file_operations"}
	default:
		return []string{"file_operations"}
	}
}

func (p *CostAwareTaskPlanner) getCostIcon(cost int) string {
	switch cost {
	case 0: return "🆓"
	case 1: return "💚"
	case 2: return "💛"
	case 3: return "🧡"
	case 5: return "❤️"
	case 8: return "💜"
	default: return "❓"
	}
}

// parseTasksFromResponse parses tasks from the manager's response (enhanced for cost info)
func (p *CostAwareTaskPlanner) parseTasksFromResponse(response string) ([]*kanban.Task, error) {
	tasks := []*kanban.Task{}
	
	lines := strings.Split(response, "\n")
	var currentTask *kanban.Task
	
	for _, line := range lines {
		line = strings.TrimSpace(line)
		
		// Skip empty lines and markers
		if line == "" || line == "```" || line == "---" {
			if currentTask != nil {
				tasks = append(tasks, currentTask)
				currentTask = nil
			}
			continue
		}
		
		// Parse task ID and title
		if strings.Contains(line, ":") && strings.HasPrefix(line, "TASK-") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				currentTask = &kanban.Task{
					ID:       strings.TrimSpace(parts[0]),
					Title:    strings.TrimSpace(parts[1]),
					Status:   kanban.StatusTodo,
					Metadata: make(map[string]string),
				}
			}
		} else if currentTask != nil {
			// Parse task properties
			if strings.HasPrefix(line, "Description:") {
				currentTask.Description = strings.TrimSpace(strings.TrimPrefix(line, "Description:"))
			} else if strings.HasPrefix(line, "Capabilities:") {
				capsStr := strings.TrimSpace(strings.TrimPrefix(line, "Capabilities:"))
				currentTask.Metadata["capabilities"] = capsStr
			} else if strings.HasPrefix(line, "Dependencies:") {
				depsStr := strings.TrimSpace(strings.TrimPrefix(line, "Dependencies:"))
				if depsStr != "none" {
					currentTask.Dependencies = strings.Split(depsStr, ",")
					for i := range currentTask.Dependencies {
						currentTask.Dependencies[i] = strings.TrimSpace(currentTask.Dependencies[i])
					}
				}
			} else if strings.HasPrefix(line, "Complexity:") {
				currentTask.Metadata["complexity"] = strings.TrimSpace(strings.TrimPrefix(line, "Complexity:"))
			} else if strings.HasPrefix(line, "PreferredCost:") {
				currentTask.Metadata["preferred_cost"] = strings.TrimSpace(strings.TrimPrefix(line, "PreferredCost:"))
			}
		}
	}
	
	// Add last task if any
	if currentTask != nil {
		tasks = append(tasks, currentTask)
	}
	
	if len(tasks) == 0 {
		return nil, gerror.New(gerror.ErrCodeOrchestration, "no tasks found in response", nil).
			WithComponent("orchestrator").
			WithOperation("parseTasksFromResponse")
	}
	
	return tasks, nil
}