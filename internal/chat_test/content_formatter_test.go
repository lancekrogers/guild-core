// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package chat_test

import (
	"strings"
	"testing"
	"time"

	"github.com/guild-ventures/guild-core/internal/chat"
)

func TestNewContentFormatter(t *testing.T) {
	renderer, err := chat.NewMarkdownRenderer(80)
	if err != nil {
		t.Fatalf("Failed to create markdown renderer: %v", err)
	}

	tests := []struct {
		name  string
		width int
	}{
		{
			name:  "standard width",
			width: 80,
		},
		{
			name:  "narrow width",
			width: 40,
		},
		{
			name:  "wide width",
			width: 120,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			formatter := chat.NewContentFormatter(renderer, tt.width)
			if formatter == nil {
				t.Error("NewContentFormatter() returned nil")
			}
			// Note: We can't access private fields like width and markdownRenderer
			// from outside the package, so we'll test behavior instead
		})
	}
}

func TestContentFormatter_FormatAgentResponse(t *testing.T) {
	renderer, err := chat.NewMarkdownRenderer(80)
	if err != nil {
		t.Fatalf("Failed to create markdown renderer: %v", err)
	}
	formatter := chat.NewContentFormatter(renderer, 80)

	tests := []struct {
		name     string
		content  string
		agentID  string
		validate func(string) bool
	}{
		{
			name:    "simple text response",
			content: "Task completed successfully",
			agentID: "worker-1",
			validate: func(output string) bool {
				return strings.Contains(output, "Task completed successfully")
			},
		},
		{
			name:    "response with code",
			content: "Here's the function:\n```go\nfunc test() {}\n```",
			agentID: "coder-1",
			validate: func(output string) bool {
				// Should include agent attribution for complex responses
				return strings.Contains(output, "coder-1") && strings.Contains(output, "func test")
			},
		},
		{
			name:    "long response with attribution",
			content: strings.Repeat("This is a long response. ", 20),
			agentID: "analyzer-1",
			validate: func(output string) bool {
				// Long responses should show agent attribution
				return strings.Contains(output, "analyzer-1") && strings.Contains(output, "long response")
			},
		},
		{
			name:    "empty agent ID",
			content: "Response without agent ID",
			agentID: "",
			validate: func(output string) bool {
				// Should not include agent attribution
				return strings.Contains(output, "Response without agent ID") && !strings.Contains(output, "🤖")
			},
		},
		{
			name:    "markdown formatted response",
			content: "# Analysis Results\n\n- **Item 1**: Passed\n- **Item 2**: Failed",
			agentID: "tester-1",
			validate: func(output string) bool {
				return strings.Contains(output, "Analysis Results") &&
					strings.Contains(output, "Item 1") &&
					strings.Contains(output, "Item 2")
			},
		},
		{
			name:    "response with inline code",
			content: "Use `guild chat` to start",
			agentID: "helper-1",
			validate: func(output string) bool {
				return strings.Contains(output, "guild chat")
			},
		},
		{
			name:    "empty content",
			content: "",
			agentID: "worker-1",
			validate: func(output string) bool {
				return output == ""
			},
		},
		{
			name:    "unicode content",
			content: "Medieval theming with emojis: 🏰 ⚔️ 🛡️",
			agentID: "theme-1",
			validate: func(output string) bool {
				return strings.Contains(output, "🏰") &&
					strings.Contains(output, "⚔️") &&
					strings.Contains(output, "🛡️")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := formatter.FormatAgentResponse(tt.content, tt.agentID)
			if !tt.validate(output) {
				t.Errorf("FormatAgentResponse() validation failed for %s\nGot: %s", tt.name, output)
			}
		})
	}
}

func TestContentFormatter_FormatSystemMessage(t *testing.T) {
	renderer, err := chat.NewMarkdownRenderer(80)
	if err != nil {
		t.Fatalf("Failed to create markdown renderer: %v", err)
	}
	formatter := chat.NewContentFormatter(renderer, 80)

	tests := []struct {
		name     string
		content  string
		validate func(string) bool
	}{
		{
			name:    "simple system message",
			content: "System initialized",
			validate: func(output string) bool {
				return strings.Contains(output, "System initialized")
			},
		},
		{
			name:    "important system message - error",
			content: "Connection error: timeout",
			validate: func(output string) bool {
				// Should be highlighted with border
				return strings.Contains(output, "Connection error")
			},
		},
		{
			name:    "important system message - completed",
			content: "Task completed successfully",
			validate: func(output string) bool {
				// Should be highlighted
				return strings.Contains(output, "Task completed")
			},
		},
		{
			name:    "warning message",
			content: "Warning: low memory",
			validate: func(output string) bool {
				return strings.Contains(output, "Warning: low memory")
			},
		},
		{
			name:    "status update",
			content: "Agent started processing",
			validate: func(output string) bool {
				return strings.Contains(output, "Agent started")
			},
		},
		{
			name:    "markdown in system message",
			content: "System status:\n- **CPU**: 45%\n- **Memory**: 2GB",
			validate: func(output string) bool {
				return strings.Contains(output, "CPU") && strings.Contains(output, "Memory")
			},
		},
		{
			name:    "disconnection message",
			content: "Agent disconnected from server",
			validate: func(output string) bool {
				return strings.Contains(output, "disconnected")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := formatter.FormatSystemMessage(tt.content)
			if !tt.validate(output) {
				t.Errorf("FormatSystemMessage() validation failed for %s", tt.name)
			}
		})
	}
}

