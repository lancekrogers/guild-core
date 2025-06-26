// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package commission

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/guild-ventures/guild-core/pkg/gerror"
	"github.com/guild-ventures/guild-core/pkg/observability"
)

// RefinementWriter generates refined commission documentation in markdown format
type RefinementWriter struct {
	outputDir string
}

// OutputResult contains information about generated files
type OutputResult struct {
	OutputDirectory    string    `json:"output_directory"`
	FilesGenerated     []string  `json:"files_generated"`
	AnalysisFile       string    `json:"analysis_file"`
	TasksFile          string    `json:"tasks_file"`
	ImplementationFile string    `json:"implementation_file"`
	AssignmentsFile    string    `json:"assignments_file"`
	TimelineFile       string    `json:"timeline_file"`
	GeneratedAt        time.Time `json:"generated_at"`
}

// NewRefinementWriter creates a new refinement documentation writer
func NewRefinementWriter(outputDir string) *RefinementWriter {
	return &RefinementWriter{
		outputDir: outputDir,
	}
}

// WriteRefined generates comprehensive markdown documentation for a refined commission
func (rw *RefinementWriter) WriteRefined(ctx context.Context, refined *RefinedCommission) (*OutputResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("commission.refinement_writer").
			WithOperation("WriteRefined")
	}

	logger := observability.GetLogger(ctx)
	ctx = observability.WithComponent(ctx, "commission.refinement_writer")
	ctx = observability.WithOperation(ctx, "WriteRefined")

	startTime := time.Now()
	logger.InfoContext(ctx, "Generating refined commission documentation",
		"commission_id", refined.Original.ID,
		"commission_title", refined.Original.Title,
		"output_dir", rw.outputDir)

	// Create refined directory structure
	refinedDir := filepath.Join(rw.outputDir, "refined", sanitizeFilename(refined.Original.Title))
	err := os.MkdirAll(refinedDir, 0755)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to create refined directory").
			WithComponent("commission.refinement_writer").
			WithOperation("WriteRefined").
			WithDetails("refined_dir", refinedDir)
	}

	result := &OutputResult{
		OutputDirectory: refinedDir,
		FilesGenerated:  make([]string, 0),
		GeneratedAt:     time.Now(),
	}

	// Generate analysis document
	analysisFile := filepath.Join(refinedDir, "analysis.md")
	err = rw.writeAnalysis(ctx, analysisFile, refined)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to write analysis file").
			WithComponent("commission.refinement_writer").
			WithOperation("WriteRefined")
	}
	result.AnalysisFile = analysisFile
	result.FilesGenerated = append(result.FilesGenerated, "analysis.md")

	// Generate task breakdown document
	tasksFile := filepath.Join(refinedDir, "tasks.md")
	err = rw.writeTasks(ctx, tasksFile, refined)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to write tasks file").
			WithComponent("commission.refinement_writer").
			WithOperation("WriteRefined")
	}
	result.TasksFile = tasksFile
	result.FilesGenerated = append(result.FilesGenerated, "tasks.md")

	// Generate implementation plan document
	planFile := filepath.Join(refinedDir, "implementation_plan.md")
	err = rw.writePlan(ctx, planFile, refined)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to write implementation plan").
			WithComponent("commission.refinement_writer").
			WithOperation("WriteRefined")
	}
	result.ImplementationFile = planFile
	result.FilesGenerated = append(result.FilesGenerated, "implementation_plan.md")

	// Generate agent assignments document
	assignmentsFile := filepath.Join(refinedDir, "agent_assignments.md")
	err = rw.writeAssignments(ctx, assignmentsFile, refined)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to write assignments file").
			WithComponent("commission.refinement_writer").
			WithOperation("WriteRefined")
	}
	result.AssignmentsFile = assignmentsFile
	result.FilesGenerated = append(result.FilesGenerated, "agent_assignments.md")

	// Generate timeline document
	timelineFile := filepath.Join(refinedDir, "timeline.md")
	err = rw.writeTimeline(ctx, timelineFile, refined)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to write timeline file").
			WithComponent("commission.refinement_writer").
			WithOperation("WriteRefined")
	}
	result.TimelineFile = timelineFile
	result.FilesGenerated = append(result.FilesGenerated, "timeline.md")

	// Generate README for the refined commission
	readmeFile := filepath.Join(refinedDir, "README.md")
	err = rw.writeReadme(ctx, readmeFile, refined)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to write README file").
			WithComponent("commission.refinement_writer").
			WithOperation("WriteRefined")
	}
	result.FilesGenerated = append(result.FilesGenerated, "README.md")

	duration := time.Since(startTime)
	logger.InfoContext(ctx, "Completed refined commission documentation generation",
		"commission_id", refined.Original.ID,
		"files_generated", len(result.FilesGenerated),
		"output_directory", refinedDir,
		"duration_ms", duration.Milliseconds())

	return result, nil
}

