// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package reasoning

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/guild-framework/guild-core/pkg/agents/core"
	"github.com/guild-framework/guild-core/pkg/gerror"
	"github.com/muesli/reflow/wordwrap"
)

// ReasoningDisplay is a production-grade UI component for displaying reasoning
type ReasoningDisplay struct {
	// Components
	viewport viewport.Model
	spinner  spinner.Model

	// Reasoning state
	streamer  *core.ReasoningStreamer
	eventChan <-chan core.StreamEvent
	blocks    []*core.ThinkingBlock
	chain     *core.ReasoningChainEnhanced

	// Display state
	width       int
	height      int
	focused     bool
	streaming   bool
	interrupted bool
	showDetails bool
	collapsed   map[string]bool

	// Styling
	styles *ReasoningStyles

	// Metrics
	startTime  time.Time
	endTime    time.Time
	tokenCount int

	// Thread safety
	mu sync.RWMutex
}

// ReasoningStyles defines the visual styling for reasoning display
type ReasoningStyles struct {
	// Container styles
	Container       lipgloss.Style
	ActiveContainer lipgloss.Style

	// Block styles
	BlockContainer lipgloss.Style
	BlockHeader    lipgloss.Style
	BlockContent   lipgloss.Style

	// Type-specific styles
	AnalysisStyle      lipgloss.Style
	PlanningStyle      lipgloss.Style
	DecisionStyle      lipgloss.Style
	ToolSelectionStyle lipgloss.Style
	VerificationStyle  lipgloss.Style
	HypothesisStyle    lipgloss.Style
	ErrorRecoveryStyle lipgloss.Style

	// Metadata styles
	ConfidenceHigh   lipgloss.Style
	ConfidenceMedium lipgloss.Style
	ConfidenceLow    lipgloss.Style

	// Decision point styles
	DecisionPoint     lipgloss.Style
	SelectedOption    lipgloss.Style
	AlternativeOption lipgloss.Style

	// Quality indicator styles
	QualityExcellent lipgloss.Style
	QualityGood      lipgloss.Style
	QualityFair      lipgloss.Style
	QualityPoor      lipgloss.Style

	// Status styles
	StreamingIndicator lipgloss.Style
	InterruptedStyle   lipgloss.Style
	CompletedStyle     lipgloss.Style

	// Insight styles
	InsightContainer lipgloss.Style
	InsightHeader    lipgloss.Style
	InsightContent   lipgloss.Style
}

