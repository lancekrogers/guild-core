// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package parser

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// AlertSeverity represents the severity of an alert
type AlertSeverity string

const (
	AlertSeverityInfo     AlertSeverity = "info"
	AlertSeverityWarning  AlertSeverity = "warning"
	AlertSeverityCritical AlertSeverity = "critical"
)

// Alert represents a parser system alert
type Alert struct {
	ID          string                 `json:"id"`
	Severity    AlertSeverity          `json:"severity"`
	Title       string                 `json:"title"`
	Description string                 `json:"description"`
	Timestamp   time.Time              `json:"timestamp"`
	Source      string                 `json:"source"`
	Labels      map[string]string      `json:"labels"`
	Metadata    map[string]interface{} `json:"metadata"`
	Resolved    bool                   `json:"resolved"`
	ResolvedAt  *time.Time             `json:"resolved_at,omitempty"`
}

// AlertCondition defines when an alert should fire
type AlertCondition struct {
	Name        string
	Description string
	Severity    AlertSeverity
	Check       func(metrics HealthMetrics) bool
	Message     func(metrics HealthMetrics) string
}

// AlertManager manages parser alerts
type AlertManager struct {
	mu         sync.RWMutex
	alerts     map[string]*Alert
	conditions []AlertCondition
	handlers   []AlertHandler
	
	// Rate limiting
	lastAlertTime map[string]time.Time
	cooldown      time.Duration
}

// AlertHandler processes alerts
type AlertHandler interface {
	Handle(alert Alert) error
}

// NewAlertManager creates a new alert manager
func NewAlertManager() *AlertManager {
	am := &AlertManager{
		alerts:        make(map[string]*Alert),
		lastAlertTime: make(map[string]time.Time),
		cooldown:      5 * time.Minute, // Prevent alert spam
	}
	
	// Define default alert conditions
	am.conditions = []AlertCondition{
		{
			Name:        "high_failure_rate",
			Description: "Parser failure rate is too high",
			Severity:    AlertSeverityCritical,
			Check: func(m HealthMetrics) bool {
				return m.SuccessRate < 0.9 && m.TotalParses > 100
			},
			Message: func(m HealthMetrics) string {
				return fmt.Sprintf("Parser success rate dropped to %.2f%% (threshold: 90%%)", m.SuccessRate*100)
			},
		},
		{
			Name:        "high_latency",
			Description: "Parser latency is elevated",
			Severity:    AlertSeverityWarning,
			Check: func(m HealthMetrics) bool {
				return m.P95Latency > 500 // 500ms
			},
			Message: func(m HealthMetrics) string {
				return fmt.Sprintf("Parser P95 latency is %.2fms (threshold: 500ms)", m.P95Latency)
			},
		},
		{
			Name:        "extreme_latency",
			Description: "Parser latency is critically high",
			Severity:    AlertSeverityCritical,
			Check: func(m HealthMetrics) bool {
				return m.P99Latency > 2000 // 2 seconds
			},
			Message: func(m HealthMetrics) string {
				return fmt.Sprintf("Parser P99 latency is %.2fms (threshold: 2000ms)", m.P99Latency)
			},
		},
		{
			Name:        "no_recent_parses",
			Description: "No parsing activity detected",
			Severity:    AlertSeverityWarning,
			Check: func(m HealthMetrics) bool {
				if m.LastParseTime == "" {
					return false // No parses yet is okay
				}
				lastParse, err := time.Parse(time.RFC3339, m.LastParseTime)
				if err != nil {
					return false
				}
				return time.Since(lastParse) > 5*time.Minute
			},
			Message: func(m HealthMetrics) string {
				return "No parsing activity in the last 5 minutes"
			},
		},
		{
			Name:        "low_parse_rate",
			Description: "Parse rate is unusually low",
			Severity:    AlertSeverityInfo,
			Check: func(m HealthMetrics) bool {
				// Only alert if we've been running for a while
				return m.TotalParses > 1000 && m.ParseRate < 0.1
			},
			Message: func(m HealthMetrics) string {
				return fmt.Sprintf("Parse rate dropped to %.2f/s", m.ParseRate)
			},
		},
		{
			Name:        "format_imbalance",
			Description: "Unusual distribution of formats",
			Severity:    AlertSeverityInfo,
			Check: func(m HealthMetrics) bool {
				if len(m.FormatDistribution) < 2 {
					return false
				}
				total := int64(0)
				for _, count := range m.FormatDistribution {
					total += count
				}
				if total < 100 {
					return false
				}
				// Alert if one format is >90% of traffic
				for _, count := range m.FormatDistribution {
					if float64(count)/float64(total) > 0.9 {
						return true
					}
				}
				return false
			},
			Message: func(m HealthMetrics) string {
				return "Format distribution is heavily skewed"
			},
		},
	}
	
	return am
}

