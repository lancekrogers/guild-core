// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package monitoring

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/lancekrogers/guild-core/pkg/gerror"
)

// AlertType represents different types of alerts
type AlertType string

const (
	AlertTypeSLOViolation    AlertType = "slo_violation"
	AlertTypeHighLatency     AlertType = "high_latency"
	AlertTypeHighErrorRate   AlertType = "high_error_rate"
	AlertTypeLowCacheHitRate AlertType = "low_cache_hit_rate"
	AlertTypeHighMemoryUsage AlertType = "high_memory_usage"
	AlertTypeHighCPUUsage    AlertType = "high_cpu_usage"
	AlertTypeGoroutineLeak   AlertType = "goroutine_leak"
)

// AlertSeverity represents alert severity levels
type AlertSeverity string

const (
	AlertSeverityInfo     AlertSeverity = "info"
	AlertSeverityWarning  AlertSeverity = "warning"
	AlertSeverityCritical AlertSeverity = "critical"
)

// Alert represents a monitoring alert
type Alert struct {
	ID          string                 `json:"id"`
	Type        AlertType              `json:"type"`
	Severity    AlertSeverity          `json:"severity"`
	Title       string                 `json:"title"`
	Description string                 `json:"description"`
	Timestamp   time.Time              `json:"timestamp"`
	Data        map[string]interface{} `json:"data"`
	Resolved    bool                   `json:"resolved"`
	ResolvedAt  *time.Time             `json:"resolved_at,omitempty"`
}

// AlertManager manages alerts and notifications
type AlertManager struct {
	alerts    map[string]*Alert
	handlers  []AlertHandler
	mu        sync.RWMutex
	maxAlerts int
}

// AlertHandler interface for alert handling
type AlertHandler interface {
	HandleAlert(alert *Alert) error
	GetName() string
}

// NewAlertManager creates a new alert manager
func NewAlertManager() *AlertManager {
	return &AlertManager{
		alerts:    make(map[string]*Alert),
		handlers:  make([]AlertHandler, 0),
		maxAlerts: 1000,
	}
}

// TriggerAlert triggers a new alert
func (am *AlertManager) TriggerAlert(alertType AlertType, data interface{}) {
	am.mu.Lock()
	defer am.mu.Unlock()

	alert := &Alert{
		ID:          generateAlertID(),
		Type:        alertType,
		Severity:    am.determineSeverity(alertType),
		Title:       am.generateTitle(alertType),
		Description: am.generateDescription(alertType, data),
		Timestamp:   time.Now(),
		Data:        am.convertToMap(data),
		Resolved:    false,
	}

	// Store alert
	am.alerts[alert.ID] = alert

	// Clean up old alerts if needed
	if len(am.alerts) > am.maxAlerts {
		am.cleanupOldAlerts()
	}

	// Send to handlers
	for _, handler := range am.handlers {
		go func(h AlertHandler) {
			if err := h.HandleAlert(alert); err != nil {
				// Log error but don't fail the alert
			}
		}(handler)
	}
}

// ResolveAlert resolves an alert
func (am *AlertManager) ResolveAlert(alertID string) error {
	am.mu.Lock()
	defer am.mu.Unlock()

	alert, exists := am.alerts[alertID]
	if !exists {
		return gerror.New(gerror.ErrCodeNotFound, "alert not found", nil)
	}

	if !alert.Resolved {
		now := time.Now()
		alert.Resolved = true
		alert.ResolvedAt = &now
	}

	return nil
}

// AddHandler adds an alert handler
func (am *AlertManager) AddHandler(handler AlertHandler) {
	am.mu.Lock()
	defer am.mu.Unlock()
	am.handlers = append(am.handlers, handler)
}

// GetActiveAlerts returns currently active alerts
func (am *AlertManager) GetActiveAlerts() []*Alert {
	am.mu.RLock()
	defer am.mu.RUnlock()

	var active []*Alert
	for _, alert := range am.alerts {
		if !alert.Resolved {
			active = append(active, alert)
		}
	}

	return active
}

// GetAlertHistory returns alert history
func (am *AlertManager) GetAlertHistory(limit int) []*Alert {
	am.mu.RLock()
	defer am.mu.RUnlock()

	var alerts []*Alert
	for _, alert := range am.alerts {
		alerts = append(alerts, alert)
	}

	// Sort by timestamp (newest first)
	sort.Slice(alerts, func(i, j int) bool {
		return alerts[i].Timestamp.After(alerts[j].Timestamp)
	})

	if limit > 0 && len(alerts) > limit {
		alerts = alerts[:limit]
	}

	return alerts
}

// determineSeverity determines alert severity based on type
func (am *AlertManager) determineSeverity(alertType AlertType) AlertSeverity {
	switch alertType {
	case AlertTypeSLOViolation, AlertTypeHighLatency:
		return AlertSeverityCritical
	case AlertTypeHighErrorRate, AlertTypeHighMemoryUsage:
		return AlertSeverityWarning
	case AlertTypeLowCacheHitRate, AlertTypeHighCPUUsage:
		return AlertSeverityWarning
	case AlertTypeGoroutineLeak:
		return AlertSeverityWarning
	default:
		return AlertSeverityInfo
	}
}

