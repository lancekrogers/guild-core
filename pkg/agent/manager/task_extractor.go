// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package manager

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/guild-ventures/guild-core/pkg/gerror"
	"github.com/guild-ventures/guild-core/pkg/prompts/layered"
	"github.com/guild-ventures/guild-core/pkg/prompts/standard/templates/agent/extraction"
)

// TaskExtractor uses layered prompts and LLM intelligence to extract tasks from refined content
type TaskExtractor struct {
	artisanClient ArtisanClient
	promptManager layered.LayeredManager
}

// NewTaskExtractor creates a new task extractor
func NewTaskExtractor(
	artisanClient ArtisanClient,
	promptManager layered.LayeredManager,
) *TaskExtractor {
	return &TaskExtractor{
		artisanClient: artisanClient,
		promptManager: promptManager,
	}
}

// ExtractedTask represents a task extracted by the LLM
type ExtractedTask struct {
	ID                   string                `json:"id"`
	Title                string                `json:"title"`
	Description          string                `json:"description"`
	Category             string                `json:"category"`
	Priority             string                `json:"priority"`
	EstimatedHours       *float64              `json:"estimatedHours"`
	Dependencies         []string              `json:"dependencies"`
	RequiredCapabilities []string              `json:"requiredCapabilities"`
	Metadata             ExtractedTaskMetadata `json:"metadata"`
}

// ExtractedTaskMetadata contains additional task information
type ExtractedTaskMetadata struct {
	SourceSection      string   `json:"sourceSection"`
	Rationale          string   `json:"rationale"`
	AcceptanceCriteria []string `json:"acceptanceCriteria,omitempty"`
	TechnicalNotes     string   `json:"technicalNotes,omitempty"`
}

// ExtractionResult contains all extracted tasks and metadata
type ExtractionResult struct {
	ExtractionMetadata ExtractionMetadata `json:"extractionMetadata"`
	Tasks              []ExtractedTask    `json:"tasks"`
	TaskRelationships  TaskRelationships  `json:"taskRelationships"`
}

// ExtractionMetadata contains information about the extraction process
type ExtractionMetadata struct {
	CommissionID    string          `json:"commissionId"`
	ExtractedAt     string          `json:"extractedAt"`
	TotalTasks      int             `json:"totalTasks"`
	ContentAnalysis ContentAnalysis `json:"contentAnalysis"`
}

// ContentAnalysis provides insights about the analyzed content
type ContentAnalysis struct {
	Structure    string `json:"structure"`
	Completeness string `json:"completeness"`
	Clarity      string `json:"clarity"`
}

// TaskRelationships describes how tasks relate to each other
type TaskRelationships struct {
	Phases       []TaskPhase `json:"phases"`
	CriticalPath []string    `json:"criticalPath"`
}

// TaskPhase groups related tasks
type TaskPhase struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	TaskIDs     []string `json:"taskIds"`
}

// ExtractTasks uses layered prompts to extract tasks from refined content
func (te *TaskExtractor) ExtractTasks(ctx context.Context, refinedCommission *RefinedCommission) (*ExtractionResult, error) {
	// Build the layered prompt for task extraction
	prompt, err := te.buildExtractionPrompt(ctx, refinedCommission)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to build extraction prompt").
			WithComponent("manager").
			WithOperation("ExtractTasks").
			WithDetails("commission_id", refinedCommission.CommissionID)
	}

	// Call the LLM to extract tasks
	response, err := te.artisanClient.Complete(ctx, ArtisanRequest{
		SystemPrompt: prompt,
		UserPrompt:   "Extract all actionable tasks from the refined commission content provided in the system prompt. Output valid JSON following the specified format.",
		Temperature:  0.3,  // Lower temperature for more consistent extraction
		MaxTokens:    8000, // Enough for comprehensive task lists
	})
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeAgent, "failed to extract tasks").
			WithComponent("manager").
			WithOperation("ExtractTasks").
			WithDetails("commission_id", refinedCommission.CommissionID)
	}

	// Parse the JSON response
	var result ExtractionResult
	if err := json.Unmarshal([]byte(response.Content), &result); err != nil {
		// If JSON parsing fails, try to extract JSON from the response
		jsonContent := extractJSON(response.Content)
		if jsonContent == "" {
			return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to parse extraction result").
				WithComponent("manager").
				WithOperation("ExtractTasks").
				WithDetails("commission_id", refinedCommission.CommissionID).
				WithDetails("response_length", len(response.Content))
		}
		if err := json.Unmarshal([]byte(jsonContent), &result); err != nil {
			return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to parse extracted JSON").
				WithComponent("manager").
				WithOperation("ExtractTasks").
				WithDetails("commission_id", refinedCommission.CommissionID).
				WithDetails("json_length", len(jsonContent))
		}
	}

	// Validate and enhance the result
	te.validateAndEnhanceResult(&result, refinedCommission)

	return &result, nil
}

