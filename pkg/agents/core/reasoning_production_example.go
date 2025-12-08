// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package core

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/guild-framework/guild-core/pkg/config"
	"github.com/guild-framework/guild-core/pkg/observability"
	"github.com/guild-framework/guild-core/pkg/providers"
)

// ProductionReasoningExample demonstrates a complete production setup
func ProductionReasoningExample() {
	// Initialize context with observability
	ctx := context.Background()
	ctx = observability.WithComponent(ctx, "reasoning_production")
	logger := observability.GetLogger(ctx)

	// 1. Initialize reasoning system with SQLite storage
	logger.InfoContext(ctx, "Initializing reasoning system...")

	reasoningConfig := ReasoningSystemConfig{
		ExtractorConfig: ReasoningConfig{
			EnableCaching:      true,
			CacheMaxSize:       10000,
			CacheTTL:           15 * time.Minute,
			MaxReasoningLength: 20000,
			MinConfidence:      0.0,
			MaxConfidence:      1.0,
			StrictValidation:   true,
		},
		StorageConfig: ReasoningStorageConfig{
			RetentionDays:     90,
			MaxChainsPerQuery: 5000,
			EnableCompression: true,
			StorageBackend:    "sqlite",
		},
		EnableAnalytics: true,
		DatabasePath:    "", // Use default .guild/memory.db
	}

	reasoningSystem, err := NewReasoningSystem(ctx, reasoningConfig)
	if err != nil {
		log.Fatalf("Failed to create reasoning system: %v", err)
	}
	defer reasoningSystem.Close(ctx)

	// 2. Start maintenance tasks
	maintenanceCtx, cancelMaintenance := context.WithCancel(ctx)
	defer cancelMaintenance()
	reasoningSystem.StartMaintenance(maintenanceCtx)

	logger.InfoContext(ctx, "Reasoning system initialized",
		"database_path", ".guild/memory.db",
		"retention_days", reasoningConfig.StorageConfig.RetentionDays,
		"analytics_enabled", reasoningConfig.EnableAnalytics)

	// 3. Create agent with reasoning capabilities
	agentConfig := &config.EnhancedAgentConfig{
		ID:           "production_agent",
		Name:         "Production Reasoning Agent",
		Type:         "worker",
		Role:         "General purpose assistant with advanced reasoning",
		Model:        "gpt-4",
		Temperature:  0.7,
		Capabilities: []string{"reasoning", "analysis", "code_generation", "problem_solving"},
		Reasoning: config.ReasoningConfig{
			Enabled:                    true,
			ShowThinking:               true,
			MinConfidenceDisplay:       0.3,
			DeepReasoningMinComplexity: 0.5,
			IncludeInPrompt:            true,
		},
	}

	// In production, these would come from dependency injection
	llmClient := createProductionLLMClient(ctx)

	// Create agent
	agent := &WorkerAgent{
		ID:        agentConfig.ID,
		Name:      agentConfig.Name,
		LLMClient: llmClient,
	}
	agent.SetCapabilities(agentConfig.Capabilities)
	agent.SetDescription(agentConfig.Role)

	// Enhance with reasoning
	err = reasoningSystem.EnhanceAgent(agent)
	if err != nil {
		log.Fatalf("Failed to enhance agent: %v", err)
	}

	logger.InfoContext(ctx, "Created production agent with reasoning",
		"agent_id", agent.ID,
		"capabilities", agent.GetCapabilities())

	// 4. Example: Process tasks with reasoning
	tasks := []struct {
		description string
		complexity  string
	}{
		{
			description: "What is the capital of France?",
			complexity:  "simple",
		},
		{
			description: `Analyze this Go code and suggest improvements:
func getData(ids []int) map[int]string {
    result := make(map[int]string)
    for _, id := range ids {
        data := fetchFromDB(id)
        if data != nil {
            result[id] = data.Value
        }
    }
    return result
}`,
			complexity: "complex",
		},
		{
			description: "Design a microservices architecture for an e-commerce platform",
			complexity:  "very_complex",
		},
	}

	for _, task := range tasks {
		logger.InfoContext(ctx, "Processing task",
			"complexity", task.complexity,
			"description_length", len(task.description))

		// Execute with reasoning
		response, err := agent.ExecuteWithReasoning(ctx, task.description)
		if err != nil {
			logger.WithError(err).ErrorContext(ctx, "Task execution failed")
			continue
		}

		// Log results
		logger.InfoContext(ctx, "Task completed",
			"complexity", task.complexity,
			"confidence", response.Confidence,
			"has_reasoning", response.Reasoning != "",
			"response_length", len(response.Content),
			"reasoning_length", len(response.Reasoning))

		// Display results
		fmt.Printf("\n=== Task: %s ===\n", task.complexity)
		if response.Reasoning != "" {
			fmt.Printf("🤔 Reasoning (Confidence: %.2f):\n%s\n\n", response.Confidence, response.Reasoning)
		}
		fmt.Printf("📝 Response:\n%s\n", response.Content)
		fmt.Println("=" + strings.Repeat("=", 50))
	}

	// 5. Analytics and insights
	time.Sleep(2 * time.Second) // Allow async storage to complete

	// Get performance stats
	stats, err := reasoningSystem.Storage.GetStats(ctx, agent.ID, time.Time{}, time.Now())
	if err != nil {
		logger.WithError(err).ErrorContext(ctx, "Failed to get stats")
	} else {
		logger.InfoContext(ctx, "Agent performance statistics",
			"total_tasks", stats.TotalChains,
			"avg_confidence", fmt.Sprintf("%.2f", stats.AvgConfidence),
			"success_rate", fmt.Sprintf("%.2f%%", stats.SuccessRate*100),
			"avg_duration", stats.AvgDuration)

		// Get insights
		insights, err := reasoningSystem.GetInsights(ctx, agent.ID)
		if err == nil && len(insights) > 0 {
			fmt.Printf("\n📊 Performance Insights:\n")
			for _, insight := range insights {
				fmt.Printf("• %s\n", insight)
			}
		}
	}

	// 6. Query reasoning history
	fmt.Printf("\n📚 Recent Reasoning Chains:\n")
	recentChains, err := reasoningSystem.Storage.Query(ctx, &ReasoningQuery{
		AgentID: agent.ID,
		Limit:   5,
		OrderBy: "created_at",
	})

	if err == nil {
		for i, chain := range recentChains {
			fmt.Printf("%d. [%.2f confidence] %s\n",
				i+1,
				chain.Confidence,
				truncateString(chain.Content, 80))
		}
	}

	// 7. Pattern analysis
	patterns, err := reasoningSystem.Storage.GetPatterns(ctx, "", 5)
	if err == nil && len(patterns) > 0 {
		fmt.Printf("\n🔍 Identified Reasoning Patterns:\n")
		for _, pattern := range patterns {
			fmt.Printf("• %s (seen %d times, %.1f%% success)\n",
				pattern.Pattern,
				pattern.Occurrences,
				pattern.AvgSuccess*100)
		}
	}

	// 8. Graceful shutdown handling
	fmt.Printf("\n✅ Production reasoning system running. Press Ctrl+C to shutdown...\n")

	// Set up signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Wait for shutdown signal
	<-sigChan

	fmt.Println("\n🛑 Shutting down...")

	// Cancel maintenance tasks
	cancelMaintenance()

	// Final stats
	finalStats, err := reasoningSystem.Storage.GetStats(ctx, agent.ID, time.Time{}, time.Now())
	if err == nil {
		logger.InfoContext(ctx, "Final session statistics",
			"total_reasoning_chains", finalStats.TotalChains,
			"avg_confidence", fmt.Sprintf("%.2f", finalStats.AvgConfidence),
			"success_rate", fmt.Sprintf("%.2f%%", finalStats.SuccessRate*100))
	}

	// Close reasoning system (handled by defer)
	fmt.Println("✅ Shutdown complete")
}

