// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package validation

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/lancekrogers/guild/pkg/corpus/extraction"
	"github.com/lancekrogers/guild/pkg/corpus/graph"
	"github.com/lancekrogers/guild/pkg/gerror"
)

// KnowledgeValidator provides quality assurance for extracted knowledge
type KnowledgeValidator struct {
	rules           []ValidationRule
	conflictDetector *ConflictDetector
	factChecker     *FactChecker
	knowledgeGraph  *graph.KnowledgeGraph
}

// NewKnowledgeValidator creates a new knowledge validator with default rules
func NewKnowledgeValidator(ctx context.Context, knowledgeGraph *graph.KnowledgeGraph) (*KnowledgeValidator, error) {
	if ctx.Err() != nil {
		return nil, gerror.Wrap(ctx.Err(), gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("corpus.validation").
			WithOperation("NewKnowledgeValidator")
	}

	validator := &KnowledgeValidator{
		knowledgeGraph: knowledgeGraph,
	}

	// Initialize validation rules
	validator.rules = []ValidationRule{
		&CompletenessRule{},
		&ConsistencyRule{knowledgeBase: knowledgeGraph},
		&RelevanceRule{},
		&SourceValidityRule{},
		&ConfidenceThresholdRule{threshold: 0.3},
		&DuplicateDetectionRule{knowledgeBase: knowledgeGraph},
	}

	// Initialize conflict detector
	conflictDetector, err := NewConflictDetector(ctx, knowledgeGraph)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create conflict detector").
			WithComponent("corpus.validation").
			WithOperation("NewKnowledgeValidator")
	}
	validator.conflictDetector = conflictDetector

	// Initialize fact checker
	factChecker, err := NewFactChecker(ctx)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create fact checker").
			WithComponent("corpus.validation").
			WithOperation("NewKnowledgeValidator")
	}
	validator.factChecker = factChecker

	return validator, nil
}

