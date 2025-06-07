package manager

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/guild-ventures/guild-core/pkg/gerror"
	"github.com/guild-ventures/guild-core/pkg/kanban"
)

// ResponseParserImpl implements the ResponseParser interface
// It parses LLM responses containing hierarchical markdown structures
// and extracts tasks for the kanban board
type ResponseParserImpl struct {
	// Patterns for parsing different sections
	filePattern     *regexp.Regexp
	taskPattern     *regexp.Regexp
	headerPattern   *regexp.Regexp
	metadataPattern *regexp.Regexp
}

// NewResponseParser creates a new response parser
func NewResponseParser() *ResponseParserImpl {
	return &ResponseParserImpl{
		// Match file sections: ## File: path/to/file.md
		filePattern: regexp.MustCompile(`(?m)^##\s+File:\s+(.+)$`),

		// Match task definitions with various formats
		// - CATEGORY-NUMBER: Title (priority: high, estimate: 2h)
		// - [ ] Task description
		// - Task: description
		taskPattern: regexp.MustCompile(`(?m)^[\s-]*(?:\[[ x]\]|\*|-)?\s*(?:Task:|TASK-\d+:|[A-Z]+-\d+:)?\s*(.+?)(?:\s*\(.*?\))?\s*$`),

		// Match headers for sections
		headerPattern: regexp.MustCompile(`(?m)^(#{1,6})\s+(.+)$`),

		// Match metadata in parentheses
		metadataPattern: regexp.MustCompile(`\((.*?)\)`),
	}
}

// ParseResponse implements the ResponseParser interface
func (p *ResponseParserImpl) ParseResponse(response *ArtisanResponse) (*FileStructure, error) {
	return p.ParseResponseWithContext(context.Background(), response)
}

// ParseResponseWithContext implements parsing with context support
func (p *ResponseParserImpl) ParseResponseWithContext(ctx context.Context, response *ArtisanResponse) (*FileStructure, error) {
	if response == nil || response.Content == "" {
		return nil, gerror.New(gerror.ErrCodeValidation, "empty response content", nil).
			WithComponent("manager").
			WithOperation("ParseResponseWithContext")
	}

	content := response.Content

	// Try to parse as structured file response first
	if structure := p.parseFileStructure(content); structure != nil && len(structure.Files) > 0 {
		return structure, nil
	}

	// Otherwise, parse as a single hierarchical document
	return p.parseSingleDocument(content)
}

// parseFileStructure attempts to parse multiple file definitions
func (p *ResponseParserImpl) parseFileStructure(content string) *FileStructure {
	// Split content by file markers
	parts := p.filePattern.Split(content, -1)
	if len(parts) <= 1 {
		return nil
	}

	matches := p.filePattern.FindAllStringSubmatch(content, -1)
	files := make([]*FileEntry, 0, len(matches))

	for i, match := range matches {
		if len(match) < 2 {
			continue
		}

		filePath := strings.TrimSpace(match[1])
		// Get the content after this file marker and before the next
		fileContent := ""
		if i+1 < len(parts) {
			fileContent = strings.TrimSpace(parts[i+1])
		}

		// Extract tasks from this file
		tasks := p.extractTasks(fileContent)

		files = append(files, &FileEntry{
			Path:       filePath,
			Content:    fileContent,
			Type:       FileTypeMarkdown,
			TasksCount: len(tasks),
			Metadata: map[string]interface{}{
				"tasks":      tasks,
				"source":     "llm_response",
				"parser":     "response_parser",
			},
		})
	}

	return &FileStructure{
		RootDir: ".",
		Files:   files,
	}
}

