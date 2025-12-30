package components

import (
	"context"
	"fmt"
	"math"
	"strings"
	"sync"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/guild-framework/guild-core/internal/ui/chat/common/utils"
	"github.com/guild-framework/guild-core/internal/ui/chat/components"
	"github.com/guild-framework/guild-core/pkg/gerror"
	"go.uber.org/zap"
)

// PolishedComponents provides high-quality UI components matching Claude Code standards
type PolishedComponents struct {
	theme      *utils.Styles
	animator   *components.AnimationManager
	feedback   *HapticFeedback
	logger     *zap.Logger
	ctx        context.Context
	cancelFunc context.CancelFunc
	mu         sync.RWMutex
}

// NewPolishedComponents creates a new set of polished UI components
func NewPolishedComponents(ctx context.Context, logger *zap.Logger) (*PolishedComponents, error) {
	if ctx == nil {
		return nil, gerror.New(gerror.ErrCodeInvalidInput, "context cannot be nil", nil).
			WithComponent("ui.polish").
			WithOperation("NewPolishedComponents")
	}

	if logger == nil {
		logger = zap.NewNop()
	}

	// Create cancellable context for component lifecycle
	componentCtx, cancel := context.WithCancel(ctx)

	theme := utils.NewClaudeCodeStyles()

	animator := components.NewAnimationManager(componentCtx)
	if err := startAnimatorWithRecovery(animator, logger); err != nil {
		cancel()
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to start animator").
			WithComponent("ui.polish").
			WithOperation("NewPolishedComponents")
	}

	pc := &PolishedComponents{
		theme:      theme,
		animator:   animator,
		feedback:   NewHapticFeedback(true),
		logger:     logger.With(zap.String("component", "ui.polish")),
		ctx:        componentCtx,
		cancelFunc: cancel,
	}

	pc.logger.Info("polished components initialized",
		zap.String("theme", "claude-code"),
		zap.Bool("animations_enabled", true))

	return pc, nil
}

// startAnimatorWithRecovery starts the animator with panic recovery
func startAnimatorWithRecovery(animator *components.AnimationManager, logger *zap.Logger) (err error) {
	defer func() {
		if r := recover(); r != nil {
			logger.Error("animator start panicked",
				zap.Any("panic", r),
				zap.Stack("stack"))
			err = gerror.New(gerror.ErrCodeInternal, "animator start panicked", nil).
				WithComponent("ui.polish").
				WithOperation("startAnimatorWithRecovery").
				WithDetails("panic", r)
		}
	}()

	animator.Start()
	return nil
}

// Shutdown gracefully shuts down the polished components
func (pc *PolishedComponents) Shutdown(ctx context.Context) error {
	pc.mu.Lock()
	defer pc.mu.Unlock()

	pc.logger.Info("shutting down polished components")

	// Cancel the component context
	pc.cancelFunc()

	// Stop the animator
	if pc.animator != nil {
		pc.animator.Stop()
	}

	pc.logger.Info("polished components shutdown complete")
	return nil
}

// LoadingIndicator provides various loading animations
type LoadingIndicator struct {
	style    LoadingStyle
	message  string
	progress float64
	frame    int
	theme    *utils.Styles
	logger   *zap.Logger
	mu       sync.RWMutex
}

// LoadingStyle defines different loading animation styles
type LoadingStyle int

const (
	LoadingSpinner LoadingStyle = iota
	LoadingProgress
	LoadingDots
	LoadingPulse
	LoadingElastic
)

// String returns the string representation of LoadingStyle
func (ls LoadingStyle) String() string {
	switch ls {
	case LoadingSpinner:
		return "spinner"
	case LoadingProgress:
		return "progress"
	case LoadingDots:
		return "dots"
	case LoadingPulse:
		return "pulse"
	case LoadingElastic:
		return "elastic"
	default:
		return "unknown"
	}
}

// NewLoadingIndicator creates a new loading indicator
func NewLoadingIndicator(ctx context.Context, style LoadingStyle, message string, theme *utils.Styles, logger *zap.Logger) (*LoadingIndicator, error) {
	if err := ctx.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("ui.polish").
			WithOperation("NewLoadingIndicator")
	}

	if theme == nil {
		return nil, gerror.New(gerror.ErrCodeInvalidInput, "theme cannot be nil", nil).
			WithComponent("ui.polish").
			WithOperation("NewLoadingIndicator")
	}

	if logger == nil {
		logger = zap.NewNop()
	}

	li := &LoadingIndicator{
		style:   style,
		message: message,
		theme:   theme,
		logger: logger.With(
			zap.String("component", "LoadingIndicator"),
			zap.String("style", style.String()),
		),
	}

	li.logger.Debug("loading indicator created",
		zap.String("message", message))

	return li, nil
}

