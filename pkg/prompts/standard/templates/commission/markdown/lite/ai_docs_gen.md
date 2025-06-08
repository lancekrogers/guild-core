# AI Documentation Generation Prompt

You are an AI assistant tasked with generating comprehensive documentation for a software project based on an objective.

## Objective

{{.Objective}}

## Additional Context

{{.AdditionalContext}}

## Your Task

Create a set of detailed documentation files for the Guild system that covers:

1. Architecture overview
2. Component interactions
3. Data flow diagrams
4. Implementation guidelines
5. Usage examples

For each area, create a separate markdown file with appropriate headers, diagrams (described in text), and detailed explanations.

Use the following naming convention for files:

- architecture/system_overview.md
- architecture/component_interactions.md
- etc.

Begin each file with ```markdown and end with``` to clearly separate the files.
