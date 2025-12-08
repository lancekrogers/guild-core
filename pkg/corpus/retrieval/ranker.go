// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package retrieval

import (
	"math"
	"sort"
	"strings"
	"time"

	"github.com/guild-framework/guild-core/pkg/gerror"
)

// Scorer defines an interface for document scoring
type Scorer interface {
	Name() string
	Score(doc Document, query Query) float64
}

// ResultRanker implements sophisticated document ranking
type ResultRanker struct {
	scorers []Scorer
	weights map[string]float64
}

// NewResultRanker creates a new result ranker with default scorers
func NewResultRanker() *ResultRanker {
	ranker := &ResultRanker{
		scorers: make([]Scorer, 0),
		weights: make(map[string]float64),
	}

	// Add default scorers with weights
	ranker.AddScorer(&RecencyScorer{}, 0.15)
	ranker.AddScorer(&RelevanceScorer{}, 0.35)
	ranker.AddScorer(&ContextualScorer{}, 0.30)
	ranker.AddScorer(&AuthorityScorer{}, 0.20)

	return ranker
}

// AddScorer adds a scorer with the specified weight
func (rr *ResultRanker) AddScorer(scorer Scorer, weight float64) error {
	if scorer == nil {
		return gerror.New(gerror.ErrCodeValidation, "scorer cannot be nil", nil).
			WithComponent("ResultRanker").
			WithOperation("AddScorer")
	}

	if weight < 0 || weight > 1 {
		return gerror.New(gerror.ErrCodeValidation, "weight must be between 0 and 1", nil).
			WithComponent("ResultRanker").
			WithOperation("AddScorer")
	}

	rr.scorers = append(rr.scorers, scorer)
	rr.weights[scorer.Name()] = weight

	return nil
}

// Rank applies all scorers and returns ranked documents
func (rr *ResultRanker) Rank(docs []Document, query Query) []RankedDocument {
	ranked := make([]RankedDocument, 0, len(docs))

	for _, doc := range docs {
		scoreDetails := make(map[string]float64)

		// Start with original document score as base
		finalScore := doc.Score
		scoreDetails["original"] = doc.Score

		// Apply each scorer
		for _, scorer := range rr.scorers {
			score := scorer.Score(doc, query)
			scoreDetails[scorer.Name()] = score

			// Apply weight
			weight := rr.weights[scorer.Name()]
			finalScore += score * weight
		}

		ranked = append(ranked, RankedDocument{
			Document:     doc,
			FinalScore:   finalScore,
			ScoreDetails: scoreDetails,
		})
	}

	// Sort by final score (highest first)
	sort.Slice(ranked, func(i, j int) bool {
		return ranked[i].FinalScore > ranked[j].FinalScore
	})

	return ranked
}

// RecencyScorer scores documents based on how recently they were modified
type RecencyScorer struct{}

// Name returns the scorer name
func (rs *RecencyScorer) Name() string {
	return "recency"
}

// Score calculates a recency score using exponential decay
func (rs *RecencyScorer) Score(doc Document, query Query) float64 {
	// Try to get last modified time from metadata
	if lastModified, ok := doc.Metadata["last_modified"]; ok {
		var modTime time.Time

		switch t := lastModified.(type) {
		case time.Time:
			modTime = t
		case string:
			// Try to parse time string
			if parsed, err := time.Parse(time.RFC3339, t); err == nil {
				modTime = parsed
			} else {
				return 0.5 // Default score if parsing fails
			}
		default:
			return 0.5 // Default score if type is unexpected
		}

		age := time.Since(modTime)

		// Exponential decay with weekly half-life
		// Documents lose half their recency score every week
		return math.Exp(-age.Hours() * math.Ln2 / 168)
	}

	// If no timestamp available, return middle score
	return 0.5
}

// RelevanceScorer scores documents based on content relevance
type RelevanceScorer struct {
	embedder Embedder
}

// NewRelevanceScorer creates a new relevance scorer
func NewRelevanceScorer(embedder Embedder) *RelevanceScorer {
	return &RelevanceScorer{embedder: embedder}
}

// Name returns the scorer name
func (rs *RelevanceScorer) Name() string {
	return "relevance"
}

// Score calculates relevance based on vector similarity or keyword overlap
func (rs *RelevanceScorer) Score(doc Document, query Query) float64 {
	// Use vector similarity if available
	if baseScore, ok := doc.Metadata["vector_score"].(float64); ok {
		return baseScore
	}

	// Fallback to keyword overlap
	return rs.keywordOverlap(doc.Content, query.Text)
}

