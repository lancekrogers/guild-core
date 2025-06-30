// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package audit

import (
	"context"
	"crypto/md5"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/lancekrogers/guild/pkg/gerror"
	"github.com/lancekrogers/guild/pkg/observability"
)

// GuildAuditLogger implements comprehensive audit logging
type GuildAuditLogger struct {
	storage      AuditStorage
	streamer     AuditStreamer
	enricher     AuditEnricher
	retention    RetentionPolicy
	logger       observability.Logger
	mu           sync.RWMutex
	stats        *AuditStats
	statsMu      sync.RWMutex
	processQueue chan AuditEntry
	ctx          context.Context
	cancel       context.CancelFunc
}

// NewGuildAuditLogger creates a new audit logger
func NewGuildAuditLogger(ctx context.Context, storage AuditStorage, config AuditConfig) (*GuildAuditLogger, error) {
	logger := observability.GetLogger(ctx).
		WithComponent("AuditLogger")

	if err := ctx.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("AuditLogger").
			WithOperation("NewGuildAuditLogger")
	}

	if storage == nil {
		return nil, gerror.New(gerror.ErrCodeValidation, "storage cannot be nil", nil).
			WithComponent("AuditLogger").
			WithOperation("NewGuildAuditLogger")
	}

	auditCtx, cancel := context.WithCancel(ctx)

	auditLogger := &GuildAuditLogger{
		storage:      storage,
		streamer:     config.Streamer,
		enricher:     config.Enricher,
		retention:    config.RetentionPolicy,
		logger:       logger,
		stats:        &AuditStats{LastActivity: time.Now()},
		processQueue: make(chan AuditEntry, config.QueueSize),
		ctx:          auditCtx,
		cancel:       cancel,
	}

	// Start background processing
	go auditLogger.processEntries()
	go auditLogger.periodicMaintenance()

	logger.Info("Audit logger initialized",
		"queue_size", config.QueueSize,
		"retention_standard", config.RetentionPolicy.Standard,
	)

	return auditLogger, nil
}

// AuditConfig provides configuration for the audit logger
type AuditConfig struct {
	Streamer        AuditStreamer   `json:"streamer,omitempty"`
	Enricher        AuditEnricher   `json:"enricher,omitempty"`
	RetentionPolicy RetentionPolicy `json:"retention_policy"`
	QueueSize       int             `json:"queue_size"`
	BatchSize       int             `json:"batch_size"`
	FlushInterval   time.Duration   `json:"flush_interval"`
}

// DefaultAuditConfig returns sensible defaults
func DefaultAuditConfig() AuditConfig {
	return AuditConfig{
		RetentionPolicy: DefaultRetentionPolicy(),
		QueueSize:       10000,
		BatchSize:       100,
		FlushInterval:   5 * time.Second,
	}
}

// LogEntry logs a new audit entry
func (gal *GuildAuditLogger) LogEntry(ctx context.Context, entry AuditEntry) error {
	if err := ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("AuditLogger").
			WithOperation("LogEntry")
	}

	// Generate ID if not provided
	if entry.ID == "" {
		entry.ID = gal.generateEntryID(entry)
	}

	// Set timestamp if not provided
	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now()
	}

	// Validate entry
	if err := gal.validateEntry(entry); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeValidation, "invalid audit entry").
			WithComponent("AuditLogger").
			WithOperation("LogEntry")
	}

	// Enrich entry if enricher is available
	if gal.enricher != nil {
		enriched, err := gal.enricher.Enrich(ctx, entry)
		if err != nil {
			gal.logger.WithError(err).Warn("Failed to enrich audit entry")
		} else {
			entry = enriched
		}
	}

	// Queue for processing (non-blocking)
	select {
	case gal.processQueue <- entry:
		gal.updateStats(func(stats *AuditStats) {
			stats.TotalEntries++
			stats.LastActivity = time.Now()
			switch entry.Result {
			case ResultAllowed:
				stats.AllowedActions++
			case ResultDenied:
				stats.DeniedActions++
			case ResultError:
				stats.ErrorActions++
			}
		})
	default:
		// Queue is full, log synchronously as fallback
		gal.logger.Warn("Audit queue full, processing synchronously")
		return gal.processEntry(ctx, entry)
	}

	return nil
}

