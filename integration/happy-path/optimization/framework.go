// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package optimization

import (
	"context"
	"math"
	"math/rand"
	"sync"
	"testing"
	"time"

	"github.com/guild-framework/guild-core/pkg/gerror"
)

// OptimizationTestFramework provides continuous performance optimization testing
type OptimizationTestFramework struct {
	t               *testing.T
	cleanup         []func()
	baselineStorage *BaselineStorage
	mu              sync.RWMutex
}

// OptimizationTarget defines an optimization target
type OptimizationTarget struct {
	Metric       string
	CurrentValue interface{}
	TargetValue  interface{}
	Strategy     OptimizationStrategy
}

// OptimizationStrategy represents different optimization strategies
type OptimizationStrategy int

const (
	OptimizationStrategyMemoryPooling OptimizationStrategy = iota
	OptimizationStrategyObjectReuse
	OptimizationStrategyCaching
	OptimizationStrategyIndexOptimization
	OptimizationStrategyAlgorithmRefinement
	OptimizationStrategyParallelization
	OptimizationStrategyResourcePooling
)

// String returns the string representation of OptimizationStrategy
func (s OptimizationStrategy) String() string {
	switch s {
	case OptimizationStrategyMemoryPooling:
		return "MemoryPooling"
	case OptimizationStrategyObjectReuse:
		return "ObjectReuse"
	case OptimizationStrategyCaching:
		return "Caching"
	case OptimizationStrategyIndexOptimization:
		return "IndexOptimization"
	case OptimizationStrategyAlgorithmRefinement:
		return "AlgorithmRefinement"
	case OptimizationStrategyParallelization:
		return "Parallelization"
	case OptimizationStrategyResourcePooling:
		return "ResourcePooling"
	default:
		return "Unknown"
	}
}

// BaselineConfig configures baseline collection
type BaselineConfig struct {
	MetricsToCollect []string
	CollectionPeriod time.Duration
	SamplingInterval time.Duration
}

// BaselineCollector collects performance baselines
type BaselineCollector struct {
	config       BaselineConfig
	measurements map[string][]float64
	startTime    time.Time
	isCollecting bool
	mu           sync.RWMutex
}

// LoadLevel represents different load levels
type LoadLevel int

const (
	LoadLevelLight LoadLevel = iota
	LoadLevelMedium
	LoadLevelHeavy
	LoadLevelExtreme
)

// LoadConfig configures load generation
type LoadConfig struct {
	UserLoad            LoadLevel
	OperationMix        OperationMix
	Duration            time.Duration
	IdenticalToBaseline bool
}

// OperationMix defines the distribution of operations
type OperationMix struct {
	ReadOperations    float64
	WriteOperations   float64
	ComputeOperations float64
	NetworkOperations float64
}

// LoadGenerator generates realistic load patterns
type LoadGenerator struct {
	config    LoadConfig
	isRunning bool
	mu        sync.RWMutex
}

// OptimizerConfig configures optimization strategies
type OptimizerConfig struct {
	Target            OptimizationTarget
	BaselineValue     interface{}
	GradualRollout    bool
	SafetyChecks      bool
	RollbackOnFailure bool
}

// Optimizer applies specific optimization strategies
type Optimizer struct {
	config   OptimizerConfig
	strategy OptimizationStrategy
	state    OptimizerState
	mu       sync.RWMutex
}

// OptimizerState represents the current state of an optimizer
type OptimizerState int

const (
	OptimizerStateIdle OptimizerState = iota
	OptimizerStateApplying
	OptimizerStateValidating
	OptimizerStateCompleted
	OptimizerStateRolledBack
)

// OptimizationResult contains the results of an optimization
type OptimizationResult struct {
	Strategy         OptimizationStrategy
	Success          bool
	ImprovementRatio float64
	BaselineValue    interface{}
	OptimizedValue   interface{}
	AppliedAt        time.Time
	Duration         time.Duration
	ValidationPassed bool
	Metadata         map[string]interface{}
}

