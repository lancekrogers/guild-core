// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package chat_test

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	// New modular imports
	"github.com/lancekrogers/guild/internal/ui/formatting"
	"github.com/lancekrogers/guild/internal/ui/visual"
	"github.com/lancekrogers/guild/pkg/config"
)

// TestRichContentIntegration tests markdown rendering components
func TestRichContentIntegration(t *testing.T) {
	t.Run("markdown_rendering", func(t *testing.T) {
		// Test markdown renderer with various content
		renderer, err := formatting.NewMarkdownRenderer(80)
		require.NoError(t, err)

		// Test different types of markdown content
		testCases := []struct {
			name     string
			content  string
			contains []string
		}{
			{
				name:     "headers_and_emphasis",
				content:  "# Task Complete\n\nThis is **bold** and *italic* text",
				contains: []string{"Task Complete", "bold", "italic"},
			},
			{
				name:     "code_blocks",
				content:  "```go\nfunc main() {\n    fmt.Println(\"Hello Guild!\")\n}\n```",
				contains: []string{"func main", "fmt.Println", "Hello Guild"},
			},
			{
				name:     "lists",
				content:  "Features:\n- Feature A\n- Feature B\n\nThat's it!",
				contains: []string{"Features", "Feature A", "Feature B"},
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				rendered := renderer.Render(tc.content)
				assert.NotEmpty(t, rendered)

				for _, expected := range tc.contains {
					assert.Contains(t, rendered, expected,
						"Rendered content should contain '%s'", expected)
				}
			})
		}
	})

	t.Run("content_formatter_integration", func(t *testing.T) {
		// Test content formatter with markdown renderer
		renderer, err := formatting.NewMarkdownRenderer(80)
		require.NoError(t, err)

		formatter := formatting.NewContentFormatter(renderer, 80, "/tmp")
		require.NotNil(t, formatter)

		// Test different message types
		testMessages := []struct {
			msgType string
			content string
			agentID string
		}{
			{"agent", "# Analysis Complete\n\nI've reviewed the code.", "analyst"},
			{"system", "Task completed successfully.", ""},
			{"user", "Please **analyze** this code", ""},
		}

		for _, msg := range testMessages {
			formatted := formatter.FormatMessage(msg.msgType, msg.content, nil)
			assert.NotEmpty(t, formatted)
			assert.Contains(t, formatted, msg.content)
		}
	})

	t.Run("performance_large_content", func(t *testing.T) {
		// Test with large markdown content
		renderer, err := formatting.NewMarkdownRenderer(80)
		require.NoError(t, err)

		largeContent := strings.Repeat("# Heading\n\nThis is a paragraph with **bold** text.\n\n", 50)

		start := time.Now()
		rendered := renderer.Render(largeContent)
		duration := time.Since(start)

		assert.NotEmpty(t, rendered)
		// Should render within reasonable time even with large content
		assert.Less(t, duration, 100*time.Millisecond)
	})
}

// TestStatusIntegration tests status tracking and display components
// TODO: Re-enable these tests once AgentStatusTracker and related components are integrated
/*
func TestStatusIntegration_Disabled(t *testing.T) {
	guildConfig := createTestConfig()

	t.Run("agent_status_updates", func(t *testing.T) {
		// Initialize status tracker
		tracker := status.NewStatusTracker(context.Background())
		require.NotNil(t, tracker)

		// Register agent first
		agentInfo := status.AgentInfo{
			ID:     "developer",
			Name:   "Developer Agent",
			Type:   "developer",
			Status: status.StatusIdle,
		}
		err := tracker.RegisterAgent(agentInfo)
		require.NoError(t, err)

		// Simulate agent status update
		err = tracker.UpdateAgentStatus("developer", status.StatusWorking, "Implementing authentication")
		require.NoError(t, err)

		// Verify status is tracked
		currentInfo, err := tracker.GetAgentInfo("developer")
		require.NoError(t, err)
		assert.NotNil(t, currentInfo)
		assert.Equal(t, status.StatusWorking, currentInfo.Status)
	})

	t.Run("status_display_integration", func(t *testing.T) {
		// Test status display rendering
		tracker := status.NewStatusTracker(context.Background())
		display := status.NewStatusDisplay(tracker, 40, 20)
		require.NotNil(t, display)

		// Add multiple agent statuses
		agents := []string{"manager", "developer", "reviewer"}
		states := []status.AgentStatus{status.StatusThinking, status.StatusWorking, status.StatusIdle}

		for i, agentID := range agents {
			agentStatus := &status.AgentStatus{
				ID:    agentID,
				Name:  strings.Title(agentID) + " Agent",
				State: states[i],
			}
			tracker.UpdateAgentStatus(agentID, agentStatus)
		}

		// Render status panel
		statusView := display.RenderStatusPanel()
		assert.NotEmpty(t, statusView)

		// Verify all agents appear
		for _, agent := range agents {
			assert.Contains(t, statusView, agent)
		}
	})

	t.Run("agent_indicators", func(t *testing.T) {
		// Test agent indicators system
		indicators := chat.NewAgentIndicators()
		require.NotNil(t, indicators)

		indicators.StartAnimations()
		defer indicators.StopAnimations()

		// Test different agent states
		agents := []string{"manager", "developer", "reviewer"}

		for _, agentID := range agents {
			indicators.SetWorkingAnimation(agentID, "Processing task")
			indicator := indicators.GetCurrentIndicator(agentID)
			assert.NotEmpty(t, indicator)
		}
	})
}
*/

