// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

// Package prompt provides prompt chain management for MCP
package prompt

import (
	"context"
	"fmt"
	"time"

	"github.com/guild-framework/guild-core/pkg/gerror"
	"github.com/guild-framework/guild-core/pkg/mcp/protocol"
)

// Processor defines the interface for processing prompts
type Processor interface {
	// Process handles a prompt input
	Process(ctx context.Context, input *Input) (*Output, error)

	// SetNext sets the next processor in the chain
	SetNext(next Processor) Processor

	// GetName returns the processor name
	GetName() string
}

// Input represents input to a prompt processing step
type Input struct {
	Text       string
	Parameters map[string]interface{}
	Metadata   map[string]string
	History    []*Exchange
}

// Output represents the result from a prompt processing step
type Output struct {
	Text     string
	Metadata map[string]string
	Cost     protocol.CostReport
}

// Exchange represents a complete prompt-response pair
type Exchange struct {
	Input     *Input
	Output    *Output
	Processor string
	StartTime time.Time
	EndTime   time.Time
	Metrics   protocol.CostReport
}

// Chain represents a chain of processors
type Chain struct {
	processors []Processor
	analyzer   Analyzer
}

// NewChain creates a new prompt processing chain
func NewChain(processors ...Processor) *Chain {
	// Link processors
	for i := 0; i < len(processors)-1; i++ {
		processors[i].SetNext(processors[i+1])
	}

	return &Chain{
		processors: processors,
		analyzer:   NewAnalyzer(),
	}
}

// Process executes the chain
func (c *Chain) Process(ctx context.Context, input *Input) (*Output, error) {
	if len(c.processors) == 0 {
		return nil, gerror.New(gerror.ErrCodeValidation, "no processors in chain", nil).
			WithComponent("mcp_prompt").
			WithOperation("Process")
	}

	// Start chain analysis
	chainID := c.analyzer.StartChain(ctx, input)
	defer c.analyzer.EndChain(ctx, chainID)

	// Process through first processor (which calls next)
	output, err := c.processors[0].Process(ctx, input)
	if err != nil {
		c.analyzer.RecordError(ctx, chainID, err)
		return nil, err
	}

	return output, nil
}

// WithAnalyzer sets a custom analyzer
func (c *Chain) WithAnalyzer(analyzer Analyzer) *Chain {
	c.analyzer = analyzer
	return c
}

// BaseProcessor provides common functionality for processors
type BaseProcessor struct {
	name string
	next Processor
}

// NewBaseProcessor creates a new base processor
func NewBaseProcessor(name string) *BaseProcessor {
	return &BaseProcessor{
		name: name,
	}
}

// Process is the default implementation that passes to next
func (b *BaseProcessor) Process(ctx context.Context, input *Input) (*Output, error) {
	if b.next == nil {
		return nil, gerror.New(gerror.ErrCodeInternal, "end of chain reached", nil).
			WithComponent("mcp_prompt").
			WithOperation("Process").
			WithDetails("processor_name", b.name)
	}
	return b.next.Process(ctx, input)
}

// SetNext sets the next processor
func (b *BaseProcessor) SetNext(next Processor) Processor {
	b.next = next
	return next
}

// GetName returns the processor name
func (b *BaseProcessor) GetName() string {
	return b.name
}

// Common Processors

// ValidationProcessor validates input
type ValidationProcessor struct {
	*BaseProcessor
	validator func(*Input) error
}

// NewValidationProcessor creates a validation processor
func NewValidationProcessor(validator func(*Input) error) *ValidationProcessor {
	return &ValidationProcessor{
		BaseProcessor: NewBaseProcessor("validation"),
		validator:     validator,
	}
}

// Process validates the input
func (p *ValidationProcessor) Process(ctx context.Context, input *Input) (*Output, error) {
	if p.validator != nil {
		if err := p.validator(input); err != nil {
			return nil, gerror.Wrap(err, gerror.ErrCodeValidation, "validation failed").
				WithComponent("mcp_prompt").
				WithOperation("ValidationProcessor.Process")
		}
	}

	return p.BaseProcessor.Process(ctx, input)
}

// EnhancementProcessor enhances prompts with context
type EnhancementProcessor struct {
	*BaseProcessor
	enhancer func(context.Context, *Input) (*Input, error)
}

// NewEnhancementProcessor creates an enhancement processor
func NewEnhancementProcessor(enhancer func(context.Context, *Input) (*Input, error)) *EnhancementProcessor {
	return &EnhancementProcessor{
		BaseProcessor: NewBaseProcessor("enhancement"),
		enhancer:      enhancer,
	}
}

// Process enhances the input
func (p *EnhancementProcessor) Process(ctx context.Context, input *Input) (*Output, error) {
	enhanced := input
	if p.enhancer != nil {
		var err error
		enhanced, err = p.enhancer(ctx, input)
		if err != nil {
			return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "enhancement failed").
				WithComponent("mcp_prompt").
				WithOperation("EnhancementProcessor.Process")
		}
	}

	return p.BaseProcessor.Process(ctx, enhanced)
}

