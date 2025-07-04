// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

// internal/buildutil/tasks/happy.go
package tasks

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/lancekrogers/guild/internal/buildutil/ui"
)

// HappyPathResult tracks happy path test results
type HappyPathResult struct {
	Suite    string
	Pass     bool
	Duration time.Duration
	Details  string
}

// Happy runs happy path performance and SLA validation tests
func Happy(verbose bool) error {
	ui.Section("Running Happy Path Tests")

	// Define happy path test suites with their expected purpose
	suites := []HappyPathSuite{
		{
			Path:        "integration/happy-path/dev-tools",
			Name:        "Development Tools Performance",
			Description: "Multi-language codebase analysis and code intelligence validation",
			Timeout:     "2m",
			Points:      15,
		},
		{
			Path:        "integration/happy-path/sla",
			Name:        "SLA Validation Framework", 
			Description: "End-to-end SLA monitoring and compliance validation",
			Timeout:     "5m",
			Points:      12,
		},
		{
			Path:        "integration/happy-path/optimization",
			Name:        "Performance Optimization",
			Description: "Continuous optimization and baseline management",
			Timeout:     "8m",
			Points:      8,
		},
		{
			Path:        "integration/happy-path/agent-orchestration",
			Name:        "Agent Orchestration",
			Description: "Agent selection and execution performance validation",
			Timeout:     "6m",
			Points:      10,
		},
		{
			Path:        "integration/happy-path/tui-cli",
			Name:        "TUI/CLI Interface",
			Description: "Theme switching and interface responsiveness validation",
			Timeout:     "4m",
			Points:      10,
		},
	}

	// Verify all test suites exist
	if err := verifyHappyPathSuites(suites); err != nil {
		return fmt.Errorf("happy path test verification failed: %w", err)
	}

	if verbose {
		fmt.Printf("Running %d happy path test suites\n", len(suites))
	}

	results := make([]HappyPathResult, 0, len(suites))
	total := len(suites)
	failures := 0
	totalPoints := 0
	earnedPoints := 0

	// Run each test suite
	for i, suite := range suites {
		ui.Progress(i+1, total, fmt.Sprintf("Testing %s", suite.Name))

		start := time.Now()
		result, err := runHappyPathSuite(suite, verbose)
		duration := time.Since(start)

		if err != nil {
			result = HappyPathResult{
				Suite:    suite.Name,
				Pass:     false,
				Duration: duration,
				Details:  err.Error(),
			}
			failures++
		} else {
			result.Duration = duration
			if result.Pass {
				earnedPoints += suite.Points
			} else {
				failures++
			}
		}

		results = append(results, result)
		totalPoints += suite.Points
	}

	ui.ClearProgress()

	// Calculate totals
	var totalTime time.Duration
	passed := 0
	for _, r := range results {
		totalTime += r.Duration
		if r.Pass {
			passed++
		}
	}

	// Display detailed summary
	rows := [][]string{
		{"Test Suite", "Points", "Status", "Time", "Details"},
	}

	for i, r := range results {
		suite := suites[i]
		
		status := "✓ PASS"
		points := fmt.Sprintf("%d/%d", 0, suite.Points)
		if r.Pass {
			status = "✓ PASS"
			points = fmt.Sprintf("%d/%d", suite.Points, suite.Points)
		} else {
			status = "✗ FAIL"
		}

		if ui.ColourEnabled() {
			if r.Pass {
				status = ui.Green + status + ui.Reset
				points = ui.Green + points + ui.Reset
			} else {
				status = ui.Red + status + ui.Reset
				points = ui.Red + points + ui.Reset
			}
		}

		details := r.Details
		if len(details) > 40 {
			details = details[:37] + "..."
		}
		if details == "" {
			details = suite.Description
			if len(details) > 40 {
				details = details[:37] + "..."
			}
		}

		rows = append(rows, []string{
			r.Suite,
			points,
			status,
			fmt.Sprintf("%.1fs", r.Duration.Seconds()),
			details,
		})
	}

	// Add totals row
	totalStatus := fmt.Sprintf("%d/%d suites", passed, len(results))
	totalPointsStr := fmt.Sprintf("%d/%d pts", earnedPoints, totalPoints)
	
	if ui.ColourEnabled() {
		if failures > 0 {
			totalStatus = ui.Red + totalStatus + ui.Reset
			totalPointsStr = ui.Red + totalPointsStr + ui.Reset
		} else {
			totalStatus = ui.Green + totalStatus + ui.Reset
			totalPointsStr = ui.Green + totalPointsStr + ui.Reset
		}
	}

	rows = append(rows, []string{
		"TOTAL",
		totalPointsStr,
		totalStatus,
		fmt.Sprintf("%.1fs", totalTime.Seconds()),
		getScoreGrade(earnedPoints, totalPoints),
	})

	success := failures == 0

	// Use custom status messages for happy path results
	successMsg := fmt.Sprintf("🎯 ALL TESTS PASSED - %d/%d POINTS EARNED", earnedPoints, totalPoints)
	failMsg := fmt.Sprintf("❌ TESTS FAILED - %d/%d POINTS (%d SUITES FAILED)", earnedPoints, totalPoints, failures)

	ui.SummaryCardWithStatus("Happy Path Test Results", rows, fmt.Sprintf("%.1fs", totalTime.Seconds()), success, successMsg, failMsg)

	// Additional performance insights
	if verbose && len(results) > 0 {
		fmt.Println()
		ui.Section("Performance Insights")
		showPerformanceInsights(results, suites)
	}

	if failures > 0 {
		return fmt.Errorf("%d happy path test suites failed", failures)
	}

	return nil
}

