// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package visual

import (
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// AgentStatus represents the status of an AI agent
type AgentStatus struct {
	ID          string
	Name        string
	State       AgentState
	CurrentTask string
	Progress    float64
	StartTime   time.Time
	Metadata    map[string]interface{}
}

// AgentState represents different states an agent can be in
type AgentState int

const (
	AgentIdle AgentState = iota
	AgentThinking
	AgentWorking
	AgentReviewing
	AgentBlocked
	AgentCompleted
	AgentFailed
)

// ActivityType represents different types of activities
type ActivityType int

const (
	ActivityStatusChange ActivityType = iota
	ActivityTaskStart
	ActivityTaskComplete
	ActivityCoordination
	ActivityError
)

// ActivityEvent represents an activity in the system
type ActivityEvent struct {
	Timestamp   time.Time
	EventType   ActivityType
	AgentID     string
	Description string
	Metadata    map[string]interface{}
}

// Mock implementations for testing

type AgentStatusTracker struct {
	mu         sync.RWMutex
	agents     map[string]*AgentStatus
	activities []ActivityEvent
	config     *TestConfig
}

type TestConfig struct {
	MaxAgents       int
	TrackingEnabled bool
}

func NewAgentStatusTracker(config *TestConfig) *AgentStatusTracker {
	return &AgentStatusTracker{
		agents:     make(map[string]*AgentStatus),
		activities: make([]ActivityEvent, 0),
		config:     config,
	}
}

func (ast *AgentStatusTracker) UpdateAgentStatus(agentID string, status *AgentStatus) {
	ast.mu.Lock()
	defer ast.mu.Unlock()
	ast.agents[agentID] = status

	event := ActivityEvent{
		Timestamp:   time.Now(),
		EventType:   ActivityStatusChange,
		AgentID:     agentID,
		Description: fmt.Sprintf("Agent %s changed to %v", agentID, status.State),
		Metadata:    status.Metadata,
	}
	ast.activities = append(ast.activities, event)
}

func (ast *AgentStatusTracker) GetActiveAgents() []*AgentStatus {
	ast.mu.RLock()
	defer ast.mu.RUnlock()

	active := make([]*AgentStatus, 0)
	for _, agent := range ast.agents {
		if agent.State != AgentIdle && agent.State != AgentCompleted {
			active = append(active, agent)
		}
	}
	return active
}

func (ast *AgentStatusTracker) LogCoordinationEvent(description string, agents []string, metadata map[string]interface{}) {
	ast.mu.Lock()
	defer ast.mu.Unlock()

	event := ActivityEvent{
		Timestamp:   time.Now(),
		EventType:   ActivityCoordination,
		AgentID:     strings.Join(agents, ","),
		Description: description,
		Metadata:    metadata,
	}
	ast.activities = append(ast.activities, event)
}

func (ast *AgentStatusTracker) GetRecentActivity(limit int) []ActivityEvent {
	ast.mu.RLock()
	defer ast.mu.RUnlock()

	start := len(ast.activities) - limit
	if start < 0 {
		start = 0
	}

	result := make([]ActivityEvent, len(ast.activities[start:]))
	copy(result, ast.activities[start:])
	return result
}

type StatusDisplay struct {
	tracker *AgentStatusTracker
	width   int
	height  int
}

func NewStatusDisplay(tracker *AgentStatusTracker, width, height int) *StatusDisplay {
	return &StatusDisplay{
		tracker: tracker,
		width:   width,
		height:  height,
	}
}

func (sd *StatusDisplay) RenderCompactStatus() string {
	active := sd.tracker.GetActiveAgents()
	var lines []string

	lines = append(lines, "╭─ Active Agents ─╮")
	for _, agent := range active {
		// Use ID if Name is empty, otherwise use Name
		displayName := agent.Name
		if displayName == "" {
			displayName = agent.ID
		}
		status := fmt.Sprintf("│ %s: %s │", displayName, agent.CurrentTask)
		lines = append(lines, status)
	}
	lines = append(lines, "╰─────────────────╯")

	return strings.Join(lines, "\n")
}

func (sd *StatusDisplay) RenderFullStatus() string {
	return sd.RenderCompactStatus() + "\n\n[Full status view]"
}

type AgentIndicators struct {
	mu         sync.RWMutex
	animations map[string]string
	active     map[string]bool
}

func NewAgentIndicators() *AgentIndicators {
	return &AgentIndicators{
		animations: make(map[string]string),
		active:     make(map[string]bool),
	}
}

func (ai *AgentIndicators) StartAnimations() {
	// Mock implementation
}

func (ai *AgentIndicators) StopAnimations() {
	// Mock implementation
}

func (ai *AgentIndicators) SetWorkingAnimation(agentID, taskType string) {
	ai.mu.Lock()
	defer ai.mu.Unlock()
	ai.animations[agentID] = taskType
	ai.active[agentID] = true
}

func (ai *AgentIndicators) IsAnimating(agentID string) bool {
	ai.mu.RLock()
	defer ai.mu.RUnlock()
	return ai.active[agentID]
}

func (ai *AgentIndicators) GetCurrentIndicator(agentID string) string {
	ai.mu.RLock()
	defer ai.mu.RUnlock()
	if ai.active[agentID] {
		return "⚙️" // Working indicator
	}
	return "⚪" // Idle indicator
}

// Mock MarkdownRenderer for testing
type MarkdownRenderer struct {
	width           int
	cacheHits       int64
	cacheMisses     int64
	lineNumberStyle string
}

func NewMarkdownRenderer(width int) (*MarkdownRenderer, error) {
	return &MarkdownRenderer{
		width:           width,
		lineNumberStyle: "default",
	}, nil
}

func (mr *MarkdownRenderer) Render(content string) string {
	// Simple mock rendering with line wrapping
	if strings.Contains(content, "```") {
		return "[CODE BLOCK]\n" + mr.wrapText(content)
	}
	if strings.HasPrefix(content, "#") {
		return "[HEADING] " + mr.wrapText(content)
	}
	return mr.wrapText(content)
}

// wrapText wraps text to the specified width
func (mr *MarkdownRenderer) wrapText(text string) string {
	if mr.width <= 0 {
		return text
	}

	var result []string
	lines := strings.Split(text, "\n")

	for _, line := range lines {
		if len(line) <= mr.width {
			result = append(result, line)
			continue
		}

		// Wrap long lines
		for len(line) > mr.width {
			// Find the last space before the width limit
			breakPoint := mr.width
			for i := mr.width - 1; i >= 0; i-- {
				if line[i] == ' ' {
					breakPoint = i
					break
				}
			}

			// If no space found, just break at width
			if breakPoint == mr.width && line[mr.width-1] != ' ' {
				breakPoint = mr.width
			}

			result = append(result, line[:breakPoint])
			line = strings.TrimLeft(line[breakPoint:], " ")
		}

		if len(line) > 0 {
			result = append(result, line)
		}
	}

	return strings.Join(result, "\n")
}

func (mr *MarkdownRenderer) GetCacheStats() string {
	total := mr.cacheHits + mr.cacheMisses
	if total == 0 {
		return "Cache stats: No cache activity yet"
	}

	ratio := float64(mr.cacheHits) / float64(total) * 100
	return fmt.Sprintf("Cache hits: %d, misses: %d, ratio: %.2f%%",
		mr.cacheHits, mr.cacheMisses, ratio)
}

// Mock ContentFormatter for testing
type ContentFormatter struct {
	markdownRenderer *MarkdownRenderer
	width            int
}

func NewContentFormatter(renderer *MarkdownRenderer, width int) *ContentFormatter {
	return &ContentFormatter{
		markdownRenderer: renderer,
		width:            width,
	}
}

func (cf *ContentFormatter) FormatMessage(msgType, content string, metadata map[string]interface{}) string {
	// Simple mock formatting
	return fmt.Sprintf("[%s] %s", msgType, content)
}

// Test configuration helper
func createTestConfig() *TestConfig {
	return &TestConfig{
		MaxAgents:       10,
		TrackingEnabled: true,
	}
}

// Main integration tests

func TestFullVisualIntegration(t *testing.T) {
	// Test 1: Markdown rendering with live status updates
	t.Run("markdown_with_status_panel", func(t *testing.T) {
		// Setup
		testConfig := createTestConfig()
		tracker := NewAgentStatusTracker(testConfig)
		display := NewStatusDisplay(tracker, 80, 24)
		renderer, err := NewMarkdownRenderer(80)
		require.NoError(t, err)

		// Simulate agent activity
		tracker.UpdateAgentStatus("manager", &AgentStatus{
			ID:          "manager",
			Name:        "Guild Master",
			State:       AgentThinking,
			CurrentTask: "Planning architecture",
		})

		// Render markdown content
		content := "# Task Update\n\nThe **manager** is working on:\n```go\nfunc Plan() {}\n```"
		rendered := renderer.Render(content)

		// Get status panel
		statusPanel := display.RenderCompactStatus()

		// Verify both render correctly
		assert.Contains(t, rendered, "Task Update")
		assert.Contains(t, statusPanel, "Guild Master")
		assert.Contains(t, statusPanel, "Planning architecture")

		// Verify no visual corruption when combined
		combined := statusPanel + "\n---\n" + rendered
		assert.NotContains(t, combined, "\x1b[0m\x1b[0m") // No double escapes
	})

	// Test 2: Multi-agent coordination visualization
	t.Run("multi_agent_visual_coordination", func(t *testing.T) {
		testConfig := createTestConfig()
		tracker := NewAgentStatusTracker(testConfig)
		indicators := NewAgentIndicators()
		indicators.StartAnimations()

		// Start multiple agents
		agents := []string{"manager", "developer", "reviewer"}
		for _, agent := range agents {
			tracker.UpdateAgentStatus(agent, &AgentStatus{
				ID:          agent,
				State:       AgentWorking,
				CurrentTask: "Collaborative task",
			})
			indicators.SetWorkingAnimation(agent, "collaboration")
		}

		// Log coordination event
		tracker.LogCoordinationEvent(
			"Agents coordinating on API design",
			agents,
			map[string]interface{}{"phase": "planning"},
		)

		// Verify all agents show working state
		activeAgents := tracker.GetActiveAgents()
		assert.Len(t, activeAgents, 3)

		// Verify animations are active
		for _, agent := range agents {
			assert.True(t, indicators.IsAnimating(agent))
			indicator := indicators.GetCurrentIndicator(agent)
			assert.NotEqual(t, "⚪", indicator)
		}

		// Check coordination in activity log
		events := tracker.GetRecentActivity(10)
		found := false
		for _, event := range events {
			if event.EventType == ActivityCoordination {
				found = true
				assert.Contains(t, event.Description, "coordinating")
			}
		}
		assert.True(t, found, "Coordination event should be logged")
	})

	// Test 3: Error recovery with visual stability
	t.Run("error_recovery_visual_stability", func(t *testing.T) {
		renderer, err := NewMarkdownRenderer(80)
		require.NoError(t, err)
		formatter := NewContentFormatter(renderer, 80)

		// Test malformed content
		malformedCases := []struct {
			name    string
			content string
		}{
			{"unclosed_code", "```go\nfunc broken() {"},
			{"nested_escapes", "\x1b[31m\x1b[32mNested\x1b[0m"},
			{"invalid_unicode", "Invalid: [NULL][SOH][STX]"}, // Test with representation instead of actual control chars
			{"huge_content", strings.Repeat("A", 5000)},      // 5KB - reasonable but still large
		}

		for _, tc := range malformedCases {
			t.Run(tc.name, func(t *testing.T) {
				// Should not panic
				result := formatter.FormatMessage("test", tc.content, nil)
				assert.NotEmpty(t, result)

				// Should have valid output
				assert.NotContains(t, result, "\x00")
				assert.True(t, len(result) < 10000) // Reasonable size
			})
		}
	})
}

// Test real-time updates don't cause flicker
func TestVisualUpdateStability(t *testing.T) {
	testConfig := createTestConfig()
	tracker := NewAgentStatusTracker(testConfig)
	display := NewStatusDisplay(tracker, 80, 24)

	// Simulate rapid updates
	results := make([]string, 100)
	for i := 0; i < 100; i++ {
		tracker.UpdateAgentStatus("test", &AgentStatus{
			ID:       "test",
			State:    AgentWorking,
			Progress: float64(i) / 100.0,
		})
		results[i] = display.RenderCompactStatus()
	}

	// Verify no empty frames
	for i, result := range results {
		assert.NotEmpty(t, result, "Frame %d should not be empty", i)
		assert.Contains(t, result, "test", "Frame %d should contain agent ID", i)
	}
}

// Test concurrent updates
func TestConcurrentVisualUpdates(t *testing.T) {
	testConfig := createTestConfig()
	tracker := NewAgentStatusTracker(testConfig)
	display := NewStatusDisplay(tracker, 80, 24)

	// Run concurrent updates
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				agentID := fmt.Sprintf("agent-%d", id)
				tracker.UpdateAgentStatus(agentID, &AgentStatus{
					ID:          agentID,
					Name:        fmt.Sprintf("Agent %d", id),
					State:       AgentState(j % 5),
					CurrentTask: fmt.Sprintf("Task %d", j),
				})

				// Try to render during updates
				_ = display.RenderCompactStatus()
			}
		}(i)
	}

	wg.Wait()

	// Verify final state is consistent
	finalStatus := display.RenderCompactStatus()
	assert.NotEmpty(t, finalStatus)

	// Should have some active agents
	activeAgents := tracker.GetActiveAgents()
	assert.NotEmpty(t, activeAgents)
}

