// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package audit

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCraftAuditResult_String(t *testing.T) {
	tests := []struct {
		result   AuditResult
		expected string
	}{
		{ResultAllowed, "allowed"},
		{ResultDenied, "denied"},
		{ResultError, "error"},
		{ResultTimeout, "timeout"},
		{ResultBlocked, "blocked"},
		{AuditResult(999), "unknown"}, // Invalid result
	}

	for _, test := range tests {
		t.Run(test.expected, func(t *testing.T) {
			assert.Equal(t, test.expected, test.result.String())
		})
	}
}

func TestGuildComplianceStandard_String(t *testing.T) {
	tests := []struct {
		standard ComplianceStandard
		expected string
	}{
		{ComplianceSOC2, "SOC 2"},
		{ComplianceISO27001, "ISO 27001"},
		{ComplianceGDPR, "GDPR"},
		{ComplianceHIPAA, "HIPAA"},
		{CompliancePCIDSS, "PCI DSS"},
		{ComplianceFedRAMP, "FedRAMP"},
		{ComplianceCustom, "Custom"},
		{ComplianceStandard(999), "Unknown"}, // Invalid standard
	}

	for _, test := range tests {
		t.Run(test.expected, func(t *testing.T) {
			assert.Equal(t, test.expected, test.standard.String())
		})
	}
}

func TestJourneymanDefaultRetentionPolicy(t *testing.T) {
	policy := DefaultRetentionPolicy()

	assert.Equal(t, 90*24*time.Hour, policy.Standard)
	assert.Equal(t, 365*24*time.Hour, policy.HighRisk)
	assert.Equal(t, 7*365*24*time.Hour, policy.Compliance)
	assert.Equal(t, 10*365*24*time.Hour, policy.Incidents)
}

func TestCraftAuditEntry_RequiredFields(t *testing.T) {
	now := time.Now()

	entry := AuditEntry{
		ID:        "test-entry",
		Timestamp: now,
		AgentID:   "agent1",
		Resource:  "file:/project/main.go",
		Action:    "read",
		Result:    ResultAllowed,
	}

	// Verify required fields are set
	assert.NotEmpty(t, entry.ID)
	assert.False(t, entry.Timestamp.IsZero())
	assert.NotEmpty(t, entry.AgentID)
	assert.NotEmpty(t, entry.Resource)
	assert.NotEmpty(t, entry.Action)
	assert.Equal(t, ResultAllowed, entry.Result)
}

func TestGuildAuditEntry_OptionalFields(t *testing.T) {
	entry := AuditEntry{
		ID:         "test-entry",
		Timestamp:  time.Now(),
		AgentID:    "agent1",
		UserID:     "user1",
		SessionID:  "session1",
		RequestID:  "request1",
		Resource:   "file:/project/main.go",
		Action:     "read",
		Result:     ResultAllowed,
		Reason:     "permission granted",
		Duration:   100 * time.Millisecond,
		IPAddress:  "192.168.1.100",
		UserAgent:  "GuildAgent/1.0",
		Metadata:   map[string]interface{}{"key": "value"},
		RiskScore:  3,
		Compliance: []string{"SOC2", "ISO27001"},
	}

	// Verify optional fields
	assert.Equal(t, "user1", entry.UserID)
	assert.Equal(t, "session1", entry.SessionID)
	assert.Equal(t, "request1", entry.RequestID)
	assert.Equal(t, "permission granted", entry.Reason)
	assert.Equal(t, 100*time.Millisecond, entry.Duration)
	assert.Equal(t, "192.168.1.100", entry.IPAddress)
	assert.Equal(t, "GuildAgent/1.0", entry.UserAgent)
	assert.Equal(t, "value", entry.Metadata["key"])
	assert.Equal(t, 3, entry.RiskScore)
	assert.Contains(t, entry.Compliance, "SOC2")
	assert.Contains(t, entry.Compliance, "ISO27001")
}

func TestJourneymanAuditFilter_TimeRange(t *testing.T) {
	start := time.Now().Add(-1 * time.Hour)
	end := time.Now()

	filter := AuditFilter{
		StartTime: &start,
		EndTime:   &end,
		AgentID:   "agent1",
		Limit:     100,
	}

	assert.Equal(t, start, *filter.StartTime)
	assert.Equal(t, end, *filter.EndTime)
	assert.Equal(t, "agent1", filter.AgentID)
	assert.Equal(t, 100, filter.Limit)
}

