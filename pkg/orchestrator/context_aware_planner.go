package orchestrator

import (
	"context"
	"fmt"
	"strings"

	"github.com/guild-ventures/guild-core/pkg/agent"
	"github.com/guild-ventures/guild-core/pkg/config"
	"github.com/guild-ventures/guild-core/pkg/kanban"
	"github.com/guild-ventures/guild-core/pkg/commission"
	"github.com/guild-ventures/guild-core/pkg/registry"
)

// ContextAwareTaskPlanner uses the manager agent's intelligence to assign tasks
// based on full context from agent configurations
type ContextAwareTaskPlanner struct {
	managerAgent      agent.Agent
	kanbanBoard       KanbanManager
	componentRegistry registry.ComponentRegistry
}

// NewContextAwareTaskPlanner creates a planner that uses LLM intelligence for assignments
func NewContextAwareTaskPlanner(
	managerAgent agent.Agent,
	kanbanBoard KanbanManager,
	componentRegistry registry.ComponentRegistry,
) *ContextAwareTaskPlanner {
	return &ContextAwareTaskPlanner{
		managerAgent:      managerAgent,
		kanbanBoard:       kanbanBoard,
		componentRegistry: componentRegistry,
	}
}

// PlanTasks decomposes an objective into tasks
func (p *ContextAwareTaskPlanner) PlanTasks(ctx context.Context, obj *objective.Objective, guild *config.GuildConfig) ([]*kanban.Task, error) {
	// Build a planning prompt with full agent context
	prompt := p.buildContextAwarePlanningPrompt(obj, guild)
	
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
		createdTask, err := p.kanbanBoard.CreateTask(ctx, task.Title, task.Description)
		if err != nil {
			return nil, fmt.Errorf("failed to create task on kanban board: %w", err)
		}
		
		createdTask.Status = task.Status
		createdTask.Metadata = task.Metadata
		createdTask.Dependencies = task.Dependencies
		
		if err := p.kanbanBoard.UpdateTask(ctx, createdTask); err != nil {
			return nil, fmt.Errorf("failed to update task: %w", err)
		}
		
		task.ID = createdTask.ID
	}
	
	return tasks, nil
}

// AssignTasks uses manager agent intelligence to assign tasks based on full context
func (p *ContextAwareTaskPlanner) AssignTasks(ctx context.Context, tasks []*kanban.Task, guild *config.GuildConfig) error {
	// Build assignment prompt with full agent configurations and context
	prompt := p.buildContextAwareAssignmentPrompt(tasks, guild)
	
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
	for taskID, assignment := range assignments {
		task, err := p.kanbanBoard.GetTask(ctx, taskID)
		if err != nil {
			continue
		}
		
		// Update task with assignment and reasoning
		task.AssignedTo = assignment.AgentID
		task.Metadata["assigned_to"] = assignment.AgentID
		task.Metadata["assignment_reason"] = assignment.Reason
		task.Metadata["assignment_confidence"] = assignment.Confidence
		
		if err := p.kanbanBoard.UpdateTask(ctx, task); err != nil {
			return fmt.Errorf("failed to update task assignment: %w", err)
		}
	}
	
	return nil
}

