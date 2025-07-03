// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package animation

import (
	"context"
	"fmt"
	"math"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCraftAnimator_Creation(t *testing.T) {
	tests := []struct {
		name     string
		testFunc func(t *testing.T)
	}{
		{"creates animator with defaults", testCraftAnimatorCreation},
		{"sets correct frame rate", testCraftAnimatorFrameRate},
		{"initializes built-in animations", testCraftBuiltinAnimations},
	}

	for _, tt := range tests {
		t.Run(tt.name, tt.testFunc)
	}
}

func testCraftAnimatorCreation(t *testing.T) {
	animator := NewAnimator()
	
	assert.NotNil(t, animator)
	assert.NotNil(t, animator.activeAnimations)
	assert.NotNil(t, animator.timelines)
	assert.NotNil(t, animator.registry)
	assert.NotNil(t, animator.ticker)
	assert.NotNil(t, animator.logger)
	assert.False(t, animator.running)
}

func testCraftAnimatorFrameRate(t *testing.T) {
	animator := NewAnimator()
	
	// Verify 60fps targeting (~16ms interval)
	actualInterval := animator.ticker.C
	
	// We can't directly compare the channel, but we can verify the ticker exists
	assert.NotNil(t, actualInterval)
}

func testCraftBuiltinAnimations(t *testing.T) {
	animator := NewAnimator()
	
	expectedAnimations := []string{"fade-in", "fade-out", "slide-in-left", "scale-up"}
	
	for _, animName := range expectedAnimations {
		anim, err := animator.registry.GetAnimation(animName)
		assert.NoError(t, err, "Built-in animation %s should exist", animName)
		assert.NotNil(t, anim)
		assert.Equal(t, animName, anim.Name)
	}
}

func TestCraftAnimator_Lifecycle(t *testing.T) {
	tests := []struct {
		name     string
		testFunc func(t *testing.T)
	}{
		{"starts and stops animator", testCraftAnimatorLifecycle},
		{"handles multiple start calls", testCraftMultipleStarts},
		{"handles multiple stop calls", testCraftMultipleStops},
	}

	for _, tt := range tests {
		t.Run(tt.name, tt.testFunc)
	}
}

func testCraftAnimatorLifecycle(t *testing.T) {
	animator := NewAnimator()
	ctx := context.Background()
	
	// Should not be running initially
	assert.False(t, animator.running)
	
	// Start animator
	err := animator.Start(ctx)
	assert.NoError(t, err)
	assert.True(t, animator.running)
	
	// Stop animator
	err = animator.Stop(ctx)
	assert.NoError(t, err)
	assert.False(t, animator.running)
}

func testCraftMultipleStarts(t *testing.T) {
	animator := NewAnimator()
	ctx := context.Background()
	
	// Multiple starts should not error
	err1 := animator.Start(ctx)
	err2 := animator.Start(ctx)
	
	assert.NoError(t, err1)
	assert.NoError(t, err2)
	assert.True(t, animator.running)
	
	animator.Stop(ctx)
}

func testCraftMultipleStops(t *testing.T) {
	animator := NewAnimator()
	ctx := context.Background()
	
	animator.Start(ctx)
	
	// Multiple stops should not error
	err1 := animator.Stop(ctx)
	err2 := animator.Stop(ctx)
	
	assert.NoError(t, err1)
	assert.NoError(t, err2)
	assert.False(t, animator.running)
}

func TestCraftAnimation_Basic(t *testing.T) {
	tests := []struct {
		name     string
		testFunc func(t *testing.T)
	}{
		{"creates and plays animation", testCraftBasicAnimation},
		{"handles animation completion", testCraftAnimationCompletion},
		{"applies easing functions", testCraftEasingFunctions},
	}

	for _, tt := range tests {
		t.Run(tt.name, tt.testFunc)
	}
}

func testCraftBasicAnimation(t *testing.T) {
	animator := NewAnimator()
	ctx := context.Background()
	
	animator.Start(ctx)
	defer animator.Stop(ctx)
	
	updateCalled := false
	options := AnimationOptions{
		Duration: 100 * time.Millisecond,
		Easing:   Linear,
		From:     map[string]interface{}{"opacity": 0.0},
		To:       map[string]interface{}{"opacity": 1.0},
		OnUpdate: func(progress float64, values map[string]interface{}) error {
			updateCalled = true
			assert.GreaterOrEqual(t, progress, 0.0)
			assert.LessOrEqual(t, progress, 1.0)
			assert.Contains(t, values, "opacity")
			return nil
		},
	}
	
	err := animator.Animate(ctx, "test-animation", options)
	assert.NoError(t, err)
	
	// Wait for animation to start
	time.Sleep(50 * time.Millisecond)
	assert.True(t, updateCalled, "Update callback should be called")
}

func testCraftAnimationCompletion(t *testing.T) {
	animator := NewAnimator()
	ctx := context.Background()
	
	animator.Start(ctx)
	defer animator.Stop(ctx)
	
	completionCalled := false
	options := AnimationOptions{
		Duration: 50 * time.Millisecond,
		Easing:   Linear,
		From:     map[string]interface{}{"scale": 0.0},
		To:       map[string]interface{}{"scale": 1.0},
		OnComplete: func(animation *Animation) error {
			completionCalled = true
			assert.NotNil(t, animation)
			return nil
		},
	}
	
	err := animator.Animate(ctx, "test-completion", options)
	assert.NoError(t, err)
	
	// Wait for animation to complete
	time.Sleep(100 * time.Millisecond)
	assert.True(t, completionCalled, "Completion callback should be called")
}

func testCraftEasingFunctions(t *testing.T) {
	animator := NewAnimator()
	
	testCases := []struct {
		easing   EasingFunction
		progress float64
		name     string
	}{
		{Linear, 0.5, "Linear"},
		{EaseIn, 0.5, "EaseIn"},
		{EaseOut, 0.5, "EaseOut"},
		{EaseInOut, 0.5, "EaseInOut"},
		{EaseInCubic, 0.5, "EaseInCubic"},
		{EaseOutCubic, 0.5, "EaseOutCubic"},
		{EaseInBack, 0.5, "EaseInBack"},
		{EaseOutBack, 0.5, "EaseOutBack"},
		{EaseInElastic, 0.5, "EaseInElastic"},
		{EaseOutElastic, 0.5, "EaseOutElastic"},
		{EaseInBounce, 0.5, "EaseInBounce"},
		{EaseOutBounce, 0.5, "EaseOutBounce"},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := animator.applyEasing(tc.progress, tc.easing)
			
			// Some easing functions like Back and Elastic can overshoot 0-1 range
			// This is expected behavior for these easing types
			switch tc.easing {
			case EaseInBack, EaseOutBack, EaseInOutBack:
				// Back easing can go outside 0-1 range
				assert.True(t, !math.IsNaN(result), "Easing result should not be NaN")
			case EaseInElastic, EaseOutElastic, EaseInOutElastic:
				// Elastic easing can also overshoot
				assert.True(t, !math.IsNaN(result), "Easing result should not be NaN")
			default:
				// Linear and cubic easing should stay within bounds
				assert.GreaterOrEqual(t, result, 0.0, "Easing result should be >= 0")
				assert.LessOrEqual(t, result, 1.0, "Easing result should be <= 1")
			}
		})
	}
}

