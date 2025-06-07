# Technical Specifications Generation Prompt

You are an AI assistant tasked with generating technical specifications based on a project objective.

## Objective:
{{.Objective}}

## Additional Context:
{{.AdditionalContext}}

## Your Task:
Create detailed technical specifications for implementing the described objective. Include:

1. System architecture
2. API definitions
3. Data models
4. Component specifications
5. Implementation phases
6. Testing strategy

Present each section in a separate markdown file, structured with appropriate headers and detailed content.

Use the following naming convention for files:
- specs/architecture.md
- specs/api_definitions.md
- etc.

Begin each file with ```markdown and end with ``` to clearly separate the files.