// View renders the loading indicator
func (li *LoadingIndicator) View(ctx context.Context) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("ui.polish").
			WithOperation("LoadingIndicator.View")
	}

	li.mu.RLock()
	defer li.mu.RUnlock()

	var result string
	var err error

	switch li.style {
	case LoadingSpinner:
		result = li.renderSpinner()
	case LoadingProgress:
		result = li.renderProgress()
	case LoadingDots:
		result = li.renderDots()
	case LoadingPulse:
		result = li.renderPulse()
	case LoadingElastic:
		result = li.renderElastic()
	default:
		err = gerror.New(gerror.ErrCodeInvalidInput, "unknown loading style", nil).
			WithComponent("ui.polish").
			WithOperation("LoadingIndicator.View").
			WithDetails("style", li.style)
	}

	if err != nil {
		return "", err
	}

	// Log every 100th frame for performance monitoring
	if li.frame%100 == 0 {
		li.logger.Debug("loading indicator frame",
			zap.Int("frame", li.frame),
			zap.Float64("progress", li.progress))
	}

	return result, nil
}

// renderSpinner creates a smooth spinning animation
func (li *LoadingIndicator) renderSpinner() string {
	// Use braille characters for smooth rotation
	frames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	frame := frames[li.frame%len(frames)]

	spinnerStyle := li.theme.StatusInfo
	messageStyle := li.theme.Base

	return fmt.Sprintf("%s %s",
		spinnerStyle.Render(frame),
		messageStyle.Render(li.message))
}

// renderProgress creates a progress bar with percentage
func (li *LoadingIndicator) renderProgress() string {
	barWidth := 30
	filled := int(li.progress * float64(barWidth))

	bar := strings.Builder{}
	bar.WriteString("[")

	for i := 0; i < barWidth; i++ {
		if i < filled {
			bar.WriteString("█")
		} else if i == filled {
			bar.WriteString("▓")
		} else {
			bar.WriteString("░")
		}
	}

	bar.WriteString("]")

	progressStyle := li.theme.StatusInfo
	percentage := int(li.progress * 100)

	return fmt.Sprintf("%s %3d%% %s",
		progressStyle.Render(bar.String()),
		percentage,
		li.theme.Base.Render(li.message))
}

// renderDots creates an animated dot sequence
func (li *LoadingIndicator) renderDots() string {
	dots := li.frame % 4
	dotString := strings.Repeat("●", dots) + strings.Repeat("○", 3-dots)

	return fmt.Sprintf("%s %s",
		li.theme.Base.Render(li.message),
		li.theme.StatusInfo.Render(dotString))
}

// renderPulse creates a pulsing effect
func (li *LoadingIndicator) renderPulse() string {
	pulseChars := []string{"◯", "◉", "●", "◉"}
	char := pulseChars[li.frame%len(pulseChars)]

	return fmt.Sprintf("%s %s",
		li.theme.StatusInfo.Bold(li.frame%2 == 0).Render(char),
		li.theme.Base.Render(li.message))
}

// renderElastic creates an elastic loading animation
func (li *LoadingIndicator) renderElastic() string {
	// Elastic bounce effect
	phase := float64(li.frame%20) / 20.0
	bounce := math.Sin(phase * math.Pi * 2)
	spaces := int(math.Abs(bounce * 5))

	return fmt.Sprintf("%s%s %s",
		strings.Repeat(" ", spaces),
		li.theme.StatusInfo.Render("●"),
		li.theme.Base.Render(li.message))
}

// Update advances the animation frame
func (li *LoadingIndicator) Update(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("ui.polish").
			WithOperation("LoadingIndicator.Update")
	}

	li.mu.Lock()
	defer li.mu.Unlock()

	li.frame++

	// Performance monitoring - warn if frame rate drops
	if li.frame%60 == 0 {
		li.logger.Debug("animation performance check",
			zap.Int("frame", li.frame),
			zap.String("style", li.style.String()))
	}

	return nil
}

