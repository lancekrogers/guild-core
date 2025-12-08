// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package core

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/guild-framework/guild-core/pkg/gerror"
)

// PatternAnalyzer analyzes reasoning chains for patterns
type PatternAnalyzer struct {
	sequenceDetector    *SequenceDetector
	structureAnalyzer   *StructureAnalyzer
	performanceAnalyzer *PerformanceAnalyzer
}

// NewPatternAnalyzer creates a new pattern analyzer
func NewPatternAnalyzer() *PatternAnalyzer {
	return &PatternAnalyzer{
		sequenceDetector:    NewSequenceDetector(),
		structureAnalyzer:   NewStructureAnalyzer(),
		performanceAnalyzer: NewPerformanceAnalyzer(),
	}
}

// DiscoverPatterns discovers patterns in a reasoning chain
func (pa *PatternAnalyzer) DiscoverPatterns(chain *ReasoningChainEnhanced) []PatternCandidate {
	candidates := make([]PatternCandidate, 0)

	// Detect sequence patterns
	sequences := pa.sequenceDetector.DetectSequences(chain.Blocks)
	for _, seq := range sequences {
		if seq.Frequency >= 2 && seq.Confidence > 0.7 {
			candidate := PatternCandidate{
				Name:        pa.generateSequenceName(seq),
				Description: pa.generateSequenceDescription(seq),
				Signature: PatternSignature{
					InputTypes:      seq.InputTypes,
					OutputTypes:     seq.OutputTypes,
					TypicalSequence: seq.Sequence,
					MinBlocks:       len(seq.Sequence),
					MaxBlocks:       len(seq.Sequence) * 2,
				},
				Features: pa.extractSequenceFeatures(seq),
			}
			candidates = append(candidates, candidate)
		}
	}

	// Detect structural patterns
	structures := pa.structureAnalyzer.AnalyzeStructure(chain)
	for _, struct_ := range structures {
		if struct_.IsPattern && struct_.Confidence > 0.75 {
			candidate := PatternCandidate{
				Name:        struct_.Name,
				Description: struct_.Description,
				Signature:   struct_.Signature,
				Features:    struct_.Features,
			}
			candidates = append(candidates, candidate)
		}
	}

	// Detect performance patterns
	perfPatterns := pa.performanceAnalyzer.AnalyzePerformance(chain)
	for _, pp := range perfPatterns {
		if pp.IsSignificant {
			candidate := PatternCandidate{
				Name:        pp.Name,
				Description: pp.Description,
				Signature:   pp.Signature,
				Features:    pp.Features,
			}
			candidates = append(candidates, candidate)
		}
	}

	return candidates
}

// AnalyzeFailure analyzes why a pattern failed
func (pa *PatternAnalyzer) AnalyzeFailure(pattern *LearnedPattern, chain *ReasoningChainEnhanced, feedback *ReasoningFeedback) []PatternIssue {
	issues := make([]PatternIssue, 0)

	// Check sequence violations
	if !pa.sequenceDetector.MatchesSignature(chain.Blocks, pattern.Signature) {
		issues = append(issues, PatternIssue{
			Type:         "sequence_mismatch",
			Description:  "Chain sequence doesn't match pattern signature",
			SuggestedFix: "Relax sequence requirements or add alternative paths",
		})
	}

	// Check constraint violations
	for _, constraint := range pattern.Signature.Constraints {
		if !pa.checkConstraintViolation(chain, constraint) {
			issues = append(issues, PatternIssue{
				Type:         "constraint_violation",
				Field:        constraint.Field,
				Description:  fmt.Sprintf("Constraint %s %s %v violated", constraint.Field, constraint.Operator, constraint.Value),
				SuggestedFix: "Adjust constraint threshold",
			})
		}
	}

	// Check performance issues
	if chain.Performance.BacktrackCount > 3 {
		issues = append(issues, PatternIssue{
			Type:         "performance_issue",
			Feature:      "backtrack_count",
			Description:  "Excessive backtracks indicate pattern inefficiency",
			SuggestedFix: "Add early validation steps",
		})
	}

	// Check quality issues
	if chain.Quality.Overall < pattern.Statistics.AverageScore*0.8 {
		issues = append(issues, PatternIssue{
			Type:         "quality_degradation",
			Description:  "Quality significantly below pattern average",
			SuggestedFix: "Review and update pattern template",
		})
	}

	return issues
}

// generateSequenceName creates a name for a sequence pattern
func (pa *PatternAnalyzer) generateSequenceName(seq *DetectedSequence) string {
	if len(seq.Sequence) == 0 {
		return "unknown_sequence"
	}

	// Create name from sequence types
	parts := make([]string, 0)
	for _, t := range seq.Sequence {
		parts = append(parts, string(t))
	}

	return strings.ToLower(strings.Join(parts, "_"))
}

// generateSequenceDescription creates a description
func (pa *PatternAnalyzer) generateSequenceDescription(seq *DetectedSequence) string {
	if len(seq.Sequence) == 0 {
		return "Unknown sequence pattern"
	}

	desc := "Pattern: "
	for i, t := range seq.Sequence {
		if i > 0 {
			desc += " → "
		}
		desc += pa.getTypeDescription(t)
	}

	return desc
}

