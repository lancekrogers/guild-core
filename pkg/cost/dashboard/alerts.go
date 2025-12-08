// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package dashboard

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/guild-framework/guild-core/pkg/gerror"
	"github.com/guild-framework/guild-core/pkg/observability"
)

// AlertManager manages cost-related alerts and notifications
type AlertManager struct {
	rules    []AlertRule
	history  *AlertHistory
	notifier *Notifier
	config   *AlertConfig
}

// Alert represents a cost alert
type Alert struct {
	ID         string                 `json:"id"`
	Type       AlertType              `json:"type"`
	Severity   string                 `json:"severity"`
	Message    string                 `json:"message"`
	Timestamp  time.Time              `json:"timestamp"`
	Data       map[string]interface{} `json:"data,omitempty"`
	Resolved   bool                   `json:"resolved"`
	ResolvedAt *time.Time             `json:"resolved_at,omitempty"`
}

// AlertType defines types of alerts
type AlertType string

const (
	AlertTypeBudget   AlertType = "budget"
	AlertTypeAnomaly  AlertType = "anomaly"
	AlertTypeSpike    AlertType = "spike"
	AlertTypeUsage    AlertType = "usage"
	AlertTypeProvider AlertType = "provider"
)

// AlertRule defines the interface for alert rules
type AlertRule interface {
	Evaluate(ctx context.Context, metrics Metrics) (*Alert, error)
	GetType() AlertType
	GetName() string
}

// Metrics contains metrics for alert evaluation
type Metrics struct {
	TotalCost      float64            `json:"total_cost"`
	BudgetLimit    float64            `json:"budget_limit"`
	HourlyRate     float64            `json:"hourly_rate"`
	CostByAgent    map[string]float64 `json:"cost_by_agent"`
	CostByProvider map[string]float64 `json:"cost_by_provider"`
	Period         time.Duration      `json:"period"`
	Timestamp      time.Time          `json:"timestamp"`
}

// AlertConfig contains alert manager configuration
type AlertConfig struct {
	MaxActiveAlerts     int           `json:"max_active_alerts"`
	AlertRetention      time.Duration `json:"alert_retention"`
	NotificationDelay   time.Duration `json:"notification_delay"`
	EnableNotifications bool          `json:"enable_notifications"`
}

// NewAlertManager creates a new alert manager
func NewAlertManager(ctx context.Context) (*AlertManager, error) {
	ctx = observability.WithComponent(ctx, "cost.alert_manager")
	ctx = observability.WithOperation(ctx, "NewAlertManager")

	config := &AlertConfig{
		MaxActiveAlerts:     50,
		AlertRetention:      7 * 24 * time.Hour, // 7 days
		NotificationDelay:   time.Minute,
		EnableNotifications: true,
	}

	history, err := NewAlertHistory(ctx)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create alert history").
			WithComponent("cost.alert_manager").
			WithOperation("NewAlertManager")
	}

	notifier, err := NewNotifier(ctx)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create notifier").
			WithComponent("cost.alert_manager").
			WithOperation("NewAlertManager")
	}

	manager := &AlertManager{
		history:  history,
		notifier: notifier,
		config:   config,
	}

	// Register default alert rules
	manager.registerDefaultRules(ctx)

	return manager, nil
}

// registerDefaultRules registers default alert rules
func (am *AlertManager) registerDefaultRules(ctx context.Context) {
	am.rules = []AlertRule{
		NewBudgetAlertRule(80.0, "high"),     // 80% budget usage
		NewBudgetAlertRule(95.0, "critical"), // 95% budget usage
		NewAnomalyAlertRule(2.0),             // 2x normal usage
		NewSpikeAlertRule(5.0, time.Hour),    // 5x normal in 1 hour
		NewUsageAlertRule(1000.0, "daily"),   // $1000 daily limit
	}
}