// DefaultReasoningStyles returns the default styling configuration
func DefaultReasoningStyles() *ReasoningStyles {
	// Define color palette
	primary := lipgloss.Color("#7D56F4")
	secondary := lipgloss.Color("#5A67D8")
	success := lipgloss.Color("#48BB78")
	warning := lipgloss.Color("#ED8936")
	danger := lipgloss.Color("#F56565")
	muted := lipgloss.Color("#718096")

	// Base styles
	baseContainer := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		Padding(1)

	return &ReasoningStyles{
		// Container styles
		Container: baseContainer.
			BorderForeground(muted),
		ActiveContainer: baseContainer.
			BorderForeground(primary),

		// Block styles
		BlockContainer: lipgloss.NewStyle().
			Margin(1, 0).
			Padding(1).
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(muted),
		BlockHeader: lipgloss.NewStyle().
			Bold(true).
			Foreground(primary),
		BlockContent: lipgloss.NewStyle().
			Margin(0, 1),

		// Type-specific styles
		AnalysisStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#4ECDC4")),
		PlanningStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#95E1D3")),
		DecisionStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#F38181")),
		ToolSelectionStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFD93D")),
		VerificationStyle: lipgloss.NewStyle().
			Foreground(success),
		HypothesisStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#C7CEEA")),
		ErrorRecoveryStyle: lipgloss.NewStyle().
			Foreground(danger),

		// Metadata styles
		ConfidenceHigh: lipgloss.NewStyle().
			Foreground(success).
			Bold(true),
		ConfidenceMedium: lipgloss.NewStyle().
			Foreground(warning),
		ConfidenceLow: lipgloss.NewStyle().
			Foreground(danger),

		// Decision point styles
		DecisionPoint: lipgloss.NewStyle().
			Margin(1, 2).
			Foreground(secondary),
		SelectedOption: lipgloss.NewStyle().
			Bold(true).
			Foreground(success),
		AlternativeOption: lipgloss.NewStyle().
			Foreground(muted).
			Italic(true),

		// Quality indicator styles
		QualityExcellent: lipgloss.NewStyle().
			Foreground(success).
			SetString("⭐⭐⭐⭐⭐"),
		QualityGood: lipgloss.NewStyle().
			Foreground(success).
			SetString("⭐⭐⭐⭐"),
		QualityFair: lipgloss.NewStyle().
			Foreground(warning).
			SetString("⭐⭐⭐"),
		QualityPoor: lipgloss.NewStyle().
			Foreground(danger).
			SetString("⭐⭐"),

		// Status styles
		StreamingIndicator: lipgloss.NewStyle().
			Foreground(primary).
			Bold(true),
		InterruptedStyle: lipgloss.NewStyle().
			Foreground(warning).
			Bold(true),
		CompletedStyle: lipgloss.NewStyle().
			Foreground(success).
			Bold(true),

		// Insight styles
		InsightContainer: lipgloss.NewStyle().
			Margin(1, 0).
			Padding(1).
			BorderStyle(lipgloss.DoubleBorder()).
			BorderForeground(primary),
		InsightHeader: lipgloss.NewStyle().
			Bold(true).
			Foreground(primary),
		InsightContent: lipgloss.NewStyle().
			Margin(0, 1).
			Italic(true),
	}
}

// NewReasoningDisplay creates a new reasoning display component
func NewReasoningDisplay(width, height int) *ReasoningDisplay {
	rd := &ReasoningDisplay{
		width:     width,
		height:    height,
		styles:    DefaultReasoningStyles(),
		collapsed: make(map[string]bool),
		blocks:    make([]*core.ThinkingBlock, 0),
		startTime: time.Now(),
	}

	// Initialize viewport
	rd.viewport = viewport.New(width-4, height-8) // Account for borders and status
	rd.viewport.SetContent("")

	// Initialize spinner
	rd.spinner = spinner.New()
	rd.spinner.Spinner = spinner.Dot
	rd.spinner.Style = rd.styles.StreamingIndicator

	return rd
}

// StartStreaming starts streaming reasoning events
func (rd *ReasoningDisplay) StartStreaming(ctx context.Context, streamer *core.ReasoningStreamer) error {
	rd.mu.Lock()
	defer rd.mu.Unlock()

	if rd.streaming {
		return gerror.New(gerror.ErrCodeConflict, "already streaming", nil).
			WithComponent("reasoning_display")
	}

	rd.streamer = streamer
	rd.eventChan = streamer.EventChannel()
	rd.streaming = true
	rd.interrupted = false
	rd.startTime = time.Now()

	return nil
}

// Init implements tea.Model
func (rd *ReasoningDisplay) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model
func (rd *ReasoningDisplay) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			// Handle interruption
			if rd.streaming && !rd.interrupted {
				rd.handleInterruption()
			}
		case "tab":
			// Toggle details view
			rd.showDetails = !rd.showDetails
			rd.updateViewport()
		case "c":
			// Toggle collapse for current block
			if currentBlock := rd.getCurrentBlock(); currentBlock != nil {
				rd.toggleCollapse(currentBlock.ID)
			}
		case "ctrl+r":
			// Refresh display
			rd.updateViewport()
		}

	case tea.WindowSizeMsg:
		rd.width = msg.Width
		rd.height = msg.Height
		rd.viewport.Width = msg.Width - 4
		rd.viewport.Height = msg.Height - 8
		rd.updateViewport()

	case spinner.TickMsg:
		if rd.streaming {
			var cmd tea.Cmd
			rd.spinner, cmd = rd.spinner.Update(msg)
			cmds = append(cmds, cmd)
		}

	case core.StreamEvent:
		rd.handleStreamEvent(msg)
		rd.updateViewport()
	}

	// Update viewport
	var cmd tea.Cmd
	rd.viewport, cmd = rd.viewport.Update(msg)
	cmds = append(cmds, cmd)

	// Check for streaming events
	if rd.streaming {
		cmds = append(cmds, rd.checkForEvents())
	}

	return rd, tea.Batch(cmds...)
}

