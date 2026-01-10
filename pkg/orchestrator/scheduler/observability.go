// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package scheduler

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/lancekrogers/guild-core/pkg/gerror"
)

// MetricsCollector collects and exposes scheduler metrics
type MetricsCollector struct {
	// Task metrics
	tasksSubmitted atomic.Int64
	tasksCompleted atomic.Int64
	tasksFailed    atomic.Int64
	tasksCancelled atomic.Int64
	tasksInQueue   atomic.Int64
	tasksRunning   atomic.Int64

	// Timing metrics
	taskLatencies  *LatencyTracker
	queueWaitTimes *LatencyTracker
	executionTimes *LatencyTracker

	// Agent metrics
	agentMetrics map[string]*AgentMetrics
	agentMu      sync.RWMutex

	// Resource metrics
	cpuUsage      atomic.Int64 // in millicores
	memoryUsage   atomic.Int64 // in MB
	apiQuotaUsage map[string]*QuotaMetrics
	quotaMu       sync.RWMutex

	// Error metrics
	errorsByCode map[gerror.ErrorCode]*atomic.Int64
	errorMu      sync.RWMutex

	// Circuit breaker metrics
	circuitOpenCount atomic.Int64
	circuitHalfOpen  atomic.Int64
}

// AgentMetrics tracks metrics for individual agents
type AgentMetrics struct {
	TasksAssigned      atomic.Int64
	TasksCompleted     atomic.Int64
	TasksFailed        atomic.Int64
	TotalExecutionTime atomic.Int64 // in milliseconds
	CurrentLoad        atomic.Int32
	LastActivity       atomic.Int64 // unix timestamp
}

// QuotaMetrics tracks API quota usage
type QuotaMetrics struct {
	Provider       string
	RequestsPerMin atomic.Int64
	TokensPerMin   atomic.Int64
	LastReset      atomic.Int64 // unix timestamp
	QuotaExceeded  atomic.Int64
}