// EvaluateAlerts evaluates all alert rules against current metrics
func (am *AlertManager) EvaluateAlerts(ctx context.Context, metrics Metrics) ([]Alert, error) {
	ctx = observability.WithComponent(ctx, "cost.alert_manager")
	ctx = observability.WithOperation(ctx, "EvaluateAlerts")

	var alerts []Alert

	for _, rule := range am.rules {
		alert, err := rule.Evaluate(ctx, metrics)
		if err != nil {
			// Log error but continue with other rules
			continue
		}

		if alert != nil {
			// Store alert in history
			if err := am.history.Store(ctx, *alert); err != nil {
				// Log error but continue
				continue
			}

			alerts = append(alerts, *alert)

			// Send notification if enabled
			if am.config.EnableNotifications {
				go am.notifier.Send(ctx, *alert)
			}
		}
	}

	return alerts, nil
}

// GetActive returns active (unresolved) alerts
func (am *AlertManager) GetActive(ctx context.Context) ([]Alert, error) {
	ctx = observability.WithComponent(ctx, "cost.alert_manager")
	ctx = observability.WithOperation(ctx, "GetActive")

	return am.history.GetActive(ctx, am.config.MaxActiveAlerts)
}

// ResolveAlert resolves an alert by ID
func (am *AlertManager) ResolveAlert(ctx context.Context, alertID string) error {
	ctx = observability.WithComponent(ctx, "cost.alert_manager")
	ctx = observability.WithOperation(ctx, "ResolveAlert")

	return am.history.Resolve(ctx, alertID)
}

// BudgetAlertRule checks for budget threshold violations
type BudgetAlertRule struct {
	threshold float64 // percentage
	severity  string
}

// NewBudgetAlertRule creates a new budget alert rule
func NewBudgetAlertRule(threshold float64, severity string) *BudgetAlertRule {
	return &BudgetAlertRule{
		threshold: threshold,
		severity:  severity,
	}
}

// GetType returns the alert type
func (bar *BudgetAlertRule) GetType() AlertType {
	return AlertTypeBudget
}

// GetName returns the rule name
func (bar *BudgetAlertRule) GetName() string {
	return fmt.Sprintf("budget_threshold_%.0f", bar.threshold)
}

// Evaluate evaluates the budget alert rule
func (bar *BudgetAlertRule) Evaluate(ctx context.Context, metrics Metrics) (*Alert, error) {
	if metrics.BudgetLimit <= 0 {
		return nil, nil // No budget configured
	}

	budgetUsed := (metrics.TotalCost / metrics.BudgetLimit) * 100

	if budgetUsed > bar.threshold {
		return &Alert{
			ID:       generateAlertID(),
			Type:     AlertTypeBudget,
			Severity: bar.severity,
			Message: fmt.Sprintf("Budget usage at %.1f%% (threshold: %.0f%%)",
				budgetUsed, bar.threshold),
			Timestamp: time.Now(),
			Data: map[string]interface{}{
				"budget_used":  budgetUsed,
				"threshold":    bar.threshold,
				"total_cost":   metrics.TotalCost,
				"budget_limit": metrics.BudgetLimit,
			},
		}, nil
	}

	return nil, nil
}

// AnomalyAlertRule detects cost anomalies
type AnomalyAlertRule struct {
	detector  *AnomalyDetector
	threshold float64 // multiplier for normal usage
}

// NewAnomalyAlertRule creates a new anomaly alert rule
func NewAnomalyAlertRule(threshold float64) *AnomalyAlertRule {
	return &AnomalyAlertRule{
		detector:  NewAnomalyDetector(),
		threshold: threshold,
	}
}

// GetType returns the alert type
func (aar *AnomalyAlertRule) GetType() AlertType {
	return AlertTypeAnomaly
}

// GetName returns the rule name
func (aar *AnomalyAlertRule) GetName() string {
	return fmt.Sprintf("anomaly_threshold_%.1fx", aar.threshold)
}

