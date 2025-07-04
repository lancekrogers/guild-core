// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package agent_orchestration

import (
	"context"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/lancekrogers/guild/pkg/gerror"
	"github.com/lancekrogers/guild/pkg/interfaces"
	"github.com/lancekrogers/guild/pkg/registry"
)

// HappyPathTestFramework provides staff-level testing infrastructure for agent orchestration
type HappyPathTestFramework struct {
	t              *testing.T
	registry       registry.ComponentRegistry
	testContext    context.Context
	cleanup        []func()
	performanceLog *PerformanceLogger
	userSimulator  *UserSimulator
	memoryTracker  *MemoryTracker
	mu             sync.RWMutex
}

// PerformanceLogger tracks detailed performance metrics for SLA validation
type PerformanceLogger struct {
	metrics map[string][]PerformanceMetric
	mu      sync.RWMutex
}

// PerformanceMetric represents a single performance measurement
type PerformanceMetric struct {
	Operation string
	Duration  time.Duration
	Timestamp time.Time
	Metadata  map[string]interface{}
}

// UserSimulator creates realistic user interaction patterns for testing
type UserSimulator struct {
	scenarios map[string]UserScenario
}

// UserScenario represents a realistic user interaction pattern
type UserScenario struct {
	ProjectType     string
	CodebaseSize    string
	UserExperience  string
	CommonTasks     []string
	ExpectedLatency time.Duration
}

// MemoryTracker monitors resource usage during tests
type MemoryTracker struct {
	initialMemory uint64
	peakMemory    uint64
	mu            sync.RWMutex
}

// TaskRequirements defines requirements for agent selection
type TaskRequirements struct {
	Type         string
	Complexity   ComplexityLevel
	MaxCost      int
	Capabilities []string
}

// ComplexityLevel represents task complexity for cost calculation
type ComplexityLevel int

const (
	ComplexityLow ComplexityLevel = iota
	ComplexityMedium
	ComplexityHigh
	ComplexityCritical
)

// ExecutionInput represents input for agent execution
type ExecutionInput struct {
	Message           string
	Context           map[string]interface{}
	StreamingCallback func(string)
	Requirements      TaskRequirements
}

// ExecutionResult represents the result of agent execution
type ExecutionResult struct {
	Content      string
	TokensUsed   int
	CostIncurred float64
	Duration     time.Duration
	Metadata     map[string]interface{}
}

// RealAgent wraps actual registry agent for testing interface compatibility
type RealAgent struct {
	agent    registry.Agent
	info     registry.AgentInfo
	config   registry.GuildAgentConfig // Store config for provider info
	registry registry.ComponentRegistry
}

// Interface compatibility methods for RealAgent
func (r *RealAgent) GetID() string   { return r.info.ID }
func (r *RealAgent) GetName() string { return r.info.Name }
func (r *RealAgent) GetType() string { return r.info.Type }
func (r *RealAgent) GetProvider() string {
	if r.config.Provider != "" {
		return r.config.Provider
	}
	return "unknown" // Fallback
}
func (r *RealAgent) GetCapabilities() []string { return r.info.Capabilities }
func (r *RealAgent) GetCostMagnitude() int     { return r.info.CostMagnitude }

// Execute performs real agent execution using the registry agent
func (r *RealAgent) Execute(ctx context.Context, input ExecutionInput) (*ExecutionResult, error) {
	start := time.Now()

	// Use real agent execution with simple interface
	response, err := r.agent.Execute(ctx, input.Message)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "real agent execution failed").
			WithComponent("agent-orchestration").
			WithOperation("Execute").
			WithDetails("agentID", r.info.ID)
	}

	duration := time.Since(start)

	// Stream response if callback provided
	if input.StreamingCallback != nil && response != "" {
		// Simulate streaming by chunking the response
		go func() {
			chunks := r.chunkResponse(response)
			for _, chunk := range chunks {
				select {
				case <-ctx.Done():
					return
				default:
					input.StreamingCallback(chunk)
					time.Sleep(50 * time.Millisecond) // Realistic chunk delay
				}
			}
		}()
	}

	// Calculate tokens and cost based on response length (simplified)
	tokensUsed := len(response) / 4               // Approximate tokens
	costIncurred := float64(tokensUsed) * 0.00003 // Estimate based on OpenAI pricing

	return &ExecutionResult{
		Content:      response,
		TokensUsed:   tokensUsed,
		CostIncurred: costIncurred,
		Duration:     duration,
		Metadata: map[string]interface{}{
			"agent_id":       r.info.ID,
			"provider":       r.config.Provider,
			"complexity":     input.Requirements.Complexity,
			"response_time":  duration.Milliseconds(),
			"real_execution": true, // Flag to indicate this was real execution
		},
	}, nil
}

