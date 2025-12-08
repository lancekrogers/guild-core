// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package core

import (
	"context"
	"fmt"
	"math"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/guild-framework/guild-core/pkg/gerror"
	"github.com/guild-framework/guild-core/pkg/observability"
)

// DefaultReasoningAnalyzer provides analytics on reasoning chains
type DefaultReasoningAnalyzer struct {
	storage ReasoningStorage
	cache   *analyticsCache
	mu      sync.RWMutex
}

// analyticsCache caches computed analytics
type analyticsCache struct {
	correlations map[string]float64
	patterns     map[string][]*ReasoningPattern
	insights     map[string][]string
	mu           sync.RWMutex
}

// NewDefaultReasoningAnalyzer creates a new default reasoning analyzer
func NewDefaultReasoningAnalyzer(storage ReasoningStorage) (*DefaultReasoningAnalyzer, error) {
	if storage == nil {
		return nil, gerror.New(gerror.ErrCodeValidation, "storage cannot be nil", nil).
			WithComponent("reasoning_analyzer").
			WithOperation("NewDefaultReasoningAnalyzer")
	}

	return &DefaultReasoningAnalyzer{
		storage: storage,
		cache: &analyticsCache{
			correlations: make(map[string]float64),
			patterns:     make(map[string][]*ReasoningPattern),
			insights:     make(map[string][]string),
		},
	}, nil
}

// AnalyzeConfidenceCorrelation analyzes correlation between confidence and success
func (a *DefaultReasoningAnalyzer) AnalyzeConfidenceCorrelation(ctx context.Context, chains []*ReasoningChain) (float64, error) {
	if err := ctx.Err(); err != nil {
		return 0, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("reasoning_analyzer").
			WithOperation("AnalyzeConfidenceCorrelation")
	}

	logger := observability.GetLogger(ctx).
		WithComponent("reasoning_analyzer").
		WithOperation("AnalyzeConfidenceCorrelation")

	if len(chains) < 2 {
		return 0, gerror.New(gerror.ErrCodeValidation, "need at least 2 chains for correlation analysis", nil).
			WithComponent("reasoning_analyzer").
			WithOperation("AnalyzeConfidenceCorrelation")
	}

	// Calculate Pearson correlation coefficient
	var sumX, sumY, sumXY, sumX2, sumY2 float64
	n := float64(len(chains))

	for _, chain := range chains {
		x := chain.Confidence
		y := 0.0
		if chain.Success {
			y = 1.0
		}

		sumX += x
		sumY += y
		sumXY += x * y
		sumX2 += x * x
		sumY2 += y * y
	}

	// Pearson correlation formula
	numerator := n*sumXY - sumX*sumY
	denominator := math.Sqrt((n*sumX2 - sumX*sumX) * (n*sumY2 - sumY*sumY))

	if denominator == 0 {
		logger.WarnContext(ctx, "Zero denominator in correlation calculation")
		return 0, nil
	}

	correlation := numerator / denominator

	// Cache result
	a.cache.mu.Lock()
	cacheKey := fmt.Sprintf("corr_%d", len(chains))
	a.cache.correlations[cacheKey] = correlation
	a.cache.mu.Unlock()

	logger.InfoContext(ctx, "Calculated confidence-success correlation",
		"correlation", correlation,
		"sample_size", len(chains))

	return correlation, nil
}

// IdentifyPatterns identifies common reasoning patterns
func (a *DefaultReasoningAnalyzer) IdentifyPatterns(ctx context.Context, chains []*ReasoningChain) ([]*ReasoningPattern, error) {
	if err := ctx.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("reasoning_analyzer").
			WithOperation("IdentifyPatterns")
	}

	logger := observability.GetLogger(ctx).
		WithComponent("reasoning_analyzer").
		WithOperation("IdentifyPatterns")

	// Pattern extraction strategies
	patterns := make(map[string]*patternInfo)

	// Strategy 1: Common phrases
	phrasePatterns := a.extractPhrasePatterns(chains)
	for id, info := range phrasePatterns {
		patterns[id] = info
	}

	// Strategy 2: Structural patterns (e.g., numbered lists, bullet points)
	structuralPatterns := a.extractStructuralPatterns(chains)
	for id, info := range structuralPatterns {
		if existing, ok := patterns[id]; ok {
			existing.occurrences += info.occurrences
			existing.examples = append(existing.examples, info.examples...)
		} else {
			patterns[id] = info
		}
	}

	// Strategy 3: Confidence patterns
	confidencePatterns := a.extractConfidencePatterns(chains)
	for id, info := range confidencePatterns {
		patterns[id] = info
	}

	// Convert to ReasoningPattern objects
	var result []*ReasoningPattern
	for id, info := range patterns {
		if info.occurrences < 3 { // Minimum threshold
			continue
		}

		pattern := &ReasoningPattern{
			ID:          id,
			Pattern:     info.pattern,
			TaskType:    info.taskType,
			Occurrences: info.occurrences,
			AvgSuccess:  info.avgSuccess,
			Examples:    info.examples,
			Metadata: map[string]interface{}{
				"confidence_range": info.confidenceRange,
				"pattern_type":     info.patternType,
			},
		}

		result = append(result, pattern)
	}

	// Sort by occurrences
	sort.Slice(result, func(i, j int) bool {
		return result[i].Occurrences > result[j].Occurrences
	})

	logger.InfoContext(ctx, "Identified reasoning patterns",
		"total_patterns", len(result),
		"chains_analyzed", len(chains))

	return result, nil
}

