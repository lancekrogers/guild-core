// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package benchmarks

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// PerformanceReport represents the benchmark results
type PerformanceReport struct {
	Timestamp     time.Time              `json:"timestamp"`
	SystemInfo    SystemInfo             `json:"system_info"`
	Goals         DevelopmentPhaseGoals  `json:"development_goals"`
	Results       map[string]BenchResult `json:"results"`
	Summary       Summary                `json:"summary"`
	Bottlenecks   []Bottleneck           `json:"bottlenecks"`
	Optimizations []Optimization         `json:"optimizations"`
}

// SystemInfo contains system information
type SystemInfo struct {
	OS         string `json:"os"`
	Arch       string `json:"arch"`
	CPUs       int    `json:"cpus"`
	GoVersion  string `json:"go_version"`
	CommitHash string `json:"commit_hash"`
}

// DevelopmentPhaseGoals defines the performance targets
type DevelopmentPhaseGoals struct {
	MaxLatency          string  `json:"max_latency"`
	MinTokenReduction   float64 `json:"min_token_reduction"`
	MaxTokenReduction   float64 `json:"max_token_reduction"`
	MinCacheHitRate     float64 `json:"min_cache_hit_rate"`
	MaxMemoryPerService string  `json:"max_memory_per_service"`
}

// BenchResult represents a single benchmark result
type BenchResult struct {
	Name        string             `json:"name"`
	Iterations  int                `json:"iterations"`
	Metrics     map[string]float64 `json:"metrics"`
	PassFail    string             `json:"pass_fail"`
	FailReasons []string           `json:"fail_reasons,omitempty"`
}

// Summary provides overall performance summary
type Summary struct {
	TotalTests      int     `json:"total_tests"`
	PassedTests     int     `json:"passed_tests"`
	FailedTests     int     `json:"failed_tests"`
	AvgLatency      float64 `json:"avg_latency_ms"`
	P95Latency      float64 `json:"p95_latency_ms"`
	P99Latency      float64 `json:"p99_latency_ms"`
	TokenReduction  float64 `json:"avg_token_reduction"`
	CacheHitRate    float64 `json:"cache_hit_rate"`
	MemoryFootprint float64 `json:"memory_footprint_kb"`
	MeetsTargets    bool    `json:"meets_all_targets"`
}

// Bottleneck identifies performance bottlenecks
type Bottleneck struct {
	Component   string  `json:"component"`
	Issue       string  `json:"issue"`
	Impact      string  `json:"impact"`
	CurrentPerf float64 `json:"current_performance"`
	Target      float64 `json:"target"`
}

// Optimization suggests performance improvements
type Optimization struct {
	Area           string `json:"area"`
	Suggestion     string `json:"suggestion"`
	ExpectedImpact string `json:"expected_impact"`
	Implementation string `json:"implementation"`
	Priority       string `json:"priority"`
}

// GeneratePerformanceReport runs benchmarks and generates a comprehensive report
func GeneratePerformanceReport() (*PerformanceReport, error) {
	report := &PerformanceReport{
		Timestamp: time.Now(),
		SystemInfo: SystemInfo{
			OS:        runtime.GOOS,
			Arch:      runtime.GOARCH,
			CPUs:      runtime.NumCPU(),
			GoVersion: runtime.Version(),
		},
		Goals: DevelopmentPhaseGoals{
			MaxLatency:          "100ms",
			MinTokenReduction:   15.0,
			MaxTokenReduction:   25.0,
			MinCacheHitRate:     80.0,
			MaxMemoryPerService: "1MB",
		},
		Results: make(map[string]BenchResult),
	}

	// Get git commit hash
	if hash, err := exec.Command("git", "rev-parse", "HEAD").Output(); err == nil {
		report.SystemInfo.CommitHash = strings.TrimSpace(string(hash))[:8]
	}

	// Run benchmarks
	benchmarks := []string{
		"BenchmarkSuggestionLatency",
		"BenchmarkTokenOptimization",
		"BenchmarkConcurrentAccess",
		"BenchmarkCacheEffectiveness",
		"BenchmarkMemoryUsage",
		"BenchmarkProviderChain",
		"BenchmarkIntegrationFlow",
	}

	for _, bench := range benchmarks {
		result, err := runBenchmark(bench)
		if err != nil {
			return nil, fmt.Errorf("failed to run %s: %w", bench, err)
		}
		report.Results[bench] = result
	}

	// Analyze results
	report.Summary = analyzeSummary(report.Results)
	report.Bottlenecks = identifyBottlenecks(report.Results, report.Goals)
	report.Optimizations = suggestOptimizations(report.Bottlenecks)

	return report, nil
}

