// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package worktree

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/lancekrogers/guild/pkg/gerror"
)

// MergeValidator validates merges before they are executed
type MergeValidator struct {
	testRunner  *TestRunner
	codeQuality *QualityChecker
}

// NewMergeValidator creates a new merge validator
func NewMergeValidator() *MergeValidator {
	return &MergeValidator{
		testRunner:  NewTestRunner(),
		codeQuality: NewQualityChecker(),
	}
}

// ValidateMerge validates a worktree before merging
func (mv *MergeValidator) ValidateMerge(ctx context.Context, wt *Worktree) (*MergeValidation, error) {
	if ctx.Err() != nil {
		return nil, gerror.Wrap(ctx.Err(), gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("git.worktree.validation").
			WithOperation("ValidateMerge")
	}

	validation := &MergeValidation{
		WorktreeID: wt.ID,
		Valid:      true,
		StartedAt:  time.Now(),
	}

	// Run tests
	testResult, err := mv.testRunner.RunTests(ctx, wt.Path)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to run tests").
			WithComponent("git.worktree.validation").
			WithOperation("ValidateMerge").
			WithDetails("worktree_id", wt.ID)
	}

	validation.TestResult = testResult
	if !testResult.Passed {
		validation.Valid = false
		validation.Issues = append(validation.Issues, MergeIssue{
			Type:     "test_failure",
			Severity: "error",
			Message:  fmt.Sprintf("%d tests failed", testResult.Failed),
			Details:  testResult.Failures,
		})
	}

	// Check code quality
	qualityResult, err := mv.codeQuality.Check(ctx, wt.Path)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to check code quality").
			WithComponent("git.worktree.validation").
			WithOperation("ValidateMerge").
			WithDetails("worktree_id", wt.ID)
	}

	validation.QualityResult = qualityResult
	if qualityResult.Score < 7.0 {
		validation.Valid = false
		validation.Issues = append(validation.Issues, MergeIssue{
			Type:     "quality_degradation",
			Severity: "warning",
			Message:  fmt.Sprintf("Code quality score: %.1f/10 (threshold: 7.0)", qualityResult.Score),
			Details:  qualityResult.Issues,
		})
	}

	// Check for merge artifacts
	artifacts, err := mv.checkMergeArtifacts(ctx, wt)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to check merge artifacts").
			WithComponent("git.worktree.validation").
			WithOperation("ValidateMerge").
			WithDetails("worktree_id", wt.ID)
	}

	if len(artifacts) > 0 {
		validation.Valid = false
		validation.Issues = append(validation.Issues, MergeIssue{
			Type:     "merge_artifacts",
			Severity: "error",
			Message:  "Unresolved merge conflicts detected",
			Details:  artifacts,
		})
	}

	// Check for security issues
	securityIssues, err := mv.checkSecurity(ctx, wt)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to check security").
			WithComponent("git.worktree.validation").
			WithOperation("ValidateMerge").
			WithDetails("worktree_id", wt.ID)
	}

	if len(securityIssues) > 0 {
		validation.Valid = false
		validation.Issues = append(validation.Issues, MergeIssue{
			Type:     "security_issues",
			Severity: "error",
			Message:  fmt.Sprintf("Found %d security issues", len(securityIssues)),
			Details:  securityIssues,
		})
	}

	validation.CompletedAt = time.Now()
	return validation, nil
}

// checkMergeArtifacts checks for unresolved merge conflicts
func (mv *MergeValidator) checkMergeArtifacts(ctx context.Context, wt *Worktree) ([]interface{}, error) {
	var artifacts []interface{}

	// Check for conflict markers in files
	cmd := exec.CommandContext(ctx, "grep", "-r", "-n", "^<<<<<<<\\|^=======\\|^>>>>>>>", wt.Path)
	output, err := cmd.Output()

	// grep returns exit code 1 when no matches found, which is good
	if err != nil && cmd.ProcessState.ExitCode() != 1 {
		return nil, err
	}

	if len(output) > 0 {
		lines := strings.Split(strings.TrimSpace(string(output)), "\n")
		for _, line := range lines {
			artifacts = append(artifacts, map[string]interface{}{
				"type":     "conflict_marker",
				"location": line,
			})
		}
	}

	// Check for .orig files
	cmd = exec.CommandContext(ctx, "find", wt.Path, "-name", "*.orig")
	output, err = cmd.Output()
	if err != nil {
		return artifacts, nil // Continue even if find fails
	}

	if len(output) > 0 {
		files := strings.Split(strings.TrimSpace(string(output)), "\n")
		for _, file := range files {
			if file != "" {
				artifacts = append(artifacts, map[string]interface{}{
					"type": "backup_file",
					"file": file,
				})
			}
		}
	}

	return artifacts, nil
}

