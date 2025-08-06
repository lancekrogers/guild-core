// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

//go:build integration
// +build integration

package tui_cli

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestThemeSwitching_CriticalPerformanceSLA validates 60 FPS requirement
// This is a CRITICAL test that validates the most important user experience SLA:
// Theme switching must complete within 16ms to maintain 60 FPS user experience.
// This test loads substantial content to stress test theme application.
func TestThemeSwitching_CriticalPerformanceSLA(t *testing.T) {
	framework := NewTUITestFramework(t)
	defer framework.Cleanup()

	app := framework.StartApp(TUIConfig{
		InitialTheme: "dark",
		ContentSize:  "large", // Stress test with substantial content
		WindowSize:   TUISize{Width: 200, Height: 60},
	})
	defer app.Quit()

	// Load substantial content to stress test theme switching
	framework.LoadComplexContent(app, ComplexContent{
		CodeBlocks:     10,
		MarkdownTables: 5,
		ColoredText:    100,
		TotalLines:     500,
	})

	t.Log("=== CRITICAL SLA TEST: Theme Switching Performance ===")
	t.Log("SLA Requirement: ≤16ms per theme switch (60 FPS)")

	themes := []string{"dark", "light", "high-contrast", "colorblind-friendly", "minimal"}
	switchTimes := make([]time.Duration, 0, len(themes))
	violations := 0

	for i, theme := range themes {
		t.Run(fmt.Sprintf("switch_to_%s", theme), func(t *testing.T) {
			t.Logf("--- Theme Switch %d: Switching to %s ---", i+1, theme)

			// CRITICAL: Theme switch must complete within 16ms for 60 FPS
			start := time.Now()
			app.SwitchTheme(theme)
			switchTime := time.Since(start)

			switchTimes = append(switchTimes, switchTime)

			// HARD SLA REQUIREMENT - this is critical for user experience
			if switchTime > 16*time.Millisecond {
				violations++
				t.Errorf("❌ CRITICAL SLA VIOLATION: Theme switch to %s took %v, exceeds 16ms (60 FPS) requirement",
					theme, switchTime)
			} else {
				t.Logf("✅ Theme switch to %s: %v (within 16ms SLA)", theme, switchTime)
			}

			// Validate visual consistency after switch
			screenshot := app.CaptureScreenshot()
			expectedPath := fmt.Sprintf("testdata/visual-regression/theme_%s_complex.png", theme)

			if framework.ShouldUpdateGoldenFiles() {
				framework.SaveGoldenFile(expectedPath, screenshot)
				t.Logf("📸 Saved golden file: %s", expectedPath)
			} else {
				consistency := framework.CompareWithGolden(screenshot, expectedPath)
				assert.GreaterOrEqual(t, consistency, 0.95,
					"Visual consistency below 95%% for theme %s: %.2f%%", theme, consistency*100)
				t.Logf("✓ Visual consistency: %.1f%% for theme %s", consistency*100, theme)
			}

			// Validate theme application completeness
			themeMetrics := app.GetThemeMetrics()
			assert.Equal(t, theme, themeMetrics.ActiveTheme)
			assert.Equal(t, 100.0, themeMetrics.ApplicationCompleteness,
				"Theme must be 100%% applied to all UI elements")
			assert.Zero(t, themeMetrics.InconsistentElements,
				"No elements should have inconsistent theming")

			t.Logf("✓ Theme %s fully applied (%.1f%% completeness, %d inconsistencies)",
				theme, themeMetrics.ApplicationCompleteness, themeMetrics.InconsistentElements)
		})
	}

	// Analyze overall theme switching performance
	averageTime := calculateAverage(switchTimes)
	maxTime := calculateMax(switchTimes)
	p95Time := calculatePercentile(switchTimes, 0.95)

	t.Logf("📊 Theme Switching Performance Analysis:")
	t.Logf("   - Themes tested: %d", len(themes))
	t.Logf("   - Average time: %v", averageTime)
	t.Logf("   - Maximum time: %v", maxTime)
	t.Logf("   - P95 time: %v", p95Time)
	t.Logf("   - SLA violations: %d out of %d", violations, len(themes))
	t.Logf("   - Success rate: %.1f%%", float64(len(themes)-violations)/float64(len(themes))*100)

	// CRITICAL: All switches must meet SLA for 60 FPS user experience
	assert.Equal(t, 0, violations,
		"CRITICAL FAILURE: %d theme switches violated 16ms SLA out of %d total", violations, len(themes))

	assert.LessOrEqual(t, maxTime, 16*time.Millisecond,
		"Maximum theme switch time exceeded SLA: %v > 16ms", maxTime)

	assert.LessOrEqual(t, p95Time, 12*time.Millisecond,
		"P95 theme switch time should be well under SLA: %v", p95Time)

	assert.LessOrEqual(t, averageTime, 10*time.Millisecond,
		"Average theme switch time should be well under SLA: %v", averageTime)

	// Performance regression detection
	if framework.HasPerformanceBaseline() {
		baseline := framework.GetPerformanceBaseline("theme_switching")
		regression := float64(averageTime-baseline.Average) / float64(baseline.Average)

		assert.LessOrEqual(t, regression, 0.1,
			"Theme switching performance regression detected: %.1f%% slower than baseline", regression*100)

		t.Logf("✓ Performance regression check passed (%.1f%% vs baseline)", regression*100)
	}

	if violations == 0 {
		t.Log("🎉 ALL THEME SWITCHES MEET 60 FPS SLA REQUIREMENT!")
	}
}