// writeAnalysis generates the analysis.md file
func (rw *RefinementWriter) writeAnalysis(ctx context.Context, filepath string, refined *RefinedCommission) error {
	var content strings.Builder

	// Header
	content.WriteString("# Commission Analysis\n\n")
	content.WriteString(fmt.Sprintf("**Commission:** %s\n", refined.Original.Title))
	content.WriteString(fmt.Sprintf("**Analysis Generated:** %s\n\n", refined.RefinedAt.Format("2006-01-02 15:04:05")))

	// Executive Summary
	content.WriteString("## Executive Summary\n\n")
	content.WriteString(fmt.Sprintf("- **Scope:** %s\n", refined.Analysis.Scope))
	content.WriteString(fmt.Sprintf("- **Estimated Effort:** %s\n", refined.Analysis.EstimatedEffort))
	content.WriteString(fmt.Sprintf("- **Total Tasks:** %d\n", len(refined.Tasks)))
	content.WriteString(fmt.Sprintf("- **Total Complexity:** %d points\n", refined.TotalComplexity))
	content.WriteString(fmt.Sprintf("- **Estimated Duration:** %.1f hours\n\n", refined.EstimatedDuration.Hours()))

	// Requirements Analysis
	content.WriteString("## Requirements Analysis\n\n")
	content.WriteString(fmt.Sprintf("**Total Requirements Identified:** %d\n\n", len(refined.Analysis.Requirements)))

	for i, req := range refined.Analysis.Requirements {
		content.WriteString(fmt.Sprintf("### Requirement %d: %s\n\n", i+1, req.Type))
		content.WriteString(fmt.Sprintf("**Priority:** %s\n\n", req.Priority))
		content.WriteString(fmt.Sprintf("**Description:** %s\n\n", req.Description))
		
		if len(req.Acceptance) > 0 {
			content.WriteString("**Acceptance Criteria:**\n")
			for _, criterion := range req.Acceptance {
				content.WriteString(fmt.Sprintf("- %s\n", criterion))
			}
			content.WriteString("\n")
		}
	}

	// Technical Stack
	if len(refined.Analysis.TechnicalStack) > 0 {
		content.WriteString("## Technical Stack\n\n")
		for _, tech := range refined.Analysis.TechnicalStack {
			content.WriteString(fmt.Sprintf("- %s\n", tech))
		}
		content.WriteString("\n")
	}

	// Key Deliverables
	if len(refined.Analysis.KeyDeliverables) > 0 {
		content.WriteString("## Key Deliverables\n\n")
		for _, deliverable := range refined.Analysis.KeyDeliverables {
			content.WriteString(fmt.Sprintf("- %s\n", deliverable))
		}
		content.WriteString("\n")
	}

	// Success Criteria
	if len(refined.Analysis.SuccessCriteria) > 0 {
		content.WriteString("## Success Criteria\n\n")
		for _, criterion := range refined.Analysis.SuccessCriteria {
			content.WriteString(fmt.Sprintf("- %s\n", criterion))
		}
		content.WriteString("\n")
	}

	// Risk Factors
	if len(refined.Analysis.RiskFactors) > 0 {
		content.WriteString("## Risk Factors\n\n")
		for _, risk := range refined.Analysis.RiskFactors {
			content.WriteString(fmt.Sprintf("- ⚠️ %s\n", risk))
		}
		content.WriteString("\n")
	}

	return writeFile(filepath, content.String())
}

