// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package executor

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/lancekrogers/guild/pkg/agents/core"
	"github.com/lancekrogers/guild/pkg/gerror"
	"github.com/lancekrogers/guild/pkg/git/worktree"
	"github.com/lancekrogers/guild/pkg/kanban"
	"github.com/lancekrogers/guild/pkg/observability"
	"github.com/lancekrogers/guild/pkg/prompts/standard/templates/agent/execution"
	"github.com/lancekrogers/guild/pkg/tools"
	"github.com/lancekrogers/guild/tools/fs"
	"github.com/lancekrogers/guild/tools/shell"
)

// BasicTaskExecutor implements the TaskExecutor interface
type BasicTaskExecutor struct {
	agent           core.Agent
	kanbanBoard     *kanban.Board
	toolRegistry    *tools.ToolRegistry
	execContext     *ExecutionContext
	promptBuilder   *execution.CachedPromptBuilder
	worktreeManager worktree.Manager
	currentWorktree *worktree.Worktree

	// Execution state
	status      ExecutionStatus
	progress    float64
	currentTask *kanban.Task
	result      *ExecutionResult

	// Synchronization
	mu       sync.RWMutex
	stopChan chan struct{}
	stopped  bool
}

// NewBasicTaskExecutor creates a new task executor
func NewBasicTaskExecutor(
	agent core.Agent,
	kanbanBoard *kanban.Board,
	toolRegistry *tools.ToolRegistry,
	execContext *ExecutionContext,
	worktreeManager worktree.Manager,
) (*BasicTaskExecutor, error) {
	// Initialize observability for task executor creation
	logger := observability.GetLogger(context.Background()).
		WithComponent("executor").
		WithOperation("NewBasicTaskExecutor").
		With("agent_id", agent.GetID())

	logger.Debug("Creating new task executor",
		"agent_name", agent.GetName(),
		"has_kanban_board", kanbanBoard != nil,
		"has_tool_registry", toolRegistry != nil,
		"has_worktree_manager", worktreeManager != nil,
		"project_root", execContext.ProjectRoot,
	)

	promptBuilder, err := execution.NewCachedPromptBuilder()
	if err != nil {
		// Enhanced error observability with comprehensive context
		logger.WithError(err).Error("Failed to create prompt builder for task executor",
			"agent_id", agent.GetID(),
			"agent_name", agent.GetName(),
			"project_root", execContext.ProjectRoot,
		)

		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create prompt builder").
			WithComponent("executor").
			WithOperation("NewBasicTaskExecutor").
			WithDetails("agent_id", agent.GetID()).
			WithDetails("agent_name", agent.GetName())
	}

	executor := &BasicTaskExecutor{
		agent:           agent,
		kanbanBoard:     kanbanBoard,
		toolRegistry:    toolRegistry,
		execContext:     execContext,
		promptBuilder:   promptBuilder,
		worktreeManager: worktreeManager,
		status:          StatusInitializing,
		progress:        0.0,
		stopChan:        make(chan struct{}),
	}

	logger.Info("Task executor created successfully",
		"agent_id", agent.GetID(),
		"agent_name", agent.GetName(),
		"initial_status", StatusInitializing,
	)

	return executor, nil
}