// keywordOverlap calculates similarity based on keyword overlap
func (rs *RelevanceScorer) keywordOverlap(content, query string) float64 {
	contentWords := rs.extractWords(strings.ToLower(content))
	queryWords := rs.extractWords(strings.ToLower(query))

	if len(queryWords) == 0 {
		return 0.0
	}

	// Count overlapping words
	overlap := 0
	for _, queryWord := range queryWords {
		for _, contentWord := range contentWords {
			if queryWord == contentWord {
				overlap++
				break
			}
		}
	}

	// Normalize by query length
	return float64(overlap) / float64(len(queryWords))
}

// extractWords extracts words from text, filtering out stop words
func (rs *RelevanceScorer) extractWords(text string) []string {
	words := strings.Fields(text)

	// Simple stop words list
	stopWords := map[string]bool{
		"the": true, "a": true, "an": true, "and": true, "or": true, "but": true,
		"in": true, "on": true, "at": true, "to": true, "for": true, "of": true,
		"with": true, "by": true, "is": true, "are": true, "was": true, "were": true,
	}

	filtered := make([]string, 0, len(words))
	for _, word := range words {
		word = strings.Trim(word, ".,!?;:")
		if len(word) > 2 && !stopWords[word] {
			filtered = append(filtered, word)
		}
	}

	return filtered
}

// ContextualScorer scores documents based on contextual relevance
type ContextualScorer struct{}

// Name returns the scorer name
func (cs *ContextualScorer) Name() string {
	return "contextual"
}

// Score calculates contextual relevance based on query context
func (cs *ContextualScorer) Score(doc Document, query Query) float64 {
	score := 0.0

	// Boost if document mentions current files
	for _, file := range query.Context.CurrentFiles {
		if strings.Contains(doc.Content, file) {
			score += 0.2
		}
	}

	// Boost if document has matching tags
	if docTags, ok := doc.Metadata["tags"]; ok {
		docTagsSlice := cs.convertToStringSlice(docTags)
		for _, queryTag := range query.Context.Tags {
			for _, docTag := range docTagsSlice {
				if strings.EqualFold(queryTag, docTag) {
					score += 0.15
					break
				}
			}
		}
	}

	// Boost if from same agent's previous work
	if author, ok := doc.Metadata["author"].(string); ok {
		if author == query.Context.AgentID {
			score += 0.1
		}
	}

	// Boost if document type matches current task context
	if docType, ok := doc.Metadata["type"].(string); ok {
		// Determine task type from current files
		taskType := cs.inferTaskType(query.Context.CurrentFiles)
		if taskType != "" && strings.Contains(docType, taskType) {
			score += 0.1
		}
	}

	// Cap the score at 1.0
	return math.Min(score, 1.0)
}

// convertToStringSlice safely converts interface{} to []string
func (cs *ContextualScorer) convertToStringSlice(value interface{}) []string {
	switch v := value.(type) {
	case []string:
		return v
	case []interface{}:
		result := make([]string, 0, len(v))
		for _, item := range v {
			if str, ok := item.(string); ok {
				result = append(result, str)
			}
		}
		return result
	case string:
		return []string{v}
	default:
		return []string{}
	}
}

// inferTaskType attempts to determine the type of task from current files
func (cs *ContextualScorer) inferTaskType(files []string) string {
	for _, file := range files {
		lower := strings.ToLower(file)
		// Check for testing patterns first (higher priority)
		if strings.Contains(lower, "test") {
			return "testing"
		} else if strings.HasSuffix(lower, ".md") {
			return "documentation"
		} else if strings.HasSuffix(lower, ".go") {
			return "golang"
		} else if strings.HasSuffix(lower, ".js") || strings.HasSuffix(lower, ".ts") {
			return "javascript"
		} else if strings.HasSuffix(lower, ".py") {
			return "python"
		}
	}
	return ""
}

// AuthorityScorer scores documents based on their authority/citation count
type AuthorityScorer struct {
	citationGraph CitationGraph
}

// CitationGraph interface for getting document authority scores
type CitationGraph interface {
	GetAuthority(documentID string) float64
}

// NewAuthorityScorer creates a new authority scorer
func NewAuthorityScorer(citationGraph CitationGraph) *AuthorityScorer {
	return &AuthorityScorer{citationGraph: citationGraph}
}

// Name returns the scorer name
func (as *AuthorityScorer) Name() string {
	return "authority"
}

// Score calculates authority score using PageRank-style algorithm
func (as *AuthorityScorer) Score(doc Document, query Query) float64 {
	if as.citationGraph == nil {
		// Fallback: use view count or creation time as proxy
		if viewCount, ok := doc.Metadata["view_count"].(float64); ok {
			// Normalize view count to 0-1 range (assuming max 100 views)
			return math.Min(viewCount/100.0, 1.0)
		}

		// If no citation graph available, return neutral score
		return 0.5
	}

	// Use citation graph to get authority score
	return as.citationGraph.GetAuthority(doc.ID)
}
