package main

import (
	"context"
	"fmt"
	"sync"
	"time"
	
	"github.com/guild-ventures/guild-core/pkg/agent"
	"github.com/guild-ventures/guild-core/pkg/orchestrator"
)

// OrchestrationPatterns demonstrates common orchestration patterns

// Pattern 1: Pipeline Pattern - Sequential execution with data flow
func PipelinePattern(ctx context.Context, agents []*agent.WorkerAgent) {
	fmt.Println("🔄 Pipeline Pattern")
	
	// Each agent processes data and passes to the next
	var result string
	for i, agent := range agents {
		input := result
		if i == 0 {
			input = "Initial data"
		}
		
		// Execute agent
		result, err := agent.Execute(ctx, input)
		if err != nil {
			fmt.Printf("Pipeline failed at stage %d: %v\n", i, err)
			return
		}
		
		fmt.Printf("Stage %d complete: %s\n", i+1, agent.GetName())
	}
	
	fmt.Printf("Pipeline result: %s\n", result)
}

// Pattern 2: Fan-Out/Fan-In - Parallel execution with result aggregation
func FanOutFanInPattern(ctx context.Context, coordinator *agent.WorkerAgent, workers []*agent.WorkerAgent) {
	fmt.Println("🌐 Fan-Out/Fan-In Pattern")
	
	// Fan-out: Distribute work to multiple agents
	var wg sync.WaitGroup
	results := make(chan string, len(workers))
	
	for _, worker := range workers {
		wg.Add(1)
		go func(w *agent.WorkerAgent) {
			defer wg.Done()
			
			// Each worker processes independently
			result, err := w.Execute(ctx, "Process subset of data")
			if err != nil {
				fmt.Printf("Worker %s failed: %v\n", w.GetName(), err)
				return
			}
			
			results <- result
		}(worker)
	}
	
	// Wait for all workers
	go func() {
		wg.Wait()
		close(results)
	}()
	
	// Fan-in: Aggregate results
	var aggregatedResults []string
	for result := range results {
		aggregatedResults = append(aggregatedResults, result)
	}
	
	// Coordinator processes aggregated results
	finalResult, _ := coordinator.Execute(ctx, fmt.Sprintf("Aggregate: %v", aggregatedResults))
	fmt.Printf("Aggregated result: %s\n", finalResult)
}

// Pattern 3: Scatter-Gather - Query multiple sources and combine
func ScatterGatherPattern(ctx context.Context, agents []*agent.WorkerAgent, query string) {
	fmt.Println("📡 Scatter-Gather Pattern")
	
	// Scatter: Send query to all agents
	type Response struct {
		AgentID string
		Result  string
		Error   error
	}
	
	responses := make(chan Response, len(agents))
	
	for _, agent := range agents {
		go func(a *agent.WorkerAgent) {
			result, err := a.Execute(ctx, query)
			responses <- Response{
				AgentID: a.GetID(),
				Result:  result,
				Error:   err,
			}
		}(agent)
	}
	
	// Gather: Collect responses with timeout
	gathered := make([]Response, 0, len(agents))
	timeout := time.After(30 * time.Second)
	
	for i := 0; i < len(agents); i++ {
		select {
		case resp := <-responses:
			if resp.Error == nil {
				gathered = append(gathered, resp)
				fmt.Printf("Received from %s: %s\n", resp.AgentID, resp.Result)
			}
		case <-timeout:
			fmt.Println("Timeout waiting for responses")
			break
		}
	}
	
	fmt.Printf("Gathered %d responses\n", len(gathered))
}

// Pattern 4: Choreography - Agents coordinate through events
func ChoreographyPattern(ctx context.Context, eventBus *orchestrator.EventBus, agents []*agent.WorkerAgent) {
	fmt.Println("💃 Choreography Pattern")
	
	// Each agent listens for events and responds accordingly
	for _, agent := range agents {
		a := agent // Capture for closure
		
		// Agent subscribes to relevant events
		eventBus.Subscribe("task.available", func(event orchestrator.Event) {
			task := event.Data.(string)
			
			// Agent decides if it can handle the task
			if canHandle(a, task) {
				fmt.Printf("%s handling task: %s\n", a.GetName(), task)
				
				// Execute task
				result, _ := a.Execute(ctx, task)
				
				// Emit completion event
				eventBus.Publish(orchestrator.Event{
					Type:   "task.completed",
					Source: a.GetID(),
					Data:   result,
				})
			}
		})
	}
	
	// Emit initial events to start the choreography
	tasks := []string{"Research", "Design", "Implement", "Test", "Deploy"}
	for _, task := range tasks {
		eventBus.Publish(orchestrator.Event{
			Type:   "task.available",
			Source: "system",
			Data:   task,
		})
		
		time.Sleep(1 * time.Second) // Stagger task emissions
	}
}

// Pattern 5: Saga Pattern - Distributed transaction with compensation
type SagaStep struct {
	Execute    func(ctx context.Context) error
	Compensate func(ctx context.Context) error
	Agent      *agent.WorkerAgent
}

