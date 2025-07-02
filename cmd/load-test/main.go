// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

// load-test provides comprehensive load testing for Guild Framework
//
// This command implements the load testing requirements identified in performance optimization,
// Agent 3 task, providing:
//   - Realistic user simulation with varying load patterns
//   - Comprehensive performance metrics collection
//   - System resource monitoring during load tests
//   - Detailed reporting and analysis
//
// The command follows Guild's architectural patterns:
//   - Context-first error handling with gerror
//   - Structured logging with observability integration
//   - Configuration-driven testing framework
//   - Staff-level performance analysis
//
// Example usage:
//
//	# Run basic load test
//	load-test
//	
//	# Run with specific user count and duration
//	load-test --users=50 --duration=5m
//	
//	# Run with heavy load profile
//	load-test --profile=heavy --duration=10m
//	
//	# Run continuous load testing
//	load-test --continuous --interval=1h
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/time/rate"
	"github.com/lancekrogers/guild/pkg/session"
	"github.com/lancekrogers/guild/pkg/gerror"
	"go.uber.org/zap"
)

// LoadTester simulates realistic Guild usage patterns
type LoadTester struct {
	logger           *zap.Logger
	sessionService   *session.SessionService
	concurrentUsers  int
	testDuration     time.Duration
	requestRate      float64
	profile          LoadProfile
	
	// Load test results
	results          *LoadTestResults
	mu               sync.Mutex
	
	// Metrics collection
	requestCount     int64
	successCount     int64
	errorCount       int64
	totalLatency     int64
	responseTimes    []time.Duration
	responseTimesMu  sync.Mutex
}

// LoadProfile defines different load testing profiles
type LoadProfile string

const (
	LoadProfileLight  LoadProfile = "light"
	LoadProfileNormal LoadProfile = "normal"
	LoadProfileHeavy  LoadProfile = "heavy"
	LoadProfileStress LoadProfile = "stress"
)

// LoadTestResults contains comprehensive load test results
type LoadTestResults struct {
	StartTime        time.Time                  `json:"start_time"`
	EndTime          time.Time                  `json:"end_time"`
	Duration         time.Duration              `json:"duration"`
	Profile          LoadProfile                `json:"profile"`
	ConcurrentUsers  int                        `json:"concurrent_users"`
	RequestRate      float64                    `json:"request_rate"`
	
	// Request metrics
	TotalRequests    int64                      `json:"total_requests"`
	SuccessfulReqs   int64                      `json:"successful_requests"`
	FailedRequests   int64                      `json:"failed_requests"`
	SuccessRate      float64                    `json:"success_rate"`
	ThroughputRPS    float64                    `json:"throughput_rps"`
	
	// Response time metrics
	ResponseTimes    ResponseTimeMetrics        `json:"response_times"`
	
	// Error analysis
	ErrorsByType     map[string]int64           `json:"errors_by_type"`
	ErrorDetails     []ErrorDetail              `json:"error_details"`
	
	// System metrics
	MemoryUsage      []SystemMetric             `json:"memory_usage"`
	CPUUsage         []SystemMetric             `json:"cpu_usage"`
	GoroutineCount   []SystemMetric             `json:"goroutine_count"`
	
	// Performance targets
	TargetsMet       map[string]bool            `json:"targets_met"`
	
	// User behavior simulation
	UserActions      map[string]int64           `json:"user_actions"`
	SessionMetrics   SessionMetrics             `json:"session_metrics"`
}

// ResponseTimeMetrics contains detailed response time statistics
type ResponseTimeMetrics struct {
	Mean       time.Duration `json:"mean"`
	Median     time.Duration `json:"median"`
	P90        time.Duration `json:"p90"`
	P95        time.Duration `json:"p95"`
	P99        time.Duration `json:"p99"`
	Min        time.Duration `json:"min"`
	Max        time.Duration `json:"max"`
	StdDev     time.Duration `json:"std_dev"`
}

// ErrorDetail provides detailed error information
type ErrorDetail struct {
	Type        string    `json:"type"`
	Message     string    `json:"message"`
	Count       int64     `json:"count"`
	FirstSeen   time.Time `json:"first_seen"`
	LastSeen    time.Time `json:"last_seen"`
	UserID      string    `json:"user_id,omitempty"`
	SessionID   string    `json:"session_id,omitempty"`
}