// AddHandler adds an alert handler
func (am *AlertManager) AddHandler(handler AlertHandler) {
	am.mu.Lock()
	defer am.mu.Unlock()
	am.handlers = append(am.handlers, handler)
}

// AddCondition adds a custom alert condition
func (am *AlertManager) AddCondition(condition AlertCondition) {
	am.mu.Lock()
	defer am.mu.Unlock()
	am.conditions = append(am.conditions, condition)
}

// CheckAlerts evaluates all conditions and fires alerts as needed
func (am *AlertManager) CheckAlerts(metrics HealthMetrics) {
	am.mu.Lock()
	defer am.mu.Unlock()
	
	// Track which conditions are currently firing
	firingConditions := make(map[string]bool)
	
	for _, condition := range am.conditions {
		if condition.Check(metrics) {
			firingConditions[condition.Name] = true
			
			// Check if we should create a new alert
			if am.shouldCreateAlert(condition.Name) {
				alert := Alert{
					ID:          fmt.Sprintf("%s_%d", condition.Name, time.Now().Unix()),
					Severity:    condition.Severity,
					Title:       condition.Description,
					Description: condition.Message(metrics),
					Timestamp:   time.Now(),
					Source:      "parser",
					Labels: map[string]string{
						"condition": condition.Name,
						"component": "tool_parser",
					},
					Metadata: map[string]interface{}{
						"metrics": metrics,
					},
				}
				
				am.alerts[condition.Name] = &alert
				am.lastAlertTime[condition.Name] = time.Now()
				
				// Send to handlers
				am.sendAlert(alert)
			}
		}
	}
	
	// Resolve alerts for conditions that are no longer firing
	for name, alert := range am.alerts {
		if !firingConditions[name] && !alert.Resolved {
			alert.Resolved = true
			now := time.Now()
			alert.ResolvedAt = &now
			
			// Send resolution
			am.sendAlert(*alert)
		}
	}
}

// shouldCreateAlert checks if we should create a new alert
func (am *AlertManager) shouldCreateAlert(conditionName string) bool {
	// Check if alert already exists and is not resolved
	if existing, ok := am.alerts[conditionName]; ok && !existing.Resolved {
		return false
	}
	
	// Check cooldown
	if lastTime, ok := am.lastAlertTime[conditionName]; ok {
		if time.Since(lastTime) < am.cooldown {
			return false
		}
	}
	
	return true
}

// sendAlert sends an alert to all handlers
func (am *AlertManager) sendAlert(alert Alert) {
	for _, handler := range am.handlers {
		go func(h AlertHandler) {
			if err := h.Handle(alert); err != nil {
				// Log error (in production, this would use proper logging)
				fmt.Printf("Alert handler error: %v\n", err)
			}
		}(handler)
	}
}

// GetActiveAlerts returns all active alerts
func (am *AlertManager) GetActiveAlerts() []Alert {
	am.mu.RLock()
	defer am.mu.RUnlock()
	
	var active []Alert
	for _, alert := range am.alerts {
		if !alert.Resolved {
			active = append(active, *alert)
		}
	}
	return active
}

// GetAllAlerts returns all alerts (active and resolved)
func (am *AlertManager) GetAllAlerts() []Alert {
	am.mu.RLock()
	defer am.mu.RUnlock()
	
	var all []Alert
	for _, alert := range am.alerts {
		all = append(all, *alert)
	}
	return all
}

