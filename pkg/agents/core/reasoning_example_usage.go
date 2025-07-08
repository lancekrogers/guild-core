// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package core

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/lancekrogers/guild/pkg/config"
	"github.com/lancekrogers/guild/pkg/observability"
)

// ExampleReasoningUsage demonstrates how to use the enhanced reasoning system
func ExampleReasoningUsage() {
	// Initialize context with observability
	ctx := context.Background()
	ctx = observability.WithComponent(ctx, "reasoning_example")
	logger := observability.GetLogger(ctx)

	// 1. Configure reasoning system
	reasoningConfig := DefaultReasoningConfig()
	reasoningConfig.EnableCaching = true
	reasoningConfig.CacheTTL = 5 * time.Minute
	reasoningConfig.MaxReasoningLength = 5000

	// 2. Create storage (using in-memory for example)
	storageConfig := ReasoningStorageConfig{
		RetentionDays:     30,
		MaxChainsPerQuery: 1000,
		EnableCompression: true,
		StorageBackend:    "memory",
	}

	storage, err := NewMemoryReasoningStorage(storageConfig)
	if err != nil {
		log.Fatalf("Failed to create storage: %v", err)
	}
	defer storage.Close(ctx)

	// 3. Create reasoning extractor
	_, err = NewReasoningExtractor(reasoningConfig)
	if err != nil {
		log.Fatalf("Failed to create extractor: %v", err)
	}

	// 4. Create analytics analyzer
	analyzer, err := NewReasoningAnalyzer(storage)
	if err != nil {
		log.Fatalf("Failed to create analyzer: %v", err)
	}

	// 5. Create enhanced agent factory
	// Note: In real usage, baseFactory would be obtained from dependency injection
	var baseFactory Factory // This would be injected
	enhancedFactory, err := NewEnhancedAgentFactory(baseFactory, reasoningConfig, storage)
	if err != nil {
		log.Fatalf("Failed to create enhanced factory: %v", err)
	}

	// 6. Create an agent with reasoning capabilities
	agentConfig := &config.EnhancedAgentConfig{
		ID:           "example_agent",
		Name:         "Example Reasoning Agent",
		Type:         "worker",
		Role:         "General purpose assistant with reasoning",
		Model:        "gpt-4",
		Temperature:  0.7,
		Capabilities: []string{"reasoning", "analysis", "code_generation"},
		Reasoning: config.ReasoningConfig{
			Enabled:              true,
			ShowThinking:         true,
			MinConfidenceDisplay: 0.3,
			IncludeInPrompt:      true,
		},
	}

	// Mock LLM client for example
	llmClient := &exampleMockLLMClient{}

	agent, err := enhancedFactory.CreateAgent(
		ctx,
		agentConfig,
		llmClient,
		nil, // memory manager
		nil, // tool registry
		nil, // commission manager
		nil, // cost manager
	)
	if err != nil {
		log.Fatalf("Failed to create agent: %v", err)
	}

	// 7. Use the agent with reasoning
	workerAgent := agent.(*WorkerAgent)

	// Example 1: Simple task
	response1, err := workerAgent.ExecuteWithReasoning(ctx, "What is the capital of France?")
	if err != nil {
		logger.WithError(err).Error("Failed to execute simple task")
	} else {
		fmt.Printf("Simple Task Response:\n")
		fmt.Printf("Content: %s\n", response1.Content)
		fmt.Printf("Reasoning: %s\n", response1.Reasoning)
		fmt.Printf("Confidence: %.2f\n\n", response1.Confidence)
	}

	// Example 2: Complex task requiring deep reasoning
	complexRequest := `Analyze the following code and suggest improvements:

func processData(data []int) int {
    result := 0
    for i := 0; i < len(data); i++ {
        if data[i] > 0 {
            result = result + data[i]
        }
    }
    return result
}`

	response2, err := workerAgent.ExecuteWithReasoning(ctx, complexRequest)
	if err != nil {
		logger.WithError(err).Error("Failed to execute complex task")
	} else {
		fmt.Printf("Complex Task Response:\n")
		fmt.Printf("Content: %s\n", response2.Content)
		fmt.Printf("Reasoning: %s\n", response2.Reasoning)
		fmt.Printf("Confidence: %.2f\n\n", response2.Confidence)
	}

	// 8. Query stored reasoning chains
	time.Sleep(100 * time.Millisecond) // Allow async storage to complete

	query := &ReasoningQuery{
		AgentID:   "example_agent",
		Limit:     10,
		OrderBy:   "confidence",
		Ascending: false,
	}

	chains, err := storage.Query(ctx, query)
	if err != nil {
		logger.WithError(err).Error("Failed to query chains")
	} else {
		fmt.Printf("Stored Reasoning Chains: %d\n", len(chains))
		for i, chain := range chains {
			fmt.Printf("  %d. Confidence: %.2f, Success: %v, Duration: %s\n",
				i+1, chain.Confidence, chain.Success, chain.Duration)
		}
	}

	// 9. Get analytics
	stats, err := storage.GetStats(ctx, "example_agent", time.Time{}, time.Now())
	if err != nil {
		logger.WithError(err).Error("Failed to get stats")
	} else {
		fmt.Printf("\nReasoning Statistics:\n")
		fmt.Printf("Total Chains: %d\n", stats.TotalChains)
		fmt.Printf("Average Confidence: %.2f\n", stats.AvgConfidence)
		fmt.Printf("Success Rate: %.2f%%\n", stats.SuccessRate*100)
		fmt.Printf("Average Duration: %s\n", stats.AvgDuration)
	}

	// 10. Generate insights
	insights, err := analyzer.GenerateInsights(ctx, stats)
	if err != nil {
		logger.WithError(err).Error("Failed to generate insights")
	} else {
		fmt.Printf("\nInsights:\n")
		for _, insight := range insights {
			fmt.Printf("- %s\n", insight)
		}
	}

	// 11. Analyze patterns
	patterns, err := analyzer.IdentifyPatterns(ctx, chains)
	if err != nil {
		logger.WithError(err).Error("Failed to identify patterns")
	} else {
		fmt.Printf("\nIdentified Patterns:\n")
		for _, pattern := range patterns {
			fmt.Printf("- Pattern: %s, Occurrences: %d, Success Rate: %.2f\n",
				pattern.Pattern, pattern.Occurrences, pattern.AvgSuccess)
		}
	}

	// 12. Clean up old chains (retention management)
	deleted, err := storage.Delete(ctx, time.Now().Add(-24*time.Hour))
	if err != nil {
		logger.WithError(err).Error("Failed to delete old chains")
	} else {
		fmt.Printf("\nDeleted %d old reasoning chains\n", deleted)
	}
}

