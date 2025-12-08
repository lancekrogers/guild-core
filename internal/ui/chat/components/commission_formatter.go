// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package components

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/guild-framework/guild-core/pkg/commission"
	"github.com/guild-framework/guild-core/pkg/gerror"
)

// CommissionFormatter provides rich formatting for commission documents
type CommissionFormatter struct {
	styles   CommissionStyles
	maxWidth int
	ctx      context.Context
}

// CommissionStyles defines styling for different commission elements
type CommissionStyles struct {
	Header      lipgloss.Style
	Subtitle    lipgloss.Style
	Box         lipgloss.Style
	Metadata    lipgloss.Style
	Requirement lipgloss.Style
	Technology  lipgloss.Style
	Status      lipgloss.Style
	Warning     lipgloss.Style
	Success     lipgloss.Style
	Separator   lipgloss.Style
	Highlight   lipgloss.Style
}

// NewCommissionFormatter creates a new commission formatter
func NewCommissionFormatter(ctx context.Context, maxWidth int) (*CommissionFormatter, error) {
	if ctx == nil {
		return nil, gerror.New(gerror.ErrCodeInvalidInput, "context cannot be nil", nil).
			WithComponent("components.commission_formatter").
			WithOperation("NewCommissionFormatter")
	}

	if maxWidth < 40 {
		maxWidth = 40 // Minimum reasonable width
	}

	return &CommissionFormatter{
		styles:   createCommissionStyles(),
		maxWidth: maxWidth,
		ctx:      ctx,
	}, nil
}

// createCommissionStyles creates the default styling for commission formatting
func createCommissionStyles() CommissionStyles {
	return CommissionStyles{
		Header: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("141")). // Purple
			MarginBottom(1),

		Subtitle: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("208")). // Orange
			MarginTop(1).
			MarginBottom(1),

		Box: lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240")). // Gray
			Padding(1, 2).
			MarginBottom(1),

		Metadata: lipgloss.NewStyle().
			Background(lipgloss.Color("236")). // Dark gray
			Foreground(lipgloss.Color("254")). // Light gray
			Padding(1, 2).
			MarginBottom(1),

		Requirement: lipgloss.NewStyle().
			Foreground(lipgloss.Color("82")). // Green
			PaddingLeft(2),

		Technology: lipgloss.NewStyle().
			Foreground(lipgloss.Color("111")). // Blue
			PaddingLeft(2),

		Status: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("226")), // Yellow

		Warning: lipgloss.NewStyle().
			Foreground(lipgloss.Color("226")). // Yellow
			Bold(true),

		Success: lipgloss.NewStyle().
			Foreground(lipgloss.Color("82")). // Green
			Bold(true),

		Separator: lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")). // Gray
			MarginTop(1).
			MarginBottom(1),

		Highlight: lipgloss.NewStyle().
			Background(lipgloss.Color("208")). // Orange
			Foreground(lipgloss.Color("0")).   // Black
			Bold(true).
			Padding(0, 1),
	}
}

// FormatDraft formats a commission draft for display
func (cf *CommissionFormatter) FormatDraft(commission *commission.Commission) (string, error) {
	if err := cf.ctx.Err(); err != nil {
		return "", gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("components.commission_formatter").
			WithOperation("FormatDraft")
	}

	if commission == nil {
		return "", gerror.New(gerror.ErrCodeInvalidInput, "commission cannot be nil", nil).
			WithComponent("components.commission_formatter").
			WithOperation("FormatDraft")
	}

	var b strings.Builder

	// Header with icon
	header := cf.styles.Header.Render("📋 Commission Draft")
	b.WriteString(header + "\n")

	// Separator line
	separator := cf.createSeparatorLine(cf.maxWidth - 4)
	b.WriteString(cf.styles.Separator.Render(separator) + "\n\n")

	// Metadata box
	metadataContent := cf.formatMetadata(commission)
	metaBox := cf.styles.Metadata.Width(cf.maxWidth - 8).Render(metadataContent)
	b.WriteString(metaBox + "\n\n")

	// Commission content sections
	if commission.Title != "" {
		b.WriteString(cf.styles.Subtitle.Render("📌 Title") + "\n")
		b.WriteString(cf.wrapText(commission.Title, cf.maxWidth-4) + "\n\n")
	}

	if commission.Description != "" {
		b.WriteString(cf.styles.Subtitle.Render("📝 Description") + "\n")
		b.WriteString(cf.wrapText(commission.Description, cf.maxWidth-4) + "\n\n")
	}

	// Requirements section with checkmarks
	if len(commission.Requirements) > 0 {
		b.WriteString(cf.styles.Subtitle.Render("✅ Requirements") + "\n")
		for _, req := range commission.Requirements {
			reqLine := cf.styles.Requirement.Render(fmt.Sprintf("  ✓ %s", req))
			b.WriteString(reqLine + "\n")
		}
		b.WriteString("\n")
	}

	// Tasks section
	if len(commission.Tasks) > 0 {
		b.WriteString(cf.styles.Subtitle.Render("📋 Tasks") + "\n")
		for _, task := range commission.Tasks {
			status := "○"
			if task.Status == "done" {
				status = "✓"
			} else if task.Status == "in_progress" {
				status = "▶"
			}
			taskLine := cf.styles.Technology.Render(fmt.Sprintf("  %s %s", status, task.Title))
			b.WriteString(taskLine + "\n")
		}
		b.WriteString("\n")
	}

	// Metadata information
	if len(commission.Metadata) > 0 {
		b.WriteString(cf.styles.Subtitle.Render("📊 Additional Info") + "\n")
		metadataInfo := cf.formatCommissionMetadata(commission)
		b.WriteString(metadataInfo + "\n\n")
	}

	// Footer with action options
	b.WriteString(cf.formatActionFooter())

	return b.String(), nil
}

