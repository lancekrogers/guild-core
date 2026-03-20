// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

// internal/buildutil/tasks/integration.go
package tasks

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/lancekrogers/guild-core/internal/buildutil/ui"
)

// IntegrationResult tracks integration test results
type IntegrationResult struct {
	Suite    string
	Pass     bool
	Duration time.Duration
}

// Integration runs integration tests
func Integration(verbose bool) error {
	ui.Section("Running Integration Tests")

	suites, err := discoverIntegrationSuites()
	if err != nil {
		return fmt.Errorf("failed to discover integration test suites: %w", err)
	}

	if len(suites) == 0 {
		ui.Status("No integration tests found", true)
		return nil
	}

	if verbose {
		fmt.Printf("Found %d integration test suites\n", len(suites))
	}

	results := make([]IntegrationResult, 0, len(suites))
	total := len(suites)
	failures := 0

	// Run each test suite
	for i, suite := range suites {
		name := strings.TrimPrefix(suite, "integration/")

		ui.Progress(i+1, total, fmt.Sprintf("Testing %s", name))

		start := time.Now()
		cmd := exec.Command("go", "test", "-tags", "integration", "-timeout", "2m", "./"+suite)

		if verbose {
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
		}

		pass := cmd.Run() == nil
		duration := time.Since(start)

		results = append(results, IntegrationResult{
			Suite:    name,
			Pass:     pass,
			Duration: duration,
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

	// Display summary
	rows := [][]string{
		{"Test Suite", "Status", "Time"},
	}

	for _, r := range results {
		status := "✓ PASS"
		if !r.Pass {
			status = "✗ FAIL"
		}
		if ui.ColourEnabled() {
			if r.Pass {
				status = ui.Green + status + ui.Reset
			} else {
				status = ui.Red + status + ui.Reset
			}
		}

		rows = append(rows, []string{
			r.Suite,
			status,
			fmt.Sprintf("%.2fs", r.Duration.Seconds()),
		})
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

	// Use custom status messages for integration test results
	successMsg := "✓ ALL INTEGRATION TESTS PASSED"
	failMsg := fmt.Sprintf("✗ %d INTEGRATION TESTS FAILED", failures)

	ui.SummaryCardWithStatus("Integration Test Summary", rows, fmt.Sprintf("%.2fs", totalTime.Seconds()), success, successMsg, failMsg)

	if failures > 0 {
		return fmt.Errorf("%d integration test suites failed", failures)
	}

	return nil
}

// discoverIntegrationSuites finds all integration test directories
func discoverIntegrationSuites() ([]string, error) {
	var suites []string

	// Check if integration directory exists
	if _, err := os.Stat("integration"); os.IsNotExist(err) {
		return suites, nil
	}

	// Walk the integration directory
	err := filepath.Walk("integration", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors
		}

		// Look for directories with _test.go files
		if info.IsDir() && path != "integration" {
			// Check if directory has test files
			matches, _ := filepath.Glob(filepath.Join(path, "*_test.go"))
			if len(matches) > 0 {
				suites = append(suites, path)
			}
		}

		return nil
	})

	return suites, err
}