func TestContentFormatter_FormatErrorMessage(t *testing.T) {
	renderer, err := chat.NewMarkdownRenderer(80)
	if err != nil {
		t.Fatalf("Failed to create markdown renderer: %v", err)
	}
	formatter := chat.NewContentFormatter(renderer, 80)

	tests := []struct {
		name     string
		content  string
		validate func(string) bool
	}{
		{
			name:    "simple error",
			content: "File not found",
			validate: func(output string) bool {
				// Should contain error icon and message
				return strings.Contains(output, "❌") && strings.Contains(output, "File not found")
			},
		},
		{
			name:    "error with details",
			content: "Compilation failed:\n```\nundefined: fmt.Printlnn\n```",
			validate: func(output string) bool {
				return strings.Contains(output, "❌") &&
					strings.Contains(output, "Compilation failed") &&
					strings.Contains(output, "undefined")
			},
		},
		{
			name:    "error with markdown",
			content: "**Critical Error**: Database connection lost\n- Retry count: 3\n- Last attempt: failed",
			validate: func(output string) bool {
				return strings.Contains(output, "Critical Error") &&
					strings.Contains(output, "Retry count") &&
					strings.Contains(output, "Last attempt")
			},
		},
		{
			name:    "empty error",
			content: "",
			validate: func(output string) bool {
				// Should still have error formatting
				return strings.Contains(output, "❌")
			},
		},
		{
			name:    "multi-line error",
			content: "Stack trace:\n  at main.go:42\n  at handler.go:15\n  at server.go:200",
			validate: func(output string) bool {
				return strings.Contains(output, "Stack trace") &&
					strings.Contains(output, "main.go:42") &&
					strings.Contains(output, "handler.go:15")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := formatter.FormatErrorMessage(tt.content)
			if !tt.validate(output) {
				t.Errorf("FormatErrorMessage() validation failed for %s\nGot: %s", tt.name, output)
			}
		})
	}
}

func TestContentFormatter_FormatToolOutput(t *testing.T) {
	renderer, err := chat.NewMarkdownRenderer(80)
	if err != nil {
		t.Fatalf("Failed to create markdown renderer: %v", err)
	}
	formatter := chat.NewContentFormatter(renderer, 80)

	tests := []struct {
		name     string
		content  string
		toolName string
		validate func(string) bool
	}{
		{
			name:     "simple tool output",
			content:  "File created successfully",
			toolName: "FileWriter",
			validate: func(output string) bool {
				return strings.Contains(output, "🔧") &&
					strings.Contains(output, "FileWriter") &&
					strings.Contains(output, "File created successfully")
			},
		},
		{
			name:     "tool output with code",
			content:  "Generated code:\n```go\ntype Config struct {}\n```",
			toolName: "CodeGenerator",
			validate: func(output string) bool {
				return strings.Contains(output, "CodeGenerator") &&
					strings.Contains(output, "type Config")
			},
		},
		{
			name:     "tool output with logs",
			content:  "[INFO] Starting process\n[DEBUG] Loading config\n[INFO] Process complete",
			toolName: "ShellExecutor",
			validate: func(output string) bool {
				return strings.Contains(output, "ShellExecutor") &&
					strings.Contains(output, "[INFO]") &&
					strings.Contains(output, "[DEBUG]")
			},
		},
		{
			name:     "empty tool output",
			content:  "",
			toolName: "EmptyTool",
			validate: func(output string) bool {
				return strings.Contains(output, "🔧") && strings.Contains(output, "EmptyTool")
			},
		},
		{
			name:     "structured data output",
			content:  "Results:\n- Test 1: PASS\n- Test 2: PASS\n- Test 3: FAIL",
			toolName: "TestRunner",
			validate: func(output string) bool {
				return strings.Contains(output, "TestRunner") &&
					strings.Contains(output, "PASS") &&
					strings.Contains(output, "FAIL")
			},
		},
		{
			name:     "JSON output",
			content:  `{"status": "success", "files": ["main.go", "test.go"]}`,
			toolName: "JSONParser",
			validate: func(output string) bool {
				return strings.Contains(output, "JSONParser") &&
					strings.Contains(output, "status") &&
					strings.Contains(output, "success")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := formatter.FormatToolOutput(tt.content, tt.toolName)
			if !tt.validate(output) {
				t.Errorf("FormatToolOutput() validation failed for %s", tt.name)
			}
		})
	}
}

