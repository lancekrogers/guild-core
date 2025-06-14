package dev

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// PythonTestFramework implements pytest framework
type PythonTestFramework struct{}

// Name returns the framework name
func (p *PythonTestFramework) Name() string {
	return "pytest"
}

// Detect checks if this is a Python project with pytest
func (p *PythonTestFramework) Detect(path string) bool {
	return fileExists("pytest.ini") ||
		   fileExists("pyproject.toml") ||
		   fileExists("setup.py") ||
		   fileExists("requirements.txt") ||
		   len(findTestFiles(path, "test_*.py")) > 0 ||
		   len(findTestFiles(path, "*_test.py")) > 0
}

// BuildCommand builds the pytest command
func (p *PythonTestFramework) BuildCommand(input TestRunnerInput) ([]string, error) {
	cmd := []string{"pytest"}

	// Add verbose flag
	if input.Verbose {
		cmd = append(cmd, "-v")
	}

	// Add coverage flag
	if input.Coverage {
		cmd = append(cmd, "--cov=.", "--cov-report=term-missing")
	}

	// Add parallel flag (requires pytest-xdist)
	if input.Parallel {
		cmd = append(cmd, "-n", "auto")
	}

	// Add timeout
	if input.Timeout > 0 {
		cmd = append(cmd, "--timeout", fmt.Sprintf("%d", input.Timeout))
	}

	// Add fail fast
	if input.FailFast {
		cmd = append(cmd, "-x")
	}

	// Add pattern as -k flag
	if input.Pattern != "" {
		cmd = append(cmd, "-k", input.Pattern)
	}

	// Add markers/tags
	if len(input.Tags) > 0 {
		for _, tag := range input.Tags {
			cmd = append(cmd, "-m", tag)
		}
	}

	// Add path
	if input.Path != "" {
		cmd = append(cmd, input.Path)
	}

	return cmd, nil
}

// ParseOutput parses pytest output
func (p *PythonTestFramework) ParseOutput(output string, exitCode int) (*TestResult, error) {
	lines := strings.Split(output, "\n")
	
	var tests []TestCase
	var summary TestSummary
	var coverage *CoverageReport

	// Regex patterns for pytest output
	testResultRegex := regexp.MustCompile(`^(.+\.py)::\s*(.+)\s+(PASSED|FAILED|SKIPPED|ERROR)`)
	summaryRegex := regexp.MustCompile(`^=+\s*(\d+)\s+failed,?\s*(\d+)\s+passed`)
	coverageRegex := regexp.MustCompile(`^TOTAL\s+\d+\s+\d+\s+(\d+)%`)
	failureHeaderRegex := regexp.MustCompile(`^_+\s+(.+)\s+_+$`)
	
	var currentFailure *TestCase
	var errorLines []string
	var inFailureSection bool

	for _, line := range lines {
		originalLine := line
		line = strings.TrimSpace(line)
		
		if line == "" {
			continue
		}

		// Test result line
		if match := testResultRegex.FindStringSubmatch(line); match != nil {
			file := match[1]
			testName := match[2]
			status := strings.ToLower(match[3])

			test := TestCase{
				Name:   testName,
				File:   file,
				Status: status,
			}

			// Count results
			switch status {
			case "passed":
				summary.Passed++
			case "failed", "error":
				summary.Failed++
			case "skipped":
				summary.Skipped++
			}
			summary.Total++

			tests = append(tests, test)
		}

		// Failure section header
		if match := failureHeaderRegex.FindStringSubmatch(line); match != nil {
			testName := match[1]
			inFailureSection = true
			
			// Find the corresponding test
			for i := range tests {
				if strings.Contains(testName, tests[i].Name) {
					currentFailure = &tests[i]
					break
				}
			}
			errorLines = []string{}
			continue
		}

		// End of failure section
		if strings.HasPrefix(line, "=") && inFailureSection {
			if currentFailure != nil && len(errorLines) > 0 {
				currentFailure.Error = &TestError{
					Message:    strings.Join(errorLines, "\n"),
					Type:       "assertion_error",
					StackTrace: strings.Join(errorLines, "\n"),
				}
			}
			inFailureSection = false
			currentFailure = nil
			errorLines = []string{}
		}

		// Collect error lines
		if inFailureSection && currentFailure != nil {
			errorLines = append(errorLines, originalLine)
		}

		// Summary line
		if match := summaryRegex.FindStringSubmatch(line); match != nil {
			failed, _ := strconv.Atoi(match[1])
			passed, _ := strconv.Atoi(match[2])
			summary.Failed = failed
			summary.Passed = passed
			summary.Total = failed + passed + summary.Skipped
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
			"python_framework": "pytest",
		},
	}

	return result, nil
}

// SupportsCoverage returns true as pytest supports coverage
func (p *PythonTestFramework) SupportsCoverage() bool {
	return true
}

// SupportsParallel returns true as pytest supports parallel testing with pytest-xdist
func (p *PythonTestFramework) SupportsParallel() bool {
	return true
}