// SetProgress updates the progress for progress bar style
func (li *LoadingIndicator) SetProgress(ctx context.Context, progress float64) error {
	if err := ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("ui.polish").
			WithOperation("LoadingIndicator.SetProgress")
	}

	li.mu.Lock()
	defer li.mu.Unlock()

	oldProgress := li.progress
	li.progress = math.Max(0, math.Min(1, progress))

	// Log significant progress changes
	if math.Abs(li.progress-oldProgress) >= 0.1 {
		li.logger.Debug("progress updated",
			zap.Float64("old", oldProgress),
			zap.Float64("new", li.progress))
	}

	return nil
}

// Tooltip provides context-sensitive help text
type Tooltip struct {
	content   string
	position  TooltipPosition
	visible   bool
	delay     time.Duration
	showTimer *time.Timer
	theme     *utils.Styles
	logger    *zap.Logger
	mu        sync.Mutex
}

// TooltipPosition defines where the tooltip appears
type TooltipPosition int

const (
	TooltipAbove TooltipPosition = iota
	TooltipBelow
	TooltipLeft
	TooltipRight
	TooltipAuto
)

// String returns the string representation of TooltipPosition
func (tp TooltipPosition) String() string {
	switch tp {
	case TooltipAbove:
		return "above"
	case TooltipBelow:
		return "below"
	case TooltipLeft:
		return "left"
	case TooltipRight:
		return "right"
	case TooltipAuto:
		return "auto"
	default:
		return "unknown"
	}
}

// NewTooltip creates a new tooltip
func NewTooltip(ctx context.Context, content string, position TooltipPosition, theme *utils.Styles, logger *zap.Logger) (*Tooltip, error) {
	if err := ctx.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("ui.polish").
			WithOperation("NewTooltip")
	}

	if theme == nil {
		return nil, gerror.New(gerror.ErrCodeInvalidInput, "theme cannot be nil", nil).
			WithComponent("ui.polish").
			WithOperation("NewTooltip")
	}

	if logger == nil {
		logger = zap.NewNop()
	}

	t := &Tooltip{
		content:  content,
		position: position,
		delay:    500 * time.Millisecond,
		theme:    theme,
		logger: logger.With(
			zap.String("component", "Tooltip"),
			zap.String("position", position.String()),
		),
	}

	t.logger.Debug("tooltip created",
		zap.String("content", content),
		zap.Duration("delay", t.delay))

	return t, nil
}

// Show displays the tooltip after a delay
func (t *Tooltip) Show(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("ui.polish").
			WithOperation("Tooltip.Show")
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	// Clean up existing timer
	if t.showTimer != nil {
		t.showTimer.Stop()
	}

	// Create timer with context awareness
	t.showTimer = time.AfterFunc(t.delay, func() {
		t.mu.Lock()
		defer t.mu.Unlock()

		// Check context before showing
		if ctx.Err() == nil {
			t.visible = true
			t.logger.Debug("tooltip shown",
				zap.String("content", t.content))
		}
	})

	t.logger.Debug("tooltip show scheduled",
		zap.Duration("delay", t.delay))

	return nil
}

// Hide immediately hides the tooltip
func (t *Tooltip) Hide(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("ui.polish").
			WithOperation("Tooltip.Hide")
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	if t.showTimer != nil {
		t.showTimer.Stop()
		t.showTimer = nil
	}

	wasVisible := t.visible
	t.visible = false

	if wasVisible {
		t.logger.Debug("tooltip hidden",
			zap.String("content", t.content))
	}

	return nil
}

// View renders the tooltip if visible
func (t *Tooltip) View(ctx context.Context) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("ui.polish").
			WithOperation("Tooltip.View")
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	if !t.visible || t.content == "" {
		return "", nil
	}

	// Create tooltip style with rounded corners and shadow effect
	tooltipStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("#252526")). // Claude Code surface color
		Foreground(lipgloss.Color("#d4d4d4")). // Claude Code text color
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#007acc")). // Claude Code primary color
		Padding(0, 1).
		MarginTop(1)

	return tooltipStyle.Render(t.content), nil
}

// Cleanup releases resources
func (t *Tooltip) Cleanup() {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.showTimer != nil {
		t.showTimer.Stop()
		t.showTimer = nil
	}

	t.logger.Debug("tooltip cleanup complete")
}