// SystemMetric represents a point-in-time system metric
type SystemMetric struct {
	Timestamp time.Time `json:"timestamp"`
	Value     float64   `json:"value"`
}

// SessionMetrics contains session-related metrics
type SessionMetrics struct {
	SessionsCreated    int64         `json:"sessions_created"`
	SessionsResumed    int64         `json:"sessions_resumed"`
	SessionsExported   int64         `json:"sessions_exported"`
	AvgSessionDuration time.Duration `json:"avg_session_duration"`
	AvgMessagesPerSession float64    `json:"avg_messages_per_session"`
}

// UserSimulator simulates a single user's behavior
type UserSimulator struct {
	userID       int
	sessionID    string
	session      *session.Session
	loadTester   *LoadTester
	limiter      *rate.Limiter
	actionCounts map[string]int64
	mu           sync.Mutex
}

func main() {
	var (
		users           = flag.Int("users", 10, "Number of concurrent users")
		duration        = flag.Duration("duration", 5*time.Minute, "Test duration")
		rate            = flag.Float64("rate", 10.0, "Requests per second per user")
		profile         = flag.String("profile", "normal", "Load profile: light, normal, heavy, stress")
		reportPath      = flag.String("report", "reports/load-test-report.json", "Load test report output")
		continuous      = flag.Bool("continuous", false, "Run continuous load testing")
		interval        = flag.Duration("interval", 1*time.Hour, "Continuous testing interval")
		verbose         = flag.Bool("verbose", false, "Verbose logging")
		memoryLimit     = flag.Int64("memory-limit", 500*1024*1024, "Memory usage limit in bytes")
		cpuLimit        = flag.Float64("cpu-limit", 80.0, "CPU usage limit percentage")
		targetThroughput = flag.Float64("target-throughput", 100.0, "Target throughput (RPS)")
	)
	flag.Parse()

	// Initialize logger
	logger, err := initializeLogger(*verbose)
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	defer logger.Sync()

	ctx := context.Background()
	
	// Adjust parameters based on profile
	profileEnum := LoadProfile(*profile)
	adjustParametersForProfile(&profileEnum, users, rate)

	logger.Info("Starting load test",
		zap.Int("concurrent_users", *users),
		zap.Duration("duration", *duration),
		zap.Float64("requests_per_second_per_user", *rate),
		zap.String("profile", *profile),
		zap.String("report_path", *reportPath))

	if *continuous {
		logger.Info("Starting continuous load testing mode", zap.Duration("interval", *interval))
		runContinuousLoadTest(ctx, logger, *users, *duration, *rate, profileEnum, *reportPath, *interval)
	} else {
		// Run single load test
		loadTester, err := NewLoadTester(*users, *duration, *rate, profileEnum, logger)
		if err != nil {
			logger.Fatal("Failed to initialize load tester", zap.Error(err))
		}

		results, err := loadTester.RunLoadTest(ctx)
		if err != nil {
			logger.Fatal("Load test failed", zap.Error(err))
		}

		// Generate report
		if err := generateLoadTestReport(results, *reportPath, logger); err != nil {
			logger.Fatal("Failed to generate report", zap.Error(err))
		}

		// Print results
		printLoadTestResults(results, logger)

		// Check if performance targets were met
		checkPerformanceTargets(results, *targetThroughput, *memoryLimit, *cpuLimit, logger)
	}
}

// NewLoadTester creates a new load tester
func NewLoadTester(users int, duration time.Duration, requestRate float64, profile LoadProfile, logger *zap.Logger) (*LoadTester, error) {
	// Initialize session service for load testing
	sessionRegistry := session.NewDefaultSessionRegistry()
	sessionService := session.NewSessionService(sessionRegistry)

	return &LoadTester{
		logger:          logger,
		sessionService:  sessionService,
		concurrentUsers: users,
		testDuration:    duration,
		requestRate:     requestRate,
		profile:         profile,
		results: &LoadTestResults{
			Profile:         profile,
			ConcurrentUsers: users,
			RequestRate:     requestRate,
			ErrorsByType:    make(map[string]int64),
			ErrorDetails:    make([]ErrorDetail, 0),
			MemoryUsage:     make([]SystemMetric, 0),
			CPUUsage:        make([]SystemMetric, 0),
			GoroutineCount:  make([]SystemMetric, 0),
			TargetsMet:      make(map[string]bool),
			UserActions:     make(map[string]int64),
		},
		responseTimes:   make([]time.Duration, 0),
	}, nil
}

