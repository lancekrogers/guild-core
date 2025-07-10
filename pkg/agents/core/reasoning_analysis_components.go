// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package core

import (
	"context"
	"fmt"
	"math"
	"strings"
	"sync"
	"time"
)

// QualityScorer evaluates reasoning quality
type QualityScorer struct {
	weights map[string]float64
	cache   *sync.Map
}

// NewQualityScorer creates a new quality scorer
func NewQualityScorer() *QualityScorer {
	return &QualityScorer{
		weights: map[string]float64{
			"coherence":    0.25,
			"completeness": 0.25,
			"depth":        0.20,
			"accuracy":     0.20,
			"innovation":   0.10,
		},
		cache: &sync.Map{},
	}
}

// Score evaluates the quality of a reasoning chain
func (qs *QualityScorer) Score(ctx context.Context, chain *ReasoningChainEnhanced) (QualityMetrics, error) {
	// Check cache
	if cached, ok := qs.cache.Load(chain.ID); ok {
		return cached.(QualityMetrics), nil
	}

	metrics := QualityMetrics{
		Coherence:    qs.scoreCoherence(chain),
		Completeness: qs.scoreCompleteness(chain),
		Depth:        qs.scoreDepth(chain),
		Accuracy:     qs.scoreAccuracy(chain),
		Innovation:   qs.scoreInnovation(chain),
	}

	// Calculate weighted overall score
	metrics.Overall = 0
	for metric, weight := range qs.weights {
		switch metric {
		case "coherence":
			metrics.Overall += metrics.Coherence * weight
		case "completeness":
			metrics.Overall += metrics.Completeness * weight
		case "depth":
			metrics.Overall += metrics.Depth * weight
		case "accuracy":
			metrics.Overall += metrics.Accuracy * weight
		case "innovation":
			metrics.Overall += metrics.Innovation * weight
		}
	}

	// Cache result
	qs.cache.Store(chain.ID, metrics)

	return metrics, nil
}

// scoreCoherence evaluates how well thoughts connect
func (qs *QualityScorer) scoreCoherence(chain *ReasoningChainEnhanced) float64 {
	if len(chain.Blocks) < 2 {
		return 1.0 // Single block is perfectly coherent
	}

	coherenceScore := 0.0
	transitions := 0

	for i := 1; i < len(chain.Blocks); i++ {
		prev := chain.Blocks[i-1]
		curr := chain.Blocks[i]

		// Check logical flow
		transitionScore := qs.evaluateTransition(prev, curr)
		coherenceScore += transitionScore
		transitions++
	}

	// Check for circular reasoning
	if qs.hasCircularReasoning(chain) {
		coherenceScore *= 0.7 // Penalty for circular logic
	}

	return math.Min(coherenceScore/float64(transitions), 1.0)
}

// evaluateTransition scores the logical flow between blocks
func (qs *QualityScorer) evaluateTransition(prev, curr *ThinkingBlock) float64 {
	score := 0.5 // Base score

	// Parent-child relationship is good
	if curr.ParentID != nil && *curr.ParentID == prev.ID {
		score += 0.3
	}

	// Type transitions matter
	goodTransitions := map[ThinkingType][]ThinkingType{
		ThinkingTypeAnalysis:       {ThinkingTypePlanning, ThinkingTypeDecisionMaking},
		ThinkingTypePlanning:       {ThinkingTypeToolSelection, ThinkingTypeVerification},
		ThinkingTypeDecisionMaking: {ThinkingTypeVerification, ThinkingTypeToolSelection},
		ThinkingTypeErrorRecovery:  {ThinkingTypeAnalysis, ThinkingTypePlanning},
	}

	if validNext, ok := goodTransitions[prev.Type]; ok {
		for _, validType := range validNext {
			if curr.Type == validType {
				score += 0.2
				break
			}
		}
	}

	return score
}

