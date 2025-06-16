// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package dev

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// GoTestFramework implements the Go testing framework
type GoTestFramework struct{}

// Name returns the framework name
func (g *GoTestFramework) Name() string {
	return "go"
}

// Detect checks if this is a Go project
func (g *GoTestFramework) Detect(path string) bool {
	return fileExists("go.mod") ||
		fileExists("go.sum") ||
		len(findTestFiles(path, "*_test.go")) > 0
}

// BuildCommand builds the go test command
func (g *GoTestFramework) BuildCommand(input TestRunnerInput) ([]string, error) {
	cmd := []string{"go", "test"}

	// Add verbose flag
	if input.Verbose {
		cmd = append(cmd, "-v")
	}

	// Add coverage flag
	if input.Coverage {
		cmd = append(cmd, "-cover", "-coverprofile=coverage.out")
	}

	// Add parallel flag
	if input.Parallel {
		cmd = append(cmd, "-parallel", "4")
	}

	// Add timeout
	if input.Timeout > 0 {
		cmd = append(cmd, "-timeout", fmt.Sprintf("%ds", input.Timeout))
	}

	// Add fail fast
	if input.FailFast {
		cmd = append(cmd, "-failfast")
	}

	// Add pattern as run flag
	if input.Pattern != "" {
		cmd = append(cmd, "-run", input.Pattern)
	}

	// Add tags
	if len(input.Tags) > 0 {
		cmd = append(cmd, "-tags", strings.Join(input.Tags, ","))
	}

	// Add path or default to ./...
	if input.Path != "" {
		cmd = append(cmd, input.Path)
	} else {
		cmd = append(cmd, "./...")
	}

	return cmd, nil
}

// ParseOutput parses Go test output
func (g *GoTestFramework) ParseOutput(output string, exitCode int) (*TestResult, error) {
	lines := strings.Split(output, "\n")

	var tests []TestCase
	var currentPackage string
	var summary TestSummary
	var coverage *CoverageReport

	// Regex patterns for Go test output
	testRunRegex := regexp.MustCompile(`^=== RUN\s+(.+)$`)
	testResultRegex := regexp.MustCompile(`^\s*--- (PASS|FAIL|SKIP):\s+(\S+)\s+\(([0-9.]+)s\)`)
	packageRegex := regexp.MustCompile(`^(ok|FAIL)\s+(\S+)\s+([0-9.]+)s`)
	coverageRegex := regexp.MustCompile(`coverage:\s+([0-9.]+)%\s+of\s+statements`)

	var currentTest *TestCase
	var errorLines []string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Package detection
		if match := packageRegex.FindStringSubmatch(line); match != nil {
			currentPackage = match[2]
		}

		// Test start
		if match := testRunRegex.FindStringSubmatch(line); match != nil {
			currentTest = &TestCase{
				Name:    match[1],
				Package: currentPackage,
				Status:  "running",
			}
		}

		// Test result
		if match := testResultRegex.FindStringSubmatch(line); match != nil {
			status := strings.ToLower(match[1])
			testName := match[2]
			duration, _ := time.ParseDuration(match[3] + "s")

			// Find or create test case
			var test *TestCase
			for i := range tests {
				if tests[i].Name == testName {
					test = &tests[i]
					break
				}
			}
			if test == nil {
				test = &TestCase{
					Name:    testName,
					Package: currentPackage,
				}
				tests = append(tests, *test)
				test = &tests[len(tests)-1]
			}

			test.Status = status
			test.Duration = duration

			// Count test results
			switch status {
			case "pass":
				summary.Passed++
			case "fail":
				summary.Failed++
				// Process error lines
				if len(errorLines) > 0 {
					test.Error = &TestError{
						Message:    strings.Join(errorLines, "\n"),
						Type:       "test_failure",
						StackTrace: strings.Join(errorLines, "\n"),
					}
					errorLines = []string{}
				}
			case "skip":
				summary.Skipped++
			}
			summary.Total++
		}

		// Failure details
		if strings.Contains(line, "FAIL:") ||
			strings.Contains(line, "panic:") ||
			(currentTest != nil && currentTest.Status == "running" &&
				(strings.Contains(line, "Error:") || strings.Contains(line, "expected"))) {
			errorLines = append(errorLines, line)
		}

		// Coverage information
		if match := coverageRegex.FindStringSubmatch(line); match != nil {
			percentage, _ := strconv.ParseFloat(match[1], 64)
			coverage = &CoverageReport{
				Percentage: percentage,
			}
			summary.CoveragePercent = percentage
		}
	}

	summary.Success = summary.Failed == 0 && exitCode == 0

	result := &TestResult{
		Summary:  summary,
		Tests:    tests,
		Coverage: coverage,
		Output:   output,
		Metadata: map[string]string{
			"go_version": "detected",
		},
	}

	return result, nil
}

// SupportsCoverage returns true as Go supports coverage
func (g *GoTestFramework) SupportsCoverage() bool {
	return true
}

// SupportsParallel returns true as Go supports parallel testing
func (g *GoTestFramework) SupportsParallel() bool {
	return true
}

// findTestFiles finds test files matching a pattern
func findTestFiles(dir, pattern string) []string {
	files, err := findFiles(dir, pattern)
	if err != nil {
		return []string{}
	}
	return files
}