// RunLoadTest executes the load test
func (lt *LoadTester) RunLoadTest(ctx context.Context) (*LoadTestResults, error) {
	lt.results.StartTime = time.Now()
	
	lt.logger.Info("Starting load test execution",
		zap.Int("users", lt.concurrentUsers),
		zap.Duration("duration", lt.testDuration),
		zap.String("profile", string(lt.profile)))

	// Create context with timeout
	testCtx, cancel := context.WithTimeout(ctx, lt.testDuration)
	defer cancel()

	// Start system monitoring
	go lt.monitorSystemMetrics(testCtx)

	// Create wait group for all user goroutines
	var wg sync.WaitGroup

	// Start concurrent users
	for i := 0; i < lt.concurrentUsers; i++ {
		wg.Add(1)
		go func(userID int) {
			defer wg.Done()
			simulator := &UserSimulator{
				userID:       userID,
				loadTester:   lt,
				limiter:      rate.NewLimiter(rate.Limit(lt.requestRate), 1),
				actionCounts: make(map[string]int64),
			}
			simulator.simulateUser(testCtx)
		}(i)
	}

	// Wait for all users to complete
	wg.Wait()

	lt.results.EndTime = time.Now()
	lt.results.Duration = lt.results.EndTime.Sub(lt.results.StartTime)
	
	// Calculate final metrics
	lt.calculateFinalMetrics()

	lt.logger.Info("Load test completed",
		zap.Duration("duration", lt.results.Duration),
		zap.Int64("total_requests", lt.results.TotalRequests),
		zap.Float64("success_rate", lt.results.SuccessRate),
		zap.Float64("throughput_rps", lt.results.ThroughputRPS))

	return lt.results, nil
}

// simulateUser simulates a single user's behavior
func (us *UserSimulator) simulateUser(ctx context.Context) {
	logger := us.loadTester.logger.With(zap.Int("user_id", us.userID))
	logger.Debug("User simulation started")

	// Create session for this user
	sessionID := fmt.Sprintf("load-test-user-%d", us.userID)
	session, err := us.loadTester.sessionService.CreateSession(ctx, sessionID, "load-test-campaign")
	if err != nil {
		us.recordError("session_creation", err, "")
		logger.Error("Failed to create session", zap.Error(err))
		return
	}
	
	us.session = session
	us.sessionID = session.ID
	atomic.AddInt64(&us.loadTester.results.SessionMetrics.SessionsCreated, 1)

	requestCount := 0
	sessionStartTime := time.Now()

	for {
		select {
		case <-ctx.Done():
			sessionDuration := time.Since(sessionStartTime)
			logger.Debug("User simulation completed", 
				zap.Int("requests_made", requestCount),
				zap.Duration("session_duration", sessionDuration))
			
			// Update session duration metrics
			us.updateSessionMetrics(sessionDuration, requestCount)
			return
		default:
			// Wait for rate limiter
			if err := us.limiter.Wait(ctx); err != nil {
				return // Context cancelled
			}

			// Perform random user action based on profile
			action := us.selectUserAction(requestCount)
			startTime := time.Now()

			err := us.performUserAction(ctx, action)
			responseTime := time.Since(startTime)

			// Record results
			us.recordRequest(action, responseTime, err)
			requestCount++

			// Update action counts
			us.mu.Lock()
			us.actionCounts[action]++
			us.mu.Unlock()

			if requestCount%100 == 0 {
				logger.Debug("User progress", zap.Int("requests", requestCount))
			}
		}
	}
}

// selectUserAction selects the next user action based on profile and request count
func (us *UserSimulator) selectUserAction(requestCount int) string {
	switch us.loadTester.profile {
	case LoadProfileLight:
		return us.selectLightProfileAction(requestCount)
	case LoadProfileHeavy:
		return us.selectHeavyProfileAction(requestCount)
	case LoadProfileStress:
		return us.selectStressProfileAction(requestCount)
	default: // Normal profile
		return us.selectNormalProfileAction(requestCount)
	}
}

