# Task Layer: Specific Task Requirements

## Current Task
<task>
### Title: {{.TaskTitle}}

### Description
{{.TaskDescription}}

### Requirements
{{range .Requirements}}
- {{.}}
{{end}}

### Constraints
{{range .Constraints}}
- {{.}}
{{end}}

### Priority: {{.Priority}}
### Due Date: {{.DueDate}}
### Estimated Hours: {{.EstimatedHours}}
</task>

## Dependencies
<dependencies>
{{range .TaskDependencies}}
### Depends On: {{.TaskID}}
- **Title**: {{.Title}}
- **Status**: {{.Status}}
- **Output Location**: {{.OutputPath}}
{{end}}
</dependencies>

## Expected Deliverables
<deliverables>
{{range .Deliverables}}
### {{.Name}}
- **Type**: {{.Type}}
- **Format**: {{.Format}}
- **Location**: {{.ExpectedPath}}
- **Description**: {{.Description}}
{{end}}
</deliverables>