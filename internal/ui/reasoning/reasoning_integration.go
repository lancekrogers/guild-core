// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package reasoning

import (
	"context"
	"fmt"
	"sync"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	viewutil "github.com/guild-framework/guild-core/internal/ui/view"
	"github.com/guild-framework/guild-core/pkg/agents/core"
	"github.com/guild-framework/guild-core/pkg/gerror"
	"github.com/guild-framework/guild-core/pkg/observability"
	"github.com/guild-framework/guild-core/pkg/session"
)

// ReasoningIntegration manages the integration between reasoning and chat UI
type ReasoningIntegration struct {
	// Components
	display   *ReasoningDisplay
	chatModel tea.Model // Reference to chat model

	// Configuration
	config ReasoningIntegrationConfig

	// State management
	enabled   bool
	visible   bool
	minimized bool
	position  IntegrationPosition

	// Reasoning management
	activeChains map[string]*ActiveReasoning // messageID -> reasoning

	// Thread safety
	mu sync.RWMutex
}

// ReasoningIntegrationConfig configures the integration
type ReasoningIntegrationConfig struct {
	// Display settings
	EnableByDefault    bool `json:"enable_by_default"`
	AutoShow           bool `json:"auto_show"`
	MinimizeOnComplete bool `json:"minimize_on_complete"`
	ShowInsights       bool `json:"show_insights"`
	ShowQualityMetrics bool `json:"show_quality_metrics"`

	// Performance settings
	MaxConcurrentChains int           `json:"max_concurrent_chains"`
	StreamBufferSize    int           `json:"stream_buffer_size"`
	UpdateInterval      time.Duration `json:"update_interval"`

	// Interruption settings
	AllowInterruption bool   `json:"allow_interruption"`
	InterruptionKey   string `json:"interruption_key"` // default: "esc"

	// Layout settings
	DefaultPosition IntegrationPosition `json:"default_position"`
	MinHeight       int                 `json:"min_height"`
	MaxHeight       int                 `json:"max_height"`
}

// IntegrationPosition defines where reasoning is displayed
type IntegrationPosition string

const (
	PositionBottom  IntegrationPosition = "bottom"
	PositionRight   IntegrationPosition = "right"
	PositionOverlay IntegrationPosition = "overlay"
	PositionSplit   IntegrationPosition = "split"
)

// ActiveReasoning tracks an active reasoning process
type ActiveReasoning struct {
	MessageID string
	Streamer  *core.ReasoningStreamer
	Chain     *core.ReasoningChainEnhanced
	StartTime time.Time
	EndTime   time.Time
	Status    ReasoningStatus
	Error     error
}

// ReasoningStatus represents the status of reasoning
type ReasoningStatus string

const (
	StatusStreaming   ReasoningStatus = "streaming"
	StatusComplete    ReasoningStatus = "complete"
	StatusInterrupted ReasoningStatus = "interrupted"
	StatusError       ReasoningStatus = "error"
)

// DefaultReasoningIntegrationConfig returns default configuration
func DefaultReasoningIntegrationConfig() ReasoningIntegrationConfig {
	return ReasoningIntegrationConfig{
		EnableByDefault:    true,
		AutoShow:           true,
		MinimizeOnComplete: false,
		ShowInsights:       true,
		ShowQualityMetrics: true,

		MaxConcurrentChains: 3,
		StreamBufferSize:    100,
		UpdateInterval:      50 * time.Millisecond,

		AllowInterruption: true,
		InterruptionKey:   "esc",

		DefaultPosition: PositionBottom,
		MinHeight:       10,
		MaxHeight:       30,
	}
}

// NewReasoningIntegration creates a new reasoning integration
func NewReasoningIntegration(config ReasoningIntegrationConfig) *ReasoningIntegration {
	return &ReasoningIntegration{
		config:       config,
		enabled:      config.EnableByDefault,
		visible:      false,
		position:     config.DefaultPosition,
		activeChains: make(map[string]*ActiveReasoning),
	}
}

// Initialize initializes the integration with the chat model
func (ri *ReasoningIntegration) Initialize(chatModel tea.Model, width, height int) {
	ri.mu.Lock()
	defer ri.mu.Unlock()

	ri.chatModel = chatModel

	// Calculate display dimensions based on position
	displayWidth, displayHeight := ri.calculateDimensions(width, height)
	ri.display = NewReasoningDisplay(displayWidth, displayHeight)
}

