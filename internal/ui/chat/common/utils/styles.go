// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package utils

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

// Styles provides centralized styling for the Guild Chat interface
type Styles struct {
	// Base styles
	Base      lipgloss.Style
	Container lipgloss.Style

	// Pane styles
	OutputPane lipgloss.Style
	InputPane  lipgloss.Style
	StatusPane lipgloss.Style

	// Focus states
	FocusedPane   lipgloss.Style
	UnfocusedPane lipgloss.Style

	// Message styles
	UserMessage   lipgloss.Style
	AgentMessage  lipgloss.Style
	SystemMessage lipgloss.Style
	ErrorMessage  lipgloss.Style

	// Status styles
	StatusInfo    lipgloss.Style
	StatusSuccess lipgloss.Style
	StatusWarning lipgloss.Style
	StatusError   lipgloss.Style

	// Interactive elements
	Button        lipgloss.Style
	ButtonFocused lipgloss.Style
	Link          lipgloss.Style

	// Code and syntax
	CodeBlock     lipgloss.Style
	InlineCode    lipgloss.Style
	SyntaxKeyword lipgloss.Style
	SyntaxString  lipgloss.Style
	SyntaxComment lipgloss.Style

	// Medieval theme elements
	Banner    lipgloss.Style
	Separator lipgloss.Style
	Highlight lipgloss.Style

	// Completion and search
	CompletionItem         lipgloss.Style
	CompletionItemSelected lipgloss.Style
	SearchMatch            lipgloss.Style
	SearchMatchCurrent     lipgloss.Style
}

// NewStyles creates a new styles collection with Guild Chat theming
func NewStyles() *Styles {
	// Color palette for Guild Chat
	var (
		guildPurple    = lipgloss.Color("141") // Primary purple
		guildOrange    = lipgloss.Color("208") // Medieval orange/amber
		guildGreen     = lipgloss.Color("82")  // Success green
		guildRed       = lipgloss.Color("196") // Error red
		guildYellow    = lipgloss.Color("226") // Warning yellow
		guildGray      = lipgloss.Color("240") // Muted gray
		guildLightGray = lipgloss.Color("254") // Light text
		guildDarkGray  = lipgloss.Color("236") // Dark backgrounds
		guildBlue      = lipgloss.Color("39")  // Links
	)

	s := &Styles{}

	// Base styles
	s.Base = lipgloss.NewStyle().
		Foreground(guildLightGray)

	s.Container = lipgloss.NewStyle().
		Padding(0, 1)

	// Pane styles
	s.OutputPane = lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(guildGray).
		Padding(0, 1)

	s.InputPane = lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(guildPurple).
		Padding(0, 1)

	s.StatusPane = lipgloss.NewStyle().
		Background(guildDarkGray).
		Foreground(guildLightGray).
		Padding(0, 1)

	// Focus states
	s.FocusedPane = lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(guildPurple).
		Bold(true)

	s.UnfocusedPane = lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(guildGray)

	// Message styles
	s.UserMessage = lipgloss.NewStyle().
		Foreground(guildPurple).
		Bold(true)

	s.AgentMessage = lipgloss.NewStyle().
		Foreground(guildGreen)

	s.SystemMessage = lipgloss.NewStyle().
		Foreground(guildGray).
		Italic(true)

	s.ErrorMessage = lipgloss.NewStyle().
		Foreground(guildRed).
		Bold(true)

	// Status styles
	s.StatusInfo = lipgloss.NewStyle().
		Foreground(guildPurple)

	s.StatusSuccess = lipgloss.NewStyle().
		Foreground(guildGreen)

	s.StatusWarning = lipgloss.NewStyle().
		Foreground(guildYellow)

	s.StatusError = lipgloss.NewStyle().
		Foreground(guildRed)

	// Interactive elements
	s.Button = lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(guildGray).
		Padding(0, 2).
		Foreground(guildLightGray)

	s.ButtonFocused = s.Button.Copy().
		BorderForeground(guildPurple).
		Background(guildPurple).
		Foreground(lipgloss.Color("15")). // White text
		Bold(true)

	s.Link = lipgloss.NewStyle().
		Foreground(guildBlue).
		Underline(true)

	// Code and syntax
	s.CodeBlock = lipgloss.NewStyle().
		Background(guildDarkGray).
		Foreground(guildLightGray).
		Padding(1, 2).
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(guildGray)

	s.InlineCode = lipgloss.NewStyle().
		Background(guildDarkGray).
		Foreground(guildOrange).
		Padding(0, 1)

	s.SyntaxKeyword = lipgloss.NewStyle().
		Foreground(guildPurple).
		Bold(true)

	s.SyntaxString = lipgloss.NewStyle().
		Foreground(guildGreen)

	s.SyntaxComment = lipgloss.NewStyle().
		Foreground(guildGray).
		Italic(true)

	// Medieval theme elements
	s.Banner = lipgloss.NewStyle().
		Foreground(guildOrange).
		Bold(true).
		Align(lipgloss.Center)

	s.Separator = lipgloss.NewStyle().
		Foreground(guildGray)

	s.Highlight = lipgloss.NewStyle().
		Background(guildOrange).
		Foreground(lipgloss.Color("0")). // Black text
		Bold(true)

	// Completion and search
	s.CompletionItem = lipgloss.NewStyle().
		Foreground(guildLightGray).
		Padding(0, 1)

	s.CompletionItemSelected = lipgloss.NewStyle().
		Background(guildPurple).
		Foreground(lipgloss.Color("15")). // White text
		Bold(true).
		Padding(0, 1)

	s.SearchMatch = lipgloss.NewStyle().
		Background(guildYellow).
		Foreground(lipgloss.Color("0")) // Black text

	s.SearchMatchCurrent = lipgloss.NewStyle().
		Background(guildOrange).
		Foreground(lipgloss.Color("0")). // Black text
		Bold(true)

	return s
}