// buildContextAwarePlanningPrompt creates a prompt with full agent context
func (p *ContextAwareTaskPlanner) buildContextAwarePlanningPrompt(obj *objective.Objective, guild *config.GuildConfig) string {
	var prompt strings.Builder
	
	prompt.WriteString("You are the Guild Master (manager agent) for the ")
	prompt.WriteString(guild.Name)
	prompt.WriteString(" guild. Your task is to decompose the following objective into concrete, actionable tasks.\n\n")
	
	prompt.WriteString("## Objective\n")
	prompt.WriteString(obj.Format())
	prompt.WriteString("\n\n")
	
	prompt.WriteString("## Available Guild Members (Agents)\n\n")
	
	// Get all registered agents from the registry to include cost information
	registeredAgents := p.componentRegistry.Agents().(*registry.DefaultAgentRegistry).GetRegisteredAgents()
	agentMap := make(map[string]registry.GuildAgentConfig)
	for _, agent := range registeredAgents {
		agentMap[agent.ID] = agent
	}
	
	// Build detailed agent profiles from guild config
	for _, agent := range guild.Agents {
		prompt.WriteString(fmt.Sprintf("### %s (%s)\n", agent.Name, agent.ID))
		prompt.WriteString(fmt.Sprintf("- **Type**: %s\n", agent.Type))
		prompt.WriteString(fmt.Sprintf("- **Provider**: %s\n", agent.Provider))
		prompt.WriteString(fmt.Sprintf("- **Model**: %s\n", agent.Model))
		prompt.WriteString(fmt.Sprintf("- **Capabilities**: %s\n", strings.Join(agent.Capabilities, ", ")))
		
		// Include cost information if available from registry
		if _, exists := agentMap[agent.ID]; exists {
			costMagnitude := agent.GetEffectiveCostMagnitude()
			prompt.WriteString(fmt.Sprintf("- **Cost Magnitude**: %d (", costMagnitude))
			switch costMagnitude {
			case 0:
				prompt.WriteString("Free - tool-only, no LLM costs")
			case 1:
				prompt.WriteString("Very Cheap - basic LLM usage")
			case 2:
				prompt.WriteString("Cheap - standard LLM usage")
			case 3:
				prompt.WriteString("Moderate - quality LLM usage")
			case 5:
				prompt.WriteString("Expensive - advanced LLM usage")
			case 8:
				prompt.WriteString("Premium - top-tier LLM usage")
			}
			prompt.WriteString(")\n")
			
			contextWindow := agent.GetEffectiveContextWindow()
			prompt.WriteString(fmt.Sprintf("- **Context Window**: %d tokens\n", contextWindow))
			prompt.WriteString(fmt.Sprintf("- **Context Reset Strategy**: %s\n", agent.GetEffectiveContextReset()))
		}
		
		if agent.Description != "" {
			prompt.WriteString(fmt.Sprintf("- **Description**: %s\n", agent.Description))
		}
		
		if len(agent.Tools) > 0 {
			prompt.WriteString(fmt.Sprintf("- **Available Tools**: %s\n", strings.Join(agent.Tools, ", ")))
		}
		
		prompt.WriteString("\n")
	}
	
	prompt.WriteString("## Instructions\n")
	prompt.WriteString("Break down the objective into specific tasks. Consider:\n")
	prompt.WriteString("1. Which agents are best suited for each type of work\n")
	prompt.WriteString("2. The cost implications of using different agents\n")
	prompt.WriteString("3. Dependencies between tasks\n")
	prompt.WriteString("4. The complexity and criticality of each task\n\n")
	
	prompt.WriteString("For each task, provide:\n")
	prompt.WriteString("1. A unique task ID (e.g., TASK-001)\n")
	prompt.WriteString("2. A clear, concise title\n")
	prompt.WriteString("3. A detailed description\n")
	prompt.WriteString("4. Required capabilities\n")
	prompt.WriteString("5. Dependencies on other tasks (if any)\n")
	prompt.WriteString("6. Estimated complexity (low, medium, high)\n")
	prompt.WriteString("7. Criticality (low, medium, high, critical)\n\n")
	
	prompt.WriteString("Format your response as follows:\n")
	prompt.WriteString("```\n")
	prompt.WriteString("TASK-001: [Title]\n")
	prompt.WriteString("Description: [Detailed description]\n")
	prompt.WriteString("Capabilities: [capability1, capability2]\n")
	prompt.WriteString("Dependencies: [TASK-XXX, TASK-YYY] or none\n")
	prompt.WriteString("Complexity: [low|medium|high]\n")
	prompt.WriteString("Criticality: [low|medium|high|critical]\n")
	prompt.WriteString("---\n")
	prompt.WriteString("```\n")
	
	return prompt.String()
}