// FormatSummary formats a brief commission summary
func (cf *CommissionFormatter) FormatSummary(commission *commission.Commission) (string, error) {
	if err := cf.ctx.Err(); err != nil {
		return "", gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("components.commission_formatter").
			WithOperation("FormatSummary")
	}

	if commission == nil {
		return "", gerror.New(gerror.ErrCodeInvalidInput, "commission cannot be nil", nil).
			WithComponent("components.commission_formatter").
			WithOperation("FormatSummary")
	}

	var b strings.Builder

	// Compact header
	title := commission.Title
	if title == "" {
		title = commission.ID
	}

	header := cf.styles.Header.Render(fmt.Sprintf("📋 %s", title))
	b.WriteString(header + "\n")

	// Key info in one line
	var keyInfo []string
	if commission.Priority != "" {
		keyInfo = append(keyInfo, fmt.Sprintf("Priority: %s", commission.Priority))
	}
	if len(commission.Requirements) > 0 {
		keyInfo = append(keyInfo, fmt.Sprintf("%d requirements", len(commission.Requirements)))
	}
	if len(commission.Tasks) > 0 {
		keyInfo = append(keyInfo, fmt.Sprintf("%d tasks", len(commission.Tasks)))
	}

	if len(keyInfo) > 0 {
		info := strings.Join(keyInfo, " | ")
		b.WriteString(cf.styles.Metadata.Render(info) + "\n")
	}

	// Brief description
	if commission.Description != "" {
		description := commission.Description
		if len(description) > 100 {
			description = description[:97] + "..."
		}
		b.WriteString(cf.wrapText(description, cf.maxWidth-4) + "\n")
	}

	return b.String(), nil
}

// FormatProgress formats commission progress information
func (cf *CommissionFormatter) FormatProgress(commission *commission.Commission, progress float64, stage string) (string, error) {
	if err := cf.ctx.Err(); err != nil {
		return "", gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("components.commission_formatter").
			WithOperation("FormatProgress")
	}

	var b strings.Builder

	// Progress header
	header := cf.styles.Header.Render(fmt.Sprintf("📋 %s - Progress", commission.Title))
	b.WriteString(header + "\n")

	// Progress bar
	progressBar := cf.createProgressBar(progress, cf.maxWidth-4)
	b.WriteString(progressBar + "\n")

	// Current stage
	stageText := cf.styles.Status.Render(fmt.Sprintf("Current Stage: %s", stage))
	b.WriteString(stageText + "\n")

	// Completion percentage
	percentage := fmt.Sprintf("%.0f%% Complete", progress*100)
	b.WriteString(cf.styles.Success.Render(percentage) + "\n")

	return b.String(), nil
}

// formatMetadata formats commission metadata into a readable string
func (cf *CommissionFormatter) formatMetadata(commission *commission.Commission) string {
	var parts []string

	if commission.ID != "" {
		parts = append(parts, fmt.Sprintf("ID: %s", commission.ID))
	}

	if commission.Priority != "" {
		parts = append(parts, fmt.Sprintf("Priority: %s", commission.Priority))
	}

	if !commission.CreatedAt.IsZero() {
		created := commission.CreatedAt.Format("Jan 2, 2006 15:04")
		parts = append(parts, fmt.Sprintf("Created: %s", created))
	}

	// Add completion percentage
	if commission.Completion > 0 {
		parts = append(parts, fmt.Sprintf("Completion: %.0f%%", commission.Completion*100))
	}

	// Add complexity estimate if available
	if commission.Metadata != nil {
		if complexity, exists := commission.Metadata["complexity"]; exists {
			parts = append(parts, fmt.Sprintf("Complexity: %v", complexity))
		}
	}

	return strings.Join(parts, "\n")
}