// checkSecurity performs basic security checks
func (mv *MergeValidator) checkSecurity(ctx context.Context, wt *Worktree) ([]interface{}, error) {
	var issues []interface{}

	// Check for potential secrets
	secretPatterns := []string{
		"password\\s*=",
		"api[_-]?key\\s*=",
		"secret\\s*=",
		"token\\s*=",
		"-----BEGIN.*PRIVATE KEY-----",
	}

	for _, pattern := range secretPatterns {
		cmd := exec.CommandContext(ctx, "grep", "-r", "-i", "-n", pattern, wt.Path)
		output, err := cmd.Output()

		// grep returns exit code 1 when no matches found
		if err != nil && cmd.ProcessState.ExitCode() != 1 {
			continue
		}

		if len(output) > 0 {
			lines := strings.Split(strings.TrimSpace(string(output)), "\n")
			for _, line := range lines {
				issues = append(issues, map[string]interface{}{
					"type":     "potential_secret",
					"pattern":  pattern,
					"location": line,
				})
			}
		}
	}

	return issues, nil
}

// MergeValidation contains the result of merge validation
type MergeValidation struct {
	WorktreeID    string         `json:"worktree_id"`
	Valid         bool           `json:"valid"`
	Issues        []MergeIssue   `json:"issues"`
	TestResult    *TestResult    `json:"test_result,omitempty"`
	QualityResult *QualityResult `json:"quality_result,omitempty"`
	StartedAt     time.Time      `json:"started_at"`
	CompletedAt   time.Time      `json:"completed_at"`
}

// MergeIssue represents an issue found during merge validation
type MergeIssue struct {
	Type     string      `json:"type"`
	Severity string      `json:"severity"`
	Message  string      `json:"message"`
	Details  interface{} `json:"details,omitempty"`
}

// TestRunner runs tests in a worktree
type TestRunner struct{}

// NewTestRunner creates a new test runner
func NewTestRunner() *TestRunner {
	return &TestRunner{}
}

// RunTests runs tests in the given path
func (tr *TestRunner) RunTests(ctx context.Context, path string) (*TestResult, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	result := &TestResult{
		StartedAt: time.Now(),
	}

	// Try different test commands based on what's available
	testCommands := [][]string{
		{"make", "test"},
		{"npm", "test"},
		{"go", "test", "./..."},
		{"python", "-m", "pytest"},
		{"mvn", "test"},
	}

	var cmd *exec.Cmd
	var testCmd []string

	// Find an available test command
	for _, cmdArgs := range testCommands {
		if tr.commandExists(cmdArgs[0]) && tr.hasTestConfig(path, cmdArgs[0]) {
			cmd = exec.CommandContext(ctx, cmdArgs[0], cmdArgs[1:]...)
			testCmd = cmdArgs
			break
		}
	}

	if cmd == nil {
		// No test runner found
		result.Passed = true
		result.Message = "No test runner found"
		result.CompletedAt = time.Now()
		return result, nil
	}

	cmd.Dir = path
	output, err := cmd.CombinedOutput()
	result.Output = string(output)
	result.CompletedAt = time.Now()

	if err != nil {
		result.Passed = false
		result.Message = "Tests failed"
		result.Failures = tr.parseTestFailures(string(output), testCmd[0])
		result.Failed = len(result.Failures)
	} else {
		result.Passed = true
		result.Message = "All tests passed"
	}

	// Parse test statistics if available
	result.Total, _ = tr.parseTestStats(string(output), testCmd[0])

	return result, nil
}

// commandExists checks if a command exists
func (tr *TestRunner) commandExists(cmd string) bool {
	_, err := exec.LookPath(cmd)
	return err == nil
}

// hasTestConfig checks if the directory has test configuration for the command
func (tr *TestRunner) hasTestConfig(path, cmd string) bool {
	switch cmd {
	case "make":
		return tr.fileExists(filepath.Join(path, "Makefile"))
	case "npm":
		return tr.fileExists(filepath.Join(path, "package.json"))
	case "go":
		return tr.fileExists(filepath.Join(path, "go.mod"))
	case "python":
		return tr.fileExists(filepath.Join(path, "pytest.ini")) ||
			tr.fileExists(filepath.Join(path, "setup.py")) ||
			tr.fileExists(filepath.Join(path, "pyproject.toml"))
	case "mvn":
		return tr.fileExists(filepath.Join(path, "pom.xml"))
	}
	return false
}

// fileExists checks if a file exists
func (tr *TestRunner) fileExists(path string) bool {
	_, err := exec.Command("test", "-f", path).Output()
	return err == nil
}

// parseTestFailures parses test failures from output
func (tr *TestRunner) parseTestFailures(output, runner string) []string {
	var failures []string
	lines := strings.Split(output, "\n")

	switch runner {
	case "go":
		for _, line := range lines {
			if strings.Contains(line, "FAIL:") {
				failures = append(failures, strings.TrimSpace(line))
			}
		}
	case "npm":
		for _, line := range lines {
			if strings.Contains(line, "✗") || strings.Contains(line, "FAIL") {
				failures = append(failures, strings.TrimSpace(line))
			}
		}
	default:
		// Generic failure parsing
		for _, line := range lines {
			if strings.Contains(strings.ToUpper(line), "FAIL") ||
				strings.Contains(strings.ToUpper(line), "ERROR") {
				failures = append(failures, strings.TrimSpace(line))
			}
		}
	}

	return failures
}