func TestCraftAuditFilter_ResultPointer(t *testing.T) {
	result := ResultDenied
	filter := AuditFilter{
		Result: &result,
	}

	assert.NotNil(t, filter.Result)
	assert.Equal(t, ResultDenied, *filter.Result)
}

func TestGuildAuditStats_Metrics(t *testing.T) {
	stats := AuditStats{
		TotalEntries:     1000,
		AllowedActions:   800,
		DeniedActions:    150,
		ErrorActions:     50,
		UniqueAgents:     10,
		UniqueSessions:   25,
		AverageRiskScore: 3.5,
		HighRiskActions:  75,
		LastActivity:     time.Now(),
		RetentionPeriod:  90 * 24 * time.Hour,
		StorageSize:      1024 * 1024, // 1MB
	}

	assert.Equal(t, int64(1000), stats.TotalEntries)
	assert.Equal(t, int64(800), stats.AllowedActions)
	assert.Equal(t, int64(150), stats.DeniedActions)
	assert.Equal(t, int64(50), stats.ErrorActions)
	assert.Equal(t, int64(10), stats.UniqueAgents)
	assert.Equal(t, int64(25), stats.UniqueSessions)
	assert.Equal(t, 3.5, stats.AverageRiskScore)
	assert.Equal(t, int64(75), stats.HighRiskActions)
	assert.Equal(t, 90*24*time.Hour, stats.RetentionPeriod)
	assert.Equal(t, int64(1024*1024), stats.StorageSize)
}

func TestJourneymanComplianceReport_Structure(t *testing.T) {
	now := time.Now()
	period := ReportPeriod{
		Start: now.Add(-24 * time.Hour),
		End:   now,
		Name:  "Last 24 Hours",
	}

	summary := ComplianceSummary{
		TotalChecks:     100,
		PassedChecks:    85,
		FailedChecks:    15,
		ComplianceScore: 85.0,
		RiskLevel:       "Medium",
		TrendDirection:  "Improving",
		LastAssessment:  now.Add(-1 * time.Hour),
		NextAssessment:  now.Add(23 * time.Hour),
	}

	violation := ComplianceViolation{
		ID:          "viol_001",
		Rule:        "High-Risk Access Denied",
		Description: "High-risk access attempt was denied",
		Severity:    "High",
		Category:    "Access Control",
		DetectedAt:  now,
		Remediation: "Review access controls",
		Status:      "Open",
	}

	attestation := Attestation{
		ID:          "att_001",
		Type:        "Security Controls",
		Description: "All security controls are properly implemented",
		Attestor:    "Security Team",
		AttestedAt:  now.Add(-30 * 24 * time.Hour),
		ValidUntil:  now.Add(335 * 24 * time.Hour),
	}

	report := ComplianceReport{
		ID:              "rpt_001",
		GeneratedAt:     now,
		Period:          period,
		Compliance:      ComplianceSOC2,
		Summary:         summary,
		Violations:      []ComplianceViolation{violation},
		Recommendations: []string{"Implement MFA", "Regular access reviews"},
		Attestations:    []Attestation{attestation},
		Metadata:        map[string]interface{}{"version": "1.0"},
	}

	assert.Equal(t, "rpt_001", report.ID)
	assert.Equal(t, period, report.Period)
	assert.Equal(t, ComplianceSOC2, report.Compliance)
	assert.Equal(t, summary, report.Summary)
	assert.Len(t, report.Violations, 1)
	assert.Equal(t, "viol_001", report.Violations[0].ID)
	assert.Len(t, report.Recommendations, 2)
	assert.Contains(t, report.Recommendations, "Implement MFA")
	assert.Len(t, report.Attestations, 1)
	assert.Equal(t, "att_001", report.Attestations[0].ID)
	assert.Equal(t, "1.0", report.Metadata["version"])
}

