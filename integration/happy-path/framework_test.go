package happypath

import (
	"context"
	"fmt"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/lancekrogers/guild/pkg/gerror"
	"github.com/stretchr/testify/require"
)

// HappyPathTestFramework provides common utilities for happy path testing
type HappyPathTestFramework struct {
	t           *testing.T
	cleanup     []func()
	portManager *PortManager
	mu          sync.Mutex
}

// PortManager manages port allocation for tests
type PortManager struct {
	basePort int
	used     map[int]bool
	mu       sync.Mutex
}

// NewPortManager creates a new port manager
func NewPortManager(basePort int) *PortManager {
	return &PortManager{
		basePort: basePort,
		used:     make(map[int]bool),
	}
}

// GetAvailablePort returns an available port
func (pm *PortManager) GetAvailablePort() (int, error) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	for port := pm.basePort; port < pm.basePort+1000; port++ {
		if pm.used[port] {
			continue
		}

		// Test if port is actually available
		if pm.isPortAvailable(port) {
			pm.used[port] = true
			return port, nil
		}
	}

	return 0, gerror.New(gerror.ErrCodeInternal, "no available ports", nil)
}

// ReleasePort releases a port back to the pool
func (pm *PortManager) ReleasePort(port int) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	delete(pm.used, port)
}

// isPortAvailable checks if a port is available
func (pm *PortManager) isPortAvailable(port int) bool {
	addr := fmt.Sprintf(":%d", port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return false
	}
	listener.Close()
	return true
}

// NewHappyPathTestFramework creates a new test framework
func NewHappyPathTestFramework(t *testing.T) *HappyPathTestFramework {
	return &HappyPathTestFramework{
		t:           t,
		cleanup:     make([]func(), 0),
		portManager: NewPortManager(8000),
	}
}

// GetAvailablePort returns an available port for testing
func (f *HappyPathTestFramework) GetAvailablePort() int {
	port, err := f.portManager.GetAvailablePort()
	require.NoError(f.t, err, "Failed to get available port")

	f.cleanup = append(f.cleanup, func() {
		f.portManager.ReleasePort(port)
	})

	return port
}

// AddCleanup adds a cleanup function
func (f *HappyPathTestFramework) AddCleanup(fn func()) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.cleanup = append(f.cleanup, fn)
}

// Cleanup performs all registered cleanup operations
func (f *HappyPathTestFramework) Cleanup() {
	f.mu.Lock()
	defer f.mu.Unlock()

	for i := len(f.cleanup) - 1; i >= 0; i-- {
		f.cleanup[i]()
	}
}

// WaitForCondition waits for a condition to be met
func (f *HappyPathTestFramework) WaitForCondition(ctx context.Context, condition func() bool, checkInterval time.Duration) error {
	ticker := time.NewTicker(checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if condition() {
				return nil
			}
		}
	}
}

// ParallelExecute executes multiple functions in parallel
func (f *HappyPathTestFramework) ParallelExecute(ctx context.Context, fns ...func(context.Context) error) error {
	var wg sync.WaitGroup
	errChan := make(chan error, len(fns))

	for _, fn := range fns {
		wg.Add(1)
		go func(fn func(context.Context) error) {
			defer wg.Done()
			if err := fn(ctx); err != nil {
				errChan <- err
			}
		}(fn)
	}

	wg.Wait()
	close(errChan)

	// Return first error if any
	for err := range errChan {
		return err
	}

	return nil
}

// TestMetrics tracks test execution metrics
type TestMetrics struct {
	TestName      string
	StartTime     time.Time
	EndTime       time.Time
	Duration      time.Duration
	Success       bool
	ErrorMessage  string
	Assertions    int
	MemoryUsageMB int
}

// MetricsCollector collects test metrics
type MetricsCollector struct {
	metrics []TestMetrics
	mu      sync.Mutex
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector() *MetricsCollector {
	return &MetricsCollector{
		metrics: make([]TestMetrics, 0),
	}
}

// StartTest starts tracking a test
func (mc *MetricsCollector) StartTest(testName string) *TestMetrics {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	metric := TestMetrics{
		TestName:  testName,
		StartTime: time.Now(),
	}

	mc.metrics = append(mc.metrics, metric)
	return &mc.metrics[len(mc.metrics)-1]
}

// EndTest ends tracking a test
func (mc *MetricsCollector) EndTest(metric *TestMetrics, success bool, errorMessage string) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	metric.EndTime = time.Now()
	metric.Duration = metric.EndTime.Sub(metric.StartTime)
	metric.Success = success
	metric.ErrorMessage = errorMessage
}

