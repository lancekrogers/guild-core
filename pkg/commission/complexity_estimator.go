// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package commission

import (
	"fmt"
	"math"
	"strings"
)

// ComplexityEstimator implements multi-factor complexity estimation for tasks
type ComplexityEstimator struct {
	factors []ComplexityFactor
}

// ComplexityFactor represents a factor that contributes to task complexity
type ComplexityFactor struct {
	Name        string                        `json:"name"`
	Description string                        `json:"description"`
	Weight      float64                       `json:"weight"`
	Calculator  func(task *RefinedTask) float64 `json:"-"` // Not serialized
}

// ComplexityMetrics contains detailed complexity breakdown
type ComplexityMetrics struct {
	TotalScore     float64            `json:"total_score"`
	NormalizedScore int               `json:"normalized_score"` // 1-8 scale
	FactorScores   map[string]float64 `json:"factor_scores"`
	Justification  string             `json:"justification"`
}

// NewComplexityEstimator creates a new complexity estimator with predefined factors
func NewComplexityEstimator() *ComplexityEstimator {
	ce := &ComplexityEstimator{
		factors: []ComplexityFactor{},
	}
	
	ce.initializeFactors()
	return ce
}

// Estimate estimates the complexity of a task using all factors
func (ce *ComplexityEstimator) Estimate(task *RefinedTask) int {
	metrics := ce.EstimateWithMetrics(task)
	return metrics.NormalizedScore
}

// EstimateWithMetrics provides detailed complexity estimation with factor breakdown
func (ce *ComplexityEstimator) EstimateWithMetrics(task *RefinedTask) *ComplexityMetrics {
	totalScore := 0.0
	factorScores := make(map[string]float64)
	
	// Calculate score for each factor
	for _, factor := range ce.factors {
		score := factor.Calculator(task) * factor.Weight
		totalScore += score
		factorScores[factor.Name] = score
	}
	
	// Normalize to 1-8 Fibonacci scale
	normalized := ce.normalize(totalScore)
	
	// Generate justification
	justification := ce.generateJustification(task, factorScores, normalized)
	
	return &ComplexityMetrics{
		TotalScore:      totalScore,
		NormalizedScore: normalized,
		FactorScores:    factorScores,
		Justification:   justification,
	}
}

// initializeFactors sets up the complexity factors
func (ce *ComplexityEstimator) initializeFactors() {
	ce.factors = []ComplexityFactor{
		{
			Name:        "task_type_complexity",
			Description: "Base complexity based on task type",
			Weight:      0.25,
			Calculator:  ce.calculateTaskTypeComplexity,
		},
		{
			Name:        "implementation_scope",
			Description: "Complexity based on implementation scope and lines of code estimate",
			Weight:      0.20,
			Calculator:  ce.calculateImplementationScope,
		},
		{
			Name:        "dependency_complexity",
			Description: "Complexity from task dependencies and prerequisites",
			Weight:      0.15,
			Calculator:  ce.calculateDependencyComplexity,
		},
		{
			Name:        "technical_complexity",
			Description: "Technical difficulty and domain expertise required",
			Weight:      0.15,
			Calculator:  ce.calculateTechnicalComplexity,
		},
		{
			Name:        "integration_complexity",
			Description: "Complexity from external integrations and APIs",
			Weight:      0.10,
			Calculator:  ce.calculateIntegrationComplexity,
		},
		{
			Name:        "testing_complexity",
			Description: "Test coverage and validation requirements",
			Weight:      0.10,
			Calculator:  ce.calculateTestingComplexity,
		},
		{
			Name:        "uncertainty_factor",
			Description: "Unknown requirements and potential unknowns",
			Weight:      0.05,
			Calculator:  ce.calculateUncertaintyFactor,
		},
	}
}

// Factor calculation methods