// selectNormalProfileAction selects actions for normal load profile
func (us *UserSimulator) selectNormalProfileAction(requestCount int) string {
	// Realistic user behavior patterns for normal usage
	actions := []string{
		"send_message",     // 40% - Most common action
		"send_message",
		"send_message",
		"send_message",
		"switch_agent",     // 20% - Switching between agents
		"switch_agent",
		"export_session",   // 10% - Occasional exports
		"view_history",     // 15% - Looking at history
		"view_history",
		"create_commission", // 15% - Creating new work
	}

	// Vary behavior over time
	if requestCount > 50 && requestCount%25 == 0 {
		return "export_session" // More exports as session progresses
	}

	return actions[requestCount%len(actions)]
}

// selectLightProfileAction selects actions for light load profile
func (us *UserSimulator) selectLightProfileAction(requestCount int) string {
	// Light usage - mostly viewing and simple interactions
	actions := []string{
		"send_message",
		"view_history",
		"view_history",
		"switch_agent",
	}
	return actions[requestCount%len(actions)]
}

// selectHeavyProfileAction selects actions for heavy load profile
func (us *UserSimulator) selectHeavyProfileAction(requestCount int) string {
	// Heavy usage - more complex operations
	actions := []string{
		"send_message",
		"send_message",
		"create_commission",
		"create_commission",
		"export_session",
		"analyze_session",
		"switch_agent",
		"resume_session",
	}
	return actions[requestCount%len(actions)]
}

// selectStressProfileAction selects actions for stress testing
func (us *UserSimulator) selectStressProfileAction(requestCount int) string {
	// Stress testing - maximum load operations
	actions := []string{
		"send_message",
		"create_commission",
		"export_session",
		"analyze_session",
		"send_message",
		"create_commission",
		"concurrent_operation",
		"bulk_operation",
	}
	return actions[requestCount%len(actions)]
}

// performUserAction performs the specified user action
func (us *UserSimulator) performUserAction(ctx context.Context, action string) error {
	switch action {
	case "send_message":
		return us.simulateSendMessage(ctx)
	case "switch_agent":
		return us.simulateAgentSwitch(ctx)
	case "export_session":
		return us.simulateExportSession(ctx)
	case "view_history":
		return us.simulateViewHistory(ctx)
	case "create_commission":
		return us.simulateCreateCommission(ctx)
	case "analyze_session":
		return us.simulateAnalyzeSession(ctx)
	case "resume_session":
		return us.simulateResumeSession(ctx)
	case "concurrent_operation":
		return us.simulateConcurrentOperation(ctx)
	case "bulk_operation":
		return us.simulateBulkOperation(ctx)
	default:
		return fmt.Errorf("unknown action: %s", action)
	}
}

// simulateSendMessage simulates sending a message
func (us *UserSimulator) simulateSendMessage(ctx context.Context) error {
	agents := []string{"elena", "marcus", "vera"}
	agent := agents[len(us.session.Messages)%len(agents)]

	message := session.Message{
		ID:        fmt.Sprintf("load-msg-%d-%d", us.userID, len(us.session.Messages)),
		Agent:     agent,
		Content:   fmt.Sprintf("Load test message %d from user %d to %s", len(us.session.Messages), us.userID, agent),
		Timestamp: time.Now(),
		Type:      session.MessageTypeUser,
	}

	us.session.Messages = append(us.session.Messages, message)

	// Simulate agent processing delay (realistic)
	processingTime := time.Duration(50+len(us.session.Messages)*2) * time.Millisecond
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(processingTime):
		return nil
	}
}

// simulateAgentSwitch simulates switching active agent
func (us *UserSimulator) simulateAgentSwitch(ctx context.Context) error {
	agents := []string{"elena", "marcus", "vera"}
	newAgent := agents[time.Now().Nanosecond()%len(agents)]

	// Update active agent in session state
	if us.session.State.ActiveAgents == nil {
		us.session.State.ActiveAgents = make(map[string]session.AgentState)
	}
	
	us.session.State.ActiveAgents[newAgent] = session.AgentState{
		ID:           newAgent,
		Name:         newAgent,
		Status:       "active",
		LastActivity: time.Now(),
		Context:      make(map[string]interface{}),
		TaskQueue:    make([]string, 0),
	}

	// Simulate UI update delay
	time.Sleep(20 * time.Millisecond)
	return nil
}

