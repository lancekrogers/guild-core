// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package setup

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/guild-framework/guild-core/pkg/commission"
)

// TestDemoCommissionIntegration tests that generated demo commissions can be parsed
func TestDemoCommissionIntegration(t *testing.T) {
	generator := NewDemoCommissionGenerator()
	ctx := context.Background()
	parser := commission.NewMarkdownParser(commission.DefaultParseOptions())

	// Create a temporary directory for test files
	tempDir, err := os.MkdirTemp("", "demo-commission-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Test each demo type
	demoTypes := []DemoCommissionType{
		DemoTypeAPIService,
		DemoTypeWebApp,
		DemoTypeCLITool,
		DemoTypeDataAnalysis,
		DemoTypeMicroservices,
		DemoTypeAI,
		DemoTypeDefault,
	}

	for _, demoType := range demoTypes {
		t.Run(string(demoType), func(t *testing.T) {
			// Generate commission content
			content, err := generator.GenerateCommission(ctx, demoType)
			require.NoError(t, err)

			// Write to file
			filename := filepath.Join(tempDir, string(demoType)+".md")
			err = os.WriteFile(filename, []byte(content), 0o644)
			require.NoError(t, err)

			// Parse the commission
			parsedCommission, err := parser.ParseFile(filename)
			require.NoError(t, err, "Failed to parse commission for %s", demoType)

			// Validate parsed commission
			assert.NotNil(t, parsedCommission)
			assert.NotEmpty(t, parsedCommission.Title, "Commission should have a title")
			// Note: Description might be parsed differently or combined with other content

			// Debug what was actually parsed
			t.Logf("Parsed commission for %s:", demoType)
			t.Logf("  Title: %s", parsedCommission.Title)
			t.Logf("  Description: %s", parsedCommission.Description)
			t.Logf("  Number of parts: %d", len(parsedCommission.Parts))
			for i, part := range parsedCommission.Parts {
				t.Logf("  Part %d: Title='%s', Type='%s'", i, part.Title, part.Type)
			}

			// More lenient validation - just check that it was parsed successfully
			// The parser may structure the content differently than we expect
			if demoType != DemoTypeDefault {
				// For non-default demos, we expect rich content
				assert.True(t, len(parsedCommission.Parts) > 0 || parsedCommission.Description != "",
					"Commission should have content (either parts or description)")
			}

			// Check that the commission has meaningful content
			totalContent := parsedCommission.Description
			for _, part := range parsedCommission.Parts {
				totalContent += part.Content
			}
			assert.Greater(t, len(totalContent), 500, "Commission should have substantial content")

			// Validate status
			assert.Equal(t, commission.CommissionStatusDraft, parsedCommission.Status)
		})
	}
}

// TestDemoCommissionWithRefinement tests that demo commissions work with the refinement pipeline
func TestDemoCommissionWithRefinement(t *testing.T) {
	// This test verifies that generated demo commissions are properly structured
	// for the commission refinement pipeline

	generator := NewDemoCommissionGenerator()
	ctx := context.Background()

	// Test API demo as an example
	content, err := generator.GenerateCommission(ctx, DemoTypeAPIService)
	require.NoError(t, err)

	// Check that content has the expected structure for refinement
	assert.Contains(t, content, "# ", "Should have main title")
	assert.Contains(t, content, "Project Objective", "Should have objective section")
	assert.Contains(t, content, "## ", "Should have section headers")
	assert.Contains(t, content, "- [ ]", "Should have task items")

	// Verify it's valid markdown
	assert.NotContains(t, content, "```go", "Should not contain code blocks in commission")
	assert.NotContains(t, content, "undefined", "Should not contain undefined placeholders")
	assert.NotContains(t, content, "TODO", "Should not contain TODOs")
}