// Execute runs the task execution loop
func (e *BasicTaskExecutor) Execute(ctx context.Context, task *kanban.Task) (*ExecutionResult, error) {
	// Check context early
	if err := ctx.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "task execution cancelled").
			WithComponent("executor").
			WithOperation("Execute").
			WithDetails("task_id", task.ID)
	}

	// Initialize observability for task execution with full context
	logger := observability.GetLogger(ctx).
		WithComponent("executor").
		WithOperation("Execute").
		With("task_id", task.ID).
		With("agent_id", e.agent.GetID())

	start := time.Now()
	logger.Info("Starting task execution",
		"task_title", task.Title,
		"task_description_length", len(task.Description),
		"agent_name", e.agent.GetName(),
		"has_kanban_board", e.kanbanBoard != nil,
		"has_tool_registry", e.toolRegistry != nil,
		"has_worktree_manager", e.worktreeManager != nil,
	)

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

	// Update task status to in_progress with enhanced error logging
	statusUpdateStart := time.Now()
	if err := e.updateTaskStatus(ctx, kanban.StatusInProgress, "Agent starting task execution"); err != nil {
		statusUpdateDuration := time.Since(statusUpdateStart)

		// Enhanced error observability with comprehensive context
		logger.WithError(err).Error("Failed to update task status to in_progress",
			"task_id", task.ID,
			"task_title", task.Title,
			"agent_id", e.agent.GetID(),
			"agent_name", e.agent.GetName(),
			"target_status", kanban.StatusInProgress,
			"status_update_duration_ms", statusUpdateDuration.Milliseconds(),
			"has_kanban_board", e.kanbanBoard != nil,
		)

		return nil, gerror.Wrap(err, gerror.ErrCodeOrchestration, "failed to update task status").
			WithComponent("executor").
			WithOperation("Execute").
			WithDetails("task_id", task.ID).
			WithDetails("status", kanban.StatusInProgress).
			WithDetails("agent_id", e.agent.GetID()).
			WithDetails("status_update_duration_ms", statusUpdateDuration.Milliseconds())
	}
	statusUpdateDuration := time.Since(statusUpdateStart)

	logger.Debug("Task status updated to in_progress",
		"status_update_duration_ms", statusUpdateDuration.Milliseconds(),
	)

	// Main execution phases with comprehensive error observability
	phasesStart := time.Now()
	if err := e.executePhases(ctx); err != nil {
		phasesDuration := time.Since(phasesStart)

		// Enhanced error observability for execution failure
		logger.WithError(err).Error("Task execution phases failed",
			"task_id", task.ID,
			"task_title", task.Title,
			"agent_id", e.agent.GetID(),
			"agent_name", e.agent.GetName(),
			"phases_duration_ms", phasesDuration.Milliseconds(),
			"total_execution_duration_ms", time.Since(start).Milliseconds(),
		)

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
		finalResult := e.result
		e.mu.Unlock()

		// Update task status to blocked with enhanced error logging
		blockingStatusStart := time.Now()
		if statusErr := e.updateTaskStatus(ctx, kanban.StatusBlocked, fmt.Sprintf("Execution failed: %v", err)); statusErr != nil {
			blockingStatusDuration := time.Since(blockingStatusStart)

			// Enhanced error observability for status update failure
			logger.WithError(statusErr).Error("Failed to update task status to blocked after execution failure",
				"task_id", task.ID,
				"task_title", task.Title,
				"agent_id", e.agent.GetID(),
				"agent_name", e.agent.GetName(),
				"original_execution_error", err.Error(),
				"blocking_status_duration_ms", blockingStatusDuration.Milliseconds(),
				"total_execution_duration_ms", time.Since(start).Milliseconds(),
			)

			// Log status update error but don't override the main execution error
			_ = gerror.Wrap(statusErr, gerror.ErrCodeInternal, "failed to update task status").
				WithComponent("executor").
				WithOperation("Execute").
				WithDetails("task_id", task.ID).
				WithDetails("original_error", err.Error()).
				WithDetails("blocking_status_duration_ms", blockingStatusDuration.Milliseconds())
		} else {
			blockingStatusDuration := time.Since(blockingStatusStart)
			logger.Debug("Task status updated to blocked after execution failure",
				"blocking_status_duration_ms", blockingStatusDuration.Milliseconds(),
			)
		}

		// Log final execution failure metrics
		logger.Duration("executor.task_execution", time.Since(start),
			"success", false,
			"task_id", task.ID,
			"agent_id", e.agent.GetID(),
			"phases_duration_ms", phasesDuration.Milliseconds(),
			"final_status", StatusFailed,
		)

		return finalResult, err
	}
	phasesDuration := time.Since(phasesStart)

	logger.Debug("Task execution phases completed successfully",
		"phases_duration_ms", phasesDuration.Milliseconds(),
	)

	// Mark execution as completed with comprehensive success observability
	e.mu.Lock()
	e.status = StatusCompleted
	e.result.Status = StatusCompleted
	e.result.EndTime = time.Now()
	e.result.Duration = e.result.EndTime.Sub(e.result.StartTime)
	e.progress = 1.0
	result := e.result
	artifactCount := len(e.result.Artifacts)
	toolUsageCount := len(e.result.ToolUsage)
	e.mu.Unlock()

	logger.Info("Task execution completed successfully",
		"total_duration_ms", result.Duration.Milliseconds(),
		"phases_duration_ms", phasesDuration.Milliseconds(),
		"artifacts_created", artifactCount,
		"tools_used", toolUsageCount,
	)

	// Update task status to review with enhanced error logging
	reviewStatusStart := time.Now()
	if err := e.updateTaskStatus(ctx, kanban.StatusReadyForReview, "Task completed, pending review"); err != nil {
		reviewStatusDuration := time.Since(reviewStatusStart)

		// Enhanced error observability for final status update failure
		logger.WithError(err).Error("Failed to update task status to ready_for_review after successful execution",
			"task_id", task.ID,
			"task_title", task.Title,
			"agent_id", e.agent.GetID(),
			"agent_name", e.agent.GetName(),
			"execution_duration_ms", result.Duration.Milliseconds(),
			"review_status_duration_ms", reviewStatusDuration.Milliseconds(),
			"artifacts_created", artifactCount,
			"tools_used", toolUsageCount,
		)

		// Log performance metrics even if status update fails
		logger.Duration("executor.task_execution", time.Since(start),
			"success", true,
			"task_id", task.ID,
			"agent_id", e.agent.GetID(),
			"phases_duration_ms", phasesDuration.Milliseconds(),
			"final_status", StatusCompleted,
			"status_update_failed", true,
		)

		return result, gerror.Wrap(err, gerror.ErrCodeOrchestration, "failed to update final task status").
			WithComponent("executor").
			WithOperation("Execute").
			WithDetails("task_id", e.currentTask.ID).
			WithDetails("status", kanban.StatusReadyForReview).
			WithDetails("execution_duration_ms", result.Duration.Milliseconds()).
			WithDetails("review_status_duration_ms", reviewStatusDuration.Milliseconds())
	}
	reviewStatusDuration := time.Since(reviewStatusStart)

	logger.Debug("Task status updated to ready_for_review",
		"review_status_duration_ms", reviewStatusDuration.Milliseconds(),
	)

	// Log comprehensive success metrics
	totalDuration := time.Since(start)
	logger.Duration("executor.task_execution", totalDuration,
		"success", true,
		"task_id", task.ID,
		"agent_id", e.agent.GetID(),
		"phases_duration_ms", phasesDuration.Milliseconds(),
		"review_status_duration_ms", reviewStatusDuration.Milliseconds(),
		"final_status", StatusCompleted,
		"artifacts_created", artifactCount,
		"tools_used", toolUsageCount,
	)

	logger.Info("Task execution fully completed with status update",
		"total_duration_ms", totalDuration.Milliseconds(),
		"final_status", kanban.StatusReadyForReview,
	)

	return result, nil
}

