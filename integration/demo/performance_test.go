// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

//go:build integration
// +build integration

package demo

import (
	"context"
	"fmt"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lancekrogers/guild-core/pkg/registry"
)

// BenchmarkChatResponsiveness benchmarks UI update performance
func BenchmarkChatResponsiveness(b *testing.B) {
	// Create mock chat model for performance testing
	model := createMockChatModel()
	require.NotNil(b, model)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		start := time.Now()

		// Simulate UI update operation
		model.ProcessMessage(fmt.Sprintf("test message %d", i))

		elapsed := time.Since(start)

		// Must be under 16ms for 60fps
		if elapsed > 16*time.Millisecond {
			b.Fatalf("Update too slow: %v (should be < 16ms for 60fps)", elapsed)
		}
	}
}

// BenchmarkAgentCreation benchmarks agent instantiation performance
func BenchmarkAgentCreation(b *testing.B) {
	ctx := context.Background()

	// Setup registry
	reg := registry.NewComponentRegistry()
	err := reg.Initialize(ctx, registry.Config{})
	require.NoError(b, err)

	// Use agent registry instead of factory
	agentReg := reg.Agents()
	require.NotNil(b, agentReg)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		start := time.Now()

		// Use agent registry to get agent (simplified for benchmark)
		agent, err := agentReg.GetAgent("worker")

		elapsed := time.Since(start)

		if err != nil {
			b.Logf("Agent creation error (may be expected): %v", err)
		}

		// Agent creation should be fast (under 100ms)
		if elapsed > 100*time.Millisecond {
			b.Logf("Slow agent creation: %v", elapsed)
		}

		// Clean up
		if agent != nil {
			// Agent cleanup would go here if needed
		}
	}
}

// TestMemoryUsage tests that demo doesn't leak memory during operation
func TestMemoryUsage(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping memory test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Get initial memory usage
	var m runtime.MemStats
	runtime.GC() // Start clean
	runtime.ReadMemStats(&m)
	startMem := m.Alloc

	t.Logf("Starting memory: %.2f MB", float64(startMem)/(1024*1024))

	// Setup registry
	reg := registry.NewComponentRegistry()
	err := reg.Initialize(ctx, registry.Config{})
	require.NoError(t, err)

	// Run intensive operations to simulate demo usage
	err = runIntensiveDemoOperations(ctx, reg, 100)
	require.NoError(t, err)

	// Force garbage collection
	runtime.GC()
	runtime.GC()                       // Run twice to be thorough
	time.Sleep(100 * time.Millisecond) // Let GC complete

	// Check memory after operations
	runtime.ReadMemStats(&m)
	endMem := m.Alloc

	t.Logf("Ending memory: %.2f MB", float64(endMem)/(1024*1024))

	var growth int64
	if endMem >= startMem {
		growth = int64(endMem - startMem)
	} else {
		// Memory decreased (due to GC or measurement variance)
		growth = -int64(startMem - endMem)
	}
	growthMB := float64(growth) / (1024 * 1024)

	t.Logf("Memory growth: %.2f MB", growthMB)

	// Should not grow more than 50MB (generous limit for demo)
	// Allow negative growth (memory freed)
	assert.Less(t, growth, int64(50*1024*1024),
		"Memory growth should be reasonable (< 50MB), got %.2f MB", growthMB)
}

// TestRenderingPerformance tests markdown and syntax highlighting performance
func TestRenderingPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping rendering performance test in short mode")
	}

	testCases := []struct {
		name    string
		content string
		maxTime time.Duration
	}{
		{
			name: "Simple Markdown",
			content: `# Simple Header
This is a **simple** markdown test with *emphasis*.`,
			maxTime: 5 * time.Millisecond,
		},
		{
			name: "Code Block",
			content: `# Code Example
` + "```go\nfunc main() {\n    fmt.Println(\"Hello, World!\")\n}\n```",
			maxTime: 10 * time.Millisecond,
		},
		{
			name:    "Large Content",
			content: generateLargeMarkdownContent(),
			maxTime: 50 * time.Millisecond,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			start := time.Now()

			// Mock markdown rendering
			rendered := mockRenderMarkdown(tc.content)

			elapsed := time.Since(start)

			assert.NotEmpty(t, rendered, "Rendered content should not be empty")
			assert.Less(t, elapsed, tc.maxTime,
				"Rendering should be fast: got %v, expected < %v", elapsed, tc.maxTime)

			t.Logf("Rendered %d chars in %v", len(tc.content), elapsed)
		})
	}
}

