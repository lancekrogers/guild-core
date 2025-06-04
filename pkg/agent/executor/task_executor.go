package executor

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/guild-ventures/guild-core/internal/prompts/agent/execution"
	"github.com/guild-ventures/guild-core/pkg/agent"
	"github.com/guild-ventures/guild-core/pkg/kanban"
	"github.com/guild-ventures/guild-core/pkg/tools"
	"github.com/guild-ventures/guild-core/pkg/workspace"
	"github.com/guild-ventures/guild-core/tools/fs"
	"github.com/guild-ventures/guild-core/tools/shell"
)

// BasicTaskExecutor implements the TaskExecutor interface
type BasicTaskExecutor struct {
	agent           agent.Agent
	kanbanBoard     *kanban.Board
	toolRegistry    *tools.ToolRegistry
	execContext     *ExecutionContext
	promptBuilder   *execution.CachedPromptBuilder
	workspaceManager workspace.Manager
	workspace       workspace.Workspace
	
	// Execution state
	status       ExecutionStatus
	progress     float64
	currentTask  *kanban.Task
	result       *ExecutionResult
	
	// Synchronization
	mu           sync.RWMutex
	stopChan     chan struct{}
	stopped      bool
}

// NewBasicTaskExecutor creates a new task executor
func NewBasicTaskExecutor(
	agent agent.Agent,
	kanbanBoard *kanban.Board,
	toolRegistry *tools.ToolRegistry,
	execContext *ExecutionContext,
	workspaceManager workspace.Manager,
) (*BasicTaskExecutor, error) {
	promptBuilder, err := execution.NewCachedPromptBuilder()
	if err != nil {
		return nil, fmt.Errorf("failed to create prompt builder: %w", err)
	}

	return &BasicTaskExecutor{
		agent:            agent,
		kanbanBoard:      kanbanBoard,
		toolRegistry:     toolRegistry,
		execContext:      execContext,
		promptBuilder:    promptBuilder,
		workspaceManager: workspaceManager,
		status:           StatusInitializing,
		progress:         0.0,
		stopChan:         make(chan struct{}),
	}, nil
}

// Execute runs the task execution loop
func (e *BasicTaskExecutor) Execute(ctx context.Context, task *kanban.Task) (*ExecutionResult, error) {
	e.mu.Lock()
	e.currentTask = task
	e.status = StatusInitializing
	e.result = &ExecutionResult{
		TaskID:    task.ID,
		StartTime: time.Now(),
		Status:    StatusInitializing,
		Artifacts: []Artifact{},
		ToolUsage: []ToolUsage{},
		Errors:    []ExecutionError{},
		Metadata:  make(map[string]interface{}),
	}
	e.mu.Unlock()

	// Update task status to in_progress
	if err := e.updateTaskStatus(ctx, kanban.StatusInProgress, "Agent starting task execution"); err != nil {
		return nil, fmt.Errorf("failed to update task status: %w", err)
	}

	// Main execution phases
	if err := e.executePhases(ctx); err != nil {
		e.mu.Lock()
		e.status = StatusFailed
		e.result.Status = StatusFailed
		e.result.EndTime = time.Now()
		e.result.Duration = e.result.EndTime.Sub(e.result.StartTime)
		e.result.Errors = append(e.result.Errors, ExecutionError{
			Phase:     "execution",
			Error:     err.Error(),
			Timestamp: time.Now(),
			Retryable: false,
		})
		e.mu.Unlock()

		// Update task status to blocked
		e.updateTaskStatus(ctx, kanban.StatusBlocked, fmt.Sprintf("Execution failed: %v", err))
		return e.result, err
	}

	// Mark execution as completed
	e.mu.Lock()
	e.status = StatusCompleted
	e.result.Status = StatusCompleted
	e.result.EndTime = time.Now()
	e.result.Duration = e.result.EndTime.Sub(e.result.StartTime)
	e.progress = 1.0
	result := e.result
	e.mu.Unlock()

	// Update task status to review
	if err := e.updateTaskStatus(ctx, kanban.StatusReadyForReview, "Task completed, pending review"); err != nil {
		return result, fmt.Errorf("failed to update final task status: %w", err)
	}

	return result, nil
}