// writeTasks generates the tasks.md file
func (rw *RefinementWriter) writeTasks(ctx context.Context, filepath string, refined *RefinedCommission) error {
	var content strings.Builder

	// Header
	content.WriteString("# Task Breakdown\n\n")
	content.WriteString(fmt.Sprintf("**Commission:** %s\n", refined.Original.Title))
	content.WriteString(fmt.Sprintf("**Total Tasks:** %d\n", len(refined.Tasks)))
	content.WriteString(fmt.Sprintf("**Total Complexity:** %d points\n\n", refined.TotalComplexity))

	// Task Summary by Type
	tasksByType := make(map[string][]*RefinedTask)
	for _, task := range refined.Tasks {
		tasksByType[task.Type] = append(tasksByType[task.Type], task)
	}

	content.WriteString("## Task Summary by Type\n\n")
	for taskType, tasks := range tasksByType {
		totalComplexity := 0
		for _, task := range tasks {
			totalComplexity += task.Complexity
		}
		content.WriteString(fmt.Sprintf("- **%s**: %d tasks (%d complexity points)\n", 
			strings.Title(taskType), len(tasks), totalComplexity))
	}
	content.WriteString("\n")

	// Detailed Task List
	content.WriteString("## Detailed Task List\n\n")

	// Sort tasks by type and then by complexity
	sortedTasks := make([]*RefinedTask, len(refined.Tasks))
	copy(sortedTasks, refined.Tasks)
	sort.Slice(sortedTasks, func(i, j int) bool {
		if sortedTasks[i].Type != sortedTasks[j].Type {
			return sortedTasks[i].Type < sortedTasks[j].Type
		}
		return sortedTasks[i].Complexity > sortedTasks[j].Complexity
	})

	currentType := ""
	for _, task := range sortedTasks {
		// Add type header if changed
		if task.Type != currentType {
			currentType = task.Type
			content.WriteString(fmt.Sprintf("### %s Tasks\n\n", strings.Title(currentType)))
		}

		// Task details
		content.WriteString(fmt.Sprintf("#### %s\n\n", task.Title))
		content.WriteString(fmt.Sprintf("- **ID:** %s\n", task.ID))
		content.WriteString(fmt.Sprintf("- **Type:** %s\n", task.Type))
		content.WriteString(fmt.Sprintf("- **Complexity:** %d points (%s)\n", 
			task.Complexity, getComplexityLabel(task.Complexity)))
		content.WriteString(fmt.Sprintf("- **Estimated Hours:** %.1f\n", task.EstimatedHours))
		
		if task.AssignedAgent != "" {
			content.WriteString(fmt.Sprintf("- **Assigned Agent:** %s\n", task.AssignedAgent))
		}
		
		if len(task.Dependencies) > 0 {
			content.WriteString(fmt.Sprintf("- **Dependencies:** %s\n", strings.Join(task.Dependencies, ", ")))
		}

		content.WriteString(fmt.Sprintf("\n**Description:** %s\n\n", task.Description))

		// Add metadata if present
		if len(task.Metadata) > 0 {
			content.WriteString("**Metadata:**\n")
			for key, value := range task.Metadata {
				content.WriteString(fmt.Sprintf("- %s: %s\n", key, value))
			}
			content.WriteString("\n")
		}

		content.WriteString("---\n\n")
	}

	return writeFile(filepath, content.String())
}

