# .claude/init.md

## Initialization Instructions

## Project Context

Review the project context in ai_docs/project_context

Next Please review the following key resources:

1. First, read the Specs Index at ai_docs/specs_index.md to understand project structure and requirements
2. Then, check the AI Docs Index at ai_docs/README.md for implementation guidance
3. **IMPORTANT**: Study the Guild lore and naming conventions at ai_docs/project_context/
4. Track progress by following the system described in the documents found in ai_docs/progress_tracking/

## Existing Implementation Check

Before implementing any new components, **ALWAYS** check for existing code:

```bash
# Look for existing implementations related to the feature
find . -type f -name "*.go" -exec grep -l "relevant_term" {} \;
# For example, for the objective system, check:
find . -type f -name "*.go" -exec grep -l "objective" {} \;
```

## Commands to Run

Run these commands to understand the project structure:

```bash
git ls-files | grep -E '\.go$' | sort
tree -I ".git|.idea|bin|ai_docs|specs|.claude" --dirsfirst
go list -m all
```

## Code Structure Guidelines

- Place interfaces in `interface.go` files within their respective packages
- Keep implementation files separate from interface definitions
- Follow standard Go project layout (cmd/, pkg/, internal/)
- Place prompt files in `internal/prompts/[domain]/markdown/` as .md files

## Coding Guidelines

- Review the documents found in ai_docs/coding_guidelines

## Testing Requirements

⚠️ IMPORTANT: All code must include comprehensive unit tests. Reference these commands:

- Follow the testing guidelines found in the documents in ai_docs/testing/
- After writing a test verify that it is accurate and the code being tested is correctly

I will not accept any implementation without corresponding tests.

This file is loaded automatically when Claude Code resets. I've loaded the project context for you. We're working on the Guild framework, an agent orchestration system.

Let me know what component you'd like to work on today, and I'll help you implement it following our established patterns.
