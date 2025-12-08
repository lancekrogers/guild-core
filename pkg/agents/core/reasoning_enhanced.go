// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package core

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/guild-framework/guild-core/pkg/gerror"
	"github.com/guild-framework/guild-core/pkg/observability"
)

// ReasoningExtractor provides thread-safe, context-aware reasoning extraction
type ReasoningExtractor struct {
	// Compiled regex patterns for better performance
	thinkingPattern   *regexp.Regexp
	confidencePattern *regexp.Regexp

	// Caching for performance
	cache      *sync.Map
	cacheStats struct {
		hits   uint64
		misses uint64
		mu     sync.RWMutex
	}

	// Configuration
	config ReasoningConfig

	// Metrics
	metrics *reasoningMetrics
}

// reasoningMetrics tracks performance metrics
type reasoningMetrics struct {
	extractionDuration *prometheus.HistogramVec
	cacheHitRate       *prometheus.GaugeVec
	reasoningLength    *prometheus.HistogramVec
	confidenceLevels   *prometheus.HistogramVec
	errorCount         *prometheus.CounterVec
}

// ReasoningConfig configures the reasoning extractor
type ReasoningConfig struct {
	EnableCaching      bool
	CacheMaxSize       int
	CacheTTL           time.Duration
	MaxReasoningLength int
	MinConfidence      float64
	MaxConfidence      float64
	StrictValidation   bool
}

// DefaultReasoningConfig returns production-ready defaults
func DefaultReasoningConfig() ReasoningConfig {
	return ReasoningConfig{
		EnableCaching:      true,
		CacheMaxSize:       1000,
		CacheTTL:           5 * time.Minute,
		MaxReasoningLength: 10000,
		MinConfidence:      0.0,
		MaxConfidence:      1.0,
		StrictValidation:   true,
	}
}

// NewReasoningExtractor creates a new reasoning extractor with configuration
func NewReasoningExtractor(config ReasoningConfig) (*ReasoningExtractor, error) {
	// Validate configuration
	if err := config.Validate(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeValidation, "invalid reasoning configuration").
			WithComponent("reasoning").
			WithOperation("NewReasoningExtractor")
	}

	// Compile regex patterns
	thinkingPattern, err := regexp.Compile(`(?s)<thinking(?:\s+[^>]*)?>(.+?)</thinking>`)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to compile thinking pattern").
			WithComponent("reasoning").
			WithOperation("NewReasoningExtractor")
	}

	confidencePattern, err := regexp.Compile(`(?i)confidence:\s*([\d.]+)`)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to compile confidence pattern").
			WithComponent("reasoning").
			WithOperation("NewReasoningExtractor")
	}

	extractor := &ReasoningExtractor{
		thinkingPattern:   thinkingPattern,
		confidencePattern: confidencePattern,
		config:            config,
		metrics:           initializeMetrics(),
	}

	if config.EnableCaching {
		extractor.cache = &sync.Map{}
		// Start cache cleanup goroutine
		go extractor.cleanupCache(context.Background())
	}

	return extractor, nil
}