// Evaluate evaluates the anomaly alert rule
func (aar *AnomalyAlertRule) Evaluate(ctx context.Context, metrics Metrics) (*Alert, error) {
	anomaly := aar.detector.Detect(ctx, metrics)
	if anomaly != nil && anomaly.Magnitude >= aar.threshold {
		return &Alert{
			ID:       generateAlertID(),
			Type:     AlertTypeAnomaly,
			Severity: aar.getSeverity(anomaly.Magnitude),
			Message: fmt.Sprintf("Cost anomaly detected: %s (%.1fx normal)",
				anomaly.Description, anomaly.Magnitude),
			Timestamp: time.Now(),
			Data: map[string]interface{}{
				"anomaly":   anomaly,
				"magnitude": anomaly.Magnitude,
				"threshold": aar.threshold,
			},
		}, nil
	}

	return nil, nil
}

// getSeverity determines severity based on anomaly magnitude
func (aar *AnomalyAlertRule) getSeverity(magnitude float64) string {
	if magnitude >= 5.0 {
		return "critical"
	} else if magnitude >= 3.0 {
		return "high"
	} else if magnitude >= 2.0 {
		return "medium"
	}
	return "low"
}

// SpikeAlertRule detects cost spikes in a time window
type SpikeAlertRule struct {
	threshold float64       // multiplier for normal usage
	window    time.Duration // time window to check
}

// NewSpikeAlertRule creates a new spike alert rule
func NewSpikeAlertRule(threshold float64, window time.Duration) *SpikeAlertRule {
	return &SpikeAlertRule{
		threshold: threshold,
		window:    window,
	}
}

// GetType returns the alert type
func (sar *SpikeAlertRule) GetType() AlertType {
	return AlertTypeSpike
}

// GetName returns the rule name
func (sar *SpikeAlertRule) GetName() string {
	return fmt.Sprintf("spike_threshold_%.1fx_%s", sar.threshold, sar.window.String())
}

// Evaluate evaluates the spike alert rule
func (sar *SpikeAlertRule) Evaluate(ctx context.Context, metrics Metrics) (*Alert, error) {
	// Calculate expected cost for the time window
	expectedHourlyRate := 10.0 // Default expected hourly rate
	expectedCost := expectedHourlyRate * metrics.Period.Hours()

	if metrics.TotalCost >= expectedCost*sar.threshold {
		return &Alert{
			ID:       generateAlertID(),
			Type:     AlertTypeSpike,
			Severity: "high",
			Message: fmt.Sprintf("Cost spike detected: $%.2f in %s (%.1fx expected)",
				metrics.TotalCost, metrics.Period.String(), metrics.TotalCost/expectedCost),
			Timestamp: time.Now(),
			Data: map[string]interface{}{
				"actual_cost":   metrics.TotalCost,
				"expected_cost": expectedCost,
				"multiplier":    metrics.TotalCost / expectedCost,
				"window":        sar.window.String(),
			},
		}, nil
	}

	return nil, nil
}

// UsageAlertRule checks for absolute usage limits
type UsageAlertRule struct {
	limit  float64 // absolute cost limit
	period string  // "hourly", "daily", "monthly"
}

// NewUsageAlertRule creates a new usage alert rule
func NewUsageAlertRule(limit float64, period string) *UsageAlertRule {
	return &UsageAlertRule{
		limit:  limit,
		period: period,
	}
}

// GetType returns the alert type
func (uar *UsageAlertRule) GetType() AlertType {
	return AlertTypeUsage
}

// GetName returns the rule name
func (uar *UsageAlertRule) GetName() string {
	return fmt.Sprintf("usage_limit_%.0f_%s", uar.limit, uar.period)
}

// Evaluate evaluates the usage alert rule
func (uar *UsageAlertRule) Evaluate(ctx context.Context, metrics Metrics) (*Alert, error) {
	// Convert current cost to the period rate
	var periodCost float64
	var periodName string

	switch uar.period {
	case "hourly":
		periodCost = metrics.HourlyRate
		periodName = "hour"
	case "daily":
		periodCost = metrics.HourlyRate * 24
		periodName = "day"
	case "monthly":
		periodCost = metrics.HourlyRate * 24 * 30
		periodName = "month"
	default:
		periodCost = metrics.TotalCost
		periodName = "period"
	}

	if periodCost >= uar.limit {
		return &Alert{
			ID:       generateAlertID(),
			Type:     AlertTypeUsage,
			Severity: "high",
			Message: fmt.Sprintf("Usage limit exceeded: $%.2f per %s (limit: $%.2f)",
				periodCost, periodName, uar.limit),
			Timestamp: time.Now(),
			Data: map[string]interface{}{
				"period_cost": periodCost,
				"limit":       uar.limit,
				"period":      uar.period,
			},
		}, nil
	}

	return nil, nil
}