func TestContentFormatter_FormatThinkingMessage(t *testing.T) {
	renderer, err := chat.NewMarkdownRenderer(80)
	if err != nil {
		t.Fatalf("Failed to create markdown renderer: %v", err)
	}
	formatter := chat.NewContentFormatter(renderer, 80)

	tests := []struct {
		name     string
		content  string
		agentID  string
		validate func(string) bool
	}{
		{
			name:    "simple thinking",
			content: "Analyzing the request...",
			agentID: "analyzer-1",
			validate: func(output string) bool {
				return strings.Contains(output, "🤔") &&
					strings.Contains(output, "analyzer-1") &&
					strings.Contains(output, "Analyzing")
			},
		},
		{
			name:    "thinking without agent ID",
			content: "Processing information...",
			agentID: "",
			validate: func(output string) bool {
				return strings.Contains(output, "🤔") &&
					strings.Contains(output, "Processing") &&
					!strings.Contains(output, "🤔 ")
			},
		},
		{
			name:    "complex thinking with markdown",
			content: "Considering options:\n- **Option A**: Fast but risky\n- **Option B**: Slow but safe",
			agentID: "planner-1",
			validate: func(output string) bool {
				return strings.Contains(output, "planner-1") &&
					strings.Contains(output, "Option A") &&
					strings.Contains(output, "Option B")
			},
		},
		{
			name:    "empty thinking",
			content: "",
			agentID: "thinker-1",
			validate: func(output string) bool {
				return strings.Contains(output, "🤔") && strings.Contains(output, "thinker-1")
			},
		},
		{
			name:    "thinking with code analysis",
			content: "The function `processData()` appears to have a complexity of O(n²)",
			agentID: "reviewer-1",
			validate: func(output string) bool {
				return strings.Contains(output, "reviewer-1") &&
					strings.Contains(output, "processData") &&
					strings.Contains(output, "O(n²)")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := formatter.FormatThinkingMessage(tt.content, tt.agentID)
			if !tt.validate(output) {
				t.Errorf("FormatThinkingMessage() validation failed for %s", tt.name)
			}
		})
	}
}

func TestContentFormatter_FormatWorkingMessage(t *testing.T) {
	renderer, err := chat.NewMarkdownRenderer(80)
	if err != nil {
		t.Fatalf("Failed to create markdown renderer: %v", err)
	}
	formatter := chat.NewContentFormatter(renderer, 80)

	tests := []struct {
		name     string
		content  string
		agentID  string
		validate func(string) bool
	}{
		{
			name:    "simple working message",
			content: "Executing task...",
			agentID: "worker-1",
			validate: func(output string) bool {
				return strings.Contains(output, "⚙️") &&
					strings.Contains(output, "worker-1") &&
					strings.Contains(output, "Executing task")
			},
		},
		{
			name:    "working without agent ID",
			content: "Building project...",
			agentID: "",
			validate: func(output string) bool {
				return strings.Contains(output, "⚙️") &&
					strings.Contains(output, "Building project")
			},
		},
		{
			name:    "working with progress",
			content: "Processing files: 10/100 (10%)",
			agentID: "processor-1",
			validate: func(output string) bool {
				return strings.Contains(output, "processor-1") &&
					strings.Contains(output, "10/100") &&
					strings.Contains(output, "10%")
			},
		},
		{
			name:    "working with markdown list",
			content: "Running tests:\n- ✓ Test 1\n- ✓ Test 2\n- ⏳ Test 3",
			agentID: "tester-1",
			validate: func(output string) bool {
				return strings.Contains(output, "tester-1") &&
					strings.Contains(output, "✓") &&
					strings.Contains(output, "⏳")
			},
		},
		{
			name:    "empty working message",
			content: "",
			agentID: "worker-1",
			validate: func(output string) bool {
				return strings.Contains(output, "⚙️") && strings.Contains(output, "worker-1")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := formatter.FormatWorkingMessage(tt.content, tt.agentID)
			if !tt.validate(output) {
				t.Errorf("FormatWorkingMessage() validation failed for %s", tt.name)
			}
		})
	}
}