// TestThemeSwitching_RapidSequence validates performance under rapid theme changes
func TestThemeSwitching_RapidSequence(t *testing.T) {
	framework := NewTUITestFramework(t)
	defer framework.Cleanup()

	app := framework.StartApp(TUIConfig{
		InitialTheme: "dark",
		ContentSize:  "medium",
		WindowSize:   TUISize{Width: 150, Height: 40},
	})
	defer app.Quit()

	// Load moderate content for rapid switching test
	framework.LoadComplexContent(app, ComplexContent{
		CodeBlocks:     5,
		MarkdownTables: 3,
		ColoredText:    50,
		TotalLines:     250,
	})

	themes := []string{"dark", "light", "high-contrast", "colorblind-friendly", "minimal"}
	sequences := []struct {
		name        string
		repetitions int
		interval    time.Duration
	}{
		{
			name:        "Rapid consecutive switches",
			repetitions: 20,
			interval:    0, // No delay between switches
		},
		{
			name:        "Fast user switching",
			repetitions: 15,
			interval:    50 * time.Millisecond,
		},
		{
			name:        "Stress test sequence",
			repetitions: 50,
			interval:    10 * time.Millisecond,
		},
	}

	for _, sequence := range sequences {
		t.Run(sequence.name, func(t *testing.T) {
			var switchTimes []time.Duration
			violations := 0
			sequenceStart := time.Now()

			for i := 0; i < sequence.repetitions; i++ {
				theme := themes[i%len(themes)]

				if sequence.interval > 0 {
					time.Sleep(sequence.interval)
				}

				start := time.Now()
				app.SwitchTheme(theme)
				switchTime := time.Since(start)

				switchTimes = append(switchTimes, switchTime)

				// Each switch must still meet SLA even in rapid sequence
				if switchTime > 16*time.Millisecond {
					violations++
				}
			}

			sequenceDuration := time.Since(sequenceStart)
			averageTime := calculateAverage(switchTimes)
			maxTime := calculateMax(switchTimes)

			// Validate sequence performance
			assert.LessOrEqual(t, violations, sequence.repetitions/10,
				"Too many SLA violations in sequence: %d out of %d", violations, sequence.repetitions)

			assert.LessOrEqual(t, maxTime, 20*time.Millisecond, // Slightly relaxed for rapid sequence
				"Maximum time too high in rapid sequence: %v", maxTime)

			assert.LessOrEqual(t, averageTime, 12*time.Millisecond,
				"Average time too high in rapid sequence: %v", averageTime)

			// UI should remain responsive during rapid switching
			assert.True(t, app.IsResponsive(), "App should remain responsive during rapid theme switching")

			t.Logf("Rapid sequence results for %s:", sequence.name)
			t.Logf("  - Switches: %d", sequence.repetitions)
			t.Logf("  - Total time: %v", sequenceDuration)
			t.Logf("  - Average switch time: %v", averageTime)
			t.Logf("  - Maximum switch time: %v", maxTime)
			t.Logf("  - SLA violations: %d", violations)
			t.Logf("  - Success rate: %.1f%%", float64(sequence.repetitions-violations)/float64(sequence.repetitions)*100)
		})
	}
}