// buildExtractionPrompt creates a layered prompt for task extraction
func (te *TaskExtractor) buildExtractionPrompt(ctx context.Context, refinedCommission *RefinedCommission) (string, error) {
	// Prepare the prompt data
	promptData := te.preparePromptData(refinedCommission)

	// Load prompt templates
	baseLayer, err := te.loadPromptTemplate(ctx, "internal/prompts/agent/extraction/base_layer.md")
	if err != nil {
		return "", gerror.Wrap(err, gerror.ErrCodeInternal, "failed to load base layer").
			WithComponent("manager").
			WithOperation("buildExtractionPrompt")
	}

	guildLayer, err := te.loadPromptTemplate(ctx, "internal/prompts/agent/extraction/guild_layer.md")
	if err != nil {
		return "", gerror.Wrap(err, gerror.ErrCodeInternal, "failed to load guild layer").
			WithComponent("manager").
			WithOperation("buildExtractionPrompt")
	}

	domainLayer, err := te.loadPromptTemplate(ctx, "internal/prompts/agent/extraction/domain_layer.md")
	if err != nil {
		return "", gerror.Wrap(err, gerror.ErrCodeInternal, "failed to load domain layer").
			WithComponent("manager").
			WithOperation("buildExtractionPrompt")
	}

	contextLayer, err := te.loadPromptTemplate(ctx, "internal/prompts/agent/extraction/context_layer.md")
	if err != nil {
		return "", gerror.Wrap(err, gerror.ErrCodeInternal, "failed to load context layer").
			WithComponent("manager").
			WithOperation("buildExtractionPrompt")
	}

	contentLayer, err := te.loadPromptTemplate(ctx, "internal/prompts/agent/extraction/content_layer.md")
	if err != nil {
		return "", gerror.Wrap(err, gerror.ErrCodeInternal, "failed to load content layer").
			WithComponent("manager").
			WithOperation("buildExtractionPrompt")
	}

	executionLayer, err := te.loadPromptTemplate(ctx, "internal/prompts/agent/extraction/execution_layer.md")
	if err != nil {
		return "", gerror.Wrap(err, gerror.ErrCodeInternal, "failed to load execution layer").
			WithComponent("manager").
			WithOperation("buildExtractionPrompt")
	}

	// Apply template data to variable layers
	domainLayer = te.renderTemplate(domainLayer, promptData)
	contextLayer = te.renderTemplate(contextLayer, promptData)
	contentLayer = te.renderTemplate(contentLayer, promptData)

	// Concatenate all layers in order
	return te.concatenatePrompts([]string{
		baseLayer,
		guildLayer,
		domainLayer,
		contextLayer,
		contentLayer,
		executionLayer,
	}), nil
}

// preparePromptData creates the data structure for prompt templates
func (te *TaskExtractor) preparePromptData(refinedCommission *RefinedCommission) map[string]interface{} {
	// Get domain type from metadata
	domainType := "general"
	if dt, ok := refinedCommission.Metadata["domain"].(string); ok {
		domainType = dt
	}

	// Get commission title
	title := fmt.Sprintf("Commission %s", refinedCommission.CommissionID)
	if t, ok := refinedCommission.Metadata["original_title"].(string); ok {
		title = t
	}

	// Combine all file contents
	var refinedContent string
	for _, file := range refinedCommission.Structure.Files {
		refinedContent += fmt.Sprintf("## File: %s\n\n%s\n\n", file.Path, file.Content)
	}

	// Build domain context based on type
	domainContext := te.getDomainContext(domainType)

	return map[string]interface{}{
		"DomainType":      domainType,
		"DomainContext":   domainContext,
		"CommissionID":    refinedCommission.CommissionID,
		"CommissionTitle": title,
		"CommissionGoals": "Extract all actionable tasks from the refined commission",
		"RefinedContent":  refinedContent,
		"ContentFormat":   "Markdown with hierarchical structure",
	}
}

// getDomainContext provides domain-specific context
func (te *TaskExtractor) getDomainContext(domainType string) string {
	contexts := map[string]string{
		"web-app": `Web applications typically include:
- Frontend user interfaces with components and routing
- Backend APIs with authentication and data management
- Database design and migrations
- Deployment and hosting configuration
- Testing at multiple levels (unit, integration, e2e)`,

		"cli-tool": `CLI tools typically include:
- Command parsing and validation
- Core functionality implementation
- Configuration management
- Output formatting and user feedback
- Installation and distribution`,

		"library": `Libraries typically include:
- Core API design and implementation
- Documentation and examples
- Testing and benchmarks
- Version management and compatibility
- Distribution and packaging`,

		"microservice": `Microservices typically include:
- Service API definition
- Business logic implementation
- Inter-service communication
- Data persistence and caching
- Monitoring and observability
- Container and orchestration setup`,

		"general": `Software projects typically include:
- Architecture and design
- Core functionality implementation
- Testing and quality assurance
- Documentation
- Deployment and operations`,
	}

	if context, exists := contexts[domainType]; exists {
		return context
	}
	return contexts["general"]
}