// Init implements tea.Model
func (ri *ReasoningIntegration) Init() tea.Cmd {
	return nil
}

// Update handles tea.Model updates
func (ri *ReasoningIntegration) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Handle visibility toggles
		switch msg.String() {
		case "ctrl+r":
			ri.toggleVisibility()
		case "ctrl+m":
			if ri.visible {
				ri.toggleMinimized()
			}
		}

		// Forward to display if visible and not minimized
		if ri.visible && !ri.minimized {
			newDisplay, cmd := ri.display.Update(msg)
			ri.display = newDisplay.(*ReasoningDisplay)
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		}

	case tea.WindowSizeMsg:
		// Update display dimensions
		if ri.display != nil {
			width, height := ri.calculateDimensions(msg.Width, msg.Height)
			newDisplay, cmd := ri.display.Update(tea.WindowSizeMsg{
				Width:  width,
				Height: height,
			})
			ri.display = newDisplay.(*ReasoningDisplay)
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		}

	case StartReasoningMsg:
		// Handle new reasoning session
		cmds = append(cmds, ri.handleStartReasoning(msg))

	case core.StreamEvent:
		// Forward stream events to display
		if ri.display != nil && ri.visible {
			newDisplay, cmd := ri.display.Update(msg)
			ri.display = newDisplay.(*ReasoningDisplay)
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		}

		// Update active reasoning
		ri.updateActiveReasoning(msg)

	case ReasoningCompleteMsg:
		// Handle reasoning completion
		ri.handleReasoningComplete(msg)

		// Auto-minimize if configured
		if ri.config.MinimizeOnComplete && ri.visible {
			ri.minimized = true
		}
	}

	// Check for active reasoning updates
	if ri.hasActiveReasoning() {
		cmds = append(cmds, ri.checkReasoningUpdates())
	}

	return ri, tea.Batch(cmds...)
}

// View implements tea.Model
func (ri *ReasoningIntegration) View() tea.View {
	// For now, return empty string as this is integrated into chat view
	return tea.NewView("")
}

// ViewWithChat renders the reasoning integration with chat content
func (ri *ReasoningIntegration) ViewWithChat(chatView string, width, height int) string {
	ri.mu.RLock()
	defer ri.mu.RUnlock()

	if !ri.enabled || !ri.visible {
		return chatView
	}

	// Render based on position
	switch ri.position {
	case PositionBottom:
		return ri.renderBottomLayout(chatView, width, height)
	case PositionRight:
		return ri.renderRightLayout(chatView, width, height)
	case PositionOverlay:
		return ri.renderOverlayLayout(chatView, width, height)
	case PositionSplit:
		return ri.renderSplitLayout(chatView, width, height)
	default:
		return chatView
	}
}

// StartReasoning starts reasoning for a message
func (ri *ReasoningIntegration) StartReasoning(ctx context.Context, messageID string, prompt string) error {
	ri.mu.Lock()
	defer ri.mu.Unlock()

	// Check concurrent limit
	activeCount := 0
	for _, ar := range ri.activeChains {
		if ar.Status == StatusStreaming {
			activeCount++
		}
	}

	if activeCount >= ri.config.MaxConcurrentChains {
		return gerror.New(gerror.ErrCodeResourceLimit, "max concurrent reasoning chains reached", nil).
			WithComponent("reasoning_integration").
			WithDetails("limit", ri.config.MaxConcurrentChains)
	}

	// Create reasoning streamer
	parser := core.NewThinkingBlockParser(nil) // Pass nil for metrics in UI context
	chainBuilder := core.NewReasoningChainBuilder("", "", "")
	streamer := core.NewReasoningStreamer(parser, chainBuilder, nil) // Pass nil for metrics

	// Streamer is ready to receive events

	// Track active reasoning
	ri.activeChains[messageID] = &ActiveReasoning{
		MessageID: messageID,
		Streamer:  streamer,
		StartTime: time.Now(),
		Status:    StatusStreaming,
	}

	// Auto-show if configured
	if ri.config.AutoShow && !ri.visible {
		ri.visible = true
		ri.minimized = false
	}

	// Connect to display
	if ri.display != nil && ri.visible {
		return ri.display.StartStreaming(ctx, streamer)
	}

	return nil
}

