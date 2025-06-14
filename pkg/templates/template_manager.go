package templates

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	texttemplate "text/template"
	"time"

	"github.com/guild-ventures/guild-core/pkg/gerror"
	"github.com/guild-ventures/guild-core/pkg/project"
)

// TemplateManager handles template storage, retrieval, and variable substitution
type TemplateManager struct {
	templateDir     string
	contextProvider ContextProvider
	templates       map[string]*Template
	loadedAt        time.Time
}

// Template represents a reusable template with metadata
type Template struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Category    string                 `json:"category"`
	Tags        []string               `json:"tags"`
	Variables   []TemplateVariable     `json:"variables"`
	Content     string                 `json:"content"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
	UsageCount  int                    `json:"usage_count"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// TemplateVariable defines a variable that can be substituted in templates
type TemplateVariable struct {
	Name         string      `json:"name"`
	Description  string      `json:"description"`
	Type         string      `json:"type"` // string, number, boolean, array, object
	Required     bool        `json:"required"`
	DefaultValue interface{} `json:"default_value,omitempty"`
	Options      []string    `json:"options,omitempty"` // For enum-type variables
	Pattern      string      `json:"pattern,omitempty"` // Regex pattern for validation
}

// ContextProvider interface for getting contextual information
type ContextProvider interface {
	GetProjectContext() (map[string]interface{}, error)
	GetUserContext() (map[string]interface{}, error)
	GetSystemContext() (map[string]interface{}, error)
}

// DefaultContextProvider provides basic context information
type DefaultContextProvider struct{}

// TemplateContext contains all available context for template rendering
type TemplateContext struct {
	Project   map[string]interface{} `json:"project"`
	User      map[string]interface{} `json:"user"`
	System    map[string]interface{} `json:"system"`
	Variables map[string]interface{} `json:"variables"`
	Custom    map[string]interface{} `json:"custom"`
}

// TemplateSearchResult represents a template search result with relevance
type TemplateSearchResult struct {
	Template  *Template
	Relevance float64
	Matches   []string
}

// NewTemplateManager creates a new template manager
func NewTemplateManager(projectDir string) (*TemplateManager, error) {
	templateDir := filepath.Join(projectDir, ".guild", "templates")
	
	// Create templates directory if it doesn't exist
	if err := os.MkdirAll(templateDir, 0755); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeIO, "failed to create templates directory")
	}
	
	tm := &TemplateManager{
		templateDir:     templateDir,
		contextProvider: &DefaultContextProvider{},
		templates:       make(map[string]*Template),
	}
	
	// Load existing templates
	if err := tm.LoadTemplates(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to load templates")
	}
	
	// Create default templates if none exist
	if len(tm.templates) == 0 {
		if err := tm.createDefaultTemplates(); err != nil {
			return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create default templates")
		}
	}
	
	return tm, nil
}

// LoadTemplates loads all templates from the templates directory
func (tm *TemplateManager) LoadTemplates() error {
	tm.templates = make(map[string]*Template)
	
	err := filepath.WalkDir(tm.templateDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		
		if d.IsDir() || !strings.HasSuffix(d.Name(), ".json") {
			return nil
		}
		
		// Load template from file
		template, err := tm.loadTemplateFromFile(path)
		if err != nil {
			// Log error but continue loading other templates
			return nil
		}
		
		tm.templates[template.ID] = template
		return nil
	})
	
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeIO, "failed to walk templates directory")
	}
	
	tm.loadedAt = time.Now()
	return nil
}

// loadTemplateFromFile loads a single template from a JSON file
func (tm *TemplateManager) loadTemplateFromFile(path string) (*Template, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeIO, "failed to read template file")
	}
	
	var template Template
	if err := json.Unmarshal(data, &template); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInvalidInput, "failed to parse template JSON")
	}
	
	// Load content from separate file if not embedded
	if template.Content == "" {
		contentPath := strings.TrimSuffix(path, ".json") + ".md"
		if contentData, err := os.ReadFile(contentPath); err == nil {
			template.Content = string(contentData)
		}
	}
	
	return &template, nil
}