// buildContextAwareAssignmentPrompt creates a prompt for intelligent task assignment
func (p *ContextAwareTaskPlanner) buildContextAwareAssignmentPrompt(tasks []*kanban.Task, guild *config.GuildConfig) string {
	var prompt strings.Builder
	
	prompt.WriteString("You are the Guild Master making intelligent task assignments based on agent capabilities, costs, and project needs.\n\n")
	
	prompt.WriteString("## Tasks to Assign\n\n")
	for _, task := range tasks {
		prompt.WriteString(fmt.Sprintf("### %s: %s\n", task.ID, task.Title))
		prompt.WriteString(fmt.Sprintf("- Description: %s\n", task.Description))
		
		if caps, ok := task.Metadata["capabilities"]; ok {
			prompt.WriteString(fmt.Sprintf("- Required Capabilities: %s\n", caps))
		}
		if complexity, ok := task.Metadata["complexity"]; ok {
			prompt.WriteString(fmt.Sprintf("- Complexity: %s\n", complexity))
		}
		if criticality, ok := task.Metadata["criticality"]; ok {
			prompt.WriteString(fmt.Sprintf("- Criticality: %s\n", criticality))
		}
		if len(task.Dependencies) > 0 {
			prompt.WriteString(fmt.Sprintf("- Dependencies: %s\n", strings.Join(task.Dependencies, ", ")))
		}
		prompt.WriteString("\n")
	}
	
	prompt.WriteString("## Available Guild Members with Full Context\n\n")
	
	// Get current workloads if available
	workloads := p.getCurrentWorkloads(tasks)
	
	// Get all registered agents from the registry for complete information
	registeredAgents := p.componentRegistry.Agents().(*registry.DefaultAgentRegistry).GetRegisteredAgents()
	agentMap := make(map[string]registry.GuildAgentConfig)
	for _, agent := range registeredAgents {
		agentMap[agent.ID] = agent
	}
	
	for _, agent := range guild.Agents {
		prompt.WriteString(fmt.Sprintf("### %s (%s)\n", agent.Name, agent.ID))
		prompt.WriteString(fmt.Sprintf("- **Type**: %s\n", agent.Type))
		prompt.WriteString(fmt.Sprintf("- **Capabilities**: %s\n", strings.Join(agent.Capabilities, ", ")))
		
		// Cost analysis
		costMagnitude := agent.GetEffectiveCostMagnitude()
		prompt.WriteString(fmt.Sprintf("- **Cost**: %d - ", costMagnitude))
		switch costMagnitude {
		case 0:
			prompt.WriteString("Free (tool-only agent, no LLM costs)")
		case 1:
			prompt.WriteString("Very economical")
		case 2:
			prompt.WriteString("Budget-friendly")
		case 3:
			prompt.WriteString("Moderate cost")
		case 5:
			prompt.WriteString("Premium cost")
		case 8:
			prompt.WriteString("Highest cost")
		}
		prompt.WriteString("\n")
		
		// Context window affects ability to handle complex tasks
		contextWindow := agent.GetEffectiveContextWindow()
		prompt.WriteString(fmt.Sprintf("- **Context Capacity**: %d tokens (", contextWindow))
		if contextWindow >= 100000 {
			prompt.WriteString("can handle very complex, multi-file tasks")
		} else if contextWindow >= 32000 {
			prompt.WriteString("good for moderately complex tasks")
		} else {
			prompt.WriteString("best for focused, single-purpose tasks")
		}
		prompt.WriteString(")\n")
		
		// Current workload
		if workload, exists := workloads[agent.ID]; exists && workload > 0 {
			prompt.WriteString(fmt.Sprintf("- **Current Workload**: %d tasks assigned\n", workload))
		} else {
			prompt.WriteString("- **Current Workload**: Available\n")
		}
		
		// Special considerations
		if agent.Type == "manager" {
			prompt.WriteString("- **Special**: Best for planning, architecture, and coordination tasks\n")
		} else if agent.Type == "specialist" {
			prompt.WriteString("- **Special**: Expert in their domain, use for critical domain-specific tasks\n")
		}
		
		prompt.WriteString("\n")
	}
	
	prompt.WriteString("## Assignment Guidelines\n\n")
	prompt.WriteString("Consider these factors when making assignments:\n")
	prompt.WriteString("1. **Capability Match**: Agent must have required capabilities\n")
	prompt.WriteString("2. **Cost Efficiency**: Balance cost with task criticality\n")
	prompt.WriteString("3. **Workload Balance**: Distribute tasks to avoid overloading\n")
	prompt.WriteString("4. **Task Complexity**: Match complex tasks with capable agents\n")
	prompt.WriteString("5. **Critical Path**: Prioritize critical tasks with reliable agents\n\n")
	
	prompt.WriteString("For each assignment, explain your reasoning considering the full context.\n\n")
	
	prompt.WriteString("Format your response as:\n")
	prompt.WriteString("```\n")
	prompt.WriteString("TASK-001: agent_id\n")
	prompt.WriteString("Reason: [Detailed explanation of why this agent was chosen considering capabilities, cost, workload, and task requirements]\n")
	prompt.WriteString("Confidence: [high|medium|low]\n")
	prompt.WriteString("---\n")
	prompt.WriteString("```\n")
	
	return prompt.String()
}

