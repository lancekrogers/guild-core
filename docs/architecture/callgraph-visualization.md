# Callgraph Visualization with go-callvis

This guide shows how to map Guild Core integration paths with go-callvis. The goal is a clear picture of how the Great Hall (CLI), daemon, and integration services call into one another.

## Prerequisites

- `go-callvis` in your PATH (already installed on this machine).
- Graphviz installed for image output (`dot` binary).
- Run commands from `projects/guild-core`.

## Quick Start (Interactive)

Launch an interactive callgraph server and open it in your browser:

```bash
cd projects/guild-core
go-callvis -nostd -group pkg -focus github.com/guild-framework/guild-core/cmd/guild ./cmd/guild
```

The graph opens at `http://localhost:7878` by default. Add `-skipbrowser` if you want to open it manually.

## Static Graphs for Documentation

Store generated images under `docs/images/callgraphs/` so they can be linked from docs.

```bash
cd projects/guild-core
mkdir -p docs/images/callgraphs
```

### CLI -> Daemon Integration

High-level package view for the Great Hall CLI calling into daemon orchestration.

```bash
go-callvis -nostd -nointer -group pkg -graphviz -format svg \
  -file docs/images/callgraphs/cli-daemon.svg \
  -focus github.com/guild-framework/guild-core/cmd/guild \
  -limit github.com/guild-framework/guild-core/cmd/guild,github.com/guild-framework/guild-core/internal/daemon,github.com/guild-framework/guild-core/internal/daemonconn,github.com/guild-framework/guild-core/internal/integration \
  ./cmd/guild
```

### Daemon Lifecycle

Focus the daemon package to see lifecycle wiring and core dependencies.

```bash
go-callvis -nostd -group pkg -graphviz -format svg \
  -file docs/images/callgraphs/daemon-lifecycle.svg \
  -focus github.com/guild-framework/guild-core/internal/daemon \
  -limit github.com/guild-framework/guild-core/internal/daemon,github.com/guild-framework/guild-core/internal/integration,github.com/guild-framework/guild-core/pkg \
  ./internal/daemon
```

### Integration Services Overview

Map the service layer and its bridges into the broader system.

```bash
go-callvis -nostd -group pkg -graphviz -format svg \
  -file docs/images/callgraphs/integration-services.svg \
  -focus github.com/guild-framework/guild-core/internal/integration/services \
  -limit github.com/guild-framework/guild-core/internal/integration,github.com/guild-framework/guild-core/pkg \
  ./internal/integration/services
```

## Tuning for Clarity

- Use `-nointer` for a high-level view (exported functions only).
- Try `-group pkg,type` to cluster by package and type for deeper inspection.
- Add `-rankdir TB` for a top-down layout when graphs are wide.
- Use `-include` or `-limit` to keep graphs focused and readable.

## Troubleshooting

- Missing packages or errors about downloads mean you need to run `go mod download` first.
- If SVG output fails, confirm `dot` is in PATH and try again with `-graphviz`.