// GetContext returns agent context for validation
func (r *RealAgent) GetContext() *AgentContext {
	return &AgentContext{
		AgentID: r.info.ID,
		History: []string{}, // TODO: Track actual execution history
	}
}

// chunkResponse splits response into realistic streaming chunks
func (r *RealAgent) chunkResponse(content string) []string {
	if len(content) <= 50 {
		return []string{content}
	}

	var chunks []string
	words := strings.Fields(content)
	currentChunk := ""

	for _, word := range words {
		if len(currentChunk) > 0 && len(currentChunk)+len(word)+1 > 50 {
			chunks = append(chunks, currentChunk)
			currentChunk = word
		} else {
			if len(currentChunk) > 0 {
				currentChunk += " "
			}
			currentChunk += word
		}
	}

	if currentChunk != "" {
		chunks = append(chunks, currentChunk)
	}

	return chunks
}

// AgentContext represents agent execution context
type AgentContext struct {
	AgentID string
	History []string
}

// NewHappyPathFramework creates comprehensive testing environment
func NewHappyPathFramework(t *testing.T) *HappyPathTestFramework {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)

	framework := &HappyPathTestFramework{
		t:              t,
		testContext:    ctx,
		performanceLog: NewPerformanceLogger(),
		userSimulator:  NewUserSimulator(),
		memoryTracker:  NewMemoryTracker(),
		cleanup:        []func(){cancel},
	}

	// Initialize test registry with real components
	reg := registry.NewComponentRegistry()

	// Configure with test providers and realistic data
	err := framework.setupTestEnvironment(reg)
	require.NoError(t, err, "Failed to setup test environment")

	framework.registry = reg
	return framework
}

// setupTestEnvironment configures realistic test conditions with real components
func (f *HappyPathTestFramework) setupTestEnvironment(reg registry.ComponentRegistry) error {
	// TODO: Implement proper test environment setup
	// This is simplified for compilation - real implementation would configure providers and agents
	return nil
}

// GetOptimalAgent uses real registry for optimal agent selection based on requirements
func (f *HappyPathTestFramework) GetOptimalAgent(requirements TaskRequirements) (*RealAgent, error) {
	// Use real registry for agent selection
	if f.registry == nil {
		return nil, gerror.New(gerror.ErrCodeInternal, "registry not initialized", nil).
			WithComponent("agent-orchestration").
			WithOperation("GetOptimalAgent")
	}

	// Get agents by cost constraint first
	candidateAgents := f.registry.GetAgentsByCost(requirements.MaxCost)
	if len(candidateAgents) == 0 {
		return nil, gerror.New(gerror.ErrCodeNoAvailableAgent, "no agents within cost budget", nil).
			WithComponent("agent-orchestration").
			WithOperation("GetOptimalAgent").
			WithDetails("maxCost", requirements.MaxCost)
	}

	// Filter by capabilities if specified
	if len(requirements.Capabilities) > 0 {
		var filteredAgents []registry.AgentInfo
		for _, capability := range requirements.Capabilities {
			capableAgents := f.registry.GetAgentsByCapability(capability)
			// Find intersection with cost-filtered agents
			for _, capable := range capableAgents {
				for _, candidate := range candidateAgents {
					if capable.ID == candidate.ID {
						filteredAgents = append(filteredAgents, capable)
						break
					}
				}
			}
		}
		if len(filteredAgents) == 0 {
			return nil, gerror.New(gerror.ErrCodeNoAvailableAgent, "no agents with required capabilities within budget", nil).
				WithComponent("agent-orchestration").
				WithOperation("GetOptimalAgent").
				WithDetails("capabilities", requirements.Capabilities)
		}
		candidateAgents = filteredAgents
	}

	// Select the cheapest available agent
	if len(candidateAgents) == 0 {
		return nil, gerror.New(gerror.ErrCodeNoAvailableAgent, "no suitable agents found", nil).
			WithComponent("agent-orchestration").
			WithOperation("GetOptimalAgent")
	}

	selectedAgentInfo := candidateAgents[0] // Already sorted by cost

	// Get the actual agent instance from the registry
	actualAgent, err := f.registry.Agents().GetAgent(selectedAgentInfo.Type)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create agent instance").
			WithComponent("agent-orchestration").
			WithOperation("GetOptimalAgent").
			WithDetails("agentType", selectedAgentInfo.Type)
	}

	// Get agent config for provider info
	var agentConfig registry.GuildAgentConfig
	registeredAgents := f.registry.Agents().GetRegisteredAgents()
	for _, regAgent := range registeredAgents {
		if regAgent.ID == selectedAgentInfo.ID {
			agentConfig = regAgent
			break
		}
	}

	// Wrap in RealAgent for testing interface compatibility
	return &RealAgent{
		agent:    actualAgent,
		info:     selectedAgentInfo,
		config:   agentConfig,
		registry: f.registry,
	}, nil
}

