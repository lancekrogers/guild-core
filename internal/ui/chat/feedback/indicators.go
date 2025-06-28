// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package feedback

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/lancekrogers/guild/pkg/gerror"
)

// TypingIndicator provides animated typing indicators for agents
type TypingIndicator struct {
	agent     string
	animation []string
	frame     int
	style     lipgloss.Style
	ctx       context.Context
}

// NewTypingIndicator creates a new typing indicator for an agent
func NewTypingIndicator(ctx context.Context, agent string) *TypingIndicator {
	if ctx == nil {
		ctx = context.Background()
	}

	return &TypingIndicator{
		agent: agent,
		animation: []string{
			"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏",
		},
		frame: 0,
		style: lipgloss.NewStyle().
			Foreground(lipgloss.Color("147")), // Light purple
		ctx: ctx,
	}
}

// View renders the current typing indicator
func (ti *TypingIndicator) View() string {
	if err := ti.ctx.Err(); err != nil {
		return ""
	}

	dots := strings.Repeat(".", (ti.frame/3)%4)

	return ti.style.Render(fmt.Sprintf("%s %s is thinking%s",
		ti.animation[ti.frame%len(ti.animation)],
		ti.agent,
		dots))
}

// Tick advances the animation frame
func (ti *TypingIndicator) Tick() {
	ti.frame = (ti.frame + 1) % (len(ti.animation) * 4) // 4x cycle for dots
}

// SetStyle updates the styling of the indicator
func (ti *TypingIndicator) SetStyle(style lipgloss.Style) {
	ti.style = style
}

// ProcessingState represents different agent processing states
type ProcessingState int

const (
	StateListening ProcessingState = iota
	StateAnalyzing
	StateGenerating
	StateRefining
	StateValidating
	StateCompleting
)

// String returns a human-readable processing state name
func (ps ProcessingState) String() string {
	states := []string{
		"Listening",
		"Analyzing",
		"Generating",
		"Refining",
		"Validating",
		"Completing",
	}
	if int(ps) < len(states) {
		return states[ps]
	}
	return "Processing"
}

// Display returns a styled display string for the processing state
func (ps ProcessingState) Display() string {
	displays := map[ProcessingState]string{
		StateListening:  "🎧 Listening...",
		StateAnalyzing:  "🔍 Analyzing requirements...",
		StateGenerating: "📝 Generating response...",
		StateRefining:   "✨ Refining output...",
		StateValidating: "🔍 Validating results...",
		StateCompleting: "✅ Completing task...",
	}

	display, exists := displays[ps]
	if !exists {
		display = "⚙️ Processing..."
	}

	return display
}

// Icon returns an icon for the processing state
func (ps ProcessingState) Icon() string {
	icons := map[ProcessingState]string{
		StateListening:  "🎧",
		StateAnalyzing:  "🔍",
		StateGenerating: "📝",
		StateRefining:   "✨",
		StateValidating: "🔍",
		StateCompleting: "✅",
	}

	icon, exists := icons[ps]
	if !exists {
		icon = "⚙️"
	}

	return icon
}

// ProcessingIndicator combines typing animation with processing states
type ProcessingIndicator struct {
	agent           string
	state           ProcessingState
	typingIndicator *TypingIndicator
	startTime       time.Time
	ctx             context.Context
}

// NewProcessingIndicator creates a new processing indicator
func NewProcessingIndicator(ctx context.Context, agent string, state ProcessingState) (*ProcessingIndicator, error) {
	if ctx == nil {
		return nil, gerror.New(gerror.ErrCodeInvalidInput, "context cannot be nil", nil).
			WithComponent("feedback.indicators").
			WithOperation("NewProcessingIndicator")
	}

	if agent == "" {
		return nil, gerror.New(gerror.ErrCodeInvalidInput, "agent name cannot be empty", nil).
			WithComponent("feedback.indicators").
			WithOperation("NewProcessingIndicator")
	}

	return &ProcessingIndicator{
		agent:           agent,
		state:           state,
		typingIndicator: NewTypingIndicator(ctx, agent),
		startTime:       time.Now(),
		ctx:             ctx,
	}, nil
}