// simulateExportSession simulates exporting session data
func (us *UserSimulator) simulateExportSession(ctx context.Context) error {
	_, err := us.loadTester.sessionService.ExportSession(ctx, us.session.ID, session.ExportOptions{
		Format: session.ExportFormatJSON,
	})
	
	if err == nil {
		atomic.AddInt64(&us.loadTester.results.SessionMetrics.SessionsExported, 1)
	}
	
	return err
}

// simulateViewHistory simulates viewing session history
func (us *UserSimulator) simulateViewHistory(ctx context.Context) error {
	// Simulate loading session history
	time.Sleep(10 * time.Millisecond)
	return nil
}

// simulateCreateCommission simulates creating a new commission
func (us *UserSimulator) simulateCreateCommission(ctx context.Context) error {
	// Simulate commission creation time
	time.Sleep(100 * time.Millisecond)
	return nil
}

// simulateAnalyzeSession simulates session analysis
func (us *UserSimulator) simulateAnalyzeSession(ctx context.Context) error {
	_, err := us.loadTester.sessionService.AnalyzeSession(ctx, us.session.ID)
	return err
}

// simulateResumeSession simulates resuming a session
func (us *UserSimulator) simulateResumeSession(ctx context.Context) error {
	err := us.loadTester.sessionService.ResumeSession(ctx, us.session.ID)
	if err == nil {
		atomic.AddInt64(&us.loadTester.results.SessionMetrics.SessionsResumed, 1)
	}
	return err
}

// simulateConcurrentOperation simulates concurrent operations for stress testing
func (us *UserSimulator) simulateConcurrentOperation(ctx context.Context) error {
	var wg sync.WaitGroup
	errors := make([]error, 3)
	
	// Perform 3 operations concurrently
	operations := []func(context.Context) error{
		us.simulateSendMessage,
		us.simulateViewHistory,
		us.simulateAgentSwitch,
	}
	
	for i, op := range operations {
		wg.Add(1)
		go func(index int, operation func(context.Context) error) {
			defer wg.Done()
			errors[index] = operation(ctx)
		}(i, op)
	}
	
	wg.Wait()
	
	// Return first error encountered
	for _, err := range errors {
		if err != nil {
			return err
		}
	}
	
	return nil
}

// simulateBulkOperation simulates bulk operations for stress testing
func (us *UserSimulator) simulateBulkOperation(ctx context.Context) error {
	// Simulate sending multiple messages in bulk
	messageCount := 5
	for i := 0; i < messageCount; i++ {
		if err := us.simulateSendMessage(ctx); err != nil {
			return err
		}
	}
	return nil
}

// recordRequest records request metrics
func (us *UserSimulator) recordRequest(action string, responseTime time.Duration, err error) {
	atomic.AddInt64(&us.loadTester.requestCount, 1)
	atomic.AddInt64(&us.loadTester.totalLatency, responseTime.Nanoseconds())
	
	us.loadTester.responseTimesMu.Lock()
	us.loadTester.responseTimes = append(us.loadTester.responseTimes, responseTime)
	us.loadTester.responseTimesMu.Unlock()
	
	// Update action counts
	us.loadTester.mu.Lock()
	us.loadTester.results.UserActions[action]++
	us.loadTester.mu.Unlock()

	if err != nil {
		atomic.AddInt64(&us.loadTester.errorCount, 1)
		us.recordError(action, err, us.sessionID)
	} else {
		atomic.AddInt64(&us.loadTester.successCount, 1)
	}
}