// CompareAgents compares reasoning performance between agents
func (a *DefaultReasoningAnalyzer) CompareAgents(ctx context.Context, agentIDs []string, startTime, endTime time.Time) (map[string]*ReasoningStats, error) {
	if err := ctx.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("reasoning_analyzer").
			WithOperation("CompareAgents")
	}

	logger := observability.GetLogger(ctx).
		WithComponent("reasoning_analyzer").
		WithOperation("CompareAgents")

	results := make(map[string]*ReasoningStats)
	var wg sync.WaitGroup
	var mu sync.Mutex
	errors := make(chan error, len(agentIDs))

	for _, agentID := range agentIDs {
		wg.Add(1)
		go func(id string) {
			defer wg.Done()

			stats, err := a.storage.GetStats(ctx, id, startTime, endTime)
			if err != nil {
				errors <- gerror.Wrap(err, gerror.ErrCodeInternal, "failed to get stats").
					WithDetails("agent_id", id)
				return
			}

			mu.Lock()
			results[id] = stats
			mu.Unlock()
		}(agentID)
	}

	wg.Wait()
	close(errors)

	// Check for errors
	for err := range errors {
		logger.WithError(err).WarnContext(ctx, "Failed to get stats for agent")
	}

	if len(results) == 0 {
		return nil, gerror.New(gerror.ErrCodeNotFound, "no stats available for any agent", nil).
			WithComponent("reasoning_analyzer").
			WithOperation("CompareAgents")
	}

	logger.InfoContext(ctx, "Compared agent reasoning performance",
		"agents_compared", len(results),
		"time_range", endTime.Sub(startTime))

	return results, nil
}

// GenerateInsights generates actionable insights from reasoning data
func (a *DefaultReasoningAnalyzer) GenerateInsights(ctx context.Context, stats *ReasoningStats) ([]string, error) {
	if err := ctx.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("reasoning_analyzer").
			WithOperation("GenerateInsights")
	}

	if stats == nil {
		return nil, gerror.New(gerror.ErrCodeValidation, "stats cannot be nil", nil).
			WithComponent("reasoning_analyzer").
			WithOperation("GenerateInsights")
	}

	var insights []string

	// Insight 1: Confidence analysis
	if stats.AvgConfidence < 0.5 {
		insights = append(insights, fmt.Sprintf(
			"Low average confidence (%.2f) suggests the agent may need better training or clearer instructions",
			stats.AvgConfidence,
		))
	} else if stats.AvgConfidence > 0.85 {
		insights = append(insights, fmt.Sprintf(
			"High average confidence (%.2f) indicates the agent is performing well on familiar tasks",
			stats.AvgConfidence,
		))
	}

	// Insight 2: Success rate analysis
	if stats.SuccessRate < 0.7 {
		insights = append(insights, fmt.Sprintf(
			"Success rate of %.1f%% is below target. Consider reviewing failed tasks for patterns",
			stats.SuccessRate*100,
		))
	}

	// Insight 3: Task type distribution
	if len(stats.TaskTypeDistrib) > 0 {
		mostCommon := a.findMostCommonTaskType(stats.TaskTypeDistrib)
		insights = append(insights, fmt.Sprintf(
			"Most common task type is '%s' (%.1f%% of tasks). Consider optimizing prompts for this type",
			mostCommon.taskType,
			mostCommon.percentage,
		))
	}

	// Insight 4: Performance trends
	if stats.AvgDuration > 30*time.Second {
		insights = append(insights, fmt.Sprintf(
			"Average task duration of %s is high. Consider breaking down complex tasks",
			stats.AvgDuration,
		))
	}

	// Insight 5: Confidence distribution
	lowConfCount := stats.ConfidenceDistrib["very_low"] + stats.ConfidenceDistrib["low"]
	if float64(lowConfCount)/float64(stats.TotalChains) > 0.3 {
		insights = append(insights,
			"Over 30% of tasks have low confidence. Review these tasks to identify knowledge gaps",
		)
	}

	return insights, nil
}

