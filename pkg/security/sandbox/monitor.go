// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package sandbox

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/guild-framework/guild-core/pkg/gerror"
	"github.com/guild-framework/guild-core/pkg/observability"
)

// GuildSecurityMonitor implements security monitoring for sandbox operations
type GuildSecurityMonitor struct {
	rules         map[string]SecurityRule
	alertHandlers []AlertHandler
	events        chan SecurityEvent
	alerts        chan SecurityAlert
	logger        observability.Logger
	running       bool
	mu            sync.RWMutex
	ctx           context.Context
	cancel        context.CancelFunc
	stats         MonitorStats
	statsMu       sync.RWMutex
}

// MonitorStats tracks security monitoring statistics
type MonitorStats struct {
	EventsProcessed int64         `json:"events_processed"`
	AlertsGenerated int64         `json:"alerts_generated"`
	RulesTriggered  int64         `json:"rules_triggered"`
	Uptime          time.Duration `json:"uptime"`
	StartTime       time.Time     `json:"start_time"`
}

// AlertHandler defines an interface for handling security alerts
type AlertHandler interface {
	HandleAlert(ctx context.Context, alert SecurityAlert) error
}

// NewSecurityMonitor creates a new security monitor
func NewSecurityMonitor(ctx context.Context) (SecurityMonitor, error) {
	logger := observability.GetLogger(ctx).
		WithComponent("SecurityMonitor")

	if err := ctx.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("SecurityMonitor").
			WithOperation("NewSecurityMonitor")
	}

	monitorCtx, cancel := context.WithCancel(ctx)

	monitor := &GuildSecurityMonitor{
		rules:         make(map[string]SecurityRule),
		alertHandlers: make([]AlertHandler, 0),
		events:        make(chan SecurityEvent, 1000),
		alerts:        make(chan SecurityAlert, 100),
		logger:        logger,
		ctx:           monitorCtx,
		cancel:        cancel,
		stats: MonitorStats{
			StartTime: time.Now(),
		},
	}

	// Add default security rules
	monitor.addDefaultRules()

	logger.Info("Security monitor initialized")
	return monitor, nil
}

// StartMonitoring begins monitoring for security events
func (sm *GuildSecurityMonitor) StartMonitoring(ctx context.Context) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if err := ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("SecurityMonitor").
			WithOperation("StartMonitoring")
	}

	if sm.running {
		return gerror.New(gerror.ErrCodeAlreadyExists, "security monitor already running", nil).
			WithComponent("SecurityMonitor").
			WithOperation("StartMonitoring")
	}

	sm.running = true
	sm.updateStats(func(stats *MonitorStats) {
		stats.StartTime = time.Now()
	})

	// Start event processing goroutine
	go sm.processEvents()

	// Start alert handling goroutine
	go sm.handleAlerts()

	sm.logger.Info("Security monitoring started")
	return nil
}

// StopMonitoring stops security monitoring
func (sm *GuildSecurityMonitor) StopMonitoring() error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if !sm.running {
		return nil
	}

	sm.running = false
	sm.cancel()

	// Close channels
	close(sm.events)
	close(sm.alerts)

	sm.logger.Info("Security monitoring stopped")
	return nil
}

// AddRule adds a security monitoring rule
func (sm *GuildSecurityMonitor) AddRule(rule SecurityRule) error {
	if rule == nil {
		return gerror.New(gerror.ErrCodeValidation, "rule cannot be nil", nil).
			WithComponent("SecurityMonitor").
			WithOperation("AddRule")
	}

	if rule.ID() == "" {
		return gerror.New(gerror.ErrCodeValidation, "rule ID cannot be empty", nil).
			WithComponent("SecurityMonitor").
			WithOperation("AddRule")
	}

	sm.mu.Lock()
	defer sm.mu.Unlock()

	sm.rules[rule.ID()] = rule
	sm.logger.Info("Security rule added", "rule_id", rule.ID(), "rule_name", rule.Name())

	return nil
}

