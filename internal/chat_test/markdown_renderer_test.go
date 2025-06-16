// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package chat_test

import (
	"strings"
	"testing"

	"github.com/guild-ventures/guild-core/internal/chat"
)

func TestNewMarkdownRenderer(t *testing.T) {
	tests := []struct {
		name    string
		width   int
		wantErr bool
	}{
		{
			name:    "valid width",
			width:   80,
			wantErr: false,
		},
		{
			name:    "small width",
			width:   20,
			wantErr: false,
		},
		{
			name:    "large width",
			width:   200,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			renderer, err := chat.NewMarkdownRenderer(tt.width)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewMarkdownRenderer() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && renderer == nil {
				t.Error("NewMarkdownRenderer() returned nil renderer")
			}
		})
	}
}

func TestMarkdownRenderer_Render(t *testing.T) {
	renderer, err := chat.NewMarkdownRenderer(80)
	if err != nil {
		t.Fatalf("Failed to create renderer: %v", err)
	}

	tests := []struct {
		name     string
		content  string
		validate func(string) bool
	}{
		{
			name:    "plain text unchanged",
			content: "This is plain text without markdown",
			validate: func(output string) bool {
				return output == "This is plain text without markdown"
			},
		},
		{
			name:    "headers",
			content: "# Header 1\n## Header 2\n### Header 3",
			validate: func(output string) bool {
				// Glamour will render headers with formatting
				return strings.Contains(output, "Header 1") &&
					strings.Contains(output, "Header 2") &&
					strings.Contains(output, "Header 3")
			},
		},
		{
			name:    "bold and italic",
			content: "This is **bold** and this is *italic*",
			validate: func(output string) bool {
				return strings.Contains(output, "bold") && strings.Contains(output, "italic")
			},
		},
		{
			name:    "lists",
			content: "- Item 1\n- Item 2\n  - Subitem",
			validate: func(output string) bool {
				return strings.Contains(output, "Item 1") &&
					strings.Contains(output, "Item 2") &&
					strings.Contains(output, "Subitem")
			},
		},
		{
			name:    "mixed content with code",
			content: "Here's a function:\n```go\nfunc test() {}\n```\nAnd some **bold** text.",
			validate: func(output string) bool {
				return strings.Contains(output, "func test") && strings.Contains(output, "bold")
			},
		},
		{
			name:    "inline code",
			content: "Use `fmt.Println()` to print",
			validate: func(output string) bool {
				return strings.Contains(output, "fmt.Println")
			},
		},
		{
			name:    "links",
			content: "[Guild Framework](https://github.com/guild-ventures/guild-core)",
			validate: func(output string) bool {
				return strings.Contains(output, "Guild Framework")
			},
		},
		{
			name:    "blockquote",
			content: "> This is a quote\n> With multiple lines",
			validate: func(output string) bool {
				return strings.Contains(output, "This is a quote") &&
					strings.Contains(output, "With multiple lines")
			},
		},
		{
			name:    "horizontal rule",
			content: "Above\n\n---\n\nBelow",
			validate: func(output string) bool {
				return strings.Contains(output, "Above") && strings.Contains(output, "Below")
			},
		},
		{
			name:    "empty content",
			content: "",
			validate: func(output string) bool {
				return output == ""
			},
		},
		{
			name:    "malformed markdown",
			content: "**Unclosed bold",
			validate: func(output string) bool {
				// Should still contain the text even if markdown is malformed
				return strings.Contains(output, "Unclosed bold")
			},
		},
		{
			name:    "complex nested markdown",
			content: "# Title\n\nThis has **bold _nested italic_** text and `code`.\n\n```python\nprint('test')\n```",
			validate: func(output string) bool {
				return strings.Contains(output, "Title") &&
					strings.Contains(output, "bold") &&
					strings.Contains(output, "italic") &&
					strings.Contains(output, "print")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := renderer.Render(tt.content)
			if !tt.validate(output) {
				t.Errorf("Render() validation failed for %s\nInput: %s\nOutput: %s", tt.name, tt.content, output)
			}
		})
	}
}

func TestMarkdownRenderer_RenderInlineCode(t *testing.T) {
	renderer, err := chat.NewMarkdownRenderer(80)
	if err != nil {
		t.Fatalf("Failed to create renderer: %v", err)
	}

	tests := []struct {
		name string
		code string
		want string
	}{
		{
			name: "simple function call",
			code: "fmt.Println()",
			want: "fmt.Println()",
		},
		{
			name: "code with spaces",
			code: "go test ./...",
			want: "go test ./...",
		},
		{
			name: "empty code",
			code: "",
			want: "",
		},
		{
			name: "code with special characters",
			code: "x := &Config{}",
			want: "x := &Config{}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := renderer.RenderInlineCode(tt.code)
			// Check that the output contains the code (styled output will have escape sequences)
			if !strings.Contains(output, tt.want) {
				t.Errorf("RenderInlineCode() = %v, want to contain %v", output, tt.want)
			}
		})
	}
}