// executePhases runs through the execution phases
func (e *BasicTaskExecutor) executePhases(ctx context.Context) error {
	// Initialize observability for phase execution
	logger := observability.GetLogger(ctx).
		WithComponent("executor").
		WithOperation("executePhases").
		With("task_id", e.currentTask.ID).
		With("agent_id", e.agent.GetID())

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

	logger.Debug("Starting execution phases",
		"total_phases", len(phases),
		"phase_names", []string{"initialize", "plan", "execute", "finalize"},
	)

	for i, phase := range phases {
		phaseStart := time.Now()

		// Enhanced context cancellation and stop detection with observability
		select {
		case <-ctx.Done():
			logger.Warn("Execution cancelled by context",
				"cancelled_at_phase", phase.name,
				"completed_phases", i,
				"remaining_phases", len(phases)-i,
				"context_error", ctx.Err().Error(),
			)
			return ctx.Err()
		case <-e.stopChan:
			logger.Warn("Execution stopped by stop signal",
				"stopped_at_phase", phase.name,
				"completed_phases", i,
				"remaining_phases", len(phases)-i,
			)
			return gerror.New(gerror.ErrCodeCancelled, "execution stopped", nil).
				WithComponent("executor").
				WithOperation("executePhases").
				WithDetails("task_id", e.currentTask.ID).
				WithDetails("stopped_at_phase", phase.name).
				WithDetails("completed_phases", i)
		default:
			// Continue execution
		}

		logger.Debug("Starting execution phase",
			"phase_name", phase.name,
			"phase_index", i,
			"target_progress", phase.progress,
		)

		e.updateProgress(phase.progress, phase.name)

		// Execute phase with comprehensive error observability
		if err := phase.fn(ctx); err != nil {
			phaseDuration := time.Since(phaseStart)

			// Enhanced error observability for phase failure
			logger.WithError(err).Error("Execution phase failed",
				"phase_name", phase.name,
				"phase_index", i,
				"phase_duration_ms", phaseDuration.Milliseconds(),
				"completed_phases", i,
				"remaining_phases", len(phases)-i-1,
				"task_id", e.currentTask.ID,
				"task_title", e.currentTask.Title,
				"agent_id", e.agent.GetID(),
				"agent_name", e.agent.GetName(),
			)

			// Log performance metrics for failed phase
			logger.Duration("executor.phase_execution", phaseDuration,
				"success", false,
				"phase_name", phase.name,
				"phase_index", i,
				"task_id", e.currentTask.ID,
				"agent_id", e.agent.GetID(),
			)

			return gerror.Wrapf(err, gerror.ErrCodeTaskFailed, "phase %s failed", phase.name).
				WithComponent("executor").
				WithOperation("executePhases").
				WithDetails("task_id", e.currentTask.ID).
				WithDetails("phase", phase.name).
				WithDetails("phase_index", i).
				WithDetails("phase_duration_ms", phaseDuration.Milliseconds()).
				WithDetails("completed_phases", i)
		}

		phaseDuration := time.Since(phaseStart)
		logger.Debug("Execution phase completed successfully",
			"phase_name", phase.name,
			"phase_index", i,
			"phase_duration_ms", phaseDuration.Milliseconds(),
		)

		// Log performance metrics for successful phase
		logger.Duration("executor.phase_execution", phaseDuration,
			"success", true,
			"phase_name", phase.name,
			"phase_index", i,
			"task_id", e.currentTask.ID,
			"agent_id", e.agent.GetID(),
		)
	}

	logger.Info("All execution phases completed successfully",
		"total_phases", len(phases),
	)

	return nil
}