// PostOptimizationConfig configures post-optimization validation
type PostOptimizationConfig struct {
	BaselineMetrics     map[string]interface{}
	OptimizationResults map[string]*OptimizationResult
	ValidationPeriod    time.Duration
}

// PostOptimizationCollector collects metrics after optimization
type PostOptimizationCollector struct {
	config       PostOptimizationConfig
	measurements map[string][]float64
	startTime    time.Time
	isCollecting bool
	mu           sync.RWMutex
}

// StabilityConfig configures stability monitoring
type StabilityConfig struct {
	OptimizedMetrics   map[string]interface{}
	VarianceThreshold  float64
	MonitoringInterval time.Duration
}

// StabilityMonitor monitors optimization stability
type StabilityMonitor struct {
	config       StabilityConfig
	measurements map[string][]float64
	isMonitoring bool
	mu           sync.RWMutex
}

// StabilityResults contains stability monitoring results
type StabilityResults map[string]*StabilityMetric

// StabilityMetric represents stability metrics for a single measurement
type StabilityMetric struct {
	MetricName  string
	Variance    float64
	Consistency float64
	TrendSlope  float64
	Outliers    int
	SampleCount int
}

// BaselineStorage manages performance baselines
type BaselineStorage struct {
	baselines map[string]*PerformanceBaseline
	mu        sync.RWMutex
}

// PerformanceBaseline represents a performance baseline
type PerformanceBaseline struct {
	Metric     string
	Value      float64
	Timestamp  time.Time
	Samples    []float64
	Statistics BaselineStatistics
}

// BaselineStatistics contains statistical information about a baseline
type BaselineStatistics struct {
	Mean   float64
	Median float64
	StdDev float64
	Min    float64
	Max    float64
	P95    float64
	P99    float64
}

// NewOptimizationTestFramework creates a new optimization testing framework
func NewOptimizationTestFramework(t *testing.T) *OptimizationTestFramework {
	framework := &OptimizationTestFramework{
		t:               t,
		cleanup:         []func(){},
		baselineStorage: NewBaselineStorage(),
	}

	t.Cleanup(func() {
		framework.Cleanup()
	})

	return framework
}

// StartBaselineCollection begins baseline performance collection
func (f *OptimizationTestFramework) StartBaselineCollection(config BaselineConfig) (*BaselineCollector, error) {
	collector := &BaselineCollector{
		config:       config,
		measurements: make(map[string][]float64),
		startTime:    time.Now(),
		isCollecting: false,
	}

	// Initialize measurement arrays
	for _, metric := range config.MetricsToCollect {
		collector.measurements[metric] = make([]float64, 0)
	}

	return collector, nil
}

// CreateLoadGenerator creates a load generator
func (f *OptimizationTestFramework) CreateLoadGenerator(config LoadConfig) *LoadGenerator {
	return &LoadGenerator{
		config:    config,
		isRunning: false,
	}
}

// ExecuteLoad executes load and collects metrics
func (l *LoadGenerator) ExecuteLoad(collector *BaselineCollector) map[string]interface{} {
	l.mu.Lock()
	l.isRunning = true
	l.mu.Unlock()

	defer func() {
		l.mu.Lock()
		l.isRunning = false
		l.mu.Unlock()
	}()

	// Start baseline collection
	collector.Start()
	defer collector.Stop()

	// Simulate load execution
	endTime := time.Now().Add(l.config.Duration)

	for time.Now().Before(endTime) {
		l.simulateOperations()
		time.Sleep(1 * time.Second)
	}

	// Collect final metrics
	return collector.GetCollectedMetrics()
}

// GetTypicalOperationMix returns a typical operation mix
func (f *OptimizationTestFramework) GetTypicalOperationMix() OperationMix {
	return OperationMix{
		ReadOperations:    0.4,
		WriteOperations:   0.3,
		ComputeOperations: 0.2,
		NetworkOperations: 0.1,
	}
}

// CreateOptimizer creates an optimizer for a specific strategy
func (f *OptimizationTestFramework) CreateOptimizer(strategy OptimizationStrategy, config OptimizerConfig) (*Optimizer, error) {
	return &Optimizer{
		config:   config,
		strategy: strategy,
		state:    OptimizerStateIdle,
	}, nil
}

