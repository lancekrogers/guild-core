// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

//go:build integration
// +build integration

package visual

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lancekrogers/guild/internal/ui/chat/session"
	"github.com/lancekrogers/guild/internal/ui/progress"
	"github.com/lancekrogers/guild/internal/ui/tools"
)

// Mock implementations for testing

// mockSessionStore implements session.SessionStore for testing
type mockSessionStore struct {
	sessions map[string]*session.Session
	messages map[string][]*session.Message
}

func (m *mockSessionStore) CreateSession(ctx context.Context, s *session.Session) error {
	if m.sessions == nil {
		m.sessions = make(map[string]*session.Session)
	}
	m.sessions[s.ID] = s
	return nil
}

func (m *mockSessionStore) GetSession(ctx context.Context, id string) (*session.Session, error) {
	if s, ok := m.sessions[id]; ok {
		return s, nil
	}
	return nil, fmt.Errorf("session not found")
}

func (m *mockSessionStore) GetMessages(ctx context.Context, sessionID string) ([]*session.Message, error) {
	if msgs, ok := m.messages[sessionID]; ok {
		return msgs, nil
	}
	return []*session.Message{}, nil
}

// Implement other required SessionStore methods with minimal functionality
func (m *mockSessionStore) ListSessions(ctx context.Context, limit, offset int32) ([]*session.Session, error) { return nil, nil }
func (m *mockSessionStore) ListSessionsByCampaign(ctx context.Context, campaignID string) ([]*session.Session, error) { return nil, nil }
func (m *mockSessionStore) UpdateSession(ctx context.Context, s *session.Session) error { return nil }
func (m *mockSessionStore) DeleteSession(ctx context.Context, id string) error { return nil }
func (m *mockSessionStore) SearchSessions(ctx context.Context, query string, limit, offset int32) ([]*session.Session, error) { return nil, nil }
func (m *mockSessionStore) CountSessions(ctx context.Context) (int64, error) { return 0, nil }
func (m *mockSessionStore) SaveMessage(ctx context.Context, message *session.Message) error { return nil }
func (m *mockSessionStore) GetMessage(ctx context.Context, id string) (*session.Message, error) { return nil, nil }
func (m *mockSessionStore) GetMessagesPaginated(ctx context.Context, sessionID string, limit, offset int32) ([]*session.Message, error) { return nil, nil }
func (m *mockSessionStore) GetMessagesAfter(ctx context.Context, sessionID string, after time.Time) ([]*session.Message, error) { return nil, nil }
func (m *mockSessionStore) CountMessages(ctx context.Context, sessionID string) (int64, error) { return 0, nil }
func (m *mockSessionStore) DeleteMessage(ctx context.Context, id string) error { return nil }
func (m *mockSessionStore) SearchMessages(ctx context.Context, query string, limit, offset int32) ([]*session.MessageSearchResult, error) { return nil, nil }
func (m *mockSessionStore) CreateBookmark(ctx context.Context, bookmark *session.Bookmark) error { return nil }
func (m *mockSessionStore) GetBookmark(ctx context.Context, id string) (*session.Bookmark, error) { return nil, nil }
func (m *mockSessionStore) GetBookmarks(ctx context.Context, sessionID string) ([]*session.BookmarkWithDetails, error) { return nil, nil }
func (m *mockSessionStore) DeleteBookmark(ctx context.Context, id string) error { return nil }
func (m *mockSessionStore) GetBookmarksByMessage(ctx context.Context, messageID string) ([]*session.Bookmark, error) { return nil, nil }

// KanbanVisualizer provides visualization functionality for kanban boards
type KanbanVisualizer struct {
}

// NewKanbanVisualizer creates a new kanban visualizer
func NewKanbanVisualizer() *KanbanVisualizer {
	return &KanbanVisualizer{}
}

// TaskBoard represents a simplified task board for visualization
type TaskBoard struct {
	Name  string
	Tasks []SimpleTask
}

// SimpleTask represents a simplified task for visualization
type SimpleTask struct {
	ID         string
	Title      string
	Status     TaskStatus
	AssignedTo string
	CreatedAt  time.Time
}

// TaskStatus represents task status for visualization
type TaskStatus string

