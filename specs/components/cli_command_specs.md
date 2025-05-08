# 🧠 Goal

Define the CLI command structure and behavior for the Guild command-line interface.

# 📂 Context

The CLI is the primary way users interact with the Guild framework. It must be intuitive, scriptable, and extensible.

# 🔧 Requirements

## Commands

- `guild start` - scaffolds a new project
- `guild add agent` - adds a new agent to guild.yaml
- `guild add guild` - configures a guild with agents and an objective
- `guild add objective` - creates and registers a new objective
- `guild run` - runs a configured guild against an objective
- `guild monitor` - displays the Kanban board

## Flags and Output

- All commands support `--json` for machine-readable output
- Output logs written to `.guild/logs/`

# 📌 Tags

- cli
- ux
- commands

# 🔗 Related

- `/specs/components/manager_task_loop.md`
- `/specs/features/kanban_board.md`

# 🧪 Self-Validation

- Each command executes without crashing
- `guild start` creates a valid project layout
- Invalid inputs result in descriptive errors
