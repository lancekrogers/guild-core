# 🚀 Implementing Agent Orchestration

A practical guide to implementing multi-agent orchestration in the Guild framework.

## Quick Start

### 1. Basic Setup

```go
package main

import (
    "context"
    "github.com/blockhead-consulting/guild/pkg/agent"
    "github.com/blockhead-consulting/guild/pkg/orchestrator"
)

func main() {
    ctx := context.Background()
    
    // Create event bus
    eventBus := orchestrator.NewEventBus()
    
    // Create dispatcher
    dispatcher := orchestrator.NewTaskDispatcher(kanbanManager, eventBus)
    
    // Configure orchestrator
    config := &orchestrator.Config{
        MaxConcurrentAgents: 3,
        ExecutionMode:       "parallel",
    }
    
    // Create orchestrator
    orch := orchestrator.NewOrchestrator(config, dispatcher, eventBus)
    
    // Add agents
    orch.AddAgent(researchAgent)
    orch.AddAgent(developerAgent)
    
    // Start orchestration
    orch.Start(ctx)
}
```

### 2. Creating Cost-Aware Agents

```go
func createCostAwareAgent(name string, budgets map[agent.CostType]float64) *agent.WorkerAgent {
    agent := agent.NewWorkerAgent(
        generateID(),
        name,
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

// Example usage
researchAgent := createCostAwareAgent("Research Analyst", map[agent.CostType]float64{
    agent.CostTypeLLM:  5.00,  // $5 for LLM
    agent.CostTypeTool: 1.00,  // $1 for tools
})
```

### 3. Implementing Event Handlers

```go
func setupEventHandlers(eventBus *orchestrator.EventBus) {
    // Task events
    eventBus.Subscribe(orchestrator.EventTaskAssigned, func(event orchestrator.Event) {
        log.Printf("Task assigned: %v", event.Data)
    })
    
    eventBus.Subscribe(orchestrator.EventTaskCompleted, func(event orchestrator.Event) {
        taskID := event.Data.(string)
        updateObjectiveProgress(taskID)
    })
    
    // Agent events
    eventBus.Subscribe(orchestrator.EventAgentError, func(event orchestrator.Event) {
        handleAgentError(event)
    })
    
    // Cost events
    eventBus.Subscribe("cost.threshold.warning", func(event orchestrator.Event) {
        notifyBudgetWarning(event)
    })
}
```

## Common Orchestration Scenarios

### 1. Research and Development Workflow

```go
func orchestrateRnDWorkflow(ctx context.Context, topic string) error {
    // Phase 1: Research
    researchObjective := &objective.Objective{
        Title:       fmt.Sprintf("Research %s", topic),
        Description: "Gather information and analyze current state",
        Tasks: []*objective.ObjectiveTask{
            {Title: "Literature review"},
            {Title: "Market analysis"},
            {Title: "Technical feasibility"},
        },
    }
    
    // Phase 2: Design
    designObjective := &objective.Objective{
        Title:       fmt.Sprintf("Design %s solution", topic),
        Description: "Create architecture and specifications",
        Tasks: []*objective.ObjectiveTask{
            {Title: "System architecture"},
            {Title: "API design"},
            {Title: "Database schema"},
        },
    }
    
    // Phase 3: Implementation
    implementationObjective := &objective.Objective{
        Title:       fmt.Sprintf("Implement %s", topic),
        Description: "Build the solution",
        Tasks: []*objective.ObjectiveTask{
            {Title: "Core functionality"},
            {Title: "Integration"},
            {Title: "Testing"},
        },
    }
    
    // Create pipeline
    pipeline := []struct {
        objective *objective.Objective
        agents    []string
    }{
        {researchObjective, []string{"research-analyst"}},
        {designObjective, []string{"architect", "developer"}},
        {implementationObjective, []string{"developer", "qa-engineer"}},
    }
    
    // Execute pipeline
    for _, phase := range pipeline {
        if err := executePhase(ctx, phase.objective, phase.agents); err != nil {
            return fmt.Errorf("phase %s failed: %w", phase.objective.Title, err)
        }
    }
    
    return nil
}
```