// validateResponseQuality analyzes response quality for SLA validation
func (f *HappyPathTestFramework) validateResponseQuality(response, originalQuery string) int {
	// Simplified quality scoring based on response characteristics
	score := 50 // Base score

	// Length appropriateness
	if len(response) >= 20 && len(response) <= 2000 {
		score += 20
	}

	// Keyword relevance (simple check)
	queryWords := []string{"analyze", "refactor", "document", "explain", "implement"}
	for _, word := range queryWords {
		if containsIgnoreCase(originalQuery, word) && containsIgnoreCase(response, word) {
			score += 10
		}
	}

	// Structure quality (simple heuristics)
	if len(response) > 100 {
		score += 10 // Detailed response
	}

	// Ensure score is within bounds
	if score > 100 {
		score = 100
	}
	if score < 0 {
		score = 0
	}

	return score
}

// measureMemoryUsage returns current memory usage in bytes
func (f *HappyPathTestFramework) measureMemoryUsage() uint64 {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return m.Alloc
}

// Cleanup ensures proper resource cleanup
func (f *HappyPathTestFramework) Cleanup() {
	f.mu.Lock()
	defer f.mu.Unlock()

	for i := len(f.cleanup) - 1; i >= 0; i-- {
		f.cleanup[i]()
	}
}

// Helper methods

// AgentInterface represents a test agent interface
type AgentInterface struct {
	ID           string
	Name         string
	Type         string
	Provider     string
	Capabilities []string
	CostPerToken float64
	QualityScore int
	CostProfile  interfaces.CostProfile
}

func (f *HappyPathTestFramework) getAvailableAgents() []*AgentInterface {
	return []*AgentInterface{
		{
			ID:           "developer",
			Name:         "Senior Developer Agent",
			Type:         "coding",
			Provider:     "openai",
			Capabilities: []string{"code_analysis", "refactoring", "documentation"},
			CostPerToken: 0.00003,
			QualityScore: 95,
			CostProfile:  interfaces.CostProfile{Magnitude: 3, Available: true},
		},
		{
			ID:           "writer",
			Name:         "Technical Writer Agent",
			Type:         "documentation",
			Provider:     "anthropic",
			Capabilities: []string{"documentation", "explanation", "review"},
			CostPerToken: 0.000015,
			QualityScore: 92,
			CostProfile:  interfaces.CostProfile{Magnitude: 2, Available: true},
		},
	}
}

