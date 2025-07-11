package performance

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/lancekrogers/guild/pkg/gerror"
)

// Profiler provides comprehensive performance profiling capabilities
type Profiler struct {
	enabled    atomic.Bool
	profiles   map[string]*ProfileData
	benchmarks map[string]*BenchmarkData
	mu         sync.RWMutex
	startTime  time.Time
	hooks      []ProfileHook
}

// ProfileData represents profiling data for a specific operation
type ProfileData struct {
	Name             string
	StartTime        time.Time
	EndTime          time.Time
	Duration         time.Duration
	MemoryBefore     runtime.MemStats
	MemoryAfter      runtime.MemStats
	GoroutinesBefore int
	GoroutinesAfter  int
	AllocsBefore     uint64
	AllocsAfter      uint64
	Samples          []ProfileSample
}

// ProfileSample represents a single profiling sample
type ProfileSample struct {
	Timestamp   time.Time
	CPUUsage    float64
	MemoryUsage uint64
	Goroutines  int
	Allocations uint64
	GCPauses    uint64
}

// BenchmarkData represents benchmark results
type BenchmarkData struct {
	Name             string
	Iterations       int
	TotalDuration    time.Duration
	AvgDuration      time.Duration
	MinDuration      time.Duration
	MaxDuration      time.Duration
	AllocationsPerOp uint64
	BytesPerOp       uint64
	Samples          []time.Duration
}

// ProfileHook allows custom profiling hooks
type ProfileHook interface {
	OnProfileStart(name string)
	OnProfileEnd(name string, data *ProfileData)
	OnBenchmarkStart(name string)
	OnBenchmarkEnd(name string, data *BenchmarkData)
}

// NewProfiler creates a new profiler
func NewProfiler() *Profiler {
	return &Profiler{
		profiles:   make(map[string]*ProfileData),
		benchmarks: make(map[string]*BenchmarkData),
		startTime:  time.Now(),
	}
}

// Enable enables profiling
func (p *Profiler) Enable() {
	p.enabled.Store(true)
}

// Disable disables profiling
func (p *Profiler) Disable() {
	p.enabled.Store(false)
}

// IsEnabled returns whether profiling is enabled
func (p *Profiler) IsEnabled() bool {
	return p.enabled.Load()
}

// AddHook adds a profiling hook
func (p *Profiler) AddHook(hook ProfileHook) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.hooks = append(p.hooks, hook)
}

// StartProfile starts profiling an operation
func (p *Profiler) StartProfile(name string) *ProfileContext {
	if !p.enabled.Load() {
		return &ProfileContext{enabled: false}
	}

	var memBefore runtime.MemStats
	runtime.ReadMemStats(&memBefore)
	goroutinesBefore := runtime.NumGoroutine()

	profile := &ProfileData{
		Name:             name,
		StartTime:        time.Now(),
		MemoryBefore:     memBefore,
		GoroutinesBefore: goroutinesBefore,
		AllocsBefore:     memBefore.Mallocs,
		Samples:          make([]ProfileSample, 0),
	}

	p.mu.Lock()
	p.profiles[name] = profile
	p.mu.Unlock()

	// Notify hooks
	for _, hook := range p.hooks {
		hook.OnProfileStart(name)
	}

	return &ProfileContext{
		profiler: p,
		name:     name,
		enabled:  true,
	}
}

// EndProfile ends profiling an operation
func (p *Profiler) EndProfile(name string) *ProfileData {
	if !p.enabled.Load() {
		return nil
	}

	p.mu.Lock()
	profile, exists := p.profiles[name]
	p.mu.Unlock()

	if !exists {
		return nil
	}

	var memAfter runtime.MemStats
	runtime.ReadMemStats(&memAfter)
	goroutinesAfter := runtime.NumGoroutine()

	profile.EndTime = time.Now()
	profile.Duration = profile.EndTime.Sub(profile.StartTime)
	profile.MemoryAfter = memAfter
	profile.GoroutinesAfter = goroutinesAfter
	profile.AllocsAfter = memAfter.Mallocs

	// Notify hooks
	for _, hook := range p.hooks {
		hook.OnProfileEnd(name, profile)
	}

	return profile
}

