package tools_test

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	
	"github.com/guild-ventures/guild-core/pkg/lsp"
	lsptools "github.com/guild-ventures/guild-core/pkg/lsp/tools"
)

func TestLSPToolsIntegration(t *testing.T) {
	// Skip if gopls is not available
	if _, err := exec.LookPath("gopls"); err != nil {
		t.Skip("gopls not found in PATH, skipping LSP tools integration tests")
	}

	// Create temporary workspace
	tmpDir, err := os.MkdirTemp("", "lsp-tools-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create a test Go file with rich content
	testFile := filepath.Join(tmpDir, "calculator.go")
	testContent := `package calculator

import (
	"errors"
	"fmt"
)

// Calculator provides basic arithmetic operations
type Calculator struct {
	precision int
}

// NewCalculator creates a new calculator instance
func NewCalculator(precision int) *Calculator {
	return &Calculator{
		precision: precision,
	}
}

// Add performs addition of two numbers
func (c *Calculator) Add(a, b float64) float64 {
	return a + b
}

// Divide performs division with error handling
func (c *Calculator) Divide(a, b float64) (float64, error) {
	if b == 0 {
		return 0, errors.New("division by zero")
	}
	return a / b, nil
}

// Format formats a number with the calculator's precision
func (c *Calculator) Format(value float64) string {
	format := fmt.Sprintf("%%.%df", c.precision)
	return fmt.Sprintf(format, value)
}
`
	err = os.WriteFile(testFile, []byte(testContent), 0644)
	require.NoError(t, err)

	// Create go.mod
	goModContent := `module calculator

go 1.21
`
	err = os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goModContent), 0644)
	require.NoError(t, err)

	// Create LSP manager
	manager, err := lsp.NewManager("")
	require.NoError(t, err)
	defer manager.Shutdown(context.Background())

	ctx := context.Background()

	t.Run("CompletionTool", func(t *testing.T) {
		tool := lsptools.NewCompletionTool(manager)
		
		// Test completion after "c." in the Add method
		input := map[string]interface{}{
			"file":   testFile,
			"line":   21,  // Line with "return a + b"
			"column": 2,   // After 'c.'
		}
		
		inputJSON, err := json.Marshal(input)
		require.NoError(t, err)
		
		result, err := tool.Execute(ctx, string(inputJSON))
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.True(t, result.Success)
		
		// Parse result
		var completionResult lsptools.CompletionResult
		err = json.Unmarshal([]byte(result.Output), &completionResult)
		require.NoError(t, err)
		
		// Should have completions for Calculator methods
		assert.True(t, len(completionResult.Items) > 0)
		
		// Check metadata
		assert.Equal(t, testFile, result.Metadata["file"])
		assert.NotEmpty(t, result.Metadata["completion_count"])
	})

	t.Run("DefinitionTool", func(t *testing.T) {
		tool := lsptools.NewDefinitionTool(manager)
		
		// Test going to definition of Calculator type in NewCalculator
		input := map[string]interface{}{
			"file":   testFile,
			"line":   13,  // Line with "func NewCalculator"
			"column": 30,  // On '*Calculator'
		}
		
		inputJSON, err := json.Marshal(input)
		require.NoError(t, err)
		
		result, err := tool.Execute(ctx, string(inputJSON))
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.True(t, result.Success)
		
		// Parse result
		var defResult lsptools.DefinitionResult
		err = json.Unmarshal([]byte(result.Output), &defResult)
		require.NoError(t, err)
		
		// Should find the Calculator type definition
		assert.GreaterOrEqual(t, len(defResult.Locations), 1)
		if len(defResult.Locations) > 0 {
			// Should point to line 8 where Calculator is defined
			assert.Equal(t, testFile, defResult.Locations[0].File)
			assert.Equal(t, 8, defResult.Locations[0].Line)
		}
	})

	t.Run("ReferencesTool", func(t *testing.T) {
		tool := lsptools.NewReferencesTool(manager)
		
		// Test finding references to the precision field
		input := map[string]interface{}{
			"file":               testFile,
			"line":               9,   // Line with "precision int"
			"column":             1,   // On 'precision'
			"include_declaration": true,
		}
		
		inputJSON, err := json.Marshal(input)
		require.NoError(t, err)
		
		result, err := tool.Execute(ctx, string(inputJSON))
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.True(t, result.Success)
		
		// Parse result
		var refResult lsptools.ReferencesResult
		err = json.Unmarshal([]byte(result.Output), &refResult)
		require.NoError(t, err)
		
		// Should find at least 3 references (declaration, NewCalculator, Format)
		assert.GreaterOrEqual(t, refResult.TotalCount, 3)
	})

	t.Run("HoverTool", func(t *testing.T) {
		tool := lsptools.NewHoverTool(manager)
		
		// Test hover over Calculator type
		input := map[string]interface{}{
			"file":   testFile,
			"line":   8,   // Line with "type Calculator struct"
			"column": 5,   // On 'Calculator'
		}
		
		inputJSON, err := json.Marshal(input)
		require.NoError(t, err)
		
		result, err := tool.Execute(ctx, string(inputJSON))
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.True(t, result.Success)
		
		// Parse result
		var hoverResult lsptools.HoverResult
		err = json.Unmarshal([]byte(result.Output), &hoverResult)
		require.NoError(t, err)
		
		// Should have hover content
		assert.NotEmpty(t, hoverResult.Content)
		// Content should mention Calculator
		assert.Contains(t, hoverResult.Content, "Calculator")
	})
}