// RemoveRule removes a security monitoring rule
func (sm *GuildSecurityMonitor) RemoveRule(ruleID string) error {
	if ruleID == "" {
		return gerror.New(gerror.ErrCodeValidation, "rule ID cannot be empty", nil).
			WithComponent("SecurityMonitor").
			WithOperation("RemoveRule")
	}

	sm.mu.Lock()
	defer sm.mu.Unlock()

	if _, exists := sm.rules[ruleID]; !exists {
		return gerror.New(gerror.ErrCodeNotFound, "rule not found", nil).
			WithComponent("SecurityMonitor").
			WithOperation("RemoveRule").
			WithDetails("rule_id", ruleID)
	}

	delete(sm.rules, ruleID)
	sm.logger.Info("Security rule removed", "rule_id", ruleID)

	return nil
}

// GetAlerts retrieves recent security alerts
func (sm *GuildSecurityMonitor) GetAlerts(ctx context.Context, filter AlertFilter) ([]SecurityAlert, error) {
	if err := ctx.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("SecurityMonitor").
			WithOperation("GetAlerts")
	}

	// This is a simplified implementation
	// In a real system, you would query a persistent storage backend
	alerts := make([]SecurityAlert, 0)

	// For now, return empty slice as we don't have persistent storage in this implementation
	sm.logger.Debug("Retrieved security alerts", "filter", fmt.Sprintf("%+v", filter))

	return alerts, nil
}

// MonitorCommand monitors a specific command execution
func (sm *GuildSecurityMonitor) MonitorCommand(ctx context.Context, cmd Command) error {
	if err := ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("SecurityMonitor").
			WithOperation("MonitorCommand")
	}

	// Create security event
	event := SecurityEvent{
		Type:      "command_execution",
		Timestamp: time.Now(),
		AgentID:   getAgentIDFromContext(ctx),
		Command:   cmd,
		Resource:  fmt.Sprintf("command:%s", cmd.Name),
		Action:    "execute",
		Blocked:   false,
		Reason:    "monitoring",
		Metadata: map[string]interface{}{
			"command_args": cmd.Args,
			"working_dir":  cmd.Dir,
			"environment":  cmd.Env,
		},
	}

	// Send event for processing (non-blocking)
	select {
	case sm.events <- event:
		sm.logger.Debug("Command monitoring event queued", "command", cmd.String())
	default:
		sm.logger.Warn("Event queue full, dropping command monitoring event", "command", cmd.String())
	}

	return nil
}

// AddAlertHandler adds an alert handler
func (sm *GuildSecurityMonitor) AddAlertHandler(handler AlertHandler) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	sm.alertHandlers = append(sm.alertHandlers, handler)
	sm.logger.Info("Alert handler added")
}

// GetStats returns monitoring statistics
func (sm *GuildSecurityMonitor) GetStats() MonitorStats {
	sm.statsMu.RLock()
	defer sm.statsMu.RUnlock()

	stats := sm.stats
	stats.Uptime = time.Since(stats.StartTime)
	return stats
}

// Private methods

func (sm *GuildSecurityMonitor) processEvents() {
	sm.logger.Info("Started security event processing")

	for {
		select {
		case <-sm.ctx.Done():
			sm.logger.Info("Security event processing stopped")
			return
		case event, ok := <-sm.events:
			if !ok {
				sm.logger.Info("Event channel closed, stopping processing")
				return
			}

			sm.updateStats(func(stats *MonitorStats) {
				stats.EventsProcessed++
			})

			// Process event against all rules
			sm.evaluateEvent(event)
		}
	}
}

func (sm *GuildSecurityMonitor) evaluateEvent(event SecurityEvent) {
	sm.mu.RLock()
	rules := make([]SecurityRule, 0, len(sm.rules))
	for _, rule := range sm.rules {
		if rule.IsEnabled() {
			rules = append(rules, rule)
		}
	}
	sm.mu.RUnlock()

	// Evaluate event against all enabled rules
	for _, rule := range rules {
		alert, err := rule.Evaluate(sm.ctx, event)
		if err != nil {
			sm.logger.WithError(err).Warn("Error evaluating security rule", "rule_id", rule.ID())
			continue
		}

		if alert != nil {
			sm.updateStats(func(stats *MonitorStats) {
				stats.RulesTriggered++
				stats.AlertsGenerated++
			})

			// Send alert for handling
			select {
			case sm.alerts <- *alert:
				sm.logger.Debug("Security alert generated", "rule_id", rule.ID(), "severity", alert.Severity)
			default:
				sm.logger.Warn("Alert queue full, dropping security alert", "rule_id", rule.ID())
			}
		}
	}
}

