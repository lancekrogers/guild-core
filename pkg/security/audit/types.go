// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package audit

import (
	"context"
	"time"
)

// AuditEntry represents a single audit log entry
type AuditEntry struct {
	ID         string                 `json:"id"`
	Timestamp  time.Time              `json:"timestamp"`
	AgentID    string                 `json:"agent_id"`
	UserID     string                 `json:"user_id,omitempty"`
	SessionID  string                 `json:"session_id,omitempty"`
	RequestID  string                 `json:"request_id,omitempty"`
	Resource   string                 `json:"resource"`
	Action     string                 `json:"action"`
	Result     AuditResult            `json:"result"`
	Reason     string                 `json:"reason,omitempty"`
	Duration   time.Duration          `json:"duration,omitempty"`
	IPAddress  string                 `json:"ip_address,omitempty"`
	UserAgent  string                 `json:"user_agent,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
	RiskScore  int                    `json:"risk_score,omitempty"`
	Compliance []string               `json:"compliance,omitempty"`
}

// AuditResult represents the outcome of an audited action
type AuditResult int

const (
	ResultAllowed AuditResult = iota
	ResultDenied
	ResultError
	ResultTimeout
	ResultBlocked
)

func (ar AuditResult) String() string {
	switch ar {
	case ResultAllowed:
		return "allowed"
	case ResultDenied:
		return "denied"
	case ResultError:
		return "error"
	case ResultTimeout:
		return "timeout"
	case ResultBlocked:
		return "blocked"
	default:
		return "unknown"
	}
}

// AuditFilter defines criteria for querying audit logs
type AuditFilter struct {
	StartTime    *time.Time   `json:"start_time,omitempty"`
	EndTime      *time.Time   `json:"end_time,omitempty"`
	AgentID      string       `json:"agent_id,omitempty"`
	UserID       string       `json:"user_id,omitempty"`
	SessionID    string       `json:"session_id,omitempty"`
	Resource     string       `json:"resource,omitempty"`
	Action       string       `json:"action,omitempty"`
	Result       *AuditResult `json:"result,omitempty"`
	IPAddress    string       `json:"ip_address,omitempty"`
	MinRiskScore int          `json:"min_risk_score,omitempty"`
	Compliance   []string     `json:"compliance,omitempty"`
	Limit        int          `json:"limit,omitempty"`
	Offset       int          `json:"offset,omitempty"`
	SortBy       string       `json:"sort_by,omitempty"`
	SortOrder    string       `json:"sort_order,omitempty"`
}

// AuditStats provides statistics about audit log activity
type AuditStats struct {
	TotalEntries     int64         `json:"total_entries"`
	AllowedActions   int64         `json:"allowed_actions"`
	DeniedActions    int64         `json:"denied_actions"`
	ErrorActions     int64         `json:"error_actions"`
	UniqueAgents     int64         `json:"unique_agents"`
	UniqueSessions   int64         `json:"unique_sessions"`
	AverageRiskScore float64       `json:"average_risk_score"`
	HighRiskActions  int64         `json:"high_risk_actions"`
	LastActivity     time.Time     `json:"last_activity"`
	RetentionPeriod  time.Duration `json:"retention_period"`
	StorageSize      int64         `json:"storage_size_bytes"`
}

// ComplianceReport represents a compliance audit report
type ComplianceReport struct {
	ID              string                 `json:"id"`
	GeneratedAt     time.Time              `json:"generated_at"`
	Period          ReportPeriod           `json:"period"`
	Compliance      ComplianceStandard     `json:"compliance"`
	Summary         ComplianceSummary      `json:"summary"`
	Violations      []ComplianceViolation  `json:"violations"`
	Recommendations []string               `json:"recommendations"`
	Attestations    []Attestation          `json:"attestations"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
}

// ReportPeriod defines the time range for a compliance report
type ReportPeriod struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
	Name  string    `json:"name"` // e.g., "Q1 2025", "January 2025"
}

// ComplianceStandard represents a compliance framework
type ComplianceStandard int

const (
	ComplianceSOC2 ComplianceStandard = iota
	ComplianceISO27001
	ComplianceGDPR
	ComplianceHIPAA
	CompliancePCIDSS
	ComplianceFedRAMP
	ComplianceCustom
)

func (cs ComplianceStandard) String() string {
	switch cs {
	case ComplianceSOC2:
		return "SOC 2"
	case ComplianceISO27001:
		return "ISO 27001"
	case ComplianceGDPR:
		return "GDPR"
	case ComplianceHIPAA:
		return "HIPAA"
	case CompliancePCIDSS:
		return "PCI DSS"
	case ComplianceFedRAMP:
		return "FedRAMP"
	case ComplianceCustom:
		return "Custom"
	default:
		return "Unknown"
	}
}

// ComplianceSummary provides high-level compliance metrics
type ComplianceSummary struct {
	TotalChecks     int       `json:"total_checks"`
	PassedChecks    int       `json:"passed_checks"`
	FailedChecks    int       `json:"failed_checks"`
	ComplianceScore float64   `json:"compliance_score"`
	RiskLevel       string    `json:"risk_level"`
	TrendDirection  string    `json:"trend_direction"`
	LastAssessment  time.Time `json:"last_assessment"`
	NextAssessment  time.Time `json:"next_assessment"`
}

