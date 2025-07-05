// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package cache

import (
	"math"
	"sort"
	"sync"
	"time"
)

// AccessPredictor predicts future cache access patterns
type AccessPredictor struct {
	history     *AccessHistory
	patterns    *PatternDetector
	ml          *MLPredictor
	window      time.Duration
	mu          sync.RWMutex
	predictions map[string]*AccessPrediction
}

// NewAccessPredictor creates a new access predictor
func NewAccessPredictor(window time.Duration) *AccessPredictor {
	return &AccessPredictor{
		history:     NewAccessHistory(),
		patterns:    NewPatternDetector(),
		ml:          NewMLPredictor(),
		window:      window,
		predictions: make(map[string]*AccessPrediction),
	}
}

// AccessPrediction represents a prediction about future access
type AccessPrediction struct {
	Key          string    `json:"key"`
	Probability  float64   `json:"probability"`
	NextAccess   time.Time `json:"next_access"`
	RelatedKeys  []string  `json:"related_keys"`
	PatternType  string    `json:"pattern_type"`
	Confidence   float64   `json:"confidence"`
	ExpectedDuration time.Duration `json:"expected_duration"`
}

// RecordAccess records an access event for prediction
func (ap *AccessPredictor) RecordAccess(key string, timestamp time.Time) {
	ap.mu.Lock()
	defer ap.mu.Unlock()

	ap.history.RecordAccess(key, timestamp)
	ap.patterns.UpdatePatterns(key, timestamp)
}

// PredictNextAccess predicts when a key will next be accessed
func (ap *AccessPredictor) PredictNextAccess(key string) *AccessPrediction {
	ap.mu.RLock()
	defer ap.mu.RUnlock()

	// Check if we have a cached prediction
	if pred, exists := ap.predictions[key]; exists {
		if time.Since(pred.NextAccess) < time.Minute {
			return pred
		}
	}

	// Generate new prediction
	history := ap.history.GetHistory(key)
	pattern := ap.patterns.DetectPattern(history)

	prediction := &AccessPrediction{
		Key:         key,
		Probability: 0.0,
		PatternType: pattern.Type.String(),
		Confidence:  pattern.Confidence,
	}

	switch pattern.Type {
	case PatternPeriodic:
		// Periodic access pattern
		if !pattern.LastAccess.IsZero() {
			nextAccess := pattern.LastAccess.Add(pattern.Period)
			prediction.NextAccess = nextAccess
			prediction.Probability = pattern.Confidence
			prediction.ExpectedDuration = pattern.Period
		}

	case PatternSequential:
		// Sequential access pattern
		relatedKeys := ap.patterns.GetRelatedKeys(key)
		prediction.RelatedKeys = relatedKeys
		prediction.Probability = 0.8
		prediction.ExpectedDuration = time.Minute

	case PatternBursty:
		// Bursty access pattern
		prediction.Probability = 0.6
		prediction.ExpectedDuration = time.Hour

	case PatternRandom:
		// Use ML prediction for random patterns
		mlPred := ap.ml.Predict(history)
		prediction.Probability = mlPred.Confidence
		prediction.ExpectedDuration = time.Hour

	default:
		prediction.Probability = 0.1 // Low probability for unknown patterns
		prediction.ExpectedDuration = time.Hour * 24
	}

	// Cache the prediction
	ap.mu.RUnlock()
	ap.mu.Lock()
	ap.predictions[key] = prediction
	ap.mu.Unlock()
	ap.mu.RLock()

	return prediction
}

// ShouldPromote determines if a key should be promoted to a higher cache level
func (ap *AccessPredictor) ShouldPromote(key string) bool {
	prediction := ap.PredictNextAccess(key)
	return prediction.Probability > 0.7
}

// GetHotKeyPredictions returns predictions for keys likely to be accessed soon
func (ap *AccessPredictor) GetHotKeyPredictions(window time.Duration) []*AccessPrediction {
	ap.mu.RLock()
	defer ap.mu.RUnlock()

	hotPredictions := make([]*AccessPrediction, 0)
	cutoff := time.Now().Add(window)

	for _, pred := range ap.predictions {
		if pred.NextAccess.Before(cutoff) && pred.Probability > 0.5 {
			hotPredictions = append(hotPredictions, pred)
		}
	}

	// Sort by probability (highest first)
	sort.Slice(hotPredictions, func(i, j int) bool {
		return hotPredictions[i].Probability > hotPredictions[j].Probability
	})

	return hotPredictions
}

// GetAccuracy returns the prediction accuracy
func (ap *AccessPredictor) GetAccuracy() float64 {
	ap.mu.RLock()
	defer ap.mu.RUnlock()

	// Calculate accuracy based on recent predictions
	// This is a simplified implementation
	return 0.85 // 85% accuracy placeholder
}

