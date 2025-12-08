// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package validation

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/guild-framework/guild-core/pkg/corpus/extraction"
	"github.com/guild-framework/guild-core/pkg/corpus/graph"
	"github.com/guild-framework/guild-core/pkg/gerror"
)

// ConflictDetector identifies conflicts between pieces of knowledge
type ConflictDetector struct {
	knowledgeGraph *graph.KnowledgeGraph
	conflictRules  []ConflictRule
}

// ConflictRule interface for different types of conflict detection
type ConflictRule interface {
	DetectConflict(ctx context.Context, new extraction.ExtractedKnowledge, existing *graph.KnowledgeNode) *ConflictInfo
	GetType() string
}

// NewConflictDetector creates a new conflict detector
func NewConflictDetector(ctx context.Context, knowledgeGraph *graph.KnowledgeGraph) (*ConflictDetector, error) {
	if ctx.Err() != nil {
		return nil, gerror.Wrap(ctx.Err(), gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("corpus.validation").
			WithOperation("NewConflictDetector")
	}

	detector := &ConflictDetector{
		knowledgeGraph: knowledgeGraph,
		conflictRules: []ConflictRule{
			&SemanticConflictRule{},
			&TemporalConflictRule{},
			&AuthorityConflictRule{},
			&ContextualConflictRule{},
		},
	}

	return detector, nil
}

// FindConflicts identifies conflicts between new knowledge and existing knowledge
func (cd *ConflictDetector) FindConflicts(ctx context.Context, knowledge extraction.ExtractedKnowledge) ([]ConflictInfo, error) {
	if ctx.Err() != nil {
		return nil, gerror.Wrap(ctx.Err(), gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("corpus.validation").
			WithOperation("FindConflicts")
	}

	if cd.knowledgeGraph == nil {
		return []ConflictInfo{}, nil
	}

	var conflicts []ConflictInfo

	// Find potentially conflicting knowledge
	query := graph.NewQueryBuilder().
		WithText(knowledge.Content).
		WithNodeTypes(graph.NodeType(knowledge.Type)).
		WithMinConfidence(0.4).
		WithLimit(10).
		Build()

	candidates, err := cd.knowledgeGraph.Query(ctx, query)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to query knowledge graph").
			WithComponent("corpus.validation").
			WithOperation("FindConflicts")
	}

	// Apply conflict detection rules
	for _, candidate := range candidates {
		if candidate.ID == knowledge.ID {
			continue // Skip self
		}

		for _, rule := range cd.conflictRules {
			if ctx.Err() != nil {
				return conflicts, ctx.Err()
			}

			if conflict := rule.DetectConflict(ctx, knowledge, candidate); conflict != nil {
				conflicts = append(conflicts, *conflict)
				break // One conflict per candidate is sufficient
			}
		}
	}

	return conflicts, nil
}

// SemanticConflictRule detects semantic conflicts between knowledge
type SemanticConflictRule struct{}

func (scr *SemanticConflictRule) GetType() string { return "semantic" }

func (scr *SemanticConflictRule) DetectConflict(ctx context.Context, new extraction.ExtractedKnowledge, existing *graph.KnowledgeNode) *ConflictInfo {
	newContent := strings.ToLower(new.Content)
	existingContent := strings.ToLower(existing.Content)

	// Define semantic opposites
	opposites := [][]string{
		{"use", "avoid"},
		{"recommended", "not recommended"},
		{"should", "should not"},
		{"enable", "disable"},
		{"allow", "deny"},
		{"secure", "insecure"},
		{"fast", "slow"},
		{"efficient", "inefficient"},
		{"good", "bad"},
		{"correct", "incorrect"},
		{"valid", "invalid"},
		{"safe", "unsafe"},
		{"stable", "unstable"},
		{"reliable", "unreliable"},
	}

	for _, pair := range opposites {
		word1, word2 := pair[0], pair[1]

		// Check for direct contradictions
		if (strings.Contains(newContent, word1) && strings.Contains(existingContent, word2)) ||
			(strings.Contains(newContent, word2) && strings.Contains(existingContent, word1)) {

			// Calculate confidence based on context
			confidence := scr.calculateConflictConfidence(newContent, existingContent, word1, word2)

			if confidence > 0.6 {
				return &ConflictInfo{
					Description:            scr.buildConflictDescription(new, existing, word1, word2),
					Severity:               scr.determineSeverity(confidence),
					ConflictingKnowledgeID: existing.ID,
					ConflictType:           "semantic_opposition",
					Confidence:             confidence,
				}
			}
		}
	}

	return nil
}

func (scr *SemanticConflictRule) calculateConflictConfidence(newContent, existingContent, word1, word2 string) float64 {
	confidence := 0.6 // Base confidence

	// Increase confidence if the conflicting words are prominent
	newWords := strings.Fields(newContent)
	existingWords := strings.Fields(existingContent)

	for i, word := range newWords {
		if word == word1 || word == word2 {
			// Higher confidence if the word appears early in the content
			position := float64(i) / float64(len(newWords))
			confidence += (1.0 - position) * 0.2
			break
		}
	}

	for i, word := range existingWords {
		if word == word1 || word == word2 {
			position := float64(i) / float64(len(existingWords))
			confidence += (1.0 - position) * 0.2
			break
		}
	}

	// Cap confidence
	if confidence > 0.95 {
		confidence = 0.95
	}

	return confidence
}

func (scr *SemanticConflictRule) buildConflictDescription(new extraction.ExtractedKnowledge, existing *graph.KnowledgeNode, word1, word2 string) string {
	return fmt.Sprintf("Semantic conflict detected: new knowledge suggests '%s' while existing knowledge '%s' suggests '%s'",
		scr.findRelevantContext(new.Content, []string{word1, word2}),
		existing.ID,
		scr.findRelevantContext(existing.Content, []string{word1, word2}))
}

func (scr *SemanticConflictRule) findRelevantContext(content string, words []string) string {
	sentences := strings.Split(content, ".")
	for _, sentence := range sentences {
		for _, word := range words {
			if strings.Contains(strings.ToLower(sentence), word) {
				return strings.TrimSpace(sentence)
			}
		}
	}
	return content[:minInt(100, len(content))]
}

func (scr *SemanticConflictRule) determineSeverity(confidence float64) string {
	if confidence > 0.8 {
		return "high"
	} else if confidence > 0.6 {
		return "medium"
	}
	return "low"
}

// TemporalConflictRule detects conflicts based on timing and recency
type TemporalConflictRule struct{}

func (tcr *TemporalConflictRule) GetType() string { return "temporal" }

func (tcr *TemporalConflictRule) DetectConflict(ctx context.Context, new extraction.ExtractedKnowledge, existing *graph.KnowledgeNode) *ConflictInfo {
	// Check if both knowledge pieces are about the same topic but have different time-sensitive information
	similarity := tcr.calculateTopicSimilarity(new.Content, existing.Content)

	if similarity < 0.7 {
		return nil // Not similar enough to be conflicting
	}

	// Look for time-sensitive indicators
	timeSensitiveTerms := []string{
		"version", "latest", "current", "now", "recently", "new", "old",
		"deprecated", "obsolete", "updated", "released", "beta", "stable",
	}

	newHasTimeSensitive := tcr.hasTimeSensitiveContent(new.Content, timeSensitiveTerms)
	existingHasTimeSensitive := tcr.hasTimeSensitiveContent(existing.Content, timeSensitiveTerms)

	if newHasTimeSensitive && existingHasTimeSensitive {
		// Both have time-sensitive content, check if new supersedes old
		if new.Timestamp.After(existing.CreatedAt) {
			confidence := 0.7
			if new.Timestamp.Sub(existing.CreatedAt).Hours() > 24*30 { // More than a month
				confidence = 0.8
			}

			return &ConflictInfo{
				Description:            fmt.Sprintf("Temporal conflict: newer knowledge may supersede older knowledge from %s", existing.CreatedAt.Format("2006-01-02")),
				Severity:               "medium",
				ConflictingKnowledgeID: existing.ID,
				ConflictType:           "temporal_supersession",
				Confidence:             confidence,
			}
		}
	}

	return nil
}

func (tcr *TemporalConflictRule) calculateTopicSimilarity(content1, content2 string) float64 {
	// Simple word overlap similarity
	words1 := strings.Fields(strings.ToLower(content1))
	words2 := strings.Fields(strings.ToLower(content2))

	set1 := make(map[string]bool)
	set2 := make(map[string]bool)

	for _, word := range words1 {
		if len(word) > 3 { // Skip short words
			set1[word] = true
		}
	}
	for _, word := range words2 {
		if len(word) > 3 {
			set2[word] = true
		}
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

func (tcr *TemporalConflictRule) hasTimeSensitiveContent(content string, terms []string) bool {
	contentLower := strings.ToLower(content)
	for _, term := range terms {
		if strings.Contains(contentLower, term) {
			return true
		}
	}
	return false
}

// AuthorityConflictRule detects conflicts based on source authority
type AuthorityConflictRule struct{}

func (acr *AuthorityConflictRule) GetType() string { return "authority" }

func (acr *AuthorityConflictRule) DetectConflict(ctx context.Context, new extraction.ExtractedKnowledge, existing *graph.KnowledgeNode) *ConflictInfo {
	// Simple authority-based conflict detection
	// In a real implementation, this would consider source credibility, expertise, etc.

	newAuthority := acr.calculateSourceAuthority(new.Source)
	existingAuthority := acr.calculateExistingAuthority(existing)

	// Only flag as conflict if there's a significant authority difference
	if math.Abs(newAuthority-existingAuthority) > 0.3 &&
		acr.hasConflictingContent(new.Content, existing.Content) {

		higherAuthority := "new"
		if existingAuthority > newAuthority {
			higherAuthority = "existing"
		}

		return &ConflictInfo{
			Description:            fmt.Sprintf("Authority conflict: %s knowledge has higher authority score (%.2f vs %.2f)", higherAuthority, maxFloat64(newAuthority, existingAuthority), minFloat64(newAuthority, existingAuthority)),
			Severity:               "medium",
			ConflictingKnowledgeID: existing.ID,
			ConflictType:           "authority_mismatch",
			Confidence:             0.6,
		}
	}

	return nil
}

func (acr *AuthorityConflictRule) calculateSourceAuthority(source extraction.Source) float64 {
	// Simple authority scoring
	score := 0.5 // Base score

	switch source.Type {
	case "documentation":
		score = 0.9
	case "code":
		score = 0.8
	case "commit":
		score = 0.7
	case "chat":
		score = 0.5
	}

	// Boost for recent sources
	if time.Since(source.Timestamp).Hours() < 24*7 { // Within a week
		score += 0.1
	}

	return minFloat64(score, 1.0)
}

func (acr *AuthorityConflictRule) calculateExistingAuthority(existing *graph.KnowledgeNode) float64 {
	// Use confidence as a proxy for authority
	return existing.Confidence
}

func (acr *AuthorityConflictRule) hasConflictingContent(content1, content2 string) bool {
	// Simple check for conflicting statements
	return strings.Contains(strings.ToLower(content1), "not") != strings.Contains(strings.ToLower(content2), "not")
}

// ContextualConflictRule detects conflicts based on context
type ContextualConflictRule struct{}

func (ccr *ContextualConflictRule) GetType() string { return "contextual" }

func (ccr *ContextualConflictRule) DetectConflict(ctx context.Context, new extraction.ExtractedKnowledge, existing *graph.KnowledgeNode) *ConflictInfo {
	// Check for context mismatches that could lead to conflicts
	newContext := ccr.extractContext(new)
	existingContext := ccr.extractExistingContext(existing)

	if ccr.hasContextConflict(newContext, existingContext) {
		return &ConflictInfo{
			Description:            "Contextual conflict: knowledge applicable to different contexts or conditions",
			Severity:               "low",
			ConflictingKnowledgeID: existing.ID,
			ConflictType:           "contextual_mismatch",
			Confidence:             0.5,
		}
	}

	return nil
}

func (ccr *ContextualConflictRule) extractContext(knowledge extraction.ExtractedKnowledge) map[string]string {
	context := make(map[string]string)

	// Extract from metadata
	for key, value := range knowledge.Context {
		if str, ok := value.(string); ok {
			context[key] = str
		}
	}

	// Extract from source
	context["source_type"] = knowledge.Source.Type

	return context
}

func (ccr *ContextualConflictRule) extractExistingContext(existing *graph.KnowledgeNode) map[string]string {
	context := make(map[string]string)

	for key, value := range existing.Properties {
		if str, ok := value.(string); ok {
			context[key] = str
		}
	}

	return context
}

func (ccr *ContextualConflictRule) hasContextConflict(newContext, existingContext map[string]string) bool {
	// Simple context conflict detection
	// In practice, this would be more sophisticated

	for key, newValue := range newContext {
		if existingValue, exists := existingContext[key]; exists {
			if newValue != existingValue && key != "source_type" {
				return true
			}
		}
	}

	return false
}

// Utility functions
func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxFloat64(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

func minFloat64(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func absFloat64(a float64) float64 {
	if a < 0 {
		return -a
	}
	return a
}
