package observability

import (
	"fmt"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// MetricsRegistry wraps Prometheus registry
type MetricsRegistry struct {
	registry *prometheus.Registry

	// Core metrics
	requestDuration *prometheus.HistogramVec
	requestTotal    *prometheus.CounterVec
	errorTotal      *prometheus.CounterVec
	activeRequests  *prometheus.GaugeVec

	// Agent metrics
	agentTaskTotal    *prometheus.CounterVec
	agentTaskDuration *prometheus.HistogramVec
	agentTokenUsage   *prometheus.CounterVec
	agentCost         *prometheus.CounterVec
	agentUtilization  *prometheus.GaugeVec

	// Task metrics
	taskQueueSize *prometheus.GaugeVec
	taskProcessed *prometheus.CounterVec
	taskDuration  *prometheus.HistogramVec
	taskRetries   *prometheus.CounterVec

	// Storage metrics
	storageOperations *prometheus.CounterVec
	storageDuration   *prometheus.HistogramVec
	storageErrors     *prometheus.CounterVec

	// Provider metrics
	providerRequests *prometheus.CounterVec
	providerDuration *prometheus.HistogramVec
	providerTokens   *prometheus.CounterVec
	providerCost     *prometheus.CounterVec
	providerErrors   *prometheus.CounterVec
}

// MetricsConfig holds metrics configuration
type MetricsConfig struct {
	Namespace   string
	Subsystem   string
	ServiceName string
	Enabled     bool
}

// DefaultMetricsConfig returns default metrics configuration
func DefaultMetricsConfig() *MetricsConfig {
	return &MetricsConfig{
		Namespace:   "guild",
		Subsystem:   "",
		ServiceName: getEnv("GUILD_SERVICE", "guild"),
		Enabled:     getEnv("GUILD_METRICS_ENABLED", "true") == "true",
	}
}

// InitMetrics initializes Prometheus metrics
func InitMetrics(config *MetricsConfig) *MetricsRegistry {
	if config == nil {
		config = DefaultMetricsConfig()
	}

	if !config.Enabled {
		// Return no-op registry
		return &MetricsRegistry{
			registry: prometheus.NewRegistry(),
		}
	}

	registry := prometheus.NewRegistry()

	// Register default collectors
	registry.MustRegister(
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
	)

	m := &MetricsRegistry{
		registry: registry,
	}

	// Initialize core metrics
	m.requestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: config.Namespace,
			Subsystem: config.Subsystem,
			Name:      "request_duration_seconds",
			Help:      "Request duration in seconds",
			Buckets:   prometheus.DefBuckets,
		},
		[]string{"method", "endpoint", "status"},
	)

	m.requestTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: config.Namespace,
			Subsystem: config.Subsystem,
			Name:      "request_total",
			Help:      "Total number of requests",
		},
		[]string{"method", "endpoint", "status"},
	)

	m.errorTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: config.Namespace,
			Subsystem: config.Subsystem,
			Name:      "error_total",
			Help:      "Total number of errors",
		},
		[]string{"code", "component", "operation"},
	)

	m.activeRequests = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: config.Namespace,
			Subsystem: config.Subsystem,
			Name:      "active_requests",
			Help:      "Number of active requests",
		},
		[]string{"endpoint"},
	)

	// Initialize agent metrics
	m.agentTaskTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: config.Namespace,
			Subsystem: "agent",
			Name:      "task_total",
			Help:      "Total number of tasks processed by agents",
		},
		[]string{"agent_id", "agent_type", "status"},
	)

	m.agentTaskDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: config.Namespace,
			Subsystem: "agent",
			Name:      "task_duration_seconds",
			Help:      "Task processing duration by agents",
			Buckets:   prometheus.DefBuckets,
		},
		[]string{"agent_id", "agent_type"},
	)

	m.agentTokenUsage = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: config.Namespace,
			Subsystem: "agent",
			Name:      "token_usage_total",
			Help:      "Total tokens used by agents",
		},
		[]string{"agent_id", "agent_type", "token_type"},
	)

	m.agentCost = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: config.Namespace,
			Subsystem: "agent",
			Name:      "cost_dollars",
			Help:      "Total cost in dollars by agents",
		},
		[]string{"agent_id", "agent_type", "provider"},
	)

	m.agentUtilization = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: config.Namespace,
			Subsystem: "agent",
			Name:      "utilization_ratio",
			Help:      "Agent utilization ratio (0-1)",
		},
		[]string{"agent_id", "agent_type"},
	)

	// Initialize task metrics
	m.taskQueueSize = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: config.Namespace,
			Subsystem: "task",
			Name:      "queue_size",
			Help:      "Number of tasks in queue",
		},
		[]string{"status"},
	)

	m.taskProcessed = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: config.Namespace,
			Subsystem: "task",
			Name:      "processed_total",
			Help:      "Total number of tasks processed",
		},
		[]string{"status", "campaign_id"},
	)

	m.taskDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: config.Namespace,
			Subsystem: "task",
			Name:      "duration_seconds",
			Help:      "Task execution duration",
			Buckets:   prometheus.DefBuckets,
		},
		[]string{"task_type"},
	)

	m.taskRetries = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: config.Namespace,
			Subsystem: "task",
			Name:      "retries_total",
			Help:      "Total number of task retries",
		},
		[]string{"reason"},
	)

	// Initialize storage metrics
	m.storageOperations = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: config.Namespace,
			Subsystem: "storage",
			Name:      "operations_total",
			Help:      "Total number of storage operations",
		},
		[]string{"operation", "table", "status"},
	)

	m.storageDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: config.Namespace,
			Subsystem: "storage",
			Name:      "operation_duration_seconds",
			Help:      "Storage operation duration",
			Buckets:   prometheus.ExponentialBuckets(0.001, 2, 10), // 1ms to ~1s
		},
		[]string{"operation", "table"},
	)

	m.storageErrors = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: config.Namespace,
			Subsystem: "storage",
			Name:      "errors_total",
			Help:      "Total number of storage errors",
		},
		[]string{"operation", "table", "error_type"},
	)

	// Initialize provider metrics
	m.providerRequests = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: config.Namespace,
			Subsystem: "provider",
			Name:      "requests_total",
			Help:      "Total number of provider requests",
		},
		[]string{"provider", "model", "status"},
	)

	m.providerDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: config.Namespace,
			Subsystem: "provider",
			Name:      "request_duration_seconds",
			Help:      "Provider request duration",
			Buckets:   prometheus.DefBuckets,
		},
		[]string{"provider", "model"},
	)

	m.providerTokens = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: config.Namespace,
			Subsystem: "provider",
			Name:      "tokens_total",
			Help:      "Total tokens used by provider",
		},
		[]string{"provider", "model", "token_type"},
	)

	m.providerCost = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: config.Namespace,
			Subsystem: "provider",
			Name:      "cost_dollars",
			Help:      "Total cost in dollars by provider",
		},
		[]string{"provider", "model"},
	)

	m.providerErrors = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: config.Namespace,
			Subsystem: "provider",
			Name:      "errors_total",
			Help:      "Total number of provider errors",
		},
		[]string{"provider", "model", "error_type"},
	)

	// Register all metrics
	registry.MustRegister(
		m.requestDuration, m.requestTotal, m.errorTotal, m.activeRequests,
		m.agentTaskTotal, m.agentTaskDuration, m.agentTokenUsage, m.agentCost, m.agentUtilization,
		m.taskQueueSize, m.taskProcessed, m.taskDuration, m.taskRetries,
		m.storageOperations, m.storageDuration, m.storageErrors,
		m.providerRequests, m.providerDuration, m.providerTokens, m.providerCost, m.providerErrors,
	)

	return m
}

