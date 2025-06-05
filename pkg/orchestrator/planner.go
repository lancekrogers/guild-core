package orchestrator

import (
	"context"
	"fmt"
	"strings"

	"github.com/guild-ventures/guild-core/pkg/agent"
	"github.com/guild-ventures/guild-core/pkg/config"
	"github.com/guild-ventures/guild-core/pkg/kanban"
	"github.com/guild-ventures/guild-core/pkg/commission"
)

// TaskPlanner decomposes objectives into tasks and assigns them to agents
type TaskPlanner interface {
	// PlanTasks decomposes an objective into tasks
	PlanTasks(ctx context.Context, obj *objective.Objective, guild *config.GuildConfig) ([]*kanban.Task, error)
	
	// AssignTasks assigns tasks to agents based on capabilities
	AssignTasks(ctx context.Context, tasks []*kanban.Task, guild *config.GuildConfig) error
}

// ManagerTaskPlanner uses the manager agent to plan tasks
type ManagerTaskPlanner struct {
	managerAgent agent.Agent
	kanbanBoard  *kanban.Board
}

// NewManagerTaskPlanner creates a new manager-based task planner
func NewManagerTaskPlanner(managerAgent agent.Agent, kanbanBoard *kanban.Board) *ManagerTaskPlanner {
	return &ManagerTaskPlanner{
		managerAgent: managerAgent,
		kanbanBoard:  kanbanBoard,
	}
}

// PlanTasks uses the manager agent to decompose an objective into tasks
func (p *ManagerTaskPlanner) PlanTasks(ctx context.Context, obj *objective.Objective, guild *config.GuildConfig) ([]*kanban.Task, error) {
	// Build a prompt for the manager agent
	prompt := p.buildPlanningPrompt(obj, guild)
	
	// Execute the planning request
	response, err := p.managerAgent.Execute(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("manager agent failed to plan tasks: %w", err)
	}
	
	// Parse the response into tasks
	tasks, err := p.parseTasksFromResponse(response)
	if err != nil {
		return nil, fmt.Errorf("failed to parse tasks from response: %w", err)
	}
	
	// Add tasks to kanban board
	for _, task := range tasks {
		// Create task on board
		createdTask, err := p.kanbanBoard.CreateTask(ctx, task.Title, task.Description)
		if err != nil {
			return nil, fmt.Errorf("failed to create task on kanban board: %w", err)
		}
		
		// Update task with parsed data
		createdTask.Status = task.Status
		createdTask.Metadata = task.Metadata
		createdTask.Dependencies = task.Dependencies
		
		// Update the task
		if err := p.kanbanBoard.UpdateTask(ctx, createdTask); err != nil {
			return nil, fmt.Errorf("failed to update task: %w", err)
		}
		
		// Update our reference
		task.ID = createdTask.ID
	}
	
	return tasks, nil
}

// AssignTasks assigns tasks to agents based on their capabilities
func (p *ManagerTaskPlanner) AssignTasks(ctx context.Context, tasks []*kanban.Task, guild *config.GuildConfig) error {
	// Build a prompt for task assignment
	prompt := p.buildAssignmentPrompt(tasks, guild)
	
	// Execute the assignment request
	response, err := p.managerAgent.Execute(ctx, prompt)
	if err != nil {
		return fmt.Errorf("manager agent failed to assign tasks: %w", err)
	}
	
	// Parse assignments from response
	assignments, err := p.parseAssignmentsFromResponse(response, tasks, guild)
	if err != nil {
		return fmt.Errorf("failed to parse assignments: %w", err)
	}
	
	// Apply assignments to tasks
	for taskID, agentID := range assignments {
		task, err := p.kanbanBoard.GetTask(ctx, taskID)
		if err != nil {
			continue // Skip if task not found
		}
		
		// Update task assignment
		task.AssignedTo = agentID
		task.Metadata["assigned_to"] = agentID
		if err := p.kanbanBoard.UpdateTask(ctx, task); err != nil {
			return fmt.Errorf("failed to update task assignment: %w", err)
		}
	}
	
	return nil
}

// buildPlanningPrompt creates a prompt for task planning
func (p *ManagerTaskPlanner) buildPlanningPrompt(obj *objective.Objective, guild *config.GuildConfig) string {
	var prompt strings.Builder
	
	prompt.WriteString("You are the manager agent for the ")
	prompt.WriteString(guild.Name)
	prompt.WriteString(" guild. Your task is to decompose the following objective into concrete, actionable tasks.\n\n")
	
	prompt.WriteString("## Objective\n")
	prompt.WriteString(obj.Format())
	prompt.WriteString("\n\n")
	
	prompt.WriteString("## Available Agents and Capabilities\n")
	for _, agent := range guild.Agents {
		prompt.WriteString(fmt.Sprintf("- **%s** (%s): %s\n", agent.Name, agent.ID, strings.Join(agent.Capabilities, ", ")))
	}
	prompt.WriteString("\n")
	
	prompt.WriteString("## Instructions\n")
	prompt.WriteString("Break down the objective into specific tasks. For each task, provide:\n")
	prompt.WriteString("1. A unique task ID (e.g., TASK-001)\n")
	prompt.WriteString("2. A clear, concise title\n")
	prompt.WriteString("3. A detailed description\n")
	prompt.WriteString("4. Required capabilities (from the list above)\n")
	prompt.WriteString("5. Dependencies on other tasks (if any)\n")
	prompt.WriteString("6. Estimated complexity (low, medium, high)\n\n")
	
	prompt.WriteString("Format your response as follows:\n")
	prompt.WriteString("```\n")
	prompt.WriteString("TASK-001: [Title]\n")
	prompt.WriteString("Description: [Detailed description]\n")
	prompt.WriteString("Capabilities: [capability1, capability2]\n")
	prompt.WriteString("Dependencies: [TASK-XXX, TASK-YYY] or none\n")
	prompt.WriteString("Complexity: [low|medium|high]\n")
	prompt.WriteString("---\n")
	prompt.WriteString("```\n")
	
	return prompt.String()
}