// GetMetrics returns collected metrics
func (mc *MetricsCollector) GetMetrics() []TestMetrics {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	result := make([]TestMetrics, len(mc.metrics))
	copy(result, mc.metrics)
	return result
}

// PrintSummary prints a summary of test metrics
func (mc *MetricsCollector) PrintSummary(t *testing.T) {
	metrics := mc.GetMetrics()

	totalTests := len(metrics)
	successfulTests := 0
	totalDuration := time.Duration(0)

	for _, metric := range metrics {
		if metric.Success {
			successfulTests++
		}
		totalDuration += metric.Duration
	}

	t.Logf("📊 Test Execution Summary:")
	t.Logf("   - Total Tests: %d", totalTests)
	t.Logf("   - Successful: %d", successfulTests)
	t.Logf("   - Failed: %d", totalTests-successfulTests)
	t.Logf("   - Success Rate: %.1f%%", float64(successfulTests)/float64(totalTests)*100)
	t.Logf("   - Total Duration: %v", totalDuration)
	t.Logf("   - Average Duration: %v", totalDuration/time.Duration(totalTests))
}

// MockResourceMonitor monitors resource usage during tests
type MockResourceMonitor struct {
	startTime time.Time
	samples   []ResourceSample
	mu        sync.RWMutex
}

// ResourceSample represents a resource usage sample
type ResourceSample struct {
	Timestamp  time.Time
	MemoryMB   int
	CPUPercent float64
	Goroutines int
}

// NewMockResourceMonitor creates a new resource monitor
func NewMockResourceMonitor() *MockResourceMonitor {
	return &MockResourceMonitor{
		startTime: time.Now(),
		samples:   make([]ResourceSample, 0),
	}
}

// Start starts resource monitoring
func (m *MockResourceMonitor) Start() {
	go m.monitor()
}

// monitor runs the monitoring loop
func (m *MockResourceMonitor) monitor() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			m.collectSample()
		}
	}
}

// collectSample collects a resource usage sample
func (m *MockResourceMonitor) collectSample() {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Mock resource usage values
	sample := ResourceSample{
		Timestamp:  time.Now(),
		MemoryMB:   100 + len(m.samples)*2,            // Simulate gradual memory increase
		CPUPercent: 10.0 + float64(len(m.samples)%20), // Simulate CPU variation
		Goroutines: 50 + len(m.samples)/10,            // Simulate goroutine growth
	}

	m.samples = append(m.samples, sample)

	// Keep only last 100 samples
	if len(m.samples) > 100 {
		m.samples = m.samples[1:]
	}
}

// GetPeakUsage returns peak resource usage
func (m *MockResourceMonitor) GetPeakUsage() ResourceSample {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if len(m.samples) == 0 {
		return ResourceSample{}
	}

	peak := m.samples[0]
	for _, sample := range m.samples {
		if sample.MemoryMB > peak.MemoryMB {
			peak.MemoryMB = sample.MemoryMB
		}
		if sample.CPUPercent > peak.CPUPercent {
			peak.CPUPercent = sample.CPUPercent
		}
		if sample.Goroutines > peak.Goroutines {
			peak.Goroutines = sample.Goroutines
		}
	}

	return peak
}

// GetAverageUsage returns average resource usage
func (m *MockResourceMonitor) GetAverageUsage() ResourceSample {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if len(m.samples) == 0 {
		return ResourceSample{}
	}

	totalMemory := 0
	totalCPU := 0.0
	totalGoroutines := 0

	for _, sample := range m.samples {
		totalMemory += sample.MemoryMB
		totalCPU += sample.CPUPercent
		totalGoroutines += sample.Goroutines
	}

	count := len(m.samples)

	return ResourceSample{
		Timestamp:  time.Now(),
		MemoryMB:   totalMemory / count,
		CPUPercent: totalCPU / float64(count),
		Goroutines: totalGoroutines / count,
	}
}

// TestDataGenerator generates realistic test data
type TestDataGenerator struct {
	projectPath string
	fileTypes   []string
}

// NewTestDataGenerator creates a new test data generator
func NewTestDataGenerator(projectPath string) *TestDataGenerator {
	return &TestDataGenerator{
		projectPath: projectPath,
		fileTypes:   []string{".go", ".md", ".yaml", ".json", ".sql"},
	}
}

