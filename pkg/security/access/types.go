// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package access

import (
	"context"
	"time"
)

// AccessRequest represents a request to access a tool or resource
type AccessRequest struct {
	AgentID    string                 `json:"agent_id"`
	ToolName   string                 `json:"tool_name"`
	Action     string                 `json:"action"`
	Parameters map[string]interface{} `json:"parameters"`
	Conditions []string               `json:"conditions,omitempty"`
	Timestamp  time.Time              `json:"timestamp"`
	RequestID  string                 `json:"request_id,omitempty"`
	SessionID  string                 `json:"session_id,omitempty"`
	IPAddress  string                 `json:"ip_address,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// AccessDecision represents the result of an access control check
type AccessDecision struct {
	Allowed    bool                   `json:"allowed"`
	Reason     string                 `json:"reason"`
	Resource   string                 `json:"resource"`
	Action     string                 `json:"action"`
	AgentID    string                 `json:"agent_id,omitempty"`
	CheckTime  time.Duration          `json:"check_time"`
	CacheHit   bool                   `json:"cache_hit"`
	Timestamp  time.Time              `json:"timestamp"`
	Conditions []string               `json:"conditions,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// AccessEvent represents an access control event for auditing
type AccessEvent struct {
	Type      string                 `json:"type"`
	Timestamp time.Time              `json:"timestamp"`
	AgentID   string                 `json:"agent_id"`
	Resource  string                 `json:"resource"`
	Action    string                 `json:"action"`
	Reason    string                 `json:"reason"`
	RequestID string                 `json:"request_id,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// AuditEntry represents a single audit log entry
type AuditEntry struct {
	ID        string                 `json:"id"`
	Timestamp time.Time              `json:"timestamp"`
	AgentID   string                 `json:"agent_id"`
	UserID    string                 `json:"user_id,omitempty"`
	Resource  string                 `json:"resource"`
	Action    string                 `json:"action"`
	Result    string                 `json:"result"` // "allowed", "denied", "success", "error"
	Reason    string                 `json:"reason,omitempty"`
	Duration  time.Duration          `json:"duration,omitempty"`
	IPAddress string                 `json:"ip_address,omitempty"`
	SessionID string                 `json:"session_id,omitempty"`
	RequestID string                 `json:"request_id,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// AccessStats provides statistics about access control operations
type AccessStats struct {
	CacheStats      CacheStats `json:"cache_stats"`
	ChecksPerformed int64      `json:"checks_performed"`
	CacheHitRate    float64    `json:"cache_hit_rate"`
}

// CacheStats provides statistics about the permission cache
type CacheStats struct {
	Size      int   `json:"size"`
	MaxSize   int   `json:"max_size"`
	Hits      int64 `json:"hits"`
	Misses    int64 `json:"misses"`
	Evictions int64 `json:"evictions"`
}

// PolicyRule represents a conditional access policy
type PolicyRule struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Conditions  []PolicyCondition      `json:"conditions"`
	Action      PolicyAction           `json:"action"`
	Priority    int                    `json:"priority"`
	Enabled     bool                   `json:"enabled"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// PolicyCondition defines when a policy should be applied
type PolicyCondition struct {
	Type     string      `json:"type"`     // "time", "agent_role", "resource_pattern", etc.
	Operator string      `json:"operator"` // "equals", "contains", "matches", etc.
	Value    interface{} `json:"value"`
}

// PolicyAction defines what to do when a policy applies
type PolicyAction struct {
	Type   string      `json:"type"`   // "allow", "deny", "require_approval", etc.
	Config interface{} `json:"config"` // Additional configuration for the action
}

// SecurityAlert represents a security-related alert
type SecurityAlert struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"`
	Severity    AlertSeverity          `json:"severity"`
	Title       string                 `json:"title"`
	Description string                 `json:"description"`
	AgentID     string                 `json:"agent_id,omitempty"`
	Resource    string                 `json:"resource,omitempty"`
	Action      string                 `json:"action,omitempty"`
	Timestamp   time.Time              `json:"timestamp"`
	Resolved    bool                   `json:"resolved"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// AlertSeverity defines the severity levels for security alerts
type AlertSeverity int

const (
	SeverityLow AlertSeverity = iota
	SeverityMedium
	SeverityHigh
	SeverityCritical
)

func (s AlertSeverity) String() string {
	switch s {
	case SeverityLow:
		return "low"
	case SeverityMedium:
		return "medium"
	case SeverityHigh:
		return "high"
	case SeverityCritical:
		return "critical"
	default:
		return "unknown"
	}
}

// Interfaces

// AuditLogger defines the interface for audit logging
type AuditLogger interface {
	// LogAllowed logs a successful access grant
	LogAllowed(ctx context.Context, entry AuditEntry) error

	// LogDenied logs an access denial
	LogDenied(ctx context.Context, entry AuditEntry) error

	// LogExecution logs tool execution results
	LogExecution(ctx context.Context, entry AuditEntry) error

	// Query retrieves audit entries based on filters
	Query(ctx context.Context, filter AuditFilter) ([]AuditEntry, error)

	// GetStats returns audit logger statistics
	GetStats() AuditStats
}

// AuditFilter defines criteria for querying audit logs
type AuditFilter struct {
	StartTime *time.Time `json:"start_time,omitempty"`
	EndTime   *time.Time `json:"end_time,omitempty"`
	AgentID   string     `json:"agent_id,omitempty"`
	Resource  string     `json:"resource,omitempty"`
	Action    string     `json:"action,omitempty"`
	Result    string     `json:"result,omitempty"`
	Limit     int        `json:"limit,omitempty"`
	Offset    int        `json:"offset,omitempty"`
}

// AuditStats provides statistics about audit operations
type AuditStats struct {
	TotalEntries     int64   `json:"total_entries"`
	EntriesPerSecond float64 `json:"entries_per_second"`
	StorageSize      int64   `json:"storage_size_bytes"`
	ErrorRate        float64 `json:"error_rate"`
}

// EventBus defines the interface for publishing access control events
type EventBus interface {
	// PublishAccessEvent publishes an access control event
	PublishAccessEvent(ctx context.Context, event AccessEvent) error

	// PublishSecurityAlert publishes a security alert
	PublishSecurityAlert(ctx context.Context, alert SecurityAlert) error
}

// PolicyManager defines the interface for managing access policies
type PolicyManager interface {
	// AddPolicy adds a new access policy
	AddPolicy(ctx context.Context, policy PolicyRule) error

	// UpdatePolicy modifies an existing access policy
	UpdatePolicy(ctx context.Context, policy PolicyRule) error

	// DeletePolicy removes an access policy
	DeletePolicy(ctx context.Context, policyID string) error

	// GetPolicy retrieves a policy by ID
	GetPolicy(policyID string) (*PolicyRule, error)

	// ListPolicies returns all policies
	ListPolicies() []*PolicyRule

	// EvaluatePolicy checks if a policy applies to a request
	EvaluatePolicy(ctx context.Context, policy PolicyRule, request AccessRequest) bool
}

// SecurityMonitor defines the interface for monitoring security events
type SecurityMonitor interface {
	// StartMonitoring begins monitoring for security events
	StartMonitoring(ctx context.Context) error

	// StopMonitoring stops security monitoring
	StopMonitoring() error

	// AddRule adds a security monitoring rule
	AddRule(rule SecurityRule) error

	// RemoveRule removes a security monitoring rule
	RemoveRule(ruleID string) error

	// GetAlerts retrieves recent security alerts
	GetAlerts(filter AlertFilter) ([]SecurityAlert, error)
}

// SecurityRule defines a rule for security monitoring
type SecurityRule interface {
	// ID returns the rule identifier
	ID() string

	// Name returns the rule name
	Name() string

	// Evaluate checks if the rule triggers on an event
	Evaluate(ctx context.Context, event AccessEvent) (*SecurityAlert, error)

	// IsEnabled returns whether the rule is active
	IsEnabled() bool
}

// AlertFilter defines criteria for querying security alerts
type AlertFilter struct {
	StartTime *time.Time    `json:"start_time,omitempty"`
	EndTime   *time.Time    `json:"end_time,omitempty"`
	Severity  AlertSeverity `json:"severity,omitempty"`
	Type      string        `json:"type,omitempty"`
	AgentID   string        `json:"agent_id,omitempty"`
	Resolved  *bool         `json:"resolved,omitempty"`
	Limit     int           `json:"limit,omitempty"`
	Offset    int           `json:"offset,omitempty"`
}

// ComplianceReport represents a compliance audit report
type ComplianceReport struct {
	ID              string                `json:"id"`
	Title           string                `json:"title"`
	Description     string                `json:"description"`
	Period          ReportPeriod          `json:"period"`
	GeneratedAt     time.Time             `json:"generated_at"`
	GeneratedBy     string                `json:"generated_by"`
	Violations      []ComplianceViolation `json:"violations"`
	Statistics      ComplianceStats       `json:"statistics"`
	Recommendations []string              `json:"recommendations"`
	Status          string                `json:"status"`
}

// ReportPeriod defines the time period for a compliance report
type ReportPeriod struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
	Type  string    `json:"type"` // "daily", "weekly", "monthly", "quarterly", "annual"
}

// ComplianceViolation represents a compliance rule violation
type ComplianceViolation struct {
	ID          string                 `json:"id"`
	RuleID      string                 `json:"rule_id"`
	RuleName    string                 `json:"rule_name"`
	Severity    AlertSeverity          `json:"severity"`
	Description string                 `json:"description"`
	Timestamp   time.Time              `json:"timestamp"`
	AgentID     string                 `json:"agent_id,omitempty"`
	Resource    string                 `json:"resource,omitempty"`
	Action      string                 `json:"action,omitempty"`
	Evidence    map[string]interface{} `json:"evidence,omitempty"`
	Status      string                 `json:"status"` // "open", "investigating", "resolved", "false_positive"
}

// ComplianceStats provides statistics for compliance reporting
type ComplianceStats struct {
	TotalAccesses         int64         `json:"total_accesses"`
	AllowedAccesses       int64         `json:"allowed_accesses"`
	DeniedAccesses        int64         `json:"denied_accesses"`
	ViolationsFound       int           `json:"violations_found"`
	CriticalViolations    int           `json:"critical_violations"`
	HighViolations        int           `json:"high_violations"`
	MediumViolations      int           `json:"medium_violations"`
	LowViolations         int           `json:"low_violations"`
	ComplianceScore       float64       `json:"compliance_score"` // 0-100%
	AverageResponseTime   time.Duration `json:"average_response_time"`
	PolicyCoverage        float64       `json:"policy_coverage"` // 0-100%
	UnauthorizedAttempts  int64         `json:"unauthorized_attempts"`
	SuccessfulEscalations int64         `json:"successful_escalations"`
}
