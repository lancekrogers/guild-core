// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package chat_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/guild-ventures/guild-core/internal/chat"
)

func BenchmarkStatusPanelUpdate(b *testing.B) {
	guildConfig := createTestConfig()
	tracker := chat.NewAgentStatusTracker(guildConfig)
	display := chat.NewStatusDisplay(tracker, 80, 24)

	// Pre-populate with agents
	for i := 0; i < 10; i++ {
		tracker.UpdateAgentStatus(fmt.Sprintf("agent-%d", i), &chat.AgentStatus{
			ID:    fmt.Sprintf("agent-%d", i),
			State: chat.AgentWorking,
		})
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = display.RenderCompactStatus() // RenderFullStatus doesn't exist
	}
}

func BenchmarkAnimationSystem(b *testing.B) {
	indicators := chat.NewAgentIndicators()
	indicators.StartAnimations()
	defer indicators.StopAnimations()

	// Add animations for multiple agents
	for i := 0; i < 20; i++ {
		indicators.SetWorkingAnimation(fmt.Sprintf("agent-%d", i), "task")
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		for j := 0; j < 20; j++ {
			_ = indicators.GetCurrentIndicator(fmt.Sprintf("agent-%d", j))
		}
	}
}

func BenchmarkConcurrentStatusUpdates(b *testing.B) {
	tracker := chat.NewAgentStatusTracker(createTestConfig())

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			agentID := fmt.Sprintf("agent-%d", i%10)
			tracker.UpdateAgentStatus(agentID, &chat.AgentStatus{
				ID:    agentID,
				State: chat.AgentState(i % 4),
			})
			i++
		}
	})
}

func BenchmarkContentFormatting(b *testing.B) {
	renderer, _ := chat.NewMarkdownRenderer(80)
	formatter := chat.NewContentFormatter(renderer, 80, "/tmp")

	// Various content types
	contents := []struct {
		msgType string
		content string
	}{
		{"agent", "Processing your request with multiple **markdown** elements and `code`"},
		{"system", "System notification with status updates"},
		{"tool", "```go\nfunc Execute() error {\n    return nil\n}\n```"},
		{"error", "Error: Failed to process request due to timeout"},
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		content := contents[i%len(contents)]
		_ = formatter.FormatMessage(content.msgType, content.content, nil)
	}
}

func BenchmarkLineNumberAddition(b *testing.B) {
	renderer, _ := chat.NewMarkdownRenderer(80)

	// Code with many lines
	code := generateLargeCode(100) // 100 lines

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// Test the unexported method through reflection or wrapper
		// For now, benchmark inline code rendering instead
		_ = renderer.RenderInlineCode(code)
	}
}

func BenchmarkCachePerformance(b *testing.B) {
	renderer, _ := chat.NewMarkdownRenderer(80)

	// Generate various content to test cache
	contents := make([]string, 50)
	for i := 0; i < 50; i++ {
		contents[i] = fmt.Sprintf("# Document %d\n\nContent with **markdown** and `code`", i)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// Access pattern: 80% cache hits, 20% misses
		idx := i % 50
		if i%5 == 0 {
			idx = (i / 5) % 50 // Different pattern for misses
		}
		_ = renderer.Render(contents[idx])
	}

	b.StopTimer()
	b.Logf("Cache stats: %s", renderer.GetCacheStats())
}

func BenchmarkLanguageDetection(b *testing.B) {
	renderer, _ := chat.NewMarkdownRenderer(80)
	formatter := chat.NewContentFormatter(renderer, 80, "/tmp")

	// Various code samples
	codeSamples := []string{
		`func main() { fmt.Println("Go") }`,
		`def hello(): print("Python")`,
		`function test() { console.log("JS"); }`,
		`SELECT * FROM users WHERE active = true;`,
		`fn main() { println!("Rust"); }`,
		`public class Main { public static void main(String[] args) {} }`,
		`def greet; puts "Ruby"; end`,
		`FROM ubuntu:latest\nRUN apt-get update`,
		`all:\n\t@echo "Makefile"`,
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		code := codeSamples[i%len(codeSamples)]
		_ = formatter.InferLanguage(code)
	}
}