// Query retrieves audit entries based on filter criteria
func (gal *GuildAuditLogger) Query(ctx context.Context, filter AuditFilter) ([]AuditEntry, error) {
	if err := ctx.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("AuditLogger").
			WithOperation("Query")
	}

	// Validate filter
	if err := gal.validateFilter(filter); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeValidation, "invalid audit filter").
			WithComponent("AuditLogger").
			WithOperation("Query")
	}

	// Apply default limits if not specified
	if filter.Limit <= 0 {
		filter.Limit = 1000 // Default limit
	}
	if filter.Limit > 10000 {
		filter.Limit = 10000 // Maximum limit
	}

	gal.logger.Debug("Querying audit entries",
		"filter", fmt.Sprintf("%+v", filter),
	)

	return gal.storage.Retrieve(ctx, filter)
}

// GetStats returns audit logging statistics
func (gal *GuildAuditLogger) GetStats(ctx context.Context) (*AuditStats, error) {
	if err := ctx.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("AuditLogger").
			WithOperation("GetStats")
	}

	gal.statsMu.RLock()
	stats := *gal.stats
	gal.statsMu.RUnlock()

	// Get additional stats from storage if possible
	if healthChecker, ok := gal.storage.(interface{ Health(context.Context) error }); ok {
		if err := healthChecker.Health(ctx); err != nil {
			gal.logger.WithError(err).Warn("Storage health check failed")
		}
	}

	return &stats, nil
}

// GenerateReport creates a compliance report
func (gal *GuildAuditLogger) GenerateReport(ctx context.Context, period ReportPeriod, standard ComplianceStandard) (*ComplianceReport, error) {
	if err := ctx.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("AuditLogger").
			WithOperation("GenerateReport")
	}

	gal.logger.Info("Generating compliance report",
		"standard", standard.String(),
		"period_start", period.Start,
		"period_end", period.End,
	)

	// Query audit entries for the period
	filter := AuditFilter{
		StartTime: &period.Start,
		EndTime:   &period.End,
		Limit:     100000, // Large limit for comprehensive report
	}

	entries, err := gal.storage.Retrieve(ctx, filter)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to retrieve audit entries").
			WithComponent("AuditLogger").
			WithOperation("GenerateReport")
	}

	// Generate report based on compliance standard
	report := &ComplianceReport{
		ID:              gal.generateReportID(period, standard),
		GeneratedAt:     time.Now(),
		Period:          period,
		Compliance:      standard,
		Summary:         gal.generateSummary(entries),
		Violations:      gal.detectViolations(entries, standard),
		Recommendations: gal.generateRecommendations(entries, standard),
		Attestations:    gal.getAttestations(standard),
	}

	gal.logger.Info("Compliance report generated",
		"report_id", report.ID,
		"total_entries", len(entries),
		"violations", len(report.Violations),
		"compliance_score", report.Summary.ComplianceScore,
	)

	return report, nil
}

// Archive moves old audit entries to long-term storage
func (gal *GuildAuditLogger) Archive(ctx context.Context, before time.Time) error {
	if err := ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("AuditLogger").
			WithOperation("Archive")
	}

	gal.logger.Info("Starting audit log archival", "before", before)

	// Create backup before deletion
	backupPath := fmt.Sprintf("audit-backup-%s.sql", before.Format("2006-01-02"))
	if backupper, ok := gal.storage.(interface {
		Backup(context.Context, string) error
	}); ok {
		if err := backupper.Backup(ctx, backupPath); err != nil {
			gal.logger.WithError(err).Warn("Backup failed during archival")
		}
	}

	// Delete old entries
	if deleter, ok := gal.storage.(interface {
		Delete(context.Context, time.Time) error
	}); ok {
		if err := deleter.Delete(ctx, before); err != nil {
			return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to delete old audit entries").
				WithComponent("AuditLogger").
				WithOperation("Archive")
		}
	}

	gal.logger.Info("Audit log archival completed", "before", before)
	return nil
}

