// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package audit

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// Mock implementations

type MockAuditStorage struct {
	mock.Mock
}

func (m *MockAuditStorage) Store(ctx context.Context, entry AuditEntry) error {
	args := m.Called(ctx, entry)
	return args.Error(0)
}

func (m *MockAuditStorage) Retrieve(ctx context.Context, filter AuditFilter) ([]AuditEntry, error) {
	args := m.Called(ctx, filter)
	return args.Get(0).([]AuditEntry), args.Error(1)
}

func (m *MockAuditStorage) Count(ctx context.Context, filter AuditFilter) (int64, error) {
	args := m.Called(ctx, filter)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockAuditStorage) Delete(ctx context.Context, before time.Time) error {
	args := m.Called(ctx, before)
	return args.Error(0)
}

func (m *MockAuditStorage) Backup(ctx context.Context, destination string) error {
	args := m.Called(ctx, destination)
	return args.Error(0)
}

func (m *MockAuditStorage) Health(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

type MockAuditStreamer struct {
	mock.Mock
}

func (m *MockAuditStreamer) Stream(ctx context.Context, entry AuditEntry) error {
	args := m.Called(ctx, entry)
	return args.Error(0)
}

func (m *MockAuditStreamer) AddDestination(destination StreamDestination) error {
	args := m.Called(destination)
	return args.Error(0)
}

func (m *MockAuditStreamer) RemoveDestination(destinationID string) error {
	args := m.Called(destinationID)
	return args.Error(0)
}

func (m *MockAuditStreamer) GetDestinations() []StreamDestination {
	args := m.Called()
	return args.Get(0).([]StreamDestination)
}

type MockAuditEnricher struct {
	mock.Mock
}

func (m *MockAuditEnricher) Enrich(ctx context.Context, entry AuditEntry) (AuditEntry, error) {
	args := m.Called(ctx, entry)
	return args.Get(0).(AuditEntry), args.Error(1)
}

func (m *MockAuditEnricher) CalculateRiskScore(ctx context.Context, entry AuditEntry) (int, error) {
	args := m.Called(ctx, entry)
	return args.Int(0), args.Error(1)
}

func (m *MockAuditEnricher) AddComplianceTags(ctx context.Context, entry AuditEntry) ([]string, error) {
	args := m.Called(ctx, entry)
	return args.Get(0).([]string), args.Error(1)
}

// Test cases

func TestCraftGuildAuditLogger_NewGuildAuditLogger(t *testing.T) {
	ctx := context.Background()
	mockStorage := &MockAuditStorage{}
	config := DefaultAuditConfig()

	logger, err := NewGuildAuditLogger(ctx, mockStorage, config)
	require.NoError(t, err)
	assert.NotNil(t, logger)
	assert.Equal(t, mockStorage, logger.storage)
	assert.Equal(t, config.QueueSize, cap(logger.processQueue))

	// Clean up
	logger.Close()
}

func TestGuildGuildAuditLogger_NewGuildAuditLogger_NilStorage(t *testing.T) {
	ctx := context.Background()
	config := DefaultAuditConfig()

	logger, err := NewGuildAuditLogger(ctx, nil, config)
	assert.Error(t, err)
	assert.Nil(t, logger)
	assert.Contains(t, err.Error(), "storage cannot be nil")
}

func TestJourneymanGuildAuditLogger_LogEntry_Success(t *testing.T) {
	ctx := context.Background()
	mockStorage := &MockAuditStorage{}
	config := DefaultAuditConfig()
	config.QueueSize = 1000 // Ensure queue doesn't fill up

	logger, err := NewGuildAuditLogger(ctx, mockStorage, config)
	require.NoError(t, err)
	defer logger.Close()

	// Set up mock expectations
	mockStorage.On("Store", mock.Anything, mock.MatchedBy(func(entry AuditEntry) bool {
		return entry.AgentID == "agent1" && entry.Resource == "file:/test" && entry.Action == "read"
	})).Return(nil)

	// Log entry
	entry := AuditEntry{
		AgentID:   "agent1",
		Resource:  "file:/test",
		Action:    "read",
		Result:    ResultAllowed,
		Timestamp: time.Now(),
	}

	err = logger.LogEntry(ctx, entry)
	require.NoError(t, err)

	// Give time for background processing
	time.Sleep(100 * time.Millisecond)
	mockStorage.AssertExpectations(t)
}

func TestCraftGuildAuditLogger_LogEntry_WithEnricher(t *testing.T) {
	ctx := context.Background()
	mockStorage := &MockAuditStorage{}
	mockEnricher := &MockAuditEnricher{}

	config := DefaultAuditConfig()
	config.Enricher = mockEnricher
	config.QueueSize = 1000

	logger, err := NewGuildAuditLogger(ctx, mockStorage, config)
	require.NoError(t, err)
	defer logger.Close()

	// Set up enricher expectations
	enrichedEntry := AuditEntry{
		AgentID:    "agent1",
		Resource:   "file:/test",
		Action:     "read",
		Result:     ResultAllowed,
		RiskScore:  5,
		Compliance: []string{"SOC2"},
		Timestamp:  time.Now(),
	}

	mockEnricher.On("Enrich", mock.Anything, mock.MatchedBy(func(entry AuditEntry) bool {
		return entry.AgentID == "agent1"
	})).Return(enrichedEntry, nil)

	mockStorage.On("Store", mock.Anything, mock.MatchedBy(func(entry AuditEntry) bool {
		return entry.RiskScore == 5 && len(entry.Compliance) == 1
	})).Return(nil)

	// Log entry
	entry := AuditEntry{
		AgentID:   "agent1",
		Resource:  "file:/test",
		Action:    "read",
		Result:    ResultAllowed,
		Timestamp: time.Now(),
	}

	err = logger.LogEntry(ctx, entry)
	require.NoError(t, err)

	// Give time for background processing
	time.Sleep(100 * time.Millisecond)
	mockStorage.AssertExpectations(t)
	mockEnricher.AssertExpectations(t)
}

func TestGuildGuildAuditLogger_LogEntry_WithStreamer(t *testing.T) {
	ctx := context.Background()
	mockStorage := &MockAuditStorage{}
	mockStreamer := &MockAuditStreamer{}

	config := DefaultAuditConfig()
	config.Streamer = mockStreamer
	config.QueueSize = 1000

	logger, err := NewGuildAuditLogger(ctx, mockStorage, config)
	require.NoError(t, err)
	defer logger.Close()

	// Set up expectations
	mockStorage.On("Store", mock.Anything, mock.Anything).Return(nil)
	mockStreamer.On("Stream", mock.Anything, mock.MatchedBy(func(entry AuditEntry) bool {
		return entry.AgentID == "agent1"
	})).Return(nil)

	// Log entry
	entry := AuditEntry{
		AgentID:   "agent1",
		Resource:  "file:/test",
		Action:    "read",
		Result:    ResultAllowed,
		Timestamp: time.Now(),
	}

	err = logger.LogEntry(ctx, entry)
	require.NoError(t, err)

	// Give time for background processing
	time.Sleep(100 * time.Millisecond)
	mockStorage.AssertExpectations(t)
	mockStreamer.AssertExpectations(t)
}

func TestJourneymanGuildAuditLogger_LogEntry_Validation(t *testing.T) {
	ctx := context.Background()
	mockStorage := &MockAuditStorage{}
	config := DefaultAuditConfig()

	logger, err := NewGuildAuditLogger(ctx, mockStorage, config)
	require.NoError(t, err)
	defer logger.Close()

	tests := []struct {
		name    string
		entry   AuditEntry
		wantErr bool
		errMsg  string
	}{
		{
			name: "missing agent ID",
			entry: AuditEntry{
				Resource: "file:/test",
				Action:   "read",
				Result:   ResultAllowed,
			},
			wantErr: true,
			errMsg:  "agent ID cannot be empty",
		},
		{
			name: "missing resource",
			entry: AuditEntry{
				AgentID: "agent1",
				Action:  "read",
				Result:  ResultAllowed,
			},
			wantErr: true,
			errMsg:  "resource cannot be empty",
		},
		{
			name: "missing action",
			entry: AuditEntry{
				AgentID:  "agent1",
				Resource: "file:/test",
				Result:   ResultAllowed,
			},
			wantErr: true,
			errMsg:  "action cannot be empty",
		},
		{
			name: "valid entry",
			entry: AuditEntry{
				AgentID:  "agent1",
				Resource: "file:/test",
				Action:   "read",
				Result:   ResultAllowed,
			},
			wantErr: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := logger.LogEntry(ctx, test.entry)
			if test.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), test.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestCraftGuildAuditLogger_Query(t *testing.T) {
	ctx := context.Background()
	mockStorage := &MockAuditStorage{}
	config := DefaultAuditConfig()
	config.QueueSize = 0 // Force synchronous processing

	logger, err := NewGuildAuditLogger(ctx, mockStorage, config)
	require.NoError(t, err)
	defer logger.Close()

	expectedEntries := []AuditEntry{
		{ID: "1", AgentID: "agent1", Resource: "file:/test", Action: "read", Result: ResultAllowed},
		{ID: "2", AgentID: "agent1", Resource: "file:/test2", Action: "write", Result: ResultDenied},
	}

	filter := AuditFilter{
		AgentID: "agent1",
		Limit:   100,
	}

	mockStorage.On("Retrieve", mock.Anything, mock.MatchedBy(func(f AuditFilter) bool {
		return f.AgentID == "agent1" && f.Limit == 100
	})).Return(expectedEntries, nil)

	entries, err := logger.Query(ctx, filter)
	require.NoError(t, err)
	assert.Equal(t, expectedEntries, entries)
	mockStorage.AssertExpectations(t)
}

func TestGuildGuildAuditLogger_Query_Limits(t *testing.T) {
	ctx := context.Background()
	mockStorage := &MockAuditStorage{}
	config := DefaultAuditConfig()
	config.QueueSize = 0 // Force synchronous processing

	logger, err := NewGuildAuditLogger(ctx, mockStorage, config)
	require.NoError(t, err)
	defer logger.Close()

	tests := []struct {
		name        string
		inputLimit  int
		expectLimit int
	}{
		{"no limit", 0, 1000},
		{"small limit", 50, 50},
		{"excessive limit", 50000, 10000},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			filter := AuditFilter{Limit: test.inputLimit}

			mockStorage.On("Retrieve", mock.Anything, mock.MatchedBy(func(f AuditFilter) bool {
				return f.Limit == test.expectLimit
			})).Return([]AuditEntry{}, nil).Once()

			_, err := logger.Query(ctx, filter)
			require.NoError(t, err)
		})
	}

	mockStorage.AssertExpectations(t)
}

func TestJourneymanGuildAuditLogger_GetStats(t *testing.T) {
	ctx := context.Background()
	mockStorage := &MockAuditStorage{}
	config := DefaultAuditConfig()
	config.QueueSize = 0 // Force synchronous processing

	logger, err := NewGuildAuditLogger(ctx, mockStorage, config)
	require.NoError(t, err)
	defer logger.Close()

	// Update stats by logging some entries
	logger.updateStats(func(stats *AuditStats) {
		stats.TotalEntries = 100
		stats.AllowedActions = 80
		stats.DeniedActions = 15
		stats.ErrorActions = 5
	})

	stats, err := logger.GetStats(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(100), stats.TotalEntries)
	assert.Equal(t, int64(80), stats.AllowedActions)
	assert.Equal(t, int64(15), stats.DeniedActions)
	assert.Equal(t, int64(5), stats.ErrorActions)
}

func TestCraftGuildAuditLogger_GenerateReport(t *testing.T) {
	ctx := context.Background()
	mockStorage := &MockAuditStorage{}
	config := DefaultAuditConfig()
	config.QueueSize = 0 // Force synchronous processing

	logger, err := NewGuildAuditLogger(ctx, mockStorage, config)
	require.NoError(t, err)
	defer logger.Close()

	// Mock audit entries for report generation
	entries := []AuditEntry{
		{ID: "1", AgentID: "agent1", Resource: "file:/test", Action: "read", Result: ResultAllowed, RiskScore: 2},
		{ID: "2", AgentID: "agent1", Resource: "file:/sensitive", Action: "read", Result: ResultDenied, RiskScore: 8},
		{ID: "3", AgentID: "agent2", Resource: "file:/public", Action: "read", Result: ResultAllowed, RiskScore: 1},
	}

	period := ReportPeriod{
		Start: time.Now().Add(-24 * time.Hour),
		End:   time.Now(),
		Name:  "Last 24 Hours",
	}

	mockStorage.On("Retrieve", mock.Anything, mock.MatchedBy(func(f AuditFilter) bool {
		return f.StartTime != nil && f.EndTime != nil && f.Limit == 100000
	})).Return(entries, nil)

	report, err := logger.GenerateReport(ctx, period, ComplianceSOC2)
	require.NoError(t, err)
	assert.NotEmpty(t, report.ID)
	assert.Equal(t, period, report.Period)
	assert.Equal(t, ComplianceSOC2, report.Compliance)
	assert.Equal(t, 3, report.Summary.TotalChecks)
	assert.Equal(t, 2, report.Summary.PassedChecks)
	assert.Equal(t, 1, report.Summary.FailedChecks)
	assert.InDelta(t, 66.67, report.Summary.ComplianceScore, 0.1)
	assert.NotEmpty(t, report.Violations)
	assert.NotEmpty(t, report.Recommendations)

	mockStorage.AssertExpectations(t)
}

func TestGuildGuildAuditLogger_Archive(t *testing.T) {
	ctx := context.Background()
	mockStorage := &MockAuditStorage{}
	config := DefaultAuditConfig()
	config.QueueSize = 0 // Force synchronous processing

	logger, err := NewGuildAuditLogger(ctx, mockStorage, config)
	require.NoError(t, err)
	defer logger.Close()

	archiveTime := time.Now().Add(-90 * 24 * time.Hour)

	// Mock storage methods for archival
	mockStorage.On("Backup", mock.Anything, mock.MatchedBy(func(path string) bool {
		return path != ""
	})).Return(nil)

	mockStorage.On("Delete", mock.Anything, archiveTime).Return(nil)

	err = logger.Archive(ctx, archiveTime)
	require.NoError(t, err)
	mockStorage.AssertExpectations(t)
}

func TestJourneymanGuildAuditLogger_Close(t *testing.T) {
	ctx := context.Background()
	mockStorage := &MockAuditStorage{}
	config := DefaultAuditConfig()
	config.QueueSize = 10

	logger, err := NewGuildAuditLogger(ctx, mockStorage, config)
	require.NoError(t, err)

	// Add some entries to queue before closing
	mockStorage.On("Store", mock.Anything, mock.Anything).Return(nil).Maybe()

	entry := AuditEntry{
		AgentID:  "agent1",
		Resource: "file:/test",
		Action:   "read",
		Result:   ResultAllowed,
	}

	logger.LogEntry(ctx, entry)

	// Close should process remaining entries
	err = logger.Close()
	assert.NoError(t, err)
}

func TestCraftGuildAuditLogger_ViolationDetection(t *testing.T) {
	ctx := context.Background()
	mockStorage := &MockAuditStorage{}
	config := DefaultAuditConfig()
	config.QueueSize = 0 // Force synchronous processing

	logger, err := NewGuildAuditLogger(ctx, mockStorage, config)
	require.NoError(t, err)
	defer logger.Close()

	// Test high-risk violation detection
	entries := []AuditEntry{
		{ID: "high-risk", Result: ResultDenied, RiskScore: 9, IPAddress: "192.168.1.100"},
		{ID: "suspicious-ip", Result: ResultAllowed, RiskScore: 3, IPAddress: "10.0.0.1"},
		{ID: "normal", Result: ResultAllowed, RiskScore: 2, IPAddress: "192.168.1.101"},
	}

	violations := logger.detectViolations(entries, ComplianceSOC2)

	// Should detect high-risk and suspicious IP violations
	assert.Len(t, violations, 2)

	highRiskFound := false
	suspiciousIPFound := false
	for _, violation := range violations {
		if violation.Rule == "High-Risk Access Denied" {
			highRiskFound = true
		}
		if violation.Rule == "Suspicious IP Access" {
			suspiciousIPFound = true
		}
	}

	assert.True(t, highRiskFound, "Should detect high-risk violation")
	assert.True(t, suspiciousIPFound, "Should detect suspicious IP violation")
}

func TestGuildGuildAuditLogger_RecommendationGeneration(t *testing.T) {
	ctx := context.Background()
	mockStorage := &MockAuditStorage{}
	config := DefaultAuditConfig()
	config.QueueSize = 0 // Force synchronous processing

	logger, err := NewGuildAuditLogger(ctx, mockStorage, config)
	require.NoError(t, err)
	defer logger.Close()

	// Test with high denial rate
	entries := make([]AuditEntry, 100)
	for i := 0; i < 100; i++ {
		result := ResultAllowed
		if i < 20 { // 20% denied
			result = ResultDenied
		}
		entries[i] = AuditEntry{
			ID:     fmt.Sprintf("entry-%d", i),
			Result: result,
		}
	}

	recommendations := logger.generateRecommendations(entries, ComplianceSOC2)

	// Should include standard recommendations plus high denial rate recommendation
	assert.Greater(t, len(recommendations), 4)

	found := false
	for _, rec := range recommendations {
		if rec == "High number of denied access attempts detected - review permission model" {
			found = true
			break
		}
	}
	assert.True(t, found, "Should recommend reviewing permission model for high denial rate")
}

func TestJourneymanGuildAuditLogger_ContextCancellation(t *testing.T) {
	ctx := context.Background()
	mockStorage := &MockAuditStorage{}
	config := DefaultAuditConfig()
	config.QueueSize = 0 // Force synchronous processing

	logger, err := NewGuildAuditLogger(ctx, mockStorage, config)
	require.NoError(t, err)
	defer logger.Close()

	// Create cancelled context
	cancelledCtx, cancel := context.WithCancel(ctx)
	cancel()

	entry := AuditEntry{
		AgentID:  "agent1",
		Resource: "file:/test",
		Action:   "read",
		Result:   ResultAllowed,
	}

	// Should handle cancellation gracefully
	err = logger.LogEntry(cancelledCtx, entry)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "context cancelled")
}

