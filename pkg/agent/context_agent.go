package agent

import (
	"context"
	"fmt"
	"time"

	guildcontext "github.com/guild-ventures/guild-core/pkg/context"
	"github.com/guild-ventures/guild-core/pkg/gerror"
)

// ContextAwareAgent represents an agent that uses the Guild context system
type ContextAwareAgent struct {
	ID           string
	Name         string
	AgentType    string
	Capabilities []string

	// Context-aware components - discovered through context instead of injected
	defaultProvider string
	systemPrompt    string

	// Status tracking
	status     AgentStatus
	taskCount  int64
	errorCount int64
}

// AgentStatus represents the current status of an agent
type AgentStatus struct {
	State          string                 `json:"state"`           // idle, busy, error, disabled
	CurrentTask    string                 `json:"current_task"`    // description of current task
	LastActive     time.Time              `json:"last_active"`     // last activity timestamp
	TaskCount      int64                  `json:"task_count"`      // total tasks executed
	SuccessCount   int64                  `json:"success_count"`   // successful tasks
	ErrorCount     int64                  `json:"error_count"`     // failed tasks
	AverageLatency time.Duration          `json:"average_latency"` // average task execution time
	Metadata       map[string]interface{} `json:"metadata"`        // additional status information
}

// newContextAwareAgent creates a new context-aware agent (private constructor)
func newContextAwareAgent(id, name, agentType string, capabilities []string) *ContextAwareAgent {
	return &ContextAwareAgent{
		ID:           id,
		Name:         name,
		AgentType:    agentType,
		Capabilities: capabilities,
		status: AgentStatus{
			State:      "idle",
			LastActive: time.Now(),
			Metadata:   make(map[string]interface{}),
		},
	}
}

// Execute runs a task using the context system for component discovery
func (a *ContextAwareAgent) Execute(ctx context.Context, request string) (string, error) {
	startTime := time.Now()

	// Update agent status
	a.status.State = "busy"
	a.status.CurrentTask = request
	a.status.LastActive = time.Now()
	a.taskCount++

	// Create enhanced context for this execution
	execCtx := guildcontext.CreateComponentContext(ctx, "agent", a.ID, "execute")

	// Log the execution start
	if logger := guildcontext.GetLogger(execCtx); logger != nil {
		logger.Info("Agent executing request", guildcontext.LogFields(execCtx)...)
	}

	// Execute the request
	result, err := a.executeWithContext(execCtx, request)

	// Update status and metrics
	duration := time.Since(startTime)
	a.status.LastActive = time.Now()

	if err != nil {
		a.status.State = "error"
		a.errorCount++

		// Log error with context
		if logger := guildcontext.GetLogger(execCtx); logger != nil {
			logger.Error("Agent execution failed", append(guildcontext.LogFields(execCtx), "error", err.Error(), "duration_ms", duration.Milliseconds())...)
		}
	} else {
		a.status.State = "idle"
		a.status.SuccessCount++

		// Log success with context
		if logger := guildcontext.GetLogger(execCtx); logger != nil {
			logger.Info("Agent execution completed", append(guildcontext.LogFields(execCtx), "duration_ms", duration.Milliseconds())...)
		}
	}

	// Update average latency
	if a.taskCount > 0 {
		totalLatency := a.status.AverageLatency * time.Duration(a.taskCount-1)
		a.status.AverageLatency = (totalLatency + duration) / time.Duration(a.taskCount)
	}

	return result, err
}

// executeWithContext performs the actual execution using context-aware patterns
func (a *ContextAwareAgent) executeWithContext(ctx context.Context, request string) (string, error) {
	// Determine the best provider for this request
	providerName, err := a.selectProvider(ctx, request)
	if err != nil {
		return "", gerror.Wrap(err, gerror.ErrCodeAgent, "failed to select provider").
			WithComponent("agent").
			WithOperation("executeWithContext").
			WithDetails("agent_id", a.ID).
			WithDetails("agent_type", a.AgentType)
	}

	// Enhance context with provider information
	ctx = guildcontext.WithProvider(ctx, providerName)

	// Create system-enhanced prompt
	enhancedPrompt := a.createSystemPrompt(request)

	// Execute with the selected provider
	result, err := guildcontext.CompleteWithProvider(ctx, providerName, enhancedPrompt)
	if err != nil {
		return "", gerror.Wrap(err, gerror.ErrCodeProvider, "provider execution failed").
			WithComponent("agent").
			WithOperation("executeWithContext").
			WithDetails("agent_id", a.ID).
			WithDetails("provider", providerName)
	}

	// Post-process the result if needed
	processedResult := a.postProcessResult(ctx, request, result)

	return processedResult, nil
}

// selectProvider chooses the best provider for the given request
func (a *ContextAwareAgent) selectProvider(ctx context.Context, request string) (string, error) {
	// If agent has a default provider preference, try to use it
	if a.defaultProvider != "" {
		// Check if the preferred provider is available
		if _, err := guildcontext.GetProviderFromContext(ctx, a.defaultProvider); err == nil {
			return a.defaultProvider, nil
		}
	}

	// Determine task type based on agent capabilities and request content
	taskType := a.determineTaskType(request)

	// Use context-aware provider selection
	return guildcontext.SelectBestProvider(ctx, taskType, map[string]interface{}{
		"agent_type":     a.AgentType,
		"capabilities":   a.Capabilities,
		"request_length": len(request),
	})
}

