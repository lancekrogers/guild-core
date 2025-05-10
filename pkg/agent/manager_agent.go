package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/blockhead-consulting/Guild/pkg/kanban"
	"github.com/blockhead-consulting/Guild/pkg/memory"
	"github.com/blockhead-consulting/Guild/pkg/objective"
	"github.com/blockhead-consulting/Guild/pkg/providers"
	"github.com/blockhead-consulting/Guild/tools"
)

const (
	// Default prompt template for manager agents
	managerAgentPrompt = `You are {{.AgentName}}, a manager agent responsible for coordinating other worker agents.

Your current objective is:
{{.ObjectiveTitle}}
{{.ObjectiveDescription}}

Current task board status:
{{.BoardStatus}}

Available worker agents:
{{.WorkerAgents}}

You can:
1. Create new tasks for worker agents
2. Assign existing tasks to specific worker agents
3. Check the status of tasks and objectives
4. Provide feedback and guidance to worker agents

When creating tasks, be specific and actionable. Break down complex objectives into manageable tasks.

To perform an action, respond with a JSON message in this format:
{
  "thoughts": "your step-by-step reasoning about what to do next",
  "action": {
    "type": "create_task | assign_task | check_status | provide_feedback",
    "parameters": {
      // action-specific parameters
    }
  }
}

If you believe the objective is complete, respond with:
{
  "thoughts": "why you believe the objective is complete",
  "final_report": "comprehensive summary of what was accomplished"
}

Think strategically and plan tasks effectively to achieve the objective.
`
)

// GuildMaster implements a master artisan that coordinates other craftsmen
type GuildMaster struct {
	*GuildMember
	kanbanManager *kanban.Manager
	workerAgents  map[string]GuildArtisan // Map of craftsman IDs to artisans
}

// NewGuildMaster creates a new guild master artisan
func NewGuildMaster(
	config *AgentConfig,
	llmClient providers.LLMClient,
	memoryManager memory.ChainManager,
	toolRegistry *tools.ToolRegistry,
	objectiveMgr *objective.Manager,
	kanbanManager *kanban.Manager,
) *GuildMaster {
	member := NewGuildMember(config, llmClient, memoryManager, toolRegistry, objectiveMgr)

	return &GuildMaster{
		GuildMember:   member,
		kanbanManager: kanbanManager,
		workerAgents:  make(map[string]GuildArtisan),
	}
}

// RegisterCraftsman registers a craftsman artisan with the guild master
func (a *GuildMaster) RegisterCraftsman(artisan GuildArtisan) {
	a.workerAgents[artisan.ID()] = artisan
}