// TestComponentInteraction tests how different components work together
func TestComponentInteraction(t *testing.T) {
	t.Run("renderer_and_formatter_compatibility", func(t *testing.T) {
		// Create components
		renderer, err := formatting.NewMarkdownRenderer(80)
		require.NoError(t, err)

		formatter := formatting.NewContentFormatter(renderer, 80, "/tmp")
		require.NotNil(t, formatter)

		// Test that they work together without conflicts
		content := "# API Response\n\n```json\n{\"status\": \"success\"}\n```"

		// Format through the formatter
		formatted := formatter.FormatMessage("agent", content, nil)
		assert.NotEmpty(t, formatted)
		assert.Contains(t, formatted, "API Response")
		assert.Contains(t, formatted, "status")
	})

	// TODO: Re-enable this test once AgentStatusTracker and StatusDisplay are integrated
	/*
		t.Run("status_tracking_with_multiple_agents", func(t *testing.T) {
			guildConfig := createTestConfig()
			tracker := status.NewStatusTracker(context.Background())
			display := status.NewStatusDisplay(tracker, 60, 25)

			// Simulate realistic agent workflow
			workflow := []struct {
				agentID string
				state   status.AgentStatus
				task    string
			}{
				{"manager", status.StatusThinking, "Planning project structure"},
				{"developer", status.StatusWorking, "Implementing core features"},
				{"reviewer", status.StatusIdle, ""},
				{"manager", status.StatusWorking, "Coordinating team efforts"},
				{"developer", status.StatusThinking, "Debugging test failures"},
			}

			for _, step := range workflow {
				agentStatus := &status.AgentStatus{
					ID:           step.agentID,
					Name:         strings.Title(step.agentID) + " Agent",
					State:        step.state,
					CurrentTask:  step.task,
					LastActivity: time.Now(),
				}
				tracker.UpdateAgentStatus(step.agentID, agentStatus)
			}

			// Verify final states
			managerStatus := tracker.GetAgentStatus("manager")
			assert.Equal(t, status.StatusWorking, managerStatus.State)
			assert.Equal(t, "Coordinating team efforts", managerStatus.CurrentTask)

			developerStatus := tracker.GetAgentStatus("developer")
			assert.Equal(t, status.StatusThinking, developerStatus.State)

			// Render status display
			statusView := display.RenderStatusPanel()
			assert.NotEmpty(t, statusView)
			assert.Contains(t, statusView, "manager")
			assert.Contains(t, statusView, "developer")
			assert.Contains(t, statusView, "reviewer")
		})
	*/
}