// writePlan generates the implementation_plan.md file
func (rw *RefinementWriter) writePlan(ctx context.Context, filepath string, refined *RefinedCommission) error {
	var content strings.Builder

	// Header
	content.WriteString("# Implementation Plan\n\n")
	content.WriteString(fmt.Sprintf("**Commission:** %s\n", refined.Original.Title))
	content.WriteString(fmt.Sprintf("**Plan Generated:** %s\n\n", refined.RefinedAt.Format("2006-01-02 15:04:05")))

	// Project Overview
	content.WriteString("## Project Overview\n\n")
	content.WriteString(fmt.Sprintf("**Scope:** %s\n", refined.Analysis.Scope))
	content.WriteString(fmt.Sprintf("**Estimated Duration:** %.1f hours (%.1f days)\n", 
		refined.EstimatedDuration.Hours(), refined.EstimatedDuration.Hours()/8))
	content.WriteString(fmt.Sprintf("**Start Date:** %s\n", refined.Timeline.StartDate.Format("2006-01-02")))
	content.WriteString(fmt.Sprintf("**End Date:** %s\n\n", refined.Timeline.EndDate.Format("2006-01-02")))

	// Implementation Phases
	content.WriteString("## Implementation Phases\n\n")

	// Group tasks by phase (based on type)
	phases := []string{"design", "implementation", "testing", "documentation"}
	for _, phase := range phases {
		phaseTasks := make([]*RefinedTask, 0)
		for _, task := range refined.Tasks {
			if task.Type == phase {
				phaseTasks = append(phaseTasks, task)
			}
		}

		if len(phaseTasks) == 0 {
			continue
		}

		content.WriteString(fmt.Sprintf("### Phase: %s\n\n", strings.Title(phase)))
		
		totalHours := 0.0
		totalComplexity := 0
		for _, task := range phaseTasks {
			totalHours += task.EstimatedHours
			totalComplexity += task.Complexity
		}

		content.WriteString(fmt.Sprintf("- **Tasks:** %d\n", len(phaseTasks)))
		content.WriteString(fmt.Sprintf("- **Estimated Hours:** %.1f\n", totalHours))
		content.WriteString(fmt.Sprintf("- **Complexity Points:** %d\n\n", totalComplexity))

		content.WriteString("**Tasks in this phase:**\n")
		for _, task := range phaseTasks {
			content.WriteString(fmt.Sprintf("- %s (%d points, %.1fh)\n", 
				task.Title, task.Complexity, task.EstimatedHours))
		}
		content.WriteString("\n")
	}

	// Milestones
	if len(refined.Timeline.Milestones) > 0 {
		content.WriteString("## Milestones\n\n")
		for i, milestone := range refined.Timeline.Milestones {
			content.WriteString(fmt.Sprintf("### Milestone %d: %s\n\n", i+1, milestone.Name))
			content.WriteString(fmt.Sprintf("**Target Date:** %s\n", milestone.TargetDate.Format("2006-01-02")))
			content.WriteString(fmt.Sprintf("**Description:** %s\n", milestone.Description))
			content.WriteString(fmt.Sprintf("**Associated Tasks:** %d\n\n", len(milestone.TaskIDs)))
		}
	}

	// Critical Path
	if len(refined.Timeline.CriticalPath) > 0 {
		content.WriteString("## Critical Path\n\n")
		content.WriteString("The following tasks are on the critical path and any delays will impact the project timeline:\n\n")
		for _, taskID := range refined.Timeline.CriticalPath {
			// Find the task
			for _, task := range refined.Tasks {
				if task.ID == taskID {
					content.WriteString(fmt.Sprintf("- **%s** (%d points, %.1fh)\n", 
						task.Title, task.Complexity, task.EstimatedHours))
					break
				}
			}
		}
		content.WriteString("\n")
	}

	// Recommendations
	content.WriteString("## Implementation Recommendations\n\n")
	content.WriteString("### Parallel Work Streams\n\n")
	content.WriteString("Tasks that can be worked on in parallel:\n")
	content.WriteString("- Design tasks can begin immediately\n")
	content.WriteString("- Documentation can be prepared alongside implementation\n")
	content.WriteString("- Testing can begin as soon as implementation tasks are complete\n\n")

	content.WriteString("### Risk Mitigation\n\n")
	if len(refined.Analysis.RiskFactors) > 0 {
		content.WriteString("Address the following risks early in the project:\n")
		for _, risk := range refined.Analysis.RiskFactors {
			content.WriteString(fmt.Sprintf("- %s\n", risk))
		}
	} else {
		content.WriteString("No significant risks identified. Regular progress reviews recommended.\n")
	}
	content.WriteString("\n")

	return writeFile(filepath, content.String())
}

