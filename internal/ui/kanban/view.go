// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package kanban

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/guild-ventures/guild-core/pkg/kanban"
)

// View renders the kanban board
func (m *Model) View() string {
	if m.error != nil {
		return m.renderError()
	}

	if m.loading {
		return m.renderLoading()
	}

	if m.showHelp {
		return m.renderHelp()
	}

	// Build the main board view
	var sections []string

	// Header
	sections = append(sections, m.renderHeader())

	// Column headers
	sections = append(sections, m.renderColumnHeaders())

	// Task columns
	sections = append(sections, m.renderTaskColumns())

	// Status bar
	sections = append(sections, m.renderStatusBar())

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

// renderHeader renders the board header
func (m *Model) renderHeader() string {
	// Calculate stats
	totalTasks := 0
	activeTasks := 0
	blockedTasks := 0

	for status, tasks := range m.taskCache {
		totalTasks += len(tasks)
		if status == string(kanban.StatusInProgress) {
			activeTasks = len(tasks)
		} else if status == string(kanban.StatusBlocked) {
			blockedTasks = len(tasks)
		}
	}

	header := fmt.Sprintf(
		"🏰 Workshop Board | Tasks: %d | Active: %d | Blocked: %d | FPS: %.1f",
		totalTasks, activeTasks, blockedTasks, m.fps,
	)

	width := m.viewport.Width
	if width < 80 {
		width = 80
	}

	style := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("12")).
		Background(lipgloss.Color("235")).
		Width(width).
		Align(lipgloss.Center).
		Padding(0, 1)

	return style.Render(header)
}

// renderColumnHeaders renders the column headers
func (m *Model) renderColumnHeaders() string {
	colWidth := (m.viewport.Width - 6) / 5 // -6 for borders and padding
	if colWidth < 15 {
		colWidth = 15
	}

	var headers []string

	for i, col := range m.columns {
		// Column title with count
		title := fmt.Sprintf("%s (%d)", col.Title, col.TotalTasks)

		// Style based on focus and status
		style := columnHeaderStyle.Copy().Width(colWidth)

		if i == m.viewport.FocusedColumn {
			style = style.
				Background(lipgloss.Color("237")).
				BorderStyle(lipgloss.NormalBorder()).
				BorderBottom(true).
				BorderForeground(lipgloss.Color("12"))
		}

		// Color code by status
		switch col.Status {
		case kanban.StatusTodo:
			style = style.Foreground(lipgloss.Color("11")) // Yellow
		case kanban.StatusInProgress:
			style = style.Foreground(lipgloss.Color("14")) // Cyan
		case kanban.StatusBlocked:
			style = style.Foreground(lipgloss.Color("9")) // Red
		case kanban.StatusReadyForReview:
			style = style.Foreground(lipgloss.Color("13")) // Purple
		case kanban.StatusDone:
			style = style.Foreground(lipgloss.Color("10")) // Green
		}

		headers = append(headers, style.Render(title))
	}

	return lipgloss.JoinHorizontal(lipgloss.Top, headers...)
}

// renderTaskColumns renders the task columns
func (m *Model) renderTaskColumns() string {
	colWidth := (m.viewport.Width - 6) / 5
	if colWidth < 15 {
		colWidth = 15
	}

	// Check if board is empty
	totalTasks := 0
	for _, col := range m.columns {
		totalTasks += col.TotalTasks
	}

	// If board is empty, show helpful workshop content
	if totalTasks == 0 {
		emptyMsg := headerStyle.Render("🏗️  The workshop board is empty") + "\n\n" +
			"Start by creating a commission:\n" +
			"  guild commission create \"Build a REST API\"\n\n" +
			"Or import existing tasks:\n" +
			"  guild kanban import\n"
		return lipgloss.NewStyle().
			Width(m.viewport.Width - 4).
			Align(lipgloss.Center).
			MarginTop(2).
			Render(emptyMsg)
	}

	// Build each column
	var columns []string

	for i, col := range m.columns {
		columnContent := m.renderColumn(i, col, colWidth)
		columns = append(columns, columnContent)
	}

	// Join columns horizontally
	return lipgloss.JoinHorizontal(lipgloss.Top, columns...)
}

// renderColumn renders a single column
func (m *Model) renderColumn(colIndex int, col Column, width int) string {
	var rows []string

	// Scroll indicator (top)
	if col.ScrollOffset > 0 {
		indicator := fmt.Sprintf("↑ %d more", col.ScrollOffset)
		style := scrollIndicatorStyle.Copy().Width(width).Align(lipgloss.Center)
		rows = append(rows, style.Render(indicator))
	} else {
		rows = append(rows, strings.Repeat(" ", width))
	}

	// Render visible tasks
	for i := 0; i < m.viewport.VisibleRows; i++ {
		if i < len(col.Tasks) {
			task := col.Tasks[i]
			taskView := m.renderTask(task, width, colIndex == m.viewport.FocusedColumn && i == 0)
			rows = append(rows, taskView)
		} else {
			// Empty row
			rows = append(rows, strings.Repeat(" ", width))
		}
	}

	// Scroll indicator (bottom)
	remainingTasks := col.TotalTasks - col.ScrollOffset - len(col.Tasks)
	if remainingTasks > 0 {
		indicator := fmt.Sprintf("↓ %d more", remainingTasks)
		style := scrollIndicatorStyle.Copy().Width(width).Align(lipgloss.Center)
		rows = append(rows, style.Render(indicator))
	} else {
		rows = append(rows, strings.Repeat(" ", width))
	}

	// Create column with border
	columnStyle := lipgloss.NewStyle().
		Width(width).
		Height(m.viewport.VisibleRows + 2). // +2 for scroll indicators
		BorderStyle(lipgloss.NormalBorder()).
		BorderLeft(colIndex > 0).
		BorderRight(colIndex < 4).
		BorderTop(false).
		BorderBottom(false)

	if colIndex == m.viewport.FocusedColumn {
		columnStyle = columnStyle.BorderForeground(lipgloss.Color("12"))
	} else {
		columnStyle = columnStyle.BorderForeground(lipgloss.Color("240"))
	}

	content := lipgloss.JoinVertical(lipgloss.Left, rows...)
	return columnStyle.Render(content)
}