func TestContentFormatter_FormatUserMessage(t *testing.T) {
	renderer, err := chat.NewMarkdownRenderer(80)
	if err != nil {
		t.Fatalf("Failed to create markdown renderer: %v", err)
	}
	formatter := chat.NewContentFormatter(renderer, 80)

	tests := []struct {
		name     string
		content  string
		validate func(string) bool
	}{
		{
			name:    "simple user message",
			content: "Create a new feature",
			validate: func(output string) bool {
				return output == "Create a new feature"
			},
		},
		{
			name:    "user message with command",
			content: "@worker-1 implement the login system",
			validate: func(output string) bool {
				return strings.Contains(output, "@worker-1") &&
					strings.Contains(output, "implement the login system")
			},
		},
		{
			name:    "user message with markdown",
			content: "Please create a function that:\n- Takes **two** parameters\n- Returns a `string`",
			validate: func(output string) bool {
				return strings.Contains(output, "two") && strings.Contains(output, "string")
			},
		},
		{
			name:    "user message with code",
			content: "Fix this code:\n```go\nfmt.Printlnn(\"typo\")\n```",
			validate: func(output string) bool {
				return strings.Contains(output, "Fix this code") &&
					strings.Contains(output, "fmt.Printlnn")
			},
		},
		{
			name:    "empty user message",
			content: "",
			validate: func(output string) bool {
				return output == ""
			},
		},
		{
			name:    "user message with special characters",
			content: "What does `x && !y || z` mean?",
			validate: func(output string) bool {
				return strings.Contains(output, "&&") &&
					strings.Contains(output, "!y") &&
					strings.Contains(output, "||")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := formatter.FormatUserMessage(tt.content)
			if !tt.validate(output) {
				t.Errorf("FormatUserMessage() validation failed for %s", tt.name)
			}
		})
	}
}

func TestContentFormatter_FormatTimestamp(t *testing.T) {
	renderer, err := chat.NewMarkdownRenderer(80)
	if err != nil {
		t.Fatalf("Failed to create markdown renderer: %v", err)
	}
	formatter := chat.NewContentFormatter(renderer, 80)

	tests := []struct {
		name      string
		timestamp time.Time
		validate  func(string) bool
	}{
		{
			name:      "morning timestamp",
			timestamp: time.Date(2025, 1, 1, 9, 30, 45, 0, time.UTC),
			validate: func(output string) bool {
				return strings.Contains(output, "09:30:45")
			},
		},
		{
			name:      "afternoon timestamp",
			timestamp: time.Date(2025, 1, 1, 15, 45, 30, 0, time.UTC),
			validate: func(output string) bool {
				return strings.Contains(output, "15:45:30")
			},
		},
		{
			name:      "midnight timestamp",
			timestamp: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
			validate: func(output string) bool {
				return strings.Contains(output, "00:00:00")
			},
		},
		{
			name:      "single digit time",
			timestamp: time.Date(2025, 1, 1, 1, 2, 3, 0, time.UTC),
			validate: func(output string) bool {
				return strings.Contains(output, "01:02:03")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := formatter.FormatTimestamp(tt.timestamp)
			if !tt.validate(output) {
				t.Errorf("FormatTimestamp() = %v, validation failed", output)
			}
		})
	}
}

func TestContentFormatter_UpdateWidth(t *testing.T) {
	renderer, err := chat.NewMarkdownRenderer(80)
	if err != nil {
		t.Fatalf("Failed to create markdown renderer: %v", err)
	}
	formatter := chat.NewContentFormatter(renderer, 80)

	tests := []struct {
		name     string
		newWidth int
	}{
		{
			name:     "increase width",
			newWidth: 120,
		},
		{
			name:     "decrease width",
			newWidth: 60,
		},
		{
			name:     "same width",
			newWidth: 80,
		},
		{
			name:     "minimum width",
			newWidth: 20,
		},
		{
			name:     "maximum width",
			newWidth: 200,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			formatter.UpdateWidth(tt.newWidth)
			// We can't directly check the width field, but we can test that
			// the method doesn't panic and subsequent formatting works
			output := formatter.FormatUserMessage("test message")
			if !strings.Contains(output, "test message") {
				t.Error("UpdateWidth() affected basic message formatting")
			}
		})
	}
}

func TestContentFormatter_SetTheme(t *testing.T) {
	renderer, err := chat.NewMarkdownRenderer(80)
	if err != nil {
		t.Fatalf("Failed to create markdown renderer: %v", err)
	}
	formatter := chat.NewContentFormatter(renderer, 80)

	tests := []struct {
		name  string
		theme string
	}{
		{
			name:  "medieval theme",
			theme: "medieval",
		},
		{
			name:  "modern theme",
			theme: "modern",
		},
		{
			name:  "minimal theme",
			theme: "minimal",
		},
		{
			name:  "unknown theme",
			theme: "unknown",
		},
		{
			name:  "empty theme",
			theme: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Should not panic
			formatter.SetTheme(tt.theme)

			// Test that formatting still works after theme change
			output := formatter.FormatSystemMessage("test message")
			if !strings.Contains(output, "test message") {
				t.Error("SetTheme() broke basic message formatting")
			}
		})
	}
}