### 2. Document Processing Pipeline

```go
func orchestrateDocumentPipeline(ctx context.Context, documents []string) {
    // Stage 1: Extract
    extractTasks := createExtractionTasks(documents)
    extractAgent := agents["data-extractor"]
    
    // Stage 2: Transform
    transformTasks := make([]*kanban.Task, 0)
    transformAgent := agents["data-transformer"]
    
    // Stage 3: Analyze
    analysisTasks := make([]*kanban.Task, 0)
    analysisAgent := agents["data-analyst"]
    
    // Stage 4: Report
    reportTask := createReportTask()
    reportAgent := agents["report-writer"]
    
    // Execute pipeline with dependencies
    pipeline := orchestrator.Pipeline{
        Stages: []orchestrator.Stage{
            {Name: "Extract", Agent: extractAgent, Tasks: extractTasks},
            {Name: "Transform", Agent: transformAgent, Tasks: transformTasks},
            {Name: "Analyze", Agent: analysisAgent, Tasks: analysisTasks},
            {Name: "Report", Agent: reportAgent, Tasks: []*kanban.Task{reportTask}},
        },
    }
    
    // Execute with monitoring
    executePipelineWithMonitoring(ctx, pipeline)
}
```

### 3. Customer Support Automation

```go
func orchestrateSupportTicket(ctx context.Context, ticket SupportTicket) {
    // Create specialized agents for support
    agents := map[string]*agent.WorkerAgent{
        "classifier": createSupportAgent("Ticket Classifier"),
        "researcher": createSupportAgent("Knowledge Base Researcher"),
        "responder":  createSupportAgent("Response Generator"),
        "escalator":  createSupportAgent("Escalation Manager"),
    }
    
    // Step 1: Classify ticket
    classification, _ := agents["classifier"].Execute(ctx, 
        fmt.Sprintf("Classify ticket: %s", ticket.Description))
    
    // Step 2: Research solution
    if classification.Severity == "low" {
        solution, _ := agents["researcher"].Execute(ctx,
            fmt.Sprintf("Find solution for: %s", ticket.Issue))
        
        // Step 3: Generate response
        response, _ := agents["responder"].Execute(ctx,
            fmt.Sprintf("Create response using: %s", solution))
        
        sendResponse(ticket, response)
    } else {
        // Escalate to human
        agents["escalator"].Execute(ctx,
            fmt.Sprintf("Escalate ticket: %s", ticket.ID))
    }
}
```

## Advanced Patterns

### 1. Adaptive Orchestration

```go
type AdaptiveOrchestrator struct {
    *orchestrator.BaseOrchestrator
    metrics    *MetricsCollector
    strategies map[string]OrchestrationStrategy
}

func (ao *AdaptiveOrchestrator) Execute(ctx context.Context, objective *objective.Objective) error {
    // Select strategy based on objective characteristics
    strategy := ao.selectStrategy(objective)
    
    // Monitor execution
    ao.metrics.StartTracking(objective.ID)
    defer ao.metrics.StopTracking(objective.ID)
    
    // Execute with selected strategy
    result := strategy.Execute(ctx, objective, ao.agents)
    
    // Adapt based on results
    ao.adaptStrategy(strategy, result)
    
    return result.Error
}

func (ao *AdaptiveOrchestrator) selectStrategy(obj *objective.Objective) OrchestrationStrategy {
    // Select based on objective properties
    switch {
    case obj.Priority == "critical":
        return ao.strategies["parallel-fast"]
    case len(obj.Tasks) > 10:
        return ao.strategies["distributed"]
    case obj.Metadata["cost_sensitive"] == "true":
        return ao.strategies["cost-optimized"]
    default:
        return ao.strategies["balanced"]
    }
}
```

### 2. Self-Healing Orchestration

