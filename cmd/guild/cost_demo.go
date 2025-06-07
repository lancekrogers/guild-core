package main

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/guild-ventures/guild-core/pkg/gerror"
	"github.com/guild-ventures/guild-core/pkg/registry"
	"github.com/guild-ventures/guild-core/tools"
)

// costCmd demonstrates the cost-based agent and tool selection system
var costCmd = &cobra.Command{
	Use:   "cost-demo",
	Short: "Demonstrate cost-based agent and tool selection",
	Long: `Demonstrates the enhanced cost-based selection system for agents and tools.
	
This command shows how Guild intelligently selects the most cost-effective
agents and tools for different tasks, using the Fibonacci cost magnitude scale:

Cost Scale:
- 0: Free (bash/tool operations only)
- 1: Cheap API usage (Claude Haiku, basic models)
- 2: Low-mid cost (GPT-3.5, mid-tier models)
- 3: Mid cost (Claude Sonnet, capable models)
- 5: High cost (GPT-4, advanced models)
- 8: Most expensive (Claude Opus, premium models)

Examples:
  guild cost-demo --budget 3 --capability coding
  guild cost-demo --list-agents
  guild cost-demo --list-tools --max-cost 1
  guild cost-demo --simulate-task review`,
	RunE: runCostDemo,
}

var (
	budget     int
	capability string
	listAgents bool
	listTools  bool
	maxCost    int
	simulate   string
)

func init() {
	costCmd.Flags().IntVar(&budget, "budget", 5, "Maximum cost budget for selection")
	costCmd.Flags().StringVar(&capability, "capability", "", "Required capability (e.g., coding, review, planning)")
	costCmd.Flags().BoolVar(&listAgents, "list-agents", false, "List all available agents with cost information")
	costCmd.Flags().BoolVar(&listTools, "list-tools", false, "List all available tools with cost information")
	costCmd.Flags().IntVar(&maxCost, "max-cost", 8, "Maximum cost for listing agents/tools")
	costCmd.Flags().StringVar(&simulate, "simulate-task", "", "Simulate task assignment (coding, review, planning, etc.)")
}

func runCostDemo(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	
	// Create and initialize registry
	componentRegistry := registry.NewComponentRegistry()
	
	// Initialize with basic configuration
	config := registry.Config{
		Agents: registry.AgentConfigYaml{
			DefaultType: "worker",
		},
		Tools: registry.ToolConfig{
			EnabledTools: []string{"shell", "git", "http_client"},
		},
	}
	
	if err := componentRegistry.Initialize(ctx, config); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to initialize registry").
			WithComponent("cli").
			WithOperation("runCostDemo")
	}
	
	// Register some demo agents to show the system working
	if err := registerDemoAgents(componentRegistry); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to register demo agents").
			WithComponent("cli").
			WithOperation("runCostDemo")
	}
	
	if err := registerDemoTools(componentRegistry); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to register demo tools").
			WithComponent("cli").
			WithOperation("runCostDemo")
	}
	
	// Execute the requested operation
	switch {
	case listAgents:
		return showAgentsList(componentRegistry, maxCost)
	case listTools:
		return showToolsList(componentRegistry, maxCost)
	case capability != "":
		return showCapabilitySelection(componentRegistry, capability, budget)
	case simulate != "":
		return simulateTaskAssignment(componentRegistry, simulate, budget)
	default:
		return showOverview(componentRegistry, budget)
	}
}

