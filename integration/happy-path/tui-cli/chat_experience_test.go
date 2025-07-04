// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package tui_cli

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestChatExperience_HappyPath validates real-time chat interaction quality
// This test ensures the chat interface meets critical SLA requirements and provides
// excellent user experience through realistic conversation scenarios.
func TestChatExperience_HappyPath(t *testing.T) {
	framework := NewTUITestFramework(t)
	defer framework.Cleanup()

	// CRITICAL SLA: Chat interface load ≤500ms
	loadStart := time.Now()
	app := framework.StartChatApp(TUIConfig{
		Width:    120,
		Height:   40,
		Theme:    "dark",
		VimMode:  false,
		MockMode: true, // Use controlled responses for testing
	})
	defer app.Quit()

	loadTime := time.Since(loadStart)
	assert.LessOrEqual(t, loadTime, 500*time.Millisecond,
		"Chat interface load time exceeded SLA: %v > 500ms", loadTime)

	t.Logf("✓ Chat interface loaded in %v (SLA: ≤500ms)", loadTime)

	// Validate initial state
	assert.True(t, app.IsResponsive(), "Chat interface must be responsive")
	assert.True(t, app.IsConnectedToDaemon(), "Must connect to daemon")

	conversationScenarios := []struct {
		name                 string
		userInput            string
		expectedResponseTime time.Duration
		responseValidation   func(string) bool
		uiValidation         func(*TUIApp) bool
		streamingRequired    bool
		qualityThreshold     float64
	}{
		{
			name:                 "Initial greeting and system check",
			userInput:            "hello, can you help me understand this codebase?",
			expectedResponseTime: 3 * time.Second,
			responseValidation: func(response string) bool {
				return len(response) > 20 &&
					(strings.Contains(strings.ToLower(response), "hello") ||
						strings.Contains(strings.ToLower(response), "help") ||
						strings.Contains(strings.ToLower(response), "understand"))
			},
			uiValidation: func(app *TUIApp) bool {
				return app.HasWelcomeMessage() && app.ShowsAgentStatus()
			},
			streamingRequired: true,
			qualityThreshold:  0.85,
		},
		{
			name:                 "Code analysis request",
			userInput:            "analyze the main.go file for potential improvements",
			expectedResponseTime: 10 * time.Second,
			responseValidation: func(response string) bool {
				return len(response) > 100 &&
					(strings.Contains(strings.ToLower(response), "main.go") ||
						strings.Contains(strings.ToLower(response), "analyze") ||
						strings.Contains(strings.ToLower(response), "improvement"))
			},
			uiValidation: func(app *TUIApp) bool {
				return app.HasCodeSyntaxHighlighting() && app.ShowsProgressIndicator()
			},
			streamingRequired: true,
			qualityThreshold:  0.90,
		},
		{
			name:                 "Follow-up with context",
			userInput:            "what about the registry package we discussed?",
			expectedResponseTime: 5 * time.Second,
			responseValidation: func(response string) bool {
				return len(response) > 50 &&
					(strings.Contains(strings.ToLower(response), "registry") ||
						strings.Contains(strings.ToLower(response), "package"))
			},
			uiValidation: func(app *TUIApp) bool {
				return app.MaintainsConversationHistory()
			},
			streamingRequired: true,
			qualityThreshold:  0.85,
		},
		{
			name:                 "Complex refactoring question",
			userInput:            "suggest comprehensive refactoring strategies for improving code maintainability",
			expectedResponseTime: 15 * time.Second,
			responseValidation: func(response string) bool {
				return len(response) > 150 &&
					(strings.Contains(strings.ToLower(response), "refactor") ||
						strings.Contains(strings.ToLower(response), "maintainability") ||
						strings.Contains(strings.ToLower(response), "strategy"))
			},
			uiValidation: func(app *TUIApp) bool {
				return app.ShowsProgressIndicator() && app.HasCodeSyntaxHighlighting()
			},
			streamingRequired: true,
			qualityThreshold:  0.88,
		},
		{
			name:                 "Quick clarification",
			userInput:            "can you explain the last point?",
			expectedResponseTime: 3 * time.Second,
			responseValidation: func(response string) bool {
				return len(response) > 30 &&
					(strings.Contains(strings.ToLower(response), "explain") ||
						strings.Contains(strings.ToLower(response), "point") ||
						strings.Contains(strings.ToLower(response), "clarify"))
			},
			uiValidation: func(app *TUIApp) bool {
				return app.MaintainsConversationHistory()
			},
			streamingRequired: true,
			qualityThreshold:  0.80,
		},
	}

	for i, scenario := range conversationScenarios {
		t.Run(scenario.name, func(t *testing.T) {
			t.Logf("=== Conversation Step %d: %s ===", i+1, scenario.name)

			// Send message and start timing
			messageStart := time.Now()
			app.SendMessage(scenario.userInput)

			// Validate immediate UI response (should be instantaneous)
			assert.True(t, app.ShowsMessageInHistory(scenario.userInput),
				"User message should appear immediately in chat history")
			assert.True(t, app.ShowsTypingIndicator(),
				"Should show typing indicator while agent responds")

			t.Logf("✓ Message sent and UI updated immediately")

			// Wait for agent response with timeout
			response, err := app.WaitForResponse(scenario.expectedResponseTime + 5*time.Second)
			responseTime := time.Since(messageStart)

			// Validate response timing and content
			require.NoError(t, err, "Must receive response within timeout")
			assert.LessOrEqual(t, responseTime, scenario.expectedResponseTime,
				"Response time exceeded SLA: %v > %v", responseTime, scenario.expectedResponseTime)

			t.Logf("✓ Response received in %v (SLA: ≤%v)", responseTime, scenario.expectedResponseTime)

			// Validate response quality
			assert.True(t, scenario.responseValidation(response),
				"Response validation failed for: %s\nResponse: %s", scenario.name, response)

			t.Logf("✓ Response content validation passed")

			// Validate UI state after response
			assert.True(t, scenario.uiValidation(app),
				"UI validation failed for: %s", scenario.name)

			t.Logf("✓ UI state validation passed")

			// Validate streaming behavior (responses should stream, not appear all at once)
			if scenario.streamingRequired {
				streamingBehavior := app.GetStreamingMetrics()
				assert.Greater(t, streamingBehavior.ChunkCount, 1,
					"Response should be streamed in multiple chunks")
				assert.LessOrEqual(t, streamingBehavior.AverageChunkDelay, 200*time.Millisecond,
					"Streaming chunks should appear smoothly")

				t.Logf("✓ Streaming: %d chunks, avg delay: %v",
					streamingBehavior.ChunkCount, streamingBehavior.AverageChunkDelay)
			}

			// Performance metrics validation
			metrics := app.GetPerformanceMetrics()
			assert.Greater(t, metrics.ResponsivenessScore, scenario.qualityThreshold,
				"Responsiveness score below threshold: %.2f < %.2f",
				metrics.ResponsivenessScore, scenario.qualityThreshold)

			t.Logf("✓ Conversation step %d completed successfully (time: %v, quality: %.1f%%)",
				i+1, responseTime, metrics.ResponsivenessScore*100)
		})
	}

	// Validate conversation quality overall
	conversationHistory := app.GetConversationHistory()
	expectedMessageCount := len(conversationScenarios) * 2 // User + Agent messages
	assert.Len(t, conversationHistory.Messages, expectedMessageCount,
		"Should have all user messages and agent responses")

	// Validate conversation context is maintained
	for i, message := range conversationHistory.Messages {
		assert.NotEmpty(t, message.Content, "Message %d should not be empty", i+1)
		assert.NotZero(t, message.Timestamp, "Message %d must have timestamp", i+1)
		assert.Contains(t, []string{"user", "assistant"}, message.Role,
			"Message %d must have valid role", i+1)
	}

	// Performance summary
	overallMetrics := app.GetPerformanceMetrics()
	t.Logf("📊 Chat Experience Performance Summary:")
	t.Logf("   - Interface Load Time: %v", loadTime)
	t.Logf("   - Average Response Time: %v", overallMetrics.AverageResponseTime)
	t.Logf("   - Memory Usage: %d MB", overallMetrics.MemoryUsageMB)
	t.Logf("   - UI Responsiveness: %.1f%%", overallMetrics.ResponsivenessScore*100)
	t.Logf("   - Messages Exchanged: %d", len(conversationHistory.Messages))

	// Final SLA validation
	assert.LessOrEqual(t, loadTime, 500*time.Millisecond, "Load time SLA validation")
	assert.GreaterOrEqual(t, overallMetrics.ResponsivenessScore, 0.85, "Overall responsiveness SLA")
	assert.LessOrEqual(t, overallMetrics.MemoryUsageMB, 100, "Memory usage should be reasonable")
}

