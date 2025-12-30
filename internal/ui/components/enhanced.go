// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

// Package components provides enhanced premium UI components for Guild Framework
//
// This package implements the enhanced component library requirements identified in performance optimization,
// Agent 1 task, providing:
//   - Premium UI components with Claude Code visual parity
//   - Seamless integration with theme system and animations
//   - Agent-specific styling and identification
//   - Responsive and accessible component design
//
// The package follows Guild's architectural patterns:
//   - Context-first error handling with gerror
//   - Interface-driven design for testability
//   - Theme integration for consistent styling
//   - Animation integration for smooth interactions
//
// Example usage:
//
//	// Create component library
//	library := NewComponentLibrary(themeManager, animator)
//
//	// Render enhanced button
//	button := Button{
//		Text:    "Create Commission",
//		Variant: ButtonPrimary,
//		Size:    ButtonSizeMedium,
//		OnClick: handleCreateCommission,
//	}
//	rendered := library.RenderButton(ctx, button)
//
//	// Render agent badge
//	badge := AgentBadge{
//		AgentID:  "agent-1",
//		Status:   AgentOnline,
//		ShowName: true,
//		Animated: true,
//	}
//	badgeView := library.RenderAgentBadge(ctx, badge)
package components

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/guild-framework/guild-core/internal/ui"
	"github.com/guild-framework/guild-core/internal/ui/animation"
	"github.com/guild-framework/guild-core/internal/ui/theme"
	"github.com/guild-framework/guild-core/pkg/gerror"
	"go.uber.org/zap"
)

// Package version for compatibility tracking
const (
	Version     = "1.0.0"
	APIVersion  = "v1"
	PackageName = "enhanced-components"
)

// ComponentLibrary provides enhanced UI components with seamless integration
type ComponentLibrary struct {
	themeManager *theme.ThemeManager
	animator     *animation.Animator
	logger       *zap.Logger
	mu           sync.RWMutex
}

// NewComponentLibrary creates a new enhanced component library
func NewComponentLibrary(themeManager *theme.ThemeManager, animator *animation.Animator) *ComponentLibrary {
	logger, _ := zap.NewDevelopment()

	return &ComponentLibrary{
		themeManager: themeManager,
		animator:     animator,
		logger:       logger.Named("component-library"),
	}
}

// Button represents an enhanced button component
type Button struct {
	Text     string        `json:"text"`
	Variant  ButtonVariant `json:"variant"`
	Size     ButtonSize    `json:"size"`
	State    ButtonState   `json:"state"`
	Icon     string        `json:"icon,omitempty"`
	OnClick  tea.Cmd       `json:"-"`
	Disabled bool          `json:"disabled"`
	Loading  bool          `json:"loading"`
	Width    int           `json:"width,omitempty"`
}

// ButtonVariant defines button styling variants
type ButtonVariant int

const (
	ButtonPrimary ButtonVariant = iota
	ButtonSecondary
	ButtonAccent
	ButtonSuccess
	ButtonWarning
	ButtonDanger
	ButtonGhost
	ButtonLink
)

// ButtonSize defines button size variants
type ButtonSize int

const (
	ButtonSizeSmall ButtonSize = iota
	ButtonSizeMedium
	ButtonSizeLarge
	ButtonSizeXLarge
)

// ButtonState defines button interaction states
type ButtonState int

const (
	ButtonStateNormal ButtonState = iota
	ButtonStateHover
	ButtonStateActive
	ButtonStateFocus
	ButtonStatePressed
)

// RenderButton renders an enhanced button with animations and theming
func (cl *ComponentLibrary) RenderButton(ctx context.Context, button Button) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("component-library").
			WithOperation("RenderButton")
	}

	cl.mu.RLock()
	defer cl.mu.RUnlock()

	// Get theme-aware base style
	baseStyle := cl.getButtonBaseStyle(button.Variant)

	// Apply size modifications
	baseStyle = cl.applyButtonSize(baseStyle, button.Size)

	// Apply state modifications
	baseStyle = cl.applyButtonState(baseStyle, button.State, button.Disabled)

	// Handle width if specified
	if button.Width > 0 {
		baseStyle = baseStyle.Width(button.Width)
	}

	// Prepare content
	content := cl.prepareButtonContent(button)

	// Apply loading or disabled styling
	if button.Loading {
		content = cl.addLoadingIndicator(content)
	}

	if button.Disabled {
		baseStyle = baseStyle.Foreground(lipgloss.Color("#666666"))
	}

	return baseStyle.Render(content), nil
}

// Modal represents an enhanced modal component
type Modal struct {
	Title       string         `json:"title"`
	Content     string         `json:"content"`
	Width       int            `json:"width"`
	Height      int            `json:"height"`
	Closable    bool           `json:"closable"`
	Backdrop    bool           `json:"backdrop"`
	Animation   ModalAnimation `json:"animation"`
	Buttons     []Button       `json:"buttons"`
	OnClose     tea.Cmd        `json:"-"`
	CustomClass string         `json:"custom_class,omitempty"`
}

// ModalAnimation defines modal animation types
type ModalAnimation int

const (
	ModalFadeIn ModalAnimation = iota
	ModalSlideIn
	ModalZoomIn
	ModalSlideUp
	ModalElastic
)