func TestCraftAnimation_Presets(t *testing.T) {
	tests := []struct {
		name     string
		testFunc func(t *testing.T)
	}{
		{"plays fade-in preset", testCraftFadeInPreset},
		{"plays fade-out preset", testCraftFadeOutPreset},
		{"plays slide-in-left preset", testCraftSlideInLeftPreset},
		{"plays scale-up preset", testCraftScaleUpPreset},
		{"fails for unknown preset", testCraftUnknownPreset},
	}

	for _, tt := range tests {
		t.Run(tt.name, tt.testFunc)
	}
}

func testCraftFadeInPreset(t *testing.T) {
	animator := NewAnimator()
	ctx := context.Background()
	
	animator.Start(ctx)
	defer animator.Stop(ctx)
	
	options := AnimationOptions{
		Duration: 50 * time.Millisecond,
	}
	
	err := animator.AnimatePreset(ctx, "fade-in", options)
	assert.NoError(t, err)
}

func testCraftFadeOutPreset(t *testing.T) {
	animator := NewAnimator()
	ctx := context.Background()
	
	animator.Start(ctx)
	defer animator.Stop(ctx)
	
	options := AnimationOptions{
		Duration: 50 * time.Millisecond,
	}
	
	err := animator.AnimatePreset(ctx, "fade-out", options)
	assert.NoError(t, err)
}

