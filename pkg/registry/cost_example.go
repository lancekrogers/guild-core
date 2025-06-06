package registry

import (
	"context"
	"fmt"
	"log"
)

// ExampleCostBasedSelection demonstrates how to use the cost-based registry system
// for intelligent agent and tool selection in Guild
func ExampleCostBasedSelection() {
	// Create a component registry
	registry := NewComponentRegistry()
	
	// Initialize with mock configuration (in practice this loads from guild.yaml)
	config := Config{
		Agents: AgentConfigYaml{
			DefaultType: "worker",
		},
		Tools: ToolConfig{
			EnabledTools: []string{"shell", "git", "http_client"},
		},
	}
	
	ctx := context.Background()
	if err := registry.Initialize(ctx, config); err != nil {
		log.Printf("Failed to initialize registry: %v", err)
		return
	}
	
	// Example 1: Find the cheapest agent for coding tasks
	cheapestCoder, err := registry.GetCheapestAgentByCapability("coding")
	if err != nil {
		log.Printf("No coding agent found: %v", err)
	} else {
		fmt.Printf("Cheapest coding agent: %s (cost: %d)\n", 
			cheapestCoder.Name, cheapestCoder.CostMagnitude)
	}
	
	// Example 2: Get all agents within budget
	budgetAgents := registry.GetAgentsByCost(3) // Max cost magnitude 3
	fmt.Printf("Agents within budget (≤3): %d agents\n", len(budgetAgents))
	for _, agent := range budgetAgents {
		fmt.Printf("  - %s: cost %d, capabilities: %v\n", 
			agent.Name, agent.CostMagnitude, agent.Capabilities)
	}
	
	// Example 3: Find cheapest tool for file operations
	cheapestFileTool, err := registry.GetCheapestToolByCapability("file_operations")
	if err != nil {
		log.Printf("No file operations tool found: %v", err)
	} else {
		fmt.Printf("Cheapest file tool: %s (cost: %d)\n", 
			cheapestFileTool.Name, cheapestFileTool.CostMagnitude)
	}
	
	// Example 4: Get all tools within budget
	budgetTools := registry.GetToolsByCost(1) // Only free and cheap tools
	fmt.Printf("Tools within budget (≤1): %d tools\n", len(budgetTools))
	for _, tool := range budgetTools {
		fmt.Printf("  - %s: cost %d, capabilities: %v\n", 
			tool.Name, tool.CostMagnitude, tool.Capabilities)
	}
}

// CostAwareTaskAssignment demonstrates how an orchestrator might use
// the cost system for intelligent task assignment
func CostAwareTaskAssignment(registry ComponentRegistry, task Task, maxCost int) (*AgentInfo, error) {
	// Determine required capability from task type
	requiredCapability := mapTaskToCapability(task.Type)
	
	// Find the cheapest agent that can handle this task
	agent, err := registry.GetCheapestAgentByCapability(requiredCapability)
	if err != nil {
		return nil, fmt.Errorf("no agent available for capability '%s': %w", requiredCapability, err)
	}
	
	// Check if agent is within budget
	if agent.CostMagnitude > maxCost {
		return nil, fmt.Errorf("cheapest agent (cost %d) exceeds budget %d", agent.CostMagnitude, maxCost)
	}
	
	return agent, nil
}

// CostAwareToolSelection demonstrates how an agent might choose tools
// based on cost constraints
func CostAwareToolSelection(registry ComponentRegistry, capability string, maxCost int) (*ToolInfo, error) {
	// Find the cheapest tool for the required capability
	tool, err := registry.GetCheapestToolByCapability(capability)
	if err != nil {
		return nil, fmt.Errorf("no tool available for capability '%s': %w", capability, err)
	}
	
	// Check if tool is within budget
	if tool.CostMagnitude > maxCost {
		return nil, fmt.Errorf("cheapest tool (cost %d) exceeds budget %d", tool.CostMagnitude, maxCost)
	}
	
	return tool, nil
}

