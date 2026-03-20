// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package reasoning

import (
	"fmt"
	"testing"
	"time"

	"github.com/lancekrogers/guild-core/pkg/agents/core"
	viewutil "github.com/lancekrogers/guild-core/internal/ui/view"
	"github.com/stretchr/testify/require"
)

// BenchmarkReasoningDisplay benchmarks the reasoning display performance
func BenchmarkReasoningDisplay(b *testing.B) {
	// Test different block counts
	blockCounts := []int{10, 50, 100, 500, 1000}

	for _, count := range blockCounts {
		b.Run(fmt.Sprintf("blocks_%d", count), func(b *testing.B) {
			// Create test blocks
			blocks := generateTestBlocks(count)
			display := NewReasoningDisplay(80, 24)

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				// Simulate rendering
				display.blocks = blocks
				display.View()
			}
		})
	}
}

// BenchmarkRenderTime benchmarks render time for different scenarios
func BenchmarkRenderTime(b *testing.B) {
	scenarios := []struct {
		name        string
		blockCount  int
		contentSize int
		nested      bool
	}{
		{"small_simple", 10, 100, false},
		{"medium_simple", 50, 200, false},
		{"large_simple", 100, 300, false},
		{"small_nested", 10, 100, true},
		{"medium_nested", 50, 200, true},
		{"large_nested", 100, 300, true},
	}

	for _, scenario := range scenarios {
		b.Run(scenario.name, func(b *testing.B) {
			display := NewReasoningDisplay(120, 40)

			// Generate blocks based on scenario
			var blocks []*core.ThinkingBlock
			if scenario.nested {
				blocks = generateNestedBlocks(scenario.blockCount, scenario.contentSize)
			} else {
				blocks = generateTestBlocksWithSize(scenario.blockCount, scenario.contentSize)
			}

			// Add all blocks to display
			display.blocks = blocks

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				// Measure render time
				start := time.Now()
				rendered := viewutil.String(display.View())
				elapsed := time.Since(start)

				// Ensure render time is under 10ms
				if elapsed > 10*time.Millisecond {
					b.Errorf("Render time exceeded 10ms: %v", elapsed)
				}

				// Ensure output is not empty
				if len(rendered) == 0 {
					b.Error("Rendered output is empty")
				}
			}
		})
	}
}

// BenchmarkCollapseExpand benchmarks collapse/expand performance
func BenchmarkCollapseExpand(b *testing.B) {
	display := NewReasoningDisplay(80, 24)

	// Add 100 blocks
	blocks := generateTestBlocks(100)
	display.blocks = blocks

	b.Run("collapse_all", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			// Collapse all blocks
			for _, block := range blocks {
				display.toggleCollapse(block.ID)
			}
		}
	})

	b.Run("expand_all", func(b *testing.B) {
		// First collapse all
		for _, block := range blocks {
			display.collapsed[block.ID] = true
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			// Expand all blocks
			for _, block := range blocks {
				display.toggleCollapse(block.ID)
			}
		}
	})
}

// BenchmarkMemoryUsage benchmarks memory usage
func BenchmarkMemoryUsage(b *testing.B) {
	blockCounts := []int{100, 500, 1000, 5000}

	for _, count := range blockCounts {
		b.Run(fmt.Sprintf("blocks_%d", count), func(b *testing.B) {
			b.ReportAllocs()

			display := NewReasoningDisplay(80, 24)
			blocks := generateTestBlocksWithSize(count, 500) // Large content

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				// Add blocks and render
				display.blocks = blocks
				display.View()

				// Clear for next iteration
				display.blocks = nil
			}
		})
	}
}

// TestRenderPerformanceRequirements tests specific performance requirements
func TestRenderPerformanceRequirements(t *testing.T) {
	display := NewReasoningDisplay(120, 40)

	t.Run("60fps_requirement", func(t *testing.T) {
		// Add 50 blocks (requirement threshold)
		blocks := generateTestBlocks(50)
		display.blocks = blocks

		// Measure 100 renders
		var totalTime time.Duration
		renders := 100

		for i := 0; i < renders; i++ {
			start := time.Now()
			display.View()
			totalTime += time.Since(start)
		}

		avgRenderTime := totalTime / time.Duration(renders)
		frameTime := time.Second / 60 // 16.67ms for 60 FPS

		require.Less(t, avgRenderTime, frameTime,
			"Average render time %v exceeds 60 FPS requirement (%v)",
			avgRenderTime, frameTime)
	})

	t.Run("10ms_render_requirement", func(t *testing.T) {
		// Test with various block counts
		testCases := []int{10, 50, 100}

		for _, blockCount := range testCases {
			t.Run(fmt.Sprintf("%d_blocks", blockCount), func(t *testing.T) {
				display := NewReasoningDisplay(120, 40)
				blocks := generateTestBlocks(blockCount)

				display.blocks = blocks

				// Measure render time
				start := time.Now()
				display.View()
				elapsed := time.Since(start)

				require.Less(t, elapsed, 10*time.Millisecond,
					"Render time %v exceeds 10ms requirement for %d blocks",
					elapsed, blockCount)
			})
		}
	})
}

// Helper functions

func generateTestBlocks(count int) []*core.ThinkingBlock {
	return generateTestBlocksWithSize(count, 200)
}

func generateTestBlocksWithSize(count, contentSize int) []*core.ThinkingBlock {
	blocks := make([]*core.ThinkingBlock, count)
	types := []core.ThinkingType{
		core.ThinkingTypeAnalysis,
		core.ThinkingTypePlanning,
		core.ThinkingTypeDecisionMaking,
		core.ThinkingTypeToolSelection,
		core.ThinkingTypeVerification,
	}

	for i := 0; i < count; i++ {
		blocks[i] = &core.ThinkingBlock{
			ID:        fmt.Sprintf("block_%d", i),
			Type:      types[i%len(types)],
			Content:   generateContent(contentSize),
			Timestamp: time.Now(),
			Metadata: map[string]interface{}{
				"confidence": 0.8 + float64(i%20)/100,
				"quality":    "good",
			},
		}
	}
	return blocks
}

func generateNestedBlocks(count, contentSize int) []*core.ThinkingBlock {
	blocks := make([]*core.ThinkingBlock, 0, count)

	// Create parent blocks
	parentCount := count / 5
	for i := 0; i < parentCount; i++ {
		parentID := fmt.Sprintf("parent_%d", i)
		parent := &core.ThinkingBlock{
			ID:        parentID,
			Type:      core.ThinkingTypePlanning,
			Content:   generateContent(contentSize),
			Timestamp: time.Now(),
			ChildIDs:  make([]string, 0),
		}

		// Add child blocks
		childCount := 4
		for j := 0; j < childCount; j++ {
			child := &core.ThinkingBlock{
				ID:        fmt.Sprintf("child_%d_%d", i, j),
				ParentID:  &parentID,
				Type:      core.ThinkingTypeAnalysis,
				Content:   generateContent(contentSize / 2),
				Timestamp: time.Now(),
			}
			parent.ChildIDs = append(parent.ChildIDs, child.ID)
			blocks = append(blocks, child)
		}

		blocks = append(blocks, parent)
	}

	return blocks
}

func generateContent(size int) string {
	// Generate realistic content of specified size
	words := []string{
		"analyzing", "considering", "evaluating", "processing",
		"determining", "calculating", "reasoning", "thinking",
		"exploring", "investigating", "examining", "reviewing",
	}

	content := ""
	wordCount := 0
	for len(content) < size {
		content += words[wordCount%len(words)] + " "
		wordCount++
	}

	return content[:size]
}