// TestThemeSwitching_ContentVariations validates performance across different content types
func TestThemeSwitching_ContentVariations(t *testing.T) {
	framework := NewTUITestFramework(t)
	defer framework.Cleanup()

	contentVariations := []struct {
		name     string
		content  ComplexContent
		slaLimit time.Duration
	}{
		{
			name: "Minimal content",
			content: ComplexContent{
				CodeBlocks:     1,
				MarkdownTables: 0,
				ColoredText:    10,
				TotalLines:     50,
			},
			slaLimit: 8 * time.Millisecond,
		},
		{
			name: "Medium content",
			content: ComplexContent{
				CodeBlocks:     5,
				MarkdownTables: 2,
				ColoredText:    50,
				TotalLines:     200,
			},
			slaLimit: 12 * time.Millisecond,
		},
		{
			name: "Heavy content",
			content: ComplexContent{
				CodeBlocks:     15,
				MarkdownTables: 8,
				ColoredText:    200,
				TotalLines:     1000,
			},
			slaLimit: 16 * time.Millisecond,
		},
		{
			name: "Extreme content",
			content: ComplexContent{
				CodeBlocks:     25,
				MarkdownTables: 15,
				ColoredText:    500,
				TotalLines:     2000,
			},
			slaLimit: 20 * time.Millisecond, // Slightly relaxed for extreme content
		},
	}

	themes := []string{"dark", "light", "high-contrast"}

	for _, variation := range contentVariations {
		t.Run(variation.name, func(t *testing.T) {
			app := framework.StartApp(TUIConfig{
				InitialTheme: "dark",
				ContentSize:  "variable",
				WindowSize:   TUISize{Width: 180, Height: 50},
			})
			defer app.Quit()

			// Load specific content variation
			framework.LoadComplexContent(app, variation.content)

			var switchTimes []time.Duration
			violations := 0

			for _, theme := range themes {
				start := time.Now()
				app.SwitchTheme(theme)
				switchTime := time.Since(start)

				switchTimes = append(switchTimes, switchTime)

				if switchTime > variation.slaLimit {
					violations++
				}
			}

			averageTime := calculateAverage(switchTimes)
			maxTime := calculateMax(switchTimes)

			// Validate content-specific performance
			assert.Equal(t, 0, violations,
				"SLA violations for %s: %d out of %d", variation.name, violations, len(themes))

			assert.LessOrEqual(t, maxTime, variation.slaLimit,
				"Maximum time exceeded for %s: %v > %v", variation.name, maxTime, variation.slaLimit)

			assert.LessOrEqual(t, averageTime, variation.slaLimit*8/10,
				"Average time too high for %s: %v", variation.name, averageTime)

			t.Logf("%s performance:", variation.name)
			t.Logf("  - Content: %d code blocks, %d tables, %d colored elements, %d lines",
				variation.content.CodeBlocks, variation.content.MarkdownTables,
				variation.content.ColoredText, variation.content.TotalLines)
			t.Logf("  - Average theme switch: %v", averageTime)
			t.Logf("  - Maximum theme switch: %v", maxTime)
			t.Logf("  - SLA limit: %v", variation.slaLimit)
			t.Logf("  - Violations: %d", violations)
		})
	}
}

// TestThemeSwitching_MemoryImpact validates memory efficiency during theme changes
func TestThemeSwitching_MemoryImpact(t *testing.T) {
	framework := NewTUITestFramework(t)
	defer framework.Cleanup()

	app := framework.StartApp(TUIConfig{
		InitialTheme: "dark",
		ContentSize:  "large",
		WindowSize:   TUISize{Width: 200, Height: 60},
	})
	defer app.Quit()

	// Load substantial content for memory testing
	framework.LoadComplexContent(app, ComplexContent{
		CodeBlocks:     20,
		MarkdownTables: 10,
		ColoredText:    300,
		TotalLines:     1500,
	})

	// Measure initial memory usage
	initialMetrics := app.GetPerformanceMetrics()
	initialMemory := initialMetrics.MemoryUsageMB

	themes := []string{"dark", "light", "high-contrast", "colorblind-friendly", "minimal"}
	memoryMeasurements := make([]int, 0, len(themes)*2) // Before and after each switch

	// Perform theme switches and monitor memory
	for i, theme := range themes {
		// Measure memory before switch
		beforeMetrics := app.GetPerformanceMetrics()
		memoryMeasurements = append(memoryMeasurements, beforeMetrics.MemoryUsageMB)

		// Perform theme switch
		start := time.Now()
		app.SwitchTheme(theme)
		switchTime := time.Since(start)

		// Measure memory after switch
		afterMetrics := app.GetPerformanceMetrics()
		memoryMeasurements = append(memoryMeasurements, afterMetrics.MemoryUsageMB)

		// Validate switch still meets performance SLA
		assert.LessOrEqual(t, switchTime, 16*time.Millisecond,
			"Theme switch %d performance degraded: %v", i+1, switchTime)

		// Memory should not increase significantly
		memoryIncrease := afterMetrics.MemoryUsageMB - beforeMetrics.MemoryUsageMB
		assert.LessOrEqual(t, memoryIncrease, 5,
			"Memory increased too much during theme switch %d: %d MB", i+1, memoryIncrease)

		t.Logf("Theme switch to %s: %v, memory: %d→%d MB (+%d)",
			theme, switchTime, beforeMetrics.MemoryUsageMB, afterMetrics.MemoryUsageMB, memoryIncrease)
	}

	// Analyze overall memory behavior
	finalMetrics := app.GetPerformanceMetrics()
	totalMemoryIncrease := finalMetrics.MemoryUsageMB - initialMemory

	assert.LessOrEqual(t, totalMemoryIncrease, 20,
		"Total memory increase too high: %d MB", totalMemoryIncrease)

	assert.LessOrEqual(t, finalMetrics.MemoryUsageMB, 150,
		"Final memory usage too high: %d MB", finalMetrics.MemoryUsageMB)

	// Calculate memory statistics
	maxMemory := initialMemory
	minMemory := initialMemory
	for _, memory := range memoryMeasurements {
		if memory > maxMemory {
			maxMemory = memory
		}
		if memory < minMemory {
			minMemory = memory
		}
	}

	t.Logf("📊 Memory Impact Analysis:")
	t.Logf("   - Initial memory: %d MB", initialMemory)
	t.Logf("   - Final memory: %d MB", finalMetrics.MemoryUsageMB)
	t.Logf("   - Total increase: %d MB", totalMemoryIncrease)
	t.Logf("   - Peak memory: %d MB", maxMemory)
	t.Logf("   - Memory range: %d-%d MB", minMemory, maxMemory)

	// Memory usage should be stable and efficient
	memoryRange := maxMemory - minMemory
	assert.LessOrEqual(t, memoryRange, 30,
		"Memory usage range too wide: %d MB", memoryRange)
}