// TestChatInterface_StressTest validates performance under heavy load
func TestChatInterface_StressTest(t *testing.T) {
	framework := NewTUITestFramework(t)
	defer framework.Cleanup()

	app := framework.StartChatApp(TUIConfig{
		Width:    120,
		Height:   40,
		Theme:    "dark",
		VimMode:  false,
		MockMode: true,
	})
	defer app.Quit()

	// Stress test scenarios
	stressScenarios := []struct {
		name            string
		messageCount    int
		messageSize     int
		maxResponseTime time.Duration
	}{
		{
			name:            "Rapid fire messages",
			messageCount:    20,
			messageSize:     50,
			maxResponseTime: 3 * time.Second,
		},
		{
			name:            "Large message content",
			messageCount:    5,
			messageSize:     1000,
			maxResponseTime: 8 * time.Second,
		},
		{
			name:            "Extended conversation",
			messageCount:    50,
			messageSize:     100,
			maxResponseTime: 5 * time.Second,
		},
	}

	for _, scenario := range stressScenarios {
		t.Run(scenario.name, func(t *testing.T) {
			startTime := time.Now()
			var totalResponseTime time.Duration
			responseTimeouts := 0
			uiFailures := 0

			for i := 0; i < scenario.messageCount; i++ {
				message := generateTestMessage(scenario.messageSize, i)

				messageStart := time.Now()
				app.SendMessage(message)

				// Verify message appears in history
				if !app.ShowsMessageInHistory(message) {
					uiFailures++
				}

				// Wait for response
				response, err := app.WaitForResponse(scenario.maxResponseTime + 2*time.Second)
				responseTime := time.Since(messageStart)
				totalResponseTime += responseTime

				if err != nil || responseTime > scenario.maxResponseTime {
					responseTimeouts++
				}

				// Validate response quality
				if len(response) < 10 {
					t.Logf("Warning: Short response for message %d", i+1)
				}
			}

			testDuration := time.Since(startTime)
			avgResponseTime := totalResponseTime / time.Duration(scenario.messageCount)

			// Validate stress test results
			assert.LessOrEqual(t, responseTimeouts, scenario.messageCount/10,
				"Too many response timeouts: %d out of %d", responseTimeouts, scenario.messageCount)

			assert.Equal(t, 0, uiFailures, "UI should remain responsive under stress")

			assert.LessOrEqual(t, avgResponseTime, scenario.maxResponseTime,
				"Average response time too high: %v > %v", avgResponseTime, scenario.maxResponseTime)

			// Memory usage should remain reasonable
			metrics := app.GetPerformanceMetrics()
			assert.LessOrEqual(t, metrics.MemoryUsageMB, 200,
				"Memory usage too high under stress: %d MB", metrics.MemoryUsageMB)

			t.Logf("Stress test results for %s:", scenario.name)
			t.Logf("  - Messages processed: %d", scenario.messageCount)
			t.Logf("  - Total time: %v", testDuration)
			t.Logf("  - Average response time: %v", avgResponseTime)
			t.Logf("  - Response timeouts: %d", responseTimeouts)
			t.Logf("  - UI failures: %d", uiFailures)
			t.Logf("  - Memory usage: %d MB", metrics.MemoryUsageMB)
		})
	}
}

