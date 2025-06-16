// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package kanban

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/guild-ventures/guild-core/pkg/kanban"
)

// Commands for kanban board operations

// moveTaskCmd moves a task to a different status
func (m *Model) moveTaskCmd(taskID string, newStatus kanban.TaskStatus) tea.Cmd {
	return func() tea.Msg {
		board, err := m.kanbanManager.GetBoard(m.ctx, m.boardID)
		if err != nil {
			return errorMsg{err}
		}

		task, err := board.GetTask(m.ctx, taskID)
		if err != nil {
			return errorMsg{err}
		}

		err = task.UpdateStatus(newStatus, "user", "Status changed via UI")
		if err != nil {
			return errorMsg{err}
		}

		err = board.UpdateTask(m.ctx, task)
		if err != nil {
			return errorMsg{err}
		}

		return taskUpdatedMsg{task}
	}
}

// assignTaskCmd assigns a task to an agent
func (m *Model) assignTaskCmd(taskID string, agentID string) tea.Cmd {
	return func() tea.Msg {
		board, err := m.kanbanManager.GetBoard(m.ctx, m.boardID)
		if err != nil {
			return errorMsg{err}
		}

		task, err := board.GetTask(m.ctx, taskID)
		if err != nil {
			return errorMsg{err}
		}

		task.UpdateAssignee(agentID, "user", "Assigned via UI")

		err = board.UpdateTask(m.ctx, task)
		if err != nil {
			return errorMsg{err}
		}

		return taskUpdatedMsg{task}
	}
}

// updateProgressCmd updates task progress
func (m *Model) updateProgressCmd(taskID string, progress int) tea.Cmd {
	return func() tea.Msg {
		board, err := m.kanbanManager.GetBoard(m.ctx, m.boardID)
		if err != nil {
			return errorMsg{err}
		}

		task, err := board.GetTask(m.ctx, taskID)
		if err != nil {
			return errorMsg{err}
		}

		err = task.UpdateProgress(progress, "user", "Progress updated via UI")
		if err != nil {
			return errorMsg{err}
		}

		err = board.UpdateTask(m.ctx, task)
		if err != nil {
			return errorMsg{err}
		}

		return taskUpdatedMsg{task}
	}
}

// createTaskCmd creates a new task
func (m *Model) createTaskCmd(title, description string, status kanban.TaskStatus) tea.Cmd {
	return func() tea.Msg {
		board, err := m.kanbanManager.GetBoard(m.ctx, m.boardID)
		if err != nil {
			return errorMsg{err}
		}

		task := kanban.NewTask(title, description)
		task.Status = status

		_, err = board.CreateTask(m.ctx, task.Title, task.Description)
		if err != nil {
			return errorMsg{err}
		}

		return taskUpdatedMsg{task}
	}
}
