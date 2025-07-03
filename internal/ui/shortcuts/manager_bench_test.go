// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package shortcuts

import (
	"context"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"go.uber.org/zap"
)

// BenchmarkShortcutManager_HandleKeyPress benchmarks key press handling
// Target: <10ms for key handling operations
func BenchmarkShortcutManager_HandleKeyPress(b *testing.B) {
	sm := NewShortcutManagerWithLogger(zap.NewNop())
	ctx := context.Background()
	
	keys := []string{
		"ctrl+shift+p", "ctrl+p", "ctrl+1", "ctrl+2", "ctrl+3",
		"ctrl+n", "ctrl+f", "esc", "ctrl+shift+d",
	}
	
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		key := keys[i%len(keys)]
		cmd := sm.HandleKeyPress(ctx, key)
		_ = cmd // Prevent optimization
	}
}

// BenchmarkShortcutManager_RegisterShortcut benchmarks shortcut registration
func BenchmarkShortcutManager_RegisterShortcut(b *testing.B) {
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		sm := NewShortcutManagerWithLogger(zap.NewNop())
		
		shortcut := &Shortcut{
			ID:          "benchmark_shortcut",
			Key:         "ctrl+b",
			Command:     "benchmark.command",
			Description: "Benchmark shortcut",
			Handler:     func(ctx context.Context) tea.Cmd { return nil },
			Enabled:     true,
		}
		
		err := sm.RegisterShortcut(shortcut)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkCommandPalette_FilterCommands benchmarks command filtering
// Target: <50ms for search operations
func BenchmarkCommandPalette_FilterCommands(b *testing.B) {
	palette := NewCommandPalette()
	
	queries := []string{
		"commission", "agent", "shortcut", "theme", "debug",
		"performance", "chat", "kanban", "help", "search",
	}
	
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		query := queries[i%len(queries)]
		palette.filterCommands(query)
	}
}

// BenchmarkCommandPalette_Navigation benchmarks command palette navigation
func BenchmarkCommandPalette_Navigation(b *testing.B) {
	palette := NewCommandPalette()
	
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		switch i % 4 {
		case 0:
			palette.SelectNext()
		case 1:
			palette.SelectPrevious()
		case 2:
			palette.Show()
		case 3:
			palette.Hide()
		}
	}
}

// BenchmarkShortcutManager_SetContext benchmarks context switching
func BenchmarkShortcutManager_SetContext(b *testing.B) {
	sm := NewShortcutManagerWithLogger(zap.NewNop())
	ctx := context.Background()
	
	contexts := []string{"global", "chat", "kanban", "modal", "search"}
	
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		contextName := contexts[i%len(contexts)]
		err := sm.SetContext(ctx, contextName)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkShortcutManager_ThreadSafety benchmarks concurrent shortcut operations
func BenchmarkShortcutManager_ThreadSafety(b *testing.B) {
	sm := NewShortcutManagerWithLogger(zap.NewNop())
	
	b.ResetTimer()
	b.ReportAllocs()
	
	b.RunParallel(func(pb *testing.PB) {
		ctx := context.Background()
		i := 0
		
		for pb.Next() {
			switch i % 5 {
			case 0:
				sm.HandleKeyPress(ctx, "ctrl+shift+p")
			case 1:
				sm.SetContext(ctx, "chat")
			case 2:
				sm.ListShortcuts()
			case 3:
				sm.ShowCommandPalette()
			case 4:
				sm.HideCommandPalette()
			}
			i++
		}
	})
}

// BenchmarkKeyNormalization benchmarks key combination normalization
func BenchmarkKeyNormalization(b *testing.B) {
	sm := NewShortcutManagerWithLogger(zap.NewNop())
	
	keys := []string{
		"CTRL+SHIFT+P", "ctrl+alt+d", "command+k", "option+cmd+r",
		"control+shift+alt+x", "shift+ctrl+alt+y", "CMD+OPTION+Z",
	}
	
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		key := keys[i%len(keys)]
		normalized := sm.normalizeKey(key)
		_ = normalized // Prevent optimization
	}
}

