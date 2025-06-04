# Execution Layer: Current Step Guidance

## Current Execution Phase: {{.Phase}}

<current_step>
### Step {{.StepNumber}} of {{.TotalSteps}}: {{.StepName}}

### Objective for This Step
{{.StepObjective}}

### Expected Actions
{{range .ExpectedActions}}
1. {{.}}
{{end}}

### Success Indicators
{{range .SuccessIndicators}}
- {{.}}
{{end}}

### Potential Issues to Watch For
{{range .PotentialIssues}}
- {{.}}
{{end}}
</current_step>

## Progress Status
- **Overall Progress**: {{.OverallProgress}}%
- **Current Phase Progress**: {{.PhaseProgress}}%
- **Time Elapsed**: {{.TimeElapsed}}
- **Estimated Time Remaining**: {{.EstimatedTimeRemaining}}

## Previous Step Results
{{if .PreviousStepResult}}
<previous_result>
{{.PreviousStepResult}}
</previous_result>
{{end}}

## Next Steps Preview
{{range .NextSteps}}
- {{.}}
{{end}}