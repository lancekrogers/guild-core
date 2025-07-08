// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package monitoring

import (
	"sync"
	"time"
)

// Comparator represents comparison operators for SLOs
type Comparator string

const (
	ComparatorLessThan    Comparator = "lt"
	ComparatorGreaterThan Comparator = "gt"
	ComparatorEquals      Comparator = "eq"
)

// Compare performs the comparison
func (c Comparator) Compare(actual, target float64) bool {
	switch c {
	case ComparatorLessThan:
		return actual < target
	case ComparatorGreaterThan:
		return actual > target
	case ComparatorEquals:
		return actual == target
	default:
		return false
	}
}

// SLO represents a Service Level Objective
type SLO struct {
	Name        string        `json:"name"`
	Target      float64       `json:"target"`
	Window      time.Duration `json:"window"`
	Metric      string        `json:"metric"`
	Comparator  Comparator    `json:"comparator"`
	Description string        `json:"description"`
	ErrorBudget float64       `json:"error_budget"`
}

// SLOViolation represents a violation of an SLO
type SLOViolation struct {
	SLO         SLO           `json:"slo"`
	ActualValue float64       `json:"actual_value"`
	Timestamp   time.Time     `json:"timestamp"`
	ErrorBudget float64       `json:"error_budget"`
	Severity    string        `json:"severity"`
	Duration    time.Duration `json:"duration"`
}

// SLOMonitor monitors Service Level Objectives
type SLOMonitor struct {
	objectives    []SLO
	violations    []SLOViolation
	mu            sync.RWMutex
	maxViolations int
}

// NewSLOMonitor creates a new SLO monitor
func NewSLOMonitor() *SLOMonitor {
	return &SLOMonitor{
		objectives:    make([]SLO, 0),
		violations:    make([]SLOViolation, 0),
		maxViolations: 1000,
	}
}

// AddSLO adds a new SLO to monitor
func (sm *SLOMonitor) AddSLO(slo SLO) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.objectives = append(sm.objectives, slo)
}

// RemoveSLO removes an SLO by name
func (sm *SLOMonitor) RemoveSLO(name string) bool {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	for i, slo := range sm.objectives {
		if slo.Name == name {
			sm.objectives = append(sm.objectives[:i], sm.objectives[i+1:]...)
			return true
		}
	}
	return false
}

// CheckSLOs checks all SLOs against current metrics
func (sm *SLOMonitor) CheckSLOs(metrics *MetricsCollector) []SLOViolation {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	var violations []SLOViolation

	for _, slo := range sm.objectives {
		value := metrics.GetMetricValue(slo.Metric, slo.Window)

		if !slo.Comparator.Compare(value, slo.Target) {
			violation := SLOViolation{
				SLO:         slo,
				ActualValue: value,
				Timestamp:   time.Now(),
				ErrorBudget: sm.calculateErrorBudget(slo, value),
				Severity:    sm.calculateSeverity(slo, value),
			}

			violations = append(violations, violation)
			sm.recordViolation(violation)
		}
	}

	return violations
}

// GetSLOs returns all configured SLOs
func (sm *SLOMonitor) GetSLOs() []SLO {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	slos := make([]SLO, len(sm.objectives))
	copy(slos, sm.objectives)
	return slos
}

// GetViolations returns recent SLO violations
func (sm *SLOMonitor) GetViolations(limit int) []SLOViolation {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	violations := make([]SLOViolation, len(sm.violations))
	copy(violations, sm.violations)

	if limit > 0 && len(violations) > limit {
		violations = violations[:limit]
	}

	return violations
}

// GetSLOStatus returns the current status of all SLOs
func (sm *SLOMonitor) GetSLOStatus(metrics *MetricsCollector) []SLOStatus {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	var statuses []SLOStatus

	for _, slo := range sm.objectives {
		value := metrics.GetMetricValue(slo.Metric, slo.Window)

		status := SLOStatus{
			SLO:          slo,
			CurrentValue: value,
			Status:       sm.getSLOStatusString(slo, value),
			ErrorBudget:  sm.calculateErrorBudget(slo, value),
			LastChecked:  time.Now(),
		}

		statuses = append(statuses, status)
	}

	return statuses
}

// SLOStatus represents the current status of an SLO
type SLOStatus struct {
	SLO          SLO       `json:"slo"`
	CurrentValue float64   `json:"current_value"`
	Status       string    `json:"status"`
	ErrorBudget  float64   `json:"error_budget"`
	LastChecked  time.Time `json:"last_checked"`
}

// recordViolation records a new SLO violation
func (sm *SLOMonitor) recordViolation(violation SLOViolation) {
	sm.violations = append(sm.violations, violation)

	// Keep only recent violations
	if len(sm.violations) > sm.maxViolations {
		sm.violations = sm.violations[len(sm.violations)-sm.maxViolations:]
	}
}

// calculateErrorBudget calculates the error budget consumption
func (sm *SLOMonitor) calculateErrorBudget(slo SLO, actualValue float64) float64 {
	switch slo.Comparator {
	case ComparatorLessThan:
		if actualValue > slo.Target {
			return (actualValue - slo.Target) / slo.Target * 100
		}
		return 0
	case ComparatorGreaterThan:
		if actualValue < slo.Target {
			return (slo.Target - actualValue) / slo.Target * 100
		}
		return 0
	default:
		return 0
	}
}

