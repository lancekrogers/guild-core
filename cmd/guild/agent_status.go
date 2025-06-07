package main

import (
	"sync"
	"time"

	"github.com/guild-ventures/guild-core/pkg/config"
)

// AgentStatusTracker monitors agent states and activities
type AgentStatusTracker struct {
	agents          map[string]*AgentStatus  // agentID -> status
	activeTools     map[string]*ToolStatus   // toolID -> status  
	globalActivity  []ActivityEvent          // Recent activity log
	updateChannel   chan AgentStatusUpdate   // Real-time updates
	guildConfig     *config.GuildConfig
	mutex           sync.RWMutex             // Thread-safe access
	maxActivity     int                      // Maximum activity events to keep
}

// AgentStatus represents current state of an agent
type AgentStatus struct {
	ID              string
	Name            string
	Type            string            // "manager", "worker", "specialist"
	State           AgentState        // idle, thinking, working, blocked
	CurrentTask     string            // Current task description
	Progress        float64           // 0.0 to 1.0
	LastActivity    time.Time
	ActiveTools     []string          // Currently running tools
	Capabilities    []string          // Agent specializations
	CostMagnitude   int              // Cost tier (1-5)
	TotalCost       float64          // Total cost accumulated
	TasksCompleted  int              // Number of completed tasks
	StartTime       time.Time        // When agent became active
}

// ToolStatus represents current state of a tool execution
type ToolStatus struct {
	ID          string
	Name        string
	AgentID     string
	State       ToolState
	Progress    float64
	StartTime   time.Time
	EndTime     *time.Time
	Cost        float64
	Parameters  map[string]string
	Result      string
	Error       string
}

// AgentState represents agent activity states
type AgentState int

const (
	AgentIdle AgentState = iota
	AgentThinking    // Processing input, planning
	AgentWorking     // Executing tasks, using tools
	AgentBlocked     // Waiting for input/resources
	AgentOffline     // Not available
)

// String returns the string representation of AgentState
func (s AgentState) String() string {
	switch s {
	case AgentIdle:
		return "idle"
	case AgentThinking:
		return "thinking"
	case AgentWorking:
		return "working"
	case AgentBlocked:
		return "blocked"
	case AgentOffline:
		return "offline"
	default:
		return "unknown"
	}
}

// ToolState represents tool execution states
type ToolState int

const (
	ToolStarting ToolState = iota
	ToolRunning
	ToolCompleted
	ToolFailed
	ToolCancelled
)

// String returns the string representation of ToolState
func (s ToolState) String() string {
	switch s {
	case ToolStarting:
		return "starting"
	case ToolRunning:
		return "running"
	case ToolCompleted:
		return "completed"
	case ToolFailed:
		return "failed"
	case ToolCancelled:
		return "cancelled"
	default:
		return "unknown"
	}
}

// ActivityEvent represents a logged activity event
type ActivityEvent struct {
	Timestamp   time.Time
	AgentID     string
	EventType   ActivityEventType
	Description string
	Metadata    map[string]interface{}
}

// ActivityEventType represents different types of activity events
type ActivityEventType int

const (
	ActivityAgentStarted ActivityEventType = iota
	ActivityAgentStopped
	ActivityTaskStarted
	ActivityTaskCompleted
	ActivityTaskFailed
	ActivityToolStarted
	ActivityToolCompleted
	ActivityToolFailed
	ActivityStateChanged
	ActivityCoordination // Multi-agent coordination event
)

// String returns the string representation of ActivityEventType
func (e ActivityEventType) String() string {
	switch e {
	case ActivityAgentStarted:
		return "agent_started"
	case ActivityAgentStopped:
		return "agent_stopped"
	case ActivityTaskStarted:
		return "task_started"
	case ActivityTaskCompleted:
		return "task_completed"
	case ActivityTaskFailed:
		return "task_failed"
	case ActivityToolStarted:
		return "tool_started"
	case ActivityToolCompleted:
		return "tool_completed"
	case ActivityToolFailed:
		return "tool_failed"
	case ActivityStateChanged:
		return "state_changed"
	case ActivityCoordination:
		return "coordination"
	default:
		return "unknown"
	}
}