// Apply applies the optimization strategy
func (o *Optimizer) Apply(ctx context.Context) (*OptimizationResult, error) {
	o.mu.Lock()
	if o.state != OptimizerStateIdle {
		o.mu.Unlock()
		return nil, gerror.New(gerror.ErrCodeConflict, "optimizer not in idle state", nil).
			WithComponent("optimization").
			WithOperation("Apply")
	}
	o.state = OptimizerStateApplying
	o.mu.Unlock()

	defer func() {
		o.mu.Lock()
		o.state = OptimizerStateCompleted
		o.mu.Unlock()
	}()

	start := time.Now()

	// Simulate optimization application based on strategy
	err := o.applyStrategy(ctx)
	if err != nil {
		return nil, err
	}

	// Simulate validation
	o.mu.Lock()
	o.state = OptimizerStateValidating
	o.mu.Unlock()

	validationPassed := o.validateOptimization(ctx)

	// Calculate improvement
	baselineValue := convertToFloat64(o.config.BaselineValue)
	optimizedValue := o.simulateOptimizedValue(baselineValue)
	improvementRatio := o.calculateImprovementRatio(baselineValue, optimizedValue)

	result := &OptimizationResult{
		Strategy:         o.strategy,
		Success:          validationPassed,
		ImprovementRatio: improvementRatio,
		BaselineValue:    baselineValue,
		OptimizedValue:   optimizedValue,
		Duration:         time.Since(start),
		ValidationPassed: validationPassed,
		Metadata: map[string]interface{}{
			"strategy":        o.strategy.String(),
			"gradual_rollout": o.config.GradualRollout,
			"safety_checks":   o.config.SafetyChecks,
		},
	}

	return result, nil
}

// StartPostOptimizationCollection starts post-optimization metric collection
func (f *OptimizationTestFramework) StartPostOptimizationCollection(config PostOptimizationConfig) (*PostOptimizationCollector, error) {
	collector := &PostOptimizationCollector{
		config:       config,
		measurements: make(map[string][]float64),
		startTime:    time.Now(),
		isCollecting: false,
	}

	// Initialize measurement arrays
	for metric := range config.BaselineMetrics {
		collector.measurements[metric] = make([]float64, 0)
	}

	return collector, nil
}

// CalculateImprovement calculates improvement percentage
func (f *OptimizationTestFramework) CalculateImprovement(baseline, optimized interface{}, metricName string) float64 {
	baselineValue := f.convertToFloat64(baseline)
	optimizedValue := f.convertToFloat64(optimized)

	// For metrics where lower is better (like latency, memory usage)
	if f.isLowerBetterMetric(metricName) {
		return (baselineValue - optimizedValue) / baselineValue
	}

	// For metrics where higher is better (like throughput, success rate)
	return (optimizedValue - baselineValue) / baselineValue
}

// CalculateRegression calculates regression percentage
func (f *OptimizationTestFramework) CalculateRegression(baseline, current interface{}, metricName string) float64 {
	baselineValue := f.convertToFloat64(baseline)
	currentValue := f.convertToFloat64(current)

	// For metrics where lower is better, regression is an increase
	if f.isLowerBetterMetric(metricName) {
		if currentValue > baselineValue {
			return (currentValue - baselineValue) / baselineValue
		}
		return 0.0
	}

	// For metrics where higher is better, regression is a decrease
	if currentValue < baselineValue {
		return (baselineValue - currentValue) / baselineValue
	}
	return 0.0
}

// CreateStabilityMonitor creates a stability monitor
func (f *OptimizationTestFramework) CreateStabilityMonitor(config StabilityConfig) *StabilityMonitor {
	return &StabilityMonitor{
		config:       config,
		measurements: make(map[string][]float64),
		isMonitoring: false,
	}
}