// CraftSolution runs the guild master's execution cycle
func (a *GuildMaster) CraftSolution(ctx context.Context) error {
	// Check if the agent has a task
	if a.currentTask == nil {
		return fmt.Errorf("no task assigned")
	}
	
	// Update status
	a.state.Status = StatusWorking
	a.state.UpdatedAt = time.Now().UTC()
	
	// Create a memory chain for this execution if needed
	var chainID string
	var err error
	
	if len(a.state.Memory) == 0 {
		// No existing memory chain, create one
		chainID, err = a.memoryManager.CreateChain(ctx, a.config.ID, a.currentTask.ID)
		if err != nil {
			return fmt.Errorf("failed to create memory chain: %w", err)
		}
		a.state.Memory = append(a.state.Memory, chainID)
	} else {
		// Use the last memory chain
		chainID = a.state.Memory[len(a.state.Memory)-1]
	}
	
	// Extract objective ID from task metadata
	objectiveID := ""
	if id, ok := a.currentTask.Metadata["objective_id"]; ok {
		objectiveID = id
	}
	
	// Build prompt context
	boardStatus, err := a.getBoardStatus(ctx)
	if err != nil {
		return fmt.Errorf("failed to get board status: %w", err)
	}
	
	workerAgentInfo := a.getWorkerAgentInfo()
	
	var objectiveTitle, objectiveDescription string
	if objectiveID != "" {
		obj, err := a.objectiveMgr.GetObjective(ctx, objectiveID)
		if err == nil {
			objectiveTitle = obj.Title
			objectiveDescription = obj.Description
		}
	} else {
		objectiveTitle = a.currentTask.Title
		objectiveDescription = a.currentTask.Description
	}
	
	// Build the prompt
	prompt := managerAgentPrompt
	prompt = strings.Replace(prompt, "{{.AgentName}}", a.config.Name, -1)
	prompt = strings.Replace(prompt, "{{.ObjectiveTitle}}", objectiveTitle, -1)
	prompt = strings.Replace(prompt, "{{.ObjectiveDescription}}", objectiveDescription, -1)
	prompt = strings.Replace(prompt, "{{.BoardStatus}}", boardStatus, -1)
	prompt = strings.Replace(prompt, "{{.WorkerAgents}}", workerAgentInfo, -1)
	
	// Add the prompt to the memory chain
	err = a.memoryManager.AddMessage(ctx, chainID, memory.Message{
		Role:      "system",
		Content:   prompt,
		Timestamp: time.Now().UTC(),
	})
	if err != nil {
		return fmt.Errorf("failed to add prompt to memory: %w", err)
	}
	
	// Execute the agent loop
	return a.executeLoop(ctx, chainID, objectiveID)
}

// getBoardStatus gets the current status of the kanban board
func (a *ManagerAgent) getBoardStatus(ctx context.Context) (string, error) {
	var statusBuilder strings.Builder
	
	// Get all tasks by status
	statuses := []kanban.TaskStatus{
		kanban.StatusTodo,
		kanban.StatusInProgress,
		kanban.StatusBlocked,
		kanban.StatusDone,
	}
	
	for _, status := range statuses {
		tasks, err := a.kanbanManager.ListTasksByStatus(ctx, status)
		if err != nil {
			continue
		}
		
		statusBuilder.WriteString(fmt.Sprintf("### %s Tasks (%d):\n", status, len(tasks)))
		
		for _, task := range tasks {
			assigneeInfo := ""
			if task.AssignedTo != "" {
				assigneeInfo = fmt.Sprintf(" [Assigned to: %s]", task.AssignedTo)
			}
			statusBuilder.WriteString(fmt.Sprintf("- %s%s\n", task.Title, assigneeInfo))
		}
		
		statusBuilder.WriteString("\n")
	}
	
	return statusBuilder.String(), nil
}

// getWorkerAgentInfo gets information about available worker agents
func (a *ManagerAgent) getWorkerAgentInfo() string {
	var infoBuilder strings.Builder
	
	for id, agent := range a.workerAgents {
		status := string(agent.Status())
		currentTask := ""
		
		if state := agent.GetState(); state != nil && state.CurrentTask != "" {
			currentTask = fmt.Sprintf(" - Working on task: %s", state.CurrentTask)
		}
		
		infoBuilder.WriteString(fmt.Sprintf("- %s (%s) - Status: %s%s\n", agent.Name(), id, status, currentTask))
	}
	
	return infoBuilder.String()
}

// ManagerResponse represents a structured response from the manager agent
type ManagerResponse struct {
	Thoughts    string                 `json:"thoughts"`
	Action      *ManagerAction         `json:"action,omitempty"`
	FinalReport string                 `json:"final_report,omitempty"`
}

// ManagerAction represents an action the manager wants to take
type ManagerAction struct {
	Type       string                 `json:"type"`
	Parameters map[string]interface{} `json:"parameters"`
}