// ProfileContext provides context for a profiling session
type ProfileContext struct {
	profiler *Profiler
	name     string
	enabled  bool
}

// AddSample adds a profiling sample
func (pc *ProfileContext) AddSample() {
	if !pc.enabled {
		return
	}

	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)

	sample := ProfileSample{
		Timestamp:   time.Now(),
		MemoryUsage: mem.Alloc,
		Goroutines:  runtime.NumGoroutine(),
		Allocations: mem.Mallocs,
		GCPauses:    mem.PauseTotalNs,
	}

	pc.profiler.mu.Lock()
	if profile, exists := pc.profiler.profiles[pc.name]; exists {
		profile.Samples = append(profile.Samples, sample)
	}
	pc.profiler.mu.Unlock()
}

// End ends the profiling context
func (pc *ProfileContext) End() *ProfileData {
	if !pc.enabled {
		return nil
	}
	return pc.profiler.EndProfile(pc.name)
}

// Benchmark runs a benchmark for a given function
func (p *Profiler) Benchmark(name string, fn func()) *BenchmarkData {
	if !p.enabled.Load() {
		return nil
	}

	benchmark := &BenchmarkData{
		Name:        name,
		Samples:     make([]time.Duration, 0),
		MinDuration: time.Hour, // Initialize to large value
	}

	// Notify hooks
	for _, hook := range p.hooks {
		hook.OnBenchmarkStart(name)
	}

	// Warm up
	for i := 0; i < 3; i++ {
		fn()
	}

	// Initial measurement to determine iterations
	start := time.Now()
	fn()
	duration := time.Since(start)

	// Determine iterations based on target duration (1 second)
	targetDuration := time.Second
	iterations := int(targetDuration / duration)
	if iterations < 1 {
		iterations = 1
	}
	if iterations > 1000000 {
		iterations = 1000000
	}

	var memBefore runtime.MemStats
	runtime.ReadMemStats(&memBefore)

	// Run benchmark
	totalStart := time.Now()
	for i := 0; i < iterations; i++ {
		sampleStart := time.Now()
		fn()
		sampleDuration := time.Since(sampleStart)

		benchmark.Samples = append(benchmark.Samples, sampleDuration)
		benchmark.TotalDuration += sampleDuration

		if sampleDuration < benchmark.MinDuration {
			benchmark.MinDuration = sampleDuration
		}
		if sampleDuration > benchmark.MaxDuration {
			benchmark.MaxDuration = sampleDuration
		}
	}
	_ = time.Since(totalStart) // Track total duration

	var memAfter runtime.MemStats
	runtime.ReadMemStats(&memAfter)

	benchmark.Iterations = iterations
	benchmark.AvgDuration = benchmark.TotalDuration / time.Duration(iterations)
	benchmark.AllocationsPerOp = (memAfter.Mallocs - memBefore.Mallocs) / uint64(iterations)
	benchmark.BytesPerOp = (memAfter.TotalAlloc - memBefore.TotalAlloc) / uint64(iterations)

	p.mu.Lock()
	p.benchmarks[name] = benchmark
	p.mu.Unlock()

	// Notify hooks
	for _, hook := range p.hooks {
		hook.OnBenchmarkEnd(name, benchmark)
	}

	return benchmark
}

// GetProfile returns profiling data for an operation
func (p *Profiler) GetProfile(name string) *ProfileData {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.profiles[name]
}

// GetBenchmark returns benchmark data for an operation
func (p *Profiler) GetBenchmark(name string) *BenchmarkData {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.benchmarks[name]
}

// GetAllProfiles returns all profiling data
func (p *Profiler) GetAllProfiles() map[string]*ProfileData {
	p.mu.RLock()
	defer p.mu.RUnlock()

	profiles := make(map[string]*ProfileData)
	for name, profile := range p.profiles {
		profiles[name] = profile
	}
	return profiles
}