// Monitor monitors stability for the specified duration
func (s *StabilityMonitor) Monitor(ctx context.Context) StabilityResults {
	s.mu.Lock()
	s.isMonitoring = true

	// Initialize measurement arrays
	for metric := range s.config.OptimizedMetrics {
		s.measurements[metric] = make([]float64, 0)
	}
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		s.isMonitoring = false
		s.mu.Unlock()
	}()

	// Monitor stability
	ticker := time.NewTicker(s.config.MonitoringInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return s.calculateStabilityResults()
		case <-ticker.C:
			s.collectStabilityMeasurements()
		}
	}
}

// UpdatePerformanceBaselines updates performance baselines
func (f *OptimizationTestFramework) UpdatePerformanceBaselines(metrics map[string]interface{}) error {
	for metricName, value := range metrics {
		baseline := &PerformanceBaseline{
			Metric:    metricName,
			Value:     f.convertToFloat64(value),
			Timestamp: time.Now(),
			Samples:   []float64{f.convertToFloat64(value)},
		}

		baseline.Statistics = f.calculateBaselineStatistics(baseline.Samples)

		f.baselineStorage.Store(metricName, baseline)
	}

	return nil
}

// Cleanup performs framework cleanup
func (f *OptimizationTestFramework) Cleanup() {
	f.mu.Lock()
	defer f.mu.Unlock()

	for i := len(f.cleanup) - 1; i >= 0; i-- {
		f.cleanup[i]()
	}
}

// Helper methods

func (c *BaselineCollector) Start() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.isCollecting = true
	c.startTime = time.Now()

	// Start collection goroutine
	go c.collectMetrics()
}

func (c *BaselineCollector) Stop() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.isCollecting = false
}

func (c *BaselineCollector) collectMetrics() {
	ticker := time.NewTicker(c.config.SamplingInterval)
	defer ticker.Stop()

	for {
		c.mu.RLock()
		if !c.isCollecting {
			c.mu.RUnlock()
			return
		}
		c.mu.RUnlock()

		// Simulate metric collection
		for _, metric := range c.config.MetricsToCollect {
			value := c.simulateMetricValue(metric)

			c.mu.Lock()
			c.measurements[metric] = append(c.measurements[metric], value)
			c.mu.Unlock()
		}

		<-ticker.C
	}
}

func (c *BaselineCollector) simulateMetricValue(metric string) float64 {
	// Simulate realistic metric values with some variance
	switch metric {
	case "memory_usage":
		return 450000000 + rand.Float64()*100000000 // 450-550MB
	case "gc_frequency":
		return 8 + rand.Float64()*4 // 8-12 GCs per minute
	case "agent_selection_time":
		return 1.5 + rand.Float64()*0.6 // 1.5-2.1 seconds
	case "search_response_time":
		return 350 + rand.Float64()*100 // 350-450ms
	case "cpu_usage":
		return 20 + rand.Float64()*15 // 20-35%
	case "throughput":
		return 100 + rand.Float64()*20 // 100-120 ops/sec
	default:
		return rand.Float64() * 100
	}
}

func (c *BaselineCollector) GetCollectedMetrics() map[string]interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()

	metrics := make(map[string]interface{})

	for metric, measurements := range c.measurements {
		if len(measurements) > 0 {
			// Calculate average
			sum := 0.0
			for _, value := range measurements {
				sum += value
			}
			metrics[metric] = sum / float64(len(measurements))
		}
	}

	return metrics
}

func (l *LoadGenerator) simulateOperations() {
	// Simulate various operations based on operation mix
	mix := l.config.OperationMix

	if rand.Float64() < mix.ReadOperations {
		l.simulateReadOperation()
	}
	if rand.Float64() < mix.WriteOperations {
		l.simulateWriteOperation()
	}
	if rand.Float64() < mix.ComputeOperations {
		l.simulateComputeOperation()
	}
	if rand.Float64() < mix.NetworkOperations {
		l.simulateNetworkOperation()
	}
}

func (l *LoadGenerator) simulateReadOperation() {
	// Simulate read operation latency
	time.Sleep(time.Duration(rand.Intn(50)) * time.Millisecond)
}

func (l *LoadGenerator) simulateWriteOperation() {
	// Simulate write operation latency
	time.Sleep(time.Duration(rand.Intn(100)) * time.Millisecond)
}