// RenderModal renders an enhanced modal with backdrop and animations
func (cl *ComponentLibrary) RenderModal(ctx context.Context, modal Modal) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("component-library").
			WithOperation("RenderModal")
	}

	cl.mu.RLock()
	defer cl.mu.RUnlock()

	theme := cl.themeManager.GetCurrentTheme()
	if theme == nil {
		return "", gerror.New(ui.ErrCodeUIThemeNotFound, "no theme available", nil).
			WithComponent("component-library").
			WithOperation("RenderModal")
	}

	// Create modal container
	containerStyle := lipgloss.NewStyle().
		Width(modal.Width).
		Height(modal.Height).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(theme.Colors.Border.Base)).
		Background(lipgloss.Color(theme.Colors.Surface.Base)).
		Padding(1)

	// Create title bar
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color(theme.Colors.Text.Primary)).
		Background(lipgloss.Color(theme.Colors.Primary.Base)).
		Padding(0, 1).
		Width(modal.Width - 2)

	// Create content area
	contentStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(theme.Colors.Text.Primary)).
		Padding(1).
		Width(modal.Width - 4).
		Height(modal.Height - 6) // Account for title and buttons

	// Create button row if buttons exist
	buttonRow := ""
	if len(modal.Buttons) > 0 {
		buttons := make([]string, len(modal.Buttons))
		for i, btn := range modal.Buttons {
			rendered, err := cl.RenderButton(ctx, btn)
			if err != nil {
				cl.logger.Warn("Failed to render modal button", zap.Error(err))
				continue
			}
			buttons[i] = rendered
		}

		buttonRowStyle := lipgloss.NewStyle().
			Width(modal.Width - 2).
			Align(lipgloss.Right).
			MarginTop(1)

		buttonRow = buttonRowStyle.Render(strings.Join(buttons, " "))
	}

	// Close button if closable
	closeButton := ""
	if modal.Closable {
		closeStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color(theme.Colors.Text.Secondary)).
			Align(lipgloss.Right)
		closeButton = closeStyle.Render("✕")
	}

	// Assemble modal content
	modalContent := lipgloss.JoinVertical(lipgloss.Left,
		titleStyle.Render(modal.Title+closeButton),
		contentStyle.Render(modal.Content),
		buttonRow,
	)

	modalContainer := containerStyle.Render(modalContent)

	// Add backdrop if enabled
	if modal.Backdrop {
		backdropStyle := lipgloss.NewStyle().
			Width(100). // Would be terminal width in real implementation
			Height(30). // Would be terminal height in real implementation
			Background(lipgloss.Color("#000000")).
			AlignHorizontal(lipgloss.Center).
			AlignVertical(lipgloss.Center)

		return backdropStyle.Render(modalContainer), nil
	}

	return modalContainer, nil
}

// AgentBadge represents an agent identification badge
type AgentBadge struct {
	AgentID    string      `json:"agent_id"`
	Status     AgentStatus `json:"status"`
	Size       BadgeSize   `json:"size"`
	ShowName   bool        `json:"show_name"`
	ShowStatus bool        `json:"show_status"`
	Animated   bool        `json:"animated"`
	Tooltip    string      `json:"tooltip,omitempty"`
}

// AgentStatus defines agent status states
type AgentStatus int

const (
	AgentOnline AgentStatus = iota
	AgentBusy
	AgentOffline
	AgentThinking
	AgentError
)

// BadgeSize defines badge size variants
type BadgeSize int

const (
	BadgeSizeSmall BadgeSize = iota
	BadgeSizeMedium
	BadgeSizeLarge
)

// RenderAgentBadge renders an agent identification badge
func (cl *ComponentLibrary) RenderAgentBadge(ctx context.Context, badge AgentBadge) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("component-library").
			WithOperation("RenderAgentBadge")
	}

	cl.mu.RLock()
	defer cl.mu.RUnlock()

	theme := cl.themeManager.GetCurrentTheme()
	if theme == nil {
		return "", gerror.New(gerror.ErrCodeInternal, "no theme available", nil).
			WithComponent("component-library").
			WithOperation("RenderAgentBadge")
	}

	// Get agent-specific color
	agentColor, exists := theme.Colors.AgentColors[badge.AgentID]
	if !exists {
		agentColor = theme.Colors.Primary
	}

	// Base badge style
	badgeStyle := lipgloss.NewStyle().
		Background(lipgloss.Color(agentColor.Base)).
		Foreground(lipgloss.Color(agentColor.Inverse)).
		Padding(0, 1)

	// Apply size modifications
	switch badge.Size {
	case BadgeSizeSmall:
		badgeStyle = badgeStyle.Padding(0).Width(3)
	case BadgeSizeLarge:
		badgeStyle = badgeStyle.Padding(1, 2)
	}

	// Get status indicator
	statusIndicator := cl.getAgentStatusIndicator(badge.Status, badge.Animated)

	// Prepare content
	content := statusIndicator
	if badge.ShowName {
		agentName := cl.formatAgentName(badge.AgentID)
		content += " " + agentName
	}
	if badge.ShowStatus && badge.Status != AgentOnline {
		statusText := cl.getStatusText(badge.Status)
		content += " " + statusText
	}

	return badgeStyle.Render(content), nil
}

// ProgressBar represents an enhanced progress indicator
type ProgressBar struct {
	Progress    float64       `json:"progress"` // 0.0 to 1.0
	Width       int           `json:"width"`
	Height      int           `json:"height"`
	ShowPercent bool          `json:"show_percent"`
	ShowLabel   bool          `json:"show_label"`
	Label       string        `json:"label,omitempty"`
	Animated    bool          `json:"animated"`
	Style       ProgressStyle `json:"style"`
}

// ProgressStyle defines progress bar styling variants
type ProgressStyle int

const (
	ProgressStyleBar ProgressStyle = iota
	ProgressCircle
	ProgressRing
	ProgressDots
)