func TestCraftGuildAuditLogger_DefaultConfig(t *testing.T) {
	config := DefaultAuditConfig()

	assert.Equal(t, 10000, config.QueueSize)
	assert.Equal(t, 100, config.BatchSize)
	assert.Equal(t, 5*time.Second, config.FlushInterval)
	assert.Equal(t, 90*24*time.Hour, config.RetentionPolicy.Standard)
	assert.Equal(t, 365*24*time.Hour, config.RetentionPolicy.HighRisk)
	assert.Equal(t, 7*365*24*time.Hour, config.RetentionPolicy.Compliance)
	assert.Equal(t, 10*365*24*time.Hour, config.RetentionPolicy.Incidents)
}

// Benchmark tests

func BenchmarkCraftGuildAuditLogger_LogEntry(b *testing.B) {
	ctx := context.Background()
	mockStorage := &MockAuditStorage{}
	config := DefaultAuditConfig()
	config.QueueSize = 100000 // Large queue to avoid blocking

	logger, err := NewGuildAuditLogger(ctx, mockStorage, config)
	require.NoError(b, err)
	defer logger.Close()

	// Mock storage to avoid actual I/O
	mockStorage.On("Store", mock.Anything, mock.Anything).Return(nil)

	entry := AuditEntry{
		AgentID:  "agent1",
		Resource: "file:/project/main.go",
		Action:   "read",
		Result:   ResultAllowed,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		entry.ID = fmt.Sprintf("entry-%d", i)
		logger.LogEntry(ctx, entry)
	}
}

func BenchmarkGuildGuildAuditLogger_GenerateReport(b *testing.B) {
	ctx := context.Background()
	mockStorage := &MockAuditStorage{}
	config := DefaultAuditConfig()

	logger, err := NewGuildAuditLogger(ctx, mockStorage, config)
	require.NoError(b, err)
	defer logger.Close()

	// Create sample entries for report generation
	entries := make([]AuditEntry, 1000)
	for i := 0; i < 1000; i++ {
		entries[i] = AuditEntry{
			ID:       fmt.Sprintf("entry-%d", i),
			AgentID:  "agent1",
			Resource: "file:/test",
			Action:   "read",
			Result:   ResultAllowed,
		}
	}

	period := ReportPeriod{
		Start: time.Now().Add(-24 * time.Hour),
		End:   time.Now(),
	}

	mockStorage.On("Retrieve", mock.Anything, mock.Anything).Return(entries, nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.GenerateReport(ctx, period, ComplianceSOC2)
	}
}
