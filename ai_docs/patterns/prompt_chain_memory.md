# Prompt Chain Memory

Agents in Guild store a prompt-response chain per task to support recall and refinement.

## Memory Structure

- Stored in BoltDB or in-memory cache.
- Each entry contains: prompt, response, timestamp, task ID.

## Use Cases

- Replay prompt history during context prime.
- Use last N prompts to seed new ones.

## Example

```
Prompt #3:
"How should I design the interface for the agent system?"

Response:
"Start by defining the Task and Result types..."
```