// parseSingleDocument parses a single hierarchical document
func (p *ResponseParserImpl) parseSingleDocument(content string) (*FileStructure, error) {
	// Extract all tasks from the content
	tasks := p.extractTasks(content)

	// Create the main file entry
	mainFile := &FileEntry{
		Path:       "commission_refined.md",
		Content:    content,
		Type:       FileTypeMarkdown,
		TasksCount: len(tasks),
		Metadata: map[string]interface{}{
			"tasks":      tasks,
			"source":     "llm_response",
			"parser":     "response_parser",
			"single_doc": true,
		},
	}

	// If we found tasks in a specific structure, create task files
	files := []*FileEntry{mainFile}

	// Group tasks by category if they follow CATEGORY-NUMBER pattern
	tasksByCategory := p.groupTasksByCategory(tasks)
	for category, categoryTasks := range tasksByCategory {
		if category == "" || category == "general" {
			continue
		}

		// Create a file for this category (without duplicating task metadata)
		taskContent := p.formatTasksAsMarkdown(categoryTasks)
		files = append(files, &FileEntry{
			Path:       fmt.Sprintf("tasks/%s_tasks.md", strings.ToLower(category)),
			Content:    taskContent,
			Type:       FileTypeMarkdown,
			TasksCount: len(categoryTasks),
			Metadata: map[string]interface{}{
				"category": category,
				// Don't duplicate tasks in metadata - they're already in the main file
			},
		})
	}

	return &FileStructure{
		RootDir: ".",
		Files:   files,
	}, nil
}

// extractTasks extracts task definitions from content
func (p *ResponseParserImpl) extractTasks(content string) []TaskInfo {
	var tasks []TaskInfo
	lines := strings.Split(content, "\n")

	inTaskSection := false
	currentSection := ""

	for i, line := range lines {
		trimmedLine := strings.TrimSpace(line)

		// Check for section headers
		if headerMatch := p.headerPattern.FindStringSubmatch(line); len(headerMatch) > 0 {
			headerLevel := len(headerMatch[1])
			headerText := strings.ToLower(headerMatch[2])

			// Look for task-related sections
			if strings.Contains(headerText, "task") ||
			   strings.Contains(headerText, "requirement") ||
			   strings.Contains(headerText, "implementation") ||
			   strings.Contains(headerText, "work item") {
				inTaskSection = true
				currentSection = headerMatch[2]
			} else if headerLevel <= 2 {
				// Higher level header, exit task section
				inTaskSection = false
			}
		}

		// Extract tasks based on various patterns
		task := p.parseTaskLine(trimmedLine, i, currentSection)
		if task != nil {
			tasks = append(tasks, *task)
		} else if inTaskSection && p.looksLikeTask(trimmedLine) {
			// Generic task detection in task sections
			task = &TaskInfo{
				ID:          fmt.Sprintf("TASK-%d", len(tasks)+1),
				Title:       trimmedLine,
				Description: trimmedLine,
				Section:     currentSection,
				LineNumber:  i,
			}
			tasks = append(tasks, *task)
		}
	}

	return tasks
}

// parseTaskLine attempts to parse a task from a line
func (p *ResponseParserImpl) parseTaskLine(line string, lineNum int, section string) *TaskInfo {
	// Pattern 1: CATEGORY-NUMBER: Title (metadata)
	categoryPattern := regexp.MustCompile(`^[\s-]*([A-Z]+)-(\d+):\s*(.+?)(?:\s*\((.*?)\))?\s*$`)
	if match := categoryPattern.FindStringSubmatch(line); len(match) > 0 {
		task := &TaskInfo{
			ID:          match[1] + "-" + match[2],
			Category:    match[1],
			Number:      match[2],
			Title:       match[3],
			Section:     section,
			LineNumber:  lineNum,
		}

		// Parse metadata if present
		if len(match) > 4 && match[4] != "" {
			p.parseTaskMetadata(task, match[4])
		}

		return task
	}

	// Pattern 2: - [ ] Task description
	checkboxPattern := regexp.MustCompile(`^[\s-]*\[[ x]\]\s+(.+)$`)
	if match := checkboxPattern.FindStringSubmatch(line); len(match) > 0 {
		return &TaskInfo{
			ID:          fmt.Sprintf("TASK-%d", lineNum),
			Title:       match[1],
			Description: match[1],
			Section:     section,
			LineNumber:  lineNum,
		}
	}

	// Pattern 3: Task: description
	taskPattern := regexp.MustCompile(`^[\s-]*Task:\s*(.+)$`)
	if match := taskPattern.FindStringSubmatch(line); len(match) > 0 {
		return &TaskInfo{
			ID:          fmt.Sprintf("TASK-%d", lineNum),
			Title:       match[1],
			Description: match[1],
			Section:     section,
			LineNumber:  lineNum,
		}
	}

	return nil
}