// determineTaskType analyzes the request to determine the type of task
func (a *ContextAwareAgent) determineTaskType(request string) string {
	// Simple task type detection based on agent capabilities and request content
	requestLower := fmt.Sprintf("%s %s", request, a.AgentType)

	// Check agent capabilities first
	for _, capability := range a.Capabilities {
		switch capability {
		case "coding", "development", "programming":
			if containsAny(requestLower, []string{"code", "function", "implement", "debug", "refactor"}) {
				return "coding"
			}
		case "analysis", "reasoning":
			if containsAny(requestLower, []string{"analyze", "explain", "reason", "think", "evaluate"}) {
				return "reasoning"
			}
		case "writing", "content":
			if containsAny(requestLower, []string{"write", "draft", "compose", "create", "generate"}) {
				return "writing"
			}
		}
	}

	// Default based on request characteristics
	if len(request) < 100 {
		return "fast"
	}

	return "general"
}

// createSystemPrompt creates an enhanced prompt with system context
func (a *ContextAwareAgent) createSystemPrompt(request string) string {
	systemPrompt := a.systemPrompt
	if systemPrompt == "" {
		// Create default system prompt based on agent type and capabilities
		systemPrompt = fmt.Sprintf("You are %s, a %s agent", a.Name, a.AgentType)
		if len(a.Capabilities) > 0 {
			systemPrompt += fmt.Sprintf(" specialized in %v", a.Capabilities)
		}
		systemPrompt += ". Provide helpful, accurate, and concise responses."
	}

	// Combine system prompt with user request
	return fmt.Sprintf("%s\n\nUser Request: %s", systemPrompt, request)
}

// postProcessResult applies any post-processing to the result
func (a *ContextAwareAgent) postProcessResult(ctx context.Context, request, result string) string {
	// Add agent signature if configured
	if a.AgentType == "manager" {
		result = fmt.Sprintf("%s\n\n--- Response from %s (Manager Agent) ---", result, a.Name)
	}

	// Add cost information if available
	if costInfo := guildcontext.GetCostInfo(ctx); costInfo != nil && costInfo.Used > 0 {
		result = fmt.Sprintf("%s\n\n[Cost: $%.4f]", result, costInfo.Used)
	}

	return result
}

// GetID returns the agent's unique identifier
func (a *ContextAwareAgent) GetID() string {
	return a.ID
}

// GetName returns the agent's display name
func (a *ContextAwareAgent) GetName() string {
	return a.Name
}

// GetCapabilities returns the agent's capabilities
func (a *ContextAwareAgent) GetCapabilities() []string {
	return a.Capabilities
}

// GetStatus returns the agent's current status
func (a *ContextAwareAgent) GetStatus() guildcontext.AgentStatus {
	return guildcontext.AgentStatus{
		State:          a.status.State,
		CurrentTask:    a.status.CurrentTask,
		LastActive:     a.status.LastActive,
		TaskCount:      a.status.TaskCount,
		SuccessCount:   a.status.SuccessCount,
		ErrorCount:     a.status.ErrorCount,
		AverageLatency: a.status.AverageLatency,
		Metadata:       a.status.Metadata,
	}
}

// SetDefaultProvider sets the agent's preferred provider
func (a *ContextAwareAgent) SetDefaultProvider(providerName string) {
	a.defaultProvider = providerName
}

// SetSystemPrompt sets the agent's system prompt
func (a *ContextAwareAgent) SetSystemPrompt(prompt string) {
	a.systemPrompt = prompt
}

// GetAgentType returns the agent's type
func (a *ContextAwareAgent) GetAgentType() string {
	return a.AgentType
}

// UpdateCapabilities updates the agent's capabilities
func (a *ContextAwareAgent) UpdateCapabilities(capabilities []string) {
	a.Capabilities = capabilities
}

// AddMetadata adds metadata to the agent's status
func (a *ContextAwareAgent) AddMetadata(key string, value interface{}) {
	if a.status.Metadata == nil {
		a.status.Metadata = make(map[string]interface{})
	}
	a.status.Metadata[key] = value
}

// GetMetadata retrieves metadata from the agent's status
func (a *ContextAwareAgent) GetMetadata(key string) interface{} {
	if a.status.Metadata == nil {
		return nil
	}
	return a.status.Metadata[key]
}

// Reset resets the agent's status and counters
func (a *ContextAwareAgent) Reset() {
	a.status = AgentStatus{
		State:      "idle",
		LastActive: time.Now(),
		Metadata:   make(map[string]interface{}),
	}
	a.taskCount = 0
	a.errorCount = 0
}

// containsAny checks if the text contains any of the given keywords
func containsAny(text string, keywords []string) bool {
	for _, keyword := range keywords {
		if contains(text, keyword) {
			return true
		}
	}
	return false
}

// contains checks if the text contains the keyword (simple case-insensitive check)
func contains(text, keyword string) bool {
	// Simple implementation - in production you might want more sophisticated matching
	return len(text) >= len(keyword) &&
		findSubstring(text, keyword) != -1
}

// findSubstring performs a simple case-insensitive substring search
func findSubstring(text, substr string) int {
	if len(substr) == 0 {
		return 0
	}
	if len(text) < len(substr) {
		return -1
	}

	// Convert to lowercase for case-insensitive comparison
	textLower := toLowerCase(text)
	substrLower := toLowerCase(substr)

	for i := 0; i <= len(textLower)-len(substrLower); i++ {
		if textLower[i:i+len(substrLower)] == substrLower {
			return i
		}
	}
	return -1
}

// toLowerCase converts a string to lowercase
func toLowerCase(s string) string {
	result := make([]byte, len(s))
	for i, b := range []byte(s) {
		if b >= 'A' && b <= 'Z' {
			result[i] = b + 32
		} else {
			result[i] = b
		}
	}
	return string(result)
}
