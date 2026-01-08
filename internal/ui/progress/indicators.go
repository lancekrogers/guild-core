// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package progress

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"charm.land/bubbles/v2/progress"
	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// Indicator manages visual progress indicators for operations
type Indicator struct {
	spinner  spinner.Model
	progress progress.Model
	message  string
	percent  float64
	style    lipgloss.Style
	active   bool
	mu       sync.RWMutex
}

// NewIndicator creates a new progress indicator
func NewIndicator() *Indicator {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	p := progress.New(progress.WithDefaultBlend())

	return &Indicator{
		spinner:  s,
		progress: p,
		style:    lipgloss.NewStyle().Margin(1, 2),
	}
}

// SpinnerMsg represents a spinner update message
type SpinnerMsg struct {
	Message string
}

// ProgressMsg represents a progress update message
type ProgressMsg struct {
	Message string
	Current int
	Total   int
}

// CompletionMsg represents operation completion
type CompletionMsg struct {
	Success bool
	Message string
}

// Init initializes the indicator for Bubble Tea
func (i *Indicator) Init() tea.Cmd {
	return i.spinner.Tick
}

// Update handles Bubble Tea messages
func (i *Indicator) Update(msg tea.Msg) (*Indicator, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case SpinnerMsg:
		i.mu.Lock()
		i.message = msg.Message
		i.active = true
		i.mu.Unlock()
		i.spinner, cmd = i.spinner.Update(msg)
		return i, cmd

	case ProgressMsg:
		i.mu.Lock()
		i.message = msg.Message
		i.percent = float64(msg.Current) / float64(msg.Total)
		i.active = true
		i.mu.Unlock()
		return i, nil

	case CompletionMsg:
		i.mu.Lock()
		i.active = false
		i.message = msg.Message
		i.mu.Unlock()
		return i, nil

	case spinner.TickMsg:
		i.spinner, cmd = i.spinner.Update(msg)
		return i, cmd

	default:
		return i, nil
	}
}

// View renders the indicator
func (i *Indicator) View() string {
	i.mu.RLock()
	defer i.mu.RUnlock()

	if !i.active {
		return ""
	}

	if i.percent > 0 {
		// Show progress bar
		progressBar := i.progress.ViewAs(i.percent)
		status := fmt.Sprintf("%s: %.0f%%", i.message, i.percent*100)

		return i.style.Render(
			fmt.Sprintf("%s\n%s", status, progressBar),
		)
	}

	// Show spinner
	return i.style.Render(fmt.Sprintf("%s %s", i.spinner.View(), i.message))
}

// ShowSpinner displays a spinner with message
func (i *Indicator) ShowSpinner(message string) tea.Cmd {
	return func() tea.Msg {
		return SpinnerMsg{Message: message}
	}
}

// ShowProgress displays a progress bar
func (i *Indicator) ShowProgress(message string, current, total int) tea.Cmd {
	return func() tea.Msg {
		return ProgressMsg{
			Message: message,
			Current: current,
			Total:   total,
		}
	}
}

// Complete marks the operation as complete
func (i *Indicator) Complete(success bool, message string) tea.Cmd {
	return func() tea.Msg {
		return CompletionMsg{
			Success: success,
			Message: message,
		}
	}
}

// MultiStageProgress manages multi-stage operation progress
type MultiStageProgress struct {
	stages   []Stage
	current  int
	progress *Indicator
	mu       sync.RWMutex
}

// Stage represents a stage in a multi-stage operation
type Stage struct {
	Name        string
	Weight      float64
	Progress    float64
	Status      StageStatus
	StartTime   time.Time
	EndTime     time.Time
	Error       error
	Description string
}

// StageStatus represents the status of a stage
type StageStatus int

const (
	StageStatusPending StageStatus = iota
	StageStatusInProgress
	StageStatusCompleted
	StageStatusFailed
)

// NewMultiStageProgress creates a new multi-stage progress tracker
func NewMultiStageProgress(stages []string) *MultiStageProgress {
	s := make([]Stage, len(stages))
	weight := 1.0 / float64(len(stages))

	for i, name := range stages {
		s[i] = Stage{
			Name:   name,
			Weight: weight,
			Status: StageStatusPending,
		}
	}

	return &MultiStageProgress{
		stages:   s,
		progress: NewIndicator(),
	}
}

// MultiStageMsg represents multi-stage progress updates
type MultiStageMsg struct {
	StageIndex int
	Progress   float64
	Message    string
}

// StageStartMsg represents stage start
type StageStartMsg struct {
	StageIndex int
}

// StageCompleteMsg represents stage completion
type StageCompleteMsg struct {
	StageIndex int
	Success    bool
	Error      error
}

// Init initializes multi-stage progress
func (m *MultiStageProgress) Init() tea.Cmd {
	return m.progress.Init()
}

// Update handles multi-stage progress updates
func (m *MultiStageProgress) Update(msg tea.Msg) (*MultiStageProgress, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case StageStartMsg:
		m.mu.Lock()
		m.current = msg.StageIndex
		if msg.StageIndex < len(m.stages) {
			m.stages[msg.StageIndex].Status = StageStatusInProgress
			m.stages[msg.StageIndex].StartTime = time.Now()
		}
		m.mu.Unlock()
		return m, m.updateOverallProgress()

	case MultiStageMsg:
		m.mu.Lock()
		if msg.StageIndex < len(m.stages) {
			m.stages[msg.StageIndex].Progress = msg.Progress
		}
		m.mu.Unlock()
		return m, m.updateOverallProgress()

	case StageCompleteMsg:
		m.mu.Lock()
		if msg.StageIndex < len(m.stages) {
			stage := &m.stages[msg.StageIndex]
			stage.EndTime = time.Now()
			stage.Progress = 1.0
			if msg.Success {
				stage.Status = StageStatusCompleted
			} else {
				stage.Status = StageStatusFailed
				stage.Error = msg.Error
			}
		}
		m.mu.Unlock()
		return m, m.updateOverallProgress()

	default:
		m.progress, cmd = m.progress.Update(msg)
		return m, cmd
	}
}