// calculateSeverity calculates the severity of an SLO violation
func (sm *SLOMonitor) calculateSeverity(slo SLO, actualValue float64) string {
	errorBudget := sm.calculateErrorBudget(slo, actualValue)

	switch {
	case errorBudget > 50:
		return "critical"
	case errorBudget > 20:
		return "high"
	case errorBudget > 5:
		return "medium"
	default:
		return "low"
	}
}

// getSLOStatusString returns a string representation of SLO status
func (sm *SLOMonitor) getSLOStatusString(slo SLO, value float64) string {
	if slo.Comparator.Compare(value, slo.Target) {
		return "healthy"
	}

	errorBudget := sm.calculateErrorBudget(slo, value)
	switch {
	case errorBudget > 50:
		return "critical"
	case errorBudget > 20:
		return "warning"
	default:
		return "degraded"
	}
}

// ErrorBudgetCalculator calculates error budgets for SLOs
type ErrorBudgetCalculator struct {
	mu sync.RWMutex
}

// NewErrorBudgetCalculator creates a new error budget calculator
func NewErrorBudgetCalculator() *ErrorBudgetCalculator {
	return &ErrorBudgetCalculator{}
}

// CalculateErrorBudget calculates the error budget for an SLO
func (ebc *ErrorBudgetCalculator) CalculateErrorBudget(slo SLO, actualValue float64, window time.Duration) *ErrorBudget {
	ebc.mu.RLock()
	defer ebc.mu.RUnlock()

	budget := &ErrorBudget{
		SLOName:   slo.Name,
		Target:    slo.Target,
		Actual:    actualValue,
		Window:    window,
		Timestamp: time.Now(),
	}

	// Calculate budget consumption
	switch slo.Comparator {
	case ComparatorLessThan:
		if actualValue > slo.Target {
			budget.Consumed = (actualValue - slo.Target) / slo.Target
			budget.Remaining = 1.0 - budget.Consumed
		} else {
			budget.Consumed = 0
			budget.Remaining = 1.0
		}
	case ComparatorGreaterThan:
		if actualValue < slo.Target {
			budget.Consumed = (slo.Target - actualValue) / slo.Target
			budget.Remaining = 1.0 - budget.Consumed
		} else {
			budget.Consumed = 0
			budget.Remaining = 1.0
		}
	}

	// Ensure values are in valid range
	if budget.Consumed < 0 {
		budget.Consumed = 0
	}
	if budget.Remaining < 0 {
		budget.Remaining = 0
	}
	if budget.Remaining > 1.0 {
		budget.Remaining = 1.0
	}

	return budget
}

// ErrorBudget represents error budget information
type ErrorBudget struct {
	SLOName   string        `json:"slo_name"`
	Target    float64       `json:"target"`
	Actual    float64       `json:"actual"`
	Consumed  float64       `json:"consumed"`
	Remaining float64       `json:"remaining"`
	Window    time.Duration `json:"window"`
	Timestamp time.Time     `json:"timestamp"`
}

// SLOReporter generates SLO reports
type SLOReporter struct {
	monitor *SLOMonitor
	mu      sync.RWMutex
}

// NewSLOReporter creates a new SLO reporter
func NewSLOReporter(monitor *SLOMonitor) *SLOReporter {
	return &SLOReporter{
		monitor: monitor,
	}
}

// GenerateReport generates a comprehensive SLO report
func (sr *SLOReporter) GenerateReport(metrics *MetricsCollector) *SLOReport {
	sr.mu.RLock()
	defer sr.mu.RUnlock()

	report := &SLOReport{
		Timestamp:   time.Now(),
		SLOStatuses: sr.monitor.GetSLOStatus(metrics),
		Violations:  sr.monitor.GetViolations(50), // Last 50 violations
	}

	// Calculate summary statistics
	report.Summary = sr.calculateSummary(report.SLOStatuses)

	return report
}

// SLOReport contains comprehensive SLO information
type SLOReport struct {
	Timestamp   time.Time      `json:"timestamp"`
	SLOStatuses []SLOStatus    `json:"slo_statuses"`
	Violations  []SLOViolation `json:"violations"`
	Summary     *SLOSummary    `json:"summary"`
}

// SLOSummary contains summary statistics for SLOs
type SLOSummary struct {
	TotalSLOs       int     `json:"total_slos"`
	HealthySLOs     int     `json:"healthy_slos"`
	DegradedSLOs    int     `json:"degraded_slos"`
	CriticalSLOs    int     `json:"critical_slos"`
	OverallHealth   float64 `json:"overall_health"`
	ViolationsToday int     `json:"violations_today"`
}

// calculateSummary calculates summary statistics
func (sr *SLOReporter) calculateSummary(statuses []SLOStatus) *SLOSummary {
	summary := &SLOSummary{
		TotalSLOs: len(statuses),
	}

	for _, status := range statuses {
		switch status.Status {
		case "healthy":
			summary.HealthySLOs++
		case "degraded", "warning":
			summary.DegradedSLOs++
		case "critical":
			summary.CriticalSLOs++
		}
	}

	// Calculate overall health percentage
	if summary.TotalSLOs > 0 {
		summary.OverallHealth = float64(summary.HealthySLOs) / float64(summary.TotalSLOs) * 100
	}

	// Count violations in the last 24 hours
	cutoff := time.Now().Add(-24 * time.Hour)
	violations := sr.monitor.GetViolations(0) // Get all violations
	for _, violation := range violations {
		if violation.Timestamp.After(cutoff) {
			summary.ViolationsToday++
		}
	}

	return summary
}
