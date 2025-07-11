package logging

import (
	"log/slog"
	"math"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"
)

// LevelSampler samples logs based on their level
type LevelSampler struct {
	debugRate float64
	infoRate  float64
	mu        sync.Mutex
	rand      *rand.Rand
}

// Sample determines if a log should be sampled based on level
func (s *LevelSampler) Sample(level slog.Level, msg string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.rand == nil {
		s.rand = rand.New(rand.NewSource(time.Now().UnixNano()))
	}

	switch level {
	case slog.LevelDebug:
		return s.rand.Float64() < s.debugRate
	case slog.LevelInfo:
		return s.rand.Float64() < s.infoRate
	default:
		// Always log warnings and errors
		return true
	}
}

// RateSampler samples logs at a fixed rate
type RateSampler struct {
	rate    float64
	counter atomic.Uint64
}

// Sample determines if a log should be sampled based on rate
func (s *RateSampler) Sample(level slog.Level, msg string) bool {
	// Always log errors and warnings
	if level >= slog.LevelWarn {
		return true
	}

	// Simple counter-based sampling
	count := s.counter.Add(1)
	threshold := uint64(1.0 / s.rate)
	return count%threshold == 0
}

// AdaptiveSampler adjusts sampling rate based on load
type AdaptiveSampler struct {
	targetRate  int
	window      time.Duration
	currentLoad uint64 // atomic float64 stored as uint64

	mu          sync.Mutex
	windowStart time.Time
	windowCount int
	decisions   map[string]time.Time // Track recent decisions to avoid duplicates
}

// Sample determines if a log should be sampled based on current load
func (s *AdaptiveSampler) Sample(level slog.Level, msg string) bool {
	// Always log errors and warnings
	if level >= slog.LevelWarn {
		return true
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()

	// Initialize window if needed
	if s.windowStart.IsZero() {
		s.windowStart = now
		s.decisions = make(map[string]time.Time)
	}

	// Reset window if expired
	if now.Sub(s.windowStart) > s.window {
		s.windowStart = now
		s.windowCount = 0
		// Clean old decisions
		for key, timestamp := range s.decisions {
			if now.Sub(timestamp) > s.window*2 {
				delete(s.decisions, key)
			}
		}
	}

	// Check if we've seen this message recently
	key := msg + level.String()
	if lastSeen, ok := s.decisions[key]; ok {
		if now.Sub(lastSeen) < time.Second {
			// Skip duplicate messages within 1 second
			return false
		}
	}

	// Check if we're within rate limit
	if s.windowCount >= s.targetRate {
		return false
	}

	// Log this message
	s.windowCount++
	s.decisions[key] = now
	load := float64(s.windowCount) / s.window.Seconds()
	atomic.StoreUint64(&s.currentLoad, math.Float64bits(load))

	return true
}

// Load returns the current logging load (logs per second)
func (s *AdaptiveSampler) Load() float64 {
	bits := atomic.LoadUint64(&s.currentLoad)
	return math.Float64frombits(bits)
}

// CompositeSampler combines multiple sampling strategies
type CompositeSampler struct {
	samplers []Sampler
}

// NewCompositeSampler creates a sampler that requires all samplers to pass
func NewCompositeSampler(samplers ...Sampler) *CompositeSampler {
	return &CompositeSampler{
		samplers: samplers,
	}
}

// Sample returns true only if all samplers return true
func (c *CompositeSampler) Sample(level slog.Level, msg string) bool {
	for _, sampler := range c.samplers {
		if !sampler.Sample(level, msg) {
			return false
		}
	}
	return true
}

// MessagePatternSampler samples based on message patterns
type MessagePatternSampler struct {
	patterns map[string]float64 // pattern -> sample rate
	mu       sync.RWMutex
	rand     *rand.Rand
}

// NewMessagePatternSampler creates a pattern-based sampler
func NewMessagePatternSampler(patterns map[string]float64) *MessagePatternSampler {
	return &MessagePatternSampler{
		patterns: patterns,
		rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// Sample determines if a log should be sampled based on message pattern
func (s *MessagePatternSampler) Sample(level slog.Level, msg string) bool {
	// Always log errors and warnings
	if level >= slog.LevelWarn {
		return true
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	// Check each pattern
	for pattern, rate := range s.patterns {
		if containsPattern(msg, pattern) {
			return s.rand.Float64() < rate
		}
	}

	// Default: sample everything not matching patterns
	return true
}

// AddPattern adds or updates a pattern's sample rate
func (s *MessagePatternSampler) AddPattern(pattern string, rate float64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.patterns[pattern] = rate
}

// containsPattern checks if a message contains a pattern (simple substring match)
func containsPattern(msg, pattern string) bool {
	// Simple implementation - could be enhanced with regex
	return len(msg) >= len(pattern) &&
		(msg == pattern ||
			len(msg) > len(pattern) &&
				containsSubstring(msg, pattern))
}

// containsSubstring is a simple substring search
func containsSubstring(s, substr string) bool {
	if len(substr) == 0 {
		return true
	}
	if len(substr) > len(s) {
		return false
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