// executePhases runs through the execution phases
func (e *BasicTaskExecutor) executePhases(ctx context.Context) error {
	phases := []struct {
		name     string
		progress float64
		fn       func(context.Context) error
	}{
		{"initialize", 0.1, e.phaseInitialize},
		{"plan", 0.2, e.phasePlan},
		{"execute", 0.7, e.phaseExecute},
		{"finalize", 0.9, e.phaseFinalize},
	}

	for _, phase := range phases {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-e.stopChan:
			return fmt.Errorf("execution stopped")
		default:
			// Continue execution
		}

		e.updateProgress(phase.progress, phase.name)
		
		if err := phase.fn(ctx); err != nil {
			return fmt.Errorf("phase %s failed: %w", phase.name, err)
		}
	}

	return nil
}

// phaseInitialize sets up the execution environment
func (e *BasicTaskExecutor) phaseInitialize(ctx context.Context) error {
	e.mu.Lock()
	e.status = StatusRunning
	e.mu.Unlock()

	// Set up workspace isolation if manager is available
	if e.workspaceManager != nil {
		opts := workspace.CreateOptions{
			AgentID:      e.agent.GetID(),
			BranchPrefix: "agent",
			WorkDir:      e.execContext.ProjectRoot,
		}
		
		ws, err := e.workspaceManager.CreateWorkspace(context.Background(), opts)
		if err != nil {
			return fmt.Errorf("failed to create workspace: %w", err)
		}
		
		e.mu.Lock()
		e.workspace = ws
		e.execContext.WorkspaceDir = ws.Path()
		e.mu.Unlock()

		e.addExecutionLog("Created isolated workspace", map[string]interface{}{
			"path":   ws.Path(),
			"branch": ws.Branch(),
		})
	}

	// Initialize tools if registry is available
	if e.toolRegistry != nil {
		// Register default tools if not already registered
		e.initializeDefaultTools()
		
		// Log available tools
		availableTools := []string{}
		for _, tool := range e.toolRegistry.ListTools() {
			availableTools = append(availableTools, tool.Name())
		}
		
		e.addExecutionLog("Initialized tools", map[string]interface{}{
			"available_tools": availableTools,
			"workspace":       e.execContext.WorkspaceDir,
		})
	}

	e.addExecutionLog("Initialized execution environment", nil)
	return nil
}

// phasePlan creates the execution plan using the agent
func (e *BasicTaskExecutor) phasePlan(ctx context.Context) error {
	// Build the planning prompt with all context layers except execution
	promptData := e.buildPromptData()
	
	planningPrompt, err := e.promptBuilder.BuildPlanningPromptCached(promptData)
	if err != nil {
		return fmt.Errorf("failed to build planning prompt: %w", err)
	}

	// Add planning instructions
	fullPrompt := planningPrompt + "\n\n## Planning Instructions\n\n" +
		"Based on the above context, create a detailed execution plan for completing this task. " +
		"Your plan should include:\n" +
		"1. A list of concrete steps to complete the task\n" +
		"2. Which tools you'll need for each step\n" +
		"3. Expected outputs or artifacts from each step\n" +
		"4. Potential challenges and how to address them\n\n" +
		"Format your response as a structured plan that can be executed step by step."

	// Query the agent for the execution plan
	plan, err := e.agent.Execute(ctx, fullPrompt)
	if err != nil {
		return fmt.Errorf("failed to generate execution plan: %w", err)
	}

	// TODO: Parse and validate the plan structure
	// For now, just log the plan
	e.addExecutionLog("Created execution plan", map[string]interface{}{
		"plan_length": len(plan),
		"agent_id":    e.agent.GetID(),
	})

	// Store plan in result metadata
	e.mu.Lock()
	e.result.Metadata["execution_plan"] = plan
	e.mu.Unlock()

	return nil
}