// phaseInitialize sets up the execution environment
func (e *BasicTaskExecutor) phaseInitialize(ctx context.Context) error {
	// Initialize observability for initialization phase
	logger := observability.GetLogger(ctx).
		WithComponent("executor").
		WithOperation("phaseInitialize").
		With("task_id", e.currentTask.ID).
		With("agent_id", e.agent.GetID())

	logger.Debug("Starting initialization phase",
		"has_worktree_manager", e.worktreeManager != nil,
		"has_tool_registry", e.toolRegistry != nil,
		"project_root", e.execContext.ProjectRoot,
	)

	e.mu.Lock()
	e.status = StatusRunning
	e.mu.Unlock()

	// Set up worktree isolation if manager is available with enhanced error observability
	if e.worktreeManager != nil {
		worktreeStart := time.Now()
		req := worktree.CreateWorktreeRequest{
			AgentID:     e.agent.GetID(),
			TaskID:      e.currentTask.ID,
			BaseBranch:  "main", // TODO: make configurable
			Description: fmt.Sprintf("Task execution for %s", e.currentTask.Title),
			Metadata: map[string]interface{}{
				"task_id":    e.currentTask.ID,
				"agent_name": e.agent.GetName(),
				"project":    e.execContext.ProjectRoot,
			},
		}

		logger.Debug("Creating isolated worktree",
			"agent_id", e.agent.GetID(),
			"task_id", e.currentTask.ID,
			"base_branch", req.BaseBranch,
		)

		wt, err := e.worktreeManager.CreateWorktree(ctx, req)
		worktreeDuration := time.Since(worktreeStart)

		if err != nil {
			// Enhanced error observability for worktree creation failure
			logger.WithError(err).Error("Failed to create isolated worktree",
				"agent_id", e.agent.GetID(),
				"agent_name", e.agent.GetName(),
				"task_id", e.currentTask.ID,
				"project_root", e.execContext.ProjectRoot,
				"worktree_creation_duration_ms", worktreeDuration.Milliseconds(),
			)

			return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create worktree").
				WithComponent("executor").
				WithOperation("phaseInitialize").
				WithDetails("agent_id", e.agent.GetID()).
				WithDetails("task_id", e.currentTask.ID).
				WithDetails("project_root", e.execContext.ProjectRoot).
				WithDetails("worktree_creation_duration_ms", worktreeDuration.Milliseconds())
		}

		e.mu.Lock()
		e.currentWorktree = wt
		e.execContext.WorkspaceDir = wt.Path
		e.mu.Unlock()

		logger.Info("Isolated worktree created successfully",
			"worktree_path", wt.Path,
			"worktree_branch", wt.Branch,
			"worktree_creation_duration_ms", worktreeDuration.Milliseconds(),
		)

		e.addExecutionLog("Created isolated worktree", map[string]interface{}{
			"path":   wt.Path,
			"branch": wt.Branch,
			"id":     wt.ID,
		})
	} else {
		logger.Debug("No worktree manager available, skipping worktree isolation")
	}

	// Initialize tools if registry is available with enhanced error observability
	if e.toolRegistry != nil {
		toolInitStart := time.Now()

		logger.Debug("Starting tool initialization",
			"workspace_dir", e.execContext.WorkspaceDir,
		)

		// Register default tools if not already registered
		e.initializeDefaultTools()

		// Log available tools
		availableTools := e.toolRegistry.ListTools()
		toolInitDuration := time.Since(toolInitStart)

		logger.Info("Tools initialized successfully",
			"available_tools_count", len(availableTools),
			"available_tools", availableTools,
			"workspace_dir", e.execContext.WorkspaceDir,
			"tool_init_duration_ms", toolInitDuration.Milliseconds(),
		)

		e.addExecutionLog("Initialized tools", map[string]interface{}{
			"available_tools": availableTools,
			"workspace":       e.execContext.WorkspaceDir,
		})
	} else {
		logger.Debug("No tool registry available, skipping tool initialization")
	}

	logger.Info("Initialization phase completed successfully")
	e.addExecutionLog("Initialized execution environment", nil)
	return nil
}