// getTypeDescription returns a human-readable description
func (pa *PatternAnalyzer) getTypeDescription(t ThinkingType) string {
	descriptions := map[ThinkingType]string{
		ThinkingTypeAnalysis:       "Analyze",
		ThinkingTypePlanning:       "Plan",
		ThinkingTypeDecisionMaking: "Decide",
		ThinkingTypeToolSelection:  "Select Tools",
		ThinkingTypeVerification:   "Verify",
		ThinkingTypeHypothesis:     "Hypothesize",
		ThinkingTypeErrorRecovery:  "Recover",
	}

	if desc, ok := descriptions[t]; ok {
		return desc
	}
	return string(t)
}

// extractSequenceFeatures extracts features from a sequence
func (pa *PatternAnalyzer) extractSequenceFeatures(seq *DetectedSequence) []PatternFeature {
	features := []PatternFeature{
		{
			Name:       "sequence_length",
			Type:       FeatureTypeStructural,
			Weight:     0.8,
			Value:      len(seq.Sequence),
			Importance: 0.7,
		},
		{
			Name:       "frequency",
			Type:       FeatureTypeStructural,
			Weight:     0.9,
			Value:      seq.Frequency,
			Importance: 0.8,
		},
	}

	// Add type-specific features
	for _, t := range seq.Sequence {
		features = append(features, PatternFeature{
			Name:       fmt.Sprintf("contains_%s", t),
			Type:       FeatureTypeStructural,
			Weight:     0.6,
			Value:      true,
			Importance: 0.5,
		})
	}

	return features
}

// checkConstraintViolation checks if a constraint is violated
func (pa *PatternAnalyzer) checkConstraintViolation(chain *ReasoningChainEnhanced, constraint PatternConstraint) bool {
	// Simplified constraint checking
	switch constraint.Type {
	case ConstraintTypeComplexity:
		// Check complexity constraints
		return true
	case ConstraintTypeTemporal:
		// Check time constraints
		if constraint.Field == "max_duration" {
			if maxDur, ok := constraint.Value.(time.Duration); ok {
				return chain.EndTime.Sub(chain.StartTime) <= maxDur
			}
		}
	}

	return true
}

// SequenceDetector detects thinking sequences
type SequenceDetector struct {
	minLength int
	maxLength int
}

// NewSequenceDetector creates a new sequence detector
func NewSequenceDetector() *SequenceDetector {
	return &SequenceDetector{
		minLength: 2,
		maxLength: 7,
	}
}

// DetectedSequence represents a detected pattern sequence
type DetectedSequence struct {
	Sequence    []ThinkingType
	Frequency   int
	Confidence  float64
	InputTypes  []ThinkingType
	OutputTypes []ThinkingType
	Examples    []int // Indices where sequence occurs
}

// DetectSequences finds repeated sequences in blocks
func (sd *SequenceDetector) DetectSequences(blocks []*ThinkingBlock) []*DetectedSequence {
	sequences := make(map[string]*DetectedSequence)

	// Extract all possible sequences
	for length := sd.minLength; length <= sd.maxLength && length <= len(blocks); length++ {
		for i := 0; i <= len(blocks)-length; i++ {
			seq := sd.extractSequence(blocks[i : i+length])
			key := sd.sequenceKey(seq)

			if existing, ok := sequences[key]; ok {
				existing.Frequency++
				existing.Examples = append(existing.Examples, i)
			} else {
				sequences[key] = &DetectedSequence{
					Sequence:   seq,
					Frequency:  1,
					Confidence: sd.calculateConfidence(blocks[i : i+length]),
					Examples:   []int{i},
				}

				// Determine input/output types
				if i > 0 {
					sequences[key].InputTypes = []ThinkingType{blocks[i-1].Type}
				}
				if i+length < len(blocks) {
					sequences[key].OutputTypes = []ThinkingType{blocks[i+length].Type}
				}
			}
		}
	}

	// Convert to slice and filter
	result := make([]*DetectedSequence, 0)
	for _, seq := range sequences {
		if seq.Frequency >= 2 {
			result = append(result, seq)
		}
	}

	// Sort by frequency
	sort.Slice(result, func(i, j int) bool {
		return result[i].Frequency > result[j].Frequency
	})

	return result
}

// MatchesSignature checks if blocks match a pattern signature
func (sd *SequenceDetector) MatchesSignature(blocks []*ThinkingBlock, signature PatternSignature) bool {
	if len(blocks) < signature.MinBlocks || len(blocks) > signature.MaxBlocks {
		return false
	}

	// Check if sequence matches
	for i, expectedType := range signature.TypicalSequence {
		if i >= len(blocks) || blocks[i].Type != expectedType {
			return false
		}
	}

	return true
}

// extractSequence extracts thinking types from blocks
func (sd *SequenceDetector) extractSequence(blocks []*ThinkingBlock) []ThinkingType {
	seq := make([]ThinkingType, len(blocks))
	for i, block := range blocks {
		seq[i] = block.Type
	}
	return seq
}

// sequenceKey creates a key for a sequence
func (sd *SequenceDetector) sequenceKey(seq []ThinkingType) string {
	parts := make([]string, len(seq))
	for i, t := range seq {
		parts[i] = string(t)
	}
	return strings.Join(parts, "|")
}

// calculateConfidence calculates sequence confidence
func (sd *SequenceDetector) calculateConfidence(blocks []*ThinkingBlock) float64 {
	if len(blocks) == 0 {
		return 0
	}

	totalConfidence := 0.0
	for _, block := range blocks {
		totalConfidence += block.Confidence
	}

	return totalConfidence / float64(len(blocks))
}

// StructureAnalyzer analyzes structural patterns
type StructureAnalyzer struct {
	patterns []StructuralPattern
}