// HappyPathSuite defines a happy path test suite configuration
type HappyPathSuite struct {
	Path        string
	Name        string
	Description string
	Timeout     string
	Points      int
}

// verifyHappyPathSuites checks that all test suites exist
func verifyHappyPathSuites(suites []HappyPathSuite) error {
	missing := []string{}
	
	for _, suite := range suites {
		if _, err := os.Stat(suite.Path); os.IsNotExist(err) {
			missing = append(missing, suite.Path)
		}
	}
	
	if len(missing) > 0 {
		return fmt.Errorf("missing happy path test suites: %s", strings.Join(missing, ", "))
	}
	
	return nil
}

// runHappyPathSuite executes a single happy path test suite
func runHappyPathSuite(suite HappyPathSuite, verbose bool) (HappyPathResult, error) {
	cmd := exec.Command("go", "test", "-v", "-short", "-timeout", suite.Timeout, "./"+suite.Path)

	var output strings.Builder
	if verbose {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	} else {
		cmd.Stdout = &output
		cmd.Stderr = &output
	}

	err := cmd.Run()
	
	result := HappyPathResult{
		Suite: suite.Name,
		Pass:  err == nil,
	}

	if err != nil {
		// Extract useful error information
		outputStr := output.String()
		if strings.Contains(outputStr, "FAIL") {
			result.Details = "Test failures detected"
		} else if strings.Contains(outputStr, "timeout") {
			result.Details = "Test timeout exceeded"
		} else {
			result.Details = "Test execution failed"
		}
		return result, nil // Don't return error, we handle it in the result
	}

	result.Details = "All tests passed"
	return result, nil
}

// getTotalPoints calculates total points available
func getTotalPoints(suites []HappyPathSuite) int {
	total := 0
	for _, suite := range suites {
		total += suite.Points
	}
	return total
}

// getScoreGrade returns a grade based on points earned
func getScoreGrade(earned, total int) string {
	if total == 0 {
		return "N/A"
	}
	
	percentage := float64(earned) / float64(total) * 100
	
	switch {
	case percentage >= 95:
		return "A+ (Excellent)"
	case percentage >= 90:
		return "A (Great)"
	case percentage >= 85:
		return "B+ (Good)"
	case percentage >= 80:
		return "B (Satisfactory)"
	case percentage >= 75:
		return "C+ (Needs Work)"
	case percentage >= 70:
		return "C (Poor)"
	default:
		return "F (Failed)"
	}
}

// showPerformanceInsights displays additional performance analysis
func showPerformanceInsights(results []HappyPathResult, suites []HappyPathSuite) {
	// Find slowest test
	var slowest HappyPathResult
	for _, r := range results {
		if r.Duration > slowest.Duration {
			slowest = r
		}
	}

	// Calculate average time
	var totalTime time.Duration
	for _, r := range results {
		totalTime += r.Duration
	}
	avgTime := totalTime / time.Duration(len(results))

	insights := []string{
		fmt.Sprintf("Slowest suite: %s (%.1fs)", slowest.Suite, slowest.Duration.Seconds()),
		fmt.Sprintf("Average time: %.1fs per suite", avgTime.Seconds()),
		fmt.Sprintf("Total execution: %.1fs", totalTime.Seconds()),
	}

	// Performance recommendations
	if slowest.Duration > 5*time.Minute {
		insights = append(insights, "⚠️  Consider optimizing slowest test suite")
	}
	if totalTime > 20*time.Minute {
		insights = append(insights, "⚠️  Total runtime exceeds 20 minutes")
	}

	for _, insight := range insights {
		fmt.Printf("  %s\n", insight)
	}
}