// AgentStatusUpdate represents an update to agent status
type AgentStatusUpdate struct {
	AgentID string
	Status  *AgentStatus
	Event   *ActivityEvent
}

// NewAgentStatusTracker creates a new agent status tracker
func NewAgentStatusTracker(guildConfig *config.GuildConfig) *AgentStatusTracker {
	tracker := &AgentStatusTracker{
		agents:         make(map[string]*AgentStatus),
		activeTools:    make(map[string]*ToolStatus),
		globalActivity: make([]ActivityEvent, 0),
		updateChannel:  make(chan AgentStatusUpdate, 100), // Buffered channel
		guildConfig:    guildConfig,
		maxActivity:    100, // Keep last 100 activity events
	}

	// Initialize agent statuses from guild config
	if guildConfig != nil {
		for _, agentConfig := range guildConfig.Agents {
			tracker.initializeAgent(agentConfig)
		}
	}

	return tracker
}

// initializeAgent creates initial status for an agent from config
func (t *AgentStatusTracker) initializeAgent(agentConfig config.AgentConfig) {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	status := &AgentStatus{
		ID:           agentConfig.ID,
		Name:         agentConfig.Name,
		Type:         agentConfig.Type,
		State:        AgentOffline, // Start offline until activated
		CurrentTask:  "",
		Progress:     0.0,
		LastActivity: time.Now(),
		ActiveTools:  []string{},
		Capabilities: agentConfig.Capabilities,
		CostMagnitude: determineCostMagnitude(agentConfig.Provider),
		TotalCost:    0.0,
		TasksCompleted: 0,
		StartTime:    time.Now(),
	}

	t.agents[agentConfig.ID] = status
}

// determineCostMagnitude assigns cost tier based on provider
func determineCostMagnitude(provider string) int {
	switch provider {
	case "ollama":
		return 1 // Very low cost (local)
	case "deepseek":
		return 2 // Low cost
	case "openai":
		return 4 // High cost
	case "anthropic":
		return 5 // Very high cost
	case "claude-code":
		return 3 // Medium cost
	default:
		return 3 // Default medium cost
	}
}

// UpdateAgentStatus updates the status of a specific agent
func (t *AgentStatusTracker) UpdateAgentStatus(agentID string, status *AgentStatus) {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	// Update the agent status
	status.LastActivity = time.Now()
	t.agents[agentID] = status

	// Create activity event for state changes
	if existingStatus, exists := t.agents[agentID]; exists {
		if existingStatus.State != status.State {
			event := ActivityEvent{
				Timestamp:   time.Now(),
				AgentID:     agentID,
				EventType:   ActivityStateChanged,
				Description: "Agent state changed from " + existingStatus.State.String() + " to " + status.State.String(),
				Metadata: map[string]interface{}{
					"old_state": existingStatus.State.String(),
					"new_state": status.State.String(),
				},
			}
			t.addActivityEvent(event)
		}
	}

	// Send update through channel (non-blocking)
	select {
	case t.updateChannel <- AgentStatusUpdate{AgentID: agentID, Status: status}:
	default:
		// Channel full, skip this update
	}
}

// GetAgentStatus retrieves the status of a specific agent
func (t *AgentStatusTracker) GetAgentStatus(agentID string) *AgentStatus {
	t.mutex.RLock()
	defer t.mutex.RUnlock()

	if status, exists := t.agents[agentID]; exists {
		// Return a copy to prevent external modification
		statusCopy := *status
		return &statusCopy
	}
	return nil
}

