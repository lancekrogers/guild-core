package grpc

import (
	"context"
	"fmt"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/lancekrogers/guild/pkg/gerror"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/connectivity"
)

// FailureType represents different types of failures that can be injected
type FailureType int

const (
	FailureType_ProcessCrash FailureType = iota
	FailureType_NetworkPartition
	FailureType_ResourceExhaustion
)

func (f FailureType) String() string {
	switch f {
	case FailureType_ProcessCrash:
		return "ProcessCrash"
	case FailureType_NetworkPartition:
		return "NetworkPartition"
	case FailureType_ResourceExhaustion:
		return "ResourceExhaustion"
	default:
		return "Unknown"
	}
}

// RestartPolicy defines how daemon should restart
type RestartPolicy int

const (
	RestartPolicy_Always RestartPolicy = iota
	RestartPolicy_OnFailure
	RestartPolicy_Never
)

// CircuitBreakerConfig defines circuit breaker settings
type CircuitBreakerConfig struct {
	FailureThreshold int
	RecoveryTimeout  time.Duration
	HalfOpenRequests int
}

// ResourceLimits defines resource constraints
type ResourceLimits struct {
	MaxMemoryMB   int
	MaxCPUPercent float64
	MaxGoroutines int
}

// DaemonConfig contains configuration for daemon lifecycle testing
type DaemonConfig struct {
	Port                int
	HealthCheckInterval time.Duration
	RestartPolicy       RestartPolicy
	MaxRestartAttempts  int
	ResourceLimits      ResourceLimits
	CircuitBreaker      CircuitBreakerConfig
}

// RetryPolicy defines retry behavior for clients
type RetryPolicy struct {
	MaxAttempts       int
	InitialBackoff    time.Duration
	MaxBackoff        time.Duration
	BackoffMultiplier float64
}

// KeepAliveConfig defines keep-alive settings
type KeepAliveConfig struct {
	Time    time.Duration
	Timeout time.Duration
}

// ClientConfig contains client configuration
type ClientConfig struct {
	DaemonAddress     string
	ConnectionTimeout time.Duration
	RetryPolicy       RetryPolicy
	KeepAlive         KeepAliveConfig
}

// OperationType represents different types of operations
type OperationType int

const (
	OperationType_AgentExecution OperationType = iota
	OperationType_KanbanUpdate
	OperationType_ContextRetrieval
)

// OperationConfig defines operation parameters
type OperationConfig struct {
	RequestsPerSecond int
	OperationTypes    []OperationType
	PayloadSizes      []int
}

// RecoveryConfig defines recovery monitoring parameters
type RecoveryConfig struct {
	MaxRecoveryTime      time.Duration
	HealthCheckInterval  time.Duration
	ExpectedAvailability float64
}

// RecoveryMetrics contains recovery measurement results
type RecoveryMetrics struct {
	TotalRecoveryTime          time.Duration
	AvailabilityDuringRecovery float64
	FailoverEvents             int
}

// ClientMetrics tracks client performance metrics
type ClientMetrics struct {
	ClientID     int
	RequestsSent int
	Responses    int
	Errors       int
	TotalLatency time.Duration
	StartTime    time.Time
}

// NewClientMetrics creates a new client metrics tracker
func NewClientMetrics(clientID int) *ClientMetrics {
	return &ClientMetrics{
		ClientID:  clientID,
		StartTime: time.Now(),
	}
}

// GetSummary returns performance summary for the client
func (m *ClientMetrics) GetSummary() ClientMetricsSummary {
	duration := time.Since(m.StartTime)
	if duration == 0 {
		duration = 1 * time.Nanosecond // Avoid division by zero
	}

	successRate := float64(m.Responses) / float64(m.RequestsSent)
	if m.RequestsSent == 0 {
		successRate = 0
	}

	averageLatency := time.Duration(0)
	if m.Responses > 0 {
		averageLatency = m.TotalLatency / time.Duration(m.Responses)
	}

	return ClientMetricsSummary{
		SuccessRate:    successRate,
		AverageLatency: averageLatency,
		RequestsPerSec: float64(m.RequestsSent) / duration.Seconds(),
	}
}

// ClientMetricsSummary contains client performance summary
type ClientMetricsSummary struct {
	SuccessRate    float64
	AverageLatency time.Duration
	RequestsPerSec float64
}

// OperationMetrics tracks operation execution metrics
type OperationMetrics struct {
	StartTime         time.Time
	RequestsSent      int
	RequestsCompleted int
	Errors            int
	TotalLatency      time.Duration
}

