// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package tasks

import (
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/lancekrogers/guild/internal/buildutil/ui"
)

// E2EOptions configures E2E test execution
type E2EOptions struct {
	Verbose bool
	Timeout time.Duration
	Suite   string // Specific test suite to run
}

// E2E runs end-to-end tests
func E2E(opts E2EOptions) error {
	ui.Section("Running E2E Tests")

	// Set default timeout
	if opts.Timeout == 0 {
		opts.Timeout = 10 * time.Minute
	}

	// Set test environment
	env := os.Environ()
	env = append(env,
		"GUILD_MOCK_PROVIDER=true",
		"GUILD_TEST_MODE=true",
		"NO_COLOR=1",
		"GUILD_LOG_LEVEL=warn",
	)

	// Build the binary first
	ui.Status("Building Guild binary for E2E tests...", true)
	buildCmd := exec.Command("go", "build", "-o", "bin/guild", "./cmd/guild")
	buildCmd.Env = env
	if err := buildCmd.Run(); err != nil {
		return fmt.Errorf("failed to build Guild binary: %w", err)
	}

	// Determine test package
	testPkg := "./integration/e2e/..."
	if opts.Suite != "" {
		testPkg = fmt.Sprintf("./integration/e2e/%s", opts.Suite)
	}

	// Run E2E tests
	ui.Status("Running E2E test suite...", true)

	args := []string{
		"test",
		"-timeout", opts.Timeout.String(),
		"-tags", "e2e",
		testPkg,
	}

	if opts.Verbose {
		args = append(args, "-v")
	}

	start := time.Now()
	cmd := exec.Command("go", args...)
	cmd.Env = env

	if opts.Verbose {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}

	err := cmd.Run()
	duration := time.Since(start)

	success := err == nil

	// Display results
	rows := [][]string{
		{"Test Type", "Status", "Duration"},
	}

	status := "✓ PASS"
	if !success {
		status = "✗ FAIL"
	}

	if ui.ColourEnabled() {
		if success {
			status = ui.Green + status + ui.Reset
		} else {
			status = ui.Red + status + ui.Reset
		}
	}

	rows = append(rows, []string{
		"E2E Tests",
		status,
		fmt.Sprintf("%.2fs", duration.Seconds()),
	})

	successMsg := "✓ E2E TESTS PASSED"
	failMsg := "✗ E2E TESTS FAILED"

	ui.SummaryCardWithStatus("E2E Test Results", rows, fmt.Sprintf("%.2fs", duration.Seconds()), success, successMsg, failMsg)

	if !success {
		return fmt.Errorf("E2E tests failed")
	}

	return nil
}

// ValidateDemo runs the demo validation script
func ValidateDemo() error {
	ui.Section("Validating Demo Scripts")

	ui.Status("Running demo validation...", true)

	cmd := exec.Command("./scripts/validate-demos.sh")
	cmd.Env = append(os.Environ(),
		"GUILD_MOCK_PROVIDER=true",
		"GUILD_TEST_MODE=true",
		"NO_COLOR=1",
	)

	start := time.Now()

	// For demo validation, we want to see the output
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	duration := time.Since(start)

	success := err == nil

	if success {
		ui.Status("✓ Demo validation completed successfully", true)
	} else {
		ui.Status("✗ Demo validation failed", false)
	}

	if ui.ColourEnabled() {
		statusMsg := "Demo validation completed"
		if success {
			statusMsg = ui.Green + "✓ " + statusMsg + ui.Reset
		} else {
			statusMsg = ui.Red + "✗ " + statusMsg + ui.Reset
		}
		fmt.Printf("\n%s in %.2fs\n", statusMsg, duration.Seconds())
	} else {
		fmt.Printf("\nDemo validation completed in %.2fs\n", duration.Seconds())
	}

	if !success {
		return fmt.Errorf("demo validation failed")
	}

	return nil
}
