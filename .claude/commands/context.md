## Project Context

Please review the following key resources:

1. First, read the Specs Index at ai_docs/specs_index.md to understand project requirements
2. Then, check the AI Docs Index at ai_docs/README.md for implementation guidance
3. Review the Guild lore at specs/naming_conventions_and_lore/lore.md

## Commands to Run

Run these commands to understand the project structure:

```bash
git ls-files | grep -E '\.go$' | sort
tree -I "node_modules|.git|.idea|bin" --dirsfirst
go list -m all
```

## Testing Requirements

⚠️ IMPORTANT: All code must include unit tests. When you write any implementation, you must also include:

1. Unit tests for public functions and methods
2. Tests for both success and error conditions
3. Mocks for external dependencies
4. Context cancellation tests where appropriate

I will not accept any implementation without corresponding tests.