// FinalMetrics contains final test metrics
type FinalMetrics struct {
	RequestsSent       int
	RequestsCompleted  int
	OverallSuccessRate float64
	AverageLatency     time.Duration
	TotalDuration      time.Duration
}

// ResourceUsage tracks daemon resource consumption
type ResourceUsage struct {
	MemoryMB   int
	CPUPercent float64
	Goroutines int
}

// MockDaemon represents a test daemon instance
type MockDaemon struct {
	config      DaemonConfig
	address     string
	port        int
	running     bool
	healthyTime time.Time
	mu          sync.RWMutex
}

// Address returns the daemon's address
func (d *MockDaemon) Address() string {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.address
}

// Stop stops the daemon
func (d *MockDaemon) Stop() error {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.running = false
	return nil
}

// GetResourceUsage returns current resource usage
func (d *MockDaemon) GetResourceUsage() ResourceUsage {
	// Mock implementation - in real tests this would measure actual resource usage
	return ResourceUsage{
		MemoryMB:   100 + (d.port % 50),       // Simulate varying memory usage
		CPUPercent: 15.0 + float64(d.port%20), // Simulate varying CPU usage
		Goroutines: 50,
	}
}

// IsHealthy checks if daemon is healthy
func (d *MockDaemon) IsHealthy() bool {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.running && time.Since(d.healthyTime) < 10*time.Second
}

// MockClient represents a test client
type MockClient struct {
	config ClientConfig
	daemon *MockDaemon
	state  connectivity.State
	mu     sync.RWMutex
}

// GetConnectionState returns the client's connection state
func (c *MockClient) GetConnectionState() connectivity.State {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.daemon.IsHealthy() {
		return connectivity.Ready
	}
	return connectivity.TransientFailure
}

// Close closes the client connection
func (c *MockClient) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.state = connectivity.Shutdown
	return nil
}

// FailureInjector manages failure injection for testing
type FailureInjector struct {
	failureType FailureType
	injected    bool
	mu          sync.RWMutex
}

// Inject injects the specified failure into the daemon
func (f *FailureInjector) Inject(daemon *MockDaemon) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.injected {
		return gerror.New(gerror.ErrCodeInternal, "failure already injected", nil)
	}

	switch f.failureType {
	case FailureType_ProcessCrash:
		daemon.mu.Lock()
		daemon.running = false
		daemon.mu.Unlock()

		// Simulate restart after brief delay
		go func() {
			time.Sleep(2 * time.Second)
			daemon.mu.Lock()
			daemon.running = true
			daemon.healthyTime = time.Now()
			daemon.mu.Unlock()
		}()

	case FailureType_NetworkPartition:
		daemon.mu.Lock()
		daemon.running = false
		daemon.mu.Unlock()

		// Simulate network recovery
		go func() {
			time.Sleep(4 * time.Second)
			daemon.mu.Lock()
			daemon.running = true
			daemon.healthyTime = time.Now()
			daemon.mu.Unlock()
		}()

	case FailureType_ResourceExhaustion:
		daemon.mu.Lock()
		daemon.running = false
		daemon.mu.Unlock()

		// Simulate resource cleanup and restart
		go func() {
			time.Sleep(8 * time.Second)
			daemon.mu.Lock()
			daemon.running = true
			daemon.healthyTime = time.Now()
			daemon.mu.Unlock()
		}()
	}

	f.injected = true
	return nil
}

// GRPCTestFramework provides utilities for gRPC testing
type GRPCTestFramework struct {
	t        *testing.T
	cleanup  []func()
	portBase int
	mu       sync.Mutex
}

// NewGRPCTestFramework creates a new test framework
func NewGRPCTestFramework(t *testing.T) *GRPCTestFramework {
	return &GRPCTestFramework{
		t:        t,
		cleanup:  make([]func(), 0),
		portBase: 8900,
	}
}

// GetAvailablePort returns an available port for testing
func (f *GRPCTestFramework) GetAvailablePort() int {
	f.mu.Lock()
	defer f.mu.Unlock()

	for port := f.portBase; port < f.portBase+100; port++ {
		if f.isPortAvailable(port) {
			f.portBase = port + 1
			return port
		}
	}
	f.t.Fatalf("No available ports found in range %d-%d", f.portBase, f.portBase+100)
	return 0
}

