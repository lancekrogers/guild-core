package main

import (
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/lipgloss"
)

func TestNewContentFormatter(t *testing.T) {
	renderer, err := NewMarkdownRenderer(80)
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
			formatter := NewContentFormatter(renderer, tt.width)
			if formatter == nil {
				t.Error("NewContentFormatter() returned nil")
			}
			if formatter.width != tt.width {
				t.Errorf("NewContentFormatter() width = %v, want %v", formatter.width, tt.width)
			}
			if formatter.markdownRenderer == nil {
				t.Error("NewContentFormatter() markdownRenderer is nil")
			}
			if len(formatter.messageStyles) == 0 {
				t.Error("NewContentFormatter() messageStyles is empty")
			}
		})
	}
}

func TestContentFormatter_FormatAgentResponse(t *testing.T) {
	renderer, err := NewMarkdownRenderer(80)
	if err != nil {
		t.Fatalf("Failed to create markdown renderer: %v", err)
	}
	formatter := NewContentFormatter(renderer, 80)

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
	renderer, err := NewMarkdownRenderer(80)
	if err != nil {
		t.Fatalf("Failed to create markdown renderer: %v", err)
	}
	formatter := NewContentFormatter(renderer, 80)

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
	renderer, err := NewMarkdownRenderer(80)
	if err != nil {
		t.Fatalf("Failed to create markdown renderer: %v", err)
	}
	formatter := NewContentFormatter(renderer, 80)

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
	renderer, err := NewMarkdownRenderer(80)
	if err != nil {
		t.Fatalf("Failed to create markdown renderer: %v", err)
	}
	formatter := NewContentFormatter(renderer, 80)

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
	renderer, err := NewMarkdownRenderer(80)
	if err != nil {
		t.Fatalf("Failed to create markdown renderer: %v", err)
	}
	formatter := NewContentFormatter(renderer, 80)

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
	renderer, err := NewMarkdownRenderer(80)
	if err != nil {
		t.Fatalf("Failed to create markdown renderer: %v", err)
	}
	formatter := NewContentFormatter(renderer, 80)

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
	renderer, err := NewMarkdownRenderer(80)
	if err != nil {
		t.Fatalf("Failed to create markdown renderer: %v", err)
	}
	formatter := NewContentFormatter(renderer, 80)

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
	renderer, err := NewMarkdownRenderer(80)
	if err != nil {
		t.Fatalf("Failed to create markdown renderer: %v", err)
	}
	formatter := NewContentFormatter(renderer, 80)

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

func TestContentFormatter_isImportantSystemMessage(t *testing.T) {
	renderer, err := NewMarkdownRenderer(80)
	if err != nil {
		t.Fatalf("Failed to create markdown renderer: %v", err)
	}
	formatter := NewContentFormatter(renderer, 80)

	tests := []struct {
		name    string
		content string
		want    bool
	}{
		{
			name:    "error message",
			content: "Error: connection timeout",
			want:    true,
		},
		{
			name:    "failed message",
			content: "Build failed with exit code 1",
			want:    true,
		},
		{
			name:    "warning message",
			content: "Warning: deprecated function used",
			want:    true,
		},
		{
			name:    "critical message",
			content: "Critical: system overload",
			want:    true,
		},
		{
			name:    "completed message",
			content: "Task completed successfully",
			want:    true,
		},
		{
			name:    "ready message",
			content: "System ready for input",
			want:    true,
		},
		{
			name:    "started message",
			content: "Process started",
			want:    true,
		},
		{
			name:    "disconnected message",
			content: "Client disconnected",
			want:    true,
		},
		{
			name:    "regular message",
			content: "Processing request",
			want:    false,
		},
		{
			name:    "empty message",
			content: "",
			want:    false,
		},
		{
			name:    "case insensitive check",
			content: "ERROR: SOMETHING WENT WRONG",
			want:    true,
		},
		{
			name:    "partial match",
			content: "The operation has finished",
			want:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := formatter.isImportantSystemMessage(tt.content); got != tt.want {
				t.Errorf("isImportantSystemMessage() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestContentFormatter_GetMessageStyle(t *testing.T) {
	renderer, err := NewMarkdownRenderer(80)
	if err != nil {
		t.Fatalf("Failed to create markdown renderer: %v", err)
	}
	formatter := NewContentFormatter(renderer, 80)

	tests := []struct {
		name        string
		messageType string
		shouldExist bool
	}{
		{
			name:        "agent style",
			messageType: "agent",
			shouldExist: true,
		},
		{
			name:        "system style",
			messageType: "system",
			shouldExist: true,
		},
		{
			name:        "error style",
			messageType: "error",
			shouldExist: true,
		},
		{
			name:        "tool style",
			messageType: "tool",
			shouldExist: true,
		},
		{
			name:        "user style",
			messageType: "user",
			shouldExist: true,
		},
		{
			name:        "thinking style",
			messageType: "thinking",
			shouldExist: true,
		},
		{
			name:        "working style",
			messageType: "working",
			shouldExist: true,
		},
		{
			name:        "unknown style",
			messageType: "unknown",
			shouldExist: false,
		},
		{
			name:        "empty style",
			messageType: "",
			shouldExist: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			style := formatter.GetMessageStyle(tt.messageType)
			// Check if we got the expected style or fallback
			if tt.shouldExist {
				if _, exists := formatter.messageStyles[tt.messageType]; !exists {
					t.Errorf("GetMessageStyle() should return style for %s", tt.messageType)
				}
			} else {
				// Should return system style as fallback
				systemStyle := formatter.messageStyles["system"]
				if style.String() != systemStyle.String() {
					t.Errorf("GetMessageStyle() should return system style for unknown type %s", tt.messageType)
				}
			}
		})
	}
}

func TestContentFormatter_UpdateWidth(t *testing.T) {
	renderer, err := NewMarkdownRenderer(80)
	if err != nil {
		t.Fatalf("Failed to create markdown renderer: %v", err)
	}
	formatter := NewContentFormatter(renderer, 80)

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
			if formatter.width != tt.newWidth {
				t.Errorf("UpdateWidth() width = %v, want %v", formatter.width, tt.newWidth)
			}
		})
	}
}

func TestContentFormatter_SetTheme(t *testing.T) {
	renderer, err := NewMarkdownRenderer(80)
	if err != nil {
		t.Fatalf("Failed to create markdown renderer: %v", err)
	}
	formatter := NewContentFormatter(renderer, 80)

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

			// Verify theme was applied
			switch tt.theme {
			case "modern":
				// Check if modern theme was applied
				// Just verify the styles exist
				if len(formatter.messageStyles) == 0 {
					t.Error("Modern theme not applied correctly")
				}
			case "minimal":
				// Check if minimal theme was applied (all styles should be the same)
				agentStyle := formatter.messageStyles["agent"]
				systemStyle := formatter.messageStyles["system"]
				if agentStyle.String() != systemStyle.String() {
					t.Error("Minimal theme not applied correctly")
				}
			}
		})
	}
}

