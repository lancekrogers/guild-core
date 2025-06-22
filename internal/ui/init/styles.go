// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package init

import (
	"github.com/charmbracelet/lipgloss"
)

// Styles defines all the styles used in the init TUI
type Styles struct {
	// Layout
	Container lipgloss.Style
	Section   lipgloss.Style

	// Typography
	Title    lipgloss.Style
	Subtitle lipgloss.Style
	Label    lipgloss.Style
	Value    lipgloss.Style
	Success  lipgloss.Style
	Error    lipgloss.Style
	Warning  lipgloss.Style
	Info     lipgloss.Style

	// Components
	InputBox    lipgloss.Style
	ProgressBar lipgloss.Style
	Spinner     lipgloss.Style
	Help        lipgloss.Style

	// Demo selection
	DemoItem         lipgloss.Style
	DemoItemSelected lipgloss.Style
	DemoDescription  lipgloss.Style

	// Borders
	BorderStyle lipgloss.Border
}

// NewStyles creates a new style configuration
func NewStyles() *Styles {
	// Define colors
	primaryColor := lipgloss.Color("205")  // Pink
	secondaryColor := lipgloss.Color("86") // Green
	errorColor := lipgloss.Color("196")    // Red
	warningColor := lipgloss.Color("214")  // Orange
	mutedColor := lipgloss.Color("241")    // Gray
	accentColor := lipgloss.Color("99")    // Purple

	// Use modern rounded border for a cleaner look
	modernBorder := lipgloss.RoundedBorder()

	return &Styles{
		// Layout
		Container: lipgloss.NewStyle().
			Padding(1, 2).
			Border(modernBorder).
			BorderForeground(primaryColor),

		Section: lipgloss.NewStyle().
			MarginBottom(1).
			Padding(0, 1),

		// Typography
		Title: lipgloss.NewStyle().
			Bold(true).
			Foreground(primaryColor).
			MarginBottom(1).
			Align(lipgloss.Center),

		Subtitle: lipgloss.NewStyle().
			Foreground(mutedColor).
			Italic(true).
			MarginBottom(1),

		Label: lipgloss.NewStyle().
			Foreground(accentColor).
			Bold(true),

		Value: lipgloss.NewStyle().
			Foreground(lipgloss.Color("255")),

		Success: lipgloss.NewStyle().
			Foreground(secondaryColor).
			Bold(true),

		Error: lipgloss.NewStyle().
			Foreground(errorColor).
			Bold(true),

		Warning: lipgloss.NewStyle().
			Foreground(warningColor),

		Info: lipgloss.NewStyle().
			Foreground(mutedColor).
			Italic(true),

		// Components
		InputBox: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(accentColor).
			Padding(0, 1).
			Width(60),

		ProgressBar: lipgloss.NewStyle().
			Foreground(primaryColor),

		Spinner: lipgloss.NewStyle().
			Foreground(primaryColor),

		Help: lipgloss.NewStyle().
			Foreground(mutedColor).
			MarginTop(1),

		// Demo selection
		DemoItem: lipgloss.NewStyle().
			PaddingLeft(2).
			Foreground(lipgloss.Color("255")),

		DemoItemSelected: lipgloss.NewStyle().
			PaddingLeft(1).
			Foreground(primaryColor).
			Bold(true).
			Background(lipgloss.Color("236")),

		DemoDescription: lipgloss.NewStyle().
			Foreground(mutedColor).
			PaddingLeft(4),

		BorderStyle: modernBorder,
	}
}

// RenderHeader creates a styled header with medieval flair
func (s *Styles) RenderHeader(title, subtitle string) string {
	header := lipgloss.JoinVertical(
		lipgloss.Center,
		s.Title.Render("🏰 "+title+" 🏰"),
		s.Subtitle.Render(subtitle),
	)

	return s.Container.Render(header)
}

// RenderSuccess creates a success message with icon
func (s *Styles) RenderSuccess(message string) string {
	return s.Success.Render("✅ " + message)
}

// RenderError creates an error message with icon
func (s *Styles) RenderError(message string) string {
	return s.Error.Render("❌ " + message)
}

// RenderWarning creates a warning message with icon
func (s *Styles) RenderWarning(message string) string {
	return s.Warning.Render("⚠️  " + message)
}

// RenderInfo creates an info message with icon
func (s *Styles) RenderInfo(message string) string {
	return s.Info.Render("ℹ️  " + message)
}

// RenderLabelValue creates a formatted label-value pair
func (s *Styles) RenderLabelValue(label, value string) string {
	return lipgloss.JoinHorizontal(
		lipgloss.Left,
		s.Label.Width(15).Render(label+":"),
		s.Value.Render(value),
	)
}