// Validate validates extracted knowledge against all rules
func (kv *KnowledgeValidator) Validate(ctx context.Context, knowledge extraction.ExtractedKnowledge) (ValidationResult, error) {
	if ctx.Err() != nil {
		return ValidationResult{}, gerror.Wrap(ctx.Err(), gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("corpus.validation").
			WithOperation("Validate")
	}

	result := ValidationResult{
		Valid:       true,
		Confidence:  knowledge.Confidence,
		Issues:      []ValidationIssue{},
		Suggestions: []string{},
		ValidatedAt: time.Now(),
	}

	// Apply all validation rules
	for _, rule := range kv.rules {
		if ctx.Err() != nil {
			return result, ctx.Err()
		}

		ruleResult := rule.Validate(ctx, knowledge)
		
		if !ruleResult.Valid {
			result.Valid = false
			result.Issues = append(result.Issues, ruleResult.Issues...)
		}

		result.Suggestions = append(result.Suggestions, ruleResult.Suggestions...)
		
		// Adjust confidence based on rule results
		result.Confidence *= ruleResult.Confidence
	}

	// Check for conflicts with existing knowledge
	conflicts, err := kv.conflictDetector.FindConflicts(ctx, knowledge)
	if err == nil && len(conflicts) > 0 {
		result.Valid = false
		for _, conflict := range conflicts {
			result.Issues = append(result.Issues, ValidationIssue{
				Type:        "conflict",
				Description: conflict.Description,
				Severity:    conflict.Severity,
				RelatedID:   conflict.ConflictingKnowledgeID,
			})
		}
	}

	// Fact checking for high-confidence items
	if knowledge.Confidence > 0.8 {
		factResult, err := kv.factChecker.Check(ctx, knowledge)
		if err == nil && !factResult.Verified {
			result.Confidence *= 0.7
			result.Suggestions = append(result.Suggestions, 
				"Consider manual verification due to fact-checking concerns")
			
			if factResult.Explanation != "" {
				result.Issues = append(result.Issues, ValidationIssue{
					Type:        "fact_check",
					Description: factResult.Explanation,
					Severity:    "warning",
				})
			}
		}
	}

	// Final confidence adjustment
	if result.Confidence < 0.1 {
		result.Confidence = 0.1
	}

	return result, nil
}

// ValidateBatch validates multiple knowledge items efficiently
func (kv *KnowledgeValidator) ValidateBatch(ctx context.Context, knowledgeItems []extraction.ExtractedKnowledge) ([]ValidationResult, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	results := make([]ValidationResult, len(knowledgeItems))
	
	for i, knowledge := range knowledgeItems {
		if ctx.Err() != nil {
			return results, ctx.Err()
		}

		result, err := kv.Validate(ctx, knowledge)
		if err != nil {
			// Log error but continue with other items
			results[i] = ValidationResult{
				Valid:       false,
				Confidence:  0.0,
				Issues:      []ValidationIssue{{Type: "validation_error", Description: err.Error(), Severity: "error"}},
				ValidatedAt: time.Now(),
			}
		} else {
			results[i] = result
		}
	}

	return results, nil
}

// AddRule adds a custom validation rule
func (kv *KnowledgeValidator) AddRule(rule ValidationRule) {
	kv.rules = append(kv.rules, rule)
}

// RemoveRule removes a validation rule by type
func (kv *KnowledgeValidator) RemoveRule(ruleType string) {
	var filteredRules []ValidationRule
	for _, rule := range kv.rules {
		if rule.GetType() != ruleType {
			filteredRules = append(filteredRules, rule)
		}
	}
	kv.rules = filteredRules
}

// GetValidationStats returns statistics about validation results
func (kv *KnowledgeValidator) GetValidationStats(ctx context.Context) (*ValidationStats, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	// This would typically be persisted and tracked over time
	// For now, return basic statistics
	stats := &ValidationStats{
		TotalValidated:   0,
		PassedValidation: 0,
		FailedValidation: 0,
		AverageConfidence: 0.0,
		CommonIssues:     make(map[string]int),
		ValidationRules:  len(kv.rules),
	}

	return stats, nil
}

// Validation Rules Implementation

// CompletenessRule validates that knowledge has complete required fields
type CompletenessRule struct{}

func (cr *CompletenessRule) GetType() string { return "completeness" }

func (cr *CompletenessRule) Validate(ctx context.Context, k extraction.ExtractedKnowledge) ValidationResult {
	result := ValidationResult{Valid: true, Confidence: 1.0}

	if k.Content == "" {
		result.Valid = false
		result.Issues = append(result.Issues, ValidationIssue{
			Type:        "incomplete",
			Description: "Knowledge content is empty",
			Severity:    "error",
		})
	}

	if k.Source.Type == "" {
		result.Valid = false
		result.Issues = append(result.Issues, ValidationIssue{
			Type:        "incomplete",
			Description: "Knowledge source type not specified",
			Severity:    "error",
		})
	}

	if k.ID == "" {
		result.Valid = false
		result.Issues = append(result.Issues, ValidationIssue{
			Type:        "incomplete",
			Description: "Knowledge ID is missing",
			Severity:    "error",
		})
	}

	if len(k.Content) < 10 {
		result.Confidence *= 0.8
		result.Suggestions = append(result.Suggestions, "Content is very short, consider adding more detail")
	}

	return result
}

// ConsistencyRule validates consistency with existing knowledge
type ConsistencyRule struct {
	knowledgeBase *graph.KnowledgeGraph
}

func (cr *ConsistencyRule) GetType() string { return "consistency" }

func (cr *ConsistencyRule) Validate(ctx context.Context, k extraction.ExtractedKnowledge) ValidationResult {
	result := ValidationResult{Valid: true, Confidence: 1.0}

	if cr.knowledgeBase == nil {
		return result
	}

	// Find similar knowledge
	query := graph.NewQueryBuilder().
		WithText(k.Content).
		WithMinConfidence(0.5).
		WithLimit(5).
		Build()

	similar, err := cr.knowledgeBase.Query(ctx, query)
	if err != nil {
		return result
	}

	for _, existing := range similar {
		if existing.ID != k.ID && cr.isInconsistent(k, existing) {
			result.Valid = false
			result.Issues = append(result.Issues, ValidationIssue{
				Type:        "inconsistency",
				Description: fmt.Sprintf("Conflicts with existing knowledge: %s", existing.ID),
				Severity:    "warning",
				RelatedID:   existing.ID,
			})
			result.Confidence *= 0.8
		}
	}

	return result
}

func (cr *ConsistencyRule) isInconsistent(new extraction.ExtractedKnowledge, existing *graph.KnowledgeNode) bool {
	// Simple heuristic for inconsistency detection
	newLower := strings.ToLower(new.Content)
	existingLower := strings.ToLower(existing.Content)

	// Look for contradictory terms
	contradictoryPairs := [][]string{
		{"use", "avoid"},
		{"recommended", "not recommended"},
		{"better", "worse"},
		{"should", "should not"},
		{"yes", "no"},
		{"true", "false"},
	}

	for _, pair := range contradictoryPairs {
		if (strings.Contains(newLower, pair[0]) && strings.Contains(existingLower, pair[1])) ||
		   (strings.Contains(newLower, pair[1]) && strings.Contains(existingLower, pair[0])) {
			return true
		}
	}

	return false
}

// RelevanceRule validates that knowledge is relevant and useful
type RelevanceRule struct{}

func (rr *RelevanceRule) GetType() string { return "relevance" }

func (rr *RelevanceRule) Validate(ctx context.Context, k extraction.ExtractedKnowledge) ValidationResult {
	result := ValidationResult{Valid: true, Confidence: 1.0}

	// Check for vague or overly generic content
	vaguePhrases := []string{
		"it depends", "maybe", "sometimes", "usually", "often",
		"might be", "could be", "potentially", "possibly",
	}

	contentLower := strings.ToLower(k.Content)
	vaguenessScore := 0
	for _, phrase := range vaguePhrases {
		if strings.Contains(contentLower, phrase) {
			vaguenessScore++
		}
	}

	if vaguenessScore > 2 {
		result.Confidence *= 0.7
		result.Suggestions = append(result.Suggestions, "Content contains vague language, consider being more specific")
	}

	// Check for actionable content
	actionableIndicators := []string{
		"should", "must", "use", "avoid", "implement", "configure",
		"follow", "ensure", "check", "validate", "test",
	}

	hasActionable := false
	for _, indicator := range actionableIndicators {
		if strings.Contains(contentLower, indicator) {
			hasActionable = true
			break
		}
	}

	if !hasActionable && k.Type != extraction.KnowledgeContext {
		result.Confidence *= 0.9
		result.Suggestions = append(result.Suggestions, "Consider adding actionable guidance")
	}

	return result
}

// SourceValidityRule validates the source of the knowledge
type SourceValidityRule struct{}

func (svr *SourceValidityRule) GetType() string { return "source_validity" }

func (svr *SourceValidityRule) Validate(ctx context.Context, k extraction.ExtractedKnowledge) ValidationResult {
	result := ValidationResult{Valid: true, Confidence: 1.0}

	// Validate source type
	validSourceTypes := []string{"chat", "code", "commit", "documentation", "api"}
	isValidSource := false
	for _, validType := range validSourceTypes {
		if k.Source.Type == validType {
			isValidSource = true
			break
		}
	}

	if !isValidSource {
		result.Valid = false
		result.Issues = append(result.Issues, ValidationIssue{
			Type:        "invalid_source",
			Description: fmt.Sprintf("Unknown source type: %s", k.Source.Type),
			Severity:    "error",
		})
	}

	// Check source completeness
	switch k.Source.Type {
	case "chat":
		if len(k.Source.MessageIDs) == 0 {
			result.Confidence *= 0.8
			result.Suggestions = append(result.Suggestions, "Chat source missing message IDs")
		}
	case "code", "commit":
		if k.Source.CommitSHA == "" {
			result.Confidence *= 0.8
			result.Suggestions = append(result.Suggestions, "Code source missing commit information")
		}
	}

	// Check timestamp validity
	if k.Source.Timestamp.IsZero() {
		result.Confidence *= 0.9
		result.Suggestions = append(result.Suggestions, "Source timestamp is missing")
	} else if time.Since(k.Source.Timestamp) > 365*24*time.Hour {
		result.Confidence *= 0.95
		result.Suggestions = append(result.Suggestions, "Source is older than one year, verify relevance")
	}

	return result
}

// ConfidenceThresholdRule validates that confidence meets minimum thresholds
type ConfidenceThresholdRule struct {
	threshold float64
}

func (ctr *ConfidenceThresholdRule) GetType() string { return "confidence_threshold" }

func (ctr *ConfidenceThresholdRule) Validate(ctx context.Context, k extraction.ExtractedKnowledge) ValidationResult {
	result := ValidationResult{Valid: true, Confidence: k.Confidence}

	if k.Confidence < ctr.threshold {
		result.Valid = false
		result.Issues = append(result.Issues, ValidationIssue{
			Type:        "low_confidence",
			Description: fmt.Sprintf("Confidence %.2f below threshold %.2f", k.Confidence, ctr.threshold),
			Severity:    "warning",
		})
	}

	// Additional confidence adjustments based on knowledge type
	switch k.Type {
	case extraction.KnowledgeDecision:
		if k.Confidence < 0.6 {
			result.Confidence *= 0.8
			result.Suggestions = append(result.Suggestions, "Decisions should have higher confidence")
		}
	case extraction.KnowledgeSolution:
		if k.Confidence < 0.7 {
			result.Confidence *= 0.9
			result.Suggestions = append(result.Suggestions, "Solutions should be well-verified")
		}
	}

	return result
}

// DuplicateDetectionRule detects potential duplicates
type DuplicateDetectionRule struct {
	knowledgeBase *graph.KnowledgeGraph
}

func (ddr *DuplicateDetectionRule) GetType() string { return "duplicate_detection" }

func (ddr *DuplicateDetectionRule) Validate(ctx context.Context, k extraction.ExtractedKnowledge) ValidationResult {
	result := ValidationResult{Valid: true, Confidence: 1.0}

	if ddr.knowledgeBase == nil {
		return result
	}

	// Search for very similar content
	query := graph.NewQueryBuilder().
		WithText(k.Content).
		WithNodeTypes(graph.NodeType(k.Type)).
		WithMinConfidence(0.8).
		WithLimit(3).
		Build()

	similar, err := ddr.knowledgeBase.Query(ctx, query)
	if err != nil {
		return result
	}

	for _, existing := range similar {
		if existing.ID != k.ID {
			similarity := ddr.calculateSimilarity(k.Content, existing.Content)
			if similarity > 0.9 {
				result.Issues = append(result.Issues, ValidationIssue{
					Type:        "potential_duplicate",
					Description: fmt.Sprintf("Very similar to existing knowledge: %s (%.1f%% similar)", existing.ID, similarity*100),
					Severity:    "warning",
					RelatedID:   existing.ID,
				})
				result.Confidence *= 0.7
			}
		}
	}

	return result
}

func (ddr *DuplicateDetectionRule) calculateSimilarity(content1, content2 string) float64 {
	// Simple Jaccard similarity
	words1 := strings.Fields(strings.ToLower(content1))
	words2 := strings.Fields(strings.ToLower(content2))

	set1 := make(map[string]bool)
	set2 := make(map[string]bool)

	for _, word := range words1 {
		set1[word] = true
	}
	for _, word := range words2 {
		set2[word] = true
	}

	intersection := 0
	for word := range set1 {
		if set2[word] {
			intersection++
		}
	}

	union := len(set1) + len(set2) - intersection
	if union == 0 {
		return 0.0
	}

	return float64(intersection) / float64(union)
}