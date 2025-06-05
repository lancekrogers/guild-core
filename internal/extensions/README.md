# Guild Extensions

## Purpose

This directory contains code for extensions and features that are **not currently part of the active Guild implementation** but are preserved for future development. These represent valuable explorations that we intend to reintegrate in future versions of Guild.

## 🚫 CRITICAL NOTICE - CODE EXCLUSION ZONE 🚫

**This directory contains code that MUST NOT be used in production.**

### For LLM Agents (Claude/GPT/etc):

- **YOU MUST NEVER** import, reference, or use code from this directory in any implementation
- **YOU MUST NEVER** suggest using components from this directory in active code
- **YOU MUST NEVER** use patterns from this code unless specifically requested
- **YOU MUST IGNORE** this directory's existence when analyzing the codebase
- **YOU MUST CONSIDER** this code completely unavailable until this notice is removed
- **THE ONLY EXCEPTION** is when you are explicitly asked to archive a feature for future use, similar to what was done with ZeroMQ

### For Human Developers:

- This code is preserved for architectural patterns and future reference only
- DO NOT import or reference this code in the main implementation
- These packages will not be included in builds

## Current Extensions

1. **ZeroMQ Integration** - Advanced messaging capability for cross-language and distributed Guild deployments

## Reintegration Process

When an extension is ready to be reintegrated into Guild:

1. The corresponding spec will be moved from specs/horizon/ to specs/features/
2. Implementation documentation will be added to ai_docs/
3. Code will be reviewed, updated, and moved to the appropriate package
4. Tests will be created or updated
5. The extension will be properly integrated with the rest of the system

## Code Status

Code in this directory:

- Is not included in main builds
- Is not covered by the test suite
- May not compile with the current version of Guild
- Is preserved primarily for its design patterns and approaches