// View implements tea.Model
func (rd *ReasoningDisplay) View() string {
	rd.mu.RLock()
	defer rd.mu.RUnlock()

	// Build the view
	var builder strings.Builder

	// Header
	builder.WriteString(rd.renderHeader())
	builder.WriteString("\n")

	// Main viewport
	containerStyle := rd.styles.Container
	if rd.focused {
		containerStyle = rd.styles.ActiveContainer
	}

	builder.WriteString(containerStyle.Render(rd.viewport.View()))
	builder.WriteString("\n")

	// Status bar
	builder.WriteString(rd.renderStatusBar())

	return builder.String()
}

// Focus sets the focus state
func (rd *ReasoningDisplay) Focus() {
	rd.mu.Lock()
	defer rd.mu.Unlock()
	rd.focused = true
}

// Blur removes focus
func (rd *ReasoningDisplay) Blur() {
	rd.mu.Lock()
	defer rd.mu.Unlock()
	rd.focused = false
}

// renderHeader renders the header section
func (rd *ReasoningDisplay) renderHeader() string {
	var status string
	var statusStyle lipgloss.Style

	if rd.streaming {
		status = rd.spinner.View() + " Reasoning..."
		statusStyle = rd.styles.StreamingIndicator
	} else if rd.interrupted {
		status = "⚠️  Interrupted"
		statusStyle = rd.styles.InterruptedStyle
	} else if rd.chain != nil {
		status = "✓ Complete"
		statusStyle = rd.styles.CompletedStyle
	} else {
		status = "Ready"
		statusStyle = rd.styles.Container.Foreground(lipgloss.Color("#718096"))
	}

	header := lipgloss.JoinHorizontal(
		lipgloss.Left,
		lipgloss.NewStyle().Bold(true).Render("AI Reasoning"),
		lipgloss.NewStyle().Width(20).Render(" "),
		statusStyle.Render(status),
	)

	return lipgloss.NewStyle().
		Width(rd.width).
		Padding(0, 1).
		Render(header)
}

// renderStatusBar renders the status bar
func (rd *ReasoningDisplay) renderStatusBar() string {
	var parts []string

	// Block count
	parts = append(parts, fmt.Sprintf("Blocks: %d", len(rd.blocks)))

	// Token count
	if rd.tokenCount > 0 {
		parts = append(parts, fmt.Sprintf("Tokens: %d", rd.tokenCount))
	}

	// Duration
	duration := rd.endTime.Sub(rd.startTime)
	if rd.endTime.IsZero() && rd.streaming {
		duration = time.Since(rd.startTime)
	}
	parts = append(parts, fmt.Sprintf("Duration: %s", duration.Round(time.Millisecond)))

	// Quality score
	if rd.chain != nil {
		qualityIndicator := rd.getQualityIndicator(rd.chain.Quality.Overall)
		parts = append(parts, fmt.Sprintf("Quality: %s", qualityIndicator))
	}

	// Controls hint
	controls := "ESC: interrupt | TAB: details | C: collapse"

	statusLeft := strings.Join(parts, " | ")

	return lipgloss.NewStyle().
		Width(rd.width).
		Padding(0, 1).
		Foreground(lipgloss.Color("#718096")).
		Render(lipgloss.JoinHorizontal(
			lipgloss.Left,
			statusLeft,
			lipgloss.NewStyle().Width(rd.width-len(statusLeft)-len(controls)-2).Render(" "),
			controls,
		))
}

// updateViewport updates the viewport content
func (rd *ReasoningDisplay) updateViewport() {
	content := rd.renderContent()
	rd.viewport.SetContent(content)
}

