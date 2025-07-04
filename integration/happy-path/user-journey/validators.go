// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package user_journey

import (
	"context"
	"time"
)

// ResponseTimeValidator validates response time requirements
type ResponseTimeValidator struct{}

// Validate validates response time criteria
func (v *ResponseTimeValidator) Validate(ctx context.Context, criteria map[string]interface{}, actual interface{}) ValidationResult {
	stepResult, ok := actual.(*StepResult)
	if !ok {
		return ValidationResult{
			Type:    ValidationTypeResponseTime,
			Passed:  false,
			Details: map[string]interface{}{"error": "invalid step result type"},
		}
	}

	maxTime, exists := criteria["max_time"]
	if !exists {
		return ValidationResult{
			Type:    ValidationTypeResponseTime,
			Passed:  false,
			Details: map[string]interface{}{"error": "max_time criteria missing"},
		}
	}

	var maxDuration time.Duration
	switch v := maxTime.(type) {
	case int:
		maxDuration = time.Duration(v) * time.Millisecond
	case float64:
		maxDuration = time.Duration(v) * time.Millisecond
	case time.Duration:
		maxDuration = v
	default:
		return ValidationResult{
			Type:    ValidationTypeResponseTime,
			Passed:  false,
			Details: map[string]interface{}{"error": "invalid max_time type"},
		}
	}

	actualTime := stepResult.ActualTime
	passed := actualTime <= maxDuration

	return ValidationResult{
		Type:        ValidationTypeResponseTime,
		Passed:      passed,
		ActualValue: float64(actualTime.Milliseconds()),
		Threshold:   float64(maxDuration.Milliseconds()),
		Details: map[string]interface{}{
			"actual_time_ms":   actualTime.Milliseconds(),
			"max_time_ms":      maxDuration.Milliseconds(),
			"performance_ratio": float64(actualTime) / float64(maxDuration),
		},
	}
}

// AccuracyValidator validates accuracy requirements
type AccuracyValidator struct{}

// Validate validates accuracy criteria
func (v *AccuracyValidator) Validate(ctx context.Context, criteria map[string]interface{}, actual interface{}) ValidationResult {
	stepResult, ok := actual.(*StepResult)
	if !ok {
		return ValidationResult{
			Type:    ValidationTypeAccuracy,
			Passed:  false,
			Details: map[string]interface{}{"error": "invalid step result type"},
		}
	}

	threshold, exists := criteria["threshold"]
	if !exists {
		threshold = 0.95 // Default 95% accuracy threshold
	}

	thresholdValue, ok := threshold.(float64)
	if !ok {
		if intVal, ok := threshold.(int); ok {
			thresholdValue = float64(intVal)
		} else {
			return ValidationResult{
				Type:    ValidationTypeAccuracy,
				Passed:  false,
				Details: map[string]interface{}{"error": "invalid threshold type"},
			}
		}
	}

	// Calculate accuracy based on successful actions
	totalActions := len(stepResult.UserActions)
	successfulActions := 0
	
	for _, action := range stepResult.UserActions {
		if action.Success {
			successfulActions++
		}
	}

	var actualAccuracy float64
	if totalActions > 0 {
		actualAccuracy = float64(successfulActions) / float64(totalActions)
	} else {
		actualAccuracy = 1.0 // No actions = perfect accuracy
	}

	passed := actualAccuracy >= thresholdValue

	return ValidationResult{
		Type:        ValidationTypeAccuracy,
		Passed:      passed,
		ActualValue: actualAccuracy,
		Threshold:   thresholdValue,
		Details: map[string]interface{}{
			"successful_actions": successfulActions,
			"total_actions":      totalActions,
			"accuracy_ratio":     actualAccuracy,
		},
	}
}

// UserExperienceValidator validates user experience requirements
type UserExperienceValidator struct{}

// Validate validates user experience criteria
func (v *UserExperienceValidator) Validate(ctx context.Context, criteria map[string]interface{}, actual interface{}) ValidationResult {
	stepResult, ok := actual.(*StepResult)
	if !ok {
		return ValidationResult{
			Type:    ValidationTypeUserExperience,
			Passed:  false,
			Details: map[string]interface{}{"error": "invalid step result type"},
		}
	}

	threshold, exists := criteria["threshold"]
	if !exists {
		threshold = 0.8 // Default 80% user experience threshold
	}

	thresholdValue, ok := threshold.(float64)
	if !ok {
		if intVal, ok := threshold.(int); ok {
			thresholdValue = float64(intVal)
		} else {
			return ValidationResult{
				Type:    ValidationTypeUserExperience,
				Passed:  false,
				Details: map[string]interface{}{"error": "invalid threshold type"},
			}
		}
	}

	// Calculate user experience score based on multiple factors
	var uxScore float64

	// Factor 1: Step success (40% weight)
	if stepResult.Success {
		uxScore += 0.4
	}

	// Factor 2: Performance (30% weight)
	if stepResult.ActualTime <= stepResult.TargetTime {
		uxScore += 0.3
	} else {
		// Partial credit based on how close to target
		ratio := float64(stepResult.TargetTime) / float64(stepResult.ActualTime)
		if ratio > 0.5 { // If within 2x target time
			uxScore += 0.3 * ratio
		}
	}

	// Factor 3: Error count (20% weight)
	errorCount := len(stepResult.Issues)
	if errorCount == 0 {
		uxScore += 0.2
	} else if errorCount <= 2 {
		uxScore += 0.1 // Partial credit for few errors
	}

	// Factor 4: Action quality (10% weight)
	totalActions := len(stepResult.UserActions)
	if totalActions > 0 {
		relevanceSum := 0.0
		for _, action := range stepResult.UserActions {
			if relevance, exists := action.Metrics["relevance_score"]; exists {
				relevanceSum += relevance
			} else {
				relevanceSum += 0.8 // Default relevance if not measured
			}
		}
		avgRelevance := relevanceSum / float64(totalActions)
		uxScore += 0.1 * avgRelevance
	} else {
		uxScore += 0.1 // Full credit if no actions required
	}

	passed := uxScore >= thresholdValue

	return ValidationResult{
		Type:        ValidationTypeUserExperience,
		Passed:      passed,
		ActualValue: uxScore,
		Threshold:   thresholdValue,
		Details: map[string]interface{}{
			"ux_score":         uxScore,
			"step_success":     stepResult.Success,
			"performance_ratio": float64(stepResult.TargetTime) / float64(stepResult.ActualTime),
			"error_count":      len(stepResult.Issues),
			"action_count":     len(stepResult.UserActions),
		},
	}
}