// View renders the multi-stage progress
func (m *MultiStageProgress) View() string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var parts []string

	// Overall progress
	parts = append(parts, m.progress.View())

	// Stage details
	for _, stage := range m.stages {
		icon := m.getStageIcon(stage.Status)
		name := stage.Name

		if stage.Status == StageStatusInProgress {
			name = lipgloss.NewStyle().Bold(true).Render(name)
		}

		line := fmt.Sprintf("%s %s", icon, name)

		if stage.Status == StageStatusInProgress && stage.Progress > 0 {
			line += fmt.Sprintf(" (%.0f%%)", stage.Progress*100)
		}

		if stage.Status == StageStatusFailed && stage.Error != nil {
			line += fmt.Sprintf(" - %v", stage.Error)
		}

		parts = append(parts, "  "+line)
	}

	return strings.Join(parts, "\n")
}

// getStageIcon returns the appropriate icon for a stage status
func (m *MultiStageProgress) getStageIcon(status StageStatus) string {
	switch status {
	case StageStatusPending:
		return "⏳"
	case StageStatusInProgress:
		return "🔄"
	case StageStatusCompleted:
		return "✅"
	case StageStatusFailed:
		return "❌"
	default:
		return "❓"
	}
}

// updateOverallProgress calculates and updates overall progress
func (m *MultiStageProgress) updateOverallProgress() tea.Cmd {
	var total float64
	var completed int

	for i, stage := range m.stages {
		if i < m.current {
			total += stage.Weight
			if stage.Status == StageStatusCompleted {
				completed++
			}
		} else if i == m.current {
			total += stage.Weight * stage.Progress
		}
	}

	// Convert to integer percentage
	current := int(total * 100)
	return m.progress.ShowProgress("Overall Progress", current, 100)
}

// StartStage starts a specific stage
func (m *MultiStageProgress) StartStage(index int) tea.Cmd {
	return func() tea.Msg {
		return StageStartMsg{StageIndex: index}
	}
}

// UpdateStage updates stage progress
func (m *MultiStageProgress) UpdateStage(index int, progress float64, message string) tea.Cmd {
	return func() tea.Msg {
		return MultiStageMsg{
			StageIndex: index,
			Progress:   progress,
			Message:    message,
		}
	}
}

// CompleteStage marks a stage as complete
func (m *MultiStageProgress) CompleteStage(index int, success bool, err error) tea.Cmd {
	return func() tea.Msg {
		return StageCompleteMsg{
			StageIndex: index,
			Success:    success,
			Error:      err,
		}
	}
}

// GetCurrentStage returns the current active stage
func (m *MultiStageProgress) GetCurrentStage() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.current
}

// GetStageProgress returns progress for a specific stage
func (m *MultiStageProgress) GetStageProgress(index int) float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if index < 0 || index >= len(m.stages) {
		return 0
	}

	return m.stages[index].Progress
}

// IsComplete returns true if all stages are complete
func (m *MultiStageProgress) IsComplete() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, stage := range m.stages {
		if stage.Status != StageStatusCompleted {
			return false
		}
	}

	return true
}

// HasErrors returns true if any stage has failed
func (m *MultiStageProgress) HasErrors() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, stage := range m.stages {
		if stage.Status == StageStatusFailed {
			return true
		}
	}

	return false
}

// GetSummary returns a summary of the operation
func (m *MultiStageProgress) GetSummary() string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var completed, failed int
	var totalDuration time.Duration

	for _, stage := range m.stages {
		switch stage.Status {
		case StageStatusCompleted:
			completed++
			if !stage.EndTime.IsZero() && !stage.StartTime.IsZero() {
				totalDuration += stage.EndTime.Sub(stage.StartTime)
			}
		case StageStatusFailed:
			failed++
		}
	}

	if failed > 0 {
		return fmt.Sprintf("❌ Operation failed: %d/%d stages completed, %d failed",
			completed, len(m.stages), failed)
	}

	if completed == len(m.stages) {
		return fmt.Sprintf("✅ Operation completed successfully in %v",
			totalDuration.Round(time.Millisecond))
	}

	return fmt.Sprintf("🔄 Operation in progress: %d/%d stages completed",
		completed, len(m.stages))
}

// Commission-specific progress tracker
func NewCommissionProgress() *MultiStageProgress {
	stages := []string{
		"⚡ Analyzing Requirements",
		"🧩 Breaking Down Tasks",
		"👥 Assigning to Agents",
		"🔨 Generating Implementation",
		"✅ Review & Validation",
	}

	return NewMultiStageProgress(stages)
}

// Tool execution progress tracker
func NewToolExecutionProgress(toolName string) *MultiStageProgress {
	stages := []string{
		fmt.Sprintf("🔧 Preparing %s", toolName),
		"📋 Validating Parameters",
		"⚡ Executing Operation",
		"📊 Processing Results",
	}

	return NewMultiStageProgress(stages)
}