// writeAssignments generates the agent_assignments.md file
func (rw *RefinementWriter) writeAssignments(ctx context.Context, filepath string, refined *RefinedCommission) error {
	var content strings.Builder

	// Header
	content.WriteString("# Agent Assignments\n\n")
	content.WriteString(fmt.Sprintf("**Commission:** %s\n", refined.Original.Title))
	content.WriteString(fmt.Sprintf("**Assignments Generated:** %s\n\n", refined.RefinedAt.Format("2006-01-02 15:04:05")))

	// Agent Resources Summary
	if refined.AgentResources != nil {
		content.WriteString("## Available Agent Resources\n\n")
		content.WriteString(fmt.Sprintf("**Total Agents:** %d\n", refined.AgentResources.TotalAgents))
		content.WriteString(fmt.Sprintf("**Total Capacity:** %.1f hours/week\n", refined.AgentResources.TotalCapacity))
		content.WriteString(fmt.Sprintf("**Cost Range:** %s\n\n", refined.AgentResources.CostRange))

		// Individual agent capabilities
		content.WriteString("### Agent Capabilities\n\n")
		for _, agent := range refined.AgentResources.AvailableAgents {
			content.WriteString(fmt.Sprintf("#### %s (%s)\n\n", agent.Name, agent.ID))
			content.WriteString(fmt.Sprintf("- **Type:** %s\n", agent.Type))
			content.WriteString(fmt.Sprintf("- **Specialty:** %s\n", agent.Specialty))
			content.WriteString(fmt.Sprintf("- **Cost Magnitude:** %d\n", agent.CostMagnitude))
			content.WriteString(fmt.Sprintf("- **Availability:** %s\n", agent.Availability))
			
			if len(agent.Capabilities) > 0 {
				content.WriteString(fmt.Sprintf("- **Capabilities:** %s\n", strings.Join(agent.Capabilities, ", ")))
			}
			
			if len(agent.Languages) > 0 {
				content.WriteString(fmt.Sprintf("- **Languages:** %s\n", strings.Join(agent.Languages, ", ")))
			}
			
			if len(agent.Frameworks) > 0 {
				content.WriteString(fmt.Sprintf("- **Frameworks:** %s\n", strings.Join(agent.Frameworks, ", ")))
			}
			
			content.WriteString("\n")
		}
	}

	// Task Assignments
	content.WriteString("## Task Assignments\n\n")

	if len(refined.Assignments) == 0 {
		content.WriteString("No agent assignments were made. Tasks will need to be assigned manually.\n\n")
	} else {
		// Summary by agent
		content.WriteString("### Assignment Summary\n\n")
		totalAssignedTasks := 0
		for agentID, tasks := range refined.Assignments {
			totalHours := 0.0
			totalComplexity := 0
			for _, task := range tasks {
				totalHours += task.EstimatedHours
				totalComplexity += task.Complexity
			}
			
			// Find agent name
			agentName := agentID
			if refined.AgentResources != nil {
				for _, agent := range refined.AgentResources.AvailableAgents {
					if agent.ID == agentID {
						agentName = agent.Name
						break
					}
				}
			}
			
			content.WriteString(fmt.Sprintf("- **%s** (%s): %d tasks, %.1f hours, %d complexity points\n",
				agentName, agentID, len(tasks), totalHours, totalComplexity))
			totalAssignedTasks += len(tasks)
		}
		
		unassignedTasks := len(refined.Tasks) - totalAssignedTasks
		if unassignedTasks > 0 {
			content.WriteString(fmt.Sprintf("- **Unassigned**: %d tasks\n", unassignedTasks))
		}
		content.WriteString("\n")

		// Detailed assignments
		content.WriteString("### Detailed Assignments\n\n")
		for agentID, tasks := range refined.Assignments {
			// Find agent info
			agentName := agentID
			agentType := "unknown"
			if refined.AgentResources != nil {
				for _, agent := range refined.AgentResources.AvailableAgents {
					if agent.ID == agentID {
						agentName = agent.Name
						agentType = agent.Type
						break
					}
				}
			}

			content.WriteString(fmt.Sprintf("#### %s (%s - %s)\n\n", agentName, agentID, agentType))

			// Sort tasks by complexity (highest first)
			sortedTasks := make([]*RefinedTask, len(tasks))
			copy(sortedTasks, tasks)
			sort.Slice(sortedTasks, func(i, j int) bool {
				return sortedTasks[i].Complexity > sortedTasks[j].Complexity
			})

			for _, task := range sortedTasks {
				content.WriteString(fmt.Sprintf("**%s**\n", task.Title))
				content.WriteString(fmt.Sprintf("- ID: %s\n", task.ID))
				content.WriteString(fmt.Sprintf("- Type: %s\n", task.Type))
				content.WriteString(fmt.Sprintf("- Complexity: %d points\n", task.Complexity))
				content.WriteString(fmt.Sprintf("- Estimated Hours: %.1f\n", task.EstimatedHours))
				if len(task.Dependencies) > 0 {
					content.WriteString(fmt.Sprintf("- Dependencies: %s\n", strings.Join(task.Dependencies, ", ")))
				}
				content.WriteString("\n")
			}
		}
	}

	// Unassigned tasks
	unassignedTasks := make([]*RefinedTask, 0)
	assignedTaskIDs := make(map[string]bool)
	for _, tasks := range refined.Assignments {
		for _, task := range tasks {
			assignedTaskIDs[task.ID] = true
		}
	}
	
	for _, task := range refined.Tasks {
		if !assignedTaskIDs[task.ID] {
			unassignedTasks = append(unassignedTasks, task)
		}
	}

	if len(unassignedTasks) > 0 {
		content.WriteString("## Unassigned Tasks\n\n")
		content.WriteString("The following tasks still need agent assignment:\n\n")
		
		for _, task := range unassignedTasks {
			content.WriteString(fmt.Sprintf("- **%s** (%s, %d points, %.1fh)\n",
				task.Title, task.Type, task.Complexity, task.EstimatedHours))
		}
		content.WriteString("\n")
	}

	return writeFile(filepath, content.String())
}

