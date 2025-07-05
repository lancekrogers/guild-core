// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package dev

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/lancekrogers/guild/pkg/gerror"
	"github.com/lancekrogers/guild/tools"
)

// TestRunnerTool executes tests and analyzes results for AI assistance
type TestRunnerTool struct {
	*tools.BaseTool
	frameworks map[string]TestFramework
}

// TestRunnerInput represents input parameters for running tests
type TestRunnerInput struct {
	Path        string            `json:"path,omitempty"`        // Path to test file/directory
	Pattern     string            `json:"pattern,omitempty"`     // Test name pattern to match
	Framework   string            `json:"framework,omitempty"`   // Test framework (auto, go, jest, pytest, etc.)
	Coverage    bool              `json:"coverage,omitempty"`    // Generate coverage report
	Verbose     bool              `json:"verbose,omitempty"`     // Verbose output
	Timeout     int               `json:"timeout,omitempty"`     // Timeout in seconds
	Parallel    bool              `json:"parallel,omitempty"`    // Run tests in parallel
	Environment map[string]string `json:"environment,omitempty"` // Environment variables
	Tags        []string          `json:"tags,omitempty"`        // Test tags to include/exclude
	FailFast    bool              `json:"fail_fast,omitempty"`   // Stop on first failure
}

// TestResult represents the complete result of test execution
type TestResult struct {
	Framework   string            `json:"framework"`
	Summary     TestSummary       `json:"summary"`
	Tests       []TestCase        `json:"tests"`
	Coverage    *CoverageReport   `json:"coverage,omitempty"`
	Duration    time.Duration     `json:"duration"`
	Output      string            `json:"output"`
	Command     string            `json:"command"`
	ExitCode    int               `json:"exit_code"`
	Suggestions []TestSuggestion  `json:"suggestions,omitempty"`
	Metadata    map[string]string `json:"metadata"`
}

// TestSummary provides high-level test statistics
type TestSummary struct {
	Total           int     `json:"total"`
	Passed          int     `json:"passed"`
	Failed          int     `json:"failed"`
	Skipped         int     `json:"skipped"`
	Success         bool    `json:"success"`
	CoveragePercent float64 `json:"coverage_percent,omitempty"`
}

// TestCase represents a single test case
type TestCase struct {
	Name     string        `json:"name"`
	Package  string        `json:"package,omitempty"`
	Status   string        `json:"status"` // passed, failed, skipped
	Duration time.Duration `json:"duration"`
	Error    *TestError    `json:"error,omitempty"`
	Output   string        `json:"output,omitempty"`
	File     string        `json:"file,omitempty"`
	Line     int           `json:"line,omitempty"`
}

// TestError represents test failure information
type TestError struct {
	Message    string `json:"message"`
	Type       string `json:"type"`
	StackTrace string `json:"stack_trace,omitempty"`
	Expected   string `json:"expected,omitempty"`
	Actual     string `json:"actual,omitempty"`
	Diff       string `json:"diff,omitempty"`
}

