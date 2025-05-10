# Claude Code Commands

These commands provide context and guidance for developing Guild:
| Command | Purpose |
| ----------------------------- | -------------------------------------------------------- |
| @context | Load overall project context and structure |
| @build_project | Detailed implementation plan for all Guild components |
| @review_existing_code | Guide for reviewing code before implementing features |
| @implement_objective_system | Guide for implementing complete Objective system |
| @implement_objective_ui | Guide for implementing Objective UI with Bubble Tea |
| @implement_prompt_system | Guide for implementing internal prompt management system |
| @implement_generator_package | Guide for implementing Generator package |
| @implement_agent | Guide for implementing Agent component |
| @implement_memory | Guide for implementing Memory system |
| @implement_kanban | Guide for implementing Kanban system |
| @implement_tools | Guide for implementing Tools system |
| @implement_orchestrator | Guide for implementing Orchestrator/Guild |

## Usage Tips

1. Always start with `@context` to load project context
2. Use `@review_existing_code` before implementing new features
3. Use specific implementation guides for detailed component guidance
4. Refer to `@build_project` for the overall implementation plan and dependencies

## Command Workflow

For implementing the objective system:

1. `@context` - Load project context
2. `@review_existing_code` - Check what exists already
3. `@implement_prompt_system` - Implement prompt management
4. `@implement_generator_package` - Implement generator functionality
5. `@implement_objective_system` - Implement core objective functionality
6. `@implement_objective_ui` - Implement Bubble Tea UI

For other components, follow similar workflow patterns, always starting with context and reviewing existing code.