const (
	StatusTodo       TaskStatus = "todo"
	StatusInProgress TaskStatus = "in_progress"
	StatusDone       TaskStatus = "done"
)

// RenderBoard renders a task board as a string
func (kv *KanbanVisualizer) RenderBoard(board *TaskBoard) string {
	var lines []string
	lines = append(lines, fmt.Sprintf("📋 %s", board.Name))
	lines = append(lines, "")

	// Group tasks by status
	todoTasks := []SimpleTask{}
	inProgressTasks := []SimpleTask{}
	doneTasks := []SimpleTask{}

	for _, task := range board.Tasks {
		switch task.Status {
		case StatusTodo:
			todoTasks = append(todoTasks, task)
		case StatusInProgress:
			inProgressTasks = append(inProgressTasks, task)
		case StatusDone:
			doneTasks = append(doneTasks, task)
		}
	}

	// Render columns
	lines = append(lines, "📝 Todo")
	for _, task := range todoTasks {
		lines = append(lines, fmt.Sprintf("  - %s", task.Title))
	}
	lines = append(lines, "")

	lines = append(lines, "🔄 In Progress")
	for _, task := range inProgressTasks {
		lines = append(lines, fmt.Sprintf("  - %s", task.Title))
	}
	lines = append(lines, "")

	lines = append(lines, "✅ Done")
	for _, task := range doneTasks {
		lines = append(lines, fmt.Sprintf("  - %s", task.Title))
	}

	return strings.Join(lines, "\n")
}

// GetBoardStats returns statistics about the board
func (kv *KanbanVisualizer) GetBoardStats(board *TaskBoard) map[string]int {
	stats := map[string]int{
		"total":       len(board.Tasks),
		"todo":        0,
		"in_progress": 0,
		"done":        0,
	}

	for _, task := range board.Tasks {
		switch task.Status {
		case StatusTodo:
			stats["todo"]++
		case StatusInProgress:
			stats["in_progress"]++
		case StatusDone:
			stats["done"]++
		}
	}

	return stats
}

// RenderStats renders board statistics
func (kv *KanbanVisualizer) RenderStats(board *TaskBoard) string {
	stats := kv.GetBoardStats(board)
	var lines []string

	lines = append(lines, fmt.Sprintf("Total: %d", stats["total"]))
	lines = append(lines, fmt.Sprintf("Todo: %d", stats["todo"]))
	lines = append(lines, fmt.Sprintf("In Progress: %d", stats["in_progress"]))
	lines = append(lines, fmt.Sprintf("Done: %d", stats["done"]))

	return strings.Join(lines, "\n")
}

// CampaignProgressTracker tracks progress of campaigns
type CampaignProgressTracker struct {
	campaignName string
	tasks        []SimpleTask
	startTime    time.Time
}

// NewCampaignProgressTracker creates a new campaign progress tracker
func NewCampaignProgressTracker(campaignName string) *CampaignProgressTracker {
	return &CampaignProgressTracker{
		campaignName: campaignName,
		tasks:        make([]SimpleTask, 0),
		startTime:    time.Now(),
	}
}

// AddTask adds a task to track
func (cpt *CampaignProgressTracker) AddTask(task SimpleTask) {
	cpt.tasks = append(cpt.tasks, task)
}

// GetProgress calculates the completion progress (0.0 to 1.0)
func (cpt *CampaignProgressTracker) GetProgress() float64 {
	if len(cpt.tasks) == 0 {
		return 0.0
	}

	doneCount := 0
	for _, task := range cpt.tasks {
		if task.Status == StatusDone {
			doneCount++
		}
	}

	return float64(doneCount) / float64(len(cpt.tasks))
}

// RenderProgress renders the campaign progress
func (cpt *CampaignProgressTracker) RenderProgress() string {
	progress := cpt.GetProgress()
	duration := time.Since(cpt.startTime)

	var lines []string
	lines = append(lines, fmt.Sprintf("🎯 Campaign: %s", cpt.campaignName))
	lines = append(lines, fmt.Sprintf("📊 Progress: %.1f%% (%d/%d tasks)", progress*100, int(progress*float64(len(cpt.tasks))), len(cpt.tasks)))
	lines = append(lines, fmt.Sprintf("⏱️  Duration: %s", duration.Round(time.Second)))

	return strings.Join(lines, "\n")
}

