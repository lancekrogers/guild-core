// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package chat

import (
	"fmt"
	"sync"
	"time"
)

// AgentIndicators provides real-time visual feedback and animations
type AgentIndicators struct {
	animations     map[string]*Animation // agentID -> current animation
	blinkStates    map[string]bool       // Activity blink states
	updateTicker   *time.Ticker          // Animation update timer
	enabled        bool                  // Enable/disable animations
	mutex          sync.RWMutex          // Thread-safe access
	animationSpeed time.Duration         // Animation frame duration
}

// Animation represents a status animation
type Animation struct {
	Type         AnimationType // spinner, pulse, progress, blink
	Duration     time.Duration // Animation duration
	StartTime    time.Time     // When animation started
	Frames       []string      // Animation frames
	CurrentFrame int           // Current frame index
	Infinite     bool          // Whether animation loops indefinitely
	AgentID      string        // Associated agent ID
	Context      string        // Additional context (task, tool, etc.)
}

// AnimationType represents different types of visual animations
type AnimationType int

const (
	AnimationSpinner       AnimationType = iota // Spinning indicator
	AnimationPulse                              // Pulsing effect
	AnimationProgress                           // Progress bar animation
	AnimationBlink                              // Blinking indicator
	AnimationThinking                           // Thinking animation sequence
	AnimationWorking                            // Working animation sequence
	AnimationCoordination                       // Multi-agent coordination
	AnimationToolExecution                      // Tool execution animation
)

// String returns the string representation of AnimationType
func (t AnimationType) String() string {
	switch t {
	case AnimationSpinner:
		return "spinner"
	case AnimationPulse:
		return "pulse"
	case AnimationProgress:
		return "progress"
	case AnimationBlink:
		return "blink"
	case AnimationThinking:
		return "thinking"
	case AnimationWorking:
		return "working"
	case AnimationCoordination:
		return "coordination"
	case AnimationToolExecution:
		return "tool_execution"
	default:
		return "unknown"
	}
}

// NewAgentIndicators creates a new agent indicators system
func NewAgentIndicators() *AgentIndicators {
	return &AgentIndicators{
		animations:     make(map[string]*Animation),
		blinkStates:    make(map[string]bool),
		enabled:        true,
		animationSpeed: 500 * time.Millisecond, // Default animation speed
	}
}

// StartAnimations begins the animation update loop
func (ai *AgentIndicators) StartAnimations() {
	if ai.updateTicker != nil {
		ai.StopAnimations() // Stop existing ticker
	}

	ai.updateTicker = time.NewTicker(ai.animationSpeed)
	go ai.animationLoop()
}

// StopAnimations stops the animation update loop
func (ai *AgentIndicators) StopAnimations() {
	if ai.updateTicker != nil {
		ai.updateTicker.Stop()
		ai.updateTicker = nil
	}
}

// animationLoop processes animation updates
func (ai *AgentIndicators) animationLoop() {
	for range ai.updateTicker.C {
		ai.updateAnimations()
	}
}

// updateAnimations advances all active animations by one frame
func (ai *AgentIndicators) updateAnimations() {
	ai.mutex.Lock()
	defer ai.mutex.Unlock()

	now := time.Now()
	var toRemove []string

	for agentID, animation := range ai.animations {
		// Check if animation has expired (for non-infinite animations)
		if !animation.Infinite && now.Sub(animation.StartTime) > animation.Duration {
			toRemove = append(toRemove, agentID)
			continue
		}

		// Advance to next frame
		animation.CurrentFrame = (animation.CurrentFrame + 1) % len(animation.Frames)

		// Update blink states for blinking animations
		if animation.Type == AnimationBlink {
			ai.blinkStates[agentID] = !ai.blinkStates[agentID]
		}
	}

	// Remove expired animations
	for _, agentID := range toRemove {
		delete(ai.animations, agentID)
		delete(ai.blinkStates, agentID)
	}
}

// SetThinkingAnimation sets a thinking animation for an agent
func (ai *AgentIndicators) SetThinkingAnimation(agentID string) {
	if !ai.enabled {
		return
	}

	animation := &Animation{
		Type:         AnimationThinking,
		Duration:     0, // Infinite until stopped
		StartTime:    time.Now(),
		Frames:       ai.getThinkingFrames(),
		CurrentFrame: 0,
		Infinite:     true,
		AgentID:      agentID,
		Context:      "thinking",
	}

	ai.mutex.Lock()
	ai.animations[agentID] = animation
	ai.mutex.Unlock()
}

// SetWorkingAnimation sets a working animation for an agent
func (ai *AgentIndicators) SetWorkingAnimation(agentID, task string) {
	if !ai.enabled {
		return
	}

	animation := &Animation{
		Type:         AnimationWorking,
		Duration:     0, // Infinite until stopped
		StartTime:    time.Now(),
		Frames:       ai.getWorkingFrames(),
		CurrentFrame: 0,
		Infinite:     true,
		AgentID:      agentID,
		Context:      task,
	}

	ai.mutex.Lock()
	ai.animations[agentID] = animation
	ai.mutex.Unlock()
}