// StructuralPattern represents a structural pattern type
type StructuralPattern struct {
	Name        string
	Description string
	Detector    func(*ReasoningChainEnhanced) bool
	Extractor   func(*ReasoningChainEnhanced) (PatternSignature, []PatternFeature)
}

// NewStructureAnalyzer creates a new structure analyzer
func NewStructureAnalyzer() *StructureAnalyzer {
	sa := &StructureAnalyzer{
		patterns: make([]StructuralPattern, 0),
	}

	// Register structural patterns
	sa.patterns = append(sa.patterns, StructuralPattern{
		Name:        "branching_exploration",
		Description: "Explores multiple alternatives before deciding",
		Detector:    sa.detectBranchingExploration,
		Extractor:   sa.extractBranchingFeatures,
	})

	sa.patterns = append(sa.patterns, StructuralPattern{
		Name:        "iterative_refinement",
		Description: "Repeatedly refines approach based on feedback",
		Detector:    sa.detectIterativeRefinement,
		Extractor:   sa.extractIterativeFeatures,
	})

	sa.patterns = append(sa.patterns, StructuralPattern{
		Name:        "hierarchical_decomposition",
		Description: "Breaks down problems into sub-problems",
		Detector:    sa.detectHierarchicalDecomposition,
		Extractor:   sa.extractHierarchicalFeatures,
	})

	return sa
}

// StructureAnalysisResult represents analysis results
type StructureAnalysisResult struct {
	Name        string
	Description string
	IsPattern   bool
	Confidence  float64
	Signature   PatternSignature
	Features    []PatternFeature
}

// AnalyzeStructure analyzes chain structure
func (sa *StructureAnalyzer) AnalyzeStructure(chain *ReasoningChainEnhanced) []StructureAnalysisResult {
	results := make([]StructureAnalysisResult, 0)

	for _, pattern := range sa.patterns {
		if pattern.Detector(chain) {
			signature, features := pattern.Extractor(chain)

			result := StructureAnalysisResult{
				Name:        pattern.Name,
				Description: pattern.Description,
				IsPattern:   true,
				Confidence:  sa.calculateStructuralConfidence(chain, features),
				Signature:   signature,
				Features:    features,
			}

			results = append(results, result)
		}
	}

	return results
}

// detectBranchingExploration detects branching pattern
func (sa *StructureAnalyzer) detectBranchingExploration(chain *ReasoningChainEnhanced) bool {
	// Count decision points with multiple alternatives
	branchCount := 0
	for _, block := range chain.Blocks {
		for _, dp := range block.DecisionPoints {
			if len(dp.Alternatives) > 2 {
				branchCount++
			}
		}
	}

	return branchCount >= 2
}

// extractBranchingFeatures extracts features for branching
func (sa *StructureAnalyzer) extractBranchingFeatures(chain *ReasoningChainEnhanced) (PatternSignature, []PatternFeature) {
	signature := PatternSignature{
		InputTypes:      []ThinkingType{ThinkingTypeAnalysis},
		OutputTypes:     []ThinkingType{ThinkingTypeDecisionMaking},
		TypicalSequence: []ThinkingType{ThinkingTypeAnalysis, ThinkingTypeDecisionMaking, ThinkingTypeVerification},
		MinBlocks:       3,
		MaxBlocks:       10,
	}

	features := []PatternFeature{
		{
			Name:       "branch_count",
			Type:       FeatureTypeStructural,
			Weight:     0.9,
			Value:      sa.countBranches(chain),
			Importance: 0.8,
		},
		{
			Name:       "avg_alternatives",
			Type:       FeatureTypeStructural,
			Weight:     0.7,
			Value:      sa.avgAlternatives(chain),
			Importance: 0.6,
		},
	}

	return signature, features
}

// detectIterativeRefinement detects iterative pattern
func (sa *StructureAnalyzer) detectIterativeRefinement(chain *ReasoningChainEnhanced) bool {
	// Look for repeated analysis-error-analysis cycles
	cycleCount := 0
	lastWasError := false

	for _, block := range chain.Blocks {
		if block.Type == ThinkingTypeErrorRecovery {
			lastWasError = true
		} else if block.Type == ThinkingTypeAnalysis && lastWasError {
			cycleCount++
			lastWasError = false
		}
	}

	return cycleCount >= 2
}

// extractIterativeFeatures extracts iterative features
func (sa *StructureAnalyzer) extractIterativeFeatures(chain *ReasoningChainEnhanced) (PatternSignature, []PatternFeature) {
	signature := PatternSignature{
		InputTypes:      []ThinkingType{ThinkingTypeAnalysis},
		OutputTypes:     []ThinkingType{ThinkingTypeVerification},
		TypicalSequence: []ThinkingType{ThinkingTypeAnalysis, ThinkingTypeErrorRecovery, ThinkingTypeAnalysis},
		MinBlocks:       4,
		MaxBlocks:       15,
	}

	features := []PatternFeature{
		{
			Name:       "iteration_count",
			Type:       FeatureTypeStructural,
			Weight:     0.8,
			Value:      chain.Performance.IterationCount,
			Importance: 0.7,
		},
		{
			Name:       "refinement_success",
			Type:       FeatureTypePerformance,
			Weight:     0.9,
			Value:      chain.Quality.Overall > 0.7,
			Importance: 0.8,
		},
	}

	return signature, features
}

// detectHierarchicalDecomposition detects hierarchical pattern
func (sa *StructureAnalyzer) detectHierarchicalDecomposition(chain *ReasoningChainEnhanced) bool {
	// Check for nested thinking blocks
	maxDepth := 0
	for _, block := range chain.Blocks {
		if block.Depth > maxDepth {
			maxDepth = block.Depth
		}
	}

	return maxDepth >= 2
}