// hasCircularReasoning detects circular logic
func (qs *QualityScorer) hasCircularReasoning(chain *ReasoningChainEnhanced) bool {
	// Simple check: look for repeated content patterns
	contentMap := make(map[string]int)

	for _, block := range chain.Blocks {
		// Normalize content for comparison
		normalized := strings.ToLower(strings.TrimSpace(block.Content))
		words := strings.Fields(normalized)

		// Check for significant repeated phrases (3+ words)
		for i := 0; i <= len(words)-3; i++ {
			phrase := strings.Join(words[i:i+3], " ")
			contentMap[phrase]++

			if contentMap[phrase] > 2 {
				return true // Same phrase appears too often
			}
		}
	}

	return false
}

// scoreCompleteness evaluates coverage of problem space
func (qs *QualityScorer) scoreCompleteness(chain *ReasoningChainEnhanced) float64 {
	requiredTypes := []ThinkingType{
		ThinkingTypeAnalysis,
		ThinkingTypePlanning,
		ThinkingTypeVerification,
	}

	presentTypes := make(map[ThinkingType]bool)
	for _, block := range chain.Blocks {
		presentTypes[block.Type] = true
	}

	// Count required types present
	present := 0
	for _, reqType := range requiredTypes {
		if presentTypes[reqType] {
			present++
		}
	}

	baseScore := float64(present) / float64(len(requiredTypes))

	// Bonus for decision points
	decisionCount := 0
	for _, block := range chain.Blocks {
		decisionCount += len(block.DecisionPoints)
	}

	if decisionCount > 0 {
		baseScore = math.Min(baseScore+0.1*float64(decisionCount), 1.0)
	}

	return baseScore
}

// scoreDepth evaluates level of analysis
func (qs *QualityScorer) scoreDepth(chain *ReasoningChainEnhanced) float64 {
	if len(chain.Blocks) == 0 {
		return 0
	}

	// Average depth of blocks
	totalDepth := 0
	maxDepth := 0

	for _, block := range chain.Blocks {
		totalDepth += block.Depth
		if block.Depth > maxDepth {
			maxDepth = block.Depth
		}
	}

	avgDepth := float64(totalDepth) / float64(len(chain.Blocks))

	// Score based on both average and max depth
	depthScore := (avgDepth/3.0 + float64(maxDepth)/5.0) / 2.0

	// Consider structured data presence
	structuredCount := 0
	for _, block := range chain.Blocks {
		if block.StructuredData != nil {
			structuredCount++
		}
	}

	structureBonus := float64(structuredCount) / float64(len(chain.Blocks)) * 0.3

	return math.Min(depthScore+structureBonus, 1.0)
}

// scoreAccuracy evaluates correctness (simplified without ground truth)
func (qs *QualityScorer) scoreAccuracy(chain *ReasoningChainEnhanced) float64 {
	// Base accuracy on confidence and error recovery
	avgConfidence := 0.0
	errorBlocks := 0

	for _, block := range chain.Blocks {
		avgConfidence += block.Confidence
		if block.Type == ThinkingTypeErrorRecovery {
			errorBlocks++
		}
	}

	if len(chain.Blocks) > 0 {
		avgConfidence /= float64(len(chain.Blocks))
	}

	// Penalize for errors
	errorPenalty := 1.0 - (float64(errorBlocks) * 0.1)
	errorPenalty = math.Max(errorPenalty, 0.5)

	return avgConfidence * errorPenalty
}

// scoreInnovation evaluates novel approaches
func (qs *QualityScorer) scoreInnovation(chain *ReasoningChainEnhanced) float64 {
	// Look for unique patterns and approaches
	innovationScore := 0.3 // Base score

	// Multiple alternatives considered
	totalAlternatives := 0
	for _, block := range chain.Blocks {
		for _, dp := range block.DecisionPoints {
			totalAlternatives += len(dp.Alternatives)
		}
	}

	if totalAlternatives > 3 {
		innovationScore += 0.3
	}

	// Strategy adaptations show flexibility
	if len(chain.Strategy.Adaptations) > 0 {
		innovationScore += 0.2
	}

	// Hypothesis generation shows creativity
	for _, block := range chain.Blocks {
		if block.Type == ThinkingTypeHypothesis {
			innovationScore += 0.2
			break
		}
	}

	return math.Min(innovationScore, 1.0)
}