// RenderProgressBar renders an enhanced progress indicator
func (cl *ComponentLibrary) RenderProgressBar(ctx context.Context, progress ProgressBar) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("component-library").
			WithOperation("RenderProgressBar")
	}

	cl.mu.RLock()
	defer cl.mu.RUnlock()

	theme := cl.themeManager.GetCurrentTheme()
	if theme == nil {
		return "", gerror.New(gerror.ErrCodeInternal, "no theme available", nil).
			WithComponent("component-library").
			WithOperation("RenderProgressBar")
	}

	switch progress.Style {
	case ProgressStyleBar:
		return cl.renderLinearProgress(progress, theme), nil
	case ProgressCircle:
		return cl.renderCircularProgress(progress, theme), nil
	case ProgressRing:
		return cl.renderRingProgress(progress, theme), nil
	case ProgressDots:
		return cl.renderDotProgress(progress, theme), nil
	default:
		return cl.renderLinearProgress(progress, theme), nil
	}
}

// ChatMessage represents an enhanced chat message component
type ChatMessage struct {
	Content   string      `json:"content"`
	AgentID   string      `json:"agent_id"`
	Timestamp time.Time   `json:"timestamp"`
	Type      MessageType `json:"type"`
	Reactions []Reaction  `json:"reactions"`
	Metadata  MessageMeta `json:"metadata"`
	Animated  bool        `json:"animated"`
}

// MessageType defines message types
type MessageType int

const (
	MessageUser MessageType = iota
	MessageAgent
	MessageSystem
	MessageError
	MessageSuccess
	MessageWarning
	MessageInfo
)

// Reaction represents a message reaction
type Reaction struct {
	Emoji  string   `json:"emoji"`
	Count  int      `json:"count"`
	Users  []string `json:"users"`
	Active bool     `json:"active"` // If current user reacted
}

// MessageMeta contains message metadata
type MessageMeta struct {
	Edited   bool      `json:"edited"`
	EditedAt time.Time `json:"edited_at,omitempty"`
	ThreadID string    `json:"thread_id,omitempty"`
	ReplyTo  string    `json:"reply_to,omitempty"`
	Mentions []string  `json:"mentions"`
	Tags     []string  `json:"tags"`
}

// InputComponent represents an enhanced input field with agent mention support
type InputComponent struct {
	ID              string   `json:"id"`
	Value           string   `json:"value"`
	Placeholder     string   `json:"placeholder"`
	Width           int      `json:"width"`
	Multiline       bool     `json:"multiline"`
	MaxLength       int      `json:"max_length"`
	ShowCharCount   bool     `json:"show_char_count"`
	AgentMentions   bool     `json:"agent_mentions"`
	Suggestions     []string `json:"suggestions"`
	ValidationError string   `json:"validation_error"`
	OnSubmit        tea.Cmd  `json:"-"`
	OnChange        tea.Cmd  `json:"-"`
	OnAgentMention  tea.Cmd  `json:"-"`
}

// LayoutContainer represents a flexible layout container
type LayoutContainer struct {
	ID        string         `json:"id"`
	Type      LayoutType     `json:"type"`
	Direction FlexDirection  `json:"direction"`
	Gap       int            `json:"gap"`
	Padding   int            `json:"padding"`
	Children  []interface{}  `json:"children"`
	Align     AlignItems     `json:"align"`
	Justify   JustifyContent `json:"justify"`
	Wrap      bool           `json:"wrap"`
}

// LayoutType defines the type of layout
type LayoutType int

const (
	LayoutFlex LayoutType = iota
	LayoutGrid
	LayoutAbsolute
)

// FlexDirection defines flex layout direction
type FlexDirection int

const (
	FlexRow FlexDirection = iota
	FlexColumn
	FlexRowReverse
	FlexColumnReverse
)

// AlignItems defines flex alignment
type AlignItems int

const (
	AlignStart AlignItems = iota
	AlignCenter
	AlignEnd
	AlignStretch
)

// JustifyContent defines flex justification
type JustifyContent int

const (
	JustifyStart JustifyContent = iota
	JustifyCenter
	JustifyEnd
	JustifyBetween
	JustifyAround
	JustifyEvenly
)

// EnhancedStatusIndicator represents an enhanced status indicator component
type EnhancedStatusIndicator struct {
	ID       string             `json:"id"`
	Status   EnhancedStatusType `json:"status"`
	Text     string             `json:"text"`
	Animated bool               `json:"animated"`
	Size     IndicatorSize      `json:"size"`
}

// EnhancedStatusType defines enhanced status indicator types
type EnhancedStatusType int

const (
	EnhancedStatusSuccess EnhancedStatusType = iota
	EnhancedStatusWarning
	EnhancedStatusError
	EnhancedStatusInfo
	EnhancedStatusPending
	EnhancedStatusOffline
)

// IndicatorSize defines status indicator sizes
type IndicatorSize int

const (
	IndicatorSmall IndicatorSize = iota
	IndicatorMedium
	IndicatorLarge
)

// ScrollableContainer provides scrolling functionality for content
type ScrollableContainer struct {
	ID             string      `json:"id"`
	Content        interface{} `json:"content"`
	Height         int         `json:"height"`
	Width          int         `json:"width"`
	ScrollPosition int         `json:"scroll_position"`
	TotalHeight    int         `json:"total_height"`
	ShowScrollbar  bool        `json:"show_scrollbar"`
	WrapContent    bool        `json:"wrap_content"`
	FocusFollows   bool        `json:"focus_follows"` // Auto-scroll to focused element
}

