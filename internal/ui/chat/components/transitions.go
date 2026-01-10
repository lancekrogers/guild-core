// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package components

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/lancekrogers/guild-core/pkg/gerror"
)

// StatusTransition manages smooth transitions between status states
type StatusTransition struct {
	from      string
	to        string
	progress  float64
	duration  time.Duration
	startTime time.Time
	ctx       context.Context
}

// NewStatusTransition creates a new status transition
func NewStatusTransition(ctx context.Context, from, to string, duration time.Duration) (*StatusTransition, error) {
	if ctx == nil {
		return nil, gerror.New(gerror.ErrCodeInvalidInput, "context cannot be nil", nil).
			WithComponent("components.transitions").
			WithOperation("NewStatusTransition")
	}

	if duration <= 0 {
		duration = 500 * time.Millisecond // Default duration
	}

	return &StatusTransition{
		from:      from,
		to:        to,
		progress:  0.0,
		duration:  duration,
		startTime: time.Now(),
		ctx:       ctx,
	}, nil
}

// Update updates the transition progress based on elapsed time
func (st *StatusTransition) Update() error {
	if err := st.ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("components.transitions").
			WithOperation("Update")
	}

	elapsed := time.Since(st.startTime)
	st.progress = math.Min(1.0, float64(elapsed)/float64(st.duration))
	return nil
}

// IsComplete returns true if the transition is complete
func (st *StatusTransition) IsComplete() bool {
	return st.progress >= 1.0
}

// View renders the current transition state
func (st *StatusTransition) View() string {
	if st.progress >= 1.0 {
		return st.to
	}

	return st.renderFadeTransition()
}

// renderFadeTransition renders a fade effect between statuses
func (st *StatusTransition) renderFadeTransition() string {
	// Use different fade characters for visual effect
	fadeChars := []string{"█", "▓", "▒", "░", " "}
	fadeIndex := int(st.progress * float64(len(fadeChars)-1))

	if fadeIndex >= len(fadeChars) {
		fadeIndex = len(fadeChars) - 1
	}

	// Create fade effect
	if st.progress < 0.5 {
		// Fading out from source
		return fmt.Sprintf("%s %s", st.from, fadeChars[fadeIndex])
	} else {
		// Fading in to target
		reverseIndex := len(fadeChars) - 1 - fadeIndex
		return fmt.Sprintf("%s %s", fadeChars[reverseIndex], st.to)
	}
}

// GetProgress returns the current transition progress (0.0 to 1.0)
func (st *StatusTransition) GetProgress() float64 {
	return st.progress
}

// TransitionManager manages multiple status transitions
type TransitionManager struct {
	transitions map[string]*StatusTransition
	ctx         context.Context
}

// NewTransitionManager creates a new transition manager
func NewTransitionManager(ctx context.Context) *TransitionManager {
	if ctx == nil {
		ctx = context.Background()
	}

	return &TransitionManager{
		transitions: make(map[string]*StatusTransition),
		ctx:         ctx,
	}
}

// StartTransition starts a new transition for a given key
func (tm *TransitionManager) StartTransition(key, from, to string, duration time.Duration) error {
	if err := tm.ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("components.transitions").
			WithOperation("StartTransition")
	}

	transition, err := NewStatusTransition(tm.ctx, from, to, duration)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create transition").
			WithComponent("components.transitions").
			WithOperation("StartTransition").
			WithDetails("key", key)
	}

	tm.transitions[key] = transition
	return nil
}

// UpdateAll updates all active transitions
func (tm *TransitionManager) UpdateAll() error {
	if err := tm.ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("components.transitions").
			WithOperation("UpdateAll")
	}

	// Track completed transitions for removal
	var completed []string

	for key, transition := range tm.transitions {
		if err := transition.Update(); err != nil {
			return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to update transition").
				WithComponent("components.transitions").
				WithOperation("UpdateAll").
				WithDetails("key", key)
		}

		if transition.IsComplete() {
			completed = append(completed, key)
		}
	}

	// Remove completed transitions
	for _, key := range completed {
		delete(tm.transitions, key)
	}

	return nil
}

// GetTransition returns the current view for a transition
func (tm *TransitionManager) GetTransition(key string) string {
	transition, exists := tm.transitions[key]
	if !exists {
		return ""
	}

	return transition.View()
}

// IsTransitioning returns true if a transition is active for the key
func (tm *TransitionManager) IsTransitioning(key string) bool {
	_, exists := tm.transitions[key]
	return exists
}

// GetActiveTransitions returns the number of active transitions
func (tm *TransitionManager) GetActiveTransitions() int {
	return len(tm.transitions)
}

// AnimatedText provides smooth text transitions with various effects
type AnimatedText struct {
	text       string
	style      lipgloss.Style
	animation  AnimationType
	frame      int
	speed      time.Duration
	lastUpdate time.Time
	ctx        context.Context
}

