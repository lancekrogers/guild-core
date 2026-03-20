// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package benchmarks

import (
	"testing"
	"time"

	"github.com/lancekrogers/guild-core/internal/ui/chat/services"
)

// BenchmarkSimpleSuggestion demonstrates basic suggestion performance
// This is a simple benchmark that can be run to verify the system works
func BenchmarkSimpleSuggestion(b *testing.B) {
	service := setupSuggestionService(b)

	// Simple test query
	message := "How do I create a REST API?"
	context := &services.SuggestionContext{
		ConversationID: "demo",
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		start := time.Now()
		cmd := service.GetSuggestions(message, context)
		result := cmd()
		latency := time.Since(start)

		// Verify we got a valid response
		if len(result.Suggestions) == 0 {
			b.Error("Expected suggestions in response")
		}

		// Report latency
		b.ReportMetric(float64(latency.Microseconds()), "latency_us")

		// Check if meets target
		if latency > TargetLatency {
			b.Logf("Latency %v exceeds target %v", latency, TargetLatency)
		}
	}
}

// BenchmarkCacheDemo demonstrates cache effectiveness
func BenchmarkCacheDemo(b *testing.B) {
	service := setupSuggestionService(b)

	// Use same query to demonstrate caching
	message := "Implement user authentication"
	context := &services.SuggestionContext{
		ConversationID: "cache_demo",
	}

	var cachedCount, uncachedCount int

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		cmd := service.GetSuggestions(message, context)
		result := cmd()

		if result.FromCache {
			cachedCount++
		} else {
			uncachedCount++
		}
	}

	cacheHitRate := float64(cachedCount) / float64(cachedCount+uncachedCount) * 100
	b.ReportMetric(cacheHitRate, "cache_hit_rate_%")
	b.ReportMetric(float64(cachedCount), "cached_responses")
	b.ReportMetric(float64(uncachedCount), "uncached_responses")

	b.Logf("Cache hit rate: %.2f%% (%d cached, %d uncached)",
		cacheHitRate, cachedCount, uncachedCount)
}

// BenchmarkTokenOptimizationDemo shows token reduction in action
func BenchmarkTokenOptimizationDemo(b *testing.B) {
	service := setupSuggestionService(b)

	// Create a large context that needs optimization
	largeContext := generateContext(10000) // 10KB context

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		// Test optimization
		optimized := service.OptimizeContext(largeContext)

		originalTokens := estimateTokens(largeContext)
		optimizedTokens := estimateTokens(optimized)
		reduction := float64(originalTokens-optimizedTokens) / float64(originalTokens) * 100

		b.ReportMetric(float64(originalTokens), "original_tokens")
		b.ReportMetric(float64(optimizedTokens), "optimized_tokens")
		b.ReportMetric(reduction, "reduction_%")

		if i == 0 { // Log once
			b.Logf("Token optimization: %d → %d tokens (%.1f%% reduction)",
				originalTokens, optimizedTokens, reduction)
		}
	}
}
