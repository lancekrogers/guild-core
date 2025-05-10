# Objective: Implement the Guild Objective Planning System

## Goal

Create an interface for managing project objectives in Guild that enables structured planning, agent alignment, and reproducible agent task execution.

## Context

This tool will support the user in refining a markdown-based objective into a fully structured project plan. It will also coordinate the generation and review of `/ai_docs` and `/specs` directories used by agents.

## Requirements

### Phase 1: Guided Objective Creation

- If no objective exists or none is passed, ask the user:

  ```
  Describe your objective:
  ```

- Process the response using the system prompt for objective construction.
- Generate a markdown file with:
  - `Goal`, `Context`, `Requirements`, `Tags`, `Related`
- Ask the user:
  - "Would you like to accept, retry, or refine this objective?"
  - Handle accordingly and loop back until user accepts.

### Phase 2: Project Directory Generation

- Create a new directory for the objective, named by slug.
- Inside it, create:
  - `/ai_docs/` — contains documents for agents to read
  - `/specs/` — contains deeper technical context
- Generate stub files in each using the parsed objective content.

### Phase 3: Terminal UI

- Display:
  - Objective status
  - Iteration count
  - Command list
- Accept commands such as:
  - `add-context <text>`
  - `regenerate`
  - `suggest`
  - `ready`

### Phase 4: Linking & Document Ingestion

- When users reference `@spec/...` or `@ai_docs/...`, locate and embed those documents in the next LLM prompt.
- Maintain a session-level document map.
- Inject documents in a consistent order with separator headers:

  ```
  ## Attached Context: @spec/foo/bar.md
  ```

## Related Specs

- @spec/features/objectives/objective_ui.md