// SaveTemplate saves a template to the templates directory
func (tm *TemplateManager) SaveTemplate(template *Template) error {
	if template.ID == "" {
		template.ID = tm.generateTemplateID(template.Name)
	}
	
	if template.CreatedAt.IsZero() {
		template.CreatedAt = time.Now()
	}
	template.UpdatedAt = time.Now()
	
	// Save metadata as JSON
	metadataPath := filepath.Join(tm.templateDir, template.ID+".json")
	templateCopy := *template
	
	// Don't embed content in JSON if it's large
	if len(template.Content) > 1000 {
		templateCopy.Content = ""
		
		// Save content separately
		contentPath := filepath.Join(tm.templateDir, template.ID+".md")
		if err := os.WriteFile(contentPath, []byte(template.Content), 0644); err != nil {
			return gerror.Wrap(err, gerror.ErrCodeIO, "failed to write template content")
		}
	}
	
	data, err := json.MarshalIndent(&templateCopy, "", "  ")
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to marshal template")
	}
	
	if err := os.WriteFile(metadataPath, data, 0644); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeIO, "failed to write template file")
	}
	
	// Update in-memory cache
	tm.templates[template.ID] = template
	
	return nil
}

// GetTemplate retrieves a template by ID
func (tm *TemplateManager) GetTemplate(id string) (*Template, error) {
	template, exists := tm.templates[id]
	if !exists {
		return nil, gerror.New(gerror.ErrCodeNotFound, fmt.Sprintf("template not found: %s", id), nil)
	}
	
	return template, nil
}

// ListTemplates returns all templates, optionally filtered by category
func (tm *TemplateManager) ListTemplates(category string) ([]*Template, error) {
	var templates []*Template
	
	for _, template := range tm.templates {
		if category == "" || template.Category == category {
			templates = append(templates, template)
		}
	}
	
	// Sort by usage count (descending) then by name
	sort.Slice(templates, func(i, j int) bool {
		if templates[i].UsageCount != templates[j].UsageCount {
			return templates[i].UsageCount > templates[j].UsageCount
		}
		return templates[i].Name < templates[j].Name
	})
	
	return templates, nil
}

// SearchTemplates searches for templates matching the query
func (tm *TemplateManager) SearchTemplates(query string, limit int) ([]*TemplateSearchResult, error) {
	if query == "" {
		return nil, gerror.New(gerror.ErrCodeInvalidInput, "search query cannot be empty", nil)
	}
	
	query = strings.ToLower(query)
	var results []*TemplateSearchResult
	
	for _, template := range tm.templates {
		result := &TemplateSearchResult{Template: template}
		var matches []string
		
		// Search in name
		if strings.Contains(strings.ToLower(template.Name), query) {
			result.Relevance += 10.0
			matches = append(matches, "name")
		}
		
		// Search in description
		if strings.Contains(strings.ToLower(template.Description), query) {
			result.Relevance += 5.0
			matches = append(matches, "description")
		}
		
		// Search in tags
		for _, tag := range template.Tags {
			if strings.Contains(strings.ToLower(tag), query) {
				result.Relevance += 3.0
				matches = append(matches, "tag:"+tag)
			}
		}
		
		// Search in content
		if strings.Contains(strings.ToLower(template.Content), query) {
			result.Relevance += 1.0
			matches = append(matches, "content")
		}
		
		// Boost based on usage
		result.Relevance += float64(template.UsageCount) * 0.1
		
		if result.Relevance > 0 {
			result.Matches = matches
			results = append(results, result)
		}
	}
	
	// Sort by relevance
	sort.Slice(results, func(i, j int) bool {
		return results[i].Relevance > results[j].Relevance
	})
	
	// Apply limit
	if limit > 0 && len(results) > limit {
		results = results[:limit]
	}
	
	return results, nil
}

// RenderTemplate renders a template with the provided variables
func (tm *TemplateManager) RenderTemplate(templateID string, variables map[string]interface{}) (string, error) {
	template, err := tm.GetTemplate(templateID)
	if err != nil {
		return "", err
	}
	
	// Build context
	context, err := tm.buildContext(variables)
	if err != nil {
		return "", gerror.Wrap(err, gerror.ErrCodeInternal, "failed to build template context")
	}
	
	// Validate required variables
	if err := tm.validateVariables(template, variables); err != nil {
		return "", err
	}
	
	// Parse and execute template
	tmpl, err := texttemplate.New(templateID).Parse(template.Content)
	if err != nil {
		return "", gerror.Wrap(err, gerror.ErrCodeInvalidInput, "failed to parse template")
	}
	
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, context); err != nil {
		return "", gerror.Wrap(err, gerror.ErrCodeInternal, "failed to execute template")
	}
	
	// Update usage count
	template.UsageCount++
	template.UpdatedAt = time.Now()
	tm.SaveTemplate(template) // Best effort, don't fail on error
	
	return buf.String(), nil
}