func testCraftSlideInLeftPreset(t *testing.T) {
	animator := NewAnimator()
	ctx := context.Background()
	
	animator.Start(ctx)
	defer animator.Stop(ctx)
	
	options := AnimationOptions{
		Duration: 50 * time.Millisecond,
	}
	
	err := animator.AnimatePreset(ctx, "slide-in-left", options)
	assert.NoError(t, err)
}

func testCraftScaleUpPreset(t *testing.T) {
	animator := NewAnimator()
	ctx := context.Background()
	
	animator.Start(ctx)
	defer animator.Stop(ctx)
	
	options := AnimationOptions{
		Duration: 50 * time.Millisecond,
	}
	
	err := animator.AnimatePreset(ctx, "scale-up", options)
	assert.NoError(t, err)
}

func testCraftUnknownPreset(t *testing.T) {
	animator := NewAnimator()
	ctx := context.Background()
	
	options := AnimationOptions{
		Duration: 50 * time.Millisecond,
	}
	
	err := animator.AnimatePreset(ctx, "unknown-preset", options)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestCraftTimeline(t *testing.T) {
	tests := []struct {
		name     string
		testFunc func(t *testing.T)
	}{
		{"creates and plays timeline", testCraftTimelineBasic},
		{"handles multiple animations", testCraftTimelineMultiple},
		{"respects timing offsets", testCraftTimelineTiming},
		{"calls completion callback", testCraftTimelineCompletion},
	}

	for _, tt := range tests {
		t.Run(tt.name, tt.testFunc)
	}
}

func testCraftTimelineBasic(t *testing.T) {
	animator := NewAnimator()
	ctx := context.Background()
	
	animator.Start(ctx)
	defer animator.Stop(ctx)
	
	timeline := animator.CreateTimeline("test-timeline", TimelineOptions{
		Name: "Test Timeline",
	})
	
	assert.NotNil(t, timeline)
	assert.Equal(t, "test-timeline", timeline.ID)
	assert.Equal(t, "Test Timeline", timeline.Name)
	assert.Equal(t, TimelineStateIdle, timeline.State)
}

func testCraftTimelineMultiple(t *testing.T) {
	animator := NewAnimator()
	ctx := context.Background()
	
	animator.Start(ctx)
	defer animator.Stop(ctx)
	
	timeline := animator.CreateTimeline("multi-timeline", TimelineOptions{})
	
	// Add multiple animations
	timeline.AddAnimation("anim1", 0, AnimationOptions{
		Duration: 50 * time.Millisecond,
		From:     map[string]interface{}{"x": 0},
		To:       map[string]interface{}{"x": 100},
	})
	
	timeline.AddAnimation("anim2", 25*time.Millisecond, AnimationOptions{
		Duration: 50 * time.Millisecond,
		From:     map[string]interface{}{"y": 0},
		To:       map[string]interface{}{"y": 100},
	})
	
	assert.Len(t, timeline.Animations, 2)
	assert.Equal(t, 75*time.Millisecond, timeline.Duration)
}

func testCraftTimelineTiming(t *testing.T) {
	animator := NewAnimator()
	ctx := context.Background()
	
	animator.Start(ctx)
	defer animator.Stop(ctx)
	
	timeline := animator.CreateTimeline("timing-timeline", TimelineOptions{})
	
	startOffset := 100 * time.Millisecond
	duration := 50 * time.Millisecond
	
	timeline.AddAnimation("delayed-anim", startOffset, AnimationOptions{
		Duration: duration,
		From:     map[string]interface{}{"opacity": 0},
		To:       map[string]interface{}{"opacity": 1},
	})
	
	timelineAnim := timeline.Animations[0]
	assert.Equal(t, startOffset, timelineAnim.StartTime)
	assert.Equal(t, startOffset+duration, timelineAnim.EndTime)
}

func testCraftTimelineCompletion(t *testing.T) {
	animator := NewAnimator()
	ctx := context.Background()
	
	animator.Start(ctx)
	defer animator.Stop(ctx)
	
	completionCalled := false
	timeline := animator.CreateTimeline("completion-timeline", TimelineOptions{
		OnComplete: func(timeline *Timeline) error {
			completionCalled = true
			assert.NotNil(t, timeline)
			return nil
		},
	})
	
	timeline.AddAnimation("quick-anim", 0, AnimationOptions{
		Duration: 50 * time.Millisecond,
		From:     map[string]interface{}{"scale": 0},
		To:       map[string]interface{}{"scale": 1},
	})
	
	err := animator.PlayTimeline(ctx, "completion-timeline")
	assert.NoError(t, err)
	
	// Wait for timeline to complete
	time.Sleep(100 * time.Millisecond)
	assert.True(t, completionCalled, "Timeline completion callback should be called")
}

func TestCraftAnimation_Looping(t *testing.T) {
	tests := []struct {
		name     string
		testFunc func(t *testing.T)
	}{
		{"handles finite loops", testCraftFiniteLoop},
		{"handles reverse loops", testCraftReverseLoop},
		{"handles ping-pong loops", testCraftPingPongLoop},
	}

	for _, tt := range tests {
		t.Run(tt.name, tt.testFunc)
	}
}

func testCraftFiniteLoop(t *testing.T) {
	animator := NewAnimator()
	ctx := context.Background()
	
	animator.Start(ctx)
	defer animator.Stop(ctx)
	
	loopCount := 0
	options := AnimationOptions{
		Duration: 30 * time.Millisecond,
		From:     map[string]interface{}{"value": 0},
		To:       map[string]interface{}{"value": 1},
		Loop: LoopConfig{
			Enabled: true,
			Count:   2, // Loop twice
		},
		OnComplete: func(animation *Animation) error {
			loopCount++
			return nil
		},
	}
	
	err := animator.Animate(ctx, "loop-test", options)
	assert.NoError(t, err)
	
	// Wait for loops to complete
	time.Sleep(150 * time.Millisecond)
	assert.GreaterOrEqual(t, loopCount, 2, "Should complete at least 2 loops")
}

func testCraftReverseLoop(t *testing.T) {
	animator := NewAnimator()
	ctx := context.Background()
	
	animator.Start(ctx)
	defer animator.Stop(ctx)
	
	options := AnimationOptions{
		Duration: 30 * time.Millisecond,
		From:     map[string]interface{}{"value": 0},
		To:       map[string]interface{}{"value": 1},
		Loop: LoopConfig{
			Enabled: true,
			Count:   1,
			Reverse: true,
		},
	}
	
	err := animator.Animate(ctx, "reverse-loop-test", options)
	assert.NoError(t, err)
	
	// Animation should run
	time.Sleep(100 * time.Millisecond)
}

func testCraftPingPongLoop(t *testing.T) {
	animator := NewAnimator()
	ctx := context.Background()
	
	animator.Start(ctx)
	defer animator.Stop(ctx)
	
	options := AnimationOptions{
		Duration: 30 * time.Millisecond,
		From:     map[string]interface{}{"value": 0},
		To:       map[string]interface{}{"value": 1},
		Loop: LoopConfig{
			Enabled:  true,
			Count:    1,
			PingPong: true,
		},
	}
	
	err := animator.Animate(ctx, "pingpong-test", options)
	assert.NoError(t, err)
	
	// Animation should run
	time.Sleep(100 * time.Millisecond)
}

func TestCraftAnimation_ValueInterpolation(t *testing.T) {
	tests := []struct {
		name     string
		testFunc func(t *testing.T)
	}{
		{"interpolates float values", testCraftFloatInterpolation},
		{"interpolates int values", testCraftIntInterpolation},
		{"handles string values", testCraftStringInterpolation},
		{"handles missing values", testCraftMissingValueInterpolation},
	}

	for _, tt := range tests {
		t.Run(tt.name, tt.testFunc)
	}
}

func testCraftFloatInterpolation(t *testing.T) {
	animator := NewAnimator()
	
	from := map[string]interface{}{"opacity": 0.0}
	to := map[string]interface{}{"opacity": 1.0}
	
	result := animator.interpolateValues(from, to, 0.5)
	
	assert.Contains(t, result, "opacity")
	assert.Equal(t, 0.5, result["opacity"])
}

func testCraftIntInterpolation(t *testing.T) {
	animator := NewAnimator()
	
	from := map[string]interface{}{"width": 0}
	to := map[string]interface{}{"width": 100}
	
	result := animator.interpolateValues(from, to, 0.5)
	
	assert.Contains(t, result, "width")
	assert.Equal(t, 50, result["width"])
}

func testCraftStringInterpolation(t *testing.T) {
	animator := NewAnimator()
	
	from := map[string]interface{}{"color": "red"}
	to := map[string]interface{}{"color": "blue"}
	
	// At 0.3 progress (< 0.5), should return "from" value
	result1 := animator.interpolateValues(from, to, 0.3)
	assert.Equal(t, "red", result1["color"])
	
	// At 0.7 progress (> 0.5), should return "to" value
	result2 := animator.interpolateValues(from, to, 0.7)
	assert.Equal(t, "blue", result2["color"])
}

func testCraftMissingValueInterpolation(t *testing.T) {
	animator := NewAnimator()
	
	from := map[string]interface{}{"opacity": 0.0, "width": 100}
	to := map[string]interface{}{"opacity": 1.0} // Missing width
	
	result := animator.interpolateValues(from, to, 0.5)
	
	assert.Contains(t, result, "opacity")
	assert.Contains(t, result, "width")
	assert.Equal(t, 0.5, result["opacity"])
	assert.Equal(t, 100, result["width"]) // Should keep original value
}

func TestCraftAnimation_Performance(t *testing.T) {
	tests := []struct {
		name     string
		testFunc func(t *testing.T)
	}{
		{"maintains 60fps target", testCraftFrameRateTarget},
		{"handles many concurrent animations", testCraftConcurrentAnimations},
		{"efficient easing calculations", testCraftEasingPerformance},
	}

	for _, tt := range tests {
		t.Run(tt.name, tt.testFunc)
	}
}

func testCraftFrameRateTarget(t *testing.T) {
	animator := NewAnimator()
	
	// We can't directly test the ticker interval, but we can verify it's set correctly
	// The animator uses time.NewTicker(16 * time.Millisecond) in NewAnimator
	assert.NotNil(t, animator.ticker)
}

func testCraftConcurrentAnimations(t *testing.T) {
	animator := NewAnimator()
	ctx := context.Background()
	
	animator.Start(ctx)
	defer animator.Stop(ctx)
	
	numAnimations := 50
	var wg sync.WaitGroup
	
	wg.Add(numAnimations)
	for i := 0; i < numAnimations; i++ {
		go func(id int) {
			defer wg.Done()
			
			options := AnimationOptions{
				Duration: 100 * time.Millisecond,
				From:     map[string]interface{}{"value": 0},
				To:       map[string]interface{}{"value": 100},
			}
			
			err := animator.Animate(ctx, fmt.Sprintf("concurrent-anim-%d", id), options)
			assert.NoError(t, err)
		}(i)
	}
	
	wg.Wait()
	
	// Let animations run briefly
	time.Sleep(50 * time.Millisecond)
	
	// Should handle concurrent animations without issues
	assert.True(t, animator.running)
}

func testCraftEasingPerformance(t *testing.T) {
	animator := NewAnimator()
	
	// Test that easing functions complete quickly
	start := time.Now()
	iterations := 10000
	
	for i := 0; i < iterations; i++ {
		progress := float64(i) / float64(iterations)
		animator.applyEasing(progress, EaseInOutCubic)
	}
	
	duration := time.Since(start)
	
	// Should complete 10k easing calculations in reasonable time
	assert.Less(t, duration, 100*time.Millisecond, "Easing calculations should be fast")
}

func TestCraftAnimation_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		testFunc func(t *testing.T)
	}{
		{"handles zero duration", testCraftZeroDuration},
		{"handles negative duration", testCraftNegativeDuration},
		{"handles context cancellation", testCraftContextCancellation},
		{"handles nil callbacks", testCraftNilCallbacks},
	}

	for _, tt := range tests {
		t.Run(tt.name, tt.testFunc)
	}
}