// renderContent renders the main content
func (rd *ReasoningDisplay) renderContent() string {
	var builder strings.Builder

	// Render each thinking block
	for i, block := range rd.blocks {
		if i > 0 {
			builder.WriteString("\n")
		}
		builder.WriteString(rd.renderThinkingBlock(block))
	}

	// Render insights if available
	if rd.chain != nil && len(rd.chain.Insights) > 0 {
		builder.WriteString("\n\n")
		builder.WriteString(rd.renderInsights(rd.chain.Insights))
	}

	// Render quality analysis if complete
	if rd.chain != nil && !rd.streaming {
		builder.WriteString("\n\n")
		builder.WriteString(rd.renderQualityAnalysis(rd.chain.Quality))
	}

	return builder.String()
}

// renderThinkingBlock renders a single thinking block
func (rd *ReasoningDisplay) renderThinkingBlock(block *core.ThinkingBlock) string {
	// Get style for block type
	typeStyle := rd.getTypeStyle(block.Type)

	// Build header
	header := fmt.Sprintf("%s %s", rd.getTypeIcon(block.Type), typeStyle.Render(string(block.Type)))

	// Add confidence if available
	if block.Confidence > 0 {
		confidenceStyle := rd.getConfidenceStyle(block.Confidence)
		header += fmt.Sprintf(" %s", confidenceStyle.Render(fmt.Sprintf("(%.0f%%)", block.Confidence*100)))
	}

	// Add timestamp
	if !block.Timestamp.IsZero() {
		elapsed := block.Timestamp.Sub(rd.startTime).Round(time.Millisecond)
		header += lipgloss.NewStyle().Foreground(lipgloss.Color("#718096")).Render(fmt.Sprintf(" +%s", elapsed))
	}

	// Check if collapsed
	if rd.collapsed[block.ID] {
		return rd.styles.BlockHeader.Render(header + " [collapsed]")
	}

	// Build content
	var contentBuilder strings.Builder

	// Main content
	if block.Content != "" {
		wrapped := wordwrap.String(block.Content, rd.width-8)
		contentBuilder.WriteString(rd.styles.BlockContent.Render(wrapped))
	}

	// Decision points
	if len(block.DecisionPoints) > 0 {
		contentBuilder.WriteString("\n")
		contentBuilder.WriteString(rd.renderDecisionPoints(block.DecisionPoints))
	}

	// Tool context
	if block.ToolContext != nil {
		contentBuilder.WriteString("\n")
		contentBuilder.WriteString(rd.renderToolContext(block.ToolContext))
	}

	// Error context
	if block.ErrorContext != nil {
		contentBuilder.WriteString("\n")
		contentBuilder.WriteString(rd.renderErrorContext(block.ErrorContext))
	}

	// Combine header and content
	return rd.styles.BlockContainer.Render(
		rd.styles.BlockHeader.Render(header) + "\n" +
			contentBuilder.String(),
	)
}

// renderDecisionPoints renders decision points
func (rd *ReasoningDisplay) renderDecisionPoints(points []core.DecisionPoint) string {
	var builder strings.Builder

	for _, dp := range points {
		builder.WriteString(rd.styles.DecisionPoint.Render(fmt.Sprintf("→ %s", dp.Decision)))
		builder.WriteString("\n")

		// Selected option
		builder.WriteString(rd.styles.SelectedOption.Render(fmt.Sprintf("  ✓ %s", dp.Decision)))
		if dp.Rationale != "" {
			builder.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#718096")).Render(fmt.Sprintf(" - %s", dp.Rationale)))
		}
		builder.WriteString("\n")

		// Alternatives (if showing details)
		if rd.showDetails && len(dp.Alternatives) > 0 {
			for _, alt := range dp.Alternatives {
				builder.WriteString(rd.styles.AlternativeOption.Render(fmt.Sprintf("    • %s", alt.Option)))
				builder.WriteString("\n")
			}
		}
	}

	return strings.TrimRight(builder.String(), "\n")
}