// executeLoop executes the manager agent's main loop
func (a *ManagerAgent) executeLoop(ctx context.Context, chainID, objectiveID string) error {
	for {
		// Check if the context is cancelled
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			// Continue execution
		}
		
		// Get the most recent messages as context
		messages, err := a.memoryManager.BuildContext(ctx, a.config.ID, a.currentTask.ID, a.config.MaxTokens/2)
		if err != nil {
			return fmt.Errorf("failed to build context: %w", err)
		}
		
		// Create a completion request
		req := &providers.CompletionRequest{
			Prompt:      buildPromptFromMessages(messages),
			MaxTokens:   a.config.MaxTokens,
			Temperature: a.config.Temperature,
		}
		
		// Call the LLM
		resp, err := a.llmClient.Complete(ctx, req)
		if err != nil {
			// Log error and continue
			a.state.LastError = err.Error()
			a.SaveState(ctx)
			continue
		}
		
		// Add the response to memory
		err = a.memoryManager.AddMessage(ctx, chainID, memory.Message{
			Role:       "assistant",
			Content:    resp.Text,
			Timestamp:  time.Now().UTC(),
			TokenUsage: resp.TokensUsed,
		})
		if err != nil {
			return fmt.Errorf("failed to add response to memory: %w", err)
		}
		
		// Parse the response
		managerResp, err := parseManagerResponse(resp.Text)
		if err != nil {
			// Add error message to memory
			errorMsg := fmt.Sprintf("Failed to parse response: %v\nPlease respond with valid JSON in the required format.", err)
			a.memoryManager.AddMessage(ctx, chainID, memory.Message{
				Role:      "system",
				Content:   errorMsg,
				Timestamp: time.Now().UTC(),
			})
			continue
		}
		
		// Check if the manager has a final report
		if managerResp.FinalReport != "" {
			// Objective completed
			a.completeObjective(ctx, objectiveID, managerResp.FinalReport)
			return nil
		}
		
		// Handle manager action
		if managerResp.Action != nil {
			result, err := a.handleManagerAction(ctx, managerResp.Action, objectiveID)
			if err != nil {
				// Add error message to memory
				errorMsg := fmt.Sprintf("Error executing action: %v", err)
				a.memoryManager.AddMessage(ctx, chainID, memory.Message{
					Role:      "system",
					Content:   errorMsg,
					Timestamp: time.Now().UTC(),
				})
				continue
			}
			
			// Add action result to memory
			a.memoryManager.AddMessage(ctx, chainID, memory.Message{
				Role:      "system",
				Content:   result,
				Timestamp: time.Now().UTC(),
			})
		}
		
		// Sleep briefly to avoid overwhelming the system
		time.Sleep(2 * time.Second)
	}
}

// handleManagerAction handles a manager action
func (a *ManagerAgent) handleManagerAction(ctx context.Context, action *ManagerAction, objectiveID string) (string, error) {
	switch action.Type {
	case "create_task":
		return a.handleCreateTask(ctx, action.Parameters, objectiveID)
	case "assign_task":
		return a.handleAssignTask(ctx, action.Parameters)
	case "check_status":
		return a.handleCheckStatus(ctx, action.Parameters)
	case "provide_feedback":
		return a.handleProvideFeedback(ctx, action.Parameters)
	default:
		return "", fmt.Errorf("unknown action type: %s", action.Type)
	}
}

// handleCreateTask handles the create_task action
func (a *ManagerAgent) handleCreateTask(ctx context.Context, params map[string]interface{}, objectiveID string) (string, error) {
	// Extract parameters
	title, ok := params["title"].(string)
	if !ok || title == "" {
		return "", fmt.Errorf("task title is required")
	}
	
	description, _ := params["description"].(string)
	priority, _ := params["priority"].(string)
	
	// Get the board from params or use a default board
	boardID, _ := params["board_id"].(string)
	if boardID == "" {
		// List boards and use the first one
		boards, err := a.kanbanManager.ListBoards(ctx)
		if err != nil || len(boards) == 0 {
			return "", fmt.Errorf("no kanban boards available")
		}
		boardID = boards[0].ID
	}
	
	board, err := a.kanbanManager.GetBoard(ctx, boardID)
	if err != nil {
		return "", fmt.Errorf("board not found: %w", err)
	}
	
	// Create the task
	task, err := board.CreateTask(ctx, title, description)
	if err != nil {
		return "", fmt.Errorf("failed to create task: %w", err)
	}
	
	// Set task properties
	if priority != "" {
		task.Priority = kanban.TaskPriority(priority)
	}
	
	// Add objective ID to metadata if provided
	if objectiveID != "" {
		if task.Metadata == nil {
			task.Metadata = make(map[string]string)
		}
		task.Metadata["objective_id"] = objectiveID
	}
	
	// Update the task
	if err := board.UpdateTask(ctx, task); err != nil {
		return "", fmt.Errorf("failed to update task: %w", err)
	}
	
	return fmt.Sprintf("Task created successfully: %s (ID: %s)", task.Title, task.ID), nil
}