// StatusIndicator shows status with smooth animations
type StatusIndicator struct {
	status   Status
	message  string
	animated bool
	frame    int
	theme    *utils.Styles
	logger   *zap.Logger
	mu       sync.RWMutex
}

// Status represents different status states
type Status int

const (
	StatusIdle Status = iota
	StatusLoading
	StatusSuccess
	StatusWarning
	StatusError
	StatusInfo
)

// String returns the string representation of Status
func (s Status) String() string {
	switch s {
	case StatusIdle:
		return "idle"
	case StatusLoading:
		return "loading"
	case StatusSuccess:
		return "success"
	case StatusWarning:
		return "warning"
	case StatusError:
		return "error"
	case StatusInfo:
		return "info"
	default:
		return "unknown"
	}
}

// NewStatusIndicator creates a new status indicator
func NewStatusIndicator(ctx context.Context, theme *utils.Styles, logger *zap.Logger) (*StatusIndicator, error) {
	if err := ctx.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("ui.polish").
			WithOperation("NewStatusIndicator")
	}

	if theme == nil {
		return nil, gerror.New(gerror.ErrCodeInvalidInput, "theme cannot be nil", nil).
			WithComponent("ui.polish").
			WithOperation("NewStatusIndicator")
	}

	if logger == nil {
		logger = zap.NewNop()
	}

	si := &StatusIndicator{
		status:   StatusIdle,
		animated: true,
		theme:    theme,
		logger: logger.With(
			zap.String("component", "StatusIndicator"),
		),
	}

	si.logger.Debug("status indicator created")

	return si, nil
}

// SetStatus updates the current status
func (si *StatusIndicator) SetStatus(ctx context.Context, status Status, message string) error {
	if err := ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("ui.polish").
			WithOperation("StatusIndicator.SetStatus")
	}

	si.mu.Lock()
	defer si.mu.Unlock()

	oldStatus := si.status
	si.status = status
	si.message = message
	si.frame = 0

	si.logger.Info("status changed",
		zap.String("old_status", oldStatus.String()),
		zap.String("new_status", status.String()),
		zap.String("message", message))

	return nil
}

// View renders the status indicator
func (si *StatusIndicator) View(ctx context.Context) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("ui.polish").
			WithOperation("StatusIndicator.View")
	}

	si.mu.RLock()
	defer si.mu.RUnlock()

	icon := si.getIcon()
	style := si.getStyle()

	if si.animated && si.status == StatusLoading {
		return si.renderAnimated(icon, style), nil
	}

	return fmt.Sprintf("%s %s",
		style.Render(icon),
		si.theme.Base.Render(si.message)), nil
}

// getIcon returns the icon for the current status
func (si *StatusIndicator) getIcon() string {
	switch si.status {
	case StatusLoading:
		frames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
		return frames[si.frame%len(frames)]
	case StatusSuccess:
		return "✓"
	case StatusWarning:
		return "⚠"
	case StatusError:
		return "✗"
	case StatusInfo:
		return "ℹ"
	default:
		return "•"
	}
}

// getStyle returns the style for the current status
func (si *StatusIndicator) getStyle() lipgloss.Style {
	switch si.status {
	case StatusSuccess:
		return si.theme.StatusSuccess
	case StatusWarning:
		return si.theme.StatusWarning
	case StatusError:
		return si.theme.StatusError
	case StatusInfo:
		return si.theme.StatusInfo
	default:
		return si.theme.Base
	}
}

// renderAnimated creates animated status display
func (si *StatusIndicator) renderAnimated(icon string, style lipgloss.Style) string {
	// Pulse effect for loading
	if si.frame%10 < 5 {
		style = style.Bold(true)
	}

	return fmt.Sprintf("%s %s",
		style.Render(icon),
		si.theme.Base.Render(si.message))
}

// Update advances the animation
func (si *StatusIndicator) Update(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("ui.polish").
			WithOperation("StatusIndicator.Update")
	}

	si.mu.Lock()
	defer si.mu.Unlock()

	si.frame++

	// Log animation performance
	if si.frame%60 == 0 && si.status == StatusLoading {
		si.logger.Debug("status animation frame",
			zap.Int("frame", si.frame),
			zap.String("status", si.status.String()))
	}

	return nil
}

// HapticFeedback provides visual feedback for user actions
type HapticFeedback struct {
	enabled       bool
	flashDuration time.Duration
	logger        *zap.Logger
	mu            sync.RWMutex
}