func SagaPattern(ctx context.Context, steps []SagaStep) {
	fmt.Println("🔄 Saga Pattern")
	
	completedSteps := make([]SagaStep, 0)
	
	// Execute steps in order
	for i, step := range steps {
		fmt.Printf("Executing step %d with %s\n", i+1, step.Agent.GetName())
		
		if err := step.Execute(ctx); err != nil {
			fmt.Printf("Step %d failed: %v\n", i+1, err)
			
			// Compensate in reverse order
			fmt.Println("Compensating previous steps...")
			for j := len(completedSteps) - 1; j >= 0; j-- {
				if err := completedSteps[j].Compensate(ctx); err != nil {
					fmt.Printf("Compensation failed at step %d: %v\n", j, err)
				}
			}
			return
		}
		
		completedSteps = append(completedSteps, step)
	}
	
	fmt.Println("Saga completed successfully")
}

// Pattern 6: Circuit Breaker - Protect against cascading failures
type CircuitBreaker struct {
	agent         *agent.WorkerAgent
	failureCount  int
	maxFailures   int
	state         string // "closed", "open", "half-open"
	lastFailTime  time.Time
	cooldownTime  time.Duration
	mu            sync.Mutex
}

func (cb *CircuitBreaker) Execute(ctx context.Context, request string) (string, error) {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	
	// Check circuit state
	switch cb.state {
	case "open":
		// Check if cooldown period has passed
		if time.Since(cb.lastFailTime) > cb.cooldownTime {
			cb.state = "half-open"
			cb.failureCount = 0
		} else {
			return "", fmt.Errorf("circuit breaker is open")
		}
	}
	
	// Try to execute
	result, err := cb.agent.Execute(ctx, request)
	
	if err != nil {
		cb.failureCount++
		cb.lastFailTime = time.Now()
		
		if cb.failureCount >= cb.maxFailures {
			cb.state = "open"
			fmt.Printf("Circuit breaker opened for %s\n", cb.agent.GetName())
		}
		
		return "", err
	}
	
	// Success - reset if in half-open state
	if cb.state == "half-open" {
		cb.state = "closed"
		cb.failureCount = 0
		fmt.Printf("Circuit breaker closed for %s\n", cb.agent.GetName())
	}
	
	return result, nil
}

// Pattern 7: Bulkhead - Isolate resources to prevent total failure
func BulkheadPattern(ctx context.Context, agents []*agent.WorkerAgent, maxConcurrent int) {
	fmt.Println("🚢 Bulkhead Pattern")
	
	// Create semaphore for each agent
	semaphores := make(map[string]chan struct{})
	for _, agent := range agents {
		semaphores[agent.GetID()] = make(chan struct{}, maxConcurrent)
	}
	
	// Process requests with resource isolation
	processRequest := func(agent *agent.WorkerAgent, request string) {
		sem := semaphores[agent.GetID()]
		
		// Acquire semaphore
		sem <- struct{}{}
		defer func() { <-sem }()
		
		// Execute with isolated resources
		result, err := agent.Execute(ctx, request)
		if err != nil {
			fmt.Printf("Error in bulkhead %s: %v\n", agent.GetName(), err)
		} else {
			fmt.Printf("Bulkhead %s processed: %s\n", agent.GetName(), result)
		}
	}
	
	// Simulate concurrent requests
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		for _, agent := range agents {
			wg.Add(1)
			go func(a *agent.WorkerAgent, req int) {
				defer wg.Done()
				processRequest(a, fmt.Sprintf("Request %d", req))
			}(agent, i)
		}
	}
	
	wg.Wait()
}

// Pattern 8: Dynamic Routing - Route tasks based on agent capabilities
type RoutingRule struct {
	Condition func(task string) bool
	AgentIDs  []string
}

func DynamicRoutingPattern(ctx context.Context, agents map[string]*agent.WorkerAgent, rules []RoutingRule, tasks []string) {
	fmt.Println("🔀 Dynamic Routing Pattern")
	
	for _, task := range tasks {
		// Find matching rule
		var selectedAgent *agent.WorkerAgent
		
		for _, rule := range rules {
			if rule.Condition(task) {
				// Select agent based on availability and cost
				selectedAgent = selectBestAgent(agents, rule.AgentIDs)
				break
			}
		}
		
		if selectedAgent != nil {
			fmt.Printf("Routing '%s' to %s\n", task, selectedAgent.GetName())
			result, _ := selectedAgent.Execute(ctx, task)
			fmt.Printf("Result: %s\n", result)
		} else {
			fmt.Printf("No suitable agent for task: %s\n", task)
		}
	}
}

// selectBestAgent selects the best agent based on criteria
func selectBestAgent(agents map[string]*agent.WorkerAgent, candidateIDs []string) *agent.WorkerAgent {
	var bestAgent *agent.WorkerAgent
	lowestCost := float64(999999)
	
	for _, id := range candidateIDs {
		if agent, ok := agents[id]; ok {
			// Check agent's current cost
			report := agent.GetCostReport()
			if costs, ok := report["total_costs"].(map[string]float64); ok {
				totalCost := 0.0
				for _, cost := range costs {
					totalCost += cost
				}
				
				if totalCost < lowestCost {
					lowestCost = totalCost
					bestAgent = agent
				}
			}
		}
	}
	
	return bestAgent
}

