package main

import (
	"context"
	"fmt"
	"log"
	
	"github.com/blockhead-consulting/guild/pkg/agent"
	"github.com/blockhead-consulting/guild/pkg/memory"
	"github.com/blockhead-consulting/guild/pkg/memory/rag"
	"github.com/blockhead-consulting/guild/pkg/objective"
	"github.com/blockhead-consulting/guild/pkg/tools"
	"github.com/blockhead-consulting/guild/pkg/providers/openai"
)

// Example of setting up cost-aware agents with budgets
func main() {
	// Initialize components
	ctx := context.Background()
	
	// Create LLM client
	llmClient, err := openai.NewClient(openai.Config{
		APIKey: "your-api-key",
		Model:  "gpt-3.5-turbo",
	})
	if err != nil {
		log.Fatal(err)
	}
	
	// Create memory manager (simplified for example)
	memoryStore := memory.NewBoltStore("data/memory.db")
	memoryManager := memory.NewBoltChainManager(memoryStore)
	
	// Create tool registry with costs
	toolRegistry := tools.NewToolRegistry()
	
	// Register tools with specific costs
	shellTool := shell.NewShellTool()
	toolRegistry.RegisterToolWithCost(shellTool, 0.01) // $0.01 per shell command
	
	fileTool := fs.NewFileTool()
	toolRegistry.RegisterToolWithCost(fileTool, 0.001) // $0.001 per file operation
	
	httpTool := http.NewHTTPTool()
	toolRegistry.RegisterToolWithCost(httpTool, 0.05) // $0.05 per HTTP request
	
	// Create objective manager
	objectiveManager := objective.NewManager(memoryStore)
	
	// Create cost-aware worker agent
	workerAgent := agent.NewWorkerAgent(
		"worker-1",
		"Research Assistant",
		llmClient,
		memoryManager,
		toolRegistry,
		objectiveManager,
	)
	
	// Set budgets for the worker agent
	workerAgent.SetCostBudget(agent.CostTypeLLM, 5.00)    // $5 for LLM usage
	workerAgent.SetCostBudget(agent.CostTypeTool, 1.00)   // $1 for tool usage
	workerAgent.SetCostBudget(agent.CostTypeStorage, 0.50) // $0.50 for storage
	
	// Create manager agent with higher budgets
	managerAgent := agent.NewManagerAgent(
		"manager-1",
		"Project Manager",
		llmClient,
		memoryManager,
		toolRegistry,
		objectiveManager,
	)
	
	// Set manager budgets
	managerAgent.SetCostBudget(agent.CostTypeLLM, 20.00)   // $20 for coordination
	managerAgent.SetCostBudget(agent.CostTypeTool, 2.00)   // $2 for tools
	
	// Example: Execute task with cost awareness
	request := "Research the latest trends in AI and summarize in 3 bullet points"
	
	// Check if we can afford the request
	estimatedCost := estimateLLMCost(request)
	currentCosts := workerAgent.GetCurrentCosts()
	
	fmt.Printf("Current LLM spend: $%.4f\n", currentCosts["llm"])
	fmt.Printf("Estimated cost for request: $%.4f\n", estimatedCost)
	
	// Execute only if within budget
	if currentCosts["llm"]+estimatedCost <= 5.00 {
		response, err := workerAgent.CostAwareExecute(ctx, request)
		if err != nil {
			log.Printf("Execution failed: %v", err)
		} else {
			fmt.Printf("Response: %s\n", response)
		}
	} else {
		fmt.Println("Request would exceed budget, using fallback approach...")
		// Use cheaper alternative or cached response
	}
	
	// Get cost report
	costReport := workerAgent.GetCostReport()
	printCostReport(costReport)
	
	// Example: Dynamic budget allocation
	dynamicBudgetAllocation(managerAgent, workerAgent)
	
	// Example: Cost optimization with RAG
	demonstrateCostOptimizedRAG(ctx, workerAgent)
}

// estimateLLMCost provides a rough cost estimate for a request
func estimateLLMCost(request string) float64 {
	// Rough estimation: 1 character ≈ 0.25 tokens
	promptTokens := len(request) / 4
	estimatedCompletionTokens := 200 // Assume moderate response
	
	// GPT-3.5-turbo pricing: $0.50 per 1M prompt tokens
	costPerMillionTokens := 0.50
	
	totalTokens := promptTokens + estimatedCompletionTokens
	return float64(totalTokens) * costPerMillionTokens / 1_000_000
}