// OptimizeWorkflow demonstrates a complete workflow optimization
// that considers both agent and tool costs
func OptimizeWorkflow(registry ComponentRegistry, tasks []Task, totalBudget int) (WorkflowPlan, error) {
	plan := WorkflowPlan{
		Tasks:       make([]TaskAssignment, 0, len(tasks)),
		TotalCost:   0,
		BudgetUsed:  0,
	}
	
	for _, task := range tasks {
		// Allocate budget proportionally to remaining tasks
		remainingTasks := len(tasks) - len(plan.Tasks)
		taskBudget := (totalBudget - plan.TotalCost) / remainingTasks
		
		// Find optimal agent for this task
		agent, err := CostAwareTaskAssignment(registry, task, taskBudget)
		if err != nil {
			return plan, fmt.Errorf("failed to assign task %s: %w", task.ID, err)
		}
		
		// Find optimal tools for this agent
		toolBudget := taskBudget - agent.CostMagnitude
		var tools []ToolInfo
		
		for _, toolCap := range getRequiredToolCapabilities(task.Type) {
			tool, err := CostAwareToolSelection(registry, toolCap, toolBudget)
			if err != nil {
				// Tool not required, skip
				continue
			}
			tools = append(tools, *tool)
			toolBudget -= tool.CostMagnitude
		}
		
		assignment := TaskAssignment{
			Task:      task,
			Agent:     *agent,
			Tools:     tools,
			Cost:      agent.CostMagnitude + sumToolCosts(tools),
		}
		
		plan.Tasks = append(plan.Tasks, assignment)
		plan.TotalCost += assignment.Cost
	}
	
	plan.BudgetUsed = float64(plan.TotalCost) / float64(totalBudget) * 100
	return plan, nil
}

// Helper types for workflow optimization
type WorkflowPlan struct {
	Tasks      []TaskAssignment `json:"tasks"`
	TotalCost  int              `json:"total_cost"`
	BudgetUsed float64          `json:"budget_used_percent"`
}

type TaskAssignment struct {
	Task  Task        `json:"task"`
	Agent AgentInfo   `json:"agent"`
	Tools []ToolInfo  `json:"tools"`
	Cost  int         `json:"cost"`
}

// Helper functions
func mapTaskToCapability(taskType string) string {
	mapping := map[string]string{
		"code_review":    "review",
		"implementation": "coding",
		"documentation":  "documentation",
		"planning":       "planning",
		"testing":        "testing",
		"deployment":     "devops",
	}
	
	if capability, exists := mapping[taskType]; exists {
		return capability
	}
	return "general" // Default capability
}

func getRequiredToolCapabilities(taskType string) []string {
	mapping := map[string][]string{
		"code_review":    {"file_operations", "git"},
		"implementation": {"file_operations", "execution"},
		"documentation":  {"file_operations"},
		"planning":       {"documentation"},
		"testing":        {"execution", "file_operations"},
		"deployment":     {"execution", "network"},
	}
	
	if capabilities, exists := mapping[taskType]; exists {
		return capabilities
	}
	return []string{"file_operations"} // Default tools
}

func sumToolCosts(tools []ToolInfo) int {
	total := 0
	for _, tool := range tools {
		total += tool.CostMagnitude
	}
	return total
}

// ExampleUsagePatterns shows common usage patterns for the cost system
func ExampleUsagePatterns() {
	registry := NewComponentRegistry()
	
	// Pattern 1: Budget-constrained development
	fmt.Println("=== Budget-Constrained Development ===")
	lowBudgetAgents := registry.GetAgentsByCost(1) // Only cheap agents
	fmt.Printf("Available agents for low budget: %d\n", len(lowBudgetAgents))
	
	// Pattern 2: Emergency high-priority task (no budget constraints)
	fmt.Println("=== Emergency Task (No Budget Limits) ===")
	allAgents := registry.GetAgentsByCost(8) // All agents including expensive ones
	fmt.Printf("All available agents: %d\n", len(allAgents))
	
	// Pattern 3: Tool-only automation (zero cost)
	fmt.Println("=== Tool-Only Automation ===")
	freeTools := registry.GetToolsByCost(0) // Only free tools
	fmt.Printf("Free automation tools: %d\n", len(freeTools))
	
	// Pattern 4: Capability-first selection
	fmt.Println("=== Capability-First Selection ===")
	if agent, err := registry.GetCheapestAgentByCapability("architecture"); err == nil {
		fmt.Printf("Best architecture agent: %s (cost: %d)\n", agent.Name, agent.CostMagnitude)
	}
}