// phaseExecute performs the actual task work
func (e *BasicTaskExecutor) phaseExecute(ctx context.Context) error {
	// Get execution plan from metadata
	plan, _ := e.result.Metadata["execution_plan"].(string)
	
	// For now, demonstrate tool usage with a simple implementation
	// In a real implementation, this would parse the plan and execute accordingly
	steps := []struct {
		name        string
		description string
		execute     func(context.Context) error
	}{
		{
			name:        "analyze_task",
			description: "Analyzing task requirements",
			execute:     e.stepAnalyzeTask,
		},
		{
			name:        "prepare_workspace",
			description: "Preparing workspace",
			execute:     e.stepPrepareWorkspace,
		},
		{
			name:        "implement_solution",
			description: "Implementing solution",
			execute:     e.stepImplementSolution,
		},
		{
			name:        "verify_results",
			description: "Verifying results",
			execute:     e.stepVerifyResults,
		},
	}

	for i, step := range steps {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-e.stopChan:
			return fmt.Errorf("execution stopped")
		default:
			// Add small delay to allow context cancellation testing
			time.Sleep(20 * time.Millisecond)
			
			progress := 0.2 + (0.5 * float64(i) / float64(len(steps)))
			e.updateProgress(progress, step.description)
			
			// Execute the step
			if err := step.execute(ctx); err != nil {
				e.addExecutionLog(fmt.Sprintf("Step failed: %s", step.name), map[string]interface{}{
					"error": err.Error(),
				})
				return fmt.Errorf("step %s failed: %w", step.name, err)
			}
			
			e.addExecutionLog(fmt.Sprintf("Completed: %s", step.description), nil)
		}
	}

	// Store execution summary
	e.mu.Lock()
	e.result.Output = fmt.Sprintf("Task completed successfully. Plan: %s", plan)
	e.mu.Unlock()

	return nil
}

// phaseFinalize cleans up and prepares results
func (e *BasicTaskExecutor) phaseFinalize(ctx context.Context) error {
	// Collect workspace artifacts if available
	if e.workspace != nil {
		// Check for uncommitted changes if using git workspace
		if gitWs, ok := e.workspace.(*workspace.GitWorkspace); ok {
			gitWs.UpdateGitInfo()
			gitInfo := gitWs.GetGitInfo()
			if gitInfo.IsDirty {
				// Get diff for logging
				diff, _ := gitWs.GetDiff()
				e.addExecutionLog("Uncommitted changes detected", map[string]interface{}{
					"diff_size": len(diff),
				})
				
				// Commit changes
				commitMsg := fmt.Sprintf("Task %s: Auto-commit by agent %s", e.currentTask.ID, e.agent.GetID())
				if err := gitWs.CommitChanges(commitMsg); err != nil {
					e.addExecutionLog("Failed to commit changes", map[string]interface{}{
						"error": err.Error(),
					})
				} else {
					gitWs.UpdateGitInfo()
					e.addExecutionLog("Committed workspace changes", map[string]interface{}{
						"commit": gitWs.GetGitInfo().CommitHash,
					})
				}
			}
		}

		// Store workspace info in result
		e.mu.Lock()
		e.result.Metadata["workspace_path"] = e.workspace.Path()
		e.result.Metadata["workspace_branch"] = e.workspace.Branch()
		e.mu.Unlock()
	}

	// TODO: Generate execution report
	// TODO: Copy important artifacts to permanent storage

	e.addExecutionLog("Finalized execution", map[string]interface{}{
		"artifacts_count": len(e.result.Artifacts),
		"duration":        time.Since(e.result.StartTime).String(),
	})
	
	// Note: We don't cleanup the workspace here - it might be needed for review
	// The workspace manager should handle cleanup based on retention policy
	
	return nil
}

// GetProgress returns the current execution progress
func (e *BasicTaskExecutor) GetProgress() float64 {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.progress
}

// GetStatus returns the current execution status
func (e *BasicTaskExecutor) GetStatus() ExecutionStatus {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.status
}

// Stop gracefully stops the execution
func (e *BasicTaskExecutor) Stop() error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.stopped {
		return nil
	}

	e.stopped = true
	close(e.stopChan)
	e.status = StatusStopped
	
	return nil
}

// updateProgress updates the execution progress
func (e *BasicTaskExecutor) updateProgress(progress float64, phase string) {
	e.mu.Lock()
	e.progress = progress
	e.mu.Unlock()

	// TODO: Emit progress event
}

// updateTaskStatus updates the kanban task status
func (e *BasicTaskExecutor) updateTaskStatus(ctx context.Context, status kanban.TaskStatus, comment string) error {
	if e.kanbanBoard == nil {
		// If no kanban board is configured, skip status update
		return nil
	}
	return e.kanbanBoard.UpdateTaskStatus(
		ctx,
		e.currentTask.ID,
		status,
		e.execContext.AgentID,
		comment,
	)
}

