// Package observability provides production-ready observability for the Guild framework.
//
// It includes:
//   - Structured logging with context propagation
//   - Distributed tracing with OpenTelemetry
//   - Metrics collection with Prometheus
//   - Request tracking and correlation
//   - Error tracking integration
//
// # Basic Usage
//
//	// Initialize observability
//	logger := observability.NewLogger(nil)
//	tracer, _ := observability.InitTracing(ctx, nil)
//	metrics := observability.InitGlobalMetrics(nil)
//
//	// Use in your code
//	ctx = observability.EnsureRequestContext(ctx)
//	logger.InfoContext(ctx, "Processing request")
//
//	// Trace operations
//	ctx, span := observability.StartSpan(ctx, "process_task")
//	defer span.End()
//
//	// Record metrics
//	metrics.RecordRequest("POST", "/api/tasks", "200", time.Since(start))
//
// # Context Propagation
//
// The package ensures consistent context propagation for request tracking:
//
//	ctx = observability.WithRequestID(ctx, requestID)
//	ctx = observability.WithComponent(ctx, "task-processor")
//	ctx = observability.WithOperation(ctx, "create-task")
//
// # Error Integration
//
// Works seamlessly with the gerror package:
//
//	err := gerror.New(gerror.ErrCodeValidation, "invalid input", nil)
//	logger.WithError(err).ErrorContext(ctx, "Failed to process request")
//	observability.RecordError(ctx, err) // Records in trace
//
// # Configuration
//
// Configuration via environment variables:
//   - GUILD_ENV: Environment (development, staging, production)
//   - GUILD_SERVICE: Service name
//   - GUILD_VERSION: Service version
//   - GUILD_LOG_LEVEL: Log level (debug, info, warn, error)
//   - GUILD_TRACING_ENABLED: Enable tracing (true/false)
//   - GUILD_METRICS_ENABLED: Enable metrics (true/false)
//   - OTEL_EXPORTER_OTLP_ENDPOINT: OpenTelemetry endpoint
package observability