// StopReasoning stops reasoning for a message
func (ri *ReasoningIntegration) StopReasoning(ctx context.Context, messageID string) error {
	ri.mu.Lock()
	defer ri.mu.Unlock()

	active, exists := ri.activeChains[messageID]
	if !exists {
		return gerror.New(gerror.ErrCodeNotFound, "reasoning not found", nil).
			WithComponent("reasoning_integration").
			WithDetails("message_id", messageID)
	}

	if active.Status != StatusStreaming {
		return gerror.New(gerror.ErrCodeValidation, "reasoning not streaming", nil).
			WithComponent("reasoning_integration").
			WithDetails("status", active.Status)
	}

	// Interrupt the streamer
	active.Streamer.Interrupt()

	active.Status = StatusInterrupted
	active.EndTime = time.Now()

	return nil
}

// GetReasoningChain gets the reasoning chain for a message
func (ri *ReasoningIntegration) GetReasoningChain(messageID string) (*core.ReasoningChainEnhanced, error) {
	ri.mu.RLock()
	defer ri.mu.RUnlock()

	active, exists := ri.activeChains[messageID]
	if !exists {
		return nil, gerror.New(gerror.ErrCodeNotFound, "reasoning not found", nil).
			WithComponent("reasoning_integration").
			WithDetails("message_id", messageID)
	}

	if active.Chain == nil {
		return nil, gerror.New(gerror.ErrCodeValidation, "reasoning chain not complete", nil).
			WithComponent("reasoning_integration").
			WithDetails("status", active.Status)
	}

	return active.Chain, nil
}

// Layout rendering methods

func (ri *ReasoningIntegration) renderBottomLayout(chatView string, width, height int) string {
	displayHeight := ri.calculateDisplayHeight(height)
	chatHeight := height - displayHeight - 1 // -1 for separator

	// Resize chat view
	chatView = ri.resizeView(chatView, width, chatHeight)

	// Render reasoning display
	var reasoningView string
	if ri.minimized {
		reasoningView = ri.renderMinimizedView(width)
	} else {
		reasoningView = viewutil.String(ri.display.View())
	}

	// Combine with separator
	separator := lipgloss.NewStyle().
		Width(width).
		BorderStyle(lipgloss.NormalBorder()).
		BorderTop(true).
		BorderBottom(false).
		BorderLeft(false).
		BorderRight(false).
		Render("")

	return lipgloss.JoinVertical(
		lipgloss.Left,
		chatView,
		separator,
		reasoningView,
	)
}

func (ri *ReasoningIntegration) renderRightLayout(chatView string, width, height int) string {
	displayWidth := width / 2
	chatWidth := width - displayWidth - 1 // -1 for separator

	// Resize views
	chatView = ri.resizeView(chatView, chatWidth, height)

	var reasoningView string
	if ri.minimized {
		reasoningView = ri.renderMinimizedView(displayWidth)
	} else {
		reasoningView = viewutil.String(ri.display.View())
	}

	// Combine with separator
	separator := lipgloss.NewStyle().
		Height(height).
		BorderStyle(lipgloss.NormalBorder()).
		BorderLeft(true).
		BorderRight(false).
		BorderTop(false).
		BorderBottom(false).
		Render("")

	return lipgloss.JoinHorizontal(
		lipgloss.Top,
		chatView,
		separator,
		reasoningView,
	)
}

func (ri *ReasoningIntegration) renderOverlayLayout(chatView string, width, height int) string {
	// Render reasoning as overlay
	overlayWidth := width * 3 / 4
	overlayHeight := height * 2 / 3

	var reasoningView string
	if ri.minimized {
		reasoningView = ri.renderMinimizedView(overlayWidth)
	} else {
		reasoningView = viewutil.String(ri.display.View())
	}

	// Create overlay with shadow effect
	overlay := lipgloss.NewStyle().
		Width(overlayWidth).
		Height(overlayHeight).
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#7D56F4")).
		Background(lipgloss.Color("#1a1b26")).
		Padding(1).
		Render(reasoningView)

	// Position overlay in center
	return lipgloss.Place(
		width,
		height,
		lipgloss.Center,
		lipgloss.Center,
		overlay,
		lipgloss.WithWhitespaceChars(""),
		lipgloss.WithWhitespaceForeground(lipgloss.NoColor{}),
	)
}

