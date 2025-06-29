// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package extraction

import (
	"context"
	"regexp"
	"strings"

	"github.com/lancekrogers/guild/pkg/gerror"
)

// NLPProcessor provides natural language processing capabilities for knowledge extraction
type NLPProcessor struct {
	entityExtractor *EntityExtractor
	relationExtractor *RelationExtractor
}

// NewNLPProcessor creates a new NLP processor with default extractors
func NewNLPProcessor(ctx context.Context) (*NLPProcessor, error) {
	if ctx.Err() != nil {
		return nil, gerror.Wrap(ctx.Err(), gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("corpus.extraction").
			WithOperation("NewNLPProcessor")
	}

	entityExtractor := NewEntityExtractor()
	relationExtractor := NewRelationExtractor()

	return &NLPProcessor{
		entityExtractor:   entityExtractor,
		relationExtractor: relationExtractor,
	}, nil
}

// ExtractEntities extracts named entities from an exchange
func (nlp *NLPProcessor) ExtractEntities(ctx context.Context, exchange Exchange) ([]Entity, error) {
	if ctx.Err() != nil {
		return nil, gerror.Wrap(ctx.Err(), gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("corpus.extraction").
			WithOperation("ExtractEntities")
	}

	var allEntities []Entity

	for _, msg := range exchange.Messages {
		entities, err := nlp.entityExtractor.ExtractFromText(ctx, msg.Content)
		if err != nil {
			// Log error but continue processing other messages
			continue
		}
		allEntities = append(allEntities, entities...)
	}

	// Deduplicate entities
	return nlp.deduplicateEntities(allEntities), nil
}

// ExtractRelations extracts relationships between entities from an exchange
func (nlp *NLPProcessor) ExtractRelations(ctx context.Context, exchange Exchange) ([]Relation, error) {
	if ctx.Err() != nil {
		return nil, gerror.Wrap(ctx.Err(), gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("corpus.extraction").
			WithOperation("ExtractRelations")
	}

	var allRelations []Relation

	for _, msg := range exchange.Messages {
		relations, err := nlp.relationExtractor.ExtractFromText(ctx, msg.Content)
		if err != nil {
			// Log error but continue processing other messages
			continue
		}
		allRelations = append(allRelations, relations...)
	}

	// Deduplicate relations
	return nlp.deduplicateRelations(allRelations), nil
}

// deduplicateEntities removes duplicate entities
func (nlp *NLPProcessor) deduplicateEntities(entities []Entity) []Entity {
	seen := make(map[string]Entity)
	
	for _, entity := range entities {
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

// deduplicateRelations removes duplicate relations
func (nlp *NLPProcessor) deduplicateRelations(relations []Relation) []Relation {
	seen := make(map[string]Relation)
	
	for _, relation := range relations {
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

// EntityExtractor extracts named entities from text
type EntityExtractor struct {
	patterns map[string]*regexp.Regexp
}

// NewEntityExtractor creates a new entity extractor
func NewEntityExtractor() *EntityExtractor {
	patterns := map[string]*regexp.Regexp{
		"technology": regexp.MustCompile(`(?i)\b(JavaScript|Python|Go|Golang|Java|React|Vue|Angular|Node\.js|Docker|Kubernetes|PostgreSQL|MongoDB|MySQL|Redis|AWS|Azure|GCP|Git|GitHub|GitLab|API|REST|GraphQL|JSON|XML|HTTP|HTTPS|TCP|UDP|SQL|NoSQL|microservices?|serverless|OAuth|JWT|SSL|TLS|DevOps|CI/CD|Terraform|Ansible|Jenkins|webpack|npm|yarn|pip|cargo|maven|gradle)\b`),
		"file_type": regexp.MustCompile(`(?i)\b\w+\.(js|ts|py|go|java|cpp|c|h|css|html|json|xml|yaml|yml|md|txt|sql|sh|bat|ps1|dockerfile|makefile)\b`),
		"url": regexp.MustCompile(`https?://[^\s]+`),
		"database": regexp.MustCompile(`(?i)\b(database|db|table|collection|schema|migration|query|transaction|index|primary key|foreign key|constraint)\b`),
		"infrastructure": regexp.MustCompile(`(?i)\b(server|container|cluster|load balancer|proxy|cache|queue|worker|service|endpoint|middleware|gateway|firewall|VPC|subnet|security group)\b`),
		"programming_concept": regexp.MustCompile(`(?i)\b(function|method|class|interface|struct|variable|constant|parameter|argument|return|callback|promise|async|await|thread|process|goroutine|channel|mutex|lock|race condition|deadlock|memory leak|garbage collection)\b`),
	}

	return &EntityExtractor{
		patterns: patterns,
	}
}

// ExtractFromText extracts entities from a text string
func (ee *EntityExtractor) ExtractFromText(ctx context.Context, text string) ([]Entity, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	var entities []Entity

	for entityType, pattern := range ee.patterns {
		matches := pattern.FindAllString(text, -1)
		for _, match := range matches {
			entity := Entity{
				Name:       strings.TrimSpace(match),
				Type:       entityType,
				Confidence: ee.calculateEntityConfidence(match, entityType),
			}
			entities = append(entities, entity)
		}
	}

	return entities, nil
}

// calculateEntityConfidence calculates confidence based on entity characteristics
func (ee *EntityExtractor) calculateEntityConfidence(match, entityType string) float64 {
	confidence := 0.7 // Base confidence

	// Adjust based on entity type
	switch entityType {
	case "technology":
		confidence = 0.9 // High confidence for known technologies
	case "file_type":
		confidence = 0.95 // Very high confidence for file extensions
	case "url":
		confidence = 0.98 // Almost certain for URLs
	case "database":
		confidence = 0.8
	case "infrastructure":
		confidence = 0.8
	case "programming_concept":
		confidence = 0.75
	}

	// Adjust based on match characteristics
	if len(match) < 3 {
		confidence *= 0.7 // Lower confidence for very short matches
	}

	if strings.Contains(match, ".") && entityType != "file_type" && entityType != "url" {
		confidence *= 0.9 // Slightly lower for dotted names (could be false positives)
	}

	return confidence
}

// RelationExtractor extracts relationships between entities from text
type RelationExtractor struct {
	patterns map[string]*regexp.Regexp
}

// NewRelationExtractor creates a new relation extractor
func NewRelationExtractor() *RelationExtractor {
	patterns := map[string]*regexp.Regexp{
		"uses": regexp.MustCompile(`(?i)(\w+)\s+(uses?|using|utilized?|employs?)\s+(\w+)`),
		"depends_on": regexp.MustCompile(`(?i)(\w+)\s+(depends? on|requires?|needs?)\s+(\w+)`),
		"replaces": regexp.MustCompile(`(?i)(\w+)\s+(replaces?|replacing|substitutes?|instead of)\s+(\w+)`),
		"implements": regexp.MustCompile(`(?i)(\w+)\s+(implements?|implementing|extends?|inherits? from)\s+(\w+)`),
		"integrates_with": regexp.MustCompile(`(?i)(\w+)\s+(integrates? with|connects? to|works with)\s+(\w+)`),
		"configured_with": regexp.MustCompile(`(?i)(\w+)\s+(configured with|setup with|using)\s+(\w+)`),
	}

	return &RelationExtractor{
		patterns: patterns,
	}
}

// ExtractFromText extracts relations from a text string
func (re *RelationExtractor) ExtractFromText(ctx context.Context, text string) ([]Relation, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	var relations []Relation

	for relationType, pattern := range re.patterns {
		matches := pattern.FindAllStringSubmatch(text, -1)
		for _, match := range matches {
			if len(match) >= 4 {
				relation := Relation{
					Subject:    strings.TrimSpace(match[1]),
					Predicate:  relationType,
					Object:     strings.TrimSpace(match[3]),
					Confidence: re.calculateRelationConfidence(match, relationType),
				}
				relations = append(relations, relation)
			}
		}
	}

	return relations, nil
}

// calculateRelationConfidence calculates confidence based on relation characteristics
func (re *RelationExtractor) calculateRelationConfidence(match []string, relationType string) float64 {
	confidence := 0.6 // Base confidence for relations

	// Adjust based on relation type
	switch relationType {
	case "uses":
		confidence = 0.8
	case "depends_on":
		confidence = 0.85
	case "replaces":
		confidence = 0.9
	case "implements":
		confidence = 0.85
	case "integrates_with":
		confidence = 0.75
	case "configured_with":
		confidence = 0.7
	}

	// Adjust based on subject/object characteristics
	subject := strings.TrimSpace(match[1])
	object := strings.TrimSpace(match[3])

	// Higher confidence if both subject and object look like technical terms
	if re.looksLikeTechnicalTerm(subject) && re.looksLikeTechnicalTerm(object) {
		confidence += 0.1
	}

	// Lower confidence if either is very short or generic
	if len(subject) < 3 || len(object) < 3 {
		confidence *= 0.8
	}

	genericTerms := []string{"it", "this", "that", "thing", "stuff", "something"}
	for _, term := range genericTerms {
		if strings.EqualFold(subject, term) || strings.EqualFold(object, term) {
			confidence *= 0.5
			break
		}
	}

	return confidence
}

// looksLikeTechnicalTerm checks if a term appears to be technical
func (re *RelationExtractor) looksLikeTechnicalTerm(term string) bool {
	// Simple heuristics for technical terms
	term = strings.ToLower(term)
	
	// Contains common technical patterns
	technicalPatterns := []string{
		"api", "db", "sql", "http", "tcp", "json", "xml", "js", "py", "go",
		"server", "client", "service", "config", "auth", "oauth", "jwt",
		"docker", "k8s", "aws", "gcp", "azure", "git", "npm", "pip",
	}
	
	for _, pattern := range technicalPatterns {
		if strings.Contains(term, pattern) {
			return true
		}
	}
	
	// Has technical naming conventions (camelCase, snake_case, kebab-case)
	if strings.Contains(term, "_") || strings.Contains(term, "-") {
		return true
	}
	
	// Mixed case (likely camelCase or PascalCase)
	hasUpper := false
	hasLower := false
	for _, r := range term {
		if r >= 'A' && r <= 'Z' {
			hasUpper = true
		}
		if r >= 'a' && r <= 'z' {
			hasLower = true
		}
	}
	
	return hasUpper && hasLower
}