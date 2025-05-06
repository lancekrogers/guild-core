# 🛠️ CLI Tools & Code Assistants

## Code Assistants

- **Claude Code terminal**: included in Claude Max subscriptions; used for interactive code generation and editing
- **Aider**: open-source CLI code assistant; can be configured per-agent for scoped refactoring or implementation tasks

Agents can spawn these assistants as subprocesses and treat their output as part of the prompt chain or final output.

## Command-Line Tools

Each agent may be configured with a custom set of command-line tools.

These tools can:

- Replace repetitive LLM calls
- Standardize transformations (e.g. scaffold generation, file formatting)
- Reduce prompt token costs

## Tool Configuration Format

Each tool should include:

- `name`: Tool identifier
- `cmd`: Executable command
- `context_description`: Natural language description of when to use it
- `args`: Optional templateable arguments (e.g. `{{task}}`)
- `working_dir`: Optional working directory override

### Example

```yaml
tools:
  - name: tree2scaffold
    cmd: "./bin/tree2scaffold"
    context_description: "Convert a folder into a scaffold config before asking LLM to write code."
  - name: goreleaser
    cmd: "goreleaser release --clean"
    context_description: "Create release builds for Go-based CLIs."
  - name: aider
    cmd: "aider --yes --message '{{task}}'"
    context_description: "Refactor code with scoped assistant."
```

## Tool Invocation Behavior

- Agents are responsible for deciding whether to use a tool or LLM
- Managers may override tool decisions for cost control or standardization
- CLI output can be streamed, summarized, or stored depending on config
