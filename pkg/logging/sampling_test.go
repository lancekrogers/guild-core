package logging

import (
	"fmt"
	"log/slog"
	"testing"
	"time"
)

func TestLevelSampler(t *testing.T) {
	sampler := &LevelSampler{
		debugRate: 0.5,
		infoRate:  0.8,
	}

	// Test warn and error always pass
	for i := 0; i < 100; i++ {
		if !sampler.Sample(slog.LevelWarn, "warn message") {
			t.Error("Warn level should always be sampled")
		}
		if !sampler.Sample(slog.LevelError, "error message") {
			t.Error("Error level should always be sampled")
		}
	}

	// Test debug sampling (should be around 50%)
	debugCount := 0
	iterations := 1000
	for i := 0; i < iterations; i++ {
		if sampler.Sample(slog.LevelDebug, "debug message") {
			debugCount++
		}
	}

	// Allow 10% margin
	expectedDebug := int(float64(iterations) * sampler.debugRate)
	margin := int(float64(iterations) * 0.1)
	if debugCount < expectedDebug-margin || debugCount > expectedDebug+margin {
		t.Errorf("Debug sampling rate off: got %d/%d, expected ~%d", debugCount, iterations, expectedDebug)
	}

	// Test info sampling (should be around 80%)
	infoCount := 0
	for i := 0; i < iterations; i++ {
		if sampler.Sample(slog.LevelInfo, "info message") {
			infoCount++
		}
	}

	expectedInfo := int(float64(iterations) * sampler.infoRate)
	if infoCount < expectedInfo-margin || infoCount > expectedInfo+margin {
		t.Errorf("Info sampling rate off: got %d/%d, expected ~%d", infoCount, iterations, expectedInfo)
	}
}

func TestRateSampler(t *testing.T) {
	sampler := &RateSampler{
		rate: 0.1, // Sample 1 in 10
	}

	// Test that errors and warnings always pass
	for i := 0; i < 100; i++ {
		if !sampler.Sample(slog.LevelWarn, "warn") {
			t.Error("Warn should always be sampled")
		}
		if !sampler.Sample(slog.LevelError, "error") {
			t.Error("Error should always be sampled")
		}
	}

	// Test rate sampling
	count := 0
	iterations := 1000
	for i := 0; i < iterations; i++ {
		if sampler.Sample(slog.LevelInfo, "info") {
			count++
		}
	}

	// With rate 0.1, we expect about 100 samples
	expected := int(float64(iterations) * sampler.rate)
	margin := 20
	if count < expected-margin || count > expected+margin {
		t.Errorf("Rate sampling off: got %d/%d, expected ~%d", count, iterations, expected)
	}
}

func TestAdaptiveSampler(t *testing.T) {
	sampler := &AdaptiveSampler{
		targetRate: 10,
		window:     100 * time.Millisecond,
	}

	// Test that errors and warnings always pass
	for i := 0; i < 20; i++ {
		if !sampler.Sample(slog.LevelWarn, "warn") {
			t.Error("Warn should always be sampled")
		}
		if !sampler.Sample(slog.LevelError, "error") {
			t.Error("Error should always be sampled")
		}
	}

	// Test rate limiting within window
	count := 0
	for i := 0; i < 20; i++ {
		if sampler.Sample(slog.LevelInfo, "info") {
			count++
		}
	}

	if count > sampler.targetRate {
		t.Errorf("Sampled %d messages, but target rate is %d", count, sampler.targetRate)
	}

	// Test window reset
	time.Sleep(150 * time.Millisecond)

	count = 0
	for i := 0; i < 20; i++ {
		// Use different messages to avoid deduplication
		msg := fmt.Sprintf("info message %d", i)
		if sampler.Sample(slog.LevelInfo, msg) {
			count++
		}
	}

	if count == 0 {
		t.Error("No messages sampled after window reset")
	}
}

func TestAdaptiveSamplerDeduplication(t *testing.T) {
	sampler := &AdaptiveSampler{
		targetRate: 100,
		window:     1 * time.Second,
	}

	// Same message within 1 second should be deduplicated
	msg := "duplicate message"
	if !sampler.Sample(slog.LevelInfo, msg) {
		t.Error("First occurrence should be sampled")
	}

	// Immediate duplicate should be filtered
	if sampler.Sample(slog.LevelInfo, msg) {
		t.Error("Duplicate within 1 second should be filtered")
	}

	// Different message should pass
	if !sampler.Sample(slog.LevelInfo, "different message") {
		t.Error("Different message should be sampled")
	}

	// After 1 second, duplicate should pass again
	time.Sleep(1100 * time.Millisecond)
	if !sampler.Sample(slog.LevelInfo, msg) {
		t.Error("Duplicate after 1 second should be sampled")
	}
}

