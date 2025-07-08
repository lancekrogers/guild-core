// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

// Package animation provides comprehensive animation framework for Guild Framework UI
//
// This package implements the animation requirements identified in performance optimization,
// Agent 1 task, providing:
//   - Smooth transitions and animations for UI components
//   - Performance-optimized animation system with 60fps targeting
//   - Easing functions and timeline management
//   - Integration with theme system for consistent motion design
//
// The package follows Guild's architectural patterns:
//   - Context-first error handling with gerror
//   - Interface-driven design for testability
//   - Registry pattern for animation plugins
//   - Observability integration
//
// Example usage:
//
//	// Create animator
//	animator := NewAnimator()
//
//	// Animate component opacity
//	err := animator.Animate(ctx, "fade-in", AnimationOptions{
//		Duration: 300 * time.Millisecond,
//		Easing:   EaseInOut,
//		From:     map[string]interface{}{"opacity": 0.0},
//		To:       map[string]interface{}{"opacity": 1.0},
//	})
//
//	// Create timeline with multiple animations
//	timeline := animator.CreateTimeline("ui-transition")
//	timeline.AddAnimation("fade-out", 0, fadeOutOptions)
//	timeline.AddAnimation("slide-in", 200*time.Millisecond, slideInOptions)
//	timeline.Play(ctx)
package animation

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/lancekrogers/guild/pkg/gerror"
	"go.uber.org/zap"
)

// Package version for compatibility tracking
const (
	Version     = "1.0.0"
	APIVersion  = "v1"
	PackageName = "animation"
)

// Animator manages animations and provides smooth UI transitions
type Animator struct {
	activeAnimations map[string]*ActiveAnimation
	timelines        map[string]*Timeline
	registry         AnimationRegistry
	ticker           *time.Ticker
	running          bool
	mu               sync.RWMutex
	logger           *zap.Logger
}

// Animation represents a single animation configuration
type Animation struct {
	ID         string                 `json:"id"`
	Name       string                 `json:"name"`
	Duration   time.Duration          `json:"duration"`
	Delay      time.Duration          `json:"delay"`
	Easing     EasingFunction         `json:"easing"`
	From       map[string]interface{} `json:"from"`
	To         map[string]interface{} `json:"to"`
	Loop       LoopConfig             `json:"loop"`
	OnUpdate   UpdateCallback         `json:"-"`
	OnComplete CompletionCallback     `json:"-"`
}

// ActiveAnimation tracks a running animation
type ActiveAnimation struct {
	Animation   *Animation
	StartTime   time.Time
	CurrentTime time.Time
	Progress    float64
	State       AnimationState
	Context     context.Context
	CancelFunc  context.CancelFunc
}

// Timeline manages multiple coordinated animations
type Timeline struct {
	ID         string                     `json:"id"`
	Name       string                     `json:"name"`
	Duration   time.Duration              `json:"duration"`
	Animations []*TimelineAnimation       `json:"animations"`
	State      TimelineState              `json:"state"`
	OnComplete TimelineCompletionCallback `json:"-"`
	Context    context.Context            `json:"-"`
	CancelFunc context.CancelFunc         `json:"-"`
}

// TimelineAnimation represents an animation within a timeline
type TimelineAnimation struct {
	Animation *Animation    `json:"animation"`
	StartTime time.Duration `json:"start_time"`
	EndTime   time.Duration `json:"end_time"`
}

// AnimationOptions configures animation behavior
type AnimationOptions struct {
	Duration   time.Duration          `json:"duration"`
	Delay      time.Duration          `json:"delay"`
	Easing     EasingFunction         `json:"easing"`
	From       map[string]interface{} `json:"from"`
	To         map[string]interface{} `json:"to"`
	Loop       LoopConfig             `json:"loop"`
	OnUpdate   UpdateCallback         `json:"-"`
	OnComplete CompletionCallback     `json:"-"`
}

// TimelineOptions configures timeline behavior
type TimelineOptions struct {
	Name       string                     `json:"name"`
	OnComplete TimelineCompletionCallback `json:"-"`
}

// LoopConfig defines animation looping behavior
type LoopConfig struct {
	Enabled  bool `json:"enabled"`
	Count    int  `json:"count"`     // -1 for infinite
	Reverse  bool `json:"reverse"`   // Alternate direction
	PingPong bool `json:"ping_pong"` // Forward then backward
}

