// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

// Package context provides context formatting for prompt injection
package context

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"

	"github.com/lancekrogers/guild-core/pkg/gerror"
)

// Context interface is used by formatters (defined in parent package)
type Context interface {
	GetCommissionID() string
	GetCommissionTitle() string
	GetCurrentTask() TaskContext
	GetRelevantSections() []Section
	GetRelatedTasks() []TaskContext
}

// TaskContext represents information about a task
type TaskContext struct {
	ID            string
	Title         string
	Description   string
	SourceSection string
	Priority      string
	Estimate      string
	Dependencies  []string
	Capabilities  []string
}

// Section represents a section from the commission hierarchy
type Section struct {
	Level   int
	Path    string
	Title   string
	Content string
	Tasks   []TaskContext
}

// XMLFormatter formats context as XML for efficient token usage
type XMLFormatter struct {
	template *template.Template
}

// NewXMLFormatter creates a new XML formatter
func NewXMLFormatter() (*XMLFormatter, error) {
	tmpl := `<guild-context>
  <commission id="{{.GetCommissionID}}">
    <title>{{.GetCommissionTitle}}</title>
  </commission>

  <current-task>
    <id>{{.GetCurrentTask.ID}}</id>
    <title>{{.GetCurrentTask.Title}}</title>
    <description>{{.GetCurrentTask.Description}}</description>
    <source-section>{{.GetCurrentTask.SourceSection}}</source-section>
    <priority>{{.GetCurrentTask.Priority}}</priority>
    <estimate>{{.GetCurrentTask.Estimate}}</estimate>
    {{if .GetCurrentTask.Dependencies}}<dependencies>{{range .GetCurrentTask.Dependencies}}
      <dependency>{{.}}</dependency>
    {{end}}</dependencies>{{end}}
    {{if .GetCurrentTask.Capabilities}}<capabilities>{{range .GetCurrentTask.Capabilities}}
      <capability>{{.}}</capability>
    {{end}}</capabilities>{{end}}
  </current-task>

  {{if .GetRelevantSections}}<relevant-sections>{{range .GetRelevantSections}}
    <section level="{{.Level}}" path="{{.Path}}">
      <title>{{.Title}}</title>
      <content>{{.Content}}</content>
      {{if .Tasks}}<tasks>{{range .Tasks}}
        <task id="{{.ID}}">{{.Title}}</task>
      {{end}}</tasks>{{end}}
    </section>
  {{end}}</relevant-sections>{{end}}

  {{if .GetRelatedTasks}}<related-tasks>{{range .GetRelatedTasks}}
    <task id="{{.ID}}">
      <title>{{.Title}}</title>
      {{if .Dependencies}}<dependencies>{{range .Dependencies}}
        <dependency>{{.}}</dependency>
      {{end}}</dependencies>{{end}}
    </task>
  {{end}}</related-tasks>{{end}}
</guild-context>`

	t, err := template.New("xml-context").Parse(tmpl)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInvalidInput, "failed to parse XML template").
			WithComponent("prompts").
			WithOperation("NewXMLFormatter").
			WithDetails("template_type", "xml-context")
	}

	return &XMLFormatter{template: t}, nil
}

// FormatAsXML formats context as XML
func (f *XMLFormatter) FormatAsXML(ctx Context) (string, error) {
	var buf bytes.Buffer
	if err := f.template.Execute(&buf, ctx); err != nil {
		return "", gerror.Wrap(err, gerror.ErrCodeInternal, "failed to execute XML template").
			WithComponent("prompts").
			WithOperation("FormatAsXML").
			WithDetails("template_type", "xml-context")
	}
	return buf.String(), nil
}

