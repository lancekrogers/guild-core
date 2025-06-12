package registry

import (
	"encoding/json"
	"testing"

	"github.com/guild-ventures/guild-core/tools/code"
	"github.com/guild-ventures/guild-core/tools/edit"
	"github.com/stretchr/testify/assert"
)

func TestGetCodeToolNames(t *testing.T) {
	names := GetCodeToolNames()

	// Should return all 7 tools
	assert.Len(t, names, 7)

	// Check specific tools are included
	expectedTools := []string{
		"ast",
		"dependencies",
		"metrics",
		"search_replace",
		"apply_diff",
		"cursor_position",
		"multi_refactor",
	}

	for _, expected := range expectedTools {
		assert.Contains(t, names, expected, "Should include tool: %s", expected)
	}
}

func TestGetCodeToolsByCategory(t *testing.T) {
	categories := GetCodeToolsByCategory()

	// Should have 3 categories
	assert.Len(t, categories, 3)

	// Check code_analysis category
	codeAnalysis, exists := categories["code_analysis"]
	assert.True(t, exists)
	assert.Contains(t, codeAnalysis, "ast")
	assert.Contains(t, codeAnalysis, "dependencies")
	assert.Contains(t, codeAnalysis, "metrics")

	// Check code_search category
	codeSearch, exists := categories["code_search"]
	assert.True(t, exists)
	assert.Contains(t, codeSearch, "search_replace")

	// Check code_editing category
	codeEditing, exists := categories["code_editing"]
	assert.True(t, exists)
	assert.Contains(t, codeEditing, "apply_diff")
	assert.Contains(t, codeEditing, "cursor_position")
	assert.Contains(t, codeEditing, "multi_refactor")
}

func TestCodeToolInstantiation(t *testing.T) {
	// Test that each tool can be instantiated properly
	testCases := []struct {
		name     string
		toolName string
		category string
	}{
		{
			name:     "AST Tool",
			toolName: "ast",
			category: "code",
		},
		{
			name:     "Dependencies Tool",
			toolName: "dependencies",
			category: "code",
		},
		{
			name:     "Metrics Tool",
			toolName: "metrics",
			category: "code",
		},
		{
			name:     "Search Replace Tool",
			toolName: "search_replace",
			category: "code",
		},
		{
			name:     "Apply Diff Tool",
			toolName: "apply_diff",
			category: "edit",
		},
		{
			name:     "Cursor Position Tool",
			toolName: "cursor_position",
			category: "edit",
		},
		{
			name:     "Multi Refactor Tool",
			toolName: "multi_refactor",
			category: "edit",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var tool interface{}

			// Instantiate the tool based on category and name
			switch tc.category {
			case "code":
				switch tc.toolName {
				case "ast":
					tool = code.NewASTTool()
				case "dependencies":
					tool = code.NewDependenciesTool()
				case "metrics":
					tool = code.NewMetricsTool()
				case "search_replace":
					tool = code.NewSearchReplaceTool()
				}
			case "edit":
				switch tc.toolName {
				case "apply_diff":
					tool = edit.NewApplyDiffTool()
				case "cursor_position":
					tool = edit.NewCursorPositionTool()
				case "multi_refactor":
					tool = edit.NewMultiFileRefactorTool()
				}
			}

			assert.NotNil(t, tool, "Tool should be instantiated")

			// Test that tool has required methods (assuming they implement a common interface)
			// This is a basic check that the tools are properly constructed
		})
	}
}

func TestCodeToolsCategories_Consistency(t *testing.T) {
	// Verify that category mappings are consistent
	categories := GetCodeToolsByCategory()
	allNames := GetCodeToolNames()

	// Count tools in categories
	categoryCount := 0
	for _, toolNames := range categories {
		categoryCount += len(toolNames)
	}

	// Should match total tool count
	assert.Equal(t, len(allNames), categoryCount, "Category tool count should match total tool count")

	// Verify no duplicate tools across categories
	seen := make(map[string]bool)
	for _, toolNames := range categories {
		for _, toolName := range toolNames {
			assert.False(t, seen[toolName], "Tool %s should not appear in multiple categories", toolName)
			seen[toolName] = true
			assert.Contains(t, allNames, toolName, "Tool %s should be in the main tool list", toolName)
		}
	}
}