// Handler returns the Prometheus HTTP handler
func (m *MetricsRegistry) Handler() http.Handler {
	return promhttp.HandlerFor(m.registry, promhttp.HandlerOpts{
		Registry: m.registry,
	})
}

// Core metric methods

func (m *MetricsRegistry) RecordRequest(method, endpoint, status string, duration time.Duration) {
	if m.requestDuration != nil {
		m.requestDuration.WithLabelValues(method, endpoint, status).Observe(duration.Seconds())
		m.requestTotal.WithLabelValues(method, endpoint, status).Inc()
	}
}

func (m *MetricsRegistry) RecordError(code, component, operation string) {
	if m.errorTotal != nil {
		m.errorTotal.WithLabelValues(code, component, operation).Inc()
	}
}

func (m *MetricsRegistry) SetActiveRequests(endpoint string, count float64) {
	if m.activeRequests != nil {
		m.activeRequests.WithLabelValues(endpoint).Set(count)
	}
}

// Agent metric methods

func (m *MetricsRegistry) RecordAgentTask(agentID, agentType, status string) {
	if m.agentTaskTotal != nil {
		m.agentTaskTotal.WithLabelValues(agentID, agentType, status).Inc()
	}
}

func (m *MetricsRegistry) RecordAgentTaskDuration(agentID, agentType string, duration time.Duration) {
	if m.agentTaskDuration != nil {
		m.agentTaskDuration.WithLabelValues(agentID, agentType).Observe(duration.Seconds())
	}
}