// TestThemeSwitching_ConcurrentOperations validates theme switching during other operations
func TestThemeSwitching_ConcurrentOperations(t *testing.T) {
	framework := NewTUITestFramework(t)
	defer framework.Cleanup()

	app := framework.StartApp(TUIConfig{
		InitialTheme: "dark",
		ContentSize:  "medium",
		WindowSize:   TUISize{Width: 160, Height: 45},
	})
	defer app.Quit()

	// Load content for concurrent operations test
	framework.LoadComplexContent(app, ComplexContent{
		CodeBlocks:     8,
		MarkdownTables: 4,
		ColoredText:    100,
		TotalLines:     400,
	})

	// Simulate concurrent operations while switching themes
	done := make(chan bool, 3)
	results := make(chan time.Duration, 10)

	// Start background operations
	go func() {
		defer func() { done <- true }()
		// Simulate continuous UI updates
		for i := 0; i < 50; i++ {
			app.SendMessage(fmt.Sprintf("Background message %d", i))
			time.Sleep(20 * time.Millisecond)
		}
	}()

	go func() {
		defer func() { done <- true }()
		// Simulate user interactions
		for i := 0; i < 20; i++ {
			response, _ := app.WaitForResponse(1 * time.Second)
			_ = response
			time.Sleep(100 * time.Millisecond)
		}
	}()

	go func() {
		defer func() { done <- true }()
		// Perform theme switches during concurrent operations
		themes := []string{"light", "dark", "high-contrast", "minimal", "colorblind-friendly"}
		for i := 0; i < 10; i++ {
			theme := themes[i%len(themes)]
			start := time.Now()
			app.SwitchTheme(theme)
			switchTime := time.Since(start)
			results <- switchTime
			time.Sleep(200 * time.Millisecond)
		}
	}()

	// Wait for all operations to complete
	for i := 0; i < 3; i++ {
		select {
		case <-done:
			// Operation completed
		case <-time.After(30 * time.Second):
			t.Fatal("Timeout waiting for concurrent operations")
		}
	}

	// Analyze concurrent theme switching performance
	close(results)
	var switchTimes []time.Duration
	violations := 0

	for switchTime := range results {
		switchTimes = append(switchTimes, switchTime)
		if switchTime > 25*time.Millisecond { // Slightly relaxed SLA for concurrent ops
			violations++
		}
	}

	require.Greater(t, len(switchTimes), 0, "Should have recorded theme switch times")

	averageTime := calculateAverage(switchTimes)
	maxTime := calculateMax(switchTimes)

	// Validate concurrent performance
	assert.LessOrEqual(t, violations, len(switchTimes)/5,
		"Too many violations during concurrent operations: %d out of %d", violations, len(switchTimes))

	assert.LessOrEqual(t, maxTime, 30*time.Millisecond,
		"Maximum concurrent switch time too high: %v", maxTime)

	assert.LessOrEqual(t, averageTime, 20*time.Millisecond,
		"Average concurrent switch time too high: %v", averageTime)

	// App should remain responsive
	assert.True(t, app.IsResponsive(), "App should remain responsive during concurrent operations")

	t.Logf("Concurrent operations results:")
	t.Logf("  - Theme switches: %d", len(switchTimes))
	t.Logf("  - Average switch time: %v", averageTime)
	t.Logf("  - Maximum switch time: %v", maxTime)
	t.Logf("  - SLA violations: %d", violations)
	t.Logf("  - Success rate: %.1f%%", float64(len(switchTimes)-violations)/float64(len(switchTimes))*100)
}