func TestCraftStreamDestination_Configuration(t *testing.T) {
	filter := &AuditFilter{
		AgentID: "agent1",
		Result:  func() *AuditResult { r := ResultDenied; return &r }(),
	}

	destination := StreamDestination{
		ID:       "dest_001",
		Type:     "elasticsearch",
		Endpoint: "https://es.example.com:9200",
		Headers: map[string]string{
			"Authorization": "Bearer token123",
			"Content-Type":  "application/json",
		},
		Filter:  filter,
		Enabled: true,
		Retries: 3,
		Timeout: 30 * time.Second,
	}

	assert.Equal(t, "dest_001", destination.ID)
	assert.Equal(t, "elasticsearch", destination.Type)
	assert.Equal(t, "https://es.example.com:9200", destination.Endpoint)
	assert.Equal(t, "Bearer token123", destination.Headers["Authorization"])
	assert.Equal(t, "application/json", destination.Headers["Content-Type"])
	assert.NotNil(t, destination.Filter)
	assert.Equal(t, "agent1", destination.Filter.AgentID)
	assert.Equal(t, ResultDenied, *destination.Filter.Result)
	assert.True(t, destination.Enabled)
	assert.Equal(t, 3, destination.Retries)
	assert.Equal(t, 30*time.Second, destination.Timeout)
}

func TestGuildComplianceViolation_Details(t *testing.T) {
	now := time.Now()
	dueDate := now.Add(7 * 24 * time.Hour)

	auditEntry := &AuditEntry{
		ID:       "entry_001",
		AgentID:  "agent1",
		Resource: "file:/sensitive/data",
		Action:   "read",
		Result:   ResultDenied,
	}

	violation := ComplianceViolation{
		ID:          "viol_001",
		Rule:        "Sensitive Data Access",
		Description: "Unauthorized access attempt to sensitive data",
		Severity:    "Critical",
		Category:    "Data Protection",
		DetectedAt:  now,
		AuditEntry:  auditEntry,
		Remediation: "Investigate and strengthen access controls",
		Status:      "Open",
		Assignee:    "security-team",
		DueDate:     &dueDate,
	}

	assert.Equal(t, "viol_001", violation.ID)
	assert.Equal(t, "Sensitive Data Access", violation.Rule)
	assert.Equal(t, "Critical", violation.Severity)
	assert.Equal(t, "Data Protection", violation.Category)
	assert.NotNil(t, violation.AuditEntry)
	assert.Equal(t, "entry_001", violation.AuditEntry.ID)
	assert.Equal(t, "security-team", violation.Assignee)
	assert.NotNil(t, violation.DueDate)
	assert.Equal(t, dueDate, *violation.DueDate)
}

func TestJourneymanAttestation_Evidence(t *testing.T) {
	attestation := Attestation{
		ID:          "att_001",
		Type:        "Penetration Test",
		Description: "Annual penetration testing completed",
		Attestor:    "External Security Firm",
		AttestedAt:  time.Now().Add(-30 * 24 * time.Hour),
		ValidUntil:  time.Now().Add(335 * 24 * time.Hour),
		Evidence: []string{
			"pentest-report-2025.pdf",
			"remediation-plan.docx",
			"executive-summary.pdf",
		},
		Signature:   "SHA256:abc123...",
		Certificate: "cert-chain.pem",
	}

	assert.Equal(t, "att_001", attestation.ID)
	assert.Equal(t, "Penetration Test", attestation.Type)
	assert.Equal(t, "External Security Firm", attestation.Attestor)
	assert.Len(t, attestation.Evidence, 3)
	assert.Contains(t, attestation.Evidence, "pentest-report-2025.pdf")
	assert.Contains(t, attestation.Evidence, "remediation-plan.docx")
	assert.Contains(t, attestation.Evidence, "executive-summary.pdf")
	assert.Equal(t, "SHA256:abc123...", attestation.Signature)
	assert.Equal(t, "cert-chain.pem", attestation.Certificate)
}

func TestCraftReportPeriod_Quarters(t *testing.T) {
	q1Start := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	q1End := time.Date(2025, 3, 31, 23, 59, 59, 999999999, time.UTC)

	q1Period := ReportPeriod{
		Start: q1Start,
		End:   q1End,
		Name:  "Q1 2025",
	}

	assert.Equal(t, "Q1 2025", q1Period.Name)
	assert.Equal(t, 2025, q1Period.Start.Year())
	assert.Equal(t, time.January, q1Period.Start.Month())
	assert.Equal(t, 1, q1Period.Start.Day())
	assert.Equal(t, 2025, q1Period.End.Year())
	assert.Equal(t, time.March, q1Period.End.Month())
	assert.Equal(t, 31, q1Period.End.Day())
}