func (ri *ReasoningIntegration) renderSplitLayout(chatView string, width, height int) string {
	// Equal split
	halfHeight := height / 2

	// Resize views
	chatView = ri.resizeView(chatView, width, halfHeight-1)

	var reasoningView string
	if ri.minimized {
		reasoningView = ri.renderMinimizedView(width)
	} else {
		reasoningView = viewutil.String(ri.display.View())
	}

	// Combine with double border
	separator := lipgloss.NewStyle().
		Width(width).
		BorderStyle(lipgloss.DoubleBorder()).
		BorderTop(true).
		BorderBottom(true).
		BorderLeft(false).
		BorderRight(false).
		Render("")

	return lipgloss.JoinVertical(
		lipgloss.Left,
		chatView,
		separator,
		reasoningView,
	)
}

func (ri *ReasoningIntegration) renderMinimizedView(width int) string {
	// Get current metrics
	metrics := ri.display.GetMetrics()

	var status string
	var statusStyle lipgloss.Style

	if metrics.Streaming {
		status = "🔄 Reasoning..."
		statusStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#7D56F4"))
	} else if metrics.Interrupted {
		status = "⚠️  Interrupted"
		statusStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#ED8936"))
	} else {
		status = fmt.Sprintf("✓ Complete (%.1fs)", metrics.Duration.Seconds())
		statusStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#48BB78"))
	}

	content := fmt.Sprintf("%s | %d blocks | %d tokens | Quality: %.0f%%",
		status,
		metrics.BlockCount,
		metrics.TokenCount,
		metrics.QualityScore*100,
	)

	hint := "Ctrl+M to expand"

	return lipgloss.NewStyle().
		Width(width).
		Padding(0, 1).
		Render(lipgloss.JoinHorizontal(
			lipgloss.Left,
			statusStyle.Render(content),
			lipgloss.NewStyle().Width(width-len(content)-len(hint)-4).Render(" "),
			lipgloss.NewStyle().Foreground(lipgloss.Color("#718096")).Render(hint),
		))
}

// Helper methods

func (ri *ReasoningIntegration) calculateDimensions(totalWidth, totalHeight int) (int, int) {
	switch ri.position {
	case PositionBottom:
		height := ri.calculateDisplayHeight(totalHeight)
		return totalWidth, height
	case PositionRight:
		width := totalWidth / 2
		return width, totalHeight
	case PositionOverlay:
		width := totalWidth * 3 / 4
		height := totalHeight * 2 / 3
		return width, height
	case PositionSplit:
		height := totalHeight / 2
		return totalWidth, height
	default:
		return totalWidth, ri.config.MinHeight
	}
}

func (ri *ReasoningIntegration) calculateDisplayHeight(totalHeight int) int {
	height := totalHeight / 3

	if height < ri.config.MinHeight {
		height = ri.config.MinHeight
	}
	if height > ri.config.MaxHeight {
		height = ri.config.MaxHeight
	}

	return height
}

func (ri *ReasoningIntegration) resizeView(view string, width, height int) string {
	// Simple resize - in production, use proper view resizing
	return lipgloss.NewStyle().
		Width(width).
		Height(height).
		MaxWidth(width).
		MaxHeight(height).
		Render(view)
}

func (ri *ReasoningIntegration) toggleVisibility() {
	ri.mu.Lock()
	defer ri.mu.Unlock()

	ri.visible = !ri.visible
	if ri.visible {
		ri.minimized = false
	}
}

func (ri *ReasoningIntegration) toggleMinimized() {
	ri.mu.Lock()
	defer ri.mu.Unlock()

	ri.minimized = !ri.minimized
}

func (ri *ReasoningIntegration) hasActiveReasoning() bool {
	ri.mu.RLock()
	defer ri.mu.RUnlock()

	for _, ar := range ri.activeChains {
		if ar.Status == StatusStreaming {
			return true
		}
	}
	return false
}