// NewHapticFeedback creates a new haptic feedback manager
func NewHapticFeedback(enabled bool) *HapticFeedback {
	return &HapticFeedback{
		enabled:       enabled,
		flashDuration: 200 * time.Millisecond,
		logger:        zap.NewNop(),
	}
}

// NewHapticFeedbackWithLogger creates a haptic feedback manager with logging
func NewHapticFeedbackWithLogger(ctx context.Context, enabled bool, logger *zap.Logger) (*HapticFeedback, error) {
	if err := ctx.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("ui.polish").
			WithOperation("NewHapticFeedbackWithLogger")
	}

	if logger == nil {
		logger = zap.NewNop()
	}

	hf := &HapticFeedback{
		enabled:       enabled,
		flashDuration: 200 * time.Millisecond,
		logger: logger.With(
			zap.String("component", "HapticFeedback"),
		),
	}

	hf.logger.Debug("haptic feedback created",
		zap.Bool("enabled", enabled),
		zap.Duration("flash_duration", hf.flashDuration))

	return hf, nil
}

// Success provides positive feedback
func (hf *HapticFeedback) Success(ctx context.Context) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("ui.polish").
			WithOperation("HapticFeedback.Success")
	}

	hf.mu.RLock()
	defer hf.mu.RUnlock()

	if !hf.enabled {
		return "", nil
	}

	// Green flash effect
	flashStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("#3fb950")).
		Foreground(lipgloss.Color("#ffffff")).
		Bold(true).
		Padding(0, 1)

	hf.logger.Debug("success feedback triggered")

	return flashStyle.Render("✓ Success"), nil
}

// Error provides error feedback
func (hf *HapticFeedback) Error(ctx context.Context) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("ui.polish").
			WithOperation("HapticFeedback.Error")
	}

	hf.mu.RLock()
	defer hf.mu.RUnlock()

	if !hf.enabled {
		return "", nil
	}

	// Red shake effect
	flashStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("#f85149")).
		Foreground(lipgloss.Color("#ffffff")).
		Bold(true).
		Padding(0, 1)

	hf.logger.Debug("error feedback triggered")

	return flashStyle.Render("✗ Error"), nil
}

// Info provides informational feedback
func (hf *HapticFeedback) Info(ctx context.Context) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("ui.polish").
			WithOperation("HapticFeedback.Info")
	}

	hf.mu.RLock()
	defer hf.mu.RUnlock()

	if !hf.enabled {
		return "", nil
	}

	// Blue highlight effect
	flashStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("#58a6ff")).
		Foreground(lipgloss.Color("#ffffff")).
		Padding(0, 1)

	hf.logger.Debug("info feedback triggered")

	return flashStyle.Render("ℹ Info"), nil
}

// SetEnabled updates the enabled state
func (hf *HapticFeedback) SetEnabled(ctx context.Context, enabled bool) error {
	if err := ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("ui.polish").
			WithOperation("HapticFeedback.SetEnabled")
	}

	hf.mu.Lock()
	defer hf.mu.Unlock()

	oldEnabled := hf.enabled
	hf.enabled = enabled

	hf.logger.Info("haptic feedback state changed",
		zap.Bool("old_enabled", oldEnabled),
		zap.Bool("new_enabled", enabled))

	return nil
}

// MicroInteractions provides subtle UI enhancements
type MicroInteractions struct {
	animator *components.AnimationManager
	theme    *utils.Styles
	logger   *zap.Logger
	mu       sync.RWMutex
}

// NewMicroInteractions creates a new micro-interactions manager
func NewMicroInteractions(ctx context.Context, animator *components.AnimationManager, theme *utils.Styles, logger *zap.Logger) (*MicroInteractions, error) {
	if err := ctx.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("ui.polish").
			WithOperation("NewMicroInteractions")
	}

	if animator == nil {
		return nil, gerror.New(gerror.ErrCodeInvalidInput, "animator cannot be nil", nil).
			WithComponent("ui.polish").
			WithOperation("NewMicroInteractions")
	}

	if theme == nil {
		return nil, gerror.New(gerror.ErrCodeInvalidInput, "theme cannot be nil", nil).
			WithComponent("ui.polish").
			WithOperation("NewMicroInteractions")
	}

	if logger == nil {
		logger = zap.NewNop()
	}

	mi := &MicroInteractions{
		animator: animator,
		theme:    theme,
		logger: logger.With(
			zap.String("component", "MicroInteractions"),
		),
	}

	mi.logger.Debug("micro interactions created")

	return mi, nil
}