// AccessibilityInfo provides accessibility metadata
type AccessibilityInfo struct {
	Role              string            `json:"role"`
	Label             string            `json:"label"`
	Description       string            `json:"description"`
	AriaLive          AriaLiveMode      `json:"aria_live"`
	AriaExpanded      *bool             `json:"aria_expanded,omitempty"`
	AriaSelected      *bool             `json:"aria_selected,omitempty"`
	AriaCurrent       string            `json:"aria_current,omitempty"`
	KeyboardShortcuts map[string]string `json:"keyboard_shortcuts,omitempty"`
	TabIndex          int               `json:"tab_index"`
}

// AriaLiveMode defines how screen readers announce changes
type AriaLiveMode string

const (
	AriaLiveOff       AriaLiveMode = "off"
	AriaLivePolite    AriaLiveMode = "polite"
	AriaLiveAssertive AriaLiveMode = "assertive"
)

// FocusableComponent represents a component that can receive focus
type FocusableComponent struct {
	ID            string            `json:"id"`
	Component     interface{}       `json:"component"`
	Focusable     bool              `json:"focusable"`
	FocusOrder    int               `json:"focus_order"`
	FocusStyle    lipgloss.Style    `json:"-"`
	Accessibility AccessibilityInfo `json:"accessibility"`
	OnFocus       tea.Cmd           `json:"-"`
	OnBlur        tea.Cmd           `json:"-"`
	OnKeyPress    tea.Cmd           `json:"-"`
}

// RenderChatMessage renders an enhanced chat message with agent styling
func (cl *ComponentLibrary) RenderChatMessage(ctx context.Context, message ChatMessage) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("component-library").
			WithOperation("RenderChatMessage")
	}

	cl.mu.RLock()
	defer cl.mu.RUnlock()

	theme := cl.themeManager.GetCurrentTheme()
	if theme == nil {
		return "", gerror.New(gerror.ErrCodeInternal, "no theme available", nil).
			WithComponent("component-library").
			WithOperation("RenderChatMessage")
	}

	// Message bubble styling based on type
	bubbleStyle := cl.getMessageBubbleStyle(message.Type, theme)

	// Header with agent info and timestamp
	headerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(theme.Colors.Text.Secondary)).
		Bold(true).
		PaddingBottom(1)

	header := cl.formatMessageHeader(message, theme)

	// Content styling
	contentStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(theme.Colors.Text.Primary)).
		PaddingLeft(1).
		PaddingRight(1)

	// Reactions if any
	reactionsRow := ""
	if len(message.Reactions) > 0 {
		reactionsRow = cl.renderReactions(message.Reactions, theme)
	}

	// Metadata indicators
	metaIndicators := cl.renderMessageMetadata(message.Metadata, theme)

	// Assemble message
	messageContent := lipgloss.JoinVertical(lipgloss.Left,
		headerStyle.Render(header),
		contentStyle.Render(message.Content),
		reactionsRow,
		metaIndicators,
	)

	return bubbleStyle.Render(messageContent), nil
}

// Helper methods for component rendering

func (cl *ComponentLibrary) getButtonBaseStyle(variant ButtonVariant) lipgloss.Style {
	if cl.themeManager == nil {
		return lipgloss.NewStyle()
	}

	theme := cl.themeManager.GetCurrentTheme()
	if theme == nil {
		return lipgloss.NewStyle()
	}

	baseStyle := lipgloss.NewStyle().
		Padding(0, 2).
		Align(lipgloss.Center)

	switch variant {
	case ButtonPrimary:
		return baseStyle.
			Background(lipgloss.Color(theme.Colors.Primary.Base)).
			Foreground(lipgloss.Color(theme.Colors.Primary.Inverse)).
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color(theme.Colors.Primary.Dark))
	case ButtonSecondary:
		return baseStyle.
			Background(lipgloss.Color(theme.Colors.Surface.Base)).
			Foreground(lipgloss.Color(theme.Colors.Text.Primary)).
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color(theme.Colors.Border.Base))
	case ButtonAccent:
		return baseStyle.
			Background(lipgloss.Color(theme.Colors.Accent.Base)).
			Foreground(lipgloss.Color(theme.Colors.Accent.Inverse))
	case ButtonSuccess:
		return baseStyle.
			Background(lipgloss.Color(theme.Colors.Success.Base)).
			Foreground(lipgloss.Color(theme.Colors.Success.Inverse))
	case ButtonWarning:
		return baseStyle.
			Background(lipgloss.Color(theme.Colors.Warning.Base)).
			Foreground(lipgloss.Color(theme.Colors.Warning.Inverse))
	case ButtonDanger:
		return baseStyle.
			Background(lipgloss.Color(theme.Colors.Error.Base)).
			Foreground(lipgloss.Color(theme.Colors.Error.Inverse))
	case ButtonGhost:
		return baseStyle.
			Background(lipgloss.Color("transparent")).
			Foreground(lipgloss.Color(theme.Colors.Text.Primary)).
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color(theme.Colors.Border.Base))
	case ButtonLink:
		return baseStyle.
			Background(lipgloss.Color("transparent")).
			Foreground(lipgloss.Color(theme.Colors.Text.Link)).
			Underline(true)
	default:
		return baseStyle
	}
}

func (cl *ComponentLibrary) applyButtonSize(style lipgloss.Style, size ButtonSize) lipgloss.Style {
	switch size {
	case ButtonSizeSmall:
		return style.Padding(0, 1).Height(1)
	case ButtonSizeMedium:
		return style.Padding(0, 2).Height(2)
	case ButtonSizeLarge:
		return style.Padding(1, 3).Height(3)
	case ButtonSizeXLarge:
		return style.Padding(1, 4).Height(4)
	default:
		return style
	}
}