// TestProgressIndicators tests the progress indicator system
func TestProgressIndicators(t *testing.T) {
	t.Run("Basic Spinner", func(t *testing.T) {
		indicator := progress.NewIndicator()
		require.NotNil(t, indicator)

		// Test initialization
		cmd := indicator.Init()
		assert.NotNil(t, cmd)

		// Test spinner message
		spinnerCmd := indicator.ShowSpinner("Processing...")
		msg := spinnerCmd()

		spinnerMsg, ok := msg.(progress.SpinnerMsg)
		assert.True(t, ok)
		assert.Equal(t, "Processing...", spinnerMsg.Message)

		// Test update
		updated, _ := indicator.Update(spinnerMsg)
		assert.NotNil(t, updated)

		// Test view
		view := updated.View()
		assert.Contains(t, view, "Processing...")
	})

	t.Run("Progress Bar", func(t *testing.T) {
		indicator := progress.NewIndicator()

		// Test progress update
		progressCmd := indicator.ShowProgress("Loading", 5, 10)
		msg := progressCmd()

		progressMsg, ok := msg.(progress.ProgressMsg)
		assert.True(t, ok)
		assert.Equal(t, "Loading", progressMsg.Message)
		assert.Equal(t, 5, progressMsg.Current)
		assert.Equal(t, 10, progressMsg.Total)

		// Test update
		updated, _ := indicator.Update(progressMsg)
		assert.NotNil(t, updated)

		// Test view contains progress bar
		view := updated.View()
		assert.Contains(t, view, "Loading")
		assert.Contains(t, view, "50%")
	})

	t.Run("Multi-Stage Progress", func(t *testing.T) {
		stages := []string{"Stage 1", "Stage 2", "Stage 3"}
		multiProgress := progress.NewMultiStageProgress(stages)
		require.NotNil(t, multiProgress)

		// Test initialization
		cmd := multiProgress.Init()
		assert.NotNil(t, cmd)

		// Test starting a stage
		startCmd := multiProgress.StartStage(0)
		msg := startCmd()

		startMsg, ok := msg.(progress.StageStartMsg)
		assert.True(t, ok)
		assert.Equal(t, 0, startMsg.StageIndex)

		// Test update
		updated, updateCmd := multiProgress.Update(startMsg)
		assert.NotNil(t, updated)
		assert.NotNil(t, updateCmd)

		// Test stage progress
		assert.Equal(t, 0, updated.GetCurrentStage())

		// Test completion
		completeCmd := updated.CompleteStage(0, true, nil)
		completeMsg := completeCmd()

		stageCompleteMsg, ok := completeMsg.(progress.StageCompleteMsg)
		assert.True(t, ok)
		assert.True(t, stageCompleteMsg.Success)
		assert.Nil(t, stageCompleteMsg.Error)
	})
}

// TestToolVisualization tests the tool execution visualizer
func TestToolVisualization(t *testing.T) {
	visualizer := tools.NewToolVisualizer()
	require.NotNil(t, visualizer)

	t.Run("Tool Execution Display", func(t *testing.T) {
		// Start tool execution
		opID := visualizer.StartToolExecution("multi_edit_tool", map[string]interface{}{
			"file":  "main.go",
			"edits": 3,
		})
		assert.NotEmpty(t, opID)

		// Update progress
		visualizer.UpdateToolProgress(opID, "Validating", 0.5)

		// Complete execution
		visualizer.CompleteToolExecution(opID, true, "3 edits applied", nil)

		// Check statistics
		stats := visualizer.GetStats()
		assert.Equal(t, 1, stats.TotalExecutions)
		assert.Equal(t, 1, stats.SuccessfulOps)
		assert.Equal(t, 0, stats.FailedOps)
	})

	t.Run("Statistics", func(t *testing.T) {
		// Execute some operations to generate stats
		opID1 := visualizer.StartToolExecution("test_tool", map[string]interface{}{})
		visualizer.CompleteToolExecution(opID1, true, "success", nil)

		opID2 := visualizer.StartToolExecution("test_tool", map[string]interface{}{})
		visualizer.CompleteToolExecution(opID2, false, nil, assert.AnError)

		stats := visualizer.GetStats()
		assert.GreaterOrEqual(t, stats.TotalExecutions, 2)
		assert.GreaterOrEqual(t, stats.SuccessfulOps, 1)
		assert.GreaterOrEqual(t, stats.FailedOps, 1)
		assert.Contains(t, stats.ToolUsageCounts, "test_tool")
	})
}