// TestShortcutPerformanceThresholds validates shortcut system performance
func TestShortcutPerformanceThresholds(t *testing.T) {
	sm := NewShortcutManagerWithLogger(zap.NewNop())
	ctx := context.Background()
	
	t.Run("KeyHandlingThreshold", func(t *testing.T) {
		threshold := 10 * time.Millisecond
		
		start := time.Now()
		cmd := sm.HandleKeyPress(ctx, "ctrl+shift+p")
		duration := time.Since(start)
		
		_ = cmd // Prevent optimization
		
		if duration > threshold {
			t.Errorf("Key handling took %v, exceeds threshold of %v", duration, threshold)
		}
		
		t.Logf("Key handling completed in %v", duration)
	})
	
	t.Run("SearchOperationThreshold", func(t *testing.T) {
		palette := NewCommandPalette()
		threshold := 50 * time.Millisecond
		
		start := time.Now()
		palette.filterCommands("comprehensive search query with multiple terms")
		duration := time.Since(start)
		
		if duration > threshold {
			t.Errorf("Search operation took %v, exceeds threshold of %v", duration, threshold)
		}
		
		t.Logf("Search operation completed in %v", duration)
	})
	
	t.Run("ContextSwitchingThreshold", func(t *testing.T) {
		threshold := 5 * time.Millisecond
		
		start := time.Now()
		err := sm.SetContext(ctx, "chat")
		duration := time.Since(start)
		
		if err != nil {
			t.Fatal(err)
		}
		
		if duration > threshold {
			t.Errorf("Context switching took %v, exceeds threshold of %v", duration, threshold)
		}
		
		t.Logf("Context switching completed in %v", duration)
	})
	
	t.Run("ShortcutRegistrationThreshold", func(t *testing.T) {
		threshold := 5 * time.Millisecond
		
		shortcut := &Shortcut{
			ID:          "performance_test",
			Key:         "ctrl+t",
			Command:     "test.command",
			Description: "Performance test shortcut",
			Handler:     func(ctx context.Context) tea.Cmd { return nil },
			Enabled:     true,
		}
		
		start := time.Now()
		err := sm.RegisterShortcut(shortcut)
		duration := time.Since(start)
		
		if err != nil {
			t.Fatal(err)
		}
		
		if duration > threshold {
			t.Errorf("Shortcut registration took %v, exceeds threshold of %v", duration, threshold)
		}
		
		t.Logf("Shortcut registration completed in %v", duration)
	})
}

// BenchmarkMemoryUsage tests memory efficiency of shortcut system
func BenchmarkMemoryUsage(b *testing.B) {
	b.Run("ShortcutManagerMemory", func(b *testing.B) {
		b.ReportAllocs()
		
		for i := 0; i < b.N; i++ {
			sm := NewShortcutManagerWithLogger(zap.NewNop())
			ctx := context.Background()
			
			// Register multiple shortcuts to test memory usage
			for j := 0; j < 20; j++ {
				shortcut := &Shortcut{
					ID:      "memory_test_" + string(rune(j)),
					Key:     "ctrl+" + string(rune('a'+j)),
					Command: "memory.test",
					Handler: func(ctx context.Context) tea.Cmd { return nil },
					Enabled: true,
				}
				sm.RegisterShortcut(shortcut)
			}
			
			// Perform operations to test memory efficiency
			sm.HandleKeyPress(ctx, "ctrl+a")
			sm.SetContext(ctx, "chat")
			sm.ListShortcuts()
		}
	})
	
	b.Run("CommandPaletteMemory", func(b *testing.B) {
		b.ReportAllocs()
		
		for i := 0; i < b.N; i++ {
			palette := NewCommandPalette()
			
			// Test memory usage during filtering operations
			queries := []string{"commission", "agent", "debug", "performance", "search"}
			for _, query := range queries {
				palette.filterCommands(query)
			}
			
			// Test navigation memory usage
			for j := 0; j < 10; j++ {
				palette.SelectNext()
				palette.SelectPrevious()
			}
		}
	})
	
	b.Run("ShortcutCachingMemory", func(b *testing.B) {
		sm := NewShortcutManagerWithLogger(zap.NewNop())
		ctx := context.Background()
		
		b.ReportAllocs()
		b.ResetTimer()
		
		// Test that repeated operations don't cause memory leaks
		for i := 0; i < b.N; i++ {
			key := "ctrl+" + string(rune('a'+(i%26))) // Cycle through 26 keys
			sm.HandleKeyPress(ctx, key)
		}
	})
}