// generateTitle generates alert title
func (am *AlertManager) generateTitle(alertType AlertType) string {
	switch alertType {
	case AlertTypeSLOViolation:
		return "SLO Violation Detected"
	case AlertTypeHighLatency:
		return "High Response Time Detected"
	case AlertTypeHighErrorRate:
		return "High Error Rate Detected"
	case AlertTypeLowCacheHitRate:
		return "Low Cache Hit Rate Detected"
	case AlertTypeHighMemoryUsage:
		return "High Memory Usage Detected"
	case AlertTypeHighCPUUsage:
		return "High CPU Usage Detected"
	case AlertTypeGoroutineLeak:
		return "Potential Goroutine Leak Detected"
	default:
		return "Performance Alert"
	}
}

// generateDescription generates alert description
func (am *AlertManager) generateDescription(alertType AlertType, data interface{}) string {
	dataMap := am.convertToMap(data)

	switch alertType {
	case AlertTypeSLOViolation:
		if slo, ok := dataMap["slo"].(string); ok {
			return fmt.Sprintf("SLO '%s' has been violated", slo)
		}
		return "A service level objective has been violated"

	case AlertTypeHighLatency:
		if latency, ok := dataMap["p95_response_time"].(time.Duration); ok {
			return fmt.Sprintf("P95 response time is %v, exceeding threshold", latency)
		}
		return "Response time is above acceptable threshold"

	case AlertTypeHighErrorRate:
		if rate, ok := dataMap["error_rate"].(float64); ok {
			return fmt.Sprintf("Error rate is %.2f%%, exceeding threshold", rate*100)
		}
		return "Error rate is above acceptable threshold"

	default:
		return "Performance metric is outside acceptable range"
	}
}

// convertToMap converts interface{} to map[string]interface{}
func (am *AlertManager) convertToMap(data interface{}) map[string]interface{} {
	if dataMap, ok := data.(map[string]interface{}); ok {
		return dataMap
	}

	// Convert other types to map
	result := make(map[string]interface{})
	result["data"] = data
	return result
}

// cleanupOldAlerts removes old resolved alerts
func (am *AlertManager) cleanupOldAlerts() {
	cutoff := time.Now().Add(-24 * time.Hour) // Keep alerts for 24 hours

	for id, alert := range am.alerts {
		if alert.Resolved && alert.ResolvedAt != nil && alert.ResolvedAt.Before(cutoff) {
			delete(am.alerts, id)
		}
	}
}

// generateAlertID generates a unique alert ID
func generateAlertID() string {
	return fmt.Sprintf("alert-%d", time.Now().UnixNano())
}

// ConsoleAlertHandler outputs alerts to console
type ConsoleAlertHandler struct {
	name string
}

// NewConsoleAlertHandler creates a new console alert handler
func NewConsoleAlertHandler() *ConsoleAlertHandler {
	return &ConsoleAlertHandler{
		name: "console",
	}
}

// HandleAlert handles an alert by printing to console
func (cah *ConsoleAlertHandler) HandleAlert(alert *Alert) error {
	severity := strings.ToUpper(string(alert.Severity))
	fmt.Printf("[%s] %s: %s - %s\n",
		severity,
		alert.Timestamp.Format("2006-01-02 15:04:05"),
		alert.Title,
		alert.Description)
	return nil
}

// GetName returns the handler name
func (cah *ConsoleAlertHandler) GetName() string {
	return cah.name
}

// EmailAlertHandler sends alerts via email (placeholder)
type EmailAlertHandler struct {
	name       string
	recipients []string
	smtpConfig *SMTPConfig
}

// SMTPConfig contains SMTP configuration
type SMTPConfig struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
	From     string `json:"from"`
}

// NewEmailAlertHandler creates a new email alert handler
func NewEmailAlertHandler(recipients []string, config *SMTPConfig) *EmailAlertHandler {
	return &EmailAlertHandler{
		name:       "email",
		recipients: recipients,
		smtpConfig: config,
	}
}

// HandleAlert handles an alert by sending email
func (eah *EmailAlertHandler) HandleAlert(alert *Alert) error {
	// This would implement actual email sending
	// For now, just log that we would send an email
	fmt.Printf("EMAIL ALERT: Would send email to %v about: %s\n",
		eah.recipients, alert.Title)
	return nil
}

// GetName returns the handler name
func (eah *EmailAlertHandler) GetName() string {
	return eah.name
}

// SlackAlertHandler sends alerts to Slack (placeholder)
type SlackAlertHandler struct {
	name       string
	webhookURL string
	channel    string
}

// NewSlackAlertHandler creates a new Slack alert handler
func NewSlackAlertHandler(webhookURL, channel string) *SlackAlertHandler {
	return &SlackAlertHandler{
		name:       "slack",
		webhookURL: webhookURL,
		channel:    channel,
	}
}

// HandleAlert handles an alert by sending to Slack
func (sah *SlackAlertHandler) HandleAlert(alert *Alert) error {
	// This would implement actual Slack webhook posting
	// For now, just log that we would send to Slack
	fmt.Printf("SLACK ALERT: Would send to %s: %s\n",
		sah.channel, alert.Title)
	return nil
}

// GetName returns the handler name
func (sah *SlackAlertHandler) GetName() string {
	return sah.name
}