// TestChatInterface_ErrorRecovery validates graceful error handling
func TestChatInterface_ErrorRecovery(t *testing.T) {
	framework := NewTUITestFramework(t)
	defer framework.Cleanup()

	app := framework.StartChatApp(TUIConfig{
		Width:    120,
		Height:   40,
		Theme:    "dark",
		VimMode:  false,
		MockMode: true,
	})
	defer app.Quit()

	errorScenarios := []struct {
		name                  string
		message               string
		expectedErrorHandling bool
		recoveryTime          time.Duration
	}{
		{
			name:                  "Empty message",
			message:               "",
			expectedErrorHandling: true,
			recoveryTime:          1 * time.Second,
		},
		{
			name:                  "Very long message",
			message:               strings.Repeat("test ", 1000),
			expectedErrorHandling: false,
			recoveryTime:          10 * time.Second,
		},
		{
			name:                  "Special characters",
			message:               "Test with special chars: !@#$%^&*()[]{}|\\:;\"'<>?,./",
			expectedErrorHandling: false,
			recoveryTime:          3 * time.Second,
		},
		{
			name:                  "Unicode and emojis",
			message:               "Test with unicode: 🚀 🎯 ✨ and unicode text: αβγδε",
			expectedErrorHandling: false,
			recoveryTime:          3 * time.Second,
		},
	}

	for _, scenario := range errorScenarios {
		t.Run(scenario.name, func(t *testing.T) {
			initialResponsiveness := app.IsResponsive()
			require.True(t, initialResponsiveness, "App should be responsive before test")

			app.SendMessage(scenario.message)

			// Check if app remains responsive
			time.Sleep(100 * time.Millisecond) // Allow processing
			assert.True(t, app.IsResponsive(), "App should remain responsive after sending message")

			// Wait for response or error handling
			response, err := app.WaitForResponse(scenario.recoveryTime + 2*time.Second)

			if scenario.expectedErrorHandling {
				// For error cases, we expect either an error or a helpful error message
				if err == nil {
					assert.NotEmpty(t, response, "Should receive error message")
					assert.Contains(t, strings.ToLower(response),
						"error", "Error should be communicated to user")
				}
			} else {
				// For valid cases, should succeed
				assert.NoError(t, err, "Should not error for valid input")
				assert.NotEmpty(t, response, "Should receive valid response")
			}

			// App should remain functional after error
			assert.True(t, app.IsResponsive(), "App should remain responsive after error recovery")
			assert.True(t, app.IsConnectedToDaemon(), "Should maintain daemon connection")

			t.Logf("Error scenario '%s' handled appropriately", scenario.name)
		})
	}
}