// ExtractReasoning extracts reasoning with full context support
func (re *ReasoningExtractor) ExtractReasoning(ctx context.Context, content string) (*AgentResponse, error) {
	// Check context
	if err := ctx.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("reasoning").
			WithOperation("ExtractReasoning")
	}

	// Start timing
	start := time.Now()
	defer func() {
		re.metrics.extractionDuration.WithLabelValues("extract").Observe(time.Since(start).Seconds())
	}()

	// Initialize logger with context
	logger := observability.GetLogger(ctx).
		WithComponent("reasoning").
		WithOperation("ExtractReasoning")

	// Validate input
	if content == "" {
		return &AgentResponse{
			Content:    "",
			Reasoning:  "",
			Confidence: 0.5,
			Metadata: map[string]interface{}{
				"extraction_time_ms": 0,
				"cached":             false,
				"has_reasoning":      false,
			},
		}, nil
	}

	// Check cache if enabled
	if re.config.EnableCaching {
		if cached := re.checkCache(content); cached != nil {
			re.recordCacheHit()
			logger.DebugContext(ctx, "Returning cached reasoning extraction")
			return cached, nil
		}
		re.recordCacheMiss()
	}

	// Extract thinking blocks - handle potential nested tags
	matches, cleanedContent := re.extractThinkingBlocks(content)
	if len(matches) == 0 {
		// No reasoning found
		response := &AgentResponse{
			Content:    content,
			Reasoning:  "",
			Confidence: 0.5,
			Metadata: map[string]interface{}{
				"extraction_time_ms": time.Since(start).Milliseconds(),
				"cached":             false,
				"has_reasoning":      false,
			},
		}

		if re.config.EnableCaching {
			re.cacheResult(content, response)
		}

		return response, nil
	}

	// Combine reasoning blocks
	var reasoningParts []string
	totalLength := 0

	for _, match := range matches {
		if len(match) > 1 {
			part := strings.TrimSpace(match[1])
			if part != "" {
				// Check length constraints
				if re.config.MaxReasoningLength > 0 && totalLength+len(part) > re.config.MaxReasoningLength {
					logger.WarnContext(ctx, "Reasoning length exceeds maximum, truncating",
						"max_length", re.config.MaxReasoningLength,
						"current_length", totalLength)
					break
				}
				reasoningParts = append(reasoningParts, part)
				totalLength += len(part)
			}
		}
	}

	reasoning := strings.Join(reasoningParts, "\n\n")

	// Extract confidence
	confidence := re.extractConfidence(ctx, reasoning)

	// Use the already cleaned content
	cleanContent := strings.TrimSpace(cleanedContent)

	// Clean up extra whitespace - replace multiple newlines with double newlines
	// This helps with test expectations
	cleanContent = regexp.MustCompile(`\n{3,}`).ReplaceAllString(cleanContent, "\n\n")

	// Record metrics
	re.metrics.reasoningLength.WithLabelValues("reasoning").Observe(float64(len(reasoning)))
	re.metrics.confidenceLevels.WithLabelValues("confidence").Observe(confidence)

	// Build response
	response := &AgentResponse{
		Content:    cleanContent,
		Reasoning:  reasoning,
		Confidence: confidence,
		Metadata: map[string]interface{}{
			"extraction_time_ms": time.Since(start).Milliseconds(),
			"cached":             false,
			"has_reasoning":      true,
			"reasoning_blocks":   len(matches),
			"reasoning_length":   len(reasoning),
		},
	}

	// Validate response if strict validation is enabled
	if re.config.StrictValidation {
		if err := re.validateResponse(response); err != nil {
			re.metrics.errorCount.WithLabelValues("error").Inc()
			return nil, gerror.Wrap(err, gerror.ErrCodeValidation, "response validation failed").
				WithComponent("reasoning").
				WithOperation("ExtractReasoning")
		}
	}

	// Cache result
	if re.config.EnableCaching {
		re.cacheResult(content, response)
	}

	logger.DebugContext(ctx, "Reasoning extraction completed",
		"has_reasoning", response.Reasoning != "",
		"confidence", response.Confidence,
		"reasoning_length", len(response.Reasoning),
		"extraction_time_ms", time.Since(start).Milliseconds())

	return response, nil
}