// AnimationState represents the current state of an animation
type AnimationState int

const (
	AnimationStateIdle AnimationState = iota
	AnimationStatePlaying
	AnimationStatePaused
	AnimationStateComplete
	AnimationStateCancelled
)

// TimelineState represents the current state of a timeline
type TimelineState int

const (
	TimelineStateIdle TimelineState = iota
	TimelineStatePlaying
	TimelineStatePaused
	TimelineStateComplete
	TimelineStateCancelled
)

// EasingFunction defines easing curve types
type EasingFunction int

const (
	Linear EasingFunction = iota
	EaseIn
	EaseOut
	EaseInOut
	EaseInQuad
	EaseOutQuad
	EaseInOutQuad
	EaseInCubic
	EaseOutCubic
	EaseInOutCubic
	EaseInQuart
	EaseOutQuart
	EaseInOutQuart
	EaseInBack
	EaseOutBack
	EaseInOutBack
	EaseInElastic
	EaseOutElastic
	EaseInOutElastic
	EaseInBounce
	EaseOutBounce
	EaseInOutBounce
)

// Callbacks for animation events
type UpdateCallback func(progress float64, values map[string]interface{}) error
type CompletionCallback func(animation *Animation) error
type TimelineCompletionCallback func(timeline *Timeline) error

// AnimationRegistry manages animation presets and plugins
type AnimationRegistry interface {
	RegisterAnimation(name string, animation *Animation) error
	GetAnimation(name string) (*Animation, error)
	ListAnimations() []string
}

// NewAnimator creates a new animation system
func NewAnimator() *Animator {
	logger, _ := zap.NewDevelopment()

	animator := &Animator{
		activeAnimations: make(map[string]*ActiveAnimation),
		timelines:        make(map[string]*Timeline),
		registry:         NewDefaultAnimationRegistry(),
		ticker:           time.NewTicker(16 * time.Millisecond), // ~60fps
		running:          false,
		logger:           logger.Named("animator"),
	}

	// Register built-in animations
	animator.registerBuiltinAnimations()

	return animator
}

// Start begins the animation loop
func (a *Animator) Start(ctx context.Context) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.running {
		return nil
	}

	a.running = true
	go a.animationLoop(ctx)

	a.logger.Info("Animation system started")
	return nil
}

// Stop halts the animation system
func (a *Animator) Stop(ctx context.Context) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if !a.running {
		return nil
	}

	a.running = false
	a.ticker.Stop()

	// Cancel all active animations
	for _, anim := range a.activeAnimations {
		if anim.CancelFunc != nil {
			anim.CancelFunc()
		}
	}

	// Cancel all active timelines
	for _, timeline := range a.timelines {
		if timeline.CancelFunc != nil {
			timeline.CancelFunc()
		}
	}

	a.logger.Info("Animation system stopped")
	return nil
}

// Animate starts a new animation
func (a *Animator) Animate(ctx context.Context, animationID string, options AnimationOptions) error {
	animation := &Animation{
		ID:         animationID,
		Name:       animationID,
		Duration:   options.Duration,
		Delay:      options.Delay,
		Easing:     options.Easing,
		From:       options.From,
		To:         options.To,
		Loop:       options.Loop,
		OnUpdate:   options.OnUpdate,
		OnComplete: options.OnComplete,
	}

	return a.playAnimation(ctx, animation)
}

// AnimatePreset plays a predefined animation
func (a *Animator) AnimatePreset(ctx context.Context, presetName string, options AnimationOptions) error {
	preset, err := a.registry.GetAnimation(presetName)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeNotFound, "animation preset not found").
			WithComponent("animator").
			WithOperation("AnimatePreset").
			WithDetails("preset", presetName)
	}

	// Override preset with provided options
	animation := *preset
	if options.Duration > 0 {
		animation.Duration = options.Duration
	}
	if options.Delay > 0 {
		animation.Delay = options.Delay
	}
	if options.From != nil {
		animation.From = options.From
	}
	if options.To != nil {
		animation.To = options.To
	}
	if options.OnUpdate != nil {
		animation.OnUpdate = options.OnUpdate
	}
	if options.OnComplete != nil {
		animation.OnComplete = options.OnComplete
	}

	return a.playAnimation(ctx, &animation)
}