func (cl *ComponentLibrary) applyButtonState(style lipgloss.Style, state ButtonState, disabled bool) lipgloss.Style {
	if disabled {
		return style.Foreground(lipgloss.Color("#666666"))
	}

	switch state {
	case ButtonStateHover:
		return style.Bold(true)
	case ButtonStateActive:
		return style.Bold(true).Reverse(true)
	case ButtonStateFocus:
		return style.BorderStyle(lipgloss.DoubleBorder())
	case ButtonStatePressed:
		return style.Reverse(true)
	default:
		return style
	}
}

func (cl *ComponentLibrary) prepareButtonContent(button Button) string {
	content := button.Text

	if button.Icon != "" {
		content = button.Icon + " " + content
	}

	return content
}

func (cl *ComponentLibrary) addLoadingIndicator(content string) string {
	// Simple loading indicator - could be enhanced with animation
	frames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	frame := frames[int(time.Now().UnixNano()/100000000)%len(frames)]
	return frame + " " + content
}

func (cl *ComponentLibrary) getAgentStatusIndicator(status AgentStatus, animated bool) string {
	switch status {
	case AgentOnline:
		return "●" // Green dot
	case AgentBusy:
		if animated {
			// Could add blinking animation
			return "◐"
		}
		return "◐" // Half circle (working)
	case AgentOffline:
		return "○" // Empty circle
	case AgentThinking:
		if animated {
			// Could add thinking animation
			return "◒"
		}
		return "◒" // Animated thinking indicator
	case AgentError:
		return "✗" // Error indicator
	default:
		return "●"
	}
}

func (cl *ComponentLibrary) formatAgentName(agentID string) string {
	// Convert agent-1 to Agent 1, custom-name to Custom Name, etc.
	parts := strings.Split(agentID, "-")
	if len(parts) >= 2 {
		result := make([]string, len(parts))
		for i, part := range parts {
			if len(part) > 0 {
				result[i] = strings.ToUpper(part[:1]) + part[1:]
			}
		}
		return strings.Join(result, " ")
	}

	// Single word, capitalize first letter
	if len(agentID) > 0 {
		return strings.ToUpper(agentID[:1]) + agentID[1:]
	}
	return agentID
}

func (cl *ComponentLibrary) getStatusText(status AgentStatus) string {
	switch status {
	case AgentBusy:
		return "(busy)"
	case AgentOffline:
		return "(offline)"
	case AgentThinking:
		return "(thinking)"
	case AgentError:
		return "(error)"
	default:
		return ""
	}
}

func (cl *ComponentLibrary) renderLinearProgress(progress ProgressBar, theme *theme.Theme) string {
	if progress.Width <= 0 {
		progress.Width = 30
	}

	filled := int(progress.Progress * float64(progress.Width))
	empty := progress.Width - filled

	filledStyle := lipgloss.NewStyle().Background(lipgloss.Color(theme.Colors.Primary.Base))
	emptyStyle := lipgloss.NewStyle().Background(lipgloss.Color(theme.Colors.Surface.Dark))

	bar := filledStyle.Render(strings.Repeat("█", filled)) +
		emptyStyle.Render(strings.Repeat("░", empty))

	result := bar

	if progress.ShowPercent {
		percent := fmt.Sprintf("%.0f%%", progress.Progress*100)
		result += " " + percent
	}

	if progress.ShowLabel && progress.Label != "" {
		labelStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color(theme.Colors.Text.Secondary)).
			PaddingBottom(1)
		result = labelStyle.Render(progress.Label) + "\n" + result
	}

	return result
}

func (cl *ComponentLibrary) renderCircularProgress(progress ProgressBar, theme *theme.Theme) string {
	// Simplified circular progress using Unicode characters
	percentage := int(progress.Progress * 8)
	circles := []string{"○", "◔", "◑", "◕", "●", "●", "●", "●", "●"}

	if percentage >= len(circles) {
		percentage = len(circles) - 1
	}

	circle := circles[percentage]

	if progress.ShowPercent {
		percent := fmt.Sprintf("%.0f%%", progress.Progress*100)
		return circle + " " + percent
	}

	return circle
}

func (cl *ComponentLibrary) renderRingProgress(progress ProgressBar, theme *theme.Theme) string {
	// Ring progress using box drawing characters
	segments := 8
	filled := int(progress.Progress * float64(segments))

	ring := "◯"
	if filled >= segments/2 {
		ring = "◉"
	} else if filled > 0 {
		ring = "◐"
	}

	return ring
}

func (cl *ComponentLibrary) renderDotProgress(progress ProgressBar, theme *theme.Theme) string {
	dots := 5
	filled := int(progress.Progress * float64(dots))

	result := ""
	for i := 0; i < dots; i++ {
		if i < filled {
			result += "●"
		} else {
			result += "○"
		}
	}

	return result
}

func (cl *ComponentLibrary) getMessageBubbleStyle(msgType MessageType, theme *theme.Theme) lipgloss.Style {
	baseStyle := lipgloss.NewStyle().
		Padding(1, 2).
		MarginBottom(1)

	switch msgType {
	case MessageUser:
		return baseStyle.
			Background(lipgloss.Color(theme.Colors.Primary.Light)).
			Foreground(lipgloss.Color(theme.Colors.Primary.Inverse)).
			Align(lipgloss.Right)
	case MessageAgent:
		return baseStyle.
			Background(lipgloss.Color(theme.Colors.Surface.Base)).
			Foreground(lipgloss.Color(theme.Colors.Text.Primary)).
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color(theme.Colors.Border.Base))
	case MessageSystem:
		return baseStyle.
			Background(lipgloss.Color(theme.Colors.Info.Light)).
			Foreground(lipgloss.Color(theme.Colors.Info.Inverse))
	case MessageError:
		return baseStyle.
			Background(lipgloss.Color(theme.Colors.Error.Light)).
			Foreground(lipgloss.Color(theme.Colors.Error.Inverse))
	case MessageSuccess:
		return baseStyle.
			Background(lipgloss.Color(theme.Colors.Success.Light)).
			Foreground(lipgloss.Color(theme.Colors.Success.Inverse))
	case MessageWarning:
		return baseStyle.
			Background(lipgloss.Color(theme.Colors.Warning.Light)).
			Foreground(lipgloss.Color(theme.Colors.Warning.Inverse))
	default:
		return baseStyle
	}
}

