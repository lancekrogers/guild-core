package evaluation

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/guild-ventures/guild-core/pkg/gerror"
	"github.com/guild-ventures/guild-core/pkg/prompts/standard"
)

// PromptEvaluator evaluates prompt effectiveness and quality
type PromptEvaluator struct {
	manager *standard.EnhancedPromptManager
	tests   map[string][]*PromptTest
	results map[string]*EvaluationResult
}

// PromptTest defines a test case for a prompt
type PromptTest struct {
	Name           string
	PromptID       string
	TestData       map[string]interface{}
	ExpectedOutput PromptAssertion
	Tags           []string
}

// PromptAssertion defines expected output characteristics
type PromptAssertion interface {
	Assert(output string) error
	Description() string
}

// EvaluationResult contains the results of evaluating a prompt
type EvaluationResult struct {
	PromptID       string
	TotalTests     int
	PassedTests    int
	FailedTests    int
	SuccessRate    float64
	AverageTokens  int
	ExecutionTime  time.Duration
	FailureDetails []TestFailure
}

// TestFailure contains details about a failed test
type TestFailure struct {
	TestName string
	Error    string
	Output   string
}

// NewPromptEvaluator creates a new prompt evaluator
func NewPromptEvaluator(manager *standard.EnhancedPromptManager) *PromptEvaluator {
	return &PromptEvaluator{
		manager: manager,
		tests:   make(map[string][]*PromptTest),
		results: make(map[string]*EvaluationResult),
	}
}

// RegisterTest registers a test for a prompt
func (e *PromptEvaluator) RegisterTest(test *PromptTest) error {
	// Validate the prompt exists
	if _, err := e.manager.GetMetadata(test.PromptID); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeNotFound, "prompt not found").
			WithComponent("prompts").
			WithOperation("RegisterTest").
			WithDetails("prompt_id", test.PromptID)
	}

	// Validate test data
	if err := e.manager.ValidatePrompt(test.PromptID, test.TestData); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeValidation, "invalid test data").
			WithComponent("prompts").
			WithOperation("RegisterTest").
			WithDetails("prompt_id", test.PromptID)
	}

	e.tests[test.PromptID] = append(e.tests[test.PromptID], test)
	return nil
}

// EvaluatePrompt runs all tests for a specific prompt
func (e *PromptEvaluator) EvaluatePrompt(ctx context.Context, promptID string) (*EvaluationResult, error) {
	tests, exists := e.tests[promptID]
	if !exists {
		return nil, gerror.New(gerror.ErrCodeNotFound, "no tests registered for prompt", nil).
			WithComponent("prompts").
			WithOperation("EvaluatePrompt").
			WithDetails("prompt_id", promptID)
	}

	result := &EvaluationResult{
		PromptID:   promptID,
		TotalTests: len(tests),
	}

	startTime := time.Now()
	totalTokens := 0

	for _, test := range tests {
		// Render the prompt
		rendered, err := e.manager.RenderPrompt(promptID, test.TestData)
		if err != nil {
			result.FailedTests++
			result.FailureDetails = append(result.FailureDetails, TestFailure{
				TestName: test.Name,
				Error:    fmt.Sprintf("render error: %v", err),
			})
			continue
		}

		// Count tokens (simple approximation)
		tokens := len(strings.Fields(rendered))
		totalTokens += tokens

		// In a real implementation, you would call the LLM here
		// For now, we'll simulate the output
		output := simulateLLMOutput(rendered)

		// Assert the output
		if err := test.ExpectedOutput.Assert(output); err != nil {
			result.FailedTests++
			result.FailureDetails = append(result.FailureDetails, TestFailure{
				TestName: test.Name,
				Error:    err.Error(),
				Output:   output,
			})
		} else {
			result.PassedTests++
		}
	}

	result.ExecutionTime = time.Since(startTime)
	result.SuccessRate = float64(result.PassedTests) / float64(result.TotalTests)
	if result.TotalTests > 0 {
		result.AverageTokens = totalTokens / result.TotalTests
	}

	e.results[promptID] = result
	return result, nil
}

// EvaluateAll runs tests for all registered prompts
func (e *PromptEvaluator) EvaluateAll(ctx context.Context) (map[string]*EvaluationResult, error) {
	results := make(map[string]*EvaluationResult)

	for promptID := range e.tests {
		result, err := e.EvaluatePrompt(ctx, promptID)
		if err != nil {
			return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "error evaluating prompt").
				WithComponent("prompts").
				WithOperation("EvaluateAll").
				WithDetails("prompt_id", promptID)
		}
		results[promptID] = result
	}

	return results, nil
}

// GetResult retrieves the evaluation result for a prompt
func (e *PromptEvaluator) GetResult(promptID string) (*EvaluationResult, bool) {
	result, exists := e.results[promptID]
	return result, exists
}

// GenerateReport generates a human-readable evaluation report
func (e *PromptEvaluator) GenerateReport() string {
	var report strings.Builder

	report.WriteString("# Prompt Evaluation Report\n\n")
	report.WriteString(fmt.Sprintf("Generated: %s\n\n", time.Now().Format(time.RFC3339)))

	for promptID, result := range e.results {
		report.WriteString(fmt.Sprintf("## Prompt: %s\n", promptID))
		report.WriteString(fmt.Sprintf("- Total Tests: %d\n", result.TotalTests))
		report.WriteString(fmt.Sprintf("- Passed: %d\n", result.PassedTests))
		report.WriteString(fmt.Sprintf("- Failed: %d\n", result.FailedTests))
		report.WriteString(fmt.Sprintf("- Success Rate: %.2f%%\n", result.SuccessRate*100))
		report.WriteString(fmt.Sprintf("- Average Tokens: %d\n", result.AverageTokens))
		report.WriteString(fmt.Sprintf("- Execution Time: %v\n", result.ExecutionTime))

		if len(result.FailureDetails) > 0 {
			report.WriteString("\n### Failures:\n")
			for _, failure := range result.FailureDetails {
				report.WriteString(fmt.Sprintf("- **%s**: %s\n", failure.TestName, failure.Error))
			}
		}
		report.WriteString("\n")
	}

	return report.String()
}

// simulateLLMOutput simulates LLM output for testing purposes
func simulateLLMOutput(prompt string) string {
	// In a real implementation, this would call an actual LLM
	// For testing, we'll return a simple response
	if strings.Contains(prompt, "objective") {
		return "# Goal\nTest objective\n\n# Context\nTest context\n\n# Requirements\n- Test requirement"
	}
	return "Simulated response for: " + prompt[:50] + "..."
}