func (ri *ReasoningIntegration) checkReasoningUpdates() tea.Cmd {
	return tea.Tick(ri.config.UpdateInterval, func(time.Time) tea.Msg {
		return CheckReasoningUpdatesMsg{}
	})
}

func (ri *ReasoningIntegration) handleStartReasoning(msg StartReasoningMsg) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		if err := ri.StartReasoning(ctx, msg.MessageID, msg.Prompt); err != nil {
			logger := observability.GetLogger(ctx)
			logger.ErrorContext(ctx, "Failed to start reasoning", "error", err)
			return ReasoningErrorMsg{
				MessageID: msg.MessageID,
				Error:     err,
			}
		}
		return nil
	}
}

func (ri *ReasoningIntegration) updateActiveReasoning(event core.StreamEvent) {
	ri.mu.Lock()
	defer ri.mu.Unlock()

	// Find active reasoning for this event
	for _, ar := range ri.activeChains {
		if ar.Status == StatusStreaming {
			switch event.Type {
			case core.StreamEventContentChunk:
				// Check if it's the final chunk with complete chain
				if chain, ok := event.Data.(*core.ReasoningChainEnhanced); ok {
					ar.Chain = chain
					ar.Status = StatusComplete
					ar.EndTime = time.Now()
				}
			case core.StreamEventError:
				if err, ok := event.Data.(error); ok {
					ar.Error = err
					ar.Status = StatusError
					ar.EndTime = time.Now()
				}
			case core.StreamEventInterrupted:
				ar.Status = StatusInterrupted
				ar.EndTime = time.Now()
			}
		}
	}
}

func (ri *ReasoningIntegration) handleReasoningComplete(msg ReasoningCompleteMsg) {
	ri.mu.Lock()
	defer ri.mu.Unlock()

	if ar, exists := ri.activeChains[msg.MessageID]; exists {
		ar.Chain = msg.Chain
		ar.Status = StatusComplete
		ar.EndTime = time.Now()
	}
}

// Messages for tea.Model communication

// StartReasoningMsg signals to start reasoning
type StartReasoningMsg struct {
	MessageID string
	Prompt    string
}

// ReasoningCompleteMsg signals reasoning completion
type ReasoningCompleteMsg struct {
	MessageID string
	Chain     *core.ReasoningChainEnhanced
}

// ReasoningErrorMsg signals a reasoning error
type ReasoningErrorMsg struct {
	MessageID string
	Error     error
}

// CheckReasoningUpdatesMsg signals to check for updates
type CheckReasoningUpdatesMsg struct{}

// Integration with message types

// ReasoningMessage extends session.Message with reasoning support
type ReasoningMessage struct {
	session.Message
	ReasoningChain *core.ReasoningChainEnhanced `json:"reasoning_chain,omitempty"`
	ShowReasoning  bool                         `json:"show_reasoning"`
}

// IsEnabled returns if reasoning integration is enabled
func (ri *ReasoningIntegration) IsEnabled() bool {
	ri.mu.RLock()
	defer ri.mu.RUnlock()
	return ri.enabled
}

// SetEnabled enables or disables reasoning integration
func (ri *ReasoningIntegration) SetEnabled(enabled bool) {
	ri.mu.Lock()
	defer ri.mu.Unlock()
	ri.enabled = enabled
	if !enabled {
		ri.visible = false
	}
}

// GetActiveReasoningCount returns the number of active reasoning chains
func (ri *ReasoningIntegration) GetActiveReasoningCount() int {
	ri.mu.RLock()
	defer ri.mu.RUnlock()

	count := 0
	for _, ar := range ri.activeChains {
		if ar.Status == StatusStreaming {
			count++
		}
	}
	return count
}

// CleanupCompleted removes completed reasoning chains older than duration
func (ri *ReasoningIntegration) CleanupCompleted(olderThan time.Duration) {
	ri.mu.Lock()
	defer ri.mu.Unlock()

	now := time.Now()
	for id, ar := range ri.activeChains {
		if ar.Status != StatusStreaming && now.Sub(ar.EndTime) > olderThan {
			delete(ri.activeChains, id)
		}
	}
}