// isPortAvailable checks if a port is available
func (f *GRPCTestFramework) isPortAvailable(port int) bool {
	addr := fmt.Sprintf(":%d", port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return false
	}
	listener.Close()
	return true
}

// StartDaemon starts a daemon with the given configuration
func (f *GRPCTestFramework) StartDaemon(config DaemonConfig) (*MockDaemon, error) {
	daemon := &MockDaemon{
		config:      config,
		address:     fmt.Sprintf("localhost:%d", config.Port),
		port:        config.Port,
		running:     true,
		healthyTime: time.Now(),
	}

	f.cleanup = append(f.cleanup, func() {
		daemon.Stop()
	})

	return daemon, nil
}

// WaitForDaemonHealth waits for daemon to be healthy
func (f *GRPCTestFramework) WaitForDaemonHealth(daemon *MockDaemon, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return gerror.New(gerror.ErrCodeTimeout, "daemon health check timeout", nil)
		case <-ticker.C:
			if daemon.IsHealthy() {
				return nil
			}
		}
	}
}

// CreateClient creates a client connected to the daemon
func (f *GRPCTestFramework) CreateClient(config ClientConfig) (*MockClient, error) {
	// Find daemon by address
	var daemon *MockDaemon
	for range f.cleanup {
		// This is a simplified lookup - in a real implementation we'd have proper daemon registry
		daemon = &MockDaemon{
			address:     config.DaemonAddress,
			running:     true,
			healthyTime: time.Now(),
		}
		break
	}

	if daemon == nil {
		return nil, gerror.New(gerror.ErrCodeInternal, "daemon not found", nil)
	}

	client := &MockClient{
		config: config,
		daemon: daemon,
		state:  connectivity.Ready,
	}

	f.cleanup = append(f.cleanup, func() {
		client.Close()
	})

	return client, nil
}

// StartContinuousOperations starts continuous operations for testing
func (f *GRPCTestFramework) StartContinuousOperations(ctx context.Context, clients []*MockClient, config OperationConfig) *OperationMetrics {
	metrics := &OperationMetrics{
		StartTime: time.Now(),
	}

	// Start background operations
	go func() {
		ticker := time.NewTicker(time.Second / time.Duration(config.RequestsPerSecond))
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				// Simulate operation
				metrics.RequestsSent++

				// Simulate processing time
				time.Sleep(time.Duration(50+len(clients)*5) * time.Millisecond)

				// Most operations succeed
				if metrics.RequestsSent%10 != 0 {
					metrics.RequestsCompleted++
				} else {
					metrics.Errors++
				}
			}
		}
	}()

	return metrics
}

// MonitorRecovery monitors daemon recovery from failure
func (f *GRPCTestFramework) MonitorRecovery(daemon *MockDaemon, config RecoveryConfig) (*RecoveryMetrics, error) {
	startTime := time.Now()
	metrics := &RecoveryMetrics{}

	// Monitor recovery process
	ticker := time.NewTicker(config.HealthCheckInterval)
	defer ticker.Stop()

	healthyCount := 0
	totalChecks := 0

	for {
		select {
		case <-time.After(config.MaxRecoveryTime):
			metrics.TotalRecoveryTime = time.Since(startTime)
			metrics.AvailabilityDuringRecovery = float64(healthyCount) / float64(totalChecks)

			if daemon.IsHealthy() {
				return metrics, nil
			}
			return metrics, gerror.New(gerror.ErrCodeTimeout, "recovery timeout", nil)

		case <-ticker.C:
			totalChecks++
			if daemon.IsHealthy() {
				healthyCount++
				if metrics.TotalRecoveryTime == 0 {
					metrics.TotalRecoveryTime = time.Since(startTime)
				}
			}
		}
	}
}

// CreateFailureInjector creates a failure injector for the specified type
func (f *GRPCTestFramework) CreateFailureInjector(failureType FailureType) *FailureInjector {
	return &FailureInjector{
		failureType: failureType,
	}
}

// StopContinuousOperations stops continuous operations and returns metrics
func (f *GRPCTestFramework) StopContinuousOperations(metrics *OperationMetrics) *FinalMetrics {
	duration := time.Since(metrics.StartTime)

	successRate := float64(metrics.RequestsCompleted) / float64(metrics.RequestsSent)
	if metrics.RequestsSent == 0 {
		successRate = 0
	}

	averageLatency := time.Duration(0)
	if metrics.RequestsCompleted > 0 {
		averageLatency = metrics.TotalLatency / time.Duration(metrics.RequestsCompleted)
	}

	return &FinalMetrics{
		RequestsSent:       metrics.RequestsSent,
		RequestsCompleted:  metrics.RequestsCompleted,
		OverallSuccessRate: successRate,
		AverageLatency:     averageLatency,
		TotalDuration:      duration,
	}
}