func registerDemoAgents(componentRegistry registry.ComponentRegistry) error {
	agentRegistry, ok := componentRegistry.Agents().(*registry.DefaultAgentRegistry)
	if !ok {
		return gerror.New(gerror.ErrCodeInternal, "failed to get agent registry", nil).
			WithComponent("cli").
			WithOperation("registerDemoAgents")
	}
	
	demoAgents := []registry.GuildAgentConfig{
		{
			ID:            "tools-agent",
			Name:          "Tools Agent",
			Type:          "worker",
			Provider:      "local",
			Model:         "",
			Capabilities:  []string{"file_operations", "git", "shell"},
			CostMagnitude: 0,
		},
		{
			ID:            "quick-coder",
			Name:          "Quick Coder",
			Type:          "worker",
			Provider:      "anthropic",
			Model:         "claude-3-haiku",
			Capabilities:  []string{"coding", "documentation", "testing"},
			CostMagnitude: 1,
		},
		{
			ID:            "balanced-dev",
			Name:          "Balanced Developer",
			Type:          "worker",
			Provider:      "openai",
			Model:         "gpt-3.5-turbo",
			Capabilities:  []string{"coding", "frontend", "backend"},
			CostMagnitude: 2,
		},
		{
			ID:            "senior-architect",
			Name:          "Senior Architect",
			Type:          "specialist",
			Provider:      "anthropic",
			Model:         "claude-3-sonnet",
			Capabilities:  []string{"architecture", "review", "planning"},
			CostMagnitude: 3,
		},
		{
			ID:            "expert-advisor",
			Name:          "Expert Advisor",
			Type:          "manager",
			Provider:      "openai",
			Model:         "gpt-4-turbo",
			Capabilities:  []string{"planning", "management", "strategy"},
			CostMagnitude: 5,
		},
		{
			ID:            "ai-specialist",
			Name:          "AI Specialist",
			Type:          "manager",
			Provider:      "anthropic",
			Model:         "claude-3-opus",
			Capabilities:  []string{"planning", "research", "architecture"},
			CostMagnitude: 8,
		},
	}
	
	for _, agent := range demoAgents {
		if err := agentRegistry.RegisterGuildAgent(agent); err != nil {
			return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to register agent").
				WithComponent("cli").
				WithOperation("registerDemoAgents").
				WithDetails("agent_id", agent.ID)
		}
	}
	
	return nil
}

func registerDemoTools(componentRegistry registry.ComponentRegistry) error {
	toolRegistry, ok := componentRegistry.Tools().(*registry.DefaultToolRegistry)
	if !ok {
		return gerror.New(gerror.ErrCodeInternal, "failed to get tool registry", nil).
			WithComponent("cli").
			WithOperation("registerDemoTools")
	}
	
	// Register demo tools (these would be actual tool implementations in practice)
	demoTools := []struct {
		name         string
		costMagnitude int
		capabilities []string
	}{
		{"shell", 0, []string{"execution", "file_operations"}},
		{"git", 0, []string{"version_control", "collaboration"}},
		{"file_system", 0, []string{"file_operations", "read", "write"}},
		{"http_client", 1, []string{"network", "api", "web"}},
		{"database", 1, []string{"data", "persistence", "query"}},
		{"docker", 2, []string{"containers", "deployment"}},
		{"kubernetes", 3, []string{"orchestration", "deployment", "scaling"}},
		{"ai_analysis", 5, []string{"ai", "analysis", "intelligence"}},
	}
	
	for _, tool := range demoTools {
		// Create a mock tool for demonstration
		mockTool := &DemoTool{name: tool.name}
		if err := toolRegistry.RegisterToolWithCost(tool.name, mockTool, tool.costMagnitude, tool.capabilities); err != nil {
			return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to register tool").
				WithComponent("cli").
				WithOperation("registerDemoTools").
				WithDetails("tool_name", tool.name)
		}
	}
	
	return nil
}

func showAgentsList(componentRegistry registry.ComponentRegistry, maxCost int) error {
	agents := componentRegistry.GetAgentsByCost(maxCost)
	
	fmt.Printf("🏰 Available Agents (Cost ≤ %d)\n", maxCost)
	fmt.Printf("═══════════════════════════════════════\n\n")
	
	if len(agents) == 0 {
		fmt.Printf("No agents available within cost budget of %d\n", maxCost)
		return nil
	}
	
	for _, agent := range agents {
		costIcon := getCostIcon(agent.CostMagnitude)
		fmt.Printf("%s %s (ID: %s)\n", costIcon, agent.Name, agent.ID)
		fmt.Printf("   Cost: %d | Type: %s\n", agent.CostMagnitude, agent.Type)
		fmt.Printf("   Capabilities: %v\n", agent.Capabilities)
		fmt.Printf("\n")
	}
	
	return nil
}

func showToolsList(componentRegistry registry.ComponentRegistry, maxCost int) error {
	tools := componentRegistry.GetToolsByCost(maxCost)
	
	fmt.Printf("🔧 Available Tools (Cost ≤ %d)\n", maxCost)
	fmt.Printf("═══════════════════════════════════════\n\n")
	
	if len(tools) == 0 {
		fmt.Printf("No tools available within cost budget of %d\n", maxCost)
		return nil
	}
	
	for _, tool := range tools {
		costIcon := getCostIcon(tool.CostMagnitude)
		fmt.Printf("%s %s\n", costIcon, tool.Name)
		fmt.Printf("   Cost: %d | Available: %t\n", tool.CostMagnitude, tool.Available)
		fmt.Printf("   Capabilities: %v\n", tool.Capabilities)
		fmt.Printf("\n")
	}
	
	return nil
}

