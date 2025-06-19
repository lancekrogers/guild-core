// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

// +build ignore

package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/guild-ventures/guild-core/benchmarks"
)

func main() {
	fmt.Println("🚀 Running Suggestion System Performance Benchmarks...")
	fmt.Println("================================================")
	fmt.Printf("Started at: %s\n\n", time.Now().Format("15:04:05"))

	// Generate performance report
	report, err := benchmarks.GeneratePerformanceReport()
	if err != nil {
		log.Fatalf("Failed to generate performance report: %v", err)
	}

	// Create reports directory
	reportsDir := "benchmarks/reports"
	if err := os.MkdirAll(reportsDir, 0755); err != nil {
		log.Fatalf("Failed to create reports directory: %v", err)
	}

	// Save JSON report
	timestamp := time.Now().Format("2006-01-02_15-04-05")
	jsonFile := filepath.Join(reportsDir, fmt.Sprintf("performance_report_%s.json", timestamp))
	if err := benchmarks.SaveReport(report, jsonFile); err != nil {
		log.Fatalf("Failed to save JSON report: %v", err)
	}
	fmt.Printf("✅ JSON report saved to: %s\n", jsonFile)

	// Generate and save markdown report
	markdownContent := benchmarks.GenerateMarkdownReport(report)
	mdFile := filepath.Join(reportsDir, fmt.Sprintf("performance_report_%s.md", timestamp))
	if err := os.WriteFile(mdFile, []byte(markdownContent), 0644); err != nil {
		log.Fatalf("Failed to save markdown report: %v", err)
	}
	fmt.Printf("✅ Markdown report saved to: %s\n", mdFile)

	// Also save as latest
	latestJSON := filepath.Join(reportsDir, "latest_performance_report.json")
	if err := benchmarks.SaveReport(report, latestJSON); err != nil {
		log.Printf("Warning: Failed to save latest JSON report: %v", err)
	}

	latestMD := filepath.Join(reportsDir, "latest_performance_report.md")
	if err := os.WriteFile(latestMD, []byte(markdownContent), 0644); err != nil {
		log.Printf("Warning: Failed to save latest markdown report: %v", err)
	}

	// Print summary to console
	fmt.Println("\n📊 Performance Summary")
	fmt.Println("=====================")
	
	status := "✅ PASS"
	if !report.Summary.MeetsTargets {
		status = "❌ FAIL"
	}
	fmt.Printf("Overall Status: %s\n", status)
	fmt.Printf("Tests Passed: %d/%d\n", report.Summary.PassedTests, report.Summary.TotalTests)
	fmt.Printf("Average Latency: %.2fms (target: <100ms)\n", report.Summary.AvgLatency)
	fmt.Printf("P95 Latency: %.2fms\n", report.Summary.P95Latency)
	fmt.Printf("P99 Latency: %.2fms\n", report.Summary.P99Latency)
	fmt.Printf("Token Reduction: %.1f%% (target: 15-25%%)\n", report.Summary.TokenReduction)
	fmt.Printf("Cache Hit Rate: %.1f%% (target: ≥80%%)\n", report.Summary.CacheHitRate)
	fmt.Printf("Memory/Service: %.0f KB (target: <1MB)\n", report.Summary.MemoryFootprint)

	// Print bottlenecks if any
	if len(report.Bottlenecks) > 0 {
		fmt.Println("\n⚠️  Bottlenecks Identified")
		fmt.Println("========================")
		for _, b := range report.Bottlenecks {
			fmt.Printf("- %s: %s (current: %.2f, target: %.2f)\n", 
				b.Component, b.Issue, b.CurrentPerf, b.Target)
		}
	}

	// Print top optimizations
	if len(report.Optimizations) > 0 {
		fmt.Println("\n💡 Top Optimizations")
		fmt.Println("==================")
		highPriority := 0
		for _, opt := range report.Optimizations {
			if opt.Priority == "HIGH" {
				fmt.Printf("- [%s] %s: %s\n", opt.Priority, opt.Area, opt.Suggestion)
				highPriority++
			}
			if highPriority >= 3 {
				break
			}
		}
	}

	fmt.Printf("\n✨ Benchmarks completed at: %s\n", time.Now().Format("15:04:05"))
	fmt.Printf("📁 Full reports available in: %s\n", reportsDir)

	// Exit with appropriate code
	if !report.Summary.MeetsTargets {
		os.Exit(1)
	}
}