```go
type SelfHealingOrchestrator struct {
    *orchestrator.BaseOrchestrator
    healthChecker *HealthChecker
    recovery      *RecoveryManager
}

func (sho *SelfHealingOrchestrator) monitorHealth(ctx context.Context) {
    ticker := time.NewTicker(30 * time.Second)
    defer ticker.Stop()
    
    for {
        select {
        case <-ticker.C:
            // Check agent health
            for agentID, agent := range sho.agents {
                health := sho.healthChecker.CheckAgent(agent)
                
                if !health.IsHealthy {
                    log.Printf("Agent %s unhealthy: %v", agentID, health.Issues)
                    
                    // Attempt recovery
                    if err := sho.recovery.RecoverAgent(agent); err != nil {
                        // Replace agent
                        newAgent := sho.createReplacementAgent(agent)
                        sho.ReplaceAgent(agentID, newAgent)
                    }
                }
            }
            
        case <-ctx.Done():
            return
        }
    }
}
```

### 3. Cost-Optimized Orchestration

```go
type CostOptimizedOrchestrator struct {
    *orchestrator.BaseOrchestrator
    costPredictor *CostPredictor
    budgetManager *BudgetManager
}

func (coo *CostOptimizedOrchestrator) AssignTask(ctx context.Context, task *kanban.Task) error {
    // Get available agents
    availableAgents := coo.getAvailableAgents()
    
    // Predict cost for each agent
    predictions := make(map[string]float64)
    for _, agent := range availableAgents {
        cost := coo.costPredictor.PredictTaskCost(agent, task)
        predictions[agent.GetID()] = cost
    }
    
    // Select most cost-effective agent within budget
    selectedAgent := coo.selectCostEffectiveAgent(predictions, task)
    
    if selectedAgent == nil {
        return fmt.Errorf("no agent available within budget for task %s", task.ID)
    }
    
    // Reserve budget
    if err := coo.budgetManager.ReserveBudget(selectedAgent.GetID(), predictions[selectedAgent.GetID()]); err != nil {
        return err
    }
    
    // Assign task
    return coo.dispatcher.AssignTask(ctx, task.ID, selectedAgent.GetID(), "cost-optimizer", "Most cost-effective")
}
```

## Testing Orchestration

### 1. Unit Testing Orchestration Components

```go
func TestOrchestrator_TaskAssignment(t *testing.T) {
    // Create mocks
    mockAgent := &MockAgent{
        id:   "test-agent",
        name: "Test Agent",
    }
    
    mockDispatcher := &MockDispatcher{
        assignFunc: func(ctx context.Context, taskID, agentID, assignerID, reason string) error {
            if agentID != mockAgent.id {
                t.Errorf("Expected agent ID %s, got %s", mockAgent.id, agentID)
            }
            return nil
        },
    }
    
    // Create orchestrator
    orch := orchestrator.NewOrchestrator(config, mockDispatcher, eventBus)
    orch.AddAgent(mockAgent)
    
    // Test task assignment
    task := &kanban.Task{ID: "test-task"}
    err := orch.AssignTask(context.Background(), task)
    
    assert.NoError(t, err)
}
```

### 2. Integration Testing Pipelines

```go
func TestPipeline_EndToEnd(t *testing.T) {
    ctx := context.Background()
    
    // Setup test infrastructure
    infra := setupTestInfrastructure(t)
    defer infra.Cleanup()
    
    // Create test agents
    agents := createTestAgents(infra)
    
    // Create test pipeline
    pipeline := createTestPipeline(agents)
    
    // Execute pipeline
    err := pipeline.Execute(ctx, testObjective)
    assert.NoError(t, err)
    
    // Verify results
    verifyPipelineResults(t, pipeline)
    
    // Check costs
    totalCost := calculateTotalCost(agents)
    assert.Less(t, totalCost, 10.0, "Pipeline should cost less than $10")
}
```

### 3. Simulating Failures

```go
func TestOrchestrator_FailureRecovery(t *testing.T) {
    // Create agent that fails on specific tasks
    failingAgent := &FailingAgent{
        WorkerAgent:  *createTestAgent(),
        failureCount: 0,
        maxFailures:  2,
    }
    
    // Setup orchestrator with recovery
    orch := createOrchestratorWithRecovery()
    orch.AddAgent(failingAgent)
    
    // Execute tasks
    var wg sync.WaitGroup
    for i := 0; i < 5; i++ {
        wg.Add(1)
        go func(taskNum int) {
            defer wg.Done()
            
            task := createTestTask(taskNum)
            err := orch.ExecuteTask(context.Background(), task)
            
            // Should eventually succeed after retries
            assert.NoError(t, err)
        }(i)
    }
    
    wg.Wait()
    
    // Verify recovery happened
    assert.Greater(t, failingAgent.failureCount, 0)
    assert.Equal(t, 5, failingAgent.successCount)
}
```