// runBenchmark executes a single benchmark and parses results
func runBenchmark(name string) (BenchResult, error) {
	cmd := exec.Command("go", "test", "-bench", name, "-benchmem", "-benchtime=10s", "./benchmarks")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return BenchResult{}, fmt.Errorf("benchmark failed: %s", string(output))
	}

	return parseBenchmarkOutput(name, string(output)), nil
}

// parseBenchmarkOutput parses go test benchmark output
func parseBenchmarkOutput(name string, output string) BenchResult {
	result := BenchResult{
		Name:     name,
		Metrics:  make(map[string]float64),
		PassFail: "PASS",
	}

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.Contains(line, name) && strings.Contains(line, "ns/op") {
			// Parse benchmark line
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				if iterations, err := strconv.Atoi(parts[1]); err == nil {
					result.Iterations = iterations
				}
				if nsPerOp, err := strconv.ParseFloat(strings.TrimSuffix(parts[2], "ns/op"), 64); err == nil {
					result.Metrics["ns/op"] = nsPerOp
				}
			}

			// Parse custom metrics
			metricRegex := regexp.MustCompile(`(\w+):\s*([\d.]+)`)
			matches := metricRegex.FindAllStringSubmatch(line, -1)
			for _, match := range matches {
				if len(match) == 3 {
					if value, err := strconv.ParseFloat(match[2], 64); err == nil {
						result.Metrics[match[1]] = value
					}
				}
			}
		}

		// Check for failures
		if strings.Contains(line, "FAIL") && strings.Contains(line, name) {
			result.PassFail = "FAIL"
			if strings.Contains(line, "exceeds target") {
				result.FailReasons = append(result.FailReasons, line)
			}
		}
	}

	return result
}

// analyzeSummary calculates overall performance metrics
func analyzeSummary(results map[string]BenchResult) Summary {
	summary := Summary{
		TotalTests: len(results),
	}

	var latencies []float64
	var tokenReductions []float64
	var cacheHits []float64
	var memoryUsage []float64

	for _, result := range results {
		if result.PassFail == "PASS" {
			summary.PassedTests++
		} else {
			summary.FailedTests++
		}

		// Collect latency metrics
		if avgMs, ok := result.Metrics["avg_ms"]; ok {
			latencies = append(latencies, avgMs)
		}
		if p95Ms, ok := result.Metrics["p95_ms"]; ok {
			summary.P95Latency = max(summary.P95Latency, p95Ms)
		}
		if p99Ms, ok := result.Metrics["p99_ms"]; ok {
			summary.P99Latency = max(summary.P99Latency, p99Ms)
		}

		// Collect other metrics
		if reduction, ok := result.Metrics["reduction_%"]; ok {
			tokenReductions = append(tokenReductions, reduction)
		}
		if hitRate, ok := result.Metrics["cache_hit_%"]; ok {
			cacheHits = append(cacheHits, hitRate)
		}
		if memKB, ok := result.Metrics["KB/service"]; ok {
			memoryUsage = append(memoryUsage, memKB)
		}
	}

	// Calculate averages
	if len(latencies) > 0 {
		summary.AvgLatency = average(latencies)
	}
	if len(tokenReductions) > 0 {
		summary.TokenReduction = average(tokenReductions)
	}
	if len(cacheHits) > 0 {
		summary.CacheHitRate = average(cacheHits)
	}
	if len(memoryUsage) > 0 {
		summary.MemoryFootprint = average(memoryUsage)
	}

	// Check if all targets are met
	summary.MeetsTargets = summary.FailedTests == 0 &&
		summary.AvgLatency <= 100 &&
		summary.TokenReduction >= 15 &&
		summary.CacheHitRate >= 80 &&
		summary.MemoryFootprint <= 1024

	return summary
}

