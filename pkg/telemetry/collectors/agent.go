package collectors

import (
	"context"
	"time"

	"github.com/lancekrogers/guild-core/pkg/gerror"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// AgentCollector collects agent-specific metrics
type AgentCollector struct {
	meter                metric.Meter
	executionTime        metric.Float64Histogram
	taskCompletions      metric.Int64Counter
	taskFailures         metric.Int64Counter
	concurrentExecutions metric.Int64UpDownCounter
	toolInvocations      metric.Int64Counter
	contextSwitches      metric.Int64Counter
	memoryRetrieval      metric.Float64Histogram
}

// NewAgentCollector creates a new agent metrics collector
func NewAgentCollector(meter metric.Meter) (*AgentCollector, error) {
	ac := &AgentCollector{meter: meter}

	if err := ac.initializeMetrics(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to initialize agent metrics")
	}

	return ac, nil
}

// initializeMetrics creates all agent metric instruments
func (ac *AgentCollector) initializeMetrics() error {
	var err error

	ac.executionTime, err = ac.meter.Float64Histogram(
		"guild.agent.execution.duration",
		metric.WithDescription("Time taken for agent to execute tasks"),
		metric.WithUnit("s"),
	)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create execution time histogram")
	}

	ac.taskCompletions, err = ac.meter.Int64Counter(
		"guild.agent.tasks.completed",
		metric.WithDescription("Number of tasks completed by agents"),
	)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create task completions counter")
	}

	ac.taskFailures, err = ac.meter.Int64Counter(
		"guild.agent.tasks.failed",
		metric.WithDescription("Number of tasks failed by agents"),
	)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create task failures counter")
	}

	ac.concurrentExecutions, err = ac.meter.Int64UpDownCounter(
		"guild.agent.executions.concurrent",
		metric.WithDescription("Number of concurrent agent executions"),
	)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create concurrent executions counter")
	}

	ac.toolInvocations, err = ac.meter.Int64Counter(
		"guild.agent.tools.invocations",
		metric.WithDescription("Number of tool invocations by agents"),
	)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create tool invocations counter")
	}

	ac.contextSwitches, err = ac.meter.Int64Counter(
		"guild.agent.context.switches",
		metric.WithDescription("Number of context switches during agent execution"),
	)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create context switches counter")
	}

	ac.memoryRetrieval, err = ac.meter.Float64Histogram(
		"guild.agent.memory.retrieval.duration",
		metric.WithDescription("Time taken to retrieve from agent memory"),
		metric.WithUnit("s"),
	)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create memory retrieval histogram")
	}

	return nil
}

// RecordExecution records metrics for an agent execution
func (ac *AgentCollector) RecordExecution(ctx context.Context, agentName string, taskType string, duration time.Duration, success bool) {
	attrs := []attribute.KeyValue{
		attribute.String("agent.name", agentName),
		attribute.String("task.type", taskType),
		attribute.Bool("success", success),
	}

	ac.executionTime.Record(ctx, duration.Seconds(), metric.WithAttributes(attrs...))

	if success {
		ac.taskCompletions.Add(ctx, 1, metric.WithAttributes(attrs...))
	} else {
		ac.taskFailures.Add(ctx, 1, metric.WithAttributes(attrs...))
	}
}

// RecordConcurrentStart records the start of a concurrent execution
func (ac *AgentCollector) RecordConcurrentStart(ctx context.Context, agentName string) {
	attrs := []attribute.KeyValue{
		attribute.String("agent.name", agentName),
	}
	ac.concurrentExecutions.Add(ctx, 1, metric.WithAttributes(attrs...))
}

// RecordConcurrentEnd records the end of a concurrent execution
func (ac *AgentCollector) RecordConcurrentEnd(ctx context.Context, agentName string) {
	attrs := []attribute.KeyValue{
		attribute.String("agent.name", agentName),
	}
	ac.concurrentExecutions.Add(ctx, -1, metric.WithAttributes(attrs...))
}

// RecordToolInvocation records a tool invocation
func (ac *AgentCollector) RecordToolInvocation(ctx context.Context, agentName string, toolName string) {
	attrs := []attribute.KeyValue{
		attribute.String("agent.name", agentName),
		attribute.String("tool.name", toolName),
	}
	ac.toolInvocations.Add(ctx, 1, metric.WithAttributes(attrs...))
}

// RecordContextSwitch records a context switch
func (ac *AgentCollector) RecordContextSwitch(ctx context.Context, agentName string, fromContext, toContext string) {
	attrs := []attribute.KeyValue{
		attribute.String("agent.name", agentName),
		attribute.String("context.from", fromContext),
		attribute.String("context.to", toContext),
	}
	ac.contextSwitches.Add(ctx, 1, metric.WithAttributes(attrs...))
}

// RecordMemoryRetrieval records memory retrieval metrics
func (ac *AgentCollector) RecordMemoryRetrieval(ctx context.Context, agentName string, retrievalType string, duration time.Duration, itemCount int) {
	attrs := []attribute.KeyValue{
		attribute.String("agent.name", agentName),
		attribute.String("retrieval.type", retrievalType),
		attribute.Int("item.count", itemCount),
	}
	ac.memoryRetrieval.Record(ctx, duration.Seconds(), metric.WithAttributes(attrs...))
}
