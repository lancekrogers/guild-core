//go:build ignore
// +build ignore

package main

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/guild-ventures/guild-core/pkg/gerror"
)

// Demo script to showcase rich content rendering
func main() {
	// Create markdown renderer
	renderer, err := NewMarkdownRenderer(80)
	if err != nil {
		log.Fatal(err)
	}

	// Create content formatter
	formatter := NewContentFormatter(renderer, 80)

	fmt.Println("=== Guild Rich Content Rendering Demo ===\n")

	// Demo 1: Agent Response with Code
	fmt.Println("1. Agent Response with Code:")
	agentContent := `I've analyzed your code and found the issue. Here's the corrected version:

` + "```go" + `
func ProcessData(items []string) error {
    if len(items) == 0 {
        return gerror.New(gerror.ErrCodeInvalidInput, "no items to process", nil).
            WithComponent("demo").
            WithOperation("ProcessData")
    }

    for i, item := range items {
        fmt.Printf("Processing item %d: %s\n", i+1, item)
    }

    return nil
}
` + "```" + `

The main changes:
- Added **error handling** for empty input
- Improved **logging** with item indices
- Returns an *error* type for better error propagation`

	fmt.Println(formatter.FormatAgentResponse(agentContent, "code-analyzer"))
	fmt.Println("\n" + strings.Repeat("-", 80) + "\n")

	// Demo 2: System Message
	fmt.Println("2. System Messages:")
	systemMessages := []string{
		"Build completed successfully",
		"Error: Failed to connect to database",
		"Warning: Deprecated function detected",
	}

	for _, msg := range systemMessages {
		fmt.Println(formatter.FormatSystemMessage(msg))
		time.Sleep(100 * time.Millisecond)
	}
	fmt.Println("\n" + strings.Repeat("-", 80) + "\n")

	// Demo 3: Tool Output
	fmt.Println("3. Tool Output:")
	toolOutput := `Executed: go test ./...
` + "```" + `
=== RUN   TestProcessData
--- PASS: TestProcessData (0.00s)
=== RUN   TestValidateInput
--- PASS: TestValidateInput (0.00s)
PASS
ok      github.com/guild/example    0.123s
` + "```"

	fmt.Println(formatter.FormatToolOutput(toolOutput, "TestRunner"))
	fmt.Println("\n" + strings.Repeat("-", 80) + "\n")

	// Demo 4: Error Message with Details
	fmt.Println("4. Error Message:")
	errorContent := `**Compilation Error**

The following errors were found:
- Line 42: undefined variable ` + "`config`" + `
- Line 56: missing return statement
- Line 73: type mismatch in assignment

Please fix these issues before proceeding.`

	fmt.Println(formatter.FormatErrorMessage(errorContent))
	fmt.Println("\n" + strings.Repeat("-", 80) + "\n")

	// Demo 5: Thinking/Planning Message
	fmt.Println("5. Agent Thinking:")
	thinkingContent := "Analyzing the codebase structure to determine the best refactoring approach..."
	fmt.Println(formatter.FormatThinkingMessage(thinkingContent, "architect"))

	// Demo 6: Working Message with Progress
	fmt.Println("\n6. Agent Working:")
	workingContent := `Refactoring in progress:
- ✓ Updated interfaces
- ✓ Migrated old implementations
- ⏳ Running tests...
- ⏳ Updating documentation`

	fmt.Println(formatter.FormatWorkingMessage(workingContent, "refactor-bot"))
	fmt.Println("\n" + strings.Repeat("-", 80) + "\n")

	// Demo 7: Complex Markdown
	fmt.Println("7. Complex Markdown Rendering:")
	complexContent := `# Project Analysis Report

## Summary
The codebase analysis revealed **3 critical issues** and *5 minor improvements*.

### Critical Issues
1. **Memory Leak** in the connection pool
2. **Race Condition** in the worker threads
3. **SQL Injection** vulnerability in user input handling

### Recommendations
> "The code quality can be significantly improved with proper error handling and testing"

#### Immediate Actions
- [ ] Fix memory leak (Priority: **HIGH**)
- [ ] Add mutex locks (Priority: **HIGH**)
- [ ] Sanitize inputs (Priority: **CRITICAL**)

#### Code Example
Here's how to fix the race condition:

` + "```go" + `
var mu sync.Mutex

func SafeUpdate(data *SharedData) {
    mu.Lock()
    defer mu.Unlock()

    // Safe operations here
    data.Value++
}
` + "```" + `

---

For more details, see the [full report](https://internal.docs/analysis).`

	fmt.Println(formatter.FormatAgentResponse(complexContent, "security-auditor"))

	fmt.Println("\n=== Demo Complete ===")
}