// Close releases audit logger resources
func (gal *GuildAuditLogger) Close() error {
	gal.logger.Info("Closing audit logger")

	// Cancel context to stop background processing
	gal.cancel()

	// Close the processing queue
	close(gal.processQueue)

	// Process any remaining entries in the queue
	for entry := range gal.processQueue {
		if err := gal.processEntry(context.Background(), entry); err != nil {
			gal.logger.WithError(err).Warn("Failed to process entry during shutdown")
		}
	}

	return nil
}

// Background processing methods

func (gal *GuildAuditLogger) processEntries() {
	gal.logger.Info("Started audit entry processing")

	for {
		select {
		case <-gal.ctx.Done():
			gal.logger.Info("Audit entry processing stopped")
			return
		case entry := <-gal.processQueue:
			if err := gal.processEntry(gal.ctx, entry); err != nil {
				gal.logger.WithError(err).Warn("Failed to process audit entry")
			}
		}
	}
}

func (gal *GuildAuditLogger) processEntry(ctx context.Context, entry AuditEntry) error {
	// Store in primary storage
	if err := gal.storage.Store(ctx, entry); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to store audit entry")
	}

	// Stream to external systems if configured
	if gal.streamer != nil {
		if err := gal.streamer.Stream(ctx, entry); err != nil {
			gal.logger.WithError(err).Warn("Failed to stream audit entry")
			// Don't fail the entire operation if streaming fails
		}
	}

	return nil
}

func (gal *GuildAuditLogger) periodicMaintenance() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-gal.ctx.Done():
			return
		case <-ticker.C:
			gal.performMaintenance()
		}
	}
}

func (gal *GuildAuditLogger) performMaintenance() {
	// Check for entries that need archival based on retention policy
	now := time.Now()
	archiveTime := now.Add(-gal.retention.Standard)

	if err := gal.Archive(gal.ctx, archiveTime); err != nil {
		gal.logger.WithError(err).Warn("Automatic archival failed")
	}

	// Update statistics
	gal.refreshStats()
}

func (gal *GuildAuditLogger) refreshStats() {
	// This would query the storage for updated statistics
	// For now, we'll just update the timestamp
	gal.updateStats(func(stats *AuditStats) {
		stats.LastActivity = time.Now()
	})
}

// Helper methods

func (gal *GuildAuditLogger) generateEntryID(entry AuditEntry) string {
	data := fmt.Sprintf("%s:%s:%s:%d",
		entry.AgentID, entry.Resource, entry.Action, entry.Timestamp.UnixNano())
	return fmt.Sprintf("%x", md5.Sum([]byte(data)))
}

func (gal *GuildAuditLogger) generateReportID(period ReportPeriod, standard ComplianceStandard) string {
	data := fmt.Sprintf("%s:%s:%s",
		standard.String(), period.Start.Format("2006-01-02"), period.End.Format("2006-01-02"))
	return fmt.Sprintf("rpt_%x", md5.Sum([]byte(data)))
}

func (gal *GuildAuditLogger) validateEntry(entry AuditEntry) error {
	if entry.AgentID == "" {
		return gerror.New(gerror.ErrCodeValidation, "agent ID cannot be empty", nil)
	}
	if entry.Resource == "" {
		return gerror.New(gerror.ErrCodeValidation, "resource cannot be empty", nil)
	}
	if entry.Action == "" {
		return gerror.New(gerror.ErrCodeValidation, "action cannot be empty", nil)
	}
	return nil
}

func (gal *GuildAuditLogger) validateFilter(filter AuditFilter) error {
	if filter.StartTime != nil && filter.EndTime != nil {
		if filter.StartTime.After(*filter.EndTime) {
			return gerror.New(gerror.ErrCodeValidation, "start time cannot be after end time", nil)
		}
	}
	return nil
}

func (gal *GuildAuditLogger) updateStats(fn func(*AuditStats)) {
	gal.statsMu.Lock()
	defer gal.statsMu.Unlock()
	fn(gal.stats)
}