// View renders the processing indicator
func (pi *ProcessingIndicator) View() string {
	if err := pi.ctx.Err(); err != nil {
		return ""
	}

	// Use typing indicator animation with processing state
	animation := pi.typingIndicator.animation[pi.typingIndicator.frame%len(pi.typingIndicator.animation)]

	// Calculate elapsed time
	elapsed := time.Since(pi.startTime)

	// Style based on how long it's been processing
	style := pi.typingIndicator.style
	if elapsed > 30*time.Second {
		style = style.Foreground(lipgloss.Color("226")) // Yellow for long operations
	} else if elapsed > 10*time.Second {
		style = style.Foreground(lipgloss.Color("208")) // Orange for medium operations
	}

	baseText := fmt.Sprintf("%s %s %s",
		animation,
		pi.agent,
		pi.state.Display())

	// Add elapsed time for long operations
	if elapsed > 5*time.Second {
		baseText += fmt.Sprintf(" (%ds)", int(elapsed.Seconds()))
	}

	return style.Render(baseText)
}

// Tick advances the animation
func (pi *ProcessingIndicator) Tick() {
	pi.typingIndicator.Tick()
}

// UpdateState changes the processing state
func (pi *ProcessingIndicator) UpdateState(state ProcessingState) {
	pi.state = state
	// Reset start time for new state
	pi.startTime = time.Now()
}

// GetElapsed returns the elapsed time since the indicator started
func (pi *ProcessingIndicator) GetElapsed() time.Duration {
	return time.Since(pi.startTime)
}

// IsStale returns true if the indicator has been running for a long time
func (pi *ProcessingIndicator) IsStale() bool {
	return time.Since(pi.startTime) > 60*time.Second
}

// AgentFeedbackManager manages feedback indicators for multiple agents
type AgentFeedbackManager struct {
	indicators map[string]*ProcessingIndicator
	ctx        context.Context
}

// NewAgentFeedbackManager creates a new feedback manager
func NewAgentFeedbackManager(ctx context.Context) *AgentFeedbackManager {
	if ctx == nil {
		ctx = context.Background()
	}

	return &AgentFeedbackManager{
		indicators: make(map[string]*ProcessingIndicator),
		ctx:        ctx,
	}
}

// SetAgentState sets the processing state for an agent
func (afm *AgentFeedbackManager) SetAgentState(agent string, state ProcessingState) error {
	if err := afm.ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("feedback.manager").
			WithOperation("SetAgentState")
	}

	existing, exists := afm.indicators[agent]
	if exists {
		existing.UpdateState(state)
	} else {
		indicator, err := NewProcessingIndicator(afm.ctx, agent, state)
		if err != nil {
			return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create processing indicator").
				WithComponent("feedback.manager").
				WithOperation("SetAgentState").
				WithDetails("agent", agent)
		}
		afm.indicators[agent] = indicator
	}

	return nil
}

// RemoveAgent removes the indicator for an agent
func (afm *AgentFeedbackManager) RemoveAgent(agent string) {
	delete(afm.indicators, agent)
}

// GetIndicator returns the indicator for an agent
func (afm *AgentFeedbackManager) GetIndicator(agent string) (*ProcessingIndicator, bool) {
	indicator, exists := afm.indicators[agent]
	return indicator, exists
}

// TickAll advances all agent indicators
func (afm *AgentFeedbackManager) TickAll() {
	for _, indicator := range afm.indicators {
		indicator.Tick()
	}
}

// GetActiveAgents returns a list of agents with active indicators
func (afm *AgentFeedbackManager) GetActiveAgents() []string {
	agents := make([]string, 0, len(afm.indicators))
	for agent := range afm.indicators {
		agents = append(agents, agent)
	}
	return agents
}

// RemoveStaleIndicators removes indicators that have been running too long
func (afm *AgentFeedbackManager) RemoveStaleIndicators() []string {
	var removed []string
	for agent, indicator := range afm.indicators {
		if indicator.IsStale() {
			delete(afm.indicators, agent)
			removed = append(removed, agent)
		}
	}
	return removed
}

// ViewSummary returns a summary view of all active indicators
func (afm *AgentFeedbackManager) ViewSummary() string {
	if len(afm.indicators) == 0 {
		return ""
	}

	var parts []string
	for _, indicator := range afm.indicators {
		parts = append(parts, indicator.View())
	}

	return strings.Join(parts, " | ")
}
