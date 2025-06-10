package edit

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApplyDiffTool_NewApplyDiffTool(t *testing.T) {
	tool := NewApplyDiffTool()
	assert.NotNil(t, tool)
	assert.Equal(t, "apply_diff", tool.Name())
	assert.Equal(t, "edit", tool.Category())
}

func TestApplyDiffTool_Execute_SimpleDiff(t *testing.T) {
	// Create a temporary file
	originalContent := `package main

import "fmt"

func main() {
	fmt.Println("Hello, World!")
}
`
	
	tmpFile, err := os.CreateTemp("", "test_*.go")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())
	
	_, err = tmpFile.WriteString(originalContent)
	require.NoError(t, err)
	tmpFile.Close()

	// Create a unified diff
	diff := `--- a/` + tmpFile.Name() + `
+++ b/` + tmpFile.Name() + `
@@ -4,5 +4,5 @@ import "fmt"
 
 func main() {
-	fmt.Println("Hello, World!")
+	fmt.Println("Hello, Universe!")
 }
`

	tool := NewApplyDiffTool()
	
	params := ApplyDiffParams{
		Diff:       diff,
		TargetFile: tmpFile.Name(),
		DryRun:     true,
	}
	
	input, err := json.Marshal(params)
	require.NoError(t, err)
	
	result, err := tool.Execute(context.Background(), string(input))
	require.NoError(t, err)
	assert.NotNil(t, result)
	
	// Should show preview
	assert.Contains(t, result.Output, "Diff Application Preview")
	assert.Contains(t, result.Output, "Hello, World!")
	assert.Contains(t, result.Output, "Hello, Universe!")
}

func TestApplyDiffTool_Execute_ApplyDiff(t *testing.T) {
	// Create a temporary file
	originalContent := `package main

import "fmt"

func main() {
	fmt.Println("Hello, World!")
}
`
	
	tmpFile, err := os.CreateTemp("", "test_*.go")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())
	
	_, err = tmpFile.WriteString(originalContent)
	require.NoError(t, err)
	tmpFile.Close()

	// Create a unified diff
	diff := `--- a/` + tmpFile.Name() + `
+++ b/` + tmpFile.Name() + `
@@ -4,5 +4,5 @@ import "fmt"
 
 func main() {
-	fmt.Println("Hello, World!")
+	fmt.Println("Hello, Universe!")
 }
`

	tool := NewApplyDiffTool()
	
	params := ApplyDiffParams{
		Diff:       diff,
		TargetFile: tmpFile.Name(),
		Backup:     true,
	}
	
	input, err := json.Marshal(params)
	require.NoError(t, err)
	
	result, err := tool.Execute(context.Background(), string(input))
	require.NoError(t, err)
	assert.NotNil(t, result)
	
	// Should apply diff
	assert.Contains(t, result.Output, "Diff Applied Successfully")
	
	// Verify file was modified
	modifiedContent, err := os.ReadFile(tmpFile.Name())
	require.NoError(t, err)
	assert.Contains(t, string(modifiedContent), "Hello, Universe!")
	assert.NotContains(t, string(modifiedContent), "Hello, World!")
	
	// Check backup was created
	assert.Contains(t, result.Output, "Backup created:")
}

func TestApplyDiffTool_Execute_ReverseDiff(t *testing.T) {
	// Create a temporary file with modified content
	modifiedContent := `package main

import "fmt"

func main() {
	fmt.Println("Hello, Universe!")
}
`
	
	tmpFile, err := os.CreateTemp("", "test_*.go")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())
	
	_, err = tmpFile.WriteString(modifiedContent)
	require.NoError(t, err)
	tmpFile.Close()

	// Create a diff to reverse
	diff := `--- a/` + tmpFile.Name() + `
+++ b/` + tmpFile.Name() + `
@@ -4,5 +4,5 @@ import "fmt"
 
 func main() {
-	fmt.Println("Hello, World!")
+	fmt.Println("Hello, Universe!")
 }
`

	tool := NewApplyDiffTool()
	
	params := ApplyDiffParams{
		Diff:       diff,
		TargetFile: tmpFile.Name(),
		Reverse:    true,
	}
	
	input, err := json.Marshal(params)
	require.NoError(t, err)
	
	result, err := tool.Execute(context.Background(), string(input))
	require.NoError(t, err)
	assert.NotNil(t, result)
	
	// Should apply reverse diff
	assert.Contains(t, result.Output, "Diff Applied Successfully")
	assert.Contains(t, result.Output, "Mode: Reverse application")
	
	// Verify file was reverted
	revertedContent, err := os.ReadFile(tmpFile.Name())
	require.NoError(t, err)
	assert.Contains(t, string(revertedContent), "Hello, World!")
	assert.NotContains(t, string(revertedContent), "Hello, Universe!")
}

