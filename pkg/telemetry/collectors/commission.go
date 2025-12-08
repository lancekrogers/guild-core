package collectors

import (
	"context"
	"time"

	"github.com/guild-framework/guild-core/pkg/gerror"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// CommissionCollector collects commission execution metrics
type CommissionCollector struct {
	meter               metric.Meter
	executionTime       metric.Float64Histogram
	tasksGenerated      metric.Int64Counter
	tasksCompleted      metric.Int64Counter
	tasksFailed         metric.Int64Counter
	refinements         metric.Int64Counter
	complexityScore     metric.Float64Histogram
	activeCommissions   metric.Int64UpDownCounter
	queueDepth          metric.Int64ObservableGauge
	resourceUtilization metric.Float64Histogram
}

// NewCommissionCollector creates a new commission metrics collector
func NewCommissionCollector(meter metric.Meter) (*CommissionCollector, error) {
	cc := &CommissionCollector{meter: meter}

	if err := cc.initializeMetrics(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to initialize commission metrics")
	}

	return cc, nil
}

// initializeMetrics creates all commission metric instruments
func (cc *CommissionCollector) initializeMetrics() error {
	var err error

	cc.executionTime, err = cc.meter.Float64Histogram(
		"guild.commission.execution.duration",
		metric.WithDescription("Time taken to execute commissions"),
		metric.WithUnit("s"),
	)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create execution time histogram")
	}

	cc.tasksGenerated, err = cc.meter.Int64Counter(
		"guild.commission.tasks.generated",
		metric.WithDescription("Number of tasks generated from commissions"),
	)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create tasks generated counter")
	}

	cc.tasksCompleted, err = cc.meter.Int64Counter(
		"guild.commission.tasks.completed",
		metric.WithDescription("Number of commission tasks completed"),
	)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create tasks completed counter")
	}

	cc.tasksFailed, err = cc.meter.Int64Counter(
		"guild.commission.tasks.failed",
		metric.WithDescription("Number of commission tasks failed"),
	)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create tasks failed counter")
	}

	cc.refinements, err = cc.meter.Int64Counter(
		"guild.commission.refinements",
		metric.WithDescription("Number of commission refinements"),
	)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create refinements counter")
	}

	cc.complexityScore, err = cc.meter.Float64Histogram(
		"guild.commission.complexity.score",
		metric.WithDescription("Complexity score of commissions"),
	)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create complexity score histogram")
	}

	cc.activeCommissions, err = cc.meter.Int64UpDownCounter(
		"guild.commission.active",
		metric.WithDescription("Number of active commissions"),
	)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create active commissions counter")
	}

	cc.queueDepth, err = cc.meter.Int64ObservableGauge(
		"guild.commission.queue.depth",
		metric.WithDescription("Depth of commission queue"),
	)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create queue depth gauge")
	}

	cc.resourceUtilization, err = cc.meter.Float64Histogram(
		"guild.commission.resource.utilization",
		metric.WithDescription("Resource utilization per commission"),
		metric.WithUnit("%"),
	)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create resource utilization histogram")
	}

	return nil
}

// RecordExecution records commission execution metrics
func (cc *CommissionCollector) RecordExecution(ctx context.Context, commissionID string, commissionType string, duration time.Duration, success bool) {
	attrs := []attribute.KeyValue{
		attribute.String("commission.id", commissionID),
		attribute.String("commission.type", commissionType),
		attribute.Bool("success", success),
	}

	cc.executionTime.Record(ctx, duration.Seconds(), metric.WithAttributes(attrs...))
}

// RecordTaskGenerated records task generation
func (cc *CommissionCollector) RecordTaskGenerated(ctx context.Context, commissionID string, taskType string, count int64) {
	attrs := []attribute.KeyValue{
		attribute.String("commission.id", commissionID),
		attribute.String("task.type", taskType),
	}
	cc.tasksGenerated.Add(ctx, count, metric.WithAttributes(attrs...))
}

// RecordTaskCompletion records task completion
func (cc *CommissionCollector) RecordTaskCompletion(ctx context.Context, commissionID string, taskID string, success bool) {
	attrs := []attribute.KeyValue{
		attribute.String("commission.id", commissionID),
		attribute.String("task.id", taskID),
		attribute.Bool("success", success),
	}

	if success {
		cc.tasksCompleted.Add(ctx, 1, metric.WithAttributes(attrs...))
	} else {
		cc.tasksFailed.Add(ctx, 1, metric.WithAttributes(attrs...))
	}
}

// RecordRefinement records commission refinement
func (cc *CommissionCollector) RecordRefinement(ctx context.Context, commissionID string, refinementType string) {
	attrs := []attribute.KeyValue{
		attribute.String("commission.id", commissionID),
		attribute.String("refinement.type", refinementType),
	}
	cc.refinements.Add(ctx, 1, metric.WithAttributes(attrs...))
}

// RecordComplexity records commission complexity
func (cc *CommissionCollector) RecordComplexity(ctx context.Context, commissionID string, score float64) {
	attrs := []attribute.KeyValue{
		attribute.String("commission.id", commissionID),
	}
	cc.complexityScore.Record(ctx, score, metric.WithAttributes(attrs...))
}

// RecordCommissionStart records the start of a commission
func (cc *CommissionCollector) RecordCommissionStart(ctx context.Context, commissionID string) {
	attrs := []attribute.KeyValue{
		attribute.String("commission.id", commissionID),
	}
	cc.activeCommissions.Add(ctx, 1, metric.WithAttributes(attrs...))
}

// RecordCommissionEnd records the end of a commission
func (cc *CommissionCollector) RecordCommissionEnd(ctx context.Context, commissionID string) {
	attrs := []attribute.KeyValue{
		attribute.String("commission.id", commissionID),
	}
	cc.activeCommissions.Add(ctx, -1, metric.WithAttributes(attrs...))
}

// RecordResourceUtilization records resource utilization
func (cc *CommissionCollector) RecordResourceUtilization(ctx context.Context, commissionID string, resourceType string, utilization float64) {
	attrs := []attribute.KeyValue{
		attribute.String("commission.id", commissionID),
		attribute.String("resource.type", resourceType),
	}
	cc.resourceUtilization.Record(ctx, utilization, metric.WithAttributes(attrs...))
}

// ObserveQueueDepth sets up queue depth observation
func (cc *CommissionCollector) ObserveQueueDepth(queueFunc func() int64) error {
	_, err := cc.meter.RegisterCallback(
		func(_ context.Context, o metric.Observer) error {
			o.ObserveInt64(cc.queueDepth, queueFunc())
			return nil
		},
		cc.queueDepth,
	)
	return err
}