// LatencyTracker tracks latency distributions
type LatencyTracker struct {
	count       atomic.Int64
	sum         atomic.Int64 // in microseconds
	min         atomic.Int64
	max         atomic.Int64
	buckets     []atomic.Int64 // histogram buckets
	bucketEdges []int64        // bucket boundaries in microseconds
	mu          sync.RWMutex
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector() *MetricsCollector {
	return &MetricsCollector{
		taskLatencies:  NewLatencyTracker(),
		queueWaitTimes: NewLatencyTracker(),
		executionTimes: NewLatencyTracker(),
		agentMetrics:   make(map[string]*AgentMetrics),
		apiQuotaUsage:  make(map[string]*QuotaMetrics),
		errorsByCode:   make(map[gerror.ErrorCode]*atomic.Int64),
	}
}

// NewLatencyTracker creates a new latency tracker with default buckets
func NewLatencyTracker() *LatencyTracker {
	// Buckets: 0-1ms, 1-10ms, 10-50ms, 50-100ms, 100-500ms, 500ms-1s, 1s-5s, 5s+
	bucketEdges := []int64{
		1000,    // 1ms
		10000,   // 10ms
		50000,   // 50ms
		100000,  // 100ms
		500000,  // 500ms
		1000000, // 1s
		5000000, // 5s
	}

	lt := &LatencyTracker{
		bucketEdges: bucketEdges,
		buckets:     make([]atomic.Int64, len(bucketEdges)+1),
	}

	lt.min.Store(int64(^uint64(0) >> 1)) // Max int64
	return lt
}

// RecordLatency records a latency measurement
func (lt *LatencyTracker) RecordLatency(duration time.Duration) {
	micros := duration.Microseconds()

	lt.count.Add(1)
	lt.sum.Add(micros)

	// Update min/max
	for {
		oldMin := lt.min.Load()
		if micros >= oldMin || lt.min.CompareAndSwap(oldMin, micros) {
			break
		}
	}

	for {
		oldMax := lt.max.Load()
		if micros <= oldMax || lt.max.CompareAndSwap(oldMax, micros) {
			break
		}
	}

	// Update histogram
	bucketIdx := len(lt.buckets) - 1
	for i, edge := range lt.bucketEdges {
		if micros <= edge {
			bucketIdx = i
			break
		}
	}
	lt.buckets[bucketIdx].Add(1)
}

// GetStats returns latency statistics
func (lt *LatencyTracker) GetStats() LatencyStats {
	count := lt.count.Load()
	if count == 0 {
		return LatencyStats{}
	}

	sum := lt.sum.Load()
	avg := time.Duration(sum/count) * time.Microsecond

	// Calculate percentiles from histogram
	bucketCounts := make([]int64, len(lt.buckets))
	for i := range lt.buckets {
		bucketCounts[i] = lt.buckets[i].Load()
	}

	return LatencyStats{
		Count:   count,
		Average: avg,
		Min:     time.Duration(lt.min.Load()) * time.Microsecond,
		Max:     time.Duration(lt.max.Load()) * time.Microsecond,
		P50:     lt.calculatePercentile(bucketCounts, 0.50),
		P90:     lt.calculatePercentile(bucketCounts, 0.90),
		P95:     lt.calculatePercentile(bucketCounts, 0.95),
		P99:     lt.calculatePercentile(bucketCounts, 0.99),
	}
}

// LatencyStats contains latency statistics
type LatencyStats struct {
	Count   int64
	Average time.Duration
	Min     time.Duration
	Max     time.Duration
	P50     time.Duration
	P90     time.Duration
	P95     time.Duration
	P99     time.Duration
}

func (lt *LatencyTracker) calculatePercentile(bucketCounts []int64, percentile float64) time.Duration {
	total := int64(0)
	for _, count := range bucketCounts {
		total += count
	}

	if total == 0 {
		return 0
	}

	target := int64(float64(total) * percentile)
	cumulative := int64(0)

	for i, count := range bucketCounts {
		cumulative += count
		if cumulative >= target {
			// Estimate within bucket
			if i == 0 {
				return time.Duration(lt.bucketEdges[0]/2) * time.Microsecond
			} else if i < len(lt.bucketEdges) {
				lower := lt.bucketEdges[i-1]
				upper := lt.bucketEdges[i]
				return time.Duration((lower+upper)/2) * time.Microsecond
			} else {
				// Last bucket
				return time.Duration(lt.bucketEdges[len(lt.bucketEdges)-1]) * time.Microsecond
			}
		}
	}

	return time.Duration(lt.max.Load()) * time.Microsecond
}

// RecordTaskSubmitted records a task submission
func (mc *MetricsCollector) RecordTaskSubmitted() {
	mc.tasksSubmitted.Add(1)
	mc.tasksInQueue.Add(1)
}

// RecordTaskStarted records task execution start
func (mc *MetricsCollector) RecordTaskStarted(taskID string, agentID string, queueTime time.Duration) {
	mc.tasksInQueue.Add(-1)
	mc.tasksRunning.Add(1)
	mc.queueWaitTimes.RecordLatency(queueTime)

	mc.getOrCreateAgentMetrics(agentID).TasksAssigned.Add(1)
	mc.getOrCreateAgentMetrics(agentID).CurrentLoad.Add(1)
}

// RecordTaskCompleted records successful task completion
func (mc *MetricsCollector) RecordTaskCompleted(taskID string, agentID string, executionTime time.Duration) {
	mc.tasksRunning.Add(-1)
	mc.tasksCompleted.Add(1)
	mc.executionTimes.RecordLatency(executionTime)

	agent := mc.getOrCreateAgentMetrics(agentID)
	agent.TasksCompleted.Add(1)
	agent.CurrentLoad.Add(-1)
	agent.TotalExecutionTime.Add(executionTime.Milliseconds())
	agent.LastActivity.Store(time.Now().Unix())
}

// RecordTaskFailed records task failure
func (mc *MetricsCollector) RecordTaskFailed(taskID string, agentID string, err error, executionTime time.Duration) {
	mc.tasksRunning.Add(-1)
	mc.tasksFailed.Add(1)
	mc.executionTimes.RecordLatency(executionTime)

	agent := mc.getOrCreateAgentMetrics(agentID)
	agent.TasksFailed.Add(1)
	agent.CurrentLoad.Add(-1)
	agent.LastActivity.Store(time.Now().Unix())

	// Record error by code
	if gerr, ok := err.(*gerror.GuildError); ok {
		mc.errorMu.Lock()
		if _, exists := mc.errorsByCode[gerr.Code]; !exists {
			mc.errorsByCode[gerr.Code] = &atomic.Int64{}
		}
		mc.errorMu.Unlock()
		mc.errorsByCode[gerr.Code].Add(1)
	}
}

// RecordTaskCancelled records task cancellation
func (mc *MetricsCollector) RecordTaskCancelled(taskID string) {
	mc.tasksRunning.Add(-1)
	mc.tasksCancelled.Add(1)
}

// RecordAPIQuotaUsage records API quota usage
func (mc *MetricsCollector) RecordAPIQuotaUsage(provider string, requests int, tokens int) {
	mc.quotaMu.Lock()
	quota, exists := mc.apiQuotaUsage[provider]
	if !exists {
		quota = &QuotaMetrics{
			Provider: provider,
		}
		mc.apiQuotaUsage[provider] = quota
		quota.LastReset.Store(time.Now().Unix())
	}
	mc.quotaMu.Unlock()

	quota.RequestsPerMin.Add(int64(requests))
	quota.TokensPerMin.Add(int64(tokens))
}

// RecordQuotaExceeded records when API quota is exceeded
func (mc *MetricsCollector) RecordQuotaExceeded(provider string) {
	mc.quotaMu.RLock()
	quota, exists := mc.apiQuotaUsage[provider]
	mc.quotaMu.RUnlock()

	if exists {
		quota.QuotaExceeded.Add(1)
	}
}

// RecordCircuitBreakerState records circuit breaker state changes
func (mc *MetricsCollector) RecordCircuitBreakerState(agentID string, state CircuitState) {
	switch state {
	case CircuitOpen:
		mc.circuitOpenCount.Add(1)
	case CircuitHalfOpen:
		mc.circuitHalfOpen.Add(1)
	}
}

// GetMetrics returns current metrics snapshot
func (mc *MetricsCollector) GetMetrics() *MetricsSnapshot {
	snapshot := &MetricsSnapshot{
		Timestamp: time.Now(),
		Tasks: TaskMetrics{
			Submitted: mc.tasksSubmitted.Load(),
			Completed: mc.tasksCompleted.Load(),
			Failed:    mc.tasksFailed.Load(),
			Cancelled: mc.tasksCancelled.Load(),
			InQueue:   mc.tasksInQueue.Load(),
			Running:   mc.tasksRunning.Load(),
		},
		Latencies: LatencyMetrics{
			TaskLatency:   mc.taskLatencies.GetStats(),
			QueueWaitTime: mc.queueWaitTimes.GetStats(),
			ExecutionTime: mc.executionTimes.GetStats(),
		},
		Agents:   mc.getAgentMetricsSnapshot(),
		APIQuota: mc.getQuotaMetricsSnapshot(),
		Errors:   mc.getErrorMetricsSnapshot(),
		CircuitBreakers: CircuitBreakerMetrics{
			OpenCount:     mc.circuitOpenCount.Load(),
			HalfOpenCount: mc.circuitHalfOpen.Load(),
		},
	}

	// Calculate derived metrics
	if snapshot.Tasks.Submitted > 0 {
		snapshot.Tasks.SuccessRate = float64(snapshot.Tasks.Completed) / float64(snapshot.Tasks.Submitted) * 100
	}

	return snapshot
}

// MetricsSnapshot represents a point-in-time metrics snapshot
type MetricsSnapshot struct {
	Timestamp       time.Time
	Tasks           TaskMetrics
	Latencies       LatencyMetrics
	Agents          map[string]AgentMetricsSnapshot
	APIQuota        map[string]QuotaMetricsSnapshot
	Errors          map[string]int64
	CircuitBreakers CircuitBreakerMetrics
}

// TaskMetrics contains task-related metrics
type TaskMetrics struct {
	Submitted   int64
	Completed   int64
	Failed      int64
	Cancelled   int64
	InQueue     int64
	Running     int64
	SuccessRate float64
}

// LatencyMetrics contains latency distributions
type LatencyMetrics struct {
	TaskLatency   LatencyStats
	QueueWaitTime LatencyStats
	ExecutionTime LatencyStats
}

// AgentMetricsSnapshot contains agent metrics
type AgentMetricsSnapshot struct {
	TasksAssigned        int64
	TasksCompleted       int64
	TasksFailed          int64
	CurrentLoad          int32
	AverageExecutionTime time.Duration
	LastActivity         time.Time
	Utilization          float64
}

// QuotaMetricsSnapshot contains API quota metrics
type QuotaMetricsSnapshot struct {
	Provider       string
	RequestsPerMin int64
	TokensPerMin   int64
	QuotaExceeded  int64
	LastReset      time.Time
}

// CircuitBreakerMetrics contains circuit breaker metrics
type CircuitBreakerMetrics struct {
	OpenCount     int64
	HalfOpenCount int64
}

func (mc *MetricsCollector) getOrCreateAgentMetrics(agentID string) *AgentMetrics {
	mc.agentMu.RLock()
	metrics, exists := mc.agentMetrics[agentID]
	mc.agentMu.RUnlock()

	if exists {
		return metrics
	}

	mc.agentMu.Lock()
	defer mc.agentMu.Unlock()

	// Double-check after acquiring write lock
	if metrics, exists = mc.agentMetrics[agentID]; exists {
		return metrics
	}

	metrics = &AgentMetrics{}
	mc.agentMetrics[agentID] = metrics
	return metrics
}

func (mc *MetricsCollector) getAgentMetricsSnapshot() map[string]AgentMetricsSnapshot {
	mc.agentMu.RLock()
	defer mc.agentMu.RUnlock()

	snapshot := make(map[string]AgentMetricsSnapshot)

	for agentID, metrics := range mc.agentMetrics {
		completed := metrics.TasksCompleted.Load()
		var avgExecTime time.Duration
		if completed > 0 {
			avgExecTime = time.Duration(metrics.TotalExecutionTime.Load()/completed) * time.Millisecond
		}

		assigned := metrics.TasksAssigned.Load()
		var utilization float64
		if assigned > 0 {
			utilization = float64(completed) / float64(assigned) * 100
		}

		snapshot[agentID] = AgentMetricsSnapshot{
			TasksAssigned:        assigned,
			TasksCompleted:       completed,
			TasksFailed:          metrics.TasksFailed.Load(),
			CurrentLoad:          metrics.CurrentLoad.Load(),
			AverageExecutionTime: avgExecTime,
			LastActivity:         time.Unix(metrics.LastActivity.Load(), 0),
			Utilization:          utilization,
		}
	}

	return snapshot
}

func (mc *MetricsCollector) getQuotaMetricsSnapshot() map[string]QuotaMetricsSnapshot {
	mc.quotaMu.RLock()
	defer mc.quotaMu.RUnlock()

	snapshot := make(map[string]QuotaMetricsSnapshot)

	for provider, metrics := range mc.apiQuotaUsage {
		snapshot[provider] = QuotaMetricsSnapshot{
			Provider:       provider,
			RequestsPerMin: metrics.RequestsPerMin.Load(),
			TokensPerMin:   metrics.TokensPerMin.Load(),
			QuotaExceeded:  metrics.QuotaExceeded.Load(),
			LastReset:      time.Unix(metrics.LastReset.Load(), 0),
		}
	}

	return snapshot
}

func (mc *MetricsCollector) getErrorMetricsSnapshot() map[string]int64 {
	mc.errorMu.RLock()
	defer mc.errorMu.RUnlock()

	snapshot := make(map[string]int64)

	for code, count := range mc.errorsByCode {
		snapshot[string(code)] = count.Load()
	}

	return snapshot
}

// ResetQuotaMetrics resets quota metrics (should be called periodically)
func (mc *MetricsCollector) ResetQuotaMetrics() {
	mc.quotaMu.Lock()
	defer mc.quotaMu.Unlock()

	now := time.Now().Unix()
	for _, quota := range mc.apiQuotaUsage {
		quota.RequestsPerMin.Store(0)
		quota.TokensPerMin.Store(0)
		quota.LastReset.Store(now)
	}
}

// TraceContext provides distributed tracing support
type TraceContext struct {
	TraceID      string
	SpanID       string
	ParentSpanID string
	Baggage      map[string]string
}

// SpanRecorder records trace spans
type SpanRecorder interface {
	StartSpan(ctx context.Context, operation string, attributes map[string]interface{}) (context.Context, func())
	RecordError(ctx context.Context, err error)
	SetAttributes(ctx context.Context, attributes map[string]interface{})
}

// NoOpSpanRecorder is a no-op implementation
type NoOpSpanRecorder struct{}

func (n *NoOpSpanRecorder) StartSpan(ctx context.Context, operation string, attributes map[string]interface{}) (context.Context, func()) {
	return ctx, func() {}
}

func (n *NoOpSpanRecorder) RecordError(ctx context.Context, err error) {}

func (n *NoOpSpanRecorder) SetAttributes(ctx context.Context, attributes map[string]interface{}) {}

// EventLogger logs significant events
type EventLogger interface {
	LogEvent(ctx context.Context, event Event)
}

// Event represents a significant scheduler event
type Event struct {
	Type       EventType
	Timestamp  time.Time
	TaskID     string
	AgentID    string
	Attributes map[string]interface{}
}

// EventType represents the type of event
type EventType string

const (
	EventTaskSubmitted     EventType = "task.submitted"
	EventTaskStarted       EventType = "task.started"
	EventTaskCompleted     EventType = "task.completed"
	EventTaskFailed        EventType = "task.failed"
	EventTaskRetried       EventType = "task.retried"
	EventTaskDeadLettered  EventType = "task.dead_lettered"
	EventAgentRegistered   EventType = "core.registered"
	EventAgentUnregistered EventType = "core.unregistered"
	EventAgentHealthChange EventType = "core.health_changed"
	EventCircuitOpened     EventType = "circuit.opened"
	EventCircuitClosed     EventType = "circuit.closed"
	EventQuotaExceeded     EventType = "quota.exceeded"
)

// NoOpEventLogger is a no-op implementation
type NoOpEventLogger struct{}

func (n *NoOpEventLogger) LogEvent(ctx context.Context, event Event) {}

// MetricsExporter exports metrics to external systems
type MetricsExporter interface {
	Export(ctx context.Context, metrics *MetricsSnapshot) error
}

// PrometheusExporter exports metrics in Prometheus format
type PrometheusExporter struct {
	endpoint string
}

func NewPrometheusExporter(endpoint string) *PrometheusExporter {
	return &PrometheusExporter{endpoint: endpoint}
}

func (pe *PrometheusExporter) Export(ctx context.Context, metrics *MetricsSnapshot) error {
	// This would normally export to Prometheus
	// For now, it's a placeholder
	return nil
}

// MetricsFormatters provides metric formatting utilities
func FormatMetricsAsText(metrics *MetricsSnapshot) string {
	return fmt.Sprintf(`Scheduler Metrics Report
========================
Timestamp: %s

Task Metrics:
  Submitted: %d
  Completed: %d (%.2f%% success rate)
  Failed: %d
  Cancelled: %d
  In Queue: %d
  Running: %d

Latency Metrics:
  Queue Wait Time:
    Average: %v, P50: %v, P90: %v, P99: %v
  Execution Time:
    Average: %v, P50: %v, P90: %v, P99: %v

Agent Metrics:
%s

API Quota Usage:
%s

Circuit Breakers:
  Open: %d
  Half-Open: %d
`,
		metrics.Timestamp.Format(time.RFC3339),
		metrics.Tasks.Submitted,
		metrics.Tasks.Completed,
		metrics.Tasks.SuccessRate,
		metrics.Tasks.Failed,
		metrics.Tasks.Cancelled,
		metrics.Tasks.InQueue,
		metrics.Tasks.Running,
		metrics.Latencies.QueueWaitTime.Average,
		metrics.Latencies.QueueWaitTime.P50,
		metrics.Latencies.QueueWaitTime.P90,
		metrics.Latencies.QueueWaitTime.P99,
		metrics.Latencies.ExecutionTime.Average,
		metrics.Latencies.ExecutionTime.P50,
		metrics.Latencies.ExecutionTime.P90,
		metrics.Latencies.ExecutionTime.P99,
		formatAgentMetrics(metrics.Agents),
		formatQuotaMetrics(metrics.APIQuota),
		metrics.CircuitBreakers.OpenCount,
		metrics.CircuitBreakers.HalfOpenCount,
	)
}

func formatAgentMetrics(agents map[string]AgentMetricsSnapshot) string {
	if len(agents) == 0 {
		return "  No agents registered"
	}

	result := ""
	for agentID, metrics := range agents {
		result += fmt.Sprintf("  %s: Tasks=%d/%d (%.2f%% utilization), Load=%d, Avg Time=%v\n",
			agentID,
			metrics.TasksCompleted,
			metrics.TasksAssigned,
			metrics.Utilization,
			metrics.CurrentLoad,
			metrics.AverageExecutionTime,
		)
	}
	return result
}

func formatQuotaMetrics(quotas map[string]QuotaMetricsSnapshot) string {
	if len(quotas) == 0 {
		return "  No API quota tracking"
	}

	result := ""
	for provider, metrics := range quotas {
		result += fmt.Sprintf("  %s: %d req/min, %d tokens/min, %d quota exceeded\n",
			provider,
			metrics.RequestsPerMin,
			metrics.TokensPerMin,
			metrics.QuotaExceeded,
		)
	}
	return result
}