// GetActiveAgents returns all currently active agents (not offline)
func (t *AgentStatusTracker) GetActiveAgents() []*AgentStatus {
	t.mutex.RLock()
	defer t.mutex.RUnlock()

	var activeAgents []*AgentStatus
	for _, status := range t.agents {
		if status.State != AgentOffline {
			// Return a copy to prevent external modification
			statusCopy := *status
			activeAgents = append(activeAgents, &statusCopy)
		}
	}
	return activeAgents
}

// GetAllAgents returns all agents regardless of state
func (t *AgentStatusTracker) GetAllAgents() []*AgentStatus {
	t.mutex.RLock()
	defer t.mutex.RUnlock()

	var allAgents []*AgentStatus
	for _, status := range t.agents {
		// Return a copy to prevent external modification
		statusCopy := *status
		allAgents = append(allAgents, &statusCopy)
	}
	return allAgents
}

// GetRecentActivity returns recent activity events up to the specified limit
func (t *AgentStatusTracker) GetRecentActivity(limit int) []ActivityEvent {
	t.mutex.RLock()
	defer t.mutex.RUnlock()

	if limit <= 0 || limit > len(t.globalActivity) {
		limit = len(t.globalActivity)
	}

	// Return the most recent events (from the end of the slice)
	start := len(t.globalActivity) - limit
	if start < 0 {
		start = 0
	}

	events := make([]ActivityEvent, limit)
	copy(events, t.globalActivity[start:])
	return events
}

// UpdateToolStatus updates the status of a tool execution
func (t *AgentStatusTracker) UpdateToolStatus(toolID string, status *ToolStatus) {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	t.activeTools[toolID] = status

	// Create activity event for tool state changes
	var eventType ActivityEventType
	switch status.State {
	case ToolStarting:
		eventType = ActivityToolStarted
	case ToolCompleted:
		eventType = ActivityToolCompleted
	case ToolFailed:
		eventType = ActivityToolFailed
	default:
		return // Don't log every intermediate state
	}

	event := ActivityEvent{
		Timestamp:   time.Now(),
		AgentID:     status.AgentID,
		EventType:   eventType,
		Description: "Tool " + status.Name + " " + status.State.String(),
		Metadata: map[string]interface{}{
			"tool_id":   toolID,
			"tool_name": status.Name,
			"state":     status.State.String(),
			"progress":  status.Progress,
		},
	}
	t.addActivityEvent(event)

	// Update agent's active tools list
	if agentStatus, exists := t.agents[status.AgentID]; exists {
		if status.State == ToolStarting || status.State == ToolRunning {
			// Add to active tools if not already present
			found := false
			for _, tool := range agentStatus.ActiveTools {
				if tool == toolID {
					found = true
					break
				}
			}
			if !found {
				agentStatus.ActiveTools = append(agentStatus.ActiveTools, toolID)
			}
		} else {
			// Remove from active tools
			for i, tool := range agentStatus.ActiveTools {
				if tool == toolID {
					agentStatus.ActiveTools = append(agentStatus.ActiveTools[:i], agentStatus.ActiveTools[i+1:]...)
					break
				}
			}
		}
	}
}

// GetToolStatus retrieves the status of a specific tool execution
func (t *AgentStatusTracker) GetToolStatus(toolID string) *ToolStatus {
	t.mutex.RLock()
	defer t.mutex.RUnlock()

	if status, exists := t.activeTools[toolID]; exists {
		// Return a copy to prevent external modification
		statusCopy := *status
		return &statusCopy
	}
	return nil
}

// StartTaskTracking begins tracking a task for an agent
func (t *AgentStatusTracker) StartTaskTracking(agentID, taskDescription string) {
	if status := t.GetAgentStatus(agentID); status != nil {
		status.CurrentTask = taskDescription
		status.Progress = 0.0
		status.State = AgentWorking
		t.UpdateAgentStatus(agentID, status)

		// Log activity event
		event := ActivityEvent{
			Timestamp:   time.Now(),
			AgentID:     agentID,
			EventType:   ActivityTaskStarted,
			Description: "Started task: " + taskDescription,
			Metadata: map[string]interface{}{
				"task": taskDescription,
			},
		}
		t.logActivityEvent(event)
	}
}