// printCostReport displays a formatted cost report
func printCostReport(report map[string]interface{}) {
	fmt.Println("\n=== Cost Report ===")
	
	if totalCosts, ok := report["total_costs"].(map[string]float64); ok {
		for costType, amount := range totalCosts {
			fmt.Printf("%s: $%.6f\n", costType, amount)
		}
	}
	
	if budgets, ok := report["budgets"].(map[string]float64); ok {
		fmt.Println("\n=== Budgets ===")
		for costType, budget := range budgets {
			spent := 0.0
			if totalCosts, ok := report["total_costs"].(map[string]float64); ok {
				spent = totalCosts[costType]
			}
			percentage := (spent / budget) * 100
			fmt.Printf("%s: $%.2f (%.1f%% used)\n", costType, budget, percentage)
		}
	}
	
	fmt.Printf("\nGrand Total: $%.6f\n", report["grand_total"])
}

// dynamicBudgetAllocation demonstrates dynamic budget allocation between agents
func dynamicBudgetAllocation(manager *agent.ManagerAgent, worker *agent.WorkerAgent) {
	fmt.Println("\n=== Dynamic Budget Allocation ===")
	
	// Check worker's remaining budget
	workerReport := worker.GetCostReport()
	workerLLMSpent := 0.0
	if costs, ok := workerReport["total_costs"].(map[string]float64); ok {
		workerLLMSpent = costs["llm"]
	}
	
	workerLLMBudget := 5.00 // Original budget
	remaining := workerLLMBudget - workerLLMSpent
	
	// If worker is running low on budget, manager can allocate more
	if remaining < 1.00 {
		fmt.Printf("Worker running low on budget (%.2f remaining)\n", remaining)
		
		// Manager allocates additional budget
		additionalBudget := 2.00
		newBudget := workerLLMBudget + additionalBudget
		worker.SetCostBudget(agent.CostTypeLLM, newBudget)
		
		fmt.Printf("Manager allocated additional $%.2f to worker\n", additionalBudget)
		fmt.Printf("Worker's new LLM budget: $%.2f\n", newBudget)
	}
}

// demonstrateCostOptimizedRAG shows how RAG can reduce costs
func demonstrateCostOptimizedRAG(ctx context.Context, worker *agent.WorkerAgent) {
	fmt.Println("\n=== Cost-Optimized RAG Example ===")
	
	// Create a simple retriever (mock for demo)
	retriever, _ := rag.NewRetriever(ctx, nil, rag.Config{
		CollectionName: "knowledge_base",
		MaxResults:     3,
	})
	
	// Wrap agent with RAG capabilities
	ragAgent := rag.NewAgentWrapper(worker, retriever, rag.Config{
		MaxResults: 3,
		ChunkSize:  500,
	})
	
	// Compare costs: with and without RAG
	query := "What are the company's policies on remote work?"
	
	// Without RAG: full context needed
	fullContextQuery := query + " (Note: Include all details about eligibility, equipment, hours, etc.)"
	costWithoutRAG := estimateLLMCost(fullContextQuery)
	
	// With RAG: shorter query, context provided by retrieval
	costWithRAG := estimateLLMCost(query) * 0.7 // Assume 30% reduction due to retrieved context
	
	fmt.Printf("Cost without RAG: $%.6f\n", costWithoutRAG)
	fmt.Printf("Cost with RAG: $%.6f\n", costWithRAG)
	fmt.Printf("Savings: $%.6f (%.1f%%)\n", 
		costWithoutRAG-costWithRAG, 
		((costWithoutRAG-costWithRAG)/costWithoutRAG)*100)
}

// Example of cost-aware error handling
func costAwareErrorHandling(agent *agent.WorkerAgent, request string) {
	ctx := context.Background()
	
	// Try with expensive model first
	response, err := agent.CostAwareExecute(ctx, request)
	
	if err != nil {
		if strings.Contains(err.Error(), "budget exceeded") {
			fmt.Println("Budget exceeded, trying with cheaper alternative...")
			
			// Switch to cheaper model or use cached response
			// This would require modifying the agent's LLM client
			fmt.Println("Using cached response or cheaper model...")
		} else {
			log.Printf("Execution error: %v", err)
		}
	} else {
		fmt.Printf("Success: %s\n", response)
	}
}