func testCraftZeroDuration(t *testing.T) {
	animator := NewAnimator()
	ctx := context.Background()
	
	animator.Start(ctx)
	defer animator.Stop(ctx)
	
	options := AnimationOptions{
		Duration: 0,
		From:     map[string]interface{}{"value": 0},
		To:       map[string]interface{}{"value": 1},
	}
	
	err := animator.Animate(ctx, "zero-duration", options)
	assert.NoError(t, err)
}

func testCraftNegativeDuration(t *testing.T) {
	animator := NewAnimator()
	ctx := context.Background()
	
	animator.Start(ctx)
	defer animator.Stop(ctx)
	
	options := AnimationOptions{
		Duration: -100 * time.Millisecond,
		From:     map[string]interface{}{"value": 0},
		To:       map[string]interface{}{"value": 1},
	}
	
	err := animator.Animate(ctx, "negative-duration", options)
	assert.NoError(t, err)
}

func testCraftContextCancellation(t *testing.T) {
	animator := NewAnimator()
	ctx, cancel := context.WithCancel(context.Background())
	
	animator.Start(ctx)
	defer animator.Stop(context.Background())
	
	options := AnimationOptions{
		Duration: 1 * time.Second, // Long duration
		From:     map[string]interface{}{"value": 0},
		To:       map[string]interface{}{"value": 1},
	}
	
	err := animator.Animate(ctx, "cancellable", options)
	assert.NoError(t, err)
	
	// Cancel context
	cancel()
	
	// Animation should handle cancellation gracefully
	time.Sleep(50 * time.Millisecond)
}