// TestErrorHandling tests graceful degradation of components
func TestErrorHandling(t *testing.T) {
	t.Run("markdown_renderer_edge_cases", func(t *testing.T) {
		renderer, err := formatting.NewMarkdownRenderer(80)
		require.NoError(t, err)

		// Test with problematic content
		edgeCases := []string{
			"",                          // Empty content
			"\x00\x01\x02",              // Invalid UTF-8
			strings.Repeat("x", 100000), // Very long content
			"**unclosed bold",           // Malformed markdown
			"```\nunclosed code block",  // Incomplete code block
		}

		for i, content := range edgeCases {
			t.Run(fmt.Sprintf("edge_case_%d", i), func(t *testing.T) {
				// Should not panic
				assert.NotPanics(t, func() {
					rendered := renderer.Render(content)
					// Should return something, even if just empty or safe fallback
					assert.NotNil(t, rendered)
				})
			})
		}
	})

	// TODO: Re-enable this test once the API is finalized
	/*
		t.Run("status_tracker_error_conditions", func(t *testing.T) {
			// Test with invalid agent IDs
			// Implementation needs updating for new API
		})
	*/

	// TODO: Re-enable this test once AgentStatusTracker is integrated
	/*
		t.Run("concurrent_access_safety", func(t *testing.T) {
			// Test thread safety of components
			guildConfig := createTestConfig()
			tracker := status.NewStatusTracker(context.Background())

			// Test concurrent status updates
			done := make(chan bool, 10)

			for i := 0; i < 10; i++ {
				go func(agentNum int) {
					defer func() { done <- true }()

					agentID := fmt.Sprintf("agent-%d", agentNum)
					for j := 0; j < 50; j++ {
						agentStatus := &status.AgentStatus{
							ID:    agentID,
							State: status.AgentStatus(j % 4),
						}
						tracker.UpdateAgentStatus(agentID, agentStatus)

						// Brief pause to allow interleaving
						time.Sleep(time.Microsecond)
					}
				}(i)
			}

			// Wait for all goroutines to complete
			for i := 0; i < 10; i++ {
				select {
				case <-done:
					// Success
				case <-time.After(5 * time.Second):
					t.Fatal("Timeout waiting for concurrent operations")
				}
			}

			// Should not have crashed and should have some final states
			for i := 0; i < 10; i++ {
				agentID := fmt.Sprintf("agent-%d", i)
				status := tracker.GetAgentStatus(agentID)
				assert.NotNil(t, status, "Agent %s should have a status", agentID)
			}
		})
	*/
}

// TestPerformanceBaseline establishes performance baselines
func TestPerformanceBaseline(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance tests in short mode")
	}

	t.Run("markdown_rendering_performance", func(t *testing.T) {
		renderer, err := formatting.NewMarkdownRenderer(80)
		require.NoError(t, err)

		// Test with various content sizes
		testSizes := []struct {
			name string
			size int
		}{
			{"small", 100},
			{"medium", 1000},
			{"large", 10000},
		}

		for _, tc := range testSizes {
			t.Run(tc.name, func(t *testing.T) {
				content := strings.Repeat("# Header\n\nParagraph with **bold** text.\n\n", tc.size/50)

				start := time.Now()
				rendered := renderer.Render(content)
				duration := time.Since(start)

				assert.NotEmpty(t, rendered)

				// Performance expectations
				maxDuration := time.Duration(tc.size/10) * time.Millisecond
				if maxDuration < 10*time.Millisecond {
					maxDuration = 10 * time.Millisecond
				}

				assert.Less(t, duration, maxDuration,
					"Rendering %d chars should take less than %v, took %v",
					tc.size, maxDuration, duration)
			})
		}
	})

	// TODO: Re-enable this test once AgentStatusTracker is integrated
	/*
		t.Run("status_update_performance", func(t *testing.T) {
			guildConfig := createTestConfig()
			tracker := status.NewStatusTracker(context.Background())

			// Measure status update performance
			agentCount := 100
			updatesPerAgent := 100

			start := time.Now()
			for i := 0; i < agentCount; i++ {
				agentID := fmt.Sprintf("agent-%d", i)
				for j := 0; j < updatesPerAgent; j++ {
					agentStatus := &status.AgentStatus{
						ID:    agentID,
						State: status.AgentStatus(j % 4),
					}
					tracker.UpdateAgentStatus(agentID, agentStatus)
				}
			}
			duration := time.Since(start)

			totalUpdates := agentCount * updatesPerAgent
			updatesPerSecond := float64(totalUpdates) / duration.Seconds()

			t.Logf("Performed %d status updates in %v (%.0f updates/sec)",
				totalUpdates, duration, updatesPerSecond)

			// Should handle at least 10000 updates per second
			assert.Greater(t, updatesPerSecond, 10000.0,
				"Status updates should be fast enough for real-time use")
		})
	*/
}

// Helper function to create test config
func createTestConfig() *config.GuildConfig {
	return &config.GuildConfig{
		Name:    "Test Guild",
		Version: "1.0.0",
		Agents: []config.AgentConfig{
			{
				ID:           "manager",
				Name:         "Manager Agent",
				Type:         "manager",
				Provider:     "mock",
				Model:        "test-model",
				Capabilities: []string{"planning", "coordination"},
			},
			{
				ID:           "developer",
				Name:         "Developer Agent",
				Type:         "worker",
				Provider:     "mock",
				Model:        "test-model",
				Capabilities: []string{"coding", "testing"},
			},
			{
				ID:           "reviewer",
				Name:         "Reviewer Agent",
				Type:         "worker",
				Provider:     "mock",
				Model:        "test-model",
				Capabilities: []string{"review", "quality"},
			},
		},
	}
}

