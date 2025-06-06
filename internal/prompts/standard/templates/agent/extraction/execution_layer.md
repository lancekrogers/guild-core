---
id: "task-extraction-execution"
version: "1.0.0"
category: "agent"
subcategory: "extraction"
complexity: 1
tags: ["extraction", "execution", "output", "json"]
created: "2025-01-06T12:00:00Z"
updated: "2025-01-06T12:00:00Z"
---

# Task Extraction Execution

## Your Mission

Analyze the provided refined commission content and extract all actionable tasks. Output them in the structured JSON format specified below.

## Output Format

Produce a JSON object with the following structure:

```json
{
  "extractionMetadata": {
    "commissionId": "string",
    "extractedAt": "ISO 8601 timestamp",
    "totalTasks": number,
    "contentAnalysis": {
      "structure": "description of content structure",
      "completeness": "assessment of coverage",
      "clarity": "assessment of task clarity"
    }
  },
  "tasks": [
    {
      "id": "CATEGORY-NNN",
      "title": "Clear, action-oriented title",
      "description": "Detailed description of what needs to be done",
      "category": "ARCH|AUTH|API|UI|DATA|TEST|DOC|INFRA|SEC|PERF|OTHER",
      "priority": "high|medium|low",
      "estimatedHours": number or null,
      "dependencies": ["task-id-1", "task-id-2"] or [],
      "requiredCapabilities": ["capability1", "capability2"],
      "metadata": {
        "sourceSection": "where in the content this came from",
        "rationale": "why this task is needed",
        "acceptanceCriteria": ["criterion1", "criterion2"] or null,
        "technicalNotes": "any technical details" or null
      }
    }
  ],
  "taskRelationships": {
    "phases": [
      {
        "name": "Phase name",
        "description": "Phase description",
        "taskIds": ["task-id-1", "task-id-2"]
      }
    ],
    "criticalPath": ["task-id-1", "task-id-2", "..."]
  }
}
```

## Extraction Guidelines

1. **ID Generation**: Create logical IDs using CATEGORY-NNN format. Ensure uniqueness and logical grouping.

2. **Title Clarity**: Titles should be clear, actionable, and specific. Start with a verb when possible.

3. **Description Completeness**: Include enough detail for an artisan to understand the task without referring back to the source document.

4. **Priority Assignment**:
   - **High**: Critical path, blocks other work, or core functionality
   - **Medium**: Important but not blocking, enhances functionality
   - **Low**: Nice to have, optimizations, or polish

5. **Estimation**:
   - Provide hours when you can reasonably estimate
   - Consider complexity, testing, and review time
   - Use null if insufficient information

6. **Dependencies**:
   - Identify technical dependencies (must complete X before Y)
   - Include logical dependencies (makes sense to do X first)
   - Use task IDs for clear references

7. **Capabilities**:
   - Match to Guild's standard capabilities
   - Include all skills needed for the task
   - Help with artisan assignment

8. **Metadata**:
   - Track where the task came from in the source
   - Explain why it's needed (helps with prioritization)
   - Include acceptance criteria when available
   - Add technical notes for implementation guidance

## Quality Checks

Before outputting, ensure:
- All actionable items from the content are captured
- Task IDs are unique and follow the naming convention
- Dependencies form a valid graph (no cycles)
- Priorities make sense relative to each other
- The critical path is logical and complete
- Descriptions are clear and self-contained

## Begin Extraction

Analyze the refined commission content and produce the structured task list following the guidelines above. Focus on completeness, clarity, and actionability.