// identifyBottlenecks finds performance issues
func identifyBottlenecks(results map[string]BenchResult, goals DevelopmentPhaseGoals) []Bottleneck {
	var bottlenecks []Bottleneck

	// Check latency
	for name, result := range results {
		if avgMs, ok := result.Metrics["avg_ms"]; ok && avgMs > 100 {
			bottlenecks = append(bottlenecks, Bottleneck{
				Component:   name,
				Issue:       "Latency exceeds target",
				Impact:      fmt.Sprintf("%.0fms average latency", avgMs),
				CurrentPerf: avgMs,
				Target:      100,
			})
		}
	}

	// Check token reduction
	if tokenResult, ok := results["BenchmarkTokenOptimization"]; ok {
		if reduction, ok := tokenResult.Metrics["reduction_%"]; ok && reduction < goals.MinTokenReduction {
			bottlenecks = append(bottlenecks, Bottleneck{
				Component:   "Token Optimization",
				Issue:       "Insufficient token reduction",
				Impact:      fmt.Sprintf("Only %.1f%% reduction achieved", reduction),
				CurrentPerf: reduction,
				Target:      goals.MinTokenReduction,
			})
		}
	}

	// Check cache effectiveness
	if cacheResult, ok := results["BenchmarkCacheEffectiveness"]; ok {
		if hitRate, ok := cacheResult.Metrics["cache_hit_%"]; ok && hitRate < goals.MinCacheHitRate {
			bottlenecks = append(bottlenecks, Bottleneck{
				Component:   "Cache System",
				Issue:       "Low cache hit rate",
				Impact:      fmt.Sprintf("%.1f%% hit rate", hitRate),
				CurrentPerf: hitRate,
				Target:      goals.MinCacheHitRate,
			})
		}
	}

	// Check memory usage
	if memResult, ok := results["BenchmarkMemoryUsage"]; ok {
		if memKB, ok := memResult.Metrics["KB/service"]; ok && memKB > 1024 {
			bottlenecks = append(bottlenecks, Bottleneck{
				Component:   "Memory Management",
				Issue:       "High memory footprint",
				Impact:      fmt.Sprintf("%.0f KB per service", memKB),
				CurrentPerf: memKB,
				Target:      1024,
			})
		}
	}

	return bottlenecks
}

// suggestOptimizations provides improvement recommendations
func suggestOptimizations(bottlenecks []Bottleneck) []Optimization {
	var optimizations []Optimization

	for _, bottleneck := range bottlenecks {
		switch bottleneck.Component {
		case "BenchmarkSuggestionLatency", "BenchmarkIntegrationFlow":
			optimizations = append(optimizations, Optimization{
				Area:           "Latency Reduction",
				Suggestion:     "Implement request batching and parallel provider queries",
				ExpectedImpact: fmt.Sprintf("Reduce latency from %.0fms to <100ms", bottleneck.CurrentPerf),
				Implementation: "Batch multiple suggestion requests and query providers concurrently",
				Priority:       "HIGH",
			})

		case "Token Optimization":
			optimizations = append(optimizations, Optimization{
				Area:           "Token Efficiency",
				Suggestion:     "Implement semantic compression and context pruning",
				ExpectedImpact: fmt.Sprintf("Increase reduction from %.1f%% to >15%%", bottleneck.CurrentPerf),
				Implementation: "Use embedding-based similarity to remove redundant context",
				Priority:       "MEDIUM",
			})

		case "Cache System":
			optimizations = append(optimizations, Optimization{
				Area:           "Cache Performance",
				Suggestion:     "Implement LRU eviction and preemptive cache warming",
				ExpectedImpact: fmt.Sprintf("Increase hit rate from %.1f%% to >80%%", bottleneck.CurrentPerf),
				Implementation: "Add LRU eviction policy and warm cache with common queries",
				Priority:       "MEDIUM",
			})

		case "Memory Management":
			optimizations = append(optimizations, Optimization{
				Area:           "Memory Optimization",
				Suggestion:     "Implement suggestion pooling and memory recycling",
				ExpectedImpact: fmt.Sprintf("Reduce memory from %.0f KB to <1MB", bottleneck.CurrentPerf),
				Implementation: "Use sync.Pool for suggestion objects and limit cache size",
				Priority:       "LOW",
			})
		}
	}

	// Add general optimizations
	optimizations = append(optimizations, Optimization{
		Area:           "Concurrent Performance",
		Suggestion:     "Implement connection pooling and request coalescing",
		ExpectedImpact: "Improve concurrent request handling by 50%",
		Implementation: "Use connection pools and coalesce duplicate requests",
		Priority:       "HIGH",
	})

	return optimizations
}

// helper functions
func average(nums []float64) float64 {
	if len(nums) == 0 {
		return 0
	}
	sum := 0.0
	for _, n := range nums {
		sum += n
	}
	return sum / float64(len(nums))
}