// CompletenessValidator validates completeness requirements
type CompletenessValidator struct{}

// Validate validates completeness criteria
func (v *CompletenessValidator) Validate(ctx context.Context, criteria map[string]interface{}, actual interface{}) ValidationResult {
	stepResult, ok := actual.(*StepResult)
	if !ok {
		return ValidationResult{
			Type:    ValidationTypeCompleteness,
			Passed:  false,
			Details: map[string]interface{}{"error": "invalid step result type"},
		}
	}

	threshold, exists := criteria["threshold"]
	if !exists {
		threshold = 0.9 // Default 90% completeness threshold
	}

	thresholdValue, ok := threshold.(float64)
	if !ok {
		return ValidationResult{
			Type:    ValidationTypeCompleteness,
			Passed:  false,
			Details: map[string]interface{}{"error": "invalid threshold type"},
		}
	}

	// Calculate completeness based on required actions completion
	requiredActions, exists := criteria["required_actions"]
	if !exists {
		// If no required actions specified, use all actions
		requiredActions = len(stepResult.UserActions)
	}

	var requiredCount int
	switch v := requiredActions.(type) {
	case int:
		requiredCount = v
	case float64:
		requiredCount = int(v)
	default:
		requiredCount = len(stepResult.UserActions)
	}

	completedActions := 0
	for _, action := range stepResult.UserActions {
		if action.Success {
			completedActions++
		}
	}

	var completeness float64
	if requiredCount > 0 {
		completeness = float64(completedActions) / float64(requiredCount)
	} else {
		completeness = 1.0
	}

	passed := completeness >= thresholdValue

	return ValidationResult{
		Type:        ValidationTypeCompleteness,
		Passed:      passed,
		ActualValue: completeness,
		Threshold:   thresholdValue,
		Details: map[string]interface{}{
			"completed_actions": completedActions,
			"required_actions":  requiredCount,
			"completeness_ratio": completeness,
		},
	}
}

// SystemHealthValidator validates system health requirements
type SystemHealthValidator struct{}

// Validate validates system health criteria
func (v *SystemHealthValidator) Validate(ctx context.Context, criteria map[string]interface{}, actual interface{}) ValidationResult {
	stepResult, ok := actual.(*StepResult)
	if !ok {
		return ValidationResult{
			Type:    ValidationTypeSystemHealth,
			Passed:  false,
			Details: map[string]interface{}{"error": "invalid step result type"},
		}
	}

	// System health is considered good if:
	// 1. No critical issues
	// 2. Response times are reasonable
	// 3. Error rates are low

	criticalIssues := 0
	highSeverityIssues := 0
	totalIssues := len(stepResult.Issues)

	for _, issue := range stepResult.Issues {
		switch issue.Severity {
		case IssueSeverityCritical:
			criticalIssues++
		case IssueSeverityHigh:
			highSeverityIssues++
		}
	}

	// Calculate health score
	healthScore := 1.0

	// Deduct for critical issues
	if criticalIssues > 0 {
		healthScore -= 0.5 // Critical issues severely impact health
	}

	// Deduct for high severity issues
	if highSeverityIssues > 0 {
		healthScore -= 0.2 * float64(highSeverityIssues)
	}

	// Deduct for performance issues
	if stepResult.ActualTime > stepResult.TargetTime*2 {
		healthScore -= 0.2 // Severe performance impact
	} else if stepResult.ActualTime > stepResult.TargetTime {
		healthScore -= 0.1 // Moderate performance impact
	}

	// Ensure health score doesn't go below 0
	if healthScore < 0 {
		healthScore = 0
	}

	threshold := 0.8 // Default 80% health threshold
	if thresholdVal, exists := criteria["threshold"]; exists {
		if thresholdFloat, ok := thresholdVal.(float64); ok {
			threshold = thresholdFloat
		}
	}

	passed := healthScore >= threshold

	return ValidationResult{
		Type:        ValidationTypeSystemHealth,
		Passed:      passed,
		ActualValue: healthScore,
		Threshold:   threshold,
		Details: map[string]interface{}{
			"health_score":         healthScore,
			"critical_issues":      criticalIssues,
			"high_severity_issues": highSeverityIssues,
			"total_issues":         totalIssues,
			"performance_impact":   float64(stepResult.ActualTime) / float64(stepResult.TargetTime),
		},
	}
}