// AnimationType represents different animation types
type AnimationType int

const (
	AnimationNone AnimationType = iota
	AnimationTypewriter
	AnimationFade
	AnimationPulse
	AnimationBounce
)

// NewAnimatedText creates a new animated text component
func NewAnimatedText(ctx context.Context, text string, animation AnimationType) (*AnimatedText, error) {
	if ctx == nil {
		return nil, gerror.New(gerror.ErrCodeInvalidInput, "context cannot be nil", nil).
			WithComponent("components.transitions").
			WithOperation("NewAnimatedText")
	}

	return &AnimatedText{
		text:       text,
		style:      lipgloss.NewStyle(),
		animation:  animation,
		frame:      0,
		speed:      100 * time.Millisecond,
		lastUpdate: time.Now(),
		ctx:        ctx,
	}, nil
}

// Update advances the animation frame
func (at *AnimatedText) Update() error {
	if err := at.ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("components.transitions").
			WithOperation("Update")
	}

	now := time.Now()
	if now.Sub(at.lastUpdate) >= at.speed {
		at.frame++
		at.lastUpdate = now
	}

	return nil
}

// View renders the animated text
func (at *AnimatedText) View() string {
	switch at.animation {
	case AnimationTypewriter:
		return at.renderTypewriter()
	case AnimationFade:
		return at.renderFade()
	case AnimationPulse:
		return at.renderPulse()
	case AnimationBounce:
		return at.renderBounce()
	default:
		return at.style.Render(at.text)
	}
}

// renderTypewriter renders typewriter effect
func (at *AnimatedText) renderTypewriter() string {
	visible := at.frame
	if visible > len(at.text) {
		visible = len(at.text)
	}

	// Add cursor effect
	text := at.text[:visible]
	if visible < len(at.text) {
		text += "▐"
	}

	return at.style.Render(text)
}

// renderFade renders fade in/out effect
func (at *AnimatedText) renderFade() string {
	// Cycle through different opacity levels
	cycle := at.frame % 20

	var style lipgloss.Style
	if cycle < 10 {
		// Fading in
		opacity := float64(cycle) / 10.0
		color := interpolateColor("240", "254", opacity)
		style = at.style.Copy().Foreground(lipgloss.Color(color))
	} else {
		// Fading out
		opacity := 1.0 - float64(cycle-10)/10.0
		color := interpolateColor("240", "254", opacity)
		style = at.style.Copy().Foreground(lipgloss.Color(color))
	}

	return style.Render(at.text)
}

// renderPulse renders pulsing effect
func (at *AnimatedText) renderPulse() string {
	// Pulse between normal and bold
	cycle := at.frame % 10

	if cycle < 5 {
		return at.style.Copy().Bold(true).Render(at.text)
	} else {
		return at.style.Render(at.text)
	}
}

// renderBounce renders bouncing effect (simulated with spacing)
func (at *AnimatedText) renderBounce() string {
	cycle := at.frame % 8

	// Add varying amounts of vertical "bounce" using line breaks
	switch cycle {
	case 0, 4:
		return at.style.Render(at.text)
	case 1, 3:
		return at.style.Render(" " + at.text)
	case 2:
		return at.style.Render("  " + at.text)
	default:
		return at.style.Render(at.text)
	}
}

// SetStyle updates the text style
func (at *AnimatedText) SetStyle(style lipgloss.Style) {
	at.style = style
}

// SetSpeed updates the animation speed
func (at *AnimatedText) SetSpeed(speed time.Duration) {
	at.speed = speed
}

// SetText updates the text content
func (at *AnimatedText) SetText(text string) {
	at.text = text
	at.frame = 0 // Reset animation
}

// interpolateColor interpolates between two colors based on progress
func interpolateColor(fromColor, toColor string, progress float64) string {
	// Simple interpolation between gray scale colors
	// This is a simplified version - full implementation would handle RGB

	if progress <= 0 {
		return fromColor
	}
	if progress >= 1 {
		return toColor
	}

	// For simplicity, just return the target color at 50% progress
	if progress >= 0.5 {
		return toColor
	}
	return fromColor
}

// ProgressIndicator provides animated progress visualization
type ProgressIndicator struct {
	progress       float64
	total          float64
	width          int
	style          lipgloss.Style
	showPercentage bool
	ctx            context.Context
}

// NewProgressIndicator creates a new progress indicator
func NewProgressIndicator(ctx context.Context, width int) (*ProgressIndicator, error) {
	if ctx == nil {
		return nil, gerror.New(gerror.ErrCodeInvalidInput, "context cannot be nil", nil).
			WithComponent("components.transitions").
			WithOperation("NewProgressIndicator")
	}

	if width < 10 {
		width = 10
	}

	return &ProgressIndicator{
		progress:       0,
		total:          100,
		width:          width,
		style:          lipgloss.NewStyle().Foreground(lipgloss.Color("82")), // Green
		showPercentage: true,
		ctx:            ctx,
	}, nil
}