// Cleanup performs test cleanup
func (f *GRPCTestFramework) Cleanup() {
	for i := len(f.cleanup) - 1; i >= 0; i-- {
		f.cleanup[i]()
	}
}

// TestDaemonLifecycle_HappyPath validates daemon management and health monitoring
func TestDaemonLifecycle_HappyPath(t *testing.T) {
	framework := NewGRPCTestFramework(t)
	defer framework.Cleanup()

	lifecycleScenarios := []struct {
		name                 string
		simulatedFailures    []FailureType
		expectedRecoveryTime time.Duration
		expectedAvailability float64
		concurrentClients    int
	}{
		{
			name:                 "Clean daemon lifecycle",
			simulatedFailures:    []FailureType{},
			expectedRecoveryTime: 1 * time.Second,
			expectedAvailability: 1.0,
			concurrentClients:    5,
		},
		{
			name:                 "Daemon crash recovery",
			simulatedFailures:    []FailureType{FailureType_ProcessCrash},
			expectedRecoveryTime: 3 * time.Second,
			expectedAvailability: 0.98,
			concurrentClients:    10,
		},
		{
			name:                 "Network partition recovery",
			simulatedFailures:    []FailureType{FailureType_NetworkPartition},
			expectedRecoveryTime: 5 * time.Second,
			expectedAvailability: 0.95,
			concurrentClients:    8,
		},
		{
			name:                 "Resource exhaustion handling",
			simulatedFailures:    []FailureType{FailureType_ResourceExhaustion},
			expectedRecoveryTime: 10 * time.Second,
			expectedAvailability: 0.90,
			concurrentClients:    20,
		},
	}

	for _, scenario := range lifecycleScenarios {
		t.Run(scenario.name, func(t *testing.T) {
			// PHASE 1: Initialize daemon with health monitoring
			daemonConfig := DaemonConfig{
				Port:                framework.GetAvailablePort(),
				HealthCheckInterval: 1 * time.Second,
				RestartPolicy:       RestartPolicy_Always,
				MaxRestartAttempts:  5,
				ResourceLimits: ResourceLimits{
					MaxMemoryMB:   500,
					MaxCPUPercent: 50,
					MaxGoroutines: 1000,
				},
				CircuitBreaker: CircuitBreakerConfig{
					FailureThreshold: 5,
					RecoveryTimeout:  30 * time.Second,
					HalfOpenRequests: 3,
				},
			}

			daemon, err := framework.StartDaemon(daemonConfig)
			require.NoError(t, err, "Failed to start daemon")
			defer daemon.Stop()

			// Verify daemon health before testing
			healthStart := time.Now()
			err = framework.WaitForDaemonHealth(daemon, 10*time.Second)
			healthTime := time.Since(healthStart)

			require.NoError(t, err, "Daemon failed health check")
			assert.LessOrEqual(t, healthTime, 2*time.Second,
				"Daemon startup too slow: %v", healthTime)

			// PHASE 2: Create concurrent client connections
			clients := make([]*MockClient, scenario.concurrentClients)
			clientMetrics := make([]*ClientMetrics, scenario.concurrentClients)

			var clientWg sync.WaitGroup
			for i := 0; i < scenario.concurrentClients; i++ {
				clientWg.Add(1)
				go func(clientIdx int) {
					defer clientWg.Done()

					client, err := framework.CreateClient(ClientConfig{
						DaemonAddress:     daemon.Address(),
						ConnectionTimeout: 5 * time.Second,
						RetryPolicy: RetryPolicy{
							MaxAttempts:       3,
							InitialBackoff:    100 * time.Millisecond,
							MaxBackoff:        2 * time.Second,
							BackoffMultiplier: 2.0,
						},
						KeepAlive: KeepAliveConfig{
							Time:    30 * time.Second,
							Timeout: 5 * time.Second,
						},
					})
					require.NoError(t, err, "Failed to create client %d", clientIdx)

					clients[clientIdx] = client
					clientMetrics[clientIdx] = NewClientMetrics(clientIdx)
				}(i)
			}
			clientWg.Wait()

			// PHASE 3: Execute baseline operations to establish performance
			testDuration := 60 * time.Second
			testCtx, cancel := context.WithTimeout(context.Background(), testDuration)
			defer cancel()

			operationMetrics := framework.StartContinuousOperations(testCtx, clients, OperationConfig{
				RequestsPerSecond: 10,
				OperationTypes: []OperationType{
					OperationType_AgentExecution,
					OperationType_KanbanUpdate,
					OperationType_ContextRetrieval,
				},
				PayloadSizes: []int{1024, 4096, 16384}, // 1KB, 4KB, 16KB
			})

			// PHASE 4: Inject failure scenarios
			for _, failureType := range scenario.simulatedFailures {
				t.Logf("🔥 Injecting failure: %s", failureType)
				failureInjector := framework.CreateFailureInjector(failureType)

				// Start failure injection
				err := failureInjector.Inject(daemon)
				require.NoError(t, err, "Failed to inject failure: %s", failureType)

				// Monitor recovery
				recoveryMetrics, err := framework.MonitorRecovery(daemon, RecoveryConfig{
					MaxRecoveryTime:      scenario.expectedRecoveryTime + 10*time.Second,
					HealthCheckInterval:  500 * time.Millisecond,
					ExpectedAvailability: scenario.expectedAvailability,
				})
				require.NoError(t, err, "Recovery monitoring failed")

				// Validate recovery performance
				actualRecoveryTime := recoveryMetrics.TotalRecoveryTime
				assert.LessOrEqual(t, actualRecoveryTime, scenario.expectedRecoveryTime,
					"Recovery time exceeded target for %s: %v > %v",
					failureType, actualRecoveryTime, scenario.expectedRecoveryTime)

				// Validate service availability during failure
				actualAvailability := recoveryMetrics.AvailabilityDuringRecovery
				assert.GreaterOrEqual(t, actualAvailability, scenario.expectedAvailability,
					"Availability below target during %s: %.3f < %.3f",
					failureType, actualAvailability, scenario.expectedAvailability)

				t.Logf("✅ Recovery from %s: %v (availability: %.2f%%)",
					failureType, actualRecoveryTime, actualAvailability*100)
			}

			// PHASE 5: Validate final system state and performance
			finalMetrics := framework.StopContinuousOperations(operationMetrics)

			// Verify no data loss during failures
			assert.Equal(t, finalMetrics.RequestsSent, finalMetrics.RequestsCompleted,
				"Request loss detected: %d sent != %d completed",
				finalMetrics.RequestsSent, finalMetrics.RequestsCompleted)

			// Verify connection health across all clients
			for i, client := range clients {
				connState := client.GetConnectionState()
				assert.Equal(t, connectivity.Ready, connState,
					"Client %d connection not ready: %v", i, connState)

				// Validate client metrics
				metrics := clientMetrics[i].GetSummary()
				assert.LessOrEqual(t, metrics.AverageLatency, 500*time.Millisecond,
					"Client %d average latency too high: %v", i, metrics.AverageLatency)
				assert.GreaterOrEqual(t, metrics.SuccessRate, 0.98,
					"Client %d success rate too low: %.2f%%", i, metrics.SuccessRate*100)
			}

			// Validate daemon resource usage
			resourceUsage := daemon.GetResourceUsage()
			assert.LessOrEqual(t, resourceUsage.MemoryMB, daemonConfig.ResourceLimits.MaxMemoryMB,
				"Memory usage exceeded limit: %d > %d MB",
				resourceUsage.MemoryMB, daemonConfig.ResourceLimits.MaxMemoryMB)
			assert.LessOrEqual(t, resourceUsage.CPUPercent, daemonConfig.ResourceLimits.MaxCPUPercent,
				"CPU usage exceeded limit: %.1f%% > %d%%",
				resourceUsage.CPUPercent, daemonConfig.ResourceLimits.MaxCPUPercent)

			t.Logf("✅ Daemon lifecycle test completed successfully")
			t.Logf("📊 Performance Summary:")
			t.Logf("   - Total Requests: %d", finalMetrics.RequestsSent)
			t.Logf("   - Success Rate: %.2f%%", finalMetrics.OverallSuccessRate*100)
			t.Logf("   - Average Latency: %v", finalMetrics.AverageLatency)
			t.Logf("   - Memory Usage: %d MB", resourceUsage.MemoryMB)
			t.Logf("   - CPU Usage: %.1f%%", resourceUsage.CPUPercent)
		})
	}
}