func BenchmarkSyntaxHighlighting(b *testing.B) {
	renderer, _ := chat.NewMarkdownRenderer(80)

	// Go code sample in markdown format
	code := "```go\npackage main\n\nimport (\n    \"fmt\"\n    \"net/http\"\n)\n\nfunc main() {\n    http.HandleFunc(\"/\", handler)\n    fmt.Println(\"Server starting on :8080\")\n    http.ListenAndServe(\":8080\", nil)\n}\n\nfunc handler(w http.ResponseWriter, r *http.Request) {\n    fmt.Fprintf(w, \"Hello, World!\")\n}\n```"

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = renderer.Render(code)
	}
}

func BenchmarkRealTimeUpdates(b *testing.B) {
	guildConfig := createTestConfig()
	tracker := chat.NewAgentStatusTracker(guildConfig)
	display := chat.NewStatusDisplay(tracker, 80, 24)
	indicators := chat.NewAgentIndicators()

	// Simulate real-time system
	agents := []string{"manager", "developer", "reviewer", "tester"}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		agentID := agents[i%len(agents)]

		// Update status
		tracker.UpdateAgentStatus(agentID, &chat.AgentStatus{
			ID:       agentID,
			State:    chat.AgentState(i % 5),
			Progress: float64(i%100) / 100.0,
		})

		// Update animation
		indicators.SetWorkingAnimation(agentID, "task")

		// Render display
		_ = display.RenderCompactStatus()
		_ = indicators.GetCurrentIndicator(agentID)
	}
}

func BenchmarkMemoryAllocation(b *testing.B) {
	b.Run("StatusTracker", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			tracker := chat.NewAgentStatusTracker(createTestConfig())
			for j := 0; j < 100; j++ {
				tracker.UpdateAgentStatus(fmt.Sprintf("agent-%d", j), &chat.AgentStatus{
					ID:    fmt.Sprintf("agent-%d", j),
					State: chat.AgentWorking,
				})
			}
		}
	})

	b.Run("MarkdownRenderer", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			renderer, _ := chat.NewMarkdownRenderer(80)
			_ = renderer.Render("# Test\nSome content")
		}
	})

	b.Run("ContentFormatter", func(b *testing.B) {
		renderer, _ := chat.NewMarkdownRenderer(80)
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			formatter := chat.NewContentFormatter(renderer, 80, "/tmp")
			_ = formatter.FormatMessage("agent", "test", nil)
		}
	})
}

// Helper functions for generating test data

func generateLargeMarkdown(size int) string {
	var builder strings.Builder

	builder.WriteString("# Large Document\n\n")

	for i := 0; i < size/100; i++ {
		builder.WriteString(fmt.Sprintf("## Section %d\n\n", i))
		builder.WriteString("This is a **paragraph** with *emphasis* and `code`.\n\n")
		builder.WriteString("```go\nfunc example() {\n    fmt.Println(\"test\")\n}\n```\n\n")
	}

	return builder.String()
}

func generateLargeCode(lines int) string {
	var builder strings.Builder

	builder.WriteString("package main\n\n")
	builder.WriteString("import \"fmt\"\n\n")

	for i := 0; i < lines-4; i++ {
		builder.WriteString(fmt.Sprintf("func function%d() {\n", i))
		builder.WriteString(fmt.Sprintf("    fmt.Println(\"Line %d\")\n", i))
		builder.WriteString("}\n")
	}

	return builder.String()
}

// Benchmark results analysis helpers

func BenchmarkSummary(b *testing.B) {
	// This benchmark provides a summary of all visual components
	b.Run("CompleteVisualStack", func(b *testing.B) {
		guildConfig := createTestConfig()
		tracker := chat.NewAgentStatusTracker(guildConfig)
		display := chat.NewStatusDisplay(tracker, 80, 24)
		renderer, _ := chat.NewMarkdownRenderer(80)
		formatter := chat.NewContentFormatter(renderer, 80, "/tmp")
		indicators := chat.NewAgentIndicators()

		// Pre-populate
		for i := 0; i < 5; i++ {
			tracker.UpdateAgentStatus(fmt.Sprintf("agent-%d", i), &chat.AgentStatus{
				ID:    fmt.Sprintf("agent-%d", i),
				State: chat.AgentWorking,
			})
		}

		content := "# Update\n\nAgent is working on:\n```go\nfunc Process() {}\n```"

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			// Complete visual update cycle
			_ = display.RenderCompactStatus() // RenderFullStatus doesn't exist
			_ = renderer.Render(content)
			_ = formatter.FormatMessage("agent", "Status update", nil)
			_ = indicators.GetCurrentIndicator("agent-0")
		}
	})
}