// PatternMatcher identifies patterns in reasoning
type PatternMatcher struct {
	patterns map[string]*Pattern
	mu       sync.RWMutex
}

// Pattern represents a reasoning pattern
type Pattern struct {
	ID          string
	Name        string
	Description string
	Signature   []ThinkingType
	MinBlocks   int
	Detector    func([]*ThinkingBlock) float64
}

// NewPatternMatcher creates a new pattern matcher
func NewPatternMatcher() *PatternMatcher {
	pm := &PatternMatcher{
		patterns: make(map[string]*Pattern),
	}

	// Register common patterns
	pm.registerBuiltInPatterns()

	return pm
}

// registerBuiltInPatterns adds standard reasoning patterns
func (pm *PatternMatcher) registerBuiltInPatterns() {
	patterns := []*Pattern{
		{
			ID:          "divide-conquer",
			Name:        "Divide and Conquer",
			Description: "Breaking complex problems into sub-problems",
			Signature:   []ThinkingType{ThinkingTypeAnalysis, ThinkingTypePlanning},
			MinBlocks:   3,
			Detector:    pm.detectDivideConquer,
		},
		{
			ID:          "hypothesis-test",
			Name:        "Hypothesis Testing",
			Description: "Forming and testing hypotheses",
			Signature:   []ThinkingType{ThinkingTypeHypothesis, ThinkingTypeVerification},
			MinBlocks:   2,
			Detector:    pm.detectHypothesisTesting,
		},
		{
			ID:          "iterative-refinement",
			Name:        "Iterative Refinement",
			Description: "Gradually improving through iterations",
			Signature:   []ThinkingType{ThinkingTypeAnalysis, ThinkingTypeErrorRecovery, ThinkingTypeAnalysis},
			MinBlocks:   3,
			Detector:    pm.detectIterativeRefinement,
		},
		{
			ID:          "tool-orchestration",
			Name:        "Tool Orchestration",
			Description: "Coordinating multiple tools effectively",
			Signature:   []ThinkingType{ThinkingTypeToolSelection},
			MinBlocks:   2,
			Detector:    pm.detectToolOrchestration,
		},
	}

	for _, pattern := range patterns {
		pm.patterns[pattern.ID] = pattern
	}
}

// FindPatterns identifies patterns in thinking blocks
func (pm *PatternMatcher) FindPatterns(blocks []*ThinkingBlock) []PatternMatch {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	matches := []PatternMatch{}

	for _, pattern := range pm.patterns {
		if len(blocks) < pattern.MinBlocks {
			continue
		}

		// Check for pattern
		confidence := pattern.Detector(blocks)
		if confidence > 0.5 {
			match := PatternMatch{
				PatternID:    pattern.ID,
				PatternName:  pattern.Name,
				Confidence:   confidence,
				Occurrences:  pm.findOccurrences(blocks, pattern),
				Implications: pm.getImplications(pattern),
			}
			matches = append(matches, match)
		}
	}

	return matches
}

// detectDivideConquer detects divide-and-conquer pattern
func (pm *PatternMatcher) detectDivideConquer(blocks []*ThinkingBlock) float64 {
	// Look for analysis followed by multiple sub-plans
	analysisFound := false
	subProblems := 0

	for _, block := range blocks {
		if block.Type == ThinkingTypeAnalysis {
			analysisFound = true
		}

		if analysisFound && block.Type == ThinkingTypePlanning {
			if block.StructuredData != nil && len(block.StructuredData.Steps) > 1 {
				subProblems++
			}
		}
	}

	if analysisFound && subProblems >= 2 {
		return 0.8
	}

	return 0
}

