// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package components

import (
	"context"
	"testing"
	"time"

	"github.com/guild-framework/guild-core/internal/ui/animation"
	"github.com/guild-framework/guild-core/internal/ui/theme"
)

// BenchmarkComponentLibrary_RenderButton benchmarks button rendering performance
// Target: <10ms for component rendering
func BenchmarkComponentLibrary_RenderButton(b *testing.B) {
	cl := setupComponentLibrary(nil)
	ctx := context.Background()

	button := Button{
		Text:    "Benchmark Button",
		Variant: ButtonPrimary,
		Size:    ButtonSizeMedium,
		State:   ButtonStateNormal,
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := cl.RenderButton(ctx, button)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkComponentLibrary_RenderModal benchmarks modal rendering performance
// Target: <10ms for component rendering
func BenchmarkComponentLibrary_RenderModal(b *testing.B) {
	cl := setupComponentLibrary(nil)
	ctx := context.Background()

	modal := Modal{
		Title:    "Benchmark Modal",
		Content:  "This is benchmark content for performance testing",
		Width:    60,
		Height:   20,
		Closable: true,
		Backdrop: true,
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := cl.RenderModal(ctx, modal)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkComponentLibrary_RenderAgentBadge benchmarks agent badge rendering
// Target: <10ms for component rendering
func BenchmarkComponentLibrary_RenderAgentBadge(b *testing.B) {
	cl := setupComponentLibrary(nil)
	ctx := context.Background()

	badge := AgentBadge{
		AgentID:  "benchmark-agent",
		Status:   AgentOnline,
		Size:     BadgeSizeMedium,
		ShowName: true,
		Animated: true,
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := cl.RenderAgentBadge(ctx, badge)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkComponentLibrary_RenderProgressBar benchmarks progress bar rendering
// Target: <10ms for component rendering
func BenchmarkComponentLibrary_RenderProgressBar(b *testing.B) {
	cl := setupComponentLibrary(nil)
	ctx := context.Background()

	progress := ProgressBar{
		Progress:    0.75,
		Width:       30,
		ShowPercent: true,
		ShowLabel:   true,
		Label:       "Benchmark Progress",
		Style:       ProgressStyleBar,
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := cl.RenderProgressBar(ctx, progress)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkComponentLibrary_RenderChatMessage benchmarks chat message rendering
// Target: <10ms for component rendering
func BenchmarkComponentLibrary_RenderChatMessage(b *testing.B) {
	cl := setupComponentLibrary(nil)
	ctx := context.Background()

	message := ChatMessage{
		Content:   "This is a benchmark chat message for performance testing",
		AgentID:   "benchmark-agent",
		Timestamp: time.Now(),
		Type:      MessageAgent,
		Reactions: []Reaction{
			{Emoji: "👍", Count: 3, Active: true},
			{Emoji: "🎉", Count: 1, Active: false},
		},
		Metadata: MessageMeta{
			Edited:   true,
			Mentions: []string{"agent-2", "agent-3"},
		},
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := cl.RenderChatMessage(ctx, message)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkButtonVariants benchmarks all button variants
func BenchmarkButtonVariants(b *testing.B) {
	cl := setupComponentLibrary(nil)
	ctx := context.Background()

	variants := []ButtonVariant{
		ButtonPrimary, ButtonSecondary, ButtonAccent,
		ButtonSuccess, ButtonWarning, ButtonDanger,
		ButtonGhost, ButtonLink,
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		variant := variants[i%len(variants)]
		button := Button{
			Text:    "Variant Button",
			Variant: variant,
			Size:    ButtonSizeMedium,
		}

		_, err := cl.RenderButton(ctx, button)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkProgressStyles benchmarks all progress bar styles
func BenchmarkProgressStyles(b *testing.B) {
	cl := setupComponentLibrary(nil)
	ctx := context.Background()

	styles := []ProgressStyle{
		ProgressStyleBar, ProgressCircle,
		ProgressRing, ProgressDots,
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		style := styles[i%len(styles)]
		progress := ProgressBar{
			Progress: 0.5,
			Width:    20,
			Style:    style,
		}

		_, err := cl.RenderProgressBar(ctx, progress)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkComponentLibrary_ThreadSafety benchmarks concurrent component rendering
func BenchmarkComponentLibrary_ThreadSafety(b *testing.B) {
	cl := setupComponentLibrary(nil)

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		ctx := context.Background()
		i := 0

		for pb.Next() {
			switch i % 5 {
			case 0:
				button := Button{Text: "Concurrent Button", Variant: ButtonPrimary}
				cl.RenderButton(ctx, button)
			case 1:
				modal := Modal{Title: "Concurrent Modal", Width: 40, Height: 15}
				cl.RenderModal(ctx, modal)
			case 2:
				badge := AgentBadge{AgentID: "concurrent-agent", Status: AgentOnline}
				cl.RenderAgentBadge(ctx, badge)
			case 3:
				progress := ProgressBar{Progress: 0.5, Style: ProgressStyleBar}
				cl.RenderProgressBar(ctx, progress)
			case 4:
				message := ChatMessage{Content: "Concurrent message", Type: MessageAgent}
				cl.RenderChatMessage(ctx, message)
			}
			i++
		}
	})
}

// TestComponentPerformanceThresholds validates component rendering performance
func TestComponentPerformanceThresholds(t *testing.T) {
	cl := setupComponentLibrary(t)
	ctx := context.Background()

	threshold := 10 * time.Millisecond

	t.Run("ButtonRenderingThreshold", func(t *testing.T) {
		button := Button{
			Text:    "Performance Test Button",
			Variant: ButtonPrimary,
			Size:    ButtonSizeLarge,
		}

		start := time.Now()
		_, err := cl.RenderButton(ctx, button)
		duration := time.Since(start)

		if err != nil {
			t.Fatal(err)
		}

		if duration > threshold {
			t.Errorf("Button rendering took %v, exceeds threshold of %v", duration, threshold)
		}

		t.Logf("Button rendering completed in %v", duration)
	})

	t.Run("ModalRenderingThreshold", func(t *testing.T) {
		modal := Modal{
			Title:    "Performance Test Modal",
			Content:  "This is content for performance testing",
			Width:    80,
			Height:   25,
			Backdrop: true,
			Buttons: []Button{
				{Text: "OK", Variant: ButtonPrimary},
				{Text: "Cancel", Variant: ButtonSecondary},
			},
		}

		start := time.Now()
		_, err := cl.RenderModal(ctx, modal)
		duration := time.Since(start)

		if err != nil {
			t.Fatal(err)
		}

		if duration > threshold {
			t.Errorf("Modal rendering took %v, exceeds threshold of %v", duration, threshold)
		}

		t.Logf("Modal rendering completed in %v", duration)
	})

	t.Run("AgentBadgeRenderingThreshold", func(t *testing.T) {
		badge := AgentBadge{
			AgentID:    "performance-test-agent",
			Status:     AgentThinking,
			Size:       BadgeSizeLarge,
			ShowName:   true,
			ShowStatus: true,
			Animated:   true,
		}

		start := time.Now()
		_, err := cl.RenderAgentBadge(ctx, badge)
		duration := time.Since(start)

		if err != nil {
			t.Fatal(err)
		}

		if duration > threshold {
			t.Errorf("Agent badge rendering took %v, exceeds threshold of %v", duration, threshold)
		}

		t.Logf("Agent badge rendering completed in %v", duration)
	})

	t.Run("ChatMessageRenderingThreshold", func(t *testing.T) {
		message := ChatMessage{
			Content:   "This is a comprehensive performance test message with reactions and metadata",
			AgentID:   "performance-agent",
			Timestamp: time.Now(),
			Type:      MessageAgent,
			Reactions: []Reaction{
				{Emoji: "👍", Count: 5, Active: true},
				{Emoji: "❤️", Count: 3, Active: false},
				{Emoji: "🚀", Count: 2, Active: true},
			},
			Metadata: MessageMeta{
				Edited:   true,
				ReplyTo:  "previous-message",
				Mentions: []string{"agent-2", "agent-3", "user"},
				Tags:     []string{"important", "performance"},
			},
		}

		start := time.Now()
		_, err := cl.RenderChatMessage(ctx, message)
		duration := time.Since(start)

		if err != nil {
			t.Fatal(err)
		}

		if duration > threshold {
			t.Errorf("Chat message rendering took %v, exceeds threshold of %v", duration, threshold)
		}

		t.Logf("Chat message rendering completed in %v", duration)
	})
}

// BenchmarkMemoryUsage tests memory efficiency of component rendering
func BenchmarkMemoryUsage(b *testing.B) {
	b.Run("ComponentLibraryMemory", func(b *testing.B) {
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			themeManager := theme.NewThemeManager()
			animator := animation.NewAnimator()
			cl := NewComponentLibrary(themeManager, animator)

			ctx := context.Background()

			// Render multiple components to test memory usage
			button := Button{Text: "Memory Test", Variant: ButtonPrimary}
			cl.RenderButton(ctx, button)

			modal := Modal{Title: "Memory Test", Width: 40, Height: 15}
			cl.RenderModal(ctx, modal)

			badge := AgentBadge{AgentID: "memory-agent", Status: AgentOnline}
			cl.RenderAgentBadge(ctx, badge)
		}
	})

	b.Run("ComponentReuseMemory", func(b *testing.B) {
		cl := setupComponentLibrary(nil)
		ctx := context.Background()

		button := Button{Text: "Reuse Test", Variant: ButtonPrimary}

		b.ReportAllocs()
		b.ResetTimer()

		// Test memory efficiency when reusing components
		for i := 0; i < b.N; i++ {
			cl.RenderButton(ctx, button)
		}
	})
}
