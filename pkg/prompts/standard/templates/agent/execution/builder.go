// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package execution

import (
	"bytes"
	"fmt"
	"text/template"
	"time"

	"github.com/lancekrogers/guild/pkg/gerror"
)

// Layer represents a prompt layer type
type Layer string

const (
	LayerBase      Layer = "base"
	LayerContext   Layer = "context"
	LayerTask      Layer = "task"
	LayerTool      Layer = "tool"
	LayerExecution Layer = "execution"
)

// PromptBuilder builds layered prompts for agent execution
type PromptBuilder struct {
	templates map[Layer]*template.Template
	loader    *Loader
}

// NewPromptBuilder creates a new execution prompt builder
func NewPromptBuilder() (*PromptBuilder, error) {
	loader := NewLoader()
	builder := &PromptBuilder{
		templates: make(map[Layer]*template.Template),
		loader:    loader,
	}

	// Load all layer templates
	layers := []struct {
		layer Layer
		file  string
	}{
		{LayerBase, "base_layer.md"},
		{LayerContext, "context_layer.md"},
		{LayerTask, "task_layer.md"},
		{LayerTool, "tool_layer.md"},
		{LayerExecution, "execution_layer.md"},
	}

	for _, l := range layers {
		content, err := loader.LoadPrompt(l.file)
		if err != nil {
			return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to load prompt layer").
				WithComponent("prompts").
				WithOperation("NewPromptBuilder").
				WithDetails("layer", string(l.layer)).
				WithDetails("file", l.file)
		}

		tmpl, err := template.New(string(l.layer)).Parse(content)
		if err != nil {
			return nil, gerror.Wrap(err, gerror.ErrCodeValidation, "failed to parse template").
				WithComponent("prompts").
				WithOperation("NewPromptBuilder").
				WithDetails("layer", string(l.layer))
		}

		builder.templates[l.layer] = tmpl
	}

	return builder, nil
}

// BuildPrompt builds a complete prompt with specified layers and data
func (b *PromptBuilder) BuildPrompt(layers []Layer, data map[string]interface{}) (string, error) {
	var result bytes.Buffer

	// Add timestamp and metadata
	result.WriteString(fmt.Sprintf("<!-- Generated: %s -->\n", time.Now().UTC().Format(time.RFC3339)))
	result.WriteString(fmt.Sprintf("<!-- Layers: %v -->\n\n", layers))

	// Process each requested layer
	for i, layer := range layers {
		tmpl, exists := b.templates[layer]
		if !exists {
			return "", gerror.New(gerror.ErrCodeValidation, "unknown prompt layer", nil).
				WithComponent("prompts").
				WithOperation("BuildPrompt").
				WithDetails("layer", string(layer))
		}

		// Add layer separator
		if i > 0 {
			result.WriteString("\n---\n\n")
		}

		// Execute template with data
		if err := tmpl.Execute(&result, data); err != nil {
			return "", gerror.Wrap(err, gerror.ErrCodeInternal, "failed to execute template").
				WithComponent("prompts").
				WithOperation("BuildPrompt").
				WithDetails("layer", string(layer))
		}
	}

	return result.String(), nil
}

