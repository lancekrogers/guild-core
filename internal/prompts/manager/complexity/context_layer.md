# Task Complexity Analysis - Context Layer

## Guild Context
Guild: {{.GuildName}}
Project: {{.ProjectType}}
Available Agents: {{.AgentCount}}
Budget: {{.TokenBudget}} tokens

## Current Task Context
Task: {{.TaskDescription}}
Domain: {{.TaskDomain}}
Priority: {{.TaskPriority}}
Deadline: {{.TaskDeadline}}

## Available Agent Specializations
{{range .AvailableAgents}}
### {{.Name}} ({{.Role}})
- **Cost**: {{.CostMagnitude}}/10 ({{.Provider}}/{{.Model}})
- **Context Window**: {{.ContextWindow}} tokens
- **Specializations**: {{join .Specializations ", "}}
- **Tools**: {{join .Tools ", "}}
- **Performance**: {{.SuccessRate}}% success rate
{{end}}

## Project Resources
- **Corpus Size**: {{.CorpusDocuments}} documents
- **Codebase**: {{.CodebaseSize}} files
- **Recent Activity**: {{.RecentTasks}} tasks completed
- **Team Velocity**: {{.TeamVelocity}} tasks/day

## Constraints
- **Token Budget**: {{.TokenBudget}} tokens available
- **Time Constraint**: {{.TimeConstraint}}
- **Quality Requirements**: {{.QualityLevel}}
- **Risk Tolerance**: {{.RiskTolerance}}