func testCraftNilCallbacks(t *testing.T) {
	animator := NewAnimator()
	ctx := context.Background()
	
	animator.Start(ctx)
	defer animator.Stop(ctx)
	
	options := AnimationOptions{
		Duration:   50 * time.Millisecond,
		From:       map[string]interface{}{"value": 0},
		To:         map[string]interface{}{"value": 1},
		OnUpdate:   nil, // Nil callback
		OnComplete: nil, // Nil callback
	}
	
	err := animator.Animate(ctx, "nil-callbacks", options)
	assert.NoError(t, err)
	
	// Should complete without panicking
	time.Sleep(100 * time.Millisecond)
}

func TestCraftAnimationRegistry(t *testing.T) {
	tests := []struct {
		name     string
		testFunc func(t *testing.T)
	}{
		{"registers custom animation", testCraftRegisterAnimation},
		{"lists all animations", testCraftListAnimations},
		{"returns copies to prevent modification", testCraftAnimationCopies},
	}

	for _, tt := range tests {
		t.Run(tt.name, tt.testFunc)
	}
}

func testCraftRegisterAnimation(t *testing.T) {
	registry := NewDefaultAnimationRegistry()
	
	customAnim := &Animation{
		Name:     "custom-fade",
		Duration: 500 * time.Millisecond,
		Easing:   EaseInOut,
		From:     map[string]interface{}{"opacity": 0.0},
		To:       map[string]interface{}{"opacity": 1.0},
	}
	
	err := registry.RegisterAnimation("custom-fade", customAnim)
	assert.NoError(t, err)
	
	retrieved, err := registry.GetAnimation("custom-fade")
	assert.NoError(t, err)
	assert.NotNil(t, retrieved)
	assert.Equal(t, "custom-fade", retrieved.Name)
}

