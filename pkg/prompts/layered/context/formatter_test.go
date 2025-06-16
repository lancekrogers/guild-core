// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package context_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/guild-ventures/guild-core/pkg/prompts/layered/context"
)

// Test implementation of Context interface
type testContext struct {
	commissionID    string
	commissionTitle string
	currentTask     context.TaskContext
	sections        []context.Section
	relatedTasks    []context.TaskContext
}

func (c *testContext) GetCommissionID() string {
	return c.commissionID
}

func (c *testContext) GetCommissionTitle() string {
	return c.commissionTitle
}

func (c *testContext) GetCurrentTask() context.TaskContext {
	return c.currentTask
}

func (c *testContext) GetRelevantSections() []context.Section {
	return c.sections
}

func (c *testContext) GetRelatedTasks() []context.TaskContext {
	return c.relatedTasks
}

func TestXMLFormatter(t *testing.T) {
	formatter, err := context.NewXMLFormatter()
	require.NoError(t, err)

	t.Run("FormatAsXML_Basic", func(t *testing.T) {
		ctx := &testContext{
			commissionID:    "test-001",
			commissionTitle: "Build Test System",
			currentTask: context.TaskContext{
				ID:            "TASK-001",
				Title:         "Implement feature",
				Description:   "Implement the test feature",
				SourceSection: "2.1",
				Priority:      "high",
				Estimate:      "4h",
			},
		}

		result, err := formatter.FormatAsXML(ctx)
		require.NoError(t, err)

		// Verify structure
		assert.Contains(t, result, "<guild-context>")
		assert.Contains(t, result, "</guild-context>")
		assert.Contains(t, result, `<commission id="test-001">`)
		assert.Contains(t, result, "<title>Build Test System</title>")
		assert.Contains(t, result, "<current-task>")
		assert.Contains(t, result, "<id>TASK-001</id>")
		assert.Contains(t, result, "<title>Implement feature</title>")
		assert.Contains(t, result, "<priority>high</priority>")
	})

	t.Run("FormatAsXML_WithDependencies", func(t *testing.T) {
		ctx := &testContext{
			commissionID:    "test-002",
			commissionTitle: "Complex System",
			currentTask: context.TaskContext{
				ID:           "TASK-002",
				Title:        "Feature with deps",
				Dependencies: []string{"TASK-001", "TASK-000"},
				Capabilities: []string{"backend", "database"},
			},
		}

		result, err := formatter.FormatAsXML(ctx)
		require.NoError(t, err)

		assert.Contains(t, result, "<dependencies>")
		assert.Contains(t, result, "<dependency>TASK-001</dependency>")
		assert.Contains(t, result, "<dependency>TASK-000</dependency>")
		assert.Contains(t, result, "<capabilities>")
		assert.Contains(t, result, "<capability>backend</capability>")
		assert.Contains(t, result, "<capability>database</capability>")
	})

	t.Run("FormatAsXML_WithSections", func(t *testing.T) {
		ctx := &testContext{
			commissionID:    "test-003",
			commissionTitle: "Documented System",
			currentTask: context.TaskContext{
				ID:    "TASK-003",
				Title: "Implement from docs",
			},
			sections: []context.Section{
				{
					Level:   2,
					Path:    "2.1",
					Title:   "Authentication",
					Content: "Build secure authentication system",
					Tasks: []context.TaskContext{
						{ID: "AUTH-001", Title: "JWT implementation"},
						{ID: "AUTH-002", Title: "OAuth integration"},
					},
				},
			},
		}

		result, err := formatter.FormatAsXML(ctx)
		require.NoError(t, err)

		assert.Contains(t, result, "<relevant-sections>")
		assert.Contains(t, result, `<section level="2" path="2.1">`)
		assert.Contains(t, result, "<title>Authentication</title>")
		assert.Contains(t, result, "<content>Build secure authentication system</content>")
		assert.Contains(t, result, "<tasks>")
		assert.Contains(t, result, `<task id="AUTH-001">JWT implementation</task>`)
	})

	t.Run("FormatAsXML_WithRelatedTasks", func(t *testing.T) {
		ctx := &testContext{
			commissionID:    "test-004",
			commissionTitle: "Connected System",
			currentTask: context.TaskContext{
				ID:    "TASK-004",
				Title: "Connected task",
			},
			relatedTasks: []context.TaskContext{
				{
					ID:           "TASK-005",
					Title:        "Related feature",
					Dependencies: []string{"TASK-003"},
				},
			},
		}

		result, err := formatter.FormatAsXML(ctx)
		require.NoError(t, err)

		assert.Contains(t, result, "<related-tasks>")
		assert.Contains(t, result, `<task id="TASK-005">`)
		assert.Contains(t, result, "<title>Related feature</title>")
		assert.Contains(t, result, "<dependency>TASK-003</dependency>")
	})
}

