// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package validation

import (
	"context"
	"time"

	"github.com/lancekrogers/guild-core/pkg/corpus/extraction"
)

// ValidationRule interface for different validation rules
type ValidationRule interface {
	Validate(ctx context.Context, knowledge extraction.ExtractedKnowledge) ValidationResult
	GetType() string
}

// ValidationResult represents the result of validating a piece of knowledge
type ValidationResult struct {
	Valid       bool              `json:"valid"`
	Confidence  float64           `json:"confidence"`
	Issues      []ValidationIssue `json:"issues"`
	Suggestions []string          `json:"suggestions"`
	ValidatedAt time.Time         `json:"validated_at"`
}

// ValidationIssue represents a specific validation issue
type ValidationIssue struct {
	Type        string `json:"type"`
	Description string `json:"description"`
	Severity    string `json:"severity"` // "error", "warning", "info"
	RelatedID   string `json:"related_id,omitempty"`
}

// ConflictInfo represents a conflict between pieces of knowledge
type ConflictInfo struct {
	Description            string  `json:"description"`
	Severity               string  `json:"severity"`
	ConflictingKnowledgeID string  `json:"conflicting_knowledge_id"`
	ConflictType           string  `json:"conflict_type"`
	Confidence             float64 `json:"confidence"`
}

// FactCheckResult represents the result of fact-checking knowledge
type FactCheckResult struct {
	Verified    bool      `json:"verified"`
	Confidence  float64   `json:"confidence"`
	Source      string    `json:"source,omitempty"`
	Explanation string    `json:"explanation,omitempty"`
	CheckedAt   time.Time `json:"checked_at"`
}

// ValidationStats represents statistics about validation operations
type ValidationStats struct {
	TotalValidated    int            `json:"total_validated"`
	PassedValidation  int            `json:"passed_validation"`
	FailedValidation  int            `json:"failed_validation"`
	AverageConfidence float64        `json:"average_confidence"`
	CommonIssues      map[string]int `json:"common_issues"`
	ValidationRules   int            `json:"validation_rules"`
	LastValidated     time.Time      `json:"last_validated"`
}

// ValidationConfig represents configuration for the validation system
type ValidationConfig struct {
	EnableFactChecking      bool          `json:"enable_fact_checking"`
	EnableConflictDetection bool          `json:"enable_conflict_detection"`
	MinimumConfidence       float64       `json:"minimum_confidence"`
	MaxValidationTime       time.Duration `json:"max_validation_time"`
	BatchSize               int           `json:"batch_size"`
	CacheResults            bool          `json:"cache_results"`
}

// DefaultValidationConfig returns a default validation configuration
func DefaultValidationConfig() ValidationConfig {
	return ValidationConfig{
		EnableFactChecking:      true,
		EnableConflictDetection: true,
		MinimumConfidence:       0.3,
		MaxValidationTime:       30 * time.Second,
		BatchSize:               10,
		CacheResults:            true,
	}
}

// ValidationCache represents a cache for validation results
type ValidationCache struct {
	Results   map[string]ValidationResult `json:"results"`
	ExpiresAt map[string]time.Time        `json:"expires_at"`
	TTL       time.Duration               `json:"ttl"`
}

// NewValidationCache creates a new validation cache
func NewValidationCache(ttl time.Duration) *ValidationCache {
	return &ValidationCache{
		Results:   make(map[string]ValidationResult),
		ExpiresAt: make(map[string]time.Time),
		TTL:       ttl,
	}
}

// Get retrieves a validation result from cache
func (vc *ValidationCache) Get(knowledgeID string) (ValidationResult, bool) {
	result, exists := vc.Results[knowledgeID]
	if !exists {
		return ValidationResult{}, false
	}

	// Check if expired
	if expiry, hasExpiry := vc.ExpiresAt[knowledgeID]; hasExpiry {
		if time.Now().After(expiry) {
			delete(vc.Results, knowledgeID)
			delete(vc.ExpiresAt, knowledgeID)
			return ValidationResult{}, false
		}
	}

	return result, true
}

// Set stores a validation result in cache
func (vc *ValidationCache) Set(knowledgeID string, result ValidationResult) {
	vc.Results[knowledgeID] = result
	vc.ExpiresAt[knowledgeID] = time.Now().Add(vc.TTL)
}

