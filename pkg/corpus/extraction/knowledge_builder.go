// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package extraction

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/lancekrogers/guild-core/pkg/gerror"
)

// KnowledgeBuilder constructs structured knowledge from extracted information
type KnowledgeBuilder struct {
	confidenceThreshold float64
	maxContentLength    int
}

// NewKnowledgeBuilder creates a new knowledge builder with default settings
func NewKnowledgeBuilder() *KnowledgeBuilder {
	return &KnowledgeBuilder{
		confidenceThreshold: 0.5,
		maxContentLength:    2000,
	}
}

// BuildKnowledge constructs a complete knowledge object from extracted components
func (kb *KnowledgeBuilder) BuildKnowledge(ctx context.Context,
	knowledgeType KnowledgeType,
	content string,
	source Source,
	entities []Entity,
	relations []Relation,
	metadata map[string]interface{},
) (*ExtractedKnowledge, error) {
	if ctx.Err() != nil {
		return nil, gerror.Wrap(ctx.Err(), gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("corpus.extraction").
			WithOperation("BuildKnowledge")
	}

	if content == "" {
		return nil, gerror.New(gerror.ErrCodeInvalidInput, "knowledge content cannot be empty", nil).
			WithComponent("corpus.extraction").
			WithOperation("BuildKnowledge")
	}

	// Generate unique ID
	id := kb.generateKnowledgeID(knowledgeType)

	// Clean and truncate content if necessary
	cleanContent := kb.cleanContent(content)

	// Calculate confidence based on various factors
	confidence := kb.calculateOverallConfidence(ctx, cleanContent, entities, relations, source)

	// Build context information
	context := kb.buildContext(ctx, entities, relations, source)

	// Initialize metadata if nil
	if metadata == nil {
		metadata = make(map[string]interface{})
	}

	// Add extraction metadata
	metadata["extraction_time"] = time.Now()
	metadata["entity_count"] = len(entities)
	metadata["relation_count"] = len(relations)
	metadata["content_length"] = len(cleanContent)

	knowledge := &ExtractedKnowledge{
		ID:         id,
		Type:       knowledgeType,
		Content:    cleanContent,
		Source:     source,
		Confidence: confidence,
		Entities:   kb.filterEntitiesByConfidence(entities),
		Relations:  kb.filterRelationsByConfidence(relations),
		Validation: ValidationStatus{
			Valid:       true,
			Confidence:  confidence,
			ValidatedAt: time.Now(),
		},
		Context:   context,
		Metadata:  metadata,
		Timestamp: time.Now(),
	}

	return knowledge, nil
}

// EnrichKnowledge adds additional information to existing knowledge
func (kb *KnowledgeBuilder) EnrichKnowledge(ctx context.Context, knowledge *ExtractedKnowledge,
	additionalEntities []Entity, additionalRelations []Relation,
	additionalMetadata map[string]interface{},
) error {
	if ctx.Err() != nil {
		return gerror.Wrap(ctx.Err(), gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("corpus.extraction").
			WithOperation("EnrichKnowledge")
	}

	if knowledge == nil {
		return gerror.New(gerror.ErrCodeInvalidInput, "knowledge cannot be nil", nil).
			WithComponent("corpus.extraction").
			WithOperation("EnrichKnowledge")
	}

	// Merge entities
	allEntities := append(knowledge.Entities, additionalEntities...)
	knowledge.Entities = kb.deduplicateAndFilterEntities(allEntities)

	// Merge relations
	allRelations := append(knowledge.Relations, additionalRelations...)
	knowledge.Relations = kb.deduplicateAndFilterRelations(allRelations)

	// Merge metadata
	if knowledge.Metadata == nil {
		knowledge.Metadata = make(map[string]interface{})
	}
	for key, value := range additionalMetadata {
		knowledge.Metadata[key] = value
	}

	// Update enrichment tracking
	knowledge.Metadata["last_enriched"] = time.Now()
	knowledge.Metadata["entity_count"] = len(knowledge.Entities)
	knowledge.Metadata["relation_count"] = len(knowledge.Relations)

	// Recalculate confidence
	knowledge.Confidence = kb.calculateOverallConfidence(ctx, knowledge.Content,
		knowledge.Entities, knowledge.Relations, knowledge.Source)

	return nil
}

// MergeKnowledge combines multiple related knowledge pieces
func (kb *KnowledgeBuilder) MergeKnowledge(ctx context.Context, primary *ExtractedKnowledge,
	secondary []ExtractedKnowledge,
) (*ExtractedKnowledge, error) {
	if ctx.Err() != nil {
		return nil, gerror.Wrap(ctx.Err(), gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("corpus.extraction").
			WithOperation("MergeKnowledge")
	}

	if primary == nil {
		return nil, gerror.New(gerror.ErrCodeInvalidInput, "primary knowledge cannot be nil", nil).
			WithComponent("corpus.extraction").
			WithOperation("MergeKnowledge")
	}

	merged := &ExtractedKnowledge{
		ID:        kb.generateKnowledgeID(primary.Type),
		Type:      primary.Type,
		Content:   primary.Content,
		Source:    primary.Source,
		Entities:  make([]Entity, 0),
		Relations: make([]Relation, 0),
		Context:   make(map[string]interface{}),
		Metadata:  make(map[string]interface{}),
		Timestamp: time.Now(),
	}

	// Start with primary knowledge
	merged.Entities = append(merged.Entities, primary.Entities...)
	merged.Relations = append(merged.Relations, primary.Relations...)

	// Copy primary context and metadata
	for key, value := range primary.Context {
		merged.Context[key] = value
	}
	for key, value := range primary.Metadata {
		merged.Metadata[key] = value
	}

	// Merge secondary knowledge pieces
	var secondaryContents []string
	var allSources []Source

	allSources = append(allSources, primary.Source)

	for _, sec := range secondary {
		if sec.Type == primary.Type { // Only merge same types
			merged.Entities = append(merged.Entities, sec.Entities...)
			merged.Relations = append(merged.Relations, sec.Relations...)
			secondaryContents = append(secondaryContents, sec.Content)
			allSources = append(allSources, sec.Source)

			// Merge metadata with prefixes to avoid conflicts
			for key, value := range sec.Metadata {
				prefixedKey := fmt.Sprintf("secondary_%s_%s", sec.ID, key)
				merged.Metadata[prefixedKey] = value
			}
		}
	}

	// Append secondary contents
	if len(secondaryContents) > 0 {
		merged.Content += "\n\nRelated:\n" + strings.Join(secondaryContents, "\n")
	}

	// Deduplicate entities and relations
	merged.Entities = kb.deduplicateAndFilterEntities(merged.Entities)
	merged.Relations = kb.deduplicateAndFilterRelations(merged.Relations)

	// Add merge metadata
	merged.Metadata["merged_from"] = len(secondary) + 1
	merged.Metadata["merge_time"] = time.Now()
	merged.Metadata["source_count"] = len(allSources)

	// Calculate confidence for merged knowledge
	merged.Confidence = kb.calculateMergedConfidence(primary, secondary)

	// Set validation status
	merged.Validation = ValidationStatus{
		Valid:       true,
		Confidence:  merged.Confidence,
		ValidatedAt: time.Now(),
	}

	return merged, nil
}

// Helper methods

func (kb *KnowledgeBuilder) generateKnowledgeID(knowledgeType KnowledgeType) string {
	timestamp := time.Now().UnixNano()
	return fmt.Sprintf("%s_%d", knowledgeType.String(), timestamp)
}

func (kb *KnowledgeBuilder) cleanContent(content string) string {
	// Remove excessive whitespace
	cleaned := strings.TrimSpace(content)
	cleaned = strings.ReplaceAll(cleaned, "\n\n\n", "\n\n")
	cleaned = strings.ReplaceAll(cleaned, "  ", " ")

	// Truncate if too long
	if len(cleaned) > kb.maxContentLength {
		cleaned = cleaned[:kb.maxContentLength] + "..."
	}

	return cleaned
}

func (kb *KnowledgeBuilder) calculateOverallConfidence(ctx context.Context, content string,
	entities []Entity, relations []Relation, source Source,
) float64 {
	baseConfidence := 0.5

	// Increase confidence based on content quality
	if len(content) > 50 {
		baseConfidence += 0.1
	}
	if len(content) > 200 {
		baseConfidence += 0.1
	}

	// Increase confidence based on entity and relation count
	if len(entities) > 0 {
		baseConfidence += 0.1
	}
	if len(relations) > 0 {
		baseConfidence += 0.1
	}

	// Factor in average entity confidence
	if len(entities) > 0 {
		entityConfidenceSum := 0.0
		for _, entity := range entities {
			entityConfidenceSum += entity.Confidence
		}
		avgEntityConfidence := entityConfidenceSum / float64(len(entities))
		baseConfidence = (baseConfidence + avgEntityConfidence) / 2
	}

	// Factor in average relation confidence
	if len(relations) > 0 {
		relationConfidenceSum := 0.0
		for _, relation := range relations {
			relationConfidenceSum += relation.Confidence
		}
		avgRelationConfidence := relationConfidenceSum / float64(len(relations))
		baseConfidence = (baseConfidence + avgRelationConfidence) / 2
	}

	// Adjust based on source type
	switch source.Type {
	case "chat":
		// Chat sources can be less reliable
		baseConfidence *= 0.9
	case "code":
		// Code sources are generally more reliable
		baseConfidence *= 1.1
	case "commit":
		// Commit sources are very reliable
		baseConfidence *= 1.2
	}

	// Cap confidence
	if baseConfidence > 0.95 {
		baseConfidence = 0.95
	} else if baseConfidence < 0.1 {
		baseConfidence = 0.1
	}

	return baseConfidence
}

func (kb *KnowledgeBuilder) buildContext(ctx context.Context, entities []Entity,
	relations []Relation, source Source,
) map[string]interface{} {
	context := make(map[string]interface{})

	// Entity summary
	if len(entities) > 0 {
		entityTypes := make(map[string]int)
		for _, entity := range entities {
			entityTypes[entity.Type]++
		}
		context["entity_types"] = entityTypes
	}

	// Relation summary
	if len(relations) > 0 {
		relationTypes := make(map[string]int)
		for _, relation := range relations {
			relationTypes[relation.Predicate]++
		}
		context["relation_types"] = relationTypes
	}

	// Source context
	context["source_type"] = source.Type
	if len(source.Participants) > 0 {
		context["participants"] = source.Participants
	}

	return context
}

func (kb *KnowledgeBuilder) filterEntitiesByConfidence(entities []Entity) []Entity {
	var filtered []Entity
	for _, entity := range entities {
		if entity.Confidence >= kb.confidenceThreshold {
			filtered = append(filtered, entity)
		}
	}
	return filtered
}

func (kb *KnowledgeBuilder) filterRelationsByConfidence(relations []Relation) []Relation {
	var filtered []Relation
	for _, relation := range relations {
		if relation.Confidence >= kb.confidenceThreshold {
			filtered = append(filtered, relation)
		}
	}
	return filtered
}

func (kb *KnowledgeBuilder) deduplicateAndFilterEntities(entities []Entity) []Entity {
	seen := make(map[string]Entity)

	for _, entity := range entities {
		if entity.Confidence < kb.confidenceThreshold {
			continue
		}

		key := strings.ToLower(entity.Name) + ":" + entity.Type
		if existing, exists := seen[key]; exists {
			// Keep the entity with higher confidence
			if entity.Confidence > existing.Confidence {
				seen[key] = entity
			}
		} else {
			seen[key] = entity
		}
	}

	var result []Entity
	for _, entity := range seen {
		result = append(result, entity)
	}

	return result
}

func (kb *KnowledgeBuilder) deduplicateAndFilterRelations(relations []Relation) []Relation {
	seen := make(map[string]Relation)

	for _, relation := range relations {
		if relation.Confidence < kb.confidenceThreshold {
			continue
		}

		key := strings.ToLower(relation.Subject) + ":" + relation.Predicate + ":" + strings.ToLower(relation.Object)
		if existing, exists := seen[key]; exists {
			// Keep the relation with higher confidence
			if relation.Confidence > existing.Confidence {
				seen[key] = relation
			}
		} else {
			seen[key] = relation
		}
	}

	var result []Relation
	for _, relation := range seen {
		result = append(result, relation)
	}

	return result
}

func (kb *KnowledgeBuilder) calculateMergedConfidence(primary *ExtractedKnowledge, secondary []ExtractedKnowledge) float64 {
	if len(secondary) == 0 {
		return primary.Confidence
	}

	// Start with primary confidence
	totalConfidence := primary.Confidence
	totalWeight := 1.0

	// Add weighted secondary confidences
	for _, sec := range secondary {
		weight := 0.5 // Secondary knowledge has lower weight
		totalConfidence += sec.Confidence * weight
		totalWeight += weight
	}

	// Calculate weighted average
	mergedConfidence := totalConfidence / totalWeight

	// Apply bonus for having multiple corroborating sources
	corroborationBonus := float64(len(secondary)) * 0.05
	mergedConfidence += corroborationBonus

	// Cap confidence
	if mergedConfidence > 0.95 {
		mergedConfidence = 0.95
	}

	return mergedConfidence
}

// SetConfidenceThreshold sets the minimum confidence threshold for entities and relations
func (kb *KnowledgeBuilder) SetConfidenceThreshold(threshold float64) {
	if threshold >= 0.0 && threshold <= 1.0 {
		kb.confidenceThreshold = threshold
	}
}

// SetMaxContentLength sets the maximum length for knowledge content
func (kb *KnowledgeBuilder) SetMaxContentLength(length int) {
	if length > 0 {
		kb.maxContentLength = length
	}
}