func TestXMLFormatterMarkdown(t *testing.T) {
	formatter, err := context.NewXMLFormatter()
	require.NoError(t, err)

	t.Run("FormatAsMarkdown_Complete", func(t *testing.T) {
		ctx := &testContext{
			commissionID:    "test-md-001",
			commissionTitle: "Markdown Test System",
			currentTask: context.TaskContext{
				ID:            "TASK-MD-001",
				Title:         "Markdown task",
				Description:   "A task to test markdown formatting",
				SourceSection: "3.1.2",
				Priority:      "medium",
				Estimate:      "2h",
				Dependencies:  []string{"TASK-MD-000"},
				Capabilities:  []string{"frontend", "react"},
			},
			sections: []context.Section{
				{
					Level:   1,
					Path:    "3",
					Title:   "Frontend Architecture",
					Content: "Overview of frontend components and structure.",
				},
				{
					Level:   2,
					Path:    "3.1",
					Title:   "Component Design",
					Content: "Detailed component specifications.",
					Tasks: []context.TaskContext{
						{ID: "UI-001", Title: "Create base components"},
						{ID: "UI-002", Title: "Implement theme system"},
					},
				},
			},
			relatedTasks: []context.TaskContext{
				{
					ID:           "TASK-MD-002",
					Title:        "Setup testing framework",
					Dependencies: []string{"TASK-MD-001"},
				},
				{
					ID:    "TASK-MD-003",
					Title: "Deploy to staging",
				},
			},
		}

		result, err := formatter.FormatAsMarkdown(ctx)
		require.NoError(t, err)

		// Check commission section
		assert.Contains(t, result, "# Commission Context")
		assert.Contains(t, result, "**Commission ID**: test-md-001")
		assert.Contains(t, result, "**Title**: Markdown Test System")

		// Check current task section
		assert.Contains(t, result, "## Current Task")
		assert.Contains(t, result, "- **ID**: TASK-MD-001")
		assert.Contains(t, result, "- **Title**: Markdown task")
		assert.Contains(t, result, "- **Description**: A task to test markdown formatting")
		assert.Contains(t, result, "- **Source**: 3.1.2")
		assert.Contains(t, result, "- **Priority**: medium")
		assert.Contains(t, result, "- **Estimate**: 2h")
		assert.Contains(t, result, "- **Dependencies**: TASK-MD-000")
		assert.Contains(t, result, "- **Required Capabilities**: frontend, react")

		// Check relevant sections
		assert.Contains(t, result, "## Relevant Documentation")
		assert.Contains(t, result, "### Frontend Architecture")
		assert.Contains(t, result, "#### Component Design")
		assert.Contains(t, result, "**Related Tasks**:")
		assert.Contains(t, result, "- UI-001: Create base components")

		// Check related tasks section
		assert.Contains(t, result, "## Related Tasks")
		assert.Contains(t, result, "- **TASK-MD-002**: Setup testing framework (depends on: TASK-MD-001)")
		assert.Contains(t, result, "- **TASK-MD-003**: Deploy to staging")
	})

	t.Run("FormatAsMarkdown_Minimal", func(t *testing.T) {
		ctx := &testContext{
			commissionID:    "min-001",
			commissionTitle: "Minimal",
			currentTask: context.TaskContext{
				ID:    "MIN-001",
				Title: "Simple task",
			},
		}

		result, err := formatter.FormatAsMarkdown(ctx)
		require.NoError(t, err)

		// Should have basic structure
		assert.Contains(t, result, "# Commission Context")
		assert.Contains(t, result, "## Current Task")

		// Should not have empty sections
		assert.NotContains(t, result, "## Relevant Documentation")
		assert.NotContains(t, result, "## Related Tasks")
	})
}