// formatCommissionMetadata formats commission metadata information
func (cf *CommissionFormatter) formatCommissionMetadata(commission *commission.Commission) string {
	var parts []string

	if commission.Owner != "" {
		parts = append(parts, fmt.Sprintf("Owner: %s", commission.Owner))
	}

	if len(commission.Assignees) > 0 {
		assignees := strings.Join(commission.Assignees, ", ")
		parts = append(parts, fmt.Sprintf("Assignees: %s", assignees))
	}

	if len(commission.Tags) > 0 {
		tags := strings.Join(commission.Tags, ", ")
		parts = append(parts, fmt.Sprintf("Tags: %s", tags))
	}

	if commission.CampaignID != "" {
		parts = append(parts, fmt.Sprintf("Campaign: %s", commission.CampaignID))
	}

	result := strings.Join(parts, " | ")
	return cf.wrapText(result, cf.maxWidth-4)
}

// formatActionFooter formats the action options footer
func (cf *CommissionFormatter) formatActionFooter() string {
	var b strings.Builder

	// Separator
	separator := cf.createSeparatorLine(cf.maxWidth - 4)
	b.WriteString(cf.styles.Separator.Render(separator) + "\n")

	// Action options
	b.WriteString(cf.styles.Subtitle.Render("💡 Review Options") + "\n")

	options := []string{
		"✅ Type 'yes' or 'save' to save this commission",
		"✏️  Type 'edit' to modify sections",
		"🔄 Type 'refine' to trigger refinement",
		"❌ Type 'cancel' to abort",
	}

	for _, option := range options {
		b.WriteString(cf.styles.Requirement.Render(fmt.Sprintf("  %s", option)) + "\n")
	}

	return b.String()
}

// createProgressBar creates a visual progress bar
func (cf *CommissionFormatter) createProgressBar(progress float64, width int) string {
	if width < 10 {
		width = 10
	}

	barWidth := width - 10 // Account for brackets and percentage
	filled := int(progress * float64(barWidth))
	if filled < 0 {
		filled = 0
	} else if filled > barWidth {
		filled = barWidth
	}

	bar := "["
	for i := 0; i < barWidth; i++ {
		if i < filled {
			bar += "█"
		} else {
			bar += "░"
		}
	}
	bar += fmt.Sprintf("] %.0f%%", progress*100)

	return cf.styles.Highlight.Render(bar)
}

// createSeparatorLine creates a separator line of specified width
func (cf *CommissionFormatter) createSeparatorLine(width int) string {
	if width < 1 {
		width = 1
	}
	return strings.Repeat("─", width)
}

// wrapText wraps text to fit within the specified width
func (cf *CommissionFormatter) wrapText(text string, width int) string {
	if len(text) <= width {
		return text
	}

	var lines []string
	words := strings.Fields(text)
	currentLine := ""

	for _, word := range words {
		if len(currentLine)+len(word)+1 <= width {
			if currentLine == "" {
				currentLine = word
			} else {
				currentLine += " " + word
			}
		} else {
			if currentLine != "" {
				lines = append(lines, currentLine)
			}
			currentLine = word
		}
	}

	if currentLine != "" {
		lines = append(lines, currentLine)
	}

	return strings.Join(lines, "\n")
}

// SetWidth updates the maximum width for formatting
func (cf *CommissionFormatter) SetWidth(width int) {
	if width >= 40 {
		cf.maxWidth = width
	}
}

// SetTheme applies a different visual theme
func (cf *CommissionFormatter) SetTheme(theme string) {
	switch theme {
	case "medieval":
		cf.styles = createMedievalCommissionStyles()
	case "minimal":
		cf.styles = createMinimalCommissionStyles()
	default:
		cf.styles = createCommissionStyles()
	}
}

// createMedievalCommissionStyles creates medieval-themed styles
func createMedievalCommissionStyles() CommissionStyles {
	styles := createCommissionStyles()

	// Override with medieval colors
	medievalGold := lipgloss.Color("220")
	medievalBronze := lipgloss.Color("172")

	styles.Header = styles.Header.Foreground(medievalGold)
	styles.Subtitle = styles.Subtitle.Foreground(medievalBronze)
	styles.Highlight = styles.Highlight.Background(medievalGold)

	return styles
}

// createMinimalCommissionStyles creates minimal-themed styles
func createMinimalCommissionStyles() CommissionStyles {
	styles := createCommissionStyles()

	// Override with minimal colors
	lightGray := lipgloss.Color("254")
	mediumGray := lipgloss.Color("245")

	styles.Header = styles.Header.Foreground(lightGray)
	styles.Subtitle = styles.Subtitle.Foreground(mediumGray)
	styles.Requirement = styles.Requirement.Foreground(lightGray)
	styles.Technology = styles.Technology.Foreground(mediumGray)

	return styles
}
