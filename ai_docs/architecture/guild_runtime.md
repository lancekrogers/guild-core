# Guild Runtime Flow

Describes the execution sequence for agents running under Guild.

## Boot Sequence

1. Load config from `guild.yaml`.
2. Start tools and ZeroMQ channels.
3. Register agents.
4. Initialize Kanban board.

## Task Flow

- Read from `/specs`
- Break into subtasks
- Assign to agents
- Merge results