// exampleMockLLMClient is a simple mock for demonstration
type exampleMockLLMClient struct{}

func (m *exampleMockLLMClient) Complete(ctx context.Context, prompt string) (string, error) {
	// Simulate LLM response with reasoning
	if ctx.Err() != nil {
		return "", ctx.Err()
	}

	// Simple logic to generate different responses
	if len(prompt) < 50 {
		// Simple question
		return `<thinking>
This is a straightforward factual question.
I should provide a direct answer.
Confidence: 0.95
</thinking>

The capital of France is Paris.`, nil
	}

	// Complex question
	return `<thinking>
This code analysis requires examining several aspects:
1. Code efficiency
2. Readability
3. Go idioms
4. Potential improvements

The function sums positive integers from a slice.
Current implementation uses traditional for loop.
Confidence: 0.85
</thinking>

Here are my suggestions for improving the code:

1. **Use range syntax**: More idiomatic Go
2. **Consider early return**: If data is nil/empty
3. **Add documentation**: Explain the function's purpose
4. **Rename for clarity**: 'sumPositive' is more descriptive

Improved version:
` + "```go" + `
// sumPositive returns the sum of all positive integers in the slice.
// Returns 0 for nil or empty slices.
func sumPositive(data []int) int {
    if len(data) == 0 {
        return 0
    }
    
    sum := 0
    for _, value := range data {
        if value > 0 {
            sum += value
        }
    }
    return sum
}
` + "```", nil
}

// ExampleProductionSetup shows how to set up reasoning in production
func ExampleProductionSetup() {
	ctx := context.Background()

	// Production configuration with all features
	productionConfig := ReasoningConfig{
		EnableCaching:      true,
		CacheMaxSize:       10000,
		CacheTTL:           15 * time.Minute,
		MaxReasoningLength: 20000,
		MinConfidence:      0.0,
		MaxConfidence:      1.0,
		StrictValidation:   true,
	}

	// Use SQLite or PostgreSQL in production
	storageConfig := ReasoningStorageConfig{
		RetentionDays:     90,
		MaxChainsPerQuery: 5000,
		EnableCompression: true,
		StorageBackend:    "sqlite", // or "postgres"
	}

	// Create production components
	extractor, _ := NewReasoningExtractor(productionConfig)
	storage, _ := NewMemoryReasoningStorage(storageConfig) // Replace with SQLite/Postgres
	analyzer, _ := NewReasoningAnalyzer(storage)

	// Set up monitoring
	go monitorReasoningMetrics(ctx, extractor, storage, analyzer)

	// Set up periodic cleanup
	go periodicCleanup(ctx, storage)

	fmt.Println("Production reasoning system initialized")
}

func monitorReasoningMetrics(ctx context.Context, extractor *ReasoningExtractor, storage ReasoningStorage, analyzer *DefaultReasoningAnalyzer) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Log extractor stats
			stats := extractor.GetStats()
			fmt.Printf("Reasoning Cache Hit Rate: %.2f%%\n", stats["cache_hit_rate"].(float64)*100)

			// Analyze recent performance
			recentChains, _ := storage.Query(ctx, &ReasoningQuery{
				StartTime: time.Now().Add(-5 * time.Minute),
				Limit:     100,
			})

			if len(recentChains) > 0 {
				correlation, _ := analyzer.AnalyzeConfidenceCorrelation(ctx, recentChains)
				fmt.Printf("Recent Confidence-Success Correlation: %.2f\n", correlation)
			}
		}
	}
}

func periodicCleanup(ctx context.Context, storage ReasoningStorage) {
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Clean up chains older than retention period
			cutoff := time.Now().Add(-90 * 24 * time.Hour)
			deleted, err := storage.Delete(ctx, cutoff)
			if err != nil {
				fmt.Printf("Cleanup error: %v\n", err)
			} else {
				fmt.Printf("Cleaned up %d old reasoning chains\n", deleted)
			}
		}
	}
}