// renderToolContext renders tool usage context
func (rd *ReasoningDisplay) renderToolContext(tc *core.ToolContext) string {
	var parts []string

	parts = append(parts, fmt.Sprintf("🔧 Tool: %s", tc.ToolName))

	if tc.Purpose != "" {
		parts = append(parts, fmt.Sprintf("Purpose: %s", tc.Purpose))
	}

	if rd.showDetails {
		if tc.ExpectedOutcome != "" {
			parts = append(parts, fmt.Sprintf("Expected: %s", tc.ExpectedOutcome))
		}
		if tc.ActualOutcome != nil && *tc.ActualOutcome != "" {
			parts = append(parts, fmt.Sprintf("Actual: %s", *tc.ActualOutcome))
		}
	}

	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFD93D")).
		Margin(0, 2).
		Render(strings.Join(parts, "\n"))
}

// renderErrorContext renders error recovery context
func (rd *ReasoningDisplay) renderErrorContext(ec *core.ErrorAnalysis) string {
	var builder strings.Builder

	builder.WriteString(rd.styles.ErrorRecoveryStyle.Render("⚠️  Error Recovery"))
	builder.WriteString("\n")

	if ec.ErrorType != "" {
		builder.WriteString(fmt.Sprintf("  Type: %s\n", ec.ErrorType))
	}

	if ec.Description != "" {
		builder.WriteString(fmt.Sprintf("  Description: %s\n", ec.Description))
	}

	if ec.Recovery != "" {
		builder.WriteString(fmt.Sprintf("  Strategy: %s\n", ec.Recovery))
	}

	if rd.showDetails && ec.RootCause != "" {
		builder.WriteString(fmt.Sprintf("  Root Cause: %s\n", ec.RootCause))
	}

	return lipgloss.NewStyle().Margin(0, 2).Render(strings.TrimRight(builder.String(), "\n"))
}

// renderInsights renders reasoning insights
func (rd *ReasoningDisplay) renderInsights(insights []core.Insight) string {
	var builder strings.Builder

	builder.WriteString(rd.styles.InsightHeader.Render("💡 Key Insights"))
	builder.WriteString("\n")

	for _, insight := range insights {
		icon := rd.getInsightIcon(insight.Type)
		builder.WriteString(fmt.Sprintf("%s %s\n", icon, insight.Description))

		if rd.showDetails && insight.Source != "" {
			builder.WriteString(rd.styles.InsightContent.Render(fmt.Sprintf("  Source: %s\n", insight.Source)))
		}

		if insight.Actionable && len(insight.Actions) > 0 {
			builder.WriteString(rd.styles.InsightContent.Render(fmt.Sprintf("  → %s\n", insight.Actions[0])))
		}
	}

	return rd.styles.InsightContainer.Render(strings.TrimRight(builder.String(), "\n"))
}

// renderQualityAnalysis renders quality metrics
func (rd *ReasoningDisplay) renderQualityAnalysis(quality core.QualityMetrics) string {
	var builder strings.Builder

	builder.WriteString(lipgloss.NewStyle().Bold(true).Render("Quality Analysis"))
	builder.WriteString("\n")

	// Overall score with visual indicator
	overallIndicator := rd.getQualityIndicator(quality.Overall)
	builder.WriteString(fmt.Sprintf("Overall: %s %.0f%%\n", overallIndicator, quality.Overall*100))

	if rd.showDetails {
		// Individual metrics
		metrics := []struct {
			name  string
			value float64
		}{
			{"Coherence", quality.Coherence},
			{"Completeness", quality.Completeness},
			{"Depth", quality.Depth},
			{"Accuracy", quality.Accuracy},
			{"Innovation", quality.Innovation},
		}

		for _, m := range metrics {
			bar := rd.renderMetricBar(m.value)
			builder.WriteString(fmt.Sprintf("  %s: %s %.0f%%\n", m.name, bar, m.value*100))
		}
	}

	return lipgloss.NewStyle().
		Margin(1, 0).
		Padding(1).
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("#718096")).
		Render(strings.TrimRight(builder.String(), "\n"))
}