func (cl *ComponentLibrary) formatMessageHeader(message ChatMessage, theme *theme.Theme) string {
	agentName := cl.formatAgentName(message.AgentID)
	timestamp := message.Timestamp.Format("15:04")

	nameStyle := lipgloss.NewStyle().Bold(true)
	timeStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(theme.Colors.Text.Muted))

	return nameStyle.Render(agentName) + " " + timeStyle.Render(timestamp)
}

func (cl *ComponentLibrary) renderReactions(reactions []Reaction, theme *theme.Theme) string {
	if len(reactions) == 0 {
		return ""
	}

	reactionStyle := lipgloss.NewStyle().
		Background(lipgloss.Color(theme.Colors.Surface.Light)).
		Foreground(lipgloss.Color(theme.Colors.Text.Secondary)).
		Padding(0, 1).
		MarginTop(1).
		MarginRight(1)

	var rendered []string
	for _, reaction := range reactions {
		content := reaction.Emoji
		if reaction.Count > 1 {
			content += fmt.Sprintf(" %d", reaction.Count)
		}

		style := reactionStyle
		if reaction.Active {
			style = style.Background(lipgloss.Color(theme.Colors.Primary.Light))
		}

		rendered = append(rendered, style.Render(content))
	}

	return strings.Join(rendered, "")
}

func (cl *ComponentLibrary) renderMessageMetadata(metadata MessageMeta, theme *theme.Theme) string {
	var indicators []string

	if metadata.Edited {
		editStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color(theme.Colors.Text.Muted)).
			Italic(true)
		indicators = append(indicators, editStyle.Render("(edited)"))
	}

	if metadata.ReplyTo != "" {
		replyStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color(theme.Colors.Text.Link))
		indicators = append(indicators, replyStyle.Render("↳ reply"))
	}

	if len(metadata.Mentions) > 0 {
		mentionStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color(theme.Colors.Accent.Base))
		indicators = append(indicators, mentionStyle.Render(fmt.Sprintf("@%d", len(metadata.Mentions))))
	}

	if len(indicators) > 0 {
		return " " + strings.Join(indicators, " ")
	}

	return ""
}

// RenderInputComponent renders an enhanced input field with agent mention support
func (cl *ComponentLibrary) RenderInputComponent(ctx context.Context, input InputComponent) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("component-library").
			WithOperation("RenderInputComponent")
	}

	cl.mu.RLock()
	defer cl.mu.RUnlock()

	theme := cl.themeManager.GetCurrentTheme()
	if theme == nil {
		return "", gerror.New(gerror.ErrCodeInternal, "no theme available", nil).
			WithComponent("component-library").
			WithOperation("RenderInputComponent")
	}

	// Base input styling
	inputStyle := lipgloss.NewStyle().
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color(theme.Colors.Border.Base)).
		Padding(0, 1)

	if input.Width > 0 {
		inputStyle = inputStyle.Width(input.Width)
	}

	// Validation error styling
	if input.ValidationError != "" {
		inputStyle = inputStyle.BorderForeground(lipgloss.Color(theme.Colors.Error.Base))
	}

	// Character count display
	charCount := ""
	if input.ShowCharCount && input.MaxLength > 0 {
		charCountStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color(theme.Colors.Text.Muted)).
			Align(lipgloss.Right)
		charCount = charCountStyle.Render(fmt.Sprintf("%d/%d", len(input.Value), input.MaxLength))
	}

	// Placeholder styling
	content := input.Value
	if content == "" && input.Placeholder != "" {
		placeholderStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color(theme.Colors.Text.Muted)).
			Italic(true)
		content = placeholderStyle.Render(input.Placeholder)
	}

	// Assemble input
	inputContent := inputStyle.Render(content)
	if charCount != "" {
		inputContent = lipgloss.JoinVertical(lipgloss.Left, inputContent, charCount)
	}

	if input.ValidationError != "" {
		errorStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color(theme.Colors.Error.Base)).
			PaddingTop(1)
		inputContent = lipgloss.JoinVertical(lipgloss.Left, inputContent, errorStyle.Render(input.ValidationError))
	}

	return inputContent, nil
}

// RenderLayoutContainer renders a flexible layout container
func (cl *ComponentLibrary) RenderLayoutContainer(ctx context.Context, layout LayoutContainer) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("component-library").
			WithOperation("RenderLayoutContainer")
	}

	cl.mu.RLock()
	defer cl.mu.RUnlock()

	// Base container styling
	containerStyle := lipgloss.NewStyle()
	if layout.Padding > 0 {
		containerStyle = containerStyle.Padding(layout.Padding)
	}

	// Render children
	var renderedChildren []string
	for _, child := range layout.Children {
		// Render child based on type (simplified for now)
		childStr := fmt.Sprintf("%v", child)
		renderedChildren = append(renderedChildren, childStr)
	}

	// Apply layout based on type
	var result string
	switch layout.Type {
	case LayoutFlex:
		if layout.Direction == FlexColumn || layout.Direction == FlexColumnReverse {
			result = lipgloss.JoinVertical(cl.alignToPosition(layout.Align), renderedChildren...)
		} else {
			result = lipgloss.JoinHorizontal(cl.alignToPosition(layout.Align), renderedChildren...)
		}
	case LayoutGrid:
		// Simplified grid layout
		result = strings.Join(renderedChildren, " ")
	case LayoutAbsolute:
		// Simplified absolute layout
		result = strings.Join(renderedChildren, "\n")
	}

	return containerStyle.Render(result), nil
}