func (l *LoadGenerator) simulateComputeOperation() {
	// Simulate compute operation
	sum := 0
	for i := 0; i < 10000; i++ {
		sum += i * i
	}
	_ = sum
}

func (l *LoadGenerator) simulateNetworkOperation() {
	// Simulate network operation latency
	time.Sleep(time.Duration(rand.Intn(200)) * time.Millisecond)
}

func (o *Optimizer) applyStrategy(ctx context.Context) error {
	// Simulate strategy application time based on strategy type
	var applicationTime time.Duration

	switch o.strategy {
	case OptimizationStrategyMemoryPooling:
		applicationTime = 30 * time.Second
	case OptimizationStrategyObjectReuse:
		applicationTime = 20 * time.Second
	case OptimizationStrategyCaching:
		applicationTime = 45 * time.Second
	case OptimizationStrategyIndexOptimization:
		applicationTime = 60 * time.Second
	default:
		applicationTime = 30 * time.Second
	}

	// Simulate gradual rollout if enabled
	if o.config.GradualRollout {
		phases := 3
		phaseTime := applicationTime / time.Duration(phases)

		for i := 0; i < phases; i++ {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(phaseTime):
				// Phase completed
			}
		}
	} else {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(applicationTime):
			// Application completed
		}
	}

	return nil
}

func (o *Optimizer) validateOptimization(ctx context.Context) bool {
	if !o.config.SafetyChecks {
		return true
	}

	// Simulate validation time
	validationTime := 10 * time.Second
	select {
	case <-ctx.Done():
		return false
	case <-time.After(validationTime):
		// Validation completed
	}

	// Simulate validation success rate (95% success)
	return rand.Float64() > 0.05
}

func (o *Optimizer) simulateOptimizedValue(baselineValue float64) float64 {
	// Simulate optimization effectiveness based on strategy
	var improvementFactor float64

	switch o.strategy {
	case OptimizationStrategyMemoryPooling:
		improvementFactor = 0.15 + rand.Float64()*0.1 // 15-25% improvement
	case OptimizationStrategyObjectReuse:
		improvementFactor = 0.25 + rand.Float64()*0.15 // 25-40% improvement
	case OptimizationStrategyCaching:
		improvementFactor = 0.20 + rand.Float64()*0.15 // 20-35% improvement
	case OptimizationStrategyIndexOptimization:
		improvementFactor = 0.30 + rand.Float64()*0.15 // 30-45% improvement
	default:
		improvementFactor = 0.10 + rand.Float64()*0.10 // 10-20% improvement
	}

	// For "lower is better" metrics, we reduce the value
	targetMetric := o.config.Target.Metric
	if isLowerBetterMetric(targetMetric) {
		return baselineValue * (1.0 - improvementFactor)
	}

	// For "higher is better" metrics, we increase the value
	return baselineValue * (1.0 + improvementFactor)
}

func (o *Optimizer) calculateImprovementRatio(baseline, optimized float64) float64 {
	if baseline == 0 {
		return 0
	}

	targetMetric := o.config.Target.Metric
	if isLowerBetterMetric(targetMetric) {
		return (baseline - optimized) / baseline
	}

	return (optimized - baseline) / baseline
}

func (o *Optimizer) isLowerBetterMetric(metric string) bool {
	lowerBetterMetrics := []string{
		"memory_usage",
		"gc_frequency",
		"agent_selection_time",
		"search_response_time",
		"cpu_usage",
		"latency",
		"response_time",
		"load_time",
	}

	for _, m := range lowerBetterMetrics {
		if metric == m {
			return true
		}
	}
	return false
}

func (c *PostOptimizationCollector) Start() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.isCollecting = true
	c.startTime = time.Now()

	// Start collection goroutine
	go c.collectMetrics()
}

func (c *PostOptimizationCollector) Stop() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.isCollecting = false
}

func (c *PostOptimizationCollector) collectMetrics() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		c.mu.RLock()
		if !c.isCollecting {
			c.mu.RUnlock()
			return
		}
		c.mu.RUnlock()

		// Simulate optimized metric collection
		for metric := range c.config.BaselineMetrics {
			value := c.simulateOptimizedMetricValue(metric)

			c.mu.Lock()
			c.measurements[metric] = append(c.measurements[metric], value)
			c.mu.Unlock()
		}

		<-ticker.C
	}
}