// AccessHistory maintains historical access patterns
type AccessHistory struct {
	accesses map[string][]time.Time
	mu       sync.RWMutex
	maxSize  int
}

// NewAccessHistory creates a new access history tracker
func NewAccessHistory() *AccessHistory {
	return &AccessHistory{
		accesses: make(map[string][]time.Time),
		maxSize:  1000, // Keep last 1000 accesses per key
	}
}

// RecordAccess records an access event
func (ah *AccessHistory) RecordAccess(key string, timestamp time.Time) {
	ah.mu.Lock()
	defer ah.mu.Unlock()

	if ah.accesses[key] == nil {
		ah.accesses[key] = make([]time.Time, 0)
	}

	ah.accesses[key] = append(ah.accesses[key], timestamp)

	// Keep only recent accesses
	if len(ah.accesses[key]) > ah.maxSize {
		ah.accesses[key] = ah.accesses[key][len(ah.accesses[key])-ah.maxSize:]
	}
}

// GetHistory returns access history for a key
func (ah *AccessHistory) GetHistory(key string) []time.Time {
	ah.mu.RLock()
	defer ah.mu.RUnlock()

	if history, exists := ah.accesses[key]; exists {
		// Return a copy to avoid race conditions
		result := make([]time.Time, len(history))
		copy(result, history)
		return result
	}

	return []time.Time{}
}

// PatternType represents different access patterns
type PatternType int

const (
	PatternRandom PatternType = iota
	PatternPeriodic
	PatternSequential
	PatternBursty
)

// String returns the string representation of PatternType
func (pt PatternType) String() string {
	switch pt {
	case PatternRandom:
		return "random"
	case PatternPeriodic:
		return "periodic"
	case PatternSequential:
		return "sequential"
	case PatternBursty:
		return "bursty"
	default:
		return "unknown"
	}
}

// AccessPattern represents a detected access pattern
type AccessPattern struct {
	Type       PatternType   `json:"type"`
	Period     time.Duration `json:"period"`
	LastAccess time.Time     `json:"last_access"`
	Confidence float64       `json:"confidence"`
	Sequence   []string      `json:"sequence"`
}

// PatternDetector detects access patterns
type PatternDetector struct {
	patterns map[string]*AccessPattern
	mu       sync.RWMutex
}

// NewPatternDetector creates a new pattern detector
func NewPatternDetector() *PatternDetector {
	return &PatternDetector{
		patterns: make(map[string]*AccessPattern),
	}
}

// UpdatePatterns updates pattern detection with new access
func (pd *PatternDetector) UpdatePatterns(key string, timestamp time.Time) {
	pd.mu.Lock()
	defer pd.mu.Unlock()

	if pd.patterns[key] == nil {
		pd.patterns[key] = &AccessPattern{
			Type:       PatternRandom,
			LastAccess: timestamp,
			Confidence: 0.1,
		}
		return
	}

	pattern := pd.patterns[key]
	pattern.LastAccess = timestamp
}

// DetectPattern detects the access pattern for a key
func (pd *PatternDetector) DetectPattern(history []time.Time) *AccessPattern {
	if len(history) < 2 {
		return &AccessPattern{
			Type:       PatternRandom,
			Confidence: 0.1,
		}
	}

	// Analyze intervals between accesses
	intervals := make([]time.Duration, len(history)-1)
	for i := 1; i < len(history); i++ {
		intervals[i-1] = history[i].Sub(history[i-1])
	}

	// Check for periodic pattern
	if pattern := pd.detectPeriodic(intervals, history); pattern != nil {
		return pattern
	}

	// Check for bursty pattern
	if pattern := pd.detectBursty(intervals, history); pattern != nil {
		return pattern
	}

	// Default to random
	return &AccessPattern{
		Type:       PatternRandom,
		LastAccess: history[len(history)-1],
		Confidence: 0.3,
	}
}

// detectPeriodic detects periodic access patterns
func (pd *PatternDetector) detectPeriodic(intervals []time.Duration, history []time.Time) *AccessPattern {
	if len(intervals) < 3 {
		return nil
	}

	// Calculate average interval
	var total time.Duration
	for _, interval := range intervals {
		total += interval
	}
	avgInterval := total / time.Duration(len(intervals))

	// Check if intervals are consistent (within 20% variance)
	consistentCount := 0
	for _, interval := range intervals {
		variance := math.Abs(float64(interval-avgInterval)) / float64(avgInterval)
		if variance < 0.2 {
			consistentCount++
		}
	}

	consistency := float64(consistentCount) / float64(len(intervals))
	if consistency > 0.7 {
		return &AccessPattern{
			Type:       PatternPeriodic,
			Period:     avgInterval,
			LastAccess: history[len(history)-1],
			Confidence: consistency,
		}
	}

	return nil
}