// detectHypothesisTesting detects hypothesis testing pattern
func (pm *PatternMatcher) detectHypothesisTesting(blocks []*ThinkingBlock) float64 {
	hypothesisCount := 0
	verificationCount := 0

	for i, block := range blocks {
		if block.Type == ThinkingTypeHypothesis {
			hypothesisCount++

			// Check if followed by verification
			if i+1 < len(blocks) && blocks[i+1].Type == ThinkingTypeVerification {
				verificationCount++
			}
		}
	}

	if hypothesisCount > 0 && verificationCount > 0 {
		return float64(verificationCount) / float64(hypothesisCount)
	}

	return 0
}

// detectIterativeRefinement detects iterative improvement
func (pm *PatternMatcher) detectIterativeRefinement(blocks []*ThinkingBlock) float64 {
	iterations := 0
	lastAnalysisIndex := -1

	for i, block := range blocks {
		if block.Type == ThinkingTypeAnalysis {
			if lastAnalysisIndex >= 0 && i-lastAnalysisIndex < 5 {
				// Found another analysis within 5 blocks
				iterations++
			}
			lastAnalysisIndex = i
		}
	}

	if iterations >= 2 {
		return math.Min(float64(iterations)*0.3, 1.0)
	}

	return 0
}

// detectToolOrchestration detects tool coordination
func (pm *PatternMatcher) detectToolOrchestration(blocks []*ThinkingBlock) float64 {
	toolBlocks := 0
	coordinatedTools := 0

	for _, block := range blocks {
		if block.Type == ThinkingTypeToolSelection && block.ToolContext != nil {
			toolBlocks++

			// Check if tools are coordinated (have expected outcomes)
			if block.ToolContext.ExpectedOutcome != "" {
				coordinatedTools++
			}
		}
	}

	if toolBlocks >= 2 && coordinatedTools > 0 {
		return float64(coordinatedTools) / float64(toolBlocks)
	}

	return 0
}

// findOccurrences locates pattern occurrences
func (pm *PatternMatcher) findOccurrences(blocks []*ThinkingBlock, pattern *Pattern) []PatternOccurrence {
	occurrences := []PatternOccurrence{}

	// Simple occurrence detection
	for i, block := range blocks {
		for _, sigType := range pattern.Signature {
			if block.Type == sigType {
				occurrences = append(occurrences, PatternOccurrence{
					BlockID:  block.ID,
					Location: fmt.Sprintf("%d", i),
					Strength: 0.7, // Simplified
				})
				break
			}
		}
	}

	return occurrences
}

// getImplications returns implications of a pattern
func (pm *PatternMatcher) getImplications(pattern *Pattern) []string {
	implications := map[string][]string{
		"divide-conquer": {
			"Problem decomposition is effective",
			"Consider parallel execution of sub-tasks",
			"Ensure sub-problem solutions integrate well",
		},
		"hypothesis-test": {
			"Scientific approach to problem-solving",
			"Validation is prioritized",
			"Consider multiple hypotheses for robustness",
		},
		"iterative-refinement": {
			"Solution quality improves with iterations",
			"Consider stopping criteria to avoid over-optimization",
			"Track improvement metrics across iterations",
		},
		"tool-orchestration": {
			"Multiple tools are being coordinated",
			"Consider tool dependencies and order",
			"Monitor for tool conflicts or redundancy",
		},
	}

	return implications[pattern.ID]
}

// InsightExtractor extracts insights from reasoning
type InsightExtractor struct {
	thresholds map[InsightType]float64
}

