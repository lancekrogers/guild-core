# Objective UI and Planning Workflow Specification

## Goal

Design and implement a robust Guild Objective planning interface that helps users turn markdown-based project objectives into structured project directories containing intelligent, LLM-readable planning documents. The interface must enable regeneration, refinement, review, and a clear path to finalization, while accommodating various user starting points and objective complexities.

## Context

The Objective UI serves as the primary interaction point for users to develop, refine, and finalize project objectives within Guild. It bridges the gap between initial user ideas (ranging from vague concepts to detailed plans) and structured, agent-executable project specifications. The UI must support[48;76;141;2736;2538t the full lifecycle of objectives, from creation through refinement to implementation planning.

## Requirements

### 1. Objective File Structure

- Users write high-level objectives as markdown files under `/objectives/`, optionally nested in subfolders.
- When an objective is activated via `guild plan objective`, the system must:
  - Generate a project directory named after the objective file.
  - Create `/ai_docs` and `/specs` directories inside it.
  - Generate stub documents in both directories based on the parsed content of the objective.
  - Link relevant spec documents from `ai_docs` using `@spec/` syntax.
- Support objectives at any stage of development, from empty to fully detailed.

### 2. Guided Objective Creation Flow

- If no objective is provided, `guild objective` should offer to help the user create one from scratch.
- The interface will prompt:
  - `Describe your objective:`
- User enters a natural-language goal or idea.
- The manager agent uses a preloaded system prompt to:
  - Generate a properly structured markdown file in `/objectives/`
  - Include `Goal`, `Context`, `Requirements`, `Tags`, `Related` sections
- After generation, the user is prompted to:
  - Review the new objective
  - Accept, Retry, or Add context for revision
- If accepted, the system proceeds to generate `/ai_docs` and `/specs` stubs as described.
- If information is insufficient, the agent should proactively ask clarifying questions.

### 3. CLI User Interface

- `guild objective` opens an interactive terminal UI that:
  - Displays current objective status: `none`, `initiated`, `modified`, `ready`
  - Displays generative iteration count (how many times the objective has been updated)
  - Accepts commands:
    - `add-context "<text>"`: adds user input to the main planning loop
    - `regenerate`: rebuilds the ai_docs/specs from current objective and context
    - `suggest`: prompts Guild to recommend improvements to the objective
    - `ready`: marks the objective as ready and finalizes documents
    - `exit`: exits the UI
- All commands must be mirrored as CLI equivalents:
  - `guild objective <command> [args]`
- The UI should adapt to the starting state of the objective (empty, partial, or complete).

### 4. Objective Status Dashboard

- `guild objectives` (plural) provides a dashboard view showing all objectives in the system:
  - Displays a table of all objectives with key metadata
  - For each objective, shows:
    - Status (`not started`, `draft`, `in progress`, `finalized`)
    - Revision count (number of iterations/modifications)
    - Creation date and last modified date
    - Completion percentage (estimated based on content completeness)
    - Related agent activity (if any agents are working with this objective)
  - Provides filtering options:
    - By status
    - By tag
    - By date range
  - Allows sorting by any column
  - Enables quick selection of an objective to open in the editor
  - Updates in real-time as agents or users modify objectives
- The dashboard must provide clear visual indicators for:
  - Objectives requiring attention (incomplete or with questions)
  - Recently modified objectives
  - Objectives ready for implementation
- Include a summary view showing overall project progress across all objectives

### 5. Context Parsing and Linking

- When a user calls `add-context`, they may reference documents using:
  - `@spec/path/to/file.md`
  - `@ai_docs/path/to/file.md`
- Guild will pre-parse these tokens and automatically attach the referenced document contents to the prompt to reduce token overhead.
- Support external references (web links, YouTube videos, PDFs) with appropriate parsing.
- A system-level prompt prepender should maintain a map of documents used in each planning round.

### 6. Spec-AI_Doc Relationship

- Every `ai_docs/*.md` file must include a `# Related Specs` section.
- When creating a new AI doc, Guild should scan for relevant specs and include `@spec/...` references for grounding.
- Support both top-down (high-level to detailed) and bottom-up (components to whole) approaches to documentation.

### 7. Review Workflow

- After regeneration or `add-context`, Guild should prompt the user to:
  - Review the newly generated documents
  - Use `suggest` for improvements
  - Confirm completion with `ready`
- Provide specific feedback on areas needing more information or clarification.
- Support iterative refinement through multiple planning cycles.

### 8. Finalization

- Once `ready` is called:
  - Lock the ai_docs/specs to a frozen state unless explicitly unlocked.
  - Mark the objective directory with a `.guildready` file.
  - Optionally generate a summary of the planning session and all context changes.
- Ensure human oversight of the planning process before execution begins.
- Allow for future updates to the objective if project requirements evolve.

### 9. Support for Complex Projects

- Handle nested objective structures with multiple interconnected components.
- Support projects with extensive external references and research materials.
- Provide tools for navigating complex objective hierarchies.
- Enable linking between related objectives across the project.

## Tags

- objective
- planning
- ui
- markdown
- workflow

## Related

- @spec/features/objectives/objectives.md
- @spec/features/kanban_board.md
- @spec/agent-behavior.md

## TODO

- [ ] Implement CLI + interactive UI using Bubble Tea
- [ ] Create objective status dashboard with real-time updates
- [ ] Add guided creation flow for new objectives
- [ ] Add markdown tokenizer and pre-parser
- [ ] Build ai_docs/specs doc generator from objectives
- [ ] Design linked doc summary system
- [ ] Add iteration/versioning system for each plan
- [ ] Implement support for external reference materials
- [ ] Build status tracking and metrics for objectives