// extractHierarchicalFeatures extracts hierarchical features
func (sa *StructureAnalyzer) extractHierarchicalFeatures(chain *ReasoningChainEnhanced) (PatternSignature, []PatternFeature) {
	signature := PatternSignature{
		InputTypes:      []ThinkingType{ThinkingTypeAnalysis},
		OutputTypes:     []ThinkingType{ThinkingTypePlanning},
		TypicalSequence: []ThinkingType{ThinkingTypeAnalysis, ThinkingTypePlanning},
		MinBlocks:       4,
		MaxBlocks:       20,
	}

	features := []PatternFeature{
		{
			Name:       "max_depth",
			Type:       FeatureTypeStructural,
			Weight:     0.8,
			Value:      sa.getMaxDepth(chain),
			Importance: 0.7,
		},
		{
			Name:       "decomposition_factor",
			Type:       FeatureTypeStructural,
			Weight:     0.7,
			Value:      sa.getDecompositionFactor(chain),
			Importance: 0.6,
		},
	}

	return signature, features
}

// Helper methods for structure analyzer

func (sa *StructureAnalyzer) calculateStructuralConfidence(chain *ReasoningChainEnhanced, features []PatternFeature) float64 {
	// Base confidence on chain quality and feature strength
	confidence := chain.Quality.Overall

	// Adjust based on features
	for _, feature := range features {
		confidence *= (0.5 + feature.Weight*0.5)
	}

	return math.Min(confidence, 1.0)
}

func (sa *StructureAnalyzer) countBranches(chain *ReasoningChainEnhanced) int {
	count := 0
	for _, block := range chain.Blocks {
		count += len(block.DecisionPoints)
	}
	return count
}

func (sa *StructureAnalyzer) avgAlternatives(chain *ReasoningChainEnhanced) float64 {
	total := 0
	count := 0

	for _, block := range chain.Blocks {
		for _, dp := range block.DecisionPoints {
			total += len(dp.Alternatives)
			count++
		}
	}

	if count == 0 {
		return 0
	}

	return float64(total) / float64(count)
}

func (sa *StructureAnalyzer) getMaxDepth(chain *ReasoningChainEnhanced) int {
	maxDepth := 0
	for _, block := range chain.Blocks {
		if block.Depth > maxDepth {
			maxDepth = block.Depth
		}
	}
	return maxDepth
}

func (sa *StructureAnalyzer) getDecompositionFactor(chain *ReasoningChainEnhanced) float64 {
	// Ratio of child blocks to parent blocks
	parents := 0
	children := 0

	for _, block := range chain.Blocks {
		if block.ParentID == nil {
			parents++
		} else {
			children++
		}
	}

	if parents == 0 {
		return 0
	}

	return float64(children) / float64(parents)
}

// PerformanceAnalyzer analyzes performance patterns
type PerformanceAnalyzer struct {
	thresholds PerformanceThresholds
}

// PerformanceThresholds defines performance criteria
type PerformanceThresholds struct {
	FastThinking       time.Duration
	SlowThinking       time.Duration
	HighBacktracks     int
	LowTokensPerBlock  int
	HighTokensPerBlock int
}

// NewPerformanceAnalyzer creates a new performance analyzer
func NewPerformanceAnalyzer() *PerformanceAnalyzer {
	return &PerformanceAnalyzer{
		thresholds: PerformanceThresholds{
			FastThinking:       2 * time.Second,
			SlowThinking:       10 * time.Second,
			HighBacktracks:     3,
			LowTokensPerBlock:  50,
			HighTokensPerBlock: 500,
		},
	}
}

// PerformancePattern represents a performance pattern
type PerformancePattern struct {
	Name          string
	Description   string
	IsSignificant bool
	Signature     PatternSignature
	Features      []PatternFeature
}

// AnalyzePerformance analyzes performance patterns
func (pa *PerformanceAnalyzer) AnalyzePerformance(chain *ReasoningChainEnhanced) []PerformancePattern {
	patterns := make([]PerformancePattern, 0)

	// Check for fast thinking pattern
	if chain.Performance.ThinkingTime < pa.thresholds.FastThinking {
		patterns = append(patterns, PerformancePattern{
			Name:          "rapid_reasoning",
			Description:   "Exceptionally fast reasoning process",
			IsSignificant: true,
			Signature:     pa.createRapidSignature(chain),
			Features:      pa.extractRapidFeatures(chain),
		})
	}

	// Check for efficient token usage
	avgTokens := chain.TotalTokens / len(chain.Blocks)
	if avgTokens < pa.thresholds.LowTokensPerBlock {
		patterns = append(patterns, PerformancePattern{
			Name:          "token_efficient",
			Description:   "Highly efficient token usage",
			IsSignificant: true,
			Signature:     pa.createEfficientSignature(chain),
			Features:      pa.extractEfficientFeatures(chain),
		})
	}

	// Check for low backtrack pattern
	if chain.Performance.BacktrackCount == 0 && len(chain.Blocks) > 5 {
		patterns = append(patterns, PerformancePattern{
			Name:          "linear_progression",
			Description:   "Straight-through reasoning without backtracks",
			IsSignificant: true,
			Signature:     pa.createLinearSignature(chain),
			Features:      pa.extractLinearFeatures(chain),
		})
	}

	return patterns
}