func TestGuildRetentionPolicy_CustomValues(t *testing.T) {
	policy := RetentionPolicy{
		Standard:   30 * 24 * time.Hour,      // 30 days
		HighRisk:   180 * 24 * time.Hour,     // 6 months
		Compliance: 5 * 365 * 24 * time.Hour, // 5 years
		Incidents:  7 * 365 * 24 * time.Hour, // 7 years
	}

	assert.Equal(t, 30*24*time.Hour, policy.Standard)
	assert.Equal(t, 180*24*time.Hour, policy.HighRisk)
	assert.Equal(t, 5*365*24*time.Hour, policy.Compliance)
	assert.Equal(t, 7*365*24*time.Hour, policy.Incidents)
}

// Test edge cases and validation

func TestJourneymanAuditEntry_ZeroValues(t *testing.T) {
	var entry AuditEntry

	// Zero values should be detectable
	assert.Empty(t, entry.ID)
	assert.True(t, entry.Timestamp.IsZero())
	assert.Empty(t, entry.AgentID)
	assert.Empty(t, entry.Resource)
	assert.Empty(t, entry.Action)
	assert.Equal(t, ResultAllowed, entry.Result) // Default value
	assert.Equal(t, 0, entry.RiskScore)
	assert.Nil(t, entry.Metadata)
	assert.Empty(t, entry.Compliance)
}

func TestCraftAuditFilter_Pagination(t *testing.T) {
	filter := AuditFilter{
		Limit:     50,
		Offset:    100,
		SortBy:    "timestamp",
		SortOrder: "desc",
	}

	assert.Equal(t, 50, filter.Limit)
	assert.Equal(t, 100, filter.Offset)
	assert.Equal(t, "timestamp", filter.SortBy)
	assert.Equal(t, "desc", filter.SortOrder)
}

func TestGuildComplianceSummary_Calculations(t *testing.T) {
	summary := ComplianceSummary{
		TotalChecks:     100,
		PassedChecks:    87,
		FailedChecks:    13,
		ComplianceScore: 87.0,
		RiskLevel:       "Low",
		TrendDirection:  "Stable",
	}

	// Verify calculation consistency
	assert.Equal(t, summary.PassedChecks+summary.FailedChecks, summary.TotalChecks)
	assert.InDelta(t, float64(summary.PassedChecks)/float64(summary.TotalChecks)*100, summary.ComplianceScore, 0.1)
}

func TestJourneymanAuditFilter_ComplexFiltering(t *testing.T) {
	start := time.Now().Add(-24 * time.Hour)
	end := time.Now()
	result := ResultDenied

	filter := AuditFilter{
		StartTime:    &start,
		EndTime:      &end,
		AgentID:      "agent1",
		UserID:       "user1",
		SessionID:    "session1",
		Resource:     "file:*",
		Action:       "write",
		Result:       &result,
		IPAddress:    "192.168.1.100",
		MinRiskScore: 5,
		Compliance:   []string{"SOC2", "GDPR"},
		Limit:        25,
		Offset:       50,
		SortBy:       "risk_score",
		SortOrder:    "desc",
	}

	assert.NotNil(t, filter.StartTime)
	assert.NotNil(t, filter.EndTime)
	assert.Equal(t, "agent1", filter.AgentID)
	assert.Equal(t, "user1", filter.UserID)
	assert.Equal(t, "session1", filter.SessionID)
	assert.Equal(t, "file:*", filter.Resource)
	assert.Equal(t, "write", filter.Action)
	assert.NotNil(t, filter.Result)
	assert.Equal(t, ResultDenied, *filter.Result)
	assert.Equal(t, "192.168.1.100", filter.IPAddress)
	assert.Equal(t, 5, filter.MinRiskScore)
	assert.Len(t, filter.Compliance, 2)
	assert.Contains(t, filter.Compliance, "SOC2")
	assert.Contains(t, filter.Compliance, "GDPR")
	assert.Equal(t, 25, filter.Limit)
	assert.Equal(t, 50, filter.Offset)
	assert.Equal(t, "risk_score", filter.SortBy)
	assert.Equal(t, "desc", filter.SortOrder)
}