// CreateTimeline creates a new animation timeline
func (a *Animator) CreateTimeline(timelineID string, options TimelineOptions) *Timeline {
	timeline := &Timeline{
		ID:         timelineID,
		Name:       options.Name,
		Animations: make([]*TimelineAnimation, 0),
		State:      TimelineStateIdle,
		OnComplete: options.OnComplete,
	}

	a.mu.Lock()
	a.timelines[timelineID] = timeline
	a.mu.Unlock()

	return timeline
}

// AddAnimation adds an animation to a timeline
func (t *Timeline) AddAnimation(animationID string, startOffset time.Duration, options AnimationOptions) {
	animation := &Animation{
		ID:         fmt.Sprintf("%s_%s", t.ID, animationID),
		Name:       animationID,
		Duration:   options.Duration,
		Delay:      options.Delay,
		Easing:     options.Easing,
		From:       options.From,
		To:         options.To,
		Loop:       options.Loop,
		OnUpdate:   options.OnUpdate,
		OnComplete: options.OnComplete,
	}

	timelineAnim := &TimelineAnimation{
		Animation: animation,
		StartTime: startOffset,
		EndTime:   startOffset + animation.Duration + animation.Delay,
	}

	t.Animations = append(t.Animations, timelineAnim)

	// Update timeline duration
	if timelineAnim.EndTime > t.Duration {
		t.Duration = timelineAnim.EndTime
	}
}

// Play starts a timeline
func (a *Animator) PlayTimeline(ctx context.Context, timelineID string) error {
	a.mu.RLock()
	timeline, exists := a.timelines[timelineID]
	a.mu.RUnlock()

	if !exists {
		return gerror.New(gerror.ErrCodeNotFound, fmt.Sprintf("timeline '%s' not found", timelineID), nil).
			WithComponent("animator").
			WithOperation("PlayTimeline")
	}

	timelineCtx, cancelFunc := context.WithCancel(ctx)
	timeline.Context = timelineCtx
	timeline.CancelFunc = cancelFunc
	timeline.State = TimelineStatePlaying

	go a.executeTimeline(timelineCtx, timeline)

	a.logger.Info("Timeline started",
		zap.String("timeline", timelineID),
		zap.Duration("duration", timeline.Duration))

	return nil
}

// playAnimation executes a single animation
func (a *Animator) playAnimation(ctx context.Context, animation *Animation) error {
	animCtx, cancelFunc := context.WithCancel(ctx)

	activeAnim := &ActiveAnimation{
		Animation:  animation,
		StartTime:  time.Now().Add(animation.Delay),
		Progress:   0.0,
		State:      AnimationStatePlaying,
		Context:    animCtx,
		CancelFunc: cancelFunc,
	}

	a.mu.Lock()
	a.activeAnimations[animation.ID] = activeAnim
	a.mu.Unlock()

	a.logger.Debug("Animation started",
		zap.String("animation", animation.ID),
		zap.Duration("duration", animation.Duration))

	return nil
}

// animationLoop is the main animation update loop
func (a *Animator) animationLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-a.ticker.C:
			if !a.running {
				return
			}
			a.updateAnimations()
		}
	}
}