// Assignment represents a task assignment with reasoning
type Assignment struct {
	AgentID    string
	Reason     string
	Confidence string
}

// parseAssignmentsFromResponse parses intelligent assignments from manager's response
func (p *ContextAwareTaskPlanner) parseAssignmentsFromResponse(response string, tasks []*kanban.Task, guild *config.GuildConfig) (map[string]Assignment, error) {
	assignments := make(map[string]Assignment)
	
	lines := strings.Split(response, "\n")
	var currentTaskID string
	var currentAssignment Assignment
	
	for _, line := range lines {
		line = strings.TrimSpace(line)
		
		// Skip empty lines and markers
		if line == "" || line == "```" || line == "---" {
			if currentTaskID != "" && currentAssignment.AgentID != "" {
				assignments[currentTaskID] = currentAssignment
				currentTaskID = ""
				currentAssignment = Assignment{}
			}
			continue
		}
		
		// Parse task assignment
		if strings.Contains(line, ":") && strings.HasPrefix(line, "TASK-") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				currentTaskID = strings.TrimSpace(parts[0])
				agentID := strings.TrimSpace(parts[1])
				
				// Validate agent exists
				if _, err := guild.GetAgentByID(agentID); err == nil {
					currentAssignment.AgentID = agentID
				}
			}
		} else if strings.HasPrefix(line, "Reason:") && currentTaskID != "" {
			currentAssignment.Reason = strings.TrimSpace(strings.TrimPrefix(line, "Reason:"))
		} else if strings.HasPrefix(line, "Confidence:") && currentTaskID != "" {
			currentAssignment.Confidence = strings.TrimSpace(strings.TrimPrefix(line, "Confidence:"))
		}
	}
	
	// Add last assignment if any
	if currentTaskID != "" && currentAssignment.AgentID != "" {
		assignments[currentTaskID] = currentAssignment
	}
	
	return assignments, nil
}

// getCurrentWorkloads calculates current task assignments per agent
func (p *ContextAwareTaskPlanner) getCurrentWorkloads(tasks []*kanban.Task) map[string]int {
	workloads := make(map[string]int)
	
	for _, task := range tasks {
		if task.AssignedTo != "" {
			workloads[task.AssignedTo]++
		}
	}
	
	return workloads
}

// parseTasksFromResponse parses tasks from the manager's response
func (p *ContextAwareTaskPlanner) parseTasksFromResponse(response string) ([]*kanban.Task, error) {
	tasks := []*kanban.Task{}
	
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
			} else if strings.HasPrefix(line, "Criticality:") {
				currentTask.Metadata["criticality"] = strings.TrimSpace(strings.TrimPrefix(line, "Criticality:"))
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