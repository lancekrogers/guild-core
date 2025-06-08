# Task Complexity Analysis - Execution Layer

## Analysis Request

Analyze the following task and provide a structured decision on execution strategy:

**Task**: {{.TaskDescription}}

## Required Analysis

### 1. Complexity Assessment

Evaluate these factors:

- **Technical Scope**: How many different technologies/languages are involved?
- **Domain Breadth**: Does this span multiple areas (frontend, backend, DevOps, etc.)?
- **Integration Complexity**: How many external systems or APIs are involved?
- **Knowledge Depth**: What level of specialized expertise is required?
- **Coordination Needs**: How much inter-component communication is needed?

### 2. Effort Estimation

Consider:

- **Expected File Count**: How many files will be created/modified?
- **Development Time**: Realistic hours of focused work
- **Review Cycles**: How many iterations likely needed?
- **Testing Requirements**: Unit, integration, e2e testing scope

### 3. Risk Analysis

Identify:

- **Technical Risks**: Unknown technologies, complex integrations
- **Coordination Risks**: Dependencies between team members
- **Quality Risks**: Areas where mistakes could be costly
- **Timeline Risks**: Potential blockers or delays

## Output Requirements

Provide your analysis in this JSON format:

```json
{
  "complexity_score": <1-10>,
  "recommended_approach": "<single-agent|multi-agent|hybrid>",
  "reasoning": "<clear explanation of decision factors>",
  "agent_requirements": [
    {
      "role": "<required specialization>",
      "priority": "<high|medium|low>",
      "estimated_tokens": <token estimate>,
      "rationale": "<why this specialization is needed>"
    }
  ],
  "execution_strategy": {
    "parallel_tasks": ["<task 1>", "<task 2>"],
    "sequential_tasks": ["<task 1>", "<task 2>"],
    "dependencies": [
      {"from": "<task>", "to": "<dependent task>"}
    ]
  },
  "cost_estimate": {
    "single_agent_tokens": <estimate>,
    "multi_agent_tokens": <estimate>,
    "recommended_savings": "<percentage>"
  },
  "quality_assurance": {
    "review_points": ["<checkpoint 1>", "<checkpoint 2>"],
    "testing_strategy": "<approach>",
    "risk_mitigation": ["<mitigation 1>", "<mitigation 2>"]
  }
}
```

Base your analysis on the available agents and their capabilities provided in the context.