func (m *MetricsRegistry) RecordAgentTokenUsage(agentID, agentType, tokenType string, count int) {
	if m.agentTokenUsage != nil {
		m.agentTokenUsage.WithLabelValues(agentID, agentType, tokenType).Add(float64(count))
	}
}

func (m *MetricsRegistry) RecordAgentCost(agentID, agentType, provider string, cost float64) {
	if m.agentCost != nil {
		m.agentCost.WithLabelValues(agentID, agentType, provider).Add(cost)
	}
}

func (m *MetricsRegistry) SetAgentUtilization(agentID, agentType string, utilization float64) {
	if m.agentUtilization != nil {
		m.agentUtilization.WithLabelValues(agentID, agentType).Set(utilization)
	}
}

// Task metric methods

func (m *MetricsRegistry) SetTaskQueueSize(status string, size int) {
	if m.taskQueueSize != nil {
		m.taskQueueSize.WithLabelValues(status).Set(float64(size))
	}
}

func (m *MetricsRegistry) RecordTaskProcessed(status, campaignID string) {
	if m.taskProcessed != nil {
		m.taskProcessed.WithLabelValues(status, campaignID).Inc()
	}
}

func (m *MetricsRegistry) RecordTaskDuration(taskType string, duration time.Duration) {
	if m.taskDuration != nil {
		m.taskDuration.WithLabelValues(taskType).Observe(duration.Seconds())
	}
}

func (m *MetricsRegistry) RecordTaskRetry(reason string) {
	if m.taskRetries != nil {
		m.taskRetries.WithLabelValues(reason).Inc()
	}
}

// Storage metric methods

func (m *MetricsRegistry) RecordStorageOperation(operation, table, status string) {
	if m.storageOperations != nil {
		m.storageOperations.WithLabelValues(operation, table, status).Inc()
	}
}

func (m *MetricsRegistry) RecordStorageDuration(operation, table string, duration time.Duration) {
	if m.storageDuration != nil {
		m.storageDuration.WithLabelValues(operation, table).Observe(duration.Seconds())
	}
}

func (m *MetricsRegistry) RecordStorageError(operation, table, errorType string) {
	if m.storageErrors != nil {
		m.storageErrors.WithLabelValues(operation, table, errorType).Inc()
	}
}

// Provider metric methods

func (m *MetricsRegistry) RecordProviderRequest(provider, model, status string) {
	if m.providerRequests != nil {
		m.providerRequests.WithLabelValues(provider, model, status).Inc()
	}
}

func (m *MetricsRegistry) RecordProviderDuration(provider, model string, duration time.Duration) {
	if m.providerDuration != nil {
		m.providerDuration.WithLabelValues(provider, model).Observe(duration.Seconds())
	}
}

func (m *MetricsRegistry) RecordProviderTokens(provider, model, tokenType string, count int) {
	if m.providerTokens != nil {
		m.providerTokens.WithLabelValues(provider, model, tokenType).Add(float64(count))
	}
}

func (m *MetricsRegistry) RecordProviderCost(provider, model string, cost float64) {
	if m.providerCost != nil {
		m.providerCost.WithLabelValues(provider, model).Add(cost)
	}
}

func (m *MetricsRegistry) RecordProviderError(provider, model, errorType string) {
	if m.providerErrors != nil {
		m.providerErrors.WithLabelValues(provider, model, errorType).Inc()
	}
}

// Global metrics instance
var globalMetrics *MetricsRegistry

// InitGlobalMetrics initializes the global metrics instance
func InitGlobalMetrics(config *MetricsConfig) *MetricsRegistry {
	globalMetrics = InitMetrics(config)
	return globalMetrics
}

// GetMetrics returns the global metrics instance
func GetMetrics() *MetricsRegistry {
	if globalMetrics == nil {
		globalMetrics = InitMetrics(nil)
	}
	return globalMetrics
}

// MetricsMiddleware is HTTP middleware for recording request metrics
func MetricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Track active requests
		metrics := GetMetrics()
		metrics.SetActiveRequests(r.URL.Path, 1)
		defer metrics.SetActiveRequests(r.URL.Path, -1)

		// Wrap response writer to capture status code
		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		// Process request
		next.ServeHTTP(wrapped, r)

		// Record metrics
		duration := time.Since(start)
		status := fmt.Sprintf("%d", wrapped.statusCode)
		metrics.RecordRequest(r.Method, r.URL.Path, status, duration)
	})
}

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}