func TestApplyDiffTool_Execute_MultipleHunks(t *testing.T) {
	// Create a temporary file
	originalContent := `package main

import "fmt"

func main() {
	fmt.Println("Hello, World!")
	fmt.Println("Line 2")
	fmt.Println("Line 3")
}

func helper() {
	fmt.Println("Helper function")
}
`
	
	tmpFile, err := os.CreateTemp("", "test_*.go")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())
	
	_, err = tmpFile.WriteString(originalContent)
	require.NoError(t, err)
	tmpFile.Close()

	// Create a diff with multiple hunks
	diff := `--- a/` + tmpFile.Name() + `
+++ b/` + tmpFile.Name() + `
@@ -4,6 +4,6 @@ import "fmt"
 
 func main() {
-	fmt.Println("Hello, World!")
+	fmt.Println("Hello, Universe!")
 	fmt.Println("Line 2")
 	fmt.Println("Line 3")
 }
@@ -11,5 +11,5 @@ func main() {
 
 func helper() {
-	fmt.Println("Helper function")
+	fmt.Println("Modified helper function")
 }
`

	tool := NewApplyDiffTool()
	
	params := ApplyDiffParams{
		Diff:       diff,
		TargetFile: tmpFile.Name(),
		DryRun:     true,
	}
	
	input, err := json.Marshal(params)
	require.NoError(t, err)
	
	result, err := tool.Execute(context.Background(), string(input))
	require.NoError(t, err)
	assert.NotNil(t, result)
	
	// Should handle multiple hunks
	assert.Contains(t, result.Output, "2 hunks")
	assert.Contains(t, result.Output, "Hello, Universe!")
	assert.Contains(t, result.Output, "Modified helper")
}

func TestApplyDiffTool_Execute_ContextMismatch(t *testing.T) {
	// Create a temporary file with different content than expected by diff
	actualContent := `package main

import "fmt"

func main() {
	fmt.Println("Different content!")
}
`
	
	tmpFile, err := os.CreateTemp("", "test_*.go")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())
	
	_, err = tmpFile.WriteString(actualContent)
	require.NoError(t, err)
	tmpFile.Close()

	// Create a diff expecting different content
	diff := `--- a/` + tmpFile.Name() + `
+++ b/` + tmpFile.Name() + `
@@ -4,5 +4,5 @@ import "fmt"
 
 func main() {
-	fmt.Println("Hello, World!")
+	fmt.Println("Hello, Universe!")
 }
`

	tool := NewApplyDiffTool()
	
	params := ApplyDiffParams{
		Diff:       diff,
		TargetFile: tmpFile.Name(),
	}
	
	input, err := json.Marshal(params)
	require.NoError(t, err)
	
	result, err := tool.Execute(context.Background(), string(input))
	require.NoError(t, err)
	assert.NotNil(t, result)
	
	// Should detect conflicts
	assert.Contains(t, result.Output, "Conflicts")
	assert.Contains(t, result.Output, "Cannot apply diff")
}

func TestApplyDiffTool_Execute_AutoDetectFile(t *testing.T) {
	// Create a temporary file
	originalContent := `package main

func main() {
	println("test")
}
`
	
	tmpFile, err := os.CreateTemp("", "test_*.go")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())
	
	_, err = tmpFile.WriteString(originalContent)
	require.NoError(t, err)
	tmpFile.Close()

	// Create a diff without specifying target file
	diff := `--- a/` + tmpFile.Name() + `
+++ b/` + tmpFile.Name() + `
@@ -2,4 +2,4 @@ package main
 
 func main() {
-	println("test")
+	println("modified")
 }
`

	tool := NewApplyDiffTool()
	
	params := ApplyDiffParams{
		Diff:   diff,
		DryRun: true,
	}
	
	input, err := json.Marshal(params)
	require.NoError(t, err)
	
	result, err := tool.Execute(context.Background(), string(input))
	require.NoError(t, err)
	assert.NotNil(t, result)
	
	// Should auto-detect target file
	assert.Contains(t, result.Output, "Target File: "+tmpFile.Name())
}