func (sm *GuildSecurityMonitor) handleAlerts() {
	sm.logger.Info("Started security alert handling")

	for {
		select {
		case <-sm.ctx.Done():
			sm.logger.Info("Security alert handling stopped")
			return
		case alert, ok := <-sm.alerts:
			if !ok {
				sm.logger.Info("Alert channel closed, stopping handling")
				return
			}

			// Send alert to all handlers
			sm.mu.RLock()
			handlers := append([]AlertHandler(nil), sm.alertHandlers...) // Copy slice
			sm.mu.RUnlock()

			for _, handler := range handlers {
				if err := handler.HandleAlert(sm.ctx, alert); err != nil {
					sm.logger.WithError(err).Warn("Alert handler failed", "alert_id", alert.ID)
				}
			}
		}
	}
}

func (sm *GuildSecurityMonitor) addDefaultRules() {
	// Add default security rules
	defaultRules := []SecurityRule{
		&SuspiciousCommandRule{},
		&HighResourceUsageRule{},
		&UnauthorizedAccessRule{},
		&MaliciousFileAccessRule{},
	}

	for _, rule := range defaultRules {
		sm.rules[rule.ID()] = rule
	}

	sm.logger.Info("Default security rules added", "count", len(defaultRules))
}

func (sm *GuildSecurityMonitor) updateStats(fn func(*MonitorStats)) {
	sm.statsMu.Lock()
	defer sm.statsMu.Unlock()
	fn(&sm.stats)
}

// Default Security Rules

// SuspiciousCommandRule detects suspicious command patterns
type SuspiciousCommandRule struct {
	enabled bool
}

func (scr *SuspiciousCommandRule) ID() string {
	return "suspicious_command"
}

func (scr *SuspiciousCommandRule) Name() string {
	return "Suspicious Command Detection"
}

func (scr *SuspiciousCommandRule) Evaluate(ctx context.Context, event SecurityEvent) (*SecurityAlert, error) {
	if event.Type != "command_execution" {
		return nil, nil
	}

	cmdStr := event.Command.String()
	suspiciousPatterns := []string{
		"rm -rf /",
		"curl | sh",
		"wget | bash",
		"eval",
		"exec",
		"base64 -d",
		"nc -l",
		"python -c",
		"perl -e",
		"ruby -e",
		"node -e",
		"powershell",
		"cmd.exe",
		"/dev/tcp",
		"mkfifo",
	}

	for _, pattern := range suspiciousPatterns {
		if strings.Contains(strings.ToLower(cmdStr), pattern) {
			return &SecurityAlert{
				ID:        fmt.Sprintf("suspicious-cmd-%d", time.Now().UnixNano()),
				Rule:      scr.ID(),
				Severity:  SeverityHigh,
				Message:   fmt.Sprintf("Suspicious command pattern detected: %s", pattern),
				Event:     event,
				Timestamp: time.Now(),
			}, nil
		}
	}

	return nil, nil
}

func (scr *SuspiciousCommandRule) IsEnabled() bool {
	return true
}

func (scr *SuspiciousCommandRule) SetEnabled(enabled bool) {
	scr.enabled = enabled
}

// HighResourceUsageRule detects commands that may consume excessive resources
type HighResourceUsageRule struct {
	enabled bool
}

func (hrur *HighResourceUsageRule) ID() string {
	return "high_resource_usage"
}

func (hrur *HighResourceUsageRule) Name() string {
	return "High Resource Usage Detection"
}