// SetProgress updates the current progress
func (pi *ProgressIndicator) SetProgress(current, total float64) {
	pi.progress = current
	pi.total = total
}

// View renders the progress indicator
func (pi *ProgressIndicator) View() string {
	if pi.total == 0 {
		return ""
	}

	ratio := pi.progress / pi.total
	if ratio > 1 {
		ratio = 1
	}
	if ratio < 0 {
		ratio = 0
	}

	// Calculate bar width (accounting for brackets and percentage)
	barWidth := pi.width - 2 // For brackets
	if pi.showPercentage {
		barWidth -= 6 // For " 100%"
	}

	if barWidth < 1 {
		barWidth = 1
	}

	filled := int(ratio * float64(barWidth))

	// Build progress bar
	bar := "["
	for i := 0; i < barWidth; i++ {
		if i < filled {
			bar += "█"
		} else {
			bar += "░"
		}
	}
	bar += "]"

	// Add percentage if enabled
	if pi.showPercentage {
		percentage := int(ratio * 100)
		bar += fmt.Sprintf(" %3d%%", percentage)
	}

	return pi.style.Render(bar)
}

// SetStyle updates the progress bar style
func (pi *ProgressIndicator) SetStyle(style lipgloss.Style) {
	pi.style = style
}

// SetShowPercentage controls whether to show percentage
func (pi *ProgressIndicator) SetShowPercentage(show bool) {
	pi.showPercentage = show
}

// StatusAnimator provides animation for status changes
type StatusAnimator struct {
	currentStatus   string
	targetStatus    string
	transitionMgr   *TransitionManager
	animationFrames []string
	frameIndex      int
	lastUpdate      time.Time
	ctx             context.Context
}

// NewStatusAnimator creates a new status animator
func NewStatusAnimator(ctx context.Context) *StatusAnimator {
	return &StatusAnimator{
		transitionMgr: NewTransitionManager(ctx),
		animationFrames: []string{
			"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏",
		},
		frameIndex: 0,
		lastUpdate: time.Now(),
		ctx:        ctx,
	}
}

// TransitionTo starts a transition to a new status
func (sa *StatusAnimator) TransitionTo(newStatus string, duration time.Duration) error {
	if sa.currentStatus == newStatus {
		return nil // No transition needed
	}

	if err := sa.transitionMgr.StartTransition("main", sa.currentStatus, newStatus, duration); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to start status transition").
			WithComponent("components.transitions").
			WithOperation("TransitionTo")
	}

	sa.targetStatus = newStatus
	return nil
}

// Update updates the animator state
func (sa *StatusAnimator) Update() error {
	if err := sa.transitionMgr.UpdateAll(); err != nil {
		return err
	}

	// Update animation frame
	now := time.Now()
	if now.Sub(sa.lastUpdate) >= 100*time.Millisecond {
		sa.frameIndex = (sa.frameIndex + 1) % len(sa.animationFrames)
		sa.lastUpdate = now
	}

	// Update current status if transition is complete
	if !sa.transitionMgr.IsTransitioning("main") && sa.targetStatus != "" {
		sa.currentStatus = sa.targetStatus
		sa.targetStatus = ""
	}

	return nil
}

// View renders the current status with animation
func (sa *StatusAnimator) View() string {
	// If transitioning, show transition
	if sa.transitionMgr.IsTransitioning("main") {
		return sa.transitionMgr.GetTransition("main")
	}

	// If status indicates activity, show animation
	if strings.Contains(strings.ToLower(sa.currentStatus), "thinking") ||
		strings.Contains(strings.ToLower(sa.currentStatus), "working") ||
		strings.Contains(strings.ToLower(sa.currentStatus), "processing") {
		return fmt.Sprintf("%s %s", sa.animationFrames[sa.frameIndex], sa.currentStatus)
	}

	return sa.currentStatus
}

// GetCurrentStatus returns the current status
func (sa *StatusAnimator) GetCurrentStatus() string {
	return sa.currentStatus
}

// IsTransitioning returns true if a transition is in progress
func (sa *StatusAnimator) IsTransitioning() bool {
	return sa.transitionMgr.IsTransitioning("main")
}

// EasingFunc defines a function that modifies animation progress
type EasingFunc func(t float64) float64