func showCapabilitySelection(componentRegistry registry.ComponentRegistry, capability string, budget int) error {
	fmt.Printf("🎯 Finding Best Agent for '%s' (Budget: %d)\n", capability, budget)
	fmt.Printf("═══════════════════════════════════════════════════\n\n")
	
	// Find cheapest agent with the capability
	agent, err := componentRegistry.GetCheapestAgentByCapability(capability)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeNotFound, "no agent found with capability").
			WithComponent("cli").
			WithOperation("showCapabilitySelection").
			WithDetails("capability", capability)
	}
	
	if agent.CostMagnitude > budget {
		fmt.Printf("❌ Cheapest agent exceeds budget!\n")
		fmt.Printf("   Required: %s (Cost: %d)\n", agent.Name, agent.CostMagnitude)
		fmt.Printf("   Budget: %d\n", budget)
		return nil
	}
	
	fmt.Printf("✅ Selected Agent: %s\n", agent.Name)
	fmt.Printf("   Cost: %d (within budget of %d)\n", agent.CostMagnitude, budget)
	fmt.Printf("   Type: %s\n", agent.Type)
	fmt.Printf("   All Capabilities: %v\n", agent.Capabilities)
	
	// Show alternative agents
	fmt.Printf("\n🔄 Alternative Agents:\n")
	allAgents := componentRegistry.GetAgentsByCost(budget)
	alternativeCount := 0
	for _, alt := range allAgents {
		if alt.ID != agent.ID && hasCapability(alt.Capabilities, capability) {
			alternativeCount++
			costIcon := getCostIcon(alt.CostMagnitude)
			fmt.Printf("   %s %s (Cost: %d)\n", costIcon, alt.Name, alt.CostMagnitude)
		}
	}
	
	if alternativeCount == 0 {
		fmt.Printf("   No alternatives within budget.\n")
	}
	
	return nil
}

func simulateTaskAssignment(componentRegistry registry.ComponentRegistry, taskType string, budget int) error {
	fmt.Printf("🎮 Simulating Task Assignment\n")
	fmt.Printf("═══════════════════════════════════\n")
	fmt.Printf("Task: %s | Budget: %d\n\n", taskType, budget)
	
	// Map task types to required capabilities
	taskCapabilityMap := map[string]string{
		"coding":        "coding",
		"review":        "review",
		"planning":      "planning",
		"architecture":  "architecture",
		"documentation": "documentation",
		"testing":       "testing",
		"deployment":    "deployment",
	}
	
	capability, exists := taskCapabilityMap[taskType]
	if !exists {
		return gerror.New(gerror.ErrCodeInvalidInput, "unknown task type", nil).
			WithComponent("cli").
			WithOperation("simulateTaskAssignment").
			WithDetails("task_type", taskType)
	}
	
	// Find optimal agent
	agent, err := componentRegistry.GetCheapestAgentByCapability(capability)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeNotFound, "no agent available for task").
			WithComponent("cli").
			WithOperation("simulateTaskAssignment").
			WithDetails("task_type", taskType)
	}
	
	if agent.CostMagnitude > budget {
		fmt.Printf("❌ Task cannot be completed within budget\n")
		fmt.Printf("   Required agent cost: %d\n", agent.CostMagnitude)
		fmt.Printf("   Available budget: %d\n", budget)
		return nil
	}
	
	// Find tools for the agent
	remainingBudget := budget - agent.CostMagnitude
	fmt.Printf("✅ Task Assignment Successful!\n\n")
	fmt.Printf("📋 Selected Agent:\n")
	fmt.Printf("   %s (Cost: %d)\n", agent.Name, agent.CostMagnitude)
	fmt.Printf("   Remaining budget: %d\n\n", remainingBudget)
	
	// Show available tools within remaining budget
	tools := componentRegistry.GetToolsByCost(remainingBudget)
	fmt.Printf("🔧 Available Tools (Cost ≤ %d):\n", remainingBudget)
	if len(tools) == 0 {
		fmt.Printf("   No tools available within remaining budget\n")
	} else {
		for _, tool := range tools {
			costIcon := getCostIcon(tool.CostMagnitude)
			fmt.Printf("   %s %s (Cost: %d)\n", costIcon, tool.Name, tool.CostMagnitude)
		}
	}
	
	// Calculate total potential cost
	totalMinCost := agent.CostMagnitude
	if len(tools) > 0 {
		totalMinCost += tools[0].CostMagnitude // Add cheapest tool
	}
	
	fmt.Printf("\n💰 Cost Summary:\n")
	fmt.Printf("   Agent: %d\n", agent.CostMagnitude)
	fmt.Printf("   Cheapest tool: %d\n", func() int {
		if len(tools) > 0 { return tools[0].CostMagnitude }
		return 0
	}())
	fmt.Printf("   Total minimum: %d\n", totalMinCost)
	fmt.Printf("   Budget utilization: %.1f%%\n", float64(totalMinCost)/float64(budget)*100)
	
	return nil
}