func TestLSPToolsEdgeCases(t *testing.T) {
	ctx := context.Background()
	
	// Create a mock manager that will return errors
	manager, err := lsp.NewManager("")
	require.NoError(t, err)
	defer manager.Shutdown(ctx)

	t.Run("InvalidInput", func(t *testing.T) {
		tool := lsptools.NewCompletionTool(manager)
		
		// Test with invalid JSON
		result, err := tool.Execute(ctx, "invalid json")
		assert.Error(t, err)
		assert.Nil(t, result)
		
		// Test with missing required fields
		result, err = tool.Execute(ctx, `{"file": "test.go"}`)
		assert.Error(t, err)
		assert.Nil(t, result)
	})

	t.Run("NonExistentFile", func(t *testing.T) {
		tool := lsptools.NewDefinitionTool(manager)
		
		input := map[string]interface{}{
			"file":   "/nonexistent/file.go",
			"line":   0,
			"column": 0,
		}
		
		inputJSON, err := json.Marshal(input)
		require.NoError(t, err)
		
		_, err = tool.Execute(ctx, string(inputJSON))
		// Should handle gracefully
		assert.Error(t, err)
	})
}

func TestFormatters(t *testing.T) {
	t.Run("FormatCompletionsAsText", func(t *testing.T) {
		result := &lsptools.CompletionResult{
			Items: []lsptools.CompletionItem{
				{Label: "Println", Kind: "function", Detail: "func Println(a ...interface{})"},
				{Label: "Printf", Kind: "function", Detail: "func Printf(format string, a ...interface{})"},
			},
		}
		
		text := lsptools.FormatCompletionsAsText(result)
		assert.Contains(t, text, "Found 2 completions")
		assert.Contains(t, text, "Println")
		assert.Contains(t, text, "Printf")
	})

	t.Run("FormatDefinitionsAsText", func(t *testing.T) {
		result := &lsptools.DefinitionResult{
			Locations: []lsptools.LocationResult{
				{File: "/path/to/file.go", Line: 10, Column: 5},
			},
		}
		
		text := lsptools.FormatDefinitionsAsText(result)
		assert.Contains(t, text, "Definition found at:")
		assert.Contains(t, text, "/path/to/file.go:11:6") // 1-based in output
	})

	t.Run("FormatReferencesAsText", func(t *testing.T) {
		result := &lsptools.ReferencesResult{
			References: []lsptools.LocationResult{
				{File: "/path/to/file1.go", Line: 10, Column: 5},
				{File: "/path/to/file1.go", Line: 20, Column: 10},
				{File: "/path/to/file2.go", Line: 5, Column: 15},
			},
			TotalCount: 3,
		}
		
		text := lsptools.FormatReferencesAsText(result)
		assert.Contains(t, text, "Found 3 references")
		assert.Contains(t, text, "file1.go (2 references)")
		assert.Contains(t, text, "file2.go (1 references)")
	})

	t.Run("FormatHoverAsText", func(t *testing.T) {
		result := &lsptools.HoverResult{
			Content:  "type Calculator struct",
			Language: "go",
		}
		
		text := lsptools.FormatHoverAsText(result)
		assert.Contains(t, text, "Language: go")
		assert.Contains(t, text, "type Calculator struct")
	})
}