// createProductionLLMClient creates a production LLM client
func createProductionLLMClient(ctx context.Context) providers.LLMClient {
	// In production, this would:
	// 1. Read configuration from environment/files
	// 2. Set up proper authentication
	// 3. Configure rate limiting
	// 4. Add monitoring/metrics
	// 5. Implement retry logic

	// For this example, return a mock client
	return &productionMockLLMClient{}
}

// productionMockLLMClient simulates a production LLM
type productionMockLLMClient struct{}

func (c *productionMockLLMClient) Complete(ctx context.Context, prompt string) (string, error) {
	if ctx.Err() != nil {
		return "", ctx.Err()
	}

	// Simulate different response patterns based on prompt content
	if len(prompt) < 100 {
		// Simple query
		return `<thinking>
This is a straightforward factual question.
I should provide a direct, accurate answer.
Confidence: 0.95
</thinking>

The capital of France is Paris. It has been the capital since 987 AD and is home to over 2.1 million people in the city proper.`, nil
	}

	if strings.Contains(prompt, "code") || strings.Contains(prompt, "func") {
		// Code analysis
		return `<thinking>
This request involves code analysis. Let me examine:
1. The function structure and purpose
2. Potential improvements
3. Best practices to apply
4. Performance considerations

The code fetches data from a database for multiple IDs.
Current issues:
- No error handling from fetchFromDB
- Sequential DB calls (potential performance issue)
- No context propagation
- No logging

Confidence: 0.85
</thinking>

Here are my suggestions for improving the code:

1. **Add error handling**:
   - Check and handle errors from fetchFromDB
   - Return errors to caller for proper handling

2. **Improve performance with concurrent fetching**:
   - Use goroutines for parallel DB queries
   - Implement proper synchronization

3. **Add context support**:
   - Pass context.Context for cancellation
   - Respect context deadlines

4. **Enhanced version**:
` + "```go" + `
func getData(ctx context.Context, ids []int) (map[int]string, error) {
    result := make(map[int]string)
    mu := sync.Mutex{}
    g, ctx := errgroup.WithContext(ctx)
    
    for _, id := range ids {
        id := id // capture loop variable
        g.Go(func() error {
            data, err := fetchFromDB(ctx, id)
            if err != nil {
                return fmt.Errorf("fetch ID %d: %w", id, err)
            }
            
            mu.Lock()
            result[id] = data.Value
            mu.Unlock()
            return nil
        })
    }
    
    if err := g.Wait(); err != nil {
        return nil, err
    }
    
    return result, nil
}
` + "```", nil
	}

	// Complex architecture question
	return `<thinking>
This is a complex architectural design question requiring:
1. Understanding of microservices principles
2. E-commerce domain knowledge
3. Scalability considerations
4. Technology selection
5. Integration patterns

Key aspects to cover:
- Service boundaries
- Data management
- Communication patterns
- Security
- Scalability

Confidence: 0.78
</thinking>

# Microservices Architecture for E-commerce Platform

## Core Services

### 1. User Service
- Authentication & Authorization
- User profiles and preferences
- OAuth2/JWT implementation

### 2. Product Catalog Service
- Product information management
- Search and filtering
- Inventory tracking integration

### 3. Order Management Service
- Order processing workflow
- Order status tracking
- Integration with payment and shipping

### 4. Payment Service
- Payment processing
- Multiple payment gateway support
- PCI compliance considerations

### 5. Inventory Service
- Real-time inventory tracking
- Warehouse management
- Stock alerts and reordering

### 6. Notification Service
- Email/SMS/Push notifications
- Event-driven messaging
- Template management

## Technical Architecture

### API Gateway
- Kong or AWS API Gateway
- Rate limiting and authentication
- Request routing

### Service Mesh
- Istio for service-to-service communication
- Circuit breakers and retries
- Observability

### Data Management
- Service-specific databases
- Event sourcing for order history
- CQRS for read-heavy operations

### Message Queue
- Apache Kafka for event streaming
- RabbitMQ for task queues
- Ensure eventual consistency

## Deployment Strategy
- Kubernetes for orchestration
- GitOps with ArgoCD
- Progressive deployment with Flagger

This architecture ensures scalability, maintainability, and resilience for your e-commerce platform.`, nil
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