// SetToolExecutionAnimation sets a tool execution animation
func (ai *AgentIndicators) SetToolExecutionAnimation(agentID, toolName string) {
	if !ai.enabled {
		return
	}

	animation := &Animation{
		Type:         AnimationToolExecution,
		Duration:     0, // Infinite until stopped
		StartTime:    time.Now(),
		Frames:       ai.getToolExecutionFrames(toolName),
		CurrentFrame: 0,
		Infinite:     true,
		AgentID:      agentID,
		Context:      toolName,
	}

	ai.mutex.Lock()
	ai.animations[agentID] = animation
	ai.mutex.Unlock()
}

// SetCoordinationAnimation sets a coordination animation for multiple agents
func (ai *AgentIndicators) SetCoordinationAnimation(agentIDs []string, duration time.Duration) {
	if !ai.enabled {
		return
	}

	animation := &Animation{
		Type:         AnimationCoordination,
		Duration:     duration,
		StartTime:    time.Now(),
		Frames:       ai.getCoordinationFrames(),
		CurrentFrame: 0,
		Infinite:     false,
		Context:      fmt.Sprintf("coordination_%d_agents", len(agentIDs)),
	}

	ai.mutex.Lock()
	for _, agentID := range agentIDs {
		// Create a copy for each agent
		agentAnimation := *animation
		agentAnimation.AgentID = agentID
		ai.animations[agentID] = &agentAnimation
	}
	ai.mutex.Unlock()
}

// ClearAnimation removes any active animation for an agent
func (ai *AgentIndicators) ClearAnimation(agentID string) {
	ai.mutex.Lock()
	defer ai.mutex.Unlock()

	delete(ai.animations, agentID)
	delete(ai.blinkStates, agentID)
}

// GetCurrentIndicator returns the current visual indicator for an agent
func (ai *AgentIndicators) GetCurrentIndicator(agentID string) string {
	ai.mutex.RLock()
	defer ai.mutex.RUnlock()

	animation, exists := ai.animations[agentID]
	if !exists || !ai.enabled {
		return "⚪" // Default inactive indicator
	}

	if animation.CurrentFrame >= len(animation.Frames) {
		animation.CurrentFrame = 0
	}

	return animation.Frames[animation.CurrentFrame]
}

// GetAnimationContext returns the context of the current animation
func (ai *AgentIndicators) GetAnimationContext(agentID string) string {
	ai.mutex.RLock()
	defer ai.mutex.RUnlock()

	animation, exists := ai.animations[agentID]
	if !exists {
		return ""
	}

	return animation.Context
}

// IsAnimating returns true if an agent has an active animation
func (ai *AgentIndicators) IsAnimating(agentID string) bool {
	ai.mutex.RLock()
	defer ai.mutex.RUnlock()

	_, exists := ai.animations[agentID]
	return exists && ai.enabled
}

// EnableAnimations enables all animations
func (ai *AgentIndicators) EnableAnimations() {
	ai.enabled = true
}

// DisableAnimations disables all animations
func (ai *AgentIndicators) DisableAnimations() {
	ai.enabled = false
}

// SetAnimationSpeed sets the speed of animations
func (ai *AgentIndicators) SetAnimationSpeed(duration time.Duration) {
	ai.animationSpeed = duration

	// Restart ticker with new speed if running
	if ai.updateTicker != nil {
		ai.StopAnimations()
		ai.StartAnimations()
	}
}

// UpdateAnimation updates an agent's animation state based on AgentState
func (ai *AgentIndicators) UpdateAnimation(agentID string, state AgentState) {
	switch state {
	case AgentThinking:
		ai.SetThinkingAnimation(agentID)
	case AgentWorking:
		ai.SetWorkingAnimation(agentID, "processing")
	case AgentIdle, AgentOffline:
		ai.ClearAnimation(agentID)
	case AgentBlocked:
		ai.setBlinkingAnimation(agentID, "⏳")
	}
}

// setBlinkingAnimation sets a blinking animation with a specific icon
func (ai *AgentIndicators) setBlinkingAnimation(agentID, icon string) {
	if !ai.enabled {
		return
	}

	animation := &Animation{
		Type:         AnimationBlink,
		Duration:     0, // Infinite until stopped
		StartTime:    time.Now(),
		Frames:       []string{icon, "⚪"},
		CurrentFrame: 0,
		Infinite:     true,
		AgentID:      agentID,
		Context:      "blocked",
	}

	ai.mutex.Lock()
	ai.animations[agentID] = animation
	ai.blinkStates[agentID] = false
	ai.mutex.Unlock()
}