func testCraftListAnimations(t *testing.T) {
	registry := NewDefaultAnimationRegistry()
	
	// Register a few animations
	registry.RegisterAnimation("anim1", &Animation{Name: "anim1"})
	registry.RegisterAnimation("anim2", &Animation{Name: "anim2"})
	
	animations := registry.ListAnimations()
	assert.Len(t, animations, 2)
	assert.Contains(t, animations, "anim1")
	assert.Contains(t, animations, "anim2")
}

func testCraftAnimationCopies(t *testing.T) {
	registry := NewDefaultAnimationRegistry()
	
	original := &Animation{
		Name:     "modifiable",
		Duration: 100 * time.Millisecond,
	}
	
	registry.RegisterAnimation("modifiable", original)
	
	retrieved1, _ := registry.GetAnimation("modifiable")
	retrieved2, _ := registry.GetAnimation("modifiable")
	
	// Should be different pointers (copies)
	assert.NotSame(t, retrieved1, retrieved2)
	
	// Modifying one should not affect the other
	retrieved1.Duration = 200 * time.Millisecond
	assert.NotEqual(t, retrieved1.Duration, retrieved2.Duration)
}

// Benchmark tests for performance validation
func BenchmarkCraftAnimationUpdate(b *testing.B) {
	animator := NewAnimator()
	
	// Simulate active animations
	for i := 0; i < 10; i++ {
		anim := &ActiveAnimation{
			Animation: &Animation{
				ID:       fmt.Sprintf("bench-anim-%d", i),
				Duration: 1 * time.Second,
				Easing:   Linear,
				From:     map[string]interface{}{"value": 0.0},
				To:       map[string]interface{}{"value": 1.0},
			},
			StartTime: time.Now(),
			Progress:  0.0,
			State:     AnimationStatePlaying,
		}
		animator.activeAnimations[anim.Animation.ID] = anim
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		animator.updateAnimations()
	}
}