// Helper function for choreography pattern
func canHandle(agent *agent.WorkerAgent, task string) bool {
	// Simple capability matching
	capabilities := map[string][]string{
		"research-analyst": {"Research", "Analysis"},
		"developer":        {"Implement", "Code"},
		"qa-engineer":      {"Test", "Verify"},
		"devops":           {"Deploy", "Monitor"},
	}
	
	if agentCaps, ok := capabilities[agent.GetID()]; ok {
		for _, cap := range agentCaps {
			if cap == task {
				return true
			}
		}
	}
	
	return false
}

// Pattern 9: Adaptive Load Balancing
func AdaptiveLoadBalancing(ctx context.Context, agents []*agent.WorkerAgent, requests []string) {
	fmt.Println("⚖️ Adaptive Load Balancing")
	
	// Track agent performance metrics
	type AgentMetrics struct {
		ResponseTime   time.Duration
		SuccessRate    float64
		CurrentLoad    int
		CostEfficiency float64
	}
	
	metrics := make(map[string]*AgentMetrics)
	for _, agent := range agents {
		metrics[agent.GetID()] = &AgentMetrics{
			ResponseTime:   0,
			SuccessRate:    1.0,
			CurrentLoad:    0,
			CostEfficiency: 1.0,
		}
	}
	
	// Process requests with adaptive routing
	for _, request := range requests {
		// Select agent based on current metrics
		bestAgent := selectOptimalAgent(agents, metrics)
		
		// Track execution time
		start := time.Now()
		result, err := bestAgent.Execute(ctx, request)
		elapsed := time.Since(start)
		
		// Update metrics
		agentMetrics := metrics[bestAgent.GetID()]
		agentMetrics.ResponseTime = (agentMetrics.ResponseTime + elapsed) / 2
		
		if err != nil {
			agentMetrics.SuccessRate *= 0.9 // Decrease success rate
			fmt.Printf("Error from %s: %v\n", bestAgent.GetName(), err)
		} else {
			agentMetrics.SuccessRate = agentMetrics.SuccessRate*0.9 + 0.1 // Increase success rate
			fmt.Printf("%s completed in %v: %s\n", bestAgent.GetName(), elapsed, result)
		}
		
		// Update cost efficiency
		report := bestAgent.GetCostReport()
		if costs, ok := report["total_costs"].(map[string]float64); ok {
			totalCost := 0.0
			for _, cost := range costs {
				totalCost += cost
			}
			agentMetrics.CostEfficiency = 1.0 / (1.0 + totalCost)
		}
	}
}

// selectOptimalAgent selects the best agent based on multiple factors
func selectOptimalAgent(agents []*agent.WorkerAgent, metrics map[string]*AgentMetrics) *agent.WorkerAgent {
	var bestAgent *agent.WorkerAgent
	bestScore := -1.0
	
	for _, agent := range agents {
		metric := metrics[agent.GetID()]
		
		// Calculate composite score
		score := metric.SuccessRate * metric.CostEfficiency
		if metric.ResponseTime > 0 {
			score /= float64(metric.ResponseTime.Seconds())
		}
		
		if score > bestScore {
			bestScore = score
			bestAgent = agent
		}
	}
	
	return bestAgent
}

// Pattern 10: Consensus-Based Decision Making
func ConsensusPattern(ctx context.Context, agents []*agent.WorkerAgent, question string) {
	fmt.Println("🗳️ Consensus Pattern")
	
	// Collect opinions from all agents
	type Opinion struct {
		Agent  string
		Answer string
		Score  float64
	}
	
	opinions := make([]Opinion, 0, len(agents))
	
	// Get opinion from each agent
	for _, agent := range agents {
		response, err := agent.Execute(ctx, question)
		if err != nil {
			continue
		}
		
		// Extract confidence score (simplified)
		confidence := 0.8 // In reality, parse from response
		
		opinions = append(opinions, Opinion{
			Agent:  agent.GetName(),
			Answer: response,
			Score:  confidence,
		})
		
		fmt.Printf("%s: %s (confidence: %.2f)\n", agent.GetName(), response, confidence)
	}
	
	// Aggregate opinions to reach consensus
	consensus := aggregateOpinions(opinions)
	fmt.Printf("\nConsensus: %s\n", consensus)
}

// aggregateOpinions combines multiple opinions into a consensus
func aggregateOpinions(opinions []Opinion) string {
	// Simple majority voting with confidence weighting
	votes := make(map[string]float64)
	
	for _, opinion := range opinions {
		votes[opinion.Answer] += opinion.Score
	}
	
	// Find the answer with highest weighted votes
	var bestAnswer string
	var bestScore float64
	
	for answer, score := range votes {
		if score > bestScore {
			bestScore = score
			bestAnswer = answer
		}
	}
	
	return bestAnswer
}