// NewInsightExtractor creates a new insight extractor
func NewInsightExtractor() *InsightExtractor {
	return &InsightExtractor{
		thresholds: map[InsightType]float64{
			InsightTypePattern:      0.7,
			InsightTypeAnomaly:      0.8,
			InsightTypeOptimization: 0.6,
			InsightTypeRisk:         0.7,
			InsightTypeOpportunity:  0.75,
		},
	}
}

// ExtractFromBlock extracts insights from a single block
func (ie *InsightExtractor) ExtractFromBlock(block *ThinkingBlock) []Insight {
	insights := []Insight{}

	// Risk insights from error contexts
	if block.ErrorContext != nil {
		insights = append(insights, Insight{
			Type:        InsightTypeRisk,
			Description: "Error pattern detected: " + block.ErrorContext.Description,
			Confidence:  0.8,
			Source:      block.ID,
			Actionable:  true,
			Actions:     []string{block.ErrorContext.Prevention},
		})
	}

	// Optimization insights from decision points
	for _, dp := range block.DecisionPoints {
		if len(dp.Alternatives) > 2 {
			insights = append(insights, Insight{
				Type:        InsightTypeOptimization,
				Description: "Multiple alternatives considered for: " + dp.Decision,
				Confidence:  dp.Confidence,
				Source:      block.ID,
				Actionable:  false,
			})
		}
	}

	// Pattern insights from structured data
	if block.StructuredData != nil && len(block.StructuredData.Steps) > 3 {
		insights = append(insights, Insight{
			Type:        InsightTypePattern,
			Description: fmt.Sprintf("Structured approach with %d steps", len(block.StructuredData.Steps)),
			Confidence:  0.7,
			Source:      block.ID,
			Actionable:  true,
			Actions:     []string{"Consider automating this structured process"},
		})
	}

	return insights
}

// ExtractFromChain extracts insights from the complete chain
func (ie *InsightExtractor) ExtractFromChain(chain *ReasoningChainEnhanced) []Insight {
	insights := []Insight{}

	// Performance insights
	if chain.Performance.BacktrackCount > 2 {
		insights = append(insights, Insight{
			Type:        InsightTypeRisk,
			Description: "High backtrack count indicates planning issues",
			Confidence:  0.85,
			Source:      "analysis",
			Actionable:  true,
			Actions: []string{
				"Improve upfront analysis",
				"Add constraint checking earlier",
			},
		})
	}

	// Quality insights
	if chain.Quality.Innovation > 0.8 {
		insights = append(insights, Insight{
			Type:        InsightTypeOpportunity,
			Description: "High innovation score - novel approach successful",
			Confidence:  chain.Quality.Innovation,
			Source:      "analysis",
			Actionable:  true,
			Actions: []string{
				"Document this approach for future use",
				"Consider applying to similar problems",
			},
		})
	}

	// Pattern insights
	for _, pattern := range chain.Patterns {
		if pattern.Confidence > ie.thresholds[InsightTypePattern] {
			insights = append(insights, Insight{
				Type:        InsightTypePattern,
				Description: "Strong " + pattern.PatternName + " pattern detected",
				Confidence:  pattern.Confidence,
				Source:      "pattern_analysis",
				Actionable:  false,
			})
		}
	}

	return insights
}

// ChainAnalyzer provides comprehensive analysis
type ChainAnalyzer struct {
	scorer    *QualityScorer
	matcher   *PatternMatcher
	extractor *InsightExtractor
	// metrics   *observability.Metrics // TODO: Update to use MetricsRegistry
}

// NewChainAnalyzer creates a new analyzer
func NewChainAnalyzer() *ChainAnalyzer {
	return &ChainAnalyzer{
		scorer:    NewQualityScorer(),
		matcher:   NewPatternMatcher(),
		extractor: NewInsightExtractor(),
		// metrics:   observability.DefaultMetrics, // TODO: Update to use MetricsRegistry
	}
}