func (c *PostOptimizationCollector) simulateOptimizedMetricValue(metric string) float64 {
	// Get baseline value
	baselineValue := 0.0
	if baseline, exists := c.config.BaselineMetrics[metric]; exists {
		baselineValue = c.convertToFloat64(baseline)
	}

	// Apply optimization improvement
	if result, exists := c.config.OptimizationResults[metric]; exists {
		return c.convertToFloat64(result.OptimizedValue)
	}

	// No optimization applied, return baseline with small variance
	return baselineValue * (0.98 + rand.Float64()*0.04) // ±2% variance
}

func (c *PostOptimizationCollector) convertToFloat64(value interface{}) float64 {
	switch v := value.(type) {
	case float64:
		return v
	case float32:
		return float64(v)
	case int:
		return float64(v)
	case int64:
		return float64(v)
	case time.Duration:
		return float64(v.Nanoseconds()) / 1e6 // Convert to milliseconds
	default:
		return 0.0
	}
}

func (c *PostOptimizationCollector) GetCollectedMetrics() map[string]interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()

	metrics := make(map[string]interface{})

	for metric, measurements := range c.measurements {
		if len(measurements) > 0 {
			// Calculate average
			sum := 0.0
			for _, value := range measurements {
				sum += value
			}
			metrics[metric] = sum / float64(len(measurements))
		}
	}

	return metrics
}

func (s *StabilityMonitor) collectStabilityMeasurements() {
	s.mu.Lock()
	defer s.mu.Unlock()

	for metric, expectedValue := range s.config.OptimizedMetrics {
		// Simulate measurement with small variance
		baseValue := s.convertToFloat64(expectedValue)
		variance := baseValue * 0.02 // 2% variance
		measurement := baseValue + (rand.Float64()-0.5)*variance*2

		s.measurements[metric] = append(s.measurements[metric], measurement)
	}
}

func (s *StabilityMonitor) calculateStabilityResults() StabilityResults {
	s.mu.RLock()
	defer s.mu.RUnlock()

	results := make(StabilityResults)

	for metric, measurements := range s.measurements {
		if len(measurements) < 2 {
			continue
		}

		stabilityMetric := &StabilityMetric{
			MetricName:  metric,
			SampleCount: len(measurements),
		}

		// Calculate variance
		mean := s.calculateMean(measurements)
		variance := s.calculateVariance(measurements, mean)
		stabilityMetric.Variance = math.Sqrt(variance) / mean // Coefficient of variation

		// Calculate consistency (1 - coefficient of variation)
		stabilityMetric.Consistency = math.Max(0, 1.0-stabilityMetric.Variance)

		// Calculate trend slope (simple linear regression)
		stabilityMetric.TrendSlope = s.calculateTrendSlope(measurements)

		// Count outliers (values more than 2 standard deviations from mean)
		stdDev := math.Sqrt(variance)
		for _, value := range measurements {
			if math.Abs(value-mean) > 2*stdDev {
				stabilityMetric.Outliers++
			}
		}

		results[metric] = stabilityMetric
	}

	return results
}

func (s *StabilityMonitor) convertToFloat64(value interface{}) float64 {
	switch v := value.(type) {
	case float64:
		return v
	case float32:
		return float64(v)
	case int:
		return float64(v)
	case int64:
		return float64(v)
	case time.Duration:
		return float64(v.Nanoseconds()) / 1e6 // Convert to milliseconds
	default:
		return 0.0
	}
}

func (s *StabilityMonitor) calculateMean(values []float64) float64 {
	sum := 0.0
	for _, value := range values {
		sum += value
	}
	return sum / float64(len(values))
}

func (s *StabilityMonitor) calculateVariance(values []float64, mean float64) float64 {
	sumSquaredDiffs := 0.0
	for _, value := range values {
		diff := value - mean
		sumSquaredDiffs += diff * diff
	}
	return sumSquaredDiffs / float64(len(values))
}