// renderTask renders a single task
func (m *Model) renderTask(task *kanban.Task, width int, selected bool) string {
	// Priority indicator
	var priorityIcon string
	switch task.Priority {
	case kanban.PriorityHigh:
		priorityIcon = "🔴"
	case kanban.PriorityMedium:
		priorityIcon = "🟡"
	case kanban.PriorityLow:
		priorityIcon = "🟢"
	}

	// Progress indicator for in-progress tasks
	var progressBar string
	if task.Status == kanban.StatusInProgress && task.Progress > 0 {
		barWidth := 8
		filled := int(float64(barWidth) * float64(task.Progress) / 100.0)
		progressBar = fmt.Sprintf(" [%s%s] %d%%",
			strings.Repeat("█", filled),
			strings.Repeat("░", barWidth-filled),
			task.Progress,
		)
	}

	// Task ID and title
	titleWidth := width - 4 // Account for padding and priority
	if progressBar != "" {
		titleWidth -= len(progressBar)
	}

	title := fmt.Sprintf("[%s]", task.ID)
	if len(title) > titleWidth {
		title = title[:titleWidth-3] + "..."
	}

	// Task description (second line)
	desc := task.Title
	if len(desc) > width-4 {
		desc = desc[:width-7] + "..."
	}

	// Assignee (third line)
	assignee := ""
	if task.AssignedTo != "" {
		assignee = "@" + task.AssignedTo
		if len(assignee) > width-4 {
			assignee = assignee[:width-7] + "..."
		}
	}

	// Build task content
	lines := []string{
		fmt.Sprintf("%s %s%s", priorityIcon, title, progressBar),
		"  " + desc,
	}
	if assignee != "" {
		lines = append(lines, "  "+assignee)
	}

	// Apply style
	style := taskStyle.Copy().Width(width).MaxHeight(3)
	if selected {
		style = selectedTaskStyle.Copy().Width(width).MaxHeight(3)
	}

	// Special styling for blocked tasks
	if task.Status == kanban.StatusBlocked {
		style = style.Foreground(lipgloss.Color("9"))
	}

	return style.Render(strings.Join(lines, "\n"))
}

// renderStatusBar renders the bottom status bar
func (m *Model) renderStatusBar() string {
	var status string

	if m.viewport.SearchMode {
		status = fmt.Sprintf("🔍 Search: %s", m.viewport.SearchFilter)
	} else if m.statusMessage != "" {
		status = m.statusMessage
	} else {
		status = "[j/k] scroll | [h/l] columns | [1-5] jump | [/] search | [?] help | [q] quit"
	}

	style := lipgloss.NewStyle().
		Width(m.viewport.Width).
		Foreground(lipgloss.Color("241")).
		Background(lipgloss.Color("235")).
		Padding(0, 1)

	return style.Render(status)
}

// renderHelp renders the help screen
func (m *Model) renderHelp() string {
	help := `
🏰 Guild Workshop Board - Help

NAVIGATION:
  j/k     - Scroll up/down in current column
  J/K     - Page up/down in current column
  h/l     - Move between columns
  1-5     - Jump to column (1=TODO, 2=IN PROGRESS, etc.)

ACTIONS:
  Enter   - View task details
  /       - Search tasks
  r/R     - Refresh board
  ?       - Toggle this help
  q       - Quit

TASK INDICATORS:
  🔴      - High priority
  🟡      - Medium priority
  🟢      - Low priority
  [████░░] - Progress bar (for in-progress tasks)
  @name   - Assigned to

Press any key to return to the board...
`

	style := lipgloss.NewStyle().
		Width(m.viewport.Width).
		Height(m.viewport.Height).
		Align(lipgloss.Center, lipgloss.Center).
		Padding(2)

	return style.Render(help)
}

// renderLoading renders the loading screen
func (m *Model) renderLoading() string {
	style := lipgloss.NewStyle().
		Width(m.viewport.Width).
		Height(m.viewport.Height).
		Align(lipgloss.Center, lipgloss.Center)

	return style.Render("⏳ Loading tasks...")
}

// renderError renders the error screen
func (m *Model) renderError() string {
	style := lipgloss.NewStyle().
		Width(m.viewport.Width).
		Height(m.viewport.Height).
		Align(lipgloss.Center, lipgloss.Center).
		Foreground(lipgloss.Color("9"))

	return style.Render(fmt.Sprintf("❌ Error: %v\n\nPress 'r' to retry or 'q' to quit", m.error))
}
