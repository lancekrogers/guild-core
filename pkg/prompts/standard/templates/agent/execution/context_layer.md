# Context Layer: Project and Commission Information

## Current Commission

<commission>
{{.CommissionTitle}}

{{.CommissionDescription}}

### Success Criteria

{{range .SuccessCriteria}}

- {{.}}
{{end}}
</commission>

## Project Context

<project_context>

### Project Overview

{{.ProjectDescription}}

### Relevant Documentation

{{range .RelevantDocs}}

- {{.Path}}: {{.Summary}}
{{end}}

### Technical Context

- **Language/Framework**: {{.TechStack}}
- **Architecture**: {{.Architecture}}
- **Key Dependencies**: {{.ProjectDependencies}}
</project_context>

## Related Work

<related_tasks>
{{range .RelatedTasks}}

### {{.Title}} ({{.Status}})

- **Assigned To**: {{.AssignedTo}}
- **Key Output**: {{.Output}}
{{end}}
</related_tasks>
