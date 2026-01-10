// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package context

import (
	"context"
	"fmt"
	"time"

	"github.com/lancekrogers/guild-core/pkg/gerror"
)

// AgentClient represents a context-aware agent
type AgentClient interface {
	// Execute runs a task with full context support
	Execute(ctx context.Context, request string) (string, error)

	// GetID returns the agent's unique identifier
	GetID() string

	// GetName returns the agent's display name
	GetName() string

	// GetCapabilities returns the agent's capabilities
	GetCapabilities() []string

	// GetStatus returns the agent's current status
	GetStatus() AgentStatus
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

// TaskRequest represents a context-aware task request
type TaskRequest struct {
	ID            string                 `json:"id"`
	AgentID       string                 `json:"agent_id"`
	Content       string                 `json:"content"`
	Type          string                 `json:"type"`     // completion, analysis, coding, etc.
	Priority      int                    `json:"priority"` // 1-10, 10 being highest
	Timeout       time.Duration          `json:"timeout"`
	RequiredTools []string               `json:"required_tools"`
	Context       map[string]interface{} `json:"context"`

	// Execution context
	RequestID    string                 `json:"request_id"`
	SessionID    string                 `json:"session_id"`
	ParentTaskID string                 `json:"parent_task_id,omitempty"`
	Dependencies []string               `json:"dependencies,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// TaskResponse represents a context-aware task response
type TaskResponse struct {
	ID         string                 `json:"id"`
	TaskID     string                 `json:"task_id"`
	AgentID    string                 `json:"agent_id"`
	Result     string                 `json:"result"`
	Status     string                 `json:"status"` // success, error, timeout, cancelled
	Error      string                 `json:"error,omitempty"`
	StartTime  time.Time              `json:"start_time"`
	EndTime    time.Time              `json:"end_time"`
	Duration   time.Duration          `json:"duration"`
	TokensUsed int                    `json:"tokens_used"`
	CostUSD    float64                `json:"cost_usd"`
	ToolsUsed  []string               `json:"tools_used"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// ==============================================================================
// Context-Aware Agent Operations
// ==============================================================================

// ExecuteWithAgent runs a task using a specific agent from context
func ExecuteWithAgent(ctx context.Context, agentName, request string) (string, error) {
	// Get agent from context
	agent, err := GetAgentFromContext(ctx, agentName)
	if err != nil {
		return "", gerror.Wrap(err, gerror.ErrCodeNotFound, "failed to get agent").WithComponent("context").WithOperation("ExecuteWithAgent").WithDetails("agent_name", agentName)
	}

	// Try to cast to our context-aware interface
	if contextAgent, ok := agent.(AgentClient); ok {
		// Create enhanced context for this agent execution
		agentCtx := WithAgentID(ctx, contextAgent.GetID())
		agentCtx = WithOperation(agentCtx, "agent-execute")

		return contextAgent.Execute(agentCtx, request)
	}

	// Fallback to basic agent interface
	if basicAgent, ok := agent.(interface {
		Execute(context.Context, string) (string, error)
		GetID() string
	}); ok {
		agentCtx := WithAgentID(ctx, basicAgent.GetID())
		return basicAgent.Execute(agentCtx, request)
	}

	return "", gerror.Newf(gerror.ErrCodeInvalidInput, "agent '%s' does not implement required execution interface", agentName).WithComponent("context").WithOperation("ExecuteWithAgent")
}

// ExecuteWithDefaultAgent runs a task using the default agent from context
func ExecuteWithDefaultAgent(ctx context.Context, request string) (string, error) {
	registry, err := GetRegistryProvider(ctx)
	if err != nil {
		return "", gerror.Wrap(err, gerror.ErrCodeNotFound, "failed to get registry from context").WithComponent("context").WithOperation("ExecuteWithDefaultAgent")
	}

	// Get default agent
	agent, err := registry.Agents().GetDefaultAgent()
	if err != nil {
		return "", gerror.Wrap(err, gerror.ErrCodeNotFound, "failed to get default agent").WithComponent("context").WithOperation("ExecuteWithDefaultAgent")
	}

	// Try to cast to our context-aware interface
	if contextAgent, ok := agent.(AgentClient); ok {
		agentCtx := WithAgentID(ctx, contextAgent.GetID())
		agentCtx = WithOperation(agentCtx, "agent-execute")
		return contextAgent.Execute(agentCtx, request)
	}

	// Fallback to basic agent interface
	if basicAgent, ok := agent.(interface {
		Execute(context.Context, string) (string, error)
		GetID() string
	}); ok {
		agentCtx := WithAgentID(ctx, basicAgent.GetID())
		return basicAgent.Execute(agentCtx, request)
	}

	return "", gerror.New(gerror.ErrCodeInvalidInput, "default agent does not implement required execution interface", nil).WithComponent("context").WithOperation("ExecuteWithDefaultAgent")
}

// CreateTaskRequest creates a context-aware task request
func CreateTaskRequest(ctx context.Context, content, taskType string) TaskRequest {
	requestID := GetRequestID(ctx)
	if requestID == "" {
		requestID = fmt.Sprintf("task-%d", time.Now().UnixNano())
	}

	return TaskRequest{
		ID:        fmt.Sprintf("%s-task", requestID),
		Content:   content,
		Type:      taskType,
		Priority:  5, // Default priority
		RequestID: requestID,
		SessionID: GetSessionID(ctx),
		Context:   make(map[string]interface{}),
		Metadata:  make(map[string]interface{}),
	}
}

// ExecuteTaskRequest executes a structured task request
func ExecuteTaskRequest(ctx context.Context, agentName string, req TaskRequest) (TaskResponse, error) {
	startTime := time.Now()

	// Enhance context with task information
	taskCtx := WithOperation(ctx, fmt.Sprintf("task-%s", req.Type))
	if req.Timeout > 0 {
		var cancel context.CancelFunc
		taskCtx, cancel = context.WithTimeout(taskCtx, req.Timeout)
		defer cancel()
	}

	// Execute the task
	result, err := ExecuteWithAgent(taskCtx, agentName, req.Content)
	endTime := time.Now()

	// Create response
	response := TaskResponse{
		ID:        fmt.Sprintf("%s-response", req.ID),
		TaskID:    req.ID,
		AgentID:   req.AgentID,
		Result:    result,
		StartTime: startTime,
		EndTime:   endTime,
		Duration:  endTime.Sub(startTime),
		Metadata:  make(map[string]interface{}),
	}

	if err != nil {
		response.Status = "error"
		response.Error = err.Error()
	} else {
		response.Status = "success"
	}

	// Add cost information if available
	if costInfo := GetCostInfo(ctx); costInfo != nil {
		response.CostUSD = costInfo.Used
	}

	return response, err
}

// ==============================================================================
// Agent Selection and Routing
// ==============================================================================

// SelectBestAgent chooses the best agent for a given task type
func SelectBestAgent(ctx context.Context, taskType string, requirements map[string]interface{}) (string, error) {
	registry, err := GetRegistryProvider(ctx)
	if err != nil {
		return "", gerror.Wrap(err, gerror.ErrCodeNotFound, "failed to get registry from context").WithComponent("context").WithOperation("SelectBestAgent")
	}

	// Get all available agents
	agents := registry.Agents().ListAgents()
	if len(agents) == 0 {
		return "", gerror.New(gerror.ErrCodeNotFound, "no agents available", nil).WithComponent("context").WithOperation("SelectBestAgent")
	}

	// Simple selection logic - in production this could be more sophisticated
	// considering factors like agent capabilities, current load, past performance, etc.

	switch taskType {
	case "coding", "code-review", "debugging":
		// Look for specialized coding agents
		for _, agentName := range agents {
			if agent, err := registry.Agents().GetAgent(agentName); err == nil {
				if contextAgent, ok := agent.(AgentClient); ok {
					capabilities := contextAgent.GetCapabilities()
					for _, cap := range capabilities {
						if cap == "coding" || cap == "development" {
							return agentName, nil
						}
					}
				}
			}
		}

	case "analysis", "reasoning":
		// Look for analytical agents
		for _, agentName := range agents {
			if agent, err := registry.Agents().GetAgent(agentName); err == nil {
				if contextAgent, ok := agent.(AgentClient); ok {
					capabilities := contextAgent.GetCapabilities()
					for _, cap := range capabilities {
						if cap == "analysis" || cap == "reasoning" {
							return agentName, nil
						}
					}
				}
			}
		}

	case "general", "completion":
		// Look for general-purpose agents
		for _, agentName := range agents {
			if agent, err := registry.Agents().GetAgent(agentName); err == nil {
				if contextAgent, ok := agent.(AgentClient); ok {
					capabilities := contextAgent.GetCapabilities()
					for _, cap := range capabilities {
						if cap == "general" || cap == "completion" {
							return agentName, nil
						}
					}
				}
			}
		}
	}

	// Default to first available agent
	return agents[0], nil
}

// RouteToAgent routes a request to the most appropriate agent
func RouteToAgent(ctx context.Context, taskType, request string, requirements map[string]interface{}) (string, error) {
	// Select best agent
	agentName, err := SelectBestAgent(ctx, taskType, requirements)
	if err != nil {
		return "", gerror.Wrap(err, gerror.ErrCodeNotFound, "failed to select agent").WithComponent("context").WithOperation("RouteToAgent")
	}

	// Create enhanced context
	ctx = WithOperation(ctx, fmt.Sprintf("routed-%s", taskType))

	// Execute with selected agent
	return ExecuteWithAgent(ctx, agentName, request)
}

// ==============================================================================
// Agent Monitoring and Health
// ==============================================================================

// GetAgentStatus retrieves the current status of an agent
func GetAgentStatus(ctx context.Context, agentName string) (AgentStatus, error) {
	agent, err := GetAgentFromContext(ctx, agentName)
	if err != nil {
		return AgentStatus{}, gerror.Wrap(err, gerror.ErrCodeNotFound, "failed to get agent").WithComponent("context").WithOperation("GetAgentStatus").WithDetails("agent_name", agentName)
	}

	if contextAgent, ok := agent.(AgentClient); ok {
		return contextAgent.GetStatus(), nil
	}

	// Fallback status for basic agents
	return AgentStatus{
		State:      "unknown",
		LastActive: time.Now(),
		Metadata:   make(map[string]interface{}),
	}, nil
}

// GetAllAgentStatuses retrieves status for all registered agents
func GetAllAgentStatuses(ctx context.Context) (map[string]AgentStatus, error) {
	registry, err := GetRegistryProvider(ctx)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeNotFound, "failed to get registry from context").WithComponent("context").WithOperation("GetAllAgentStatuses")
	}

	agents := registry.Agents().ListAgents()
	statuses := make(map[string]AgentStatus)

	for _, agentName := range agents {
		if status, err := GetAgentStatus(ctx, agentName); err == nil {
			statuses[agentName] = status
		}
	}

	return statuses, nil
}

// MonitorAgentHealth checks the health of all agents
func MonitorAgentHealth(ctx context.Context) (map[string]bool, error) {
	statuses, err := GetAllAgentStatuses(ctx)
	if err != nil {
		return nil, err
	}

	health := make(map[string]bool)
	for agentName, status := range statuses {
		// Simple health check - agent is healthy if not in error state
		health[agentName] = status.State != "error" && status.State != "disabled"
	}

	return health, nil
}