// updateAnimations updates all active animations
func (a *Animator) updateAnimations() {
	a.mu.Lock()
	defer a.mu.Unlock()

	now := time.Now()
	completedAnimations := make([]string, 0)

	for id, anim := range a.activeAnimations {
		// Skip animations that haven't started yet
		if now.Before(anim.StartTime) {
			continue
		}

		// Calculate progress
		elapsed := now.Sub(anim.StartTime)
		progress := float64(elapsed) / float64(anim.Animation.Duration)

		if progress >= 1.0 {
			progress = 1.0
			completedAnimations = append(completedAnimations, id)
			anim.State = AnimationStateComplete
		}

		// Apply easing
		easedProgress := a.applyEasing(progress, anim.Animation.Easing)
		anim.Progress = easedProgress
		anim.CurrentTime = now

		// Calculate interpolated values
		values := a.interpolateValues(anim.Animation.From, anim.Animation.To, easedProgress)

		// Call update callback
		if anim.Animation.OnUpdate != nil {
			if err := anim.Animation.OnUpdate(easedProgress, values); err != nil {
				a.logger.Warn("Animation update callback failed",
					zap.String("animation", id),
					zap.Error(err))
			}
		}
	}

	// Handle completed animations
	for _, id := range completedAnimations {
		anim := a.activeAnimations[id]

		// Call completion callback
		if anim.Animation.OnComplete != nil {
			if err := anim.Animation.OnComplete(anim.Animation); err != nil {
				a.logger.Warn("Animation completion callback failed",
					zap.String("animation", id),
					zap.Error(err))
			}
		}

		// Handle looping
		if anim.Animation.Loop.Enabled && anim.Animation.Loop.Count != 0 {
			a.handleAnimationLoop(anim)
		} else {
			delete(a.activeAnimations, id)
		}
	}
}

// executeTimeline runs a timeline animation sequence
func (a *Animator) executeTimeline(ctx context.Context, timeline *Timeline) {
	startTime := time.Now()

	for {
		select {
		case <-ctx.Done():
			timeline.State = TimelineStateCancelled
			return
		default:
			elapsed := time.Since(startTime)

			// Check for animations to start
			for _, timelineAnim := range timeline.Animations {
				if elapsed >= timelineAnim.StartTime &&
					elapsed < timelineAnim.EndTime {
					// Start animation if not already running
					if _, exists := a.activeAnimations[timelineAnim.Animation.ID]; !exists {
						a.playAnimation(ctx, timelineAnim.Animation)
					}
				}
			}

			// Check if timeline is complete
			if elapsed >= timeline.Duration {
				timeline.State = TimelineStateComplete
				if timeline.OnComplete != nil {
					timeline.OnComplete(timeline)
				}
				return
			}

			time.Sleep(16 * time.Millisecond) // ~60fps
		}
	}
}

// applyEasing applies easing function to progress
func (a *Animator) applyEasing(progress float64, easing EasingFunction) float64 {
	switch easing {
	case Linear:
		return progress
	case EaseIn:
		return progress * progress
	case EaseOut:
		return 1 - (1-progress)*(1-progress)
	case EaseInOut:
		if progress < 0.5 {
			return 2 * progress * progress
		}
		return 1 - 2*(1-progress)*(1-progress)
	case EaseInQuad:
		return progress * progress
	case EaseOutQuad:
		return 1 - (1-progress)*(1-progress)
	case EaseInOutQuad:
		if progress < 0.5 {
			return 2 * progress * progress
		}
		return 1 - 2*(1-progress)*(1-progress)
	case EaseInCubic:
		return progress * progress * progress
	case EaseOutCubic:
		p := 1 - progress
		return 1 - p*p*p
	case EaseInOutCubic:
		if progress < 0.5 {
			return 4 * progress * progress * progress
		}
		p := 1 - progress
		return 1 - 4*p*p*p
	case EaseInBack:
		c := 1.70158
		return progress * progress * ((c+1)*progress - c)
	case EaseOutBack:
		c := 1.70158
		p := progress - 1
		return 1 + p*p*((c+1)*p+c)
	case EaseInOutBack:
		c := 1.70158 * 1.525
		if progress < 0.5 {
			p := progress * 2
			return 0.5 * p * p * ((c+1)*p - c)
		}
		p := (progress-0.5)*2 - 1
		return 0.5 * (1 + p*p*((c+1)*p+c))
	case EaseInElastic:
		if progress == 0 || progress == 1 {
			return progress
		}
		c := (2 * math.Pi) / 3
		return -math.Pow(2, 10*(progress-1)) * math.Sin((progress-1)*c)
	case EaseOutElastic:
		if progress == 0 || progress == 1 {
			return progress
		}
		c := (2 * math.Pi) / 3
		return 1 + math.Pow(2, -10*progress)*math.Sin(progress*c)
	case EaseInBounce:
		return 1 - a.applyEasing(1-progress, EaseOutBounce)
	case EaseOutBounce:
		if progress < 1/2.75 {
			return 7.5625 * progress * progress
		} else if progress < 2/2.75 {
			p := progress - 1.5/2.75
			return 7.5625*p*p + 0.75
		} else if progress < 2.5/2.75 {
			p := progress - 2.25/2.75
			return 7.5625*p*p + 0.9375
		} else {
			p := progress - 2.625/2.75
			return 7.5625*p*p + 0.984375
		}
	default:
		return progress
	}
}