func (s *StabilityMonitor) calculateTrendSlope(values []float64) float64 {
	if len(values) < 2 {
		return 0.0
	}

	// Simple linear regression slope calculation
	n := float64(len(values))
	sumX := 0.0
	sumY := 0.0
	sumXY := 0.0
	sumXX := 0.0

	for i, value := range values {
		x := float64(i)
		y := value
		sumX += x
		sumY += y
		sumXY += x * y
		sumXX += x * x
	}

	// Slope = (n*sumXY - sumX*sumY) / (n*sumXX - sumX*sumX)
	numerator := n*sumXY - sumX*sumY
	denominator := n*sumXX - sumX*sumX

	if denominator == 0 {
		return 0.0
	}

	return numerator / denominator
}

func (f *OptimizationTestFramework) convertToFloat64(value interface{}) float64 {
	switch v := value.(type) {
	case float64:
		return v
	case float32:
		return float64(v)
	case int:
		return float64(v)
	case int64:
		return float64(v)
	case time.Duration:
		return float64(v.Nanoseconds()) / 1e6 // Convert to milliseconds
	default:
		return 0.0
	}
}

func (f *OptimizationTestFramework) isLowerBetterMetric(metric string) bool {
	lowerBetterMetrics := []string{
		"memory_usage",
		"gc_frequency",
		"agent_selection_time",
		"search_response_time",
		"cpu_usage",
		"latency",
		"response_time",
		"load_time",
	}

	for _, m := range lowerBetterMetrics {
		if metric == m {
			return true
		}
	}
	return false
}

func (f *OptimizationTestFramework) calculateBaselineStatistics(samples []float64) BaselineStatistics {
	if len(samples) == 0 {
		return BaselineStatistics{}
	}

	// Calculate basic statistics
	sum := 0.0
	min := samples[0]
	max := samples[0]

	for _, value := range samples {
		sum += value
		if value < min {
			min = value
		}
		if value > max {
			max = value
		}
	}

	mean := sum / float64(len(samples))

	// Calculate standard deviation
	sumSquaredDiffs := 0.0
	for _, value := range samples {
		diff := value - mean
		sumSquaredDiffs += diff * diff
	}
	stdDev := math.Sqrt(sumSquaredDiffs / float64(len(samples)))

	// Calculate percentiles (simplified)
	sorted := make([]float64, len(samples))
	copy(sorted, samples)
	// Note: Would need proper sorting in real implementation

	median := mean             // Simplified
	p95 := mean + 1.645*stdDev // Approximate P95
	p99 := mean + 2.326*stdDev // Approximate P99

	return BaselineStatistics{
		Mean:   mean,
		Median: median,
		StdDev: stdDev,
		Min:    min,
		Max:    max,
		P95:    p95,
		P99:    p99,
	}
}

func NewBaselineStorage() *BaselineStorage {
	return &BaselineStorage{
		baselines: make(map[string]*PerformanceBaseline),
	}
}

func (b *BaselineStorage) Store(metric string, baseline *PerformanceBaseline) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.baselines[metric] = baseline
}

func (b *BaselineStorage) Get(metric string) (*PerformanceBaseline, bool) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	baseline, exists := b.baselines[metric]
	return baseline, exists
}

// convertToFloat64 converts various types to float64
func convertToFloat64(value interface{}) float64 {
	switch v := value.(type) {
	case float64:
		return v
	case float32:
		return float64(v)
	case int:
		return float64(v)
	case int64:
		return float64(v)
	case time.Duration:
		return float64(v.Nanoseconds()) / 1e6 // Convert to milliseconds
	default:
		return 0.0
	}
}

// isLowerBetterMetric returns true if lower values are better for the given metric
func isLowerBetterMetric(metric string) bool {
	lowerBetterMetrics := []string{
		"memory_usage",
		"gc_frequency",
		"agent_selection_time",
		"search_response_time",
		"cpu_usage",
		"latency",
		"response_time",
		"load_time",
	}

	for _, m := range lowerBetterMetrics {
		if metric == m {
			return true
		}
	}
	return false
}