// handleAssignTask handles the assign_task action
func (a *ManagerAgent) handleAssignTask(ctx context.Context, params map[string]interface{}) (string, error) {
	// Extract parameters
	taskID, ok := params["task_id"].(string)
	if !ok || taskID == "" {
		return "", fmt.Errorf("task ID is required")
	}
	
	agentID, ok := params["agent_id"].(string)
	if !ok || agentID == "" {
		return "", fmt.Errorf("agent ID is required")
	}
	
	// Check if the agent exists
	agent, exists := a.workerAgents[agentID]
	if !exists {
		return "", fmt.Errorf("agent not found: %s", agentID)
	}
	
	// Check if the agent is available
	if agent.Status() != StatusIdle {
		return "", fmt.Errorf("agent %s is not available (status: %s)", agentID, agent.Status())
	}
	
	// Get the task
	task, err := a.kanbanManager.GetTask(ctx, taskID)
	if err != nil {
		return "", fmt.Errorf("task not found: %w", err)
	}
	
	// Assign the task to the agent
	if err := a.kanbanManager.AssignTask(ctx, taskID, agentID, a.ID(), "Assigned by manager agent"); err != nil {
		return "", fmt.Errorf("failed to assign task: %w", err)
	}
	
	// Assign the task to the agent
	if err := agent.AssignTask(ctx, task); err != nil {
		return "", fmt.Errorf("failed to assign task to agent: %w", err)
	}
	
	return fmt.Sprintf("Task '%s' assigned to agent '%s'", task.Title, agent.Name()), nil
}

// handleCheckStatus handles the check_status action
func (a *ManagerAgent) handleCheckStatus(ctx context.Context, params map[string]interface{}) (string, error) {
	// Check if we're checking status of a specific task
	if taskID, ok := params["task_id"].(string); ok && taskID != "" {
		task, err := a.kanbanManager.GetTask(ctx, taskID)
		if err != nil {
			return "", fmt.Errorf("task not found: %w", err)
		}
		
		// Format task status
		status := fmt.Sprintf("Task: %s\nStatus: %s\nPriority: %s\n", 
			task.Title, task.Status, task.Priority)
		
		if task.AssignedTo != "" {
			status += fmt.Sprintf("Assigned to: %s\n", task.AssignedTo)
		}
		
		if task.CompletedAt != nil {
			status += fmt.Sprintf("Completed at: %s\n", task.CompletedAt.Format(time.RFC3339))
		}
		
		return status, nil
	}
	
	// Check if we're checking status of a specific agent
	if agentID, ok := params["agent_id"].(string); ok && agentID != "" {
		agent, exists := a.workerAgents[agentID]
		if !exists {
			return "", fmt.Errorf("agent not found: %s", agentID)
		}
		
		state := agent.GetState()
		status := fmt.Sprintf("Agent: %s\nStatus: %s\n", agent.Name(), agent.Status())
		
		if state.CurrentTask != "" {
			task, err := a.kanbanManager.GetTask(ctx, state.CurrentTask)
			if err == nil {
				status += fmt.Sprintf("Working on: %s\n", task.Title)
			} else {
				status += fmt.Sprintf("Working on task ID: %s\n", state.CurrentTask)
			}
		}
		
		return status, nil
	}
	
	// If no specific task or agent, return overall status
	boardStatus, err := a.getBoardStatus(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get board status: %w", err)
	}
	
	return boardStatus, nil
}