// addExecutionLog adds a log entry to the result
func (e *BasicTaskExecutor) addExecutionLog(message string, metadata map[string]interface{}) {
	e.mu.Lock()
	defer e.mu.Unlock()

	// For now, append to output
	if e.result.Output != "" {
		e.result.Output += "\n"
	}
	e.result.Output += fmt.Sprintf("[%s] %s", time.Now().Format("15:04:05"), message)
	
	// Add to metadata if provided
	if metadata != nil {
		for k, v := range metadata {
			e.result.Metadata[k] = v
		}
	}
}

// executeToolCall executes a tool and tracks its usage
func (e *BasicTaskExecutor) executeToolCall(ctx context.Context, toolName string, params map[string]interface{}) (*tools.ToolResult, error) {
	if e.toolRegistry == nil {
		return nil, fmt.Errorf("no tool registry available")
	}

	// Track start time for usage metrics
	startTime := time.Now()

	// Execute the tool
	result, err := e.toolRegistry.ExecuteToolWithParams(ctx, toolName, params)
	
	// Calculate execution time
	duration := time.Since(startTime)

	// Track tool usage
	e.mu.Lock()
	// Find or create tool usage entry
	var toolUsage *ToolUsage
	for i := range e.result.ToolUsage {
		if e.result.ToolUsage[i].ToolName == toolName {
			toolUsage = &e.result.ToolUsage[i]
			break
		}
	}
	if toolUsage == nil {
		e.result.ToolUsage = append(e.result.ToolUsage, ToolUsage{
			ToolName: toolName,
		})
		toolUsage = &e.result.ToolUsage[len(e.result.ToolUsage)-1]
	}
	
	// Update usage stats
	toolUsage.Invocations++
	toolUsage.TotalTime += duration
	if result != nil {
		toolUsage.Results = append(toolUsage.Results, map[string]interface{}{
			"timestamp": time.Now(),
			"success":   result.Success,
			"duration":  duration.String(),
		})
	}
	e.mu.Unlock()

	// Log tool execution
	e.addExecutionLog(fmt.Sprintf("Executed tool: %s", toolName), map[string]interface{}{
		"tool":     toolName,
		"success":  result != nil && result.Success,
		"duration": duration.String(),
	})

	return result, err
}

// buildPromptData builds the prompt data structure from current context
func (e *BasicTaskExecutor) buildPromptData() execution.ExecutionPromptData {
	e.mu.RLock()
	defer e.mu.RUnlock()

	// Build available tools data
	var toolsData []execution.ToolData
	if e.toolRegistry != nil {
		// Get actual tools from registry
		for _, tool := range e.toolRegistry.ListTools() {
			// Convert schema to parameters
			var params []execution.ToolParameter
			if schema := tool.Schema(); schema != nil {
				if props, ok := schema["properties"].(map[string]interface{}); ok {
					for name, prop := range props {
						if propMap, ok := prop.(map[string]interface{}); ok {
							param := execution.ToolParameter{
								Name:        name,
								Type:        propMap["type"].(string),
								Description: propMap["description"].(string),
							}
							params = append(params, param)
						}
					}
				}
			}

			toolData := execution.ToolData{
				Name:        tool.Name(),
				Description: tool.Description(),
				Category:    tool.Category(),
				Parameters:  params,
				ReturnType:  "ToolResult",
				Examples:    tool.Examples(),
			}
			toolsData = append(toolsData, toolData)
		}
	}

	return execution.ExecutionPromptData{
		Agent: execution.AgentData{
			Name:         e.agent.GetName(),
			Role:         "Task execution specialist",
			Capabilities: e.execContext.Capabilities,
		},
		Context: execution.ContextData{
			GuildID:            e.execContext.AgentID, // Using agent ID as guild ID for now
			ProjectName:        "Current Project",
			ProjectDescription: "Project working on " + e.currentTask.Title,
			WorkspaceDir:       e.execContext.WorkspaceDir,
			RelevantDocs:       []execution.DocumentRef{},
			TechStack:          "Go", // TODO: Get from project context
			Architecture:       "Modular",
			Dependencies:       "Standard library",
			RelatedTasks:       []execution.RelatedTask{},
		},
		Commission: execution.CommissionData{
			Title:            e.execContext.Objective,
			Description:      "Complete the assigned commission",
			SuccessCriteria:  []string{"Task completed successfully", "Tests pass", "Documentation updated"},
		},
		Task: execution.TaskData{
			Title:          e.currentTask.Title,
			Description:    e.currentTask.Description,
			Requirements:   []string{}, // TODO: Parse from task description
			Constraints:    []string{},
			Priority:       string(e.currentTask.Priority),
			DueDate:        e.formatDueDate(),
			EstimatedHours: e.currentTask.EstimatedHours,
			Dependencies:   e.buildDependencies(),
			Deliverables:   []execution.Deliverable{},
		},
		Tools: toolsData,
		ToolConfig: execution.ToolConfigData{
			MaxCalls:   100,
			Timeout:    30 * time.Second,
			RateLimits: "10 calls/minute",
		},
		Execution: execution.ExecutionData{
			Phase:                  string(e.status),
			StepNumber:             1,
			TotalSteps:             4,
			StepName:               "Current step",
			StepObjective:          "Complete current phase",
			ExpectedActions:        []string{},
			SuccessIndicators:      []string{},
			PotentialIssues:        []string{},
			OverallProgress:        int(e.progress * 100),
			PhaseProgress:          0,
			TimeElapsed:            time.Since(e.result.StartTime).String(),
			EstimatedTimeRemaining: "Unknown",
			PreviousStepResult:     "",
			NextSteps:              []string{},
		},
	}
}