func TestContentFormatter_EdgeCases(t *testing.T) {
	renderer, err := NewMarkdownRenderer(80)
	if err != nil {
		t.Fatalf("Failed to create markdown renderer: %v", err)
	}
	formatter := NewContentFormatter(renderer, 80)

	tests := []struct {
		name string
		test func()
	}{
		{
			name: "nil markdown renderer recovery",
			test: func() {
				// Create formatter with nil renderer
				nilFormatter := NewContentFormatter(nil, 80)
				// Should not panic
				defer func() {
					if r := recover(); r != nil {
						t.Errorf("Method panicked with nil renderer: %v", r)
					}
				}()
				// These would panic if not handled properly
				_ = nilFormatter.FormatAgentResponse("test", "agent")
			},
		},
		{
			name: "very long content",
			test: func() {
				longContent := strings.Repeat("This is a very long message. ", 1000)
				output := formatter.FormatAgentResponse(longContent, "agent-1")
				if !strings.Contains(output, "agent-1") {
					t.Error("Long content not handled properly")
				}
			},
		},
		{
			name: "special characters in agent ID",
			test: func() {
				output := formatter.FormatAgentResponse("Test", "agent-<script>alert('xss')</script>")
				if !strings.Contains(output, "Test") {
					t.Error("Special characters in agent ID not handled")
				}
			},
		},
		{
			name: "concurrent access",
			test: func() {
				// Test concurrent formatting
				done := make(chan bool, 10)
				for i := 0; i < 10; i++ {
					go func(n int) {
						defer func() { done <- true }()
						_ = formatter.FormatAgentResponse("Concurrent test", "agent")
						_ = formatter.FormatSystemMessage("System test")
						_ = formatter.FormatErrorMessage("Error test")
					}(i)
				}
				// Wait for all goroutines
				for i := 0; i < 10; i++ {
					<-done
				}
			},
		},
		{
			name: "empty styles map",
			test: func() {
				emptyFormatter := &ContentFormatter{
					markdownRenderer: renderer,
					width:            80,
					messageStyles:    make(map[string]lipgloss.Style),
				}
				// Should not panic
				_ = emptyFormatter.GetMessageStyle("agent")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.test()
		})
	}
}

func BenchmarkContentFormatter_FormatAgentResponse(b *testing.B) {
	renderer, err := NewMarkdownRenderer(80)
	if err != nil {
		b.Fatalf("Failed to create renderer: %v", err)
	}
	formatter := NewContentFormatter(renderer, 80)

	content := `# Agent Response

Here's the analysis:

## Code Review
` + "```go" + `
func processData(items []string) {
    for _, item := range items {
        fmt.Println(item)
    }
}
` + "```" + `

The function looks good with **O(n)** complexity.`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = formatter.FormatAgentResponse(content, "analyzer-1")
	}
}

func BenchmarkContentFormatter_FormatSystemMessage(b *testing.B) {
	renderer, err := NewMarkdownRenderer(80)
	if err != nil {
		b.Fatalf("Failed to create renderer: %v", err)
	}
	formatter := NewContentFormatter(renderer, 80)

	content := "Task completed successfully with 3 warnings"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = formatter.FormatSystemMessage(content)
	}
}

func BenchmarkContentFormatter_AllMessageTypes(b *testing.B) {
	renderer, err := NewMarkdownRenderer(80)
	if err != nil {
		b.Fatalf("Failed to create renderer: %v", err)
	}
	formatter := NewContentFormatter(renderer, 80)

	messages := []struct {
		content string
		format  func(string) string
	}{
		{"Agent response", func(s string) string { return formatter.FormatAgentResponse(s, "agent-1") }},
		{"System message", formatter.FormatSystemMessage},
		{"Error occurred", formatter.FormatErrorMessage},
		{"Tool output", func(s string) string { return formatter.FormatToolOutput(s, "tool-1") }},
		{"User input", formatter.FormatUserMessage},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, msg := range messages {
			_ = msg.format(msg.content)
		}
	}
}