// interpolateValues interpolates between from and to values
func (a *Animator) interpolateValues(from, to map[string]interface{}, progress float64) map[string]interface{} {
	result := make(map[string]interface{})

	for key, fromVal := range from {
		toVal, exists := to[key]
		if !exists {
			result[key] = fromVal
			continue
		}

		// Handle different value types
		switch fromV := fromVal.(type) {
		case float64:
			if toV, ok := toVal.(float64); ok {
				result[key] = fromV + (toV-fromV)*progress
			}
		case int:
			if toV, ok := toVal.(int); ok {
				result[key] = fromV + int(float64(toV-fromV)*progress)
			}
		case string:
			// For strings, just switch at 50% progress
			if progress < 0.5 {
				result[key] = fromV
			} else {
				result[key] = toVal
			}
		default:
			result[key] = fromVal
		}
	}

	return result
}

// handleAnimationLoop manages animation looping
func (a *Animator) handleAnimationLoop(anim *ActiveAnimation) {
	if anim.Animation.Loop.Count > 0 {
		anim.Animation.Loop.Count--
	}

	if anim.Animation.Loop.Reverse {
		// Swap from and to values
		anim.Animation.From, anim.Animation.To = anim.Animation.To, anim.Animation.From
	}

	// Reset animation
	anim.StartTime = time.Now().Add(anim.Animation.Delay)
	anim.Progress = 0.0
	anim.State = AnimationStatePlaying
}

// registerBuiltinAnimations registers common animation presets
func (a *Animator) registerBuiltinAnimations() {
	// Fade in animation
	a.registry.RegisterAnimation("fade-in", &Animation{
		Name:     "fade-in",
		Duration: 300 * time.Millisecond,
		Easing:   EaseOut,
		From:     map[string]interface{}{"opacity": 0.0},
		To:       map[string]interface{}{"opacity": 1.0},
	})

	// Fade out animation
	a.registry.RegisterAnimation("fade-out", &Animation{
		Name:     "fade-out",
		Duration: 300 * time.Millisecond,
		Easing:   EaseIn,
		From:     map[string]interface{}{"opacity": 1.0},
		To:       map[string]interface{}{"opacity": 0.0},
	})

	// Slide in from left
	a.registry.RegisterAnimation("slide-in-left", &Animation{
		Name:     "slide-in-left",
		Duration: 400 * time.Millisecond,
		Easing:   EaseOutBack,
		From:     map[string]interface{}{"x": -100, "opacity": 0.0},
		To:       map[string]interface{}{"x": 0, "opacity": 1.0},
	})

	// Scale up animation
	a.registry.RegisterAnimation("scale-up", &Animation{
		Name:     "scale-up",
		Duration: 250 * time.Millisecond,
		Easing:   EaseOutElastic,
		From:     map[string]interface{}{"scale": 0.8, "opacity": 0.0},
		To:       map[string]interface{}{"scale": 1.0, "opacity": 1.0},
	})
}

// DefaultAnimationRegistry provides basic animation registry
type DefaultAnimationRegistry struct {
	animations map[string]*Animation
	mu         sync.RWMutex
}

func NewDefaultAnimationRegistry() *DefaultAnimationRegistry {
	return &DefaultAnimationRegistry{
		animations: make(map[string]*Animation),
	}
}

func (r *DefaultAnimationRegistry) RegisterAnimation(name string, animation *Animation) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.animations[name] = animation
	return nil
}

func (r *DefaultAnimationRegistry) GetAnimation(name string) (*Animation, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	anim, exists := r.animations[name]
	if !exists {
		return nil, fmt.Errorf("animation '%s' not found", name)
	}

	// Return a copy to prevent modification
	animCopy := *anim
	return &animCopy, nil
}

func (r *DefaultAnimationRegistry) ListAnimations() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.animations))
	for name := range r.animations {
		names = append(names, name)
	}
	return names
}