// CompleteTaskTracking marks a task as completed for an agent
func (t *AgentStatusTracker) CompleteTaskTracking(agentID string, result string) {
	if status := t.GetAgentStatus(agentID); status != nil {
		status.Progress = 1.0
		status.State = AgentIdle
		status.TasksCompleted++
		previousTask := status.CurrentTask
		status.CurrentTask = ""
		t.UpdateAgentStatus(agentID, status)

		// Log activity event
		event := ActivityEvent{
			Timestamp:   time.Now(),
			AgentID:     agentID,
			EventType:   ActivityTaskCompleted,
			Description: "Completed task: " + previousTask,
			Metadata: map[string]interface{}{
				"task":   previousTask,
				"result": result,
			},
		}
		t.logActivityEvent(event)
	}
}

// LogCoordinationEvent logs a multi-agent coordination event
func (t *AgentStatusTracker) LogCoordinationEvent(description string, involvedAgents []string, metadata map[string]interface{}) {
	event := ActivityEvent{
		Timestamp:   time.Now(),
		AgentID:     "system", // System-level coordination
		EventType:   ActivityCoordination,
		Description: description,
		Metadata:    metadata,
	}

	if event.Metadata == nil {
		event.Metadata = make(map[string]interface{})
	}
	event.Metadata["involved_agents"] = involvedAgents

	t.logActivityEvent(event)
}

// StartTracking begins monitoring agent activity
func (t *AgentStatusTracker) StartTracking() {
	// Start a goroutine to process status updates
	go t.processUpdates()
}

// StopTracking stops monitoring and cleans up resources
func (t *AgentStatusTracker) StopTracking() {
	close(t.updateChannel)
}

// processUpdates handles status updates from the channel
func (t *AgentStatusTracker) processUpdates() {
	for update := range t.updateChannel {
		// Process the update (could trigger additional logic here)
		if update.Event != nil {
			t.logActivityEvent(*update.Event)
		}
	}
}

// logActivityEvent adds an activity event to the log (thread-safe public method)
func (t *AgentStatusTracker) logActivityEvent(event ActivityEvent) {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	t.addActivityEvent(event)
}

// addActivityEvent adds an activity event to the log (internal method, assumes lock held)
func (t *AgentStatusTracker) addActivityEvent(event ActivityEvent) {
	t.globalActivity = append(t.globalActivity, event)

	// Trim activity log if it exceeds maximum size
	if len(t.globalActivity) > t.maxActivity {
		// Keep the most recent events
		copy(t.globalActivity, t.globalActivity[len(t.globalActivity)-t.maxActivity:])
		t.globalActivity = t.globalActivity[:t.maxActivity]
	}
}

// GetCoordinationSummary returns a summary of multi-agent coordination activity
func (t *AgentStatusTracker) GetCoordinationSummary() map[string]interface{} {
	t.mutex.RLock()
	defer t.mutex.RUnlock()

	summary := map[string]interface{}{
		"total_agents":        len(t.agents),
		"active_agents":       0,
		"active_tools":        len(t.activeTools),
		"total_cost":          0.0,
		"total_tasks":         0,
		"coordination_events": 0,
	}

	for _, status := range t.agents {
		if status.State != AgentOffline {
			summary["active_agents"] = summary["active_agents"].(int) + 1
		}
		summary["total_cost"] = summary["total_cost"].(float64) + status.TotalCost
		summary["total_tasks"] = summary["total_tasks"].(int) + status.TasksCompleted
	}

	// Count coordination events
	for _, event := range t.globalActivity {
		if event.EventType == ActivityCoordination {
			summary["coordination_events"] = summary["coordination_events"].(int) + 1
		}
	}

	return summary
}