// GetContextualSuggestions returns template suggestions based on current context
func (tm *TemplateManager) GetContextualSuggestions(context map[string]interface{}) ([]*Template, error) {
	var suggestions []*Template
	
	// Get project context
	projectContext, err := tm.contextProvider.GetProjectContext()
	if err == nil {
		// Suggest templates based on project type, language, etc.
		if projectType, ok := projectContext["type"].(string); ok {
			for _, template := range tm.templates {
				if template.Category == projectType || contains(template.Tags, projectType) {
					suggestions = append(suggestions, template)
				}
			}
		}
	}
	
	// Add frequently used templates
	var frequentTemplates []*Template
	for _, template := range tm.templates {
		if template.UsageCount > 5 { // Arbitrary threshold
			frequentTemplates = append(frequentTemplates, template)
		}
	}
	
	// Sort frequent templates by usage
	sort.Slice(frequentTemplates, func(i, j int) bool {
		return frequentTemplates[i].UsageCount > frequentTemplates[j].UsageCount
	})
	
	// Add top frequent templates
	for i, template := range frequentTemplates {
		if i >= 3 { // Limit to top 3
			break
		}
		if !containsTemplate(suggestions, template) {
			suggestions = append(suggestions, template)
		}
	}
	
	return suggestions, nil
}

// DeleteTemplate removes a template
func (tm *TemplateManager) DeleteTemplate(id string) error {
	if _, exists := tm.templates[id]; !exists {
		return gerror.New(gerror.ErrCodeNotFound, fmt.Sprintf("template not found: %s", id), nil)
	}
	
	// Remove files
	metadataPath := filepath.Join(tm.templateDir, id+".json")
	contentPath := filepath.Join(tm.templateDir, id+".md")
	
	os.Remove(metadataPath) // Best effort
	os.Remove(contentPath)  // Best effort
	
	// Remove from cache
	delete(tm.templates, id)
	
	return nil
}

// Helper functions

func (tm *TemplateManager) buildContext(variables map[string]interface{}) (*TemplateContext, error) {
	context := &TemplateContext{
		Variables: variables,
		Custom:    make(map[string]interface{}),
	}
	
	// Get project context
	if projectContext, err := tm.contextProvider.GetProjectContext(); err == nil {
		context.Project = projectContext
	} else {
		context.Project = make(map[string]interface{})
	}
	
	// Get user context
	if userContext, err := tm.contextProvider.GetUserContext(); err == nil {
		context.User = userContext
	} else {
		context.User = make(map[string]interface{})
	}
	
	// Get system context
	if systemContext, err := tm.contextProvider.GetSystemContext(); err == nil {
		context.System = systemContext
	} else {
		context.System = make(map[string]interface{})
	}
	
	return context, nil
}

func (tm *TemplateManager) validateVariables(template *Template, variables map[string]interface{}) error {
	for _, variable := range template.Variables {
		if variable.Required {
			if _, exists := variables[variable.Name]; !exists {
				return gerror.New(gerror.ErrCodeInvalidInput, fmt.Sprintf("required variable missing: %s", variable.Name), nil)
			}
		}
		
		// Validate pattern if specified
		if variable.Pattern != "" && variables[variable.Name] != nil {
			if strValue, ok := variables[variable.Name].(string); ok {
				if matched, err := regexp.MatchString(variable.Pattern, strValue); err != nil || !matched {
					return gerror.New(gerror.ErrCodeInvalidInput, fmt.Sprintf("variable %s does not match pattern %s", variable.Name, variable.Pattern), nil)
				}
			}
		}
	}
	
	return nil
}

func (tm *TemplateManager) generateTemplateID(name string) string {
	// Create ID from name
	id := strings.ToLower(strings.ReplaceAll(name, " ", "-"))
	id = regexp.MustCompile(`[^a-z0-9-]`).ReplaceAllString(id, "")
	
	// Ensure uniqueness
	originalID := id
	counter := 1
	for {
		if _, exists := tm.templates[id]; !exists {
			break
		}
		id = fmt.Sprintf("%s-%d", originalID, counter)
		counter++
	}
	
	return id
}