// RenderStatusIndicator renders a status indicator component
func (cl *ComponentLibrary) RenderStatusIndicator(ctx context.Context, indicator EnhancedStatusIndicator) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("component-library").
			WithOperation("RenderStatusIndicator")
	}

	cl.mu.RLock()
	defer cl.mu.RUnlock()

	theme := cl.themeManager.GetCurrentTheme()
	if theme == nil {
		return "", gerror.New(gerror.ErrCodeInternal, "no theme available", nil).
			WithComponent("component-library").
			WithOperation("RenderStatusIndicator")
	}

	// Get status symbol and color
	symbol, color := cl.getStatusSymbolAndColor(indicator.Status, theme)

	// Apply size
	symbolStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(color))
	switch indicator.Size {
	case IndicatorLarge:
		symbolStyle = symbolStyle.Bold(true).PaddingRight(2)
	case IndicatorMedium:
		symbolStyle = symbolStyle.PaddingRight(1)
	case IndicatorSmall:
		// Default size
	}

	// Animate if requested
	if indicator.Animated && indicator.Status == EnhancedStatusPending {
		if cl.animator != nil {
			// Apply pulsing animation for pending status
			symbol = cl.applyPulseAnimation(symbol)
		}
	}

	// Render indicator with text
	indicatorContent := symbolStyle.Render(symbol)
	if indicator.Text != "" {
		textStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(theme.Colors.Text.Primary))
		indicatorContent = lipgloss.JoinHorizontal(lipgloss.Center, indicatorContent, textStyle.Render(indicator.Text))
	}

	return indicatorContent, nil
}

// Helper methods for new components

func (cl *ComponentLibrary) alignToPosition(align AlignItems) lipgloss.Position {
	switch align {
	case AlignStart:
		return lipgloss.Left
	case AlignCenter:
		return lipgloss.Center
	case AlignEnd:
		return lipgloss.Right
	default:
		return lipgloss.Left
	}
}

func (cl *ComponentLibrary) getStatusSymbolAndColor(status EnhancedStatusType, theme *theme.Theme) (string, string) {
	switch status {
	case EnhancedStatusSuccess:
		return "✓", theme.Colors.Success.Base
	case EnhancedStatusWarning:
		return "⚠", theme.Colors.Warning.Base
	case EnhancedStatusError:
		return "✗", theme.Colors.Error.Base
	case EnhancedStatusInfo:
		return "ℹ", theme.Colors.Info.Base
	case EnhancedStatusPending:
		return "◐", theme.Colors.Primary.Base
	case EnhancedStatusOffline:
		return "○", theme.Colors.Text.Muted
	default:
		return "•", theme.Colors.Text.Secondary
	}
}

func (cl *ComponentLibrary) applyPulseAnimation(symbol string) string {
	// Simple pulse effect by alternating opacity
	// In real implementation, this would integrate with the animator
	return symbol
}

// RenderScrollableContainer renders a scrollable container with viewport management
func (cl *ComponentLibrary) RenderScrollableContainer(ctx context.Context, container ScrollableContainer) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("component-library").
			WithOperation("RenderScrollableContainer")
	}

	cl.mu.RLock()
	defer cl.mu.RUnlock()

	theme := cl.themeManager.GetCurrentTheme()
	if theme == nil {
		return "", gerror.New(gerror.ErrCodeInternal, "no theme available", nil).
			WithComponent("component-library").
			WithOperation("RenderScrollableContainer")
	}

	// Convert content to string
	contentStr := fmt.Sprintf("%v", container.Content)
	lines := strings.Split(contentStr, "\n")

	// Calculate visible viewport
	startLine := container.ScrollPosition
	endLine := startLine + container.Height
	if endLine > len(lines) {
		endLine = len(lines)
	}
	if startLine < 0 {
		startLine = 0
	}

	// Extract visible lines
	visibleLines := lines[startLine:endLine]

	// Apply wrapping if enabled
	if container.WrapContent && container.Width > 0 {
		var wrappedLines []string
		for _, line := range visibleLines {
			wrapped := cl.wrapLine(line, container.Width)
			wrappedLines = append(wrappedLines, wrapped...)
		}
		visibleLines = wrappedLines
	}

	// Build the container
	containerStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(theme.Colors.Border.Base)).
		Height(container.Height).
		Width(container.Width).
		Padding(1)

	content := strings.Join(visibleLines, "\n")

	// Add scrollbar if enabled
	if container.ShowScrollbar && len(lines) > container.Height {
		scrollbar := cl.renderScrollbar(container.ScrollPosition, container.Height, len(lines), theme)
		content = cl.addScrollbarToContent(content, scrollbar, container.Width)
	}

	// Add scroll indicators
	if startLine > 0 {
		topIndicator := lipgloss.NewStyle().
			Foreground(lipgloss.Color(theme.Colors.Text.Muted)).
			Render(fmt.Sprintf("↑ %d more", startLine))
		content = topIndicator + "\n" + content
	}

	if endLine < len(lines) {
		bottomIndicator := lipgloss.NewStyle().
			Foreground(lipgloss.Color(theme.Colors.Text.Muted)).
			Render(fmt.Sprintf("↓ %d more", len(lines)-endLine))
		content = content + "\n" + bottomIndicator
	}

	return containerStyle.Render(content), nil
}