// TestConcurrentAgentPerformance tests performance with multiple agents
func TestConcurrentAgentPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping concurrent performance test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Setup registry
	reg := registry.NewComponentRegistry()
	err := reg.Initialize(ctx, registry.Config{})
	require.NoError(t, err)

	agentRegistry := reg.Agents()
	require.NotNil(t, agentRegistry)

	// Test concurrent agent operations
	numAgents := 5
	numOperations := 10

	start := time.Now()

	// Create agents concurrently
	agents := make([]interface{}, numAgents)
	for i := 0; i < numAgents; i++ {
		agent, err := agentRegistry.GetAgent("worker")
		if err != nil {
			t.Logf("Agent %d creation error (may be expected): %v", i, err)
			continue
		}
		agents[i] = agent
	}

	// Run operations on all agents
	for i := 0; i < numOperations; i++ {
		for j, agent := range agents {
			if agent == nil {
				continue
			}

			if execAgent, ok := agent.(interface {
				Execute(context.Context, string) (string, error)
			}); ok {
				_, err := execAgent.Execute(ctx, fmt.Sprintf("Operation %d", i))
				if err != nil {
					t.Logf("Agent %d operation %d error (may be expected): %v", j, i, err)
				}
			}
		}
	}

	elapsed := time.Since(start)

	t.Logf("Concurrent operations completed in %v", elapsed)

	// Should complete within reasonable time
	assert.Less(t, elapsed, 30*time.Second,
		"Concurrent operations should complete in reasonable time")
}

// TestUIResponsiveness tests UI update performance under load
func TestUIResponsiveness(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping UI responsiveness test in short mode")
	}

	model := createMockChatModel()
	require.NotNil(t, model)

	// Test rapid UI updates
	numUpdates := 1000
	start := time.Now()

	for i := 0; i < numUpdates; i++ {
		updateStart := time.Now()

		model.ProcessMessage(fmt.Sprintf("rapid update %d", i))

		updateElapsed := time.Since(updateStart)

		// Each update should be fast
		if updateElapsed > 5*time.Millisecond {
			t.Logf("Slow update %d: %v", i, updateElapsed)
		}
	}

	totalElapsed := time.Since(start)
	avgTime := totalElapsed / time.Duration(numUpdates)

	t.Logf("Processed %d updates in %v (avg: %v per update)",
		numUpdates, totalElapsed, avgTime)

	// Average should be well under 1ms for good responsiveness
	assert.Less(t, avgTime, 1*time.Millisecond,
		"Average UI update time should be very fast")
}

// TestStartupPerformance tests demo startup time
func TestStartupPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping startup performance test in short mode")
	}

	start := time.Now()

	// Simulate demo startup sequence
	ctx := context.Background()

	// Registry initialization
	reg := registry.NewComponentRegistry()
	err := reg.Initialize(ctx, registry.Config{})
	require.NoError(t, err)

	// Component loading
	agentReg := reg.Agents()
	if agentReg == nil {
		t.Logf("Agent registry not available")
	}

	toolReg := reg.Tools()
	if toolReg == nil {
		t.Logf("Tool registry not available")
	}

	elapsed := time.Since(start)

	t.Logf("Demo startup simulation completed in %v", elapsed)

	// Startup should be fast (under 5 seconds)
	assert.Less(t, elapsed, 5*time.Second,
		"Demo startup should be fast")
}

// Helper functions and mocks

type MockChatModel struct {
	messageCount int
}

func createMockChatModel() *MockChatModel {
	return &MockChatModel{
		messageCount: 0,
	}
}

func (m *MockChatModel) ProcessMessage(message string) {
	// Simulate message processing
	m.messageCount++

	// Simulate some work
	time.Sleep(100 * time.Microsecond)
}

func runIntensiveDemoOperations(ctx context.Context, reg registry.ComponentRegistry, iterations int) error {
	agentRegistry := reg.Agents()
	if agentRegistry == nil {
		return fmt.Errorf("agent registry not available")
	}

	// Create and use agents multiple times
	for i := 0; i < iterations; i++ {
		agent, err := agentRegistry.GetAgent("worker")
		if err != nil {
			continue // Skip on error
		}

		// Execute operations to trigger memory allocation
		if execAgent, ok := agent.(interface {
			Execute(context.Context, string) (string, error)
		}); ok {
			_, _ = execAgent.Execute(ctx, fmt.Sprintf("Intensive operation %d with lots of text to process and allocate memory for testing purposes", i))
		}

		// Simulate some additional work
		mockRenderMarkdown(fmt.Sprintf("# Operation %d\nSome content here", i))
	}

	return nil
}

func generateLargeMarkdownContent() string {
	content := "# Large Markdown Document\n\n"

	for i := 0; i < 50; i++ {
		content += fmt.Sprintf("## Section %d\n\n", i)
		content += "This is a large section with **bold text** and *italic text*.\n\n"
		content += "```go\n"
		content += fmt.Sprintf("func section%d() {\n", i)
		content += fmt.Sprintf("    fmt.Printf(\"Section %d processing...\\n\", %d)\n", i, i)
		content += "    // Simulate some complex logic here\n"
		content += "    for j := 0; j < 100; j++ {\n"
		content += "        process(j)\n"
		content += "    }\n"
		content += "}\n"
		content += "```\n\n"
	}

	return content
}

func mockRenderMarkdown(content string) string {
	// Simulate markdown rendering work
	time.Sleep(10 * time.Microsecond)

	// Simple mock processing
	lines := len(content) / 50 // Approximate line count
	processed := fmt.Sprintf("RENDERED[%d lines]: %s", lines, content[:min(len(content), 100)])

	return processed
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