// handleProvideFeedback handles the provide_feedback action
func (a *ManagerAgent) handleProvideFeedback(ctx context.Context, params map[string]interface{}) (string, error) {
	// Extract parameters
	agentID, ok := params["agent_id"].(string)
	if !ok || agentID == "" {
		return "", fmt.Errorf("agent ID is required")
	}
	
	feedback, ok := params["feedback"].(string)
	if !ok || feedback == "" {
		return "", fmt.Errorf("feedback is required")
	}
	
	// Check if the agent exists
	agent, exists := a.workerAgents[agentID]
	if !exists {
		return "", fmt.Errorf("agent not found: %s", agentID)
	}
	
	// Get the agent's current task
	state := agent.GetState()
	if state.CurrentTask == "" {
		return "", fmt.Errorf("agent %s is not working on a task", agentID)
	}
	
	// Get the memory manager for the agent
	memoryMgr := agent.GetMemoryManager()
	
	// Get the latest memory chain
	chains, err := memoryMgr.GetChainsByAgent(ctx, agentID)
	if err != nil || len(chains) == 0 {
		return "", fmt.Errorf("no memory chains found for agent")
	}
	
	// Add feedback to the agent's memory
	err = memoryMgr.AddMessage(ctx, chains[0].ID, memory.Message{
		Role:      "system",
		Content:   fmt.Sprintf("Feedback from manager: %s", feedback),
		Timestamp: time.Now().UTC(),
	})
	if err != nil {
		return "", fmt.Errorf("failed to add feedback to agent memory: %w", err)
	}
	
	return fmt.Sprintf("Feedback provided to agent '%s'", agent.Name()), nil
}

// completeObjective marks the objective as complete
func (a *ManagerAgent) completeObjective(ctx context.Context, objectiveID, summary string) {
	if objectiveID == "" {
		return
	}
	
	// Get the objective
	obj, err := a.objectiveMgr.GetObjective(ctx, objectiveID)
	if err != nil {
		return
	}
	
	// Update objective status
	obj.Status = objective.ObjectiveStatusCompleted
	now := time.Now().UTC()
	obj.CompletedAt = &now
	obj.UpdatedAt = now
	
	// Add summary to metadata
	if obj.Metadata == nil {
		obj.Metadata = make(map[string]string)
	}
	obj.Metadata["completion_summary"] = summary
	
	// Save the objective
	a.objectiveMgr.SaveObjective(ctx, obj)
	
	// Update agent state
	a.state.Status = StatusIdle
	a.state.CurrentTask = ""
	a.state.UpdatedAt = now
	
	// Save agent state
	a.SaveState(ctx)
}

// Stop stops the guild master's execution
func (a *GuildMaster) Stop(ctx context.Context) error {
	// Implementation depends on execution model
	// For simplicity, just update the state
	a.state.Status = StatusPaused
	a.state.UpdatedAt = time.Now().UTC()
	return a.SaveState(ctx)
}

// parseManagerResponse parses the manager agent's response
func parseManagerResponse(text string) (*ManagerResponse, error) {
	// Find the JSON block in the response
	jsonStart := strings.Index(text, "{")
	jsonEnd := strings.LastIndex(text, "}")
	
	if jsonStart == -1 || jsonEnd == -1 || jsonEnd < jsonStart {
		return nil, fmt.Errorf("no valid JSON found in response")
	}
	
	jsonStr := text[jsonStart : jsonEnd+1]
	
	var resp ManagerResponse
	if err := json.Unmarshal([]byte(jsonStr), &resp); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}
	
	return &resp, nil
}