// CachingProcessor implements caching for prompts
type CachingProcessor struct {
	*BaseProcessor
	cache    map[string]*Output
	keyFunc  func(*Input) string
	ttl      time.Duration
	maxItems int
}

// NewCachingProcessor creates a caching processor
func NewCachingProcessor(keyFunc func(*Input) string, ttl time.Duration) *CachingProcessor {
	return &CachingProcessor{
		BaseProcessor: NewBaseProcessor("caching"),
		cache:         make(map[string]*Output),
		keyFunc:       keyFunc,
		ttl:           ttl,
		maxItems:      1000,
	}
}

// Process checks cache before processing
func (p *CachingProcessor) Process(ctx context.Context, input *Input) (*Output, error) {
	// Generate cache key
	key := p.keyFunc(input)

	// Check cache
	if cached, exists := p.cache[key]; exists {
		// Return cached result
		return &Output{
			Text:     cached.Text,
			Metadata: cached.Metadata,
			Cost:     protocol.CostReport{}, // No cost for cached
		}, nil
	}

	// Process normally
	output, err := p.BaseProcessor.Process(ctx, input)
	if err != nil {
		return nil, err
	}

	// Cache result
	p.cache[key] = output

	// TODO: Implement TTL and max items cleanup

	return output, nil
}

// MetricsProcessor collects metrics
type MetricsProcessor struct {
	*BaseProcessor
	observer CostObserver
}

// CostObserver interface for cost tracking
type CostObserver interface {
	RecordCost(ctx context.Context, operationID string, cost protocol.CostReport)
}

// NewMetricsProcessor creates a metrics processor
func NewMetricsProcessor(observer CostObserver) *MetricsProcessor {
	return &MetricsProcessor{
		BaseProcessor: NewBaseProcessor("metrics"),
		observer:      observer,
	}
}

// Process collects metrics
func (p *MetricsProcessor) Process(ctx context.Context, input *Input) (*Output, error) {
	start := time.Now()

	// Process
	output, err := p.BaseProcessor.Process(ctx, input)

	// Record metrics
	cost := protocol.CostReport{
		StartTime:   start,
		EndTime:     time.Now(),
		LatencyCost: time.Since(start),
		OperationID: fmt.Sprintf("prompt-%d", time.Now().UnixNano()),
	}

	if output != nil {
		cost = output.Cost
		cost.StartTime = start
		cost.EndTime = time.Now()
		cost.LatencyCost = time.Since(start)
	}

	p.observer.RecordCost(ctx, cost.OperationID, cost)

	return output, err
}

// RouterProcessor routes to different processors based on conditions
type RouterProcessor struct {
	*BaseProcessor
	routes []Route
}

// Route defines a routing rule
type Route struct {
	Condition func(*Input) bool
	Processor Processor
}

// NewRouterProcessor creates a router processor
func NewRouterProcessor(routes []Route) *RouterProcessor {
	return &RouterProcessor{
		BaseProcessor: NewBaseProcessor("router"),
		routes:        routes,
	}
}

// Process routes based on conditions
func (p *RouterProcessor) Process(ctx context.Context, input *Input) (*Output, error) {
	// Check routes
	for _, route := range p.routes {
		if route.Condition(input) {
			return route.Processor.Process(ctx, input)
		}
	}

	// No route matched, use default next
	return p.BaseProcessor.Process(ctx, input)
}

// TransformProcessor transforms input/output
type TransformProcessor struct {
	*BaseProcessor
	inputTransform  func(*Input) (*Input, error)
	outputTransform func(*Output) (*Output, error)
}

// NewTransformProcessor creates a transform processor
func NewTransformProcessor(
	inputTransform func(*Input) (*Input, error),
	outputTransform func(*Output) (*Output, error),
) *TransformProcessor {
	return &TransformProcessor{
		BaseProcessor:   NewBaseProcessor("transform"),
		inputTransform:  inputTransform,
		outputTransform: outputTransform,
	}
}

// Process applies transformations
func (p *TransformProcessor) Process(ctx context.Context, input *Input) (*Output, error) {
	// Transform input if needed
	transformed := input
	if p.inputTransform != nil {
		var err error
		transformed, err = p.inputTransform(input)
		if err != nil {
			return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "input transform failed").
				WithComponent("mcp_prompt").
				WithOperation("TransformProcessor.Process")
		}
	}

	// Process
	output, err := p.BaseProcessor.Process(ctx, transformed)
	if err != nil {
		return nil, err
	}

	// Transform output if needed
	if p.outputTransform != nil {
		output, err = p.outputTransform(output)
		if err != nil {
			return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "output transform failed").
				WithComponent("mcp_prompt").
				WithOperation("TransformProcessor.Process")
		}
	}

	return output, nil
}