// GenerateRealisticProject generates a realistic project structure
func (g *TestDataGenerator) GenerateRealisticProject() map[string]string {
	return map[string]string{
		"main.go": `package main

import (
	"fmt"
	"log"
)

func main() {
	fmt.Println("Hello, Guild Framework!")
}`,
		"README.md": `# Test Project

This is a test project for Guild Framework integration testing.

## Features

- Agent orchestration
- Kanban task management  
- Multi-provider AI integration

## Usage

Run with: go run main.go
`,
		"guild.yaml": `name: test-project
version: 1.0.0
agents:
  - name: developer
    role: code-reviewer
    capabilities: ["code-analysis", "documentation"]
providers:
  - name: openai
    type: openai
    priority: 1
  - name: anthropic  
    type: anthropic
    priority: 2
`,
		"pkg/models/agent.go": `package models

type Agent struct {
	ID   string
	Name string
	Role string
}`,
		"pkg/services/orchestrator.go": `package services

import "context"

type Orchestrator struct{}

func (o *Orchestrator) Execute(ctx context.Context) error {
	return nil
}`,
	}
}

// GenerateCommissionText generates realistic commission text
func (g *TestDataGenerator) GenerateCommissionText(complexity string) string {
	switch complexity {
	case "simple":
		return "Create a simple REST API endpoint that returns a list of users"
	case "medium":
		return "Implement a user authentication system with JWT tokens, password hashing, and role-based access control"
	case "complex":
		return "Design and implement a distributed microservices architecture with API gateway, service discovery, load balancing, and event-driven communication"
	default:
		return "Implement a basic CRUD operation for managing resources"
	}
}

// ValidationHelper provides common validation utilities
type ValidationHelper struct{}

// NewValidationHelper creates a new validation helper
func NewValidationHelper() *ValidationHelper {
	return &ValidationHelper{}
}

// ValidateLatency validates response latency
func (v *ValidationHelper) ValidateLatency(t *testing.T, actual time.Duration, expected time.Duration, tolerance float64) {
	maxAllowed := time.Duration(float64(expected) * (1.0 + tolerance))
	require.LessOrEqual(t, actual, maxAllowed,
		"Latency exceeded tolerance: %v > %v (tolerance: %.1f%%)",
		actual, maxAllowed, tolerance*100)
}

// ValidateSuccessRate validates success rate
func (v *ValidationHelper) ValidateSuccessRate(t *testing.T, actual float64, expected float64, tolerance float64) {
	minAllowed := expected * (1.0 - tolerance)
	require.GreaterOrEqual(t, actual, minAllowed,
		"Success rate below tolerance: %.2f%% < %.2f%% (tolerance: %.1f%%)",
		actual*100, minAllowed*100, tolerance*100)
}

// ValidateResourceUsage validates resource usage
func (v *ValidationHelper) ValidateResourceUsage(t *testing.T, usage ResourceSample, limits ResourceSample) {
	require.LessOrEqual(t, usage.MemoryMB, limits.MemoryMB,
		"Memory usage exceeded limit: %d MB > %d MB", usage.MemoryMB, limits.MemoryMB)
	require.LessOrEqual(t, usage.CPUPercent, limits.CPUPercent,
		"CPU usage exceeded limit: %.1f%% > %.1f%%", usage.CPUPercent, limits.CPUPercent)
	require.LessOrEqual(t, usage.Goroutines, limits.Goroutines,
		"Goroutine count exceeded limit: %d > %d", usage.Goroutines, limits.Goroutines)
}

// ConcurrencyHelper provides concurrency testing utilities
type ConcurrencyHelper struct{}

// NewConcurrencyHelper creates a new concurrency helper
func NewConcurrencyHelper() *ConcurrencyHelper {
	return &ConcurrencyHelper{}
}

// ExecuteConcurrently executes functions concurrently with controlled parallelism
func (c *ConcurrencyHelper) ExecuteConcurrently(ctx context.Context, maxConcurrency int, tasks []func(context.Context) error) error {
	semaphore := make(chan struct{}, maxConcurrency)
	var wg sync.WaitGroup
	errChan := make(chan error, len(tasks))

	for _, task := range tasks {
		wg.Add(1)
		go func(task func(context.Context) error) {
			defer wg.Done()

			// Acquire semaphore
			select {
			case semaphore <- struct{}{}:
			case <-ctx.Done():
				errChan <- ctx.Err()
				return
			}

			// Execute task
			if err := task(ctx); err != nil {
				errChan <- err
			}

			// Release semaphore
			<-semaphore
		}(task)
	}

	wg.Wait()
	close(errChan)

	// Return first error if any
	for err := range errChan {
		return err
	}

	return nil
}

// LoadTestHelper provides load testing utilities
type LoadTestHelper struct {
	rampUpDuration   time.Duration
	sustainDuration  time.Duration
	rampDownDuration time.Duration
}