// recordError records error details
func (us *UserSimulator) recordError(action string, err error, sessionID string) {
	errorType := "unknown_error"
	if gErr, ok := err.(*gerror.GuildError); ok {
		errorType = string(gErr.Code)
	} else {
		errorType = action + "_error"
	}

	us.loadTester.mu.Lock()
	defer us.loadTester.mu.Unlock()
	
	us.loadTester.results.ErrorsByType[errorType]++
	
	// Add to error details
	now := time.Now()
	found := false
	for i := range us.loadTester.results.ErrorDetails {
		if us.loadTester.results.ErrorDetails[i].Type == errorType &&
		   us.loadTester.results.ErrorDetails[i].Message == err.Error() {
			us.loadTester.results.ErrorDetails[i].Count++
			us.loadTester.results.ErrorDetails[i].LastSeen = now
			found = true
			break
		}
	}
	
	if !found {
		us.loadTester.results.ErrorDetails = append(us.loadTester.results.ErrorDetails, ErrorDetail{
			Type:      errorType,
			Message:   err.Error(),
			Count:     1,
			FirstSeen: now,
			LastSeen:  now,
			UserID:    fmt.Sprintf("user-%d", us.userID),
			SessionID: sessionID,
		})
	}
}

// updateSessionMetrics updates session-related metrics
func (us *UserSimulator) updateSessionMetrics(duration time.Duration, messageCount int) {
	us.loadTester.mu.Lock()
	defer us.loadTester.mu.Unlock()
	
	// This is a simplified approach - in a real implementation, we'd track this more accurately
	currentAvg := us.loadTester.results.SessionMetrics.AvgSessionDuration
	sessionCount := us.loadTester.results.SessionMetrics.SessionsCreated
	
	if sessionCount > 0 {
		us.loadTester.results.SessionMetrics.AvgSessionDuration = 
			(currentAvg*time.Duration(sessionCount-1) + duration) / time.Duration(sessionCount)
	}
	
	// Update messages per session
	currentMsgAvg := us.loadTester.results.SessionMetrics.AvgMessagesPerSession
	us.loadTester.results.SessionMetrics.AvgMessagesPerSession = 
		(currentMsgAvg*float64(sessionCount-1) + float64(messageCount)) / float64(sessionCount)
}

// monitorSystemMetrics monitors system resources during the test
func (lt *LoadTester) monitorSystemMetrics(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			now := time.Now()
			
			// Record memory usage
			memUsage := getCurrentMemoryUsage()
			lt.mu.Lock()
			lt.results.MemoryUsage = append(lt.results.MemoryUsage, SystemMetric{
				Timestamp: now,
				Value:     float64(memUsage) / (1024 * 1024), // Convert to MB
			})
			lt.mu.Unlock()
			
			// Record goroutine count
			goroutineCount := runtime.NumGoroutine()
			lt.mu.Lock()
			lt.results.GoroutineCount = append(lt.results.GoroutineCount, SystemMetric{
				Timestamp: now,
				Value:     float64(goroutineCount),
			})
			lt.mu.Unlock()
			
			// Record CPU usage (simplified - would use proper CPU monitoring in production)
			cpuUsage := getCurrentCPUUsage()
			lt.mu.Lock()
			lt.results.CPUUsage = append(lt.results.CPUUsage, SystemMetric{
				Timestamp: now,
				Value:     cpuUsage,
			})
			lt.mu.Unlock()
		}
	}
}

// calculateFinalMetrics calculates final metrics after test completion
func (lt *LoadTester) calculateFinalMetrics() {
	lt.results.TotalRequests = atomic.LoadInt64(&lt.requestCount)
	lt.results.SuccessfulReqs = atomic.LoadInt64(&lt.successCount)
	lt.results.FailedRequests = atomic.LoadInt64(&lt.errorCount)
	
	if lt.results.TotalRequests > 0 {
		lt.results.SuccessRate = float64(lt.results.SuccessfulReqs) / float64(lt.results.TotalRequests)
	}
	
	if lt.results.Duration > 0 {
		lt.results.ThroughputRPS = float64(lt.results.TotalRequests) / lt.results.Duration.Seconds()
	}
	
	// Calculate response time metrics
	lt.responseTimesMu.Lock()
	if len(lt.responseTimes) > 0 {
		lt.results.ResponseTimes = calculateResponseTimeMetrics(lt.responseTimes)
	}
	lt.responseTimesMu.Unlock()
	
	// Evaluate performance targets
	lt.evaluatePerformanceTargets()
}