// OnMessageSent creates a subtle bounce effect
func (mi *MicroInteractions) OnMessageSent(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("ui.polish").
			WithOperation("MicroInteractions.OnMessageSent")
	}

	mi.mu.Lock()
	defer mi.mu.Unlock()

	mi.animator.Animate(&components.Animation{
		ID:         "message-sent",
		StartValue: 1.0,
		EndValue:   1.05,
		Duration:   300 * time.Millisecond,
		Easing:     components.EaseOutBounce,
		OnUpdate: func(scale float64) {
			// This would update the UI scale in the actual implementation
			if int(scale*100)%10 == 0 {
				mi.logger.Debug("message sent animation",
					zap.Float64("scale", scale))
			}
		},
		OnComplete: func() {
			// Reset scale after animation
			mi.logger.Debug("message sent animation complete")
		},
	})

	mi.logger.Info("message sent interaction triggered")

	return nil
}

// OnAgentTyping creates a smooth fade-in effect
func (mi *MicroInteractions) OnAgentTyping(ctx context.Context, agentID string) error {
	if err := ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("ui.polish").
			WithOperation("MicroInteractions.OnAgentTyping")
	}

	if agentID == "" {
		return gerror.New(gerror.ErrCodeInvalidInput, "agentID cannot be empty", nil).
			WithComponent("ui.polish").
			WithOperation("MicroInteractions.OnAgentTyping")
	}

	mi.mu.Lock()
	defer mi.mu.Unlock()

	mi.animator.Animate(&components.Animation{
		ID:         fmt.Sprintf("typing-%s", agentID),
		StartValue: 0.0,
		EndValue:   1.0,
		Duration:   200 * time.Millisecond,
		Easing:     components.EaseInQuad,
		OnUpdate: func(opacity float64) {
			// This would update typing indicator opacity
			if int(opacity*100)%25 == 0 {
				mi.logger.Debug("agent typing animation",
					zap.String("agent", agentID),
					zap.Float64("opacity", opacity))
			}
		},
	})

	mi.logger.Info("agent typing interaction triggered",
		zap.String("agent", agentID))

	return nil
}

// OnFocusChange creates a highlight effect
func (mi *MicroInteractions) OnFocusChange(ctx context.Context, elementID string) error {
	if err := ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("ui.polish").
			WithOperation("MicroInteractions.OnFocusChange")
	}

	if elementID == "" {
		return gerror.New(gerror.ErrCodeInvalidInput, "elementID cannot be empty", nil).
			WithComponent("ui.polish").
			WithOperation("MicroInteractions.OnFocusChange")
	}

	mi.mu.Lock()
	defer mi.mu.Unlock()

	mi.animator.Animate(&components.Animation{
		ID:         fmt.Sprintf("focus-%s", elementID),
		StartValue: 0.0,
		EndValue:   1.0,
		Duration:   150 * time.Millisecond,
		Easing:     components.EaseOutSine,
		OnUpdate: func(progress float64) {
			// This would update border highlight
			if int(progress*100)%50 == 0 {
				mi.logger.Debug("focus change animation",
					zap.String("element", elementID),
					zap.Float64("progress", progress))
			}
		},
	})

	mi.logger.Info("focus change interaction triggered",
		zap.String("element", elementID))

	return nil
}

// AccessibilityManager handles accessibility features
type AccessibilityManager struct {
	highContrast  bool
	reducedMotion bool
	screenReader  bool
	fontSize      int
	logger        *zap.Logger
	mu            sync.RWMutex
}

// NewAccessibilityManager creates a new accessibility manager
func NewAccessibilityManager(ctx context.Context, logger *zap.Logger) (*AccessibilityManager, error) {
	if err := ctx.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("ui.polish").
			WithOperation("NewAccessibilityManager")
	}

	if logger == nil {
		logger = zap.NewNop()
	}

	am := &AccessibilityManager{
		fontSize: 14,
		logger: logger.With(
			zap.String("component", "AccessibilityManager"),
		),
	}

	am.logger.Debug("accessibility manager created",
		zap.Int("default_font_size", am.fontSize))

	return am, nil
}