// Helper types and methods

type patternInfo struct {
	pattern         string
	taskType        string
	occurrences     int
	avgSuccess      float64
	examples        []string
	confidenceRange [2]float64
	patternType     string
}

func (a *DefaultReasoningAnalyzer) extractPhrasePatterns(chains []*ReasoningChain) map[string]*patternInfo {
	patterns := make(map[string]*patternInfo)
	phraseRegex := regexp.MustCompile(`(?i)(I should|I need to|First,|Let me|Consider|Analyze)`)

	for _, chain := range chains {
		matches := phraseRegex.FindAllString(chain.Reasoning, -1)
		for _, match := range matches {
			key := strings.ToLower(match)
			if info, ok := patterns[key]; ok {
				info.occurrences++
				if chain.Success {
					info.avgSuccess = (info.avgSuccess*float64(info.occurrences-1) + 1) / float64(info.occurrences)
				} else {
					info.avgSuccess = (info.avgSuccess * float64(info.occurrences-1)) / float64(info.occurrences)
				}
			} else {
				patterns[key] = &patternInfo{
					pattern:     match,
					taskType:    chain.TaskType,
					occurrences: 1,
					avgSuccess:  0,
					patternType: "phrase",
					examples:    []string{chain.ID},
				}
				if chain.Success {
					patterns[key].avgSuccess = 1
				}
			}
		}
	}

	return patterns
}

func (a *DefaultReasoningAnalyzer) extractStructuralPatterns(chains []*ReasoningChain) map[string]*patternInfo {
	patterns := make(map[string]*patternInfo)

	// Pattern detectors
	numberedListRegex := regexp.MustCompile(`\d+\.\s+\w+`)
	bulletPointRegex := regexp.MustCompile(`^[-*]\s+\w+`)

	for _, chain := range chains {
		var patternType string

		if numberedListRegex.MatchString(chain.Reasoning) {
			patternType = "numbered_list"
		} else if bulletPointRegex.MatchString(chain.Reasoning) {
			patternType = "bullet_points"
		} else if strings.Count(chain.Reasoning, "\n") > 5 {
			patternType = "multi_paragraph"
		} else {
			patternType = "simple"
		}

		if info, ok := patterns[patternType]; ok {
			info.occurrences++
			if len(info.examples) < 5 {
				info.examples = append(info.examples, chain.ID)
			}
		} else {
			patterns[patternType] = &patternInfo{
				pattern:     patternType,
				taskType:    chain.TaskType,
				occurrences: 1,
				patternType: "structural",
				examples:    []string{chain.ID},
			}
		}
	}

	return patterns
}

func (a *DefaultReasoningAnalyzer) extractConfidencePatterns(chains []*ReasoningChain) map[string]*patternInfo {
	patterns := make(map[string]*patternInfo)

	// Group by confidence ranges
	ranges := []struct {
		name string
		min  float64
		max  float64
	}{
		{"very_low_confidence", 0.0, 0.3},
		{"low_confidence", 0.3, 0.5},
		{"medium_confidence", 0.5, 0.7},
		{"high_confidence", 0.7, 0.9},
		{"very_high_confidence", 0.9, 1.0},
	}

	for _, r := range ranges {
		patterns[r.name] = &patternInfo{
			pattern:         r.name,
			patternType:     "confidence",
			confidenceRange: [2]float64{r.min, r.max},
			examples:        []string{},
		}
	}

	for _, chain := range chains {
		for _, r := range ranges {
			if chain.Confidence >= r.min && chain.Confidence <= r.max {
				info := patterns[r.name]
				info.occurrences++
				if len(info.examples) < 3 {
					info.examples = append(info.examples, chain.ID)
				}
				if chain.Success {
					info.avgSuccess = (info.avgSuccess*float64(info.occurrences-1) + 1) / float64(info.occurrences)
				} else {
					info.avgSuccess = (info.avgSuccess * float64(info.occurrences-1)) / float64(info.occurrences)
				}
				break
			}
		}
	}

	return patterns
}

type taskTypeInfo struct {
	taskType   string
	count      int
	percentage float64
}

func (a *DefaultReasoningAnalyzer) findMostCommonTaskType(distribution map[string]int) taskTypeInfo {
	var total int
	var mostCommon taskTypeInfo

	for _, count := range distribution {
		total += count
	}

	for taskType, count := range distribution {
		if count > mostCommon.count {
			mostCommon = taskTypeInfo{
				taskType: taskType,
				count:    count,
			}
		}
	}

	if total > 0 {
		mostCommon.percentage = float64(mostCommon.count) / float64(total) * 100
	}

	return mostCommon
}