## Performance Optimization

### 1. Caching and Memoization

```go
type CachedOrchestrator struct {
    *orchestrator.BaseOrchestrator
    cache *ResultCache
}

func (co *CachedOrchestrator) Execute(ctx context.Context, request string) (string, error) {
    // Check cache
    cacheKey := generateCacheKey(request)
    if result, found := co.cache.Get(cacheKey); found {
        return result, nil
    }
    
    // Execute normally
    result, err := co.BaseOrchestrator.Execute(ctx, request)
    if err != nil {
        return "", err
    }
    
    // Cache result
    co.cache.Set(cacheKey, result, 1*time.Hour)
    
    return result, nil
}
```

### 2. Batch Processing

```go
func (o *Orchestrator) ProcessBatch(ctx context.Context, requests []Request) []Result {
    // Group by agent capability
    groups := groupRequestsByCapability(requests)
    
    results := make([]Result, len(requests))
    var wg sync.WaitGroup
    
    // Process each group in parallel
    for capability, groupRequests := range groups {
        agent := o.selectAgentForCapability(capability)
        
        wg.Add(1)
        go func(a Agent, reqs []Request) {
            defer wg.Done()
            
            // Batch process with single agent
            batchResults := a.ProcessBatch(ctx, reqs)
            
            // Map results back
            for i, req := range reqs {
                results[req.Index] = batchResults[i]
            }
        }(agent, groupRequests)
    }
    
    wg.Wait()
    return results
}
```

## Monitoring and Debugging

### 1. Orchestration Dashboard

```go
func createOrchestrationDashboard() *Dashboard {
    dashboard := &Dashboard{
        Metrics: []Metric{
            &AgentUtilizationMetric{},
            &TaskThroughputMetric{},
            &CostPerTaskMetric{},
            &ErrorRateMetric{},
        },
        RefreshInterval: 5 * time.Second,
    }
    
    // Add visualizations
    dashboard.AddVisualization("agent-timeline", &TimelineVisualization{})
    dashboard.AddVisualization("cost-breakdown", &CostBreakdownChart{})
    dashboard.AddVisualization("task-flow", &TaskFlowDiagram{})
    
    return dashboard
}
```

### 2. Debug Logging

```go
func (o *Orchestrator) enableDebugLogging() {
    o.eventBus.Subscribe("*", func(event orchestrator.Event) {
        log.Printf("[DEBUG] Event: %s, Source: %s, Data: %v", 
            event.Type, event.Source, event.Data)
    })
    
    // Add request/response logging
    o.AddMiddleware(func(next HandlerFunc) HandlerFunc {
        return func(ctx context.Context, req Request) (Response, error) {
            start := time.Now()
            log.Printf("[DEBUG] Request: %v", req)
            
            resp, err := next(ctx, req)
            
            log.Printf("[DEBUG] Response: %v, Error: %v, Duration: %v", 
                resp, err, time.Since(start))
            
            return resp, err
        }
    })
}
```

## Best Practices Checklist

- [ ] Define clear objectives and tasks
- [ ] Set appropriate budgets for all agents
- [ ] Implement proper error handling
- [ ] Use timeouts for all operations
- [ ] Monitor costs continuously
- [ ] Log important events
- [ ] Test failure scenarios
- [ ] Document orchestration flows
- [ ] Use caching where appropriate
- [ ] Implement circuit breakers
- [ ] Plan for scaling
- [ ] Set up monitoring dashboards

## Conclusion

Implementing agent orchestration in the Guild framework enables powerful multi-agent workflows while maintaining cost control and observability. Start with simple patterns and gradually add complexity as your use cases evolve.