// createRapidSignature creates signature for rapid reasoning
func (pa *PerformanceAnalyzer) createRapidSignature(chain *ReasoningChainEnhanced) PatternSignature {
	// Extract types from chain
	types := make([]ThinkingType, 0)
	for _, block := range chain.Blocks {
		types = append(types, block.Type)
	}

	return PatternSignature{
		InputTypes:      types[:1],
		OutputTypes:     types[len(types)-1:],
		TypicalSequence: types,
		MinBlocks:       len(chain.Blocks) - 1,
		MaxBlocks:       len(chain.Blocks) + 2,
		Constraints: []PatternConstraint{
			{
				Type:     ConstraintTypeTemporal,
				Field:    "max_thinking_time",
				Operator: "<",
				Value:    pa.thresholds.FastThinking,
			},
		},
	}
}

// extractRapidFeatures extracts features for rapid reasoning
func (pa *PerformanceAnalyzer) extractRapidFeatures(chain *ReasoningChainEnhanced) []PatternFeature {
	return []PatternFeature{
		{
			Name:       "thinking_speed",
			Type:       FeatureTypePerformance,
			Weight:     0.9,
			Value:      chain.Performance.ThinkingTime.Seconds(),
			Importance: 0.9,
		},
		{
			Name:       "decision_speed",
			Type:       FeatureTypePerformance,
			Weight:     0.8,
			Value:      chain.Performance.DecisionSpeed.Seconds(),
			Importance: 0.7,
		},
		{
			Name:       "tokens_per_second",
			Type:       FeatureTypePerformance,
			Weight:     0.7,
			Value:      chain.Performance.TokensPerSecond,
			Importance: 0.6,
		},
	}
}

// createEfficientSignature creates signature for token efficiency
func (pa *PerformanceAnalyzer) createEfficientSignature(chain *ReasoningChainEnhanced) PatternSignature {
	types := make([]ThinkingType, 0)
	for _, block := range chain.Blocks {
		types = append(types, block.Type)
	}

	return PatternSignature{
		InputTypes:      types[:1],
		OutputTypes:     types[len(types)-1:],
		TypicalSequence: types,
		MinBlocks:       3,
		MaxBlocks:       15,
		Constraints: []PatternConstraint{
			{
				Type:     ConstraintTypeResource,
				Field:    "avg_tokens_per_block",
				Operator: "<",
				Value:    pa.thresholds.LowTokensPerBlock,
			},
		},
	}
}

// extractEfficientFeatures extracts features for efficiency
func (pa *PerformanceAnalyzer) extractEfficientFeatures(chain *ReasoningChainEnhanced) []PatternFeature {
	avgTokens := float64(chain.TotalTokens) / float64(len(chain.Blocks))

	return []PatternFeature{
		{
			Name:       "avg_tokens_per_block",
			Type:       FeatureTypePerformance,
			Weight:     0.9,
			Value:      avgTokens,
			Importance: 0.8,
		},
		{
			Name:       "total_tokens",
			Type:       FeatureTypePerformance,
			Weight:     0.7,
			Value:      chain.TotalTokens,
			Importance: 0.6,
		},
		{
			Name:       "conciseness_score",
			Type:       FeatureTypePerformance,
			Weight:     0.8,
			Value:      1.0 / (avgTokens / 100.0),
			Importance: 0.7,
		},
	}
}

// createLinearSignature creates signature for linear progression
func (pa *PerformanceAnalyzer) createLinearSignature(chain *ReasoningChainEnhanced) PatternSignature {
	types := make([]ThinkingType, 0)
	for _, block := range chain.Blocks {
		types = append(types, block.Type)
	}

	return PatternSignature{
		InputTypes:      types[:1],
		OutputTypes:     types[len(types)-1:],
		TypicalSequence: types,
		MinBlocks:       5,
		MaxBlocks:       20,
		Constraints: []PatternConstraint{
			{
				Type:     ConstraintTypeResource,
				Field:    "backtrack_count",
				Operator: "=",
				Value:    0,
			},
		},
	}
}

// extractLinearFeatures extracts features for linear progression
func (pa *PerformanceAnalyzer) extractLinearFeatures(chain *ReasoningChainEnhanced) []PatternFeature {
	return []PatternFeature{
		{
			Name:       "linear_flow",
			Type:       FeatureTypeStructural,
			Weight:     1.0,
			Value:      true,
			Importance: 0.9,
		},
		{
			Name:       "coherence",
			Type:       FeatureTypePerformance,
			Weight:     0.8,
			Value:      chain.Quality.Coherence,
			Importance: 0.8,
		},
		{
			Name:       "step_count",
			Type:       FeatureTypeStructural,
			Weight:     0.6,
			Value:      len(chain.Blocks),
			Importance: 0.5,
		},
	}
}

// FeatureExtractor extracts features from various sources
type FeatureExtractor struct {
	extractors map[string]func(interface{}) []PatternFeature
}

// NewFeatureExtractor creates a new feature extractor
func NewFeatureExtractor() *FeatureExtractor {
	fe := &FeatureExtractor{
		extractors: make(map[string]func(interface{}) []PatternFeature),
	}

	// Register extractors
	fe.extractors["thinking_block"] = fe.extractBlockFeatures
	fe.extractors["reasoning_chain"] = fe.extractChainFeatures
	fe.extractors["context"] = fe.extractContextFeatures

	return fe
}

// ExtractPatternFeatures extracts features for a pattern
func (fe *FeatureExtractor) ExtractPatternFeatures(candidate PatternCandidate, chain *ReasoningChainEnhanced) []PatternFeature {
	features := candidate.Features

	// Add chain-based features
	chainFeatures := fe.ExtractChainFeatures(chain)
	features = append(features, chainFeatures...)

	// Normalize and deduplicate
	features = fe.normalizeFeatures(features)

	return features
}