func (hrur *HighResourceUsageRule) Evaluate(ctx context.Context, event SecurityEvent) (*SecurityAlert, error) {
	if event.Type != "command_execution" {
		return nil, nil
	}

	cmdStr := strings.ToLower(event.Command.String())
	resourceIntensivePatterns := []string{
		"find / ",
		"dd if=",
		"tar -",
		"gzip",
		"gunzip",
		"compress",
		"stress",
		"yes ",
		":(){ :|:& };:", // Fork bomb
		"while true",
		"for i in",
	}

	for _, pattern := range resourceIntensivePatterns {
		if strings.Contains(cmdStr, pattern) {
			return &SecurityAlert{
				ID:        fmt.Sprintf("high-resource-%d", time.Now().UnixNano()),
				Rule:      hrur.ID(),
				Severity:  SeverityMedium,
				Message:   fmt.Sprintf("Potentially resource-intensive command detected: %s", pattern),
				Event:     event,
				Timestamp: time.Now(),
			}, nil
		}
	}

	return nil, nil
}

func (hrur *HighResourceUsageRule) IsEnabled() bool {
	return true
}

func (hrur *HighResourceUsageRule) SetEnabled(enabled bool) {
	hrur.enabled = enabled
}

// UnauthorizedAccessRule detects attempts to access unauthorized resources
type UnauthorizedAccessRule struct {
	enabled bool
}

func (uar *UnauthorizedAccessRule) ID() string {
	return "unauthorized_access"
}

func (uar *UnauthorizedAccessRule) Name() string {
	return "Unauthorized Access Detection"
}

func (uar *UnauthorizedAccessRule) Evaluate(ctx context.Context, event SecurityEvent) (*SecurityAlert, error) {
	if event.Blocked && strings.Contains(event.Reason, "permission denied") {
		return &SecurityAlert{
			ID:        fmt.Sprintf("unauthorized-%d", time.Now().UnixNano()),
			Rule:      uar.ID(),
			Severity:  SeverityMedium,
			Message:   fmt.Sprintf("Unauthorized access attempt to %s", event.Resource),
			Event:     event,
			Timestamp: time.Now(),
		}, nil
	}

	return nil, nil
}

func (uar *UnauthorizedAccessRule) IsEnabled() bool {
	return true
}

func (uar *UnauthorizedAccessRule) SetEnabled(enabled bool) {
	uar.enabled = enabled
}

// MaliciousFileAccessRule detects attempts to access sensitive files
type MaliciousFileAccessRule struct {
	enabled bool
}

func (mfar *MaliciousFileAccessRule) ID() string {
	return "malicious_file_access"
}

func (mfar *MaliciousFileAccessRule) Name() string {
	return "Malicious File Access Detection"
}

func (mfar *MaliciousFileAccessRule) Evaluate(ctx context.Context, event SecurityEvent) (*SecurityAlert, error) {
	if event.Type != "command_execution" {
		return nil, nil
	}

	cmdStr := strings.ToLower(event.Command.String())
	sensitivePaths := []string{
		"/etc/passwd",
		"/etc/shadow",
		"/etc/sudoers",
		"/root/",
		"/.ssh/",
		"/.aws/",
		"/proc/",
		"/sys/",
	}

	for _, path := range sensitivePaths {
		if strings.Contains(cmdStr, path) {
			return &SecurityAlert{
				ID:        fmt.Sprintf("malicious-file-%d", time.Now().UnixNano()),
				Rule:      mfar.ID(),
				Severity:  SeverityHigh,
				Message:   fmt.Sprintf("Attempt to access sensitive file/directory: %s", path),
				Event:     event,
				Timestamp: time.Now(),
			}, nil
		}
	}

	return nil, nil
}

func (mfar *MaliciousFileAccessRule) IsEnabled() bool {
	return true
}

func (mfar *MaliciousFileAccessRule) SetEnabled(enabled bool) {
	mfar.enabled = enabled
}

// Helper functions

func getAgentIDFromContext(ctx context.Context) string {
	if agentID, ok := ctx.Value("agent_id").(string); ok {
		return agentID
	}
	return "unknown"
}