// FormatAsMarkdown formats context as markdown
func (f *XMLFormatter) FormatAsMarkdown(ctx Context) (string, error) {
	var sb strings.Builder

	// Commission info
	sb.WriteString("# Commission Context\n\n")
	sb.WriteString(fmt.Sprintf("**Commission ID**: %s\n", ctx.GetCommissionID()))
	sb.WriteString(fmt.Sprintf("**Title**: %s\n\n", ctx.GetCommissionTitle()))

	// Current task
	task := ctx.GetCurrentTask()
	sb.WriteString("## Current Task\n\n")
	sb.WriteString(fmt.Sprintf("- **ID**: %s\n", task.ID))
	sb.WriteString(fmt.Sprintf("- **Title**: %s\n", task.Title))
	if task.Description != "" {
		sb.WriteString(fmt.Sprintf("- **Description**: %s\n", task.Description))
	}
	if task.SourceSection != "" {
		sb.WriteString(fmt.Sprintf("- **Source**: %s\n", task.SourceSection))
	}
	if task.Priority != "" {
		sb.WriteString(fmt.Sprintf("- **Priority**: %s\n", task.Priority))
	}
	if task.Estimate != "" {
		sb.WriteString(fmt.Sprintf("- **Estimate**: %s\n", task.Estimate))
	}
	if len(task.Dependencies) > 0 {
		sb.WriteString(fmt.Sprintf("- **Dependencies**: %s\n", strings.Join(task.Dependencies, ", ")))
	}
	if len(task.Capabilities) > 0 {
		sb.WriteString(fmt.Sprintf("- **Required Capabilities**: %s\n", strings.Join(task.Capabilities, ", ")))
	}
	sb.WriteString("\n")

	// Relevant sections
	sections := ctx.GetRelevantSections()
	if len(sections) > 0 {
		sb.WriteString("## Relevant Documentation\n\n")
		for _, section := range sections {
			// Create appropriate heading level
			heading := strings.Repeat("#", section.Level+2)
			sb.WriteString(fmt.Sprintf("%s %s\n\n", heading, section.Title))
			sb.WriteString(section.Content)
			sb.WriteString("\n\n")

			if len(section.Tasks) > 0 {
				sb.WriteString("**Related Tasks**:\n")
				for _, t := range section.Tasks {
					sb.WriteString(fmt.Sprintf("- %s: %s\n", t.ID, t.Title))
				}
				sb.WriteString("\n")
			}
		}
	}

	// Related tasks
	relatedTasks := ctx.GetRelatedTasks()
	if len(relatedTasks) > 0 {
		sb.WriteString("## Related Tasks\n\n")
		for _, rt := range relatedTasks {
			sb.WriteString(fmt.Sprintf("- **%s**: %s", rt.ID, rt.Title))
			if len(rt.Dependencies) > 0 {
				sb.WriteString(fmt.Sprintf(" (depends on: %s)", strings.Join(rt.Dependencies, ", ")))
			}
			sb.WriteString("\n")
		}
		sb.WriteString("\n")
	}

	return sb.String(), nil
}

// OptimizeForTokens truncates content to fit within token limits
func (f *XMLFormatter) OptimizeForTokens(content string, maxTokens int) (string, error) {
	// Rough approximation: 1 token ≈ 4 characters
	maxChars := maxTokens * 4

	if len(content) <= maxChars {
		return content, nil
	}

	// Find a good truncation point
	truncated := content[:maxChars]

	// Try to truncate at a closing XML tag
	lastCloseTag := strings.LastIndex(truncated, ">")
	if lastCloseTag > maxChars*3/4 { // If we found a tag in the last quarter
		truncated = truncated[:lastCloseTag+1]
	}

	// Add truncation indicator
	truncated += "\n<!-- Content truncated for token limit -->"

	return truncated, nil
}

// DefaultFormatter provides all formatting capabilities
type DefaultFormatter struct {
	*XMLFormatter
}

// NewDefaultFormatter creates a formatter with all capabilities
func NewDefaultFormatter() (*DefaultFormatter, error) {
	xmlFormatter, err := NewXMLFormatter()
	if err != nil {
		return nil, err
	}

	return &DefaultFormatter{
		XMLFormatter: xmlFormatter,
	}, nil
}