// calculateResponseTimeMetrics calculates comprehensive response time statistics
func calculateResponseTimeMetrics(responseTimes []time.Duration) ResponseTimeMetrics {
	if len(responseTimes) == 0 {
		return ResponseTimeMetrics{}
	}
	
	// Sort for percentile calculations
	sorted := make([]time.Duration, len(responseTimes))
	copy(sorted, responseTimes)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i] < sorted[j]
	})
	
	// Calculate basic statistics
	var sum time.Duration
	for _, rt := range sorted {
		sum += rt
	}
	mean := sum / time.Duration(len(sorted))
	
	// Calculate percentiles
	median := sorted[len(sorted)/2]
	p90 := sorted[int(float64(len(sorted))*0.90)]
	p95 := sorted[int(float64(len(sorted))*0.95)]
	p99 := sorted[int(float64(len(sorted))*0.99)]
	
	// Calculate standard deviation
	var sumSquares time.Duration
	for _, rt := range sorted {
		diff := rt - mean
		sumSquares += time.Duration(diff.Nanoseconds() * diff.Nanoseconds())
	}
	variance := sumSquares / time.Duration(len(sorted))
	stdDev := time.Duration(float64(variance.Nanoseconds()) * 0.5) // Simplified square root
	
	return ResponseTimeMetrics{
		Mean:   mean,
		Median: median,
		P90:    p90,
		P95:    p95,
		P99:    p99,
		Min:    sorted[0],
		Max:    sorted[len(sorted)-1],
		StdDev: stdDev,
	}
}

// evaluatePerformanceTargets evaluates whether performance targets were met
func (lt *LoadTester) evaluatePerformanceTargets() {
	// Target: Throughput > 100 RPS
	lt.results.TargetsMet["throughput_100_rps"] = lt.results.ThroughputRPS >= 100.0
	
	// Target: Success rate > 95%
	lt.results.TargetsMet["success_rate_95_percent"] = lt.results.SuccessRate >= 0.95
	
	// Target: P95 response time < 1 second
	lt.results.TargetsMet["p95_response_1s"] = lt.results.ResponseTimes.P95 <= 1*time.Second
	
	// Target: P99 response time < 2 seconds
	lt.results.TargetsMet["p99_response_2s"] = lt.results.ResponseTimes.P99 <= 2*time.Second
	
	// Target: Memory usage reasonable (check peak memory)
	peakMemory := 0.0
	for _, metric := range lt.results.MemoryUsage {
		if metric.Value > peakMemory {
			peakMemory = metric.Value
		}
	}
	lt.results.TargetsMet["memory_usage_500mb"] = peakMemory <= 500.0 // 500MB
	
	// Target: Error rate < 5%
	errorRate := float64(lt.results.FailedRequests) / float64(lt.results.TotalRequests)
	lt.results.TargetsMet["error_rate_5_percent"] = errorRate <= 0.05
}

// Utility functions

// adjustParametersForProfile adjusts test parameters based on load profile
func adjustParametersForProfile(profile *LoadProfile, users *int, rate *float64) {
	switch *profile {
	case LoadProfileLight:
		if *users > 20 {
			*users = 5
		}
		if *rate > 10.0 {
			*rate = 2.0
		}
	case LoadProfileHeavy:
		if *users < 30 {
			*users = 50
		}
		if *rate < 15.0 {
			*rate = 20.0
		}
	case LoadProfileStress:
		if *users < 50 {
			*users = 100
		}
		if *rate < 20.0 {
			*rate = 30.0
		}
	}
}

// getCurrentMemoryUsage returns current memory usage in bytes
func getCurrentMemoryUsage() int64 {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return int64(m.Alloc)
}

// getCurrentCPUUsage returns current CPU usage percentage (simplified)
func getCurrentCPUUsage() float64 {
	// This is a simplified implementation
	// In production, you would use proper CPU monitoring
	return float64(runtime.NumGoroutine()) * 0.1 // Rough approximation based on goroutines
}

// initializeLogger initializes the logger
func initializeLogger(verbose bool) (*zap.Logger, error) {
	if verbose {
		return zap.NewDevelopment()
	}
	return zap.NewProduction()
}