// ExtractChainFeatures extracts features from a chain
func (fe *FeatureExtractor) ExtractChainFeatures(chain *ReasoningChainEnhanced) []PatternFeature {
	if extractor, ok := fe.extractors["reasoning_chain"]; ok {
		return extractor(chain)
	}
	return []PatternFeature{}
}

// ExtractContextFeatures extracts features from context
func (fe *FeatureExtractor) ExtractContextFeatures(context PatternContext) []ContextFeature {
	features := []ContextFeature{
		{
			Name:  "task_complexity",
			Value: context.Complexity,
			Type:  FeatureTypeContextual,
		},
		{
			Name:  "current_blocks",
			Value: context.CurrentBlocks,
			Type:  FeatureTypeStructural,
		},
	}

	// Extract from metadata
	for k, v := range context.Metadata {
		features = append(features, ContextFeature{
			Name:  k,
			Value: v,
			Type:  FeatureTypeContextual,
		})
	}

	return features
}

// extractBlockFeatures extracts features from a thinking block
func (fe *FeatureExtractor) extractBlockFeatures(data interface{}) []PatternFeature {
	block, ok := data.(*ThinkingBlock)
	if !ok {
		return []PatternFeature{}
	}

	features := []PatternFeature{
		{
			Name:       "block_type",
			Type:       FeatureTypeStructural,
			Weight:     0.8,
			Value:      string(block.Type),
			Importance: 0.7,
		},
		{
			Name:       "confidence",
			Type:       FeatureTypeSemantic,
			Weight:     0.9,
			Value:      block.Confidence,
			Importance: 0.8,
		},
		{
			Name:       "token_count",
			Type:       FeatureTypePerformance,
			Weight:     0.6,
			Value:      block.TokenCount,
			Importance: 0.5,
		},
	}

	// Add decision features
	if len(block.DecisionPoints) > 0 {
		features = append(features, PatternFeature{
			Name:       "has_decisions",
			Type:       FeatureTypeStructural,
			Weight:     0.7,
			Value:      true,
			Importance: 0.6,
		})
	}

	return features
}

// extractChainFeatures extracts features from a reasoning chain
func (fe *FeatureExtractor) extractChainFeatures(data interface{}) []PatternFeature {
	chain, ok := data.(*ReasoningChainEnhanced)
	if !ok {
		return []PatternFeature{}
	}

	features := []PatternFeature{
		{
			Name:       "chain_length",
			Type:       FeatureTypeStructural,
			Weight:     0.7,
			Value:      len(chain.Blocks),
			Importance: 0.6,
		},
		{
			Name:       "quality_score",
			Type:       FeatureTypePerformance,
			Weight:     0.9,
			Value:      chain.Quality.Overall,
			Importance: 0.9,
		},
		{
			Name:       "thinking_time",
			Type:       FeatureTypeTemporal,
			Weight:     0.6,
			Value:      chain.Performance.ThinkingTime.Seconds(),
			Importance: 0.5,
		},
		{
			Name:       "backtrack_count",
			Type:       FeatureTypePerformance,
			Weight:     0.8,
			Value:      chain.Performance.BacktrackCount,
			Importance: 0.7,
		},
	}

	// Add insight features
	if len(chain.Insights) > 0 {
		features = append(features, PatternFeature{
			Name:       "insight_count",
			Type:       FeatureTypeSemantic,
			Weight:     0.7,
			Value:      len(chain.Insights),
			Importance: 0.6,
		})
	}

	return features
}

// extractContextFeatures extracts features from context
func (fe *FeatureExtractor) extractContextFeatures(data interface{}) []PatternFeature {
	context, ok := data.(map[string]interface{})
	if !ok {
		return []PatternFeature{}
	}

	features := []PatternFeature{}

	// Extract known context features
	if complexity, ok := context["complexity"].(float64); ok {
		features = append(features, PatternFeature{
			Name:       "context_complexity",
			Type:       FeatureTypeContextual,
			Weight:     0.7,
			Value:      complexity,
			Importance: 0.6,
		})
	}

	return features
}

// normalizeFeatures normalizes and deduplicates features
func (fe *FeatureExtractor) normalizeFeatures(features []PatternFeature) []PatternFeature {
	// Deduplicate by name
	seen := make(map[string]bool)
	normalized := make([]PatternFeature, 0)

	for _, feature := range features {
		if !seen[feature.Name] {
			// Normalize weights
			if feature.Weight > 1.0 {
				feature.Weight = 1.0
			} else if feature.Weight < 0.0 {
				feature.Weight = 0.0
			}

			normalized = append(normalized, feature)
			seen[feature.Name] = true
		}
	}

	// Sort by importance
	sort.Slice(normalized, func(i, j int) bool {
		return normalized[i].Importance > normalized[j].Importance
	})

	return normalized
}

// PatternApplicator applies patterns to enhance reasoning
type PatternApplicator struct {
	templateEngine *TemplateEngine
	validator      *PatternValidator
}

// NewPatternApplicator creates a new pattern applicator
func NewPatternApplicator() *PatternApplicator {
	return &PatternApplicator{
		templateEngine: NewTemplateEngine(),
		validator:      NewPatternValidator(),
	}
}

