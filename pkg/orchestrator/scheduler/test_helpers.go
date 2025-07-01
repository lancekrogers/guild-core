// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package scheduler

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/lancekrogers/guild/pkg/kanban"
)

// Helper function to convert capabilities
func convertCapabilities(caps []AgentCapability) []string {
	result := make([]string, len(caps))
	for i, cap := range caps {
		result[i] = string(cap)
	}
	return result
}

// Helper function to map priority
func mapToKanbanPriority(priority int) kanban.TaskPriority {
	if priority >= 80 {
		return kanban.PriorityHigh
	} else if priority >= 40 {
		return kanban.PriorityMedium
	}
	return kanban.PriorityLow
}

// mockAgentExecutor implements interfaces.AgentExecutor for testing
type mockAgentExecutor struct {
	agentID      string
	capabilities []string
	isAvailable  bool
	executeFunc  func(ctx context.Context, taskID string, payload interface{}) (interface{}, error)
	mu           sync.Mutex
}

func (m *mockAgentExecutor) GetAgentID() string {
	return m.agentID
}

func (m *mockAgentExecutor) GetCapabilities() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]string{}, m.capabilities...)
}

func (m *mockAgentExecutor) IsAvailable() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.isAvailable
}

func (m *mockAgentExecutor) Execute(ctx context.Context, taskID string, payload interface{}) (interface{}, error) {
	if m.executeFunc != nil {
		return m.executeFunc(ctx, taskID, payload)
	}
	// Default implementation
	select {
	case <-time.After(10 * time.Millisecond):
		return "completed", nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}


// TestHarness provides utilities for testing the scheduler
type TestHarness struct {
	Scheduler    *TaskScheduler
	ManagerAgent *MockManagerAgentClient
	KanbanClient *MockKanbanClient
	Agents       map[string]*mockAgentExecutor
	ctx          context.Context
	cancel       context.CancelFunc
	mu           sync.RWMutex
}

// NewTestHarness creates a new test harness
func NewTestHarness(t interface{ Helper() }, config *SchedulerConfig) *TestHarness {
	t.Helper()
	
	ctx, cancel := context.WithCancel(context.Background())
	
	managerAgent := NewMockManagerAgentClient()
	kanbanClient := NewMockKanbanClient()
	
	if config == nil {
		config = &SchedulerConfig{
			MaxConcurrentTasks: 5,
			ScheduleInterval:   10 * time.Millisecond,
			DefaultTimeout:     30 * time.Second,
			EnableMetrics:      true,
		}
	}
	
	scheduler, err := NewTaskScheduler(ctx, config, managerAgent, kanbanClient)
	if err != nil {
		panic(err)
	}
	
	return &TestHarness{
		Scheduler:    scheduler,
		ManagerAgent: managerAgent,
		KanbanClient: kanbanClient,
		Agents:       make(map[string]*mockAgentExecutor),
		ctx:          ctx,
		cancel:       cancel,
	}
}

// RegisterAgent adds an agent to the test harness
func (th *TestHarness) RegisterAgent(agentID string, capabilities []AgentCapability, executeFunc func(context.Context, string, interface{}) (interface{}, error)) error {
	th.mu.Lock()
	defer th.mu.Unlock()
	
	executor := &mockAgentExecutor{
		agentID:      agentID,
		capabilities: convertCapabilities(capabilities),
		isAvailable:  true,
		executeFunc:  executeFunc,
	}
	
	th.Agents[agentID] = executor
	
	return th.Scheduler.RegisterAgent(th.ctx, agentID, executor, capabilities)
}

// SubmitTask submits a task with pre-configured assignment
func (th *TestHarness) SubmitTask(task *SchedulableTask, assignment *TaskAssignment) error {
	// Configure assignment
	th.ManagerAgent.SetAssignment(task.ID, assignment)
	
	// Add to kanban
	th.KanbanClient.AddTask(&kanban.Task{
		ID:       task.ID,
		Title:    "Task " + task.ID,
		Status:   kanban.StatusTodo,
		Priority: mapToKanbanPriority(task.Priority),
	})
	
	// Submit to scheduler
	return th.Scheduler.SubmitTask(th.ctx, task)
}

// WaitForTasks waits for the specified number of tasks to complete
func (th *TestHarness) WaitForTasks(expectedCount int, timeout time.Duration) map[string]*TaskResult {
	results := make(map[string]*TaskResult)
	resultChan := th.Scheduler.GetResults()
	
	timeoutChan := time.After(timeout)
	for len(results) < expectedCount {
		select {
		case result := <-resultChan:
			results[result.TaskID] = &result
		case <-timeoutChan:
			return results
		}
	}
	
	return results
}

// Cleanup stops the scheduler and cleans up resources
func (th *TestHarness) Cleanup() error {
	th.cancel()
	
	stopCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	
	return th.Scheduler.Stop(stopCtx)
}

// ConcurrentTaskGenerator generates tasks concurrently for stress testing
type ConcurrentTaskGenerator struct {
	harness      *TestHarness
	taskCount    int
	commissionID string
	wg           sync.WaitGroup
}

// NewConcurrentTaskGenerator creates a new task generator
func NewConcurrentTaskGenerator(harness *TestHarness, taskCount int, commissionID string) *ConcurrentTaskGenerator {
	return &ConcurrentTaskGenerator{
		harness:      harness,
		taskCount:    taskCount,
		commissionID: commissionID,
	}
}

// GenerateTasks creates and submits tasks concurrently
func (ctg *ConcurrentTaskGenerator) GenerateTasks(concurrency int) {
	taskChan := make(chan int, ctg.taskCount)
	
	// Fill task channel
	for i := 1; i <= ctg.taskCount; i++ {
		taskChan <- i
	}
	close(taskChan)
	
	// Start workers
	for i := 0; i < concurrency; i++ {
		ctg.wg.Add(1)
		go ctg.taskWorker(taskChan)
	}
	
	ctg.wg.Wait()
}

func (ctg *ConcurrentTaskGenerator) taskWorker(taskChan <-chan int) {
	defer ctg.wg.Done()
	
	for taskNum := range taskChan {
		taskID := generateTaskID(taskNum)
		agentID := selectAgent(taskNum, len(ctg.harness.Agents))
		
		task := &SchedulableTask{
			ID:           taskID,
			CommissionID: ctg.commissionID,
			Priority:     calculatePriority(taskNum),
			Dependencies: generateDependencies(taskNum, ctg.taskCount),
		}
		
		assignment := &TaskAssignment{
			TaskID:  taskID,
			AgentID: agentID,
		}
		
		_ = ctg.harness.SubmitTask(task, assignment)
	}
}

// Helper functions for task generation
func generateTaskID(num int) string {
	return fmt.Sprintf("task-%04d", num)
}

func selectAgent(taskNum int, agentCount int) string {
	if agentCount == 0 {
		return "agent-1"
	}
	return fmt.Sprintf("agent-%d", ((taskNum-1)%agentCount)+1)
}

func calculatePriority(taskNum int) int {
	// Create varied priorities
	return 100 - (taskNum % 50)
}

func generateDependencies(taskNum, totalTasks int) []string {
	var deps []string
	
	// Create dependency chains and DAG structure
	if taskNum > 1 && taskNum%3 == 0 {
		// Depends on previous task
		deps = append(deps, generateTaskID(taskNum-1))
	}
	
	if taskNum > 5 && taskNum%5 == 0 {
		// Depends on earlier task
		deps = append(deps, generateTaskID(taskNum-5))
	}
	
	return deps
}

// MetricsCollector collects and analyzes scheduler metrics
type MetricsCollector struct {
	scheduler         *TaskScheduler
	collectionPeriod  time.Duration
	samples           []SchedulerMetricsSample
	mu                sync.Mutex
	stopChan          chan struct{}
}

// SchedulerMetricsSample represents a point-in-time metrics sample
type SchedulerMetricsSample struct {
	Timestamp         time.Time
	RunningTasks      int
	QueuedTasks       int
	CompletedTasks    int64
	FailedTasks       int64
	AgentUtilization  map[string]float64
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector(scheduler *TaskScheduler, period time.Duration) *MetricsCollector {
	return &MetricsCollector{
		scheduler:        scheduler,
		collectionPeriod: period,
		samples:          make([]SchedulerMetricsSample, 0),
		stopChan:         make(chan struct{}),
	}
}

// Start begins collecting metrics
func (mc *MetricsCollector) Start() {
	go mc.collectLoop()
}

// Stop stops collecting metrics
func (mc *MetricsCollector) Stop() {
	close(mc.stopChan)
}

func (mc *MetricsCollector) collectLoop() {
	ticker := time.NewTicker(mc.collectionPeriod)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			mc.collectSample()
		case <-mc.stopChan:
			return
		}
	}
}