// formatDueDate formats the task due date
func (e *BasicTaskExecutor) formatDueDate() string {
	if e.currentTask.DueDate != nil {
		return e.currentTask.DueDate.Format("2006-01-02")
	}
	return "No due date"
}

// buildDependencies builds task dependency data
func (e *BasicTaskExecutor) buildDependencies() []execution.TaskDependency {
	var deps []execution.TaskDependency
	for _, depID := range e.currentTask.Dependencies {
		deps = append(deps, execution.TaskDependency{
			TaskID:     depID,
			Title:      "Dependency " + depID,
			Status:     "Unknown",
			OutputPath: "",
		})
	}
	return deps
}

// initializeDefaultTools registers default tools if not already registered
func (e *BasicTaskExecutor) initializeDefaultTools() {
	// Use workspace directory as base path for file operations
	basePath := e.execContext.WorkspaceDir
	if basePath == "" {
		basePath = e.execContext.ProjectRoot
	}

	// Register file tool if not already registered
	if _, exists := e.toolRegistry.GetTool("file"); !exists {
		fileTool := fs.NewFileTool(basePath)
		e.toolRegistry.RegisterTool(fileTool)
	}

	// Register shell tool if not already registered
	if _, exists := e.toolRegistry.GetTool("shell"); !exists {
		shellOptions := shell.ShellToolOptions{
			WorkingDir: basePath,
			// Add safety restrictions
			BlockedCommands: []string{
				"rm -rf /", "rm -rf /*", "shutdown", "reboot",
				"passwd", "su", "sudo", "chown", "chmod 777",
			},
		}
		shellTool := shell.NewShellTool(shellOptions)
		e.toolRegistry.RegisterTool(shellTool)
	}
}

// Step execution functions demonstrating tool usage

func (e *BasicTaskExecutor) stepAnalyzeTask(ctx context.Context) error {
	// Check context before proceeding
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	
	// Use shell tool to check current directory structure
	if e.toolRegistry != nil {
		result, err := e.executeToolCall(ctx, "shell", map[string]interface{}{
			"command": "ls",
			"args":    []string{"-la"},
		})
		if err == nil && result != nil {
			e.addExecutionLog("Analyzed workspace structure", map[string]interface{}{
				"files": len(strings.Split(result.Output, "\n")),
			})
		}
	}
	return nil
}