func TestCompositeSampler(t *testing.T) {
	// Create composite sampler with two restrictive samplers
	sampler1 := &RateSampler{rate: 0.5}
	sampler2 := &LevelSampler{debugRate: 0.5, infoRate: 1.0}

	composite := NewCompositeSampler(sampler1, sampler2)

	// Errors should always pass through both
	for i := 0; i < 10; i++ {
		if !composite.Sample(slog.LevelError, "error") {
			t.Error("Error should always be sampled")
		}
	}

	// Debug messages need to pass both samplers
	// With both at 0.5, we expect about 25% to pass
	count := 0
	iterations := 1000
	for i := 0; i < iterations; i++ {
		if composite.Sample(slog.LevelDebug, "debug") {
			count++
		}
	}

	// Expect around 25% (0.5 * 0.5)
	expected := int(float64(iterations) * 0.25)
	margin := int(float64(iterations) * 0.1)
	if count < expected-margin || count > expected+margin {
		t.Errorf("Composite sampling off: got %d/%d, expected ~%d", count, iterations, expected)
	}
}

func TestMessagePatternSampler(t *testing.T) {
	patterns := map[string]float64{
		"health check": 0.1, // Sample 10% of health checks
		"metrics":      0.5, // Sample 50% of metrics
		"debug":        0.0, // Never sample debug
	}

	sampler := NewMessagePatternSampler(patterns)

	// Test pattern matching
	healthCount := 0
	iterations := 100
	for i := 0; i < iterations; i++ {
		if sampler.Sample(slog.LevelInfo, "health check endpoint called") {
			healthCount++
		}
	}

	// Expect around 10% but allow for randomness (0-30%)
	if healthCount > 30 || healthCount < 0 {
		t.Errorf("Health check sampling off: got %d/%d", healthCount, iterations)
	}

	// Test exact match
	for i := 0; i < 10; i++ {
		if sampler.Sample(slog.LevelInfo, "debug") {
			t.Error("Debug pattern should never be sampled (rate 0.0)")
		}
	}

	// Test non-matching messages (should always pass)
	for i := 0; i < 10; i++ {
		if !sampler.Sample(slog.LevelInfo, "unmatched message") {
			t.Error("Unmatched messages should always be sampled")
		}
	}

	// Test adding new pattern
	sampler.AddPattern("error", 0.2)
	errorCount := 0
	for i := 0; i < iterations; i++ {
		if sampler.Sample(slog.LevelInfo, "error occurred") {
			errorCount++
		}
	}

	if errorCount > 30 || errorCount < 10 {
		t.Errorf("Error pattern sampling off: got %d/%d", errorCount, iterations)
	}
}

func TestContainsPattern(t *testing.T) {
	tests := []struct {
		msg      string
		pattern  string
		expected bool
	}{
		{"health check endpoint", "health check", true},
		{"endpoint health check", "health check", true},
		{"healthcheck", "health check", false},
		{"", "pattern", false},
		{"message", "", true},
		{"exact", "exact", true},
		{"prefix match", "prefix", true},
		{"suffix match", "match", true},
	}

	for _, tt := range tests {
		t.Run(tt.msg, func(t *testing.T) {
			result := containsPattern(tt.msg, tt.pattern)
			if result != tt.expected {
				t.Errorf("containsPattern(%q, %q) = %v, want %v", tt.msg, tt.pattern, result, tt.expected)
			}
		})
	}
}

func TestAdaptiveSamplerLoad(t *testing.T) {
	sampler := &AdaptiveSampler{
		targetRate: 10,
		window:     1 * time.Second,
	}

	// Generate some load
	for i := 0; i < 5; i++ {
		sampler.Sample(slog.LevelInfo, "message")
	}

	// Check load calculation
	load := sampler.Load()
	if load <= 0 {
		t.Error("Load should be greater than 0 after sampling")
	}
	if load > 10 {
		t.Errorf("Load %f should not exceed target rate", load)
	}
}
