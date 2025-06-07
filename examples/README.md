# Guild Framework Examples

This directory contains example code demonstrating various features of the Guild Framework.

## Build Tags

The example files in this directory use the `example` build tag to exclude them from normal builds. This prevents compilation errors during regular development and testing.

To run an example, use:

```bash
go run -tags example examples/commission_refinement_example.go
```

Or build with:

```bash
go build -tags example examples/commission_refinement_example.go
```

## Available Examples

### Commission Refinement Example
`commission_refinement_example.go` - Demonstrates the complete commission refinement pipeline, including:
- Setting up the component registry
- Configuring AI providers (Anthropic, OpenAI)
- Creating and processing a commission through the GuildMaster refiner
- Generating tasks on the kanban board
- Using the TaskBridge for commission-based task queries

### Layered Prompt Example
`prompts/layered_prompt_example.go` - Shows how to use the layered prompt system:
- Creating platform and session-specific prompt layers
- Building layered prompts with context
- Managing prompt caching
- Token optimization

## Prerequisites

Before running the examples, ensure you have:
1. Set up appropriate environment variables for AI providers:
   - `ANTHROPIC_API_KEY` for Anthropic Claude
   - `OPENAI_API_KEY` for OpenAI GPT models
2. Created the `.guild` directory structure
3. Installed all dependencies with `go mod download`

## Note

These examples are for demonstration purposes and may use simplified mock implementations. They are designed to show API usage patterns rather than production-ready code.