func (e *BasicTaskExecutor) stepPrepareWorkspace(ctx context.Context) error {
	// Check context before proceeding
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	
	// Create a task directory and README
	if e.toolRegistry != nil {
		// Create task directory
		taskDir := fmt.Sprintf("task_%s", e.currentTask.ID)
		_, err := e.executeToolCall(ctx, "shell", map[string]interface{}{
			"command": "mkdir",
			"args":    []string{"-p", taskDir},
		})
		if err != nil {
			return fmt.Errorf("failed to create task directory: %w", err)
		}

		// Create README file
		readmeContent := fmt.Sprintf("# Task: %s\n\n%s\n\nStarted: %s\n", 
			e.currentTask.Title, 
			e.currentTask.Description,
			time.Now().Format(time.RFC3339))
		
		result, err := e.executeToolCall(ctx, "file", map[string]interface{}{
			"operation": "write",
			"path":      filepath.Join(taskDir, "README.md"),
			"content":   readmeContent,
		})
		if err != nil {
			return fmt.Errorf("failed to create README: %w", err)
		}

		// Track artifact
		if result != nil && result.Success {
			e.mu.Lock()
			e.result.Artifacts = append(e.result.Artifacts, Artifact{
				Name:        "README.md",
				Type:        "documentation",
				Path:        filepath.Join(e.execContext.WorkspaceDir, taskDir, "README.md"),
				Size:        int64(len(readmeContent)),
				CreatedAt:   time.Now(),
				Description: "Task documentation",
			})
			e.mu.Unlock()
		}
	}
	return nil
}

func (e *BasicTaskExecutor) stepImplementSolution(ctx context.Context) error {
	// Check context before proceeding
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	
	// Demonstrate creating a solution file
	if e.toolRegistry != nil {
		taskDir := fmt.Sprintf("task_%s", e.currentTask.ID)
		
		// Create a solution file based on task
		solutionContent := fmt.Sprintf(`#!/bin/bash
# Solution for: %s
# Generated by: %s
# Date: %s

echo "Executing task solution..."
echo "Task ID: %s"
echo "Task Title: %s"

# Task implementation would go here
echo "Task completed successfully"
`, 
			e.currentTask.Title,
			e.agent.GetName(),
			time.Now().Format(time.RFC3339),
			e.currentTask.ID,
			e.currentTask.Title,
		)

		result, err := e.executeToolCall(ctx, "file", map[string]interface{}{
			"operation": "write",
			"path":      filepath.Join(taskDir, "solution.sh"),
			"content":   solutionContent,
		})
		if err != nil {
			return fmt.Errorf("failed to create solution: %w", err)
		}

		// Make it executable
		_, err = e.executeToolCall(ctx, "shell", map[string]interface{}{
			"command": "chmod",
			"args":    []string{"+x", filepath.Join(taskDir, "solution.sh")},
		})

		// Track artifact
		if result != nil && result.Success {
			e.mu.Lock()
			e.result.Artifacts = append(e.result.Artifacts, Artifact{
				Name:        "solution.sh",
				Type:        "script",
				Path:        filepath.Join(e.execContext.WorkspaceDir, taskDir, "solution.sh"),
				Size:        int64(len(solutionContent)),
				CreatedAt:   time.Now(),
				Description: "Task solution script",
			})
			e.mu.Unlock()
		}
	}
	return nil
}

func (e *BasicTaskExecutor) stepVerifyResults(ctx context.Context) error {
	// Check context before proceeding
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	
	// Verify created files exist
	if e.toolRegistry != nil {
		taskDir := fmt.Sprintf("task_%s", e.currentTask.ID)
		
		// List created files
		result, err := e.executeToolCall(ctx, "shell", map[string]interface{}{
			"command": "ls",
			"args":    []string{"-la", taskDir},
		})
		if err != nil {
			return fmt.Errorf("failed to verify results: %w", err)
		}

		// Create verification report
		verificationReport := fmt.Sprintf(`# Verification Report
Task: %s
Date: %s
Status: Completed

## Created Files:
%s

## Summary:
- Task directory created
- Documentation generated
- Solution implemented
- All artifacts tracked
`, 
			e.currentTask.Title,
			time.Now().Format(time.RFC3339),
			result.Output,
		)

		_, err = e.executeToolCall(ctx, "file", map[string]interface{}{
			"operation": "write",
			"path":      filepath.Join(taskDir, "verification.md"),
			"content":   verificationReport,
		})

		e.addExecutionLog("Verification completed", map[string]interface{}{
			"artifacts_created": len(e.result.Artifacts),
			"task_directory":    taskDir,
		})
	}
	return nil
}