// writeTimeline generates the timeline.md file
func (rw *RefinementWriter) writeTimeline(ctx context.Context, filepath string, refined *RefinedCommission) error {
	var content strings.Builder

	// Header
	content.WriteString("# Project Timeline\n\n")
	content.WriteString(fmt.Sprintf("**Commission:** %s\n", refined.Original.Title))
	content.WriteString(fmt.Sprintf("**Timeline Generated:** %s\n\n", refined.RefinedAt.Format("2006-01-02 15:04:05")))

	// Timeline Overview
	content.WriteString("## Timeline Overview\n\n")
	content.WriteString(fmt.Sprintf("- **Start Date:** %s\n", refined.Timeline.StartDate.Format("2006-01-02")))
	content.WriteString(fmt.Sprintf("- **End Date:** %s\n", refined.Timeline.EndDate.Format("2006-01-02")))
	
	duration := refined.Timeline.EndDate.Sub(refined.Timeline.StartDate)
	content.WriteString(fmt.Sprintf("- **Duration:** %.0f days\n", duration.Hours()/24))
	content.WriteString(fmt.Sprintf("- **Buffer Days:** %d\n", refined.Timeline.BufferDays))
	content.WriteString(fmt.Sprintf("- **Total Effort:** %.1f hours\n\n", refined.EstimatedDuration.Hours()))

	// Milestones
	if len(refined.Timeline.Milestones) > 0 {
		content.WriteString("## Milestones\n\n")
		
		// Sort milestones by date
		milestones := make([]Milestone, len(refined.Timeline.Milestones))
		copy(milestones, refined.Timeline.Milestones)
		sort.Slice(milestones, func(i, j int) bool {
			return milestones[i].TargetDate.Before(milestones[j].TargetDate)
		})

		for i, milestone := range milestones {
			daysFromStart := milestone.TargetDate.Sub(refined.Timeline.StartDate).Hours() / 24
			content.WriteString(fmt.Sprintf("### %d. %s\n\n", i+1, milestone.Name))
			content.WriteString(fmt.Sprintf("- **Date:** %s (Day %.0f)\n", 
				milestone.TargetDate.Format("2006-01-02"), daysFromStart))
			content.WriteString(fmt.Sprintf("- **Description:** %s\n", milestone.Description))
			content.WriteString(fmt.Sprintf("- **Associated Tasks:** %d\n\n", len(milestone.TaskIDs)))

			// List associated tasks
			if len(milestone.TaskIDs) > 0 {
				content.WriteString("**Tasks to complete:**\n")
				for _, taskID := range milestone.TaskIDs {
					// Find the task
					for _, task := range refined.Tasks {
						if task.ID == taskID {
							content.WriteString(fmt.Sprintf("- %s (%d points)\n", task.Title, task.Complexity))
							break
						}
					}
				}
				content.WriteString("\n")
			}
		}
	}

	// Weekly Breakdown
	content.WriteString("## Weekly Breakdown\n\n")
	
	// Calculate weekly distribution
	weeks := int(duration.Hours()/24/7) + 1
	if weeks > 4 { // Don't generate excessive weeks for large projects
		weeks = 4
		content.WriteString("*Note: Showing first 4 weeks only for readability*\n\n")
	}

	for week := 1; week <= weeks; week++ {
		weekStart := refined.Timeline.StartDate.AddDate(0, 0, (week-1)*7)
		weekEnd := weekStart.AddDate(0, 0, 6)
		
		content.WriteString(fmt.Sprintf("### Week %d (%s - %s)\n\n", 
			week, weekStart.Format("Jan 2"), weekEnd.Format("Jan 2")))

		// Find tasks that should be worked on this week
		weekTasks := 0
		weekHours := 0.0
		content.WriteString("**Recommended focus:**\n")
		
		// Simple heuristic: distribute tasks evenly across weeks
		tasksPerWeek := len(refined.Tasks) / weeks
		startIdx := (week - 1) * tasksPerWeek
		endIdx := week * tasksPerWeek
		if week == weeks {
			endIdx = len(refined.Tasks) // Include remaining tasks in last week
		}

		for i := startIdx; i < endIdx && i < len(refined.Tasks); i++ {
			task := refined.Tasks[i]
			content.WriteString(fmt.Sprintf("- %s (%d points, %.1fh)\n", 
				task.Title, task.Complexity, task.EstimatedHours))
			weekTasks++
			weekHours += task.EstimatedHours
		}

		if weekTasks == 0 {
			content.WriteString("- Project completion and wrap-up\n")
		}

		content.WriteString(fmt.Sprintf("\n**Week Summary:** %d tasks, %.1f hours\n\n", weekTasks, weekHours))
	}

	// Critical Path
	if len(refined.Timeline.CriticalPath) > 0 {
		content.WriteString("## Critical Path Analysis\n\n")
		content.WriteString("The following tasks are on the critical path. Delays in these tasks will directly impact the project timeline:\n\n")

		for i, taskID := range refined.Timeline.CriticalPath {
			// Find the task
			for _, task := range refined.Tasks {
				if task.ID == taskID {
					content.WriteString(fmt.Sprintf("%d. **%s**\n", i+1, task.Title))
					content.WriteString(fmt.Sprintf("   - Complexity: %d points\n", task.Complexity))
					content.WriteString(fmt.Sprintf("   - Estimated Hours: %.1f\n", task.EstimatedHours))
					if task.AssignedAgent != "" {
						content.WriteString(fmt.Sprintf("   - Assigned: %s\n", task.AssignedAgent))
					}
					content.WriteString("\n")
					break
				}
			}
		}
	}

	return writeFile(filepath, content.String())
}