// CoverageReport represents code coverage information
type CoverageReport struct {
	TotalLines   int                    `json:"total_lines"`
	CoveredLines int                    `json:"covered_lines"`
	Percentage   float64                `json:"percentage"`
	Files        []FileCoverage         `json:"files,omitempty"`
	Uncovered    []UncoveredLine        `json:"uncovered,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// FileCoverage represents coverage for a single file
type FileCoverage struct {
	File       string  `json:"file"`
	Lines      int     `json:"lines"`
	Covered    int     `json:"covered"`
	Percentage float64 `json:"percentage"`
}

// UncoveredLine represents an uncovered line
type UncoveredLine struct {
	File   string `json:"file"`
	Line   int    `json:"line"`
	Code   string `json:"code,omitempty"`
	Reason string `json:"reason,omitempty"`
}

// TestSuggestion represents AI-generated suggestions for test improvements
type TestSuggestion struct {
	Type        string `json:"type"` // fix, improvement, coverage
	Message     string `json:"message"`
	File        string `json:"file,omitempty"`
	Line        int    `json:"line,omitempty"`
	Code        string `json:"code,omitempty"`
	Replacement string `json:"replacement,omitempty"`
}

// TestFramework interface for different testing frameworks
type TestFramework interface {
	Name() string
	Detect(path string) bool
	BuildCommand(input TestRunnerInput) ([]string, error)
	ParseOutput(output string, exitCode int) (*TestResult, error)
	SupportsCoverage() bool
	SupportsParallel() bool
}

// NewTestRunnerTool creates a new test runner tool
func NewTestRunnerTool() *TestRunnerTool {
	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"path": map[string]interface{}{
				"type":        "string",
				"description": "Path to test file or directory",
			},
			"pattern": map[string]interface{}{
				"type":        "string",
				"description": "Test name pattern to match (regex)",
			},
			"framework": map[string]interface{}{
				"type":        "string",
				"enum":        []string{"auto", "go", "jest", "pytest", "junit", "rspec", "cargo"},
				"default":     "auto",
				"description": "Test framework to use",
			},
			"coverage": map[string]interface{}{
				"type":        "boolean",
				"default":     false,
				"description": "Generate coverage report",
			},
			"verbose": map[string]interface{}{
				"type":        "boolean",
				"default":     true,
				"description": "Verbose test output",
			},
			"timeout": map[string]interface{}{
				"type":        "integer",
				"default":     300,
				"description": "Timeout in seconds",
			},
			"parallel": map[string]interface{}{
				"type":        "boolean",
				"default":     true,
				"description": "Run tests in parallel if supported",
			},
			"environment": map[string]interface{}{
				"type":        "object",
				"description": "Environment variables",
			},
			"tags": map[string]interface{}{
				"type":        "array",
				"items":       map[string]interface{}{"type": "string"},
				"description": "Test tags to include",
			},
			"fail_fast": map[string]interface{}{
				"type":        "boolean",
				"default":     false,
				"description": "Stop on first failure",
			},
		},
	}

	examples := []string{
		`{"framework": "go", "coverage": true}`,
		`{"path": "./pkg/agents/core", "pattern": "TestAgent.*", "verbose": true}`,
		`{"framework": "pytest", "path": "tests/", "coverage": true, "parallel": true}`,
		`{"framework": "jest", "pattern": "*.test.js", "coverage": true}`,
	}

	baseTool := tools.NewBaseTool(
		"test_runner",
		"Execute tests with intelligent result analysis and suggestions",
		schema,
		"development",
		false,
		examples,
	)

	// Initialize test frameworks
	frameworks := make(map[string]TestFramework)
	frameworks["go"] = &GoTestFramework{}
	frameworks["pytest"] = &PythonTestFramework{}
	frameworks["jest"] = &JavaScriptTestFramework{}
	frameworks["junit"] = &JavaTestFramework{}
	frameworks["rspec"] = &RubyTestFramework{}
	frameworks["cargo"] = &RustTestFramework{}

	return &TestRunnerTool{
		BaseTool:   baseTool,
		frameworks: frameworks,
	}
}

// Execute runs the test runner tool
func (t *TestRunnerTool) Execute(ctx context.Context, input string) (*tools.ToolResult, error) {
	var params TestRunnerInput
	if err := json.Unmarshal([]byte(input), &params); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInvalidInput, "invalid input").
			WithComponent("test_runner_tool").
			WithOperation("execute")
	}

	// Set defaults
	if params.Framework == "" {
		params.Framework = "auto"
	}
	if params.Timeout == 0 {
		params.Timeout = 300
	}
	if params.Path == "" {
		params.Path = "."
	}

	// Auto-detect framework if needed
	framework, err := t.detectOrGetFramework(params.Path, params.Framework)
	if err != nil {
		return nil, err
	}

	// Build command
	command, err := framework.BuildCommand(params)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInvalidInput, "failed to build test command").
			WithComponent("test_runner_tool").
			WithOperation("execute")
	}

	// Execute tests
	result, err := t.executeTests(ctx, command, params, framework)
	if err != nil {
		return nil, err
	}

	// Convert result to ToolResult
	resultJSON, _ := json.Marshal(result)
	metadata := map[string]string{
		"framework":   result.Framework,
		"total_tests": fmt.Sprintf("%d", result.Summary.Total),
		"passed":      fmt.Sprintf("%d", result.Summary.Passed),
		"failed":      fmt.Sprintf("%d", result.Summary.Failed),
		"success":     fmt.Sprintf("%t", result.Summary.Success),
		"duration":    result.Duration.String(),
		"coverage":    fmt.Sprintf("%.2f%%", result.Summary.CoveragePercent),
	}

	var toolErr error
	if !result.Summary.Success {
		toolErr = gerror.Newf(gerror.ErrCodeInternal, "%d test(s) failed", result.Summary.Failed).
			WithComponent("test_runner_tool").
			WithOperation("execute")
	}

	return tools.NewToolResult(string(resultJSON), metadata, toolErr, nil), nil
}

// detectOrGetFramework detects or returns the specified test framework
func (t *TestRunnerTool) detectOrGetFramework(path, frameworkName string) (TestFramework, error) {
	if frameworkName != "auto" {
		framework, exists := t.frameworks[frameworkName]
		if !exists {
			return nil, gerror.Newf(gerror.ErrCodeInvalidInput, "unsupported framework: %s", frameworkName).
				WithComponent("test_runner_tool").
				WithOperation("detectOrGetFramework")
		}
		return framework, nil
	}

	// Auto-detect framework
	for _, framework := range t.frameworks {
		if framework.Detect(path) {
			return framework, nil
		}
	}

	return nil, gerror.New(gerror.ErrCodeNotFound, "could not detect test framework", nil).
		WithComponent("test_runner_tool").
		WithOperation("detectOrGetFramework")
}

// executeTests executes the test command and parses results
func (t *TestRunnerTool) executeTests(ctx context.Context, command []string, params TestRunnerInput, framework TestFramework) (*TestResult, error) {
	// Create context with timeout
	execCtx, cancel := context.WithTimeout(ctx, time.Duration(params.Timeout)*time.Second)
	defer cancel()

	// Create command
	cmd := exec.CommandContext(execCtx, command[0], command[1:]...)
	if params.Path != "" && params.Path != "." {
		cmd.Dir = params.Path
	}

	// Set environment variables
	if len(params.Environment) > 0 {
		env := append(cmd.Env, os.Environ()...)
		for key, value := range params.Environment {
			env = append(env, fmt.Sprintf("%s=%s", key, value))
		}
		cmd.Env = env
	}

	// Execute command
	startTime := time.Now()
	output, err := cmd.CombinedOutput()
	duration := time.Since(startTime)

	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = -1
		}
	}

	// Parse output using framework-specific parser
	result, err := framework.ParseOutput(string(output), exitCode)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to parse test output").
			WithComponent("test_runner_tool").
			WithOperation("executeTests")
	}

	// Set additional result fields
	result.Duration = duration
	result.Command = strings.Join(command, " ")
	result.ExitCode = exitCode
	result.Framework = framework.Name()

	// Generate suggestions based on failures
	if result.Summary.Failed > 0 {
		result.Suggestions = t.generateSuggestions(result)
	}

	return result, nil
}

// generateSuggestions creates AI suggestions based on test failures
func (t *TestRunnerTool) generateSuggestions(result *TestResult) []TestSuggestion {
	var suggestions []TestSuggestion

	for _, test := range result.Tests {
		if test.Status == "failed" && test.Error != nil {
			suggestion := TestSuggestion{
				Type:    "fix",
				Message: t.analyzeFailure(test.Error),
				File:    test.File,
				Line:    test.Line,
			}
			suggestions = append(suggestions, suggestion)
		}
	}

	// Add coverage suggestions if coverage is low
	if result.Coverage != nil && result.Coverage.Percentage < 80 {
		suggestion := TestSuggestion{
			Type:    "coverage",
			Message: fmt.Sprintf("Test coverage is %.2f%%. Consider adding tests for uncovered code.", result.Coverage.Percentage),
		}
		suggestions = append(suggestions, suggestion)
	}

	return suggestions
}

// analyzeFailure provides intelligent analysis of test failures
func (t *TestRunnerTool) analyzeFailure(err *TestError) string {
	message := err.Message

	// Common failure patterns
	if strings.Contains(message, "assertion failed") || strings.Contains(message, "expected") {
		return "Assertion failure - check expected vs actual values. Consider updating test expectations or fixing the implementation."
	}

	if strings.Contains(message, "panic") || strings.Contains(message, "nil pointer") {
		return "Runtime panic detected - add nil checks or proper error handling before accessing objects."
	}

	if strings.Contains(message, "timeout") || strings.Contains(message, "deadline exceeded") {
		return "Test timeout - operation taking too long. Consider optimizing the code or increasing timeout values."
	}

	if strings.Contains(message, "import") || strings.Contains(message, "module") {
		return "Import/module error - check dependencies and import paths."
	}

	if strings.Contains(message, "syntax") || strings.Contains(message, "parse") {
		return "Syntax error - check code syntax and formatting."
	}

	return "Test failure - review the error message and stack trace to identify the root cause."
}

// Helper function to check if file exists
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// Helper function to find files with pattern
func findFiles(dir, pattern string) ([]string, error) {
	var files []string
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if matched, _ := filepath.Match(pattern, info.Name()); matched {
			files = append(files, path)
		}
		return nil
	})
	return files, err
}