func TestMarkdownRenderer_DetectAndRenderContent(t *testing.T) {
	renderer, err := chat.NewMarkdownRenderer(80)
	if err != nil {
		t.Fatalf("Failed to create renderer: %v", err)
	}

	tests := []struct {
		name           string
		content        string
		shouldRender   bool
		validateOutput func(string) bool
	}{
		{
			name:         "plain text",
			content:      "Just regular text without any formatting",
			shouldRender: false,
			validateOutput: func(output string) bool {
				return output == "Just regular text without any formatting"
			},
		},
		{
			name:         "markdown content",
			content:      "# Title\n\nWith **bold** text",
			shouldRender: true,
			validateOutput: func(output string) bool {
				return strings.Contains(output, "Title") && strings.Contains(output, "bold")
			},
		},
		{
			name:         "code block content",
			content:      "```go\nfunc test() {}\n```",
			shouldRender: true,
			validateOutput: func(output string) bool {
				return strings.Contains(output, "func test")
			},
		},
		{
			name:         "inline code",
			content:      "Run `go test` to test",
			shouldRender: true,
			validateOutput: func(output string) bool {
				return strings.Contains(output, "go test")
			},
		},
		{
			name:         "text with URL",
			content:      "Visit https://example.com for more info",
			shouldRender: false,
			validateOutput: func(output string) bool {
				return output == "Visit https://example.com for more info"
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := renderer.DetectAndRenderContent(tt.content)
			if !tt.validateOutput(output) {
				t.Errorf("DetectAndRenderContent() failed validation for %s", tt.name)
			}
		})
	}
}

func TestMarkdownRenderer_EdgeCases(t *testing.T) {
	renderer, err := chat.NewMarkdownRenderer(80)
	if err != nil {
		t.Fatalf("Failed to create renderer: %v", err)
	}

	tests := []struct {
		name    string
		content string
		check   func(string) bool
	}{
		{
			name:    "very long line",
			content: strings.Repeat("a", 200),
			check: func(output string) bool {
				// Should handle long lines without panic
				return len(output) > 0
			},
		},
		{
			name:    "unicode content",
			content: "# 🏰 Guild Framework\n\nWith **unicode** 中文 content",
			check: func(output string) bool {
				return strings.Contains(output, "Guild Framework") &&
					strings.Contains(output, "中文")
			},
		},
		{
			name:    "mixed line endings",
			content: "Line 1\rLine 2\r\nLine 3\n",
			check: func(output string) bool {
				return strings.Contains(output, "Line 1") &&
					strings.Contains(output, "Line 2") &&
					strings.Contains(output, "Line 3")
			},
		},
		{
			name:    "deeply nested markdown",
			content: "**Bold with *italic and `code` inside***",
			check: func(output string) bool {
				return strings.Contains(output, "Bold") &&
					strings.Contains(output, "italic") &&
					strings.Contains(output, "code")
			},
		},
		{
			name:    "multiple empty code blocks",
			content: "```\n\n```\n\n```go\n\n```",
			check: func(output string) bool {
				// Should not crash on empty blocks
				return len(output) >= 0
			},
		},
		{
			name:    "code block with only whitespace",
			content: "```\n   \n\t\n```",
			check: func(output string) bool {
				return !strings.Contains(output, "```")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Should not panic
			output := renderer.Render(tt.content)
			if !tt.check(output) {
				t.Errorf("Edge case test failed for %s", tt.name)
			}
		})
	}
}

func BenchmarkMarkdownRenderer_Render(b *testing.B) {
	renderer, err := chat.NewMarkdownRenderer(80)
	if err != nil {
		b.Fatalf("Failed to create renderer: %v", err)
	}

	content := `# Benchmark Test

This is a **benchmark** test with various markdown elements.

## Code Block

` + "```go" + `
func TestBenchmark() {
    for i := 0; i < 100; i++ {
        fmt.Printf("Iteration %d\n", i)
    }
}
` + "```" + `

## Lists

- Item 1
- Item 2
  - Subitem A
  - Subitem B

And some *italic* text with ` + "`inline code`" + `.`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = renderer.Render(content)
	}
}