func max(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

// SaveReport saves the performance report to a file
func SaveReport(report *PerformanceReport, filename string) error {
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filename, data, 0o644)
}

// GenerateMarkdownReport creates a human-readable markdown report
func GenerateMarkdownReport(report *PerformanceReport) string {
	var buf bytes.Buffer

	fmt.Fprintf(&buf, "# Suggestion System Performance Report\n\n")
	fmt.Fprintf(&buf, "Generated: %s\n", report.Timestamp.Format("2006-01-02 15:04:05"))
	fmt.Fprintf(&buf, "Commit: %s\n\n", report.SystemInfo.CommitHash)

	fmt.Fprintf(&buf, "## production enhancement Performance Targets\n\n")
	fmt.Fprintf(&buf, "- **Max Latency**: %s\n", report.Goals.MaxLatency)
	fmt.Fprintf(&buf, "- **Token Reduction**: %.0f%% - %.0f%%\n",
		report.Goals.MinTokenReduction, report.Goals.MaxTokenReduction)
	fmt.Fprintf(&buf, "- **Cache Hit Rate**: ≥%.0f%%\n", report.Goals.MinCacheHitRate)
	fmt.Fprintf(&buf, "- **Memory per Service**: %s\n\n", report.Goals.MaxMemoryPerService)

	fmt.Fprintf(&buf, "## Summary\n\n")
	status := "✅ PASS"
	if !report.Summary.MeetsTargets {
		status = "❌ FAIL"
	}
	fmt.Fprintf(&buf, "**Overall Status**: %s\n\n", status)
	fmt.Fprintf(&buf, "- Tests Passed: %d/%d\n", report.Summary.PassedTests, report.Summary.TotalTests)
	fmt.Fprintf(&buf, "- Average Latency: %.2fms\n", report.Summary.AvgLatency)
	fmt.Fprintf(&buf, "- P95 Latency: %.2fms\n", report.Summary.P95Latency)
	fmt.Fprintf(&buf, "- P99 Latency: %.2fms\n", report.Summary.P99Latency)
	fmt.Fprintf(&buf, "- Token Reduction: %.1f%%\n", report.Summary.TokenReduction)
	fmt.Fprintf(&buf, "- Cache Hit Rate: %.1f%%\n", report.Summary.CacheHitRate)
	fmt.Fprintf(&buf, "- Memory Footprint: %.0f KB/service\n\n", report.Summary.MemoryFootprint)

	fmt.Fprintf(&buf, "## Detailed Results\n\n")
	for name, result := range report.Results {
		status := "✅"
		if result.PassFail == "FAIL" {
			status = "❌"
		}
		fmt.Fprintf(&buf, "### %s %s\n\n", name, status)
		fmt.Fprintf(&buf, "- Iterations: %d\n", result.Iterations)
		for metric, value := range result.Metrics {
			fmt.Fprintf(&buf, "- %s: %.2f\n", metric, value)
		}
		if len(result.FailReasons) > 0 {
			fmt.Fprintf(&buf, "\n**Failures**:\n")
			for _, reason := range result.FailReasons {
				fmt.Fprintf(&buf, "- %s\n", reason)
			}
		}
		fmt.Fprintf(&buf, "\n")
	}

	if len(report.Bottlenecks) > 0 {
		fmt.Fprintf(&buf, "## Identified Bottlenecks\n\n")
		for _, b := range report.Bottlenecks {
			fmt.Fprintf(&buf, "### %s\n", b.Component)
			fmt.Fprintf(&buf, "- **Issue**: %s\n", b.Issue)
			fmt.Fprintf(&buf, "- **Impact**: %s\n", b.Impact)
			fmt.Fprintf(&buf, "- **Current**: %.2f | **Target**: %.2f\n\n", b.CurrentPerf, b.Target)
		}
	}

	if len(report.Optimizations) > 0 {
		fmt.Fprintf(&buf, "## Recommended Optimizations\n\n")
		for _, opt := range report.Optimizations {
			fmt.Fprintf(&buf, "### %s [%s]\n", opt.Area, opt.Priority)
			fmt.Fprintf(&buf, "- **Suggestion**: %s\n", opt.Suggestion)
			fmt.Fprintf(&buf, "- **Expected Impact**: %s\n", opt.ExpectedImpact)
			fmt.Fprintf(&buf, "- **Implementation**: %s\n\n", opt.Implementation)
		}
	}

	return buf.String()
}
