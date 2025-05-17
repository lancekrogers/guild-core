package main

import (
	"context"
	"fmt"
	"log"
	"time"
	
	"github.com/blockhead-consulting/guild/pkg/agent"
	"github.com/blockhead-consulting/guild/pkg/kanban"
	"github.com/blockhead-consulting/guild/pkg/memory"
	"github.com/blockhead-consulting/guild/pkg/objective"
	"github.com/blockhead-consulting/guild/pkg/orchestrator"
	"github.com/blockhead-consulting/guild/pkg/providers/openai"
	"github.com/blockhead-consulting/guild/pkg/tools"
)

// SimpleOrchestrationExample demonstrates basic agent orchestration
func main() {
	ctx := context.Background()
	
	// Initialize components
	store := memory.NewBoltStore("data/orchestration.db")
	defer store.Close()
	
	memoryManager := memory.NewBoltChainManager(store)
	toolRegistry := tools.NewToolRegistry()
	objectiveManager := objective.NewManager(store)
	kanbanManager := kanban.NewManager(store)
	
	// Create LLM client
	llmClient, _ := openai.NewClient(openai.Config{
		APIKey: "your-api-key",
		Model:  "gpt-3.5-turbo",
	})
	
	// Create agents with cost tracking
	researchAgent := createCostAwareAgent(
		"research-1",
		"Research Assistant",
		llmClient,
		memoryManager,
		toolRegistry,
		objectiveManager,
		map[agent.CostType]float64{
			agent.CostTypeLLM:  5.00,
			agent.CostTypeTool: 1.00,
		},
	)
	
	developerAgent := createCostAwareAgent(
		"dev-1",
		"Senior Developer",
		llmClient,
		memoryManager,
		toolRegistry,
		objectiveManager,
		map[agent.CostType]float64{
			agent.CostTypeLLM:  10.00,
			agent.CostTypeTool: 3.00,
		},
	)
	
	// Create event bus and dispatcher
	eventBus := orchestrator.NewEventBus()
	dispatcher := orchestrator.NewTaskDispatcher(kanbanManager, eventBus)
	
	// Create orchestrator
	config := &orchestrator.Config{
		MaxConcurrentAgents: 3,
		ManagerAgentID:      "manager-1",
		KanbanBoardID:       "main-board",
		ExecutionMode:       "parallel",
	}
	
	orchestratorInstance := orchestrator.NewOrchestrator(config, dispatcher, eventBus)
	
	// Add event handlers
	setupEventHandlers(eventBus)
	
	// Add agents to orchestrator
	orchestratorInstance.AddAgent(researchAgent)
	orchestratorInstance.AddAgent(developerAgent)
	
	// Create and set objective
	objective := createWebsiteObjective()
	objectiveManager.SaveObjective(ctx, objective)
	orchestratorInstance.SetObjective(objective)
	
	// Start orchestration
	fmt.Println("Starting orchestration...")
	if err := orchestratorInstance.Start(ctx); err != nil {
		log.Fatal(err)
	}
	
	// Create tasks on the kanban board
	board, _ := kanbanManager.GetBoard(ctx, "main-board")
	if board == nil {
		board, _ = kanbanManager.CreateBoard(ctx, "Main Board", "Primary task board")
	}
	
	// Create research task
	researchTask, _ := board.CreateTask(ctx, 
		"Research modern web frameworks",
		"Research and compare React, Vue, and Angular for our project needs",
	)
	researchTask.AssignedTo = "research-1"
	researchTask.Priority = kanban.TaskPriorityHigh
	board.UpdateTask(ctx, researchTask)
	
	// Create development task
	devTask, _ := board.CreateTask(ctx,
		"Implement authentication system",
		"Create secure JWT-based authentication with refresh tokens",
	)
	devTask.AssignedTo = "dev-1"
	devTask.Priority = kanban.TaskPriorityMedium
	devTask.Dependencies = []string{researchTask.ID} // Depends on research
	board.UpdateTask(ctx, devTask)
	
	// Monitor orchestration
	monitorOrchestration(orchestratorInstance, eventBus)
	
	// Let it run for a while
	time.Sleep(30 * time.Second)
	
	// Generate cost report
	generateCostReport(researchAgent, developerAgent)
	
	// Stop orchestration
	fmt.Println("Stopping orchestration...")
	orchestratorInstance.Stop(ctx)
}

// createCostAwareAgent creates an agent with cost tracking
func createCostAwareAgent(
	id, name string,
	llmClient interfaces.LLMClient,
	memoryManager memory.ChainManager,
	toolRegistry *tools.ToolRegistry,
	objectiveManager *objective.Manager,
	budgets map[agent.CostType]float64,
) *agent.WorkerAgent {
	
	agent := agent.NewWorkerAgent(
		id, name,
		llmClient,
		memoryManager,
		toolRegistry,
		objectiveManager,
	)
	
	// Set budgets
	for costType, budget := range budgets {
		agent.SetCostBudget(costType, budget)
	}
	
	return agent
}