// parseTestStats parses test statistics from output
func (tr *TestRunner) parseTestStats(output, runner string) (total int, passed int) {
	lines := strings.Split(output, "\n")

	switch runner {
	case "go":
		for _, line := range lines {
			if strings.Contains(line, "coverage:") {
				// Try to extract test count from coverage line
				// This is a simplified approach
				return 1, 1
			}
		}
	case "npm":
		for _, line := range lines {
			if strings.Contains(line, "Tests:") {
				// Parse "Tests: 5 passed, 5 total"
				parts := strings.Fields(line)
				for i, part := range parts {
					if part == "passed," && i > 0 {
						if p, err := strconv.Atoi(parts[i-1]); err == nil {
							passed = p
						}
					}
					if part == "total" && i > 0 {
						if t, err := strconv.Atoi(parts[i-1]); err == nil {
							total = t
						}
					}
				}
			}
		}
	}

	return total, passed
}

// TestResult contains the result of running tests
type TestResult struct {
	Passed      bool      `json:"passed"`
	Total       int       `json:"total"`
	Failed      int       `json:"failed"`
	Message     string    `json:"message"`
	Output      string    `json:"output"`
	Failures    []string  `json:"failures"`
	StartedAt   time.Time `json:"started_at"`
	CompletedAt time.Time `json:"completed_at"`
}

// QualityChecker checks code quality
type QualityChecker struct{}

// NewQualityChecker creates a new quality checker
func NewQualityChecker() *QualityChecker {
	return &QualityChecker{}
}

// Check checks code quality in the given path
func (qc *QualityChecker) Check(ctx context.Context, path string) (*QualityResult, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	result := &QualityResult{
		StartedAt: time.Now(),
		Score:     8.0, // Default good score
		Issues:    []QualityIssue{},
	}

	// Check different language-specific linters
	linters := [][]string{
		{"golangci-lint", "run", "--out-format", "json"},
		{"eslint", ".", "--format", "json"},
		{"flake8", ".", "--format=json"},
		{"rubocop", "--format", "json"},
	}

	for _, linterCmd := range linters {
		if qc.commandExists(linterCmd[0]) {
			if err := qc.runLinter(ctx, path, linterCmd, result); err != nil {
				// Continue with other linters even if one fails
				continue
			}
		}
	}

	// Calculate final score based on issues
	result.Score = qc.calculateScore(result.Issues)
	result.CompletedAt = time.Now()

	return result, nil
}

// runLinter runs a specific linter
func (qc *QualityChecker) runLinter(ctx context.Context, path string, linterCmd []string, result *QualityResult) error {
	cmd := exec.CommandContext(ctx, linterCmd[0], linterCmd[1:]...)
	cmd.Dir = path

	output, _ := cmd.CombinedOutput()

	// Many linters return non-zero exit codes when issues are found
	// Don't treat this as an error unless the command truly failed

	if len(output) > 0 {
		issues := qc.parseLinterOutput(string(output), linterCmd[0])
		result.Issues = append(result.Issues, issues...)
	}

	return nil
}

// parseLinterOutput parses linter output into quality issues
func (qc *QualityChecker) parseLinterOutput(output, linter string) []QualityIssue {
	var issues []QualityIssue

	// Simplified parsing - in practice would parse JSON output
	lines := strings.Split(output, "\n")

	for _, line := range lines {
		if strings.Contains(line, "error") || strings.Contains(line, "warning") {
			severity := "warning"
			if strings.Contains(line, "error") {
				severity = "error"
			}

			issues = append(issues, QualityIssue{
				Type:     "linter_issue",
				Severity: severity,
				Message:  strings.TrimSpace(line),
				Linter:   linter,
			})
		}
	}

	return issues
}

// calculateScore calculates a quality score based on issues
func (qc *QualityChecker) calculateScore(issues []QualityIssue) float64 {
	score := 10.0

	for _, issue := range issues {
		switch issue.Severity {
		case "error":
			score -= 1.0
		case "warning":
			score -= 0.5
		case "info":
			score -= 0.1
		}
	}

	if score < 0 {
		score = 0
	}

	return score
}

// commandExists checks if a command exists
func (qc *QualityChecker) commandExists(cmd string) bool {
	_, err := exec.LookPath(cmd)
	return err == nil
}

// QualityResult contains the result of code quality checking
type QualityResult struct {
	Score       float64        `json:"score"`
	Issues      []QualityIssue `json:"issues"`
	StartedAt   time.Time      `json:"started_at"`
	CompletedAt time.Time      `json:"completed_at"`
}

// QualityIssue represents a code quality issue
type QualityIssue struct {
	Type     string `json:"type"`
	Severity string `json:"severity"`
	Message  string `json:"message"`
	File     string `json:"file,omitempty"`
	Line     int    `json:"line,omitempty"`
	Linter   string `json:"linter,omitempty"`
}
