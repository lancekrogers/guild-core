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

// JavaScriptTestFramework implements Jest framework
type JavaScriptTestFramework struct{}

// Name returns the framework name
func (j *JavaScriptTestFramework) Name() string {
	return "jest"
}

// Detect checks if this is a JavaScript project with Jest
func (j *JavaScriptTestFramework) Detect(path string) bool {
	return fileExists("package.json") ||
		fileExists("jest.config.js") ||
		fileExists("jest.config.json") ||
		len(findTestFiles(path, "*.test.js")) > 0 ||
		len(findTestFiles(path, "*.test.ts")) > 0 ||
		len(findTestFiles(path, "*.spec.js")) > 0 ||
		len(findTestFiles(path, "*.spec.ts")) > 0
}

// BuildCommand builds the Jest command
func (j *JavaScriptTestFramework) BuildCommand(input TestRunnerInput) ([]string, error) {
	cmd := []string{"npm", "test"}

	// Alternative: use jest directly if available
	// cmd := []string{"jest"}

	var jestArgs []string

	// Add verbose flag
	if input.Verbose {
		jestArgs = append(jestArgs, "--verbose")
	}

	// Add coverage flag
	if input.Coverage {
		jestArgs = append(jestArgs, "--coverage")
	}

	// Add parallel settings
	if input.Parallel {
		jestArgs = append(jestArgs, "--maxWorkers=4")
	} else {
		jestArgs = append(jestArgs, "--runInBand")
	}

	// Add timeout
	if input.Timeout > 0 {
		jestArgs = append(jestArgs, "--testTimeout", fmt.Sprintf("%d", input.Timeout*1000)) // Jest uses milliseconds
	}

	// Add fail fast
	if input.FailFast {
		jestArgs = append(jestArgs, "--bail")
	}

	// Add pattern
	if input.Pattern != "" {
		jestArgs = append(jestArgs, "--testNamePattern", input.Pattern)
	}

	// Add path pattern
	if input.Path != "" && input.Path != "." {
		jestArgs = append(jestArgs, input.Path)
	}

	// Pass Jest arguments via npm
	if len(jestArgs) > 0 {
		cmd = append(cmd, "--")
		cmd = append(cmd, jestArgs...)
	}

	return cmd, nil
}

// ParseOutput parses Jest output
func (j *JavaScriptTestFramework) ParseOutput(output string, exitCode int) (*TestResult, error) {
	lines := strings.Split(output, "\n")

	var tests []TestCase
	var summary TestSummary
	var coverage *CoverageReport

	// Regex patterns for Jest output
	testSuiteRegex := regexp.MustCompile(`^\s*(PASS|FAIL)\s+(.+\.(?:js|ts|jsx|tsx))\s+\((\d+\.?\d*)\s*s\)`)
	testCaseRegex := regexp.MustCompile(`^\s*[✓×✗]\s+(.+)\s+\((\d+)\s*ms\)`)
	summaryRegex := regexp.MustCompile(`Tests:\s+(\d+)\s+failed,\s+(\d+)\s+passed,\s+(\d+)\s+total`)
	coverageRegex := regexp.MustCompile(`All files\s+\|\s+([0-9.]+)\s+\|`)

	var currentSuite string
	var errorLines []string
	var inErrorSection bool

	for _, line := range lines {
		originalLine := line
		line = strings.TrimSpace(line)

		if line == "" {
			continue
		}

		// Test suite result
		if match := testSuiteRegex.FindStringSubmatch(line); match != nil {
			status := strings.ToLower(match[1])
			suiteName := match[2]
			duration, _ := time.ParseDuration(match[3] + "s")

			currentSuite = suiteName

			// Create a test case for the suite
			test := TestCase{
				Name:     suiteName,
				File:     suiteName,
				Status:   status,
				Duration: duration,
			}

			if status == "fail" {
				summary.Failed++
			} else {
				summary.Passed++
			}
			summary.Total++

			tests = append(tests, test)
		}

		// Individual test case (if verbose output)
		if match := testCaseRegex.FindStringSubmatch(line); match != nil {
			testName := match[1]
			duration, _ := time.ParseDuration(match[2] + "ms")

			status := "passed"
			if strings.Contains(line, "×") || strings.Contains(line, "✗") {
				status = "failed"
			}

			test := TestCase{
				Name:     testName,
				File:     currentSuite,
				Status:   status,
				Duration: duration,
			}

			tests = append(tests, test)
		}

		// Error sections
		if strings.Contains(line, "● ") {
			inErrorSection = true
			errorLines = []string{originalLine}
		} else if inErrorSection && strings.HasPrefix(line, "  ") {
			errorLines = append(errorLines, originalLine)
		} else if inErrorSection && len(errorLines) > 0 {
			// End of error section, attach to last failed test
			for i := len(tests) - 1; i >= 0; i-- {
				if tests[i].Status == "failed" && tests[i].Error == nil {
					tests[i].Error = &TestError{
						Message:    strings.Join(errorLines, "\n"),
						Type:       "jest_error",
						StackTrace: strings.Join(errorLines, "\n"),
					}
					break
				}
			}
			inErrorSection = false
			errorLines = []string{}
		}

		// Summary line
		if match := summaryRegex.FindStringSubmatch(line); match != nil {
			failed, _ := strconv.Atoi(match[1])
			passed, _ := strconv.Atoi(match[2])
			total, _ := strconv.Atoi(match[3])

			summary.Failed = failed
			summary.Passed = passed
			summary.Total = total
			summary.Skipped = total - failed - passed
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
			"js_framework": "jest",
		},
	}

	return result, nil
}

// SupportsCoverage returns true as Jest supports coverage
func (j *JavaScriptTestFramework) SupportsCoverage() bool {
	return true
}

// SupportsParallel returns true as Jest supports parallel testing
func (j *JavaScriptTestFramework) SupportsParallel() bool {
	return true
}