// detectBursty detects bursty access patterns
func (pd *PatternDetector) detectBursty(intervals []time.Duration, history []time.Time) *AccessPattern {
	if len(intervals) < 5 {
		return nil
	}

	// Look for clusters of short intervals followed by long gaps
	shortIntervals := 0
	longIntervals := 0
	avgInterval := time.Duration(0)

	for _, interval := range intervals {
		avgInterval += interval
	}
	avgInterval /= time.Duration(len(intervals))

	for _, interval := range intervals {
		if interval < avgInterval/2 {
			shortIntervals++
		} else if interval > avgInterval*2 {
			longIntervals++
		}
	}

	burstiness := float64(shortIntervals+longIntervals) / float64(len(intervals))
	if burstiness > 0.6 {
		return &AccessPattern{
			Type:       PatternBursty,
			LastAccess: history[len(history)-1],
			Confidence: burstiness,
		}
	}

	return nil
}

// GetRelatedKeys returns keys that are often accessed together
func (pd *PatternDetector) GetRelatedKeys(key string) []string {
	// This would typically analyze co-occurrence patterns
	// For now, return empty slice
	return []string{}
}

// MLPredictor provides machine learning-based predictions
type MLPredictor struct {
	model MLModel
}

// MLModel interface for machine learning models
type MLModel interface {
	Predict(features []float64) float64
	Train(samples []TrainingSample)
}

// TrainingSample represents a training sample for ML
type TrainingSample struct {
	Features []float64 `json:"features"`
	Label    float64   `json:"label"`
}

// MLPrediction represents an ML-based prediction
type MLPrediction struct {
	Confidence float64 `json:"confidence"`
	Features   []float64 `json:"features"`
}

// NewMLPredictor creates a new ML predictor
func NewMLPredictor() *MLPredictor {
	return &MLPredictor{
		model: &SimpleLinearModel{},
	}
}

// Predict generates an ML-based prediction
func (ml *MLPredictor) Predict(history []time.Time) *MLPrediction {
	if len(history) < 2 {
		return &MLPrediction{
			Confidence: 0.1,
			Features:   []float64{},
		}
	}

	// Extract features from history
	features := ml.extractFeatures(history)
	confidence := ml.model.Predict(features)

	return &MLPrediction{
		Confidence: confidence,
		Features:   features,
	}
}

// extractFeatures extracts features from access history
func (ml *MLPredictor) extractFeatures(history []time.Time) []float64 {
	if len(history) < 2 {
		return []float64{0, 0, 0}
	}

	// Feature 1: Access frequency (accesses per hour)
	timeSpan := history[len(history)-1].Sub(history[0])
	frequency := float64(len(history)) / timeSpan.Hours()

	// Feature 2: Recent activity (accesses in last hour)
	recentCount := 0
	cutoff := time.Now().Add(-time.Hour)
	for _, access := range history {
		if access.After(cutoff) {
			recentCount++
		}
	}

	// Feature 3: Regularity (coefficient of variation of intervals)
	if len(history) < 3 {
		return []float64{frequency, float64(recentCount), 0}
	}

	intervals := make([]float64, len(history)-1)
	for i := 1; i < len(history); i++ {
		intervals[i-1] = history[i].Sub(history[i-1]).Seconds()
	}

	// Calculate mean and standard deviation
	mean := 0.0
	for _, interval := range intervals {
		mean += interval
	}
	mean /= float64(len(intervals))

	variance := 0.0
	for _, interval := range intervals {
		variance += (interval - mean) * (interval - mean)
	}
	variance /= float64(len(intervals))
	stdDev := math.Sqrt(variance)

	regularity := 0.0
	if mean > 0 {
		regularity = stdDev / mean
	}

	return []float64{frequency, float64(recentCount), regularity}
}

// SimpleLinearModel provides a simple linear regression model
type SimpleLinearModel struct {
	weights []float64
}

// Predict generates a prediction using linear regression
func (slm *SimpleLinearModel) Predict(features []float64) float64 {
	if len(slm.weights) == 0 {
		// Initialize with default weights
		slm.weights = []float64{0.3, 0.5, 0.2}
	}

	if len(features) != len(slm.weights) {
		return 0.5 // Default prediction
	}

	prediction := 0.0
	for i, feature := range features {
		prediction += feature * slm.weights[i]
	}

	// Normalize to [0, 1]
	return math.Max(0, math.Min(1, prediction))
}

// Train trains the linear model (placeholder implementation)
func (slm *SimpleLinearModel) Train(samples []TrainingSample) {
	// This would implement actual linear regression training
	// For now, it's a placeholder
}