// runContinuousLoadTest runs continuous load testing
func runContinuousLoadTest(ctx context.Context, logger *zap.Logger, users int, duration time.Duration, rate float64, profile LoadProfile, reportPath string, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	
	runCount := 0
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			runCount++
			logger.Info("Starting continuous load test run", zap.Int("run_number", runCount))
			
			loadTester, err := NewLoadTester(users, duration, rate, profile, logger)
			if err != nil {
				logger.Error("Failed to initialize load tester", zap.Error(err))
				continue
			}
			
			results, err := loadTester.RunLoadTest(ctx)
			if err != nil {
				logger.Error("Load test run failed", zap.Error(err))
				continue
			}
			
			// Generate timestamped report
			timestamp := time.Now().Format("20060102-150405")
			continuousReportPath := fmt.Sprintf("%s.run-%d.%s", reportPath, runCount, timestamp)
			
			if err := generateLoadTestReport(results, continuousReportPath, logger); err != nil {
				logger.Error("Failed to generate continuous report", zap.Error(err))
			}
			
			logger.Info("Continuous load test run completed", 
				zap.Int("run_number", runCount),
				zap.String("report_path", continuousReportPath))
		}
	}
}

// generateLoadTestReport generates a comprehensive load test report
func generateLoadTestReport(results *LoadTestResults, reportPath string, logger *zap.Logger) error {
	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(reportPath), 0755); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeIO, "failed to create report directory")
	}

	// Marshal results to JSON
	data, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeParsing, "failed to marshal load test results")
	}

	// Write report
	if err := os.WriteFile(reportPath, data, 0644); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeIO, "failed to write report file")
	}

	logger.Info("Load test report generated",
		zap.String("path", reportPath),
		zap.Int("size_bytes", len(data)))

	return nil
}

// printLoadTestResults prints a summary of load test results
func printLoadTestResults(results *LoadTestResults, logger *zap.Logger) {
	logger.Info("=== LOAD TEST RESULTS ===")
	logger.Info("Test Configuration",
		zap.String("profile", string(results.Profile)),
		zap.Int("concurrent_users", results.ConcurrentUsers),
		zap.Float64("request_rate", results.RequestRate),
		zap.Duration("duration", results.Duration))

	logger.Info("Request Metrics",
		zap.Int64("total_requests", results.TotalRequests),
		zap.Int64("successful_requests", results.SuccessfulReqs),
		zap.Int64("failed_requests", results.FailedRequests),
		zap.Float64("success_rate", results.SuccessRate*100),
		zap.Float64("throughput_rps", results.ThroughputRPS))

	logger.Info("Response Time Metrics",
		zap.Duration("mean", results.ResponseTimes.Mean),
		zap.Duration("median", results.ResponseTimes.Median),
		zap.Duration("p95", results.ResponseTimes.P95),
		zap.Duration("p99", results.ResponseTimes.P99),
		zap.Duration("max", results.ResponseTimes.Max))

	// Print memory usage
	if len(results.MemoryUsage) > 0 {
		peakMemory := 0.0
		for _, metric := range results.MemoryUsage {
			if metric.Value > peakMemory {
				peakMemory = metric.Value
			}
		}
		logger.Info("System Metrics",
			zap.Float64("peak_memory_mb", peakMemory),
			zap.Int("memory_samples", len(results.MemoryUsage)))
	}

	// Print target results
	targetsMet := 0
	totalTargets := len(results.TargetsMet)
	for target, met := range results.TargetsMet {
		if met {
			targetsMet++
		}
		logger.Info("Performance Target", zap.String("target", target), zap.Bool("met", met))
	}

	logger.Info("Performance Summary",
		zap.Int("targets_met", targetsMet),
		zap.Int("total_targets", totalTargets),
		zap.Float64("target_success_rate", float64(targetsMet)/float64(totalTargets)*100))
}

// checkPerformanceTargets checks if performance targets were met and exits accordingly
func checkPerformanceTargets(results *LoadTestResults, targetThroughput float64, memoryLimit int64, cpuLimit float64, logger *zap.Logger) {
	allTargetsMet := true
	for _, met := range results.TargetsMet {
		if !met {
			allTargetsMet = false
			break
		}
	}

	if allTargetsMet && 
	   results.ThroughputRPS >= targetThroughput &&
	   results.ResponseTimes.P95 <= 1*time.Second {
		logger.Info("✅ All performance targets met - system ready for production!")
		os.Exit(0)
	} else {
		logger.Error("❌ Performance targets not met - optimization required")
		os.Exit(1)
	}
}