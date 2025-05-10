# Guild Extensions

## Purpose

This directory contains code for extensions and features that are **not currently part of the active Guild implementation** but are preserved for future development. These represent valuable explorations that we intend to reintegrate in future versions of Guild.

## ⚠️ IMPORTANT NOTICE FOR AI AGENTS AND DEVELOPERS ⚠️

- **DO NOT** import or use code from this directory in the main Guild implementation
- **DO NOT** update or maintain this code unless specifically tasked with preparing it for reintegration
- **DO NOT** use this code as a reference for implementing core functionality

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