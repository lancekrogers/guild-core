package telemetry

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/guild-framework/guild-core/pkg/gerror"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.30.0"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// setupOTelProviders configures OpenTelemetry providers and exporters
func setupOTelProviders(ctx context.Context, cfg Config) (shutdown func(context.Context) error, err error) {
	var shutdownFuncs []func(context.Context) error

	// Create resource
	res, err := createResource(cfg)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create resource")
	}

	// Setup propagator
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	// Setup trace provider
	if cfg.OTLPEndpoint != "" || cfg.JaegerEndpoint != "" {
		traceShutdown, err := setupTraceProvider(ctx, cfg, res)
		if err != nil {
			return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to setup trace provider")
		}
		shutdownFuncs = append(shutdownFuncs, traceShutdown)
	}

	// Setup metric provider
	metricShutdown, err := setupMetricProvider(ctx, cfg, res)
	if err != nil {
		// Cleanup trace provider if metric setup fails
		for _, fn := range shutdownFuncs {
			_ = fn(ctx)
		}
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to setup metric provider")
	}
	shutdownFuncs = append(shutdownFuncs, metricShutdown)

	// Return composite shutdown function
	shutdown = func(ctx context.Context) error {
		var errs []error
		for _, fn := range shutdownFuncs {
			if err := fn(ctx); err != nil {
				errs = append(errs, err)
			}
		}
		if len(errs) > 0 {
			return gerror.New(gerror.ErrCodeInternal, fmt.Sprintf("shutdown errors: %v", errs), nil)
		}
		return nil
	}

	return shutdown, nil
}

// createResource creates the OTEL resource
func createResource(cfg Config) (*resource.Resource, error) {
	return resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceName(cfg.ServiceName),
		semconv.ServiceVersion(cfg.ServiceVersion),
		semconv.ServiceNamespaceKey.String(cfg.Environment),
	), nil
}

// setupTraceProvider configures the trace provider with exporters
func setupTraceProvider(ctx context.Context, cfg Config, res *resource.Resource) (func(context.Context) error, error) {
	var exporter trace.SpanExporter
	var err error

	// Prefer OTLP over Jaeger
	if cfg.OTLPEndpoint != "" {
		exporter, err = createOTLPTraceExporter(ctx, cfg.OTLPEndpoint)
	} else if cfg.JaegerEndpoint != "" {
		// Use OTLP exporter with Jaeger endpoint (Jaeger supports OTLP)
		exporter, err = createOTLPTraceExporter(ctx, cfg.JaegerEndpoint)
	}

	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create trace exporter")
	}

	provider := trace.NewTracerProvider(
		trace.WithBatcher(exporter,
			trace.WithMaxExportBatchSize(512),
			trace.WithBatchTimeout(2*time.Second),
		),
		trace.WithResource(res),
		trace.WithSampler(trace.TraceIDRatioBased(cfg.SampleRate)),
	)

	otel.SetTracerProvider(provider)

	return func(ctx context.Context) error {
		if err := provider.Shutdown(ctx); err != nil {
			return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to shutdown trace provider")
		}
		return nil
	}, nil
}

// setupMetricProvider configures the metric provider with exporters
func setupMetricProvider(ctx context.Context, cfg Config, res *resource.Resource) (func(context.Context) error, error) {
	var readers []metric.Reader

	// Setup Prometheus exporter if endpoint is configured
	if cfg.PrometheusEndpoint != "" {
		promExporter, err := prometheus.New()
		if err != nil {
			return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create prometheus exporter")
		}
		readers = append(readers, promExporter)

		// Start Prometheus HTTP server
		go func() {
			mux := http.NewServeMux()
			mux.Handle("/metrics", promhttp.Handler())
			server := &http.Server{
				Addr:              cfg.PrometheusEndpoint,
				Handler:           mux,
				ReadHeaderTimeout: 10 * time.Second,
			}
			if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				// Log error but don't fail the setup
				fmt.Printf("prometheus server error: %v\n", err)
			}
		}()
	}

	// Setup OTLP exporter if endpoint is configured
	if cfg.OTLPEndpoint != "" {
		otlpExporter, err := createOTLPMetricExporter(ctx, cfg.OTLPEndpoint)
		if err != nil {
			return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create OTLP metric exporter")
		}

		reader := metric.NewPeriodicReader(otlpExporter,
			metric.WithInterval(10*time.Second),
		)
		readers = append(readers, reader)
	}

	// Create metric provider with all readers
	opts := []metric.Option{metric.WithResource(res)}
	for _, reader := range readers {
		opts = append(opts, metric.WithReader(reader))
	}
	provider := metric.NewMeterProvider(opts...)

	otel.SetMeterProvider(provider)

	return func(ctx context.Context) error {
		if err := provider.Shutdown(ctx); err != nil {
			return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to shutdown metric provider")
		}
		return nil
	}, nil
}

// createOTLPTraceExporter creates an OTLP trace exporter
func createOTLPTraceExporter(ctx context.Context, endpoint string) (trace.SpanExporter, error) {
	conn, err := grpc.DialContext(ctx, endpoint,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
		grpc.WithTimeout(5*time.Second),
	)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to connect to OTLP endpoint")
	}

	return otlptracegrpc.New(ctx,
		otlptracegrpc.WithGRPCConn(conn),
		otlptracegrpc.WithTimeout(30*time.Second),
	)
}

// createOTLPMetricExporter creates an OTLP metric exporter
func createOTLPMetricExporter(ctx context.Context, endpoint string) (metric.Exporter, error) {
	conn, err := grpc.DialContext(ctx, endpoint,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
		grpc.WithTimeout(5*time.Second),
	)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to connect to OTLP endpoint")
	}

	return otlpmetricgrpc.New(ctx,
		otlpmetricgrpc.WithGRPCConn(conn),
		otlpmetricgrpc.WithTimeout(30*time.Second),
	)
}

// NoopTelemetry provides a no-op implementation for testing
type NoopTelemetry struct {
	*Telemetry
}

// NewNoop creates a no-op telemetry instance
func NewNoop() *NoopTelemetry {
	// Create minimal providers
	tp := trace.NewTracerProvider()
	mp := metric.NewMeterProvider()

	otel.SetTracerProvider(tp)
	otel.SetMeterProvider(mp)

	t := &Telemetry{
		meter:  mp.Meter("noop"),
		tracer: tp.Tracer("noop"),
	}

	// Initialize all metrics to avoid nil pointer dereference
	if err := t.initializeMetrics(); err != nil {
		// For noop, we can safely ignore errors
		// as these are just for testing
	}

	return &NoopTelemetry{
		Telemetry: t,
	}
}