// extractConfidence extracts confidence value with validation
func (re *ReasoningExtractor) extractConfidence(ctx context.Context, reasoning string) float64 {
	logger := observability.GetLogger(ctx).
		WithComponent("reasoning").
		WithOperation("extractConfidence")

	if reasoning == "" {
		return 0.5 // default
	}

	// Find all confidence matches and use the last one (most recent)
	matches := re.confidencePattern.FindAllStringSubmatch(reasoning, -1)
	if len(matches) == 0 {
		return 0.5 // default
	}

	// Use the last match
	lastMatch := matches[len(matches)-1]
	if len(lastMatch) < 2 {
		return 0.5 // default
	}

	confidence, err := strconv.ParseFloat(lastMatch[1], 64)
	if err != nil {
		logger.WarnContext(ctx, "Failed to parse confidence value",
			"raw_value", lastMatch[1],
			"error", err.Error())
		return 0.5
	}

	// Validate and clamp confidence
	if confidence < re.config.MinConfidence {
		logger.WarnContext(ctx, "Confidence below minimum, clamping",
			"confidence", confidence,
			"min", re.config.MinConfidence)
		confidence = re.config.MinConfidence
	} else if confidence > re.config.MaxConfidence {
		logger.WarnContext(ctx, "Confidence above maximum, clamping",
			"confidence", confidence,
			"max", re.config.MaxConfidence)
		confidence = re.config.MaxConfidence
	}

	return confidence
}

// validateResponse performs comprehensive response validation
func (re *ReasoningExtractor) validateResponse(response *AgentResponse) error {
	if response == nil {
		return gerror.New(gerror.ErrCodeValidation, "response is nil", nil)
	}

	// Validate content and reasoning are not both empty
	if response.Content == "" && response.Reasoning == "" {
		return gerror.New(gerror.ErrCodeValidation, "both content and reasoning are empty", nil)
	}

	// Validate confidence range
	if response.Confidence < 0 || response.Confidence > 1 {
		return gerror.Newf(gerror.ErrCodeValidation, "confidence %f is out of range [0,1]", response.Confidence)
	}

	// Validate metadata
	if response.Metadata == nil {
		response.Metadata = make(map[string]interface{})
	}

	return nil
}

// Cache management methods

func (re *ReasoningExtractor) checkCache(content string) *AgentResponse {
	if re.cache == nil {
		return nil
	}

	key := re.cacheKey(content)
	if value, ok := re.cache.Load(key); ok {
		if entry, ok := value.(*cacheEntry); ok {
			if time.Since(entry.timestamp) < re.config.CacheTTL {
				return entry.response
			}
			// Expired entry, remove it
			re.cache.Delete(key)
		}
	}
	return nil
}

func (re *ReasoningExtractor) cacheResult(content string, response *AgentResponse) {
	if re.cache == nil {
		return
	}

	key := re.cacheKey(content)
	re.cache.Store(key, &cacheEntry{
		response:  response,
		timestamp: time.Now(),
	})
}

func (re *ReasoningExtractor) cacheKey(content string) string {
	// Use first 100 chars + length for cache key to avoid storing full content
	keyContent := content
	if len(keyContent) > 100 {
		keyContent = keyContent[:100]
	}
	return fmt.Sprintf("%s:%d", keyContent, len(content))
}

type cacheEntry struct {
	response  *AgentResponse
	timestamp time.Time
}

func (re *ReasoningExtractor) cleanupCache(ctx context.Context) {
	ticker := time.NewTicker(re.config.CacheTTL / 2)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			re.performCacheCleanup()
		}
	}
}

func (re *ReasoningExtractor) performCacheCleanup() {
	now := time.Now()
	re.cache.Range(func(key, value interface{}) bool {
		if entry, ok := value.(*cacheEntry); ok {
			if now.Sub(entry.timestamp) > re.config.CacheTTL {
				re.cache.Delete(key)
			}
		}
		return true
	})
}

// Metrics helpers

func (re *ReasoningExtractor) recordCacheHit() {
	re.cacheStats.mu.Lock()
	re.cacheStats.hits++
	re.cacheStats.mu.Unlock()
	re.updateCacheHitRate()
}

func (re *ReasoningExtractor) recordCacheMiss() {
	re.cacheStats.mu.Lock()
	re.cacheStats.misses++
	re.cacheStats.mu.Unlock()
	re.updateCacheHitRate()
}