func TestXMLFormatterOptimization(t *testing.T) {
	formatter, err := context.NewXMLFormatter()
	require.NoError(t, err)

	t.Run("OptimizeForTokens_NoTruncation", func(t *testing.T) {
		content := "Short content that fits"
		result, err := formatter.OptimizeForTokens(content, 100)
		require.NoError(t, err)
		assert.Equal(t, content, result)
	})

	t.Run("OptimizeForTokens_Truncation", func(t *testing.T) {
		// Create content that exceeds token limit
		longContent := strings.Repeat("This is a long sentence. ", 100)
		result, err := formatter.OptimizeForTokens(longContent, 50)
		require.NoError(t, err)

		// Should be truncated to approximately 200 chars (50 tokens * 4)
		assert.LessOrEqual(t, len(result), 250) // Some buffer for truncation message
		assert.Contains(t, result, "<!-- Content truncated for token limit -->")
	})

	t.Run("OptimizeForTokens_XMLTruncation", func(t *testing.T) {
		xmlContent := `<root>
			<section>Content here</section>
			<section>More content here</section>
			<section>Even more content that will be truncated</section>
		</root>`

		result, err := formatter.OptimizeForTokens(xmlContent, 20)
		require.NoError(t, err)

		// Should truncate at a tag boundary
		assert.Contains(t, result, ">")
		assert.Contains(t, result, "<!-- Content truncated for token limit -->")
	})
}

func TestDefaultFormatter(t *testing.T) {
	formatter, err := context.NewDefaultFormatter()
	require.NoError(t, err)

	ctx := &testContext{
		commissionID:    "default-001",
		commissionTitle: "Default Test",
		currentTask: context.TaskContext{
			ID:    "DEFAULT-001",
			Title: "Test default formatter",
		},
	}

	t.Run("HasAllMethods", func(t *testing.T) {
		// Test XML formatting
		xml, err := formatter.FormatAsXML(ctx)
		require.NoError(t, err)
		assert.Contains(t, xml, "<guild-context>")

		// Test Markdown formatting
		md, err := formatter.FormatAsMarkdown(ctx)
		require.NoError(t, err)
		assert.Contains(t, md, "# Commission Context")

		// Test optimization
		optimized, err := formatter.OptimizeForTokens("test content", 10)
		require.NoError(t, err)
		assert.NotEmpty(t, optimized)
	})
}

func BenchmarkXMLFormatting(b *testing.B) {
	formatter, _ := context.NewXMLFormatter()
	ctx := &testContext{
		commissionID:    "bench-001",
		commissionTitle: "Benchmark System",
		currentTask: context.TaskContext{
			ID:           "BENCH-001",
			Title:        "Benchmark task",
			Dependencies: []string{"DEP-1", "DEP-2", "DEP-3"},
			Capabilities: []string{"cap1", "cap2", "cap3"},
		},
		sections: []context.Section{
			{Level: 1, Path: "1", Title: "Section 1", Content: "Content 1"},
			{Level: 2, Path: "1.1", Title: "Section 1.1", Content: "Content 1.1"},
			{Level: 2, Path: "1.2", Title: "Section 1.2", Content: "Content 1.2"},
		},
		relatedTasks: []context.TaskContext{
			{ID: "REL-1", Title: "Related 1"},
			{ID: "REL-2", Title: "Related 2"},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = formatter.FormatAsXML(ctx)
	}
}

func BenchmarkMarkdownFormatting(b *testing.B) {
	formatter, _ := context.NewXMLFormatter()
	ctx := &testContext{
		commissionID:    "bench-002",
		commissionTitle: "Benchmark System",
		currentTask: context.TaskContext{
			ID:    "BENCH-002",
			Title: "Benchmark task",
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = formatter.FormatAsMarkdown(ctx)
	}
}