func (mc *MetricsCollector) collectSample() {
	stats := mc.scheduler.GetSchedulerStats()
	
	sample := SchedulerMetricsSample{
		Timestamp:        time.Now(),
		RunningTasks:     stats["running_tasks"].(int),
		QueuedTasks:      stats["queued_tasks"].(int),
		CompletedTasks:   stats["tasks_completed"].(int64),
		FailedTasks:      stats["tasks_failed"].(int64),
		AgentUtilization: stats["agent_utilization"].(map[string]float64),
	}
	
	mc.mu.Lock()
	mc.samples = append(mc.samples, sample)
	mc.mu.Unlock()
}

// GetAnalysis returns analysis of collected metrics
func (mc *MetricsCollector) GetAnalysis() MetricsAnalysis {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	
	if len(mc.samples) == 0 {
		return MetricsAnalysis{}
	}
	
	analysis := MetricsAnalysis{
		TotalSamples:       len(mc.samples),
		Duration:           mc.samples[len(mc.samples)-1].Timestamp.Sub(mc.samples[0].Timestamp),
		AverageQueueDepth:  mc.calculateAverageQueueDepth(),
		MaxConcurrentTasks: mc.findMaxConcurrentTasks(),
		ThroughputPerMin:   mc.calculateThroughput(),
		AgentEfficiency:    mc.calculateAgentEfficiency(),
	}
	
	return analysis
}