// NewMedievalStyles creates styles with a stronger medieval theme
func NewMedievalStyles() *Styles {
	styles := NewStyles()

	// Override with more medieval-themed colors
	medievalGold := lipgloss.Color("220")
	medievalBronze := lipgloss.Color("172")
	medievalMaroon := lipgloss.Color("88")

	// Update key styles for medieval theme
	styles.Banner = styles.Banner.Copy().
		Foreground(medievalGold)

	styles.Highlight = styles.Highlight.Copy().
		Background(medievalGold)

	styles.UserMessage = styles.UserMessage.Copy().
		Foreground(medievalBronze)

	styles.FocusedPane = styles.FocusedPane.Copy().
		BorderForeground(medievalGold)

	styles.InputPane = styles.InputPane.Copy().
		BorderForeground(medievalGold)

	styles.StatusError = styles.StatusError.Copy().
		Foreground(medievalMaroon)

	return styles
}

// NewMinimalStyles creates styles with minimal theming
func NewMinimalStyles() *Styles {
	styles := NewStyles()

	// Gray scale colors for minimal theme
	lightGray := lipgloss.Color("254")
	mediumGray := lipgloss.Color("245")
	darkGray := lipgloss.Color("240")

	// Override with minimal colors
	styles.UserMessage = styles.UserMessage.Copy().
		Foreground(lightGray).
		Bold(false)

	styles.AgentMessage = styles.AgentMessage.Copy().
		Foreground(lightGray)

	styles.SystemMessage = styles.SystemMessage.Copy().
		Foreground(mediumGray)

	styles.FocusedPane = styles.FocusedPane.Copy().
		BorderForeground(lightGray)

	styles.InputPane = styles.InputPane.Copy().
		BorderForeground(mediumGray)

	styles.OutputPane = styles.OutputPane.Copy().
		BorderForeground(darkGray)

	return styles
}

// ApplyTheme applies a theme to existing styles
func (s *Styles) ApplyTheme(theme string) {
	switch theme {
	case "medieval":
		*s = *NewMedievalStyles()
	case "minimal":
		*s = *NewMinimalStyles()
	case "default":
		*s = *NewStyles()
	}
}