// buildAssignmentPrompt creates a prompt for task assignment
func (p *ManagerTaskPlanner) buildAssignmentPrompt(tasks []*kanban.Task, guild *config.GuildConfig) string {
	var prompt strings.Builder
	
	prompt.WriteString("You are the manager agent. Assign the following tasks to the most suitable agents based on their capabilities.\n\n")
	
	prompt.WriteString("## Tasks to Assign\n")
	for _, task := range tasks {
		capabilities := ""
		if caps, ok := task.Metadata["capabilities"]; ok {
			capabilities = caps
		}
		prompt.WriteString(fmt.Sprintf("- **%s**: %s (requires: %s)\n", 
			task.ID, task.Title, capabilities))
	}
	prompt.WriteString("\n")
	
	prompt.WriteString("## Available Agents\n")
	for _, agent := range guild.Agents {
		prompt.WriteString(fmt.Sprintf("- **%s** (%s): capabilities: %s\n", 
			agent.Name, agent.ID, strings.Join(agent.Capabilities, ", ")))
	}
	prompt.WriteString("\n")
	
	prompt.WriteString("## Instructions\n")
	prompt.WriteString("Assign each task to the most suitable agent. Consider:\n")
	prompt.WriteString("1. Agent capabilities must match task requirements\n")
	prompt.WriteString("2. Balance workload across agents\n")
	prompt.WriteString("3. Prefer specialists for their domain\n\n")
	
	prompt.WriteString("Format your response as:\n")
	prompt.WriteString("```\n")
	prompt.WriteString("TASK-001: agent_id\n")
	prompt.WriteString("TASK-002: agent_id\n")
	prompt.WriteString("```\n")
	
	return prompt.String()
}

// parseTasksFromResponse parses tasks from the manager's response
func (p *ManagerTaskPlanner) parseTasksFromResponse(response string) ([]*kanban.Task, error) {
	tasks := []*kanban.Task{}
	
	// Simple parsing - in production, use a more robust parser
	lines := strings.Split(response, "\n")
	var currentTask *kanban.Task
	
	for _, line := range lines {
		line = strings.TrimSpace(line)
		
		// Skip empty lines and markers
		if line == "" || line == "```" || line == "---" {
			if currentTask != nil {
				tasks = append(tasks, currentTask)
				currentTask = nil
			}
			continue
		}
		
		// Parse task ID and title
		if strings.Contains(line, ":") && strings.HasPrefix(line, "TASK-") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				currentTask = &kanban.Task{
					ID:       strings.TrimSpace(parts[0]),
					Title:    strings.TrimSpace(parts[1]),
					Status:   kanban.StatusTodo,
					Metadata: make(map[string]string),
				}
			}
		} else if currentTask != nil {
			// Parse task properties
			if strings.HasPrefix(line, "Description:") {
				currentTask.Description = strings.TrimSpace(strings.TrimPrefix(line, "Description:"))
			} else if strings.HasPrefix(line, "Capabilities:") {
				capsStr := strings.TrimSpace(strings.TrimPrefix(line, "Capabilities:"))
				currentTask.Metadata["capabilities"] = capsStr
			} else if strings.HasPrefix(line, "Dependencies:") {
				depsStr := strings.TrimSpace(strings.TrimPrefix(line, "Dependencies:"))
				if depsStr != "none" {
					currentTask.Dependencies = strings.Split(depsStr, ",")
					for i := range currentTask.Dependencies {
						currentTask.Dependencies[i] = strings.TrimSpace(currentTask.Dependencies[i])
					}
				}
			} else if strings.HasPrefix(line, "Complexity:") {
				currentTask.Metadata["complexity"] = strings.TrimSpace(strings.TrimPrefix(line, "Complexity:"))
			}
		}
	}
	
	// Add last task if any
	if currentTask != nil {
		tasks = append(tasks, currentTask)
	}
	
	if len(tasks) == 0 {
		return nil, fmt.Errorf("no tasks found in response")
	}
	
	return tasks, nil
}

// parseAssignmentsFromResponse parses task assignments from the manager's response
func (p *ManagerTaskPlanner) parseAssignmentsFromResponse(response string, tasks []*kanban.Task, guild *config.GuildConfig) (map[string]string, error) {
	assignments := make(map[string]string)
	
	// Parse response
	lines := strings.Split(response, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || line == "```" {
			continue
		}
		
		// Parse "TASK-XXX: agent_id"
		if strings.Contains(line, ":") && strings.HasPrefix(line, "TASK-") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				taskID := strings.TrimSpace(parts[0])
				agentID := strings.TrimSpace(parts[1])
				
				// Validate agent exists
				if _, err := guild.GetAgentByID(agentID); err == nil {
					assignments[taskID] = agentID
				}
			}
		}
	}
	
	return assignments, nil
}