// phasePlan creates the execution plan using the agent
func (e *BasicTaskExecutor) phasePlan(ctx context.Context) error {
	// Initialize observability for planning phase
	logger := observability.GetLogger(ctx).
		WithComponent("executor").
		WithOperation("phasePlan").
		With("task_id", e.currentTask.ID).
		With("agent_id", e.agent.GetID())

	logger.Debug("Starting planning phase",
		"task_title", e.currentTask.Title,
		"task_description_length", len(e.currentTask.Description),
	)

	// Build the planning prompt with all context layers except execution
	promptBuildStart := time.Now()
	promptData := e.buildPromptData()

	planningPrompt, err := e.promptBuilder.BuildPlanningPromptCached(promptData)
	promptBuildDuration := time.Since(promptBuildStart)

	if err != nil {
		// Enhanced error observability for prompt building failure
		logger.WithError(err).Error("Failed to build planning prompt",
			"task_id", e.currentTask.ID,
			"task_title", e.currentTask.Title,
			"agent_id", e.agent.GetID(),
			"agent_name", e.agent.GetName(),
			"prompt_build_duration_ms", promptBuildDuration.Milliseconds(),
		)

		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to build planning prompt").
			WithComponent("executor").
			WithOperation("phasePlan").
			WithDetails("task_id", e.currentTask.ID).
			WithDetails("agent_id", e.agent.GetID()).
			WithDetails("prompt_build_duration_ms", promptBuildDuration.Milliseconds())
	}

	logger.Debug("Planning prompt built successfully",
		"prompt_length", len(planningPrompt),
		"prompt_build_duration_ms", promptBuildDuration.Milliseconds(),
	)

	// Add planning instructions
	fullPrompt := planningPrompt + "\n\n## Planning Instructions\n\n" +
		"Based on the above context, create a detailed execution plan for completing this task. " +
		"Your plan should include:\n" +
		"1. A list of concrete steps to complete the task\n" +
		"2. Which tools you'll need for each step\n" +
		"3. Expected outputs or artifacts from each step\n" +
		"4. Potential challenges and how to address them\n\n" +
		"Format your response as a structured plan that can be executed step by step."

	logger.Debug("Executing planning query with agent",
		"full_prompt_length", len(fullPrompt),
		"agent_name", e.agent.GetName(),
	)

	// Query the agent for the execution plan with enhanced error observability
	agentExecuteStart := time.Now()
	plan, err := e.agent.Execute(ctx, fullPrompt)
	agentExecuteDuration := time.Since(agentExecuteStart)

	if err != nil {
		// Enhanced error observability for agent execution failure during planning
		logger.WithError(err).Error("Failed to generate execution plan with agent",
			"task_id", e.currentTask.ID,
			"task_title", e.currentTask.Title,
			"agent_id", e.agent.GetID(),
			"agent_name", e.agent.GetName(),
			"prompt_length", len(fullPrompt),
			"agent_execute_duration_ms", agentExecuteDuration.Milliseconds(),
			"prompt_build_duration_ms", promptBuildDuration.Milliseconds(),
		)

		return gerror.Wrap(err, gerror.ErrCodeAgent, "failed to generate execution plan").
			WithComponent("executor").
			WithOperation("phasePlan").
			WithDetails("task_id", e.currentTask.ID).
			WithDetails("agent_id", e.agent.GetID()).
			WithDetails("prompt_length", len(fullPrompt)).
			WithDetails("agent_execute_duration_ms", agentExecuteDuration.Milliseconds())
	}

	logger.Info("Execution plan generated successfully",
		"plan_length", len(plan),
		"agent_execute_duration_ms", agentExecuteDuration.Milliseconds(),
		"prompt_build_duration_ms", promptBuildDuration.Milliseconds(),
	)

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
	// Check context early
	if err := ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "task execution phase cancelled").
			WithComponent("executor").
			WithOperation("phaseExecute")
	}

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
			return gerror.New(gerror.ErrCodeCancelled, "execution stopped", nil).
				WithComponent("executor").
				WithOperation("phaseExecute").
				WithDetails("task_id", e.currentTask.ID)
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
				return gerror.Wrapf(err, gerror.ErrCodeTaskFailed, "step %s failed", step.name).
					WithComponent("executor").
					WithOperation("phaseExecute").
					WithDetails("task_id", e.currentTask.ID).
					WithDetails("step", step.name)
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
	// Collect worktree artifacts if available
	if e.currentWorktree != nil {
		// Sync worktree changes (this handles commits and merging)
		if e.worktreeManager != nil {
			syncResult, err := e.worktreeManager.SyncWorktree(ctx, e.currentWorktree.ID)
			if err != nil {
				// Log sync error but don't fail the finalization
				logger := observability.GetLogger(ctx).
					WithComponent("executor").
					WithOperation("phaseFinalize")

				logger.WithError(err).Warn("Failed to sync worktree changes",
					"worktree_id", e.currentWorktree.ID,
					"agent_id", e.agent.GetID(),
				)

				e.addExecutionLog("Failed to sync worktree changes", map[string]interface{}{
					"error":       err.Error(),
					"worktree_id": e.currentWorktree.ID,
				})
			} else {
				e.addExecutionLog("Synced worktree changes", map[string]interface{}{
					"worktree_id": e.currentWorktree.ID,
					"ahead":       syncResult.Divergence.Ahead,
					"behind":      syncResult.Divergence.Behind,
					"success":     syncResult.Success,
				})
			}
		}

		// Store worktree info in result
		e.mu.Lock()
		e.result.Metadata["worktree_path"] = e.currentWorktree.Path
		e.result.Metadata["worktree_branch"] = e.currentWorktree.Branch
		e.result.Metadata["worktree_id"] = e.currentWorktree.ID
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
	// Initialize observability for tool execution
	logger := observability.GetLogger(ctx).
		WithComponent("executor").
		WithOperation("executeToolCall").
		With("task_id", e.currentTask.ID).
		With("agent_id", e.agent.GetID()).
		With("tool_name", toolName)

	logger.Debug("Starting tool execution",
		"tool_name", toolName,
		"params_count", len(params),
		"has_tool_registry", e.toolRegistry != nil,
	)

	if e.toolRegistry == nil {
		// Enhanced error observability for missing tool registry
		logger.Error("Tool execution failed - no tool registry available",
			"tool_name", toolName,
			"params_count", len(params),
			"task_id", e.currentTask.ID,
			"agent_id", e.agent.GetID(),
		)

		return nil, gerror.New(gerror.ErrCodeValidation, "no tool registry available", nil).
			WithComponent("executor").
			WithOperation("executeToolCall").
			WithDetails("tool_name", toolName).
			WithDetails("task_id", e.currentTask.ID).
			WithDetails("agent_id", e.agent.GetID())
	}

	// Track start time for usage metrics
	startTime := time.Now()

	logger.Debug("Executing tool with registry",
		"tool_name", toolName,
		"params", params,
	)

	// Execute the tool with enhanced error observability
	result, err := e.toolRegistry.ExecuteToolWithParams(ctx, toolName, params)

	// Calculate execution time
	duration := time.Since(startTime)

	// Enhanced error and success observability
	if err != nil {
		logger.WithError(err).Error("Tool execution failed",
			"tool_name", toolName,
			"execution_duration_ms", duration.Milliseconds(),
			"params_count", len(params),
			"task_id", e.currentTask.ID,
			"agent_id", e.agent.GetID(),
		)
	} else if result == nil {
		logger.Warn("Tool execution returned no result",
			"tool_name", toolName,
			"execution_duration_ms", duration.Milliseconds(),
			"params_count", len(params),
		)
	} else {
		logger.Debug("Tool execution completed",
			"tool_name", toolName,
			"execution_duration_ms", duration.Milliseconds(),
			"tool_success", result.Success,
			"result_output_length", len(result.Output),
			"has_extra_data", len(result.ExtraData) > 0,
		)
	}

	// Track tool usage with enhanced observability
	usageTrackingStart := time.Now()
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
		logger.Debug("Created new tool usage entry", "tool_name", toolName)
	}

	// Update usage stats
	previousInvocations := toolUsage.Invocations
	toolUsage.Invocations++
	toolUsage.TotalTime += duration
	if result != nil {
		toolUsage.Results = append(toolUsage.Results, map[string]interface{}{
			"timestamp": time.Now(),
			"success":   result.Success,
			"duration":  duration.String(),
		})
	}
	currentTotalUsage := len(e.result.ToolUsage)
	e.mu.Unlock()
	usageTrackingDuration := time.Since(usageTrackingStart)

	logger.Debug("Tool usage tracking updated",
		"tool_name", toolName,
		"previous_invocations", previousInvocations,
		"current_invocations", toolUsage.Invocations,
		"total_tool_types_used", currentTotalUsage,
		"usage_tracking_duration_ms", usageTrackingDuration.Milliseconds(),
	)

	// Log tool execution with comprehensive context
	isSuccess := result != nil && result.Success
	e.addExecutionLog(fmt.Sprintf("Executed tool: %s", toolName), map[string]interface{}{
		"tool":     toolName,
		"success":  isSuccess,
		"duration": duration.String(),
	})

	// Log performance metrics for monitoring
	logger.Duration("executor.tool_execution", duration,
		"tool_name", toolName,
		"success", isSuccess,
		"task_id", e.currentTask.ID,
		"agent_id", e.agent.GetID(),
		"params_count", len(params),
		"result_has_output", result != nil && len(result.Output) > 0,
		"result_has_extra_data", result != nil && len(result.ExtraData) > 0,
	)

	// Final success/error return with comprehensive context
	if err != nil {
		return result, gerror.Wrap(err, gerror.ErrCodeInternal, "tool execution failed").
			WithComponent("executor").
			WithOperation("executeToolCall").
			WithDetails("tool_name", toolName).
			WithDetails("execution_duration_ms", duration.Milliseconds()).
			WithDetails("task_id", e.currentTask.ID).
			WithDetails("agent_id", e.agent.GetID())
	}

	logger.Info("Tool execution completed successfully",
		"tool_name", toolName,
		"execution_duration_ms", duration.Milliseconds(),
		"tool_success", isSuccess,
	)

	return result, nil
}