// Apply applies a pattern to generate enhanced output
func (pa *PatternApplicator) Apply(ctx context.Context, pattern *LearnedPattern, input *PatternInput) (*PatternOutput, error) {
	// Validate input
	if err := pa.validator.ValidateInput(pattern, input); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeValidation, "invalid pattern input").
			WithComponent("pattern_applicator")
	}

	// Generate enhanced prompt
	enhancedPrompt, err := pa.templateEngine.GeneratePrompt(pattern.Template, input)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to generate prompt").
			WithComponent("pattern_applicator")
	}

	// Generate guided steps
	guidedSteps := pa.generateGuidedSteps(pattern, input)

	// Calculate confidence
	confidence := pa.calculateApplicationConfidence(pattern, input)

	output := &PatternOutput{
		EnhancedPrompt: enhancedPrompt,
		GuidedSteps:    guidedSteps,
		Confidence:     confidence,
		Explanation:    pa.generateExplanation(pattern, input),
	}

	return output, nil
}

// generateGuidedSteps creates step-by-step guidance
func (pa *PatternApplicator) generateGuidedSteps(pattern *LearnedPattern, input *PatternInput) []GuidedStep {
	steps := make([]GuidedStep, 0)

	for _, templateStep := range pattern.Template.Steps {
		step := GuidedStep{
			Order:       templateStep.Order,
			Instruction: pa.customizeInstruction(templateStep, input),
			Hints:       pa.selectRelevantHints(pattern.Template.Hints, templateStep),
			Examples:    pa.selectRelevantExamples(pattern.Examples, templateStep),
		}

		steps = append(steps, step)
	}

	return steps
}

// customizeInstruction customizes instruction for context
func (pa *PatternApplicator) customizeInstruction(step TemplateStep, input *PatternInput) string {
	instruction := step.Description

	// Replace placeholders
	instruction = strings.ReplaceAll(instruction, "{task}", input.Task)

	// Add context-specific details
	if len(input.Constraints) > 0 {
		instruction += fmt.Sprintf(" Consider constraints: %s", strings.Join(input.Constraints, ", "))
	}

	return instruction
}

// selectRelevantHints selects hints for a step
func (pa *PatternApplicator) selectRelevantHints(hints []string, step TemplateStep) []string {
	relevant := make([]string, 0)

	for _, hint := range hints {
		// Simple relevance check
		if strings.Contains(strings.ToLower(hint), strings.ToLower(string(step.ExpectedType))) {
			relevant = append(relevant, hint)
		}
	}

	// Limit to 3 hints
	if len(relevant) > 3 {
		relevant = relevant[:3]
	}

	return relevant
}

// selectRelevantExamples selects examples for a step
func (pa *PatternApplicator) selectRelevantExamples(examples []PatternExample, step TemplateStep) []string {
	relevant := make([]string, 0)

	for _, example := range examples {
		if example.Success {
			// Extract relevant part
			for _, block := range example.Input {
				if block.Type == step.ExpectedType {
					excerpt := pa.extractExcerpt(block.Content, 100)
					relevant = append(relevant, excerpt)
					break
				}
			}
		}

		// Limit to 2 examples
		if len(relevant) >= 2 {
			break
		}
	}

	return relevant
}

// extractExcerpt extracts a brief excerpt
func (pa *PatternApplicator) extractExcerpt(content string, maxLength int) string {
	if len(content) <= maxLength {
		return content
	}

	return content[:maxLength] + "..."
}

// calculateApplicationConfidence calculates confidence
func (pa *PatternApplicator) calculateApplicationConfidence(pattern *LearnedPattern, input *PatternInput) float64 {
	// Base confidence on pattern statistics
	confidence := pattern.Statistics.SuccessRate

	// Adjust based on recent performance
	if pattern.Statistics.Trend == TrendImproving {
		confidence *= 1.1
	} else if pattern.Statistics.Trend == TrendDeclining {
		confidence *= 0.9
	}

	// Cap at 0.95
	return math.Min(confidence, 0.95)
}

// generateExplanation generates explanation
func (pa *PatternApplicator) generateExplanation(pattern *LearnedPattern, input *PatternInput) string {
	explanation := fmt.Sprintf("Applied '%s' pattern based on %d successful examples. ",
		pattern.Name, pattern.Statistics.SuccessfulUsages)

	explanation += fmt.Sprintf("This pattern has %.0f%% success rate and is %s. ",
		pattern.Statistics.SuccessRate*100, pattern.Statistics.Trend)

	if len(pattern.Template.Steps) > 0 {
		explanation += fmt.Sprintf("Follow %d guided steps for best results.",
			len(pattern.Template.Steps))
	}

	return explanation
}

// TemplateEngine generates prompts from templates
type TemplateEngine struct {
	parser *TemplateParser
}

// NewTemplateEngine creates a new template engine
func NewTemplateEngine() *TemplateEngine {
	return &TemplateEngine{
		parser: NewTemplateParser(),
	}
}

// GeneratePrompt generates a prompt from template
func (te *TemplateEngine) GeneratePrompt(template ReasoningTemplate, input *PatternInput) (string, error) {
	// Start with system prompt
	prompt := template.SystemPrompt + "\n\n"

	// Add instruction
	instruction := te.parser.ParseTemplate(template.InstructionTemplate, input)
	prompt += instruction + "\n\n"

	// Add steps
	for _, step := range template.Steps {
		prompt += fmt.Sprintf("Step %d: %s\n", step.Order, step.Description)
	}

	// Add required elements
	if len(template.RequiredElements) > 0 {
		prompt += "\nRequired elements:\n"
		for _, elem := range template.RequiredElements {
			prompt += fmt.Sprintf("- %s\n", elem)
		}
	}

	// Add warnings about forbidden elements
	if len(template.ForbiddenElements) > 0 {
		prompt += "\nAvoid:\n"
		for _, elem := range template.ForbiddenElements {
			prompt += fmt.Sprintf("- %s\n", elem)
		}
	}

	return prompt, nil
}