// Benchmark tests for performance validation
func BenchmarkMarkdownRendering(b *testing.B) {
	renderer, _ := formatting.NewMarkdownRenderer(80)
	content := "# Test Message\n\nThis is a **test** with `code` and lists:\n- Item 1\n- Item 2"

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = renderer.Render(content)
	}
}

// TODO: Re-enable this benchmark once AgentStatusTracker is integrated
/*
func BenchmarkStatusUpdates(b *testing.B) {
	guildConfig := createTestConfig()
	tracker := status.NewStatusTracker(context.Background())

	agents := []string{"manager", "developer", "reviewer", "tester"}
	states := []status.AgentStatus{status.StatusIdle, status.StatusThinking, status.StatusWorking, status.StatusError}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		agentID := agents[i%len(agents)]
		state := states[i%len(states)]

		agentStatus := &status.AgentStatus{
			ID:    agentID,
			State: state,
		}
		tracker.UpdateAgentStatus(agentID, agentStatus)
	}
}
*/

// TODO: Re-enable this benchmark once StatusDisplay is integrated
/*
func BenchmarkStatusDisplay(b *testing.B) {
	guildConfig := createTestConfig()
	tracker := status.NewStatusTracker(context.Background())
	display := status.NewStatusDisplay(tracker, 40, 20)

	// Pre-populate with some agents
	for i := 0; i < 5; i++ {
		agentStatus := &status.AgentStatus{
			ID:    fmt.Sprintf("agent-%d", i),
			State: status.StatusWorking,
		}
		tracker.UpdateAgentStatus(fmt.Sprintf("agent-%d", i), status)
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = display.RenderStatusPanel()
	}
}
*/

// TestVisualComponentsIntegration tests the new visual processing components
func TestVisualComponentsIntegration(t *testing.T) {
	t.Run("image_processor", func(t *testing.T) {
		// Test image processor initialization and basic functionality
		imageProcessor := visual.NewImageProcessor()
		require.NotNil(t, imageProcessor)

		// Set ASCII art size
		imageProcessor.SetASCIISize(80, 24)

		// Test processing content with image references
		content := "Here's an image: ![test](./test.png)"
		processed, refs, err := imageProcessor.ProcessContent(content)

		// Should not error on processing
		assert.NoError(t, err)
		assert.NotEmpty(t, processed)
		assert.NotNil(t, refs)
	})

	t.Run("code_renderer", func(t *testing.T) {
		// Test code renderer initialization
		codeRenderer := visual.NewCodeRenderer()
		require.NotNil(t, codeRenderer)

		// Set max width
		codeRenderer.SetMaxWidth(80)

		// Test processing code blocks
		content := "```go\nfunc main() {\n    fmt.Println(\"Hello\")\n}\n```"
		processed := codeRenderer.ProcessCodeBlocks(content)

		assert.NotEmpty(t, processed)
		assert.Contains(t, processed, "func main")
	})

	t.Run("mermaid_processor", func(t *testing.T) {
		// Test mermaid diagram processor
		mermaidProcessor := visual.NewMermaidProcessor()
		require.NotNil(t, mermaidProcessor)

		// Set ASCII size for diagrams
		mermaidProcessor.SetASCIISize(80, 30)

		// Test processing mermaid content
		content := "```mermaid\ngraph TD\n    A[Start] --> B[End]\n```"
		processed, diagrams, err := mermaidProcessor.ProcessContent(content)

		// Should handle mermaid content gracefully
		assert.NoError(t, err)
		assert.NotEmpty(t, processed)
		assert.NotNil(t, diagrams)
	})
}

// TestProgressIndicators tests the progress indicator components
// TODO: Re-enable this test once the progress API is updated
/*
func TestProgressIndicators_Disabled(t *testing.T) {
	t.Run("progress_indicators", func(t *testing.T) {
		// Test progress indicator initialization
		indicators := progress.NewIndicator()
		require.NotNil(t, indicators)

		// Test starting and stopping indicators
		indicators.Start("test-task", "Processing...")

		// Update progress
		indicators.Update("test-task", 50, "Halfway done")

		// Complete the task
		indicators.Complete("test-task", "Task completed")

		// Stop indicators
		indicators.Stop("test-task")

		// Should handle all operations without panic
		assert.True(t, true, "Progress indicators handled all operations")
	})
}
*/