// TestChatInterface_ConcurrentUsers simulates multiple user sessions
func TestChatInterface_ConcurrentUsers(t *testing.T) {
	framework := NewTUITestFramework(t)
	defer framework.Cleanup()

	const numUsers = 5
	const messagesPerUser = 10

	apps := make([]*TUIApp, numUsers)
	for i := 0; i < numUsers; i++ {
		apps[i] = framework.StartChatApp(TUIConfig{
			Width:    120,
			Height:   40,
			Theme:    "dark",
			VimMode:  false,
			MockMode: true,
		})
		defer apps[i].Quit()
	}

	// Simulate concurrent user interactions
	done := make(chan bool, numUsers)
	errors := make(chan error, numUsers*messagesPerUser)

	for userID := 0; userID < numUsers; userID++ {
		go func(user int, app *TUIApp) {
			defer func() { done <- true }()

			for msg := 0; msg < messagesPerUser; msg++ {
				message := generateTestMessage(100, msg)

				app.SendMessage(message)

				response, err := app.WaitForResponse(5 * time.Second)
				if err != nil {
					errors <- err
				} else if len(response) == 0 {
					errors <- assert.AnError
				}

				// Small delay between messages
				time.Sleep(100 * time.Millisecond)
			}
		}(userID, apps[userID])
	}

	// Wait for all users to complete
	for i := 0; i < numUsers; i++ {
		select {
		case <-done:
			// User completed successfully
		case <-time.After(2 * time.Minute):
			t.Fatal("Timeout waiting for concurrent users to complete")
		}
	}

	// Check for errors
	close(errors)
	errorCount := 0
	for err := range errors {
		errorCount++
		t.Logf("Concurrent user error: %v", err)
	}

	assert.LessOrEqual(t, errorCount, numUsers*messagesPerUser/10,
		"Too many errors in concurrent usage: %d", errorCount)

	// Validate all apps are still responsive
	for i, app := range apps {
		assert.True(t, app.IsResponsive(), "App %d should remain responsive", i+1)
		metrics := app.GetPerformanceMetrics()
		assert.GreaterOrEqual(t, metrics.ResponsivenessScore, 0.80,
			"App %d responsiveness too low: %.2f", i+1, metrics.ResponsivenessScore)
	}

	t.Logf("Concurrent users test completed:")
	t.Logf("  - Users: %d", numUsers)
	t.Logf("  - Messages per user: %d", messagesPerUser)
	t.Logf("  - Total errors: %d", errorCount)
	t.Logf("  - Error rate: %.1f%%", float64(errorCount)/float64(numUsers*messagesPerUser)*100)
}

// Helper functions

func generateTestMessage(size int, index int) string {
	if size <= 0 {
		return ""
	}

	baseMessage := "This is a test message "
	if size <= len(baseMessage) {
		return baseMessage[:size]
	}

	// Repeat the base message to reach desired size
	repeats := size / len(baseMessage)
	remainder := size % len(baseMessage)

	result := strings.Repeat(baseMessage, repeats)
	if remainder > 0 {
		result += baseMessage[:remainder]
	}

	// Add unique identifier
	result = result + " #" + string(rune('0'+index%10))

	return result
}
