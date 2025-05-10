## Project Context

Please review the following key resources:

1. First, read the Specs Index at ai_docs/specs_index.md to understand project structure and requirements
2. Then, check the AI Docs Index at ai_docs/README.md for implementation guidance
3. **IMPORTANT**: Study the Guild lore and naming conventions at ai_docs/project_context/lore.md
4. Examine the objectives system documentation at specs/features/objectives/objectives.md and specs/features/objectives/objective_ui.md

## Existing Implementation Check

Before implementing any new components, **ALWAYS** check for existing code:

```bash
# Look for existing implementations related to the feature
find . -type f -name "*.go" -exec grep -l "relevant_term" {} \;
# For objective system, check:
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

## Objective System Implementation

For the objective system specifically:

- Check existing code in `pkg/objective/` before writing new code
- Prompt files should go in `internal/prompts/objective/markdown/`
- Generator implementations should go in `pkg/generator/objective/`
- UI implementations should use Bubble Tea and go in `pkg/ui/objective/`

## Testing Requirements

⚠️ IMPORTANT: All code must include comprehensive unit tests. Reference these commands:

- `@tdd` - Follow Test-Driven Development practices
- `@go_testing` - Apply Go-specific testing best practices
- `@ensure_tests` - Verify test coverage is complete

When implementing the objective system, include specific test fixtures for:

- Prompt templates in `internal/prompts/objective/testdata/`
- Sample objectives in `pkg/objective/testdata/`
- Mock LLM clients in `pkg/generator/objective/mocks/`

I will not accept any implementation without corresponding tests.