// GetAllBenchmarks returns all benchmark data
func (p *Profiler) GetAllBenchmarks() map[string]*BenchmarkData {
	p.mu.RLock()
	defer p.mu.RUnlock()

	benchmarks := make(map[string]*BenchmarkData)
	for name, benchmark := range p.benchmarks {
		benchmarks[name] = benchmark
	}
	return benchmarks
}

// Clear clears all profiling and benchmark data
func (p *Profiler) Clear() {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.profiles = make(map[string]*ProfileData)
	p.benchmarks = make(map[string]*BenchmarkData)
}

// MemorySample represents a memory profiling sample
type MemorySample struct {
	Timestamp    time.Time
	Alloc        uint64
	TotalAlloc   uint64
	Sys          uint64
	Lookups      uint64
	Mallocs      uint64
	Frees        uint64
	HeapAlloc    uint64
	HeapSys      uint64
	HeapIdle     uint64
	HeapInuse    uint64
	HeapReleased uint64
	HeapObjects  uint64
	StackInuse   uint64
	StackSys     uint64
	GCCount      uint32
	PauseTotalNs uint64
}

// RegressionDetector detects performance regressions
type RegressionDetector struct {
	baselines map[string]*BenchmarkData
	threshold float64
	mu        sync.RWMutex
}

// NewRegressionDetector creates a new regression detector
func NewRegressionDetector(threshold float64) *RegressionDetector {
	if threshold <= 0 {
		threshold = 0.1 // 10% default threshold
	}

	return &RegressionDetector{
		baselines: make(map[string]*BenchmarkData),
		threshold: threshold,
	}
}

// SetBaseline sets a performance baseline for a benchmark
func (rd *RegressionDetector) SetBaseline(name string, benchmark *BenchmarkData) {
	rd.mu.Lock()
	defer rd.mu.Unlock()
	rd.baselines[name] = benchmark
}

// CheckRegression checks if a benchmark shows regression
func (rd *RegressionDetector) CheckRegression(name string, current *BenchmarkData) *RegressionResult {
	rd.mu.RLock()
	baseline, exists := rd.baselines[name]
	rd.mu.RUnlock()

	if !exists {
		return &RegressionResult{
			Name:        name,
			HasBaseline: false,
		}
	}

	durationRegression := float64(current.AvgDuration-baseline.AvgDuration) / float64(baseline.AvgDuration)
	memoryRegression := float64(int64(current.BytesPerOp)-int64(baseline.BytesPerOp)) / float64(baseline.BytesPerOp)

	return &RegressionResult{
		Name:               name,
		HasBaseline:        true,
		DurationRegression: durationRegression,
		MemoryRegression:   memoryRegression,
		IsRegression:       durationRegression > rd.threshold || memoryRegression > rd.threshold,
		Baseline:           baseline,
		Current:            current,
	}
}

// RegressionResult represents the result of a regression check
type RegressionResult struct {
	Name               string
	HasBaseline        bool
	DurationRegression float64
	MemoryRegression   float64
	IsRegression       bool
	Baseline           *BenchmarkData
	Current            *BenchmarkData
}

// String returns a string representation of the regression result
func (rr *RegressionResult) String() string {
	if !rr.HasBaseline {
		return fmt.Sprintf("No baseline for %s", rr.Name)
	}

	if rr.IsRegression {
		return fmt.Sprintf("REGRESSION in %s: Duration %.2f%%, Memory %.2f%%",
			rr.Name, rr.DurationRegression*100, rr.MemoryRegression*100)
	}

	return fmt.Sprintf("No regression in %s: Duration %.2f%%, Memory %.2f%%",
		rr.Name, rr.DurationRegression*100, rr.MemoryRegression*100)
}

// PerformanceTestSuite provides a comprehensive performance testing framework
type PerformanceTestSuite struct {
	profiler           *Profiler
	regressionDetector *RegressionDetector
	tests              map[string]TestCase
	results            map[string]*TestResult
	mu                 sync.RWMutex
}

// TestCase represents a performance test case
type TestCase struct {
	Name        string
	Setup       func() error
	Test        func() error
	Teardown    func() error
	Iterations  int
	Timeout     time.Duration
	MemoryLimit uint64
}