// NewLoadTestHelper creates a new load test helper
func NewLoadTestHelper(rampUp, sustain, rampDown time.Duration) *LoadTestHelper {
	return &LoadTestHelper{
		rampUpDuration:   rampUp,
		sustainDuration:  sustain,
		rampDownDuration: rampDown,
	}
}

// ExecuteLoadTest executes a load test with ramp-up and ramp-down
func (l *LoadTestHelper) ExecuteLoadTest(ctx context.Context, maxRPS int, taskFactory func() func() error) *LoadTestResults {
	results := &LoadTestResults{
		StartTime:          time.Now(),
		TotalRequests:      0,
		SuccessfulRequests: 0,
		FailedRequests:     0,
		Errors:             make(map[string]int),
	}

	// Implementation would include ramp-up, sustain, and ramp-down phases
	// This is a simplified mock implementation

	results.EndTime = time.Now()
	results.Duration = results.EndTime.Sub(results.StartTime)

	if results.TotalRequests > 0 {
		results.SuccessRate = float64(results.SuccessfulRequests) / float64(results.TotalRequests)
	}

	return results
}

// LoadTestResults contains load test results
type LoadTestResults struct {
	StartTime          time.Time
	EndTime            time.Time
	Duration           time.Duration
	TotalRequests      int
	SuccessfulRequests int
	FailedRequests     int
	SuccessRate        float64
	Errors             map[string]int
	AverageLatency     time.Duration
	MaxLatency         time.Duration
	MinLatency         time.Duration
}

// TestHappyPathFramework validates the framework utilities
func TestHappyPathFramework(t *testing.T) {
	framework := NewHappyPathTestFramework(t)
	defer framework.Cleanup()

	t.Run("PortManager", func(t *testing.T) {
		port1 := framework.GetAvailablePort()
		port2 := framework.GetAvailablePort()

		require.NotEqual(t, port1, port2, "Ports should be different")
		require.Greater(t, port1, 8000, "Port should be in expected range")
		require.Greater(t, port2, 8000, "Port should be in expected range")
	})

	t.Run("MetricsCollector", func(t *testing.T) {
		collector := NewMetricsCollector()

		metric := collector.StartTest("test-function")
		time.Sleep(100 * time.Millisecond)
		collector.EndTest(metric, true, "")

		metrics := collector.GetMetrics()
		require.Len(t, metrics, 1)
		require.True(t, metrics[0].Success)
		require.Greater(t, metrics[0].Duration, 100*time.Millisecond)

		collector.PrintSummary(t)
	})

	t.Run("ResourceMonitor", func(t *testing.T) {
		monitor := NewMockResourceMonitor()
		monitor.Start()

		time.Sleep(2100 * time.Millisecond) // Allow some samples

		peak := monitor.GetPeakUsage()
		avg := monitor.GetAverageUsage()

		require.Greater(t, peak.MemoryMB, 0)
		require.Greater(t, avg.MemoryMB, 0)
		require.GreaterOrEqual(t, peak.MemoryMB, avg.MemoryMB)
	})

	t.Run("TestDataGenerator", func(t *testing.T) {
		generator := NewTestDataGenerator("/test/project")

		project := generator.GenerateRealisticProject()
		require.NotEmpty(t, project)
		require.Contains(t, project, "main.go")
		require.Contains(t, project, "README.md")
		require.Contains(t, project, "guild.yaml")

		commission := generator.GenerateCommissionText("complex")
		require.Contains(t, commission, "microservices")
	})

	t.Run("ValidationHelper", func(t *testing.T) {
		validator := NewValidationHelper()

		// These should not panic/fail
		validator.ValidateLatency(t, 100*time.Millisecond, 200*time.Millisecond, 0.1)
		validator.ValidateSuccessRate(t, 0.95, 0.90, 0.1)

		usage := ResourceSample{MemoryMB: 100, CPUPercent: 50.0, Goroutines: 100}
		limits := ResourceSample{MemoryMB: 200, CPUPercent: 80.0, Goroutines: 200}
		validator.ValidateResourceUsage(t, usage, limits)
	})

	t.Run("ConcurrencyHelper", func(t *testing.T) {
		helper := NewConcurrencyHelper()

		tasks := make([]func(context.Context) error, 10)
		for i := range tasks {
			tasks[i] = func(ctx context.Context) error {
				time.Sleep(50 * time.Millisecond)
				return nil
			}
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		start := time.Now()
		err := helper.ExecuteConcurrently(ctx, 3, tasks)
		duration := time.Since(start)

		require.NoError(t, err)
		require.Less(t, duration, 1*time.Second) // Should complete faster than sequential
	})
}