// EnableHighContrast increases color contrast
func (am *AccessibilityManager) EnableHighContrast(ctx context.Context, theme *utils.Styles) error {
	if err := ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("ui.polish").
			WithOperation("AccessibilityManager.EnableHighContrast")
	}

	if theme == nil {
		return gerror.New(gerror.ErrCodeInvalidInput, "theme cannot be nil", nil).
			WithComponent("ui.polish").
			WithOperation("AccessibilityManager.EnableHighContrast")
	}

	am.mu.Lock()
	defer am.mu.Unlock()

	am.highContrast = true
	// In a real implementation, this would modify theme colors
	// For now, just log the action
	am.logger.Debug("theme contrast enhancement requested")

	am.logger.Info("high contrast mode enabled")

	return nil
}

// EnableReducedMotion disables animations
func (am *AccessibilityManager) EnableReducedMotion(ctx context.Context, animator *components.AnimationManager) error {
	if err := ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("ui.polish").
			WithOperation("AccessibilityManager.EnableReducedMotion")
	}

	am.mu.Lock()
	defer am.mu.Unlock()

	am.reducedMotion = true

	if animator != nil {
		animator.Stop()
		am.logger.Info("animations disabled for reduced motion")
	}

	am.logger.Info("reduced motion mode enabled")

	return nil
}

// EnableScreenReader adds screen reader support
func (am *AccessibilityManager) EnableScreenReader(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("ui.polish").
			WithOperation("AccessibilityManager.EnableScreenReader")
	}

	am.mu.Lock()
	defer am.mu.Unlock()

	am.screenReader = true
	// In a real implementation, this would add ARIA-like labels

	am.logger.Info("screen reader mode enabled")

	return nil
}

// SetFontSize adjusts the base font size
func (am *AccessibilityManager) SetFontSize(ctx context.Context, size int) error {
	if err := ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("ui.polish").
			WithOperation("AccessibilityManager.SetFontSize")
	}

	if size < 8 || size > 48 {
		return gerror.New(gerror.ErrCodeInvalidInput, "font size must be between 8 and 48", nil).
			WithComponent("ui.polish").
			WithOperation("AccessibilityManager.SetFontSize").
			WithDetails("size", size)
	}

	am.mu.Lock()
	defer am.mu.Unlock()

	oldSize := am.fontSize
	am.fontSize = size

	am.logger.Info("font size changed",
		zap.Int("old_size", oldSize),
		zap.Int("new_size", size))

	return nil
}

// GetAccessibilityInfo returns current accessibility settings
func (am *AccessibilityManager) GetAccessibilityInfo(ctx context.Context) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("ui.polish").
			WithOperation("AccessibilityManager.GetAccessibilityInfo")
	}

	am.mu.RLock()
	defer am.mu.RUnlock()

	var info []string

	if am.highContrast {
		info = append(info, "High Contrast: ON")
	} else {
		info = append(info, "High Contrast: OFF")
	}

	if am.reducedMotion {
		info = append(info, "Reduced Motion: ON")
	} else {
		info = append(info, "Reduced Motion: OFF")
	}

	if am.screenReader {
		info = append(info, "Screen Reader: ON")
	} else {
		info = append(info, "Screen Reader: OFF")
	}

	info = append(info, fmt.Sprintf("Font Size: %d", am.fontSize))

	result := strings.Join(info, "\n")

	am.logger.Debug("accessibility info requested",
		zap.String("info", result))

	return result, nil
}

// IsHighContrastEnabled returns whether high contrast mode is enabled
func (am *AccessibilityManager) IsHighContrastEnabled() bool {
	am.mu.RLock()
	defer am.mu.RUnlock()
	return am.highContrast
}

// IsReducedMotionEnabled returns whether reduced motion is enabled
func (am *AccessibilityManager) IsReducedMotionEnabled() bool {
	am.mu.RLock()
	defer am.mu.RUnlock()
	return am.reducedMotion
}

// IsScreenReaderEnabled returns whether screen reader mode is enabled
func (am *AccessibilityManager) IsScreenReaderEnabled() bool {
	am.mu.RLock()
	defer am.mu.RUnlock()
	return am.screenReader
}

// GetFontSize returns the current font size
func (am *AccessibilityManager) GetFontSize() int {
	am.mu.RLock()
	defer am.mu.RUnlock()
	return am.fontSize
}
