// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

// internal/buildutil/tasks/test.go
package tasks

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/guild-ventures/guild-core/internal/buildutil/ui"
)

// TestResult tracks test results for a package
type TestResult struct {
	Package  string
	Pass     bool
	Duration time.Duration
	HasTests bool
}

// Test runs go test on all packages
func Test(verbose bool) error {
	ui.Section("Testing Guild Framework")

	packages, err := discoverTestPackages()
	if err != nil {
		return fmt.Errorf("failed to discover test packages: %w", err)
	}

	if verbose {
		fmt.Printf("Found %d packages with tests\n", len(packages))
	}

	results := make([]TestResult, 0, len(packages))
	total := len(packages)
	failures := 0

	// Test each package
	for i, pkg := range packages {
		shortName := strings.TrimPrefix(pkg, "./")
		if shortName == "." {
			shortName = "root"
		}

		ui.Progress(i+1, total, fmt.Sprintf("Testing %s", shortName))

		start := time.Now()
		cmd := exec.Command("go", "test", "-short", "-timeout", "30s", pkg)

		if verbose {
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
		}

		pass := cmd.Run() == nil
		duration := time.Since(start)

		results = append(results, TestResult{
			Package:  shortName,
			Pass:     pass,
			Duration: duration,
			HasTests: true,
		})

		if !pass {
			failures++
		}
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

	// Display summary - only show packages with failures
	rows := [][]string{}
	hasFailures := failures > 0

	for _, r := range results {
		// Only include packages that failed
		if !r.Pass {
			status := "✗ FAIL"
			if ui.ColourEnabled() {
				status = ui.Red + status + ui.Reset
			}

			rows = append(rows, []string{
				r.Package,
				status,
				fmt.Sprintf("%.2fs", r.Duration.Seconds()),
			})
		}
	}

	// Add header only if there are failures to show
	if hasFailures {
		rows = append([][]string{{"Package", "Status", "Time"}}, rows...)
	}

	// Add totals row
	totalStatus := fmt.Sprintf("%d/%d passed", passed, len(results))
	if ui.ColourEnabled() {
		if failures > 0 {
			totalStatus = ui.Red + totalStatus + ui.Reset
		} else {
			totalStatus = ui.Green + totalStatus + ui.Reset
		}
	}

	rows = append(rows, []string{
		"TOTAL",
		totalStatus,
		fmt.Sprintf("%.2fs", totalTime.Seconds()),
	})

	success := failures == 0
	// Choose appropriate title based on whether there are failures
	title := "Test Summary"
	if hasFailures {
		title = "Test Failures"
	} else {
		title = "Tests Complete - All Passed"
	}

	// Use custom status messages for test results
	successMsg := "✓ ALL TESTS PASSED"
	failMsg := fmt.Sprintf("✗ %d TESTS FAILED", failures)

	ui.SummaryCardWithStatus(title, rows, fmt.Sprintf("%.2fs", totalTime.Seconds()), success, successMsg, failMsg)

	if failures > 0 {
		return fmt.Errorf("%d packages had test failures", failures)
	}

	return nil
}

// discoverTestPackages finds all packages that have tests
func discoverTestPackages() ([]string, error) {
	packages, err := discoverPackages()
	if err != nil {
		return nil, err
	}

	var testPackages []string

	for _, pkg := range packages {
		// Skip integration tests directory
		if strings.Contains(pkg, "/integration") {
			continue
		}

		// Check if package has test files
		cmd := exec.Command("go", "list", "-f", "{{.TestGoFiles}}", pkg)
		output, err := cmd.Output()
		if err != nil {
			continue
		}

		// If TestGoFiles is not empty array, package has tests
		if strings.TrimSpace(string(output)) != "[]" {
			testPackages = append(testPackages, pkg)
		}
	}

	return testPackages, nil
}