func TestApplyDiffTool_Execute_InvalidDiff(t *testing.T) {
	tool := NewApplyDiffTool()
	
	params := ApplyDiffParams{
		Diff: "not a valid diff",
	}
	
	input, err := json.Marshal(params)
	require.NoError(t, err)
	
	result, err := tool.Execute(context.Background(), string(input))
	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestApplyDiffTool_Execute_EmptyDiff(t *testing.T) {
	tool := NewApplyDiffTool()
	
	params := ApplyDiffParams{
		Diff: "",
	}
	
	input, err := json.Marshal(params)
	require.NoError(t, err)
	
	result, err := tool.Execute(context.Background(), string(input))
	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestApplyDiffTool_Execute_NonexistentFile(t *testing.T) {
	diff := `--- a/nonexistent.go
+++ b/nonexistent.go
@@ -1,3 +1,3 @@
 package main
-func main() {}
+func main() { println("test") }
`

	tool := NewApplyDiffTool()
	
	params := ApplyDiffParams{
		Diff: diff,
	}
	
	input, err := json.Marshal(params)
	require.NoError(t, err)
	
	result, err := tool.Execute(context.Background(), string(input))
	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestApplyDiffTool_Execute_InvalidJSON(t *testing.T) {
	tool := NewApplyDiffTool()
	
	result, err := tool.Execute(context.Background(), "invalid json")
	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestApplyDiffTool_Execute_ComplexDiff(t *testing.T) {
	// Create a more complex file
	originalContent := `package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) > 1 {
		fmt.Println("Hello,", os.Args[1])
	} else {
		fmt.Println("Hello, World!")
	}
}

func helper(name string) string {
	return "Hello, " + name
}
`
	
	tmpFile, err := os.CreateTemp("", "test_*.go")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())
	
	_, err = tmpFile.WriteString(originalContent)
	require.NoError(t, err)
	tmpFile.Close()

	// Create a complex diff with additions and deletions
	diff := `--- a/` + tmpFile.Name() + `
+++ b/` + tmpFile.Name() + `
@@ -3,11 +3,13 @@ package main
 import (
 	"fmt"
 	"os"
+	"strings"
 )
 
 func main() {
 	if len(os.Args) > 1 {
-		fmt.Println("Hello,", os.Args[1])
+		name := strings.TrimSpace(os.Args[1])
+		fmt.Println("Hello,", name)
 	} else {
 		fmt.Println("Hello, World!")
 	}
@@ -15,5 +17,8 @@ func main() {
 
 func helper(name string) string {
-	return "Hello, " + name
+	if name == "" {
+		return "Hello, Anonymous"
+	}
+	return "Hello, " + strings.Title(name)
 }
`

	tool := NewApplyDiffTool()
	
	params := ApplyDiffParams{
		Diff:       diff,
		TargetFile: tmpFile.Name(),
		DryRun:     true,
	}
	
	input, err := json.Marshal(params)
	require.NoError(t, err)
	
	result, err := tool.Execute(context.Background(), string(input))
	require.NoError(t, err)
	assert.NotNil(t, result)
	
	// Should handle complex diff
	assert.Contains(t, result.Output, "Changes: +")
	assert.Contains(t, result.Output, "strings")
	assert.Contains(t, result.Output, "TrimSpace")
}

func TestApplyDiffTool_Execute_EmptyFile(t *testing.T) {
	// Create an empty file
	tmpFile, err := os.CreateTemp("", "test_*.go")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	// Create a diff to add content to empty file
	diff := `--- a/` + tmpFile.Name() + `
+++ b/` + tmpFile.Name() + `
@@ -0,0 +1,3 @@
+package main
+
+func main() {}
`

	tool := NewApplyDiffTool()
	
	params := ApplyDiffParams{
		Diff:       diff,
		TargetFile: tmpFile.Name(),
		DryRun:     true,
	}
	
	input, err := json.Marshal(params)
	require.NoError(t, err)
	
	result, err := tool.Execute(context.Background(), string(input))
	require.NoError(t, err)
	assert.NotNil(t, result)
	
	// Should handle empty file
	assert.Contains(t, result.Output, "Diff Application Preview")
}

func TestApplyDiffTool_Execute_WithBackup(t *testing.T) {
	// Create a temporary file
	originalContent := `package main

func main() {
	println("original")
}
`
	
	tmpFile, err := os.CreateTemp("", "test_*.go")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())
	defer os.Remove(tmpFile.Name() + ".bak") // Clean up backup
	
	_, err = tmpFile.WriteString(originalContent)
	require.NoError(t, err)
	tmpFile.Close()

	// Create a simple diff
	diff := `--- a/` + tmpFile.Name() + `
+++ b/` + tmpFile.Name() + `
@@ -2,4 +2,4 @@ package main
 
 func main() {
-	println("original")
+	println("modified")
 }
`

	tool := NewApplyDiffTool()
	
	params := ApplyDiffParams{
		Diff:       diff,
		TargetFile: tmpFile.Name(),
		Backup:     true,
	}
	
	input, err := json.Marshal(params)
	require.NoError(t, err)
	
	result, err := tool.Execute(context.Background(), string(input))
	require.NoError(t, err)
	assert.NotNil(t, result)
	
	// Should create backup
	assert.Contains(t, result.Output, "Backup created:")
	
	// Verify backup exists and contains original content
	backupContent, err := os.ReadFile(tmpFile.Name() + ".bak")
	require.NoError(t, err)
	assert.Contains(t, string(backupContent), "original")
	
	// Verify main file was modified
	modifiedContent, err := os.ReadFile(tmpFile.Name())
	require.NoError(t, err)
	assert.Contains(t, string(modifiedContent), "modified")
}