// handleStreamEvent handles incoming stream events
func (rd *ReasoningDisplay) handleStreamEvent(event core.StreamEvent) {
	rd.mu.Lock()
	defer rd.mu.Unlock()

	switch event.Type {
	case core.StreamEventThinkingStart:
		if block, ok := event.Data.(*core.ThinkingBlock); ok {
			rd.blocks = append(rd.blocks, block)
		}

	case core.StreamEventThinkingUpdate:
		if update, ok := event.Data.(map[string]interface{}); ok {
			if blockID, ok := update["block_id"].(string); ok {
				if content, ok := update["content"].(string); ok {
					rd.updateBlock(blockID, content)
				}
			}
		}

	case core.StreamEventThinkingComplete:
		if block, ok := event.Data.(*core.ThinkingBlock); ok {
			rd.replaceBlock(block)
			rd.tokenCount += block.TokenCount
		}

	case core.StreamEventContentChunk:
		if chain, ok := event.Data.(*core.ReasoningChainEnhanced); ok {
			rd.chain = chain
			rd.streaming = false
			rd.endTime = time.Now()
		}

	case core.StreamEventError:
		if err, ok := event.Data.(error); ok {
			rd.handleError(err)
		}

	case core.StreamEventInterrupted:
		rd.interrupted = true
		rd.streaming = false
		rd.endTime = time.Now()
	}
}

// updateBlock updates a block's content
func (rd *ReasoningDisplay) updateBlock(blockID, content string) {
	for i, block := range rd.blocks {
		if block.ID == blockID {
			rd.blocks[i].Content = content
			return
		}
	}
}

// replaceBlock replaces a block with updated version
func (rd *ReasoningDisplay) replaceBlock(updated *core.ThinkingBlock) {
	for i, block := range rd.blocks {
		if block.ID == updated.ID {
			rd.blocks[i] = updated
			return
		}
	}
}

// handleInterruption handles user interruption
func (rd *ReasoningDisplay) handleInterruption() {
	if rd.streamer != nil {
		rd.streamer.Interrupt()
	}
}

// handleError handles streaming errors
func (rd *ReasoningDisplay) handleError(err error) {
	rd.mu.Lock()
	defer rd.mu.Unlock()

	// Create error block
	errorBlock := &core.ThinkingBlock{
		ID:        fmt.Sprintf("error_%d", time.Now().UnixNano()),
		Type:      core.ThinkingTypeErrorRecovery,
		Content:   fmt.Sprintf("Error: %v", err),
		Timestamp: time.Now(),
		ErrorContext: &core.ErrorAnalysis{
			ErrorType:   "StreamingError",
			Description: err.Error(),
		},
	}

	rd.blocks = append(rd.blocks, errorBlock)
	rd.streaming = false
	rd.endTime = time.Now()
}

// checkForEvents checks for new streaming events
func (rd *ReasoningDisplay) checkForEvents() tea.Cmd {
	return func() tea.Msg {
		select {
		case event, ok := <-rd.eventChan:
			if ok {
				return event
			}
			return nil
		default:
			// Continue spinner animation
			return rd.spinner.Tick()
		}
	}
}

// Helper methods

func (rd *ReasoningDisplay) getTypeStyle(t core.ThinkingType) lipgloss.Style {
	switch t {
	case core.ThinkingTypeAnalysis:
		return rd.styles.AnalysisStyle
	case core.ThinkingTypePlanning:
		return rd.styles.PlanningStyle
	case core.ThinkingTypeDecisionMaking:
		return rd.styles.DecisionStyle
	case core.ThinkingTypeToolSelection:
		return rd.styles.ToolSelectionStyle
	case core.ThinkingTypeVerification:
		return rd.styles.VerificationStyle
	case core.ThinkingTypeHypothesis:
		return rd.styles.HypothesisStyle
	case core.ThinkingTypeErrorRecovery:
		return rd.styles.ErrorRecoveryStyle
	default:
		return lipgloss.NewStyle()
	}
}