// parseTaskMetadata extracts metadata from parentheses content
func (p *ResponseParserImpl) parseTaskMetadata(task *TaskInfo, metadata string) {
	// Split by comma and parse key-value pairs
	parts := strings.Split(metadata, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if kv := strings.SplitN(part, ":", 2); len(kv) == 2 {
			key := strings.TrimSpace(strings.ToLower(kv[0]))
			value := strings.TrimSpace(kv[1])

			switch key {
			case "priority", "prio", "p":
				task.Priority = value
			case "estimate", "est", "time":
				task.Estimate = value
			case "depends", "dependencies", "dep":
				task.Dependencies = strings.Split(value, ";")
			case "assigned", "assignee":
				task.AssignedTo = value
			}
		}
	}
}

// looksLikeTask uses heuristics to identify task-like lines
func (p *ResponseParserImpl) looksLikeTask(line string) bool {
	if len(line) < 3 || len(line) > 200 {
		return false
	}

	// Skip lines that are likely headers or metadata
	if strings.HasPrefix(line, "#") || strings.HasPrefix(line, "```") {
		return false
	}

	// Look for action verbs at the beginning
	actionVerbs := []string{"implement", "create", "add", "update", "fix", "remove",
		"design", "build", "test", "deploy", "configure", "setup", "install"}

	lowerLine := strings.ToLower(line)
	for _, verb := range actionVerbs {
		if strings.HasPrefix(lowerLine, verb) || strings.Contains(lowerLine, " "+verb+" ") {
			return true
		}
	}

	return false
}

// groupTasksByCategory groups tasks by their category
func (p *ResponseParserImpl) groupTasksByCategory(tasks []TaskInfo) map[string][]TaskInfo {
	grouped := make(map[string][]TaskInfo)

	for _, task := range tasks {
		category := task.Category
		if category == "" {
			category = "general"
		}
		grouped[category] = append(grouped[category], task)
	}

	return grouped
}

// formatTasksAsMarkdown formats tasks as markdown content
func (p *ResponseParserImpl) formatTasksAsMarkdown(tasks []TaskInfo) string {
	var sb strings.Builder

	sb.WriteString("# Tasks\n\n")

	for _, task := range tasks {
		sb.WriteString(fmt.Sprintf("## %s\n\n", task.ID))
		sb.WriteString(fmt.Sprintf("**Title:** %s\n\n", task.Title))

		if task.Description != "" && task.Description != task.Title {
			sb.WriteString(fmt.Sprintf("**Description:** %s\n\n", task.Description))
		}

		if task.Priority != "" {
			sb.WriteString(fmt.Sprintf("**Priority:** %s\n\n", task.Priority))
		}

		if task.Estimate != "" {
			sb.WriteString(fmt.Sprintf("**Estimate:** %s\n\n", task.Estimate))
		}

		if len(task.Dependencies) > 0 {
			sb.WriteString(fmt.Sprintf("**Dependencies:** %s\n\n", strings.Join(task.Dependencies, ", ")))
		}

		sb.WriteString("---\n\n")
	}

	return sb.String()
}

// TaskInfo represents extracted task information
type TaskInfo struct {
	ID           string
	Category     string
	Number       string
	Title        string
	Description  string
	Priority     string
	Estimate     string
	Dependencies []string
	AssignedTo   string
	Section      string
	LineNumber   int
}

// ConvertToKanbanTask converts TaskInfo to a kanban.Task
func (t *TaskInfo) ConvertToKanbanTask(commissionID string) *kanban.Task {
	task := kanban.NewTask(t.Title, t.Description)

	// Set priority
	switch strings.ToLower(t.Priority) {
	case "high", "1", "critical":
		task.Priority = kanban.PriorityHigh
	case "low", "3":
		task.Priority = kanban.PriorityLow
	default:
		task.Priority = kanban.PriorityMedium
	}

	// Set initial status
	task.Status = kanban.StatusTodo

	// Add metadata
	task.Metadata["commission_id"] = commissionID
	task.Metadata["category"] = t.Category
	if t.Section != "" {
		task.Metadata["section"] = t.Section
	}

	// Set estimate if present
	if t.Estimate != "" {
		// TODO: Parse estimate to hours
		task.Metadata["estimate_raw"] = t.Estimate
	}

	// Set dependencies
	task.Dependencies = t.Dependencies

	// Add tags based on category
	if t.Category != "" {
		task.Tags = append(task.Tags, strings.ToLower(t.Category))
	}

	return task
}