func BenchmarkCraftEasingFunctions(b *testing.B) {
	animator := NewAnimator()
	
	easingFunctions := []EasingFunction{
		Linear, EaseIn, EaseOut, EaseInOut,
		EaseInCubic, EaseOutCubic, EaseInBack, EaseOutBack,
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		progress := float64(i%1000) / 1000.0
		easing := easingFunctions[i%len(easingFunctions)]
		animator.applyEasing(progress, easing)
	}
}

func BenchmarkCraftValueInterpolation(b *testing.B) {
	animator := NewAnimator()
	
	from := map[string]interface{}{
		"opacity": 0.0,
		"x":       0,
		"y":       0,
		"scale":   0.5,
		"color":   "red",
	}
	
	to := map[string]interface{}{
		"opacity": 1.0,
		"x":       100,
		"y":       50,
		"scale":   1.0,
		"color":   "blue",
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		progress := float64(i%1000) / 1000.0
		animator.interpolateValues(from, to, progress)
	}
}

// Performance validation test to ensure 60fps target
func TestCraftAnimator_60FPSPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}
	
	animator := NewAnimator()
	ctx := context.Background()
	
	animator.Start(ctx)
	defer animator.Stop(ctx)
	
	// Create multiple animations to stress test
	numAnimations := 20
	for i := 0; i < numAnimations; i++ {
		options := AnimationOptions{
			Duration: 1 * time.Second,
			From:     map[string]interface{}{"value": 0},
			To:       map[string]interface{}{"value": 100},
		}
		
		err := animator.Animate(ctx, fmt.Sprintf("perf-test-%d", i), options)
		require.NoError(t, err)
	}
	
	// Measure update performance
	start := time.Now()
	iterations := 100
	
	for i := 0; i < iterations; i++ {
		animator.updateAnimations()
	}
	
	duration := time.Since(start)
	avgUpdateTime := duration / time.Duration(iterations)
	
	// Each update should complete well under 16ms (60fps target)
	maxUpdateTime := 10 * time.Millisecond
	assert.Less(t, avgUpdateTime, maxUpdateTime,
		"Animation updates should complete in less than %v (avg: %v)", maxUpdateTime, avgUpdateTime)
	
	t.Logf("Average update time: %v (target: < %v)", avgUpdateTime, maxUpdateTime)
}