func (ce *ComplexityEstimator) calculateTaskTypeComplexity(task *RefinedTask) float64 {
	// Base complexity scores by task type
	baseScores := map[string]float64{
		"design":         2.0, // Design tasks are conceptual
		"implementation": 5.0, // Implementation is the core work
		"testing":        3.0, // Testing requires systematic thinking
		"documentation":  1.5, // Documentation is generally straightforward
		"deployment":     4.0, // Deployment involves multiple systems
		"integration":    6.0, // Integration is inherently complex
		"research":       3.5, // Research involves unknowns
		"refactoring":    4.5, // Refactoring requires deep understanding
	}
	
	score, exists := baseScores[task.Type]
	if !exists {
		return 3.0 // Default medium complexity
	}
	
	return score
}

func (ce *ComplexityEstimator) calculateImplementationScope(task *RefinedTask) float64 {
	content := strings.ToLower(task.Title + " " + task.Description)
	score := 0.0
	
	// Estimate lines of code based on keywords
	locIndicators := map[string]float64{
		"simple":     1.0,
		"basic":      1.5,
		"standard":   2.0,
		"complex":    4.0,
		"advanced":   5.0,
		"enterprise": 6.0,
		"comprehensive": 5.5,
		"full":       4.0,
		"complete":   3.5,
	}
	
	// Check for scope indicators
	for keyword, value := range locIndicators {
		if strings.Contains(content, keyword) {
			score = math.Max(score, value)
		}
	}
	
	// Estimate based on task description length and detail
	descriptionLength := len(task.Description)
	if descriptionLength > 200 {
		score += 1.0 // Detailed description suggests complex task
	}
	if descriptionLength > 500 {
		score += 1.0 // Very detailed description
	}
	
	// Check for multiple components or features
	componentKeywords := []string{"and", "with", "including", "plus", "also", "additionally"}
	componentCount := 0
	for _, keyword := range componentKeywords {
		if strings.Contains(content, keyword) {
			componentCount++
		}
	}
	score += float64(componentCount) * 0.5
	
	// Default minimum score
	if score == 0 {
		score = 2.0
	}
	
	return math.Min(score, 8.0) // Cap at maximum
}

func (ce *ComplexityEstimator) calculateDependencyComplexity(task *RefinedTask) float64 {
	dependencyCount := len(task.Dependencies)
	
	// Base score from dependency count
	score := float64(dependencyCount) * 0.5
	
	// Bonus for having prerequisites (indicates sequential work)
	if len(task.Prerequisites) > 0 {
		score += float64(len(task.Prerequisites)) * 0.3
	}
	
	// Check for blocking dependencies in description
	content := strings.ToLower(task.Description)
	blockingKeywords := []string{"requires", "depends on", "needs", "must have", "prerequisite"}
	for _, keyword := range blockingKeywords {
		if strings.Contains(content, keyword) {
			score += 0.5
		}
	}
	
	return math.Min(score, 6.0)
}

func (ce *ComplexityEstimator) calculateTechnicalComplexity(task *RefinedTask) float64 {
	content := strings.ToLower(task.Title + " " + task.Description)
	score := 0.0
	
	// High complexity technical keywords
	highComplexityKeywords := map[string]float64{
		"algorithm":     3.0,
		"optimization":  2.5,
		"performance":   2.0,
		"scalability":   2.5,
		"security":      2.0,
		"encryption":    3.0,
		"machine learning": 4.0,
		"ai":            3.5,
		"blockchain":    4.0,
		"distributed":   3.0,
		"microservices": 2.5,
		"real-time":     2.5,
		"concurrency":   3.0,
		"threading":     3.0,
		"async":         2.0,
		"websocket":     2.0,
		"streaming":     2.5,
	}
	
	// Medium complexity keywords
	mediumComplexityKeywords := map[string]float64{
		"database":     1.5,
		"api":          1.0,
		"rest":         1.0,
		"graphql":      2.0,
		"authentication": 1.8,
		"authorization": 2.0,
		"validation":   1.2,
		"parsing":      1.5,
		"transformation": 1.8,
		"migration":    2.0,
	}
	
	// Check for high complexity indicators
	for keyword, value := range highComplexityKeywords {
		if strings.Contains(content, keyword) {
			score = math.Max(score, value)
		}
	}
	
	// Check for medium complexity indicators
	for keyword, value := range mediumComplexityKeywords {
		if strings.Contains(content, keyword) {
			score = math.Max(score, value)
		}
	}
	
	// Check for multiple technologies
	techKeywords := []string{"frontend", "backend", "database", "api", "ui", "server", "client"}
	techCount := 0
	for _, keyword := range techKeywords {
		if strings.Contains(content, keyword) {
			techCount++
		}
	}
	if techCount > 2 {
		score += float64(techCount-2) * 0.5 // Bonus for multi-technology tasks
	}
	
	return math.Min(score, 6.0)
}