// AnomalyDetector detects cost anomalies
type AnomalyDetector struct {
	baseline *CostBaseline
}

// Anomaly represents a detected cost anomaly
type Anomaly struct {
	Description string  `json:"description"`
	Magnitude   float64 `json:"magnitude"`
	Type        string  `json:"type"`
	Confidence  float64 `json:"confidence"`
}

// CostBaseline maintains baseline cost information
type CostBaseline struct {
	AverageHourlyRate float64   `json:"average_hourly_rate"`
	StandardDeviation float64   `json:"standard_deviation"`
	LastUpdated       time.Time `json:"last_updated"`
}

// NewAnomalyDetector creates a new anomaly detector
func NewAnomalyDetector() *AnomalyDetector {
	return &AnomalyDetector{
		baseline: &CostBaseline{
			AverageHourlyRate: 10.0, // Default baseline
			StandardDeviation: 2.0,  // Default deviation
			LastUpdated:       time.Now(),
		},
	}
}

// Detect detects anomalies in cost metrics
func (ad *AnomalyDetector) Detect(ctx context.Context, metrics Metrics) *Anomaly {
	// Simple anomaly detection based on standard deviation
	if metrics.HourlyRate > ad.baseline.AverageHourlyRate+2*ad.baseline.StandardDeviation {
		magnitude := metrics.HourlyRate / ad.baseline.AverageHourlyRate
		return &Anomaly{
			Description: "Hourly cost significantly above baseline",
			Magnitude:   magnitude,
			Type:        "rate_spike",
			Confidence:  math.Min(magnitude/2.0, 1.0),
		}
	}

	return nil
}

// AlertHistory manages alert history storage
type AlertHistory struct {
	// In production, this would use database storage
	alerts []Alert
}

// NewAlertHistory creates a new alert history
func NewAlertHistory(ctx context.Context) (*AlertHistory, error) {
	return &AlertHistory{
		alerts: make([]Alert, 0),
	}, nil
}

// Store stores an alert in history
func (ah *AlertHistory) Store(ctx context.Context, alert Alert) error {
	ah.alerts = append(ah.alerts, alert)
	return nil
}

// GetActive returns active alerts
func (ah *AlertHistory) GetActive(ctx context.Context, limit int) ([]Alert, error) {
	var active []Alert

	for _, alert := range ah.alerts {
		if !alert.Resolved {
			active = append(active, alert)
		}

		if len(active) >= limit {
			break
		}
	}

	return active, nil
}

// Resolve resolves an alert
func (ah *AlertHistory) Resolve(ctx context.Context, alertID string) error {
	for i := range ah.alerts {
		if ah.alerts[i].ID == alertID {
			now := time.Now()
			ah.alerts[i].Resolved = true
			ah.alerts[i].ResolvedAt = &now
			break
		}
	}
	return nil
}

// Notifier sends alert notifications
type Notifier struct {
	// In production, this would integrate with notification services
}

// NewNotifier creates a new notifier
func NewNotifier(ctx context.Context) (*Notifier, error) {
	return &Notifier{}, nil
}

// Send sends an alert notification
func (n *Notifier) Send(ctx context.Context, alert Alert) error {
	// In production, this would send notifications via email, Slack, etc.
	// For now, just log the alert
	return nil
}

// generateAlertID generates a unique alert ID
func generateAlertID() string {
	return fmt.Sprintf("alert_%d", time.Now().UnixNano())
}

// MetricsCollector collects metrics for dashboard
type MetricsCollector struct {
	// In production, this would collect metrics from various sources
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector(ctx context.Context) (*MetricsCollector, error) {
	return &MetricsCollector{}, nil
}