// Animation frame sequences

// getThinkingFrames returns the thinking animation sequence
func (ai *AgentIndicators) getThinkingFrames() []string {
	return []string{
		"🤔", // Thinking face
		"💭", // Thought bubble
		"🧠", // Brain
		"💡", // Light bulb (idea)
		"🤔", // Back to thinking
		"💭", // Thought bubble
	}
}

// getWorkingFrames returns the working animation sequence
func (ai *AgentIndicators) getWorkingFrames() []string {
	return []string{
		"⚙️", // Gear
		"🔄",  // Arrows in circle
		"⚡",  // Lightning bolt
		"🛠️", // Hammer and wrench
		"⚙️", // Gear
		"🔧",  // Wrench
	}
}

// getToolExecutionFrames returns tool-specific animation frames
func (ai *AgentIndicators) getToolExecutionFrames(toolName string) []string {
	// Tool-specific animations based on tool type
	switch {
	case contains(toolName, "file"):
		return []string{"📁", "📄", "📝", "💾"}
	case contains(toolName, "shell"), contains(toolName, "exec"):
		return []string{"💻", "⌨️", "🖥️", "📟"}
	case contains(toolName, "web"), contains(toolName, "http"):
		return []string{"🌐", "📡", "🔗", "📊"}
	case contains(toolName, "search"):
		return []string{"🔍", "🔎", "📈", "🎯"}
	case contains(toolName, "corpus"):
		return []string{"📚", "📖", "📝", "🧠"}
	default:
		// Generic tool animation
		return []string{"🔨", "🛠️", "⚔️", "🔧"}
	}
}

// getCoordinationFrames returns coordination animation sequence
func (ai *AgentIndicators) getCoordinationFrames() []string {
	return []string{
		"⚡", // Lightning (start)
		"🔗", // Link
		"👥", // People
		"🤝", // Handshake
		"🔗", // Link
		"⚡", // Lightning (loop back)
	}
}

// GetProgressAnimation returns a progress animation for a given percentage
func (ai *AgentIndicators) GetProgressAnimation(progress float64) string {
	// Create a rotating progress indicator
	frames := []string{"◐", "◓", "◑", "◒"}
	frameIndex := int(time.Now().UnixNano()/int64(ai.animationSpeed)) % len(frames)

	progressIcon := frames[frameIndex]

	// Add percentage if significant progress
	if progress > 0.1 {
		return fmt.Sprintf("%s %.0f%%", progressIcon, progress*100)
	}

	return progressIcon
}

// GetStatusWithAnimation returns the status icon with animation if active
func (ai *AgentIndicators) GetStatusWithAnimation(agentID string, baseStatus string) string {
	if !ai.enabled {
		return baseStatus
	}

	indicator := ai.GetCurrentIndicator(agentID)
	if indicator == "⚪" {
		return baseStatus // No animation, return base status
	}

	return indicator
}

// GetCoordinationIndicator returns a special indicator for coordination events
func (ai *AgentIndicators) GetCoordinationIndicator() string {
	frames := []string{"🔗", "⚡", "🤝", "👥"}
	frameIndex := int(time.Now().UnixNano()/int64(ai.animationSpeed)) % len(frames)
	return frames[frameIndex]
}

// GetActiveAnimations returns a list of currently active animations
func (ai *AgentIndicators) GetActiveAnimations() map[string]string {
	ai.mutex.RLock()
	defer ai.mutex.RUnlock()

	active := make(map[string]string)
	for agentID, animation := range ai.animations {
		active[agentID] = animation.Type.String()
	}
	return active
}

// GetAnimationStats returns statistics about current animations
func (ai *AgentIndicators) GetAnimationStats() map[string]interface{} {
	ai.mutex.RLock()
	defer ai.mutex.RUnlock()

	stats := map[string]interface{}{
		"total_animations": len(ai.animations),
		"enabled":          ai.enabled,
		"animation_speed":  ai.animationSpeed.String(),
		"types":            make(map[string]int),
	}

	// Count animation types
	typeCounts := make(map[string]int)
	for _, animation := range ai.animations {
		typeCounts[animation.Type.String()]++
	}
	stats["types"] = typeCounts

	return stats
}

// SetupDefaultAnimations configures standard animation patterns
func (ai *AgentIndicators) SetupDefaultAnimations() {
	// Start the animation system
	ai.StartAnimations()

	// Set reasonable defaults
	ai.SetAnimationSpeed(400 * time.Millisecond) // Smooth but not too fast
	ai.EnableAnimations()
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr ||
		(len(s) > len(substr) &&
			(s[:len(substr)] == substr ||
				s[len(s)-len(substr):] == substr ||
				containsMiddle(s, substr))))
}

// Helper function to check substring in middle
func containsMiddle(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