func (ce *ComplexityEstimator) calculateIntegrationComplexity(task *RefinedTask) float64 {
	content := strings.ToLower(task.Title + " " + task.Description)
	score := 0.0
	
	// Integration indicators
	integrationKeywords := map[string]float64{
		"third-party":    2.0,
		"external":       1.5,
		"integration":    2.0,
		"api":            1.0,
		"webhook":        1.5,
		"callback":       1.5,
		"oauth":          2.0,
		"saml":           2.5,
		"ldap":           2.0,
		"payment":        2.5,
		"gateway":        2.0,
		"service":        1.0,
		"microservice":   1.5,
		"connect":        1.0,
		"interface":      0.8,
	}
	
	for keyword, value := range integrationKeywords {
		if strings.Contains(content, keyword) {
			score = math.Max(score, value)
		}
	}
	
	// Multiple system integration
	systemKeywords := []string{"system", "service", "platform", "application", "tool"}
	systemCount := 0
	for _, keyword := range systemKeywords {
		if strings.Contains(content, keyword) {
			systemCount++
		}
	}
	if systemCount > 1 {
		score += float64(systemCount-1) * 0.5
	}
	
	return math.Min(score, 5.0)
}

func (ce *ComplexityEstimator) calculateTestingComplexity(task *RefinedTask) float64 {
	content := strings.ToLower(task.Title + " " + task.Description)
	score := 0.0
	
	// Base score for task type
	if task.Type == "testing" {
		score = 2.0 // Testing tasks have inherent complexity
	} else {
		score = 1.0 // All tasks need some testing
	}
	
	// Testing complexity indicators
	testingKeywords := map[string]float64{
		"unit test":        1.0,
		"integration test": 2.0,
		"e2e test":         2.5,
		"performance test": 3.0,
		"load test":        3.0,
		"security test":    2.5,
		"automation":       1.5,
		"coverage":         1.0,
		"quality":          1.0,
		"validation":       1.2,
	}
	
	for keyword, value := range testingKeywords {
		if strings.Contains(content, keyword) {
			score = math.Max(score, value)
		}
	}
	
	// Complex features need more testing
	complexFeatures := []string{"authentication", "payment", "security", "real-time", "distributed"}
	for _, feature := range complexFeatures {
		if strings.Contains(content, feature) {
			score += 0.5
		}
	}
	
	return math.Min(score, 4.0)
}

func (ce *ComplexityEstimator) calculateUncertaintyFactor(task *RefinedTask) float64 {
	content := strings.ToLower(task.Title + " " + task.Description)
	score := 0.0
	
	// Uncertainty indicators
	uncertaintyKeywords := []string{
		"research", "investigate", "explore", "analyze", "study",
		"unclear", "unknown", "tbd", "to be determined",
		"prototype", "proof of concept", "experiment",
		"new", "innovative", "cutting-edge", "experimental",
	}
	
	for _, keyword := range uncertaintyKeywords {
		if strings.Contains(content, keyword) {
			score += 0.5
		}
	}
	
	// Vague descriptions indicate uncertainty
	vagueKeywords := []string{"somehow", "maybe", "possibly", "potentially", "might", "could"}
	for _, keyword := range vagueKeywords {
		if strings.Contains(content, keyword) {
			score += 0.3
		}
	}
	
	// Short descriptions might indicate incomplete requirements
	if len(task.Description) < 50 {
		score += 1.0
	}
	
	return math.Min(score, 3.0)
}