// loadPromptTemplate loads a prompt template file
func (te *TaskExtractor) loadPromptTemplate(ctx context.Context, path string) (string, error) {
	return extraction.LoadPromptByPath(path)
}

// renderTemplate applies data to a template (simple implementation)
func (te *TaskExtractor) renderTemplate(template string, data map[string]interface{}) string {
	// Simple template rendering - in production use a proper template engine
	result := template
	for key, value := range data {
		placeholder := fmt.Sprintf("{{.%s}}", key)
		result = strings.ReplaceAll(result, placeholder, fmt.Sprintf("%v", value))
	}
	return result
}

// concatenatePrompts joins prompts with proper separation
func (te *TaskExtractor) concatenatePrompts(prompts []string) string {
	return strings.Join(prompts, "\n\n---\n\n")
}

// validateAndEnhanceResult ensures the extraction result is valid and complete
func (te *TaskExtractor) validateAndEnhanceResult(result *ExtractionResult, refinedCommission *RefinedCommission) {
	// Ensure metadata is complete
	if result.ExtractionMetadata.CommissionID == "" {
		result.ExtractionMetadata.CommissionID = refinedCommission.CommissionID
	}
	if result.ExtractionMetadata.ExtractedAt == "" {
		result.ExtractionMetadata.ExtractedAt = time.Now().UTC().Format(time.RFC3339)
	}
	result.ExtractionMetadata.TotalTasks = len(result.Tasks)

	// Validate task IDs are unique
	idMap := make(map[string]bool)
	for i, task := range result.Tasks {
		if idMap[task.ID] {
			// Generate a new ID if duplicate
			result.Tasks[i].ID = fmt.Sprintf("%s-%d", task.ID, i)
		}
		idMap[task.ID] = true
	}

	// Ensure all tasks have required fields
	for i := range result.Tasks {
		if result.Tasks[i].Category == "" {
			result.Tasks[i].Category = "OTHER"
		}
		if result.Tasks[i].Priority == "" {
			result.Tasks[i].Priority = "medium"
		}
		if result.Tasks[i].Dependencies == nil {
			result.Tasks[i].Dependencies = []string{}
		}
		if result.Tasks[i].RequiredCapabilities == nil {
			result.Tasks[i].RequiredCapabilities = []string{}
		}
	}
}

// extractJSON attempts to extract JSON from a text response
func extractJSON(content string) string {
	// Look for JSON between ```json and ``` markers
	start := strings.Index(content, "```json")
	if start == -1 {
		// Try without json marker
		start = strings.Index(content, "```")
		if start == -1 {
			// Try to find raw JSON
			start = strings.Index(content, "{")
			if start == -1 {
				return ""
			}
			// Find the last closing brace
			end := strings.LastIndex(content, "}")
			if end == -1 || end <= start {
				return ""
			}
			return content[start : end+1]
		}
	}

	// Skip past the marker
	jsonStart := strings.Index(content[start:], "\n")
	if jsonStart == -1 {
		return ""
	}
	start += jsonStart + 1

	// Find the closing marker
	end := strings.Index(content[start:], "```")
	if end == -1 {
		return ""
	}

	return strings.TrimSpace(content[start : start+end])
}

// ConvertToTaskInfo converts an ExtractedTask to TaskInfo for compatibility
func (et *ExtractedTask) ConvertToTaskInfo() TaskInfo {
	// Extract category and number from ID if it follows CATEGORY-NNN format
	parts := strings.Split(et.ID, "-")
	category := ""
	number := ""
	if len(parts) >= 2 {
		category = parts[0]
		number = parts[1]
	}

	// Convert estimated hours to string
	estimate := ""
	if et.EstimatedHours != nil {
		hours := *et.EstimatedHours
		if hours < 8 {
			estimate = fmt.Sprintf("%.0fh", hours)
		} else if hours < 40 {
			estimate = fmt.Sprintf("%.0fd", hours/8)
		} else {
			estimate = fmt.Sprintf("%.0fw", hours/40)
		}
	}

	return TaskInfo{
		ID:           et.ID,
		Category:     category,
		Number:       number,
		Title:        et.Title,
		Description:  et.Description,
		Priority:     et.Priority,
		Estimate:     estimate,
		Dependencies: et.Dependencies,
		Section:      et.Metadata.SourceSection,
	}
}

// ConvertExtractionResultToTaskInfos converts all tasks to TaskInfo format
func ConvertExtractionResultToTaskInfos(result *ExtractionResult) []TaskInfo {
	taskInfos := make([]TaskInfo, 0, len(result.Tasks))
	for _, task := range result.Tasks {
		taskInfos = append(taskInfos, task.ConvertToTaskInfo())
	}
	return taskInfos
}