func (rd *ReasoningDisplay) getTypeIcon(t core.ThinkingType) string {
	icons := map[core.ThinkingType]string{
		core.ThinkingTypeAnalysis:       "🔍",
		core.ThinkingTypePlanning:       "📋",
		core.ThinkingTypeDecisionMaking: "🎯",
		core.ThinkingTypeToolSelection:  "🔧",
		core.ThinkingTypeVerification:   "✅",
		core.ThinkingTypeHypothesis:     "💭",
		core.ThinkingTypeErrorRecovery:  "🔄",
	}

	if icon, ok := icons[t]; ok {
		return icon
	}
	return "📝"
}

func (rd *ReasoningDisplay) getConfidenceStyle(confidence float64) lipgloss.Style {
	if confidence >= 0.8 {
		return rd.styles.ConfidenceHigh
	} else if confidence >= 0.6 {
		return rd.styles.ConfidenceMedium
	}
	return rd.styles.ConfidenceLow
}

func (rd *ReasoningDisplay) getQualityIndicator(score float64) string {
	if score >= 0.9 {
		return rd.styles.QualityExcellent.String()
	} else if score >= 0.75 {
		return rd.styles.QualityGood.String()
	} else if score >= 0.6 {
		return rd.styles.QualityFair.String()
	}
	return rd.styles.QualityPoor.String()
}

func (rd *ReasoningDisplay) getInsightIcon(t core.InsightType) string {
	icons := map[core.InsightType]string{
		core.InsightTypePattern:      "🔄",
		core.InsightTypeOptimization: "⚡",
		core.InsightTypeAnomaly:      "⚠️",
		core.InsightTypeRisk:         "🚨",
		core.InsightTypeOpportunity:  "💡",
	}

	if icon, ok := icons[t]; ok {
		return icon
	}
	return "💡"
}

func (rd *ReasoningDisplay) renderMetricBar(value float64) string {
	width := 10
	filled := int(value * float64(width))

	bar := strings.Repeat("█", filled) + strings.Repeat("░", width-filled)

	if value >= 0.8 {
		return rd.styles.ConfidenceHigh.Render(bar)
	} else if value >= 0.6 {
		return rd.styles.ConfidenceMedium.Render(bar)
	}
	return rd.styles.ConfidenceLow.Render(bar)
}

func (rd *ReasoningDisplay) toggleCollapse(blockID string) {
	rd.mu.Lock()
	defer rd.mu.Unlock()

	rd.collapsed[blockID] = !rd.collapsed[blockID]
}

func (rd *ReasoningDisplay) getCurrentBlock() *core.ThinkingBlock {
	if len(rd.blocks) == 0 {
		return nil
	}

	// Return the last block being actively updated
	for i := len(rd.blocks) - 1; i >= 0; i-- {
		// Check if block is still being updated by looking at duration
		if rd.blocks[i].Duration == 0 {
			return rd.blocks[i]
		}
	}

	// Return the last block
	return rd.blocks[len(rd.blocks)-1]
}

// GetMetrics returns display metrics
func (rd *ReasoningDisplay) GetMetrics() ReasoningDisplayMetrics {
	rd.mu.RLock()
	defer rd.mu.RUnlock()

	return ReasoningDisplayMetrics{
		BlockCount:   len(rd.blocks),
		TokenCount:   rd.tokenCount,
		Duration:     rd.endTime.Sub(rd.startTime),
		Interrupted:  rd.interrupted,
		Streaming:    rd.streaming,
		QualityScore: rd.getChainQuality(),
	}
}

func (rd *ReasoningDisplay) getChainQuality() float64 {
	if rd.chain != nil {
		return rd.chain.Quality.Overall
	}
	return 0
}

// ReasoningDisplayMetrics contains display metrics
type ReasoningDisplayMetrics struct {
	BlockCount   int
	TokenCount   int
	Duration     time.Duration
	Interrupted  bool
	Streaming    bool
	QualityScore float64
}