func (tm *TemplateManager) createDefaultTemplates() error {
	defaultTemplates := []*Template{
		{
			Name:        "API Endpoint",
			Description: "Template for creating a REST API endpoint",
			Category:    "api",
			Tags:        []string{"api", "rest", "endpoint", "http"},
			Variables: []TemplateVariable{
				{Name: "endpoint_name", Description: "Name of the endpoint", Type: "string", Required: true},
				{Name: "method", Description: "HTTP method", Type: "string", Required: true, Options: []string{"GET", "POST", "PUT", "DELETE"}},
				{Name: "description", Description: "Endpoint description", Type: "string", Required: false},
			},
			Content: `# {{.Variables.endpoint_name}} Endpoint

## Description
{{if .Variables.description}}{{.Variables.description}}{{else}}{{.Variables.method}} endpoint for {{.Variables.endpoint_name}}{{end}}

## Method
{{.Variables.method}}

## URL
/api/{{.Variables.endpoint_name | lower}}

## Request
` + "```json" + `
{
  // Request body here
}
` + "```" + `

## Response
` + "```json" + `
{
  // Response body here
}
` + "```" + `

## Implementation Notes
- [ ] Add input validation
- [ ] Add error handling
- [ ] Add authentication if needed
- [ ] Add rate limiting
- [ ] Add logging
- [ ] Add tests
`,
		},
		{
			Name:        "Bug Report",
			Description: "Template for reporting bugs",
			Category:    "documentation",
			Tags:        []string{"bug", "report", "issue", "documentation"},
			Variables: []TemplateVariable{
				{Name: "title", Description: "Bug title", Type: "string", Required: true},
				{Name: "severity", Description: "Bug severity", Type: "string", Required: true, Options: []string{"low", "medium", "high", "critical"}},
				{Name: "component", Description: "Affected component", Type: "string", Required: false},
			},
			Content: `# Bug Report: {{.Variables.title}}

## Severity
{{.Variables.severity | title}}

{{if .Variables.component}}## Component
{{.Variables.component}}
{{end}}

## Description
Brief description of the bug.

## Steps to Reproduce
1. Step one
2. Step two
3. Step three

## Expected Behavior
What you expected to happen.

## Actual Behavior
What actually happened.

## Environment
- OS: 
- Version: 
- Browser (if applicable): 

## Additional Context
Any additional information, screenshots, or logs.

## Acceptance Criteria
- [ ] Bug is reproduced
- [ ] Root cause identified
- [ ] Fix implemented
- [ ] Tests added
- [ ] Documentation updated
`,
		},
		{
			Name:        "Meeting Notes",
			Description: "Template for meeting notes",
			Category:    "documentation",
			Tags:        []string{"meeting", "notes", "documentation"},
			Variables: []TemplateVariable{
				{Name: "meeting_title", Description: "Meeting title", Type: "string", Required: true},
				{Name: "date", Description: "Meeting date", Type: "string", Required: true},
				{Name: "attendees", Description: "Meeting attendees", Type: "string", Required: false},
			},
			Content: `# {{.Variables.meeting_title}}

**Date:** {{.Variables.date}}
{{if .Variables.attendees}}**Attendees:** {{.Variables.attendees}}
{{end}}

## Agenda
1. Item 1
2. Item 2
3. Item 3

## Discussion
### Topic 1
- Discussion points
- Decisions made

### Topic 2
- Discussion points
- Decisions made

## Action Items
- [ ] Action item 1 (Assignee: Name, Due: Date)
- [ ] Action item 2 (Assignee: Name, Due: Date)

## Next Steps
- Next meeting date:
- Follow-up items:

## Notes
Additional notes and context.
`,
		},
	}
	
	for _, template := range defaultTemplates {
		if err := tm.SaveTemplate(template); err != nil {
			return err
		}
	}
	
	return nil
}

// Context provider implementations

func (dcp *DefaultContextProvider) GetProjectContext() (map[string]interface{}, error) {
	context := make(map[string]interface{})
	
	// Get project information
	if projectCtx, err := project.GetContext(); err == nil {
		context["name"] = filepath.Base(projectCtx.GetRootPath())
		context["path"] = projectCtx.GetRootPath()
		context["type"] = "unknown" // Could be enhanced to detect project type
	}
	
	context["timestamp"] = time.Now().Format(time.RFC3339)
	context["date"] = time.Now().Format("2006-01-02")
	context["time"] = time.Now().Format("15:04:05")
	
	return context, nil
}

func (dcp *DefaultContextProvider) GetUserContext() (map[string]interface{}, error) {
	context := make(map[string]interface{})
	
	// Get user information from environment
	if user := os.Getenv("USER"); user != "" {
		context["username"] = user
	}
	if home := os.Getenv("HOME"); home != "" {
		context["home"] = home
	}
	
	return context, nil
}

func (dcp *DefaultContextProvider) GetSystemContext() (map[string]interface{}, error) {
	context := make(map[string]interface{})
	
	context["os"] = os.Getenv("GOOS")
	context["arch"] = os.Getenv("GOARCH")
	
	return context, nil
}

// Utility functions

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func containsTemplate(slice []*Template, template *Template) bool {
	for _, t := range slice {
		if t.ID == template.ID {
			return true
		}
	}
	return false
}