// normalize converts the raw score to the 1-8 Fibonacci scale
func (ce *ComplexityEstimator) normalize(rawScore float64) int {
	// Define thresholds for Fibonacci complexity scale
	// Based on typical score ranges from factor calculations
	thresholds := []struct {
		max   float64
		value int
	}{
		{2.0, 1}, // Very simple (0-2)
		{4.0, 2}, // Simple (2-4)
		{7.0, 3}, // Medium-simple (4-7)
		{12.0, 5}, // Medium-complex (7-12)
		{20.0, 8}, // Very complex (12+)
	}
	
	for _, threshold := range thresholds {
		if rawScore <= threshold.max {
			return threshold.value
		}
	}
	
	return 8 // Maximum complexity
}

// generateJustification creates a human-readable explanation of the complexity score
func (ce *ComplexityEstimator) generateJustification(task *RefinedTask, factorScores map[string]float64, finalScore int) string {
	var parts []string
	
	// Add task type justification
	parts = append(parts, fmt.Sprintf("Task type '%s' has base complexity", task.Type))
	
	// Find the most significant factors
	significantFactors := []string{}
	for factorName, score := range factorScores {
		if score > 1.0 { // Significant contribution
			switch factorName {
			case "task_type_complexity":
				significantFactors = append(significantFactors, "task type")
			case "implementation_scope":
				significantFactors = append(significantFactors, "implementation scope")
			case "dependency_complexity":
				significantFactors = append(significantFactors, "dependencies")
			case "technical_complexity":
				significantFactors = append(significantFactors, "technical requirements")
			case "integration_complexity":
				significantFactors = append(significantFactors, "integrations")
			case "testing_complexity":
				significantFactors = append(significantFactors, "testing requirements")
			case "uncertainty_factor":
				significantFactors = append(significantFactors, "unknowns")
			}
		}
	}
	
	if len(significantFactors) > 0 {
		parts = append(parts, fmt.Sprintf("Significant complexity from: %s", strings.Join(significantFactors, ", ")))
	}
	
	// Add dependency information
	if len(task.Dependencies) > 0 {
		parts = append(parts, fmt.Sprintf("%d dependencies require coordination", len(task.Dependencies)))
	}
	
	// Add final assessment
	assessments := map[int]string{
		1: "Very simple task with minimal complexity",
		2: "Simple task with basic requirements",
		3: "Moderate task requiring some expertise",
		5: "Complex task requiring significant expertise",
		8: "Very complex task requiring deep expertise and careful planning",
	}
	
	if assessment, exists := assessments[finalScore]; exists {
		parts = append(parts, assessment)
	}
	
	return strings.Join(parts, ". ") + "."
}

// GetFactorDescriptions returns descriptions of all complexity factors
func (ce *ComplexityEstimator) GetFactorDescriptions() map[string]string {
	descriptions := make(map[string]string)
	for _, factor := range ce.factors {
		descriptions[factor.Name] = factor.Description
	}
	return descriptions
}

// EstimateBatch estimates complexity for multiple tasks efficiently
func (ce *ComplexityEstimator) EstimateBatch(tasks []*RefinedTask) map[string]*ComplexityMetrics {
	results := make(map[string]*ComplexityMetrics)
	
	for _, task := range tasks {
		results[task.ID] = ce.EstimateWithMetrics(task)
	}
	
	return results
}

// ValidateComplexity validates that a complexity score is reasonable
func (ce *ComplexityEstimator) ValidateComplexity(task *RefinedTask, proposedComplexity int) (bool, string) {
	metrics := ce.EstimateWithMetrics(task)
	estimatedComplexity := metrics.NormalizedScore
	
	// Allow some variance but flag major discrepancies
	variance := int(math.Abs(float64(proposedComplexity - estimatedComplexity)))
	
	if variance <= 1 {
		return true, "Complexity score is reasonable"
	} else if variance == 2 {
		return true, fmt.Sprintf("Complexity score varies from estimate (%d vs %d) but within acceptable range", 
			proposedComplexity, estimatedComplexity)
	} else {
		return false, fmt.Sprintf("Complexity score significantly differs from estimate (%d vs %d). %s", 
			proposedComplexity, estimatedComplexity, metrics.Justification)
	}
}