// Clear removes all cached results
func (vc *ValidationCache) Clear() {
	vc.Results = make(map[string]ValidationResult)
	vc.ExpiresAt = make(map[string]time.Time)
}

// CleanExpired removes expired entries from cache
func (vc *ValidationCache) CleanExpired() {
	now := time.Now()
	for id, expiry := range vc.ExpiresAt {
		if now.After(expiry) {
			delete(vc.Results, id)
			delete(vc.ExpiresAt, id)
		}
	}
}

// QualityMetrics represents quality metrics for knowledge
type QualityMetrics struct {
	Completeness   float64 `json:"completeness"`    // 0.0 to 1.0
	Consistency    float64 `json:"consistency"`     // 0.0 to 1.0
	Relevance      float64 `json:"relevance"`       // 0.0 to 1.0
	Accuracy       float64 `json:"accuracy"`        // 0.0 to 1.0
	Freshness      float64 `json:"freshness"`       // 0.0 to 1.0
	OverallQuality float64 `json:"overall_quality"` // 0.0 to 1.0
}

// CalculateOverallQuality calculates the overall quality score
func (qm *QualityMetrics) CalculateOverallQuality() {
	// Weighted average of quality dimensions
	weights := map[string]float64{
		"completeness": 0.25,
		"consistency":  0.20,
		"relevance":    0.20,
		"accuracy":     0.25,
		"freshness":    0.10,
	}

	qm.OverallQuality = (qm.Completeness*weights["completeness"] +
		qm.Consistency*weights["consistency"] +
		qm.Relevance*weights["relevance"] +
		qm.Accuracy*weights["accuracy"] +
		qm.Freshness*weights["freshness"])
}

// ValidationReport represents a comprehensive validation report
type ValidationReport struct {
	KnowledgeID      string           `json:"knowledge_id"`
	ValidationResult ValidationResult `json:"validation_result"`
	QualityMetrics   QualityMetrics   `json:"quality_metrics"`
	Recommendations  []string         `json:"recommendations"`
	TrendAnalysis    TrendAnalysis    `json:"trend_analysis"`
	GeneratedAt      time.Time        `json:"generated_at"`
}

// TrendAnalysis represents trends in knowledge quality over time
type TrendAnalysis struct {
	QualityTrend       string  `json:"quality_trend"`    // "improving", "declining", "stable"
	ConfidenceTrend    string  `json:"confidence_trend"` // "increasing", "decreasing", "stable"
	IssueCount         int     `json:"issue_count"`
	PreviousIssueCount int     `json:"previous_issue_count"`
	ChangeRate         float64 `json:"change_rate"` // Rate of change in quality
}

// ConflictResolution represents different ways to resolve conflicts
type ConflictResolution struct {
	Strategy   string    `json:"strategy"` // "merge", "replace", "ignore", "manual"
	Reasoning  string    `json:"reasoning"`
	Confidence float64   `json:"confidence"`
	ResolvedBy string    `json:"resolved_by"`
	ResolvedAt time.Time `json:"resolved_at"`
}

// ValidationAudit represents an audit trail for validation decisions
type ValidationAudit struct {
	ID            string    `json:"id"`
	KnowledgeID   string    `json:"knowledge_id"`
	Action        string    `json:"action"` // "validated", "rejected", "modified"
	Reason        string    `json:"reason"`
	ValidatorID   string    `json:"validator_id"`
	PreviousState string    `json:"previous_state"`
	NewState      string    `json:"new_state"`
	Timestamp     time.Time `json:"timestamp"`
}

// ValidationBatch represents a batch of knowledge items for validation
type ValidationBatch struct {
	ID           string                          `json:"id"`
	Items        []extraction.ExtractedKnowledge `json:"items"`
	Results      []ValidationResult              `json:"results"`
	Status       string                          `json:"status"` // "pending", "processing", "completed", "failed"
	CreatedAt    time.Time                       `json:"created_at"`
	CompletedAt  *time.Time                      `json:"completed_at,omitempty"`
	ErrorMessage string                          `json:"error_message,omitempty"`
}