// ComplianceViolation represents a compliance rule violation
type ComplianceViolation struct {
	ID          string      `json:"id"`
	Rule        string      `json:"rule"`
	Description string      `json:"description"`
	Severity    string      `json:"severity"`
	Category    string      `json:"category"`
	DetectedAt  time.Time   `json:"detected_at"`
	AuditEntry  *AuditEntry `json:"audit_entry,omitempty"`
	Remediation string      `json:"remediation"`
	Status      string      `json:"status"`
	Assignee    string      `json:"assignee,omitempty"`
	DueDate     *time.Time  `json:"due_date,omitempty"`
}

// Attestation represents a compliance attestation or certification
type Attestation struct {
	ID          string    `json:"id"`
	Type        string    `json:"type"`
	Description string    `json:"description"`
	Attestor    string    `json:"attestor"`
	AttestedAt  time.Time `json:"attested_at"`
	ValidUntil  time.Time `json:"valid_until"`
	Evidence    []string  `json:"evidence,omitempty"`
	Signature   string    `json:"signature,omitempty"`
	Certificate string    `json:"certificate,omitempty"`
}

// Interfaces

// AuditLogger defines the interface for audit logging
type AuditLogger interface {
	// LogEntry logs a new audit entry
	LogEntry(ctx context.Context, entry AuditEntry) error

	// Query retrieves audit entries based on filter criteria
	Query(ctx context.Context, filter AuditFilter) ([]AuditEntry, error)

	// GetStats returns audit logging statistics
	GetStats(ctx context.Context) (*AuditStats, error)

	// GenerateReport creates a compliance report
	GenerateReport(ctx context.Context, period ReportPeriod, standard ComplianceStandard) (*ComplianceReport, error)

	// Archive moves old audit entries to long-term storage
	Archive(ctx context.Context, before time.Time) error

	// Close releases audit logger resources
	Close() error
}

// AuditStorage defines the interface for audit data storage
type AuditStorage interface {
	// Store saves an audit entry
	Store(ctx context.Context, entry AuditEntry) error

	// Retrieve gets audit entries based on filter
	Retrieve(ctx context.Context, filter AuditFilter) ([]AuditEntry, error)

	// Count returns the number of entries matching the filter
	Count(ctx context.Context, filter AuditFilter) (int64, error)

	// Delete removes audit entries (typically for archival)
	Delete(ctx context.Context, before time.Time) error

	// Backup creates a backup of audit data
	Backup(ctx context.Context, destination string) error

	// Health checks storage health
	Health(ctx context.Context) error
}

// AuditStreamer defines the interface for real-time audit streaming
type AuditStreamer interface {
	// Stream sends audit entries to external systems
	Stream(ctx context.Context, entry AuditEntry) error

	// AddDestination adds a streaming destination
	AddDestination(destination StreamDestination) error

	// RemoveDestination removes a streaming destination
	RemoveDestination(destinationID string) error

	// GetDestinations returns all configured destinations
	GetDestinations() []StreamDestination
}

// StreamDestination represents an external audit streaming destination
type StreamDestination struct {
	ID       string            `json:"id"`
	Type     string            `json:"type"` // elasticsearch, siem, webhook, etc.
	Endpoint string            `json:"endpoint"`
	Headers  map[string]string `json:"headers,omitempty"`
	Filter   *AuditFilter      `json:"filter,omitempty"`
	Enabled  bool              `json:"enabled"`
	Retries  int               `json:"retries"`
	Timeout  time.Duration     `json:"timeout"`
}

// AuditEnricher enhances audit entries with additional context
type AuditEnricher interface {
	// Enrich adds additional context to an audit entry
	Enrich(ctx context.Context, entry AuditEntry) (AuditEntry, error)

	// CalculateRiskScore assigns a risk score to an audit entry
	CalculateRiskScore(ctx context.Context, entry AuditEntry) (int, error)

	// AddCompliance tags adds compliance framework tags
	AddComplianceTags(ctx context.Context, entry AuditEntry) ([]string, error)
}

// RetentionPolicy defines how long audit data should be retained
type RetentionPolicy struct {
	Standard   time.Duration `json:"standard"`   // Regular audit data
	HighRisk   time.Duration `json:"high_risk"`  // High-risk activities
	Compliance time.Duration `json:"compliance"` // Compliance-related data
	Incidents  time.Duration `json:"incidents"`  // Security incidents
}

// DefaultRetentionPolicy returns a sensible default retention policy
func DefaultRetentionPolicy() RetentionPolicy {
	return RetentionPolicy{
		Standard:   90 * 24 * time.Hour,       // 90 days
		HighRisk:   365 * 24 * time.Hour,      // 1 year
		Compliance: 7 * 365 * 24 * time.Hour,  // 7 years
		Incidents:  10 * 365 * 24 * time.Hour, // 10 years
	}
}
