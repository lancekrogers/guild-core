// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package theme

import (
	"context"
	"testing"
	"time"
)

// BenchmarkThemeManager_ApplyTheme benchmarks theme switching performance
// Target: <16ms for theme switching operations
func BenchmarkThemeManager_ApplyTheme(b *testing.B) {
	tm := NewThemeManager()
	ctx := context.Background()
	
	themes := []string{"claude-code-light", "claude-code-dark"}
	
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		theme := themes[i%len(themes)]
		err := tm.ApplyTheme(ctx, theme)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkThemeManager_GetComponent benchmarks component style retrieval
// Target: <10ms for component rendering operations
func BenchmarkThemeManager_GetComponent(b *testing.B) {
	tm := NewThemeManager()
	ctx := context.Background()
	tm.ApplyTheme(ctx, "claude-code-light")
	
	components := []string{
		"button.primary",
		"button.secondary", 
		"input.default",
		"chat.message",
	}
	
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		component := components[i%len(components)]
		style := tm.GetComponent(component)
		_ = style // Prevent optimization
	}
}

// BenchmarkThemeManager_GetAgentStyle benchmarks dynamic agent color generation
// Target: <10ms for agent styling operations
func BenchmarkThemeManager_GetAgentStyle(b *testing.B) {
	tm := NewThemeManager()
	ctx := context.Background()
	tm.ApplyTheme(ctx, "claude-code-light")
	
	agentIDs := []string{
		"agent-1", "agent-2", "agent-3", "agent-4", "agent-5",
		"custom-agent", "user-defined-agent", "dynamic-agent-123",
	}
	
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		agentID := agentIDs[i%len(agentIDs)]
		style := tm.GetAgentStyle(agentID)
		_ = style // Prevent optimization
	}
}

// BenchmarkAgentColorGeneration benchmarks the color generation algorithm
// Target: <5ms for color generation
func BenchmarkAgentColorGeneration(b *testing.B) {
	tm := NewThemeManager()
	ctx := context.Background()
	tm.ApplyTheme(ctx, "claude-code-light")
	
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		agentID := "test-agent-" + string(rune(i%1000))
		color := tm.generateAgentColor(agentID)
		_ = color // Prevent optimization
	}
}

// BenchmarkThemeManager_ThreadSafety benchmarks concurrent theme operations
func BenchmarkThemeManager_ThreadSafety(b *testing.B) {
	tm := NewThemeManager()
	ctx := context.Background()
	
	b.ResetTimer()
	b.ReportAllocs()
	
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			switch i % 4 {
			case 0:
				tm.ApplyTheme(ctx, "claude-code-light")
			case 1:
				tm.ApplyTheme(ctx, "claude-code-dark")
			case 2:
				tm.GetComponent("button.primary")
			case 3:
				tm.GetAgentStyle("concurrent-agent")
			}
			i++
		}
	})
}

// TestPerformanceThresholds validates performance requirements
func TestPerformanceThresholds(t *testing.T) {
	tm := NewThemeManager()
	ctx := context.Background()
	
	// Test theme switching threshold: <16ms
	t.Run("ThemeSwitchingThreshold", func(t *testing.T) {
		start := time.Now()
		err := tm.ApplyTheme(ctx, "claude-code-dark")
		duration := time.Since(start)
		
		if err != nil {
			t.Fatal(err)
		}
		
		threshold := 16 * time.Millisecond
		if duration > threshold {
			t.Errorf("Theme switching took %v, exceeds threshold of %v", duration, threshold)
		}
		
		t.Logf("Theme switching completed in %v", duration)
	})
	
	// Test component rendering threshold: <10ms
	t.Run("ComponentRenderingThreshold", func(t *testing.T) {
		start := time.Now()
		style := tm.GetComponent("button.primary")
		duration := time.Since(start)
		
		_ = style // Prevent optimization
		
		threshold := 10 * time.Millisecond
		if duration > threshold {
			t.Errorf("Component rendering took %v, exceeds threshold of %v", duration, threshold)
		}
		
		t.Logf("Component rendering completed in %v", duration)
	})
	
	// Test agent styling threshold: <10ms
	t.Run("AgentStylingThreshold", func(t *testing.T) {
		start := time.Now()
		style := tm.GetAgentStyle("performance-test-agent")
		duration := time.Since(start)
		
		_ = style // Prevent optimization
		
		threshold := 10 * time.Millisecond
		if duration > threshold {
			t.Errorf("Agent styling took %v, exceeds threshold of %v", duration, threshold)
		}
		
		t.Logf("Agent styling completed in %v", duration)
	})
	
	// Test color generation threshold: <5ms
	t.Run("ColorGenerationThreshold", func(t *testing.T) {
		start := time.Now()
		color := tm.generateAgentColor("threshold-test-agent")
		duration := time.Since(start)
		
		_ = color // Prevent optimization
		
		threshold := 5 * time.Millisecond
		if duration > threshold {
			t.Errorf("Color generation took %v, exceeds threshold of %v", duration, threshold)
		}
		
		t.Logf("Color generation completed in %v", duration)
	})
}

// BenchmarkMemoryUsage tests memory efficiency
func BenchmarkMemoryUsage(b *testing.B) {
	b.Run("ThemeManagerMemory", func(b *testing.B) {
		b.ReportAllocs()
		
		for i := 0; i < b.N; i++ {
			tm := NewThemeManager()
			ctx := context.Background()
			tm.ApplyTheme(ctx, "claude-code-light")
			
			// Generate colors for multiple agents to test memory growth
			for j := 0; j < 10; j++ {
				tm.GetAgentStyle("agent-" + string(rune(j)))
			}
		}
	})
	
	b.Run("AgentColorCaching", func(b *testing.B) {
		tm := NewThemeManager()
		ctx := context.Background()
		tm.ApplyTheme(ctx, "claude-code-light")
		
		b.ReportAllocs()
		b.ResetTimer()
		
		// Test that agent colors are cached and don't cause memory leaks
		for i := 0; i < b.N; i++ {
			agentID := "cached-agent-" + string(rune(i%100)) // Reuse 100 agent IDs
			tm.GetAgentStyle(agentID)
		}
	})
}