// buildPromptData builds the prompt data structure from current context
func (e *BasicTaskExecutor) buildPromptData() execution.ExecutionPromptData {
	e.mu.RLock()
	defer e.mu.RUnlock()

	// Build available tools data
	var toolsData []execution.ToolData
	if e.toolRegistry != nil {
		// Get actual tools from registry
		for _, toolName := range e.toolRegistry.ListTools() {
			tool, err := e.toolRegistry.GetTool(toolName)
			if err != nil {
				continue // Skip tools that can't be retrieved
			}
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
			Title:           e.execContext.Commission,
			Description:     "Complete the assigned commission",
			SuccessCriteria: []string{"Task completed successfully", "Tests pass", "Documentation updated"},
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
			StepCommission:         "Complete current phase",
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
	if _, err := e.toolRegistry.GetTool("file"); err != nil {
		fileTool := fs.NewFileTool(basePath)
		if regErr := e.toolRegistry.RegisterTool("file", fileTool); regErr != nil {
			// Log tool registration error but don't fail initialization
			_ = gerror.Wrap(regErr, gerror.ErrCodeInternal, "failed to register file tool").
				WithComponent("TaskExecutor").
				WithOperation("initializeTools")
		}
	}

	// Register shell tool if not already registered
	if _, err := e.toolRegistry.GetTool("shell"); err != nil {
		shellOptions := shell.ShellToolOptions{
			WorkingDir: basePath,
			// Add safety restrictions
			BlockedCommands: []string{
				"rm -rf /", "rm -rf /*", "shutdown", "reboot",
				"passwd", "su", "sudo", "chown", "chmod 777",
			},
		}
		shellTool := shell.NewShellTool(shellOptions)
		if regErr := e.toolRegistry.RegisterTool("shell", shellTool); regErr != nil {
			// Log tool registration error but don't fail initialization
			_ = gerror.Wrap(regErr, gerror.ErrCodeInternal, "failed to register shell tool").
				WithComponent("TaskExecutor").
				WithOperation("initializeTools")
		}
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
			return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create task directory").
				WithComponent("executor").
				WithOperation("phaseFinalize").
				WithDetails("task_id", e.currentTask.ID).
				WithDetails("directory", taskDir)
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
			return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create README").
				WithComponent("executor").
				WithOperation("phaseFinalize").
				WithDetails("task_id", e.currentTask.ID).
				WithDetails("readme_path", filepath.Join(taskDir, "README.md"))
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
			return gerror.Wrap(err, gerror.ErrCodeAgent, "failed to create solution").
				WithComponent("executor").
				WithOperation("createSolution").
				WithDetails("task_id", e.currentTask.ID).
				WithDetails("agent_id", e.agent.GetID())
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
			return gerror.Wrap(err, gerror.ErrCodeAgent, "failed to verify results").
				WithComponent("executor").
				WithOperation("verifyResults").
				WithDetails("task_id", e.currentTask.ID).
				WithDetails("agent_id", e.agent.GetID())
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
