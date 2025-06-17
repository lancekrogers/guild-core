// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package chat

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// KanbanVisualizer provides simple kanban board visualization
type KanbanVisualizer struct {
	headerStyle  lipgloss.Style
	todoStyle    lipgloss.Style
	progressStyle lipgloss.Style
	doneStyle    lipgloss.Style
	blockedStyle lipgloss.Style
}

// NewKanbanVisualizer creates a new kanban visualizer
func NewKanbanVisualizer() *KanbanVisualizer {
	return &KanbanVisualizer{
		headerStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("205")).
			Bold(true).
			Underline(true),
		
		todoStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")),
		
		progressStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("33")).
			Bold(true),
		
		doneStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("42")).
			Bold(true),
		
		blockedStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Bold(true),
	}
}

// TaskStatus represents task status
type TaskStatus string

const (
	StatusTodo       TaskStatus = "todo"
	StatusInProgress TaskStatus = "in_progress"
	StatusDone       TaskStatus = "done"
	StatusBlocked    TaskStatus = "blocked"
)

// SimpleTask represents a simple task
type SimpleTask struct {
	ID          string
	Title       string
	Description string
	Status      TaskStatus
	AssignedTo  string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// TaskBoard represents a simple task board
type TaskBoard struct {
	Name  string
	Tasks []SimpleTask
}

// RenderBoard renders a simple kanban board view
func (k *KanbanVisualizer) RenderBoard(board *TaskBoard) string {
	var sections []string
	
	// Header
	sections = append(sections, k.headerStyle.Render(fmt.Sprintf("📋 %s", board.Name)))
	
	// Group tasks by status
	tasksByStatus := make(map[TaskStatus][]SimpleTask)
	for _, task := range board.Tasks {
		tasksByStatus[task.Status] = append(tasksByStatus[task.Status], task)
	}
	
	// Render columns
	columns := []struct {
		status TaskStatus
		title  string
		style  lipgloss.Style
	}{
		{StatusTodo, "📝 Todo", k.todoStyle},
		{StatusInProgress, "🔄 In Progress", k.progressStyle},
		{StatusDone, "✅ Done", k.doneStyle},
		{StatusBlocked, "🚫 Blocked", k.blockedStyle},
	}
	
	var columnViews []string
	for _, col := range columns {
		tasks := tasksByStatus[col.status]
		view := k.renderColumn(col.title, tasks, col.style)
		columnViews = append(columnViews, view)
	}
	
	// Join columns side by side (simplified)
	sections = append(sections, strings.Join(columnViews, "\n\n"))
	
	return strings.Join(sections, "\n")
}

// renderColumn renders a single column
func (k *KanbanVisualizer) renderColumn(title string, tasks []SimpleTask, style lipgloss.Style) string {
	var lines []string
	
	lines = append(lines, style.Render(title))
	lines = append(lines, strings.Repeat("─", 20))
	
	if len(tasks) == 0 {
		lines = append(lines, "  (no tasks)")
	} else {
		for _, task := range tasks {
			taskLine := fmt.Sprintf("• %s", task.Title)
			if task.AssignedTo != "" {
				taskLine += fmt.Sprintf(" (@%s)", task.AssignedTo)
			}
			lines = append(lines, fmt.Sprintf("  %s", taskLine))
		}
	}
	
	return strings.Join(lines, "\n")
}

// GetBoardStats returns simple board statistics
func (k *KanbanVisualizer) GetBoardStats(board *TaskBoard) map[string]int {
	stats := make(map[string]int)
	
	for _, task := range board.Tasks {
		stats[string(task.Status)]++
	}
	
	stats["total"] = len(board.Tasks)
	
	return stats
}

// RenderStats renders board statistics
func (k *KanbanVisualizer) RenderStats(board *TaskBoard) string {
	stats := k.GetBoardStats(board)
	
	var parts []string
	parts = append(parts, fmt.Sprintf("Total: %d", stats["total"]))
	parts = append(parts, fmt.Sprintf("Todo: %d", stats["todo"]))
	parts = append(parts, fmt.Sprintf("In Progress: %d", stats["in_progress"]))
	parts = append(parts, fmt.Sprintf("Done: %d", stats["done"]))
	
	if stats["blocked"] > 0 {
		parts = append(parts, k.blockedStyle.Render(fmt.Sprintf("Blocked: %d", stats["blocked"])))
	}
	
	return "📊 " + strings.Join(parts, " | ")
}

// CampaignProgressTracker provides simple campaign progress tracking
type CampaignProgressTracker struct {
	campaignName string
	startTime    time.Time
	tasks        []SimpleTask
}

// NewCampaignProgressTracker creates a new progress tracker
func NewCampaignProgressTracker(campaignName string) *CampaignProgressTracker {
	return &CampaignProgressTracker{
		campaignName: campaignName,
		startTime:    time.Now(),
		tasks:        []SimpleTask{},
	}
}

// AddTask adds a task to the tracker
func (c *CampaignProgressTracker) AddTask(task SimpleTask) {
	c.tasks = append(c.tasks, task)
}

// GetProgress calculates campaign progress
func (c *CampaignProgressTracker) GetProgress() float64 {
	if len(c.tasks) == 0 {
		return 0.0
	}
	
	var completed int
	for _, task := range c.tasks {
		if task.Status == StatusDone {
			completed++
		}
	}
	
	return float64(completed) / float64(len(c.tasks))
}

// RenderProgress renders progress information
func (c *CampaignProgressTracker) RenderProgress() string {
	progress := c.GetProgress()
	completed := 0
	inProgress := 0
	blocked := 0
	
	for _, task := range c.tasks {
		switch task.Status {
		case StatusDone:
			completed++
		case StatusInProgress:
			inProgress++
		case StatusBlocked:
			blocked++
		}
	}
	
	var lines []string
	lines = append(lines, fmt.Sprintf("🎯 Campaign: %s", c.campaignName))
	lines = append(lines, fmt.Sprintf("📊 Progress: %.1f%% (%d/%d completed)", 
		progress*100, completed, len(c.tasks)))
	lines = append(lines, fmt.Sprintf("🔄 Active: %d tasks", inProgress))
	
	if blocked > 0 {
		lines = append(lines, fmt.Sprintf("🚫 Blocked: %d tasks", blocked))
	}
	
	duration := time.Since(c.startTime)
	lines = append(lines, fmt.Sprintf("⏱️  Duration: %v", duration.Round(time.Second)))
	
	return strings.Join(lines, "\n")
}