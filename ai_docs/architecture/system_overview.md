# Guild System Overview

Guild is a multi-agent orchestration framework written in Go for autonomous development workflows.

## Core Components

- `Agent`: Executes tasks with access to tools.
- `Guild`: Coordinates agents toward shared objectives.
- `Tool`: External functions accessible to agents.
- `Manager`: Oversees task flow and orchestration.

## Runtime Layers

- CLI (via `cmd/guild`)
- Internal task engine
- ZeroMQ + BoltDB backend