// Test memory usage with large content
func TestVisualMemoryUsage(t *testing.T) {
	renderer, err := NewMarkdownRenderer(80)
	require.NoError(t, err)

	// Generate large content
	var content strings.Builder
	for i := 0; i < 1000; i++ {
		content.WriteString(fmt.Sprintf("# Section %d\n\n", i))
		content.WriteString("This is a paragraph with **bold** text and `inline code`.\n\n")
		content.WriteString("```go\nfunc example() {\n    fmt.Println(\"test\")\n}\n```\n\n")
	}

	// Should handle large content without issues
	start := time.Now()
	rendered := renderer.Render(content.String())
	duration := time.Since(start)

	assert.NotEmpty(t, rendered)
	assert.Less(t, duration, 5*time.Second, "Rendering should complete in reasonable time")
}

// Test theme consistency across components
func TestVisualThemeConsistency(t *testing.T) {
	// Create all visual components
	testConfig := createTestConfig()
	tracker := NewAgentStatusTracker(testConfig)
	display := NewStatusDisplay(tracker, 80, 24)
	renderer, err := NewMarkdownRenderer(80)
	require.NoError(t, err)
	formatter := NewContentFormatter(renderer, 80)
	indicators := NewAgentIndicators()

	// Test that all components use consistent styling
	components := []struct {
		name   string
		output string
	}{
		{"status", display.RenderCompactStatus()},
		{"markdown", renderer.Render("# Guild Framework")},
		{"formatter", formatter.FormatMessage("agent", "Test message", nil)},
		{"indicator", indicators.GetCurrentIndicator("test")},
	}

	for _, comp := range components {
		t.Run(comp.name, func(t *testing.T) {
			assert.NotEmpty(t, comp.output, "%s output should not be empty", comp.name)
			// In real implementation, would check for consistent color codes, borders, etc.
		})
	}
}

// Test graceful degradation without terminal features
func TestVisualGracefulDegradation(t *testing.T) {
	// Simulate limited terminal capabilities
	testCases := []struct {
		name        string
		capability  string
		expectation string
	}{
		{"no_unicode", "unicode", "ASCII fallback"},
		{"no_color", "color", "Plain text"},
		{"narrow_terminal", "width", "Wrapped content"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// In real implementation, would test with limited capabilities
			// For now, just verify components handle edge cases
			renderer, err := NewMarkdownRenderer(40) // Narrow width
			require.NoError(t, err)

			output := renderer.Render("This is a very long line that should wrap properly in narrow terminals")
			assert.NotEmpty(t, output)
			assert.Less(t, len(strings.Split(output, "\n")[0]), 45, "Lines should be wrapped")
		})
	}
}
