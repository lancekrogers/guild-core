# OpenAI SDK Integration

This document outlines how Guild integrates with the OpenAI API for LLM-based tasks.

## Usage Strategy

- Agents use Claude by default for planning and reasoning-heavy tasks.
- Local models are preferred for implementation (LLaMA3, etc.).
- Cost control is enforced via `guild.yaml`.

## API Considerations

- Respect rate limits and token quotas.
- Use model-specific API keys when necessary.

## Prompt Tips

- Use deterministic system prompts.
- Tag cost-heavy calls in plan specs with `@ultrahink`.
