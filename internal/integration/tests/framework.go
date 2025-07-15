// Package tests provides the integration test framework for Guild
package tests

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/lancekrogers/guild/pkg/gerror"
	"github.com/lancekrogers/guild/pkg/observability"
)

// IntegrationTest represents a single integration test
type IntegrationTest struct {
	Name        string
	Description string
	Setup       func(ctx context.Context, t *testing.T) error
	Test        func(ctx context.Context, t *testing.T) error
	Teardown    func(ctx context.Context, t *testing.T) error
	Timeout     time.Duration
}

// IntegrationSuite represents a collection of integration tests
type IntegrationSuite struct {
	Name           string
	Description    string
	Tests          []IntegrationTest
	GlobalSetup    func(ctx context.Context, t *testing.T) error
	GlobalTeardown func(ctx context.Context, t *testing.T) error

	// Test execution configuration
	Parallel    bool
	MaxParallel int
	StopOnError bool
	RetryCount  int
	RetryDelay  time.Duration
}

// TestHarness provides utilities for integration testing
type TestHarness struct {
	ctx    context.Context
	logger observability.Logger

	// Component states
	services  map[string]ServiceState
	resources map[string]interface{}
	mu        sync.RWMutex

	// Test metrics
	metrics TestMetrics
}

// ServiceState tracks the state of a service during testing
type ServiceState struct {
	Name      string
	Started   bool
	Healthy   bool
	StartTime time.Time
	StopTime  time.Time
	Error     error
}

// TestMetrics tracks test execution metrics
type TestMetrics struct {
	TotalTests    int
	PassedTests   int
	FailedTests   int
	SkippedTests  int
	TotalDuration time.Duration
	mu            sync.Mutex
}

// NewTestHarness creates a new test harness
func NewTestHarness(ctx context.Context) *TestHarness {
	return &TestHarness{
		ctx:       ctx,
		logger:    observability.GetLogger(ctx),
		services:  make(map[string]ServiceState),
		resources: make(map[string]interface{}),
	}
}

// StartService starts a service and tracks its state
func (h *TestHarness) StartService(ctx context.Context, name string, starter func(context.Context) error) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	state := ServiceState{
		Name:      name,
		StartTime: time.Now(),
	}

	// Start the service
	if err := starter(ctx); err != nil {
		state.Error = gerror.Wrap(err, gerror.ErrCodeInternal, "failed to start service").
			WithComponent("test_harness").
			WithDetails("service", name)
		h.services[name] = state
		return state.Error
	}

	state.Started = true
	state.Healthy = true
	h.services[name] = state

	h.logger.InfoContext(ctx, "Service started successfully",
		"service", name,
		"duration", time.Since(state.StartTime))

	return nil
}

// StopService stops a service and tracks its state
func (h *TestHarness) StopService(ctx context.Context, name string, stopper func(context.Context) error) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	state, exists := h.services[name]
	if !exists {
		return gerror.New(gerror.ErrCodeNotFound, "service not found", nil).
			WithComponent("test_harness").
			WithDetails("service", name)
	}

	state.StopTime = time.Now()

	// Stop the service
	if err := stopper(ctx); err != nil {
		state.Error = gerror.Wrap(err, gerror.ErrCodeInternal, "failed to stop service").
			WithComponent("test_harness").
			WithDetails("service", name)
		h.services[name] = state
		return state.Error
	}

	state.Started = false
	state.Healthy = false
	h.services[name] = state

	h.logger.InfoContext(ctx, "Service stopped successfully",
		"service", name,
		"uptime", state.StopTime.Sub(state.StartTime))

	return nil
}

// WaitForService waits for a service to become healthy
func (h *TestHarness) WaitForService(ctx context.Context, name string, checker func(context.Context) error, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return gerror.Wrap(ctx.Err(), gerror.ErrCodeCancelled, "context cancelled while waiting for service").
				WithComponent("test_harness").
				WithDetails("service", name)
		case <-ticker.C:
			if time.Now().After(deadline) {
				return gerror.New(gerror.ErrCodeTimeout, "timeout waiting for service", nil).
					WithComponent("test_harness").
					WithDetails("service", name).
					WithDetails("timeout", timeout)
			}

			if err := checker(ctx); err == nil {
				h.mu.Lock()
				if state, exists := h.services[name]; exists {
					state.Healthy = true
					h.services[name] = state
				}
				h.mu.Unlock()
				return nil
			}
		}
	}
}

// SetResource stores a test resource
func (h *TestHarness) SetResource(key string, value interface{}) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.resources[key] = value
}

// GetResource retrieves a test resource
func (h *TestHarness) GetResource(key string) (interface{}, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	val, exists := h.resources[key]
	return val, exists
}