// GetMessageStyle returns the appropriate style for a message type
func (s *Styles) GetMessageStyle(messageType string) lipgloss.Style {
	switch messageType {
	case "user":
		return s.UserMessage
	case "agent":
		return s.AgentMessage
	case "system":
		return s.SystemMessage
	case "error":
		return s.ErrorMessage
	default:
		return s.Base
	}
}

// GetStatusStyle returns the appropriate style for a status level
func (s *Styles) GetStatusStyle(level string) lipgloss.Style {
	switch level {
	case "info":
		return s.StatusInfo
	case "success":
		return s.StatusSuccess
	case "warning":
		return s.StatusWarning
	case "error":
		return s.StatusError
	default:
		return s.StatusInfo
	}
}

// GetPaneStyle returns the appropriate style for a pane based on focus state
func (s *Styles) GetPaneStyle(paneType string, focused bool) lipgloss.Style {
	var baseStyle lipgloss.Style

	switch paneType {
	case "output":
		baseStyle = s.OutputPane
	case "input":
		baseStyle = s.InputPane
	case "status":
		baseStyle = s.StatusPane
	default:
		baseStyle = s.Base
	}

	if focused {
		return s.FocusedPane.Copy().Inherit(baseStyle)
	}

	return baseStyle
}

// CreateProgressBar creates a styled progress bar
func (s *Styles) CreateProgressBar(progress float64, width int) string {
	if width < 3 {
		return ""
	}

	filled := int(progress * float64(width-2)) // Account for brackets
	if filled < 0 {
		filled = 0
	} else if filled > width-2 {
		filled = width - 2
	}

	bar := "["
	for i := 0; i < width-2; i++ {
		if i < filled {
			bar += "█"
		} else {
			bar += "░"
		}
	}
	bar += "]"

	return s.Highlight.Render(bar)
}

// CreateSeparator creates a styled separator line
func (s *Styles) CreateSeparator(width int, char string) string {
	if char == "" {
		char = "─"
	}

	line := ""
	for i := 0; i < width; i++ {
		line += char
	}

	return s.Separator.Render(line)
}

// CreateBanner creates a styled banner with text
func (s *Styles) CreateBanner(text string, width int) string {
	return s.Banner.Width(width).Render(text)
}

// CreateBox creates a styled box around content
func (s *Styles) CreateBox(content string, title string) string {
	style := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("141")). // Use purple as default
		Padding(1, 2)

	// Note: BorderTitle is not available in this version of lipgloss
	// In a real implementation, you'd add the title manually or upgrade lipgloss

	return style.Render(content)
}

// CreateList creates a styled list
func (s *Styles) CreateList(items []string, numbered bool) string {
	var result string

	for i, item := range items {
		prefix := "• "
		if numbered {
			prefix = fmt.Sprintf("%d. ", i+1)
		}

		line := s.Base.Render(prefix + item)
		if i > 0 {
			result += "\n"
		}
		result += line
	}

	return result
}

// FormatTimestamp formats a timestamp with styling
func (s *Styles) FormatTimestamp(timestamp string) string {
	return s.SystemMessage.Render("[" + timestamp + "]")
}

// FormatAgentName formats an agent name with styling
func (s *Styles) FormatAgentName(name string) string {
	return s.AgentMessage.Bold(true).Render(name)
}

// FormatCommand formats a command with styling
func (s *Styles) FormatCommand(command string) string {
	return s.InlineCode.Render("/" + command)
}

// FormatError formats an error message with styling
func (s *Styles) FormatError(message string) string {
	return s.ErrorMessage.Render("❌ " + message)
}

// FormatSuccess formats a success message with styling
func (s *Styles) FormatSuccess(message string) string {
	return s.StatusSuccess.Render("✅ " + message)
}

// FormatWarning formats a warning message with styling
func (s *Styles) FormatWarning(message string) string {
	return s.StatusWarning.Render("⚠️ " + message)
}

// FormatInfo formats an info message with styling
func (s *Styles) FormatInfo(message string) string {
	return s.StatusInfo.Render("ℹ️ " + message)
}
