package evaluation

import (
	"context"
	"strings"
	"testing"

	"github.com/guild-ventures/guild-core/pkg/prompts/standard"
)

func TestPromptEvaluator(t *testing.T) {
	// Create a mock enhanced prompt manager
	manager, err := standard.NewEnhancedPromptManager()
	if err != nil {
		t.Fatalf("Failed to create prompt manager: %v", err)
	}

	evaluator := NewPromptEvaluator(manager)

	// Register test cases
	tests := []*PromptTest{
		{
			Name:     "Basic objective creation",
			PromptID: "commission.creation",
			TestData: map[string]interface{}{
				"Description": "Build a web scraper",
			},
			ExpectedOutput: &ContainsAssertion{
				Substring: "objective",
			},
		},
		{
			Name:     "Objective with context",
			PromptID: "commission.creation",
			TestData: map[string]interface{}{
				"Description": "Build a web scraper",
				"UserContext": "Using Python and BeautifulSoup",
			},
			ExpectedOutput: &MultiAssertion{
				Assertions: []PromptAssertion{
					&ContainsAssertion{Substring: "objective"},
					&ContainsAssertion{Substring: "Context"},
				},
				RequireAll: true,
			},
		},
		{
			Name:     "Quality check",
			PromptID: "commission.creation",
			TestData: map[string]interface{}{
				"Description": "Create a comprehensive data pipeline",
			},
			ExpectedOutput: &QualityAssertion{
				MinSentences:      3,
				RequireFormatting: true,
			},
		},
	}

	for _, test := range tests {
		err := evaluator.RegisterTest(test)
		if err != nil {
			t.Errorf("Failed to register test %s: %v", test.Name, err)
		}
	}

	// Run evaluation
	ctx := context.Background()
	result, err := evaluator.EvaluatePrompt(ctx, "commission.creation")
	if err != nil {
		t.Fatalf("Failed to evaluate prompt: %v", err)
	}

	// Check results
	if result.TotalTests != 3 {
		t.Errorf("Expected 3 tests, got %d", result.TotalTests)
	}

	if result.SuccessRate == 0 {
		t.Error("Expected at least some tests to pass")
	}

	// Generate report
	report := evaluator.GenerateReport()
	if !strings.Contains(report, "Prompt Evaluation Report") {
		t.Error("Report should contain title")
	}
	if !strings.Contains(report, "commission.creation") {
		t.Error("Report should contain prompt ID")
	}
}

func TestAssertions(t *testing.T) {
	testCases := []struct {
		name      string
		assertion PromptAssertion
		output    string
		shouldPass bool
	}{
		// ContainsAssertion tests
		{
			name:       "contains assertion - pass",
			assertion:  &ContainsAssertion{Substring: "hello"},
			output:     "hello world",
			shouldPass: true,
		},
		{
			name:       "contains assertion - fail",
			assertion:  &ContainsAssertion{Substring: "goodbye"},
			output:     "hello world",
			shouldPass: false,
		},
		// LengthAssertion tests
		{
			name:       "length assertion - pass",
			assertion:  &LengthAssertion{MinLength: 5, MaxLength: 20},
			output:     "hello world",
			shouldPass: true,
		},
		{
			name:       "length assertion - too short",
			assertion:  &LengthAssertion{MinLength: 20},
			output:     "hello",
			shouldPass: false,
		},
		{
			name:       "length assertion - too long",
			assertion:  &LengthAssertion{MaxLength: 5},
			output:     "hello world",
			shouldPass: false,
		},
		// StructureAssertion tests
		{
			name: "structure assertion - pass",
			assertion: &StructureAssertion{
				RequiredSections: []string{"# Goal", "# Context", "# Requirements"},
			},
			output:     "# Goal\nTest\n# Context\nTest\n# Requirements\nTest",
			shouldPass: true,
		},
		{
			name: "structure assertion - missing section",
			assertion: &StructureAssertion{
				RequiredSections: []string{"# Goal", "# Missing"},
			},
			output:     "# Goal\nTest\n# Context\nTest",
			shouldPass: false,
		},
		// QualityAssertion tests
		{
			name: "quality assertion - pass",
			assertion: &QualityAssertion{
				MinSentences:      2,
				RequireExamples:   true,
				RequireFormatting: true,
			},
			output:     "# Title\nThis is a test. For example, we can do this. **Bold text**",
			shouldPass: true,
		},
		{
			name: "quality assertion - no examples",
			assertion: &QualityAssertion{
				RequireExamples: true,
			},
			output:     "This is just plain text.",
			shouldPass: false,
		},
		// MultiAssertion tests
		{
			name: "multi assertion - all required, pass",
			assertion: &MultiAssertion{
				Assertions: []PromptAssertion{
					&ContainsAssertion{Substring: "hello"},
					&ContainsAssertion{Substring: "world"},
				},
				RequireAll: true,
			},
			output:     "hello world",
			shouldPass: true,
		},
		{
			name: "multi assertion - all required, fail",
			assertion: &MultiAssertion{
				Assertions: []PromptAssertion{
					&ContainsAssertion{Substring: "hello"},
					&ContainsAssertion{Substring: "goodbye"},
				},
				RequireAll: true,
			},
			output:     "hello world",
			shouldPass: false,
		},
		{
			name: "multi assertion - any required, pass",
			assertion: &MultiAssertion{
				Assertions: []PromptAssertion{
					&ContainsAssertion{Substring: "hello"},
					&ContainsAssertion{Substring: "goodbye"},
				},
				RequireAll: false,
			},
			output:     "hello world",
			shouldPass: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.assertion.Assert(tc.output)
			if tc.shouldPass && err != nil {
				t.Errorf("Expected assertion to pass, but got error: %v", err)
			}
			if !tc.shouldPass && err == nil {
				t.Error("Expected assertion to fail, but it passed")
			}
		})
	}
}

func TestRegexAssertion(t *testing.T) {
	// Test valid regex
	assertion, err := NewRegexAssertion(`#\s+\w+`)
	if err != nil {
		t.Fatalf("Failed to create regex assertion: %v", err)
	}

	if err := assertion.Assert("# Title"); err != nil {
		t.Errorf("Expected regex to match, but got error: %v", err)
	}

	if err := assertion.Assert("No heading here"); err == nil {
		t.Error("Expected regex to not match, but it did")
	}

	// Test invalid regex
	_, err = NewRegexAssertion(`[`)
	if err == nil {
		t.Error("Expected error for invalid regex")
	}
}

func TestEvaluationReport(t *testing.T) {
	manager, _ := standard.NewEnhancedPromptManager()
	evaluator := NewPromptEvaluator(manager)

	// Register and run a simple test
	test := &PromptTest{
		Name:     "Test",
		PromptID: "commission.creation",
		TestData: map[string]interface{}{
			"Description": "Test",
		},
		ExpectedOutput: &ContainsAssertion{Substring: "fail"},
	}
	evaluator.RegisterTest(test)
	evaluator.EvaluatePrompt(context.Background(), "commission.creation")

	// Check report contains expected elements
	report := evaluator.GenerateReport()
	expectedElements := []string{
		"Prompt Evaluation Report",
		"commission.creation",
		"Total Tests:",
		"Success Rate:",
		"Failures:",
	}

	for _, element := range expectedElements {
		if !strings.Contains(report, element) {
			t.Errorf("Report missing expected element: %s", element)
		}
	}
}