// MetricsAnalysis contains analyzed metrics
type MetricsAnalysis struct {
	TotalSamples       int
	Duration           time.Duration
	AverageQueueDepth  float64
	MaxConcurrentTasks int
	ThroughputPerMin   float64
	AgentEfficiency    map[string]float64
}

func (mc *MetricsCollector) calculateAverageQueueDepth() float64 {
	if len(mc.samples) == 0 {
		return 0
	}
	
	total := 0
	for _, sample := range mc.samples {
		total += sample.QueuedTasks
	}
	
	return float64(total) / float64(len(mc.samples))
}

func (mc *MetricsCollector) findMaxConcurrentTasks() int {
	max := 0
	for _, sample := range mc.samples {
		if sample.RunningTasks > max {
			max = sample.RunningTasks
		}
	}
	return max
}

func (mc *MetricsCollector) calculateThroughput() float64 {
	if len(mc.samples) < 2 {
		return 0
	}
	
	first := mc.samples[0]
	last := mc.samples[len(mc.samples)-1]
	
	completedDelta := last.CompletedTasks - first.CompletedTasks
	duration := last.Timestamp.Sub(first.Timestamp)
	
	if duration == 0 {
		return 0
	}
	
	return float64(completedDelta) / duration.Minutes()
}

func (mc *MetricsCollector) calculateAgentEfficiency() map[string]float64 {
	efficiency := make(map[string]float64)
	
	// Aggregate utilization across all samples
	utilizationSums := make(map[string]float64)
	utilizationCounts := make(map[string]int)
	
	for _, sample := range mc.samples {
		for agentID, utilization := range sample.AgentUtilization {
			utilizationSums[agentID] += utilization
			utilizationCounts[agentID]++
		}
	}
	
	// Calculate average utilization as efficiency
	for agentID, sum := range utilizationSums {
		if count := utilizationCounts[agentID]; count > 0 {
			efficiency[agentID] = sum / float64(count)
		}
	}
	
	return efficiency
}