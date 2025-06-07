---
id: "task-extraction-content"
version: "1.0.0"
category: "agent"
subcategory: "extraction"
complexity: 1
tags: ["extraction", "content", "input", "variable"]
variables:
  required: ["RefinedContent"]
  optional: ["ContentFormat", "ContentStructure"]
created: "2025-01-06T12:00:00Z"
updated: "2025-01-06T12:00:00Z"
---

# Content to Analyze

{{if .ContentFormat}}
The refined commission content is provided in {{.ContentFormat}} format.
{{end}}

{{if .ContentStructure}}
## Content Structure
{{.ContentStructure}}
{{end}}

## Refined Commission Content

{{.RefinedContent}}

## Analysis Guidelines

As you analyze this content:

1. **Read Comprehensively**: Understand the full scope before extracting tasks
2. **Identify Actionables**: Look for any work that needs to be done, whether explicitly stated as a task or implied by requirements
3. **Preserve Structure**: If the content has a logical structure, maintain those relationships in your task extraction
4. **Infer from Context**: Some tasks may be implied rather than stated directly
5. **Group Logically**: Related tasks should be categorized together
6. **Extract Metadata**: Pull out any priority indicators, time estimates, dependencies, or other useful information

Remember: The content may express tasks in many ways - as requirements, as user stories, as technical specifications, or simply as descriptions of what needs to exist. Your job is to understand what work needs to be done regardless of how it's expressed.
