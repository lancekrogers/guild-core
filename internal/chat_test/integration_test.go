package chat_test

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/guild-ventures/guild-core/internal/chat"
	"github.com/guild-ventures/guild-core/pkg/config"
)

// TestRichContentIntegration tests markdown rendering components
func TestRichContentIntegration(t *testing.T) {
	t.Run("markdown_rendering", func(t *testing.T) {
		// Test markdown renderer with various content
		renderer, err := chat.NewMarkdownRenderer(80)
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
		renderer, err := chat.NewMarkdownRenderer(80)
		require.NoError(t, err)
		
		formatter := chat.NewContentFormatter(renderer, 80)
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
		renderer, err := chat.NewMarkdownRenderer(80)
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
func TestStatusIntegration(t *testing.T) {
	guildConfig := createTestConfig()

	t.Run("agent_status_updates", func(t *testing.T) {
		// Initialize status tracker
		tracker := chat.NewAgentStatusTracker(guildConfig)
		require.NotNil(t, tracker)

		// Simulate agent status update
		status := &chat.AgentStatus{
			ID:           "developer",
			Name:         "Developer Agent",
			State:        chat.AgentWorking,
			CurrentTask:  "Implementing authentication",
			LastActivity: time.Now(),
		}

		tracker.UpdateAgentStatus("developer", status)

		// Verify status is tracked
		currentStatus := tracker.GetAgentStatus("developer")
		assert.NotNil(t, currentStatus)
		assert.Equal(t, chat.AgentWorking, currentStatus.State)
		assert.Equal(t, "Implementing authentication", currentStatus.CurrentTask)
	})

	t.Run("status_display_integration", func(t *testing.T) {
		// Test status display rendering
		tracker := chat.NewAgentStatusTracker(guildConfig)
		display := chat.NewStatusDisplay(tracker, 40, 20)
		require.NotNil(t, display)

		// Add multiple agent statuses
		agents := []string{"manager", "developer", "reviewer"}
		states := []chat.AgentState{chat.AgentThinking, chat.AgentWorking, chat.AgentIdle}

		for i, agentID := range agents {
			status := &chat.AgentStatus{
				ID:    agentID,
				Name:  strings.Title(agentID) + " Agent",
				State: states[i],
			}
			tracker.UpdateAgentStatus(agentID, status)
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

// TestComponentInteraction tests how different components work together
func TestComponentInteraction(t *testing.T) {
	t.Run("renderer_and_formatter_compatibility", func(t *testing.T) {
		// Create components
		renderer, err := chat.NewMarkdownRenderer(80)
		require.NoError(t, err)
		
		formatter := chat.NewContentFormatter(renderer, 80)
		require.NotNil(t, formatter)

		// Test that they work together without conflicts
		content := "# API Response\n\n```json\n{\"status\": \"success\"}\n```"
		
		// Format through the formatter
		formatted := formatter.FormatMessage("agent", content, nil)
		assert.NotEmpty(t, formatted)
		assert.Contains(t, formatted, "API Response")
		assert.Contains(t, formatted, "status")
	})

	t.Run("status_tracking_with_multiple_agents", func(t *testing.T) {
		guildConfig := createTestConfig()
		tracker := chat.NewAgentStatusTracker(guildConfig)
		display := chat.NewStatusDisplay(tracker, 60, 25)

		// Simulate realistic agent workflow
		workflow := []struct {
			agentID string
			state   chat.AgentState
			task    string
		}{
			{"manager", chat.AgentThinking, "Planning project structure"},
			{"developer", chat.AgentWorking, "Implementing core features"},
			{"reviewer", chat.AgentIdle, ""},
			{"manager", chat.AgentWorking, "Coordinating team efforts"},
			{"developer", chat.AgentThinking, "Debugging test failures"},
		}

		for _, step := range workflow {
			status := &chat.AgentStatus{
				ID:           step.agentID,
				Name:         strings.Title(step.agentID) + " Agent",
				State:        step.state,
				CurrentTask:  step.task,
				LastActivity: time.Now(),
			}
			tracker.UpdateAgentStatus(step.agentID, status)
		}

		// Verify final states
		managerStatus := tracker.GetAgentStatus("manager")
		assert.Equal(t, chat.AgentWorking, managerStatus.State)
		assert.Equal(t, "Coordinating team efforts", managerStatus.CurrentTask)

		developerStatus := tracker.GetAgentStatus("developer")
		assert.Equal(t, chat.AgentThinking, developerStatus.State)

		// Render status display
		statusView := display.RenderStatusPanel()
		assert.NotEmpty(t, statusView)
		assert.Contains(t, statusView, "manager")
		assert.Contains(t, statusView, "developer")
		assert.Contains(t, statusView, "reviewer")
	})
}

// TestErrorHandling tests graceful degradation of components
func TestErrorHandling(t *testing.T) {
	t.Run("markdown_renderer_edge_cases", func(t *testing.T) {
		renderer, err := chat.NewMarkdownRenderer(80)
		require.NoError(t, err)

		// Test with problematic content
		edgeCases := []string{
			"", // Empty content
			"\x00\x01\x02", // Invalid UTF-8
			strings.Repeat("x", 100000), // Very long content
			"**unclosed bold",             // Malformed markdown
			"```\nunclosed code block",    // Incomplete code block
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

	t.Run("status_tracker_error_conditions", func(t *testing.T) {
		guildConfig := createTestConfig()
		tracker := chat.NewAgentStatusTracker(guildConfig)

		// Test with invalid agent IDs
		invalidCases := []string{
			"", // Empty ID
			"non-existent-agent", // Agent not in config
			"agent-with-special-chars@#$%", // Special characters
		}

		for _, agentID := range invalidCases {
			// Should not panic with invalid agent IDs
			assert.NotPanics(t, func() {
				status := &chat.AgentStatus{
					ID:    agentID,
					State: chat.AgentIdle,
				}
				tracker.UpdateAgentStatus(agentID, status)
				
				// Should handle gracefully
				retrievedStatus := tracker.GetAgentStatus(agentID)
				// May be nil for invalid agents, but shouldn't crash
				_ = retrievedStatus
			})
		}
	})

	t.Run("concurrent_access_safety", func(t *testing.T) {
		// Test thread safety of components
		guildConfig := createTestConfig()
		tracker := chat.NewAgentStatusTracker(guildConfig)
		
		// Test concurrent status updates
		done := make(chan bool, 10)
		
		for i := 0; i < 10; i++ {
			go func(agentNum int) {
				defer func() { done <- true }()
				
				agentID := fmt.Sprintf("agent-%d", agentNum)
				for j := 0; j < 50; j++ {
					status := &chat.AgentStatus{
						ID:    agentID,
						State: chat.AgentState(j % 4),
					}
					tracker.UpdateAgentStatus(agentID, status)
					
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
}

// TestPerformanceBaseline establishes performance baselines
func TestPerformanceBaseline(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance tests in short mode")
	}

	t.Run("markdown_rendering_performance", func(t *testing.T) {
		renderer, err := chat.NewMarkdownRenderer(80)
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

	t.Run("status_update_performance", func(t *testing.T) {
		guildConfig := createTestConfig()
		tracker := chat.NewAgentStatusTracker(guildConfig)

		// Measure status update performance
		agentCount := 100
		updatesPerAgent := 100

		start := time.Now()
		for i := 0; i < agentCount; i++ {
			agentID := fmt.Sprintf("agent-%d", i)
			for j := 0; j < updatesPerAgent; j++ {
				status := &chat.AgentStatus{
					ID:    agentID,
					State: chat.AgentState(j % 4),
				}
				tracker.UpdateAgentStatus(agentID, status)
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
	renderer, _ := chat.NewMarkdownRenderer(80)
	content := "# Test Message\n\nThis is a **test** with `code` and lists:\n- Item 1\n- Item 2"

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = renderer.Render(content)
	}
}

func BenchmarkStatusUpdates(b *testing.B) {
	guildConfig := createTestConfig()
	tracker := chat.NewAgentStatusTracker(guildConfig)

	agents := []string{"manager", "developer", "reviewer", "tester"}
	states := []chat.AgentState{chat.AgentIdle, chat.AgentThinking, chat.AgentWorking, chat.AgentBlocked}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		agentID := agents[i%len(agents)]
		state := states[i%len(states)]

		status := &chat.AgentStatus{
			ID:    agentID,
			State: state,
		}
		tracker.UpdateAgentStatus(agentID, status)
	}
}

func BenchmarkStatusDisplay(b *testing.B) {
	guildConfig := createTestConfig()
	tracker := chat.NewAgentStatusTracker(guildConfig)
	display := chat.NewStatusDisplay(tracker, 40, 20)

	// Pre-populate with some agents
	for i := 0; i < 5; i++ {
		status := &chat.AgentStatus{
			ID:    fmt.Sprintf("agent-%d", i),
			State: chat.AgentWorking,
		}
		tracker.UpdateAgentStatus(fmt.Sprintf("agent-%d", i), status)
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = display.RenderStatusPanel()
	}
}