// ClearResolvedAlerts removes resolved alerts older than the specified duration
func (am *AlertManager) ClearResolvedAlerts(olderThan time.Duration) {
	am.mu.Lock()
	defer am.mu.Unlock()
	
	cutoff := time.Now().Add(-olderThan)
	for name, alert := range am.alerts {
		if alert.Resolved && alert.ResolvedAt != nil && alert.ResolvedAt.Before(cutoff) {
			delete(am.alerts, name)
		}
	}
}

// LogAlertHandler logs alerts to stdout (example handler)
type LogAlertHandler struct{}

func (h LogAlertHandler) Handle(alert Alert) error {
	status := "FIRING"
	if alert.Resolved {
		status = "RESOLVED"
	}
	
	fmt.Printf("[%s] %s Alert: %s - %s\n",
		status,
		alert.Severity,
		alert.Title,
		alert.Description,
	)
	return nil
}

// WebhookAlertHandler sends alerts to a webhook
type WebhookAlertHandler struct {
	URL     string
	Headers map[string]string
}

func (h WebhookAlertHandler) Handle(alert Alert) error {
	// Implementation would POST alert JSON to webhook
	// This is a placeholder
	return nil
}

// MonitoredParser wraps a parser with health monitoring and alerting
type MonitoredParser struct {
	parser        ResponseParser
	monitor       *HealthMonitor
	alertManager  *AlertManager
	checkInterval time.Duration
	
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// NewMonitoredParser creates a parser with monitoring and alerting
func NewMonitoredParser(parser ResponseParser, version string) *MonitoredParser {
	ctx, cancel := context.WithCancel(context.Background())
	
	mp := &MonitoredParser{
		parser:        parser,
		monitor:       NewHealthMonitor(parser, version),
		alertManager:  NewAlertManager(),
		checkInterval: 30 * time.Second,
		ctx:           ctx,
		cancel:        cancel,
	}
	
	// Add default alert handler
	mp.alertManager.AddHandler(LogAlertHandler{})
	
	// Start monitoring
	mp.wg.Add(1)
	go mp.monitorLoop()
	
	return mp
}

// monitorLoop runs periodic health checks
func (mp *MonitoredParser) monitorLoop() {
	defer mp.wg.Done()
	
	ticker := time.NewTicker(mp.checkInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-mp.ctx.Done():
			return
		case <-ticker.C:
			health := mp.monitor.Check(mp.ctx)
			mp.alertManager.CheckAlerts(health.Metrics)
			
			// Clean up old resolved alerts
			mp.alertManager.ClearResolvedAlerts(24 * time.Hour)
		}
	}
}

// ExtractToolCalls implements ResponseParser with monitoring
func (mp *MonitoredParser) ExtractToolCalls(response string) ([]ToolCall, error) {
	return mp.ExtractWithContext(mp.ctx, response)
}

// ExtractWithContext implements ResponseParser with monitoring
func (mp *MonitoredParser) ExtractWithContext(ctx context.Context, response string) ([]ToolCall, error) {
	mp.monitor.StartParse()
	defer mp.monitor.EndParse()
	
	start := time.Now()
	
	// Detect format for metrics
	format, _, _ := mp.parser.DetectFormat(response)
	
	// Parse
	calls, err := mp.parser.ExtractWithContext(ctx, response)
	
	// Record metrics
	duration := time.Since(start)
	success := err == nil
	mp.monitor.RecordParse(format, duration, success)
	
	return calls, err
}

// DetectFormat implements ResponseParser
func (mp *MonitoredParser) DetectFormat(response string) (ProviderFormat, float64, error) {
	return mp.parser.DetectFormat(response)
}

// GetHealth returns current health status
func (mp *MonitoredParser) GetHealth() HealthCheck {
	return mp.monitor.Check(mp.ctx)
}

// GetAlerts returns active alerts
func (mp *MonitoredParser) GetAlerts() []Alert {
	return mp.alertManager.GetActiveAlerts()
}

// Stop gracefully shuts down monitoring
func (mp *MonitoredParser) Stop() {
	mp.cancel()
	mp.wg.Wait()
}