func showOverview(componentRegistry registry.ComponentRegistry, budget int) error {
	fmt.Printf("🏰 Guild Cost-Based Selection System\n")
	fmt.Printf("═══════════════════════════════════════\n\n")
	
	// Show agents by cost tiers
	fmt.Printf("📊 Agent Distribution by Cost:\n")
	for cost := 0; cost <= 8; cost++ {
		if cost == 4 || cost == 6 || cost == 7 { continue } // Skip invalid Fibonacci values
		agents := componentRegistry.GetAgentsByCost(cost)
		var countAtThisCost int
		for _, agent := range agents {
			if agent.CostMagnitude == cost {
				countAtThisCost++
			}
		}
		if countAtThisCost > 0 {
			costIcon := getCostIcon(cost)
			fmt.Printf("   %s Cost %d: %d agents\n", costIcon, cost, countAtThisCost)
		}
	}
	
	fmt.Printf("\n🔧 Tool Distribution by Cost:\n")
	for cost := 0; cost <= 8; cost++ {
		if cost == 4 || cost == 6 || cost == 7 { continue } // Skip invalid Fibonacci values
		tools := componentRegistry.GetToolsByCost(cost)
		var countAtThisCost int
		for _, tool := range tools {
			if tool.CostMagnitude == cost {
				countAtThisCost++
			}
		}
		if countAtThisCost > 0 {
			costIcon := getCostIcon(cost)
			fmt.Printf("   %s Cost %d: %d tools\n", costIcon, cost, countAtThisCost)
		}
	}
	
	// Show what's available within budget
	fmt.Printf("\n💰 Within Budget (%d):\n", budget)
	availableAgents := componentRegistry.GetAgentsByCost(budget)
	availableTools := componentRegistry.GetToolsByCost(budget)
	fmt.Printf("   Agents: %d available\n", len(availableAgents))
	fmt.Printf("   Tools: %d available\n", len(availableTools))
	
	fmt.Printf("\n💡 Try these commands:\n")
	fmt.Printf("   guild cost-demo --list-agents --max-cost %d\n", budget)
	fmt.Printf("   guild cost-demo --capability coding --budget %d\n", budget)
	fmt.Printf("   guild cost-demo --simulate-task review --budget %d\n", budget)
	
	return nil
}

// Helper functions
func getCostIcon(cost int) string {
	switch cost {
	case 0: return "🆓" // Free
	case 1: return "💚" // Cheap
	case 2: return "💛" // Low-mid
	case 3: return "🧡" // Mid
	case 5: return "❤️"  // High
	case 8: return "💜" // Premium
	default: return "❓"
	}
}

func hasCapability(capabilities []string, target string) bool {
	for _, cap := range capabilities {
		if cap == target {
			return true
		}
	}
	return false
}

// DemoTool is a simple tool implementation for demonstration
type DemoTool struct {
	name string
}

func (d *DemoTool) Name() string        { return d.name }
func (d *DemoTool) Description() string { return fmt.Sprintf("Demo %s tool", d.name) }
func (d *DemoTool) Category() string    { return "demo" }
func (d *DemoTool) Schema() map[string]interface{} {
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
func (d *DemoTool) Execute(ctx context.Context, input string) (*tools.ToolResult, error) {
	return &tools.ToolResult{
		Output:  fmt.Sprintf("Demo result from %s", d.name),
		Success: true,
	}, nil
}
func (d *DemoTool) Examples() []string    { return []string{fmt.Sprintf("example for %s", d.name)} }
func (d *DemoTool) RequiresAuth() bool    { return false }