// RenderFocusableComponent renders a component with focus and accessibility support
func (cl *ComponentLibrary) RenderFocusableComponent(ctx context.Context, focusable FocusableComponent, hasFocus bool) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("component-library").
			WithOperation("RenderFocusableComponent")
	}

	cl.mu.RLock()
	defer cl.mu.RUnlock()

	theme := cl.themeManager.GetCurrentTheme()
	if theme == nil {
		return "", gerror.New(gerror.ErrCodeInternal, "no theme available", nil).
			WithComponent("component-library").
			WithOperation("RenderFocusableComponent")
	}

	// Render the inner component
	componentStr := fmt.Sprintf("%v", focusable.Component)

	// Apply focus styling if focused
	var style lipgloss.Style
	if hasFocus {
		// Use provided focus style or default
		if focusable.FocusStyle.String() != "" {
			style = focusable.FocusStyle
		} else {
			// Default focus style
			style = lipgloss.NewStyle().
				Border(lipgloss.ThickBorder()).
				BorderForeground(lipgloss.Color(theme.Colors.Primary.Base))
		}
	} else {
		style = lipgloss.NewStyle()
	}

	// Add accessibility indicators
	if focusable.Accessibility.Label != "" {
		// In a terminal, we can't truly implement ARIA, but we can add visual indicators
		labelStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color(theme.Colors.Text.Muted)).
			Italic(true)

		accessLabel := labelStyle.Render(fmt.Sprintf("[%s]", focusable.Accessibility.Label))
		componentStr = accessLabel + "\n" + componentStr
	}

	// Add keyboard shortcut hints
	if len(focusable.Accessibility.KeyboardShortcuts) > 0 && hasFocus {
		var shortcuts []string
		for key, action := range focusable.Accessibility.KeyboardShortcuts {
			shortcuts = append(shortcuts, fmt.Sprintf("%s: %s", key, action))
		}

		shortcutStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color(theme.Colors.Text.Secondary)).
			Italic(true)

		shortcutStr := shortcutStyle.Render(strings.Join(shortcuts, " | "))
		componentStr = componentStr + "\n" + shortcutStr
	}

	return style.Render(componentStr), nil
}

// Helper methods for scrolling and accessibility

func (cl *ComponentLibrary) wrapLine(line string, width int) []string {
	if len(line) <= width {
		return []string{line}
	}

	var wrapped []string
	for len(line) > width {
		wrapped = append(wrapped, line[:width])
		line = line[width:]
	}
	if len(line) > 0 {
		wrapped = append(wrapped, line)
	}

	return wrapped
}

func (cl *ComponentLibrary) renderScrollbar(position, viewHeight, totalHeight int, theme *theme.Theme) string {
	if totalHeight <= viewHeight {
		return ""
	}

	// Calculate scrollbar size and position
	barHeight := max(1, (viewHeight*viewHeight)/totalHeight)
	barPosition := (position * (viewHeight - barHeight)) / (totalHeight - viewHeight)

	var scrollbar []string
	for i := 0; i < viewHeight; i++ {
		if i >= barPosition && i < barPosition+barHeight {
			scrollbar = append(scrollbar, "█")
		} else {
			scrollbar = append(scrollbar, "│")
		}
	}

	scrollbarStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(theme.Colors.Border.Base))

	return scrollbarStyle.Render(strings.Join(scrollbar, "\n"))
}

func (cl *ComponentLibrary) addScrollbarToContent(content, scrollbar string, width int) string {
	contentLines := strings.Split(content, "\n")
	scrollbarLines := strings.Split(scrollbar, "\n")

	var combined []string
	for i := 0; i < len(contentLines); i++ {
		line := contentLines[i]
		if i < len(scrollbarLines) {
			// Pad content to width and add scrollbar
			padding := width - lipgloss.Width(line) - 2
			if padding < 0 {
				padding = 0
			}
			line = line + strings.Repeat(" ", padding) + " " + scrollbarLines[i]
		}
		combined = append(combined, line)
	}

	return strings.Join(combined, "\n")
}

// GetAccessibilityAnnouncement generates an announcement for screen readers
func (cl *ComponentLibrary) GetAccessibilityAnnouncement(component interface{}, change string) string {
	// In a real terminal app, this would integrate with screen reader APIs
	// For now, we return a formatted string that could be logged or announced

	switch c := component.(type) {
	case ChatMessage:
		return fmt.Sprintf("New message from %s: %s", c.AgentID, c.Content)
	case EnhancedStatusIndicator:
		statusText := cl.getStatusTextForEnhanced(c.Status)
		return fmt.Sprintf("Status changed to %s: %s", statusText, c.Text)
	case ProgressBar:
		return fmt.Sprintf("Progress: %d percent complete", int(c.Progress*100))
	default:
		return change
	}
}

func (cl *ComponentLibrary) getStatusTextForEnhanced(status EnhancedStatusType) string {
	switch status {
	case EnhancedStatusSuccess:
		return "success"
	case EnhancedStatusWarning:
		return "warning"
	case EnhancedStatusError:
		return "error"
	case EnhancedStatusInfo:
		return "information"
	case EnhancedStatusPending:
		return "pending"
	case EnhancedStatusOffline:
		return "offline"
	default:
		return "unknown"
	}
}

// max returns the maximum of two integers
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