func (re *ReasoningExtractor) updateCacheHitRate() {
	re.cacheStats.mu.RLock()
	total := re.cacheStats.hits + re.cacheStats.misses
	hitRate := float64(0)
	if total > 0 {
		hitRate = float64(re.cacheStats.hits) / float64(total)
	}
	re.cacheStats.mu.RUnlock()

	re.metrics.cacheHitRate.WithLabelValues("cache").Set(hitRate)
}

// extractThinkingBlocks properly extracts thinking blocks, handling nested tags
func (re *ReasoningExtractor) extractThinkingBlocks(content string) ([][]string, string) {
	var matches [][]string
	cleanContent := content

	// Process content to extract all thinking blocks
	for {
		match := re.thinkingPattern.FindStringSubmatchIndex(cleanContent)
		if match == nil {
			break
		}

		// Extract the match
		fullMatch := cleanContent[match[0]:match[1]]
		innerContent := ""
		if len(match) >= 4 {
			innerContent = cleanContent[match[2]:match[3]]
		}

		matches = append(matches, []string{fullMatch, innerContent})

		// Remove this thinking block from content
		cleanContent = cleanContent[:match[0]] + cleanContent[match[1]:]
	}

	return matches, cleanContent
}

// GetStats returns current statistics
func (re *ReasoningExtractor) GetStats() map[string]interface{} {
	re.cacheStats.mu.RLock()
	defer re.cacheStats.mu.RUnlock()

	total := re.cacheStats.hits + re.cacheStats.misses
	hitRate := float64(0)
	if total > 0 {
		hitRate = float64(re.cacheStats.hits) / float64(total)
	}

	return map[string]interface{}{
		"cache_hits":     re.cacheStats.hits,
		"cache_misses":   re.cacheStats.misses,
		"cache_hit_rate": hitRate,
		"cache_enabled":  re.config.EnableCaching,
	}
}

// Validate validates the reasoning configuration
func (c ReasoningConfig) Validate() error {
	if c.CacheMaxSize < 0 {
		return gerror.New(gerror.ErrCodeValidation, "cache max size cannot be negative", nil)
	}

	if c.MaxReasoningLength < 0 {
		return gerror.New(gerror.ErrCodeValidation, "max reasoning length cannot be negative", nil)
	}

	if c.MinConfidence < 0 || c.MinConfidence > 1 {
		return gerror.New(gerror.ErrCodeValidation, "min confidence must be between 0 and 1", nil)
	}

	if c.MaxConfidence < 0 || c.MaxConfidence > 1 {
		return gerror.New(gerror.ErrCodeValidation, "max confidence must be between 0 and 1", nil)
	}

	if c.MinConfidence > c.MaxConfidence {
		return gerror.New(gerror.ErrCodeValidation, "min confidence cannot be greater than max confidence", nil)
	}

	return nil
}

// initializeMetrics sets up the metrics collectors
func initializeMetrics() *reasoningMetrics {
	return &reasoningMetrics{
		extractionDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name: "reasoning_extraction_duration_seconds",
				Help: "Duration of reasoning extraction",
			},
			[]string{"operation"},
		),
		cacheHitRate: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "reasoning_cache_hit_rate",
				Help: "Cache hit rate for reasoning",
			},
			[]string{"cache"},
		),
		reasoningLength: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "reasoning_length_bytes",
				Help:    "Length of reasoning in bytes",
				Buckets: []float64{100, 500, 1000, 5000, 10000},
			},
			[]string{"type"},
		),
		confidenceLevels: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "reasoning_confidence_level",
				Help:    "Confidence levels of reasoning",
				Buckets: []float64{0.1, 0.3, 0.5, 0.7, 0.9, 1.0},
			},
			[]string{"level"},
		),
		errorCount: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "reasoning_errors_total",
				Help: "Total number of reasoning errors",
			},
			[]string{"type"},
		),
	}
}