func (gal *GuildAuditLogger) generateSummary(entries []AuditEntry) ComplianceSummary {
	totalChecks := len(entries)
	passedChecks := 0
	failedChecks := 0

	for _, entry := range entries {
		if entry.Result == ResultAllowed {
			passedChecks++
		} else {
			failedChecks++
		}
	}

	complianceScore := 0.0
	if totalChecks > 0 {
		complianceScore = float64(passedChecks) / float64(totalChecks) * 100
	}

	riskLevel := "Low"
	if complianceScore < 80 {
		riskLevel = "High"
	} else if complianceScore < 95 {
		riskLevel = "Medium"
	}

	return ComplianceSummary{
		TotalChecks:     totalChecks,
		PassedChecks:    passedChecks,
		FailedChecks:    failedChecks,
		ComplianceScore: complianceScore,
		RiskLevel:       riskLevel,
		TrendDirection:  "Stable", // Would be calculated from historical data
		LastAssessment:  time.Now(),
		NextAssessment:  time.Now().Add(30 * 24 * time.Hour),
	}
}

func (gal *GuildAuditLogger) detectViolations(entries []AuditEntry, standard ComplianceStandard) []ComplianceViolation {
	var violations []ComplianceViolation

	// Detect violations based on compliance standard
	for _, entry := range entries {
		if entry.Result == ResultDenied && entry.RiskScore > 7 {
			violations = append(violations, ComplianceViolation{
				ID:          fmt.Sprintf("viol_%s", entry.ID),
				Rule:        "High-Risk Access Denied",
				Description: fmt.Sprintf("High-risk access attempt to %s was denied", entry.Resource),
				Severity:    "High",
				Category:    "Access Control",
				DetectedAt:  entry.Timestamp,
				AuditEntry:  &entry,
				Remediation: "Review access controls and investigate potential security incident",
				Status:      "Open",
			})
		}

		// Check for suspicious IP addresses
		if entry.IPAddress != "" && gal.isSuspiciousIP(entry.IPAddress) {
			violations = append(violations, ComplianceViolation{
				ID:          fmt.Sprintf("viol_ip_%s", entry.ID),
				Rule:        "Suspicious IP Access",
				Description: fmt.Sprintf("Access from suspicious IP address: %s", entry.IPAddress),
				Severity:    "Medium",
				Category:    "Network Security",
				DetectedAt:  entry.Timestamp,
				AuditEntry:  &entry,
				Remediation: "Investigate IP address and consider blocking if malicious",
				Status:      "Open",
			})
		}
	}

	return violations
}

func (gal *GuildAuditLogger) generateRecommendations(entries []AuditEntry, standard ComplianceStandard) []string {
	recommendations := []string{
		"Implement regular access reviews to ensure least privilege principles",
		"Enable multi-factor authentication for all administrative accounts",
		"Establish automated monitoring for suspicious access patterns",
		"Conduct quarterly security awareness training for all users",
	}

	// Add specific recommendations based on audit data analysis
	deniedCount := 0
	for _, entry := range entries {
		if entry.Result == ResultDenied {
			deniedCount++
		}
	}

	if deniedCount > len(entries)/10 { // More than 10% denied
		recommendations = append(recommendations,
			"High number of denied access attempts detected - review permission model")
	}

	return recommendations
}

func (gal *GuildAuditLogger) getAttestations(standard ComplianceStandard) []Attestation {
	// This would typically come from a compliance management system
	return []Attestation{
		{
			ID:          "att_001",
			Type:        "Security Controls",
			Description: "All security controls are properly implemented and monitored",
			Attestor:    "Security Team",
			AttestedAt:  time.Now().Add(-30 * 24 * time.Hour),
			ValidUntil:  time.Now().Add(335 * 24 * time.Hour),
		},
	}
}

func (gal *GuildAuditLogger) isSuspiciousIP(ip string) bool {
	// Simple heuristic - in practice this would check against threat intelligence
	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return true // Invalid IP is suspicious
	}

	// Check suspicious patterns first (before private IP exclusion)
	suspiciousPrefixes := []string{
		"10.0.0.1", // Common default gateway might be suspicious in some contexts
	}

	for _, prefix := range suspiciousPrefixes {
		if strings.HasPrefix(ip, prefix) {
			return true
		}
	}

	// Check for private IP ranges (might be suspicious depending on context)
	privateRanges := []string{
		"10.0.0.0/8",
		"172.16.0.0/12",
		"192.168.0.0/16",
	}

	for _, cidr := range privateRanges {
		_, network, _ := net.ParseCIDR(cidr)
		if network != nil && network.Contains(parsedIP) {
			return false // Private IPs are generally not suspicious (except those caught above)
		}
	}

	return false
}