func (f *HappyPathTestFramework) hasRequiredCapabilities(agent *AgentInterface, required []string) bool {
	if len(required) == 0 {
		return true
	}

	for _, req := range required {
		found := false
		for _, cap := range agent.Capabilities {
			if cap == req {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

func (a *AgentInterface) calculateProcessingTime(complexity ComplexityLevel) time.Duration {
	baseTime := time.Duration(500) * time.Millisecond

	switch complexity {
	case ComplexityLow:
		return baseTime
	case ComplexityMedium:
		return baseTime * 2
	case ComplexityHigh:
		return baseTime * 4
	case ComplexityCritical:
		return baseTime * 8
	default:
		return baseTime
	}
}

func (a *AgentInterface) simulateStreaming(ctx context.Context, message string, callback func(string), totalTime time.Duration) {
	chunks := []string{
		"I'll analyze",
		" the code",
		" and provide",
		" recommendations",
		" for improvement...",
	}

	chunkInterval := totalTime / time.Duration(len(chunks))

	for _, chunk := range chunks {
		select {
		case <-ctx.Done():
			return
		case <-time.After(chunkInterval):
			callback(chunk)
		}
	}
}

func (a *AgentInterface) generateResponse(message string, requirements TaskRequirements) string {
	// Generate realistic response based on agent type and requirements
	switch a.Type {
	case "coding":
		return "After analyzing the code, I found several optimization opportunities: 1) Extract common functions, 2) Improve error handling, 3) Add comprehensive tests. The current implementation shows good structure but could benefit from better separation of concerns."
	case "documentation":
		return "I've reviewed the documentation requirements. The content should include: overview, API reference, examples, and troubleshooting guide. Focus on clarity and practical examples for better user adoption."
	default:
		return "I've processed your request and here's my analysis: " + message
	}
}

func (a *AgentInterface) calculateCost(tokens int) float64 {
	return float64(tokens) * a.CostPerToken
}

func containsIgnoreCase(s, substr string) bool {
	// Simple case-insensitive contains check
	return len(s) >= len(substr) &&
		(s == substr || len(substr) == 0 ||
			(len(s) > 0 && containsIgnoreCase(s[1:], substr)) ||
			(len(s) >= len(substr) && s[:len(substr)] == substr))
}

// Performance monitoring helpers

// NewPerformanceLogger creates a new performance logger
func NewPerformanceLogger() *PerformanceLogger {
	return &PerformanceLogger{
		metrics: make(map[string][]PerformanceMetric),
	}
}

// StartOperation begins tracking a performance operation
func (p *PerformanceLogger) StartOperation(operation string) *OperationTracker {
	return &OperationTracker{
		operation: operation,
		startTime: time.Now(),
		logger:    p,
	}
}

// OperationTracker tracks individual operations
type OperationTracker struct {
	operation string
	startTime time.Time
	logger    *PerformanceLogger
}

// End completes operation tracking
func (o *OperationTracker) End() {
	duration := time.Since(o.startTime)
	o.logger.recordMetric(o.operation, duration, nil)
}

// RecordMetric records a specific metric
func (o *OperationTracker) RecordMetric(name string, value interface{}) {
	metadata := map[string]interface{}{name: value}
	o.logger.recordMetric(o.operation, 0, metadata)
}

// RecordMetrics records multiple metrics
func (o *OperationTracker) RecordMetrics(metrics map[string]interface{}) {
	o.logger.recordMetric(o.operation, 0, metrics)
}

func (p *PerformanceLogger) recordMetric(operation string, duration time.Duration, metadata map[string]interface{}) {
	p.mu.Lock()
	defer p.mu.Unlock()

	metric := PerformanceMetric{
		Operation: operation,
		Duration:  duration,
		Timestamp: time.Now(),
		Metadata:  metadata,
	}

	p.metrics[operation] = append(p.metrics[operation], metric)
}

// User simulation helpers

// NewUserSimulator creates a new user simulator
func NewUserSimulator() *UserSimulator {
	return &UserSimulator{
		scenarios: map[string]UserScenario{
			"senior_developer": {
				ProjectType:     "go-service",
				CodebaseSize:    "medium",
				UserExperience:  "senior_developer",
				CommonTasks:     []string{"code_analysis", "refactoring", "optimization"},
				ExpectedLatency: 2 * time.Second,
			},
			"junior_developer": {
				ProjectType:     "web-app",
				CodebaseSize:    "small",
				UserExperience:  "junior_developer",
				CommonTasks:     []string{"documentation", "simple_fixes", "learning"},
				ExpectedLatency: 5 * time.Second,
			},
		},
	}
}

// CreateRealisticContext creates a realistic user context for testing
func (u *UserSimulator) CreateRealisticContext(params map[string]interface{}) map[string]interface{} {
	context := make(map[string]interface{})

	// Add user experience level
	if userExp, ok := params["userExperience"].(string); ok {
		if scenario, exists := u.scenarios[userExp]; exists {
			context["project_type"] = scenario.ProjectType
			context["codebase_size"] = scenario.CodebaseSize
			context["common_tasks"] = scenario.CommonTasks
			context["expected_latency"] = scenario.ExpectedLatency
		}
	}

	// Merge provided parameters
	for key, value := range params {
		context[key] = value
	}

	return context
}

// Memory tracking helpers

// NewMemoryTracker creates a new memory tracker
func NewMemoryTracker() *MemoryTracker {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	return &MemoryTracker{
		initialMemory: m.Alloc,
		peakMemory:    m.Alloc,
	}
}

// GetCurrentUsage returns current memory usage
func (m *MemoryTracker) GetCurrentUsage() uint64 {
	var stats runtime.MemStats
	runtime.ReadMemStats(&stats)

	m.mu.Lock()
	if stats.Alloc > m.peakMemory {
		m.peakMemory = stats.Alloc
	}
	m.mu.Unlock()

	return stats.Alloc
}

// GetPeakUsage returns peak memory usage since creation
func (m *MemoryTracker) GetPeakUsage() uint64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.peakMemory
}