// RunSuite executes an integration test suite
func RunSuite(t *testing.T, suite IntegrationSuite) {
	ctx := context.Background()
	logger := observability.GetLogger(ctx)

	// Global setup
	if suite.GlobalSetup != nil {
		logger.InfoContext(ctx, "Running global setup", "suite", suite.Name)
		if err := suite.GlobalSetup(ctx, t); err != nil {
			t.Fatalf("Global setup failed: %v", err)
		}
	}

	// Ensure global teardown runs
	if suite.GlobalTeardown != nil {
		defer func() {
			logger.InfoContext(ctx, "Running global teardown", "suite", suite.Name)
			if err := suite.GlobalTeardown(ctx, t); err != nil {
				t.Errorf("Global teardown failed: %v", err)
			}
		}()
	}

	// Run tests
	for _, test := range suite.Tests {
		test := test // capture loop variable

		t.Run(test.Name, func(t *testing.T) {
			if suite.Parallel && !suite.StopOnError {
				t.Parallel()
			}

			runIntegrationTest(ctx, t, test, suite.RetryCount, suite.RetryDelay)
		})

		// Stop on first error if configured
		if suite.StopOnError && t.Failed() {
			logger.WarnContext(ctx, "Stopping test suite due to error", "suite", suite.Name)
			break
		}
	}
}

// runIntegrationTest executes a single integration test with retry logic
func runIntegrationTest(ctx context.Context, t *testing.T, test IntegrationTest, retryCount int, retryDelay time.Duration) {
	logger := observability.GetLogger(ctx)

	// Apply timeout if specified
	if test.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, test.Timeout)
		defer cancel()
	}

	var lastErr error
	attempts := retryCount + 1

	for attempt := 1; attempt <= attempts; attempt++ {
		// Setup
		if test.Setup != nil {
			logger.InfoContext(ctx, "Running test setup",
				"test", test.Name,
				"attempt", attempt,
				"max_attempts", attempts)

			if err := test.Setup(ctx, t); err != nil {
				lastErr = gerror.Wrap(err, gerror.ErrCodeInternal, "test setup failed").
					WithComponent("integration_test").
					WithDetails("test", test.Name).
					WithDetails("attempt", attempt)

				if attempt < attempts {
					logger.WarnContext(ctx, "Setup failed, retrying",
						"test", test.Name,
						"attempt", attempt,
						"error", err)
					time.Sleep(retryDelay)
					continue
				}
				t.Fatalf("Setup failed after %d attempts: %v", attempts, lastErr)
			}
		}

		// Ensure teardown runs
		if test.Teardown != nil {
			defer func() {
				logger.InfoContext(ctx, "Running test teardown", "test", test.Name)
				if err := test.Teardown(ctx, t); err != nil {
					t.Errorf("Teardown failed: %v", err)
				}
			}()
		}

		// Run test
		logger.InfoContext(ctx, "Running test",
			"test", test.Name,
			"description", test.Description)

		if err := test.Test(ctx, t); err != nil {
			lastErr = gerror.Wrap(err, gerror.ErrCodeInternal, "test failed").
				WithComponent("integration_test").
				WithDetails("test", test.Name).
				WithDetails("attempt", attempt)

			if attempt < attempts {
				logger.WarnContext(ctx, "Test failed, retrying",
					"test", test.Name,
					"attempt", attempt,
					"error", err)
				time.Sleep(retryDelay)
				continue
			}
			t.Errorf("Test failed after %d attempts: %v", attempts, lastErr)
		} else {
			// Test passed
			logger.InfoContext(ctx, "Test passed",
				"test", test.Name,
				"attempt", attempt)
			return
		}
	}
}

// AssertEventually asserts that a condition becomes true within a timeout
func AssertEventually(t *testing.T, condition func() bool, timeout time.Duration, msg string) {
	deadline := time.Now().Add(timeout)
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	for {
		if condition() {
			return
		}

		select {
		case <-ticker.C:
			if time.Now().After(deadline) {
				t.Errorf("Condition not met within timeout: %s", msg)
				return
			}
		}
	}
}

// AssertNoError fails the test if err is not nil
func AssertNoError(t *testing.T, err error, msg string) {
	if err != nil {
		t.Errorf("%s: %v", msg, err)
	}
}

// AssertError fails the test if err is nil
func AssertError(t *testing.T, err error, msg string) {
	if err == nil {
		t.Errorf("%s: expected error but got nil", msg)
	}
}

// AssertEqual fails the test if expected != actual
func AssertEqual(t *testing.T, expected, actual interface{}, msg string) {
	if expected != actual {
		t.Errorf("%s: expected %v, got %v", msg, expected, actual)
	}
}