// BuildFullExecutionPrompt builds a complete execution prompt with all layers
func (b *PromptBuilder) BuildFullExecutionPrompt(data ExecutionPromptData) (string, error) {
	// Convert to generic map for template execution
	dataMap := map[string]interface{}{
		// Base layer data
		"AgentName":    data.Agent.Name,
		"AgentRole":    data.Agent.Role,
		"Capabilities": data.Agent.Capabilities,
		"GuildID":      data.Context.GuildID,
		"ProjectName":  data.Context.ProjectName,
		"WorkspaceDir": data.Context.WorkspaceDir,

		// Context layer data
		"CommissionTitle":       data.Commission.Title,
		"CommissionDescription": data.Commission.Description,
		"SuccessCriteria":       data.Commission.SuccessCriteria,
		"ProjectDescription":    data.Context.ProjectDescription,
		"RelevantDocs":          data.Context.RelevantDocs,
		"TechStack":             data.Context.TechStack,
		"Architecture":          data.Context.Architecture,
		"ProjectDependencies":   data.Context.Dependencies,
		"RelatedTasks":          data.Context.RelatedTasks,

		// Task layer data
		"TaskTitle":        data.Task.Title,
		"TaskDescription":  data.Task.Description,
		"Requirements":     data.Task.Requirements,
		"Constraints":      data.Task.Constraints,
		"Priority":         data.Task.Priority,
		"DueDate":          data.Task.DueDate,
		"EstimatedHours":   data.Task.EstimatedHours,
		"TaskDependencies": data.Task.Dependencies,
		"Deliverables":     data.Task.Deliverables,

		// Tool layer data
		"Tools":        data.Tools,
		"MaxToolCalls": data.ToolConfig.MaxCalls,
		"ToolTimeout":  data.ToolConfig.Timeout,
		"RateLimits":   data.ToolConfig.RateLimits,

		// Execution layer data
		"Phase":                  data.Execution.Phase,
		"StepNumber":             data.Execution.StepNumber,
		"TotalSteps":             data.Execution.TotalSteps,
		"StepName":               data.Execution.StepName,
		"StepCommission":         data.Execution.StepCommission,
		"ExpectedActions":        data.Execution.ExpectedActions,
		"SuccessIndicators":      data.Execution.SuccessIndicators,
		"PotentialIssues":        data.Execution.PotentialIssues,
		"OverallProgress":        data.Execution.OverallProgress,
		"PhaseProgress":          data.Execution.PhaseProgress,
		"TimeElapsed":            data.Execution.TimeElapsed,
		"EstimatedTimeRemaining": data.Execution.EstimatedTimeRemaining,
		"PreviousStepResult":     data.Execution.PreviousStepResult,
		"NextSteps":              data.Execution.NextSteps,
	}

	// Use all layers for full execution prompt
	allLayers := []Layer{LayerBase, LayerContext, LayerTask, LayerTool, LayerExecution}
	return b.BuildPrompt(allLayers, dataMap)
}

// BuildPlanningPrompt builds a prompt for the planning phase (without execution layer)
func (b *PromptBuilder) BuildPlanningPrompt(data ExecutionPromptData) (string, error) {
	// Similar to BuildFullExecutionPrompt but only uses first 4 layers
	dataMap := map[string]interface{}{
		// ... same data mapping as above, minus execution layer
		"AgentName":             data.Agent.Name,
		"AgentRole":             data.Agent.Role,
		"Capabilities":          data.Agent.Capabilities,
		"GuildID":               data.Context.GuildID,
		"ProjectName":           data.Context.ProjectName,
		"WorkspaceDir":          data.Context.WorkspaceDir,
		"CommissionTitle":       data.Commission.Title,
		"CommissionDescription": data.Commission.Description,
		"SuccessCriteria":       data.Commission.SuccessCriteria,
		"ProjectDescription":    data.Context.ProjectDescription,
		"RelevantDocs":          data.Context.RelevantDocs,
		"TechStack":             data.Context.TechStack,
		"Architecture":          data.Context.Architecture,
		"ProjectDependencies":   data.Context.Dependencies,
		"RelatedTasks":          data.Context.RelatedTasks,
		"TaskTitle":             data.Task.Title,
		"TaskDescription":       data.Task.Description,
		"Requirements":          data.Task.Requirements,
		"Constraints":           data.Task.Constraints,
		"Priority":              data.Task.Priority,
		"DueDate":               data.Task.DueDate,
		"EstimatedHours":        data.Task.EstimatedHours,
		"TaskDependencies":      data.Task.Dependencies,
		"Deliverables":          data.Task.Deliverables,
		"Tools":                 data.Tools,
		"MaxToolCalls":          data.ToolConfig.MaxCalls,
		"ToolTimeout":           data.ToolConfig.Timeout,
		"RateLimits":            data.ToolConfig.RateLimits,
	}

	// Use layers without execution for planning
	planningLayers := []Layer{LayerBase, LayerContext, LayerTask, LayerTool}
	return b.BuildPrompt(planningLayers, dataMap)
}

// GetLayerTemplate returns the template for a specific layer (for testing)
func (b *PromptBuilder) GetLayerTemplate(layer Layer) *template.Template {
	return b.templates[layer]
}