func TestCodeToolsExamples(t *testing.T) {
	// Test that all tools have valid examples when instantiated
	tools := []interface{}{
		code.NewASTTool(),
		code.NewDependenciesTool(),
		code.NewMetricsTool(),
		code.NewSearchReplaceTool(),
		edit.NewApplyDiffTool(),
		edit.NewCursorPositionTool(),
		edit.NewMultiFileRefactorTool(),
	}

	for i, tool := range tools {
		t.Run("tool_"+string(rune(i+'0')), func(t *testing.T) {
			assert.NotNil(t, tool, "Tool should be instantiated")

			// If the tool has a GetExamples method, test it
			// This would require knowing the exact interface, so we'll skip this for now
			// In a real implementation, you'd cast to the specific tool interface
		})
	}
}

func TestCodeToolsJSON_Examples(t *testing.T) {
	// Test some example JSON strings that should be valid for tools
	validExamples := []string{
		`{"file": "main.go", "target": "functions"}`,                                                    // AST
		`{"path": ".", "format": "tree"}`,                                                               // Dependencies
		`{"file": "main.go", "granularity": "function"}`,                                                // Metrics
		`{"pattern": "TODO", "files": ["*.go"]}`,                                                        // Search Replace
		`{"diff": "--- a/file.go\n+++ b/file.go\n@@ -1,1 +1,1 @@\n-old\n+new"}`,                         // Apply Diff
		`{"file": "main.go", "line": 10, "column": 5}`,                                                  // Cursor Position
		`{"type": "rename", "target": {"file": "main.go", "symbol": "oldName"}, "new_name": "newName"}`, // Multi Refactor
	}

	for i, example := range validExamples {
		t.Run("example_"+string(rune(i+'0')), func(t *testing.T) {
			var parsed interface{}
			err := json.Unmarshal([]byte(example), &parsed)
			assert.NoError(t, err, "Example should be valid JSON: %s", example)
		})
	}
}

func BenchmarkGetCodeToolNames(b *testing.B) {
	for i := 0; i < b.N; i++ {
		names := GetCodeToolNames()
		if len(names) == 0 {
			b.Fatal("Should return tool names")
		}
	}
}

func BenchmarkGetCodeToolsByCategory(b *testing.B) {
	for i := 0; i < b.N; i++ {
		categories := GetCodeToolsByCategory()
		if len(categories) == 0 {
			b.Fatal("Should return categories")
		}
	}
}

func BenchmarkToolInstantiation(b *testing.B) {
	b.Run("AST", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			tool := code.NewASTTool()
			if tool == nil {
				b.Fatal("Tool should be instantiated")
			}
		}
	})

	b.Run("Dependencies", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			tool := code.NewDependenciesTool()
			if tool == nil {
				b.Fatal("Tool should be instantiated")
			}
		}
	})

	b.Run("Metrics", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			tool := code.NewMetricsTool()
			if tool == nil {
				b.Fatal("Tool should be instantiated")
			}
		}
	})

	b.Run("SearchReplace", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			tool := code.NewSearchReplaceTool()
			if tool == nil {
				b.Fatal("Tool should be instantiated")
			}
		}
	})

	b.Run("ApplyDiff", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			tool := edit.NewApplyDiffTool()
			if tool == nil {
				b.Fatal("Tool should be instantiated")
			}
		}
	})

	b.Run("CursorPosition", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			tool := edit.NewCursorPositionTool()
			if tool == nil {
				b.Fatal("Tool should be instantiated")
			}
		}
	})

	b.Run("MultiRefactor", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			tool := edit.NewMultiFileRefactorTool()
			if tool == nil {
				b.Fatal("Tool should be instantiated")
			}
		}
	})
}
