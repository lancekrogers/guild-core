---
id: "task-extraction-context"
version: "1.0.0"
category: "agent"
subcategory: "extraction"
complexity: 1
tags: ["extraction", "context", "commission", "variable"]
variables:
  required: ["CommissionID", "CommissionTitle", "CommissionGoals"]
  optional: ["Constraints", "Technologies", "Timeline"]
created: "2025-01-06T12:00:00Z"
updated: "2025-01-06T12:00:00Z"
---

# Commission Context

You are extracting tasks for the following commission:

## Commission Details
- **ID**: {{.CommissionID}}
- **Title**: {{.CommissionTitle}}

## Commission Goals
{{.CommissionGoals}}

{{if .Constraints}}
## Constraints and Requirements
{{.Constraints}}
{{end}}

{{if .Technologies}}
## Technology Stack
{{.Technologies}}
{{end}}

{{if .Timeline}}
## Timeline Considerations
{{.Timeline}}
{{end}}

## Extraction Context

When extracting tasks, keep in mind:
1. All tasks should contribute to achieving the commission goals
2. Respect any stated constraints or requirements
3. Consider the technology choices when determining task complexity
4. Use timeline information to influence priority assignments
5. Maintain traceability between tasks and commission objectives

The extracted tasks should form a complete plan for achieving the commission's goals while respecting all constraints and preferences.