// TemplateParser parses templates
type TemplateParser struct{}

// NewTemplateParser creates a new template parser
func NewTemplateParser() *TemplateParser {
	return &TemplateParser{}
}

// ParseTemplate parses a template with input
func (tp *TemplateParser) ParseTemplate(template string, input *PatternInput) string {
	result := template

	// Replace basic placeholders
	result = strings.ReplaceAll(result, "{task}", input.Task)

	// Replace context placeholders
	for k, v := range input.Context {
		placeholder := fmt.Sprintf("{%s}", k)
		result = strings.ReplaceAll(result, placeholder, fmt.Sprintf("%v", v))
	}

	return result
}

// PatternValidator validates pattern application
type PatternValidator struct{}

// NewPatternValidator creates a new pattern validator
func NewPatternValidator() *PatternValidator {
	return &PatternValidator{}
}

// ValidateInput validates input for a pattern
func (pv *PatternValidator) ValidateInput(pattern *LearnedPattern, input *PatternInput) error {
	// Check required context
	for _, constraint := range pattern.Signature.Constraints {
		if constraint.Type == ConstraintTypeContext {
			if _, ok := input.Context[constraint.Field]; !ok {
				return fmt.Errorf("missing required context field: %s", constraint.Field)
			}
		}
	}

	// Validate task
	if input.Task == "" {
		return fmt.Errorf("task cannot be empty")
	}

	return nil
}

// PatternEvaluator evaluates pattern performance
type PatternEvaluator struct {
	scoreCalculator *ScoreCalculator
}

// NewPatternEvaluator creates a new pattern evaluator
func NewPatternEvaluator() *PatternEvaluator {
	return &PatternEvaluator{
		scoreCalculator: NewScoreCalculator(),
	}
}

// EvaluatePattern evaluates overall pattern performance
func (pe *PatternEvaluator) EvaluatePattern(pattern *LearnedPattern) PatternStatistics {
	// Calculate fresh statistics from examples
	stats := PatternStatistics{
		TotalUsages: len(pattern.Examples),
	}

	scores := make([]float64, 0)

	for _, example := range pattern.Examples {
		if example.Success {
			stats.SuccessfulUsages++
		} else {
			stats.FailedUsages++
		}

		// Calculate score for example
		score := pe.scoreCalculator.CalculateExampleScore(example)
		scores = append(scores, score)
	}

	// Calculate success rate
	if stats.TotalUsages > 0 {
		stats.SuccessRate = float64(stats.SuccessfulUsages) / float64(stats.TotalUsages)
	}

	// Calculate average score
	if len(scores) > 0 {
		sum := 0.0
		for _, s := range scores {
			sum += s
		}
		stats.AverageScore = sum / float64(len(scores))
	}

	// Get recent scores
	if len(scores) > 10 {
		stats.RecentScores = scores[len(scores)-10:]
	} else {
		stats.RecentScores = scores
	}

	// Determine trend
	stats.Trend = pe.determineTrend(stats.RecentScores)
	stats.LastEvaluation = time.Now()

	return stats
}

// EvaluateApplication evaluates a single application
func (pe *PatternEvaluator) EvaluateApplication(pattern *LearnedPattern, input *PatternInput, output *PatternOutput) float64 {
	// Base score on output confidence
	score := output.Confidence

	// Adjust based on pattern match
	if pe.outputMatchesPattern(output, pattern) {
		score *= 1.1
	}

	// Cap at 1.0
	return math.Min(score, 1.0)
}

// determineTrend determines performance trend
func (pe *PatternEvaluator) determineTrend(scores []float64) TrendDirection {
	if len(scores) < 3 {
		return TrendStable
	}

	// Simple linear regression
	sumX := 0.0
	sumY := 0.0
	sumXY := 0.0
	sumX2 := 0.0

	for i, score := range scores {
		x := float64(i)
		sumX += x
		sumY += score
		sumXY += x * score
		sumX2 += x * x
	}

	n := float64(len(scores))
	slope := (n*sumXY - sumX*sumY) / (n*sumX2 - sumX*sumX)

	if slope > 0.05 {
		return TrendImproving
	} else if slope < -0.05 {
		return TrendDeclining
	}

	return TrendStable
}

// outputMatchesPattern checks if output matches pattern
func (pe *PatternEvaluator) outputMatchesPattern(output *PatternOutput, pattern *LearnedPattern) bool {
	// Check if guided steps match template
	if len(output.GuidedSteps) != len(pattern.Template.Steps) {
		return false
	}

	// Simple validation
	return output.Confidence > 0.6
}

// ScoreCalculator calculates various scores
type ScoreCalculator struct{}

// NewScoreCalculator creates a new score calculator
func NewScoreCalculator() *ScoreCalculator {
	return &ScoreCalculator{}
}

// CalculateExampleScore calculates score for an example
func (sc *ScoreCalculator) CalculateExampleScore(example PatternExample) float64 {
	score := 0.5 // Base score

	if example.Success {
		score = 0.8
	}

	// Adjust based on performance
	if example.Performance.BacktrackCount == 0 {
		score += 0.1
	}

	if example.Performance.ThinkingTime < 5*time.Second {
		score += 0.1
	}

	return math.Min(score, 1.0)
}