// writeReadme generates the README.md file
func (rw *RefinementWriter) writeReadme(ctx context.Context, filepath string, refined *RefinedCommission) error {
	var content strings.Builder

	// Header
	content.WriteString(fmt.Sprintf("# %s - Refined Commission\n\n", refined.Original.Title))
	content.WriteString(fmt.Sprintf("**Generated:** %s\n\n", refined.RefinedAt.Format("2006-01-02 15:04:05")))

	// Quick Summary
	content.WriteString("## Quick Summary\n\n")
	content.WriteString(fmt.Sprintf("- **Scope:** %s\n", refined.Analysis.Scope))
	content.WriteString(fmt.Sprintf("- **Total Tasks:** %d\n", len(refined.Tasks)))
	content.WriteString(fmt.Sprintf("- **Complexity Points:** %d\n", refined.TotalComplexity))
	content.WriteString(fmt.Sprintf("- **Estimated Duration:** %.1f hours (%.1f days)\n", 
		refined.EstimatedDuration.Hours(), refined.EstimatedDuration.Hours()/8))
	content.WriteString(fmt.Sprintf("- **Agent Assignments:** %d agents\n\n", len(refined.Assignments)))

	// Description
	content.WriteString("## Description\n\n")
	content.WriteString(fmt.Sprintf("%s\n\n", refined.Original.Description))

	// Files in this refinement
	content.WriteString("## Refinement Files\n\n")
	content.WriteString("This refined commission includes the following documentation:\n\n")
	content.WriteString("- **[analysis.md](analysis.md)** - Detailed requirements analysis and technical assessment\n")
	content.WriteString("- **[tasks.md](tasks.md)** - Complete task breakdown with complexity estimates\n")
	content.WriteString("- **[implementation_plan.md](implementation_plan.md)** - Structured implementation approach and phases\n")
	content.WriteString("- **[agent_assignments.md](agent_assignments.md)** - Agent capabilities and task assignments\n")
	content.WriteString("- **[timeline.md](timeline.md)** - Project timeline, milestones, and critical path\n\n")

	// Next Steps
	content.WriteString("## Next Steps\n\n")
	content.WriteString("1. **Review the Analysis** - Examine the requirements analysis and technical assessment\n")
	content.WriteString("2. **Validate Task Breakdown** - Review the proposed tasks and complexity estimates\n")
	content.WriteString("3. **Approve Agent Assignments** - Confirm or adjust the proposed agent assignments\n")
	content.WriteString("4. **Import to Kanban** - Use the Guild framework to import tasks to the kanban board\n")
	content.WriteString("5. **Begin Implementation** - Start work according to the implementation plan\n\n")

	// Import Instructions
	content.WriteString("## Importing to Kanban\n\n")
	content.WriteString("To import these refined tasks to your Guild kanban board:\n\n")
	content.WriteString("```bash\n")
	content.WriteString("# Using the Guild CLI\n")
	content.WriteString(fmt.Sprintf("guild kanban import-refined %s\n", refined.Original.ID))
	content.WriteString("```\n\n")
	content.WriteString("Or use the `/add-task` command in Guild chat to manually add tasks.\n\n")

	return writeFile(filepath, content.String())
}

// Helper functions

func writeFile(filepath, content string) error {
	return os.WriteFile(filepath, []byte(content), 0644)
}


func getComplexityLabel(complexity int) string {
	labels := map[int]string{
		1: "Very Simple",
		2: "Simple", 
		3: "Medium-Simple",
		5: "Medium-Complex",
		8: "Very Complex",
	}
	
	if label, exists := labels[complexity]; exists {
		return label
	}
	
	return "Unknown"
}