// TestSessionExport tests simple session export functionality
func TestSessionExport(t *testing.T) {
	// Create a mock session store and manager
	mockStore := &mockSessionStore{}
	manager := session.NewManager(mockStore)

	// Create a test session
	testSession := &session.Session{
		ID:        "test-session",
		Name:      "Test Chat Session",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Create test messages
	testMessages := []*session.Message{
		{
			ID:        "msg1",
			SessionID: "test-session",
			Role:      session.RoleUser,
			Content:   "Hello, how are you?",
			CreatedAt: time.Now(),
		},
		{
			ID:        "msg2",
			SessionID: "test-session",
			Role:      session.RoleAssistant,
			Content:   "I'm doing well, thank you! How can I help you today?",
			CreatedAt: time.Now().Add(1 * time.Second),
		},
	}

	// Initialize maps and set up mock responses
	mockStore.sessions = make(map[string]*session.Session)
	mockStore.messages = make(map[string][]*session.Message)
	mockStore.sessions[testSession.ID] = testSession
	mockStore.messages[testSession.ID] = testMessages

	t.Run("Markdown Export", func(t *testing.T) {
		markdown, err := manager.ExportSession("test-session", session.ExportFormatMarkdown)
		require.NoError(t, err)
		
		markdownStr := string(markdown)
		assert.Contains(t, markdownStr, "Test Chat Session")
		assert.Contains(t, markdownStr, "Hello, how are you?")
		assert.Contains(t, markdownStr, "I'm doing well")
		assert.Contains(t, markdownStr, "User")
		assert.Contains(t, markdownStr, "Assistant")
	})

	t.Run("HTML Export", func(t *testing.T) {
		html, err := manager.ExportSession("test-session", session.ExportFormatHTML)
		require.NoError(t, err)
		
		htmlStr := string(html)
		assert.Contains(t, htmlStr, "<!DOCTYPE html>")
		assert.Contains(t, htmlStr, "Test Chat Session")
		assert.Contains(t, htmlStr, "Hello, how are you?")
		assert.Contains(t, htmlStr, "message user")
		assert.Contains(t, htmlStr, "message assistant")
	})
}

// TestKanbanVisualization tests the kanban helper functionality
func TestKanbanVisualization(t *testing.T) {
	visualizer := NewKanbanVisualizer()
	require.NotNil(t, visualizer)

	t.Run("Board Rendering", func(t *testing.T) {
		board := &TaskBoard{
			Name: "Test Board",
			Tasks: []SimpleTask{
				{
					ID:         "task1",
					Title:      "Complete feature A",
					Status:     StatusTodo,
					AssignedTo: "agent1",
					CreatedAt:  time.Now(),
				},
				{
					ID:         "task2",
					Title:      "Fix bug B",
					Status:     StatusInProgress,
					AssignedTo: "agent2",
					CreatedAt:  time.Now(),
				},
				{
					ID:         "task3",
					Title:      "Deploy to production",
					Status:     StatusDone,
					AssignedTo: "agent1",
					CreatedAt:  time.Now(),
				},
			},
		}

		view := visualizer.RenderBoard(board)

		assert.Contains(t, view, "📋 Test Board")
		assert.Contains(t, view, "Complete feature A")
		assert.Contains(t, view, "Fix bug B")
		assert.Contains(t, view, "Deploy to production")
		assert.Contains(t, view, "📝 Todo")
		assert.Contains(t, view, "🔄 In Progress")
		assert.Contains(t, view, "✅ Done")
	})

	t.Run("Statistics", func(t *testing.T) {
		board := &TaskBoard{
			Name: "Test Board",
			Tasks: []SimpleTask{
				{Status: StatusTodo},
				{Status: StatusInProgress},
				{Status: StatusDone},
				{Status: StatusDone},
			},
		}

		stats := visualizer.GetBoardStats(board)

		assert.Equal(t, 4, stats["total"])
		assert.Equal(t, 1, stats["todo"])
		assert.Equal(t, 1, stats["in_progress"])
		assert.Equal(t, 2, stats["done"])

		statsView := visualizer.RenderStats(board)
		assert.Contains(t, statsView, "Total: 4")
		assert.Contains(t, statsView, "Todo: 1")
		assert.Contains(t, statsView, "In Progress: 1")
		assert.Contains(t, statsView, "Done: 2")
	})
}

// TestCampaignProgress tests campaign progress tracking
func TestCampaignProgress(t *testing.T) {
	tracker := NewCampaignProgressTracker("Test Campaign")
	require.NotNil(t, tracker)

	t.Run("Progress Calculation", func(t *testing.T) {
		// Initially no progress
		assert.Equal(t, 0.0, tracker.GetProgress())

		// Add some tasks
		tracker.AddTask(SimpleTask{
			ID:     "task1",
			Title:  "Task 1",
			Status: StatusTodo,
		})
		tracker.AddTask(SimpleTask{
			ID:     "task2",
			Title:  "Task 2",
			Status: StatusDone,
		})
		tracker.AddTask(SimpleTask{
			ID:     "task3",
			Title:  "Task 3",
			Status: StatusDone,
		})

		// Should be 2/3 = 0.67 progress
		progress := tracker.GetProgress()
		assert.InDelta(t, 0.67, progress, 0.01)
	})

	t.Run("Progress Rendering", func(t *testing.T) {
		view := tracker.RenderProgress()

		assert.Contains(t, view, "🎯 Campaign: Test Campaign")
		assert.Contains(t, view, "📊 Progress:")
		assert.Contains(t, view, "⏱️  Duration:")
	})
}

// TestPerformance tests performance aspects of visual components
func TestPerformance(t *testing.T) {
	t.Run("Progress Indicator Performance", func(t *testing.T) {
		indicator := progress.NewIndicator()

		start := time.Now()

		// Simulate rapid updates
		for i := 0; i < 100; i++ {
			progressCmd := indicator.ShowProgress("Testing", i, 100)
			msg := progressCmd()
			indicator.Update(msg)
			indicator.View() // Render view
		}

		duration := time.Since(start)

		// Should complete quickly (less than 100ms for 100 updates)
		assert.Less(t, duration, 100*time.Millisecond)
	})

	t.Run("Tool Visualizer Performance", func(t *testing.T) {
		visualizer := tools.NewToolVisualizer()

		start := time.Now()

		// Simulate multiple concurrent tool executions
		opIDs := make([]string, 10)
		for i := 0; i < 10; i++ {
			opIDs[i] = visualizer.StartToolExecution("tool",
				map[string]interface{}{"test": true})
		}

		// Complete all operations
		for _, opID := range opIDs {
			visualizer.CompleteToolExecution(opID, true, "success", nil)
		}

		duration := time.Since(start)

		// Should handle multiple operations efficiently
		assert.Less(t, duration, 50*time.Millisecond)

		// Check statistics
		stats := visualizer.GetStats()
		assert.GreaterOrEqual(t, stats.TotalExecutions, 10)
		assert.GreaterOrEqual(t, stats.SuccessfulOps, 10)
	})
}

// BenchmarkProgressIndicator benchmarks progress indicator performance
func BenchmarkProgressIndicator(b *testing.B) {
	indicator := progress.NewIndicator()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		progressCmd := indicator.ShowProgress("Benchmark", i%100, 100)
		msg := progressCmd()
		indicator.Update(msg)
		indicator.View()
	}
}

// BenchmarkToolVisualizer benchmarks tool visualizer performance
func BenchmarkToolVisualizer(b *testing.B) {
	visualizer := tools.NewToolVisualizer()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		opID := visualizer.StartToolExecution("benchmark_tool", map[string]interface{}{
			"iteration": i,
		})
		visualizer.CompleteToolExecution(opID, true, "success", nil)
	}
}