// Analyze performs comprehensive analysis on a reasoning chain
func (ra *ChainAnalyzer) Analyze(ctx context.Context, chain *ReasoningChainEnhanced) (*AnalysisReport, error) {
	startTime := time.Now()
	defer func() {
		// TODO: Update to use MetricsRegistry
		// ra.metrics.RecordDuration("reasoning_analysis", time.Since(startTime))
		_ = startTime
	}()

	report := &AnalysisReport{
		ChainID:     chain.ID,
		AnalyzedAt:  time.Now(),
		Strengths:   []string{},
		Weaknesses:  []string{},
		Suggestions: []string{},
	}

	// Analyze quality
	quality, err := ra.scorer.Score(ctx, chain)
	if err != nil {
		return nil, err
	}
	report.Quality = quality

	// Find patterns
	patterns := ra.matcher.FindPatterns(chain.Blocks)
	report.Patterns = patterns

	// Extract insights
	insights := ra.extractor.ExtractFromChain(chain)
	report.Insights = insights

	// Generate recommendations
	report.Strengths = ra.identifyStrengths(chain, quality, patterns)
	report.Weaknesses = ra.identifyWeaknesses(chain, quality)
	report.Suggestions = ra.generateSuggestions(chain, quality, insights)

	return report, nil
}

// AnalysisReport contains the full analysis results
type AnalysisReport struct {
	ChainID     string         `json:"chain_id"`
	AnalyzedAt  time.Time      `json:"analyzed_at"`
	Quality     QualityMetrics `json:"quality"`
	Patterns    []PatternMatch `json:"patterns"`
	Insights    []Insight      `json:"insights"`
	Strengths   []string       `json:"strengths"`
	Weaknesses  []string       `json:"weaknesses"`
	Suggestions []string       `json:"suggestions"`
}

// identifyStrengths finds positive aspects
func (ra *ChainAnalyzer) identifyStrengths(chain *ReasoningChainEnhanced, quality QualityMetrics, patterns []PatternMatch) []string {
	strengths := []string{}

	if quality.Coherence > 0.8 {
		strengths = append(strengths, "Excellent logical flow and coherence")
	}

	if quality.Depth > 0.7 {
		strengths = append(strengths, "Deep and thorough analysis")
	}

	if len(patterns) > 0 {
		strengths = append(strengths, "Effective use of reasoning patterns")
	}

	if chain.Performance.BacktrackCount == 0 {
		strengths = append(strengths, "Efficient execution without backtracks")
	}

	return strengths
}

// identifyWeaknesses finds areas for improvement
func (ra *ChainAnalyzer) identifyWeaknesses(chain *ReasoningChainEnhanced, quality QualityMetrics) []string {
	weaknesses := []string{}

	if quality.Completeness < 0.6 {
		weaknesses = append(weaknesses, "Incomplete coverage of problem space")
	}

	if quality.Overall < 0.5 {
		weaknesses = append(weaknesses, "Low overall quality in conclusions")
	}

	if chain.Performance.BacktrackCount > 3 {
		weaknesses = append(weaknesses, "Excessive backtracks indicate poor planning")
	}

	return weaknesses
}

// generateSuggestions creates actionable recommendations
func (ra *ChainAnalyzer) generateSuggestions(chain *ReasoningChainEnhanced, quality QualityMetrics, insights []Insight) []string {
	suggestions := []string{}

	// Based on quality metrics
	if quality.Depth < 0.5 {
		suggestions = append(suggestions, "Increase analysis depth by exploring sub-problems")
	}

	if quality.Innovation < 0.3 {
		suggestions = append(suggestions, "Consider more alternative approaches")
	}

	// Based on insights
	for _, insight := range insights {
		if insight.Actionable {
			suggestions = append(suggestions, insight.Actions...)
		}
	}

	// Remove duplicates
	seen := make(map[string]bool)
	unique := []string{}
	for _, s := range suggestions {
		if !seen[s] {
			seen[s] = true
			unique = append(unique, s)
		}
	}

	return unique
}