// createWebsiteObjective creates a sample objective
func createWebsiteObjective() *objective.Objective {
	return &objective.Objective{
		ID:          "website-obj-1",
		Title:       "Build Modern Web Application",
		Description: "Create a responsive web application with authentication and real-time features",
		Status:      objective.ObjectiveStatusActive,
		Owner:       "product-owner",
		Priority:    "high",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Requirements: []string{
			"User authentication with JWT",
			"Real-time notifications",
			"Mobile-responsive design",
			"RESTful API integration",
			"Performance optimization",
		},
		Tasks: []*objective.ObjectiveTask{
			{
				ID:          "task-1",
				Title:       "Research frameworks",
				Description: "Evaluate modern web frameworks",
				Status:      "in_progress",
				Assignee:    "research-1",
			},
			{
				ID:          "task-2",
				Title:       "Design architecture",
				Description: "Create system architecture",
				Status:      "todo",
				Dependencies: []string{"task-1"},
			},
			{
				ID:          "task-3",
				Title:       "Implement auth",
				Description: "Build authentication system",
				Status:      "todo",
				Assignee:    "dev-1",
				Dependencies: []string{"task-2"},
			},
		},
	}
}

// setupEventHandlers configures event handlers for monitoring
func setupEventHandlers(eventBus *orchestrator.EventBus) {
	// Agent events
	eventBus.Subscribe(orchestrator.EventAgentStarted, func(event orchestrator.Event) {
		fmt.Printf("🚀 Agent started: %v\n", event.Data)
	})
	
	eventBus.Subscribe(orchestrator.EventAgentCompleted, func(event orchestrator.Event) {
		fmt.Printf("✅ Agent completed: %v\n", event.Data)
	})
	
	eventBus.Subscribe(orchestrator.EventAgentError, func(event orchestrator.Event) {
		fmt.Printf("❌ Agent error: %v\n", event.Data)
	})
	
	// Task events
	eventBus.Subscribe(orchestrator.EventTaskAssigned, func(event orchestrator.Event) {
		fmt.Printf("📋 Task assigned: %v\n", event.Data)
	})
	
	eventBus.Subscribe(orchestrator.EventTaskCompleted, func(event orchestrator.Event) {
		fmt.Printf("✅ Task completed: %v\n", event.Data)
	})
	
	// Orchestrator events
	eventBus.Subscribe(orchestrator.EventOrchestratorStarted, func(event orchestrator.Event) {
		fmt.Printf("🎭 Orchestrator started\n")
	})
	
	eventBus.Subscribe(orchestrator.EventOrchestratorStopped, func(event orchestrator.Event) {
		fmt.Printf("🛑 Orchestrator stopped\n")
	})
}

// monitorOrchestration monitors the orchestration progress
func monitorOrchestration(orchestrator orchestrator.Orchestrator, eventBus *orchestrator.EventBus) {
	// Create a monitoring goroutine
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		
		for {
			select {
			case <-ticker.C:
				status := orchestrator.Status()
				objective := orchestrator.GetObjective()
				
				fmt.Printf("\n📊 Orchestration Status: %s\n", status)
				if objective != nil {
					completion := objective.Completion * 100
					fmt.Printf("   Objective: %s (%.1f%% complete)\n", 
						objective.Title, completion)
				}
			}
		}
	}()
}

// generateCostReport generates a cost report for all agents
func generateCostReport(agents ...*agent.WorkerAgent) {
	fmt.Println("\n💰 Cost Report")
	fmt.Println("==============")
	
	var totalCost float64
	
	for _, agent := range agents {
		report := agent.GetCostReport()
		fmt.Printf("\n%s (%s):\n", agent.GetName(), agent.GetID())
		
		if costs, ok := report["total_costs"].(map[string]float64); ok {
			for costType, amount := range costs {
				fmt.Printf("  %s: $%.4f\n", costType, amount)
				totalCost += amount
			}
		}
		
		if budgets, ok := report["budgets"].(map[string]float64); ok {
			fmt.Println("  Budgets:")
			for costType, budget := range budgets {
				spent := 0.0
				if costs, ok := report["total_costs"].(map[string]float64); ok {
					spent = costs[costType]
				}
				percentage := (spent / budget) * 100
				fmt.Printf("    %s: $%.2f (%.1f%% used)\n", 
					costType, budget, percentage)
			}
		}
	}
	
	fmt.Printf("\nTotal Cost: $%.4f\n", totalCost)
}