// Common easing functions for smooth 60 FPS animations
var (
	// Linear - no easing, constant speed
	EaseLinear = func(t float64) float64 { return t }

	// Quadratic easing functions
	EaseInQuad    = func(t float64) float64 { return t * t }
	EaseOutQuad   = func(t float64) float64 { return t * (2 - t) }
	EaseInOutQuad = func(t float64) float64 {
		if t < 0.5 {
			return 2 * t * t
		}
		return -1 + (4-2*t)*t
	}

	// Cubic easing functions
	EaseInCubic    = func(t float64) float64 { return t * t * t }
	EaseOutCubic   = func(t float64) float64 { return 1 + (t-1)*(t-1)*(t-1) }
	EaseInOutCubic = func(t float64) float64 {
		if t < 0.5 {
			return 4 * t * t * t
		}
		return 1 + (t-1)*(2*(t-2))*(2*(t-2))
	}

	// Sine easing functions
	EaseInSine    = func(t float64) float64 { return 1 - math.Cos(t*math.Pi/2) }
	EaseOutSine   = func(t float64) float64 { return math.Sin(t * math.Pi / 2) }
	EaseInOutSine = func(t float64) float64 { return 0.5 * (1 - math.Cos(math.Pi*t)) }

	// Elastic easing - spring effect
	EaseOutElastic = func(t float64) float64 {
		if t == 0 || t == 1 {
			return t
		}
		p := 0.3
		return math.Pow(2, -10*t)*math.Sin((t-p/4)*(2*math.Pi)/p) + 1
	}

	// Bounce easing
	EaseOutBounce = func(t float64) float64 {
		if t < 1/2.75 {
			return 7.5625 * t * t
		} else if t < 2/2.75 {
			t -= 1.5 / 2.75
			return 7.5625*t*t + 0.75
		} else if t < 2.5/2.75 {
			t -= 2.25 / 2.75
			return 7.5625*t*t + 0.9375
		}
		t -= 2.625 / 2.75
		return 7.5625*t*t + 0.984375
	}

	// Back easing - overshoot effect
	EaseInBack = func(t float64) float64 {
		s := 1.70158
		return t * t * ((s+1)*t - s)
	}
	EaseOutBack = func(t float64) float64 {
		s := 1.70158
		t = t - 1
		return t*t*((s+1)*t+s) + 1
	}
)

// Animation represents a single animation with easing
type Animation struct {
	ID         string
	StartValue float64
	EndValue   float64
	Current    float64
	Duration   time.Duration
	StartTime  time.Time
	Easing     EasingFunc
	OnUpdate   func(float64)
	OnComplete func()
	ctx        context.Context
}

// AnimationManager manages multiple animations for 60 FPS rendering
type AnimationManager struct {
	animations map[string]*Animation
	ticker     *time.Ticker
	fps        int
	ctx        context.Context
}

// NewAnimationManager creates a new animation manager for 60 FPS animations
func NewAnimationManager(ctx context.Context) *AnimationManager {
	return &AnimationManager{
		animations: make(map[string]*Animation),
		fps:        60,
		ctx:        ctx,
	}
}

// Start begins the animation loop at 60 FPS
func (am *AnimationManager) Start() {
	am.ticker = time.NewTicker(time.Second / time.Duration(am.fps))

	go func() {
		for {
			select {
			case <-am.ticker.C:
				am.update()
			case <-am.ctx.Done():
				am.ticker.Stop()
				return
			}
		}
	}()
}

// Stop halts the animation loop
func (am *AnimationManager) Stop() {
	if am.ticker != nil {
		am.ticker.Stop()
	}
}

// Animate adds a new animation
func (am *AnimationManager) Animate(anim *Animation) {
	if anim.Easing == nil {
		anim.Easing = EaseInOutQuad
	}
	anim.StartTime = time.Now()
	anim.ctx = am.ctx
	am.animations[anim.ID] = anim
}

// update processes all active animations
func (am *AnimationManager) update() {
	now := time.Now()
	completed := []string{}

	for id, anim := range am.animations {
		elapsed := now.Sub(anim.StartTime)
		progress := float64(elapsed) / float64(anim.Duration)

		if progress >= 1.0 {
			// Animation complete
			anim.Current = anim.EndValue
			if anim.OnUpdate != nil {
				anim.OnUpdate(anim.Current)
			}
			if anim.OnComplete != nil {
				anim.OnComplete()
			}
			completed = append(completed, id)
			continue
		}

		// Apply easing
		easedProgress := anim.Easing(progress)

		// Interpolate value
		anim.Current = anim.StartValue + (anim.EndValue-anim.StartValue)*easedProgress

		// Callback
		if anim.OnUpdate != nil {
			anim.OnUpdate(anim.Current)
		}
	}

	// Remove completed animations
	for _, id := range completed {
		delete(am.animations, id)
	}
}

// HasAnimation checks if an animation is active
func (am *AnimationManager) HasAnimation(id string) bool {
	_, exists := am.animations[id]
	return exists
}

// StopAnimation stops a specific animation
func (am *AnimationManager) StopAnimation(id string) {
	delete(am.animations, id)
}