// TestResult represents the result of a performance test
type TestResult struct {
	Name             string
	Passed           bool
	Duration         time.Duration
	Iterations       int
	ProfileData      *ProfileData
	BenchmarkData    *BenchmarkData
	Error            error
	RegressionResult *RegressionResult
}

// NewPerformanceTestSuite creates a new performance test suite
func NewPerformanceTestSuite() *PerformanceTestSuite {
	return &PerformanceTestSuite{
		profiler:           NewProfiler(),
		regressionDetector: NewRegressionDetector(0.1),
		tests:              make(map[string]TestCase),
		results:            make(map[string]*TestResult),
	}
}

// AddTest adds a test case to the suite
func (pts *PerformanceTestSuite) AddTest(test TestCase) {
	pts.mu.Lock()
	defer pts.mu.Unlock()
	pts.tests[test.Name] = test
}

// RunTest runs a specific test case
func (pts *PerformanceTestSuite) RunTest(name string) *TestResult {
	pts.mu.RLock()
	test, exists := pts.tests[name]
	pts.mu.RUnlock()

	if !exists {
		return &TestResult{
			Name:   name,
			Passed: false,
			Error:  gerror.New(gerror.ErrCodeInternal, "test not found", nil),
		}
	}

	result := &TestResult{
		Name:       name,
		Iterations: test.Iterations,
	}

	start := time.Now()

	// Setup
	if test.Setup != nil {
		if err := test.Setup(); err != nil {
			result.Error = gerror.Wrap(err, gerror.ErrCodeInternal, "setup failed")
			result.Passed = false
			return result
		}
	}

	// Setup timeout
	ctx := context.Background()
	if test.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, test.Timeout)
		defer cancel()
	}

	// Profile the test
	pts.profiler.Enable()
	profileCtx := pts.profiler.StartProfile(name)

	// Run benchmark
	benchmarkData := pts.profiler.Benchmark(name, func() {
		if err := test.Test(); err != nil {
			result.Error = gerror.Wrap(err, gerror.ErrCodeInternal, "test failed")
			result.Passed = false
		}
	})

	// End profiling
	profileData := profileCtx.End()

	result.Duration = time.Since(start)
	result.ProfileData = profileData
	result.BenchmarkData = benchmarkData

	// Check for regression
	if benchmarkData != nil {
		result.RegressionResult = pts.regressionDetector.CheckRegression(name, benchmarkData)
	}

	// Teardown
	if test.Teardown != nil {
		if err := test.Teardown(); err != nil {
			result.Error = gerror.Wrap(err, gerror.ErrCodeInternal, "teardown failed")
			result.Passed = false
		}
	}

	// Check if test passed
	result.Passed = result.Error == nil &&
		(result.RegressionResult == nil || !result.RegressionResult.IsRegression)

	pts.mu.Lock()
	pts.results[name] = result
	pts.mu.Unlock()

	return result
}

// RunAllTests runs all test cases
func (pts *PerformanceTestSuite) RunAllTests() map[string]*TestResult {
	pts.mu.RLock()
	tests := make(map[string]TestCase)
	for name, test := range pts.tests {
		tests[name] = test
	}
	pts.mu.RUnlock()

	results := make(map[string]*TestResult)
	for name := range tests {
		results[name] = pts.RunTest(name)
	}

	return results
}

// GetResults returns all test results
func (pts *PerformanceTestSuite) GetResults() map[string]*TestResult {
	pts.mu.RLock()
	defer pts.mu.RUnlock()

	results := make(map[string]*TestResult)
	for name, result := range pts.results {
		results[name] = result
	}
	return results
}

// SetBaseline sets performance baselines from current results
func (pts *PerformanceTestSuite) SetBaseline() {
	pts.mu.RLock()
	defer pts.mu.RUnlock()

	for name, result := range pts.results {
		if result.BenchmarkData != nil {
			pts.regressionDetector.SetBaseline(name, result.BenchmarkData)
		}
	}
}
